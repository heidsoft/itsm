package cloud

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 云服务HTTP处理器
type Handler struct {
	cloudService *service.CloudService
	logger      *zap.SugaredLogger
}

// NewHandler creates a new cloud handler
func NewHandler(cloudService *service.CloudService, logger *zap.SugaredLogger) *Handler {
	return &Handler{cloudService: cloudService, logger: logger}
}

// getTenantID 从上下文获取租户ID
func (h *Handler) getTenantID(c *gin.Context) int {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return 0
	}
	id, ok := tenantID.(int)
	if !ok {
		return 0
	}
	return id
}

// ===================================
// CloudAccount Handlers
// ===================================

// CreateCloudAccount 创建云账号
// @Summary 创建云账号
// @Description 创建新的云账号
// @Tags 云管理
// @Accept json
// @Produce json
// @Param cloudAccount body dto.CreateCloudAccountRequest true "云账号信息"
// @Success 200 {object} common.Response{data=dto.CloudAccountResponse}
// @Router /api/v1/cloud/accounts [post]
func (h *Handler) CreateCloudAccount(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	var req dto.CreateCloudAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("参数绑定失败: %v", err)
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	account, err := h.cloudService.CreateCloudAccount(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Errorf("创建云账号失败: %v", err)
		common.FailWithErr(c, err, "create cloud account failed")
		return
	}

	common.Success(c, dto.ToCloudAccountResponse(account))
}

// GetCloudAccount 获取云账号详情
// @Summary 获取云账号详情
// @Description 获取指定云账号的详细信息
// @Tags 云管理
// @Accept json
// @Produce json
// @Param id path int true "云账号ID"
// @Success 200 {object} common.Response{data=dto.CloudAccountResponse}
// @Router /api/v1/cloud/accounts/{id} [get]
func (h *Handler) GetCloudAccount(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的云账号ID")
		return
	}

	account, err := h.cloudService.GetCloudAccount(c.Request.Context(), tenantID, id)
	if err != nil {
		h.logger.Errorf("获取云账号失败: %v", err)
		common.FailWithErr(c, err, "get cloud account failed")
		return
	}

	common.Success(c, dto.ToCloudAccountResponse(account))
}

// UpdateCloudAccount 更新云账号
// @Summary 更新云账号
// @Description 更新指定云账号的信息
// @Tags 云管理
// @Accept json
// @Produce json
// @Param id path int true "云账号ID"
// @Param cloudAccount body dto.UpdateCloudAccountRequest true "云账号信息"
// @Success 200 {object} common.Response{data=dto.CloudAccountResponse}
// @Router /api/v1/cloud/accounts/{id} [put]
func (h *Handler) UpdateCloudAccount(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的云账号ID")
		return
	}

	var req dto.UpdateCloudAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("参数绑定失败: %v", err)
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	account, err := h.cloudService.UpdateCloudAccount(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		h.logger.Errorf("更新云账号失败: %v", err)
		common.FailWithErr(c, err, "update cloud account failed")
		return
	}

	common.Success(c, dto.ToCloudAccountResponse(account))
}

// DeleteCloudAccount 删除云账号
// @Summary 删除云账号
// @Description 删除指定云账号
// @Tags 云管理
// @Accept json
// @Produce json
// @Param id path int true "云账号ID"
// @Success 200 {object} common.Response
// @Router /api/v1/cloud/accounts/{id} [delete]
func (h *Handler) DeleteCloudAccount(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的云账号ID")
		return
	}

	err = h.cloudService.DeleteCloudAccount(c.Request.Context(), tenantID, id)
	if err != nil {
		h.logger.Errorf("删除云账号失败: %v", err)
		common.FailWithErr(c, err, "delete cloud account failed")
		return
	}

	common.Success(c, nil)
}

// ListCloudAccounts 获取云账号列表
// @Summary 获取云账号列表
// @Description 分页获取云账号列表
// @Tags 云管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param provider query string false "云厂商过滤"
// @Param isActive query bool false "是否启用过滤"
// @Param search query string false "搜索关键词"
// @Success 200 {object} common.Response{data=dto.CloudAccountListResponse}
// @Router /api/v1/cloud/accounts [get]
func (h *Handler) ListCloudAccounts(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	var req dto.ListCloudAccountsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Errorf("参数绑定失败: %v", err)
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	accounts, total, err := h.cloudService.ListCloudAccounts(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Errorf("获取云账号列表失败: %v", err)
		common.FailWithErr(c, err, "list cloud accounts failed")
		return
	}

	responses := make([]dto.CloudAccountResponse, len(accounts))
	for i, account := range accounts {
		responses[i] = *dto.ToCloudAccountResponse(account)
	}

	response := &dto.CloudAccountListResponse{
		CloudAccounts: responses,
		Total:        total,
		Page:         req.Page,
		PageSize:     req.PageSize,
	}

	common.Success(c, response)
}

