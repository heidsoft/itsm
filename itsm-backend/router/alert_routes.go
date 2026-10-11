package router

import (
	"github.com/gin-gonic/gin"

	connectorAlert "itsm-backend/connector/alert"
	"itsm-backend/middleware"
)

// SetupAlertRoutes 注册告警接入域路由（告警源事件接收）。
func SetupAlertRoutes(tenant *gin.RouterGroup, h *connectorAlert.Handler) {
	tenant.POST("/alerts/sources/:source/ingest", middleware.RequirePermission("alert", "write"), h.Ingest)
}
