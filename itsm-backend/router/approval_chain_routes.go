package router

import (
	"github.com/gin-gonic/gin"

	approvalChainHandler "itsm-backend/handlers/approval_chain"
	escalationMatrixHandler "itsm-backend/handlers/escalation_matrix"
	"itsm-backend/middleware"
)

// SetupApprovalChainRoutes 注册审批链域路由。
func SetupApprovalChainRoutes(tenant *gin.RouterGroup, h *approvalChainHandler.Handler, escalationMatrixHandler *escalationMatrixHandler.Handler) {
	approvalChains := tenant.Group("/approval-chains")
	{
		approvalChains.GET("", middleware.RequirePermission("approval", "read"), h.ListChains)
		approvalChains.GET("/stats", middleware.RequirePermission("approval", "read"), h.GetStats)
		approvalChains.POST("", middleware.RequirePermission("approval", "write"), h.CreateChain)
		approvalChains.GET("/:id", middleware.RequirePermission("approval", "read"), h.GetChain)
		approvalChains.PUT("/:id", middleware.RequirePermission("approval", "write"), h.UpdateChain)
		approvalChains.DELETE("/:id", middleware.RequirePermission("approval", "write"), h.DeleteChain)
	}
	// ==================== Escalation Matrix ====================
	if escalationMatrixHandler != nil {
		escalationMatrixHandler.RegisterRoutes(tenant)
	}
}
