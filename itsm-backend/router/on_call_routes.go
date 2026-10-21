package router

import (
	"itsm-backend/handlers/on_call"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupOnCallRoutes 注册值班排班域路由。
//
// 权限码沿用迁移 20260824 已种下的 on_call:{read,write}，与菜单基线一致。
// 排班数据此前只能直接改库，本路由是它唯一的 HTTP 所有者。
func SetupOnCallRoutes(tenant *gin.RouterGroup, h *on_call.Handler) {
	onCall := tenant.Group("/on-call")
	{
		onCall.GET("/schedules", middleware.RequirePermission("on_call", "read"), h.ListSchedules)
		onCall.POST("/schedules", middleware.RequirePermission("on_call", "write"), h.CreateSchedule)
		onCall.GET("/schedules/:id", middleware.RequirePermission("on_call", "read"), h.GetSchedule)
		onCall.PUT("/schedules/:id", middleware.RequirePermission("on_call", "write"), h.UpdateSchedule)
		onCall.DELETE("/schedules/:id", middleware.RequirePermission("on_call", "write"), h.DeleteSchedule)

		onCall.GET("/shifts", middleware.RequirePermission("on_call", "read"), h.ListShifts)
		onCall.POST("/shifts", middleware.RequirePermission("on_call", "write"), h.CreateShift)
		onCall.PUT("/shifts/:id", middleware.RequirePermission("on_call", "write"), h.UpdateShift)
		onCall.DELETE("/shifts/:id", middleware.RequirePermission("on_call", "write"), h.DeleteShift)

		onCall.GET("/current", middleware.RequirePermission("on_call", "read"), h.Current)
	}
}
