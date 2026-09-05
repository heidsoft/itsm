package router

import (
	"github.com/gin-gonic/gin"

	surveyHandler "itsm-backend/handlers/survey"
	"itsm-backend/middleware"
)

// SetupSurveyRoutes 注册问卷调查域路由。
// 从 router.go 集中注册块抽取，行为与原内联等价。
func SetupSurveyRoutes(tenant *gin.RouterGroup, h *surveyHandler.Handler) {
	surveys := tenant.Group("/surveys")
	{
		surveys.GET("", middleware.RequirePermission("survey", "read"), h.ListSurveys)
		surveys.POST("", middleware.RequirePermission("survey", "write"), h.CreateSurvey)
		surveys.GET("/:id", middleware.RequirePermission("survey", "read"), h.GetSurvey)
		surveys.PUT("/:id", middleware.RequirePermission("survey", "write"), h.UpdateSurvey)
		surveys.GET("/:id/responses", middleware.RequirePermission("survey", "read"), h.GetSurveyResponses)
		surveys.GET("/:id/analytics", middleware.RequirePermission("survey", "read"), h.GetAnalytics)
		surveys.POST("/responses", middleware.RequirePermission("survey", "write"), h.SubmitResponse)
	}
}
