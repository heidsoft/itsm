package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type IncidentController struct {
	incidentService *service.IncidentService
	logger          *zap.SugaredLogger
}

func NewIncidentController(incidentService *service.IncidentService, logger *zap.SugaredLogger) *IncidentController {
	return &IncidentController{
		incidentService: incidentService,
		logger:          logger,
	}
}

// CreateIncident 创建事件
func (c *IncidentController) CreateIncident(ctx *gin.Context) {
	var req dto.CreateIncidentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 从JWT中获取用户信息
	userID, exists := ctx.Get("user_id")
	if !exists {
		common.Fail(ctx, common.AuthFailedCode, "用户未认证")
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.ParamErrorCode, "租户信息缺失")
		return
	}

	incident, err := c.incidentService.CreateIncident(ctx, &req, userID.(int), tenantID.(int))
	if err != nil {
		c.logger.Errorw("Failed to create incident", "error", err)
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, incident)
}

// GetIncident 获取事件详情
func (c *IncidentController) GetIncident(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	incident, err := c.incidentService.GetIncidentByID(ctx, id)
	if err != nil {
		c.logger.Errorw("Failed to get incident", "incident_id", id, "error", err)
		common.Fail(ctx, common.NotFoundCode, err.Error())
		return
	}

	common.Success(ctx, incident)
}

// ListIncidents 获取事件列表
func (c *IncidentController) ListIncidents(ctx *gin.Context) {
	var req dto.ListIncidentsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.ParamErrorCode, "租户信息缺失")
		return
	}

	response, err := c.incidentService.ListIncidents(ctx, &req, tenantID.(int))
	if err != nil {
		c.logger.Errorw("Failed to list incidents", "error", err)
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, response)
}

// UpdateIncident 更新事件
func (c *IncidentController) UpdateIncident(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	var req dto.UpdateIncidentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	incident, err := c.incidentService.UpdateIncident(ctx, id, &req)
	if err != nil {
		c.logger.Errorw("Failed to update incident", "incident_id", id, "error", err)
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, incident)
}

// UpdateIncidentStatus 更新事件状态
func (c *IncidentController) UpdateIncidentStatus(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	var req dto.UpdateIncidentStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	incident, err := c.incidentService.UpdateIncidentStatus(ctx, id, &req)
	if err != nil {
		c.logger.Errorw("Failed to update incident status", "incident_id", id, "error", err)
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, incident)
}

// GetIncidentMetrics 获取事件指标
func (c *IncidentController) GetIncidentMetrics(ctx *gin.Context) {
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.ParamErrorCode, "租户信息缺失")
		return
	}

	metrics, err := c.incidentService.GetIncidentMetrics(ctx, tenantID.(int))
	if err != nil {
		c.logger.Errorw("Failed to get incident metrics", "error", err)
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, metrics)
}

// CreateIncidentFromAlibabaCloudAlert 从阿里云告警创建事件
func (c *IncidentController) CreateIncidentFromAlibabaCloudAlert(ctx *gin.Context) {
	var req dto.AlibabaCloudAlertRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 从JWT中获取用户信息
	userID, exists := ctx.Get("user_id")
	if !exists {
		common.Fail(ctx, common.AuthFailedCode, "用户未认证")
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.ParamErrorCode, "租户信息缺失")
		return
	}

	incident, err := c.incidentService.CreateIncidentFromAlibabaCloudAlert(ctx, &req, userID.(int), tenantID.(int))
	if err != nil {
		c.logger.Errorw("Failed to create incident from Alibaba Cloud alert", "error", err)
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, incident)
}

// CreateIncidentFromSecurityEvent 从安全事件创建事件
func (c *IncidentController) CreateIncidentFromSecurityEvent(ctx *gin.Context) {
	var req dto.SecurityEventRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 从JWT中获取用户信息
	userID, exists := ctx.Get("user_id")
	if !exists {
		common.Fail(ctx, common.AuthFailedCode, "用户未认证")
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.ParamErrorCode, "租户信息缺失")
		return
	}

	incident, err := c.incidentService.CreateIncidentFromSecurityEvent(ctx, &req, userID.(int), tenantID.(int))
	if err != nil {
		c.logger.Errorw("Failed to create incident from security event", "error", err)
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, incident)
}

// CreateIncidentFromCloudProductEvent 从云产品事件创建事件
func (c *IncidentController) CreateIncidentFromCloudProductEvent(ctx *gin.Context) {
	var req dto.CloudProductEventRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 从JWT中获取用户信息
	userID, exists := ctx.Get("user_id")
	if !exists {
		common.Fail(ctx, common.AuthFailedCode, "用户未认证")
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.ParamErrorCode, "租户信息缺失")
		return
	}

	incident, err := c.incidentService.CreateIncidentFromCloudProductEvent(ctx, &req, userID.(int), tenantID.(int))
	if err != nil {
		c.logger.Errorw("Failed to create incident from cloud product event", "error", err)
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, incident)
}
