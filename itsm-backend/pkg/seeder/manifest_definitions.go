package seeder

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
