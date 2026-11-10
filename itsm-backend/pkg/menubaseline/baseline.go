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
// 菜单完全由数据库驱动：seeder/ProvisionTenant 把本基线写入各租户 menus 表，
// 前端经 GET /api/v1/auth/menus 消费，不再维护静态菜单配置。
// 主桶/管理桶的划分见 service/menu_service.go 的 isAdminMenu（按顶级路径 /admin、/system、/workflow）。
func MenuDefinitions() []MenuSpec {
	return []MenuSpec{
		// ===== 顶级主菜单（parent_path 为空） =====
		{Name: "Help Desk", Path: "/dashboard", Icon: "LayoutDashboard", PermissionCode: "", SortOrder: 10, Description: "Help Desk Overview"},
		// 个人工作台：页面早已存在（数据源 GET /api/v1/workbench，要求 dashboard:read），
		// 但从未挂过菜单，0 入口 0 引用，功能对用户不可达。
		// 命名带「我的」与顶级「服务台」（运营概览）区分——审计报告 3.3-①「三个台」病灶。
		{Name: "My workbench", Path: "/workbench", Icon: "List", ParentPath: "", PermissionCode: "dashboard:read", SortOrder: 12, Description: "Personal workbench"},
		{Name: "Service Request", Path: "/service-requests", Icon: "FileText", ParentPath: "", PermissionCode: "ticket:read", SortOrder: 20, Description: "Service Request Management"},
		{Name: "Ticket Management", Path: "/tickets", Icon: "FileText", ParentPath: "", PermissionCode: "ticket:read", SortOrder: 23, Description: "Ticket Management"},
		{Name: "My Requests", Path: "/my-requests", Icon: "User", ParentPath: "", PermissionCode: "ticket:read", SortOrder: 25, Description: "My Service Request"},
		{Name: "Incidents", Path: "/incidents", Icon: "AlertCircle", ParentPath: "", PermissionCode: "incident:read", SortOrder: 30, Description: "Incidents"},
		// 命名对齐 Description「重大事件作战室」，同时消除「NOC工作台」与「我的工作台」的「台」字撞车（审计报告 3.3-①）。
		{Name: "NOC War Room", Path: "/noc", Icon: "Activity", ParentPath: "", PermissionCode: "incident:read", SortOrder: 35, Description: "Major Incident War Room"},
		{Name: "Problems", Path: "/problems", Icon: "HelpCircle", ParentPath: "", PermissionCode: "problem:read", SortOrder: 40, Description: "Problems"},
		// 图标用 GitPullRequest：变更域语义（分支合入）；原 BarChart3 与工单统计/瓶颈分析撞车（审计报告 3.3-⑧）。
		{Name: "Changes", Path: "/changes", Icon: "GitPullRequest", ParentPath: "", PermissionCode: "change:read", SortOrder: 50, Description: "Changes"},
		{Name: "Knowledge Base", Path: "/knowledge", Icon: "Book", ParentPath: "", PermissionCode: "knowledge:read", SortOrder: 60, Description: "Knowledge Base Management"},
		// 父级是「AI 邮件报障」工作台，子项却是客户资料/支持合同/来源组织（客户主数据）——
		// 父子语义不符（审计 3.3-⑨）。改名「客户与合同」：落地页仍是报障工作台
		// （页内 Tab 即涵盖客户/合同/来源组织/排班），父级名与其下主数据子项自洽。
		{Name: "Customers and Contracts", Path: "/email-intake", Icon: "Inbox", ParentPath: "", PermissionCode: "email_intake:read", SortOrder: 65, Description: "AI email fault reporting workbench and customer master data"},
		{Name: "Service Catalog", Path: "/service-catalog", Icon: "BookOpen", ParentPath: "", PermissionCode: "service:read", SortOrder: 70, Description: "Service Catalog"},
		{Name: "Configuration Management", Path: "/cmdb", Icon: "Database", ParentPath: "", PermissionCode: "cmdb:read", SortOrder: 75, Description: "Configuration Management Database (CMDB)"},
		{Name: "Asset Management", Path: "/assets", Icon: "Monitor", ParentPath: "", PermissionCode: "asset:read", SortOrder: 80, Description: "IT Asset Management"},
		{Name: "SLA Management", Path: "/sla", Icon: "Calendar", ParentPath: "", PermissionCode: "sla:read", SortOrder: 90, Description: "SLA monitoring and configuration"},
		{Name: "Workflows", Path: "/workflow", Icon: "GitMerge", ParentPath: "", PermissionCode: "workflow:read", SortOrder: 100, Description: "Workflow automation"},
		{Name: "AI Center", Path: "/ai/chat", Icon: "Bot", ParentPath: "", PermissionCode: "ai:read", SortOrder: 110, Description: "AI Assistant and AI Competence Center"},
		// 审批待办只保留一个入口：/approvals/pending 是 redirect 到 /approvals 的兼容壳，
		// 菜单直接指向 /approvals，避免「待我审批」「审批中心」两个菜单一个目的地。
		// 页面只消费 GET /bpmn/tasks（实测无任何 approval:* 调用），故按数据源取 task:read。
		// 旧码 approval:read 只授配置角色，且 /approvals 挂在 /workflow 下会落进 admin 桶
		// （前端只给管理角色渲染），导致 agent/manager/end_user 这些真正的待办处理人没有入口。
		{Name: "My To-dos", Path: "/approvals", Icon: "CheckCircle", ParentPath: "", PermissionCode: "task:read", SortOrder: 115, Description: "Awaiting my approval and to-do tasks"},
		{Name: "Customer Management", Path: "/msp", Icon: "Building", ParentPath: "", PermissionCode: "msp:read", SortOrder: 120, Description: "Customer Management (MSP)"},
		{Name: "Release Management", Path: "/releases", Icon: "Rocket", ParentPath: "", PermissionCode: "release:read", SortOrder: 130, Description: "Release Management"},

		// ===== 顶级扩展业务菜单 =====
		// 报表中心是管理/分析面：菜单与 /reports/* 接口同取 report:read。
		// 曾用 dashboard:read（该码是全角色基线，为了让服务台落地页可用），
		// 结果最终用户侧边栏也出现整套报表入口（审计遗留项 6）。
		{Name: "Report Center", Path: "/reports", Icon: "PieChart", ParentPath: "", PermissionCode: "report:read", SortOrder: 140, Description: "ITSM Business Report Center"},
		{Name: "Standard Changes Library", Path: "/standard-changes", Icon: "BookCopy", ParentPath: "", PermissionCode: "change:read", SortOrder: 150, Description: "Standard change template library"},
		{Name: "Continuous improvement", Path: "/improvements", Icon: "TrendingUp", ParentPath: "", PermissionCode: "problem:read", SortOrder: 155, Description: "Continuous improvement tracking"},
		{Name: "ApplyMarketplace", Path: "/marketplace", Icon: "Store", ParentPath: "", PermissionCode: "marketplace:read", SortOrder: 160, Description: "Connector/Skill/Plug-in Market"},
		{Name: "My application", Path: "/installations", Icon: "Package", ParentPath: "", PermissionCode: "marketplace:read", SortOrder: 165, Description: "Installed extension application"},
		{Name: "Projects", Path: "/projects", Icon: "GanttChart", ParentPath: "", PermissionCode: "project:read", SortOrder: 170, Description: "Projects"},
		{Name: "Applications and Services", Path: "/applications", Icon: "Layers", ParentPath: "", PermissionCode: "application:read", SortOrder: 175, Description: "Application system and microservice management"},

		// ===== 子菜单：应用市场 · 配置面（2026-10-25 从系统管理归位）=====
		{Name: "Connectors and plug-ins", Path: "/admin/connectors", Icon: "Plug", ParentPath: "/marketplace", PermissionCode: "connector:write", SortOrder: 10},
		{Name: "Skill Registry", Path: "/admin/skills", Icon: "Wand2", ParentPath: "/marketplace", PermissionCode: "marketplace:write", SortOrder: 20, Description: "AI skill list and manifest management"},
		// 商品目录管理：marketplace_catalog:read 只授予 sysadmin。目录是全局表
		// （无 tenant_id），租户 admin 的 marketplace:read/write 只能安装/卸载，
		// 不能上架——所以这条菜单对租户管理员不出现，而不是出现了点进去 403。
		// 与父级「应用市场」区分：这条是平台上架/版本管理控制台，不是浏览市场（D34-4）。
		{Name: "Application shelf management", Path: "/admin/marketplace", Icon: "Store", ParentPath: "/marketplace", PermissionCode: "marketplace_catalog:read", SortOrder: 30, Description: "Extended product listing, removal and version management (platform level)"},

		// ===== 顶级独立业务条线菜单 =====
		{Name: "Audit Logs", Path: "/audit-logs", Icon: "Shield", ParentPath: "", PermissionCode: "audit:read", SortOrder: 210, Description: "Audit Logs"},
		{Name: "Notification Configuration", Path: "/notifications", Icon: "Bell", ParentPath: "", PermissionCode: "notification:read", SortOrder: 212, Description: "Notification Configuration"},

		// ===== 子菜单：工单管理 =====
		{Name: "Ticket Types", Path: "/tickets/types", Icon: "ClipboardList", ParentPath: "/tickets", PermissionCode: "ticket_type:read", SortOrder: 21},
		// 页面主数据源是 GET /api/v1/tickets/stats（ticket:read，随登录者行级数据范围收窄），
		// 但同页的「深度分析」与「导出」走 POST /tickets/analytics/{deep,export}（report:read /
		// ticket:export）。菜单按最高要求的入口取 report:read：挂在 ticket:read 上时坐席
		// 点进来就是 403（实测 code 2003），页面再把 403 渲染成全 0。
		// 原「工单仪表盘」(/tickets/dashboard) 与本条目功能重复且满是假控件，已于
		// 2026-11-01 退役；存量库由 migrations/20261101_retire_ticket_dashboard_menu.sql 停用。
		{Name: "Work order statistics", Path: "/tickets/analytics", Icon: "BarChart3", ParentPath: "/tickets", PermissionCode: "report:read", SortOrder: 22, Description: "Work order statistics view"},
		{Name: "My CC", Path: "/tickets/cc", Icon: "Mail", ParentPath: "/tickets", PermissionCode: "ticket:read", SortOrder: 24, Description: "CC me on the ticket"},
		// 页面数据源是 GET /api/v1/tickets/templates，接口要求 ticket_template:read。
		// 挂在 ticket:write 上让只写自己工单的最终用户拿到入口后立刻 403（实测 code 2003）。
		{Name: "Work order template", Path: "/tickets/templates", Icon: "Copy", ParentPath: "/tickets", PermissionCode: "ticket_template:read", SortOrder: 25, Description: "Work order template management"},

		// ===== 子菜单：工单管理 · 配置面（2026-10-25 从系统管理归位）=====
		{Name: "Ticket Categories", Path: "/admin/ticket-categories", Icon: "Tag", ParentPath: "/tickets", PermissionCode: "ticket_category:update", SortOrder: 30},
		{Name: "Work order allocation rules", Path: "/admin/tickets/assignment-rules", Icon: "GitBranch", ParentPath: "/tickets", PermissionCode: "assignment_rule:write", SortOrder: 35},
		{Name: "Work order automation rules", Path: "/admin/tickets/automation-rules", Icon: "Zap", ParentPath: "/tickets", PermissionCode: "automation_rule:write", SortOrder: 40},

		// ===== 子菜单：事件管理 =====
		{Name: "New Incident", Path: "/incidents/create", Icon: "Plus", ParentPath: "/incidents", PermissionCode: "incident:write", SortOrder: 32},
		// 值班排班从「客户与合同」归位：排班对象是事件响应的支持组，属事件域运营视图
		// （数据源 GET /on-call/schedules 要求 on_call:read）。
		{Name: "Duty Scheduling", Path: "/on-call", Icon: "Clock", ParentPath: "/incidents", PermissionCode: "on_call:read", SortOrder: 33, Description: "Support group duty schedule"},

		// ===== 子菜单：问题管理 =====
		// 页面 /problems/new 早已存在；其余六域（事件/变更/发布/资产/CI/知识）都有
		// 「新建X」快捷入口，问题域缺失会让从菜单建问题这条路径不可达。
		{Name: "New Problem", Path: "/problems/new", Icon: "Plus", ParentPath: "/problems", PermissionCode: "problem:write", SortOrder: 41},
		{Name: "Known Error", Path: "/problems/known-errors", Icon: "AlertCircle", ParentPath: "/problems", PermissionCode: "problem:read", SortOrder: 42},
		{Name: "Problem trend", Path: "/problems/trends", Icon: "TrendingUp", ParentPath: "/problems", PermissionCode: "problem:read", SortOrder: 43, Description: "Problem trend analysis"},

		// ===== 子菜单：变更管理 =====
		{Name: "New Change", Path: "/changes/new", Icon: "Plus", ParentPath: "/changes", PermissionCode: "change:write", SortOrder: 52},
		{Name: "Post-implementation review", Path: "/changes/pirs", Icon: "ClipboardCheck", ParentPath: "/changes", PermissionCode: "change:read", SortOrder: 53, Description: "Post-implementation review (PIR) of changes"},
		// ===== 子菜单：变更管理 · 配置面（2026-10-25 从系统管理归位）=====
		{Name: "Review team management", Path: "/admin/change-review", Icon: "Users", ParentPath: "/changes", PermissionCode: "change:read", SortOrder: 60},

		// ===== 子菜单：知识库 =====
		{Name: "New Article", Path: "/knowledge/articles/new", Icon: "Plus", ParentPath: "/knowledge", PermissionCode: "knowledge:write", SortOrder: 63},
		{Name: "Knowledge review", Path: "/knowledge/reviews", Icon: "CheckSquare", ParentPath: "/knowledge", PermissionCode: "knowledge:write", SortOrder: 64, Description: "Knowledge base article review"},

		// ===== 子菜单：客户与合同 =====
		// 值班排班已移出：它回答「谁在值班」，是事件响应排班，不是客户主数据
		// （审计 3.3-⑨ 的父子语义问题）。
		// URL 用业务名（/customers、/support-contracts、/source-organizations），
		// 不再挂在邮件报障技术路径下；旧路径由 next.config 重定向 + seeder 路径迁移兜底（D34-2）。
		{Name: "Customer information", Path: "/customers", Icon: "Users", ParentPath: "/email-intake", PermissionCode: "customer_master:read", SortOrder: 652},
		{Name: "Support Contract", Path: "/support-contracts", Icon: "FileText", ParentPath: "/email-intake", PermissionCode: "support_contract:read", SortOrder: 653},
		{Name: "Source organization", Path: "/source-organizations", Icon: "Globe", ParentPath: "/email-intake", PermissionCode: "customer_master:read", SortOrder: 654},

		// ===== 子菜单：服务目录 =====
		// 「-目录」补丁式后缀不专业（审计报告 3.3-②）；父菜单即服务目录，子项名「目录审批」语义自明。
		{Name: "Directory Approval", Path: "/service-catalog/approvals", Icon: "CheckCircle", ParentPath: "/service-catalog", PermissionCode: "service:read", SortOrder: 72},
		// ===== 子菜单：服务目录 · 配置面（2026-10-25 从系统管理归位）=====
		{Name: "Service CatalogsManage", Path: "/admin/service-catalogs", Icon: "Boxes", ParentPath: "/service-catalog", PermissionCode: "service_catalog:read", SortOrder: 80},

		// ===== 子菜单：CMDB =====
		{Name: "Configuration Item List", Path: "/cmdb/cis", Icon: "Server", ParentPath: "/cmdb", PermissionCode: "cmdb:read", SortOrder: 751},
		{Name: "New Configuration Item", Path: "/cmdb/cis/create", Icon: "Plus", ParentPath: "/cmdb", PermissionCode: "cmdb:write", SortOrder: 752},
		{Name: "Relationship Management", Path: "/cmdb/relationships", Icon: "GitBranch", ParentPath: "/cmdb", PermissionCode: "cmdb:read", SortOrder: 753},
		{Name: "Topology map", Path: "/cmdb/topology", Icon: "Share2", ParentPath: "/cmdb", PermissionCode: "cmdb:read", SortOrder: 754},
		{Name: "Cloud account management", Path: "/cmdb/cloud-accounts", Icon: "Cloud", ParentPath: "/cmdb", PermissionCode: "cmdb:write", SortOrder: 755, Description: "Cloud platform account management"},
		{Name: "Cloud resource list", Path: "/cmdb/cloud-resources", Icon: "CloudLightning", ParentPath: "/cmdb", PermissionCode: "cmdb:read", SortOrder: 756, Description: "Cloud resources discovered"},
		{Name: "Cloud Service Directory", Path: "/cmdb/cloud-services", Icon: "CloudDrizzle", ParentPath: "/cmdb", PermissionCode: "cmdb:read", SortOrder: 757, Description: "Cloud service product catalog"},
		{Name: "Service Map", Path: "/cmdb/registry", Icon: "Compass", ParentPath: "/cmdb", PermissionCode: "cmdb:read", SortOrder: 758, Description: "Service Graph Registry"},
		{Name: "Cloud resource check", Path: "/cmdb/reconciliation", Icon: "RefreshCw", ParentPath: "/cmdb", PermissionCode: "cmdb:write", SortOrder: 759, Description: "Cloud resource and CI check"},
		// ===== 子菜单：CMDB · 配置面（2026-10-25 从系统管理归位）=====
		{Name: "Configuration item type", Path: "/admin/cmdb-types", Icon: "Database", ParentPath: "/cmdb", PermissionCode: "cmdb:write", SortOrder: 770},

		// ===== 子菜单：资产管理 =====
		{Name: "New asset", Path: "/assets/new", Icon: "Plus", ParentPath: "/assets", PermissionCode: "asset:write", SortOrder: 82},
		{Name: "Software License", Path: "/licenses", Icon: "Key", ParentPath: "/assets", PermissionCode: "license:read", SortOrder: 83},

		// ===== 子菜单：SLA =====
		{Name: "SLA Monitor", Path: "/sla-monitor", Icon: "Activity", ParentPath: "/sla", PermissionCode: "sla:read", SortOrder: 92},
		// ===== 子菜单：SLA · 配置面（2026-10-25 从系统管理归位）=====
		{Name: "EscalationRule", Path: "/admin/escalation-rules", Icon: "AlertTriangle", ParentPath: "/sla", PermissionCode: "sla:write", SortOrder: 100},
		{Name: "Upgrade Matrix", Path: "/admin/escalation-matrices", Icon: "TrendingUp", ParentPath: "/sla", PermissionCode: "sla:read", SortOrder: 105},
		{Name: "SLA Template", Path: "/admin/sla-templates", Icon: "Layers", ParentPath: "/sla", PermissionCode: "sla:write", SortOrder: 110},
		{Name: "SLA Definition", Path: "/admin/sla-definitions", Icon: "Clock", ParentPath: "/sla", PermissionCode: "sla:write", SortOrder: 115},

		// ===== 子菜单：工作流 =====
		{Name: "Process Designer", Path: "/workflow/designer", Icon: "Edit", ParentPath: "/workflow", PermissionCode: "workflow:write", SortOrder: 102},
		{Name: "Process Instances", Path: "/workflow/instances", Icon: "Play", ParentPath: "/workflow", PermissionCode: "workflow:read", SortOrder: 103},
		{Name: "Version Management", Path: "/workflow/versions", Icon: "History", ParentPath: "/workflow", PermissionCode: "workflow:write", SortOrder: 104},
		{Name: "Monitoring Dashboard", Path: "/workflow/dashboard", Icon: "Activity", ParentPath: "/workflow", PermissionCode: "workflow:read", SortOrder: 105},
		// 图标用 Gauge（瓶颈仪表语义），消除与工单统计的 BarChart3 撞车（审计报告 3.3-⑧）。
		{Name: "Node bottleneck analysis", Path: "/workflow/bottlenecks", Icon: "Gauge", ParentPath: "/workflow", PermissionCode: "workflow:read", SortOrder: 106},
		// 与 /admin/tickets/automation-rules 是同一个配置面的两个入口，按管理面取
		// automation_rule:write：只读角色拿到菜单后每个写按钮都会 403。
		{Name: "Process trigger", Path: "/workflow/automation", Icon: "Zap", ParentPath: "/workflow", PermissionCode: "automation_rule:write", SortOrder: 107},
		// 该页读的是 BPMN 实例 SLA 指标与违规（/bpmn/sla/*，bpmn:read），
		// 不是 SLA 配置；挂在 SLA 菜单下曾让「SLA 配置」指向监控页（D34-3）。
		{Name: "Process SLA Monitoring", Path: "/workflow/sla", Icon: "Activity", ParentPath: "/workflow", PermissionCode: "bpmn:read", SortOrder: 108, Description: "BPMN process SLA indicators and violation alarms"},
		// 审批中心菜单已删除：与顶级「我的待办」同一页面（/approvals），一个目的地两个菜单
		// 会让用户在两处看到同一份待办（审计 3.3-③ 四入口问题的去重动作）。
		{Name: "Operation log", Path: "/workflow/audit", Icon: "ClipboardList", ParentPath: "/workflow", PermissionCode: "audit:read", SortOrder: 109},
		// ===== 子菜单：工作流 · 配置面（2026-10-25 从系统管理归位）=====
		{Name: "Workflow Configuration", Path: "/admin/workflows", Icon: "GitBranch", ParentPath: "/workflow", PermissionCode: "workflow:write", SortOrder: 120},
		// 「审批链」易与顶级「我的待办」混淆；此页是审批链定义/节点的配置面，
		// 与「工作流配置」同轴，改名点明配置语义（审计 3.3-③）。
		{Name: "Approval Chain Configuration", Path: "/admin/approval-chains", Icon: "Link", ParentPath: "/workflow", PermissionCode: "approval:write", SortOrder: 125},

		// ===== 子菜单：AI 助手 =====
		{Name: "AI creates work order", Path: "/tickets/ai-create", Icon: "Sparkles", ParentPath: "/ai/chat", PermissionCode: "ai:read", SortOrder: 112},
		{Name: "AI Assessment and Audit", Path: "/ai/audit", Icon: "ShieldCheck", ParentPath: "/ai/chat", PermissionCode: "ai:read", SortOrder: 113},
		{Name: "AI Approval", Path: "/ai/approval", Icon: "ShieldAlert", ParentPath: "/ai/chat", PermissionCode: "ai:read", SortOrder: 114},

		// ===== 子菜单：MSP 客户管理 =====
		// 该页是 MSP 把客户分配给服务人员/团队（客户分配），不是「子页」占位（D34-1）。
		{Name: "Customer Assignment", Path: "/msp/management", Icon: "Settings", ParentPath: "/msp", PermissionCode: "msp:write", SortOrder: 122, Description: "Assignment relationship between MSP customers and service personnel"},

		// ===== 子菜单：发布管理 =====
		{Name: "New publication", Path: "/releases/new", Icon: "Plus", ParentPath: "/releases", PermissionCode: "release:write", SortOrder: 132},

		// ===== 顶级管理菜单 =====
		// 权限码统一到 system_config:*（审计 3.2-⑤ 双轨码）：菜单曾声明 system:write，
		// 而该码只有 sysadmin 持有，其余角色能看见这棵树全靠 menu_service 的
		// system→system_config 别名兜着——声明码与真实可见性来源不一致。
		// system:* 从此只用于平台级运维端点（readiness/operations/timers）。
		{Name: "System Admin", Path: "/admin", Icon: "Settings", ParentPath: "", PermissionCode: "system_config:read", SortOrder: 200, Description: "System Admin"},

		// ===== 子菜单：系统管理（组织/权限/平台配置，2026-10-25 精简）=====
		{Name: "System Overview", Path: "/admin/overview", Icon: "LayoutDashboard", ParentPath: "/admin", PermissionCode: "system_config:read", SortOrder: 201},
		{Name: "User Management", Path: "/admin/users", Icon: "Users", ParentPath: "/admin", PermissionCode: "user:read", SortOrder: 210},
		{Name: "Role Management", Path: "/admin/roles", Icon: "Shield", ParentPath: "/admin", PermissionCode: "role:read", SortOrder: 220},
		{Name: "Group Management", Path: "/admin/groups", Icon: "Users", ParentPath: "/admin", PermissionCode: "group:read", SortOrder: 230},
		{Name: "Tenant Management", Path: "/admin/tenants", Icon: "Building", ParentPath: "/admin", PermissionCode: "tenant:read", SortOrder: 235},
		{Name: "Department Management", Path: "/admin/departments", Icon: "Building", ParentPath: "/admin", PermissionCode: "department:read", SortOrder: 240},
		// 页面主数据源 GET /api/v1/departments/:id/processes（department:read）；
		// 页面还会列 /api/v1/process-bindings（bpmn:read），与租管持码集重合。
		{Name: "DepartmentProcess", Path: "/admin/department-processes", Icon: "Workflow", ParentPath: "/admin", PermissionCode: "department:read", SortOrder: 245, Description: "Department-level process configuration"},
		{Name: "Team Management", Path: "/admin/teams", Icon: "Users", ParentPath: "/admin", PermissionCode: "team:read", SortOrder: 250},
		{Name: "Permission management", Path: "/admin/permissions", Icon: "Lock", ParentPath: "/admin", PermissionCode: "role:write", SortOrder: 280},
		{Name: "Vector storage configuration", Path: "/admin/vector-store", Icon: "Database", ParentPath: "/admin", PermissionCode: "system_config:read", SortOrder: 290},
		{Name: "System Configuration", Path: "/admin/system-config", Icon: "Settings", ParentPath: "/admin", PermissionCode: "system_config:read", SortOrder: 295},
		// 域配置继承视图：数据源 GET /api/v1/domain-configs 与 /effective（均为 bpmn:read）。
		{Name: "Configuration Inheritance", Path: "/admin/config-inheritance", Icon: "ArrowLeftRight", ParentPath: "/admin", PermissionCode: "bpmn:read", SortOrder: 296, Description: "Domain process configuration inheritance relationship"},
		{Name: "Global label", Path: "/admin/tags", Icon: "Tags", ParentPath: "/admin", PermissionCode: "system_config:read", SortOrder: 298, Description: "Global tag management"},
		// 能力矩阵控制台：/api/v1/capabilities 本身是 AUTH_ONLY，但页面是运营/售前
		// 用来核对「哪些能力还不能对外承诺」的管理视图，与系统配置同域（system:read），
		// 不给一线角色挂入口——它回答的是发布决策，不是日常作业信息。
		{Name: "Capability Matrix", Path: "/admin/capabilities", Icon: "Target", ParentPath: "/admin", PermissionCode: "system_config:read", SortOrder: 300, Description: "Capability Maturity, Deployment Readiness and Health Signals"},
		// 菜单管理页的读接口是 system_config:read、增删改是 system_config:write；
		// 菜单码取写码（管理页只读角色拿到入口后每个按钮都会 403）。
		{Name: "Menu Management", Path: "/admin/menus", Icon: "Menu", ParentPath: "/admin", PermissionCode: "system_config:write", SortOrder: 345},
	}
}
