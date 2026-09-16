package router

import (
	"time"

	"github.com/gin-gonic/gin"

	"itsm-backend/common"
	"itsm-backend/handlers"
	ticketHandler "itsm-backend/handlers/ticket"
	"itsm-backend/middleware"
)

// ---- Dashboard 默认布局/组件/配置辅助函数 ----
// 从 router.go 顶部搬移而来，仅被 dashboard/reports 路由使用。

func defaultDashboardLayout() gin.H {
	return gin.H{
		"cols":             12,
		"rows":             24,
		"margin":           []int{16, 16},
		"containerPadding": []int{16, 16},
		"rowHeight":        80,
		"isDraggable":      true,
		"isResizable":      true,
	}
}

func defaultDashboardWidgets() []gin.H {
	return []gin.H{
		{
			"id":          "ticket_overview",
			"type":        "metric",
			"title":       "工单总览",
			"description": "工单数量与状态概览",
			"position":    gin.H{"x": 0, "y": 0, "w": 3, "h": 2},
			"config":      gin.H{"showTitle": true, "showBorder": true, "metric": "total", "unit": "个"},
			"dataSource":  "tickets",
			"isVisible":   true,
		},
		{
			"id":          "ticket_trend",
			"type":        "chart",
			"title":       "工单趋势",
			"description": "近期工单创建与解决趋势",
			"position":    gin.H{"x": 3, "y": 0, "w": 6, "h": 4},
			"config":      gin.H{"showTitle": true, "showBorder": true, "chartType": "line", "xAxis": "date", "yAxis": "count"},
			"dataSource":  "ticket_trend",
			"isVisible":   true,
		},
		{
			"id":          "sla_status",
			"type":        "progress",
			"title":       "SLA 达成率",
			"description": "服务级别协议履约情况",
			"position":    gin.H{"x": 9, "y": 0, "w": 3, "h": 2},
			"config":      gin.H{"showTitle": true, "showBorder": true, "metric": "slaCompliance", "unit": "%"},
			"dataSource":  "sla",
			"isVisible":   true,
		},
	}
}

func defaultDashboardConfig() gin.H {
	now := time.Now().Format(time.RFC3339)
	return gin.H{
		"id":          1,
		"name":        "默认仪表盘",
		"description": "系统默认运维视图",
		"isDefault":   true,
		"isPublic":    false,
		"layout":      defaultDashboardLayout(),
		"widgets":     defaultDashboardWidgets(),
		"filters": []gin.H{
			{
				"id":         "time_range",
				"name":       "时间范围",
				"type":       "select",
				"field":      "timeRange",
				"options":    []gin.H{{"label": "最近7天", "value": "7d"}, {"label": "最近30天", "value": "30d"}},
				"isRequired": false,
				"isVisible":  true,
			},
		},
		"permissions": []string{},
		"createdBy":   0,
		"updatedBy":   0,
		"createdAt":   now,
		"updatedAt":   now,
		"shareSettings": gin.H{
			"isShared": false,
		},
	}
}

func defaultDashboardTemplate() gin.H {
	return gin.H{
		"id":            1,
		"name":          "ITSM 运营总览",
		"description":   "适用于服务台、SLA 与工单趋势的默认模板",
		"category":      "operations",
		"tags":          []string{"itsm", "ticket", "sla"},
		"layout":        defaultDashboardLayout(),
		"widgets":       defaultDashboardWidgets(),
		"filters":       []gin.H{},
		"isPublic":      true,
		"downloadCount": 0,
	}
}

func dashboardWidgetByID(widgetID string) gin.H {
	for _, widget := range defaultDashboardWidgets() {
		if widget["id"] == widgetID {
			return widget
		}
	}
	return gin.H{
		"id":         widgetID,
		"type":       "metric",
		"title":      widgetID,
		"position":   gin.H{"x": 0, "y": 0, "w": 3, "h": 2},
		"config":     gin.H{"showTitle": true, "showBorder": true},
		"dataSource": widgetID,
		"isVisible":  true,
	}
}

