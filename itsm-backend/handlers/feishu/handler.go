package feishu

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"itsm-backend/common"
	"itsm-backend/connector"
	feishuConn "itsm-backend/connector/builtin/feishu"
	"itsm-backend/dto"
	"itsm-backend/service"
	marketplaceService "itsm-backend/service/marketplace"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	connectorManager *connector.Manager
	syncService      *service.FeishuSyncService
	marketplace      *marketplaceService.Service
	logger           *zap.SugaredLogger
	inboundDedup     *connector.InboundDedup
	replayMu         sync.Mutex
	replayed         map[string]time.Time
	oauthStateMu     sync.Mutex
	oauthStates      map[string]oauthStateEntry
}

// oauthStateEntry 是一次 OAuth 授权的服务端 state 绑定。state 必须是不可猜测的
// 随机串，且与发起时的租户/连接器实例/用户绑定、单次消费，回调时据此防 CSRF。
type oauthStateEntry struct {
	tenantID   int
	instanceID string
	userID     int
	expiresAt  time.Time
}

const oauthStateTTL = 10 * time.Minute

func NewHandler(
	connectorManager *connector.Manager,
	syncService *service.FeishuSyncService,
	marketplace *marketplaceService.Service,
	logger *zap.SugaredLogger,
) *Handler {
	return &Handler{
		connectorManager: connectorManager,
		syncService:      syncService,
		marketplace:      marketplace,
		logger:           logger,
		replayed:         make(map[string]time.Time),
		oauthStates:      make(map[string]oauthStateEntry),
	}
}

// SetInboundDedup 注入持久化入站去重器（替代旧的 in-memory nonce map）。
// nil 时退回 in-memory 实现，保留兼容。
func (h *Handler) SetInboundDedup(d *connector.InboundDedup) { h.inboundDedup = d }

// RegisterRoutes 已删除：飞书域唯一的路由所有者是 router.SetupFeishuRoutes（由 router.go 以
// auth/public 两个分组注册）。此处曾有一份逐字重复的 4 条注册（oauth/auth-url、
// oauth/callback/:instance_id、sync/ticket/:ticket_id、webhook/:instance_id），从未被调用；
// 若与 router 那份同时生效，gin 会因重复路由直接 panic。它的两个分组参数由调用方决定，
// 旧测试正是以 RegisterRoutes(api, api) 同一分组调用——按那种形态接线会把外部回调挪进鉴权组。

// getFeishuConnector 获取当前租户的飞书连接器（仅从认证上下文取 tenant_id，禁止 query 参数绕过）
func (h *Handler) getFeishuConnector(ctx *gin.Context) (*feishuConn.Feishu, int, bool) {
	tenantID := ctx.GetInt("tenant_id")
	if tenantID <= 0 {
		return nil, 0, false
	}
	conn, ok := h.connectorManager.Get(tenantID, "feishu")
	if !ok {
		return nil, tenantID, false
	}
	fc, ok := conn.(*feishuConn.Feishu)
	return fc, tenantID, ok
}

// getFeishuConnectorPublic uses a high-entropy connector instance ID; public
// callbacks never accept an enumerable tenant ID.
func (h *Handler) getFeishuConnectorPublic(ctx *gin.Context) (*feishuConn.Feishu, int, bool) {
	instanceID := ctx.Param("instance_id")
	conn, tenantID, ok := h.connectorManager.GetByCallbackInstanceID("feishu", instanceID)
	if !ok {
		return nil, tenantID, false
	}
	fc, ok := conn.(*feishuConn.Feishu)
	return fc, tenantID, ok
}

// GetOAuthAuthURL returns the Feishu OAuth authorization URL
func (h *Handler) GetOAuthAuthURL(ctx *gin.Context) {
	fc, tenantID, ok := h.getFeishuConnector(ctx)
	if !ok {
		// 「连接器未配置」是部署/配置状态，不是服务端 bug：不能用 500。
		// 500 会被监控当成代码故障告警，也让调用方误以为是可重试的瞬时错误。
		// 用 503 并给出可执行的处置指引（2026-10-18 缺陷 P1-5）。
		common.Fail(ctx, common.ServiceUnavailableCode,
			"飞书连接器未配置：请先在「连接器」中创建飞书实例并填写 app_id / app_secret")
		return
	}
	redirectURI := ctx.Query("redirectUri")
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("%s://%s/api/v1/feishu/oauth/callback/%s", requestScheme(ctx), ctx.Request.Host, fc.CallbackInstanceID())
	}
	// state 由服务端生成不可猜测的随机串并绑定租户/实例/用户，回调时单次消费校验。
	// 绝不信任客户端传入的 state，也不再回退到可枚举的 tenant_id/user_id 拼接——
	// 那等于让攻击者用猜得到的 state 把自己的 OAuth 授权塞进受害者的连接器（CSRF）。
	state, err := h.issueOAuthState(tenantID, fc.CallbackInstanceID(), ctx.GetInt("user_id"))
	if err != nil {
		h.logger.Errorw("Failed to issue Feishu OAuth state", "tenant_id", tenantID, "err", err)
		common.Fail(ctx, common.InternalErrorCode, "生成授权状态失败，请稍后重试")
		return
	}
	common.Success(ctx, &dto.FeishuOAuthAuthURLResponse{
		AuthURL:     fc.GetOAuthAuthURL(redirectURI, state),
		RedirectURI: redirectURI,
		State:       state,
	})
}

