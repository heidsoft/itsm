// Package menubaseline provides the canonical menu baseline shared by seeder and runtime services.
// 菜单基线是系统菜单的单一事实来源，供种子数据初始化和运行时导出/对比使用。
package menubaseline

// MenuSpec 菜单规格：ParentPath 为父菜单的 Path（用于反查父菜单 ID）。
type MenuSpec struct {
	Name           string
	Path           string
	Icon           string
	ParentPath     string
	PermissionCode string
	SortOrder      int
	Description    string
}

// MenuDefinitions 返回完整菜单基线。
// 菜单规格与前端 menu-config.ts 的 getMenuConfig() 保持一致：
// - 服务运营 / 服务保障 / 报告分析 / 自动化 / AI / 扩展模块 / MSP与发布
// - 系统管理（admin 域，路径以 /admin 开头由 buildMenuTree 归到 admin 域）
func MenuDefinitions() []MenuSpec {
	return []MenuSpec{
		// ===== 顶级主菜单（parent_path 为空） =====
		{Name: "服务台", Path: "/dashboard", Icon: "LayoutDashboard", PermissionCode: "", SortOrder: 10, Description: "服务台概览"},
		{Name: "服务请求", Path: "/service-requests", Icon: "FileText", ParentPath: "", PermissionCode: "ticket:read", SortOrder: 20, Description: "服务请求管理"},
		{Name: "工单管理", Path: "/tickets", Icon: "FileText", ParentPath: "", PermissionCode: "ticket:read", SortOrder: 23, Description: "工单管理"},
		{Name: "我的请求", Path: "/my-requests", Icon: "User", ParentPath: "", PermissionCode: "ticket:read", SortOrder: 25, Description: "我的服务请求"},
		{Name: "事件管理", Path: "/incidents", Icon: "AlertCircle", ParentPath: "", PermissionCode: "incident:read", SortOrder: 30, Description: "事件管理"},
		{Name: "NOC工作台", Path: "/noc", Icon: "Activity", ParentPath: "", PermissionCode: "incident:read", SortOrder: 35, Description: "重大事件作战室"},
		{Name: "问题管理", Path: "/problems", Icon: "HelpCircle", ParentPath: "", PermissionCode: "problem:read", SortOrder: 40, Description: "问题管理"},
		{Name: "变更管理", Path: "/changes", Icon: "BarChart3", ParentPath: "", PermissionCode: "change:read", SortOrder: 50, Description: "变更管理"},
		{Name: "知识库", Path: "/knowledge", Icon: "Book", ParentPath: "", PermissionCode: "knowledge:read", SortOrder: 60, Description: "知识库管理"},
		{Name: "邮件报障", Path: "/email-intake", Icon: "Inbox", ParentPath: "", PermissionCode: "email_intake:read", SortOrder: 65, Description: "AI 邮件智能报障"},
		{Name: "服务目录", Path: "/service-catalog", Icon: "BookOpen", ParentPath: "", PermissionCode: "service:read", SortOrder: 70, Description: "服务目录"},
		{Name: "CMDB", Path: "/cmdb", Icon: "Database", ParentPath: "", PermissionCode: "cmdb:read", SortOrder: 75, Description: "配置管理数据库"},
		{Name: "资产管理", Path: "/assets", Icon: "Monitor", ParentPath: "", PermissionCode: "asset:read", SortOrder: 80, Description: "IT 资产管理"},
		{Name: "SLA 管理", Path: "/sla", Icon: "Calendar", ParentPath: "", PermissionCode: "sla:read", SortOrder: 90, Description: "SLA 监控与配置"},
		{Name: "工作流", Path: "/workflow", Icon: "GitMerge", ParentPath: "", PermissionCode: "workflow:read", SortOrder: 100, Description: "工作流自动化"},
		{Name: "AI 助手", Path: "/ai/chat", Icon: "Bot", ParentPath: "", PermissionCode: "ai:read", SortOrder: 110, Description: "AI 助手"},
		{Name: "待我审批", Path: "/approvals/pending", Icon: "CheckCircle", ParentPath: "", PermissionCode: "approval:read", SortOrder: 115, Description: "待我审批"},
		{Name: "客户管理", Path: "/msp", Icon: "Building", ParentPath: "", PermissionCode: "msp:read", SortOrder: 120, Description: "客户管理 (MSP)"},
		{Name: "发布管理", Path: "/releases", Icon: "Rocket", ParentPath: "", PermissionCode: "release:read", SortOrder: 130, Description: "发布管理"},

		// ===== 顶级扩展业务菜单（2026-09-26 补齐） =====
		{Name: "报表中心", Path: "/reports", Icon: "PieChart", ParentPath: "", PermissionCode: "report:read", SortOrder: 140, Description: "ITSM 业务报表中心"},
		{Name: "标准变更库", Path: "/standard-changes", Icon: "BookCopy", ParentPath: "", PermissionCode: "change:read", SortOrder: 150, Description: "标准变更模板库"},
		{Name: "持续改进", Path: "/improvements", Icon: "TrendingUp", ParentPath: "", PermissionCode: "problem:read", SortOrder: 155, Description: "持续改进跟踪"},
		{Name: "应用市场", Path: "/marketplace", Icon: "Store", ParentPath: "", PermissionCode: "marketplace:read", SortOrder: 160, Description: "连接器 / 技能 / 插件市场"},
		{Name: "我的应用", Path: "/installations", Icon: "Package", ParentPath: "", PermissionCode: "marketplace:read", SortOrder: 165, Description: "已安装的扩展应用"},
		{Name: "项目管理", Path: "/projects", Icon: "GanttChart", ParentPath: "", PermissionCode: "project:read", SortOrder: 170, Description: "项目管理"},
		{Name: "应用与服务", Path: "/applications", Icon: "Layers", ParentPath: "", PermissionCode: "application:read", SortOrder: 175, Description: "应用系统与微服务管理"},

		// ===== 顶级独立业务条线菜单 =====
		{Name: "审计日志", Path: "/audit-logs", Icon: "Shield", ParentPath: "", PermissionCode: "audit:read", SortOrder: 210, Description: "审计日志"},
		{Name: "通知配置", Path: "/notifications", Icon: "Bell", ParentPath: "", PermissionCode: "notification:read", SortOrder: 212, Description: "通知配置"},

		// ===== 子菜单：工单管理 =====
		{Name: "工单类型", Path: "/tickets/types", Icon: "ClipboardList", ParentPath: "/tickets", PermissionCode: "ticket_type:read", SortOrder: 21},
		{Name: "工单统计", Path: "/tickets/analytics", Icon: "BarChart3", ParentPath: "/tickets", PermissionCode: "ticket:read", SortOrder: 22, Description: "工单统计视图"},
		{Name: "工单仪表盘", Path: "/tickets/dashboard", Icon: "LayoutDashboard", ParentPath: "/tickets", PermissionCode: "ticket:read", SortOrder: 23, Description: "工单综合仪表盘"},
		{Name: "我的抄送", Path: "/tickets/cc", Icon: "Mail", ParentPath: "/tickets", PermissionCode: "ticket:read", SortOrder: 24, Description: "抄送给我的工单"},
		{Name: "工单模板", Path: "/tickets/templates", Icon: "Copy", ParentPath: "/tickets", PermissionCode: "ticket:write", SortOrder: 25, Description: "工单模板管理"},

		// ===== 子菜单：事件管理 =====
		{Name: "新建事件", Path: "/incidents/create", Icon: "Plus", ParentPath: "/incidents", PermissionCode: "incident:write", SortOrder: 32},

		// ===== 子菜单：问题管理 =====
		{Name: "已知错误", Path: "/problems/known-errors", Icon: "AlertCircle", ParentPath: "/problems", PermissionCode: "problem:read", SortOrder: 42},
		{Name: "问题趋势", Path: "/problems/trends", Icon: "TrendingUp", ParentPath: "/problems", PermissionCode: "problem:read", SortOrder: 43, Description: "问题趋势分析"},

		// ===== 子菜单：变更管理 =====
		{Name: "新建变更", Path: "/changes/new", Icon: "Plus", ParentPath: "/changes", PermissionCode: "change:write", SortOrder: 52},
		{Name: "实施后审查", Path: "/changes/pirs", Icon: "ClipboardCheck", ParentPath: "/changes", PermissionCode: "change:read", SortOrder: 53, Description: "变更实施后审查 (PIR)"},

		// ===== 子菜单：知识库 =====
		{Name: "新建文章", Path: "/knowledge/articles/new", Icon: "Plus", ParentPath: "/knowledge", PermissionCode: "knowledge:write", SortOrder: 63},
		{Name: "知识审核", Path: "/knowledge/reviews", Icon: "CheckSquare", ParentPath: "/knowledge", PermissionCode: "knowledge:write", SortOrder: 64, Description: "知识库文章审核"},

		// ===== 子菜单：邮件报障 =====
		{Name: "客户资料", Path: "/email-intake/customers", Icon: "Users", ParentPath: "/email-intake", PermissionCode: "customer_master:read", SortOrder: 652},
		{Name: "支持合同", Path: "/email-intake/contracts", Icon: "FileText", ParentPath: "/email-intake", PermissionCode: "support_contract:read", SortOrder: 653},
		{Name: "来源组织", Path: "/email-intake/sources", Icon: "Globe", ParentPath: "/email-intake", PermissionCode: "customer_master:read", SortOrder: 654},
		{Name: "值班排班", Path: "/email-intake/on-call", Icon: "Clock", ParentPath: "/email-intake", PermissionCode: "on_call:read", SortOrder: 655},

		// ===== 子菜单：服务目录 =====
		{Name: "待我审批-目录", Path: "/service-catalog/approvals", Icon: "CheckCircle", ParentPath: "/service-catalog", PermissionCode: "service:read", SortOrder: 72},

		// ===== 子菜单：CMDB =====
		{Name: "配置项列表", Path: "/cmdb/cis", Icon: "Server", ParentPath: "/cmdb", PermissionCode: "cmdb:read", SortOrder: 751},
		{Name: "新建CI", Path: "/cmdb/cis/create", Icon: "Plus", ParentPath: "/cmdb", PermissionCode: "cmdb:write", SortOrder: 752},
		{Name: "关系管理", Path: "/cmdb/relationships", Icon: "GitBranch", ParentPath: "/cmdb", PermissionCode: "cmdb:read", SortOrder: 753},
		{Name: "拓扑图", Path: "/cmdb/topology", Icon: "Share2", ParentPath: "/cmdb", PermissionCode: "cmdb:read", SortOrder: 754},
		{Name: "云账号管理", Path: "/cmdb/cloud-accounts", Icon: "Cloud", ParentPath: "/cmdb", PermissionCode: "cmdb:write", SortOrder: 755, Description: "云平台账号管理"},
		{Name: "云资源列表", Path: "/cmdb/cloud-resources", Icon: "CloudLightning", ParentPath: "/cmdb", PermissionCode: "cmdb:read", SortOrder: 756, Description: "已发现云资源"},
		{Name: "云服务目录", Path: "/cmdb/cloud-services", Icon: "CloudDrizzle", ParentPath: "/cmdb", PermissionCode: "cmdb:read", SortOrder: 757, Description: "云服务产品目录"},
		{Name: "Service Graph", Path: "/cmdb/registry", Icon: "Compass", ParentPath: "/cmdb", PermissionCode: "cmdb:read", SortOrder: 758, Description: "Service Graph 注册表"},
		{Name: "云资源核对", Path: "/cmdb/reconciliation", Icon: "RefreshCw", ParentPath: "/cmdb", PermissionCode: "cmdb:write", SortOrder: 759, Description: "云资源与 CI 核对"},

		// ===== 子菜单：资产管理 =====
		{Name: "新建资产", Path: "/assets/new", Icon: "Plus", ParentPath: "/assets", PermissionCode: "asset:write", SortOrder: 82},
		{Name: "软件许可证", Path: "/licenses", Icon: "Key", ParentPath: "/assets", PermissionCode: "license:read", SortOrder: 83},

		// ===== 子菜单：SLA =====
		{Name: "SLA 监控", Path: "/sla-monitor", Icon: "Activity", ParentPath: "/sla", PermissionCode: "sla:read", SortOrder: 92},
		{Name: "SLA 配置", Path: "/workflow/sla", Icon: "Clock", ParentPath: "/sla", PermissionCode: "sla:write", SortOrder: 93},

		// ===== 子菜单：工作流 =====
		{Name: "流程设计器", Path: "/workflow/designer", Icon: "Edit", ParentPath: "/workflow", PermissionCode: "workflow:write", SortOrder: 102},
		{Name: "流程实例", Path: "/workflow/instances", Icon: "Play", ParentPath: "/workflow", PermissionCode: "workflow:read", SortOrder: 103},
		{Name: "版本管理", Path: "/workflow/versions", Icon: "History", ParentPath: "/workflow", PermissionCode: "workflow:write", SortOrder: 104},
		{Name: "监控仪表盘", Path: "/workflow/dashboard", Icon: "Activity", ParentPath: "/workflow", PermissionCode: "workflow:read", SortOrder: 105},
		{Name: "节点瓶颈分析", Path: "/workflow/bottlenecks", Icon: "BarChart3", ParentPath: "/workflow", PermissionCode: "workflow:read", SortOrder: 106},
		{Name: "自动化规则", Path: "/workflow/automation", Icon: "Zap", ParentPath: "/workflow", PermissionCode: "workflow:write", SortOrder: 107},
		{Name: "审批中心", Path: "/approvals", Icon: "CheckSquare", ParentPath: "/workflow", PermissionCode: "approval:read", SortOrder: 108},
		{Name: "操作日志", Path: "/workflow/audit", Icon: "ClipboardList", ParentPath: "/workflow", PermissionCode: "audit:read", SortOrder: 109},

		// ===== 子菜单：AI 助手 =====
		{Name: "AI 创建工单", Path: "/tickets/ai-create", Icon: "Sparkles", ParentPath: "/ai/chat", PermissionCode: "ai:read", SortOrder: 112},
		{Name: "AI 评估与审计", Path: "/ai/audit", Icon: "ShieldCheck", ParentPath: "/ai/chat", PermissionCode: "ai:read", SortOrder: 113},
		{Name: "AI 审批", Path: "/ai/approval", Icon: "ShieldAlert", ParentPath: "/ai/chat", PermissionCode: "ai:read", SortOrder: 114},

		// ===== 子菜单：MSP 客户管理 =====
		{Name: "客户管理子页", Path: "/msp/management", Icon: "Settings", ParentPath: "/msp", PermissionCode: "msp:write", SortOrder: 122},

		// ===== 子菜单：发布管理 =====
		{Name: "新建发布", Path: "/releases/new", Icon: "Plus", ParentPath: "/releases", PermissionCode: "release:write", SortOrder: 132},

		// ===== 顶级管理菜单 =====
		{Name: "系统管理", Path: "/admin", Icon: "Settings", ParentPath: "", PermissionCode: "system:write", SortOrder: 200, Description: "系统管理"},

		// ===== 子菜单：系统管理 =====
		{Name: "系统概览", Path: "/admin/overview", Icon: "LayoutDashboard", ParentPath: "/admin", PermissionCode: "system:write", SortOrder: 201},
		{Name: "用户管理", Path: "/admin/users", Icon: "Users", ParentPath: "/admin", PermissionCode: "user:read", SortOrder: 210},
		{Name: "角色管理", Path: "/admin/roles", Icon: "Shield", ParentPath: "/admin", PermissionCode: "role:read", SortOrder: 220},
		{Name: "组管理", Path: "/admin/groups", Icon: "Users", ParentPath: "/admin", PermissionCode: "group:read", SortOrder: 230},
		{Name: "租户管理", Path: "/admin/tenants", Icon: "Building", ParentPath: "/admin", PermissionCode: "system:write", SortOrder: 235},
		{Name: "部门管理", Path: "/admin/departments", Icon: "Building", ParentPath: "/admin", PermissionCode: "department:read", SortOrder: 240},
		{Name: "团队管理", Path: "/admin/teams", Icon: "Users", ParentPath: "/admin", PermissionCode: "team:read", SortOrder: 250},
		{Name: "评审组管理", Path: "/admin/change-review", Icon: "Users", ParentPath: "/admin", PermissionCode: "change:read", SortOrder: 255},
		{Name: "工单分类", Path: "/admin/ticket-categories", Icon: "Tag", ParentPath: "/admin", PermissionCode: "ticket_category:update", SortOrder: 260},
		{Name: "工单分配规则", Path: "/admin/tickets/assignment-rules", Icon: "GitBranch", ParentPath: "/admin", PermissionCode: "ticket:read", SortOrder: 265},
		{Name: "自动化规则", Path: "/admin/tickets/automation-rules", Icon: "Zap", ParentPath: "/admin", PermissionCode: "ticket:read", SortOrder: 270},
		{Name: "审批链", Path: "/admin/approval-chains", Icon: "Link", ParentPath: "/admin", PermissionCode: "approval:write", SortOrder: 275},
		{Name: "权限管理", Path: "/admin/permissions", Icon: "Lock", ParentPath: "/admin", PermissionCode: "role:write", SortOrder: 280},
		{Name: "连接器/插件市场", Path: "/admin/connectors", Icon: "Plug", ParentPath: "/admin", PermissionCode: "connector:write", SortOrder: 285},
		{Name: "技能注册表", Path: "/admin/skills", Icon: "Wand2", ParentPath: "/admin", PermissionCode: "marketplace:write", SortOrder: 287, Description: "AI 技能清单与 manifest 管理"},
		{Name: "向量存储配置", Path: "/admin/vector-store", Icon: "Database", ParentPath: "/admin", PermissionCode: "system:read", SortOrder: 290},
		{Name: "系统配置", Path: "/admin/system-config", Icon: "Settings", ParentPath: "/admin", PermissionCode: "system:read", SortOrder: 295},
		{Name: "全局标签", Path: "/admin/tags", Icon: "Tags", ParentPath: "/admin", PermissionCode: "system:read", SortOrder: 298, Description: "全局标签管理"},
		{Name: "CMDB 类型", Path: "/admin/cmdb-types", Icon: "Database", ParentPath: "/admin", PermissionCode: "cmdb:write", SortOrder: 315},
		{Name: "升级规则", Path: "/admin/escalation-rules", Icon: "AlertTriangle", ParentPath: "/admin", PermissionCode: "sla:write", SortOrder: 320},
		{Name: "升级矩阵", Path: "/admin/escalation-matrices", Icon: "TrendingUp", ParentPath: "/admin", PermissionCode: "sla:read", SortOrder: 325},
		{Name: "SLA 模板", Path: "/admin/sla-templates", Icon: "Layers", ParentPath: "/admin", PermissionCode: "sla:write", SortOrder: 330},
		{Name: "服务目录管理", Path: "/admin/service-catalogs", Icon: "Boxes", ParentPath: "/admin", PermissionCode: "service_catalog:read", SortOrder: 335},
		{Name: "SLA 定义", Path: "/admin/sla-definitions", Icon: "Clock", ParentPath: "/admin", PermissionCode: "sla:write", SortOrder: 340},
		{Name: "菜单管理", Path: "/admin/menus", Icon: "Menu", ParentPath: "/admin", PermissionCode: "system:write", SortOrder: 345},
		{Name: "工作流配置", Path: "/admin/workflows", Icon: "GitBranch", ParentPath: "/admin", PermissionCode: "workflow:write", SortOrder: 350},
	}
}
