package seeder

// expectedCITypes and expectedStandardChanges are effective-manifest accessors:
// seed and verify must compare against the same list, otherwise an environment
// whose JSON section is empty installs built-in rows while verification
// expects none.
func (s *Seeder) expectedCITypes() []CITypeSeed {
	if len(s.config.CITypes) > 0 {
		return s.config.CITypes
	}
	return ciTypeDefinitions()
}

// expectedMarketplaceItems is the effective-manifest accessor for marketplace seeds.
// 与 CI 类型不同，市场商品表无 tenant_id，但同样存在“JSON 配置为空时回退到
// 内置默认”的契约：seed 与 verify 必须看到同一份列表，否则空 JSON 会
// 安装内置种子而校验器仍认为应为空。
func (s *Seeder) expectedMarketplaceItems() []MarketplaceItemSeed {
	if len(s.config.MarketplaceItems) > 0 {
		return s.config.MarketplaceItems
	}
	return marketplaceItemDefinitions()
}

func (s *Seeder) expectedStandardChanges() []StandardChangeSeed {
	if len(s.config.StandardChanges) > 0 {
		return s.config.StandardChanges
	}
	return standardChangeDefinitions()
}

// ticketTypeSeed and workflowTemplateSeed are the in-code product manifests
// that are not carried by the JSON seed config. They live here so the
// initialization checksum can cover them (see manifestDigestInputs).

type ticketTypeSeed struct {
	Code        string
	Name        string
	Description string
	Icon        string
	Color       string
}

func ticketTypeDefinitions() []ticketTypeSeed {
	return []ticketTypeSeed{
		{"k8s_scale", "Kubernetes Scaling", "Request to scale a Kubernetes cluster up or down", "Container", "#1890ff"},
		{"ddl_execute", "DDL Execution", "Database schema changes, index creation, and other DDL operations", "Database", "#722ed1"},
		{"data_export", "Data Export", "Export data from a database or system", "Download", "#13c2c2"},
		{"vm_apply", "Virtual Machine Request", "Request new virtual machine resources", "Desktop", "#2f54eb"},
		{"account_apply", "Account Request", "Request system, VPN, or bastion host accounts", "User", "#52c41a"},
		{"gitlab_repo_apply", "GitLab Repository Request", "Request a new GitLab repository", "Code", "#fa541c"},
		{"domain_apply", "Domain Request", "Request a domain or DNS change", "Global", "#eb2f96"},
		{"firewall_apply", "Firewall Rule Request", "Request new or modified firewall port rules", "Safety", "#fa8c16"},
		{"app_apply", "Application Request", "Request deployment of an application in Kubernetes", "Appstore", "#1890ff"},
		{"project_apply", "Project Request", "Request a new project or workspace", "Project", "#722ed1"},
		{"db_account_apply", "Database Account Request", "Request database read/write or read-only accounts", "Key", "#faad14"},
		{"general", "General Ticket", "General ticket type for other requests", "FileText", "#8c8c8c"},
	}
}

type workflowTemplateSeed struct {
	key, name, desc, domain, bpmnFile string
}

func workflowTemplateDefinitions() []workflowTemplateSeed {
	return []workflowTemplateSeed{
		{key: "generic_request", name: "General Request Workflow", desc: "General requests for administration, IT, facilities, and other services", domain: "it", bpmnFile: "templates/generic_request.bpmn"},
		{key: "change_request", name: "Change Request Workflow", desc: "ITIL standard change management with risk assessment and review board approval", domain: "change", bpmnFile: "templates/change_request.bpmn"},
		{key: "incident_response", name: "Incident Response Workflow", desc: "ITIL incident management with triage, assignment, resolution, and review", domain: "incident", bpmnFile: "templates/incident_response.bpmn"},
		{key: "service_request", name: "Service Request Workflow", desc: "Standard service request fulfillment with approval and automatic fulfillment", domain: "service_request", bpmnFile: "templates/service_request.bpmn"},
		{key: "leave_request", name: "Leave Approval Workflow", desc: "Employee leave requests with multi-level approval", domain: "hr", bpmnFile: "templates/leave_request.bpmn"},
		{key: "expense_approval", name: "Expense Reimbursement Workflow", desc: "Employee expense claims with finance approval", domain: "expense", bpmnFile: "templates/expense_approval.bpmn"},
	}
}

