package bootstrap

// B6：启动期权限目录自检接线。
// 反向验证：把 app.go 里的 ValidateAuthzCatalogAtStartup 调用注释掉，
// 真实的「strict 下目录损坏拒绝启动」链路探针必须失败——本文件的单测只覆盖
// helper 语义，链路探针在 integration 层由 app.go 调用点保证。
//
// 用户拍板：默认 strict（全部立即 strict）；report 仅显式降级。

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"itsm-backend/ent/enttest"
	"itsm-backend/internal/authz"
)

func TestResolveAuthzStartupValidationMode(t *testing.T) {
	require.Equal(t, AuthzStartupStrict, ResolveAuthzStartupValidationMode(""))
	require.Equal(t, AuthzStartupStrict, ResolveAuthzStartupValidationMode("strict"))
	require.Equal(t, AuthzStartupStrict, ResolveAuthzStartupValidationMode("bogus"), "未知值必须 fail-closed 到 strict")
	require.Equal(t, AuthzStartupReport, ResolveAuthzStartupValidationMode("report"))
	require.Equal(t, AuthzStartupReport, ResolveAuthzStartupValidationMode(" REPORT "))
}

func TestValidateAuthzCatalogAtStartup_EmptyDBSkips(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:authz_startup_empty?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	reg := authz.NewRegistry(client)
	require.NoError(t, ValidateAuthzCatalogAtStartup(context.Background(), client, reg, AuthzStartupStrict),
		"空库（迁移/播种未跑）不得阻断启动")
}

func TestValidateAuthzCatalogAtStartup_HealthyAndBroken(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:authz_startup_health?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant := client.Tenant.Create().SetName("T").SetCode("T").SetDomain("t.test").SetStatus("active").SaveX(ctx)
	reg := authz.NewRegistry(client)

	// 目录缺码：strict 阻断，report 只返回诊断。
	err := ValidateAuthzCatalogAtStartup(ctx, client, reg, AuthzStartupStrict)
	require.Error(t, err, "目录缺码时 strict 必须拒绝启动")
	require.Contains(t, err.Error(), "tenant")
	_, strictIsDiag := err.(*AuthzStartupDiagnostic)
	require.False(t, strictIsDiag, "strict 必须返回阻断错误，不能是可被 call site 忽略的诊断")

	diagErr := ValidateAuthzCatalogAtStartup(ctx, client, reg, AuthzStartupReport)
	require.Error(t, diagErr)
	_, ok := diagErr.(*AuthzStartupDiagnostic)
	require.True(t, ok, "report 模式必须返回可识别的非阻断诊断")

	// 播种完整目录后 strict 应放行。
	for _, def := range authz.Definitions() {
		// Def 的 policy 细节不影响三元组校验，用派生默认策略生成合法信封。
		encoded, e := authz.EncodeDefinition(def.Name, authz.DerivePolicy(def.Resource, def.Action))
		require.NoError(t, e)
		client.Permission.Create().SetCode(def.Code).SetName(def.Name).
			SetDescription(encoded).SetResource(def.Resource).SetAction(def.Action).
			SetTenantID(tenant.ID).SaveX(ctx)
	}
	require.NoError(t, ValidateAuthzCatalogAtStartup(ctx, client, reg, AuthzStartupStrict),
		"完整目录必须放行 strict 启动")
}
