package controller

import (
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// SLAPolicyController SLA策略控制器
type SLAPolicyController struct {
	service *service.SLAPolicyService
}

// NewSLAPolicyController 创建SLA策略控制器
func NewSLAPolicyController(client *ent.Client) *SLAPolicyController {
	return &SLAPolicyController{
		service: service.NewSLAPolicyService(client),
	}
}

// CreateSLAPolicy 创建SLA策略
// @Summary 创建SLA策略
// @Description 创建新的SLA策略
// @Tags SLA策略管理
// @Accept json
// @Produce json
// @Param request body dto.CreateSLAPolicyRequest true "SLA策略信息"
// @Success 200 {object} common.Response
// @Router /api/v1/sla-policies [post]
func (c *SLAPolicyController) CreateSLAPolicy(ctx *gin.Context) {
	var req dto.CreateSLAPolicyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	req.TenantID = tenantID
	policy, err := c.service.CreateSLAPolicy(ctx.Request.Context(), req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, dto.ToSLAPolicyResponse(policy))
}

// ListSLAPolicies 获取SLA策略列表
// @Summary 获取SLA策略列表
// @Description 获取所有SLA策略列表
// @Tags SLA策略管理
// @Accept json
// @Produce json
// @Success 200 {object} common.Response
// @Router /api/v1/sla-policies [get]
func (c *SLAPolicyController) ListSLAPolicies(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	policies, err := c.service.QuerySLAPolicies(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, dto.ToSLAPolicyResponseList(policies))
}

// GetSLAPolicy 获取单个SLA策略
// @Summary 获取单个SLA策略
// @Description 根据ID获取SLA策略详情
// @Tags SLA策略管理
// @Accept json
// @Produce json
// @Param id path int true "SLA策略ID"
// @Success 200 {object} common.Response
// @Router /api/v1/sla-policies/{id} [get]
func (c *SLAPolicyController) GetSLAPolicy(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的SLA策略ID")
		return
	}

	policy, err := c.service.GetSLAPolicyByID(ctx.Request.Context(), id)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, dto.ToSLAPolicyResponse(policy))
}

// UpdateSLAPolicy 更新SLA策略
// @Summary 更新SLA策略
// @Description 更新SLA策略信息
// @Tags SLA策略管理
// @Accept json
// @Produce json
// @Param id path int true "SLA策略ID"
// @Param request body dto.UpdateSLAPolicyRequest true "SLA策略信息"
// @Success 200 {object} common.Response
// @Router /api/v1/sla-policies/{id} [put]
func (c *SLAPolicyController) UpdateSLAPolicy(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的SLA策略ID")
		return
	}

	var req dto.UpdateSLAPolicyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, err.Error())
		return
	}

	policy, err := c.service.UpdateSLAPolicy(ctx.Request.Context(), id, req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, dto.ToSLAPolicyResponse(policy))
}

// DeleteSLAPolicy 删除SLA策略
// @Summary 删除SLA策略
// @Description 删除SLA策略
// @Tags SLA策略管理
// @Accept json
// @Produce json
// @Param id path int true "SLA策略ID"
// @Success 200 {object} common.Response
// @Router /api/v1/sla-policies/{id} [delete]
func (c *SLAPolicyController) DeleteSLAPolicy(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的SLA策略ID")
		return
	}

	err = c.service.DeleteSLAPolicy(ctx.Request.Context(), id)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "删除成功"})
}

// MatchSLAPolicy 匹配SLA策略
// @Summary 匹配SLA策略
// @Description 根据工单属性匹配最优SLA策略
// @Tags SLA策略管理
// @Accept json
// @Produce json
// @Param ticket_type query string false "工单类型"
// @Param priority query string false "优先级"
// @Param customer_tier query string false "客户等级"
// @Success 200 {object} common.Response
// @Router /api/v1/sla-policies/match [get]
func (c *SLAPolicyController) MatchSLAPolicy(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	ticketType := ctx.Query("ticket_type")
	priority := ctx.Query("priority")
	customerTier := ctx.Query("customer_tier")

	policy, err := c.service.MatchSLAPolicy(ctx.Request.Context(), tenantID, ticketType, priority, customerTier)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, dto.ToSLAPolicyResponse(policy))
}

// GetSLAComplianceRate 获取SLA合规率
// @Summary 获取SLA合规率
// @Description 获取指定时间范围内的SLA合规率
// @Tags SLA策略管理
// @Accept json
// @Produce json
// @Param start_date query string false "开始日期"
// @Param end_date query string false "结束日期"
// @Success 200 {object} common.Response
// @Router /api/v1/sla-policies/compliance-rate [get]
func (c *SLAPolicyController) GetSLAComplianceRate(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	// Note: start_date and end_date parsing would be added here
	// For now, use default time range
	rate, err := c.service.GetSLAComplianceRate(ctx.Request.Context(), tenantID, time.Now().AddDate(0, 0, -30), time.Now())
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, gin.H{"compliance_rate": rate})
}