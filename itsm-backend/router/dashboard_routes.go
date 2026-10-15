package router

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"itsm-backend/common"
	"itsm-backend/handlers"
	changeHandler "itsm-backend/handlers/change"
	problemHandler "itsm-backend/handlers/problem"
	ticketHandler "itsm-backend/handlers/ticket"
	"itsm-backend/middleware"
)

// ---- Dashboard 默认布局/组件/配置辅助函数 ----

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

// DashboardConfigStore 承载仪表盘个性化配置的持久化契约。
// 生产实现见 service.NewDashboardConfigStore（落 system_configs）。
type DashboardConfigStore interface {
	LoadDashboardConfig(ctx context.Context, tenantID, userID int) (map[string]any, error)
	SaveDashboardConfig(ctx context.Context, tenantID, userID int, cfg map[string]any) error
	LoadDashboardLayout(ctx context.Context, tenantID, userID int) (map[string]any, error)
	SaveDashboardLayout(ctx context.Context, tenantID, userID int, layout map[string]any) error
	LoadDashboardWidgets(ctx context.Context, tenantID, userID int) ([]map[string]any, error)
	SaveDashboardWidgets(ctx context.Context, tenantID, userID int, widgets []map[string]any) error
	DeleteDashboardWidgets(ctx context.Context, tenantID, userID int) error
}

// dashboardCtx 取当前租户与用户 ID。
//
// 仪表盘配置是**按用户**存取的（每个用户有自己的布局与部件集），
// 因此必须取 user_id，不能像其他 dashboard 端点那样只取 tenant。
func dashboardCtx(c *gin.Context) (tenantID, userID int, ok bool) {
	tid, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return 0, 0, false
	}
	t, valid := tid.(int)
	if !valid || t == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return 0, 0, false
	}
	uid, _ := c.Get("user_id")
	u, _ := uid.(int)
	// user_id 缺失（非 JWT 上下文）时退化为租户级共享配置，
	// 保证配置类端点在任何已鉴权上下文下都可用，不因缺字段直接 401。
	return t, u, true
}

// mergeWidgetByID 把一个部件并入现有清单：同 id 覆盖，否则追加。
// 保持顺序稳定，便于前端 diff 对比。
func mergeWidgetByID(existing []map[string]any, incoming map[string]any) []map[string]any {
	id, _ := incoming["id"].(string)
	out := make([]map[string]any, 0, len(existing)+1)
	replaced := false
	for _, w := range existing {
		if wid, _ := w["id"].(string); id != "" && wid == id {
			out = append(out, incoming)
			replaced = true
			continue
		}
		out = append(out, w)
	}
	if !replaced {
		out = append(out, incoming)
	}
	return out
}

