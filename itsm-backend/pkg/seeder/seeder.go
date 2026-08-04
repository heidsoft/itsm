package seeder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/approvalworkflow"
	"itsm-backend/ent/assetlicense"
	"itsm-backend/ent/change"
	"itsm-backend/ent/department"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/knownerror"
	"itsm-backend/ent/menu"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/processbinding"
	"itsm-backend/ent/release"
	"itsm-backend/ent/role"
	"itsm-backend/ent/rolepermission"
	"itsm-backend/ent/servicecatalog"
	"itsm-backend/ent/slaalertrule"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/slapolicy"
	"itsm-backend/ent/standardchange"
	"itsm-backend/ent/tag"
	"itsm-backend/ent/team"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/ticketcategory"
	"itsm-backend/ent/ticketview"
	"itsm-backend/ent/user"
	"itsm-backend/service"

	"itsm-backend/config"
	"itsm-backend/database"
	"itsm-backend/pkg/tenantmode"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// Force import usage for ent packages (use predicate functions)
var (
	_ = incident.TitleEQ         // Used to ensure incident package is imported
	_ = problem.TitleEQ          // Used to ensure problem package is imported
	_ = change.TitleEQ           // Used to ensure change package is imported
	_ = knowledgearticle.TitleEQ // Used to ensure knowledgearticle package is imported
	_ = ticketcategory.NameEQ    // Used to ensure ticketcategory package is imported
	_ = knownerror.TitleEQ       // Used to ensure knownerror package is imported
	_ = standardchange.TitleEQ   // Used to ensure standardchange package is imported
	_ = tag.NameEQ               // Used to ensure tag package is imported
	_ = assetlicense.NameEQ      // Used to ensure assetlicense package is imported
	_ = release.TitleEQ          // Used to ensure release package is imported
	_ = slapolicy.NameEQ         // Used to ensure slapolicy package is imported
)

// SeedConfig 种子数据配置结构
type SeedConfig struct {
	Departments       []DepartmentSeed       `json:"departments"`
	Teams             []TeamSeed             `json:"teams"`
	Roles             []RoleSeed             `json:"roles"`
	SLADefinitions    []SLADefinitionSeed    `json:"sla_definitions"`
	SLAPolicies       []SLAPolicySeed        `json:"sla_policies"`
	ServiceCatalog    []ServiceCatalogSeed   `json:"service_catalog"`
	ApprovalWorkflows []ApprovalWorkflowSeed `json:"approval_workflows"`
	ProcessBindings   []ProcessBindingSeed   `json:"process_bindings"`
	TicketViews       []TicketViewSeed       `json:"ticket_views"`
	CITypes           []CITypeSeed           `json:"ci_types"`
	// 新增：可配置的种子数据
	Incidents          []IncidentSeed         `json:"incidents"`
	Problems           []ProblemSeed          `json:"problems"`
	Changes            []ChangeSeed           `json:"changes"`
	KnowledgeArticles  []KnowledgeArticleSeed `json:"knowledge_articles"`
	IncidentCategories []TicketCategorySeed   `json:"incident_categories"`
	// 新增：标准变更模板、已知错误、标签种子数据
	StandardChanges []StandardChangeSeed `json:"standard_changes"`
	KnownErrors     []KnownErrorSeed     `json:"known_errors"`
	TicketTags      []TicketTagSeed      `json:"ticket_tags"`
	// 工作流种子配置
	SeedWorkflows bool `json:"seed_workflows"`
}

type DepartmentSeed struct {
	Name       string `json:"name"`
	Code       string `json:"code"`
	Desc       string `json:"description"`
	ParentCode string `json:"parent_code"`
}

type TeamSeed struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RoleSeed struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

type SLADefinitionSeed struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	ServiceType    string `json:"service_type"`
	Priority       string `json:"priority"`
	ResponseTime   int    `json:"response_time"`
	ResolutionTime int    `json:"resolution_time"`
}

type ServiceCatalogSeed struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	Category         string `json:"category"`
	ServiceType      string `json:"service_type"`
	RequiresApproval bool   `json:"requires_approval"`
	DeliveryTime     int    `json:"delivery_time"`
}

type ApprovalWorkflowSeed struct {
	Name       string                   `json:"name"`
	Desc       string                   `json:"description"`
	TicketType string                   `json:"ticket_type"`
	Priority   string                   `json:"priority"`
	Nodes      []map[string]interface{} `json:"nodes"`
}

type ProcessBindingSeed struct {
	BusinessType         string `json:"business_type"`
	BusinessSubType      string `json:"business_sub_type"`
	ProcessDefinitionKey string `json:"process_definition_key"`
	IsDefault            bool   `json:"is_default"`
}

type TicketViewSeed struct {
	Name     string   `json:"name"`
	Desc     string   `json:"description"`
	IsShared bool     `json:"is_shared"`
	Columns  []string `json:"columns"`
}

