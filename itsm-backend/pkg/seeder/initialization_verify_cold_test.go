package seeder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"itsm-backend/common/tenantctx"
	"itsm-backend/config"
	"itsm-backend/ent"
	"itsm-backend/ent/department"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/group"
	"itsm-backend/ent/menu"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/role"
	"itsm-backend/ent/rolepermission"
	"itsm-backend/ent/tenant"
	"itsm-backend/internal/authz"
	"itsm-backend/internal/initialization"
	"itsm-backend/pkg/tenantmode"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newColdVerifyFixture(t *testing.T) (*Seeder, *sql.DB, context.Context) {
	t.Helper()
	t.Setenv("ADMIN_PASSWORD", "cold-verify-test-password")
	t.Setenv("ITSM_SEED_CONFIG", "")
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	require.NoError(t, err)
	// query_only and total_changes apply to this one SQLite connection.
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(entsql.OpenDB(dialect.SQLite, db))))
	seeder := NewSeeder(client, zap.NewNop().Sugar(), &config.Config{
		Deployment: config.DeploymentConfig{Mode: tenantmode.DeploymentModePrivate},
	})
	ctx := tenantctx.SystemContext(context.Background(), "initialization:test", "cold verify regression")
	require.NoError(t, seeder.SeedProduction(ctx))
	return seeder, db, ctx
}

// Mirror the verify-only CLI path. Neither the fresh Seeder nor its components
// ever run Apply; the database is read-only before even constructing them.
func verifyColdReadOnly(t *testing.T, seeded *Seeder, db *sql.DB, ctx context.Context, alter func(*Seeder)) error {
	t.Helper()
	var before, after int64
	require.NoError(t, db.QueryRowContext(ctx, "SELECT total_changes()").Scan(&before))
	_, err := db.ExecContext(ctx, "PRAGMA query_only = ON")
	require.NoError(t, err)
	writeAttempts := 0
	seeded.client.Use(func(ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(context.Context, ent.Mutation) (ent.Value, error) {
			writeAttempts++
			return nil, errors.New("verify must not write")
		})
	})
	cold := NewSeeder(seeded.client, zap.NewNop().Sugar(), seeded.appConfig)
	if alter != nil {
		alter(cold)
	}
	components, err := ProductionInitializers(cold)
	require.NoError(t, err)
	scope := initialization.Scope{Type: "platform", ID: 0}
	var verifyErr error
	for _, component := range components {
		plan, err := component.Plan(ctx, scope)
		require.NoError(t, err, component.Name())
		if err := component.Verify(ctx, scope, plan); err != nil {
			verifyErr = fmt.Errorf("%s: %w", component.Name(), err)
			break
		}
	}
	require.Zero(t, writeAttempts, "verify attempted a mutation, even if its error was swallowed")
	require.NoError(t, db.QueryRowContext(ctx, "SELECT total_changes()").Scan(&after))
	require.Equal(t, before, after, "verify changed the database")
	return verifyErr
}

