package seeder

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/approvalworkflow"
	"itsm-backend/ent/asset"
	"itsm-backend/ent/assetlicense"
	"itsm-backend/ent/cloudservice"
	"itsm-backend/ent/department"
	"itsm-backend/ent/processbinding"
	"itsm-backend/ent/release"
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

// Seeder manages database seeding operations
type Seeder struct {
	client *ent.Client
	sugar  *zap.SugaredLogger
}

// NewSeeder creates a new Seeder instance
func NewSeeder(client *ent.Client, sugar *zap.SugaredLogger) *Seeder {
	return &Seeder{
		client: client,
		sugar:  sugar,
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
	s.seedAdmin(ctx) // Seed admin user
	s.seedUser1(ctx)
	s.seedSecurity1(ctx)
	s.backfillUserRole(ctx)
	s.seedCloudServiceTemplates(ctx)
	s.seedAssets(ctx)
	s.seedAssetLicenses(ctx)
	s.seedReleases(ctx)
	// 新增初始化
	s.seedSLADefinitions(ctx)
	s.seedSLAAlertRules(ctx)
	s.seedApprovalWorkflows(ctx)
	s.seedProcessBindings(ctx)
	s.seedTicketViews(ctx)
	s.seedServiceCatalog(ctx)
}

// seedDefaultTenant ensures default tenant exists
func (s *Seeder) seedDefaultTenant(ctx context.Context) {
	// 检查是否已有 default 租户
	existing, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err == nil && existing != nil {
		s.sugar.Infow("default tenant already exists", "tenant_id", existing.ID)
		return
	}

	// 创建 default 租户
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

// seedAdmin seeds default admin user with password admin123
func (s *Seeder) seedAdmin(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip admin seed", "error", err)
		return
	}
	// 检查是否已存在 admin
	existing, err := s.client.User.Query().Where(user.UsernameEQ("admin"), user.TenantIDEQ(t.ID)).First(ctx)

	// 创建或更新 admin，密码: admin123
	passHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		s.sugar.Warnw("generate bcrypt for admin failed", "error", err)
		return
	}

	if err == nil && existing != nil {
		// 存在则更新密码和角色
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

	// 不存在则创建
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

// seedDepartments seeds enterprise departments (complex structure)
func (s *Seeder) seedDepartments(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip departments seed", "error", err)
		return
	}

	// 检查是否已有部门
	existing, err := s.client.Department.Query().Where(department.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing departments failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("departments already seeded")
		return
	}

	// 创建复杂的企业部门架构
	departments := []struct {
		Name        string
		Code        string
		Desc        string
		ParentCode  string
		ManagerID   int
	}{
		// 信息技术部
		{"信息技术部", "IT", "IT整体管理", "", 0},
		{"IT基础架构", "IT-INFRA", "基础设施运维", "IT", 0},
		{"IT应用服务", "IT-APP", "应用系统运维", "IT", 0},
		{"IT安全", "IT-SEC", "信息安全管理", "IT", 0},
		{"IT项目管理", "IT-PMO", "IT项目管理办公室", "IT", 0},
		// 运营管理
		{"运营管理部", "OPS", "IT运营管理", "", 0},
		{"服务台", "OPS-SD", "一线服务支持", "OPS", 0},
		{"运维中心", "OPS-NOC", "7x24运维监控", "OPS", 0},
		{"客户服务", "OPS-CS", "客户服务体验", "OPS", 0},
		// 业务部门
		{"研发部", "RD", "产品研发", "", 0},
		{"测试部", "QA", "质量保证", "", 0},
		// 职能部门
		{"人力资源部", "HR", "人力资源管理", "", 0},
		{"财务部", "FIN", "财务管理", "", 0},
		{"行政部", "ADMIN", "行政管理", "", 0},
	}

	deptMap := make(map[string]int)
	for _, d := range departments {
		entity, err := s.client.Department.Create().
			SetName(d.Name).
			SetCode(d.Code).
			SetDescription(d.Desc).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed department failed", "error", err, "name", d.Name)
			continue
		}
		deptMap[d.Code] = entity.ID
	}
	s.sugar.Infow("enterprise departments seeded", "count", len(departments))
	_ = deptMap
}

