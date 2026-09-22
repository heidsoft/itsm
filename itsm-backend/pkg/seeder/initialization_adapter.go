package seeder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"entgo.io/ent/dialect"

	"itsm-backend/common/tenantctx"
	"itsm-backend/database"
	"itsm-backend/database/rls"
	"itsm-backend/ent"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/department"
	"itsm-backend/ent/group"
	"itsm-backend/ent/menu"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/processbinding"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/role"
	"itsm-backend/ent/rolepermission"
	"itsm-backend/ent/servicecatalog"
	"itsm-backend/ent/servicecatalogitem"
	"itsm-backend/ent/slaalertrule"
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
	apply        func(context.Context, *Seeder) error
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
	checksum := func(component string) (string, error) {
		return componentChecksum(payload, component)
	}

	checksums := make(map[string]string, len(ProductionComponentNames))
	for _, name := range ProductionComponentNames {
		value, err := checksum(name)
		if err != nil {
			return nil, fmt.Errorf("hash %s manifest: %w", name, err)
		}
		checksums[name] = value
	}
	identity := &productionComponentInitializer{
		seeder:   seeder,
		name:     "identity-rbac",
		checksum: checksums["identity-rbac"],
		apply: func(ctx context.Context, transactional *Seeder) error {
			transactional.seedDefaultTenant(ctx)
			if err := transactional.seedDepartments(ctx); err != nil {
				return err
			}
			transactional.seedTeams(ctx)
			transactional.seedGroups(ctx)
			transactional.seedRoles(ctx)
			transactional.seedPermissions(ctx)
			transactional.seedMenus(ctx)
			transactional.seedAdmin(ctx)
			transactional.seedMenuAndPermissionFixes(ctx)
			transactional.seedRolePermissions(ctx)
			return nil
		},
		verify: func(ctx context.Context, target *Seeder) error {
			return target.verifyIdentityRBAC(ctx)
		},
	}
	itil := &productionComponentInitializer{
		seeder:       seeder,
		name:         "itil-core",
		dependencies: []string{"identity-rbac"},
		checksum:     checksums["itil-core"],
		apply: func(ctx context.Context, transactional *Seeder) error {
			if err := transactional.seedTicketTypes(ctx); err != nil {
				return err
			}
			transactional.seedIncidentCategories(ctx)
			transactional.seedStandardChanges(ctx)
			transactional.seedTicketTags(ctx)
			return nil
		},
		verify: func(ctx context.Context, target *Seeder) error {
			return target.verifyITILTemplates(ctx)
		},
	}
	workflow := &productionComponentInitializer{
		seeder:       seeder,
		name:         "workflow-core",
		dependencies: []string{"identity-rbac", "itil-core"},
		checksum:     checksums["workflow-core"],
		apply: func(ctx context.Context, transactional *Seeder) error {
			transactional.seedApprovalWorkflows(ctx)
			if err := transactional.seedBPMNWorkflows(ctx); err != nil {
				return err
			}
			transactional.seedProcessBindings(ctx)
			return transactional.seedWorkflowTemplates(ctx)
		},
		verify: func(ctx context.Context, target *Seeder) error {
			return target.verifyWorkflowTemplates(ctx)
		},
	}
	sla := &productionComponentInitializer{
		seeder:       seeder,
		name:         "sla-core",
		dependencies: []string{"itil-core"},
		checksum:     checksums["sla-core"],
		apply: func(ctx context.Context, transactional *Seeder) error {
			transactional.seedSLADefinitions(ctx)
			transactional.seedSLAPolicies(ctx)
			return transactional.seedSLAAlertRules(ctx)
		},
		verify: func(ctx context.Context, target *Seeder) error {
			return target.verifySLATemplates(ctx)
		},
	}
	cmdb := &productionComponentInitializer{
		seeder:       seeder,
		name:         "cmdb-core",
		dependencies: []string{"identity-rbac"},
		checksum:     checksums["cmdb-core"],
		apply: func(ctx context.Context, transactional *Seeder) error {
			transactional.seedCITypes(ctx)
			transactional.seedCloudServiceTemplates(ctx)
			return nil
		},
		verify: func(ctx context.Context, target *Seeder) error {
			return target.verifyCMDBTemplates(ctx)
		},
	}
	extension := &productionComponentInitializer{
		seeder:       seeder,
		name:         "extension-core",
		dependencies: []string{"workflow-core", "sla-core", "cmdb-core"},
		checksum:     checksums["extension-core"],
		apply: func(ctx context.Context, transactional *Seeder) error {
			transactional.seedTicketViews(ctx)
			if err := transactional.seedServiceCatalog(ctx); err != nil {
				return err
			}
			return transactional.seedServiceCatalogItems(ctx)
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
	driver dialect.Driver,
) (initialization.Result, error) {
	if scope.Type != "platform" || scope.ID != 0 || !tenantctx.IsSystemBypass(ctx) {
		return initialization.Result{}, fmt.Errorf("%s requires platform scope and explicit system context", i.name)
	}
	if driver == nil {
		return initialization.Result{}, fmt.Errorf("%s requires a component transaction", i.name)
	}
	mode := "off"
	if i.seeder.appConfig != nil {
		mode = i.seeder.appConfig.RLS.Mode
	}
	client := ent.NewClient(ent.Driver(rls.From(driver, mode, i.seeder.sugar)))
	var mutationErr error
	client.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
			value, err := next.Mutate(ctx, mutation)
			mutationErr = errors.Join(mutationErr, err)
			return value, err
		})
	})
	database.RegisterSecurityInterceptors(client, mode)
	transactional := i.seeder.withClient(client)
	transactional.sqlDriver = driver
	applyErr := i.apply(ctx, transactional)
	if err := errors.Join(applyErr, mutationErr); err != nil {
		return initialization.Result{}, fmt.Errorf("apply %s: %w", i.name, err)
	}
	if err := i.verify(ctx, transactional); err != nil {
		return initialization.Result{}, fmt.Errorf("verify %s transaction: %w", i.name, err)
	}
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
	scope initialization.Scope,
	_ initialization.Plan,
) error {
	if scope.Type != "platform" || scope.ID != 0 {
		return fmt.Errorf("%s requires platform scope", i.name)
	}
	return i.verify(ctx, i.seeder)
}

