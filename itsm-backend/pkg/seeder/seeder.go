package seeder

import (
	"context"
	"encoding/json"
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
	"itsm-backend/ent/mspallocation"
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
	IsActive    bool   `json:"is_active"`
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
	client              *ent.Client
	sugar               *zap.SugaredLogger
	config              *SeedConfig
	appConfig           *config.Config
	bpmnTemplateService *service.BPMNTemplateService
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
			{BusinessType: "incident", ProcessDefinitionKey: "incident_emergency_flow", IsDefault: true},
			{BusinessType: "problem", ProcessDefinitionKey: "problem_management_flow", IsDefault: true},
			{BusinessType: "change", ProcessDefinitionKey: "change_normal_flow", IsDefault: true},
			{BusinessType: "service_request", ProcessDefinitionKey: "service_request_flow", IsDefault: true},
			{BusinessType: "improvement", ProcessDefinitionKey: "ticket_general_flow", IsDefault: true},
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
	rootTenant := s.seedDefaultTenant(ctx)
	s.seedDepartments(ctx)
	s.seedTeams(ctx)
	s.seedRoles(ctx)
	s.seedPermissions(ctx) // 新增：初始化权限
	s.seedMenus(ctx)       // 新增：初始化菜单
	s.backfillAdminRole(ctx)
	s.seedAdmin(ctx)
	s.seedModeTenants(ctx, rootTenant)
	s.seedUser1(ctx)
	s.seedSecurity1(ctx)
	s.backfillUserRole(ctx)
	s.seedCloudServiceTemplates(ctx)
	s.seedAssets(ctx)
	s.seedAssetLicenses(ctx)
	s.seedReleases(ctx)
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
	s.seedIncidents(ctx)              // 新增：初始化事件数据
	s.seedProblems(ctx)               // 新增：初始化问题数据
	s.seedChanges(ctx)                // 新增：初始化变更数据
	s.seedKnowledgeArticles(ctx)      // 新增：初始化知识库文章
	s.seedStandardChanges(ctx)        // 新增：初始化标准变更模板
	s.seedKnownErrors(ctx)            // 新增：初始化已知错误
	s.seedTicketTags(ctx)             // 新增：初始化标签
	s.seedMenuAndPermissionFixes(ctx) // 修复：更新菜单路径和补充缺失权限
	s.seedRolePermissions(ctx)        // 新增：为角色分配权限
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

func (s *Seeder) seedModeTenants(ctx context.Context, rootTenant *ent.Tenant) {
	if rootTenant == nil {
		return
	}

	switch s.deploymentMode() {
	case tenantmode.DeploymentModePrivate:
		s.ensureTenant(ctx, tenantSeed{
			Code:            "hq",
			Name:            "Group Headquarters",
			Type:            tenantmode.TenantTypeInternal,
			ParentTenantID:  &rootTenant.ID,
			BillingEnabled:  true,
			CostCenterCode:  "HQ-001",
			LegalEntityCode: "GROUP",
			Currency:        "CNY",
			ServiceTier:     "enterprise",
			OwnerContact:    "hq-admin@example.com",
		})
	case tenantmode.DeploymentModeSaaSMSP:
		customerOne := s.ensureTenant(ctx, tenantSeed{
			Code:            "customer-a",
			Name:            "Customer A",
			Type:            tenantmode.TenantTypeMSPCustomer,
			MSPProviderID:   &rootTenant.ID,
			ParentTenantID:  &rootTenant.ID,
			BillingEnabled:  true,
			PlanCode:        "msp-enterprise",
			CostCenterCode:  "CUS-A",
			LegalEntityCode: "CUS-A-LE",
			Currency:        "CNY",
			ServiceTier:     "gold",
			OwnerContact:    "it-manager@customer-a.example.com",
		})
		customerTwo := s.ensureTenant(ctx, tenantSeed{
			Code:            "customer-b",
			Name:            "Customer B",
			Type:            tenantmode.TenantTypeMSPCustomer,
			MSPProviderID:   &rootTenant.ID,
			ParentTenantID:  &rootTenant.ID,
			BillingEnabled:  true,
			PlanCode:        "msp-standard",
			CostCenterCode:  "CUS-B",
			LegalEntityCode: "CUS-B-LE",
			Currency:        "CNY",
			ServiceTier:     "silver",
			OwnerContact:    "ops@customer-b.example.com",
		})
		if customerOne != nil {
			s.seedDefaultMSPAllocations(ctx, rootTenant, customerOne)
		}
		if customerTwo != nil {
			s.seedDefaultMSPAllocations(ctx, rootTenant, customerTwo)
		}
	}
}

type tenantSeed struct {
	Code            string
	Name            string
	Type            string
	ParentTenantID  *int
	MSPProviderID   *int
	PlanCode        string
	BillingEnabled  bool
	CostCenterCode  string
	LegalEntityCode string
	Currency        string
	ServiceTier     string
	OwnerContact    string
}

func (s *Seeder) ensureTenant(ctx context.Context, input tenantSeed) *ent.Tenant {
	existing, err := s.client.Tenant.Query().Where(tenant.CodeEQ(input.Code)).First(ctx)
	if err == nil && existing != nil {
		updated, updateErr := existing.Update().
			SetName(input.Name).
			SetType(tenant.Type(input.Type)).
			SetStatus("active").
			SetNillableParentTenantID(input.ParentTenantID).
			SetNillableMspProviderID(input.MSPProviderID).
			SetBillingEnabled(input.BillingEnabled).
			SetNillablePlanCode(nilIfEmpty(input.PlanCode)).
			SetNillableCostCenterCode(nilIfEmpty(input.CostCenterCode)).
			SetNillableLegalEntityCode(nilIfEmpty(input.LegalEntityCode)).
			SetNillableCurrency(nilIfEmpty(input.Currency)).
			SetNillableServiceTier(nilIfEmpty(input.ServiceTier)).
			SetNillableOwnerContact(nilIfEmpty(input.OwnerContact)).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if updateErr == nil {
			return updated
		}
		s.sugar.Warnw("update tenant seed failed", "code", input.Code, "error", updateErr)
		return existing
	}

	created, createErr := s.client.Tenant.Create().
		SetName(input.Name).
		SetCode(input.Code).
		SetType(tenant.Type(input.Type)).
		SetStatus("active").
		SetNillableParentTenantID(input.ParentTenantID).
		SetNillableMspProviderID(input.MSPProviderID).
		SetBillingEnabled(input.BillingEnabled).
		SetNillablePlanCode(nilIfEmpty(input.PlanCode)).
		SetNillableCostCenterCode(nilIfEmpty(input.CostCenterCode)).
		SetNillableLegalEntityCode(nilIfEmpty(input.LegalEntityCode)).
		SetNillableCurrency(nilIfEmpty(input.Currency)).
		SetNillableServiceTier(nilIfEmpty(input.ServiceTier)).
		SetNillableOwnerContact(nilIfEmpty(input.OwnerContact)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if createErr != nil {
		s.sugar.Warnw("create tenant seed failed", "code", input.Code, "error", createErr)
		return nil
	}
	return created
}

func (s *Seeder) seedDefaultMSPAllocations(ctx context.Context, providerTenant *ent.Tenant, customerTenant *ent.Tenant) {
	if providerTenant == nil || customerTenant == nil {
		return
	}

	adminUser, err := s.client.User.Query().
		Where(
			user.UsernameEQ("admin"),
			user.TenantIDEQ(providerTenant.ID),
		).
		Only(ctx)
	if err != nil {
		return
	}

	exists, err := s.client.MSPAllocation.Query().
		Where(
			mspallocation.MspUserIDEQ(adminUser.ID),
			mspallocation.CustomerTenantIDEQ(customerTenant.ID),
			mspallocation.DeassignedAtIsNil(),
		).
		Exist(ctx)
	if err == nil && exists {
		return
	}

	if _, err := s.client.MSPAllocation.Create().
		SetMspUserID(adminUser.ID).
		SetCustomerTenantID(customerTenant.ID).
		SetRole("primary").
		SetAssignedAt(time.Now()).
		Save(ctx); err != nil {
		s.sugar.Warnw("seed MSP allocation failed", "error", err)
	}
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

	if existing != nil {
		_, err = s.client.User.Update().
			Where(user.ID(existing.ID)).
			SetPasswordHash(string(passHash)).
			SetRole("super_admin").
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("update admin password failed", "error", err)
		} else {
			s.sugar.Infow("seed admin updated", "username", "admin")
		}
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

	existing, err := s.client.Department.Query().Where(department.TenantIDEQ(t.ID)).Count(ctx)
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

	existing, err := s.client.Team.Query().Where(team.TenantIDEQ(t.ID)).Count(ctx)
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

	existing, err := s.client.Role.Query().Where(role.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing roles failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("roles already seeded")
		return
	}

	for _, r := range s.config.Roles {
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

func (s *Seeder) backfillAdminRole(ctx context.Context) {
	if _, err := s.client.User.Update().
		Where(user.UsernameEQ("admin")).
		SetRole("super_admin").
		Save(ctx); err != nil {
		s.sugar.Warnw("admin role backfill failed", "error", err)
	} else {
		s.sugar.Infow("admin role backfilled")
	}
}

func (s *Seeder) seedUser1(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip user1 seed", "error", err)
		return
	}
	_, err = s.client.User.Query().Where(user.UsernameEQ("user1"), user.TenantIDEQ(t.ID)).First(ctx)
	if err == nil {
		s.sugar.Infow("seed user1 already exists")
		return
	}
	user1Password := os.Getenv("SEED_USER1_PASSWORD")
	if user1Password == "" {
		s.sugar.Infow("SEED_USER1_PASSWORD env var not set; skip user1 seed")
		return
	}
	passHash, err := bcrypt.GenerateFromPassword([]byte(user1Password), bcrypt.DefaultCost)
	if err != nil {
		s.sugar.Warnw("generate bcrypt for user1 failed", "error", err)
		return
	}
	if _, err := s.client.User.Create().
		SetUsername("user1").
		SetRole("end_user").
		SetPasswordHash(string(passHash)).
		SetEmail("user1@example.com").
		SetName("普通用户").
		SetDepartment("IT部门").
		SetActive(true).
		SetTenantID(t.ID).
		Save(ctx); err != nil {
		s.sugar.Warnw("seed user1 failed", "error", err)
	} else {
		s.sugar.Infow("seed user1 created", "username", "user1")
	}
}

func (s *Seeder) seedSecurity1(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip security1 seed", "error", err)
		return
	}
	_, err = s.client.User.Query().Where(user.UsernameEQ("security1"), user.TenantIDEQ(t.ID)).First(ctx)
	if err == nil {
		s.sugar.Infow("seed security1 already exists")
		return
	}
	security1Password := os.Getenv("SEED_SECURITY1_PASSWORD")
	if security1Password == "" {
		s.sugar.Infow("SEED_SECURITY1_PASSWORD env var not set; skip security1 seed")
		return
	}
	passHash, err := bcrypt.GenerateFromPassword([]byte(security1Password), bcrypt.DefaultCost)
	if err != nil {
		s.sugar.Warnw("generate bcrypt for security1 failed", "error", err)
		return
	}
	if _, err := s.client.User.Create().
		SetUsername("security1").
		SetRole("security").
		SetPasswordHash(string(passHash)).
		SetEmail("security1@example.com").
		SetName("安全审批人").
		SetDepartment("IT安全").
		SetActive(true).
		SetTenantID(t.ID).
		Save(ctx); err != nil {
		s.sugar.Warnw("seed security1 failed", "error", err)
	} else {
		s.sugar.Infow("seed security1 created", "username", "security1")
	}
}

func (s *Seeder) backfillUserRole(ctx context.Context) {
	n, err := s.client.User.Update().
		Where(user.RoleEQ("user")).
		SetRole("end_user").
		Save(ctx)
	if err != nil {
		s.sugar.Warnw("backfill role user->end_user failed", "error", err)
	} else if n > 0 {
		s.sugar.Infow("backfilled roles", "updated", n)
	}

	// T005: 为测试方案 seed 额外的角色账号
	s.seedRoleTestAccounts(ctx)
}

// T005: 角色测试账号 seeder
// FR-604, R-07: engineer1 / manager1 / tenant1admin
func (s *Seeder) seedRoleTestAccounts(ctx context.Context) {
	// 确保 tenant_test 租户存在
	testTenant := s.ensureTenant(ctx, tenantSeed{
		Code: "tenant_test",
		Name: "测试租户",
		Type: "customer",
	})
	if testTenant == nil {
		s.sugar.Warnw("tenant_test 创建失败，跳过角色测试账号 seed")
		return
	}

	// 创建 engineer1 (角色: technician 用于处理工单)
	s.createTestUser(ctx, testTenant, "engineer1", "eng123", "technician", "工程师")
	// 创建 manager1 (角色: manager 用于审批)
	s.createTestUser(ctx, testTenant, "manager1", "mgr123", "manager", "审批经理")
	// 创建 tenant1admin (角色: admin 用于租户管理)
	s.createTestUser(ctx, testTenant, "tenant1admin", "ta123456", "admin", "租户管理员")
}

func (s *Seeder) createTestUser(ctx context.Context, t *ent.Tenant, username, password, roleName, displayName string) {
	// 检查用户是否已存在
	existing, err := s.client.User.Query().Where(user.UsernameEQ(username), user.TenantIDEQ(t.ID)).First(ctx)
	if err == nil && existing != nil {
		s.sugar.Infow("测试用户已存在", "username", username)
		return
	}

	// 创建新用户 - 使用 switch 确保类型正确
	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.sugar.Warnw("bcrypt 密码生成失败", "username", username, "error", err)
		return
	}

	switch roleName {
	case "technician":
		_, err = s.client.User.Create().
			SetUsername(username).
			SetRole("technician").
			SetPasswordHash(string(passHash)).
			SetEmail(username + "@test.com").
			SetName(displayName).
			SetDepartment("IT部门").
			SetActive(true).
			SetTenantID(t.ID).
			Save(ctx)
	case "manager":
		_, err = s.client.User.Create().
			SetUsername(username).
			SetRole("manager").
			SetPasswordHash(string(passHash)).
			SetEmail(username + "@test.com").
			SetName(displayName).
			SetDepartment("IT部门").
			SetActive(true).
			SetTenantID(t.ID).
			Save(ctx)
	case "admin":
		_, err = s.client.User.Create().
			SetUsername(username).
			SetRole("admin").
			SetPasswordHash(string(passHash)).
			SetEmail(username + "@test.com").
			SetName(displayName).
			SetDepartment("IT部门").
			SetActive(true).
			SetTenantID(t.ID).
			Save(ctx)
	default:
		_, err = s.client.User.Create().
			SetUsername(username).
			SetRole("end_user").
			SetPasswordHash(string(passHash)).
			SetEmail(username + "@test.com").
			SetName(displayName).
			SetDepartment("IT部门").
			SetActive(true).
			SetTenantID(t.ID).
			Save(ctx)
	}

	if err != nil {
		s.sugar.Warnw("创建测试用户失败", "username", username, "error", err)
	} else {
		s.sugar.Infow("测试用户已创建", "username", username, "role", roleName)
	}
}

// seedCloudServiceTemplates, seedAssets, seedAssets, seedReleases
// 保留原有的云服务、资产、发布等种子数据（这些数据通常比较稳定）

func (s *Seeder) seedCloudServiceTemplates(ctx context.Context) {
	// 保留原有实现...
}

func (s *Seeder) seedAssets(ctx context.Context) {
	// 保留原有实现...
}

func (s *Seeder) seedAssetLicenses(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip asset licenses seed", "error", err)
		return
	}

	// 检查是否已有许可证数据
	existing, err := s.client.AssetLicense.Query().Count(ctx)
	if err != nil {
		s.sugar.Warnw("failed to query asset licenses; skip seed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("asset licenses already seeded", "count", existing)
		return
	}

	// 默认许可证种子数据
	licenses := []struct {
		Name          string
		Vendor        string
		LicenseType   string
		TotalQuantity int
		UsedQuantity  int
		Status        string
		PurchaseDate  string
		ExpiryDate    string
		Description   string
	}{
		{"Microsoft Office 365 E3", "Microsoft", "subscription", 100, 75, "active", "2024-01-15", "2025-01-15", "Office 365 企业版订阅"},
		{"Adobe Creative Cloud", "Adobe", "per-seat", 50, 35, "active", "2024-02-20", "2025-02-20", "Adobe创意套件许可证"},
		{"VMware vSphere Enterprise", "VMware", "perpetual", 10, 8, "active", "2022-06-01", "", "VMware虚拟化平台许可证"},
		{"Slack Enterprise Grid", "Slack", "per-user", 200, 180, "expiring-soon", "2023-12-01", "2024-12-01", "Slack企业版协作平台"},
		{"AWS Enterprise Support", "Amazon", "subscription", 1, 1, "active", "2024-03-01", "2025-03-01", "AWS企业级支持服务"},
		{"Salesforce CRM Enterprise", "Salesforce", "per-user", 30, 28, "active", "2024-01-01", "2025-01-01", "Salesforce客户关系管理系统"},
		{"Zoom Enterprise", "Zoom", "perpetual", 100, 90, "active", "2023-07-15", "", "Zoom企业视频会议许可证"},
		{"Jira Software Enterprise", "Atlassian", "perpetual", 50, 42, "expired", "2022-01-01", "2024-01-01", "Jira软件项目管理工具"},
	}

	for _, lic := range licenses {
		_, err := s.client.AssetLicense.Create().
			SetName(lic.Name).
			SetVendor(lic.Vendor).
			SetLicenseType(lic.LicenseType).
			SetTotalQuantity(lic.TotalQuantity).
			SetUsedQuantity(lic.UsedQuantity).
			SetStatus(lic.Status).
			SetPurchaseDate(lic.PurchaseDate).
			SetExpiryDate(lic.ExpiryDate).
			SetDescription(lic.Description).
			SetTenantID(t.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed asset license failed", "error", err, "name", lic.Name)
		}
	}
	s.sugar.Infow("asset licenses seeded", "count", len(licenses))
}

func (s *Seeder) seedReleases(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip releases seed", "error", err)
		return
	}

	// 检查是否已有发布数据
	existing, err := s.client.Release.Query().Count(ctx)
	if err != nil {
		s.sugar.Warnw("failed to query releases; skip seed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("releases already seeded", "count", existing)
		return
	}

	// 获取测试用户
	users, err := s.client.User.Query().Where(user.TenantIDEQ(t.ID)).Limit(1).All(ctx)
	if err != nil || len(users) == 0 {
		s.sugar.Warnw("no users found; skip releases seed", "error", err)
		return
	}
	creatorID := users[0].ID

	// 默认发布种子数据
	releases := []struct {
		ReleaseNumber string
		Title         string
		Description   string
		Type          string
		Status        string
		Severity      string
		Environment   string
		PlannedDate   time.Time
	}{
		{"REL-2024-001", "v3.2.0 版本发布", "核心系统版本升级，包含性能优化和新功能", "major", "completed", "high", "production", time.Now().AddDate(0, -1, 0)},
		{"REL-2024-002", "安全补丁 KB-2024-03", "修复CVE-2024-1234高危漏洞", "hotfix", "completed", "critical", "production", time.Now().AddDate(0, -2, 0)},
		{"REL-2024-003", "数据库迁移v2.1", "数据库架构升级和索引优化", "minor", "in-progress", "medium", "staging", time.Now().AddDate(0, 0, 7)},
		{"REL-2024-004", "前端组件库升级", "React组件库升级到最新版本", "minor", "scheduled", "low", "staging", time.Now().AddDate(0, 1, 0)},
		{"REL-2024-005", "API网关版本更新", "Kong网关升级和配置优化", "patch", "draft", "medium", "dev", time.Now().AddDate(0, 0, 14)},
	}

	for _, rel := range releases {
		_, err := s.client.Release.Create().
			SetReleaseNumber(rel.ReleaseNumber).
			SetTitle(rel.Title).
			SetDescription(rel.Description).
			SetType(rel.Type).
			SetStatus(rel.Status).
			SetSeverity(rel.Severity).
			SetEnvironment(rel.Environment).
			SetPlannedReleaseDate(rel.PlannedDate).
			SetCreatedBy(creatorID).
			SetTenantID(t.ID).
			SetIsEmergency(false).
			SetRequiresApproval(true).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed release failed", "error", err, "title", rel.Title)
		}
	}
	s.sugar.Infow("releases seeded", "count", len(releases))
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

	existing, err := s.client.SLAPolicy.Query().Where(slapolicy.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing SLA policies failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("SLA policies already seeded")
		return
	}

	for _, sla := range s.config.SLAPolicies {
		_, err := s.client.SLAPolicy.Create().
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
	}
	s.sugar.Infow("SLA policies seeded", "count", len(s.config.SLAPolicies))
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

	// 检查是否已有权限
	existing, err := s.client.Permission.Query().Where(permission.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing permissions failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("permissions already seeded")
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
		{"ticket:delete", "删除工单", "ticket", "delete", "删除工单"},
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
		// 发布权限
		{"release:read", "查看发布", "release", "read", "查看发布列表和详情"},
		{"release:write", "管理发布", "release", "write", "创建、编辑发布"},
		{"release:delete", "删除发布", "release", "delete", "删除发布"},
		// 资产权限
		{"asset:read", "查看资产", "asset", "read", "查看资产列表和详情"},
		{"asset:write", "管理资产", "asset", "write", "创建、编辑资产"},
		{"asset:delete", "删除资产", "asset", "delete", "删除资产"},
		// CMDB 权限
		{"cmdb:read", "查看CMDB", "cmdb", "read", "查看配置项"},
		{"cmdb:write", "管理CMDB", "cmdb", "write", "管理配置项"},
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
		// SLA权限
		{"sla:read", "查看SLA", "sla", "read", "查看SLA定义"},
		{"sla:write", "管理SLA", "sla", "write", "管理SLA定义"},
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
		// 系统权限
		{"system:read", "查看系统", "system", "read", "查看系统配置"},
		{"system:write", "系统管理", "system", "write", "管理系统配置"},
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

	for _, p := range permissions {
		if _, err := s.client.Permission.Create().
			SetCode(p.Code).
			SetName(p.Name).
			SetResource(p.Resource).
			SetAction(p.Action).
			SetDescription(p.Description).
			SetTenantID(t.ID).
			Save(ctx); err != nil {
			s.sugar.Warnw("seed permission failed", "error", err, "code", p.Code)
		}
	}
	s.sugar.Infow("permissions seeded", "count", len(permissions))
}

// seedMenus 初始化系统菜单
func (s *Seeder) seedMenus(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip menus seed", "error", err)
		return
	}

	// 检查是否已有菜单
	existing, err := s.client.Menu.Query().Where(menu.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing menus failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("menus already seeded")
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
		{Name: "组管理", Path: "/admin/groups", Icon: "Users", PermissionCode: "group:read", SortOrder: 230},
		{Name: "部门管理", Path: "/admin/departments", Icon: "Activity", PermissionCode: "department:read", SortOrder: 240},
		{Name: "团队管理", Path: "/admin/teams", Icon: "Users", PermissionCode: "team:read", SortOrder: 250},
		{Name: "审批管理", Path: "/admin/approvals", Icon: "ClipboardList", PermissionCode: "approval:read", SortOrder: 260},
		{Name: "SLA配置", Path: "/admin/sla-definitions", Icon: "Calendar", PermissionCode: "sla:write", SortOrder: 270},
		{Name: "系统配置", Path: "/admin/system-config", Icon: "Settings", PermissionCode: "system:write", SortOrder: 280},
	}

	for _, m := range menus {
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
		{"BPMN节点分析", "/workflow/bottlenecks", "BarChart3", "workflow:read", 205},
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
	allPermIDs := make([]int, 0, len(perms))
	for _, p := range perms {
		permByCode[p.Code] = p.ID
		allPermIDs = append(allPermIDs, p.ID)
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
			"ticket:read", "ticket:write", "knowledge:read",
		},
		// 访客
		"guest": {
			"knowledge:read",
		},
	}

	// 查询所有角色并为每个角色分配权限
	roles, err := s.client.Role.Query().Where(role.TenantIDEQ(t.ID)).All(ctx)
	if err != nil {
		s.sugar.Warnw("query roles failed; skip role permissions seed", "error", err)
		return
	}

	assigned := 0
	for _, r := range roles {
		// 检查该角色是否已有权限分配（查询 role_permissions 联表）
		existingCount, err := s.client.RolePermission.Query().
			Where(rolepermission.RoleID(r.ID)).
			Count(ctx)
		if err != nil {
			s.sugar.Warnw("check role permissions failed", "error", err, "role", r.Code)
			continue
		}
		if existingCount > 0 {
			continue // 跳过已有权限的角色
		}

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

		// 为角色添加权限（直接写入 role_permissions 联表）
		created := 0
		for _, pid := range permIDs {
			_, err := s.client.RolePermission.Create().
				SetRoleID(r.ID).
				SetPermissionID(pid).
				Save(ctx)
			if err != nil {
				s.sugar.Warnw("create role-permission failed", "error", err, "role", r.Code, "permission_id", pid)
			} else {
				created++
			}
		}
		if created > 0 {
			s.sugar.Infow("role permissions assigned", "role", r.Code, "count", created)
			assigned++
		}
	}
	s.sugar.Infow("role permissions seed completed", "roles_assigned", assigned)
}

