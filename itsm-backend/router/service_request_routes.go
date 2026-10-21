package router

import (
	"github.com/gin-gonic/gin"

	"itsm-backend/handlers/provisioning"
	"itsm-backend/handlers/service_request"
	"itsm-backend/middleware"
)

// SetupServiceRequestRoutes 注册服务请求域路由。
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
		sr.GET("/:id/sla", middleware.RequirePermission("service_request", "read"), h.GetServiceRequestSLA)
		sr.GET("/:id/approvals", middleware.RequirePermission("service_request", "read"), h.ListApprovals)
		sr.PUT("/:id", middleware.RequirePermission("service_request", "write"), h.Update)
		sr.PUT("/:id/status", middleware.RequirePermission("service_request", "write"), h.UpdateStatus)
		sr.DELETE("/:id", middleware.RequirePermission("service_request", "delete"), h.Delete)
		sr.POST("/:id/approval", middleware.RequirePermission("service_request", "approve"), h.ApplyApproval)
		sr.POST("/:id/approvals", middleware.RequirePermission("service_request", "approve"), h.ApplyApproval)

		// Provisioning routes
		if provisioningHandler != nil {
			sr.POST("/:id/provision", middleware.RequirePermission("service_request", "write"), provisioningHandler.StartProvisioning)
			sr.GET("/:id/provisioning-tasks", middleware.RequirePermission("service_request", "read"), provisioningHandler.ListProvisioningTasks)
		}
	}

	// Provisioning task routes (separate path)
	//
	// execute / retry / resolve 三者是交付链路对「失败」的三种处置：
	//   execute —— 首次执行或运维强制再执行一次（受状态守卫约束，succeeded 不可重入）；
	//   retry   —— 把失败任务重新排队，受 MaxTaskAttempts 约束，每次尝试独立幂等键；
	//   resolve —— 自动重试不可能成功时的人工判定（provider 永久故障 / 资源已在平台外建好）。
	// 三者都要求 service_request:write：交付会真实创建云资源，读权限不应携带这种副作用。
	provisioning := tenant.Group("/provisioning-tasks")
	{
		provisioning.POST("/:id/execute", middleware.RequirePermission("service_request", "write"), provisioningHandler.ExecuteProvisioningTask)
		provisioning.POST("/:id/retry", middleware.RequirePermission("service_request", "write"), provisioningHandler.RetryProvisioningTask)
		provisioning.POST("/:id/resolve", middleware.RequirePermission("service_request", "write"), provisioningHandler.ResolveProvisioningTask)
	}
}
