// Package survey — 客户满意度调查 handler.
// 迁移自 controller/survey_controller.go，保持原有 API 契约不变。
package survey

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 持有 SurveyService 依赖.
type Handler struct {
	svc    *service.SurveyService
	logger *zap.SugaredLogger
}

// NewHandler 构造 survey Handler.
func NewHandler(svc *service.SurveyService, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes 注册 /api/v1/surveys 路由组.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	if rg == nil {
		return
	}
	// 满意度调查（2026-09-17 P0「越权写收口」）：
	// 问卷定义与响应写入统一按 survey:write 授权。响应提交（POST /responses）的
	// 主体是「被调查人」而非管理员，是否需要独立于管理面的授权码待批次 2 词表统一时定；
	// 当前该路径本身仍被路径预检拦截（批次 3 补映射），故无行为回归。
	surveys := rg.Group("/surveys")
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

// ListSurveys GET /api/v1/surveys
func (h *Handler) ListSurveys(ctx *gin.Context) {
	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}
	surveys, err := h.svc.GetSurveys(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, gin.H{
		"items": surveys,
		"total": len(surveys),
	})
}

// GetSurvey GET /api/v1/surveys/:id
func (h *Handler) GetSurvey(ctx *gin.Context) {
	surveyID, _ := strconv.Atoi(ctx.Param("id"))
	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}
	survey, err := h.svc.GetSurvey(ctx.Request.Context(), surveyID, tenantID)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, survey)
}

// CreateSurvey POST /api/v1/surveys
func (h *Handler) CreateSurvey(ctx *gin.Context) {
	var req dto.CreateSurveyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, 1001, "参数错误")
		return
	}
	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}
	survey, err := h.svc.CreateSurvey(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, survey)
}

// UpdateSurvey PUT /api/v1/surveys/:id
func (h *Handler) UpdateSurvey(ctx *gin.Context) {
	surveyID, _ := strconv.Atoi(ctx.Param("id"))
	var req dto.UpdateSurveyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, 1001, "参数错误")
		return
	}
	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}
	survey, err := h.svc.UpdateSurvey(ctx.Request.Context(), surveyID, &req, tenantID)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, survey)
}

// GetSurveyResponses GET /api/v1/surveys/:id/responses
func (h *Handler) GetSurveyResponses(ctx *gin.Context) {
	surveyID, _ := strconv.Atoi(ctx.Param("id"))
	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}
	responses, err := h.svc.GetSurveyResponses(ctx.Request.Context(), surveyID, tenantID)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, gin.H{
		"items": responses,
		"total": len(responses),
	})
}

// GetAnalytics GET /api/v1/surveys/:id/analytics
func (h *Handler) GetAnalytics(ctx *gin.Context) {
	surveyID, _ := strconv.Atoi(ctx.Param("id"))
	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}
	analytics, err := h.svc.GetAnalytics(ctx.Request.Context(), surveyID, tenantID)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, analytics)
}

// SubmitResponse POST /api/v1/surveys/responses
func (h *Handler) SubmitResponse(ctx *gin.Context) {
	var req dto.SubmitSurveyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, 1001, "参数错误")
		return
	}
	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}
	if err := h.svc.SubmitResponse(ctx.Request.Context(), &req, tenantID); err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, gin.H{"message": "submitted"})
}