// seedTeams seeds default teams
func (s *Seeder) seedTeams(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip teams seed", "error", err)
		return
	}

	// 检查是否已有团队
	existing, err := s.client.Team.Query().Where(team.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing teams failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("teams already seeded")
		return
	}

	// 创建复杂的企业团队结构
	teams := []struct {
		Name   string
		Desc   string
		Status string
	}{
		// IT服务支持团队
		{"服务台-L1", "一线服务支持，处理简单问题", "active"},
		{"服务台-L2", "二线技术支持，处理复杂问题", "active"},
		{"服务台-L3", "三线技术专家，处理疑难问题", "active"},
		// 基础设施团队
		{"服务器运维", "服务器运维管理", "active"},
		{"网络运维", "网络设备运维", "active"},
		{"数据库运维", "数据库运维管理", "active"},
		{"云平台运维", "云计算平台运维", "active"},
		// 应用服务团队
		{"ERP支持", "ERP系统支持", "active"},
		{"CRM支持", "CRM系统支持", "active"},
		{"OA支持", "OA办公系统支持", "active"},
		// 安全团队
		{"安全运营", "安全监控与响应", "active"},
		{"安全合规", "安全合规管理", "active"},
		// 研发团队
		{"后端开发", "后端开发团队", "active"},
		{"前端开发", "前端开发团队", "active"},
		{"移动开发", "移动端开发团队", "active"},
		{"测试团队", "测试与质量保证", "active"},
		// 客户服务
		{"客户成功", "客户成功管理", "active"},
		{"技术支持", "客户服务技术支持", "active"},
	}

	for _, tm := range teams {
		if _, err := s.client.Team.Create().
			SetName(tm.Name).
			SetDescription(tm.Desc).
			SetStatus(tm.Status).
			SetTenantID(t.ID).
			Save(ctx); err != nil {
			s.sugar.Warnw("seed team failed", "error", err, "name", tm.Name)
		}
	}
	s.sugar.Infow("enterprise teams seeded", "count", len(teams))
}

// seedRoles seeds default roles
func (s *Seeder) seedRoles(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip roles seed", "error", err)
		return
	}

	// 检查是否已有角色
	existing, err := s.client.Role.Query().Where(role.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing roles failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("roles already seeded")
		return
	}

	// 创建企业级复杂角色体系
	roles := []struct {
		Name string
		Code string
		Desc string
	}{
		// 高层管理
		{"IT总监", "it_director", "IT部门总监，全面负责IT管理"},
		{"运维总监", "ops_director", "运维部门总监"},
		// 系统管理
		{"系统管理员", "sysadmin", "系统管理员，拥有全部管理权限"},
		{"安全管理员", "security_admin", "安全管理角色"},
		{"审计管理员", "audit_admin", "审计管理角色"},
		// 运维角色
		{"运维经理", "ops_manager", "运维团队经理"},
		{"运维工程师", "ops_engineer", "运维工程师"},
		{"DBA工程师", "dba", "数据库管理员"},
		{"网络安全工程师", "network_eng", "网络工程师"},
		// 服务台
		{"服务台主管", "sd_manager", "服务台主管"},
		{"一线工程师", "l1_support", "一线支持工程师"},
		{"二线工程师", "l2_support", "二线支持工程师"},
		{"三线专家", "l3_expert", "三线技术专家"},
		// 研发角色
		{"研发经理", "rd_manager", "研发团队经理"},
		{"开发工程师", "developer", "开发工程师"},
		{"测试工程师", "qa_engineer", "测试工程师"},
		// 业务角色
		{"部门经理", "dept_manager", "部门经理"},
		{"团队主管", "team_lead", "团队主管"},
		// 普通用户
		{"普通用户", "end_user", "普通终端用户"},
		{"访客", "guest", "访客用户，权限受限"},
	}

	for _, r := range roles {
		if _, err := s.client.Role.Create().
			SetName(r.Name).
			SetCode(r.Code).
			SetDescription(r.Desc).
			SetTenantID(t.ID).
			Save(ctx); err != nil {
			s.sugar.Warnw("seed role failed", "error", err, "name", r.Name)
		}
	}
	s.sugar.Infow("enterprise roles seeded", "count", len(roles))
}

// backfillAdminRole ensures default admin account has admin role
func (s *Seeder) backfillAdminRole(ctx context.Context) {
	if _, err := s.client.User.Update().
		Where(user.UsernameEQ("admin")).
		SetRole("admin").
		Save(ctx); err != nil {
		s.sugar.Warnw("admin role backfill failed (non-fatal)", "error", err)
	} else {
		s.sugar.Infow("admin role backfilled to admin")
	}
}

// seedUser1 ensuring a default normal user exists
func (s *Seeder) seedUser1(ctx context.Context) {
	// 查找默认租户
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip user1 seed", "error", err)
		return
	}
	// 是否已存在 user1
	_, err = s.client.User.Query().Where(user.UsernameEQ("user1"), user.TenantIDEQ(t.ID)).First(ctx)
	if err == nil {
		s.sugar.Infow("seed user1 already exists")
		return
	}
	// 创建 user1，密码: user123
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

// seedSecurity1 ensuring a default security user exists
func (s *Seeder) seedSecurity1(ctx context.Context) {
	// 查找默认租户
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip security1 seed", "error", err)
		return
	}
	// 是否已存在 security1
	_, err = s.client.User.Query().Where(user.UsernameEQ("security1"), user.TenantIDEQ(t.ID)).First(ctx)
	if err == nil {
		s.sugar.Infow("seed security1 already exists")
		return
	}
	// 创建 security1，密码: security123
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
		SetDepartment("安全部").
		SetActive(true).
		SetTenantID(t.ID).
		Save(ctx); err != nil {
		s.sugar.Warnw("seed security1 failed", "error", err)
	} else {
		s.sugar.Infow("seed security1 created", "username", "security1")
	}
}

