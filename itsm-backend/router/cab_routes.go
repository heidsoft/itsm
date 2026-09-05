package router

import (
	"github.com/gin-gonic/gin"

	"itsm-backend/handlers/cab"
	"itsm-backend/middleware"
)

// SetupCABRoutes 注册 CAB（变更咨询委员会）域路由。
// 从 router.go 集中注册块抽取，行为与原内联等价。
func SetupCABRoutes(tenant *gin.RouterGroup, h *cab.Handler) {
	cabGrp := tenant.Group("/cab")
	{
		cabGrp.GET("/members", middleware.RequirePermission("change", "read"), h.ListCABMembers)
		cabGrp.POST("/members", middleware.RequirePermission("change", "write"), h.AddCABMember)
		cabGrp.PUT("/members/:id", middleware.RequirePermission("change", "write"), h.UpdateCABMember)
		cabGrp.DELETE("/members/:id", middleware.RequirePermission("change", "write"), h.RemoveCABMember)
	}
}