// OAuthCallback handles the Feishu OAuth callback
func (h *Handler) OAuthCallback(ctx *gin.Context) {
	code := ctx.Query("code")
	if code == "" {
		common.Fail(ctx, common.ParamErrorCode, "code is required")
		return
	}
	fc, tenantID, ok := h.getFeishuConnectorPublic(ctx)
	if !ok {
		common.Fail(ctx, common.ParamErrorCode, "Invalid request")
		return
	}
	// 校验并单次消费 state：必须与发起时绑定的租户/实例一致，缺失、伪造、过期或
	// 重放一律拒绝（403），且必须在换取 token 之前——否则 CSRF 授权码已被兑换。
	state := ctx.Query("state")
	if !h.consumeOAuthState(state, fc.CallbackInstanceID(), tenantID) {
		h.logger.Warnw("Rejected Feishu OAuth callback with invalid state", "tenant_id", tenantID)
		common.Fail(ctx, common.ForbiddenCode, "授权状态无效或已过期，请重新发起授权")
		return
	}
	token, err := fc.ExchangeOAuthCode(ctx.Request.Context(), code)
	if err != nil {
		h.logger.Errorw("Failed to exchange Feishu OAuth code", "err", err)
		common.Fail(ctx, common.InternalErrorCode, "Failed to exchange Feishu OAuth code")
		return
	}
	if h.marketplace != nil {
		_, err = h.marketplace.MergeConnectorInstallationConfig(ctx.Request.Context(), tenantID, "feishu", map[string]interface{}{
			"oauth": map[string]interface{}{
				"access_token":  token.AccessToken,
				"refresh_token": token.RefreshToken,
				"expires_in":    token.ExpiresIn,
				"token_type":    token.TokenType,
				"scope":         token.Scope,
				"user_id":       token.UserID,
				"open_id":       token.OpenID,
				"union_id":      token.UnionID,
			},
		})
		if err != nil {
			h.logger.Errorw("Failed to persist Feishu OAuth callback", "tenant_id", tenantID, "err", err)
			common.Fail(ctx, common.InternalErrorCode, "Failed to persist Feishu OAuth callback")
			return
		}
	}
	common.Success(ctx, &dto.FeishuOAuthCallbackResponse{
		ExpiresIn: token.ExpiresIn,
		TokenType: token.TokenType,
		Scope:     token.Scope,
		UserID:    token.UserID,
		OpenID:    token.OpenID,
		UnionID:   token.UnionID,
	})
}

// SyncTicketToFeishu syncs a ticket to Feishu as a task
func (h *Handler) SyncTicketToFeishu(ctx *gin.Context) {
	ticketIDStr := ctx.Param("ticket_id")
	ticketID, err := strconvAtoi(ticketIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "Invalid ticket ID")
		return
	}
	fc, tenantID, ok := h.getFeishuConnector(ctx)
	if !ok {
		// 「连接器未配置」是部署/配置状态，不是服务端 bug：不能用 500。
		// 500 会被监控当成代码故障告警，也让调用方误以为是可重试的瞬时错误。
		// 用 503 并给出可执行的处置指引（2026-10-18 缺陷 P1-5）。
		common.Fail(ctx, common.ServiceUnavailableCode,
			"飞书连接器未配置：请先在「连接器」中创建飞书实例并填写 app_id / app_secret")
		return
	}
	resp, err := h.syncService.SyncTicketToFeishu(ctx.Request.Context(), tenantID, ticketID, fc)
	if err != nil {
		h.logger.Errorw("Failed to sync ticket to Feishu", "ticket_id", ticketID, "tenant_id", tenantID, "err", err)
		common.RespondError(ctx, err, "操作失败")
		return
	}
	common.Success(ctx, resp)
}

