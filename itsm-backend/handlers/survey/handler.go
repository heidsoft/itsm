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

// RegisterRoutes 已删除：本域唯一的路由所有者是 router.SetupSurveyRoutes（由 router.go 注册）。
// 此处曾有一份逐字重复的 7 条注册（GET/POST /surveys、GET/PUT /surveys/:id、
// GET /surveys/:id/{responses,analytics}、POST /surveys/responses），从未被调用；
// 若两处同时生效，gin 会因重复路由直接 panic。授权口径注释已迁到 router/survey_routes.go。

// ListSurveys GET /api/v1/surveys
func (h *Handler) ListSurveys(ctx *gin.Context) {
	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}
	surveys, err := h.svc.GetSurveys(ctx.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(ctx, err, "操作失败")
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
		common.RespondError(ctx, err, "操作失败")
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
		common.RespondError(ctx, err, "操作失败")
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
		common.RespondError(ctx, err, "操作失败")
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
		common.RespondError(ctx, err, "操作失败")
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
		common.RespondError(ctx, err, "操作失败")
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
		common.RespondError(ctx, err, "操作失败")
		return
	}
	common.Success(ctx, gin.H{"message": "submitted"})
}
