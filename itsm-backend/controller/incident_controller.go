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
		c.logger.Errorw("创建事件请求参数错误", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 获取当前用户ID
	reporterID, err := c.incidentService.GetCurrentUserID(ctx)
	if err != nil {
		c.logger.Errorw("获取当前用户ID失败", "error", err)
		common.Fail(ctx, common.AuthFailedCode, "用户未登录: "+err.Error())
		return
	}

	// 获取当前租户ID
	tenantID, err := c.incidentService.GetCurrentTenantID(ctx)
	if err != nil {
		c.logger.Errorw("获取当前租户ID失败", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	// 创建事件
	incident, err := c.incidentService.CreateIncident(ctx, &req, reporterID, tenantID)
	if err != nil {
		c.logger.Errorw("创建事件失败", "error", err, "reporter_id", reporterID, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "创建事件失败: "+err.Error())
		return
	}

	c.logger.Infow("事件创建成功", "incident_id", incident.ID, "incident_number", incident.IncidentNumber)
	common.Success(ctx, incident)
}

// GetIncidents 获取事件列表
func (c *IncidentController) GetIncidents(ctx *gin.Context) {
	var req dto.GetIncidentsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		c.logger.Errorw("获取事件列表请求参数错误", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}

	// 获取当前租户ID
	tenantID, err := c.incidentService.GetCurrentTenantID(ctx)
	if err != nil {
		c.logger.Errorw("获取当前租户ID失败", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}
	req.TenantID = tenantID

	// 获取事件列表
	result, err := c.incidentService.GetIncidents(ctx, &req)
	if err != nil {
		c.logger.Errorw("获取事件列表失败", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取事件列表失败: "+err.Error())
		return
	}

	c.logger.Infow("获取事件列表成功", "total", result.Total, "page", req.Page, "size", req.Size)
	common.Success(ctx, result)
}

// GetIncident 获取单个事件
func (c *IncidentController) GetIncident(ctx *gin.Context) {
	incidentIDStr := ctx.Param("id")
	incidentID, err := strconv.Atoi(incidentIDStr)
	if err != nil {
		c.logger.Errorw("事件ID参数错误", "error", err, "incident_id", incidentIDStr)
		common.Fail(ctx, common.ParamErrorCode, "事件ID参数错误: "+err.Error())
		return
	}

	// 获取当前租户ID
	tenantID, err := c.incidentService.GetCurrentTenantID(ctx)
	if err != nil {
		c.logger.Errorw("获取当前租户ID失败", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	// 获取事件详情
	incident, err := c.incidentService.GetIncident(ctx, incidentID, tenantID)
	if err != nil {
		c.logger.Errorw("获取事件详情失败", "error", err, "incident_id", incidentID, "tenant_id", tenantID)
		common.Fail(ctx, common.NotFoundCode, "事件不存在或无权限: "+err.Error())
		return
	}

	c.logger.Infow("获取事件详情成功", "incident_id", incidentID)
	common.Success(ctx, incident)
}

// UpdateIncident 更新事件
func (c *IncidentController) UpdateIncident(ctx *gin.Context) {
	incidentIDStr := ctx.Param("id")
	incidentID, err := strconv.Atoi(incidentIDStr)
	if err != nil {
		c.logger.Errorw("事件ID参数错误", "error", err, "incident_id", incidentIDStr)
		common.Fail(ctx, common.ParamErrorCode, "事件ID参数错误: "+err.Error())
		return
	}

	var req dto.UpdateIncidentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("更新事件请求参数错误", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 获取当前租户ID
	tenantID, err := c.incidentService.GetCurrentTenantID(ctx)
	if err != nil {
		c.logger.Errorw("获取当前租户ID失败", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	// 更新事件
	incident, err := c.incidentService.UpdateIncident(ctx, incidentID, &req, tenantID)
	if err != nil {
		c.logger.Errorw("更新事件失败", "error", err, "incident_id", incidentID, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "更新事件失败: "+err.Error())
		return
	}

	c.logger.Infow("事件更新成功", "incident_id", incidentID)
	common.Success(ctx, incident)
}

// CloseIncident 关闭事件
func (c *IncidentController) CloseIncident(ctx *gin.Context) {
	incidentIDStr := ctx.Param("id")
	incidentID, err := strconv.Atoi(incidentIDStr)
	if err != nil {
		c.logger.Errorw("事件ID参数错误", "error", err, "incident_id", incidentIDStr)
		common.Fail(ctx, common.ParamErrorCode, "事件ID参数错误: "+err.Error())
		return
	}

	// 获取当前租户ID
	tenantID, err := c.incidentService.GetCurrentTenantID(ctx)
	if err != nil {
		c.logger.Errorw("获取当前租户ID失败", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	// 关闭事件
	err = c.incidentService.CloseIncident(ctx, incidentID, tenantID)
	if err != nil {
		c.logger.Errorw("关闭事件失败", "error", err, "incident_id", incidentID, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "关闭事件失败: "+err.Error())
		return
	}

	c.logger.Infow("事件关闭成功", "incident_id", incidentID)
	common.Success(ctx, nil)
}

// GetIncidentStats 获取事件统计
func (c *IncidentController) GetIncidentStats(ctx *gin.Context) {
	// 获取当前租户ID
	tenantID, err := c.incidentService.GetCurrentTenantID(ctx)
	if err != nil {
		c.logger.Errorw("获取当前租户ID失败", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	// 获取事件统计
	stats, err := c.incidentService.GetIncidentStats(ctx, tenantID)
	if err != nil {
		c.logger.Errorw("获取事件统计失败", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取事件统计失败: "+err.Error())
		return
	}

	c.logger.Infow("获取事件统计成功", "tenant_id", tenantID)
	common.Success(ctx, stats)
}

// GetConfigurationItemsForIncident 获取可关联的配置项列表
// @Summary 获取可关联的配置项列表
// @Description 获取当前租户下可关联到事件的配置项列表
// @Tags 事件管理
// @Produce json
// @Param search query string false "搜索关键词"
// @Param type query string false "配置项类型"
// @Param status query string false "配置项状态"
// @Success 200 {object} common.Response{data=[]dto.ConfigurationItemInfo}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/incidents/configuration-items [get]
func (c *IncidentController) GetConfigurationItemsForIncident(ctx *gin.Context) {
	// 获取查询参数
	search := ctx.Query("search")
	ciType := ctx.Query("type")
	status := ctx.Query("status")

	// 获取当前租户ID
	tenantID, err := c.incidentService.GetCurrentTenantID(ctx)
	if err != nil {
		c.logger.Errorw("获取当前租户ID失败", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	// 调用服务层方法获取配置项列表
	ciInfos, err := c.incidentService.GetConfigurationItemsForIncident(ctx, tenantID, search, ciType, status)
	if err != nil {
		c.logger.Errorw("获取配置项列表失败", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取配置项列表失败: "+err.Error())
		return
	}

	c.logger.Infow("获取配置项列表成功", "count", len(ciInfos), "tenant_id", tenantID)
	common.Success(ctx, ciInfos)
}
