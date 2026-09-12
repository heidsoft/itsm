package tenant

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler HTTP handler for tenant domain
type Handler struct {
	svc    Service
	logger *zap.SugaredLogger
}

// NewHandler creates a new tenant handler
func NewHandler(svc Service, logger *zap.SugaredLogger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// protectedSystemTenantCode 是平台默认/系统租户的稳定标识。它一旦被暂停或过期，
// 租户中间件会对所有请求 fail-closed（2003），连「恢复」入口自身都会 403，导致整站锁死。
const protectedSystemTenantCode = "default"

// isLockingTenantStatus 报告某状态是否会把租户移出服务（暂停/过期/删除）。
func isLockingTenantStatus(status string) bool {
	switch status {
	case "suspended", "expired", "deleted":
		return true
	default:
		return false
	}
}

// blockIfLockingSystemTenant 加载目标租户，若为系统默认租户且操作会将其锁定，
// 写入 HTTP 409 并返回 true（调用方据此中止）。加载失败时不拦截，交由后续用例处理。
func (h *Handler) blockIfLockingSystemTenant(c *gin.Context, tenantID int, status, verb string) bool {
	target, err := h.svc.GetTenant(c.Request.Context(), tenantID)
	if err != nil || target == nil {
		return false
	}
	if target.Code == protectedSystemTenantCode && isLockingTenantStatus(status) {
		common.Conflict(c, "系统默认租户不允许"+verb, gin.H{"tenantId": tenantID, "code": target.Code})
		return true
	}
	return false
}

// CreateTenant creates a new tenant
func (h *Handler) CreateTenant(c *gin.Context) {
	var req dto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("参数绑定失败: %v", err)
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	if !common.IsValidTenantCode(req.Code) {
		common.Fail(c, common.ParamErrorCode, "租户编码只能包含字母、数字、下划线或连字符，且需以字母或数字开头")
		return
	}

	tenant, err := h.svc.CreateTenant(c.Request.Context(), &req)
	if err != nil {
		h.logger.Errorf("创建租户失败: %v", err)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	h.recordAudit(c, "tenant.create", tenant.ID, "tenant_code", req.Code, "tenant_name", req.Name)
	common.Success(c, dto.ToTenantResponse(tenant))
}

// ListTenants lists tenants with pagination
func (h *Handler) ListTenants(c *gin.Context) {
	var req dto.ListTenantsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Errorf("参数绑定失败: %v", err)
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenants, total, err := h.svc.ListTenants(c.Request.Context(), &req)
	if err != nil {
		h.logger.Errorf("获取租户列表失败: %v", err)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	// Convert response format
	tenantResponses := make([]dto.TenantResponse, len(tenants))
	for i, tenant := range tenants {
		response := dto.ToTenantResponse(tenant)
		if response == nil {
			continue
		}
		tenantResponses[i] = *response
	}

	response := &dto.TenantListResponse{
		Tenants:  tenantResponses,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	common.Success(c, response)
}

// UpdateTenantStatus updates a tenant's status
func (h *Handler) UpdateTenantStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的租户ID")
		return
	}

	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("参数绑定失败: %v", err)
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	status, exists := req["status"]
	if !exists {
		common.Fail(c, 1001, "缺少状态参数")
		return
	}

	if h.blockIfLockingSystemTenant(c, id, status, "暂停或过期") {
		return
	}

	err = h.svc.UpdateTenantStatus(c.Request.Context(), id, status)
	if err != nil {
		h.logger.Errorf("更新租户状态失败: %v", err)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	h.recordAudit(c, "tenant.status.update", id, "new_status", status)
	common.Success(c, nil)
}

// GetTenant gets a tenant by ID
func (h *Handler) GetTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的租户ID")
		return
	}

	tenant, err := h.svc.GetTenant(c.Request.Context(), id)
	if err != nil {
		h.logger.Errorf("获取租户详情失败: %v", err)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, dto.ToTenantResponse(tenant))
}

// UpdateTenant updates a tenant
func (h *Handler) UpdateTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的租户ID")
		return
	}

	var req dto.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("参数绑定失败: %v", err)
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	if req.Status != nil && h.blockIfLockingSystemTenant(c, id, *req.Status, "暂停或过期") {
		return
	}

	tenant, err := h.svc.UpdateTenant(c.Request.Context(), id, &req)
	if err != nil {
		h.logger.Errorf("更新租户失败: %v", err)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	h.recordAudit(c, "tenant.update", id, "tenant_code", tenant.Code)
	common.Success(c, dto.ToTenantResponse(tenant))
}

