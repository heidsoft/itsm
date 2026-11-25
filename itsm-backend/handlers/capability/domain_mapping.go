package capability

// readmeDomain links product promises to registry keys without copying maturity.
// Subdomains document ownership but do not derive an independent README state.
type readmeDomain struct {
	name             string
	primaryKeys      []string
	subdomainOf      string
	noIndependentKey bool
	reason           string
}

var readmeDomains = []readmeDomain{
	{name: "工单与事件", primaryKeys: []string{"ticket", "incident"}},
	{name: "工单类型与动态表单", subdomainOf: "工单与事件", noIndependentKey: true, reason: "由 ticket 承载工单类型与动态表单，无独立 registry key。"},
	{name: "变更管理", primaryKeys: []string{"change"}},
	{name: "问题与 Known Error", primaryKeys: []string{"problem"}},
	{name: "服务目录与请求", primaryKeys: []string{"serviceRequest"}},
	{name: "CMDB", primaryKeys: []string{"cmdb", "cmdb.configurationItems", "cmdb.ciTypes", "cmdb.relationships", "cmdb.topology"}},
	{name: "CMDB 云发现", primaryKeys: []string{"cmdbDiscovery", "cmdbReconciliation"}, subdomainOf: "CMDB"},
	{name: "BPMN 与审批", primaryKeys: []string{"workflow"}},
	{name: "SLA", primaryKeys: []string{"sla"}},
	{name: "知识与 RAG", primaryKeys: []string{"knowledge"}},
	{name: "AI 辅助", primaryKeys: []string{"ai"}},
	{name: "通知与连接器", primaryKeys: []string{"notification", "connector.feishu", "connector.wecom", "connector.dingtalk"}},
	{name: "应用市场与扩展交付", primaryKeys: []string{"marketplace"}},
	{name: "RBAC/多租户", noIndependentKey: true, reason: "平台级承诺由 handlers/rbac 与路由鉴权、租户隔离共同承载，无独立 registry key。"},
	{name: "报表与运营", primaryKeys: []string{"report"}},
}

var internalSupportKeys = map[string]string{
	"cmdb.certification": "CMDB 数据治理的认证支撑能力，不是 README 独立承诺域。",
	"project":            "项目归属为业务资源提供组织上下文，不是 README 独立承诺域。",
	"application":        "应用归属为配置项与业务资源提供上下文，不是 README 独立承诺域。",
	"tag":                "标签为业务资源提供分类支撑，不是 README 独立承诺域。",
	"template":           "工单模板支撑工单域复用配置，不是 README 独立承诺域。",
	"identity.oidc":      "OIDC 为平台身份认证提供集成支撑，不是 README 独立承诺域。",
}

// domainMaturity fails closed for empty, missing or unrecognized definitions.
func domainMaturity(keys []string, byKey map[string]definition) (Maturity, bool) {
	if len(keys) == 0 {
		return MaturityDisabled, false
	}
	result := MaturityGA
	for _, key := range keys {
		d, ok := byKey[key]
		if !ok {
			return MaturityDisabled, false
		}
		switch d.maturity {
		case MaturityGA:
		case MaturityPilot, MaturityDisabled:
			result = MaturityPilot
		default:
			return MaturityDisabled, false
		}
	}
	return result, true
}

func expectedReadmeState(m Maturity) string {
	if m == MaturityGA {
		return "可用"
	}
	return "预览"
}
