package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// SurveyController handles survey HTTP requests
type SurveyController struct {
	svc *service.SurveyService
}

// NewSurveyController creates a new SurveyController
func NewSurveyController(svc *service.SurveyService) *SurveyController {
	return &SurveyController{svc: svc}
}

// SubmitResponse handles POST /api/v1/surveys/responses
func (c *SurveyController) SubmitResponse(ctx *gin.Context) {
	var req dto.SubmitSurveyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, 1001, "参数错误")
		return
	}
	tenantID, _ := ctx.Get("tenant_id")
	if err := c.svc.SubmitResponse(ctx.Request.Context(), &req, tenantID.(int)); err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, gin.H{"message": "submitted"})
}

// GetAnalytics handles GET /api/v1/surveys/:id/analytics
func (c *SurveyController) GetAnalytics(ctx *gin.Context) {
	surveyID, _ := strconv.Atoi(ctx.Param("id"))
	tenantID, _ := ctx.Get("tenant_id")
	analytics, err := c.svc.GetAnalytics(ctx.Request.Context(), surveyID, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, analytics)
}

// ListSurveys handles GET /api/v1/surveys
func (c *SurveyController) ListSurveys(ctx *gin.Context) {
	tenantID, _ := ctx.Get("tenant_id")
	surveys, err := c.svc.GetSurveys(ctx.Request.Context(), tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, surveys)
}

// GetSurvey handles GET /api/v1/surveys/:id
func (c *SurveyController) GetSurvey(ctx *gin.Context) {
	surveyID, _ := strconv.Atoi(ctx.Param("id"))
	tenantID, _ := ctx.Get("tenant_id")
	survey, err := c.svc.GetSurvey(ctx.Request.Context(), surveyID, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, survey)
}

// CreateSurvey handles POST /api/v1/surveys
func (c *SurveyController) CreateSurvey(ctx *gin.Context) {
	var req dto.CreateSurveyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, 1001, "参数错误")
		return
	}
	tenantID, _ := ctx.Get("tenant_id")
	survey, err := c.svc.CreateSurvey(ctx.Request.Context(), &req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, survey)
}

// UpdateSurvey handles PUT /api/v1/surveys/:id
func (c *SurveyController) UpdateSurvey(ctx *gin.Context) {
	surveyID, _ := strconv.Atoi(ctx.Param("id"))
	var req dto.UpdateSurveyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, 1001, "参数错误")
		return
	}
	tenantID, _ := ctx.Get("tenant_id")
	survey, err := c.svc.UpdateSurvey(ctx.Request.Context(), surveyID, &req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, survey)
}

// GetSurveyResponses handles GET /api/v1/surveys/:id/responses
func (c *SurveyController) GetSurveyResponses(ctx *gin.Context) {
	surveyID, _ := strconv.Atoi(ctx.Param("id"))
	tenantID, _ := ctx.Get("tenant_id")
	responses, err := c.svc.GetSurveyResponses(ctx.Request.Context(), surveyID, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, responses)
}
