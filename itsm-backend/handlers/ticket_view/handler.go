// Package ticket_view — 工单视图 handler.
// 迁移自 controller/ticket_view_controller.go，保持原有 API 契约不变。
package ticket_view

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	viewService *service.TicketViewService
	logger      *zap.SugaredLogger
}

func NewHandler(viewService *service.TicketViewService, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{
		viewService: viewService,
		logger:      logger,
	}
}

// ListTicketViews GET /api/v1/tickets/views
func (h *Handler) ListTicketViews(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	if tenantID == 0 || userID == 0 {
		common.Fail(c, common.AuthFailedCode, "认证信息缺失")
		return
	}

	views, err := h.viewService.ListTicketViews(c.Request.Context(), tenantID, &userID)
	if err != nil {
		h.logger.Errorw("Failed to list ticket views", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, dto.ListTicketViewsResponse{
		Items: views,
		Total: len(views),
	})
}

// GetTicketView GET /api/v1/tickets/views/:id
func (h *Handler) GetTicketView(c *gin.Context) {
	viewID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的视图ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	view, err := h.viewService.GetTicketView(c.Request.Context(), viewID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get ticket view", "error", err, "view_id", viewID, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, view)
}

// CreateTicketView POST /api/v1/tickets/views
func (h *Handler) CreateTicketView(c *gin.Context) {
	var req dto.CreateTicketViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	req.Normalize()

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	if tenantID == 0 || userID == 0 {
		common.Fail(c, common.AuthFailedCode, "认证信息缺失")
		return
	}

	view, err := h.viewService.CreateTicketView(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to create ticket view", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, view)
}

// UpdateTicketView PUT /api/v1/tickets/views/:id
func (h *Handler) UpdateTicketView(c *gin.Context) {
	viewID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的视图ID")
		return
	}

	var req dto.UpdateTicketViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	req.Normalize()

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	if tenantID == 0 || userID == 0 {
		common.Fail(c, common.AuthFailedCode, "认证信息缺失")
		return
	}

	view, err := h.viewService.UpdateTicketView(c.Request.Context(), viewID, &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to update ticket view", "error", err, "view_id", viewID, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, view)
}

// DeleteTicketView DELETE /api/v1/tickets/views/:id
func (h *Handler) DeleteTicketView(c *gin.Context) {
	viewID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的视图ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	if tenantID == 0 || userID == 0 {
		common.Fail(c, common.AuthFailedCode, "认证信息缺失")
		return
	}

	err = h.viewService.DeleteTicketView(c.Request.Context(), viewID, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to delete ticket view", "error", err, "view_id", viewID, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}

// ShareTicketView POST /api/v1/tickets/views/:id/share
func (h *Handler) ShareTicketView(c *gin.Context) {
	viewID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的视图ID")
		return
	}

	var req dto.ShareTicketViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	req.Normalize()

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	if tenantID == 0 || userID == 0 {
		common.Fail(c, common.AuthFailedCode, "认证信息缺失")
		return
	}

	err = h.viewService.ShareTicketView(c.Request.Context(), viewID, &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to share ticket view", "error", err, "view_id", viewID, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}