// Webhook handles Feishu webhook events
func (h *Handler) Webhook(ctx *gin.Context) {
	fc, tenantID, ok := h.getFeishuConnectorPublic(ctx)
	if !ok {
		common.Fail(ctx, common.ParamErrorCode, "Invalid request")
		return
	}

	rawData, _ := ctx.GetRawData()

	// 将 http.Header 转换为 map[string]string
	headers := make(map[string]string)
	for k, v := range ctx.Request.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	err := fc.VerifySignature(headers, rawData)
	if err != nil {
		h.logger.Errorw("Invalid Feishu webhook signature", "err", err)
		common.Fail(ctx, common.ForbiddenCode, "Invalid signature")
		return
	}
	// 先去重再处理 URL Verification：飞书每次请求都会生成新的 timestamp+nonce，
	// 重复保存配置不会撞 nonce；而把所有回调（含探活）都纳入一次性 nonce，
	// 才能保证「同一请求重放一律拒绝」这条安全契约没有例外口子。
	if err := h.consumeWebhookNonce(ctx.Request.Context(), tenantID, headers["X-Lark-Request-Timestamp"], headers["X-Lark-Request-Nonce"], rawData); err != nil {
		if errors.Is(err, errWebhookReplay) {
			h.logger.Warnw("Rejected replayed Feishu webhook",
				"tenant_id", tenantID, "timestamp", headers["X-Lark-Request-Timestamp"])
			common.Fail(ctx, common.ForbiddenCode, "Replay detected")
			return
		}
		// 去重存储不可用时 fail-closed：放过去重等于允许平台重投的同一事件
		// 被二次执行（task_event 会重复建/改飞书任务）。
		h.logger.Errorw("Feishu webhook dedup unavailable", "tenant_id", tenantID, "err", err)
		common.Fail(ctx, common.InternalErrorCode, "去重校验失败，请稍后重试")
		return
	}
	if handled := h.handleURLVerification(ctx, fc, rawData); handled {
		return
	}

	msg, err := fc.ParseInbound(rawData)
	if err != nil {
		h.logger.Errorw("Failed to parse Feishu webhook event", "err", err)
		common.Fail(ctx, common.ParamErrorCode, "Invalid event payload")
		return
	}

	if msg.Type == "task_event" {
		eventType := msg.Content
		taskData, ok := msg.Extras["task_data"].(map[string]interface{})
		if !ok {
			h.forgetWebhookEvent(ctx.Request.Context(), tenantID, headers["X-Lark-Request-Timestamp"], headers["X-Lark-Request-Nonce"])
			common.Fail(ctx, common.ParamErrorCode, "Invalid task event data")
			return
		}

		resp, err := h.syncService.HandleTaskEvent(ctx.Request.Context(), tenantID, fc, eventType, taskData)
		if err != nil {
			h.logger.Errorw("Failed to handle Feishu task event", "event_type", eventType, "err", err)
			// 处理失败必须释放去重行：飞书会重投同一事件，若行还在，
			// 重投会被判「重复」直接 200 掉，事件就此丢失。
			h.forgetWebhookEvent(ctx.Request.Context(), tenantID, headers["X-Lark-Request-Timestamp"], headers["X-Lark-Request-Nonce"])
			common.Fail(ctx, common.InternalErrorCode, "Failed to handle event")
			return
		}
		h.markWebhookProcessed(ctx.Request.Context(), tenantID, headers["X-Lark-Request-Timestamp"], headers["X-Lark-Request-Nonce"], "ok")
		common.Success(ctx, resp)
		return
	}

	h.markWebhookProcessed(ctx.Request.Context(), tenantID, headers["X-Lark-Request-Timestamp"], headers["X-Lark-Request-Nonce"], "ignored")
	common.Success(ctx, &dto.FeishuWebhookResponse{EventType: msg.Type, Action: "ignored"})
}

// errWebhookReplay 表示时间戳越界或 nonce 已消费：这是攻击/重投信号，不是服务故障。
var errWebhookReplay = errors.New("feishu webhook replay")

// issueOAuthState 生成不可猜测的随机 state，并绑定发起时的租户/连接器实例/用户。
//
// 内存存储是单进程语义：多副本部署下回调落到另一副本会因查不到 state 而拒绝
// （fail-closed，安全但牺牲可用性），HA 需要共享存储或会话粘滞，与 webhook 去重
// 的内存回退同一局限。这里不引入新持久表，保持改动 scoped。
func (h *Handler) issueOAuthState(tenantID int, instanceID string, userID int) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	state := hex.EncodeToString(buf)
	now := time.Now()
	h.oauthStateMu.Lock()
	defer h.oauthStateMu.Unlock()
	for k, e := range h.oauthStates {
		if now.After(e.expiresAt) {
			delete(h.oauthStates, k)
		}
	}
	h.oauthStates[state] = oauthStateEntry{
		tenantID:   tenantID,
		instanceID: instanceID,
		userID:     userID,
		expiresAt:  now.Add(oauthStateTTL),
	}
	return state, nil
}

