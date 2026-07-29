package seeder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/menu"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/processbinding"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/role"
	"itsm-backend/ent/rolepermission"
	"itsm-backend/ent/servicecatalog"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/standardchange"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"
	"itsm-backend/internal/initialization"
	"itsm-backend/service"
)

type productionComponentInitializer struct {
	seeder       *Seeder
	name         string
	dependencies []string
	checksum     string
	apply        func(context.Context, *Seeder)
	verify       func(context.Context, *Seeder) error
}

var ProductionComponentNames = []string{
	"identity-rbac",
	"itil-core",
	"workflow-core",
	"sla-core",
	"cmdb-core",
	"extension-core",
}

// ProductionInitializers returns the audited production component DAG. The
// legacy SeedAll helper remains a test/dev convenience and is not registered
// as a production execution unit.
func ProductionInitializers(seeder *Seeder) ([]initialization.Initializer, error) {
	if seeder == nil {
		return nil, fmt.Errorf("seeder is required")
	}
	payload, err := json.Marshal(seeder.config)
	if err != nil {
		return nil, fmt.Errorf("hash product seed manifest: %w", err)
	}
	checksum := func(component string) string {
		sum := sha256.Sum256(append(append([]byte(nil), payload...), []byte(":"+component)...))
		return hex.EncodeToString(sum[:])
	}
	identity := &productionComponentInitializer{
		seeder:   seeder,
		name:     "identity-rbac",
		checksum: checksum("identity-rbac"),
		apply: func(ctx context.Context, transactional *Seeder) {
			transactional.seedDefaultTenant(ctx)
			transactional.seedDepartments(ctx)
			transactional.seedTeams(ctx)
			transactional.seedRoles(ctx)
			transactional.seedPermissions(ctx)
			transactional.seedMenus(ctx)
			transactional.seedAdmin(ctx)
			transactional.seedMenuAndPermissionFixes(ctx)
			transactional.seedRolePermissions(ctx)
		},
		verify: func(ctx context.Context, target *Seeder) error {
			return target.verifyIdentityRBAC(ctx)
		},
	}
	itil := &productionComponentInitializer{
		seeder:       seeder,
		name:         "itil-core",
		dependencies: []string{"identity-rbac"},
		checksum:     checksum("itil-core"),
		apply: func(ctx context.Context, transactional *Seeder) {
			transactional.seedTicketTypes(ctx)
			transactional.seedIncidentCategories(ctx)
			transactional.seedStandardChanges(ctx)
			transactional.seedTicketTags(ctx)
		},
		verify: func(ctx context.Context, target *Seeder) error {
			return target.verifyITILTemplates(ctx)
		},
	}
	workflow := &productionComponentInitializer{
		seeder:       seeder,
		name:         "workflow-core",
		dependencies: []string{"identity-rbac", "itil-core"},
		checksum:     checksum("workflow-core"),
		apply: func(ctx context.Context, transactional *Seeder) {
			transactional.seedApprovalWorkflows(ctx)
			transactional.seedBPMNWorkflows(ctx)
			transactional.seedProcessBindings(ctx)
		},
		verify: func(ctx context.Context, target *Seeder) error {
			return target.verifyWorkflowTemplates(ctx)
		},
	}
	sla := &productionComponentInitializer{
		seeder:       seeder,
		name:         "sla-core",
		dependencies: []string{"itil-core"},
		checksum:     checksum("sla-core"),
		apply: func(ctx context.Context, transactional *Seeder) {
			transactional.seedSLADefinitions(ctx)
			transactional.seedSLAPolicies(ctx)
			transactional.seedSLAAlertRules(ctx)
		},
		verify: func(ctx context.Context, target *Seeder) error {
			return target.verifySLATemplates(ctx)
		},
	}
	cmdb := &productionComponentInitializer{
		seeder:       seeder,
		name:         "cmdb-core",
		dependencies: []string{"identity-rbac"},
		checksum:     checksum("cmdb-core"),
		apply: func(ctx context.Context, transactional *Seeder) {
			transactional.seedCITypes(ctx)
			transactional.seedCloudServiceTemplates(ctx)
		},
		verify: func(ctx context.Context, target *Seeder) error {
			return target.verifyCMDBTemplates(ctx)
		},
	}
	extension := &productionComponentInitializer{
		seeder:       seeder,
		name:         "extension-core",
		dependencies: []string{"workflow-core", "sla-core", "cmdb-core"},
		checksum:     checksum("extension-core"),
		apply: func(ctx context.Context, transactional *Seeder) {
			transactional.seedTicketViews(ctx)
			transactional.seedServiceCatalog(ctx)
		},
		verify: func(ctx context.Context, target *Seeder) error {
			return target.verifyExtensionTemplates(ctx)
		},
	}
	return []initialization.Initializer{identity, itil, workflow, sla, cmdb, extension}, nil
}