func TestColdVerifyProductionBaseline(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{name: "complete_with_customer_customizations_and_other_tenant"},
		{name: "missing_menu", want: "verify menus: missing /dashboard"},
		{name: "missing_permission", want: "verify permissions: missing ticket:read"},
		{name: "missing_role_grant", want: "verify role sysadmin missing permission ticket:read"},
		{name: "missing_builtin_role_grant", want: "verify role agent missing permission ticket:read"},
		{name: "foreign_permission_grant", want: "verify role sysadmin permission"},
		{name: "dangling_permission_grant", want: "verify role sysadmin permission"},
		{name: "foreign_grant_tenant", want: "verify role sysadmin missing permission ticket:read"},
		{name: "duplicate_menu_cannot_mask_missing", want: "verify menus: missing /dashboard"},
		{name: "duplicate_permission_cannot_mask_missing", want: "verify permissions: missing ticket:read"},
		{name: "missing_group", want: "verify groups: missing approvers-l1"},
		{name: "empty_groups_baseline", want: "expected baseline is empty"},
		{name: "flat_department_cannot_pass_hierarchy", want: "verify department IT-INFRA: parent expected"},
		{name: "missing_department", want: "verify departments: missing IT-SEC"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, db, ctx := newColdVerifyFixture(t)
			root, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
			require.NoError(t, err)
			other, err := s.client.Tenant.Create().SetCode("other-customer").SetName("Other Customer").Save(ctx)
			require.NoError(t, err)
			localPermission, err := s.client.Permission.Query().Where(
				permission.TenantIDEQ(root.ID), permission.CodeEQ("ticket:read"),
			).Only(ctx)
			require.NoError(t, err)
			foreignPermission, err := s.client.Permission.Create().SetTenantID(other.ID).
				SetCode("ticket:read").SetName("Other Ticket Read").SetResource("ticket").SetAction("read").Save(ctx)
			require.NoError(t, err)
			_, err = s.client.Menu.Create().SetTenantID(other.ID).SetPath("/dashboard").SetName("Other Dashboard").Save(ctx)
			require.NoError(t, err)
			adminRole, err := s.client.Role.Query().Where(role.TenantIDEQ(root.ID), role.CodeEQ("sysadmin")).Only(ctx)
			require.NoError(t, err)
			otherRole, err := s.client.Role.Create().SetTenantID(other.ID).SetCode("sysadmin").SetName("Other Admin").Save(ctx)
			require.NoError(t, err)
			_, err = s.client.RolePermission.Create().SetTenantID(other.ID).
				SetRoleID(otherRole.ID).SetPermissionID(foreignPermission.ID).Save(ctx)
			require.NoError(t, err)

			// Customer additions, including an extra grant on a managed role, must
			// neither cause false failures nor compensate for missing baseline keys.
			customPermission, err := s.client.Permission.Create().SetTenantID(root.ID).
				SetCode("customer:custom").SetName("Customer Custom").SetResource("customer").SetAction("custom").Save(ctx)
			require.NoError(t, err)
			_, err = s.client.RolePermission.Create().SetTenantID(root.ID).
				SetRoleID(adminRole.ID).SetPermissionID(customPermission.ID).Save(ctx)
			require.NoError(t, err)
			_, err = s.client.Role.Create().SetTenantID(root.ID).SetCode("customer_custom").SetName("Customer Role").Save(ctx)
			require.NoError(t, err)
			_, err = s.client.Menu.Create().SetTenantID(root.ID).SetPath("/customer/custom").SetName("Customer Menu").Save(ctx)
			require.NoError(t, err)
			_, err = s.client.Menu.Update().Where(menu.TenantIDEQ(root.ID), menu.PathEQ("/dashboard")).
				SetIsEnabled(false).SetIsVisible(false).SetName("Customized Dashboard").Save(ctx)
			require.NoError(t, err)

			var alter func(*Seeder)
			switch tc.name {
			case "missing_menu", "duplicate_menu_cannot_mask_missing":
				_, err = s.client.Menu.Delete().Where(menu.TenantIDEQ(root.ID), menu.PathEQ("/dashboard")).Exec(ctx)
				require.NoError(t, err)
				if tc.name == "duplicate_menu_cannot_mask_missing" {
					_, err = s.client.Menu.Create().SetTenantID(root.ID).SetPath("/tickets").SetName("Duplicate Tickets").Save(ctx)
					require.NoError(t, err)
				}
			case "missing_permission", "duplicate_permission_cannot_mask_missing":
				_, err = s.client.RolePermission.Delete().Where(
					rolepermission.TenantIDEQ(root.ID), rolepermission.PermissionIDEQ(localPermission.ID),
				).Exec(ctx)
				require.NoError(t, err)
				require.NoError(t, s.client.Permission.DeleteOne(localPermission).Exec(ctx))
				if tc.name == "duplicate_permission_cannot_mask_missing" {
					_, err = s.client.Permission.Create().SetTenantID(root.ID).SetCode("ticket:write").
						SetName("Duplicate Write").SetResource("ticket").SetAction("write").Save(ctx)
					require.NoError(t, err)
				}
			case "missing_role_grant", "missing_builtin_role_grant":
				targetRole := adminRole
				if tc.name == "missing_builtin_role_grant" {
					targetRole, err = s.client.Role.Query().Where(role.TenantIDEQ(root.ID), role.CodeEQ("agent")).Only(ctx)
					require.NoError(t, err)
				}
				_, err = s.client.RolePermission.Delete().Where(rolepermission.TenantIDEQ(root.ID),
					rolepermission.RoleIDEQ(targetRole.ID), rolepermission.PermissionIDEQ(localPermission.ID)).Exec(ctx)
				require.NoError(t, err)
			case "foreign_permission_grant", "dangling_permission_grant":
				permissionID := foreignPermission.ID
				if tc.name == "dangling_permission_grant" {
					permissionID = 999999
				}
				_, err = s.client.RolePermission.Update().Where(rolepermission.TenantIDEQ(root.ID),
					rolepermission.RoleIDEQ(adminRole.ID), rolepermission.PermissionIDEQ(localPermission.ID)).
					SetPermissionID(permissionID).Save(ctx)
				require.NoError(t, err)
			case "foreign_grant_tenant":
				_, err = s.client.RolePermission.Update().Where(rolepermission.TenantIDEQ(root.ID),
					rolepermission.RoleIDEQ(adminRole.ID), rolepermission.PermissionIDEQ(localPermission.ID)).
					SetTenantID(other.ID).Save(ctx)
				require.NoError(t, err)
			case "missing_group":
				_, err = s.client.Group.Delete().Where(
					group.TenantIDEQ(root.ID), group.NameEQ("approvers-l1"),
				).Exec(ctx)
				require.NoError(t, err)
			case "empty_groups_baseline":
				alter = func(cold *Seeder) { cold.config.Groups = []GroupSeed{} }
			case "flat_department_cannot_pass_hierarchy":
				require.NoError(t, s.client.Department.Update().Where(
					department.TenantIDEQ(root.ID), department.CodeEQ("IT-INFRA"),
				).ClearParentID().Exec(ctx))
			case "missing_department":
				_, err = s.client.Department.Delete().Where(
					department.TenantIDEQ(root.ID), department.CodeEQ("IT-SEC"),
				).Exec(ctx)
				require.NoError(t, err)
			}
			err = verifyColdReadOnly(t, s, db, ctx, alter)
			if tc.want == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.want)
			}
		})
	}
}