// ===================================
// CloudService Handlers
// ===================================

// CreateCloudService 创建云服务
// @Summary 创建云服务
// @Description 创建新的云服务定义
// @Tags 云管理
// @Accept json
// @Produce json
// @Param cloudService body dto.CreateCloudServiceRequest true "云服务信息"
// @Success 200 {object} common.Response{data=dto.CloudServiceResponse}
// @Router /api/v1/cloud/services [post]
func (h *Handler) CreateCloudService(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	var req dto.CreateCloudServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("参数绑定失败: %v", err)
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	service, err := h.cloudService.CreateCloudService(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Errorf("创建云服务失败: %v", err)
		common.FailWithErr(c, err, "create cloud service failed")
		return
	}

	common.Success(c, dto.ToCloudServiceResponse(service))
}

// GetCloudService 获取云服务详情
// @Summary 获取云服务详情
// @Description 获取指定云服务的详细信息
// @Tags 云管理
// @Accept json
// @Produce json
// @Param id path int true "云服务ID"
// @Success 200 {object} common.Response{data=dto.CloudServiceResponse}
// @Router /api/v1/cloud/services/{id} [get]
func (h *Handler) GetCloudService(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的云服务ID")
		return
	}

	service, err := h.cloudService.GetCloudService(c.Request.Context(), tenantID, id)
	if err != nil {
		h.logger.Errorf("获取云服务失败: %v", err)
		common.FailWithErr(c, err, "get cloud service failed")
		return
	}

	common.Success(c, dto.ToCloudServiceResponse(service))
}

// UpdateCloudService 更新云服务
// @Summary 更新云服务
// @Description 更新指定云服务的信息
// @Tags 云管理
// @Accept json
// @Produce json
// @Param id path int true "云服务ID"
// @Param cloudService body dto.UpdateCloudServiceRequest true "云服务信息"
// @Success 200 {object} common.Response{data=dto.CloudServiceResponse}
// @Router /api/v1/cloud/services/{id} [put]
func (h *Handler) UpdateCloudService(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的云服务ID")
		return
	}

	var req dto.UpdateCloudServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("参数绑定失败: %v", err)
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	service, err := h.cloudService.UpdateCloudService(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		h.logger.Errorf("更新云服务失败: %v", err)
		common.FailWithErr(c, err, "update cloud service failed")
		return
	}

	common.Success(c, dto.ToCloudServiceResponse(service))
}

// DeleteCloudService 删除云服务
// @Summary 删除云服务
// @Description 删除指定云服务
// @Tags 云管理
// @Accept json
// @Produce json
// @Param id path int true "云服务ID"
// @Success 200 {object} common.Response
// @Router /api/v1/cloud/services/{id} [delete]
func (h *Handler) DeleteCloudService(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的云服务ID")
		return
	}

	err = h.cloudService.DeleteCloudService(c.Request.Context(), tenantID, id)
	if err != nil {
		h.logger.Errorf("删除云服务失败: %v", err)
		common.FailWithErr(c, err, "delete cloud service failed")
		return
	}

	common.Success(c, nil)
}

// ListCloudServices 获取云服务列表
// @Summary 获取云服务列表
// @Description 分页获取云服务列表
// @Tags 云管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param provider query string false "云厂商过滤"
// @Param category query string false "服务分类过滤"
// @Param isSystem query bool false "是否系统预置过滤"
// @Param isActive query bool false "是否启用过滤"
// @Param parentId query int false "父级服务ID过滤"
// @Param search query string false "搜索关键词"
// @Success 200 {object} common.Response{data=dto.CloudServiceListResponse}
// @Router /api/v1/cloud/services [get]
func (h *Handler) ListCloudServices(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	var req dto.ListCloudServicesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Errorf("参数绑定失败: %v", err)
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	services, total, err := h.cloudService.ListCloudServices(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Errorf("获取云服务列表失败: %v", err)
		common.FailWithErr(c, err, "list cloud services failed")
		return
	}

	responses := make([]dto.CloudServiceResponse, len(services))
	for i, svc := range services {
		responses[i] = *dto.ToCloudServiceResponse(svc)
	}

	response := &dto.CloudServiceListResponse{
		CloudServices: responses,
		Total:        total,
		Page:         req.Page,
		PageSize:     req.PageSize,
	}

	common.Success(c, response)
}