// SetupDashboardRoutes 注册仪表盘域路由。
// fallback ticketHandler：当 TicketHandler 未装配时 /stats/tickets 回退到 DashboardHandler.GetStats。
//
// 权限资源名统一用 dashboard（read|write），必须与 RolePermissions 里的 dashboard 定义对齐；
// 用 report/widget 等资源名会让 DBOnly 模式下非 admin/manager 角色对仪表盘端点全量 403。
func SetupDashboardRoutes(tenant *gin.RouterGroup, h *handlers.DashboardHandler, ticketHandler *ticketHandler.Handler, cfgStore DashboardConfigStore) {
	dashboard := tenant.Group("/dashboard")
	{
		// 别名：/api/v1/dashboard 直接返回 overview 数据（前端默认调用）
		dashboard.GET("", middleware.RequirePermission("dashboard", "read"), h.GetOverview)
		dashboard.GET("/overview", middleware.RequirePermission("dashboard", "read"), h.GetOverview)

		// ---- 配置 / 布局 / 部件：落库（system_configs），不再是硬编码回显 ----
		// cfgStore 为 nil 时（未装配/降级）回落默认值，读写均显式标注，
		// 不再出现"POST 成功但重启丢失"的假成功。
		dashboard.GET("/config", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			tid, uid, ok := dashboardCtx(c)
			if !ok {
				return
			}
			if cfgStore != nil {
				cfg, err := cfgStore.LoadDashboardConfig(c.Request.Context(), tid, uid)
				if err != nil {
					common.FailWithErr(c, err, "读取仪表盘配置失败")
					return
				}
				if cfg != nil {
					common.Success(c, cfg)
					return
				}
			}
			common.Success(c, defaultDashboardConfig())
		})
		dashboard.POST("/config", middleware.RequirePermission("dashboard", "admin"), func(c *gin.Context) {
			tid, uid, ok := dashboardCtx(c)
			if !ok {
				return
			}
			var payload map[string]any
			if err := c.ShouldBindJSON(&payload); err != nil {
				common.ParamErrorWithErr(c, err, "请求参数错误")
				return
			}
			if cfgStore == nil {
				common.Fail(c, common.ServiceUnavailableCode, "仪表盘配置存储未就绪，无法保存")
				return
			}
			if err := cfgStore.SaveDashboardConfig(c.Request.Context(), tid, uid, payload); err != nil {
				common.FailWithErr(c, err, "保存仪表盘配置失败")
				return
			}
			common.Success(c, payload)
		})

		dashboard.GET("/layout", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			tid, uid, ok := dashboardCtx(c)
			if !ok {
				return
			}
			if cfgStore != nil {
				layout, err := cfgStore.LoadDashboardLayout(c.Request.Context(), tid, uid)
				if err != nil {
					common.FailWithErr(c, err, "读取仪表盘布局失败")
					return
				}
				if layout != nil {
					common.Success(c, layout)
					return
				}
			}
			common.Success(c, defaultDashboardLayout())
		})
		dashboard.POST("/layout", middleware.RequirePermission("dashboard", "admin"), func(c *gin.Context) {
			tid, uid, ok := dashboardCtx(c)
			if !ok {
				return
			}
			var payload map[string]any
			if err := c.ShouldBindJSON(&payload); err != nil {
				common.ParamErrorWithErr(c, err, "请求参数错误")
				return
			}
			if cfgStore == nil {
				common.Fail(c, common.ServiceUnavailableCode, "仪表盘配置存储未就绪，无法保存")
				return
			}
			if err := cfgStore.SaveDashboardLayout(c.Request.Context(), tid, uid, payload); err != nil {
				common.FailWithErr(c, err, "保存仪表盘布局失败")
				return
			}
			common.Success(c, payload)
		})

		// 可用部件集是**内置只读**的（产品定义部件类型清单），保持返回默认集。
		dashboard.GET("/widgets/available", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Success(c, defaultDashboardWidgets())
		})

		// 用户自定义部件：落库。未保存过时回落内置默认集。
		dashboard.GET("/widgets", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			tid, uid, ok := dashboardCtx(c)
			if !ok {
				return
			}
			if cfgStore != nil {
				widgets, err := cfgStore.LoadDashboardWidgets(c.Request.Context(), tid, uid)
				if err != nil {
					common.FailWithErr(c, err, "读取仪表盘部件失败")
					return
				}
				if widgets != nil {
					common.Success(c, widgets)
					return
				}
			}
			common.Success(c, defaultDashboardWidgets())
		})
		dashboard.POST("/widgets", middleware.RequirePermission("dashboard", "admin"), func(c *gin.Context) {
			tid, uid, ok := dashboardCtx(c)
			if !ok {
				return
			}
			var payload map[string]any
			if err := c.ShouldBindJSON(&payload); err != nil {
				common.ParamErrorWithErr(c, err, "请求参数错误")
				return
			}
			if cfgStore == nil {
				common.Fail(c, common.ServiceUnavailableCode, "仪表盘配置存储未就绪，无法保存")
				return
			}
			if payload["id"] == nil || payload["id"] == "" {
				payload["id"] = "custom_widget"
			}
			// 追加语义：先读现有清单再合并，避免整表覆盖掉其他部件。
			existing, err := cfgStore.LoadDashboardWidgets(c.Request.Context(), tid, uid)
			if err != nil {
				common.FailWithErr(c, err, "读取仪表盘部件失败")
				return
			}
			merged := mergeWidgetByID(existing, payload)
			if err := cfgStore.SaveDashboardWidgets(c.Request.Context(), tid, uid, merged); err != nil {
				common.FailWithErr(c, err, "保存仪表盘部件失败")
				return
			}
			common.Success(c, gin.H{"widget": payload})
		})
		dashboard.DELETE("/widgets/:widget_id", middleware.RequirePermission("dashboard", "admin"), func(c *gin.Context) {
			tid, uid, ok := dashboardCtx(c)
			if !ok {
				return
			}
			if cfgStore == nil {
				common.Fail(c, common.ServiceUnavailableCode, "仪表盘配置存储未就绪，无法删除")
				return
			}
			widgets, err := cfgStore.LoadDashboardWidgets(c.Request.Context(), tid, uid)
			if err != nil {
				common.FailWithErr(c, err, "读取仪表盘部件失败")
				return
			}
			target := c.Param("widget_id")
			kept := make([]map[string]any, 0, len(widgets))
			found := false
			for _, w := range widgets {
				if id, _ := w["id"].(string); id == target {
					found = true
					continue
				}
				kept = append(kept, w)
			}
			if !found {
				common.Fail(c, common.NotFoundCode, "部件不存在")
				return
			}
			if err := cfgStore.SaveDashboardWidgets(c.Request.Context(), tid, uid, kept); err != nil {
				common.FailWithErr(c, err, "删除仪表盘部件失败")
				return
			}
			common.Success(c, gin.H{"deletedId": target})
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
		// 注：原 GET /dashboard/metrics/performance 与 /dashboard/metrics/usage 已摘除。
		// 二者此前恒返回全 0 硬编码（loadTime/renderTime/memoryUsage、totalViews/
		// uniqueUsers/avgSessionDuration），而仓库内并没有前端性能采集与埋点数据源
		// —— 声称有这些指标而恒为 0，比没有这个端点更容易误导。
		// 前端 DashboardAPI.getPerformanceMetrics/getUsageStats 全仓零生产调用
		// （仅自身单测引用），摘除不影响任何页面。
		// 待接入真实 APM/埋点后再恢复，届时应从采集端读数而非硬编码。
		// 别名：/dashboard/metrics → 通用 stats（前端默认 fetch 路径）
		dashboard.GET("/metrics", middleware.RequirePermission("dashboard", "read"), h.GetStats)
		dashboard.GET("/ticket-trend", middleware.RequirePermission("dashboard", "read"), h.GetTicketTrend)
		dashboard.GET("/incident-distribution", middleware.RequirePermission("dashboard", "read"), h.GetIncidentDistribution)
		dashboard.GET("/sla-data", middleware.RequirePermission("dashboard", "read"), h.GetSLAData)
		dashboard.GET("/satisfaction-data", middleware.RequirePermission("dashboard", "read"), h.GetSatisfactionData)
		dashboard.GET("/quick-actions", middleware.RequirePermission("dashboard", "read"), h.GetQuickActions)
		dashboard.GET("/recent-activities", middleware.RequirePermission("dashboard", "read"), h.GetRecentActivities)
		// ---- 报表生成/导出：当前版本未实现，改为显式 501 ----
		// 此前这组端点会"成功"返回一个 id=0 的假报表对象、downloadUrl=""、
		// 以及下载接口返回纯文本 "report is not generated yet"（HTTP 200）。
		// 调用方无法区分"生成成功"与"根本没实现"，属于典型的假成功口径。
		// 现在统一回 501（能力未实现），语义与 capability registry 一致。
		// 报表的**真实**能力在 /api/v1/reports/* （见 SetupReportsRoutes，接各域真实统计）。
		dashboard.GET("/reports", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Fail(c, common.NotImplementedCode, "报表列表与生成在当前版本未实现，请使用 /api/v1/reports/*")
		})
		dashboard.POST("/reports/:report_type", middleware.RequirePermission("dashboard", "admin"), func(c *gin.Context) {
			common.Fail(c, common.NotImplementedCode, "自定义报表生成在当前版本未实现，请使用 /api/v1/reports/*")
		})
		dashboard.GET("/reports/:report_id/download", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Fail(c, common.NotImplementedCode, "报表下载在当前版本未实现，请使用各报表页的 CSV 导出")
		})
		dashboard.POST("/export", middleware.RequirePermission("dashboard", "admin"), func(c *gin.Context) {
			common.Fail(c, common.NotImplementedCode, "仪表盘导出在当前版本未实现，请使用各报表页的 CSV 导出")
		})
		// 模板集是**内置只读**的（产品提供的预置布局），保持返回内置模板。
		dashboard.GET("/templates", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Success(c, []gin.H{defaultDashboardTemplate()})
		})
		dashboard.POST("/templates/:template_id/apply", middleware.RequirePermission("dashboard", "admin"), func(c *gin.Context) {
			tid, uid, ok := dashboardCtx(c)
			if !ok {
				return
			}
			// 应用模板 = 把内置布局写入该用户的持久化配置（真实生效，不再是空回显）。
			if cfgStore == nil {
				common.Fail(c, common.ServiceUnavailableCode, "仪表盘配置存储未就绪，无法应用模板")
				return
			}
			layout := defaultDashboardLayout()
			if err := cfgStore.SaveDashboardLayout(c.Request.Context(), tid, uid, layout); err != nil {
				common.FailWithErr(c, err, "应用仪表盘模板失败")
				return
			}
			common.Success(c, gin.H{"templateId": c.Param("template_id"), "layout": layout})
		})
	}
}

