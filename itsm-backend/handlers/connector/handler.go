package connector

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/connector"
	"itsm-backend/connector/marketplace"
	"itsm-backend/dto"

	domainCommon "itsm-backend/handlers/common"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 连接器HTTP处理器
type Handler struct {
	manager  *connector.Manager
	market   *marketplace.Market
	registry *connector.Registry
	logger   *zap.SugaredLogger
	store    *connector.PersistentConfigStore
	auditRec AuditRecorder
}

// NewHandler creates a new connector handler
func NewHandler(mgr *connector.Manager, reg *connector.Registry, mkt *marketplace.Market, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{manager: mgr, market: mkt, registry: reg, logger: logger}
}

// SetAuditRecorder 注入审计写入器，用于 rotate-secret 等需要写 audit_log 的端点。
func (h *Handler) SetAuditRecorder(rec AuditRecorder) { h.auditRec = rec }

// SetPersistentStore sets the persistent config store
func (h *Handler) SetPersistentStore(store *connector.PersistentConfigStore) {
	h.store = store
}

// requireStore 校验持久化存储已注入，未注入时拒绝写操作并显性报错。
//
// 背景：历史上 store 曾被误注入到无路由挂载的 controller.ConnectorController，
// 而实际承接路由的 handler 的 h.store 始终为 nil。由于旧逻辑用
// `if h.store != nil` 静默跳过落库，配置只写入内存 manager，重启即丢失，
// 且没有任何报错。这里改为 fail-loudly，避免同类接线遗漏再次静默丢数据。
func (h *Handler) requireStore(ctx *gin.Context) bool {
	if h.store == nil {
		if h.logger != nil {
			h.logger.Errorw("connector persistent store is not configured; refusing write to avoid losing config on restart")
		}
		common.Fail(ctx, common.InternalErrorCode, "连接器配置持久化未就绪，拒绝保存")
		return false
	}
	return true
}

// ListMarket 列出市场中所有可用连接器
func (h *Handler) ListMarket(ctx *gin.Context) {
	reg := h.registry
	if reg == nil {
		reg = connector.Default()
	}
	mfs := reg.List()
	tenantID := ctx.GetInt("tenant_id")
	configs := h.manager.ListByTenant(tenantID)
	installed := make(map[string]bool, len(configs))
	enabled := make(map[string]bool, len(configs))
	for _, cfg := range configs {
		installed[cfg.Name] = true
		enabled[cfg.Name] = cfg.Enabled
	}
	health := h.manager.HealthCheckTenant(ctx.Request.Context(), tenantID)
	out := make([]dto.ConnectorManifestDTO, 0, len(mfs))
	for _, m := range mfs {
		healthy, checkedAt, lastErr := healthForManifest(health, tenantID, m.Name)
		out = append(out, dto.ConnectorManifestDTO{
			Name:                m.Name,
			Version:             m.Version,
			Title:               m.Title,
			Provider:            m.Provider,
			Type:                string(m.Type),
			Description:         m.Description,
			Author:              m.Author,
			Homepage:            m.Homepage,
			IconURL:             m.IconURL,
			Capabilities:        capToString(m.Capabilities),
			Tags:                m.Tags,
			MinITSMVer:          m.MinITSMVer,
			Local:               true,
			Installed:           installed[m.Name],
			Enabled:             enabled[m.Name],
			Healthy:             healthy,
			LastCheckedAt:       checkedAt,
			LastError:           lastErr,
			Lifecycle:           connectorLifecycle(installed[m.Name], enabled[m.Name], healthy, lastErr),
			Category:            string(m.Type),
			IsOfficial:          m.IsOfficial,
			RequiredPermissions: m.RequiredPermissions,
			Checksum:            m.Checksum,
		})
	}
	common.Success(ctx, gin.H{"items": out, "total": len(out)})
}

// ListConfigs 列出当前租户已配置的连接器实例（凭据脱敏）
func (h *Handler) ListConfigs(ctx *gin.Context) {
	tenantID := ctx.GetInt("tenant_id")
	cfgs := h.manager.ListByTenant(tenantID)
	health := h.manager.HealthCheckTenant(ctx.Request.Context(), tenantID)
	out := make([]dto.ConnectorConfigDTO, 0, len(cfgs))
	for _, cfg := range cfgs {
		out = append(out, maskConfig(cfg, health))
	}
	common.Success(ctx, gin.H{"items": out, "total": len(out)})
}

