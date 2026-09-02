// Package survey — 客户满意度调查 handler.
// 迁移自 controller/survey_controller.go，保持原有 API 契约不变。
package survey

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
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
	surveys := rg.Group("/surveys")
	{
		surveys.GET("", h.ListSurveys)
		surveys.POST("", h.CreateSurvey)
		surveys.GET("/:id", h.GetSurvey)
		surveys.PUT("/:id", h.UpdateSurvey)
		surveys.GET("/:id/responses", h.GetSurveyResponses)
		surveys.GET("/:id/analytics", h.GetAnalytics)
		surveys.POST("/responses", h.SubmitResponse)
	}
}

// ListSurveys GET /api/v1/surveys
func (h *Handler) ListSurveys(ctx *gin.Context) {
	tenantID, _ := ctx.Get("tenant_id")
	surveys, err := h.svc.GetSurveys(ctx.Request.Context(), tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, surveys)
}

// GetSurvey GET /api/v1/surveys/:id
func (h *Handler) GetSurvey(ctx *gin.Context) {
	surveyID, _ := strconv.Atoi(ctx.Param("id"))
	tenantID, _ := ctx.Get("tenant_id")
	survey, err := h.svc.GetSurvey(ctx.Request.Context(), surveyID, tenantID.(int))
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
	tenantID, _ := ctx.Get("tenant_id")
	survey, err := h.svc.CreateSurvey(ctx.Request.Context(), &req, tenantID.(int))
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
	tenantID, _ := ctx.Get("tenant_id")
	survey, err := h.svc.UpdateSurvey(ctx.Request.Context(), surveyID, &req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, survey)
}

// GetSurveyResponses GET /api/v1/surveys/:id/responses
func (h *Handler) GetSurveyResponses(ctx *gin.Context) {
	surveyID, _ := strconv.Atoi(ctx.Param("id"))
	tenantID, _ := ctx.Get("tenant_id")
	responses, err := h.svc.GetSurveyResponses(ctx.Request.Context(), surveyID, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, responses)
}

// GetAnalytics GET /api/v1/surveys/:id/analytics
func (h *Handler) GetAnalytics(ctx *gin.Context) {
	surveyID, _ := strconv.Atoi(ctx.Param("id"))
	tenantID, _ := ctx.Get("tenant_id")
	analytics, err := h.svc.GetAnalytics(ctx.Request.Context(), surveyID, tenantID.(int))
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
	tenantID, _ := ctx.Get("tenant_id")
	if err := h.svc.SubmitResponse(ctx.Request.Context(), &req, tenantID.(int)); err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, gin.H{"message": "submitted"})
}
