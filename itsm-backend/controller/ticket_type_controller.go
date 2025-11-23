package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TicketTypeController struct {
	ticketTypeService *service.TicketTypeService
	logger            *zap.SugaredLogger
}

func NewTicketTypeController(ticketTypeService *service.TicketTypeService, logger *zap.SugaredLogger) *TicketTypeController {
	return &TicketTypeController{
		ticketTypeService: ticketTypeService,
		logger:            logger,
	}
}

// CreateTicketType 创建工单类型
// @Summary 创建工单类型
// @Description 创建新的工单类型，包括自定义字段、审批流程等配置
// @Tags 工单类型
// @Accept json
// @Produce json
// @Param request body dto.CreateTicketTypeRequest true "创建工单类型请求"
// @Success 200 {object} common.Response{data=dto.TicketTypeDefinition}
// @Router /api/v1/ticket-types [post]
func (tc *TicketTypeController) CreateTicketType(c *gin.Context) {
	var req dto.CreateTicketTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	ticketType, err := tc.ticketTypeService.CreateTicketType(c.Request.Context(), &req, tenantID, userID)
	if err != nil {
		tc.logger.Errorw("Failed to create ticket type", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticketType)
}

// UpdateTicketType 更新工单类型
// @Summary 更新工单类型
// @Description 更新工单类型配置
// @Tags 工单类型
// @Accept json
// @Produce json
// @Param id path int true "工单类型ID"
// @Param request body dto.UpdateTicketTypeRequest true "更新工单类型请求"
// @Success 200 {object} common.Response{data=dto.TicketTypeDefinition}
// @Router /api/v1/ticket-types/:id [put]
func (tc *TicketTypeController) UpdateTicketType(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单类型ID")
		return
	}

	var req dto.UpdateTicketTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	ticketType, err := tc.ticketTypeService.UpdateTicketType(c.Request.Context(), id, &req, tenantID, userID)
	if err != nil {
		tc.logger.Errorw("Failed to update ticket type", "error", err, "id", id, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticketType)
}

// GetTicketType 获取工单类型详情
// @Summary 获取工单类型详情
// @Description 根据ID获取工单类型的详细信息
// @Tags 工单类型
// @Accept json
// @Produce json
// @Param id path int true "工单类型ID"
// @Success 200 {object} common.Response{data=dto.TicketTypeDefinition}
// @Router /api/v1/ticket-types/:id [get]
func (tc *TicketTypeController) GetTicketType(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单类型ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	ticketType, err := tc.ticketTypeService.GetTicketType(c.Request.Context(), id, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to get ticket type", "error", err, "id", id, "tenant_id", tenantID)
		common.Fail(c, common.NotFoundCode, "工单类型不存在")
		return
	}

	common.Success(c, ticketType)
}

// ListTicketTypes 获取工单类型列表
// @Summary 获取工单类型列表
// @Description 获取工单类型列表，支持分页和筛选
// @Tags 工单类型
// @Accept json
// @Produce json
// @Param status query string false "状态筛选"
// @Param keyword query string false "关键词搜索"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} common.Response{data=dto.TicketTypeListResponse}
// @Router /api/v1/ticket-types [get]
func (tc *TicketTypeController) ListTicketTypes(c *gin.Context) {
	var req dto.ListTicketTypesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 设置默认值
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	tenantID := c.GetInt("tenant_id")

	response, err := tc.ticketTypeService.ListTicketTypes(c.Request.Context(), &req, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to list ticket types", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, response)
}

// DeleteTicketType 删除工单类型
// @Summary 删除工单类型
// @Description 删除指定的工单类型
// @Tags 工单类型
// @Accept json
// @Produce json
// @Param id path int true "工单类型ID"
// @Success 200 {object} common.Response{data=map[string]string}
// @Router /api/v1/ticket-types/:id [delete]
func (tc *TicketTypeController) DeleteTicketType(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单类型ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	err = tc.ticketTypeService.DeleteTicketType(c.Request.Context(), id, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to delete ticket type", "error", err, "id", id, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "工单类型删除成功"})
}

