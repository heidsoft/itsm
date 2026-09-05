package router

import (
	"github.com/gin-gonic/gin"

	vendorHandler "itsm-backend/handlers/vendor"
	"itsm-backend/middleware"
)

// SetupVendorRoutes 注册 Vendor（供应商）域路由。
// 从 router.go 集中注册块抽取，行为与原内联等价。
func SetupVendorRoutes(tenant *gin.RouterGroup, h *vendorHandler.Handler) {
	vendors := tenant.Group("/vendors")
	{
		vendors.GET("", middleware.RequirePermission("vendor", "read"), h.ListVendors)
		vendors.POST("", middleware.RequirePermission("vendor", "write"), h.CreateVendor)
		vendors.GET("/:id", middleware.RequirePermission("vendor", "read"), h.GetVendor)
		vendors.DELETE("/:id", middleware.RequirePermission("vendor", "delete"), h.DeleteVendor)
	}
}
