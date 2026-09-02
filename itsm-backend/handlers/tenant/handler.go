package tenant

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler HTTP handler for tenant domain
type Handler struct {
	svc   *service.TenantService
	logger *zap.SugaredLogger
}

// NewHandler creates a new tenant handler
func NewHandler(svc *service.TenantService, logger *zap.SugaredLogger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// CreateTenant creates a new tenant
func (h *Handler) CreateTenant(c *gin.Context) {
	var req dto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("参数绑定失败: %v", err)
		common.ParamErrorWithErr(c, err, "请求参数错误")
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
