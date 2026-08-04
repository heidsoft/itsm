package seeder

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/approvalworkflow"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/menu"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/processbinding"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processdeployment"
	"itsm-backend/ent/role"
	"itsm-backend/ent/rolepermission"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/systemconfig"
	"itsm-backend/ent/tenant"
)

const CurrentTenantTemplateVersion = "1.0.0"

// ProvisionTenant installs the tenant-scoped operating baseline from the
// verified default tenant. The operation is transactional and idempotent.
func (s *Seeder) ProvisionTenant(ctx context.Context, tenantID int, templateVersion string) error {
	if tenantID <= 0 {
		return fmt.Errorf("tenant id must be positive")
	}
	if templateVersion != CurrentTenantTemplateVersion {
		return fmt.Errorf("unsupported tenant template version %q", templateVersion)
	}
	source, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	if err != nil {
		return fmt.Errorf("load product template tenant: %w", err)
	}
	if _, err := s.client.Tenant.Get(ctx, tenantID); err != nil {
		return fmt.Errorf("load target tenant: %w", err)
	}
	if source.ID == tenantID {
		return s.validateTenantReadiness(ctx, tenantID)
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin tenant provisioning: %w", err)
	}
	defer tx.Rollback()
	c := tx.Client()

	sourceRoles, err := c.Role.Query().Where(role.TenantIDEQ(source.ID)).All(ctx)
	if err != nil {
		return err
	}
	targetRoles := make(map[string]*ent.Role, len(sourceRoles))
	for _, item := range sourceRoles {
		target, err := c.Role.Query().Where(role.CodeEQ(item.Code), role.TenantIDEQ(tenantID)).Only(ctx)
		if ent.IsNotFound(err) {
			target, err = c.Role.Create().
				SetCode(item.Code).SetName(item.Name).SetDescription(item.Description).
				SetIsSystem(item.IsSystem).SetIsActive(item.IsActive).SetTenantID(tenantID).Save(ctx)
		}
		if err != nil {
			return fmt.Errorf("provision role %s: %w", item.Code, err)
		}
		targetRoles[item.Code] = target
	}

	sourcePermissions, err := c.Permission.Query().Where(permission.TenantIDEQ(source.ID)).All(ctx)
	if err != nil {
		return err
	}
	targetPermissions := make(map[string]*ent.Permission, len(sourcePermissions))
	for _, item := range sourcePermissions {
		target, err := c.Permission.Query().Where(permission.CodeEQ(item.Code), permission.TenantIDEQ(tenantID)).Only(ctx)
		if ent.IsNotFound(err) {
			target, err = c.Permission.Create().
				SetCode(item.Code).SetName(item.Name).SetDescription(item.Description).
				SetResource(item.Resource).SetAction(item.Action).SetTenantID(tenantID).Save(ctx)
		}
		if err != nil {
			return fmt.Errorf("provision permission %s: %w", item.Code, err)
		}
		targetPermissions[item.Code] = target
	}

	for _, sourceRole := range sourceRoles {
		bindings, err := c.RolePermission.Query().
			Where(rolepermission.RoleIDEQ(sourceRole.ID), rolepermission.TenantIDEQ(source.ID)).
			All(ctx)
		if err != nil {
			return err
		}
		for _, binding := range bindings {
			sourcePermission, err := c.Permission.Get(ctx, binding.PermissionID)
			if err != nil {
				return err
			}
			targetRole := targetRoles[sourceRole.Code]
			targetPermission := targetPermissions[sourcePermission.Code]
			exists, err := c.RolePermission.Query().Where(
				rolepermission.RoleIDEQ(targetRole.ID),
				rolepermission.PermissionIDEQ(targetPermission.ID),
				rolepermission.TenantIDEQ(tenantID),
			).Exist(ctx)
			if err != nil {
				return err
			}
			if !exists {
				if _, err := c.RolePermission.Create().
					SetRoleID(targetRole.ID).SetPermissionID(targetPermission.ID).
					SetTenantID(tenantID).Save(ctx); err != nil {
					return fmt.Errorf("provision role permission: %w", err)
				}
			}
		}
	}

	sourceMenus, err := c.Menu.Query().Where(menu.TenantIDEQ(source.ID)).All(ctx)
	if err != nil {
		return err
	}
	for _, item := range sourceMenus {
		exists, err := c.Menu.Query().Where(menu.PathEQ(item.Path), menu.TenantIDEQ(tenantID)).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			if _, err := c.Menu.Create().
				SetName(item.Name).SetPath(item.Path).SetIcon(item.Icon).
				SetPermissionCode(item.PermissionCode).SetSortOrder(item.SortOrder).
				SetIsVisible(item.IsVisible).SetIsEnabled(item.IsEnabled).
				SetDescription(item.Description).SetTenantID(tenantID).Save(ctx); err != nil {
				return fmt.Errorf("provision menu %s: %w", item.Path, err)
			}
		}
	}

	if err := cloneTenantTemplates(ctx, c, source.ID, tenantID); err != nil {
		return err
	}
	if err := validateTenantClone(ctx, c, source.ID, tenantID); err != nil {
		return fmt.Errorf("validate tenant template before commit: %w", err)
	}
	versionKey := fmt.Sprintf("tenant.bootstrap.version.%d", tenantID)
	version, err := c.SystemConfig.Query().Where(systemconfig.KeyEQ(versionKey), systemconfig.DeletedAtIsNil()).Only(ctx)
	if ent.IsNotFound(err) {
		_, err = c.SystemConfig.Create().
			SetKey(versionKey).SetValue(templateVersion).SetCategory("bootstrap").
			SetDescription("Tenant product template version").SetCreatedBy("system").
			SetTenantID(tenantID).Save(ctx)
	} else if err == nil {
		_, err = version.Update().SetValue(templateVersion).Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("record tenant template version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tenant provisioning: %w", err)
	}
	return validateTenantReadinessWithClient(ctx, s.client, tenantID)
}

