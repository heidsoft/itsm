package seeder

import (
	"context"
	"encoding/json"
	"itsm-backend/ent"
	"itsm-backend/ent/asset"
	"itsm-backend/ent/assetlicense"
	"itsm-backend/ent/cloudservice"
	"itsm-backend/ent/department"
	"itsm-backend/ent/release"
	"itsm-backend/ent/role"
	"itsm-backend/ent/team"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"
	"os"
	"path/filepath"
	"time"

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

// seedDepartments seeds default departments
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

	// 创建部门
	departments := []struct {
		Name string
		Code string
		Desc string
	}{
		{"IT部门", "IT", "信息技术部门"},
		{"运维部门", "OPS", "运维支持部门"},
		{"客服部门", "CS", "客户服务部门"},
	}

	for _, d := range departments {
		if _, err := s.client.Department.Create().
			SetName(d.Name).
			SetCode(d.Code).
			SetDescription(d.Desc).
			SetTenantID(t.ID).
			Save(ctx); err != nil {
			s.sugar.Warnw("seed department failed", "error", err, "name", d.Name)
		}
	}
	s.sugar.Infow("departments seeded")
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

	// 创建团队
	teams := []struct {
		Name string
		Desc string
	}{
		{"一线支持", "一线技术支持团队"},
		{"二线支持", "二线技术支持团队"},
	}

	for _, tm := range teams {
		if _, err := s.client.Team.Create().
			SetName(tm.Name).
			SetDescription(tm.Desc).
			SetTenantID(t.ID).
			Save(ctx); err != nil {
			s.sugar.Warnw("seed team failed", "error", err, "name", tm.Name)
		}
	}
	s.sugar.Infow("teams seeded")
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

	// 创建角色
	roles := []struct {
		Name string
		Code string
		Desc string
	}{
		{"管理员", "admin", "系统管理员"},
		{"工程师", "engineer", "IT工程师"},
		{"用户", "user", "普通用户"},
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
	s.sugar.Infow("roles seeded")
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
		Number      string
		Name        string
		Description string
		Type        string
		Status      string
		Category    string
		Subcategory string
		Serial      string
		Model       string
		Manufacturer string
		Vendor      string
		Location    string
		Department  string
		Tags        []string
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
			Number:   "REL-2023-001",
			Title:    "v2.0.0 主版本发布",
			Desc:     "全新 UI 设计和用户中心模块上线",
			Type:     "major",
			Status:   "completed",
			Severity: "high",
			Env:      "production",
			PlanDate: now.Add(-48 * time.Hour),
			PlanStart: now.Add(-47 * time.Hour),
			PlanEnd:   now.Add(-46 * time.Hour),
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
			Number:   "REL-2023-002",
			Title:    "v2.1.0 功能增强发布",
			Desc:     "工单流程优化和报表功能增强",
			Type:     "minor",
			Status:   "completed",
			Severity: "medium",
			Env:      "production",
			PlanDate: now.Add(-30 * 24 * time.Hour),
			PlanStart: now.Add(-29 * 24 * time.Hour),
			PlanEnd:   now.Add(-28 * 24 * time.Hour),
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
			Number:   "REL-2023-003",
			Title:    "v2.1.1 Bug 修复",
			Desc:     "修复工单附件上传失败的问题",
			Type:     "patch",
			Status:   "in-progress",
			Severity: "low",
			Env:      "production",
			PlanDate: now.Add(2 * time.Hour),
			PlanStart: now.Add(3 * time.Hour),
			PlanEnd:   now.Add(4 * time.Hour),
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
			Number:   "REL-2023-004",
			Title:    "v3.0.0 大版本升级",
			Desc:     "微服务架构改造和性能优化",
			Type:     "major",
			Status:   "scheduled",
			Severity: "critical",
			Env:      "production",
			PlanDate: now.Add(7 * 24 * time.Hour),
			PlanStart: now.Add(8 * 24 * time.Hour),
			PlanEnd:   now.Add(12 * 24 * time.Hour),
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
