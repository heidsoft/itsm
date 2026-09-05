package router

import (
	"github.com/gin-gonic/gin"

	"itsm-backend/handlers/provisioning"
	"itsm-backend/handlers/service_request"
	"itsm-backend/middleware"
)

// SetupServiceRequestRoutes 注册服务请求域路由。
// 从 router.go 的集中注册块抽取而来，路由路径/方法/中间件与抽取前逐行一致。
// 注意：provisioning 任务子路由随同搬移，保持 /service-requests 与
// /provisioning-tasks 两条路径的注册顺序不变。
func SetupServiceRequestRoutes(tenant *gin.RouterGroup, h *service_request.Handler, provisioningHandler *provisioning.Handler) {
	sr := tenant.Group("/service-requests")
	{
		sr.POST("", middleware.RequirePermission("service_request", "write"), h.Create)
		sr.GET("", middleware.RequirePermission("service_request", "read"), h.List)
		sr.GET("/me", middleware.RequirePermission("service_request", "read"), h.List)
		sr.GET("/approvals/pending", middleware.RequirePermission("service_request", "read"), h.ListPending)
		sr.GET("/:id", middleware.RequirePermission("service_request", "read"), h.Get)
		sr.GET("/:id/approvals", middleware.RequirePermission("service_request", "read"), h.ListApprovals)
		sr.PUT("/:id", middleware.RequirePermission("service_request", "write"), h.Update)
		sr.PUT("/:id/status", middleware.RequirePermission("service_request", "write"), h.UpdateStatus)
		sr.DELETE("/:id", middleware.RequirePermission("service_request", "delete"), h.Delete)
		sr.POST("/:id/approval", middleware.RequirePermission("service_request", "write"), h.ApplyApproval)
		sr.POST("/:id/approvals", middleware.RequirePermission("service_request", "write"), h.ApplyApproval)

		// Provisioning routes
		if provisioningHandler != nil {
			sr.POST("/:id/provision", middleware.RequirePermission("service_request", "write"), provisioningHandler.StartProvisioning)
			sr.GET("/:id/provisioning-tasks", middleware.RequirePermission("service_request", "read"), provisioningHandler.ListProvisioningTasks)
		}
	}

	// Provisioning task routes (separate path)
	provisioning := tenant.Group("/provisioning-tasks")
	{
		provisioning.POST("/:id/execute", middleware.RequirePermission("service_request", "write"), provisioningHandler.ExecuteProvisioningTask)
	}
}