func cloneTenantTemplates(ctx context.Context, c *ent.Client, sourceID, tenantID int) error {
	slas, err := c.SLADefinition.Query().Where(sladefinition.TenantIDEQ(sourceID)).All(ctx)
	if err != nil {
		return err
	}
	for _, item := range slas {
		exists, err := c.SLADefinition.Query().Where(sladefinition.NameEQ(item.Name), sladefinition.TenantIDEQ(tenantID)).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			if _, err := c.SLADefinition.Create().
				SetName(item.Name).SetDescription(item.Description).SetServiceType(item.ServiceType).
				SetPriority(item.Priority).SetResponseTime(item.ResponseTime).SetResolutionTime(item.ResolutionTime).
				SetBusinessHours(item.BusinessHours).SetEscalationRules(item.EscalationRules).
				SetConditions(item.Conditions).SetIsActive(item.IsActive).SetTenantID(tenantID).Save(ctx); err != nil {
				return fmt.Errorf("provision SLA %s: %w", item.Name, err)
			}
		}
	}
	ciTypes, err := c.CIType.Query().Where(citype.TenantIDEQ(sourceID)).All(ctx)
	if err != nil {
		return err
	}
	for _, item := range ciTypes {
		exists, err := c.CIType.Query().Where(citype.NameEQ(item.Name), citype.TenantIDEQ(tenantID)).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			if _, err := c.CIType.Create().
				SetName(item.Name).SetDescription(item.Description).SetIcon(item.Icon).SetColor(item.Color).
				SetAttributeSchema(item.AttributeSchema).SetIsActive(item.IsActive).SetTenantID(tenantID).Save(ctx); err != nil {
				return fmt.Errorf("provision CI type %s: %w", item.Name, err)
			}
		}
	}
	workflows, err := c.ApprovalWorkflow.Query().Where(approvalworkflow.TenantIDEQ(sourceID)).All(ctx)
	if err != nil {
		return err
	}
	for _, item := range workflows {
		exists, err := c.ApprovalWorkflow.Query().Where(approvalworkflow.NameEQ(item.Name), approvalworkflow.TenantIDEQ(tenantID)).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			if _, err := c.ApprovalWorkflow.Create().
				SetName(item.Name).SetDescription(item.Description).SetTicketType(item.TicketType).
				SetPriority(item.Priority).SetNodes(item.Nodes).SetStatus(item.Status).
				SetIsActive(item.IsActive).SetTenantID(tenantID).Save(ctx); err != nil {
				return fmt.Errorf("provision approval workflow %s: %w", item.Name, err)
			}
		}
	}
	definitions, err := c.ProcessDefinition.Query().Where(processdefinition.TenantIDEQ(sourceID)).All(ctx)
	if err != nil {
		return err
	}
	for _, item := range definitions {
		exists, err := c.ProcessDefinition.Query().Where(
			processdefinition.KeyEQ(item.Key),
			processdefinition.VersionEQ(item.Version),
			processdefinition.TenantIDEQ(tenantID),
		).Exist(ctx)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		sourceDeployment, err := c.ProcessDeployment.Get(ctx, item.DeploymentID)
		if err != nil {
			return fmt.Errorf("load process deployment for %s: %w", item.Key, err)
		}
		targetDeploymentID := fmt.Sprintf("%s-tenant-%d", sourceDeployment.DeploymentID, tenantID)
		targetDeployment, err := c.ProcessDeployment.Query().
			Where(processdeployment.DeploymentIDEQ(targetDeploymentID)).
			Only(ctx)
		if ent.IsNotFound(err) {
			targetDeployment, err = c.ProcessDeployment.Create().
				SetDeploymentID(targetDeploymentID).
				SetDeploymentName(sourceDeployment.DeploymentName).
				SetDeploymentSource(sourceDeployment.DeploymentSource).
				SetDeployedBy("tenant-provisioner").
				SetDeploymentComment(sourceDeployment.DeploymentComment).
				SetIsActive(sourceDeployment.IsActive).
				SetDeploymentCategory(sourceDeployment.DeploymentCategory).
				SetDeploymentMetadata(sourceDeployment.DeploymentMetadata).
				SetTenantID(tenantID).
				Save(ctx)
		}
		if err != nil {
			return fmt.Errorf("provision process deployment %s: %w", sourceDeployment.DeploymentID, err)
		}
		if _, err := c.ProcessDefinition.Create().
			SetKey(item.Key).SetName(item.Name).SetDescription(item.Description).
			SetVersion(item.Version).SetCategory(item.Category).SetBpmnXML(item.BpmnXML).
			SetProcessVariables(item.ProcessVariables).SetIsActive(item.IsActive).
			SetIsLatest(item.IsLatest).SetDeploymentID(targetDeployment.ID).
			SetDeploymentName(targetDeployment.DeploymentName).SetTenantID(tenantID).Save(ctx); err != nil {
			return fmt.Errorf("provision process definition %s: %w", item.Key, err)
		}
	}
	bindings, err := c.ProcessBinding.Query().Where(processbinding.TenantIDEQ(sourceID)).All(ctx)
	if err != nil {
		return err
	}
	for _, item := range bindings {
		exists, err := c.ProcessBinding.Query().Where(
			processbinding.BusinessTypeEQ(item.BusinessType),
			processbinding.BusinessSubTypeEQ(item.BusinessSubType),
			processbinding.ProcessDefinitionKeyEQ(item.ProcessDefinitionKey),
			processbinding.TenantIDEQ(tenantID),
		).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			targetDefinition, err := c.ProcessDefinition.Query().Where(
				processdefinition.KeyEQ(item.ProcessDefinitionKey),
				processdefinition.TenantIDEQ(tenantID),
				processdefinition.IsActiveEQ(true),
			).Order(ent.Desc(processdefinition.FieldVersion)).First(ctx)
			if err != nil {
				return fmt.Errorf("resolve process definition %s: %w", item.ProcessDefinitionKey, err)
			}
			if _, err := c.ProcessBinding.Create().
				SetBusinessType(item.BusinessType).SetBusinessSubType(item.BusinessSubType).
				SetProcessDefinitionKey(item.ProcessDefinitionKey).SetProcessVersion(item.ProcessVersion).
				SetIsDefault(item.IsDefault).SetPriority(item.Priority).SetIsActive(item.IsActive).
				SetScenario(item.Scenario).SetCategory(item.Category).SetConditions(item.Conditions).
				SetOverrides(item.Overrides).SetTenantID(tenantID).
				SetProcessDefinitionID(targetDefinition.ID).Save(ctx); err != nil {
				return fmt.Errorf("provision process binding: %w", err)
			}
		}
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

