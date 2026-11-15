package authz_test

import (
	"context"
	"errors"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/permission"
	"itsm-backend/internal/authz"

	_ "github.com/mattn/go-sqlite3"
)

const testTenant = 42

func TestRegistry_UsesDBAsRuntimeSourceAndFailsClosed(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:authz_registry_source?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	policy := authz.DerivePolicy("ticket", "write")
	encoded, err := authz.EncodeDefinition("编辑工单", policy)
	if err != nil {
		t.Fatal(err)
	}
	client.Permission.Create().
		SetCode("ticket:write").SetName("管理工单").SetDescription(encoded).
		SetResource("ticket").SetAction("write").SetTenantID(testTenant).
		SaveX(ctx)

	r := authz.NewRegistry(client)

	// human 显式授权；agent 未授权（默认零权限）。
	needs, min, err := r.AuthorizePrincipal(ctx, testTenant, "ticket:write", authz.PrincipalHuman)
	if err != nil || needs || min != 0 {
		t.Fatalf("human 应直接放行: needs=%v min=%d err=%v", needs, min, err)
	}
	if _, _, err := r.AuthorizePrincipal(ctx, testTenant, "ticket:write", authz.PrincipalAgent); err == nil {
		t.Fatal("未显式授权的 agent 必须被拒绝")
	}
	if _, err := r.Lookup(ctx, testTenant, "ticket:delete"); !errors.Is(err, authz.ErrDefinitionNotFound) {
		t.Fatalf("不存在的码必须 fail-closed，得到 %v", err)
	}

	// 运行期把 agent 加进白名单（中风险 → 单人审批），失效缓存后立即生效。
	policy.AllowedPrincipals = []string{authz.PrincipalHuman, authz.PrincipalAgent}
	updated, err := authz.EncodeDefinition("编辑工单", policy)
	if err != nil {
		t.Fatal(err)
	}
	client.Permission.Update().
		Where(permission.CodeEQ("ticket:write"), permission.TenantIDEQ(testTenant)).
		SetDescription(updated).SaveX(ctx)
	r.Invalidate(testTenant)

	needs, min, err = r.AuthorizePrincipal(ctx, testTenant, "ticket:write", authz.PrincipalAgent)
	if err != nil || !needs || min != 1 {
		t.Fatalf("配置后 agent 应需 1 人审批: needs=%v min=%d err=%v", needs, min, err)
	}

	// 关掉该码后任何主体都不得使用。
	policy.IsActive = false
	disabled, err := authz.EncodeDefinition("编辑工单", policy)
	if err != nil {
		t.Fatal(err)
	}
	client.Permission.Update().
		Where(permission.CodeEQ("ticket:write"), permission.TenantIDEQ(testTenant)).
		SetDescription(disabled).SaveX(ctx)
	r.Invalidate(testTenant)
	if _, err := r.Lookup(ctx, testTenant, "ticket:write"); !errors.Is(err, authz.ErrDefinitionInactive) {
		t.Fatalf("停用码必须拒绝，得到 %v", err)
	}
}

func TestRegistry_CorruptRowFailsClosedForWholeTenant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:authz_registry_corrupt?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	client.Permission.Create().
		SetCode("ticket:write").SetName("管理工单").
		SetDescription(`{"v":1,"policy":{"riskLevel":"unknown"}}`).
		SetResource("ticket").SetAction("write").SetTenantID(testTenant).
		SaveX(ctx)

	r := authz.NewRegistry(client)
	if _, err := r.All(ctx, testTenant); !errors.Is(err, authz.ErrDefinitionCorrupt) {
		t.Fatalf("损坏策略必须让整租户 fail-closed，得到 %v", err)
	}
	if _, err := r.Lookup(ctx, testTenant, "ticket:write"); !errors.Is(err, authz.ErrDefinitionCorrupt) {
		t.Fatalf("损坏策略不得按宽松默认值放行，得到 %v", err)
	}
}

func TestRegistry_MalformedCodeTripleFailsClosed(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:authz_registry_triple?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	client.Permission.Create().
		SetCode("ticket:write").SetName("管理工单").SetDescription("").
		SetResource("incident").SetAction("write").SetTenantID(testTenant).
		SaveX(ctx)

	r := authz.NewRegistry(client)
	if _, err := r.All(ctx, testTenant); !errors.Is(err, authz.ErrDefinitionCorrupt) {
		t.Fatalf("code != resource:action 必须 fail-closed，得到 %v", err)
	}
}

func TestRegistry_DuplicateCodeFailsClosed(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:authz_registry_dup?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		client.Permission.Create().
			SetCode("ticket:read").SetName("查看工单").SetDescription("").
			SetResource("ticket").SetAction("read").SetTenantID(testTenant).
			SaveX(ctx)
	}
	r := authz.NewRegistry(client)
	if _, err := r.All(ctx, testTenant); !errors.Is(err, authz.ErrDefinitionDuplicated) {
		t.Fatalf("重复码必须 fail-closed，得到 %v", err)
	}
}

func TestRegistry_UnconfiguredClientNeverFallsBackToBootstrap(t *testing.T) {
	r := authz.NewRegistry(nil)
	if _, err := r.Lookup(context.Background(), 1, "ticket:read"); !errors.Is(err, authz.ErrCatalogUnavailable) {
		t.Fatalf("未装配 client 必须 fail-closed，得到 %v", err)
	}
}
