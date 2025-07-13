package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"net/http"
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
// @Param request body dto.CreateConfigurationItemRequest true "创建配置项请求"
// @Success 200 {object} common.Response{data=dto.ConfigurationItemResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/cmdb/configuration-items [post]
func (ctrl *CMDBController) CreateConfigurationItem(c *gin.Context) {
	var req dto.CreateConfigurationItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	resp, err := ctrl.cmdbService.CreateConfigurationItem(c.Request.Context(), &req, tenantID)
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
// @Success 200 {object} common.Response{data=dto.ConfigurationItemResponse}
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

	resp, err := ctrl.cmdbService.GetConfigurationItem(c.Request.Context(), id, tenantID)
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
// @Success 200 {object} common.Response{data=dto.ConfigurationItemListResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/cmdb/configuration-items [get]
func (ctrl *CMDBController) ListConfigurationItems(c *gin.Context) {
	var req dto.ListConfigurationItemsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Size == 0 {
		req.Size = 10
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	resp, err := ctrl.cmdbService.ListConfigurationItems(c.Request.Context(), &req, tenantID)
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
// @Param request body dto.UpdateConfigurationItemRequest true "更新配置项请求"
// @Success 200 {object} common.Response{data=dto.ConfigurationItemResponse}
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

	var req dto.UpdateConfigurationItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	resp, err := ctrl.cmdbService.UpdateConfigurationItem(c.Request.Context(), id, &req, tenantID)
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

	err = ctrl.cmdbService.DeleteConfigurationItem(c.Request.Context(), id, tenantID)
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

// GetConfigurationItemStats 获取配置项统计信息
// @Summary 获取配置项统计信息
// @Description 获取配置项的统计信息，包括总数、状态分布、类型分布等
// @Tags CMDB
// @Produce json
// @Success 200 {object} common.Response{data=dto.ConfigurationItemStatsResponse}
// @Failure 500 {object} common.Response
// @Router /api/cmdb/configuration-items/stats [get]
func (ctrl *CMDBController) GetConfigurationItemStats(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	resp, err := ctrl.cmdbService.GetConfigurationItemStats(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, resp)
}