// SetupDashboardRoutes 注册仪表盘域路由。
// 从 router.go 的集中注册块抽取而来，路由路径/方法/中间件与抽取前逐行一致。
// fallback ticketHandler：当 TicketHandler 未装配时 /stats/tickets 回退到 DashboardHandler.GetStats。
//
// 2026-09-16 P0 修复：原资源名为 ("report", "read"|"update"|"create") 与 RolePermissions 不一致，
// 非 admin/manager 角色 DBOnly 模式下 dashboard 端点全量 403；端用户/服务台 agent 也无法访问仪表盘。
// 路由层统一改用 ("dashboard", "read"|"update") 与 RolePermissions 中已存在的 dashboard:read 对齐，
// create/delete 维度整合到 dashboard:write/admin；widget 资源未在 RolePermissions 中存在，
// 一并收敛到 dashboard 资源，避免路由层与权限词表双轨。
func SetupDashboardRoutes(tenant *gin.RouterGroup, h *handlers.DashboardHandler, ticketHandler *ticketHandler.Handler) {
	dashboard := tenant.Group("/dashboard")
	{
		// B5: 别名，/api/v1/dashboard 直接返回 overview 数据（前端默认调用）
		dashboard.GET("", middleware.RequirePermission("dashboard", "read"), h.GetOverview)
		dashboard.GET("/overview", middleware.RequirePermission("dashboard", "read"), h.GetOverview)
		dashboard.GET("/config", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Success(c, defaultDashboardConfig())
		})
		dashboard.POST("/config", middleware.RequirePermission("dashboard", "write"), func(c *gin.Context) {
			common.Success(c, gin.H{"success": true})
		})
		dashboard.GET("/layout", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Success(c, defaultDashboardLayout())
		})
		dashboard.POST("/layout", middleware.RequirePermission("dashboard", "write"), func(c *gin.Context) {
			common.Success(c, gin.H{"success": true})
		})
		dashboard.GET("/widgets/available", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Success(c, defaultDashboardWidgets())
		})
		dashboard.POST("/widgets", middleware.RequirePermission("dashboard", "write"), func(c *gin.Context) {
			widget := dashboardWidgetByID("custom_widget")
			var payload map[string]interface{}
			if err := c.ShouldBindJSON(&payload); err == nil {
				for key, value := range payload {
					widget[key] = value
				}
			}
			if widget["id"] == nil || widget["id"] == "" {
				widget["id"] = "custom_widget"
			}
			common.Success(c, gin.H{"widget": widget})
		})
		dashboard.GET("/widgets/:widget_id/data", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Success(c, dashboardWidgetByID(c.Param("widget_id")))
		})
		dashboard.POST("/widgets/:widget_id/refresh", middleware.RequirePermission("dashboard", "write"), func(c *gin.Context) {
			common.Success(c, dashboardWidgetByID(c.Param("widget_id")))
		})
		dashboard.PUT("/widgets/:widget_id", middleware.RequirePermission("dashboard", "write"), func(c *gin.Context) {
			widget := dashboardWidgetByID(c.Param("widget_id"))
			var payload map[string]interface{}
			if err := c.ShouldBindJSON(&payload); err == nil {
				for key, value := range payload {
					widget[key] = value
				}
			}
			common.Success(c, gin.H{"widget": widget})
		})
		dashboard.DELETE("/widgets/:widget_id", middleware.RequirePermission("dashboard", "admin"), func(c *gin.Context) {
			common.Success(c, gin.H{"success": true})
		})
		dashboard.GET("/charts/:chart_type", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Success(c, gin.H{
				"labels":   []string{},
				"datasets": []gin.H{{"label": c.Param("chart_type"), "data": []int{}}},
			})
		})
		dashboard.GET("/realtime/:data_type", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Success(c, gin.H{
				"type":      c.Param("data_type"),
				"data":      gin.H{},
				"timestamp": time.Now().Format(time.RFC3339),
			})
		})
		dashboard.GET("/stats", middleware.RequirePermission("dashboard", "read"), h.GetStats)
		if ticketHandler != nil {
			dashboard.GET("/stats/tickets", middleware.RequirePermission("dashboard", "read"), ticketHandler.GetTicketStats)
		} else {
			dashboard.GET("/stats/tickets", middleware.RequirePermission("dashboard", "read"), h.GetStats)
		}
		dashboard.GET("/stats/users", middleware.RequirePermission("dashboard", "read"), h.GetUserStats)
		dashboard.GET("/stats/system", middleware.RequirePermission("dashboard", "read"), h.GetSystemStats)
		dashboard.GET("/kpi-metrics", middleware.RequirePermission("dashboard", "read"), h.GetKPIMetrics)
		dashboard.GET("/metrics/performance", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Success(c, gin.H{
				"loadTime":      0,
				"renderTime":    0,
				"dataFetchTime": 0,
				"widgetCount":   len(defaultDashboardWidgets()),
				"memoryUsage":   0,
			})
		})
		dashboard.GET("/metrics/usage", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Success(c, gin.H{
				"totalViews":         0,
				"uniqueUsers":        0,
				"avgSessionDuration": 0,
				"mostUsedWidgets":    []gin.H{},
				"peakUsageHours":     []int{},
			})
		})
		// P1-01 别名：/dashboard/metrics → 通用 stats（前端默认 fetch 路径）
		dashboard.GET("/metrics", middleware.RequirePermission("dashboard", "read"), h.GetStats)
		dashboard.GET("/ticket-trend", middleware.RequirePermission("dashboard", "read"), h.GetTicketTrend)
		dashboard.GET("/incident-distribution", middleware.RequirePermission("dashboard", "read"), h.GetIncidentDistribution)
		dashboard.GET("/sla-data", middleware.RequirePermission("dashboard", "read"), h.GetSLAData)
		dashboard.GET("/satisfaction-data", middleware.RequirePermission("dashboard", "read"), h.GetSatisfactionData)
		dashboard.GET("/quick-actions", middleware.RequirePermission("dashboard", "read"), h.GetQuickActions)
		dashboard.GET("/recent-activities", middleware.RequirePermission("dashboard", "read"), h.GetRecentActivities)
		// /reports/* 与 /templates/* 视为高级报表生成，仍以 dashboard:read|write 收敛
		// （避免引入 DBOnly 模式下词表无 entry 的 report/widget 资源）
		dashboard.GET("/reports", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Success(c, gin.H{"reports": []gin.H{}, "total": 0, "page": 1, "pageSize": 20})
		})
		dashboard.POST("/reports/:report_type", middleware.RequirePermission("dashboard", "write"), func(c *gin.Context) {
			common.Success(c, gin.H{
				"id":         0,
				"name":       c.Param("report_type"),
				"type":       c.Param("report_type"),
				"template":   gin.H{"title": c.Param("report_type"), "sections": []gin.H{}, "filters": []gin.H{}, "timeRange": "7d", "format": "html"},
				"recipients": []string{},
				"isActive":   false,
				"createdBy":  0,
				"createdAt":  time.Now().Format(time.RFC3339),
				"updatedAt":  time.Now().Format(time.RFC3339),
			})
		})
		dashboard.GET("/reports/:report_id/download", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			c.Data(200, "text/plain; charset=utf-8", []byte("report is not generated yet"))
		})
		dashboard.POST("/export", middleware.RequirePermission("dashboard", "write"), func(c *gin.Context) {
			common.Success(c, gin.H{"downloadUrl": ""})
		})
		dashboard.GET("/templates", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Success(c, []gin.H{defaultDashboardTemplate()})
		})
		dashboard.POST("/templates", middleware.RequirePermission("dashboard", "write"), func(c *gin.Context) {
			common.Success(c, gin.H{"template": defaultDashboardTemplate()})
		})
		dashboard.POST("/templates/:template_id/apply", middleware.RequirePermission("dashboard", "write"), func(c *gin.Context) {
			common.Success(c, gin.H{"success": true, "config": defaultDashboardConfig()})
		})
	}
}

