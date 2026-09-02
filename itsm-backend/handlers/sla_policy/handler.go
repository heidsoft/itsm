// Package sla_policy 是 SLA 策略域的 HTTP handler 层（域切片架构）。
// 自 controller/sla_policy_controller.go 迁移而来（2026-09-02），
// 业务逻辑仍由 service.SLAPolicyService 承载，本包只做参数解析与响应封装。
package sla_policy

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

// Handler SLA 策略 HTTP handler
type Handler struct {
	service *service.SLAPolicyService
}

// NewHandler 创建 SLA 策略 handler 实例
func NewHandler(client *ent.Client) *Handler {
	return &Handler{
		service: service.NewSLAPolicyService(client),
	}
}

// tenantID 提取租户上下文
func tenantID(c *gin.Context) (int, bool) {
	tid, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return 0, false
	}
	return tid, true
}

// pathID 提取路径参数 ID
func pathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的SLA策略ID")
		return 0, false
	}
	return id, true
}

// CreateSLAPolicy 创建 SLA 策略
func (h *Handler) CreateSLAPolicy(ctx *gin.Context) {
	var req dto.CreateSLAPolicyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, err.Error())
		return
	}

	tid, ok := tenantID(ctx)
	if !ok {
		return
	}
	req.TenantID = tid

	policy, err := h.service.CreateSLAPolicy(ctx.Request.Context(), req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, dto.ToSLAPolicyResponse(policy))
}

// ListSLAPolicies 获取 SLA 策略列表
func (h *Handler) ListSLAPolicies(ctx *gin.Context) {
	tid, ok := tenantID(ctx)
	if !ok {
		return
	}

	policies, err := h.service.QuerySLAPolicies(ctx.Request.Context(), tid)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, dto.ToSLAPolicyResponseList(policies))
}

// GetSLAPolicy 获取单个 SLA 策略
func (h *Handler) GetSLAPolicy(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}

	tid, ok := tenantID(ctx)
	if !ok {
		return
	}

	policy, err := h.service.GetSLAPolicyByIDForTenant(ctx.Request.Context(), id, tid)
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(ctx, common.NotFoundCode, "SLA策略不存在")
			return
		}
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, dto.ToSLAPolicyResponse(policy))
}

// UpdateSLAPolicy 更新 SLA 策略
func (h *Handler) UpdateSLAPolicy(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}

	var req dto.UpdateSLAPolicyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, err.Error())
		return
	}

	tid, ok := tenantID(ctx)
	if !ok {
		return
	}

	policy, err := h.service.UpdateSLAPolicyForTenant(ctx.Request.Context(), id, tid, req)
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(ctx, common.NotFoundCode, "SLA策略不存在")
			return
		}
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, dto.ToSLAPolicyResponse(policy))
}

// DeleteSLAPolicy 删除 SLA 策略
func (h *Handler) DeleteSLAPolicy(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}

	tid, ok := tenantID(ctx)
	if !ok {
		return
	}

	err := h.service.DeleteSLAPolicyForTenant(ctx.Request.Context(), id, tid)
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(ctx, common.NotFoundCode, "SLA策略不存在")
			return
		}
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "删除成功"})
}

// MatchSLAPolicy 匹配 SLA 策略
func (h *Handler) MatchSLAPolicy(ctx *gin.Context) {
	tid, ok := tenantID(ctx)
	if !ok {
		return
	}

	ticketType := ctx.Query("ticketType")
	priority := ctx.Query("priority")
	customerTier := ctx.Query("customerTier")

	policy, err := h.service.MatchSLAPolicy(ctx.Request.Context(), tid, ticketType, priority, customerTier)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, dto.ToSLAPolicyResponse(policy))
}

// GetSLAComplianceRate 获取 SLA 合规率
func (h *Handler) GetSLAComplianceRate(ctx *gin.Context) {
	tid, ok := tenantID(ctx)
	if !ok {
		return
	}

	rate, err := h.service.GetSLAComplianceRate(ctx.Request.Context(), tid, time.Now().AddDate(0, 0, -30), time.Now())
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, gin.H{"complianceRate": rate})
}
