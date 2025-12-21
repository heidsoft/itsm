package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ServiceController 服务控制器
type ServiceController struct {
	serviceCatalogService *service.ServiceCatalogService
	serviceRequestService *service.ServiceRequestService
}

// NewServiceController 创建服务控制器实例
func NewServiceController(catalogService *service.ServiceCatalogService, requestService *service.ServiceRequestService) *ServiceController {
	return &ServiceController{
		serviceCatalogService: catalogService,
		serviceRequestService: requestService,
	}
}

// GetServiceCatalogs 获取服务目录列表
// @Summary 获取服务目录列表
// @Description 分页获取服务目录列表
// @Tags 服务目录
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param category query string false "分类过滤"
// @Param status query string false "状态过滤"
// @Success 200 {object} common.Response{data=dto.ServiceCatalogListResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/service-catalogs [get]
func (sc *ServiceController) GetServiceCatalogs(c *gin.Context) {
	var req dto.GetServiceCatalogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取租户ID
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	result, err := sc.serviceCatalogService.ListServiceCatalogs(c.Request.Context(), &req, tenantID.(int))
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, result)
}

// CreateServiceRequest 创建服务请求
// @Summary 创建服务请求
// @Description 发起新的服务请求
// @Tags 服务请求
// @Accept json
// @Produce json
// @Param request body dto.CreateServiceRequestRequest true "服务请求信息"
// @Success 200 {object} common.Response{data=dto.ServiceRequestResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/service-requests [post]
func (sc *ServiceController) CreateServiceRequest(c *gin.Context) {
	var req dto.CreateServiceRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取用户ID和租户ID
	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, 2001, "用户未认证")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	serviceRequest, err := sc.serviceRequestService.CreateServiceRequest(c.Request.Context(), &req, userID.(int), tenantID.(int))
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, serviceRequest)
}

// GetUserServiceRequests 获取当前用户的服务请求列表
// @Summary 获取当前用户的服务请求列表
// @Description 查询当前登录用户的服务请求列表
// @Tags 服务请求
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param status query string false "状态" Enums(pending,in_progress,completed,rejected)
// @Success 200 {object} common.Response{data=dto.ServiceRequestListResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/service-requests/me [get]
func (sc *ServiceController) GetUserServiceRequests(c *gin.Context) {
	var req dto.GetServiceRequestsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取用户ID和租户ID
	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, 2001, "用户未认证")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	req.UserID = userID.(int)

	result, err := sc.serviceRequestService.ListServiceRequests(c.Request.Context(), &req, tenantID.(int))
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, result)
}

// GetServiceRequestByID 获取服务请求详情
// @Summary 获取服务请求详情
// @Description 根据ID获取服务请求详情
// @Tags 服务请求
// @Accept json
// @Produce json
// @Param id path int true "服务请求ID"
// @Success 200 {object} common.Response{data=dto.ServiceRequestResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/service-requests/{id} [get]
func (sc *ServiceController) GetServiceRequestByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的服务请求ID")
		return
	}

	// 从上下文获取租户ID
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	serviceRequest, err := sc.serviceRequestService.GetServiceRequestDetail(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, serviceRequest)
}

// ApplyServiceRequestApproval 执行服务请求审批动作
// @Summary 服务请求审批
// @Description 对当前步骤执行 approve/reject（V0：默认三段审批）
// @Tags 服务请求
// @Accept json
// @Produce json
// @Param id path int true "服务请求ID"
// @Param request body dto.ServiceRequestApprovalActionRequest true "审批动作"
// @Success 200 {object} common.Response{data=dto.ServiceRequestResponse}
// @Failure 400 {object} common.Response
// @Failure 403 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/service-requests/{id}/approvals [post]
func (sc *ServiceController) ApplyServiceRequestApproval(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的服务请求ID")
		return
	}

	var req dto.ServiceRequestApprovalActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, 2001, "用户未认证")
		return
	}
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	updated, err := sc.serviceRequestService.ApplyApprovalAction(c.Request.Context(), id, tenantID.(int), userID.(int), req.Action, req.Comment)
	if err != nil {
		// 统一按业务错误返回 400/403 由 common.Fail code 区分，这里简单用 5001 并透传 message
		common.Fail(c, 5001, err.Error())
		return
	}
	common.Success(c, updated)
}

