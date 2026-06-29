package controller

import (
	"context"
	"net/http"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/connector"
	"itsm-backend/connector/builtin/feishu"
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

// GetOAuthAuthURL returns the Feishu OAuth authorization URL
// @Summary Get Feishu OAuth URL
// @Description Get the authorization URL for Feishu OAuth
// @Tags Feishu
// @Accept json
// @Produce json
// @Param redirect_uri query string true "Redirect URI after OAuth"
// @Param state query string false "State parameter for CSRF protection"
// @Success 200 {object} common.Response{data=map[string]string}
// @Router /api/v1/feishu/oauth/auth-url [get]
func (c *FeishuController) GetOAuthAuthURL(ctx *gin.Context) {
	redirectURI := ctx.Query("redirect_uri")
	if redirectURI == "" {
		common.Fail(ctx, 400, "redirect_uri is required")
		return
	}
	state := ctx.Query("state")
	if state == "" {
		state = common.GenerateRandomString(32)
	}

	// Get Feishu connector instance for current tenant
	tenantID := ctx.GetInt("tenant_id")
	conn, err := c.connectorManager.GetConnector(ctx, tenantID, "feishu")
	if err != nil {
		common.Fail(ctx, 500, "Feishu connector not configured")
		return
	}

	feishuConn, ok := conn.(*feishu.Feishu)
	if !ok {
		common.Fail(ctx, 500, "Invalid Feishu connector instance")
		return
	}

	authURL := feishuConn.GetOAuthAuthURL(redirectURI, state)
	common.Success(ctx, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

// OAuthCallback handles the Feishu OAuth callback
// @Summary Feishu OAuth callback
// @Description Handle the OAuth callback from Feishu
// @Tags Feishu
// @Accept json
// @Produce json
// @Param code query string true "Authorization code"
// @Param state query string true "State parameter"
// @Success 200 {object} common.Response{data=map[string]interface{}}
// @Router /api/v1/feishu/oauth/callback [get]
func (c *FeishuController) OAuthCallback(ctx *gin.Context) {
	code := ctx.Query("code")
	if code == "" {
		common.Fail(ctx, 400, "code is required")
		return
	}
	state := ctx.Query("state")
	if state == "" {
		common.Fail(ctx, 400, "state is required")
	}

	// Get Feishu connector instance for current tenant
	tenantID := ctx.GetInt("tenant_id")
	conn, err := c.connectorManager.GetConnector(ctx, tenantID, "feishu")
	if err != nil {
		common.Fail(ctx, 500, "Feishu connector not configured")
		return
	}

	feishuConn, ok := conn.(*feishu.Feishu)
	if !ok {
		common.Fail(ctx, 500, "Invalid Feishu connector instance")
		return
	}

	// Exchange code for token
	tokenResp, err := feishuConn.ExchangeOAuthCode(ctx, code)
	if err != nil {
		c.logger.Errorw("Failed to exchange OAuth code", "err", err)
		common.Fail(ctx, 500, "Failed to exchange authorization code")
		return
	}

	// TODO: Store the token in the connector configuration or user table
	// For now, just return it
	common.Success(ctx, gin.H{
		"access_token":  tokenResp.AccessToken,
		"refresh_token": tokenResp.RefreshToken,
		"expires_in":    tokenResp.ExpiresIn,
		"user_id":       tokenResp.UserID,
		"open_id":       tokenResp.OpenID,
	})
}

// SyncTicketToFeishu syncs a ticket to Feishu as a task
// @Summary Sync ticket to Feishu
// @Description Sync an ITSM ticket to Feishu as a task
// @Tags Feishu
// @Accept json
// @Produce json
// @Param ticket_id path int true "Ticket ID"
// @Success 200 {object} common.Response{data=feishu.FeishuTask}
// @Router /api/v1/feishu/sync/ticket/{ticket_id} [post]
func (c *FeishuController) SyncTicketToFeishu(ctx *gin.Context) {
	ticketIDStr := ctx.Param("ticket_id")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil || ticketID <= 0 {
		common.Fail(ctx, 400, "Invalid ticket ID")
		return
	}

	// Get ticket
	ticket, err := c.ticketService.GetTicketByID(ctx, ticketID)
	if err != nil {
		c.logger.Errorw("Failed to get ticket", "ticket_id", ticketID, "err", err)
		common.Fail(ctx, 404, "Ticket not found")
		return
	}

	// Get Feishu connector instance for current tenant
	tenantID := ctx.GetInt("tenant_id")
	conn, err := c.connectorManager.GetConnector(ctx, tenantID, "feishu")
	if err != nil {
		common.Fail(ctx, 500, "Feishu connector not configured")
		return
	}

	feishuConn, ok := conn.(*feishu.Feishu)
	if !ok {
		common.Fail(ctx, 500, "Invalid Feishu connector instance")
		return
	}

	// Start transaction
	tx, err := c.ticketService.Db().BeginTx(ctx, nil)
	if err != nil {
		c.logger.Errorw("Failed to start transaction", "err", err)
		common.Fail(ctx, 500, "Internal server error")
		return
	}
	defer tx.Rollback()

	// Sync ticket to Feishu
	task, err := feishuConn.SyncTicketToFeishu(ctx, tx, ticket)
	if err != nil {
		c.logger.Errorw("Failed to sync ticket to Feishu", "ticket_id", ticketID, "err", err)
		common.Fail(ctx, 500, "Failed to sync ticket: "+err.Error())
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		c.logger.Errorw("Failed to commit transaction", "err", err)
		common.Fail(ctx, 500, "Internal server error")
		return
	}

	common.Success(ctx, task)
}

// Webhook handles Feishu webhook events
// @Summary Feishu webhook
// @Description Handle webhook events from Feishu
// @Tags Feishu
// @Accept json
// @Produce json
// @Success 200 {object} common.Response
// @Router /api/v1/feishu/webhook [post]
func (c *FeishuController) Webhook(ctx *gin.Context) {
	// Get Feishu connector instance for current tenant
	tenantID := ctx.GetInt("tenant_id")
	conn, err := c.connectorManager.GetConnector(ctx, tenantID, "feishu")
	if err != nil {
		common.Fail(ctx, 500, "Feishu connector not configured")
		return
	}

	feishuConn, ok := conn.(*feishu.Feishu)
	if !ok {
		common.Fail(ctx, 500, "Invalid Feishu connector instance")
		return
	}

	// Verify signature
	err = feishuConn.VerifySignature(ctx.Request.Header, ctx.GetRawData())
	if err != nil {
		c.logger.Errorw("Invalid Feishu webhook signature", "err", err)
		common.Fail(ctx, 403, "Invalid signature")
		return
	}

	// Parse inbound message
	msg, err := feishuConn.ParseInbound(ctx.GetRawData())
	if err != nil {
		c.logger.Errorw("Failed to parse Feishu webhook event", "err", err)
		common.Fail(ctx, 400, "Invalid event payload")
		return
	}

	// Handle URL verification (return challenge)
	if msg.Type == "url_verification" {
		ctx.JSON(http.StatusOK, gin.H{
			"challenge": msg.Content,
		})
		return
	}

	// Handle task events
	if msg.Type == "task_event" {
		eventType := msg.Content
		taskData, ok := msg.Extras["task_data"].(map[string]interface{})
		if !ok {
			common.Fail(ctx, 400, "Invalid task event data")
			return
		}

		// Start transaction
		tx, err := c.ticketService.Db().BeginTx(ctx, nil)
		if err != nil {
			c.logger.Errorw("Failed to start transaction", "err", err)
			common.Fail(ctx, 500, "Internal server error")
			return
		}
		defer tx.Rollback()

		// Handle the task event
		err = feishuConn.HandleTaskEvent(ctx, eventType, taskData)
		if err != nil {
			c.logger.Errorw("Failed to handle Feishu task event", "event_type", eventType, "err", err)
			common.Fail(ctx, 500, "Failed to handle event")
			return
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			c.logger.Errorw("Failed to commit transaction", "err", err)
			common.Fail(ctx, 500, "Internal server error")
			return
		}
	}

	// Dispatch message to connector router for other handlers
	c.connectorManager.Router().Dispatch(ctx, msg)

	common.Success(ctx, nil)
}