func (i *productionComponentInitializer) Name() string { return i.name }
func (i *productionComponentInitializer) Dependencies() []string {
	return append([]string(nil), i.dependencies...)
}
func (i *productionComponentInitializer) Plan(
	_ context.Context,
	_ initialization.Scope,
) (initialization.Plan, error) {
	return initialization.Plan{
		TargetVersion:  CurrentTenantTemplateVersion,
		SourceChecksum: i.checksum,
		Actions: []initialization.Action{{
			Type:      "reconcile",
			SourceKey: i.name,
			Summary:   "reconcile audited P0/T0 component records",
		}},
	}, nil
}
func (i *productionComponentInitializer) Apply(
	ctx context.Context,
	scope initialization.Scope,
	_ initialization.Plan,
	_ int64,
) (initialization.Result, error) {
	if scope.Type != "platform" || scope.ID != 0 {
		return initialization.Result{}, fmt.Errorf("%s requires platform scope", i.name)
	}
	tx, err := i.seeder.client.Tx(ctx)
	if err != nil {
		return initialization.Result{}, fmt.Errorf("begin %s transaction: %w", i.name, err)
	}
	transactional := i.seeder.withClient(tx.Client())
	defer func() {
		_ = tx.Rollback()
	}()
	i.apply(ctx, transactional)
	if err := i.verify(ctx, transactional); err != nil {
		return initialization.Result{}, fmt.Errorf("verify %s transaction: %w", i.name, err)
	}
	if err := tx.Commit(); err != nil {
		return initialization.Result{}, fmt.Errorf("commit %s transaction: %w", i.name, err)
	}
	// Keep the immutable verification view available to Engine.Verify and the
	// verify CLI after the transaction commits. Seed helpers currently build
	// these catalogs from their in-code manifests.
	i.seeder.expectedPermissions = append(
		[]string(nil),
		transactional.expectedPermissions...,
	)
	i.seeder.expectedMenus = append([]string(nil), transactional.expectedMenus...)
	i.seeder.expectedRolePermissions = transactional.expectedRolePermissions
	return initialization.Result{
		Summary: map[string]any{"component": i.name, "version": CurrentTenantTemplateVersion},
		RollbackMetadata: map[string]any{
			"strategy": "forward-fix",
		},
	}, nil
}

func (s *Seeder) withClient(client *ent.Client) *Seeder {
	return &Seeder{
		client:                  client,
		sugar:                   s.sugar,
		config:                  s.config,
		appConfig:               s.appConfig,
		bpmnTemplateService:     service.NewBPMNTemplateService(client),
		expectedPermissions:     append([]string(nil), s.expectedPermissions...),
		expectedMenus:           append([]string(nil), s.expectedMenus...),
		expectedRolePermissions: s.expectedRolePermissions,
	}
}
func (i *productionComponentInitializer) Verify(
	ctx context.Context,
	_ initialization.Scope,
	_ initialization.Plan,
) error {
	return i.verify(ctx, i.seeder)
}

func (s *Seeder) verifyIdentityRBAC(ctx context.Context) error {
	root, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	if err != nil {
		return fmt.Errorf("verify default tenant: %w", err)
	}
	adminExists, err := s.client.User.Query().
		Where(user.UsernameEQ("admin"), user.TenantIDEQ(root.ID)).
		Exist(ctx)
	if err != nil || !adminExists {
		return fmt.Errorf("verify bootstrap administrator: exists=%t err=%w", adminExists, err)
	}
	for _, expected := range s.config.Roles {
		managedRole, err := s.client.Role.Query().
			Where(role.CodeEQ(expected.Code), role.TenantIDEQ(root.ID)).
			Only(ctx)
		if err != nil {
			return fmt.Errorf("verify role %s: %w", expected.Code, err)
		}
		expectedCodes := s.expectedRolePermissions[expected.Code]
		grants, err := s.client.RolePermission.Query().
			Where(rolepermission.RoleIDEQ(managedRole.ID), rolepermission.TenantIDEQ(root.ID)).
			All(ctx)
		if err != nil {
			return err
		}
		if len(grants) < len(expectedCodes) {
			return fmt.Errorf("verify role %s grants: expected-at-least=%d actual=%d", expected.Code, len(expectedCodes), len(grants))
		}
		actualCodes := make(map[string]struct{}, len(grants))
		for _, grant := range grants {
			grantedPermission, err := s.client.Permission.Get(ctx, grant.PermissionID)
			if err != nil {
				return fmt.Errorf("verify role %s permission %d: %w", expected.Code, grant.PermissionID, err)
			}
			actualCodes[grantedPermission.Code] = struct{}{}
		}
		for _, expectedCode := range expectedCodes {
			if _, exists := actualCodes[expectedCode]; !exists {
				return fmt.Errorf("verify role %s missing permission %s", expected.Code, expectedCode)
			}
		}
	}
	permissionCount, err := s.client.Permission.Query().Where(
		permission.TenantIDEQ(root.ID),
		permission.CodeIn(s.expectedPermissions...),
	).Count(ctx)
	if err != nil || permissionCount != len(s.expectedPermissions) {
		return fmt.Errorf("verify permissions: expected=%d actual=%d err=%w", len(s.expectedPermissions), permissionCount, err)
	}
	menuCount, err := s.client.Menu.Query().Where(
		menu.TenantIDEQ(root.ID),
		menu.PathIn(s.expectedMenus...),
	).Count(ctx)
	if err != nil || menuCount != len(s.expectedMenus) {
		return fmt.Errorf("verify menus: expected=%d actual=%d err=%w", len(s.expectedMenus), menuCount, err)
	}
	return nil
}

