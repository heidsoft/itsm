// Package ticket_dependency — 工单依赖关系 handler.
// 迁移自 controller/ticket_dependency_controller.go，保持原有 API 契约不变。
package ticket_dependency

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	dependencyService *service.TicketDependencyService
	logger          *zap.SugaredLogger
}

func NewHandler(dependencyService *service.TicketDependencyService, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{
		dependencyService: dependencyService,
		logger:          logger,
	}
}

// AnalyzeDependencyImpact GET /api/v1/tickets/:id/dependencies
func (h *Handler) AnalyzeDependencyImpact(ctx *gin.Context) {
	ticketIDStr := ctx.Param("id")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req struct {
		Action    string  `json:"action" binding:"required,oneof=close delete change_status"`
		NewStatus *string `json:"newStatus,omitempty"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误")
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	impact, err := h.dependencyService.AnalyzeDependencyImpact(
		ctx.Request.Context(),
		ticketID,
		req.Action,
		req.NewStatus,
		tenantID.(int),
	)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "分析依赖影响失败")
		return
	}

	common.Success(ctx, impact)
}