func standardChangeDefinitions() []StandardChangeSeed {
	return []StandardChangeSeed{
		{
			Title:              "Server Restart",
			Description:        "Standard server restart for routine maintenance",
			ImplementationPlan: "1. Notify affected users\n2. Stop services\n3. Restart the server\n4. Verify service recovery",
			RollbackPlan:       "If the restart fails, restore the previous state immediately",
			Justification:      "Routine maintenance",
			Category:           "服务器",
			RiskLevel:          "low",
			ImpactScope:        "low",
			ExpectedDuration:   30,
			ApprovalRequired:   false,
			AffectedCIs:        []string{"Server"},
			Prerequisites:      []string{"Notify users in advance", "Back up important data"},
			Remarks:            "For non-critical servers only",
		},
		{
			Title:              "SSL Certificate Renewal",
			Description:        "Renew an expiring SSL certificate",
			ImplementationPlan: "1. Obtain a new certificate\n2. Validate in a test environment\n3. Deploy to production\n4. Verify the certificate",
			RollbackPlan:       "Keep the previous certificate for immediate rollback if needed",
			Justification:      "The certificate is expiring and must be renewed",
			Category:           "安全",
			RiskLevel:          "low",
			ImpactScope:        "low",
			ExpectedDuration:   60,
			ApprovalRequired:   false,
			AffectedCIs:        []string{"Load Balancer", "Web Server"},
			Prerequisites:      []string{"New certificate requested", "Obtain the certificate files"},
			Remarks:            "",
		},
		{
			Title:              "Database Backup",
			Description:        "Perform a full database backup",
			ImplementationPlan: "1. Pause database writes\n2. Perform a full backup\n3. Verify backup integrity\n4. Restore database service",
			RollbackPlan:       "Cancel the operation if the backup fails",
			Justification:      "Data protection requirements",
			Category:           "数据库",
			RiskLevel:          "low",
			ImpactScope:        "medium",
			ExpectedDuration:   120,
			ApprovalRequired:   false,
			AffectedCIs:        []string{"Database Server"},
			Prerequisites:      []string{"Confirm sufficient backup storage", "Check backup tool availability"},
			Remarks:            "",
		},
		{
			Title:              "Add Firewall Rule",
			Description:        "Add an allow rule to the firewall",
			ImplementationPlan: "1. Prepare the rule change request\n2. Validate in a test environment\n3. Apply the rule in production\n4. Monitor network traffic",
			RollbackPlan:       "Remove the new rule immediately if issues occur",
			Justification:      "Business requirements need a new port",
			Category:           "网络安全",
			RiskLevel:          "medium",
			ImpactScope:        "medium",
			ExpectedDuration:   45,
			ApprovalRequired:   true,
			AffectedCIs:        []string{"Firewall", "Network Switch"},
			Prerequisites:      []string{"Security assessment completed", "Business department approval obtained"},
			Remarks:            "Security team approval required",
		},
		{
			Title:              "Application Configuration Update",
			Description:        "Update application configuration parameters",
			ImplementationPlan: "1. Back up the configuration\n2. Update parameters\n3. Restart the application\n4. Verify functionality",
			RollbackPlan:       "Restore the backed-up configuration file",
			Justification:      "Optimize system performance",
			Category:           "应用",
			RiskLevel:          "low",
			ImpactScope:        "low",
			ExpectedDuration:   30,
			ApprovalRequired:   false,
			AffectedCIs:        []string{"Application Server"},
			Prerequisites:      []string{"New configuration tested successfully"},
			Remarks:            "",
		},
	}
}

func ciTypeDefinitions() []CITypeSeed {
	return []CITypeSeed{
		{Name: "server", Description: "Server", Icon: "server", Color: "#28a745"},
		{Name: "database", Description: "Database", Icon: "database", Color: "#fd7e14"},
		{Name: "network", Description: "Network Device", Icon: "network", Color: "#17a2b8"},
		{Name: "storage", Description: "Storage Device", Icon: "storage", Color: "#e83e8c"},
		{Name: "application", Description: "Application Service", Icon: "app", Color: "#6610f2"},
		{Name: "middleware", Description: "Middleware", Icon: "middleware", Color: "#e74c3c"},
		{Name: "cloud_vm", Description: "Cloud Virtual Machine", Icon: "cloud", Color: "#6f42c1"},
		{Name: "kubernetes", Description: "Kubernetes Resource", Icon: "kubernetes", Color: "#20c997"},
	}
}