func validateTenantClone(ctx context.Context, client *ent.Client, sourceID, targetID int) error {
	sourceRoles, err := client.Role.Query().Where(role.TenantIDEQ(sourceID)).All(ctx)
	if err != nil {
		return err
	}
	for _, sourceRole := range sourceRoles {
		targetRole, err := client.Role.Query().
			Where(role.CodeEQ(sourceRole.Code), role.TenantIDEQ(targetID)).
			Only(ctx)
		if err != nil {
			return fmt.Errorf("managed role %s missing: %w", sourceRole.Code, err)
		}
		sourceGrants, err := client.RolePermission.Query().
			Where(rolepermission.RoleIDEQ(sourceRole.ID), rolepermission.TenantIDEQ(sourceID)).
			All(ctx)
		if err != nil {
			return err
		}
		for _, sourceGrant := range sourceGrants {
			sourcePermission, err := client.Permission.Get(ctx, sourceGrant.PermissionID)
			if err != nil {
				return err
			}
			targetPermission, err := client.Permission.Query().Where(
				permission.CodeEQ(sourcePermission.Code),
				permission.TenantIDEQ(targetID),
			).Only(ctx)
			if err != nil {
				return fmt.Errorf("managed permission %s missing: %w", sourcePermission.Code, err)
			}
			exists, err := client.RolePermission.Query().Where(
				rolepermission.RoleIDEQ(targetRole.ID),
				rolepermission.PermissionIDEQ(targetPermission.ID),
				rolepermission.TenantIDEQ(targetID),
			).Exist(ctx)
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("managed grant %s -> %s missing", sourceRole.Code, sourcePermission.Code)
			}
		}
	}
	sourceMenus, err := client.Menu.Query().Where(menu.TenantIDEQ(sourceID)).All(ctx)
	if err != nil {
		return err
	}
	for _, item := range sourceMenus {
		exists, err := client.Menu.Query().
			Where(menu.PathEQ(item.Path), menu.TenantIDEQ(targetID)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("managed menu %s missing", item.Path)
		}
	}
	sourceDefinitions, err := client.ProcessDefinition.Query().
		Where(processdefinition.TenantIDEQ(sourceID)).
		All(ctx)
	if err != nil {
		return err
	}
	for _, definition := range sourceDefinitions {
		exists, err := client.ProcessDefinition.Query().Where(
			processdefinition.KeyEQ(definition.Key),
			processdefinition.VersionEQ(definition.Version),
			processdefinition.TenantIDEQ(targetID),
		).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("managed process definition %s@%s missing", definition.Key, definition.Version)
		}
	}
	sourceBindings, err := client.ProcessBinding.Query().
		Where(processbinding.TenantIDEQ(sourceID)).
		All(ctx)
	if err != nil {
		return err
	}
	for _, binding := range sourceBindings {
		exists, err := client.ProcessBinding.Query().Where(
			processbinding.BusinessTypeEQ(binding.BusinessType),
			processbinding.BusinessSubTypeEQ(binding.BusinessSubType),
			processbinding.ProcessDefinitionKeyEQ(binding.ProcessDefinitionKey),
			processbinding.TenantIDEQ(targetID),
		).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("managed process binding %s/%s missing", binding.BusinessType, binding.BusinessSubType)
		}
	}
	bindings, err := client.ProcessBinding.Query().
		Where(processbinding.TenantIDEQ(targetID), processbinding.IsActiveEQ(true)).
		All(ctx)
	if err != nil {
		return err
	}
	for _, binding := range bindings {
		exists, err := client.ProcessDefinition.Query().Where(
			processdefinition.KeyEQ(binding.ProcessDefinitionKey),
			processdefinition.TenantIDEQ(targetID),
			processdefinition.IsActiveEQ(true),
		).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("active binding %s has no active process definition", binding.ProcessDefinitionKey)
		}
	}
	return validateTenantReadinessWithClient(ctx, client, targetID)
}
