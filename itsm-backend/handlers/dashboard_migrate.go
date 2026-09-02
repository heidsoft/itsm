package handlers

import (
	"itsm-backend/common"

	"github.com/gin-gonic/gin"
)

// GetDashboardData 获取仪表盘数据
// @Summary 获取仪表盘数据
// @Description 获取仪表盘KPI指标、资源分布和健康状态数据
// @Tags dashboard
// @Produce json
// @Success 200 {object} common.Response{data=dto.DashboardResponse}
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/dashboard [get]
func (h *DashboardHandler) GetDashboardData(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID <= 0 {
		common.Fail(c, common.UnauthorizedCode, "租户信息缺失")
		return
	}
	dashboardData, err := h.dashboardService.GetDashboardData(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get dashboard data", "error", err)
		common.Fail(c, common.InternalErrorCode, "获取仪表盘数据失败")
		return
	}

	common.Success(c, dashboardData)
}

// GetResourceDistribution 获取资源分布数据
// @Summary 获取资源分布数据
// @Description 获取多云资源分布数据
// @Tags dashboard
// @Produce json
// @Success 200 {object} common.Response{data=[]dto.MultiCloudResourceData}
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/dashboard/resources/distribution [get]
func (h *DashboardHandler) GetResourceDistribution(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID <= 0 {
		common.Fail(c, common.UnauthorizedCode, "租户信息缺失")
		return
	}
	dashboardData, err := h.dashboardService.GetDashboardData(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get resource distribution", "error", err)
		common.Fail(c, common.InternalErrorCode, "获取仪表盘数据失败")
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
// @Router /api/dashboard/resources/health [get]
func (h *DashboardHandler) GetResourceHealth(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID <= 0 {
		common.Fail(c, common.UnauthorizedCode, "租户信息缺失")
		return
	}
	dashboardData, err := h.dashboardService.GetDashboardData(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get resource health", "error", err)
		common.Fail(c, common.InternalErrorCode, "获取仪表盘数据失败")
		return
	}

	common.Success(c, dashboardData.ResourceHealth)
}
