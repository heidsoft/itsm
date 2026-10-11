package router

import (
	"github.com/gin-gonic/gin"

	surveyHandler "itsm-backend/handlers/survey"
	"itsm-backend/middleware"
)

// SetupSurveyRoutes 注册问卷调查域路由。
//
// 问卷定义与响应写入统一按 survey:write 授权。响应提交（POST /responses）的主体是
// 「被调查人」而非管理员，是否需要独立于管理面的授权码待定。
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
