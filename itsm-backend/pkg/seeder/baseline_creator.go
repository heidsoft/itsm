package seeder

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/user"
	"itsm-backend/service"
)

// Tenant-scoped baseline rows still need an audit owner, and the template
// tables reference users(id) with NOT NULL. This account is that owner.
// 前缀与租户生命周期共用同一常量（删除守卫据此排除基线归属账号）。
const tenantSystemAccountPrefix = service.BaselineSystemAccountPrefix

// unusablePasswordHash can never satisfy bcrypt comparison, so the account
// cannot authenticate even if it were activated. No credential is seeded.
const unusablePasswordHash = "!"

func tenantSystemAccountName(tenantID int) string {
	return fmt.Sprintf("%s%d", tenantSystemAccountPrefix, tenantID)
}

// ensureTenantSystemAccount creates the non-loginable owner account for one
// tenant. It is idempotent and never rewrites an account an operator edited.
func (s *Seeder) ensureTenantSystemAccount(ctx context.Context, tenantID int) (*ent.User, error) {
	name := tenantSystemAccountName(tenantID)
	existing, err := s.client.User.Query().
		Where(user.UsernameEQ(name), user.TenantIDEQ(tenantID)).
		Only(ctx)
	if err == nil {
		return existing, nil
	}
	if !ent.IsNotFound(err) {
		return nil, fmt.Errorf("lookup tenant system account: %w", err)
	}
	created, err := s.client.User.Create().
		SetUsername(name).
		SetEmail(name + "@tenant.invalid").
		SetName("系统基线账号").
		SetPasswordHash(unusablePasswordHash).
		SetRole(user.RoleEndUser).
		SetActive(false).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create tenant system account: %w", err)
	}
	return created, nil
}

// baselineCreator resolves the audit owner for template rows in the baseline
// tenant: the platform bootstrap administrator, or the tenant system account
// installed by identity-rbac for tenant-scoped provisioning.
func (s *Seeder) baselineCreator(ctx context.Context, tenantID int) (*ent.User, error) {
	admin, err := s.client.User.Query().
		Where(user.UsernameEQ("admin"), user.TenantIDEQ(tenantID)).
		Only(ctx)
	if err == nil {
		return admin, nil
	}
	if !ent.IsNotFound(err) {
		return nil, fmt.Errorf("lookup baseline administrator: %w", err)
	}
	account, err := s.client.User.Query().
		Where(user.UsernameEQ(tenantSystemAccountName(tenantID)), user.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"tenant %d has no baseline creator; identity-rbac must install it before template components: %w",
			tenantID, err)
	}
	return account, nil
}
