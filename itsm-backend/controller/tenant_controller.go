package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TenantController struct {
	tenantService *service.TenantService
	logger        *zap.SugaredLogger
}

func NewTenantController(tenantService *service.TenantService, logger *zap.SugaredLogger) *TenantController {
	return &TenantController{
		tenantService: tenantService,
		logger:        logger,
	}
}

// CreateTenant 创建租户
// @Summary 创建租户
// @Description 创建新的租户
// @Tags 租户管理
// @Accept json
// @Produce json
// @Param tenant body dto.CreateTenantRequest true "租户信息"
// @Success 200 {object} common.Response{data=dto.TenantResponse}
// @Failure 400 {object} common.Response
// @Router /api/admin/tenants [post]
func (tc *TenantController) CreateTenant(c *gin.Context) {
	var req dto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		tc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	tenant, err := tc.tenantService.CreateTenant(c.Request.Context(), &req)
	if err != nil {
		tc.logger.Errorf("创建租户失败: %v", err)
		common.Fail(c, 5001, "创建租户失败: "+err.Error())
		return
	}

	// 处理 Domain 字段的指针转换
	var domain *string
	if tenant.Domain != "" {
		domain = &tenant.Domain
	}

	response := &dto.TenantResponse{
		ID:        tenant.ID,
		Name:      tenant.Name,
		Code:      tenant.Code,
		Domain:    domain,
		Type:      tenant.Type,
		Status:    tenant.Status,
		ExpiresAt: &tenant.ExpiresAt,
		CreatedAt: tenant.CreatedAt,
		UpdatedAt: tenant.UpdatedAt,
	}

	common.Success(c, response)
}

// ListTenants 获取租户列表
// @Summary 获取租户列表
// @Description 分页获取租户列表
// @Tags 租户管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param status query string false "状态过滤"
// @Param type query string false "类型过滤"
// @Param search query string false "搜索关键词"
// @Success 200 {object} common.Response{data=dto.TenantListResponse}
// @Failure 400 {object} common.Response
// @Router /api/admin/tenants [get]
func (tc *TenantController) ListTenants(c *gin.Context) {
	var req dto.ListTenantsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		tc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	tenants, total, err := tc.tenantService.ListTenants(c.Request.Context(), &req)
	if err != nil {
		tc.logger.Errorf("获取租户列表失败: %v", err)
		common.Fail(c, 5001, "获取租户列表失败: "+err.Error())
		return
	}

	// 转换响应格式
	tenantResponses := make([]dto.TenantResponse, len(tenants))
	for i, tenant := range tenants {
		// 处理 Domain 字段的指针转换
		var domain *string
		if tenant.Domain != "" {
			domain = &tenant.Domain
		}

		tenantResponses[i] = dto.TenantResponse{
			ID:        tenant.ID,
			Name:      tenant.Name,
			Code:      tenant.Code,
			Domain:    domain,
			Type:      tenant.Type,
			Status:    tenant.Status,
			ExpiresAt: &tenant.ExpiresAt,
			CreatedAt: tenant.CreatedAt,
			UpdatedAt: tenant.UpdatedAt,
		}
	}

	response := &dto.TenantListResponse{
		Tenants: tenantResponses,
		Total:   total,
		Page:    req.Page,
		Size:    req.PageSize,
	}

	common.Success(c, response)
}

// UpdateTenantStatus 更新租户状态
// @Summary 更新租户状态
// @Description 更新指定租户的状态
// @Tags 租户管理
// @Accept json
// @Produce json
// @Param id path int true "租户ID"
// @Param status body map[string]string true "状态信息"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Router /api/admin/tenants/{id}/status [put]
func (tc *TenantController) UpdateTenantStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的租户ID")
		return
	}

	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		tc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	status, exists := req["status"]
	if !exists {
		common.Fail(c, 1001, "缺少状态参数")
		return
	}

	err = tc.tenantService.UpdateTenantStatus(c.Request.Context(), id, status)
	if err != nil {
		tc.logger.Errorf("更新租户状态失败: %v", err)
		common.Fail(c, 5001, "更新租户状态失败: "+err.Error())
		return
	}

	common.Success(c, nil)
}
