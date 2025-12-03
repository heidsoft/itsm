package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TicketViewController struct {
	viewService *service.TicketViewService
	logger      *zap.SugaredLogger
}

func NewTicketViewController(viewService *service.TicketViewService, logger *zap.SugaredLogger) *TicketViewController {
	return &TicketViewController{
		viewService: viewService,
		logger:      logger,
	}
}

// ListTicketViews 获取工单视图列表
// GET /api/v1/tickets/views
func (tvc *TicketViewController) ListTicketViews(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	views, err := tvc.viewService.ListTicketViews(c.Request.Context(), tenantID, &userID)
	if err != nil {
		tvc.logger.Errorw("Failed to list ticket views", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dto.ListTicketViewsResponse{
		Views: views,
		Total: len(views),
	})
}

// GetTicketView 获取工单视图详情
// GET /api/v1/tickets/views/:id
func (tvc *TicketViewController) GetTicketView(c *gin.Context) {
	viewID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的视图ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	view, err := tvc.viewService.GetTicketView(c.Request.Context(), viewID, tenantID)
	if err != nil {
		tvc.logger.Errorw("Failed to get ticket view", "error", err, "view_id", viewID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, view)
}

// CreateTicketView 创建工单视图
// POST /api/v1/tickets/views
func (tvc *TicketViewController) CreateTicketView(c *gin.Context) {
	var req dto.CreateTicketViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	view, err := tvc.viewService.CreateTicketView(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		tvc.logger.Errorw("Failed to create ticket view", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, view)
}

// UpdateTicketView 更新工单视图
// PUT /api/v1/tickets/views/:id
func (tvc *TicketViewController) UpdateTicketView(c *gin.Context) {
	viewID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的视图ID")
		return
	}

	var req dto.UpdateTicketViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	view, err := tvc.viewService.UpdateTicketView(c.Request.Context(), viewID, &req, userID, tenantID)
	if err != nil {
		tvc.logger.Errorw("Failed to update ticket view", "error", err, "view_id", viewID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, view)
}

// DeleteTicketView 删除工单视图
// DELETE /api/v1/tickets/views/:id
func (tvc *TicketViewController) DeleteTicketView(c *gin.Context) {
	viewID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的视图ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	err = tvc.viewService.DeleteTicketView(c.Request.Context(), viewID, userID, tenantID)
	if err != nil {
		tvc.logger.Errorw("Failed to delete ticket view", "error", err, "view_id", viewID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, nil)
}

// ShareTicketView 共享工单视图
// POST /api/v1/tickets/views/:id/share
func (tvc *TicketViewController) ShareTicketView(c *gin.Context) {
	viewID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的视图ID")
		return
	}

	var req dto.ShareTicketViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	err = tvc.viewService.ShareTicketView(c.Request.Context(), viewID, &req, userID, tenantID)
	if err != nil {
		tvc.logger.Errorw("Failed to share ticket view", "error", err, "view_id", viewID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, nil)
}

