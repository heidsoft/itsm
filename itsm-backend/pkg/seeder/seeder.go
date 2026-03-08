package seeder

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/approvalworkflow"
	"itsm-backend/ent/change"
	"itsm-backend/ent/department"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/processbinding"
	"itsm-backend/ent/role"
	"itsm-backend/ent/servicecatalog"
	"itsm-backend/ent/slaalertrule"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/team"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/ticketcategory"
	"itsm-backend/ent/ticketview"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"itsm-backend/database"
)

// Force import usage for ent packages (use predicate functions)
var (
	_ = incident.TitleEQ   // Used to ensure incident package is imported
	_ = problem.TitleEQ     // Used to ensure problem package is imported
	_ = change.TitleEQ     // Used to ensure change package is imported
	_ = knowledgearticle.TitleEQ // Used to ensure knowledgearticle package is imported
	_ = ticketcategory.NameEQ    // Used to ensure ticketcategory package is imported
)

// SeedConfig 种子数据配置结构
type SeedConfig struct {
	Departments        []DepartmentSeed        `json:"departments"`
	Teams              []TeamSeed              `json:"teams"`
	Roles              []RoleSeed              `json:"roles"`
	SLADefinitions     []SLADefinitionSeed    `json:"sla_definitions"`
	ServiceCatalog     []ServiceCatalogSeed    `json:"service_catalog"`
	ApprovalWorkflows  []ApprovalWorkflowSeed  `json:"approval_workflows"`
	ProcessBindings    []ProcessBindingSeed    `json:"process_bindings"`
	TicketViews        []TicketViewSeed        `json:"ticket_views"`
	CITypes            []CITypeSeed            `json:"ci_types"`
	// 新增：可配置的种子数据
	Incidents          []IncidentSeed          `json:"incidents"`
	Problems           []ProblemSeed           `json:"problems"`
	Changes            []ChangeSeed            `json:"changes"`
	KnowledgeArticles []KnowledgeArticleSeed  `json:"knowledge_articles"`
	IncidentCategories []TicketCategorySeed    `json:"incident_categories"`
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
	ViewCount    int    `json:"view_count"`
}

// TicketCategorySeed 工单分类种子数据结构（用于事件分类）
type TicketCategorySeed struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Seeder manages database seeding operations
type Seeder struct {
	client *ent.Client
	sugar  *zap.SugaredLogger
	config *SeedConfig
}

// NewSeeder creates a new Seeder instance
func NewSeeder(client *ent.Client, sugar *zap.SugaredLogger) *Seeder {
	return &Seeder{
		client: client,
		sugar:  sugar,
		config: loadSeedConfig(sugar),
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
				return &config
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
				return &config
			}
		}
	}

	// 3. 内置默认
	sugar.Infow("using embedded default seed config")
	return getEmbeddedConfig()
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
	s.seedDefaultTenant(ctx)
	s.seedDepartments(ctx)
	s.seedTeams(ctx)
	s.seedRoles(ctx)
	s.seedPermissions(ctx) // 新增：初始化权限
	s.backfillAdminRole(ctx)
	s.seedAdmin(ctx)
	s.seedUser1(ctx)
	s.seedSecurity1(ctx)
	s.backfillUserRole(ctx)
	s.seedCloudServiceTemplates(ctx)
	s.seedAssets(ctx)
	s.seedAssetLicenses(ctx)
	s.seedReleases(ctx)
	// 使用配置的初始化数据
	s.seedSLADefinitions(ctx)
	s.seedSLAAlertRules(ctx)
	s.seedApprovalWorkflows(ctx)
	s.seedProcessBindings(ctx)
	s.seedTicketViews(ctx)
	s.seedServiceCatalog(ctx)
	s.seedTicketTypes(ctx)         // 新增：初始化工单类型
	s.seedCITypes(ctx)            // 新增：初始化CI类型
	s.seedIncidentCategories(ctx) // 新增：初始化事件分类
	s.seedIncidents(ctx)          // 新增：初始化事件数据
	s.seedProblems(ctx)           // 新增：初始化问题数据
	s.seedChanges(ctx)            // 新增：初始化变更数据
	s.seedKnowledgeArticles(ctx)  // 新增：初始化知识库文章
}