// backfillUserRole migrates old role "user" to "end_user"
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

type cloudServiceTemplate struct {
	Key              string                 `json:"key"`
	ParentKey        string                 `json:"parent_key"`
	Provider         string                 `json:"provider"`
	Category         string                 `json:"category"`
	ServiceCode      string                 `json:"service_code"`
	ServiceName      string                 `json:"service_name"`
	ResourceTypeCode string                 `json:"resource_type_code"`
	ResourceTypeName string                 `json:"resource_type_name"`
	APIVersion       string                 `json:"api_version"`
	AttributeSchema  map[string]interface{} `json:"attribute_schema"`
	IsSystem         bool                   `json:"is_system"`
}

func (s *Seeder) seedCloudServiceTemplates(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip cloud service templates", "error", err)
		return
	}

	filePath := filepath.Join("config", "cmdb", "cloud_service_templates.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		s.sugar.Warnw("read cloud service templates failed", "error", err, "path", filePath)
		return
	}

	var templates []cloudServiceTemplate
	if err := json.Unmarshal(data, &templates); err != nil {
		s.sugar.Warnw("parse cloud service templates failed", "error", err)
		return
	}

	existing, err := s.client.CloudService.Query().
		Where(cloudservice.TenantID(t.ID)).
		All(ctx)
	if err != nil {
		s.sugar.Warnw("load existing cloud services failed", "error", err)
		return
	}

	existingIndex := make(map[string]*ent.CloudService, len(existing))
	for _, item := range existing {
		key := item.Provider + "|" + item.ServiceCode + "|" + item.ResourceTypeCode
		existingIndex[key] = item
	}

	keyToID := make(map[string]int, len(templates))
	for _, tpl := range templates {
		matchKey := tpl.Provider + "|" + tpl.ServiceCode + "|" + tpl.ResourceTypeCode
		if found, ok := existingIndex[matchKey]; ok {
			keyToID[tpl.Key] = found.ID
			continue
		}
		parentID := 0
		if tpl.ParentKey != "" {
			if id, ok := keyToID[tpl.ParentKey]; ok {
				parentID = id
			}
		}
		create := s.client.CloudService.Create().
			SetProvider(tpl.Provider).
			SetCategory(tpl.Category).
			SetServiceCode(tpl.ServiceCode).
			SetServiceName(tpl.ServiceName).
			SetResourceTypeCode(tpl.ResourceTypeCode).
			SetResourceTypeName(tpl.ResourceTypeName).
			SetAPIVersion(tpl.APIVersion).
			SetAttributeSchema(tpl.AttributeSchema).
			SetIsSystem(tpl.IsSystem).
			SetIsActive(true).
			SetTenantID(t.ID)
		if parentID > 0 {
			create.SetParentID(parentID)
		}
		entity, err := create.Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed cloud service failed", "error", err, "service_code", tpl.ServiceCode, "resource_type_code", tpl.ResourceTypeCode)
			continue
		}
		keyToID[tpl.Key] = entity.ID
	}
}

