// Package escalation_matrix 是升级矩阵域的 HTTP handler 层（域切片架构）。
// 业务逻辑仍由 service.EscalationMatrixService 承载，本包只做参数解析与响应封装。
package escalation_matrix

import (
	"itsm-backend/common"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 升级矩阵 HTTP handler
type Handler struct {
	logger                  *zap.SugaredLogger
	escalationMatrixService *service.EscalationMatrixService
}

// NewHandler 创建升级矩阵 handler 实例
func NewHandler(logger *zap.SugaredLogger, escalationMatrixService *service.EscalationMatrixService) *Handler {
	return &Handler{
		logger:                  logger,
		escalationMatrixService: escalationMatrixService,
	}
}

// tenantID 提取租户上下文
func tenantID(c *gin.Context) (int, bool) {
	tid, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return 0, false
	}
	id, ok := tid.(int)
	if !ok || id == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return 0, false
	}
	return id, true
}

// GetMatrix 获取当前租户的升级矩阵
func (h *Handler) GetMatrix(ctx *gin.Context) {
	tid, ok := tenantID(ctx)
	if !ok {
		return
	}
	matrix := h.escalationMatrixService.GetMatrix(tid)
	common.Success(ctx, matrix)
}

// SetMatrix 设置当前租户的升级矩阵
func (h *Handler) SetMatrix(ctx *gin.Context) {
	tid, ok := tenantID(ctx)
	if !ok {
		return
	}
	var matrix service.EscalationMatrix
	if err := ctx.ShouldBindJSON(&matrix); err != nil {
		common.ParamErrorWithErr(ctx, err, "请求参数错误")
		return
	}
	// 落库失败必须显式报错：SetMatrix 内部是「先落库成功才更新缓存」，
	// 返回 nil 意味着配置已持久化。若这里吞错，接口会返回成功但重启即丢配置。
	if err := h.escalationMatrixService.SetMatrix(tid, matrix); err != nil {
		common.FailWithErr(ctx, err, "保存升级矩阵失败")
		return
	}
	common.Success(ctx, nil)
}

// InvalidateCache 清除升级矩阵缓存
func (h *Handler) InvalidateCache(ctx *gin.Context) {
	tid, ok := tenantID(ctx)
	if !ok {
		return
	}
	h.escalationMatrixService.InvalidateCache(tid)
	common.Success(ctx, nil)
}

// RegisterRoutes 注册路由（兼容旧接口）
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// 升级矩阵属 SLA 配置面
	matrixGrp := r.Group("/escalation-matrices")
	{
		matrixGrp.GET("", middleware.RequirePermission("sla", "read"), h.GetMatrix)
		matrixGrp.PUT("", middleware.RequirePermission("sla", "write"), h.SetMatrix)
		matrixGrp.POST("/invalidate-cache", middleware.RequirePermission("sla", "write"), h.InvalidateCache)
	}
}
