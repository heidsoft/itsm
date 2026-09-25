package tenant

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"itsm-backend/common/tenantctx"
	"itsm-backend/database"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/ent/systemconfig"
	"itsm-backend/ent/tenant"
	"itsm-backend/internal/commandbus"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// newGuardedClient 打开带生产安全拦截器（软删除 + 租户写护栏）的测试客户端：
// 命令消费与初始化状态都必须在真实护栏下验证，否则跨租户写缺陷会被漏掉。
func newGuardedClient(t *testing.T) *ent.Client {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:tenantinit?mode=memory&cache=shared&_fk=1")
	database.RegisterSecurityInterceptors(client, "off")
	return client
}

type recordingInstaller struct {
	tenantIDs    []int
	systemScopes []bool
	versions     []string
	err          error
}

func (r *recordingInstaller) ProvisionTenant(ctx context.Context, tenantID int, templateVersion string) error {
	r.tenantIDs = append(r.tenantIDs, tenantID)
	r.systemScopes = append(r.systemScopes, tenantctx.IsSystemBypass(ctx))
	r.versions = append(r.versions, templateVersion)
	return r.err
}

func bootstrapCommand(tenantID int) *ent.OperationalCommand {
	return &ent.OperationalCommand{
		TenantID:      tenantID,
		CommandType:   commandbus.CommandTenantBootstrapInstall,
		AggregateType: "tenant",
		AggregateID:   tenantID,
		Status:        commandbus.StatusPending,
	}
}

// 命令消费者不得信任 payload 自报身份：必须按命令租户重新加载权威租户，
// 缺失/已删除/平台租户都不能触发安装，失败要上抛让 outbox 记录可观察失败。
func TestProvisioningCommandHandlerInstallsWithReloadedTenant(t *testing.T) {
	client := newGuardedClient(t)
	defer client.Close()
	ctx := context.Background()
	installer := &recordingInstaller{}
	handler := NewProvisioningCommandHandler(client, installer, "1.0.0", zaptest.NewLogger(t).Sugar())

	target, err := client.Tenant.Create().
		SetName("Provisioned").SetCode("provisioned").SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)

	require.NoError(t, handler.Handle(ctx, bootstrapCommand(target.ID)))
	require.Equal(t, []int{target.ID}, installer.tenantIDs)
	assert.Equal(t, []bool{true}, installer.systemScopes, "安装必须在显式 system context 下执行")
	assert.Equal(t, []string{"1.0.0"}, installer.versions)
}

func TestProvisioningCommandHandlerRejectsUnsafeCommands(t *testing.T) {
	client := newGuardedClient(t)
	defer client.Close()
	ctx := context.Background()
	installer := &recordingInstaller{}
	handler := NewProvisioningCommandHandler(client, installer, "1.0.0", zaptest.NewLogger(t).Sugar())

	platform, err := client.Tenant.Create().SetName("Platform").SetCode(protectedSystemTenantCode).
		SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)
	deleted, err := client.Tenant.Create().SetName("Gone").SetCode("gone").
		SetType(tenant.TypeSaasCustomer).SetStatus("deleted").Save(ctx)
	require.NoError(t, err)

	t.Run("unknown tenant fails closed", func(t *testing.T) {
		require.ErrorContains(t, handler.Handle(ctx, bootstrapCommand(999999)), "load tenant 999999")
	})
	t.Run("deleted tenant is refused", func(t *testing.T) {
		require.ErrorContains(t, handler.Handle(ctx, bootstrapCommand(deleted.ID)), "refusing baseline install")
	})
	t.Run("platform tenant is skipped", func(t *testing.T) {
		require.NoError(t, handler.Handle(ctx, bootstrapCommand(platform.ID)))
	})
	t.Run("malformed commands are rejected", func(t *testing.T) {
		require.Error(t, handler.Handle(ctx, nil))
		require.Error(t, handler.Handle(ctx, &ent.OperationalCommand{
			TenantID: 1, CommandType: "other", AggregateType: "tenant", AggregateID: 1,
		}))
		require.Error(t, handler.Handle(ctx, &ent.OperationalCommand{
			TenantID: 1, CommandType: commandbus.CommandTenantBootstrapInstall,
			AggregateType: "ticket", AggregateID: 1,
		}))
		require.Error(t, handler.Handle(ctx, &ent.OperationalCommand{
			TenantID: 1, CommandType: commandbus.CommandTenantBootstrapInstall,
			AggregateType: "tenant", AggregateID: 2,
		}))
	})
	require.Empty(t, installer.tenantIDs, "拒绝/跳过的命令都不允许触发安装")

	t.Run("installer failure propagates to the outbox", func(t *testing.T) {
		failing := &recordingInstaller{err: errors.New("injected install failure")}
		handler := NewProvisioningCommandHandler(client, failing, "1.0.0", zaptest.NewLogger(t).Sugar())
		target, err := client.Tenant.Create().SetName("Failing").SetCode("failing").
			SetType(tenant.TypeSaasCustomer).Save(ctx)
		require.NoError(t, err)
		require.ErrorContains(t, handler.Handle(ctx, bootstrapCommand(target.ID)), "injected install failure")
		require.Equal(t, []int{target.ID}, failing.tenantIDs)
	})
}