// seedAssets seeds default assets
func (s *Seeder) seedAssets(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip assets seed", "error", err)
		return
	}

	// 检查是否已有资产
	existing, err := s.client.Asset.Query().Where(asset.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing assets failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("assets already seeded")
		return
	}

	// 创建资产
	assets := []struct {
		Number       string
		Name         string
		Description  string
		Type         string
		Status       string
		Category     string
		Subcategory  string
		Serial       string
		Model        string
		Manufacturer string
		Vendor       string
		Location     string
		Department   string
		Tags         []string
	}{
		{"AST-001", "MacBook Pro 14寸", "开发工程师笔记本", "hardware", "in-use", "笔记本", "MacBook", "MBP-2023-001", "MacBook Pro 14 M2", "Apple", "Apple Store", "A座302", "研发部", []string{"mac", "laptop"}},
		{"AST-002", "Dell XPS 15", "测试工程师笔记本", "hardware", "in-use", "笔记本", "Dell", "DELL-2023-002", "XPS 15 9530", "Dell", "Dell", "B座201", "测试部", []string{"windows", "laptop"}},
		{"AST-003", "ThinkPad X1 Carbon", "运维工程师笔记本", "hardware", "in-use", "笔记本", "ThinkPad", "TP-2023-003", "ThinkPad X1 Carbon Gen 10", "Lenovo", "Lenovo", "C座101", "运维部", []string{"linux", "laptop"}},
		{"AST-004", "Dell PowerEdge R740", "生产环境数据库服务器", "hardware", "in-use", "服务器", "机架式", "SRV-DB-001", "PowerEdge R740", "Dell", "Dell", "机房A-1", "运维部", []string{"server", "production"}},
		{"AST-005", "Dell PowerEdge R640", "生产环境应用服务器", "hardware", "in-use", "服务器", "机架式", "SRV-APP-001", "PowerEdge R640", "Dell", "Dell", "机房A-2", "运维部", []string{"server", "production"}},
		{"AST-006", "Huawei CloudEngine 6800", "核心交换机", "hardware", "in-use", "网络设备", "交换机", "SW-CORE-001", "CloudEngine 6800", "Huawei", "Huawei", "机房网络区", "运维部", []string{"network", "switch"}},
		{"AST-007", "Visual Studio 2022", "开发工具许可证", "software", "available", "开发工具", "IDE", "VS-2022-001", "Visual Studio Professional 2022", "Microsoft", "Microsoft", "云端", "研发部", []string{"ide", "development"}},
		{"AST-008", "JetBrains All Products", "开发工具订阅", "software", "available", "开发工具", "IDE", "JB-ALL-001", "All Products Pack", "JetBrains", "JetBrains", "云端", "研发部", []string{"ide", "development"}},
		{"AST-009", "MySQL Enterprise", "数据库许可证", "license", "available", "数据库", "MySQL", "LIC-MYSQL-001", "MySQL Enterprise Edition", "Oracle", "Oracle", "云端", "运维部", []string{"database", "mysql"}},
		{"AST-010", "阿里云ECS实例", "云服务器", "cloud", "in-use", "云服务", "ECS", "CLOUD-ECS-001", "ecs.g7.2xlarge", "Aliyun", "Aliyun", "阿里云华北1", "运维部", []string{"cloud", "ecs"}},
	}

	for _, a := range assets {
		if _, err := s.client.Asset.Create().
			SetAssetNumber(a.Number).
			SetName(a.Name).
			SetDescription(a.Description).
			SetType(a.Type).
			SetStatus(a.Status).
			SetCategory(a.Category).
			SetSubcategory(a.Subcategory).
			SetSerialNumber(a.Serial).
			SetModel(a.Model).
			SetManufacturer(a.Manufacturer).
			SetVendor(a.Vendor).
			SetLocation(a.Location).
			SetDepartment(a.Department).
			SetTags(a.Tags).
			SetTenantID(t.ID).
			Save(ctx); err != nil {
			s.sugar.Warnw("seed asset failed", "error", err, "name", a.Name)
		}
	}
	s.sugar.Infow("assets seeded")
}

// seedAssetLicenses seeds default asset licenses
func (s *Seeder) seedAssetLicenses(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip licenses seed", "error", err)
		return
	}

	// 检查是否已有许可证
	existing, err := s.client.AssetLicense.Query().Where(assetlicense.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing licenses failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("licenses already seeded")
		return
	}

	// 创建许可证
	licenses := []struct {
		Name        string
		Description string
		Vendor      string
		Type        string
		Total       int
		Used        int
		Purchase    string
		Expiry      string
		Cost        float64
		Status      string
		Tags        []string
	}{
		{"Visual Studio Professional 2022", "Visual Studio 专业版 2022 许可证", "Microsoft", "perpetual", 10, 5, "2023-01-01", "2025-01-01", 11999.00, "active", []string{"ide", "microsoft"}},
		{"JetBrains All Products Pack", "JetBrains 全家桶年度订阅", "JetBrains", "subscription", 20, 8, "2023-01-01", "2024-01-01", 649.00, "active", []string{"ide", "jetbrains"}},
		{"MySQL Enterprise Edition", "MySQL 企业版许可证", "Oracle", "perpetual", 5, 2, "2022-06-01", "2027-06-01", 50000.00, "active", []string{"database", "mysql"}},
		{"Adobe Creative Cloud", "Adobe 创意云团队版", "Adobe", "subscription", 15, 10, "2023-03-01", "2024-03-01", 799.00, "active", []string{"adobe", "creative"}},
		{"Windows Server 2022", "Windows Server 2022 数据中心版", "Microsoft", "perpetual", 10, 4, "2022-10-01", "2032-10-01", 6000.00, "active", []string{"microsoft", "server"}},
		{"Red Hat Enterprise Linux", "RHEL 企业版订阅", "Red Hat", "subscription", 25, 12, "2023-01-01", "2024-01-01", 799.00, "active", []string{"linux", "redhat"}},
	}

	for _, l := range licenses {
		if _, err := s.client.AssetLicense.Create().
			SetName(l.Name).
			SetDescription(l.Description).
			SetVendor(l.Vendor).
			SetLicenseType(l.Type).
			SetTotalQuantity(l.Total).
			SetUsedQuantity(l.Used).
			SetPurchaseDate(l.Purchase).
			SetExpiryDate(l.Expiry).
			SetPurchasePrice(l.Cost).
			SetStatus(l.Status).
			SetTags(l.Tags).
			SetTenantID(t.ID).
			Save(ctx); err != nil {
			s.sugar.Warnw("seed license failed", "error", err, "name", l.Name)
		}
	}
	s.sugar.Infow("licenses seeded")
}

