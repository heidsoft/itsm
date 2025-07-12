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
// @Description 获取服务目录列表，支持分类和状态筛选
// @Tags 服务目录
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param category query string false "服务分类"
// @Param status query string false "状态" Enums(enabled,disabled)
// @Success 200 {object} common.Response{data=dto.ServiceCatalogListResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/service-catalogs [get]
func (sc *ServiceController) GetServiceCatalogs(c *gin.Context) {
	var req dto.GetServiceCatalogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}
	
	result, err := sc.serviceCatalogService.GetServiceCatalogs(c.Request.Context(), &req)
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
// @Router /api/service-requests [post]
func (sc *ServiceController) CreateServiceRequest(c *gin.Context) {
	var req dto.CreateServiceRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}
	
	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, 2001, "用户未认证")
		return
	}
	
	serviceRequest, err := sc.serviceRequestService.CreateServiceRequest(c.Request.Context(), &req, userID.(int))
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}
	
	// 获取完整信息用于响应
	fullRequest, err := sc.serviceRequestService.GetServiceRequestByID(c.Request.Context(), serviceRequest.ID)
	if err != nil {
		common.Fail(c, 5001, "获取创建的服务请求失败")
		return
	}
	
	common.Success(c, dto.ToServiceRequestResponse(fullRequest))
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
// @Router /api/service-requests/me [get]
func (sc *ServiceController) GetUserServiceRequests(c *gin.Context) {
	var req dto.GetServiceRequestsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}
	
	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, 2001, "用户未认证")
		return
	}
	
	req.UserID = userID.(int)
	
	result, err := sc.serviceRequestService.GetUserServiceRequests(c.Request.Context(), &req)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}
	
	common.Success(c, result)
}

// GetServiceRequestByID 获取服务请求详情
// @Summary 获取服务请求详情
// @Description 根据ID获取服务请求的详细信息
// @Tags 服务请求
// @Accept json
// @Produce json
// @Param id path int true "服务请求ID"
// @Success 200 {object} common.Response{data=dto.ServiceRequestResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/service-requests/{id} [get]
func (sc *ServiceController) GetServiceRequestByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的服务请求ID")
		return
	}
	
	serviceRequest, err := sc.serviceRequestService.GetServiceRequestByID(c.Request.Context(), id)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}
	
	common.Success(c, dto.ToServiceRequestResponse(serviceRequest))
}

// UpdateServiceRequestStatus 更新服务请求状态
// @Summary 更新服务请求状态
// @Description 更新服务请求的状态（审批人/管理员操作）
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
// @Router /api/service-requests/{id}/status [put]
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
	
	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, 2001, "用户未认证")
		return
	}
	
	_, err = sc.serviceRequestService.UpdateServiceRequestStatus(c.Request.Context(), id, req.Status, userID.(int))
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}
	
	// 获取更新后的完整信息
	updatedRequest, err := sc.serviceRequestService.GetServiceRequestByID(c.Request.Context(), id)
	if err != nil {
		common.Fail(c, 5001, "获取更新后的服务请求失败")
		return
	}
	
	common.Success(c, dto.ToServiceRequestResponse(updatedRequest))
}

// CreateServiceCatalog 创建服务目录
func (sc *ServiceController) CreateServiceCatalog(c *gin.Context) {
	var req dto.CreateServiceCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	catalog, err := sc.serviceCatalogService.CreateServiceCatalog(c.Request.Context(), &req)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, dto.ToServiceCatalogResponse(catalog))
}

// UpdateServiceCatalog 更新服务目录
func (sc *ServiceController) UpdateServiceCatalog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, 1001, "无效的服务目录ID")
		return
	}

	var req dto.UpdateServiceCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	catalog, err := sc.serviceCatalogService.UpdateServiceCatalog(c.Request.Context(), id, &req)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, dto.ToServiceCatalogResponse(catalog))
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

	common.Success(c, dto.ToServiceCatalogResponse(catalog))
}