func TestColdVerifyRejectsEmptyExpectations(t *testing.T) {
	cases := []struct {
		name  string
		alter func(*Seeder)
	}{
		{"permissions", func(s *Seeder) { s.expectedPermissions = nil }},
		{"menus", func(s *Seeder) { s.expectedMenus = nil }},
		{"role_permissions", func(s *Seeder) { s.expectedRolePermissions = nil }},
		{"managed_role_permissions", func(s *Seeder) { s.expectedRolePermissions["sysadmin"] = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, db, ctx := newColdVerifyFixture(t)
			err := verifyColdReadOnly(t, s, db, ctx, tc.alter)
			require.ErrorContains(t, err, "empty")
		})
	}
}

func TestColdVerifyPropagatesPermissionQueryError(t *testing.T) {
	s, db, ctx := newColdVerifyFixture(t)
	queryErr := errors.New("injected permission read failure")
	s.client.Permission.Intercept(ent.InterceptFunc(func(ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(context.Context, ent.Query) (ent.Value, error) {
			return nil, queryErr
		})
	}))
	err := verifyColdReadOnly(t, s, db, ctx, nil)
	require.ErrorIs(t, err, queryErr)
}

func TestColdVerifyRejectsNonPlatformScope(t *testing.T) {
	s, db, ctx := newColdVerifyFixture(t)
	require.NoError(t, verifyColdReadOnly(t, s, db, ctx, nil))
	components, err := ProductionInitializers(NewSeeder(s.client, zap.NewNop().Sugar(), s.appConfig))
	require.NoError(t, err)
	for _, scope := range []initialization.Scope{{Type: "tenant", ID: 1}, {Type: "platform", ID: 1}, {}} {
		for _, component := range components {
			plan, err := component.Plan(ctx, scope)
			require.NoError(t, err)
			require.ErrorContains(t, component.Verify(ctx, scope, plan), "requires platform scope")
		}
	}
}

func TestColdVerifyExpectationsAreDeterministicAndIndependent(t *testing.T) {
	permissions, menus, grants := identityRBACExpectations()
	require.Equal(t, AllDefinedPermissionCodes(), permissions)
	require.Len(t, menuDefinitions(), 78)
	require.Len(t, menus, 78)
	for code, expected := range authz.BuiltinRolePermissionCodes() {
		require.ElementsMatch(t, expected, grants[code], code)
	}
	for i := 0; i < 10; i++ {
		nextPermissions, nextMenus, nextGrants := identityRBACExpectations()
		require.Equal(t, permissions, nextPermissions)
		require.Equal(t, menus, nextMenus)
		require.Equal(t, grants, nextGrants)
		nextPermissions[0] = "modified"
		nextMenus[0] = "modified"
		nextGrants["sysadmin"][0] = "modified"
	}
}