// seedReleases seeds default releases
func (s *Seeder) seedReleases(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip releases seed", "error", err)
		return
	}

	// 获取创建用户 ID
	creator, err := s.client.User.Query().Where(user.UsernameEQ("admin"), user.TenantIDEQ(t.ID)).First(ctx)
	if err != nil {
		s.sugar.Warnw("admin user not found; skip releases seed", "error", err)
		return
	}

	// 检查是否已有发布
	existing, err := s.client.Release.Query().Where(release.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing releases failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("releases already seeded")
		return
	}

	now := time.Now()
	// 创建发布
	releases := []struct {
		Number     string
		Title      string
		Desc       string
		Type       string
		Status     string
		Severity   string
		Env        string
		PlanDate   time.Time
		PlanStart  time.Time
		PlanEnd    time.Time
		ActualDate time.Time
		Notes      string
		Rollback   string
		Validation string
		Systems    []string
		Components []string
		Steps      []string
		Tags       []string
	}{
		{
			Number:     "REL-2023-001",
			Title:      "v2.0.0 主版本发布",
			Desc:       "全新 UI 设计和用户中心模块上线",
			Type:       "major",
			Status:     "completed",
			Severity:   "high",
			Env:        "production",
			PlanDate:   now.Add(-48 * time.Hour),
			PlanStart:  now.Add(-47 * time.Hour),
			PlanEnd:    now.Add(-46 * time.Hour),
			ActualDate: now.Add(-46 * time.Hour),
			Notes:      "本次发布包含全新的用户界面设计和用户中心模块，支持用户自助服务",
			Rollback:   "1. 停止新版本服务\n2. 回滚代码到 v1.9.0\n3. 启动旧版本服务\n4. 验证核心功能正常",
			Validation: "1. 用户可以正常登录\n2. 新用户可以注册\n3. 用户信息可以修改\n4. 订单列表正常显示",
			Systems:    []string{"用户中心", "API 网关", "前端应用"},
			Components: []string{"用户服务", "认证服务", "前端 UI"},
			Steps: []string{
				"1. 备份数据库",
				"2. 部署新版本代码到预发环境",
				"3. 执行数据库迁移脚本",
				"4. 运行冒烟测试",
				"5. 切换流量到新版本",
				"6. 监控系统指标",
			},
			Tags: []string{"major", "ui", "user-center"},
		},
		{
			Number:     "REL-2023-002",
			Title:      "v2.1.0 功能增强发布",
			Desc:       "工单流程优化和报表功能增强",
			Type:       "minor",
			Status:     "completed",
			Severity:   "medium",
			Env:        "production",
			PlanDate:   now.Add(-30 * 24 * time.Hour),
			PlanStart:  now.Add(-29 * 24 * time.Hour),
			PlanEnd:    now.Add(-28 * 24 * time.Hour),
			ActualDate: now.Add(-28 * 24 * time.Hour),
			Notes:      "优化工单流转流程，新增自定义报表功能",
			Rollback:   "回滚到 v2.0.0 版本",
			Validation: "1. 工单创建流程正常\n2. 工单流转顺畅\n3. 报表可以正常生成",
			Systems:    []string{"工单系统", "报表服务"},
			Components: []string{"工单服务", "报表服务"},
			Steps: []string{
				"1. 部署新版本",
				"2. 验证工单功能",
				"3. 验证报表功能",
			},
			Tags: []string{"minor", "workflow", "report"},
		},
		{
			Number:     "REL-2023-003",
			Title:      "v2.1.1 Bug 修复",
			Desc:       "修复工单附件上传失败的问题",
			Type:       "patch",
			Status:     "in-progress",
			Severity:   "low",
			Env:        "production",
			PlanDate:   now.Add(2 * time.Hour),
			PlanStart:  now.Add(3 * time.Hour),
			PlanEnd:    now.Add(4 * time.Hour),
			Notes:      "紧急修复工单附件上传问题",
			Rollback:   "回滚到 v2.1.0",
			Validation: "1. 工单附件可以正常上传\n2. 大文件上传正常",
			Systems:    []string{"工单系统"},
			Components: []string{"附件服务"},
			Steps: []string{
				"1. 部署补丁",
				"2. 验证附件上传",
			},
			Tags: []string{"patch", "bugfix"},
		},
		{
			Number:     "REL-2023-004",
			Title:      "v3.0.0 大版本升级",
			Desc:       "微服务架构改造和性能优化",
			Type:       "major",
			Status:     "scheduled",
			Severity:   "critical",
			Env:        "production",
			PlanDate:   now.Add(7 * 24 * time.Hour),
			PlanStart:  now.Add(8 * 24 * time.Hour),
			PlanEnd:    now.Add(12 * 24 * time.Hour),
			Notes:      "系统架构升级为微服务，性能提升 50%",
			Rollback:   "1. 停止微服务\n2. 恢复单体架构版本\n3. 验证功能",
			Validation: "1. 所有服务正常运行\n2. 性能指标达标\n3. 数据一致",
			Systems:    []string{"全部系统"},
			Components: []string{"用户服务", "工单服务", "通知服务", "报表服务", "工单服务"},
			Steps: []string{
				"1. 数据库迁移",
				"2. 部署微服务",
				"3. 配置服务发现",
				"4. 配置网关",
				"5. 灰度发布",
				"6. 全量发布",
				"7. 验证监控",
			},
			Tags: []string{"major", "microservice", "performance"},
		},
	}

	for _, r := range releases {
		if _, err := s.client.Release.Create().
			SetReleaseNumber(r.Number).
			SetTitle(r.Title).
			SetDescription(r.Desc).
			SetType(r.Type).
			SetStatus(r.Status).
			SetSeverity(r.Severity).
			SetEnvironment(r.Env).
			SetPlannedReleaseDate(r.PlanDate).
			SetPlannedStartDate(r.PlanStart).
			SetPlannedEndDate(r.PlanEnd).
			SetActualReleaseDate(r.ActualDate).
			SetReleaseNotes(r.Notes).
			SetRollbackProcedure(r.Rollback).
			SetValidationCriteria(r.Validation).
			SetAffectedSystems(r.Systems).
			SetAffectedComponents(r.Components).
			SetDeploymentSteps(r.Steps).
			SetTags(r.Tags).
			SetCreatedBy(creator.ID).
			SetTenantID(t.ID).
			Save(ctx); err != nil {
			s.sugar.Warnw("seed release failed", "error", err, "number", r.Number)
		}
	}
	s.sugar.Infow("releases seeded")
}

