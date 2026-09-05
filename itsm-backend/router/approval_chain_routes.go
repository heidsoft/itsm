package router

import (
	"github.com/gin-gonic/gin"

	approvalChainHandler "itsm-backend/handlers/approval_chain"
	escalationMatrixHandler "itsm-backend/handlers/escalation_matrix"
	"itsm-backend/middleware"
)

// SetupApprovalChainRoutes 注册审批链域路由。
// 从 router.go 集中注册块抽取，行为与原内联等价（含 EscalationMatrix 自注册委托）。
func SetupApprovalChainRoutes(tenant *gin.RouterGroup, h *approvalChainHandler.Handler, escalationMatrixHandler *escalationMatrixHandler.Handler) {
	approvalChains := tenant.Group("/approval-chains")
	{
		approvalChains.GET("", middleware.RequirePermission("approval", "read"), h.ListChains)
		approvalChains.GET("/stats", middleware.RequirePermission("approval", "read"), h.GetStats)
		approvalChains.POST("", middleware.RequirePermission("approval", "create"), h.CreateChain)
		approvalChains.GET("/:id", middleware.RequirePermission("approval", "read"), h.GetChain)
		approvalChains.PUT("/:id", middleware.RequirePermission("approval", "update"), h.UpdateChain)
		approvalChains.DELETE("/:id", middleware.RequirePermission("approval", "delete"), h.DeleteChain)
	}
	// ==================== Escalation Matrix ====================
	if escalationMatrixHandler != nil {
		escalationMatrixHandler.RegisterRoutes(tenant)
	}
}
