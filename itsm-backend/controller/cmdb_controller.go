package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CMDBController struct {
	cmdbService *service.CMDBService
}

func NewCMDBController(cmdbService *service.CMDBService) *CMDBController {
	return &CMDBController{
		cmdbService: cmdbService,
	}
}

// CreateConfigurationItem 创建配置项
// @Summary 创建配置项
// @Description 创建新的配置项
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.CreateCIRequest true "创建配置项请求"
// @Success 200 {object} common.Response{data=dto.CIResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/cmdb/configuration-items [post]
func (ctrl *CMDBController) CreateConfigurationItem(c *gin.Context) {
	var req dto.CreateCIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	resp, err := ctrl.cmdbService.CreateCI(c.Request.Context(), &req)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, resp)
}

// GetConfigurationItem 获取配置项详情
// @Summary 获取配置项详情
// @Description 根据ID获取配置项详情
// @Tags CMDB
// @Produce json
// @Param id path int true "配置项ID"
// @Success 200 {object} common.Response{data=dto.CIResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/cmdb/configuration-items/{id} [get]
func (ctrl *CMDBController) GetConfigurationItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的配置项ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	resp, err := ctrl.cmdbService.GetCI(c.Request.Context(), id, tenantID)
	if err != nil {
		if err.Error() == "配置项不存在" {
			common.Fail(c, 4001, err.Error())
		} else {
			common.Fail(c, 5001, err.Error())
		}
		return
	}

	common.Success(c, resp)
}

// ListConfigurationItems 获取配置项列表
// @Summary 获取配置项列表
// @Description 分页获取配置项列表，支持多种过滤条件
// @Tags CMDB
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param type query string false "配置项类型" Enums(server,database,application,network,storage)
// @Param status query string false "状态" Enums(active,inactive,maintenance)
// @Param business_service query string false "业务服务"
// @Param owner query string false "负责人"
// @Param environment query string false "环境" Enums(production,staging,development)
// @Param search query string false "搜索关键词"
// @Success 200 {object} common.Response{data=dto.ListCIsResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/cmdb/configuration-items [get]
func (ctrl *CMDBController) ListConfigurationItems(c *gin.Context) {
	var req dto.ListCIsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	// 设置默认值
	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.Offset == 0 {
		req.Offset = 0
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}
	req.TenantID = tenantID

	resp, err := ctrl.cmdbService.ListCIs(c.Request.Context(), &req)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, resp)
}

// UpdateConfigurationItem 更新配置项
// @Summary 更新配置项
// @Description 更新配置项信息
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "配置项ID"
// @Param request body dto.UpdateCIRequest true "更新配置项请求"
// @Success 200 {object} common.Response{data=dto.CIResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/cmdb/configuration-items/{id} [put]
func (ctrl *CMDBController) UpdateConfigurationItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的配置项ID")
		return
	}

	var req dto.UpdateCIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	resp, err := ctrl.cmdbService.UpdateCI(c.Request.Context(), id, &req)
	if err != nil {
		if err.Error() == "配置项不存在" {
			common.Fail(c, 4001, err.Error())
		} else {
			common.Fail(c, 5001, err.Error())
		}
		return
	}

	common.Success(c, resp)
}

// DeleteConfigurationItem 删除配置项
// @Summary 删除配置项
// @Description 删除指定的配置项
// @Tags CMDB
// @Produce json
// @Param id path int true "配置项ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/cmdb/configuration-items/{id} [delete]
func (ctrl *CMDBController) DeleteConfigurationItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的配置项ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	err = ctrl.cmdbService.DeleteCI(c.Request.Context(), id, tenantID)
	if err != nil {
		if err.Error() == "配置项不存在" {
			common.Fail(c, 4001, err.Error())
		} else {
			common.Fail(c, 5001, err.Error())
		}
		return
	}

	common.Success(c, gin.H{"message": "删除成功"})
}

// CreateCIAttributeDefinition 创建CI属性定义
// @Summary 创建CI属性定义
// @Description 为指定的CI类型创建新的属性定义
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.CIAttributeDefinitionRequest true "属性定义信息"
// @Success 200 {object} common.Response{data=dto.CIAttributeDefinitionResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/cmdb/ci-types/attributes [post]
func (ctrl *CMDBController) CreateCIAttributeDefinition(c *gin.Context) {
	var req dto.CIAttributeDefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	result, err := ctrl.cmdbService.CreateCIAttributeDefinition(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(c, 5001, "创建属性定义失败: "+err.Error())
		return
	}

	common.Success(c, result)
}

// GetCITypeWithAttributes 获取CI类型及其属性定义
// @Summary 获取CI类型及其属性定义
// @Description 获取指定CI类型的详细信息及其所有属性定义
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI类型ID"
// @Success 200 {object} common.Response{data=dto.CITypeWithAttributesResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/cmdb/ci-types/{id}/attributes [get]
func (ctrl *CMDBController) GetCITypeWithAttributes(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, 1001, "无效的CI类型ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	result, err := ctrl.cmdbService.GetCITypeWithAttributes(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, 5001, "获取CI类型属性失败: "+err.Error())
		return
	}

	common.Success(c, result)
}

// ValidateCIAttributes 验证CI属性
// @Summary 验证CI属性
// @Description 根据CI类型的属性定义验证提供的属性值
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.ValidateCIAttributesRequest true "验证请求"
// @Success 200 {object} common.Response{data=dto.ValidateCIAttributesResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/cmdb/ci-attributes/validate [post]
func (ctrl *CMDBController) ValidateCIAttributes(c *gin.Context) {
	var req dto.ValidateCIAttributesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	result, err := ctrl.cmdbService.ValidateCIAttributes(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(c, 5001, "验证属性失败: "+err.Error())
		return
	}

	common.Success(c, result)
}

// SearchCIsByAttributes 根据属性搜索CI
// @Summary 根据属性搜索CI
// @Description 使用动态属性条件搜索配置项
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.CIAttributeSearchRequest true "搜索条件"
// @Success 200 {object} common.Response{data=dto.ListCIsResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/cmdb/cis/search-by-attributes [post]
func (ctrl *CMDBController) SearchCIsByAttributes(c *gin.Context) {
	var req dto.CIAttributeSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	result, err := ctrl.cmdbService.SearchCIsByAttributes(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(c, 5001, "搜索失败: "+err.Error())
		return
	}

	common.Success(c, result)
}