// consumeOAuthState 单次消费 state：命中即删除（无论校验是否通过），再校验过期与
// 租户/实例绑定。缺失、未知、过期、重放或绑定不符一律返回 false。
func (h *Handler) consumeOAuthState(state, instanceID string, tenantID int) bool {
	if state == "" {
		return false
	}
	h.oauthStateMu.Lock()
	defer h.oauthStateMu.Unlock()
	entry, ok := h.oauthStates[state]
	if !ok {
		return false
	}
	delete(h.oauthStates, state)
	if time.Now().After(entry.expiresAt) {
		return false
	}
	return entry.tenantID == tenantID && entry.instanceID == instanceID
}

// consumeWebhookNonce 做两件事：校验时间戳窗口，并消费 nonce（一次性）。
//
// 去重优先走持久化表（connector.InboundDedup）：内存 map 在重启后清空、多副本
// 各存一份，而飞书回调是全仓唯一会真正改写业务数据（task_event → 同步任务）的
// 入站路径，重复执行的代价是重复建/改任务。注入缺失时才回退内存 map。
//
// 返回 errWebhookReplay 表示拒绝；其他 error 表示去重存储不可用（调用方 fail-closed）。
func (h *Handler) consumeWebhookNonce(ctx context.Context, tenantID int, timestamp, nonce string, payload []byte) error {
	if tenantID <= 0 || timestamp == "" || nonce == "" {
		return errWebhookReplay
	}
	unixTimestamp, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errWebhookReplay
	}
	now := time.Now()
	requestTime := time.Unix(unixTimestamp, 0)
	if requestTime.Before(now.Add(-5*time.Minute)) || requestTime.After(now.Add(5*time.Minute)) {
		return errWebhookReplay
	}

	// A nonce is single-use within a tenant and timestamp. The signature is
	// deliberately excluded: including it would allow the same nonce to be
	// consumed repeatedly by changing the body and recomputing the signature.
	eventID := fmt.Sprintf("%s:%s", timestamp, nonce)
	if h.inboundDedup != nil {
		// 持久化表按 (tenant_id, connector_name, event_id) 唯一，租户隔离在列里，
		// event_id 不需要再嵌租户号。
		seen, err := h.inboundDedup.HandleBeforeProcess(ctx, tenantID, "feishu", eventID, payload)
		if err != nil {
			return fmt.Errorf("feishu inbound dedup: %w", err)
		}
		if seen {
			return errWebhookReplay
		}
		return nil
	}

	// 内存回退：退化成单进程语义，键必须自带租户号（没有表来隔离）。
	memKey := fmt.Sprintf("%d:%s", tenantID, eventID)
	h.replayMu.Lock()
	defer h.replayMu.Unlock()
	for k, expires := range h.replayed {
		if now.After(expires) {
			delete(h.replayed, k)
		}
	}
	if _, exists := h.replayed[memKey]; exists {
		return errWebhookReplay
	}
	h.replayed[memKey] = now.Add(5 * time.Minute)
	return nil
}

// markWebhookProcessed 把本次事件的处置结果写回去重行，供运维判断
// 「飞书重投了但我这边到底处理成功了没有」。
func (h *Handler) markWebhookProcessed(ctx context.Context, tenantID int, timestamp, nonce, status string) {
	if h.inboundDedup == nil {
		return
	}
	key := fmt.Sprintf("%s:%s", timestamp, nonce)
	if err := h.inboundDedup.MarkProcessed(ctx, tenantID, "feishu", key, status); err != nil {
		h.logger.Warnw("Failed to mark Feishu webhook processing status",
			"tenant_id", tenantID, "status", status, "err", err)
	}
}

// forgetWebhookEvent 释放去重行，让平台重投能重新处理（见 InboundDedup.Forget）。
func (h *Handler) forgetWebhookEvent(ctx context.Context, tenantID int, timestamp, nonce string) {
	if h.inboundDedup == nil {
		return
	}
	key := fmt.Sprintf("%s:%s", timestamp, nonce)
	if err := h.inboundDedup.Forget(ctx, tenantID, "feishu", key); err != nil {
		h.logger.Warnw("Failed to release Feishu webhook dedup row",
			"tenant_id", tenantID, "err", err)
	}
}

func (h *Handler) handleURLVerification(ctx *gin.Context, fc *feishuConn.Feishu, rawData []byte) bool {
	var req struct {
		Type      string `json:"type"`
		Token     string `json:"token"`
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(rawData, &req); err != nil || req.Type != "url_verification" {
		return false
	}
	if !fc.VerifyURLToken(req.Token) {
		common.Fail(ctx, common.ForbiddenCode, "Invalid verification token")
		return true
	}
	ctx.JSON(200, gin.H{"challenge": req.Challenge})
	return true
}

func requestScheme(ctx *gin.Context) string {
	if proto := ctx.GetHeader("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	if ctx.Request.TLS != nil {
		return "https"
	}
	return "http"
}

func strconvAtoi(s string) (int, error) {
	var i int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid number")
		}
		i = i*10 + int(c-'0')
	}
	return i, nil
}