// seedSLADefinitions seeds default SLA definitions
func (s *Seeder) seedSLADefinitions(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip SLA definitions seed", "error", err)
		return
	}

	// 检查是否已有 SLA 定义
	existing, err := s.client.SLADefinition.Query().Where(sladefinition.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing SLA definitions failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("SLA definitions already seeded")
		return
	}

	// 创建 SLA 定义
	slas := []struct {
		Name          string
		Desc          string
		ServiceType   string
		Priority      string
		ResponseTime  int
		ResolutionTime int
	}{
		{"SLA-P0-紧急", "P0紧急级别SLA，15分钟响应，2小时解决", "incident", "urgent", 15, 120},
		{"SLA-P1-高", "P1高级别SLA，30分钟响应，4小时解决", "incident", "high", 30, 240},
		{"SLA-P2-中", "P2中级别SLA，2小时响应，8小时解决", "incident", "medium", 120, 480},
		{"SLA-P3-低", "P3低级别SLA，4小时响应，24小时解决", "incident", "low", 240, 1440},
		{"SLA-服务请求", "服务请求标准SLA，8小时响应，3天解决", "service_request", "medium", 480, 4320},
		{"SLA-变更", "变更请求SLA，1小时响应，24小时解决", "change", "high", 60, 1440},
	}

	slaIDs := make(map[string]int)
	for _, sla := range slas {
		entity, err := s.client.SLADefinition.Create().
			SetName(sla.Name).
			SetDescription(sla.Desc).
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
		slaIDs[sla.Name] = entity.ID
	}
	s.sugar.Infow("SLA definitions seeded")
	_ = slaIDs // 用于后续告警规则关联
}

// seedSLAAlertRules seeds default SLA alert rules
func (s *Seeder) seedSLAAlertRules(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip SLA alert rules seed", "error", err)
		return
	}

	// 检查是否已有告警规则
	existing, err := s.client.SLAAlertRule.Query().Where(slaalertrule.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing SLA alert rules failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("SLA alert rules already seeded")
		return
	}

	// 获取 SLA 定义 ID
	slas, err := s.client.SLADefinition.Query().Where(sladefinition.TenantIDEQ(t.ID)).All(ctx)
	if err != nil || len(slas) == 0 {
		s.sugar.Warnw("no SLA definitions found; skip alert rules seed", "error", err)
		return
	}

	slaMap := make(map[string]int)
	for _, sla := range slas {
		slaMap[sla.Name] = sla.ID
	}

	// 创建告警规则
	alertRules := []struct {
		Name               string
		SLAKey             string
		AlertLevel         string
		Threshold          int
		NotificationChans  []string
		EscalationEnabled  bool
	}{
		{"SLA-P0-响应告警", "SLA-P0-紧急", "warning", 50, []string{"email", "sms"}, true},
		{"SLA-P0-解决告警", "SLA-P0-紧急", "critical", 80, []string{"email", "sms", "phone"}, true},
		{"SLA-P1-响应告警", "SLA-P1-高", "warning", 50, []string{"email"}, true},
		{"SLA-P1-解决告警", "SLA-P1-高", "warning", 80, []string{"email"}, true},
		{"SLA-P2-响应告警", "SLA-P2-中", "info", 50, []string{"email"}, false},
		{"SLA-P2-解决告警", "SLA-P2-中", "warning", 80, []string{"email"}, true},
		{"SLA-服务请求-响应告警", "SLA-服务请求", "info", 50, []string{"email"}, false},
		{"SLA-变更-响应告警", "SLA-变更", "warning", 50, []string{"email"}, true},
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
			SetEscalationEnabled(rule.EscalationEnabled).
			SetIsActive(true).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed SLA alert rule failed", "error", err, "name", rule.Name)
		}
	}
	s.sugar.Infow("SLA alert rules seeded")
}