// ===================================
// CloudResource Handlers
// ===================================

// CreateCloudResource 创建云资源
// @Summary 创建云资源
// @Description 创建新的云资源记录
// @Tags 云管理
// @Accept json
// @Produce json
// @Param cloudResource body dto.CreateCloudResourceRequest true "云资源信息"
// @Success 200 {object} common.Response{data=dto.CloudResourceResponse}
// @Router /api/v1/cloud/resources [post]
func (h *Handler) CreateCloudResource(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	var req dto.CreateCloudResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("参数绑定失败: %v", err)
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	resource, err := h.cloudService.CreateCloudResource(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Errorf("创建云资源失败: %v", err)
		common.FailWithErr(c, err, "create cloud resource failed")
		return
	}

	common.Success(c, dto.ToCloudResourceResponse(resource))
}

// GetCloudResource 获取云资源详情
// @Summary 获取云资源详情
// @Description 获取指定云资源的详细信息
// @Tags 云管理
// @Accept json
// @Produce json
// @Param id path int true "云资源ID"
// @Success 200 {object} common.Response{data=dto.CloudResourceResponse}
// @Router /api/v1/cloud/resources/{id} [get]
func (h *Handler) GetCloudResource(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的云资源ID")
		return
	}

	resource, err := h.cloudService.GetCloudResource(c.Request.Context(), tenantID, id)
	if err != nil {
		h.logger.Errorf("获取云资源失败: %v", err)
		common.FailWithErr(c, err, "get cloud resource failed")
		return
	}

	common.Success(c, dto.ToCloudResourceResponse(resource))
}

// UpdateCloudResource 更新云资源
// @Summary 更新云资源
// @Description 更新指定云资源的信息
// @Tags 云管理
// @Accept json
// @Produce json
// @Param id path int true "云资源ID"
// @Param cloudResource body dto.UpdateCloudResourceRequest true "云资源信息"
// @Success 200 {object} common.Response{data=dto.CloudResourceResponse}
// @Router /api/v1/cloud/resources/{id} [put]
func (h *Handler) UpdateCloudResource(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的云资源ID")
		return
	}

	var req dto.UpdateCloudResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("参数绑定失败: %v", err)
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	resource, err := h.cloudService.UpdateCloudResource(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		h.logger.Errorf("更新云资源失败: %v", err)
		common.FailWithErr(c, err, "update cloud resource failed")
		return
	}

	common.Success(c, dto.ToCloudResourceResponse(resource))
}

// DeleteCloudResource 删除云资源
// @Summary 删除云资源
// @Description 删除指定云资源
// @Tags 云管理
// @Accept json
// @Produce json
// @Param id path int true "云资源ID"
// @Success 200 {object} common.Response
// @Router /api/v1/cloud/resources/{id} [delete]
func (h *Handler) DeleteCloudResource(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的云资源ID")
		return
	}

	err = h.cloudService.DeleteCloudResource(c.Request.Context(), tenantID, id)
	if err != nil {
		h.logger.Errorf("删除云资源失败: %v", err)
		common.FailWithErr(c, err, "delete cloud resource failed")
		return
	}

	common.Success(c, nil)
}

// ListCloudResources 获取云资源列表
// @Summary 获取云资源列表
// @Description 分页获取云资源列表
// @Tags 云管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param provider query string false "云厂商过滤"
// @Param serviceId query int false "云服务ID过滤"
// @Param accountId query int false "云账号ID过滤"
// @Param region query string false "区域过滤"
// @Param resourceType query string false "资源类型过滤"
// @Param isActive query bool false "是否启用过滤"
// @Param search query string false "搜索关键词"
// @Success 200 {object} common.Response{data=dto.CloudResourceListResponse}
// @Router /api/v1/cloud/resources [get]
func (h *Handler) ListCloudResources(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	var req dto.ListCloudResourcesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Errorf("参数绑定失败: %v", err)
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	resources, total, err := h.cloudService.ListCloudResources(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.logger.Errorf("获取云资源列表失败: %v", err)
		common.FailWithErr(c, err, "list cloud resources failed")
		return
	}

	responses := make([]dto.CloudResourceResponse, len(resources))
	for i, resource := range resources {
		responses[i] = *dto.ToCloudResourceResponse(resource)
	}

	response := &dto.CloudResourceListResponse{
		CloudResources: responses,
		Total:         total,
		Page:          req.Page,
		PageSize:      req.PageSize,
	}

	common.Success(c, response)
}
