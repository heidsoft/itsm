package router

import (
	"github.com/gin-gonic/gin"

	"itsm-backend/handlers/change_review"
	"itsm-backend/middleware"
)

// SetupChangeReviewRoutes 注册变更评审组域路由。
func SetupChangeReviewRoutes(tenant *gin.RouterGroup, h *change_review.Handler) {
	grp := tenant.Group("/change-review")
	{
		grp.GET("/members", middleware.RequirePermission("change", "read"), h.ListMembers)
		grp.POST("/members", middleware.RequirePermission("change", "write"), h.AddMember)
		grp.PUT("/members/:id", middleware.RequirePermission("change", "write"), h.UpdateMember)
		grp.DELETE("/members/:id", middleware.RequirePermission("change", "write"), h.RemoveMember)
	}
}