type CITypeSeed struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`
	// IsActive 缺省为启用：预置 CI 类型应开箱可用，配置省略时不得隐式禁用
	IsActive *bool `json:"is_active"`
}

// IncidentSeed 事件种子数据结构
type IncidentSeed struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	Priority       string `json:"priority"`
	Severity       string `json:"severity"`
	IncidentNumber string `json:"incident_number"`
	Category       string `json:"category"`
}

// ProblemSeed 问题种子数据结构
type ProblemSeed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	Category    string `json:"category"`
	RootCause   string `json:"root_cause"`
	Impact      string `json:"impact"`
}

// ChangeSeed 变更种子数据结构
type ChangeSeed struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	Priority      string `json:"priority"`
	ImpactScope   string `json:"impact_scope"`
	RiskLevel     string `json:"risk_level"`
	Justification string `json:"justification"`
}

// KnowledgeArticleSeed 知识库文章种子数据结构
type KnowledgeArticleSeed struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Category    string `json:"category"`
	IsPublished bool   `json:"is_published"`
	ViewCount   int    `json:"view_count"`
}

// TicketCategorySeed 工单分类种子数据结构（用于事件分类）
type TicketCategorySeed struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

// StandardChangeSeed 标准变更模板种子数据结构
type StandardChangeSeed struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	ImplementationPlan string   `json:"implementation_plan"`
	RollbackPlan       string   `json:"rollback_plan"`
	Justification      string   `json:"justification"`
	Category           string   `json:"category"`
	RiskLevel          string   `json:"risk_level"`
	ImpactScope        string   `json:"impact_scope"`
	ExpectedDuration   int      `json:"expected_duration"`
	ApprovalRequired   bool     `json:"approval_required"`
	AffectedCIs        []string `json:"affected_cis"`
	Prerequisites      []string `json:"prerequisites"`
	Remarks            string   `json:"remarks"`
}

// KnownErrorSeed 已知错误种子数据结构
type KnownErrorSeed struct {
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Symptoms         string   `json:"symptoms"`
	RootCause        string   `json:"root_cause"`
	Workaround       string   `json:"workaround"`
	Resolution       string   `json:"resolution"`
	Status           string   `json:"status"`
	Category         string   `json:"category"`
	Severity         string   `json:"severity"`
	AffectedProducts []string `json:"affected_products"`
	AffectedCIs      []string `json:"affected_cis"`
	Keywords         []string `json:"keywords"`
}

// TicketTagSeed 标签种子数据结构
type TicketTagSeed struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

// SLAPolicySeed SLA策略种子数据结构
type SLAPolicySeed struct {
	Name                  string `json:"name"`
	Description           string `json:"description"`
	Priority              string `json:"priority"`
	ResponseTimeMinutes   int    `json:"response_time_minutes"`
	ResolutionTimeMinutes int    `json:"resolution_time_minutes"`
	ExcludeWeekends       bool   `json:"exclude_weekends"`
	ExcludeHolidays       bool   `json:"exclude_holidays"`
	IsActive              bool   `json:"is_active"`
	PriorityScore         int    `json:"priority_score"`
}

// Seeder manages database seeding operations
type Seeder struct {
	client                  *ent.Client
	sugar                   *zap.SugaredLogger
	config                  *SeedConfig
	appConfig               *config.Config
	bpmnTemplateService     *service.BPMNTemplateService
	expectedPermissions     []string
	expectedMenus           []string
	expectedRolePermissions map[string][]string
}

// NewSeeder creates a new Seeder instance
func NewSeeder(client *ent.Client, sugar *zap.SugaredLogger, appConfig *config.Config) *Seeder {
	return &Seeder{
		client:              client,
		sugar:               sugar,
		config:              loadSeedConfig(sugar),
		appConfig:           appConfig,
		bpmnTemplateService: service.NewBPMNTemplateService(client),
	}
}

// loadSeedConfig 从 JSON 文件加载种子配置
func loadSeedConfig(sugar *zap.SugaredLogger) *SeedConfig {
	// 配置加载优先级（简化版）：
	// 1. 环境变量 ITSM_SEED_CONFIG 指定文件
	// 2. ./config/seed/default.json
	// 3. 内置默认

	// 1. 环境变量
	if configPath := os.Getenv("ITSM_SEED_CONFIG"); configPath != "" {
		if data, err := os.ReadFile(configPath); err == nil {
			var config SeedConfig
			if err := json.Unmarshal(data, &config); err == nil {
				sugar.Infow("loaded seed config from env", "path", configPath)
				return mergeSeedConfig(getProductDefaultConfig(), &config)
			}
		}
	}

	// 2. 项目配置文件
	paths := []string{
		"config/seed/default.json",
		"../config/seed/default.json",
	}
	for _, path := range paths {
		if data, err := os.ReadFile(path); err == nil {
			var config SeedConfig
			if err := json.Unmarshal(data, &config); err == nil {
				sugar.Infow("loaded seed config from file", "path", path)
				return mergeSeedConfig(getProductDefaultConfig(), &config)
			}
		}
	}

	// 3. 内置默认
	sugar.Infow("using embedded default seed config")
	return getProductDefaultConfig()
}

func getProductDefaultConfig() *SeedConfig {
	cfg := getEmbeddedConfig()
	cfg.Incidents = []IncidentSeed{}
	cfg.Problems = []ProblemSeed{}
	cfg.Changes = []ChangeSeed{}
	cfg.KnowledgeArticles = []KnowledgeArticleSeed{}
	return cfg
}

func mergeSeedConfig(base *SeedConfig, override *SeedConfig) *SeedConfig {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}
	if override.Departments != nil {
		base.Departments = override.Departments
	}
	if override.Teams != nil {
		base.Teams = override.Teams
	}
	if override.Roles != nil {
		base.Roles = override.Roles
	}
	if override.SLADefinitions != nil {
		base.SLADefinitions = override.SLADefinitions
	}
	if override.SLAPolicies != nil {
		base.SLAPolicies = override.SLAPolicies
	}
	if override.ServiceCatalog != nil {
		base.ServiceCatalog = override.ServiceCatalog
	}
	if override.ApprovalWorkflows != nil {
		base.ApprovalWorkflows = override.ApprovalWorkflows
	}
	if override.ProcessBindings != nil {
		base.ProcessBindings = override.ProcessBindings
	}
	if override.TicketViews != nil {
		base.TicketViews = override.TicketViews
	}
	if override.CITypes != nil {
		base.CITypes = override.CITypes
	}
	if override.Incidents != nil {
		base.Incidents = override.Incidents
	}
	if override.Problems != nil {
		base.Problems = override.Problems
	}
	if override.Changes != nil {
		base.Changes = override.Changes
	}
	if override.KnowledgeArticles != nil {
		base.KnowledgeArticles = override.KnowledgeArticles
	}
	if override.IncidentCategories != nil {
		base.IncidentCategories = override.IncidentCategories
	}
	if override.StandardChanges != nil {
		base.StandardChanges = override.StandardChanges
	}
	if override.KnownErrors != nil {
		base.KnownErrors = override.KnownErrors
	}
	if override.TicketTags != nil {
		base.TicketTags = override.TicketTags
	}
	if override.SeedWorkflows {
		base.SeedWorkflows = true
	}
	return base
}

// getEmbeddedConfig 返回内置的默认配置
func getEmbeddedConfig() *SeedConfig {
	return &SeedConfig{
		SeedWorkflows: true,
		Departments: []DepartmentSeed{
			{Name: "信息技术部", Code: "IT", Desc: "IT整体管理"},
			{Name: "IT基础架构", Code: "IT-INFRA", Desc: "基础设施运维", ParentCode: "IT"},
			{Name: "IT应用服务", Code: "IT-APP", Desc: "应用系统运维", ParentCode: "IT"},
			{Name: "IT安全", Code: "IT-SEC", Desc: "信息安全管理", ParentCode: "IT"},
			{Name: "IT项目管理", Code: "IT-PMO", Desc: "IT项目管理", ParentCode: "IT"},
			{Name: "运营管理部", Code: "OPS", Desc: "IT运营管理"},
			{Name: "服务台", Code: "OPS-SD", Desc: "一线服务支持", ParentCode: "OPS"},
			{Name: "运维中心", Code: "OPS-NOC", Desc: "7x24运维监控", ParentCode: "OPS"},
			{Name: "客户服务", Code: "OPS-CS", Desc: "客户服务体验", ParentCode: "OPS"},
			{Name: "研发部", Code: "RD", Desc: "产品研发"},
			{Name: "测试部", Code: "QA", Desc: "质量保证"},
			{Name: "人力资源部", Code: "HR", Desc: "人力资源管理"},
			{Name: "财务部", Code: "FIN", Desc: "财务管理"},
			{Name: "行政部", Code: "ADMIN", Desc: "行政管理"},
		},
		Teams: []TeamSeed{
			{Name: "服务台-L1", Description: "一线服务支持"},
			{Name: "服务台-L2", Description: "二线技术支持"},
			{Name: "服务台-L3", Description: "三线技术专家"},
			{Name: "服务器运维", Description: "服务器运维管理"},
			{Name: "网络运维", Description: "网络设备运维"},
			{Name: "数据库运维", Description: "数据库运维管理"},
			{Name: "云平台运维", Description: "云计算平台运维"},
			{Name: "ERP支持", Description: "ERP系统支持"},
			{Name: "CRM支持", Description: "CRM系统支持"},
			{Name: "OA支持", Description: "OA办公系统支持"},
			{Name: "安全运营", Description: "安全监控与响应"},
			{Name: "安全合规", Description: "安全合规管理"},
			{Name: "后端开发", Description: "后端开发团队"},
			{Name: "前端开发", Description: "前端开发团队"},
			{Name: "移动开发", Description: "移动端开发团队"},
			{Name: "测试团队", Description: "测试与质量保证"},
			{Name: "客户成功", Description: "客户成功管理"},
			{Name: "技术支持", Description: "客户服务技术支持"},
		},
		Roles: []RoleSeed{
			{Name: "IT总监", Code: "it_director", Description: "IT部门总监"},
			{Name: "运维总监", Code: "ops_director", Description: "运维部门总监"},
			{Name: "系统管理员", Code: "sysadmin", Description: "系统管理员"},
			{Name: "安全管理员", Code: "security_admin", Description: "安全管理角色"},
			{Name: "审计管理员", Code: "audit_admin", Description: "审计管理角色"},
			{Name: "运维经理", Code: "ops_manager", Description: "运维团队经理"},
			{Name: "运维工程师", Code: "ops_engineer", Description: "运维工程师"},
			{Name: "DBA工程师", Code: "dba", Description: "数据库管理员"},
			{Name: "网络安全工程师", Code: "network_eng", Description: "网络工程师"},
			{Name: "服务台主管", Code: "sd_manager", Description: "服务台主管"},
			{Name: "一线工程师", Code: "l1_support", Description: "一线支持工程师"},
			{Name: "二线工程师", Code: "l2_support", Description: "二线支持工程师"},
			{Name: "三线专家", Code: "l3_expert", Description: "三线技术专家"},
			{Name: "研发经理", Code: "rd_manager", Description: "研发团队经理"},
			{Name: "开发工程师", Code: "developer", Description: "开发工程师"},
			{Name: "测试工程师", Code: "qa_engineer", Description: "测试工程师"},
			{Name: "部门经理", Code: "dept_manager", Description: "部门经理"},
			{Name: "团队主管", Code: "team_lead", Description: "团队主管"},
			{Name: "普通用户", Code: "end_user", Description: "普通终端用户"},
			{Name: "访客", Code: "guest", Description: "访客用户"},
		},
		SLADefinitions: []SLADefinitionSeed{
			{Name: "SLA-P0-紧急", Description: "P0紧急级别SLA", ServiceType: "incident", Priority: "urgent", ResponseTime: 15, ResolutionTime: 120},
			{Name: "SLA-P1-高", Description: "P1高级别SLA", ServiceType: "incident", Priority: "high", ResponseTime: 30, ResolutionTime: 240},
			{Name: "SLA-P2-中", Description: "P2中级别SLA", ServiceType: "incident", Priority: "medium", ResponseTime: 120, ResolutionTime: 480},
			{Name: "SLA-P3-低", Description: "P3低级别SLA", ServiceType: "incident", Priority: "low", ResponseTime: 240, ResolutionTime: 1440},
			{Name: "SLA-服务请求", Description: "服务请求标准SLA", ServiceType: "service_request", Priority: "medium", ResponseTime: 480, ResolutionTime: 4320},
			{Name: "SLA-变更", Description: "变更请求SLA", ServiceType: "change", Priority: "high", ResponseTime: 60, ResolutionTime: 1440},
		},
		ServiceCatalog: []ServiceCatalogSeed{
			{Name: "云服务器 ECS", Description: "弹性云服务器", Category: "云计算", ServiceType: "vm", RequiresApproval: true, DeliveryTime: 1},
			{Name: "云数据库 RDS", Description: "MySQL/PostgreSQL数据库", Category: "数据库", ServiceType: "rds", RequiresApproval: true, DeliveryTime: 1},
			{Name: "对象存储 OSS", Description: "海量云存储", Category: "存储", ServiceType: "oss", RequiresApproval: false, DeliveryTime: 0},
			{Name: "CDN 加速", Description: "内容分发加速", Category: "网络", ServiceType: "network", RequiresApproval: false, DeliveryTime: 0},
			{Name: "负载均衡 SLB", Description: "流量分发服务", Category: "网络", ServiceType: "network", RequiresApproval: true, DeliveryTime: 1},
			{Name: "VPN 网关", Description: "VPN加密通道", Category: "安全", ServiceType: "security", RequiresApproval: true, DeliveryTime: 2},
			{Name: "企业邮箱", Description: "企业域名邮箱", Category: "通讯", ServiceType: "custom", RequiresApproval: false, DeliveryTime: 1},
			{Name: "企业网盘", Description: "文件存储共享", Category: "协作", ServiceType: "custom", RequiresApproval: false, DeliveryTime: 0},
			{Name: "视频会议", Description: "高清视频会议", Category: "通讯", ServiceType: "custom", RequiresApproval: false, DeliveryTime: 0},
			{Name: "企业IM", Description: "即时通讯工具", Category: "通讯", ServiceType: "custom", RequiresApproval: false, DeliveryTime: 0},
			{Name: "漏洞扫描", Description: "Web漏洞扫描", Category: "安全", ServiceType: "security", RequiresApproval: true, DeliveryTime: 1},
			{Name: "渗透测试", Description: "安全渗透测试", Category: "安全", ServiceType: "security", RequiresApproval: true, DeliveryTime: 5},
			{Name: "等保合规", Description: "等级保护咨询", Category: "安全", ServiceType: "security", RequiresApproval: true, DeliveryTime: 30},
			{Name: "IT服务台", Description: "IT问题咨询支持", Category: "支持", ServiceType: "custom", RequiresApproval: false, DeliveryTime: 0},
			{Name: "软件安装", Description: "标准软件安装", Category: "支持", ServiceType: "custom", RequiresApproval: false, DeliveryTime: 1},
			{Name: "账户申请", Description: "新员工账户开通", Category: "支持", ServiceType: "custom", RequiresApproval: true, DeliveryTime: 1},
			{Name: "网络接入", Description: "网络接入申请", Category: "支持", ServiceType: "custom", RequiresApproval: true, DeliveryTime: 2},
			{Name: "域名申请", Description: "内部域名注册", Category: "支持", ServiceType: "custom", RequiresApproval: true, DeliveryTime: 3},
			{Name: "代码仓库", Description: "Git代码仓库", Category: "开发", ServiceType: "custom", RequiresApproval: false, DeliveryTime: 0},
			{Name: "CI/CD流水线", Description: "自动化部署", Category: "开发", ServiceType: "custom", RequiresApproval: false, DeliveryTime: 0},
			{Name: "测试环境", Description: "预发布测试环境", Category: "开发", ServiceType: "custom", RequiresApproval: true, DeliveryTime: 2},
			{Name: "API网关", Description: "API接口管理", Category: "开发", ServiceType: "custom", RequiresApproval: true, DeliveryTime: 3},
		},
		ApprovalWorkflows: []ApprovalWorkflowSeed{
			{Name: "P0/P1事件审批", Desc: "紧急和高优先级事件需要主管审批", TicketType: "incident", Priority: "urgent,high", Nodes: []map[string]interface{}{{"type": "approval", "name": "主管审批", "approver_type": "manager", "timeout": 60}}},
			{Name: "变更审批", Desc: "所有变更请求需要多级审批", TicketType: "change", Priority: "", Nodes: []map[string]interface{}{{"type": "approval", "name": "技术审批", "approver_type": "role", "role": "engineer", "timeout": 240}, {"type": "approval", "name": "经理审批", "approver_type": "role", "role": "manager", "timeout": 480}}},
			{Name: "服务请求审批", Desc: "高价值服务请求需要审批", TicketType: "service_request", Priority: "high", Nodes: []map[string]interface{}{{"type": "approval", "name": "服务审批", "approver_type": "manager", "timeout": 120}}},
		},
		ProcessBindings: []ProcessBindingSeed{
			{BusinessType: "ticket", BusinessSubType: "incident", ProcessDefinitionKey: "incident_emergency_flow", IsDefault: true},
			{BusinessType: "ticket", BusinessSubType: "problem", ProcessDefinitionKey: "problem_management_flow", IsDefault: true},
			{BusinessType: "ticket", BusinessSubType: "change", ProcessDefinitionKey: "change_normal_flow", IsDefault: true},
			{BusinessType: "ticket", BusinessSubType: "service_request", ProcessDefinitionKey: "service_request_flow", IsDefault: true},
			{BusinessType: "ticket", BusinessSubType: "improvement", ProcessDefinitionKey: "ticket_general_flow", IsDefault: true},
			{BusinessType: "ticket", ProcessDefinitionKey: "ticket_general_flow", IsDefault: true},
		},
		TicketViews: []TicketViewSeed{
			{Name: "我的待办工单", Desc: "分配给我的未关闭工单", IsShared: false, Columns: []string{"id", "title", "priority", "status", "assignee", "created_at"}},
			{Name: "我创建的工单", Desc: "我提交的工单", IsShared: false, Columns: []string{"id", "title", "priority", "status", "assignee", "created_at"}},
			{Name: "紧急工单", Desc: "紧急和高优先级工单", IsShared: true, Columns: []string{"id", "title", "priority", "status", "assignee", "created_at"}},
			{Name: "未分配工单", Desc: "尚未分配的工单", IsShared: true, Columns: []string{"id", "title", "priority", "status", "creator", "created_at"}},
			{Name: "已关闭工单", Desc: "已完成的工单", IsShared: false, Columns: []string{"id", "title", "priority", "status", "assignee", "closed_at"}},
		},
	}
}

// SeedAll runs all seeding operations
func (s *Seeder) SeedAll(ctx context.Context) {
	// 首先确保 default 租户存在
	s.seedDefaultTenant(ctx)
	s.seedDepartments(ctx)
	s.seedTeams(ctx)
	s.seedRoles(ctx)
	s.seedPermissions(ctx) // 新增：初始化权限
	s.seedMenus(ctx)       // 新增：初始化菜单
	s.seedAdmin(ctx)
	s.seedCloudServiceTemplates(ctx)
	// 使用配置的初始化数据
	s.seedSLADefinitions(ctx)
	s.seedSLAPolicies(ctx)
	s.seedSLAAlertRules(ctx)
	s.seedApprovalWorkflows(ctx)
	s.seedProcessBindings(ctx)
	s.seedBPMNWorkflows(ctx) // 部署BPMN工作流模板
	s.seedTicketViews(ctx)
	s.seedServiceCatalog(ctx)
	s.seedTicketTypes(ctx)            // 新增：初始化工单类型
	s.seedCITypes(ctx)                // 新增：初始化CI类型
	s.seedIncidentCategories(ctx)     // 新增：初始化事件分类
	s.seedStandardChanges(ctx)        // 新增：初始化标准变更模板
	s.seedTicketTags(ctx)             // 新增：初始化标签
	s.seedMenuAndPermissionFixes(ctx) // 修复：更新菜单路径和补充缺失权限
	s.seedRolePermissions(ctx)        // 新增：为角色分配权限
}

// SeedProduction applies product defaults and then verifies the minimum
// production invariants. Individual legacy seed helpers log and continue on
// conflict; this fail-closed verification prevents bootstrap from reporting
// success when a required tenant, identity, RBAC, menu, or product template
// was only partially initialized.
func (s *Seeder) SeedProduction(ctx context.Context) error {
	s.SeedAll(ctx)
	return s.VerifyProduction(ctx)
}

// VerifyProduction checks the complete managed baseline without writing data.
func (s *Seeder) VerifyProduction(ctx context.Context) error {
	rootTenant, err := s.client.Tenant.Query().
		Where(tenant.CodeEQ("default")).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("verify default tenant: %w", err)
	}

	checks := []struct {
		name  string
		exist func() (bool, error)
	}{
		{
			name: "administrator",
			exist: func() (bool, error) {
				return s.client.User.Query().
					Where(user.UsernameEQ("admin"), user.TenantIDEQ(rootTenant.ID)).
					Exist(ctx)
			},
		},
		{
			name: "roles",
			exist: func() (bool, error) {
				for _, expected := range s.config.Roles {
					exists, err := s.client.Role.Query().
						Where(role.CodeEQ(expected.Code), role.TenantIDEQ(rootTenant.ID)).
						Exist(ctx)
					if err != nil || !exists {
						return exists, err
					}
				}
				return len(s.config.Roles) > 0, nil
			},
		},
		{
			name: "permissions",
			exist: func() (bool, error) {
				count, err := s.client.Permission.Query().
					Where(
						permission.TenantIDEQ(rootTenant.ID),
						permission.CodeIn(s.expectedPermissions...),
					).
					Count(ctx)
				return count == len(s.expectedPermissions) && count > 0, err
			},
		},
		{
			name: "role permission bindings",
			exist: func() (bool, error) {
				roleCodes := make([]string, 0, len(s.config.Roles))
				for _, expected := range s.config.Roles {
					roleCodes = append(roleCodes, expected.Code)
				}
				roles, err := s.client.Role.Query().
					Where(role.TenantIDEQ(rootTenant.ID), role.CodeIn(roleCodes...)).
					All(ctx)
				if err != nil {
					return false, err
				}
				for _, seededRole := range roles {
					expectedCodes, managed := s.expectedRolePermissions[seededRole.Code]
					if !managed {
						continue
					}
					bindings, err := s.client.RolePermission.Query().
						Where(
							rolepermission.TenantIDEQ(rootTenant.ID),
							rolepermission.RoleIDEQ(seededRole.ID),
						).
						All(ctx)
					if err != nil {
						return false, err
					}
					actualCodes := make(map[string]struct{}, len(bindings))
					for _, binding := range bindings {
						p, err := s.client.Permission.Get(ctx, binding.PermissionID)
						if err != nil {
							return false, err
						}
						actualCodes[p.Code] = struct{}{}
					}
					if len(actualCodes) < len(expectedCodes) {
						return false, nil
					}
					for _, code := range expectedCodes {
						if _, ok := actualCodes[code]; !ok {
							return false, nil
						}
					}
				}
				return len(roles) > 0, nil
			},
		},
		{
			name: "menus",
			exist: func() (bool, error) {
				count, err := s.client.Menu.Query().
					Where(
						menu.TenantIDEQ(rootTenant.ID),
						menu.PathIn(s.expectedMenus...),
					).
					Count(ctx)
				return count == len(s.expectedMenus) && count > 0, err
			},
		},
		{
			name: "SLA definitions",
			exist: func() (bool, error) {
				return s.client.SLADefinition.Query().
					Where(sladefinition.TenantIDEQ(rootTenant.ID)).
					Exist(ctx)
			},
		},
		{
			name: "service catalog templates",
			exist: func() (bool, error) {
				return s.client.ServiceCatalog.Query().
					Where(servicecatalog.TenantIDEQ(rootTenant.ID)).
					Exist(ctx)
			},
		},
		{
			name: "standard change templates",
			exist: func() (bool, error) {
				return s.client.StandardChange.Query().
					Where(standardchange.TenantIDEQ(rootTenant.ID)).
					Exist(ctx)
			},
		},
	}

	for _, check := range checks {
		exists, err := check.exist()
		if err != nil {
			return fmt.Errorf("verify %s: %w", check.name, err)
		}
		if !exists {
			return fmt.Errorf("verify %s: required production seed is missing", check.name)
		}
	}
	return nil
}

// seedDefaultTenant ensures default tenant exists
func (s *Seeder) seedDefaultTenant(ctx context.Context) *ent.Tenant {
	rootType := tenantmode.TenantTypeInternal
	rootName := "Default Tenant"
	rootDomain := "localhost"

	switch s.deploymentMode() {
	case tenantmode.DeploymentModeSaaS:
		rootName = "SaaS Platform Tenant"
	case tenantmode.DeploymentModeSaaSMSP:
		rootType = tenantmode.TenantTypeMSPProvider
		rootName = "MSP Provider Tenant"
	}

	existing, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err == nil && existing != nil {
		updated, updateErr := existing.Update().
			SetName(rootName).
			SetDomain(rootDomain).
			SetStatus("active").
			SetType(tenant.Type(rootType)).
			SetBillingEnabled(true).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if updateErr == nil {
			existing = updated
		}
		s.sugar.Infow("default tenant already exists", "tenant_id", existing.ID)
		return existing
	}

	defaultTenant, err := s.client.Tenant.Create().
		SetName(rootName).
		SetCode("default").
		SetDomain(rootDomain).
		SetStatus("active").
		SetType(tenant.Type(rootType)).
		SetBillingEnabled(true).
		SetCurrency("CNY").
		SetServiceTier("enterprise").
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.sugar.Warnw("failed to create default tenant", "error", err)
		return nil
	}
	s.sugar.Infow("default tenant created", "tenant_id", defaultTenant.ID)
	return defaultTenant
}

func (s *Seeder) deploymentMode() string {
	if s.appConfig == nil || s.appConfig.Deployment.Mode == "" {
		return tenantmode.DeploymentModePrivate
	}
	return s.appConfig.Deployment.Mode
}

func nilIfEmpty(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func (s *Seeder) seedAdmin(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip admin seed", "error", err)
		return
	}
	existing, err := s.client.User.Query().Where(user.UsernameEQ("admin"), user.TenantIDEQ(t.ID)).First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		s.sugar.Warnw("query admin user failed", "error", err)
		return
	}
	if existing != nil {
		s.sugar.Infow("seed admin already exists; credentials preserved", "username", "admin")
		return
	}

	// Check if bootstrap token mode is enabled.
	bootstrapEnabled := os.Getenv("BOOTSTRAP_TOKEN_ENABLED") == "1"
	if bootstrapEnabled {
		// Generate and output bootstrap token for first-time setup.
		// The token must be consumed via API call, not自动 created here.
		// This is handled by cmd/initialize CLI which prints the token.
		s.sugar.Infow("bootstrap mode enabled; use initialize CLI to generate token and create admin")
		return
	}

	// Fallback: ADMIN_PASSWORD (backward compatible).
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		s.sugar.Warnw("ADMIN_PASSWORD env var not set; skip admin seed")
		return
	}
	passHash, bcryptErr := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if bcryptErr != nil {
		s.sugar.Warnw("generate bcrypt for admin failed", "error", bcryptErr)
		return
	}

	if _, err := s.client.User.Create().
		SetUsername("admin").
		SetRole("super_admin").
		SetPasswordHash(string(passHash)).
		SetEmail("admin@example.com").
		SetName("系统管理员").
		SetDepartment("IT部门").
		SetActive(true).
		SetTenantID(t.ID).
		Save(ctx); err != nil {
		s.sugar.Warnw("seed admin failed", "error", err)
	} else {
		s.sugar.Infow("seed admin created", "username", "admin")
	}
}

func (s *Seeder) seedDepartments(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip departments seed", "error", err)
		return
	}

	existing, err := s.client.Department.Query().Where(department.TenantIDEQ(t.ID), department.DeletedAtIsNil()).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing departments failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("departments already seeded")
		return
	}

	// 使用配置文件中的数据
	for _, d := range s.config.Departments {
		if _, err := s.client.Department.Create().
			SetName(d.Name).
			SetCode(d.Code).
			SetDescription(d.Desc).
			SetTenantID(t.ID).
			Save(ctx); err != nil {
			s.sugar.Warnw("seed department failed", "error", err, "name", d.Name)
		}
	}
	s.sugar.Infow("departments seeded", "count", len(s.config.Departments))
}

func (s *Seeder) seedTeams(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip teams seed", "error", err)
		return
	}

	existing, err := s.client.Team.Query().Where(team.TenantIDEQ(t.ID), team.DeletedAtIsNil()).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing teams failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("teams already seeded")
		return
	}

	for _, tm := range s.config.Teams {
		code := tm.Code
		if code == "" {
			// 从名称生成代码：去除空格，转小写
			code = strings.ToLower(strings.ReplaceAll(tm.Name, " ", "-"))
		}
		if _, err := s.client.Team.Create().
			SetName(tm.Name).
			SetCode(code).
			SetDescription(tm.Description).
			SetStatus("active").
			SetTenantID(t.ID).
			Save(ctx); err != nil {
			s.sugar.Warnw("seed team failed", "error", err, "name", tm.Name)
		}
	}
	s.sugar.Infow("teams seeded", "count", len(s.config.Teams))
}

func (s *Seeder) seedRoles(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip roles seed", "error", err)
		return
	}

	for _, r := range s.config.Roles {
		existing, err := s.client.Role.Query().
			Where(role.CodeEQ(r.Code), role.TenantIDEQ(t.ID)).
			Only(ctx)
		if err == nil {
			if _, err := existing.Update().
				SetName(r.Name).
				SetDescription(r.Description).
				Save(ctx); err != nil {
				s.sugar.Warnw("update role failed", "error", err, "code", r.Code)
			}
			continue
		}
		if !ent.IsNotFound(err) {
			s.sugar.Warnw("query role failed", "error", err, "code", r.Code)
			continue
		}
		if _, err := s.client.Role.Create().
			SetName(r.Name).
			SetCode(r.Code).
			SetDescription(r.Description).
			SetTenantID(t.ID).
			Save(ctx); err != nil {
			s.sugar.Warnw("seed role failed", "error", err, "name", r.Name)
		}
	}
	s.sugar.Infow("roles seeded", "count", len(s.config.Roles))
}

// seedCloudServiceTemplates 保留云服务模板种子入口（历史演示数据 seeder 已移除：
// 默认初始化只创建产品模板/配置，不创建假客户业务数据）

func (s *Seeder) seedCloudServiceTemplates(ctx context.Context) {
	// 保留原有实现...
}

// 以下是使用配置文件的初始化函数

func (s *Seeder) seedSLADefinitions(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip SLA definitions seed", "error", err)
		return
	}

	existing, err := s.client.SLADefinition.Query().Where(sladefinition.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing SLA definitions failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("SLA definitions already seeded")
		return
	}

	slaIDMap := make(map[string]int)
	for _, sla := range s.config.SLADefinitions {
		entity, err := s.client.SLADefinition.Create().
			SetName(sla.Name).
			SetDescription(sla.Description).
			SetServiceType(sla.ServiceType).
			SetPriority(sla.Priority).
			SetResponseTime(sla.ResponseTime).
			SetResolutionTime(sla.ResolutionTime).
			SetIsActive(true).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed SLA definition failed", "error", err, "name", sla.Name)
			continue
		}
		slaIDMap[sla.Name] = entity.ID
	}
	s.sugar.Infow("SLA definitions seeded", "count", len(s.config.SLADefinitions))
	_ = slaIDMap
}

func (s *Seeder) seedSLAPolicies(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip SLA policies seed", "error", err)
		return
	}

	policies := s.config.SLAPolicies
	if len(policies) == 0 {
		policies = defaultSLAPolicySeeds()
		s.sugar.Warnw("SLA policies not found in seed config; using built-in defaults")
	}

	created := 0
	updated := 0
	for _, sla := range policies {
		existing, err := s.client.SLAPolicy.Query().
			Where(slapolicy.TenantIDEQ(t.ID), slapolicy.NameEQ(sla.Name)).
			First(ctx)
		if err == nil {
			_, err = existing.Update().
				SetDescription(sla.Description).
				SetPriority(sla.Priority).
				SetResponseTimeMinutes(sla.ResponseTimeMinutes).
				SetResolutionTimeMinutes(sla.ResolutionTimeMinutes).
				SetExcludeWeekends(sla.ExcludeWeekends).
				SetExcludeHolidays(sla.ExcludeHolidays).
				SetIsActive(sla.IsActive).
				SetPriorityScore(sla.PriorityScore).
				Save(ctx)
			if err != nil {
				s.sugar.Warnw("update SLA policy failed", "error", err, "name", sla.Name)
				continue
			}
			updated++
			continue
		}

		_, err = s.client.SLAPolicy.Create().
			SetName(sla.Name).
			SetDescription(sla.Description).
			SetPriority(sla.Priority).
			SetResponseTimeMinutes(sla.ResponseTimeMinutes).
			SetResolutionTimeMinutes(sla.ResolutionTimeMinutes).
			SetExcludeWeekends(sla.ExcludeWeekends).
			SetExcludeHolidays(sla.ExcludeHolidays).
			SetIsActive(sla.IsActive).
			SetPriorityScore(sla.PriorityScore).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed SLA policy failed", "error", err, "name", sla.Name)
			continue
		}
		created++
	}
	s.sugar.Infow("SLA policies ensured", "total", len(policies), "created", created, "updated", updated)
}

func defaultSLAPolicySeeds() []SLAPolicySeed {
	return []SLAPolicySeed{
		{
			Name:                  "默认P1事件SLA",
			Description:           "高优先级事件响应与解决策略",
			Priority:              "high",
			ResponseTimeMinutes:   30,
			ResolutionTimeMinutes: 240,
			ExcludeWeekends:       false,
			ExcludeHolidays:       false,
			IsActive:              true,
			PriorityScore:         90,
		},
		{
			Name:                  "默认P2事件SLA",
			Description:           "中优先级事件响应与解决策略",
			Priority:              "medium",
			ResponseTimeMinutes:   120,
			ResolutionTimeMinutes: 1440,
			ExcludeWeekends:       true,
			ExcludeHolidays:       true,
			IsActive:              true,
			PriorityScore:         60,
		},
		{
			Name:                  "默认服务请求SLA",
			Description:           "标准服务请求履约策略",
			Priority:              "low",
			ResponseTimeMinutes:   240,
			ResolutionTimeMinutes: 2880,
			ExcludeWeekends:       true,
			ExcludeHolidays:       true,
			IsActive:              true,
			PriorityScore:         30,
		},
	}
}

func (s *Seeder) seedSLAAlertRules(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip SLA alert rules seed", "error", err)
		return
	}

	existing, err := s.client.SLAAlertRule.Query().Where(slaalertrule.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing SLA alert rules failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("SLA alert rules already seeded")
		return
	}

	// 简化版告警规则
	alertRules := []struct {
		Name              string
		SLAKey            string
		AlertLevel        string
		Threshold         int
		NotificationChans []string
	}{
		{"SLA-P0-响应告警", "SLA-P0-紧急", "warning", 50, []string{"email"}},
		{"SLA-P0-解决告警", "SLA-P0-紧急", "critical", 80, []string{"email", "sms"}},
		{"SLA-P1-响应告警", "SLA-P1-高", "warning", 50, []string{"email"}},
		{"SLA-P1-解决告警", "SLA-P1-高", "warning", 80, []string{"email"}},
		{"SLA-P2-响应告警", "SLA-P2-中", "info", 50, []string{"email"}},
		{"SLA-P2-解决告警", "SLA-P2-中", "warning", 80, []string{"email"}},
		{"SLA-服务请求-响应告警", "SLA-服务请求", "info", 50, []string{"email"}},
		{"SLA-变更-响应告警", "SLA-变更", "warning", 50, []string{"email"}},
	}

	// 获取 SLA 定义
	slas, err := s.client.SLADefinition.Query().Where(sladefinition.TenantIDEQ(t.ID)).All(ctx)
	if err != nil || len(slas) == 0 {
		s.sugar.Warnw("no SLA definitions found; skip alert rules seed")
		return
	}

	slaMap := make(map[string]int)
	for _, sla := range slas {
		slaMap[sla.Name] = sla.ID
	}

	for _, rule := range alertRules {
		slaID, ok := slaMap[rule.SLAKey]
		if !ok {
			continue
		}
		_, err := s.client.SLAAlertRule.Create().
			SetName(rule.Name).
			SetSLADefinitionID(slaID).
			SetAlertLevel(rule.AlertLevel).
			SetThresholdPercentage(rule.Threshold).
			SetNotificationChannels(rule.NotificationChans).
			SetEscalationEnabled(true).
			SetIsActive(true).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed SLA alert rule failed", "error", err, "name", rule.Name)
		}
	}
	s.sugar.Infow("SLA alert rules seeded", "count", len(alertRules))
}

func (s *Seeder) seedApprovalWorkflows(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip approval workflows seed", "error", err)
		return
	}

	existing, err := s.client.ApprovalWorkflow.Query().Where(approvalworkflow.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing approval workflows failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("approval workflows already seeded")
		return
	}

	for _, wf := range s.config.ApprovalWorkflows {
		_, err := s.client.ApprovalWorkflow.Create().
			SetName(wf.Name).
			SetDescription(wf.Desc).
			SetTicketType(wf.TicketType).
			SetPriority(wf.Priority).
			SetNodes(wf.Nodes).
			SetIsActive(true).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed approval workflow failed", "error", err, "name", wf.Name)
		}
	}
	s.sugar.Infow("approval workflows seeded", "count", len(s.config.ApprovalWorkflows))
}

func (s *Seeder) seedProcessBindings(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip process bindings seed", "error", err)
		return
	}

	existing, err := s.client.ProcessBinding.Query().Where(processbinding.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing process bindings failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("process bindings already seeded")
		return
	}

	for _, b := range s.config.ProcessBindings {
		_, err := s.client.ProcessBinding.Create().
			SetBusinessType(b.BusinessType).
			SetNillableBusinessSubType(nilIfEmpty(b.BusinessSubType)).
			SetProcessDefinitionKey(b.ProcessDefinitionKey).
			SetIsDefault(b.IsDefault).
			SetIsActive(true).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed process binding failed", "error", err, "business_type", b.BusinessType)
		}
	}
	s.sugar.Infow("process bindings seeded", "count", len(s.config.ProcessBindings))
}

// seedBPMNWorkflows 部署BPMN工作流模板
func (s *Seeder) seedBPMNWorkflows(ctx context.Context) {
	// 检查是否已配置部署工作流
	if s.config == nil || !s.config.SeedWorkflows {
		s.sugar.Infow("workflow seeding is disabled in config")
		return
	}

	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip BPMN workflows seed", "error", err)
		return
	}

	// 使用BPMNTemplateService加载并部署内置模板
	templates, err := s.bpmnTemplateService.LoadAndDeployTemplates(ctx, t.ID)
	if err != nil {
		s.sugar.Warnw("failed to deploy BPMN templates", "error", err)
		return
	}

	s.sugar.Infow("BPMN workflows seeded", "count", len(templates))
}

func (s *Seeder) seedTicketViews(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip ticket views seed", "error", err)
		return
	}

	admin, err := s.client.User.Query().Where(user.UsernameEQ("admin"), user.TenantIDEQ(t.ID)).First(ctx)
	if err != nil {
		s.sugar.Warnw("admin user not found; skip ticket views seed", "error", err)
		return
	}

	existing, err := s.client.TicketView.Query().Where(ticketview.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing ticket views failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("ticket views already seeded")
		return
	}

	for _, v := range s.config.TicketViews {
		filters := map[string]interface{}{}
		if v.Name == "我的待办工单" {
			filters = map[string]interface{}{"assignee_id": admin.ID, "status": []string{"open", "in_progress", "pending"}}
		} else if v.Name == "我创建的工单" {
			filters = map[string]interface{}{"creator_id": admin.ID}
		} else if v.Name == "紧急工单" {
			filters = map[string]interface{}{"priority": []string{"urgent", "high"}}
		} else if v.Name == "未分配工单" {
			filters = map[string]interface{}{"assignee_id": nil, "status": []string{"open"}}
		} else if v.Name == "已关闭工单" {
			filters = map[string]interface{}{"status": []string{"closed", "resolved"}}
		}

		_, err := s.client.TicketView.Create().
			SetName(v.Name).
			SetDescription(v.Desc).
			SetFilters(filters).
			SetColumns(v.Columns).
			SetIsShared(v.IsShared).
			SetCreatedBy(admin.ID).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed ticket view failed", "error", err, "name", v.Name)
		}
	}
	s.sugar.Infow("ticket views seeded", "count", len(s.config.TicketViews))
}

// seedPermissions 初始化系统权限
func (s *Seeder) seedPermissions(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip permissions seed", "error", err)
		return
	}

	// 定义所有权限
	permissions := []struct {
		Code        string
		Name        string
		Resource    string
		Action      string
		Description string
	}{
		// 工单权限
		{"ticket:read", "查看工单", "ticket", "read", "查看工单列表和详情"},
		{"ticket:write", "管理工单", "ticket", "write", "创建、编辑工单"},
		{"ticket:create", "创建工单", "ticket", "create", "创建工单"},
		{"ticket:update", "更新工单", "ticket", "update", "更新工单"},
		{"ticket:assign", "分派工单", "ticket", "assign", "分派工单"},
		{"ticket:escalate", "升级工单", "ticket", "escalate", "升级工单"},
		{"ticket:resolve", "解决工单", "ticket", "resolve", "解决工单"},
		{"ticket:close", "关闭工单", "ticket", "close", "关闭工单"},
		{"ticket:export", "导出工单", "ticket", "export", "导出工单"},
		{"ticket:import", "导入工单", "ticket", "import", "导入工单"},
		{"ticket:admin", "工单管理配置", "ticket", "admin", "管理工单自动化和配置"},
		{"ticket:delete", "删除工单", "ticket", "delete", "删除工单"},
		{"ticket_category:read", "查看工单分类", "ticket_category", "read", "查看工单分类"},
		{"ticket_category:create", "创建工单分类", "ticket_category", "create", "创建工单分类"},
		{"ticket_category:update", "更新工单分类", "ticket_category", "update", "更新工单分类"},
		{"ticket_category:delete", "删除工单分类", "ticket_category", "delete", "删除工单分类"},
		{"ticket_tag:read", "查看工单标签", "ticket_tag", "read", "查看工单标签"},
		{"ticket_tag:create", "创建工单标签", "ticket_tag", "create", "创建工单标签"},
		{"ticket_tag:update", "更新工单标签", "ticket_tag", "update", "更新工单标签"},
		{"ticket_tag:delete", "删除工单标签", "ticket_tag", "delete", "删除工单标签"},
		{"ticket_template:read", "查看工单模板", "ticket_template", "read", "查看工单模板"},
		{"ticket_template:create", "创建工单模板", "ticket_template", "create", "创建工单模板"},
		{"ticket_template:update", "更新工单模板", "ticket_template", "update", "更新工单模板"},
		{"ticket_template:delete", "删除工单模板", "ticket_template", "delete", "删除工单模板"},
		// 事件权限
		{"incident:read", "查看事件", "incident", "read", "查看事件列表和详情"},
		{"incident:write", "管理事件", "incident", "write", "创建、编辑事件"},
		{"incident:delete", "删除事件", "incident", "delete", "删除事件"},
		// 问题权限
		{"problem:read", "查看问题", "problem", "read", "查看问题列表和详情"},
		{"problem:write", "管理问题", "problem", "write", "创建、编辑问题"},
		{"problem:delete", "删除问题", "problem", "delete", "删除问题"},
		// 变更权限
		{"change:read", "查看变更", "change", "read", "查看变更列表和详情"},
		{"change:write", "管理变更", "change", "write", "创建、编辑变更"},
		{"change:delete", "删除变更", "change", "delete", "删除变更"},
		{"change:approve", "审批变更", "change", "approve", "变更CAB审批/驳回/回滚"},
		{"change:rollback", "回滚变更", "change", "rollback", "变更实施后回滚"},
		// 发布权限
		{"release:read", "查看发布", "release", "read", "查看发布列表和详情"},
		{"release:write", "管理发布", "release", "write", "创建、编辑发布"},
		{"release:delete", "删除发布", "release", "delete", "删除发布"},
		{"release:approve", "审批发布", "release", "approve", "发布审批/驳回"},
		{"release:rollback", "回滚发布", "release", "rollback", "发布回滚"},
		// 资产权限
		{"asset:read", "查看资产", "asset", "read", "查看资产列表和详情"},
		{"asset:write", "管理资产", "asset", "write", "创建、编辑资产"},
		{"asset:delete", "删除资产", "asset", "delete", "删除资产"},
		// CMDB 权限
		{"cmdb:read", "查看CMDB", "cmdb", "read", "查看配置项"},
		{"cmdb:write", "管理CMDB", "cmdb", "write", "管理配置项"},
		{"cmdb:delete", "删除CMDB", "cmdb", "delete", "删除配置项和关系"},
		// 报表权限
		{"report:read", "查看报表", "report", "read", "查看报表"},
		{"report:write", "管理报表", "report", "write", "创建、编辑报表"},
		// 许可证权限
		{"license:read", "查看许可证", "license", "read", "查看许可证列表和详情"},
		{"license:write", "管理许可证", "license", "write", "创建、编辑许可证"},
		{"license:delete", "删除许可证", "license", "delete", "删除许可证"},
		// 服务目录权限
		{"service:read", "查看服务", "service", "read", "查看服务目录"},
		{"service:write", "管理服务", "service", "write", "管理服务目录"},
		{"service_catalog:read", "查看服务目录", "service_catalog", "read", "查看服务目录"},
		{"service_catalog:write", "管理服务目录", "service_catalog", "write", "创建、编辑服务目录"},
		{"service_catalog:delete", "删除服务目录", "service_catalog", "delete", "删除服务目录"},
		{"service_request:read", "查看服务请求", "service_request", "read", "查看服务请求"},
		{"service_request:write", "处理服务请求", "service_request", "write", "创建、处理服务请求"},
		{"service_request:delete", "删除服务请求", "service_request", "delete", "删除服务请求"},
		// SLA权限
		{"sla:read", "查看SLA", "sla", "read", "查看SLA定义"},
		{"sla:write", "管理SLA", "sla", "write", "管理SLA定义"},
		{"sla:delete", "删除SLA", "sla", "delete", "删除SLA定义"},
		// 用户权限
		{"user:read", "查看用户", "user", "read", "查看用户列表"},
		{"user:write", "管理用户", "user", "write", "创建、编辑用户"},
		{"user:delete", "删除用户", "user", "delete", "删除用户"},
		// 组权限
		{"group:read", "查看组", "groups", "read", "查看组列表和详情"},
		{"group:write", "管理组", "groups", "write", "创建、编辑、删除组"},
		// 角色权限
		{"role:read", "查看角色", "role", "read", "查看角色列表"},
		{"role:write", "管理角色", "role", "write", "创建、编辑角色"},
		// 部门权限
		{"department:read", "查看部门", "department", "read", "查看部门列表"},
		{"department:write", "管理部门", "department", "write", "创建、编辑部门"},
		// 团队权限
		{"team:read", "查看团队", "team", "read", "查看团队列表"},
		{"team:write", "管理团队", "team", "write", "创建、编辑团队"},
		// 审批权限
		{"approval:read", "查看审批", "approval", "read", "查看审批记录"},
		{"approval:write", "管理审批", "approval", "write", "审批操作"},
		// 工作流权限
		{"workflow:read", "查看工作流", "workflow", "read", "查看工作流"},
		{"workflow:write", "管理工作流", "workflow", "write", "创建、编辑工作流"},
		// 知识库权限
		{"knowledge:read", "查看知识库", "knowledge", "read", "查看知识库文章"},
		{"knowledge:write", "管理知识库", "knowledge", "write", "创建、编辑知识库"},
		{"knowledge:delete", "删除知识库", "knowledge", "delete", "删除知识库文章"},
		// 系统权限
		{"system:read", "查看系统", "system", "read", "查看系统配置"},
		{"system:write", "系统管理", "system", "write", "管理系统配置"},
		{"org:read", "查看组织", "org", "read", "查看组织、部门和团队"},
		{"org:write", "管理组织", "org", "write", "管理组织、部门和团队"},
		{"project:read", "查看项目", "project", "read", "查看项目"},
		{"project:write", "管理项目", "project", "write", "管理项目"},
		{"application:read", "查看应用", "application", "read", "查看应用"},
		{"application:write", "管理应用", "application", "write", "管理应用"},
		{"audit:read", "查看审计", "audit", "read", "查看审计日志"},
		{"ai:read", "查看AI能力", "ai", "read", "查看AI能力和审计"},
		{"ai:write", "管理AI能力", "ai", "write", "调用和管理AI能力"},
		{"connector:write", "管理连接器", "connector", "write", "管理连接器配置"},
		{"vendor:read", "查看供应商", "vendor", "read", "查看供应商"},
		{"vendor:write", "管理供应商", "vendor", "write", "创建、编辑供应商"},
		{"vendor:delete", "删除供应商", "vendor", "delete", "删除供应商"},
		{"survey:write", "管理调研", "survey", "write", "创建、编辑满意度调研"},
		// MSP 权限
		{"msp:read", "查看MSP", "msp", "read", "查看MSP状态和上下文"},
		{"msp:write", "管理MSP", "msp", "write", "管理MSP配置"},
		{"msp_customer:read", "查看客户", "msp_customer", "read", "查看MSP客户列表和详情"},
		{"msp_customer:write", "管理客户", "msp_customer", "write", "创建、编辑MSP客户"},
		{"msp_ticket:read", "查看客户工单", "msp_ticket", "read", "查看客户工单"},
		{"msp_ticket:write", "处理客户工单", "msp_ticket", "write", "处理客户工单"},
		{"msp_allocation:read", "查看分配", "msp_allocation", "read", "查看MSP分配"},
		{"msp_allocation:write", "管理分配", "msp_allocation", "write", "创建、编辑MSP分配"},
		{"msp_report:read", "查看报表", "msp_report", "read", "查看MSP报表"},
		{"msp_report:write", "管理报表", "msp_report", "write", "生成和管理MSP报表"},
	}
	s.expectedPermissions = make([]string, 0, len(permissions))
	for _, p := range permissions {
		s.expectedPermissions = append(s.expectedPermissions, p.Code)
	}

	created := 0
	updated := 0
	for _, p := range permissions {
		existing, err := s.client.Permission.Query().
			Where(permission.CodeEQ(p.Code), permission.TenantIDEQ(t.ID)).
			First(ctx)
		if err == nil {
			_, err = existing.Update().
				SetName(p.Name).
				SetResource(p.Resource).
				SetAction(p.Action).
				SetDescription(p.Description).
				Save(ctx)
			if err != nil {
				s.sugar.Warnw("update permission failed", "error", err, "code", p.Code)
				continue
			}
			updated++
			continue
		}
		if _, err := s.client.Permission.Create().
			SetCode(p.Code).
			SetName(p.Name).
			SetResource(p.Resource).
			SetAction(p.Action).
			SetDescription(p.Description).
			SetTenantID(t.ID).
			Save(ctx); err != nil {
			s.sugar.Warnw("seed permission failed", "error", err, "code", p.Code)
			continue
		}
		created++
	}
	s.sugar.Infow("permissions ensured", "total", len(permissions), "created", created, "updated", updated)
}

// seedMenus 初始化系统菜单
func (s *Seeder) seedMenus(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip menus seed", "error", err)
		return
	}

	// 定义所有菜单
	menus := []struct {
		Name           string
		Path           string
		Icon           string
		ParentID       *int
		PermissionCode string
		SortOrder      int
	}{
		// 主菜单
		{Name: "仪表盘", Path: "/dashboard", Icon: "LayoutDashboard", PermissionCode: "", SortOrder: 10},
		{Name: "工单管理", Path: "/tickets", Icon: "FileText", PermissionCode: "ticket:read", SortOrder: 20},
		{Name: "事件管理", Path: "/incidents", Icon: "AlertCircle", PermissionCode: "incident:read", SortOrder: 30},
		{Name: "问题管理", Path: "/problems", Icon: "HelpCircle", PermissionCode: "problem:read", SortOrder: 40},
		{Name: "变更管理", Path: "/changes", Icon: "BarChart3", PermissionCode: "change:read", SortOrder: 50},
		{Name: "CMDB", Path: "/cmdb", Icon: "Database", PermissionCode: "cmdb:read", SortOrder: 60},
		{Name: "服务目录", Path: "/service-catalog", Icon: "Book", PermissionCode: "service:read", SortOrder: 70},
		{Name: "知识库", Path: "/knowledge", Icon: "HelpCircle", PermissionCode: "knowledge:read", SortOrder: 80},
		{Name: "SLA监控", Path: "/sla-dashboard", Icon: "Calendar", PermissionCode: "sla:read", SortOrder: 90},
		{Name: "报表", Path: "/reports", Icon: "TrendingUp", PermissionCode: "report:read", SortOrder: 100},
		{Name: "发布管理", Path: "/releases", Icon: "Rocket", PermissionCode: "release:read", SortOrder: 110},
		{Name: "资产管理", Path: "/assets", Icon: "Monitor", PermissionCode: "asset:read", SortOrder: 120},
		{Name: "MSP管理", Path: "/msp", Icon: "Shield", PermissionCode: "msp:read", SortOrder: 130},

		// 管理菜单
		{Name: "工作流", Path: "/workflow", Icon: "Workflow", PermissionCode: "workflow:read", SortOrder: 200},
		{Name: "用户管理", Path: "/admin/users", Icon: "Users", PermissionCode: "user:read", SortOrder: 210},
		{Name: "角色管理", Path: "/admin/roles", Icon: "Shield", PermissionCode: "role:read", SortOrder: 220},
		{Name: "组管理", Path: "/admin/groups", Icon: "Users", PermissionCode: "groups:read", SortOrder: 230},
		{Name: "部门管理", Path: "/admin/departments", Icon: "Activity", PermissionCode: "department:read", SortOrder: 240},
		{Name: "团队管理", Path: "/admin/teams", Icon: "Users", PermissionCode: "team:read", SortOrder: 250},
		{Name: "审批管理", Path: "/admin/approvals", Icon: "ClipboardList", PermissionCode: "approval:read", SortOrder: 260},
		{Name: "SLA配置", Path: "/admin/sla-definitions", Icon: "Calendar", PermissionCode: "sla:write", SortOrder: 270},
		{Name: "系统配置", Path: "/admin/system-config", Icon: "Settings", PermissionCode: "system:write", SortOrder: 280},
	}
	s.expectedMenus = make([]string, 0, len(menus))
	for _, item := range menus {
		s.expectedMenus = append(s.expectedMenus, item.Path)
	}

	for _, m := range menus {
		existing, err := s.client.Menu.Query().
			Where(menu.PathEQ(m.Path), menu.TenantIDEQ(t.ID)).
			Only(ctx)
		if err == nil {
			if _, err := existing.Update().
				SetName(m.Name).
				SetIcon(m.Icon).
				SetSortOrder(m.SortOrder).
				SetIsVisible(true).
				SetIsEnabled(true).
				SetPermissionCode(m.PermissionCode).
				Save(ctx); err != nil {
				s.sugar.Warnw("update menu failed", "error", err, "path", m.Path)
			}
			continue
		}
		if !ent.IsNotFound(err) {
			s.sugar.Warnw("query menu failed", "error", err, "path", m.Path)
			continue
		}
		builder := s.client.Menu.Create().
			SetName(m.Name).
			SetPath(m.Path).
			SetIcon(m.Icon).
			SetTenantID(t.ID).
			SetSortOrder(m.SortOrder).
			SetIsVisible(true).
			SetIsEnabled(true).
			SetPermissionCode(m.PermissionCode)

		// 设置父菜单ID
		if m.ParentID != nil {
			builder = builder.SetParentID(*m.ParentID)
		} else {
			builder = builder.SetNillableParentID(nil)
		}

		if _, err := builder.Save(ctx); err != nil {
			s.sugar.Warnw("seed menu failed", "error", err, "name", m.Name)
		}
	}
	s.sugar.Infow("menus seeded", "count", len(menus))
}

// seedMenuAndPermissionFixes 修复菜单路径和补充缺失的权限
func (s *Seeder) seedMenuAndPermissionFixes(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip fixes", "error", err)
		return
	}

	// 1. 修复菜单路径
	menuPathFixes := map[string]string{
		"/admin/sla":                "/admin/sla-definitions",
		"/admin/system":             "/admin/system-config",
		"/admin/workflows":          "/workflow",
		"/admin/tickets/assignment": "/admin/tickets/assignment-rules",
		"/admin/tickets/automation": "/admin/tickets/automation-rules",
	}

	for oldPath, newPath := range menuPathFixes {
		_, err := s.client.Menu.Update().
			Where(menu.Path(oldPath), menu.TenantIDEQ(t.ID)).
			SetPath(newPath).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("fix menu path failed", "error", err, "old_path", oldPath, "new_path", newPath)
		} else {
			s.sugar.Debugw("menu path fixed", "old_path", oldPath, "new_path", newPath)
		}
	}

	_, err = s.client.Menu.Update().
		Where(menu.Path("/admin/groups"), menu.TenantIDEQ(t.ID)).
		SetPermissionCode("groups:read").
		Save(ctx)
	if err != nil {
		s.sugar.Warnw("fix group menu permission failed", "error", err)
	}

	// 2. 补充缺失的权限
	missingPermissions := []struct {
		Code        string
		Name        string
		Resource    string
		Action      string
		Description string
	}{
		{"cmdb:read", "查看CMDB", "cmdb", "read", "查看配置项"},
		{"report:read", "查看报表", "report", "read", "查看报表"},
		{"group:read", "查看组", "groups", "read", "查看组列表和详情"},
		{"msp:read", "查看MSP", "msp", "read", "查看MSP状态和上下文"},
	}

	for _, p := range missingPermissions {
		existing, err := s.client.Permission.Query().
			Where(permission.Code(p.Code), permission.TenantIDEQ(t.ID)).
			Count(ctx)
		if err != nil {
			s.sugar.Warnw("check permission failed", "error", err, "code", p.Code)
			continue
		}
		if existing > 0 {
			continue
		}
		_, err = s.client.Permission.Create().
			SetCode(p.Code).
			SetName(p.Name).
			SetResource(p.Resource).
			SetAction(p.Action).
			SetDescription(p.Description).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("create missing permission failed", "error", err, "code", p.Code)
		} else {
			s.sugar.Infow("missing permission created", "code", p.Code)
		}
	}

	// 3. 补充缺失的菜单
	missingMenus := []struct {
		Name           string
		Path           string
		Icon           string
		PermissionCode string
		SortOrder      int
	}{
		{"工单分类", "/admin/ticket-categories", "Tag", "ticket:write", 275},
		{"CI类型管理", "/admin/cmdb-types", "Database", "cmdb:write", 290},
		{"许可证管理", "/licenses", "Key", "license:read", 125},
		{"SLA模板", "/admin/sla-templates", "Layers", "sla:write", 272},
		{"升级矩阵", "/admin/escalation-matrices", "TrendingUp", "sla:read", 273},
		{"BPMN节点分析", "/workflow/bottlenecks", "BarChart3", "workflow:read", 205},
		{"菜单管理", "/admin/menus", "List", "system:write", 285},
	}

	for _, m := range missingMenus {
		existing, err := s.client.Menu.Query().
			Where(menu.Path(m.Path), menu.TenantIDEQ(t.ID)).
			Count(ctx)
		if err != nil {
			s.sugar.Warnw("check menu failed", "error", err, "path", m.Path)
			continue
		}
		if existing > 0 {
			continue
		}
		_, err = s.client.Menu.Create().
			SetName(m.Name).
			SetPath(m.Path).
			SetIcon(m.Icon).
			SetPermissionCode(m.PermissionCode).
			SetSortOrder(m.SortOrder).
			SetIsVisible(true).
			SetIsEnabled(true).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("create missing menu failed", "error", err, "path", m.Path)
		} else {
			s.sugar.Infow("missing menu created", "path", m.Path)
		}
	}
}

// seedRolePermissions 为角色分配权限关联
func (s *Seeder) seedRolePermissions(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip role permissions seed", "error", err)
		return
	}

	// 查询所有权限，构建 code -> id 映射
	perms, err := s.client.Permission.Query().Where(permission.TenantIDEQ(t.ID)).All(ctx)
	if err != nil {
		s.sugar.Warnw("query permissions failed; skip role permissions seed", "error", err)
		return
	}
	if len(perms) == 0 {
		s.sugar.Infow("no permissions found; skip role permissions seed")
		return
	}

	permByCode := make(map[string]int, len(perms))
	for _, p := range perms {
		permByCode[p.Code] = p.ID
	}

	// 定义角色权限映射
	rolePermissionMap := map[string][]string{
		// 系统管理员：所有权限
		"sysadmin": allPermissionCodes(),
		// IT总监：全局读写（不含系统管理）
		"it_director": allExcept([]string{"system:write", "msp:write", "msp_allocation:write"}),
		// 运维总监：运维相关读写
		"ops_director": allExcept([]string{"system:write", "msp:write", "msp_allocation:write", "msp_report:write"}),
		// 运维经理：运维相关读写
		"ops_manager": {
			"ticket:read", "ticket:write", "incident:read", "incident:write",
			"problem:read", "problem:write", "change:read", "change:write",
			"asset:read", "asset:write", "cmdb:read", "cmdb:write",
			"sla:read", "workflow:read", "report:read",
			"team:read", "department:read", "user:read",
		},
		// 运维工程师：运维操作
		"ops_engineer": {
			"ticket:read", "ticket:write", "incident:read", "incident:write",
			"problem:read", "change:read", "asset:read", "asset:write",
			"cmdb:read", "cmdb:write", "sla:read", "knowledge:read", "knowledge:write",
		},
		// DBA工程师
		"dba": {
			"ticket:read", "incident:read", "problem:read", "problem:write",
			"change:read", "change:write", "asset:read", "cmdb:read", "cmdb:write",
			"knowledge:read", "knowledge:write",
		},
		// 网络安全工程师
		"network_eng": {
			"ticket:read", "incident:read", "incident:write", "problem:read",
			"change:read", "asset:read", "cmdb:read", "sla:read",
			"knowledge:read", "knowledge:write",
		},
		// 服务台主管
		"sd_manager": {
			"ticket:read", "ticket:write", "incident:read", "incident:write",
			"problem:read", "change:read", "sla:read", "sla:write",
			"knowledge:read", "knowledge:write", "report:read",
			"user:read", "team:read",
		},
		// 变更经理：负责变更生命周期、审批协同和发布联动
		"change_manager": {
			"ticket:read",
			"change:read", "change:write", "change:delete", "change:approve", "change:rollback",
			"approval:read", "approval:write",
			"release:read", "release:write", "release:approve", "release:rollback",
			"cmdb:read",
			"workflow:read", "workflow:write",
			"sla:read",
			"report:read",
			"knowledge:read", "knowledge:write",
		},
		// 服务目录管理员：负责服务目录、服务请求模板和工单模板配置
		"service_catalog_admin": {
			"service:read", "service:write",
			"service_catalog:read", "service_catalog:write", "service_catalog:delete",
			"service_request:read", "service_request:write", "service_request:delete",
			"ticket_template:read", "ticket_template:create", "ticket_template:update", "ticket_template:delete",
			"ticket_category:read", "ticket_category:create", "ticket_category:update",
			"workflow:read",
			"approval:read",
			"sla:read",
			"knowledge:read",
		},
		// 一线支持工程师
		"l1_support": {
			"ticket:read", "ticket:write", "incident:read", "incident:write",
			"knowledge:read", "user:read", "sla:read",
		},
		// 二线支持工程师
		"l2_support": {
			"ticket:read", "ticket:write", "incident:read", "incident:write",
			"problem:read", "change:read", "asset:read",
			"knowledge:read", "knowledge:write", "user:read", "sla:read",
		},
		// 三线专家
		"l3_expert": {
			"ticket:read", "ticket:write", "incident:read", "incident:write",
			"problem:read", "problem:write", "change:read", "change:write",
			"asset:read", "cmdb:read", "knowledge:read", "knowledge:write",
			"sla:read", "workflow:read",
		},
		// 研发经理
		"rd_manager": {
			"ticket:read", "problem:read", "change:read", "change:write",
			"release:read", "release:write", "workflow:read", "workflow:write",
			"knowledge:read", "knowledge:write", "report:read",
		},
		// 开发工程师
		"developer": {
			"ticket:read", "problem:read", "change:read",
			"release:read", "knowledge:read", "knowledge:write",
		},
		// 测试工程师
		"qa_engineer": {
			"ticket:read", "problem:read", "change:read",
			"release:read", "knowledge:read", "knowledge:write", "report:read",
		},
		// 安全管理员
		"security_admin": {
			"ticket:read", "incident:read", "problem:read",
			"system:read", "user:read", "role:read",
			"knowledge:read", "report:read",
		},
		// 审计管理员
		"audit_admin": {
			"ticket:read", "incident:read", "problem:read", "change:read",
			"system:read", "user:read", "role:read", "report:read",
		},
		// 部门经理
		"dept_manager": {
			"ticket:read", "ticket:write", "incident:read",
			"problem:read", "change:read", "report:read",
			"user:read", "department:read", "team:read",
			"knowledge:read",
		},
		// 团队主管
		"team_lead": {
			"ticket:read", "ticket:write", "incident:read",
			"problem:read", "change:read", "team:read",
			"user:read", "knowledge:read",
		},
		// 安全审批人：可读工单/事件/问题/变更/知识库/通知，做安全审批
		"security": {
			"ticket:read", "ticket:write",
			"incident:read", "incident:write",
			"problem:read",
			"change:read", "change:write",
			"release:read",
			"knowledge:read",
			"notification:read",
			"asset:read",
			"team:read", "user:read",
		},
		// 普通用户
		"end_user": {
			"ticket:read", "ticket:write", "knowledge:read", "service_catalog:read",
		},
		// 访客
		"guest": {
			"knowledge:read",
		},
	}
	s.expectedRolePermissions = rolePermissionMap

	// 查询所有角色并为每个角色分配权限
	roles, err := s.client.Role.Query().Where(role.TenantIDEQ(t.ID)).All(ctx)
	if err != nil {
		s.sugar.Warnw("query roles failed; skip role permissions seed", "error", err)
		return
	}

	assigned := 0
	for _, r := range roles {
		codes, ok := rolePermissionMap[r.Code]
		if !ok {
			continue // 未定义的角色跳过
		}

		// 收集该角色应拥有的权限ID
		permIDs := make([]int, 0, len(codes))
		for _, code := range codes {
			if id, exists := permByCode[code]; exists {
				permIDs = append(permIDs, id)
			}
		}
		if len(permIDs) == 0 {
			continue
		}
		managedPermissionIDs := make([]int, 0, len(s.expectedPermissions))
		for _, code := range s.expectedPermissions {
			if id, exists := permByCode[code]; exists {
				managedPermissionIDs = append(managedPermissionIDs, id)
			}
		}
		if _, err := s.client.RolePermission.Delete().
			Where(
				rolepermission.RoleIDEQ(r.ID),
				rolepermission.TenantIDEQ(t.ID),
				rolepermission.PermissionIDIn(managedPermissionIDs...),
				rolepermission.PermissionIDNotIn(permIDs...),
			).
			Exec(ctx); err != nil {
			s.sugar.Warnw("remove obsolete role permissions failed", "error", err, "role", r.Code)
		}

		// 为角色添加权限（直接写入 role_permissions 联表）
		created := 0
		for _, pid := range permIDs {
			exists, err := s.client.RolePermission.Query().
				Where(rolepermission.RoleID(r.ID), rolepermission.PermissionID(pid), rolepermission.TenantID(t.ID)).
				Exist(ctx)
			if err != nil {
				s.sugar.Warnw("check role-permission failed", "error", err, "role", r.Code, "permission_id", pid)
				continue
			}
			if exists {
				continue
			}
			_, err = s.client.RolePermission.Create().
				SetRoleID(r.ID).
				SetPermissionID(pid).
				SetTenantID(t.ID).
				Save(ctx)
			if err != nil {
				s.sugar.Warnw("create role-permission failed", "error", err, "role", r.Code, "permission_id", pid)
			} else {
				created++
			}
		}
		if created > 0 {
			s.sugar.Infow("role permissions ensured", "role", r.Code, "created", created)
			assigned++
		}
	}
	s.sugar.Infow("role permissions seed completed", "roles_assigned", assigned)
}

// allPermissionCodes 返回所有权限代码
func allPermissionCodes() []string {
	return []string{
		"ticket:read", "ticket:write", "ticket:create", "ticket:update", "ticket:assign",
		"ticket:escalate", "ticket:resolve", "ticket:close", "ticket:export", "ticket:import",
		"ticket:admin", "ticket:delete",
		"ticket_category:read", "ticket_category:create", "ticket_category:update", "ticket_category:delete",
		"ticket_tag:read", "ticket_tag:create", "ticket_tag:update", "ticket_tag:delete",
		"ticket_template:read", "ticket_template:create", "ticket_template:update", "ticket_template:delete",
		"incident:read", "incident:write", "incident:delete",
		"problem:read", "problem:write", "problem:delete",
		"change:read", "change:write", "change:delete", "change:approve", "change:rollback",
		"release:read", "release:write", "release:delete", "release:approve", "release:rollback",
		"asset:read", "asset:write", "asset:delete",
		"cmdb:read", "cmdb:write", "cmdb:delete",
		"report:read", "report:write",
		"license:read", "license:write", "license:delete",
		"service:read", "service:write",
		"service_catalog:read", "service_catalog:write", "service_catalog:delete",
		"service_request:read", "service_request:write", "service_request:delete",
		"sla:read", "sla:write", "sla:delete",
		"user:read", "user:write", "user:delete",
		"group:read", "group:write",
		"role:read", "role:write",
		"department:read", "department:write",
		"team:read", "team:write",
		"approval:read", "approval:write",
		"workflow:read", "workflow:write",
		"knowledge:read", "knowledge:write", "knowledge:delete",
		"system:read", "system:write",
		"org:read", "org:write",
		"project:read", "project:write",
		"application:read", "application:write",
		"audit:read",
		"ai:read", "ai:write",
		"connector:write",
		"vendor:read", "vendor:write", "vendor:delete",
		"survey:write",
		"msp:read", "msp:write",
		"msp_customer:read", "msp_customer:write",
		"msp_ticket:read", "msp_ticket:write",
		"msp_allocation:read", "msp_allocation:write",
		"msp_report:read", "msp_report:write",
	}
}

// allExcept 返回除指定代码外的所有权限代码
func allExcept(exclude []string) []string {
	excludeSet := make(map[string]bool, len(exclude))
	for _, code := range exclude {
		excludeSet[code] = true
	}
	result := make([]string, 0)
	for _, code := range allPermissionCodes() {
		if !excludeSet[code] {
			result = append(result, code)
		}
	}
	return result
}

func (s *Seeder) seedServiceCatalog(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip service catalog seed", "error", err)
		return
	}

	existing, err := s.client.ServiceCatalog.Query().Where(servicecatalog.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing service catalog failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("service catalog already seeded")
		return
	}

	for _, svc := range s.config.ServiceCatalog {
		_, err := s.client.ServiceCatalog.Create().
			SetName(svc.Name).
			SetDescription(svc.Description).
			SetCategory(svc.Category).
			SetServiceType(svc.ServiceType).
			SetRequiresApproval(svc.RequiresApproval).
			SetDeliveryTime(svc.DeliveryTime).
			SetStatus("active").
			SetIsActive(true).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed service catalog failed", "error", err, "name", svc.Name)
		}
	}
	s.sugar.Infow("service catalog seeded", "count", len(s.config.ServiceCatalog))
}

// seedTicketTypes 初始化默认工单类型
func (s *Seeder) seedTicketTypes(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip ticket types seed", "error", err)
		return
	}

	// 获取admin用户ID
	admin, err := s.client.User.Query().Where(user.UsernameEQ("admin"), user.TenantIDEQ(t.ID)).First(ctx)
	if err != nil {
		s.sugar.Warnw("admin user not found; skip ticket types seed", "error", err)
		return
	}

	// 检查ticket_types表是否存在
	rawDB := database.GetRawDB()
	if rawDB == nil {
		s.sugar.Warnw("rawDB not available; skip ticket types seed")
		return
	}

	// 检查 ticket_types 表是否存在
	var tableExists bool
	err = rawDB.QueryRowContext(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'ticket_types')").Scan(&tableExists)
	if err != nil || !tableExists {
		s.sugar.Infow("ticket_types table does not exist; skip seed")
		return
	}

	// 检查是否已有工单类型
	var count int
	err = rawDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM ticket_types WHERE tenant_id = $1", t.ID).Scan(&count)
	if err != nil {
		s.sugar.Warnw("check existing ticket types failed", "error", err)
		return
	}
	if count > 0 {
		s.sugar.Infow("ticket types already seeded")
		return
	}

	// 定义默认工单类型（与前端ticket-type-presets.ts保持一致）
	ticketTypes := []struct {
		Code        string
		Name        string
		Description string
		Icon        string
		Color       string
	}{
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

	for _, tt := range ticketTypes {
		_, err := rawDB.ExecContext(
			ctx, `
			INSERT INTO ticket_types (
				code, name, description, icon, color, status,
				custom_fields, approval_enabled, approval_chain,
				sla_enabled, auto_assign_enabled, assignment_rules,
				notification_config, permission_config,
				created_by, tenant_id, created_at, updated_at, usage_count
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, 0)
		`,
			tt.Code, tt.Name, tt.Description, tt.Icon, tt.Color, "active",
			"[]", false, "[]",
			false, false, "[]",
			"{}", "{}",
			admin.ID, t.ID, time.Now(), time.Now(),
		)
		if err != nil {
			s.sugar.Warnw("seed ticket type failed", "error", err, "code", tt.Code)
		}
	}
	s.sugar.Infow("ticket types seeded", "count", len(ticketTypes))
}

// seedCITypes 初始化CI类型种子数据
func (s *Seeder) seedCITypes(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip CI types seed", "error", err)
		return
	}

	// 检查是否已有CI类型
	existing, err := s.client.CIType.Query().Count(ctx)
	if err != nil {
		s.sugar.Warnw("failed to query CI types; skip seed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("CI types already seeded", "count", existing)
		return
	}

	// 使用配置中的CI类型，如果没有配置则使用默认值
	ciTypes := s.config.CITypes
	if len(ciTypes) == 0 {
		// 默认CI类型（is_active 省略，走 schema 默认启用）
		ciTypes = []CITypeSeed{
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

	for _, ct := range ciTypes {
		create := s.client.CIType.Create().
			SetName(ct.Name).
			SetDescription(ct.Description).
			SetIcon(ct.Icon).
			SetColor(ct.Color).
			SetTenantID(t.ID)
		// 仅在配置显式声明时覆盖；省略时保持 schema 默认 is_active=true，
		// 避免 Go 零值 false 把预置类型隐式种为禁用。
		if ct.IsActive != nil {
			create.SetIsActive(*ct.IsActive)
		}
		if _, err := create.Save(ctx); err != nil {
			s.sugar.Warnw("seed CI type failed", "error", err, "name", ct.Name)
		}
	}
	s.sugar.Infow("CI types seeded", "count", len(ciTypes))
}

// seedStandardChanges 初始化标准变更模板种子数据
func (s *Seeder) seedStandardChanges(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip standard changes seed", "error", err)
		return
	}

	// 检查是否已有标准变更模板
	existing, err := s.client.StandardChange.Query().Count(ctx)
	if err != nil {
		s.sugar.Warnw("failed to query standard changes; skip seed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("standard changes already seeded", "count", existing)
		return
	}

	// 获取测试用户
	users, err := s.client.User.Query().Where(user.TenantIDEQ(t.ID)).Limit(1).All(ctx)
	if err != nil || len(users) == 0 {
		s.sugar.Warnw("no users found; skip standard changes seed", "error", err)
		return
	}
	creatorID := users[0].ID

	// 使用配置中的数据，如果没有配置则使用默认值
	standardChanges := s.config.StandardChanges
	if len(standardChanges) == 0 {
		standardChanges = []StandardChangeSeed{
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

	for _, sc := range standardChanges {
		_, err := s.client.StandardChange.Create().
			SetTitle(sc.Title).
			SetDescription(sc.Description).
			SetImplementationPlan(sc.ImplementationPlan).
			SetRollbackPlan(sc.RollbackPlan).
			SetJustification(sc.Justification).
			SetCategory(sc.Category).
			SetRiskLevel(sc.RiskLevel).
			SetImpactScope(sc.ImpactScope).
			SetExpectedDuration(sc.ExpectedDuration).
			SetApprovalRequired(sc.ApprovalRequired).
			SetAffectedCis(sc.AffectedCIs).
			SetPrerequisites(sc.Prerequisites).
			SetRemarks(sc.Remarks).
			SetCreatedBy(creatorID).
			SetTenantID(t.ID).
			SetIsActive(true).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed standard change failed", "error", err, "title", sc.Title)
		}
	}
	s.sugar.Infow("standard changes seeded", "count", len(standardChanges))
}

// seedTicketTags 初始化标签种子数据
func (s *Seeder) seedTicketTags(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip ticket tags seed", "error", err)
		return
	}

	// 检查是否已有标签
	existing, err := s.client.Tag.Query().Count(ctx)
	if err != nil {
		s.sugar.Warnw("failed to query ticket tags; skip seed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("ticket tags already seeded", "count", existing)
		return
	}

	// 使用配置中的数据，如果没有配置则使用默认值
	ticketTags := s.config.TicketTags
	if len(ticketTags) == 0 {
		ticketTags = []TicketTagSeed{
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

	for _, tag := range ticketTags {
		_, err := s.client.Tag.Create().
			SetName(tag.Name).
			SetCode(tag.Code).
			SetDescription(tag.Description).
			SetColor(tag.Color).
			SetTenantID(t.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed ticket tag failed", "error", err, "name", tag.Name)
		}
	}
	s.sugar.Infow("ticket tags seeded", "count", len(ticketTags))
}

// seedIncidentCategories 初始化事件分类种子数据
func (s *Seeder) seedIncidentCategories(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip incident categories seed", "error", err)
		return
	}

	// 检查是否已有分类数据
	existing, err := s.client.TicketCategory.Query().Count(ctx)
	if err != nil {
		s.sugar.Warnw("failed to query categories; skip seed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("incident categories already seeded", "count", existing)
		return
	}

	// 使用配置中的数据，如果没有配置则使用默认值
	categories := s.config.IncidentCategories
	if len(categories) == 0 {
		categories = []TicketCategorySeed{
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

	for _, cat := range categories {
		code := cat.Code
		if code == "" {
			code = strings.ToLower(strings.ReplaceAll(cat.Name, " ", "_"))
		}
		_, err := s.client.TicketCategory.Create().
			SetName(cat.Name).
			SetCode(code).
			SetDescription(cat.Description).
			SetTenantID(t.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed incident category failed", "error", err, "name", cat.Name)
		}
	}
	s.sugar.Infow("incident categories seeded", "count", len(categories))
}