// seedApprovalWorkflows seeds default approval workflows
func (s *Seeder) seedApprovalWorkflows(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip approval workflows seed", "error", err)
		return
	}

	// 检查是否已有审批工作流
	existing, err := s.client.ApprovalWorkflow.Query().Where(approvalworkflow.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing approval workflows failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("approval workflows already seeded")
		return
	}

	// 创建审批工作流
	workflows := []struct {
		Name        string
		Desc        string
		TicketType  string
		Priority    string
		Nodes       []map[string]interface{}
	}{
		{
			Name:       "P0/P1 事件审批",
			Desc:       "紧急和高优先级事件需要主管审批",
			TicketType: "incident",
			Priority:   "urgent,high",
			Nodes: []map[string]interface{}{
				{"type": "approval", "name": "主管审批", "approver_type": "manager", "timeout": 60},
			},
		},
		{
			Name:       "变更审批",
			Desc:       "所有变更请求需要多级审批",
			TicketType: "change",
			Priority:   "",
			Nodes: []map[string]interface{}{
				{"type": "approval", "name": "技术审批", "approver_type": "role", "role": "engineer", "timeout": 240},
				{"type": "approval", "name:": "经理审批", "approver_type": "role", "role": "manager", "timeout": 480},
			},
		},
		{
			Name:       "服务请求审批",
			Desc:       "高价值服务请求需要审批",
			TicketType: "service_request",
			Priority:   "high",
			Nodes: []map[string]interface{}{
				{"type": "approval", "name": "服务审批", "approver_type": "manager", "timeout": 120},
			},
		},
	}

	for _, wf := range workflows {
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
	s.sugar.Infow("approval workflows seeded")
}

// seedProcessBindings seeds default BPMN process bindings
func (s *Seeder) seedProcessBindings(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip process bindings seed", "error", err)
		return
	}

	// 检查是否已有流程绑定
	existing, err := s.client.ProcessBinding.Query().Where(processbinding.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing process bindings failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("process bindings already seeded")
		return
	}

	// 创建流程绑定
	bindings := []struct {
		BusinessType string
		ProcessKey   string
	}{
		{"incident", "incident_process"},
		{"problem", "problem_process"},
		{"change", "change_process"},
		{"service_request", "service_request_process"},
		{"improvement", "improvement_process"},
	}

	for _, b := range bindings {
		_, err := s.client.ProcessBinding.Create().
			SetBusinessType(b.BusinessType).
			SetProcessDefinitionKey(b.ProcessKey).
			SetIsDefault(true).
			SetIsActive(true).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed process binding failed", "error", err, "business_type", b.BusinessType)
		}
	}
	s.sugar.Infow("process bindings seeded")
}

// seedTicketViews seeds default ticket views
func (s *Seeder) seedTicketViews(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip ticket views seed", "error", err)
		return
	}

	// 获取 admin 用户 ID
	admin, err := s.client.User.Query().Where(user.UsernameEQ("admin"), user.TenantIDEQ(t.ID)).First(ctx)
	if err != nil {
		s.sugar.Warnw("admin user not found; skip ticket views seed", "error", err)
		return
	}

	// 检查是否已有工单视图
	existing, err := s.client.TicketView.Query().Where(ticketview.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing ticket views failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("ticket views already seeded")
		return
	}

	// 创建工单视图
	views := []struct {
		Name        string
		Desc        string
		Filters     map[string]interface{}
		Columns     []string
		IsShared    bool
	}{
		{
			Name:     "我的待办工单",
			Desc:     "分配给我的未关闭工单",
			Filters:  map[string]interface{}{"assignee_id": admin.ID, "status": []string{"open", "in_progress", "pending"}},
			Columns:  []string{"id", "title", "priority", "status", "assignee", "created_at"},
			IsShared: false,
		},
		{
			Name:     "我创建的工单",
			Desc:     "我提交的工单",
			Filters:  map[string]interface{}{"creator_id": admin.ID},
			Columns:  []string{"id", "title", "priority", "status", "assignee", "created_at"},
			IsShared: false,
		},
		{
			Name:     "紧急工单",
			Desc:     "所有紧急和高优先级工单",
			Filters:  map[string]interface{}{"priority": []string{"urgent", "high"}},
			Columns:  []string{"id", "title", "priority", "status", "assignee", "created_at"},
			IsShared: true,
		},
		{
			Name:     "未分配工单",
			Desc:     "尚未分配给任何人的工单",
			Filters:  map[string]interface{}{"assignee_id": nil, "status": []string{"open"}},
			Columns:  []string{"id", "title", "priority", "status", "creator", "created_at"},
			IsShared: true,
		},
		{
			Name:     "已关闭工单",
			Desc:     "已完成的工单",
			Filters:  map[string]interface{}{"status": []string{"closed", "resolved"}},
			Columns:  []string{"id", "title", "priority", "status", "assignee", "closed_at"},
			IsShared: false,
		},
	}

	for _, v := range views {
		_, err := s.client.TicketView.Create().
			SetName(v.Name).
			SetDescription(v.Desc).
			SetFilters(v.Filters).
			SetColumns(v.Columns).
			SetIsShared(v.IsShared).
			SetCreatedBy(admin.ID).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed ticket view failed", "error", err, "name", v.Name)
		}
	}
	s.sugar.Infow("ticket views seeded")
}