// Provision 创建/更新一个连接器实例
func (h *Handler) Provision(ctx *gin.Context) {
	var req dto.ProvisionConnectorRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(ctx, err, "请求参数错误")
		return
	}
	tenantID := ctx.GetInt("tenant_id")
	if len(req.Credentials) == 0 {
		for _, existing := range h.manager.ListByTenant(tenantID) {
			if existing.Name == req.Name && existing.Provider == req.Provider {
				req.Credentials = existing.Credentials
				break
			}
		}
	}
	if req.Settings == nil {
		req.Settings = make(map[string]interface{})
	}
	callbackInstanceID := ""
	for _, existing := range h.manager.ListByTenant(tenantID) {
		if existing.Name == req.Name {
			callbackInstanceID, _ = existing.Settings["callbackInstanceId"].(string)
			break
		}
	}
	if callbackInstanceID == "" {
		buf := make([]byte, 24)
		if _, err := rand.Read(buf); err != nil {
			common.Fail(ctx, common.InternalErrorCode, "无法生成回调实例标识")
			return
		}
		callbackInstanceID = hex.EncodeToString(buf)
	}
	req.Settings["callbackInstanceId"] = callbackInstanceID
	cfg := connector.Config{
		TenantID:    tenantID,
		Name:        req.Name,
		Provider:    req.Provider,
		Enabled:     req.Enabled,
		Credentials: req.Credentials,
		Settings:    req.Settings,
		Labels:      req.Labels,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	for _, manifest := range h.registry.List() {
		if manifest.Name == req.Name {
			cfg.Type = manifest.Type
			break
		}
	}
	if !h.requireStore(ctx) {
		return
	}
	if err := h.manager.Provision(ctx.Request.Context(), cfg); err != nil {
		common.RespondError(ctx, err, "操作失败")
		return
	}
	if err := h.store.Save(ctx.Request.Context(), cfg); err != nil {
		h.manager.Revoke(cfg)
		common.RespondError(ctx, err, "操作失败")
		return
	}
	common.Success(ctx, maskConfig(cfg, h.manager.HealthCheckTenant(ctx.Request.Context(), tenantID)))
}

// Revoke 停用并移除一个连接器实例
func (h *Handler) Revoke(ctx *gin.Context) {
	name := ctx.Param("name")
	tenantID := ctx.GetInt("tenant_id")
	provider := ctx.Query("provider")
	if provider == "" {
		for _, cfg := range h.manager.ListByTenant(tenantID) {
			if cfg.Name == name {
				provider = cfg.Provider
				break
			}
		}
	}
	if !h.requireStore(ctx) {
		return
	}
	h.manager.Revoke(connector.Config{TenantID: tenantID, Name: name, Provider: provider})
	if err := h.store.Delete(ctx.Request.Context(), tenantID, name, provider); err != nil {
		common.RespondError(ctx, err, "操作失败")
		return
	}
	common.Success(ctx, gin.H{"name": name, "revoked": true})
}

// Send 通过指定连接器发消息
func (h *Handler) Send(ctx *gin.Context) {
	name := ctx.Param("name")
	var req dto.SendConnectorMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(ctx, err, "请求参数错误")
		return
	}
	tenantID := ctx.GetInt("tenant_id")
	msg := &connector.Message{
		Channel:  req.Channel,
		Type:     req.Type,
		Title:    req.Title,
		Content:  req.Content,
		Mentions: convertMentions(req.Mentions),
		Actions:  convertActions(req.Actions),
		Metadata: req.Metadata,
	}
	if req.Card != nil {
		msg.Card = convertCard(req.Card)
	}
	if err := h.manager.Send(ctx.Request.Context(), tenantID, name, msg); err != nil {
		common.RespondError(ctx, err, "操作失败")
		return
	}
	common.Success(ctx, gin.H{"name": name, "channel": req.Channel, "sent": true})
}

