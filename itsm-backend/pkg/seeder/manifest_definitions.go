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
		{"k8s_scale", "K8S扩缩容", "Kubernetes容器集群扩容或缩容请求", "Container", "#1890ff"},
		{"ddl_execute", "DDL执行", "数据库表结构变更、索引创建等DDL操作", "Database", "#722ed1"},
		{"data_export", "数据导出", "从数据库或系统导出数据", "Download", "#13c2c2"},
		{"vm_apply", "虚拟机申请", "申请新的虚拟机资源", "Desktop", "#2f54eb"},
		{"account_apply", "账号申请", "申请系统账号、VPN账号、堡垒机账号等", "User", "#52c41a"},
		{"gitlab_repo_apply", "GitLab代码仓库申请", "申请创建新的GitLab代码仓库", "Code", "#fa541c"},
		{"domain_apply", "域名申请", "申请新的域名或域名解析变更", "Global", "#eb2f96"},
		{"firewall_apply", "防火墙规则申请", "申请开放或变更防火墙端口规则", "Safety", "#fa8c16"},
		{"app_apply", "应用申请", "申请在K8S集群中部署新应用服务", "Appstore", "#1890ff"},
		{"project_apply", "项目申请", "申请创建新项目或项目空间", "Project", "#722ed1"},
		{"db_account_apply", "数据库账号申请", "申请数据库读写账号、只读账号等", "Key", "#faad14"},
		{"general", "其他工单", "通用工单类型，用于不属于以上分类的请求", "FileText", "#8c8c8c"},
	}
}

type workflowTemplateSeed struct {
	key, name, desc, domain, bpmnFile string
}

func workflowTemplateDefinitions() []workflowTemplateSeed {
	return []workflowTemplateSeed{
		{key: "generic_request", name: "通用申请流程", desc: "适用于各类行政、IT、设施等通用申请场景", domain: "it", bpmnFile: "templates/generic_request.bpmn"},
		{key: "change_request", name: "变更申请流程", desc: "ITIL 标准变更管理流程，包含风险评估与评审组审批", domain: "change", bpmnFile: "templates/change_request.bpmn"},
		{key: "incident_response", name: "事件响应流程", desc: "ITIL 事件管理流程，包含分级、分派、解决与回顾", domain: "incident", bpmnFile: "templates/incident_response.bpmn"},
		{key: "service_request", name: "服务请求流程", desc: "标准服务请求履行流程，支持审批与自动履行", domain: "service_request", bpmnFile: "templates/service_request.bpmn"},
		{key: "leave_request", name: "请假审批流程", desc: "员工请假申请与多级审批流程", domain: "hr", bpmnFile: "templates/leave_request.bpmn"},
		{key: "expense_approval", name: "费用报销流程", desc: "员工费用报销申请与财务审批流程", domain: "expense", bpmnFile: "templates/expense_approval.bpmn"},
	}
}

func standardChangeDefinitions() []StandardChangeSeed {
	return []StandardChangeSeed{
		{
			Title:              "服务器重启",
			Description:        "标准服务器重启流程，用于常规维护",
			ImplementationPlan: "1. 通知相关用户\n2. 停止服务\n3. 重启服务器\n4. 验证服务恢复",
			RollbackPlan:       "如果重启失败，立即回滚到重启前状态",
			Justification:      "例行维护",
			Category:           "服务器",
			RiskLevel:          "low",
			ImpactScope:        "low",
			ExpectedDuration:   30,
			ApprovalRequired:   false,
			AffectedCIs:        []string{"服务器"},
			Prerequisites:      []string{"提前通知用户", "备份重要数据"},
			Remarks:            "仅适用于非关键业务服务器",
		},
		{
			Title:              "SSL证书更新",
			Description:        "更新即将过期的SSL证书",
			ImplementationPlan: "1. 申请新证书\n2. 在测试环境验证\n3. 生产环境部署\n4. 验证证书生效",
			RollbackPlan:       "保留旧证书，发现问题可立即回滚",
			Justification:      "证书即将过期，必须更新",
			Category:           "安全",
			RiskLevel:          "low",
			ImpactScope:        "low",
			ExpectedDuration:   60,
			ApprovalRequired:   false,
			AffectedCIs:        []string{"负载均衡器", "Web服务器"},
			Prerequisites:      []string{"新证书已申请", "获取证书文件"},
			Remarks:            "",
		},
		{
			Title:              "数据库备份",
			Description:        "执行数据库全量备份",
			ImplementationPlan: "1. 停止数据库写入\n2. 执行全量备份\n3. 验证备份完整性\n4. 恢复数据库服务",
			RollbackPlan:       "备份失败时取消备份操作",
			Justification:      "数据安全要求",
			Category:           "数据库",
			RiskLevel:          "low",
			ImpactScope:        "medium",
			ExpectedDuration:   120,
			ApprovalRequired:   false,
			AffectedCIs:        []string{"数据库服务器"},
			Prerequisites:      []string{"确认备份存储空间充足", "检查备份工具可用性"},
			Remarks:            "",
		},
		{
			Title:              "防火墙规则添加",
			Description:        "添加新的防火墙放行规则",
			ImplementationPlan: "1. 准备规则变更申请\n2. 在测试环境验证\n3. 生产环境应用新规则\n4. 监控网络流量",
			RollbackPlan:       "发现异常时立即删除新添加的规则",
			Justification:      "业务需要开放新端口",
			Category:           "网络安全",
			RiskLevel:          "medium",
			ImpactScope:        "medium",
			ExpectedDuration:   45,
			ApprovalRequired:   true,
			AffectedCIs:        []string{"防火墙", "网络交换机"},
			Prerequisites:      []string{"已完成安全评估", "相关业务部门确认"},
			Remarks:            "需安全部门审批",
		},
		{
			Title:              "应用配置更新",
			Description:        "更新应用程序配置文件中的参数",
			ImplementationPlan: "1. 备份当前配置\n2. 修改配置参数\n3. 重启应用服务\n4. 验证功能正常",
			RollbackPlan:       "回滚到备份的配置文件",
			Justification:      "优化系统性能",
			Category:           "应用",
			RiskLevel:          "low",
			ImpactScope:        "low",
			ExpectedDuration:   30,
			ApprovalRequired:   false,
			AffectedCIs:        []string{"应用服务器"},
			Prerequisites:      []string{"新配置已测试通过"},
			Remarks:            "",
		},
	}
}