// 状态接口必须报告真实安装结果，而不是命令表面成功：
// 逐组件只读验证决定 ready，命令状态与历史版本标记单独如实呈现。
func TestInitializationServiceStatusReportsRealState(t *testing.T) {
	client := newGuardedClient(t)
	defer client.Close()
	ctx := context.Background()
	target, err := client.Tenant.Create().SetName("Status Tenant").SetCode("status-tenant").
		SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)

	verifier := func(context.Context, int) ([]ComponentVerification, error) {
		return []ComponentVerification{
			{Component: "identity-rbac", Verified: true},
			{Component: "workflow-core", Verified: false, Error: "verify workflow templates: missing change_request"},
		}, nil
	}
	svc := NewInitializationService(client, verifier, "1.0.0", zaptest.NewLogger(t).Sugar())

	status, err := svc.Status(ctx, target.ID)
	require.NoError(t, err)
	assert.False(t, status.Ready)
	assert.Equal(t, "none", status.CommandStatus)
	assert.Equal(t, "1.0.0", status.TemplateVersion)
	require.Len(t, status.Components, 2)
	assert.Equal(t, "workflow-core", status.Components[1].Component)
	assert.Equal(t, "verify workflow templates: missing change_request", status.Components[1].Error)

	_, err = client.OperationalCommand.Create().
		SetTenantID(target.ID).SetCommandType(commandbus.CommandTenantBootstrapInstall).
		SetAggregateType("tenant").SetAggregateID(target.ID).
		SetIdempotencyKey(fmt.Sprintf("tenant:%d:bootstrap:install:initial", target.ID)).
		SetStatus(commandbus.StatusDeadLetter).SetAttempt(5).
		SetLastError("install failed").SetAvailableAt(time.Now()).Save(ctx)
	require.NoError(t, err)
	_, err = client.SystemConfig.Create().
		SetKey(fmt.Sprintf("tenant.bootstrap.version.%d", target.ID)).SetValue("1.0.0").
		SetCategory("bootstrap").SetCreatedBy("system").SetTenantID(target.ID).Save(ctx)
	require.NoError(t, err)

	status, err = svc.Status(ctx, target.ID)
	require.NoError(t, err)
	assert.False(t, status.Ready, "dead_letter 命令不能掩盖未就绪的基线")
	assert.Equal(t, commandbus.StatusDeadLetter, status.CommandStatus)
	assert.Equal(t, 5, status.CommandAttempts)
	assert.Equal(t, "install failed", status.CommandError)
	assert.Equal(t, "1.0.0", status.RecordedVersion)
	require.NotNil(t, status.RecordedAt)

	commands, err := client.OperationalCommand.Query().Where(operationalcommand.StatusEQ(commandbus.StatusDeadLetter)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, commands)
	markers, err := client.SystemConfig.Query().Where(systemconfig.TenantIDEQ(target.ID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, markers)

	// 状态接口只回报所请求租户的验证结果，不能把别的租户的组件状态带出来。
	other, err := client.Tenant.Create().SetName("Other").SetCode("other-status").
		SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)
	perTenant := func(_ context.Context, tenantID int) ([]ComponentVerification, error) {
		if tenantID == other.ID {
			return []ComponentVerification{{Component: "identity-rbac", Verified: true}}, nil
		}
		return []ComponentVerification{{Component: "identity-rbac", Verified: false, Error: "missing"}}, nil
	}
	scoped := NewInitializationService(client, perTenant, "1.0.0", zaptest.NewLogger(t).Sugar())
	otherStatus, err := scoped.Status(ctx, other.ID)
	require.NoError(t, err)
	assert.True(t, otherStatus.Ready)
	require.Len(t, otherStatus.Components, 1)
	assert.Equal(t, "none", otherStatus.CommandStatus, "别的租户的命令状态不得串到本租户")

	_, err = svc.Status(ctx, 0)
	require.Error(t, err, "缺 tenant id 必须 fail closed")
	_, err = NewInitializationService(client, nil, "1.0.0", zaptest.NewLogger(t).Sugar()).Status(ctx, target.ID)
	require.Error(t, err, "未配置验证器必须报错，而不是返回空成功")
}