func (s *Seeder) verifyIdentityRBAC(ctx context.Context) error {
	if s.config == nil || len(s.expectedPermissions) == 0 || len(s.expectedMenus) == 0 || len(s.expectedRolePermissions) == 0 ||
		len(s.config.Groups) == 0 || len(s.config.Departments) == 0 {
		return fmt.Errorf("verify identity-rbac: expected baseline is empty")
	}
	root, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	if err != nil {
		return fmt.Errorf("verify default tenant: %w", err)
	}
	adminExists, err := s.client.User.Query().
		Where(user.UsernameEQ("admin"), user.TenantIDEQ(root.ID)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("verify bootstrap administrator: %w", err)
	}
	if !adminExists {
		return fmt.Errorf("verify bootstrap administrator: missing")
	}

	// Compare keys rather than counts: duplicates or customer-owned additions
	// cannot compensate for a missing managed permission or menu.
	permissionCodes, err := s.client.Permission.Query().Where(
		permission.TenantIDEQ(root.ID),
		permission.CodeIn(s.expectedPermissions...),
	).Select(permission.FieldCode).Strings(ctx)
	if err != nil {
		return fmt.Errorf("verify permissions: %w", err)
	}
	actualPermissions := make(map[string]struct{}, len(permissionCodes))
	for _, code := range permissionCodes {
		actualPermissions[code] = struct{}{}
	}
	for _, code := range s.expectedPermissions {
		if _, exists := actualPermissions[code]; !exists {
			return fmt.Errorf("verify permissions: missing %s", code)
		}
	}
	menuPaths, err := s.client.Menu.Query().Where(
		menu.TenantIDEQ(root.ID),
		menu.PathIn(s.expectedMenus...),
	).Select(menu.FieldPath).Strings(ctx)
	if err != nil {
		return fmt.Errorf("verify menus: %w", err)
	}
	actualMenus := make(map[string]struct{}, len(menuPaths))
	for _, path := range menuPaths {
		actualMenus[path] = struct{}{}
	}
	for _, path := range s.expectedMenus {
		if _, exists := actualMenus[path]; !exists {
			return fmt.Errorf("verify menus: missing %s", path)
		}
	}

	// Approval groups are the candidateGroups resolution keys for BPMN tasks;
	// a missing group silently produces empty task candidate sets.
	for _, expected := range s.config.Groups {
		exists, err := s.client.Group.Query().
			Where(group.NameEQ(expected.Name), group.TenantIDEQ(root.ID)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("verify group %s: %w", expected.Name, err)
		}
		if !exists {
			return fmt.Errorf("verify groups: missing %s", expected.Name)
		}
	}

	// Department hierarchy is part of the identity baseline: parent_code must be
	// resolved into a real parent link, otherwise the org tree silently stays flat.
	expectedCodes := make([]string, 0, len(s.config.Departments))
	for _, expected := range s.config.Departments {
		expectedCodes = append(expectedCodes, expected.Code)
	}
	departments, err := s.client.Department.Query().
		Where(
			department.TenantIDEQ(root.ID),
			department.CodeIn(expectedCodes...),
			department.DeletedAtIsNil(),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("verify departments: %w", err)
	}
	actualByID := make(map[int]*ent.Department, len(departments))
	idByCode := make(map[string]int, len(departments))
	countByCode := make(map[string]int, len(departments))
	for _, dept := range departments {
		actualByID[dept.ID] = dept
		idByCode[dept.Code] = dept.ID
		countByCode[dept.Code]++
	}
	for _, expected := range s.config.Departments {
		id, exists := idByCode[expected.Code]
		if !exists {
			return fmt.Errorf("verify departments: missing %s", expected.Code)
		}
		if countByCode[expected.Code] > 1 {
			return fmt.Errorf("verify departments: duplicate %s", expected.Code)
		}
		if expected.ParentCode == "" {
			continue
		}
		parentID, known := idByCode[expected.ParentCode]
		if !known {
			return fmt.Errorf("verify department %s: unknown parent_code %s", expected.Code, expected.ParentCode)
		}
		if actualByID[id].ParentID != parentID {
			return fmt.Errorf("verify department %s: parent expected %d actual %d", expected.Code, parentID, actualByID[id].ParentID)
		}
	}

	// Match seedRoles' builtin + configured role scope, not arbitrary customer roles.
	expectedRoles := append(BuiltinRoles(), s.config.Roles...)
	seenRoles := make(map[string]struct{}, len(expectedRoles))
	for _, expected := range expectedRoles {
		if _, seen := seenRoles[expected.Code]; seen {
			continue
		}
		seenRoles[expected.Code] = struct{}{}
		managedRole, err := s.client.Role.Query().
			Where(role.CodeEQ(expected.Code), role.TenantIDEQ(root.ID)).
			Only(ctx)
		if err != nil {
			return fmt.Errorf("verify role %s: %w", expected.Code, err)
		}
		expectedCodes, managed := s.expectedRolePermissions[expected.Code]
		if managed && len(expectedCodes) == 0 {
			return fmt.Errorf("verify role %s: expected permissions are empty", expected.Code)
		}
		grants, err := s.client.RolePermission.Query().
			Where(rolepermission.RoleIDEQ(managedRole.ID), rolepermission.TenantIDEQ(root.ID)).
			All(ctx)
		if err != nil {
			return fmt.Errorf("verify role %s grants: %w", expected.Code, err)
		}
		actualCodes := make(map[string]struct{}, len(grants))
		for _, grant := range grants {
			grantedPermission, err := s.client.Permission.Query().Where(
				permission.IDEQ(grant.PermissionID), permission.TenantIDEQ(root.ID),
			).Only(ctx)
			if err != nil {
				return fmt.Errorf("verify role %s permission %d in tenant %d: %w", expected.Code, grant.PermissionID, root.ID, err)
			}
			actualCodes[grantedPermission.Code] = struct{}{}
		}
		for _, expectedCode := range expectedCodes {
			if _, exists := actualCodes[expectedCode]; !exists {
				return fmt.Errorf("verify role %s missing permission %s", expected.Code, expectedCode)
			}
		}
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
	if len(s.config.SLADefinitions) == 0 {
		return fmt.Errorf("verify sla-core: expected SLA definition baseline is empty")
	}
	// Only the managed baseline definitions are checked; customer-authored SLA
	// records are not forced to carry seeded alert rules.
	for _, expected := range s.config.SLADefinitions {
		definition, err := s.client.SLADefinition.Query().
			Where(sladefinition.TenantIDEQ(tenantID), sladefinition.NameEQ(expected.Name)).
			Only(ctx)
		if err != nil {
			return fmt.Errorf("verify SLA definition %s: %w", expected.Name, err)
		}
		hasRules, err := s.client.SLAAlertRule.Query().
			Where(slaalertrule.TenantIDEQ(tenantID), slaalertrule.SLADefinitionIDEQ(definition.ID)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("verify SLA alert rules for %s: %w", expected.Name, err)
		}
		if !hasRules {
			return fmt.Errorf("verify SLA alert rules: missing for %s", expected.Name)
		}
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
	if len(s.config.ServiceCatalog) == 0 || len(s.config.ServiceCatalogItems) == 0 {
		return fmt.Errorf("verify extension-core: expected service catalog baseline is empty")
	}
	catalogIDs := make(map[string]int, len(s.config.ServiceCatalog))
	for _, expected := range s.config.ServiceCatalog {
		catalog, err := s.client.ServiceCatalog.Query().Where(
			servicecatalog.TenantIDEQ(tenantID),
			servicecatalog.NameEQ(expected.Name),
		).Only(ctx)
		if err != nil {
			return fmt.Errorf("verify service catalog %s: %w", expected.Name, err)
		}
		catalogIDs[expected.Name] = catalog.ID
	}
	// A catalog can exist while its managed items are missing, which leaves the
	// storefront empty; both levels must be present.
	for _, expected := range s.config.ServiceCatalogItems {
		parentID, known := catalogIDs[expected.CatalogName]
		if !known {
			return fmt.Errorf("verify service catalog item %s: unknown catalog %s", expected.Name, expected.CatalogName)
		}
		exists, err := s.client.ServiceCatalogItem.Query().Where(
			servicecatalogitem.TenantIDEQ(tenantID),
			servicecatalogitem.CatalogIDEQ(parentID),
			servicecatalogitem.NameEQ(expected.Name),
		).Exist(ctx)
		if err != nil {
			return fmt.Errorf("verify service catalog item %s: %w", expected.Name, err)
		}
		if !exists {
			return fmt.Errorf("verify service catalog item %s: missing under %s", expected.Name, expected.CatalogName)
		}
	}
	return nil
}
