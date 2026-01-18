package controller

import (
	"itsm-backend/common"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DashboardController 仪表盘控制器
type DashboardController struct {
	dashboardService *service.DashboardService
	logger           *zap.SugaredLogger
}

// NewDashboardController 创建仪表盘控制器实例
func NewDashboardController(dashboardService *service.DashboardService, logger *zap.SugaredLogger) *DashboardController {
	return &DashboardController{
		dashboardService: dashboardService,
		logger:           logger,
	}
}

// GetDashboardData 获取仪表盘数据
// @Summary 获取仪表盘数据
// @Description 获取仪表盘KPI指标、资源分布和健康状态数据
// @Tags dashboard
// @Produce json
// @Success 200 {object} common.Response{data=dto.DashboardResponse}
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @RouterIgnore /api/dashboard [get]
func (dc *DashboardController) GetDashboardData(c *gin.Context) {
	dashboardData, err := dc.dashboardService.GetDashboardData(c.Request.Context())
	if err != nil {
		dc.logger.Errorw("Failed to get dashboard data", "error", err)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dashboardData)
}

// GetKPIMetrics 获取KPI指标
// @Summary 获取KPI指标
// @Description 获取仪表盘KPI指标数据
// @Tags dashboard
// @Produce json
// @Success 200 {object} common.Response{data=[]dto.DashboardKPIResponse}
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @RouterIgnore /api/dashboard/kpis [get]
func (dc *DashboardController) GetKPIMetrics(c *gin.Context) {
	dashboardData, err := dc.dashboardService.GetDashboardData(c.Request.Context())
	if err != nil {
		dc.logger.Errorw("Failed to get KPI metrics", "error", err)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dashboardData.KPIs)
}

// GetResourceDistribution 获取资源分布数据
// @Summary 获取资源分布数据
// @Description 获取多云资源分布数据
// @Tags dashboard
// @Produce json
// @Success 200 {object} common.Response{data=[]dto.MultiCloudResourceData}
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @RouterIgnore /api/dashboard/resources/distribution [get]
func (dc *DashboardController) GetResourceDistribution(c *gin.Context) {
	dashboardData, err := dc.dashboardService.GetDashboardData(c.Request.Context())
	if err != nil {
		dc.logger.Errorw("Failed to get resource distribution", "error", err)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dashboardData.MultiCloudResources)
}

// GetResourceHealth 获取资源健康状态
// @Summary 获取资源健康状态
// @Description 获取资源健康状态数据
// @Tags dashboard
// @Produce json
// @Success 200 {object} common.Response{data=[]dto.ResourceHealthData}
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @RouterIgnore /api/dashboard/resources/health [get]
func (dc *DashboardController) GetResourceHealth(c *gin.Context) {
	dashboardData, err := dc.dashboardService.GetDashboardData(c.Request.Context())
	if err != nil {
		dc.logger.Errorw("Failed to get resource health", "error", err)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dashboardData.ResourceHealth)
}
