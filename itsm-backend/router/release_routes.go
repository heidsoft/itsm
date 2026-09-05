package router

import (
	"github.com/gin-gonic/gin"

	releaseHandler "itsm-backend/handlers/release"
	"itsm-backend/middleware"
)

// SetupReleaseRoutes 注册 Release（发布管理）域路由。
// 从 router.go 集中注册块抽取，行为与原内联等价。
func SetupReleaseRoutes(tenant *gin.RouterGroup, h *releaseHandler.ReleaseHandler) {
	releases := tenant.Group("/releases")
	{
		releases.GET("", middleware.RequirePermission("release", "read"), h.ListReleases)
		releases.POST("", middleware.RequirePermission("release", "write"), h.CreateRelease)
		releases.GET("/stats", middleware.RequirePermission("release", "read"), h.GetReleaseStats)
		releases.GET("/:id", middleware.RequirePermission("release", "read"), h.GetRelease)
		releases.PUT("/:id", middleware.RequirePermission("release", "write"), h.UpdateRelease)
		releases.PUT("/:id/status", middleware.RequirePermission("release", "write"), h.UpdateReleaseStatus)
		releases.POST("/:id/approve", middleware.RequirePermission("release", "approve"), h.ApproveRelease)
		releases.POST("/:id/reject", middleware.RequirePermission("release", "approve"), h.RejectRelease)
		releases.POST("/:id/rollback", middleware.RequirePermission("release", "rollback"), h.RollbackRelease)
		releases.DELETE("/:id", middleware.RequirePermission("release", "delete"), h.DeleteRelease)
	}
}
