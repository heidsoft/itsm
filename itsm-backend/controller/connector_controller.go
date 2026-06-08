package controller

import (
	"context"
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/connector"
	"itsm-backend/connector/marketplace"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ConnectorController 连接器 HTTP API
// 路由：
//   GET    /api/v1/connectors              -> 列出已注册连接器（市场视图）
//   GET    /api/v1/connectors/configs      -> 列出当前租户已配置的连接器实例
//   POST   /api/v1/connectors/configs      -> 创建/更新一个连接器配置（provision）
//   DELETE /api/v1/connectors/configs/:name -> 停用并移除一个连接器实例
//   POST   /api/v1/connectors/:name/send   -> 通过指定连接器发消息
//   POST   /api/v1/connectors/:name/test   -> 发送一条测试消息
//   GET    /api/v1/connectors/health       -> 所有实例的健康检查
//   POST   /api/v1/connectors/feishu/callback -> 飞书事件回调入口（如果安装了 feishu）
type ConnectorController struct {
	manager    *connector.Manager
	market     *marketplace.Market // optional
	registry   *connector.Registry
	logger     *zap.SugaredLogger
}

func NewConnectorController(mgr *connector.Manager, reg *connector.Registry, mkt *marketplace.Market, logger *zap.SugaredLogger) *ConnectorController {
	return &ConnectorController{manager: mgr, market: mkt, registry: reg, logger: logger}
}

// ListMarket 列出市场中所有可用连接器
func (c *ConnectorController) ListMarket(ctx *gin.Context) {
	reg := c.registry
	if reg == nil {
		reg = connector.Default()
	}
	mfs := reg.List()
	tenantID := ctx.GetInt("tenant_id")
	configs := c.manager.ListByTenant(tenantID)
	installed := make(map[string]bool, len(configs))
	for _, cfg := range configs {
		installed[cfg.Name] = true
	}
	out := make([]dto.ConnectorManifestDTO, 0, len(mfs))
	for _, m := range mfs {
		out = append(out, dto.ConnectorManifestDTO{
			Name:         m.Name,
			Version:      m.Version,
			Title:        m.Title,
			Provider:     m.Provider,
			Type:         string(m.Type),
			Description:  m.Description,
			Author:       m.Author,
			Homepage:     m.Homepage,
			IconURL:      m.IconURL,
			Capabilities: capToString(m.Capabilities),
			Tags:         m.Tags,
			MinITSMVer:   m.MinITSMVer,
			Local:        true,
			Installed:    installed[m.Name],
			Category:     string(m.Type),
		})
	}
	common.Success(ctx, gin.H{"items": out, "total": len(out)})
}

// ListConfigs 列出当前租户已配置的连接器实例（凭据脱敏）
func (c *ConnectorController) ListConfigs(ctx *gin.Context) {
	tenantID := ctx.GetInt("tenant_id")
	cfgs := c.manager.ListByTenant(tenantID)
	out := make([]dto.ConnectorConfigDTO, 0, len(cfgs))
	for _, cfg := range cfgs {
		out = append(out, maskConfig(cfg))
	}
	common.Success(ctx, gin.H{"items": out, "total": len(out)})
}

// Provision 创建/更新一个连接器实例
func (c *ConnectorController) Provision(ctx *gin.Context) {
	var req dto.ProvisionConnectorRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, err.Error())
		return
	}
	tenantID := ctx.GetInt("tenant_id")
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
	if err := c.manager.Provision(ctx.Request.Context(), cfg); err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(ctx, maskConfig(cfg))
}

// Revoke 停用并移除一个连接器实例
func (c *ConnectorController) Revoke(ctx *gin.Context) {
	name := ctx.Param("name")
	tenantID := ctx.GetInt("tenant_id")
	c.manager.Revoke(connector.Config{TenantID: tenantID, Name: name})
	common.Success(ctx, gin.H{"name": name, "revoked": true})
}

// Send 通过指定连接器发消息
func (c *ConnectorController) Send(ctx *gin.Context) {
	name := ctx.Param("name")
	var req dto.SendConnectorMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, err.Error())
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
	if err := c.manager.Send(ctx.Request.Context(), tenantID, name, msg); err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(ctx, gin.H{"name": name, "channel": req.Channel, "sent": true})
}

// Test 发送一条简单测试消息（向配置里 debug_channel 投递）
func (c *ConnectorController) Test(ctx *gin.Context) {
	name := ctx.Param("name")
	tenantID := ctx.GetInt("tenant_id")
	// 找该连接器的配置
	var channel string
	for _, cfg := range c.manager.ListByTenant(tenantID) {
		if cfg.Name == name {
			if ch, ok := cfg.Settings["debug_channel"].(string); ok {
				channel = ch
			}
		}
	}
	if channel == "" {
		common.Fail(ctx, common.ParamErrorCode, "settings.debug_channel not configured for "+name)
		return
	}
	if err := c.manager.Send(ctx.Request.Context(), tenantID, name, &connector.Message{
		Channel: channel,
		Type:    "text",
		Title:   "ITSM 连接器测试",
		Content: "这是一条来自 ITSM 的测试消息。\n时间: " + time.Now().Format(time.RFC3339),
	}); err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(ctx, gin.H{"name": name, "channel": channel, "sent": true})
}

// Health 所有运行实例健康检查
func (c *ConnectorController) Health(ctx *gin.Context) {
	res := c.manager.HealthCheckAll(context.Background())
	out := make(map[string]dto.ConnectorHealthDTO, len(res))
	for k, v := range res {
		out[k] = dto.ConnectorHealthDTO{
			OK:        v.OK,
			LatencyMs: v.LatencyMs,
			Message:   v.Message,
			CheckedAt: v.CheckedAt,
			Extra:     v.Extra,
		}
	}
	common.Success(ctx, out)
}

// FeishuCallback 飞书事件回调入口
// 注意：本方法假定 manager 中已经配置了 feishu 连接器；
// 实际签名校验和负载解析由该连接器自身完成。
func (c *ConnectorController) FeishuCallback(ctx *gin.Context) {
	body, _ := ctx.GetRawData()
	tenantID, _ := strconv.Atoi(ctx.Query("tenant_id"))
	if tenantID == 0 {
		tenantID = ctx.GetInt("tenant_id")
	}
	conn, ok := c.manager.Get(tenantID, "feishu")
	if !ok {
		// 飞书 URL Verification 仍然要回应，否则平台会反复重试
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
		ctx.JSON(401, gin.H{"code": -1, "msg": err.Error()})
		return
	}
	msg, err := rcv.ParseInbound(body)
	if err != nil {
		ctx.JSON(400, gin.H{"code": -1, "msg": err.Error()})
		return
	}
	if msg.Type == "url_verification" {
		ctx.JSON(200, gin.H{"challenge": msg.Content})
		return
	}
	// 入站消息进入 Router 派发
	if c.logger != nil {
		c.logger.Infow("feishu inbound", "type", msg.Type, "user", msg.UserID, "chat", msg.ChatID)
	}
	ctx.JSON(200, gin.H{"code": 0})
}

// helpers

func maskConfig(cfg connector.Config) dto.ConnectorConfigDTO {
	masked := make(map[string]string, len(cfg.Credentials))
	for k := range cfg.Credentials {
		masked[k] = "******"
	}
	return dto.ConnectorConfigDTO{
		Name:        cfg.Name,
		Provider:    cfg.Provider,
		Type:        string(cfg.Type),
		Enabled:     cfg.Enabled,
		Credentials: masked,
		Settings:    cfg.Settings,
		Labels:      cfg.Labels,
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
