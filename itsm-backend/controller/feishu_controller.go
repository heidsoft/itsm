package controller

import (
	"net/http"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/connector"
	feishuConn "itsm-backend/connector/builtin/feishu"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FeishuController struct {
	connectorManager *connector.Manager
	ticketService    *service.TicketService
	logger           *zap.SugaredLogger
}

func NewFeishuController(connectorManager *connector.Manager, ticketService *service.TicketService, logger *zap.SugaredLogger) *FeishuController {
	return &FeishuController{
		connectorManager: connectorManager,
		ticketService:    ticketService,
		logger:           logger,
	}
}

// getFeishuConnector 获取当前租户的飞书连接器
func (c *FeishuController) getFeishuConnector(ctx *gin.Context) (*feishuConn.Feishu, bool) {
	tenantID := ctx.GetInt("tenant_id")
	conn, ok := c.connectorManager.Get(tenantID, "feishu")
	if !ok {
		return nil, false
	}
	fc, ok := conn.(*feishuConn.Feishu)
	return fc, ok
}

// GetOAuthAuthURL returns the Feishu OAuth authorization URL
// TODO: GetOAuthAuthURL 在 Client 私有字段上，需要 Feishu 封装后暴露
func (c *FeishuController) GetOAuthAuthURL(ctx *gin.Context) {
	_, ok := c.getFeishuConnector(ctx)
	if !ok {
		common.Fail(ctx, 500, "Feishu connector not configured")
		return
	}
	common.Fail(ctx, http.StatusNotImplemented, "飞书 OAuth URL 功能待实现")
}

// OAuthCallback handles the Feishu OAuth callback
// TODO: ExchangeOAuthCode 在 Client 私有字段上，需要 Feishu 封装后暴露
func (c *FeishuController) OAuthCallback(ctx *gin.Context) {
	code := ctx.Query("code")
	if code == "" {
		common.Fail(ctx, 400, "code is required")
		return
	}
	_, ok := c.getFeishuConnector(ctx)
	if !ok {
		common.Fail(ctx, 500, "Feishu connector not configured")
		return
	}
	common.Fail(ctx, http.StatusNotImplemented, "飞书 OAuth 回调功能待实现")
}

// SyncTicketToFeishu syncs a ticket to Feishu as a task
// TODO: ticketService.GetTicketByID 和 ent.Tx 注入待完善
func (c *FeishuController) SyncTicketToFeishu(ctx *gin.Context) {
	ticketIDStr := ctx.Param("ticket_id")
	_, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		common.Fail(ctx, 400, "Invalid ticket ID")
		return
	}
	_, ok := c.getFeishuConnector(ctx)
	if !ok {
		common.Fail(ctx, 500, "Feishu connector not configured")
		return
	}
	common.Fail(ctx, http.StatusNotImplemented, "飞书工单同步功能待完善")
}

// Webhook handles Feishu webhook events
func (c *FeishuController) Webhook(ctx *gin.Context) {
	fc, ok := c.getFeishuConnector(ctx)
	if !ok {
		common.Fail(ctx, 500, "Feishu connector not configured")
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
		c.logger.Errorw("Invalid Feishu webhook signature", "err", err)
		common.Fail(ctx, 403, "Invalid signature")
		return
	}

	msg, err := fc.ParseInbound(rawData)
	if err != nil {
		c.logger.Errorw("Failed to parse Feishu webhook event", "err", err)
		common.Fail(ctx, 400, "Invalid event payload")
		return
	}

	if msg.Type == "url_verification" {
		ctx.JSON(http.StatusOK, gin.H{
			"challenge": msg.Content,
		})
		return
	}

	if msg.Type == "task_event" {
		eventType := msg.Content
		taskData, ok := msg.Extras["task_data"].(map[string]interface{})
		if !ok {
			common.Fail(ctx, 400, "Invalid task event data")
			return
		}

		err = fc.HandleTaskEvent(ctx, eventType, taskData)
		if err != nil {
			c.logger.Errorw("Failed to handle Feishu task event", "event_type", eventType, "err", err)
			common.Fail(ctx, 500, "Failed to handle event")
			return
		}
	}

	// TODO: connectorManager.Router() 待实现后恢复 Dispatch 逻辑
	common.Success(ctx, nil)
}