func (s *Seeder) defaultTenantID(ctx context.Context) (int, error) {
	root, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	if err != nil {
		return 0, err
	}
	return root.ID, nil
}

func (s *Seeder) verifyITILTemplates(ctx context.Context) error {
	tenantID, err := s.defaultTenantID(ctx)
	if err != nil {
		return err
	}
	count, err := s.client.StandardChange.Query().Where(standardchange.TenantIDEQ(tenantID)).Count(ctx)
	if err != nil || count < len(s.config.StandardChanges) {
		return fmt.Errorf("verify standard changes: expected>=%d actual=%d err=%w", len(s.config.StandardChanges), count, err)
	}
	return nil
}

func (s *Seeder) verifyWorkflowTemplates(ctx context.Context) error {
	tenantID, err := s.defaultTenantID(ctx)
	if err != nil {
		return err
	}
	bindings, err := s.client.ProcessBinding.Query().
		Where(processbinding.TenantIDEQ(tenantID), processbinding.IsActiveEQ(true)).
		All(ctx)
	if err != nil {
		return err
	}
	if len(bindings) < len(s.config.ProcessBindings) {
		return fmt.Errorf("verify process bindings: expected>=%d actual=%d", len(s.config.ProcessBindings), len(bindings))
	}
	for _, binding := range bindings {
		exists, err := s.client.ProcessDefinition.Query().Where(
			processdefinition.KeyEQ(binding.ProcessDefinitionKey),
			processdefinition.TenantIDEQ(tenantID),
			processdefinition.IsActiveEQ(true),
		).Exist(ctx)
		if err != nil || !exists {
			return fmt.Errorf("verify process definition %s: exists=%t err=%w", binding.ProcessDefinitionKey, exists, err)
		}
	}
	return nil
}

func (s *Seeder) verifySLATemplates(ctx context.Context) error {
	tenantID, err := s.defaultTenantID(ctx)
	if err != nil {
		return err
	}
	count, err := s.client.SLADefinition.Query().
		Where(sladefinition.TenantIDEQ(tenantID)).
		Count(ctx)
	if err != nil || count < len(s.config.SLADefinitions) {
		return fmt.Errorf("verify SLA definitions: expected>=%d actual=%d err=%w", len(s.config.SLADefinitions), count, err)
	}
	return nil
}

func (s *Seeder) verifyCMDBTemplates(ctx context.Context) error {
	tenantID, err := s.defaultTenantID(ctx)
	if err != nil {
		return err
	}
	exists, err := s.client.CIType.Query().Where(citype.TenantIDEQ(tenantID)).Exist(ctx)
	if err != nil || !exists {
		return fmt.Errorf("verify CI types: exists=%t err=%w", exists, err)
	}
	return nil
}

func (s *Seeder) verifyExtensionTemplates(ctx context.Context) error {
	tenantID, err := s.defaultTenantID(ctx)
	if err != nil {
		return err
	}
	count, err := s.client.ServiceCatalog.Query().
		Where(servicecatalog.TenantIDEQ(tenantID)).
		Count(ctx)
	if err != nil || count < len(s.config.ServiceCatalog) {
		return fmt.Errorf("verify service catalog: expected>=%d actual=%d err=%w", len(s.config.ServiceCatalog), count, err)
	}
	return nil
}