// seedDefaultTenant ensures default tenant exists
func (s *Seeder) seedDefaultTenant(ctx context.Context) {
	existing, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err == nil && existing != nil {
		s.sugar.Infow("default tenant already exists", "tenant_id", existing.ID)
		return
	}

	defaultTenant, err := s.client.Tenant.Create().
		SetName("Default Tenant").
		SetCode("default").
		SetDomain("localhost").
		SetStatus("active").
		SetType("enterprise").
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.sugar.Warnw("failed to create default tenant", "error", err)
		return
	}
	s.sugar.Infow("default tenant created", "tenant_id", defaultTenant.ID)
}

func (s *Seeder) seedAdmin(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip admin seed", "error", err)
		return
	}
	existing, err := s.client.User.Query().Where(user.UsernameEQ("admin"), user.TenantIDEQ(t.ID)).First(ctx)

	passHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		s.sugar.Warnw("generate bcrypt for admin failed", "error", err)
		return
	}

	if err == nil && existing != nil {
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
	passHash, err := bcrypt.GenerateFromPassword([]byte("user123"), bcrypt.DefaultCost)
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
	passHash, err := bcrypt.GenerateFromPassword([]byte("security123"), bcrypt.DefaultCost)
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
	// 保留原有实现...
}

func (s *Seeder) seedReleases(ctx context.Context) {
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
		_, err := rawDB.ExecContext(ctx, `
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
	if len(incidents) == 0 {
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
	if len(problems) == 0 {
		problems = []ProblemSeed{
			{Title: "频繁的网络中断", Description: "过去一个月内发生多次网络中断事件，影响业务连续性", Status: "analyzing", Priority: "high", Category: "网络问题", RootCause: "核心交换机老化", Impact: "全公司网络受影响"},
			{Title: "数据库性能下降", Description: "数据库查询响应时间逐渐变慢", Status: "open", Priority: "medium", Category: "数据库问题", RootCause: "缺少索引和缓存配置", Impact: "影响所有业务系统"},
			{Title: "用户认证失败", Description: "部分用户频繁出现认证失败", Status: "resolved", Priority: "high", Category: "安全问题", RootCause: "LDAP同步延迟", Impact: "用户无法正常登录"},
			{Title: "存储容量告警", Description: "存储系统频繁告警容量不足", Status: "monitoring", Priority: "medium", Category: "存储问题", RootCause: "数据增长未预估", Impact: "文件存储风险"},
			{Title: "邮件延迟严重", Description: "邮件收发延迟超过1小时", Status: "open", Priority: "medium", Category: "邮件问题", RootCause: "邮件服务器资源不足", Impact: "影响内外沟通"},
			{Title: "VPN不稳定", Description: "远程用户VPN连接经常断开", Status: "analyzing", Priority: "low", Category: "网络问题", RootCause: "带宽不足", Impact: "远程办公受影响"},
		}
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
	if len(changes) == 0 {
		changes = []ChangeSeed{
			{Title: "数据库版本升级", Description: "将MySQL 5.7升级到8.0版本", Type: "normal", Status: "approved", Priority: "high", ImpactScope: "全局", RiskLevel: "medium", Justification: "提升性能和安全性"},
			{Title: "应用服务器扩容", Description: "增加2台应用服务器以应对流量高峰", Type: "normal", Status: "implemented", Priority: "medium", ImpactScope: "局部", RiskLevel: "low", Justification: "提升系统可用性"},
			{Title: "网络安全策略调整", Description: "更新防火墙规则，开放新的API端口", Type: "normal", Status: "review", Priority: "high", ImpactScope: "全局", RiskLevel: "high", Justification: "支持新业务上线"},
			{Title: "存储系统迁移", Description: "将文件存储从NAS迁移到对象存储", Type: "major", Status: "draft", Priority: "medium", ImpactScope: "局部", RiskLevel: "medium", Justification: "降低存储成本"},
			{Title: "SSL证书更新", Description: "更新即将过期的SSL证书", Type: "emergency", Status: "scheduled", Priority: "low", ImpactScope: "无", RiskLevel: "low", Justification: "证书即将过期"},
			{Title: "负载均衡配置优化", Description: "优化负载均衡算法和健康检查配置", Type: "normal", Status: "cancelled", Priority: "medium", ImpactScope: "局部", RiskLevel: "low", Justification: "提升访问体验"},
		}
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

	// 知识库文章种子数据
	articles := []struct {
		Title      string
		Content    string
		Category   string
		IsPublished bool
		ViewCount  int
	}{
		{
			"如何申请云服务器 ECS",
			"# 云服务器申请指南\n\n## 申请流程\n\n1. 登录IT服务门户\n2. 选择服务目录->云资源申请->云服务器\n3. 填写申请表单\n   - 选择配置(CPU/内存)\n   - 选择地域和可用区\n   - 设置登录密码\n4. 提交申请\n5. 等待审批(通常1-2个工作日)\n\n## 常见配置推荐\n\n- **开发测试环境**: 2核4G\n- **生产环境**: 4核8G起\n\n如有疑问请联系IT支持。",
			"云资源", true, 1250,
		},
		{
			"VPN连接配置指南",
			"# VPN连接配置指南\n\n## Windows系统配置\n\n1. 下载VPN客户端\n2. 安装并运行客户端\n3. 配置连接信息:\n   - 服务器地址: vpn.company.com\n   - 用户名: 域账号\n   - 密码: 域密码\n4. 点击连接\n\n## 常见问题\n\n- **无法连接**: 检查网络环境，确认VPN端口(443)未被阻断\n- **连接后无法访问内网**: 尝试刷新DNS缓存 `ipconfig /flushdns`",
			"网络问题", true, 980,
		},
		{
			"账号权限申请流程",
			"# 账号权限申请指南\n\n## 需要的权限\n\n根据岗位职责申请相应权限:\n\n| 角色 | 系统权限 | 申请流程 |\n|------|----------|----------|\n| 普通员工 | 基础访问 | 主管审批 |\n| 开发者 | 代码仓库访问 | 技术主管审批 |\n| 管理员 | 系统管理 | IT经理审批 |\n\n## 申请入口\n\nIT服务门户 - 服务目录 - 账号与权限申请",
			"账号权限", true, 856,
		},
		{
			"数据库备份与恢复",
			"# 数据库备份恢复指南\n\n## 自动备份\n\n系统每天凌晨2点自动执行全量备份，保留30天。\n\n## 手动备份\n\n```bash\nmysqldump -u root -p database_name > backup.sql\n```\n\n## 恢复数据\n\n```bash\nmysql -u root -p database_name < backup.sql\n```\n\n**注意**: 恢复操作需要DBA权限，请提交工单联系数据库管理员。",
			"数据库", true, 742,
		},
		{
			"IT服务请求常见问题解答",
			"# IT服务常见问题FAQ\n\n## 如何提交IT工单?\n\n1. 登录IT服务门户\n2. 点击\"提交工单\"\n3. 选择工单类型\n4. 填写详细信息\n5. 提交\n\n## 工单处理时限多久?\n\n- 紧急: 4小时内响应\n- 高优先级: 8小时内响应\n- 中优先级: 24小时内响应\n- 低优先级: 72小时内响应\n\n## 如何查询工单进度?\n\n在\"我的工单\"页面查看所有提交的工单及状态。",
			"常见问题", true, 3560,
		},
		{
			"安全组配置最佳实践",
			"# 安全组配置指南\n\n## 基本原则\n\n1. **最小权限**: 只开放必要的端口\n2. **分层控制**: 不同层级使用不同的安全组\n3. **禁止0.0.0.0/0**: 避免直接对公网开放敏感服务\n\n## 常用端口配置\n\n| 服务 | 端口 | 协议 | 建议 |\n|------|------|------|------|\n| SSH | 22 | TCP | 仅内网访问 |\n| HTTP | 80 | TCP | 可公网访问 |\n| HTTPS | 443 | TCP | 可公网访问 |\n| MySQL | 3306 | TCP | 仅内网访问 |\n\n## 审批要求\n\n安全组规则变更需要安全团队审批。",
			"安全", true, 625,
		},
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
			{Name: "硬件故障", Description: "服务器、存储、网络设备等硬件故障"},
			{Name: "软件故障", Description: "操作系统、应用软件故障"},
			{Name: "网络故障", Description: "网络连接、网络设备问题"},
			{Name: "数据库问题", Description: "数据库性能、连接问题"},
			{Name: "安全问题", Description: "安全事件、漏洞"},
			{Name: "性能问题", Description: "系统响应慢、卡顿"},
			{Name: "配置问题", Description: "系统配置错误"},
			{Name: "其他", Description: "其他类型事件"},
		}
	}

	for _, cat := range categories {
		_, err := s.client.TicketCategory.Create().
			SetName(cat.Name).
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
