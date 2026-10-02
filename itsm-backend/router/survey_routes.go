package router

import (
	"github.com/gin-gonic/gin"

	surveyHandler "itsm-backend/handlers/survey"
	"itsm-backend/middleware"
)

// SetupSurveyRoutes 注册问卷调查域路由。
// 从 router.go 集中注册块抽取，行为与原内联等价。
//
// 授权口径（2026-09-17 P0「越权写收口」，本域唯一所有者即本函数）：
// 问卷定义与响应写入统一按 survey:write 授权。响应提交（POST /responses）的
// 主体是「被调查人」而非管理员，是否需要独立于管理面的授权码待批次 2 词表统一时定；
// 当前该路径本身仍被路径预检拦截（批次 3 补映射），故无行为回归。
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
