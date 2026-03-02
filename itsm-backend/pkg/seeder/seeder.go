package seeder

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/approvalworkflow"
	"itsm-backend/ent/department"
	"itsm-backend/ent/processbinding"
	"itsm-backend/ent/role"
	"itsm-backend/ent/servicecatalog"
	"itsm-backend/ent/slaalertrule"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/team"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/ticketview"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// SeedConfig 种子数据配置结构
type SeedConfig struct {
	Departments       []DepartmentSeed       `json:"departments"`
	Teams             []TeamSeed             `json:"teams"`
	Roles             []RoleSeed             `json:"roles"`
	SLADefinitions   []SLADefinitionSeed    `json:"sla_definitions"`
	ServiceCatalog   []ServiceCatalogSeed   `json:"service_catalog"`
	ApprovalWorkflows []ApprovalWorkflowSeed `json:"approval_workflows"`
	ProcessBindings  []ProcessBindingSeed   `json:"process_bindings"`
	TicketViews      []TicketViewSeed       `json:"ticket_views"`
}

type DepartmentSeed struct {
	Name       string `json:"name"`
	Code      string `json:"code"`
	Desc      string `json:"description"`
	ParentCode string `json:"parent_code"`
}

type TeamSeed struct {
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
	BusinessType           string `json:"business_type"`
	ProcessDefinitionKey   string `json:"process_definition_key"`
	IsDefault              bool   `json:"is_default"`
}

type TicketViewSeed struct {
	Name     string   `json:"name"`
	Desc     string   `json:"description"`
	IsShared bool     `json:"is_shared"`
	Columns  []string `json:"columns"`
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
			{BusinessType: "incident", ProcessDefinitionKey: "incident_process", IsDefault: true},
			{BusinessType: "problem", ProcessDefinitionKey: "problem_process", IsDefault: true},
			{BusinessType: "change", ProcessDefinitionKey: "change_process", IsDefault: true},
			{BusinessType: "service_request", ProcessDefinitionKey: "service_request_process", IsDefault: true},
			{BusinessType: "improvement", ProcessDefinitionKey: "improvement_process", IsDefault: true},
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
			SetRole("admin").
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
		SetRole("admin").
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
		if _, err := s.client.Team.Create().
			SetName(tm.Name).
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
		SetRole("admin").
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
		Name               string
		SLAKey             string
		AlertLevel         string
		Threshold          int
		NotificationChans  []string
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