func ciTypeDefinitions() []CITypeSeed {
	return []CITypeSeed{
		{Name: "server", Description: "服务器", Icon: "server", Color: "#28a745"},
		{Name: "database", Description: "数据库", Icon: "database", Color: "#fd7e14"},
		{Name: "network", Description: "网络设备", Icon: "network", Color: "#17a2b8"},
		{Name: "storage", Description: "存储设备", Icon: "storage", Color: "#e83e8c"},
		{Name: "application", Description: "应用服务", Icon: "app", Color: "#6610f2"},
		{Name: "middleware", Description: "中间件", Icon: "middleware", Color: "#e74c3c"},
		{Name: "cloud_vm", Description: "云虚拟机", Icon: "cloud", Color: "#6f42c1"},
		{Name: "kubernetes", Description: "Kubernetes资源", Icon: "kubernetes", Color: "#20c997"},
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
		{Name: "紧急", Code: "urgent", Description: "紧急处理的问题", Color: "#ff4d4f"},
		{Name: "重要", Code: "important", Description: "重要但不紧急", Color: "#fa8c16"},
		{Name: "bug", Code: "bug", Description: "程序缺陷", Color: "#f5222d"},
		{Name: "功能需求", Code: "feature", Description: "新功能请求", Color: "#1890ff"},
		{Name: "性能问题", Code: "performance", Description: "系统性能相关", Color: "#722ed1"},
		{Name: "安全", Code: "security", Description: "安全问题", Color: "#eb2f96"},
		{Name: "网络", Code: "network", Description: "网络相关问题", Color: "#13c2c2"},
		{Name: "数据库", Code: "database", Description: "数据库相关问题", Color: "#52c41a"},
		{Name: "待反馈", Code: "pending-feedback", Description: "等待用户反馈", Color: "#faad14"},
		{Name: "重复", Code: "duplicate", Description: "重复问题", Color: "#8c8c8c"},
		{Name: "无法复现", Code: "cannot-reproduce", Description: "无法复现的问题", Color: "#d9d9d9"},
		{Name: "已解决", Code: "resolved", Description: "已解决的问题", Color: "#52c41a"},
		{Name: "需要审核", Code: "needs-review", Description: "需要上级审核", Color: "#1677ff"},
		{Name: "高可用", Code: "high-availability", Description: "高可用相关", Color: "#fa541c"},
		{Name: "监控告警", Code: "monitoring", Description: "监控和告警相关", Color: "#fa8c16"},
	}
}

func incidentCategoryDefinitions() []TicketCategorySeed {
	return []TicketCategorySeed{
		{Name: "硬件故障", Code: "hardware", Description: "服务器、存储、网络设备等硬件故障"},
		{Name: "软件故障", Code: "software", Description: "操作系统、应用软件故障"},
		{Name: "网络故障", Code: "network", Description: "网络连接、网络设备问题"},
		{Name: "数据库问题", Code: "database", Description: "数据库性能、连接问题"},
		{Name: "安全问题", Code: "security", Description: "安全事件、漏洞"},
		{Name: "性能问题", Code: "performance", Description: "系统响应慢、卡顿"},
		{Name: "配置问题", Code: "config", Description: "系统配置错误"},
		{Name: "其他", Code: "other", Description: "其他类型事件"},
	}
}
