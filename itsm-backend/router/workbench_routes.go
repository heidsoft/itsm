package router

import (
	workbenchHandler "itsm-backend/handlers/workbench"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupWorkbenchRoutes registers workbench routes.
//
// 显式 RequirePermission 防止 RBAC 中间件族级 fallback 误命中「workbench:*」默认权限
// 扩展后全员可达——统一工作台当前业务语义归属 dashboard:read 仪表盘读权限。
func SetupWorkbenchRoutes(tenant *gin.RouterGroup, h *workbenchHandler.Handler) {
	workbench := tenant.Group("/workbench")
	{
		workbench.GET("", middleware.RequirePermission("dashboard", "read"), h.Query)
	}
}
