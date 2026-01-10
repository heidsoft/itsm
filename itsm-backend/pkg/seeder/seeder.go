package seeder

import (
	"context"
	"encoding/json"
	"itsm-backend/ent"
	"itsm-backend/ent/cloudservice"
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
	s.backfillAdminRole(ctx)
	s.seedUser1(ctx)
	s.seedSecurity1(ctx)
	s.backfillUserRole(ctx)
	s.seedCloudServiceTemplates(ctx)
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