// GetServiceRequestApprovals 获取服务请求审批记录
// @Summary 获取服务请求审批记录
// @Description 返回指定服务请求的审批记录列表
// @Tags 服务请求
// @Accept json
// @Produce json
// @Param id path int true "服务请求ID"
// @Success 200 {object} common.Response{data=[]dto.ServiceRequestApprovalResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/service-requests/{id}/approvals [get]
func (sc *ServiceController) GetServiceRequestApprovals(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的服务请求ID")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	detail, err := sc.serviceRequestService.GetServiceRequestDetail(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}
	common.Success(c, detail.Approvals)
}

// GetPendingServiceRequestApprovals 获取当前用户的审批待办
// @Summary 获取审批待办
// @Description 根据当前用户角色返回需要其处理的服务请求列表（V0：默认三段审批）
// @Tags 服务请求
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} common.Response{data=dto.ServiceRequestListResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/service-requests/approvals/pending [get]
func (sc *ServiceController) GetPendingServiceRequestApprovals(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, 2001, "用户未认证")
		return
	}
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	result, err := sc.serviceRequestService.ListPendingApprovals(c.Request.Context(), tenantID.(int), userID.(int), page, size)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}
	common.Success(c, result)
}

// UpdateServiceRequestStatus 更新服务请求状态
// @Summary 更新服务请求状态
// @Description 更新服务请求的状态
// @Tags 服务请求
// @Accept json
// @Produce json
// @Param id path int true "服务请求ID"
// @Param request body dto.UpdateServiceRequestStatusRequest true "状态更新信息"
// @Success 200 {object} common.Response{data=dto.ServiceRequestResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/service-requests/{id}/status [put]
func (sc *ServiceController) UpdateServiceRequestStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的服务请求ID")
		return
	}

	var req dto.UpdateServiceRequestStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取租户ID
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	err = sc.serviceRequestService.UpdateServiceRequestStatus(c.Request.Context(), id, req.Status, tenantID.(int))
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	// 获取更新后的服务请求
	serviceRequest, err := sc.serviceRequestService.GetServiceRequestDetail(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Fail(c, 5001, "获取更新后的服务请求失败")
		return
	}

	common.Success(c, serviceRequest)
}

// CreateServiceCatalog 创建服务目录
// @Summary 创建服务目录
// @Description 创建新的服务目录
// @Tags 服务目录
// @Accept json
// @Produce json
// @Param request body dto.CreateServiceCatalogRequest true "服务目录信息"
// @Success 200 {object} common.Response{data=dto.ServiceCatalogResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/service-catalogs [post]
func (sc *ServiceController) CreateServiceCatalog(c *gin.Context) {
	var req dto.CreateServiceCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取租户ID
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	catalog, err := sc.serviceCatalogService.CreateServiceCatalog(c.Request.Context(), &req, tenantID.(int))
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, catalog)
}

// UpdateServiceCatalog 更新服务目录
// @Summary 更新服务目录
// @Description 更新指定的服务目录
// @Tags 服务目录
// @Accept json
// @Produce json
// @Param id path int true "服务目录ID"
// @Param request body dto.UpdateServiceCatalogRequest true "服务目录更新信息"
// @Success 200 {object} common.Response{data=dto.ServiceCatalogResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/service-catalogs/{id} [put]
func (sc *ServiceController) UpdateServiceCatalog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的服务目录ID")
		return
	}

	var req dto.UpdateServiceCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取租户ID
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	catalog, err := sc.serviceCatalogService.UpdateServiceCatalog(c.Request.Context(), id, &req, tenantID.(int))
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, catalog)
}

// DeleteServiceCatalog 删除服务目录
func (sc *ServiceController) DeleteServiceCatalog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, 1001, "无效的服务目录ID")
		return
	}

	err = sc.serviceCatalogService.DeleteServiceCatalog(c.Request.Context(), id)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, nil)
}

// GetServiceCatalogByID 根据ID获取服务目录
func (sc *ServiceController) GetServiceCatalogByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, 1001, "无效的服务目录ID")
		return
	}

	catalog, err := sc.serviceCatalogService.GetServiceCatalogByID(c.Request.Context(), id)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, catalog)
}