// DeleteTenant deletes a tenant
func (h *Handler) DeleteTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的租户ID")
		return
	}

	if h.blockIfLockingSystemTenant(c, id, "deleted", "删除") {
		return
	}

	err = h.svc.DeleteTenant(c.Request.Context(), id)
	if err != nil {
		h.logger.Errorf("删除租户失败: %v", err)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	h.recordAudit(c, "tenant.delete", id)
	common.Success(c, nil)
}

// recordAudit emits a structured audit log entry for a tenant write action.
func (h *Handler) recordAudit(c *gin.Context, action string, tenantID int, fields ...any) {
	args := []any{
		"action", action,
		"tenant_id", tenantID,
		"operator_id", c.GetInt("user_id"),
		"operator_role", c.GetString("role"),
		"client_ip", c.ClientIP(),
	}
	args = append(args, fields...)
	h.logger.Infow("tenant audit", args...)
}

// ListTenantsAdmin lists all tenants for admin (no tenant middleware required)
func (h *Handler) ListTenantsAdmin(c *gin.Context) {
	var req dto.ListTenantsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Errorf("参数绑定失败: %v", err)
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenants, total, err := h.svc.ListTenants(c.Request.Context(), &req)
	if err != nil {
		h.logger.Errorf("获取租户列表失败: %v", err)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	tenantResponses := make([]dto.TenantResponse, len(tenants))
	for i, tenant := range tenants {
		response := dto.ToTenantResponse(tenant)
		if response == nil {
			continue
		}
		tenantResponses[i] = *response
	}

	response := &dto.TenantListResponse{
		Tenants:  tenantResponses,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	common.Success(c, response)
}

// GetTenantSettings returns settings for the current tenant (from JWT context)
func (h *Handler) GetTenantSettings(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		// 租户上下文缺失属认证问题，统一为 401（对齐上游 TenantMiddleware 与项目多数派）。
		common.Fail(c, common.AuthFailedCode, "无法获取租户信息")
		return
	}
	tenant, err := h.svc.GetTenant(c.Request.Context(), tenantID.(int))
	if err != nil {
		common.FailWithErr(c, err, "获取租户设置失败")
		return
	}
	common.Success(c, gin.H{
		"id":       tenant.ID,
		"name":     tenant.Name,
		"code":     tenant.Code,
		"domain":   tenant.Domain,
		"type":     tenant.Type,
		"status":   tenant.Status,
		"plan":     tenant.PlanCode,
		"tier":     tenant.ServiceTier,
		"owner":    tenant.OwnerContact,
		"billing":  tenant.BillingEnabled,
		"currency": tenant.Currency,
	})
}

// UpdateTenantSettings updates settings for the current tenant
func (h *Handler) UpdateTenantSettings(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		// 租户上下文缺失属认证问题，统一为 401（对齐上游 TenantMiddleware 与项目多数派）。
		common.Fail(c, common.AuthFailedCode, "无法获取租户信息")
		return
	}
	var req dto.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	updated, err := h.svc.UpdateTenant(c.Request.Context(), tenantID.(int), &req)
	if err != nil {
		common.FailWithErr(c, err, "更新租户设置失败")
		return
	}
	common.Success(c, gin.H{
		"id":     updated.ID,
		"name":   updated.Name,
		"code":   updated.Code,
		"domain": updated.Domain,
		"status": updated.Status,
		"plan":   updated.PlanCode,
		"tier":   updated.ServiceTier,
		"owner":  updated.OwnerContact,
	})
}