func (s *Seeder) expectedTicketTags() []TicketTagSeed {
	if len(s.config.TicketTags) > 0 {
		return s.config.TicketTags
	}
	return ticketTagDefinitions()
}

func (s *Seeder) expectedIncidentCategories() []TicketCategorySeed {
	if len(s.config.IncidentCategories) > 0 {
		return s.config.IncidentCategories
	}
	return incidentCategoryDefinitions()
}

func ticketTagDefinitions() []TicketTagSeed {
	return []TicketTagSeed{
		{Name: "Urgent", Code: "urgent", Description: "Issues requiring urgent attention", Color: "#ff4d4f"},
		{Name: "Important", Code: "important", Description: "Important but not urgent", Color: "#fa8c16"},
		{Name: "bug", Code: "bug", Description: "Software defect", Color: "#f5222d"},
		{Name: "Feature Request", Code: "feature", Description: "New feature request", Color: "#1890ff"},
		{Name: "Performance Issue", Code: "performance", Description: "System performance issues", Color: "#722ed1"},
		{Name: "Security", Code: "security", Description: "Security Issue", Color: "#eb2f96"},
		{Name: "Network", Code: "network", Description: "Network issues", Color: "#13c2c2"},
		{Name: "Database", Code: "database", Description: "Database issues", Color: "#52c41a"},
		{Name: "Awaiting Feedback", Code: "pending-feedback", Description: "Awaiting user feedback", Color: "#faad14"},
		{Name: "Duplicate", Code: "duplicate", Description: "Duplicate issue", Color: "#8c8c8c"},
		{Name: "Cannot Reproduce", Code: "cannot-reproduce", Description: "Issues that cannot be reproduced", Color: "#d9d9d9"},
		{Name: "Resolved", Code: "resolved", Description: "Resolved issues", Color: "#52c41a"},
		{Name: "Needs Review", Code: "needs-review", Description: "Supervisor review required", Color: "#1677ff"},
		{Name: "High Availability", Code: "high-availability", Description: "High availability issues", Color: "#fa541c"},
		{Name: "Monitoring Alert", Code: "monitoring", Description: "Monitoring and alerting", Color: "#fa8c16"},
	}
}

func incidentCategoryDefinitions() []TicketCategorySeed {
	return []TicketCategorySeed{
		{Name: "Hardware Failure", Code: "hardware", Description: "Server, storage, or network hardware failures"},
		{Name: "Software Failure", Code: "software", Description: "Operating system or application failures"},
		{Name: "Network Failure", Code: "network", Description: "Network connectivity or device issues"},
		{Name: "Database Issue", Code: "database", Description: "Database performance or connection issues"},
		{Name: "Security Issue", Code: "security", Description: "Security incidents and vulnerabilities"},
		{Name: "Performance Issue", Code: "performance", Description: "Slow system response or freezes"},
		{Name: "Configuration Issue", Code: "config", Description: "System configuration errors"},
		{Name: "Other", Code: "other", Description: "Other incident types"},
	}
}

