package seeder

import (
	"context"
	"encoding/json"
	"itsm-backend/ent"
	"itsm-backend/ent/cloudservice"
	"itsm-backend/ent/department"
	"itsm-backend/ent/role"
	"itsm-backend/ent/team"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"
	"os"
	"path/filepath"

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