// SetupReportsRoutes 注册报表路由。
// 从 router.go 的集中注册块抽取而来；报表数据聚合复用 DashboardHandler 各查询方法。
//
// 2026-09-16 P0 修复：与 SetupDashboardRoutes 同步，资源名收敛到 dashboard（详见其上方注释）。
func SetupReportsRoutes(tenant *gin.RouterGroup, h *handlers.DashboardHandler) {
	reports := tenant.Group("/reports")
	{
		reports.GET("", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Success(c, gin.H{
				"reports": []gin.H{
					{"id": "tickets", "name": "工单报表", "path": "/reports/tickets"},
					{"id": "incidents", "name": "事件报表", "path": "/reports/incidents"},
					{"id": "problems", "name": "问题报表", "path": "/reports/problems"},
					{"id": "changes", "name": "变更报表", "path": "/reports/changes"},
					{"id": "sla", "name": "SLA报表", "path": "/reports/sla"},
					{"id": "cmdb-quality", "name": "CMDB质量报表", "path": "/reports/cmdb-quality"},
					{"id": "catalog-usage", "name": "服务目录使用报表", "path": "/reports/catalog-usage"},
				},
			})
		})
		reports.GET("/tickets", middleware.RequirePermission("dashboard", "read"), h.GetStats)
		reports.GET("/incidents", middleware.RequirePermission("dashboard", "read"), h.GetIncidentDistribution)
		reports.GET("/problems", middleware.RequirePermission("dashboard", "read"), h.GetTicketTrend)
		reports.GET("/changes", middleware.RequirePermission("dashboard", "read"), h.GetTicketTrend)
		reports.GET("/sla", middleware.RequirePermission("dashboard", "read"), h.GetSLAData)
	}
}