// marketplaceItemDefinitions 是内置商品目录；缺这一项市场页就是空的。
//
// 设计原则：
//   - 全部 status=published（ListItems 唯一可见的状态）。
//   - 全部 is_official=true（首批预置必须由平台背书，避免来源不明）。
//   - 全部 is_free=true（私有部署场景禁止“未配置收费”造成隐性账单）。
//   - name 唯一；global catalog，无 tenant_id；不会与租户安装记录冲突。
//   - 涵盖连接器/技能/插件三类，与 Schema.Type enum 对齐。
//
// 当 JSON 配置中显式提供 marketplace_items 时，回退不生效（见 expectedMarketplaceItems）。
func marketplaceItemDefinitions() []MarketplaceItemSeed {
	return []MarketplaceItemSeed{
		{
			Name:             "feishu-connector",
			Type:             "connector",
			Title:            "Feishu Approval Connector",
			Provider:         "ITSM Foundation",
			Description:      "Integrate ITSM tickets with Feishu approvals, including bidirectional approval synchronization and message card callbacks",
			LongDescription:  "Native Feishu integration for approval creation, status callbacks, message cards, and user identity mapping. Supports change and service request approvals from Feishu.",
			IconURL:          "https://trae-api-cn.mchost.guru/api/ide/v1/text_to_image?prompt=Feishu%20logo%20badge%20blue%20square&image_size=square",
			Tags:             []string{"Feishu", "Approval", "IM", "OA"},
			Category:         "im",
			Capabilities:     []string{"approval.sync", "im.notify", "user.map"},
			RequiredPerms:    []string{"feishu.app.read", "feishu.approval.write"},
			LatestVersion:    "1.2.0",
			MinSystemVersion: "v1.6.0",
			IsOfficial:       true,
			IsFree:           true,
			Price:            0,
			License:          "Apache-2.0",
			Homepage:         "https://open.feishu.cn",
			Repository:       "https://github.com/itsm-marketplace/feishu-connector",
		},
		{
			Name:             "dingtalk-ai-assistant",
			Type:             "skill",
			Title:            "DingTalk AI Ticket Assistant",
			Provider:         "ITSM Foundation",
			Description:      "An AI ticket assistant for DingTalk group chats, supporting ticket creation, status queries, and reminders",
			LongDescription:  "A skill for operations and support chats that identifies ticket intent and uses ITSM APIs to create tickets, check status, and send reminders.",
			IconURL:          "https://trae-api-cn.mchost.guru/api/ide/v1/text_to_image?prompt=DingTalk%20logo%20blue%20square&image_size=square",
			Tags:             []string{"DingTalk", "AI", "skill", "Ticket"},
			Category:         "ai-skill",
			Capabilities:     []string{"ticket.create", "ticket.query", "ticket.escalate"},
			RequiredPerms:    []string{"dingtalk.robot.send", "ticket.write"},
			LatestVersion:    "0.9.1",
			MinSystemVersion: "v1.6.0",
			IsOfficial:       true,
			IsFree:           true,
			Price:            0,
			License:          "Apache-2.0",
			Homepage:         "https://open.dingtalk.com",
			Repository:       "https://github.com/itsm-marketplace/dingtalk-ai-assistant",
			// 交付物定位符必须命中本进程已装配的技能码：这个条目的能力
			// （自然语言创建工单/催办）由内置 ai.create_ticket 提供。
			ManifestPath: "skill://ai.create_ticket",
		},
		{
			Name:             "wecom-connector",
			Type:             "connector",
			Title:            "WeCom Connector",
			Provider:         "ITSM Foundation",
			Description:      "Integrate WeCom contacts, approvals, and application messages for internal approvals and alerts",
			LongDescription:  "Synchronize WeCom contacts, send application messages, and receive approval callbacks while reusing existing identity and approval systems.",
			IconURL:          "https://trae-api-cn.mchost.guru/api/ide/v1/text_to_image?prompt=WeCom%20logo%20green%20square&image_size=square",
			Tags:             []string{"WeCom", "Contacts", "Approval"},
			Category:         "im",
			Capabilities:     []string{"contact.sync", "approval.sync", "im.notify"},
			RequiredPerms:    []string{"wecom.contact.read", "wecom.message.send"},
			LatestVersion:    "1.0.3",
			MinSystemVersion: "v1.6.0",
			IsOfficial:       true,
			IsFree:           true,
			Price:            0,
			License:          "Apache-2.0",
			Homepage:         "https://developer.work.weixin.qq.com",
			Repository:       "https://github.com/itsm-marketplace/wecom-connector",
		},
		{
			Name:             "ai-ticket-classify",
			Type:             "skill",
			Title:            "AI Ticket Classification and Priority Suggestions",
			Provider:         "ITSM Foundation",
			Description:      "An LLM skill for ticket classification, priority recommendations, and tag suggestions",
			LongDescription:  "Runs asynchronously after ticket creation and uses knowledge and ticket history to suggest classifications, priorities, and tags. Recommendations above the configured confidence threshold are accepted; others require human confirmation.",
			IconURL:          "https://trae-api-cn.mchost.guru/api/ide/v1/text_to_image?prompt=AI%20brain%20neural%20network%20square&image_size=square",
			Tags:             []string{"AI", "skill", "Classification", "triage"},
			Category:         "ai-skill",
			Capabilities:     []string{"ticket.classify", "ticket.priority", "ticket.suggest.tags"},
			RequiredPerms:    []string{"llm.invoke", "knowledge.read"},
			LatestVersion:    "1.4.0",
			MinSystemVersion: "v1.6.0",
			IsOfficial:       true,
			IsFree:           true,
			Price:            0,
			License:          "Apache-2.0",
			Homepage:         "https://github.com/itsm-marketplace",
			Repository:       "https://github.com/itsm-marketplace/ai-ticket-classify",
			// 分类/优先级建议由内置 ai.triage 交付；定位符写运行时技能码，
			// 不是文件路径——平台不托管第三方技能代码。
			ManifestPath: "skill://ai.triage",
		},
		{
			Name:             "jira-sync-plugin",
			Type:             "plugin",
			Title:            "Jira Bidirectional Sync Plugin",
			Provider:         "Community",
			Description:      "Synchronize Jira issues and ITSM tickets, including status, comments, and attachments",
			LongDescription:  "Connect Jira projects and issues with configurable field mappings, scheduled or event-driven synchronization, and bidirectional status, comment, and attachment updates.",
			IconURL:          "https://trae-api-cn.mchost.guru/api/ide/v1/text_to_image?prompt=Jira%20logo%20blue%20square&image_size=square",
			Tags:             []string{"Jira", "plugin", "sync"},
			Category:         "integration",
			Capabilities:     []string{"issue.sync", "comment.sync", "attachment.sync"},
			RequiredPerms:    []string{"jira.api.read", "jira.api.write"},
			LatestVersion:    "0.8.0",
			MinSystemVersion: "v1.6.0",
			IsOfficial:       false,
			IsFree:           true,
			Price:            0,
			License:          "MIT",
			Homepage:         "https://www.atlassian.com/software/jira",
			Repository:       "https://github.com/itsm-marketplace/jira-sync",
			// 插件类商品没有宿主运行时可扩展，平台无法交付：这里停用而不是删除，
			// 因为已安装租户的记录与历史审计仍指向这条目录项。
			// 目录写入面（CreateCatalogItem/PublishCatalogItem）也已拒绝 plugin 类型。
			Status: "deprecated",
		},
	}
}

