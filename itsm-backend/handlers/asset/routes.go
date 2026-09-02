package asset

import (
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 注册资产与许可证路由
func SetupRoutes(auth *gin.RouterGroup, h *Handler) {
	// ==================== Assets ====================
	assets := auth.Group("/assets")
	{
		assets.GET("", middleware.RequirePermission("asset", "read"), h.ListAssets)
		assets.POST("", middleware.RequirePermission("asset", "write"), h.CreateAsset)
		assets.GET("/stats", middleware.RequirePermission("asset", "read"), h.GetAssetStats)
		assets.GET("/:id", middleware.RequirePermission("asset", "read"), h.GetAsset)
		assets.PUT("/:id", middleware.RequirePermission("asset", "write"), h.UpdateAsset)
		assets.PUT("/:id/status", middleware.RequirePermission("asset", "write"), h.UpdateAssetStatus)
		assets.PUT("/:id/assign", middleware.RequirePermission("asset", "write"), h.AssignAsset)
		assets.PUT("/:id/retire", middleware.RequirePermission("asset", "write"), h.RetireAsset)
		assets.DELETE("/:id", middleware.RequirePermission("asset", "delete"), h.DeleteAsset)
	}

	// ==================== Asset Licenses ====================
	licenses := auth.Group("/licenses")
	{
		licenses.GET("", middleware.RequirePermission("license", "read"), h.ListLicenses)
		licenses.POST("", middleware.RequirePermission("license", "write"), h.CreateLicense)
		licenses.GET("/stats", middleware.RequirePermission("license", "read"), h.GetLicenseStats)
		licenses.GET("/:id", middleware.RequirePermission("license", "read"), h.GetLicense)
		licenses.PUT("/:id", middleware.RequirePermission("license", "write"), h.UpdateLicense)
		licenses.PUT("/:id/assign", middleware.RequirePermission("license", "write"), h.AssignUsers)
		licenses.DELETE("/:id", middleware.RequirePermission("license", "delete"), h.DeleteLicense)
	}
}
