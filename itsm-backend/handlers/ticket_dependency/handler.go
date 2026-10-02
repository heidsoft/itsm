// Package ticket_dependency — 工单依赖关系 handler.
// 迁移自 controller/ticket_dependency_controller.go，保持原有 API 契约不变。
package ticket_dependency

import (
	"errors"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	dependencyService *service.TicketDependencyService
	logger            *zap.SugaredLogger
}

func NewHandler(dependencyService *service.TicketDependencyService, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{
		dependencyService: dependencyService,
		logger:            logger,
	}
}

// AnalyzeDependencyImpact GET /api/v1/tickets/:id/dependencies?action=close|delete|change_status&newStatus=...
//
// 这里原先用 ShouldBindJSON 绑定一个 action 必填的结构体，而路由只注册了 GET：
// 浏览器不带请求体，任何调用都固定得到 400「请求参数错误」，这个入口从未可用过。
// 影响分析的输入是过滤/枚举语义，按契约归属查询参数（见 dto.RelationImpactAnalysisRequest）。
func (h *Handler) AnalyzeDependencyImpact(ctx *gin.Context) {
	ticketIDStr := ctx.Param("id")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.RelationImpactAnalysisRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		h.logger.Warnw("Invalid dependency impact query", "error", err, "ticket_id", ticketID)
		common.Fail(ctx, common.ParamErrorCode, "action 必填，且只能是 close、delete、change_status 之一")
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}
	tid, ok := tenantID.(int)
	if !ok {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	impact, err := h.dependencyService.AnalyzeDependencyImpact(
		ctx.Request.Context(),
		ticketID,
		req.Action,
		req.NewStatus,
		tid,
	)
	if err != nil {
		// 跨租户与真实不存在都按「不存在」处理，不确认对方资源存在；原始错误只进日志。
		if errors.Is(err, service.ErrDependencyTicketNotFound) {
			h.logger.Warnw("Dependency impact target not found", "ticket_id", ticketID, "tenant_id", tid, "error", err)
			common.Fail(ctx, common.NotFoundCode, "工单不存在或无权访问")
			return
		}
		h.logger.Errorw("Failed to analyze dependency impact", "error", err, "ticket_id", ticketID, "tenant_id", tid)
		common.Fail(ctx, common.InternalErrorCode, "分析依赖影响失败")
		return
	}

	common.Success(ctx, impact)
}