// builtinSeedNames preserves the identity of name-keyed defaults installed before
// the English conversion, including records whose customized metadata a migration
// deliberately leaves untouched. Values are lookup aliases, not UI translations.
var builtinSeedNameAliases = map[string]map[string][]string{
	"sla_definitions": {
		"SLA-P0-Urgent":           {"SLA-P0-紧急"},
		"SLA-P1-High":             {"SLA-P1-高"},
		"SLA-P2-Medium":           {"SLA-P2-中"},
		"SLA-P3-Low":              {"SLA-P3-低"},
		"SLA-Service Request":     {"SLA-服务请求"},
		"SLA-Change":              {"SLA-变更"},
		"Incident-P0-Urgent":      {"Incident-P0-紧急"},
		"Incident-P1-High":        {"Incident-P1-高"},
		"Incident-P2-Medium":      {"Incident-P2-中"},
		"Incident-P3-Low":         {"Incident-P3-低"},
		"ServiceRequest-Standard": {"ServiceRequest-标准"},
		"Change-Normal":           {"Change-普通"},
		"Change-Emergency":        {"Change-紧急"},
	},
	"service_catalogs": {
		"ECS Cloud Server":               {"云服务器 ECS"},
		"RDS Cloud Database":             {"云数据库 RDS"},
		"OSS Object Storage":             {"对象存储 OSS"},
		"CDN Acceleration":               {"CDN 加速"},
		"SLB Load Balancer":              {"负载均衡 SLB"},
		"VPN Gateway":                    {"VPN 网关"},
		"Enterprise Email":               {"企业邮箱"},
		"Enterprise File Storage":        {"企业网盘"},
		"Video Conferencing":             {"视频会议"},
		"Enterprise Messaging":           {"企业IM"},
		"Vulnerability Scanning":         {"漏洞扫描"},
		"Penetration Testing":            {"渗透测试"},
		"Security Compliance Assessment": {"等保合规"},
		"IT Service Desk":                {"IT服务台"},
		"Software Installation":          {"软件安装"},
		"Account Request":                {"账户申请"},
		"Network Access":                 {"网络接入"},
		"Domain Request":                 {"域名申请"},
		"Code Repository":                {"代码仓库"},
		"CI/CD Pipeline":                 {"CI/CD流水线"},
		"Test Environment":               {"测试环境"},
		"API Gateway":                    {"API网关"},
	},
	"service_catalog_items": {
		"ECS Instance Request":           {"ECS 实例申请"},
		"ECS Configuration Change":       {"ECS 配置变更"},
		"ECS Incident Report":            {"ECS 故障报修"},
		"RDS Instance Request":           {"RDS 实例申请"},
		"Database Backup Recovery":       {"数据库备份恢复"},
		"IT Consultation":                {"IT 问题咨询"},
		"Password Reset":                 {"密码重置"},
		"Standard Software Installation": {"标准软件安装"},
		"New Employee Account Setup":     {"新员工账户开通"},
		"Business Access Change":         {"业务权限变更"},
		"Office Network Access":          {"办公网络接入"},
		"Internal domain registration":   {"内部域名注册"},
		"Git Repository Creation":        {"Git 仓库创建"},
		"Test Environment Request":       {"测试环境申请"},
		"Web Vulnerability Scan":         {"Web 漏洞扫描"},
		"Security penetration testing":   {"安全渗透测试"},
	},
	"approval_workflows": {
		"P0/P1 Incident Approval":   {"P0/P1事件审批"},
		"Change Approval":           {"变更审批"},
		"Service Request Approval":  {"服务请求审批"},
		"Normal Change Approval":    {"普通变更审批"},
		"Emergency Change Approval": {"紧急变更审批"},
		"Access Request Approval":   {"权限申请审批"},
	},
	"ticket_views": {
		"My Pending Tickets":    {"我的待办工单"},
		"Tickets I Created":     {"我创建的工单"},
		"Urgent Tickets":        {"紧急工单"},
		"Unassigned Tickets":    {"未分配工单"},
		"Closed Tickets":        {"已关闭工单"},
		"High Priority Tickets": {"高优先级工单"},
	},
	"ticket_tags": {
		"High Priority":     {"高优先级"},
		"Approval Required": {"需审批"},
		"AI Suggestion":     {"AI建议"},
		"Customer Impact":   {"客户影响"},
		"Urgent":            {"紧急"},
		"Important":         {"重要"},
		"Feature Request":   {"功能需求"},
		"Performance Issue": {"性能问题"},
		"Security":          {"安全"},
		"Network":           {"网络"},
		"Database":          {"数据库"},
		"Awaiting Feedback": {"待反馈"},
		"Duplicate":         {"重复"},
		"Cannot Reproduce":  {"无法复现"},
		"Resolved":          {"已解决"},
		"Needs Review":      {"需要审核"},
		"High Availability": {"高可用"},
		"Monitoring Alert":  {"监控告警"},
	},
	"standard_changes": {
		"Standard Software Installation":   {"标准软件安装"},
		"Low Risk Firewall Rule Change":    {"低风险防火墙规则调整"},
		"Database Account Access Change":   {"数据库账号权限调整"},
		"Server Restart":                   {"服务器重启"},
		"SSL Certificate Renewal":          {"SSL证书更新"},
		"Database Backup":                  {"数据库备份"},
		"Add Firewall Rule":                {"防火墙规则添加"},
		"Application Configuration Update": {"应用配置更新"},
	},
	"sla_policies": {
		"P1-Urgent Response":          {"P1-紧急响应"},
		"P2-Standard Response":        {"P2-标准响应"},
		"P3-Normal Response":          {"P3-一般响应"},
		"Default P1 Incident SLA":     {"默认P1事件SLA"},
		"Default P2 Incident SLA":     {"默认P2事件SLA"},
		"Default Service Request SLA": {"默认服务请求SLA"},
	},
}

func builtinSeedNames(kind, name string) []string {
	return append([]string{name}, builtinSeedNameAliases[kind][name]...)
}