// seedServiceCatalog seeds enterprise service catalog
func (s *Seeder) seedServiceCatalog(ctx context.Context) {
	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip service catalog seed", "error", err)
		return
	}

	// 检查是否已有服务目录
	existing, err := s.client.ServiceCatalog.Query().Where(servicecatalog.TenantIDEQ(t.ID)).Count(ctx)
	if err != nil {
		s.sugar.Warnw("check existing service catalog failed", "error", err)
		return
	}
	if existing > 0 {
		s.sugar.Infow("service catalog already seeded")
		return
	}

	// 创建企业服务目录
	services := []struct {
		Name             string
		Desc             string
		Category         string
		ServiceType      string
		RequiresApproval bool
		ApprovalLevel    int
		DeliveryTime     int
		Price            float64
		Unit             string
		Status           string
	}{
		// 基础设施服务
		{"云服务器 ECS", "弹性云服务器，支持按需扩容", "云计算", "vm", true, 1, 1, 0, "月", "active"},
		{"云数据库 RDS", "MySQL/PostgreSQL 数据库服务", "数据库", "rds", true, 1, 1, 0, "月", "active"},
		{"对象存储 OSS", "海量、安全、低成本云存储", "存储", "oss", false, 0, 0, 0, "月", "active"},
		{"CDN 加速", "内容分发加速服务", "网络", "network", false, 0, 0, 0, "月", "active"},
		{"负载均衡 SLB", "流量分发负载均衡服务", "网络", "network", true, 1, 1, 0, "月", "active"},
		{"VPN 网关", "VPN 加密通道服务", "安全", "security", true, 2, 2, 0, "月", "active"},
		// 应用服务
		{"企业邮箱", "企业域名邮箱服务", "通讯", "custom", false, 0, 1, 0, "用户/月", "active"},
		{"企业网盘", "企业级文件存储与共享", "协作", "custom", false, 0, 0, 0, "用户/月", "active"},
		{"视频会议", "高清视频会议服务", "通讯", "custom", false, 0, 0, 0, "用户/月", "active"},
		{"企业IM", "企业即时通讯工具", "通讯", "custom", false, 0, 0, 0, "用户/月", "active"},
		// 安全服务
		{"漏洞扫描", "Web应用漏洞扫描服务", "安全", "security", true, 1, 1, 0, "次", "active"},
		{"渗透测试", "安全渗透测试服务", "安全", "security", true, 2, 5, 0, "次", "active"},
		{"等保合规", "等级保护合规咨询", "安全", "security", true, 3, 30, 0, "次", "active"},
		// IT支持服务
		{"IT服务台", "IT问题咨询与支持", "支持", "custom", false, 0, 0, 0, "用户/月", "active"},
		{"软件安装", "标准软件安装申请", "支持", "custom", false, 0, 1, 0, "次", "active"},
		{"账户申请", "新员工账户开通", "支持", "custom", true, 1, 1, 0, "次", "active"},
		{"网络接入", "有线/无线网络接入申请", "支持", "custom", true, 1, 2, 0, "次", "active"},
		{"域名申请", "内部域名注册申请", "支持", "custom", true, 2, 3, 0, "次", "active"},
		// 开发服务
		{"代码仓库", "Git 代码仓库服务", "开发", "custom", false, 0, 0, 0, "用户/月", "active"},
		{"CI/CD流水线", "自动化持续集成/部署", "开发", "custom", false, 0, 0, 0, "用户/月", "active"},
		{"测试环境", "预发布测试环境申请", "开发", "custom", true, 1, 2, 0, "次", "active"},
		{"API网关", "API 接口管理与发布", "开发", "custom", true, 2, 3, 0, "月", "active"},
	}

	for _, svc := range services {
		_, err := s.client.ServiceCatalog.Create().
			SetName(svc.Name).
			SetDescription(svc.Desc).
			SetCategory(svc.Category).
			SetServiceType(svc.ServiceType).
			SetRequiresApproval(svc.RequiresApproval).
			SetApprovalLevel(svc.ApprovalLevel).
			SetDeliveryTime(svc.DeliveryTime).
			SetPrice(svc.Price).
			SetUnit(svc.Unit).
			SetStatus(svc.Status).
			SetIsActive(true).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			s.sugar.Warnw("seed service catalog failed", "error", err, "name", svc.Name)
		}
	}
	s.sugar.Infow("enterprise service catalog seeded", "count", len(services))
}