// allPermissionCodes 返回所有权限代码
func allPermissionCodes() []string {
	return []string{
		"ticket:read", "ticket:write", "ticket:delete",
		"incident:read", "incident:write", "incident:delete",
		"problem:read", "problem:write", "problem:delete",
		"change:read", "change:write", "change:delete",
		"release:read", "release:write", "release:delete",
		"asset:read", "asset:write", "asset:delete",
		"cmdb:read", "cmdb:write",
		"report:read", "report:write",
		"license:read", "license:write", "license:delete",
		"service:read", "service:write",
		"sla:read", "sla:write",
		"user:read", "user:write", "user:delete",
		"group:read", "group:write",
		"role:read", "role:write",
		"department:read", "department:write",
		"team:read", "team:write",
		"approval:read", "approval:write",
		"workflow:read", "workflow:write",
		"knowledge:read", "knowledge:write",
		"system:read", "system:write",
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

// strPtr 字符串指针辅助函数
func strPtr(s string) *string {
	return &s
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
				sla_enabled, auto_assign_enabled,
				notification_config, permission_config,
				created_by, tenant_id, created_at, updated_at, usage_count
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, 0)
		`,
			tt.Code, tt.Name, tt.Description, tt.Icon, tt.Color, "active",
			"[]", false, "[]",
			false, false,
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
		// 默认CI类型
		ciTypes = []CITypeSeed{
			{Name: "server", Description: "服务器", Icon: "server", Color: "#28a745", IsActive: true},
			{Name: "database", Description: "数据库", Icon: "database", Color: "#fd7e14", IsActive: true},
			{Name: "network", Description: "网络设备", Icon: "network", Color: "#17a2b8", IsActive: true},
			{Name: "storage", Description: "存储设备", Icon: "storage", Color: "#e83e8c", IsActive: true},
			{Name: "application", Description: "应用服务", Icon: "app", Color: "#6610f2", IsActive: true},
			{Name: "middleware", Description: "中间件", Icon: "middleware", Color: "#e74c3c", IsActive: true},
			{Name: "cloud_vm", Description: "云虚拟机", Icon: "cloud", Color: "#6f42c1", IsActive: true},
			{Name: "kubernetes", Description: "Kubernetes资源", Icon: "kubernetes", Color: "#20c997", IsActive: true},
		}
	}

	for _, ct := range ciTypes {
		_, err := s.client.CIType.Create().
			SetName(ct.Name).
			SetDescription(ct.Description).
			SetIcon(ct.Icon).
			SetColor(ct.Color).
			SetIsActive(ct.IsActive).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed CI type failed", "error", err, "name", ct.Name)
		}
	}
	s.sugar.Infow("CI types seeded", "count", len(ciTypes))
}

// seedIncidents 初始化事件种子数据
func (s *Seeder) seedIncidents(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip incidents seed", "error", err)
		return
	}

	// 检查是否已有事件数据
	existing, err := s.client.Incident.Query().Count(ctx)
	if err != nil {
		s.sugar.Warnw("failed to query incidents; skip seed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("incidents already seeded", "count", existing)
		return
	}

	// 获取一个测试用户
	users, err := s.client.User.Query().Where(user.TenantIDEQ(t.ID)).Limit(1).All(ctx)
	if err != nil || len(users) == 0 {
		s.sugar.Warnw("no users found; skip incidents seed", "error", err)
		return
	}
	reporterID := users[0].ID

	// 使用配置中的数据，如果没有配置则使用默认值
	incidents := s.config.Incidents
	if incidents == nil {
		// 默认事件种子数据
		incidents = []IncidentSeed{
			{Title: "服务器宕机", Description: "生产环境服务器突然宕机，无法访问", Status: "resolved", Priority: "critical", Severity: "critical", IncidentNumber: "INC-001", Category: "硬件故障"},
			{Title: "网络无法访问", Description: "办公区网络无法访问外网", Status: "resolved", Priority: "high", Severity: "major", IncidentNumber: "INC-002", Category: "网络故障"},
			{Title: "应用响应慢", Description: "核心业务系统响应时间超过10秒", Status: "in_progress", Priority: "medium", Severity: "minor", IncidentNumber: "INC-003", Category: "性能问题"},
			{Title: "数据库连接失败", Description: "应用无法连接数据库", Status: "new", Priority: "high", Severity: "major", IncidentNumber: "INC-004", Category: "数据库故障"},
			{Title: "用户无法登录", Description: "部分用户报告无法登录系统", Status: "investigating", Priority: "critical", Severity: "critical", IncidentNumber: "INC-005", Category: "认证问题"},
			{Title: "邮件服务异常", Description: "企业邮件收发延迟严重", Status: "resolved", Priority: "medium", Severity: "minor", IncidentNumber: "INC-006", Category: "邮件服务"},
			{Title: "VPN连接断开", Description: "远程办公用户VPN连接不稳定", Status: "new", Priority: "high", Severity: "major", IncidentNumber: "INC-007", Category: "网络故障"},
			{Title: "存储空间不足", Description: "文件服务器存储空间即将用尽", Status: "in_progress", Priority: "medium", Severity: "minor", IncidentNumber: "INC-008", Category: "存储问题"},
		}
	} else if len(incidents) == 0 {
		s.sugar.Infow("incident business sample seed disabled by config")
		return
	}

	for _, inc := range incidents {
		_, err := s.client.Incident.Create().
			SetTitle(inc.Title).
			SetDescription(inc.Description).
			SetStatus(inc.Status).
			SetPriority(inc.Priority).
			SetSeverity(inc.Severity).
			SetIncidentNumber(inc.IncidentNumber).
			SetCategory(inc.Category).
			SetReporterID(reporterID).
			SetTenantID(t.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed incident failed", "error", err, "title", inc.Title)
		}
	}
	s.sugar.Infow("incidents seeded", "count", len(incidents))
}

// seedProblems 初始化问题种子数据
func (s *Seeder) seedProblems(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip problems seed", "error", err)
		return
	}

	// 检查是否已有问题数据
	existing, err := s.client.Problem.Query().Count(ctx)
	if err != nil {
		s.sugar.Warnw("failed to query problems; skip seed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("problems already seeded", "count", existing)
		return
	}

	// 获取测试用户
	users, err := s.client.User.Query().Where(user.TenantIDEQ(t.ID)).Limit(1).All(ctx)
	if err != nil || len(users) == 0 {
		s.sugar.Warnw("no users found; skip problems seed", "error", err)
		return
	}
	creatorID := users[0].ID

	// 使用配置中的数据，如果没有配置则使用默认值
	problems := s.config.Problems
	if problems == nil {
		problems = []ProblemSeed{
			{Title: "频繁的网络中断", Description: "过去一个月内发生多次网络中断事件，影响业务连续性", Status: "analyzing", Priority: "high", Category: "网络问题", RootCause: "核心交换机老化", Impact: "全公司网络受影响"},
			{Title: "数据库性能下降", Description: "数据库查询响应时间逐渐变慢", Status: "open", Priority: "medium", Category: "数据库问题", RootCause: "缺少索引和缓存配置", Impact: "影响所有业务系统"},
			{Title: "用户认证失败", Description: "部分用户频繁出现认证失败", Status: "resolved", Priority: "high", Category: "安全问题", RootCause: "LDAP同步延迟", Impact: "用户无法正常登录"},
			{Title: "存储容量告警", Description: "存储系统频繁告警容量不足", Status: "monitoring", Priority: "medium", Category: "存储问题", RootCause: "数据增长未预估", Impact: "文件存储风险"},
			{Title: "邮件延迟严重", Description: "邮件收发延迟超过1小时", Status: "open", Priority: "medium", Category: "邮件问题", RootCause: "邮件服务器资源不足", Impact: "影响内外沟通"},
			{Title: "VPN不稳定", Description: "远程用户VPN连接经常断开", Status: "analyzing", Priority: "low", Category: "网络问题", RootCause: "带宽不足", Impact: "远程办公受影响"},
		}
	} else if len(problems) == 0 {
		s.sugar.Infow("problem business sample seed disabled by config")
		return
	}

	for _, p := range problems {
		_, err := s.client.Problem.Create().
			SetTitle(p.Title).
			SetDescription(p.Description).
			SetStatus(p.Status).
			SetPriority(p.Priority).
			SetCategory(p.Category).
			SetRootCause(p.RootCause).
			SetImpact(p.Impact).
			SetCreatedBy(creatorID).
			SetTenantID(t.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed problem failed", "error", err, "title", p.Title)
		}
	}
	s.sugar.Infow("problems seeded", "count", len(problems))
}

// seedChanges 初始化变更种子数据
func (s *Seeder) seedChanges(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip changes seed", "error", err)
		return
	}

	// 检查是否已有变更数据
	existing, err := s.client.Change.Query().Count(ctx)
	if err != nil {
		s.sugar.Warnw("failed to query changes; skip seed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("changes already seeded", "count", existing)
		return
	}

	// 获取测试用户
	users, err := s.client.User.Query().Where(user.TenantIDEQ(t.ID)).Limit(1).All(ctx)
	if err != nil || len(users) == 0 {
		s.sugar.Warnw("no users found; skip changes seed", "error", err)
		return
	}
	creatorID := users[0].ID

	// 使用配置中的数据，如果没有配置则使用默认值
	changes := s.config.Changes
	if changes == nil {
		changes = []ChangeSeed{
			{Title: "数据库版本升级", Description: "将MySQL 5.7升级到8.0版本", Type: "normal", Status: "approved", Priority: "high", ImpactScope: "全局", RiskLevel: "medium", Justification: "提升性能和安全性"},
			{Title: "应用服务器扩容", Description: "增加2台应用服务器以应对流量高峰", Type: "normal", Status: "implemented", Priority: "medium", ImpactScope: "局部", RiskLevel: "low", Justification: "提升系统可用性"},
			{Title: "网络安全策略调整", Description: "更新防火墙规则，开放新的API端口", Type: "normal", Status: "review", Priority: "high", ImpactScope: "全局", RiskLevel: "high", Justification: "支持新业务上线"},
			{Title: "存储系统迁移", Description: "将文件存储从NAS迁移到对象存储", Type: "major", Status: "draft", Priority: "medium", ImpactScope: "局部", RiskLevel: "medium", Justification: "降低存储成本"},
			{Title: "SSL证书更新", Description: "更新即将过期的SSL证书", Type: "emergency", Status: "scheduled", Priority: "low", ImpactScope: "无", RiskLevel: "low", Justification: "证书即将过期"},
			{Title: "负载均衡配置优化", Description: "优化负载均衡算法和健康检查配置", Type: "normal", Status: "cancelled", Priority: "medium", ImpactScope: "局部", RiskLevel: "low", Justification: "提升访问体验"},
		}
	} else if len(changes) == 0 {
		s.sugar.Infow("change business sample seed disabled by config")
		return
	}

	for _, c := range changes {
		_, err := s.client.Change.Create().
			SetTitle(c.Title).
			SetDescription(c.Description).
			SetType(c.Type).
			SetStatus(c.Status).
			SetPriority(c.Priority).
			SetImpactScope(c.ImpactScope).
			SetRiskLevel(c.RiskLevel).
			SetJustification(c.Justification).
			SetCreatedBy(creatorID).
			SetTenantID(t.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed change failed", "error", err, "title", c.Title)
		}
	}
	s.sugar.Infow("changes seeded", "count", len(changes))
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

// seedKnownErrors 初始化已知错误种子数据
func (s *Seeder) seedKnownErrors(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip known errors seed", "error", err)
		return
	}

	// 检查是否已有已知错误
	existing, err := s.client.KnownError.Query().Count(ctx)
	if err != nil {
		s.sugar.Warnw("failed to query known errors; skip seed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("known errors already seeded", "count", existing)
		return
	}

	// 获取测试用户
	users, err := s.client.User.Query().Where(user.TenantIDEQ(t.ID)).Limit(1).All(ctx)
	if err != nil || len(users) == 0 {
		s.sugar.Warnw("no users found; skip known errors seed", "error", err)
		return
	}
	creatorID := users[0].ID

	// 使用配置中的数据，如果没有配置则使用默认值
	knownErrors := s.config.KnownErrors
	if len(knownErrors) == 0 {
		knownErrors = []KnownErrorSeed{
			{
				Title:            "数据库连接池耗尽",
				Description:      "应用程序无法获取数据库连接，导致服务不可用",
				Symptoms:         "错误日志显示 'too many connections'；应用程序响应缓慢或超时",
				RootCause:        "数据库连接池配置过小，且存在连接泄漏",
				Workaround:       "1. 重启应用服务释放连接\n2. 临时扩大连接池大小\n3. kill掉泄漏连接的进程",
				Resolution:       "1. 修复代码中的连接泄漏问题\n2. 调整连接池参数至合理值\n3. 添加连接池监控告警",
				Status:           "resolved",
				Category:         "数据库",
				Severity:         "high",
				AffectedProducts: []string{"核心业务系统"},
				AffectedCIs:      []string{"数据库服务器", "应用服务器"},
				Keywords:         []string{"数据库", "连接池", "too many connections"},
			},
			{
				Title:            "磁盘空间不足",
				Description:      "服务器磁盘空间不足导致系统无法正常工作",
				Symptoms:         "写入文件失败；日志报错 'no space left on device'；应用程序崩溃",
				RootCause:        "日志文件未轮转；临时文件未清理；数据文件增长过快",
				Workaround:       "1. 清理临时目录\n2. 删除旧日志\n3. 扩容磁盘",
				Resolution:       "1. 配置日志轮转策略\n2. 设置磁盘空间告警\n3. 定期执行清理任务",
				Status:           "active",
				Category:         "系统",
				Severity:         "critical",
				AffectedProducts: []string{"所有系统"},
				AffectedCIs:      []string{"所有服务器"},
				Keywords:         []string{"磁盘", "空间不足", "no space"},
			},
			{
				Title:            "网络延迟增加",
				Description:      "网络延迟突然增加，影响业务响应时间",
				Symptoms:         "用户反馈系统响应慢；监控显示延迟指标上升",
				RootCause:        "网络设备负载过高；带宽被占用；路由异常",
				Workaround:       "1. 切换到备份网络链路\n2. 限制非关键业务带宽",
				Resolution:       "1. 排查网络设备CPU/内存使用率\n2. 检查是否有异常流量\n3. 优化路由配置",
				Status:           "active",
				Category:         "网络",
				Severity:         "medium",
				AffectedProducts: []string{"所有网络相关服务"},
				AffectedCIs:      []string{"路由器", "交换机", "防火墙"},
				Keywords:         []string{"网络", "延迟", "丢包"},
			},
			{
				Title:            "内存泄漏",
				Description:      "应用程序内存使用持续增长，最终导致OOM",
				Symptoms:         "内存使用率持续上升；进程被OOM killer终止",
				RootCause:        "代码中存在内存泄漏；缓存对象未正确释放",
				Workaround:       "1. 重启应用服务释放内存\n2. 限制单进程最大内存",
				Resolution:       "1. 使用内存分析工具定位泄漏点\n2. 修复代码中的泄漏问题\n3. 添加内存监控告警",
				Status:           "resolved",
				Category:         "应用",
				Severity:         "high",
				AffectedProducts: []string{"Java应用", "Node.js应用"},
				AffectedCIs:      []string{"应用服务器"},
				Keywords:         []string{"内存", "泄漏", "OOM"},
			},
			{
				Title:            "SSL证书过期",
				Description:      "SSL证书过期导致HTTPS服务无法访问",
				Symptoms:         "浏览器显示证书无效警告；HTTPS请求失败",
				RootCause:        "证书更新流程未执行；证书有效期监控缺失",
				Workaround:       "使用备份证书或临时HTTP访问",
				Resolution:       "1. 立即更新SSL证书\n2. 建立证书有效期监控和预警机制\n3. 自动化证书更新流程",
				Status:           "resolved",
				Category:         "安全",
				Severity:         "critical",
				AffectedProducts: []string{"所有HTTPS服务"},
				AffectedCIs:      []string{"Web服务器", "负载均衡器"},
				Keywords:         []string{"SSL", "证书", "过期", "HTTPS"},
			},
		}
	}

	for _, ke := range knownErrors {
		_, err := s.client.KnownError.Create().
			SetTitle(ke.Title).
			SetDescription(ke.Description).
			SetSymptoms(ke.Symptoms).
			SetRootCause(ke.RootCause).
			SetWorkaround(ke.Workaround).
			SetResolution(ke.Resolution).
			SetStatus(ke.Status).
			SetCategory(ke.Category).
			SetSeverity(ke.Severity).
			SetAffectedProducts(ke.AffectedProducts).
			SetAffectedCis(ke.AffectedCIs).
			SetKeywords(ke.Keywords).
			SetOccurrenceCount(0).
			SetCreatedBy(creatorID).
			SetTenantID(t.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed known error failed", "error", err, "title", ke.Title)
		}
	}
	s.sugar.Infow("known errors seeded", "count", len(knownErrors))
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

// seedKnowledgeArticles 初始化知识库文章种子数据
func (s *Seeder) seedKnowledgeArticles(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip knowledge articles seed", "error", err)
		return
	}

	// 检查是否已有知识库文章
	existing, err := s.client.KnowledgeArticle.Query().Count(ctx)
	if err != nil {
		s.sugar.Warnw("failed to query knowledge articles; skip seed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("knowledge articles already seeded", "count", existing)
		return
	}

	// Product defaults intentionally do not create fictional knowledge articles.
	// Explicit empty arrays in seed config mean "leave article content to the
	// customer"; nil is kept as a compatibility path for legacy embedded seeds.
	articles := s.config.KnowledgeArticles
	if articles != nil && len(articles) == 0 {
		s.sugar.Infow("knowledge article business sample seed disabled by config")
		return
	}

	// 获取测试用户
	users, err := s.client.User.Query().Where(user.TenantIDEQ(t.ID)).Limit(1).All(ctx)
	if err != nil || len(users) == 0 {
		s.sugar.Warnw("no users found; skip knowledge articles seed", "error", err)
		return
	}
	authorID := users[0].ID

	// 首先创建知识库分类
	categories := []struct {
		Name        string
		Description string
	}{
		{"云资源", "云服务器、存储、网络等资源申请和使用指南"},
		{"账号权限", "账号申请、权限配置相关指南"},
		{"网络问题", "网络连接、VPN配置问题排查"},
		{"数据库", "数据库使用和问题处理指南"},
		{"安全", "安全配置和合规相关指南"},
		{"常见问题", "常见问题解答FAQ"},
	}

	var categoryIDs []int
	for _, cat := range categories {
		c, err := s.client.TicketCategory.Create().
			SetName(cat.Name).
			SetDescription(cat.Description).
			SetTenantID(t.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed knowledge category failed", "error", err, "name", cat.Name)
			continue
		}
		categoryIDs = append(categoryIDs, c.ID)
	}

	// 如果没有分类，创建默认分类
	if len(categoryIDs) == 0 {
		defaultCat, err := s.client.TicketCategory.Create().
			SetName("默认分类").
			SetDescription("默认知识库分类").
			SetTenantID(t.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err == nil {
			categoryIDs = append(categoryIDs, defaultCat.ID)
		}
	}

	if articles == nil {
		// Legacy embedded examples are only used when no product/default seed
		// config is loaded. Current product defaults set this slice to [].
		articles = []KnowledgeArticleSeed{
			{
				Title:       "如何申请云服务器 ECS",
				Content:     "# 云服务器申请指南\n\n## 申请流程\n\n1. 登录IT服务门户\n2. 选择服务目录->云资源申请->云服务器\n3. 填写申请表单\n   - 选择配置(CPU/内存)\n   - 选择地域和可用区\n   - 设置登录密码\n4. 提交申请\n5. 等待审批(通常1-2个工作日)\n\n## 常见配置推荐\n\n- **开发测试环境**: 2核4G\n- **生产环境**: 4核8G起\n\n如有疑问请联系IT支持。",
				Category:    "云资源",
				IsPublished: true,
				ViewCount:   1250,
			},
			{
				Title:       "VPN连接配置指南",
				Content:     "# VPN连接配置指南\n\n## Windows系统配置\n\n1. 下载VPN客户端\n2. 安装并运行客户端\n3. 配置连接信息:\n   - 服务器地址: vpn.company.com\n   - 用户名: 域账号\n   - 密码: 域密码\n4. 点击连接\n\n## 常见问题\n\n- **无法连接**: 检查网络环境，确认VPN端口(443)未被阻断\n- **连接后无法访问内网**: 尝试刷新DNS缓存 `ipconfig /flushdns`",
				Category:    "网络问题",
				IsPublished: true,
				ViewCount:   980,
			},
			{
				Title:       "账号权限申请流程",
				Content:     "# 账号权限申请指南\n\n## 需要的权限\n\n根据岗位职责申请相应权限:\n\n| 角色 | 系统权限 | 申请流程 |\n|------|----------|----------|\n| 普通员工 | 基础访问 | 主管审批 |\n| 开发者 | 代码仓库访问 | 技术主管审批 |\n| 管理员 | 系统管理 | IT经理审批 |\n\n## 申请入口\n\nIT服务门户 - 服务目录 - 账号与权限申请",
				Category:    "账号权限",
				IsPublished: true,
				ViewCount:   856,
			},
			{
				Title:       "数据库备份与恢复",
				Content:     "# 数据库备份恢复指南\n\n## 自动备份\n\n系统每天凌晨2点自动执行全量备份，保留30天。\n\n## 手动备份\n\n```bash\nmysqldump -u root -p database_name > backup.sql\n```\n\n## 恢复数据\n\n```bash\nmysql -u root -p database_name < backup.sql\n```\n\n**注意**: 恢复操作需要DBA权限，请提交工单联系数据库管理员。",
				Category:    "数据库",
				IsPublished: true,
				ViewCount:   742,
			},
			{
				Title:       "IT服务请求常见问题解答",
				Content:     "# IT服务常见问题FAQ\n\n## 如何提交IT工单?\n\n1. 登录IT服务门户\n2. 点击\"提交工单\"\n3. 选择工单类型\n4. 填写详细信息\n5. 提交\n\n## 工单处理时限多久?\n\n- 紧急: 4小时内响应\n- 高优先级: 8小时内响应\n- 中优先级: 24小时内响应\n- 低优先级: 72小时内响应\n\n## 如何查询工单进度?\n\n在\"我的工单\"页面查看所有提交的工单及状态。",
				Category:    "常见问题",
				IsPublished: true,
				ViewCount:   3560,
			},
			{
				Title:       "安全组配置最佳实践",
				Content:     "# 安全组配置指南\n\n## 基本原则\n\n1. **最小权限**: 只开放必要的端口\n2. **分层控制**: 不同层级使用不同的安全组\n3. **禁止0.0.0.0/0**: 避免直接对公网开放敏感服务\n\n## 常用端口配置\n\n| 服务 | 端口 | 协议 | 建议 |\n|------|------|------|------|\n| SSH | 22 | TCP | 仅内网访问 |\n| HTTP | 80 | TCP | 可公网访问 |\n| HTTPS | 443 | TCP | 可公网访问 |\n| MySQL | 3306 | TCP | 仅内网访问 |\n\n## 审批要求\n\n安全组规则变更需要安全团队审批。",
				Category:    "安全",
				IsPublished: true,
				ViewCount:   625,
			},
		}
	}

	// Remove the category ID mapping since we now use category name directly
	for _, article := range articles {
		_, err := s.client.KnowledgeArticle.Create().
			SetTitle(article.Title).
			SetContent(article.Content).
			SetIsPublished(article.IsPublished).
			SetViewCount(article.ViewCount).
			SetAuthorID(authorID).
			SetCategory(article.Category).
			SetTenantID(t.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed knowledge article failed", "error", err, "title", article.Title)
		}
	}
	s.sugar.Infow("knowledge articles seeded", "count", len(articles))
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
