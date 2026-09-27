package router

import (
	workbenchHandler "itsm-backend/handlers/workbench"

	"github.com/gin-gonic/gin"
)

// SetupWorkbenchRoutes registers workbench routes.
func SetupWorkbenchRoutes(tenant *gin.RouterGroup, h *workbenchHandler.Handler) {
	workbench := tenant.Group("/workbench")
	{
		workbench.GET("", h.Query)
	}
}
