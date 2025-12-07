package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// TicketDependencyController 工单依赖关系控制器
type TicketDependencyController struct {
	dependencyService *service.TicketDependencyService
}

// NewTicketDependencyController 创建工单依赖关系控制器实例
func NewTicketDependencyController(dependencyService *service.TicketDependencyService) *TicketDependencyController {
	return &TicketDependencyController{
		dependencyService: dependencyService,
	}
}

// AnalyzeDependencyImpact 分析依赖关系影响
func (c *TicketDependencyController) AnalyzeDependencyImpact(ctx *gin.Context) {
	ticketIDStr := ctx.Param("id")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的工单ID: "+err.Error())
		return
	}

	var req struct {
		Action    string  `json:"action" binding:"required,oneof=close delete change_status" example:"close"`
		NewStatus *string `json:"new_status,omitempty" example:"closed"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	impact, err := c.dependencyService.AnalyzeDependencyImpact(
		ctx.Request.Context(),
		ticketID,
		req.Action,
		req.NewStatus,
		tenantID.(int),
	)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "分析依赖影响失败: "+err.Error())
		return
	}

	common.Success(ctx, impact)
}

