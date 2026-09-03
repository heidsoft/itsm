package feishu

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"itsm-backend/common"
	"itsm-backend/connector"
	feishuConn "itsm-backend/connector/builtin/feishu"
	"itsm-backend/dto"
	"itsm-backend/middleware"
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
	replayMu         sync.Mutex
	replayed         map[string]time.Time
}

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
	}
}

// RegisterRoutes registers feishu routes
func (h *Handler) RegisterRoutes(auth *gin.RouterGroup, public *gin.RouterGroup) {
	feishu := auth.Group("/feishu")
	{
		feishu.GET("/oauth/auth-url", middleware.RequirePermission("feishu", "use"), h.GetOAuthAuthURL)
		feishu.POST("/sync/ticket/:ticket_id", middleware.RequirePermission("ticket", "update"), h.SyncTicketToFeishu)
	}

	// Public callbacks (no auth required)
	public.GET("/feishu/oauth/callback/:instance_id", h.OAuthCallback)
	public.POST("/feishu/webhook/:instance_id", h.Webhook)
}

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
		common.Fail(ctx, common.InternalErrorCode, "Feishu connector not configured")
		return
	}
	redirectURI := ctx.Query("redirectUri")
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("%s://%s/api/v1/feishu/oauth/callback/%s", requestScheme(ctx), ctx.Request.Host, fc.CallbackInstanceID())
	}
	state := ctx.Query("state")
	if state == "" {
		state = fmt.Sprintf("tenant_id=%d&user_id=%d", tenantID, ctx.GetInt("user_id"))
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
		common.Fail(ctx, common.InternalErrorCode, "Feishu connector not configured")
		return
	}
	resp, err := h.syncService.SyncTicketToFeishu(ctx.Request.Context(), tenantID, ticketID, fc)
	if err != nil {
		h.logger.Errorw("Failed to sync ticket to Feishu", "ticket_id", ticketID, "tenant_id", tenantID, "err", err)
		common.Fail(ctx, common.InternalErrorCode, err.Error())
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
	if !h.consumeWebhookNonce(tenantID, headers["X-Lark-Request-Timestamp"], headers["X-Lark-Request-Nonce"]) {
		common.Fail(ctx, common.ForbiddenCode, "Replay detected")
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
			common.Fail(ctx, common.ParamErrorCode, "Invalid task event data")
			return
		}

		resp, err := h.syncService.HandleTaskEvent(ctx.Request.Context(), tenantID, fc, eventType, taskData)
		if err != nil {
			h.logger.Errorw("Failed to handle Feishu task event", "event_type", eventType, "err", err)
			common.Fail(ctx, common.InternalErrorCode, "Failed to handle event")
			return
		}
		common.Success(ctx, resp)
		return
	}

	common.Success(ctx, &dto.FeishuWebhookResponse{EventType: msg.Type, Action: "ignored"})
}

func (h *Handler) consumeWebhookNonce(tenantID int, timestamp, nonce string) bool {
	if tenantID <= 0 || timestamp == "" || nonce == "" {
		return false
	}
	unixTimestamp, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	now := time.Now()
	requestTime := time.Unix(unixTimestamp, 0)
	if requestTime.Before(now.Add(-5*time.Minute)) || requestTime.After(now.Add(5*time.Minute)) {
		return false
	}

	// A nonce is single-use within a tenant and timestamp. The signature is
	// deliberately excluded: including it would allow the same nonce to be
	// consumed repeatedly by changing the body and recomputing the signature.
	key := fmt.Sprintf("%d:%s:%s", tenantID, timestamp, nonce)
	h.replayMu.Lock()
	defer h.replayMu.Unlock()
	for k, expires := range h.replayed {
		if now.After(expires) {
			delete(h.replayed, k)
		}
	}
	if _, exists := h.replayed[key]; exists {
		return false
	}
	h.replayed[key] = now.Add(5 * time.Minute)
	return true
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
