package seeder

import (
	"context"
	"fmt"

	"itsm-backend/common/tenantctx"
	"itsm-backend/ent"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/group"
	"itsm-backend/ent/menu"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/role"
	"itsm-backend/ent/rolepermission"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/systemconfig"
	"itsm-backend/internal/initialization"
)

const CurrentTenantTemplateVersion = "1.0.0"

// ProvisionTenant installs the product baseline into one tenant by running the
// same audited component DAG used for platform initialization, inside a single
// transaction. It deliberately does not copy rows out of the default tenant:
// that tenant is a live customer tenant in a private deployment, so copying it
// would spread customer data as if it were a product template.
func (s *Seeder) ProvisionTenant(ctx context.Context, tenantID int, templateVersion string) error {
	if tenantID <= 0 {
		return fmt.Errorf("tenant id must be positive")
	}
	if templateVersion != CurrentTenantTemplateVersion {
		return fmt.Errorf("unsupported tenant template version %q", templateVersion)
	}
	if !tenantctx.IsSystemBypass(ctx) {
		return fmt.Errorf("tenant provisioning requires an explicit system context")
	}
	if s.sqlDriver == nil {
		return fmt.Errorf("tenant provisioning requires the baseline SQL driver")
	}
	target, err := s.client.Tenant.Get(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("load target tenant: %w", err)
	}
	if target.Code == platformTenantCode {
		// The platform tenant owns the platform-only steps and is installed by
		// the bootstrap DAG; re-provisioning it here would run them out of order.
		return s.validateTenantReadiness(ctx, tenantID)
	}

	scope := initialization.Scope{Type: "tenant", ID: int64(tenantID)}
	components, err := ProductionInitializers(s)
	if err != nil {
		return fmt.Errorf("create tenant initializers: %w", err)
	}
	tx, err := s.sqlDriver.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin tenant provisioning: %w", err)
	}
	driver := initialization.NewTransactionDriver(tx, s.sqlDriver.Dialect())
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for _, component := range components {
		plan, err := component.Plan(ctx, scope)
		if err != nil {
			return fmt.Errorf("plan %s for tenant %d: %w", component.Name(), tenantID, err)
		}
		if _, err := component.Apply(ctx, scope, plan, driver); err != nil {
			return fmt.Errorf("provision %s into tenant %d: %w", component.Name(), tenantID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tenant provisioning: %w", err)
	}
	committed = true

	// Post-commit verification is read-only and compares against the code
	// baseline, so a partially installed tenant can never be reported as ready.
	verified, err := ProductionInitializers(s)
	if err != nil {
		return fmt.Errorf("create verification components: %w", err)
	}
	for _, component := range verified {
		plan, err := component.Plan(ctx, scope)
		if err != nil {
			return fmt.Errorf("plan verification for %s: %w", component.Name(), err)
		}
		if err := component.Verify(ctx, scope, plan); err != nil {
			return fmt.Errorf("verify tenant %d after provisioning %s: %w", tenantID, component.Name(), err)
		}
	}
	if err := s.recordTenantTemplateVersion(ctx, tenantID, templateVersion); err != nil {
		return err
	}
	return s.validateTenantReadiness(ctx, tenantID)
}

// ComponentVerification reports one component's read-only verification result
// for a tenant scope. It is what the tenant initialization status API returns.
type ComponentVerification struct {
	Component string
	Verified  bool
	Error     string
}

// VerifyTenantBaseline verifies a tenant's product baseline without writing.
// Unlike ProvisionTenant it reports every component instead of stopping at the
// first failure, so an operator can see the whole gap at once.
func (s *Seeder) VerifyTenantBaseline(ctx context.Context, tenantID int) ([]ComponentVerification, error) {
	if tenantID <= 0 {
		return nil, fmt.Errorf("tenant id must be positive")
	}
	if _, err := s.client.Tenant.Get(ctx, tenantID); err != nil {
		return nil, fmt.Errorf("load target tenant: %w", err)
	}
	components, err := ProductionInitializers(s)
	if err != nil {
		return nil, fmt.Errorf("create verification components: %w", err)
	}
	scope := initialization.Scope{Type: "tenant", ID: int64(tenantID)}
	results := make([]ComponentVerification, 0, len(components))
	for _, component := range components {
		plan, err := component.Plan(ctx, scope)
		if err != nil {
			return results, fmt.Errorf("plan verification for %s: %w", component.Name(), err)
		}
		entry := ComponentVerification{Component: component.Name()}
		if verifyErr := component.Verify(ctx, scope, plan); verifyErr != nil {
			entry.Error = verifyErr.Error()
		} else {
			entry.Verified = true
		}
		results = append(results, entry)
	}
	return results, nil
}

// recordTenantTemplateVersion keeps the historical version marker readable for
// one release while the initialization ledger stays the source of truth.
func (s *Seeder) recordTenantTemplateVersion(ctx context.Context, tenantID int, templateVersion string) error {
	versionKey := fmt.Sprintf("tenant.bootstrap.version.%d", tenantID)
	version, err := s.client.SystemConfig.Query().
		Where(systemconfig.KeyEQ(versionKey), systemconfig.DeletedAtIsNil()).
		Only(ctx)
	switch {
	case err == nil:
		_, err = version.Update().SetValue(templateVersion).Save(ctx)
	case ent.IsNotFound(err):
		_, err = s.client.SystemConfig.Create().
			SetKey(versionKey).SetValue(templateVersion).SetCategory("bootstrap").
			SetDescription("Tenant product template version").SetCreatedBy("system").
			SetTenantID(tenantID).Save(ctx)
	default:
		return fmt.Errorf("lookup tenant template version: %w", err)
	}
	if err != nil {
		return fmt.Errorf("record tenant template version: %w", err)
	}
	return nil
}

func (s *Seeder) validateTenantReadiness(ctx context.Context, tenantID int) error {
	return validateTenantReadinessWithClient(ctx, s.client, tenantID)
}

func validateTenantReadinessWithClient(ctx context.Context, client *ent.Client, tenantID int) error {
	checks := []struct {
		name  string
		count func() (int, error)
	}{
		{"roles", func() (int, error) { return client.Role.Query().Where(role.TenantIDEQ(tenantID)).Count(ctx) }},
		{"permissions", func() (int, error) {
			return client.Permission.Query().Where(permission.TenantIDEQ(tenantID)).Count(ctx)
		}},
		{"role permissions", func() (int, error) {
			return client.RolePermission.Query().Where(rolepermission.TenantIDEQ(tenantID)).Count(ctx)
		}},
		{"menus", func() (int, error) { return client.Menu.Query().Where(menu.TenantIDEQ(tenantID)).Count(ctx) }},
		{"nested menus", func() (int, error) {
			return client.Menu.Query().Where(menu.TenantIDEQ(tenantID), menu.ParentIDNotNil()).Count(ctx)
		}},
		{"groups", func() (int, error) { return client.Group.Query().Where(group.TenantIDEQ(tenantID)).Count(ctx) }},
		{"SLA definitions", func() (int, error) {
			return client.SLADefinition.Query().Where(sladefinition.TenantIDEQ(tenantID)).Count(ctx)
		}},
		{"CI types", func() (int, error) { return client.CIType.Query().Where(citype.TenantIDEQ(tenantID)).Count(ctx) }},
	}
	for _, check := range checks {
		count, err := check.count()
		if err != nil {
			return fmt.Errorf("validate tenant %s: %w", check.name, err)
		}
		if count == 0 {
			return fmt.Errorf("validate tenant %s: no records installed", check.name)
		}
	}
	return nil
}
