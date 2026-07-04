package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CloudController struct {
	cloudService *service.CloudService
	logger       *zap.SugaredLogger
}

func NewCloudController(cloudService *service.CloudService, logger *zap.SugaredLogger) *CloudController {
	return &CloudController{
		cloudService: cloudService,
		logger:       logger,
	}
}

// getTenantID 从上下文获取租户ID
func (cc *CloudController) getTenantID(c *gin.Context) int {
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
// @Failure 400 {object} common.Response
// @Router /api/v1/cloud/accounts [post]
func (cc *CloudController) CreateCloudAccount(c *gin.Context) {
	tenantID := cc.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	var req dto.CreateCloudAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		cc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	account, err := cc.cloudService.CreateCloudAccount(c.Request.Context(), tenantID, &req)
	if err != nil {
		cc.logger.Errorf("创建云账号失败: %v", err)
		common.Fail(c, 5001, "创建云账号失败: "+err.Error())
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
// @Failure 400 {object} common.Response
// @Router /api/v1/cloud/accounts/{id} [get]
func (cc *CloudController) GetCloudAccount(c *gin.Context) {
	tenantID := cc.getTenantID(c)
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

	account, err := cc.cloudService.GetCloudAccount(c.Request.Context(), tenantID, id)
	if err != nil {
		cc.logger.Errorf("获取云账号失败: %v", err)
		common.Fail(c, 5001, "获取云账号失败: "+err.Error())
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
// @Failure 400 {object} common.Response
// @Router /api/v1/cloud/accounts/{id} [put]
func (cc *CloudController) UpdateCloudAccount(c *gin.Context) {
	tenantID := cc.getTenantID(c)
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
		cc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	account, err := cc.cloudService.UpdateCloudAccount(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		cc.logger.Errorf("更新云账号失败: %v", err)
		common.Fail(c, 5001, "更新云账号失败: "+err.Error())
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
// @Failure 400 {object} common.Response
// @Router /api/v1/cloud/accounts/{id} [delete]
func (cc *CloudController) DeleteCloudAccount(c *gin.Context) {
	tenantID := cc.getTenantID(c)
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

	err = cc.cloudService.DeleteCloudAccount(c.Request.Context(), tenantID, id)
	if err != nil {
		cc.logger.Errorf("删除云账号失败: %v", err)
		common.Fail(c, 5001, "删除云账号失败: "+err.Error())
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
// @Failure 400 {object} common.Response
// @Router /api/v1/cloud/accounts [get]
func (cc *CloudController) ListCloudAccounts(c *gin.Context) {
	tenantID := cc.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	var req dto.ListCloudAccountsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		cc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	accounts, total, err := cc.cloudService.ListCloudAccounts(c.Request.Context(), tenantID, &req)
	if err != nil {
		cc.logger.Errorf("获取云账号列表失败: %v", err)
		common.Fail(c, 5001, "获取云账号列表失败: "+err.Error())
		return
	}

	responses := make([]dto.CloudAccountResponse, len(accounts))
	for i, account := range accounts {
		responses[i] = *dto.ToCloudAccountResponse(account)
	}

	response := &dto.CloudAccountListResponse{
		CloudAccounts: responses,
		Total:         total,
		Page:          req.Page,
		PageSize:      req.PageSize,
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
// @Failure 400 {object} common.Response
// @Router /api/v1/cloud/services [post]
func (cc *CloudController) CreateCloudService(c *gin.Context) {
	tenantID := cc.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	var req dto.CreateCloudServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		cc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	service, err := cc.cloudService.CreateCloudService(c.Request.Context(), tenantID, &req)
	if err != nil {
		cc.logger.Errorf("创建云服务失败: %v", err)
		common.Fail(c, 5001, "创建云服务失败: "+err.Error())
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
// @Failure 400 {object} common.Response
// @Router /api/v1/cloud/services/{id} [get]
func (cc *CloudController) GetCloudService(c *gin.Context) {
	tenantID := cc.getTenantID(c)
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

	service, err := cc.cloudService.GetCloudService(c.Request.Context(), tenantID, id)
	if err != nil {
		cc.logger.Errorf("获取云服务失败: %v", err)
		common.Fail(c, 5001, "获取云服务失败: "+err.Error())
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
// @Failure 400 {object} common.Response
// @Router /api/v1/cloud/services/{id} [put]
func (cc *CloudController) UpdateCloudService(c *gin.Context) {
	tenantID := cc.getTenantID(c)
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
		cc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	service, err := cc.cloudService.UpdateCloudService(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		cc.logger.Errorf("更新云服务失败: %v", err)
		common.Fail(c, 5001, "更新云服务失败: "+err.Error())
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
// @Failure 400 {object} common.Response
// @Router /api/v1/cloud/services/{id} [delete]
func (cc *CloudController) DeleteCloudService(c *gin.Context) {
	tenantID := cc.getTenantID(c)
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

	err = cc.cloudService.DeleteCloudService(c.Request.Context(), tenantID, id)
	if err != nil {
		cc.logger.Errorf("删除云服务失败: %v", err)
		common.Fail(c, 5001, "删除云服务失败: "+err.Error())
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
// @Failure 400 {object} common.Response
// @Router /api/v1/cloud/services [get]
func (cc *CloudController) ListCloudServices(c *gin.Context) {
	tenantID := cc.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	var req dto.ListCloudServicesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		cc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	services, total, err := cc.cloudService.ListCloudServices(c.Request.Context(), tenantID, &req)
	if err != nil {
		cc.logger.Errorf("获取云服务列表失败: %v", err)
		common.Fail(c, 5001, "获取云服务列表失败: "+err.Error())
		return
	}

	responses := make([]dto.CloudServiceResponse, len(services))
	for i, svc := range services {
		responses[i] = *dto.ToCloudServiceResponse(svc)
	}

	response := &dto.CloudServiceListResponse{
		CloudServices: responses,
		Total:         total,
		Page:          req.Page,
		PageSize:      req.PageSize,
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
// @Failure 400 {object} common.Response
// @Router /api/v1/cloud/resources [post]
func (cc *CloudController) CreateCloudResource(c *gin.Context) {
	tenantID := cc.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	var req dto.CreateCloudResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		cc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	resource, err := cc.cloudService.CreateCloudResource(c.Request.Context(), tenantID, &req)
	if err != nil {
		cc.logger.Errorf("创建云资源失败: %v", err)
		common.Fail(c, 5001, "创建云资源失败: "+err.Error())
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
// @Failure 400 {object} common.Response
// @Router /api/v1/cloud/resources/{id} [get]
func (cc *CloudController) GetCloudResource(c *gin.Context) {
	tenantID := cc.getTenantID(c)
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

	resource, err := cc.cloudService.GetCloudResource(c.Request.Context(), tenantID, id)
	if err != nil {
		cc.logger.Errorf("获取云资源失败: %v", err)
		common.Fail(c, 5001, "获取云资源失败: "+err.Error())
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
// @Failure 400 {object} common.Response
// @Router /api/v1/cloud/resources/{id} [put]
func (cc *CloudController) UpdateCloudResource(c *gin.Context) {
	tenantID := cc.getTenantID(c)
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
		cc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	resource, err := cc.cloudService.UpdateCloudResource(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		cc.logger.Errorf("更新云资源失败: %v", err)
		common.Fail(c, 5001, "更新云资源失败: "+err.Error())
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
// @Failure 400 {object} common.Response
// @Router /api/v1/cloud/resources/{id} [delete]
func (cc *CloudController) DeleteCloudResource(c *gin.Context) {
	tenantID := cc.getTenantID(c)
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

	err = cc.cloudService.DeleteCloudResource(c.Request.Context(), tenantID, id)
	if err != nil {
		cc.logger.Errorf("删除云资源失败: %v", err)
		common.Fail(c, 5001, "删除云资源失败: "+err.Error())
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
// @Param cloudAccountId query int false "云账号ID过滤"
// @Param serviceId query int false "云服务类型ID过滤"
// @Param region query string false "Region过滤"
// @Param status query string false "资源状态过滤"
// @Param search query string false "搜索关键词"
// @Success 200 {object} common.Response{data=dto.CloudResourceListResponse}
// @Failure 400 {object} common.Response
// @Router /api/v1/cloud/resources [get]
func (cc *CloudController) ListCloudResources(c *gin.Context) {
	tenantID := cc.getTenantID(c)
	if tenantID == 0 {
		common.Fail(c, 1001, "无法获取租户信息")
		return
	}

	var req dto.ListCloudResourcesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		cc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	resources, total, err := cc.cloudService.ListCloudResources(c.Request.Context(), tenantID, &req)
	if err != nil {
		cc.logger.Errorf("获取云资源列表失败: %v", err)
		common.Fail(c, 5001, "获取云资源列表失败: "+err.Error())
		return
	}

	responses := make([]dto.CloudResourceResponse, len(resources))
	for i, resource := range resources {
		resp := dto.ToCloudResourceResponse(resource)
		responses[i] = *resp
	}

	response := &dto.CloudResourceListResponse{
		CloudResources: responses,
		Total:          total,
		Page:           req.Page,
		PageSize:       req.PageSize,
	}

	common.Success(c, response)
}