// Test 发送一条简单测试消息
func (h *Handler) Test(ctx *gin.Context) {
	name := ctx.Param("name")
	tenantID := ctx.GetInt("tenant_id")
	var channel string
	for _, cfg := range h.manager.ListByTenant(tenantID) {
		if cfg.Name == name {
			if ch, ok := connector.SettingString(cfg.Settings, "debug_channel"); ok {
				channel = ch
			}
		}
	}
	if channel == "" {
		common.Fail(ctx, common.ParamErrorCode, "settings.debug_channel not configured for "+name)
		return
	}
	if err := h.manager.Send(ctx.Request.Context(), tenantID, name, &connector.Message{
		Channel: channel,
		Type:    "text",
		Title:   "ITSM 连接器测试",
		Content: "这是一条来自 ITSM 的测试消息。\n时间: " + time.Now().Format(time.RFC3339),
	}); err != nil {
		common.RespondError(ctx, err, "操作失败")
		return
	}
	common.Success(ctx, gin.H{"name": name, "channel": channel, "sent": true})
}

// Health 对**当前租户**运行中的连接器实例做健康检查。
//
// 安全边界（此前是缺陷）：健康检查会真实拨号到第三方（SMTP / 飞书 / 钉钉），
// 曾经这里调的是 HealthCheckAll——任何持 connector.read 的租户都能
//
//	① 读到其他租户连接器的名字、provider 与在线状态（响应以 tenantID/name/provider 为键）；
//	② 借一次请求去探测别家租户的端点可达性（等于跨租户连通性探针）。
//
// 现在只拨本租户实例（Manager.HealthCheckTenant），响应按 name/provider 标识实例，
// 租户号由上下文决定并回显一次，前端不必再猜租户号。
func (h *Handler) Health(ctx *gin.Context) {
	tenantID := ctx.GetInt("tenant_id")
	if tenantID <= 0 {
		common.Fail(ctx, common.AuthFailedCode, "租户信息缺失")
		return
	}
	res := h.manager.HealthCheckTenant(ctx.Request.Context(), tenantID)
	prefix := fmt.Sprintf("%d/", tenantID)
	items := make([]dto.ConnectorHealthItemDTO, 0, len(res))
	for k, v := range res {
		// 双保险：即使 Manager 未来改动漏了租户过滤，也不把别家实例带出去。
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		name, provider, ok := splitInstanceKey(k)
		if !ok {
			continue
		}
		items = append(items, dto.ConnectorHealthItemDTO{
			Name:      name,
			Provider:  provider,
			OK:        v.OK,
			LatencyMs: v.LatencyMs,
			Message:   v.Message,
			CheckedAt: nonZeroTime(v.CheckedAt),
			Extra:     v.Extra,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name != items[j].Name {
			return items[i].Name < items[j].Name
		}
		return items[i].Provider < items[j].Provider
	})
	common.Success(ctx, dto.ConnectorHealthResponse{TenantID: tenantID, Items: items, CheckedAt: time.Now()})
}

// splitInstanceKey 拆分 Manager 的实例键 "tenantID/name/provider"。
// provider 段整体保留（不按 "/" 再切），避免 provider 值里出现分隔符时被截断。
func splitInstanceKey(key string) (name, provider string, ok bool) {
	parts := strings.SplitN(key, "/", 3)
	if len(parts) != 3 || parts[1] == "" || parts[2] == "" {
		return "", "", false
	}
	return parts[1], parts[2], true
}

// Lifecycle 连接器生命周期视图
func (h *Handler) Lifecycle(ctx *gin.Context) {
	reg := h.registry
	if reg == nil {
		reg = connector.Default()
	}
	tenantID := ctx.GetInt("tenant_id")
	configs := h.manager.ListByTenant(tenantID)
	configByName := make(map[string]connector.Config, len(configs))
	for _, cfg := range configs {
		configByName[cfg.Name] = cfg
	}
	health := h.manager.HealthCheckTenant(ctx.Request.Context(), tenantID)
	manifests := reg.List()
	out := make([]dto.ConnectorLifecycleDTO, 0, len(manifests))
	for _, m := range manifests {
		cfg, installed := configByName[m.Name]
		enabled := installed && cfg.Enabled
		healthy, checkedAt, lastErr := healthForManifest(health, tenantID, m.Name)
		out = append(out, dto.ConnectorLifecycleDTO{
			Name:          m.Name,
			Provider:      m.Provider,
			Type:          string(m.Type),
			Installed:     installed,
			Enabled:       enabled,
			Healthy:       healthy,
			Lifecycle:     connectorLifecycle(installed, enabled, healthy, lastErr),
			LastCheckedAt: checkedAt,
			LastError:     lastErr,
			Capabilities:  capToString(m.Capabilities),
		})
	}
	common.Success(ctx, gin.H{"items": out, "total": len(out)})
}

// RotateSecret 轮换一个连接器实例的凭据（立即生效）。
//
// 语义（刻意不做「双凭据宽限窗口」）：三家 IM（飞书/企微/钉钉）的 Send 与
// VerifySignature 都只读一份凭据，服务端无法同时用两套密钥收发与验签，所以不存在
// 「新旧凭据并行有效」的状态。旧实现写了一个 grace_period_at 字段，但全仓没有
// 任何代码读它——那是个假的回滚窗口；请求里的 gracePeriodHours 同时被移除。
// 要留退路只有两条：轮换前在对方平台保留旧密钥（可在控制台再启），或用旧凭据
// 再调一次 POST /connectors/configs 覆盖回来。
//
// 失败处理（两条路径都不留半死状态）：
//   - 新凭据 Init 失败 → 内存里仍是旧实例，库里也没动，返回 400 并点明"已保留原凭据"；
//   - Init 成功但落库失败 → 把运行态回滚成旧配置，避免「内存新凭据 / 库里旧凭据」漂移。
//
// 审计只记录连接器、provider、操作人、时间与**凭据键名**，绝不写凭据值。
func (h *Handler) RotateSecret(ctx *gin.Context) {
	name := ctx.Param("name")
	tenantID := ctx.GetInt("tenant_id")
	actorID := ctx.GetInt("user_id")
	if tenantID <= 0 {
		common.Fail(ctx, common.AuthFailedCode, "租户信息缺失")
		return
	}
	if name == "" {
		common.ParamError(ctx, "缺少连接器名称")
		return
	}

	var req struct {
		NewCredentials map[string]string `json:"newCredentials"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(ctx, err, "请求参数错误")
		return
	}
	if len(req.NewCredentials) == 0 {
		common.Fail(ctx, common.ParamErrorCode, "newCredentials 不能为空：至少要提交一个凭据字段")
		return
	}
	if !h.requireStore(ctx) {
		return
	}

	// provider / type / settings 必须沿用原实例：决定 Init 走哪个实现，不能由请求指定。
	previous := connector.Config{}
	found := false
	for _, cfg := range h.manager.ListByTenant(tenantID) {
		if cfg.Name == name {
			previous = cfg
			found = true
			break
		}
	}
	if !found {
		common.NotFound(ctx, "连接器实例不存在："+name)
		return
	}

	rotatedAt := time.Now().UTC()
	rotated := previous
	rotated.Credentials = req.NewCredentials
	// Settings 是 manager 内实例持有的同一个 map：必须复制后再写，
	// 否则会给运行中的实例塞进一个尚未落库的时间戳（旧实现就是这样）。
	rotated.Settings = cloneSettings(previous.Settings)
	rotated.Settings["rotated_at"] = rotatedAt
	rotated.Settings["rotated_by"] = actorID
	rotated.UpdatedAt = time.Now()

	if err := h.manager.Provision(ctx.Request.Context(), rotated); err != nil {
		h.logger.Errorw("connector rotate_secret init failed", "connector", name, "tenant_id", tenantID,
			"error", redactCredentialValues(err.Error(), req.NewCredentials))
		common.ParamError(ctx, "新凭据无法初始化，已保留原凭据")
		return
	}
	if err := h.store.Save(ctx.Request.Context(), rotated); err != nil {
		// 落库失败：把运行态改回旧配置，避免内存与库不一致。
		if restoreErr := h.manager.Provision(ctx.Request.Context(), previous); restoreErr != nil {
			h.logger.Errorw("connector rotate_secret rollback failed; connector runtime may hold unpersisted credentials",
				"connector", name, "tenant_id", tenantID, "error", restoreErr)
		}
		common.RespondError(ctx, err, "轮换失败：凭据未能持久化，已恢复原配置")
		return
	}

	if h.auditRec != nil {
		body, _ := json.Marshal(map[string]interface{}{
			"connector":      name,
			"provider":       rotated.Provider,
			"rotatedAt":      rotatedAt,
			"actorUserId":    actorID,
			"credentialKeys": sortedKeys(req.NewCredentials),
		})
		if err := h.auditRec.CreateAuditLog(ctx.Request.Context(), &domainCommon.AuditLog{
			TenantID:    tenantID,
			UserID:      actorID,
			Resource:    "connector",
			Action:      "rotate_secret",
			Method:      "POST",
			Path:        ctx.Request.URL.Path,
			StatusCode:  200,
			RequestBody: string(body),
		}); err != nil {
			// 审计写失败不回滚轮换：密钥已经在第三方平台换掉，回滚只会造成更大不一致。
			// 但必须显性留痕，便于事后补审计。
			h.logger.Errorw("connector rotate_secret audit failed (rotation already applied)",
				"connector", name, "tenant_id", tenantID, "error", err)
		}
	}

	common.Success(ctx, gin.H{
		"name":      name,
		"provider":  rotated.Provider,
		"rotated":   true,
		"rotatedAt": rotatedAt,
		"note":      "旧凭据已立即失效；如需回退，用旧凭据再调一次 POST /connectors/configs",
	})
}

// cloneSettings 浅拷贝 settings，避免修改 manager 内实例共享的 map。
func cloneSettings(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in)+2)
	for k, v := range in {
		out[k] = v
	}
	return out
}

// redactCredentialValues 把错误文案里出现的凭据值替换掉。
//
// Init 失败的错误通常只说"缺哪个键"，但第三方 SDK 的错误偶尔会回显传入的值；
// 轮换接口会把这句话原样返回给调用方，不能让它变成凭据回显通道。
func redactCredentialValues(msg string, credentials map[string]string) string {
	out := msg
	for _, v := range credentials {
		if len(v) >= 4 {
			out = strings.ReplaceAll(out, v, "******")
		}
	}
	return out
}

// sortedKeys 返回排序后的键名，用于审计「轮换了哪些字段」而不暴露值。
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// FeishuCallback 飞书事件回调入口
func (h *Handler) FeishuCallback(ctx *gin.Context) {
	body, _ := ctx.GetRawData()
	tenantID := ctx.GetInt("tenant_id")
	if tenantID <= 0 {
		h.logger.Warnw("Connector FeishuCallback: tenant_id missing in context", "remote_ip", ctx.ClientIP())
		common.Fail(ctx, common.AuthFailedCode, "租户信息缺失")
		return
	}
	conn, ok := h.manager.Get(tenantID, "feishu")
	if !ok {
		ctx.JSON(200, gin.H{"challenge": ctx.Query("challenge")})
		return
	}
	rcv, ok := conn.(connector.Receiver)
	if !ok {
		ctx.JSON(200, gin.H{"code": -1, "msg": "feishu connector is not a Receiver"})
		return
	}
	headers := map[string]string{
		"X-Lark-Request-Timestamp": ctx.GetHeader("X-Lark-Request-Timestamp"),
		"X-Lark-Request-Nonce":     ctx.GetHeader("X-Lark-Request-Nonce"),
		"X-Lark-Signature":         ctx.GetHeader("X-Lark-Signature"),
	}
	if err := rcv.VerifySignature(headers, body); err != nil {
		if h.logger != nil {
			h.logger.Warnw("Feishu callback signature verification failed", "error", err, "remote_ip", ctx.ClientIP())
		}
		ctx.JSON(401, gin.H{"code": -1, "msg": "invalid signature"})
		return
	}
	msg, err := rcv.ParseInbound(body)
	if err != nil {
		if h.logger != nil {
			h.logger.Warnw("Feishu callback payload parsing failed", "error", err, "remote_ip", ctx.ClientIP())
		}
		ctx.JSON(400, gin.H{"code": -1, "msg": "invalid payload"})
		return
	}
	if msg.Type == "url_verification" {
		ctx.JSON(200, gin.H{"challenge": msg.Content})
		return
	}
	if h.logger != nil {
		h.logger.Infow("feishu inbound", "type", msg.Type, "user", msg.UserID, "chat", msg.ChatID)
	}
	ctx.JSON(200, gin.H{"code": 0})
}

// helpers

func maskConfig(cfg connector.Config, health map[string]connector.HealthStatus) dto.ConnectorConfigDTO {
	masked := make(map[string]string, len(cfg.Credentials))
	for k := range cfg.Credentials {
		masked[k] = "******"
	}
	healthy, checkedAt, lastErr := healthForConfig(health, cfg)
	return dto.ConnectorConfigDTO{
		Name:          cfg.Name,
		Provider:      cfg.Provider,
		Type:          string(cfg.Type),
		Enabled:       cfg.Enabled,
		Healthy:       healthy,
		Lifecycle:     connectorLifecycle(true, cfg.Enabled, healthy, lastErr),
		LastCheckedAt: checkedAt,
		LastError:     lastErr,
		CreatedAt:     cfg.CreatedAt,
		UpdatedAt:     cfg.UpdatedAt,
		Credentials:   masked,
		Settings:      cfg.Settings,
		Labels:        cfg.Labels,
	}
}

func healthForConfig(health map[string]connector.HealthStatus, cfg connector.Config) (bool, *time.Time, string) {
	key := fmt.Sprintf("%d/%s/%s", cfg.TenantID, cfg.Name, cfg.Provider)
	hs, ok := health[key]
	if !ok {
		return false, nil, ""
	}
	if hs.OK {
		return true, nonZeroTime(hs.CheckedAt), ""
	}
	return false, nonZeroTime(hs.CheckedAt), hs.Message
}

func healthForManifest(health map[string]connector.HealthStatus, tenantID int, name string) (bool, *time.Time, string) {
	for key, hs := range health {
		prefix := fmt.Sprintf("%d/%s/", tenantID, name)
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			if hs.OK {
				return true, nonZeroTime(hs.CheckedAt), ""
			}
			return false, nonZeroTime(hs.CheckedAt), hs.Message
		}
	}
	return false, nil, ""
}

// nonZeroTime 把"未检查"表示为 nil，而不是指针指向 Go 零值时间。
func nonZeroTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func connectorLifecycle(installed, enabled, healthy bool, lastErr string) string {
	switch {
	case healthy:
		return "healthy"
	case enabled && lastErr != "":
		return "unhealthy"
	case enabled:
		return "enabled"
	case installed:
		return "installed"
	default:
		return "available"
	}
}

func capToString(caps []connector.Capability) []string {
	out := make([]string, 0, len(caps))
	for _, c := range caps {
		out = append(out, string(c))
	}
	return out
}

func convertMentions(in []dto.MentionDTO) []connector.Mention {
	out := make([]connector.Mention, 0, len(in))
	for _, m := range in {
		out = append(out, connector.Mention{Type: m.Type, ID: m.ID, Name: m.Name})
	}
	return out
}

func convertActions(in []dto.ActionDTO) []connector.Action {
	out := make([]connector.Action, 0, len(in))
	for _, a := range in {
		out = append(out, connector.Action{Type: a.Type, Text: a.Text, URL: a.URL, Value: a.Value})
	}
	return out
}

func convertCard(in *dto.CardPayloadDTO) *connector.Card {
	card := &connector.Card{Variables: in.Variables}
	if in.Header != nil {
		card.Header = &connector.CardHeader{Title: in.Header.Title, Subtitle: in.Header.Subtitle, Color: in.Header.Color}
	}
	for _, s := range in.Sections {
		sec := connector.CardSection{Title: s.Title}
		for _, e := range s.Content {
			sec.Content = append(sec.Content, convertElement(e))
		}
		card.Sections = append(card.Sections, sec)
	}
	for _, e := range in.Elements {
		card.Elements = append(card.Elements, convertElement(e))
	}
	return card
}

func convertElement(e dto.CardElementDTO) connector.CardElement {
	el := connector.CardElement{Type: e.Type, Text: e.Text, ImageURL: e.ImageURL, Extras: e.Extras}
	for _, kv := range e.Fields {
		el.Fields = append(el.Fields, connector.KV{Key: kv.Key, Value: kv.Value, Short: kv.Short})
	}
	if e.Action != nil {
		el.Action = &connector.Action{Type: e.Action.Type, Text: e.Action.Text, URL: e.Action.URL, Value: e.Action.Value}
	}
	return el
}