// SetupReportsRoutes 注册报表路由。权限资源名同样收敛到 dashboard。
//
// 各报表必须接**本域真实**的统计/趋势端点。此前 problems 与 changes 都挂在
// h.GetTicketTrend（工单趋势）上，属于语义错配：报表名与返回数据不是同一业务对象。
// 这里改为转发到问题域 /problems/trend 与变更域 /changes/stats 的实现。
func SetupReportsRoutes(
	tenant *gin.RouterGroup,
	h *handlers.DashboardHandler,
	problemHandler *problemHandler.Handler,
	changeHandler *changeHandler.Handler,
) {
	reports := tenant.Group("/reports")
	{
		reports.GET("", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			common.Success(c, gin.H{
				"items": []gin.H{
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
		reports.GET("/sla", middleware.RequirePermission("dashboard", "read"), h.GetSLAData)

		// 问题报表：接问题域真实趋势（problem 未装配时回退工单趋势并显式标注）。
		reports.GET("/problems", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			if problemHandler == nil {
				c.Set("reports_fallback", "problem")
				h.GetTicketTrend(c)
				return
			}
			problemHandler.GetTrends(c)
		})

		// 变更报表：接变更域真实统计（change 未装配时回退工单趋势并显式标注）。
		reports.GET("/changes", middleware.RequirePermission("dashboard", "read"), func(c *gin.Context) {
			if changeHandler == nil {
				c.Set("reports_fallback", "change")
				h.GetTicketTrend(c)
				return
			}
			changeHandler.GetStats(c)
		})
	}
}
