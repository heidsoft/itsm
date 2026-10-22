package marketplace

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"itsm-backend/connector"
	"itsm-backend/ent"
	"itsm-backend/ent/marketplaceitem"
	"itsm-backend/ent/tenantinstallation"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/schema"
	"go.uber.org/zap/zaptest"
)

// stubConnector 是可控成败的连接器桩：只有带 token 才能 Init 成功。
// 用它把「安装记录」与「运行态」两条线分开断言——测试要证明的是两者必须一致，
// 而不是连接器本身能发消息。
type stubConnector struct {
	name string
	// rejectEmptyToken=false 时 Init 不校验凭据，用于验证成功安装路径。
	rejectEmptyToken bool
	// initErr 非空时 Init 直接失败，用于模拟端点不可达这类外部故障。
	initErr error
}

func (s *stubConnector) Manifest() connector.Manifest {
	return connector.Manifest{
		Name:                s.name,
		Version:             "1.0.0",
		Title:               s.name,
		Type:                connector.TypeIM,
		RequiredPermissions: []string{"connector:write"},
	}
}

func (s *stubConnector) Init(_ context.Context, cfg connector.Config) error {
	if s.initErr != nil {
		return s.initErr
	}
	if s.rejectEmptyToken && cfg.Credentials["token"] == "" {
		return errors.New("credentials.token is required")
	}
	return nil
}

func (s *stubConnector) Send(context.Context, *connector.Message) error { return nil }

func (s *stubConnector) HealthCheck(context.Context) connector.HealthStatus {
	return connector.HealthStatus{OK: true, CheckedAt: time.Now()}
}

func (s *stubConnector) Close() error { return nil }

// newClosureFixture 建一个带独立 registry 的市场服务：注册名为 name 的桩连接器，
// 并把它注入 Service，模拟生产装配（bootstrap 里 marketplaceSvc.SetConnectorManager）。
func newClosureFixture(t *testing.T, name string, stub *stubConnector) (*Service, *ent.Client, *connector.Manager) {
	t.Helper()

	drv, err := sql.Open(dialect.SQLite, "file:marketplace_closure_"+name+"?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	client := ent.NewClient(ent.Driver(drv))
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Schema.Create(context.Background(), schema.WithGlobalUniqueID(false)); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	registry := connector.NewRegistry()
	stub.name = name
	registry.Register(func() connector.Connector { return stub })

	manager := connector.NewManager(registry, zaptest.NewLogger(t).Sugar())
	svc := NewService(client, zaptest.NewLogger(t).Sugar())
	svc.SetConnectorManager(manager)
	return svc, client, manager
}

func createConnectorItem(t *testing.T, client *ent.Client, name string) *ent.MarketplaceItem {
	t.Helper()
	item, err := client.MarketplaceItem.Create().
		SetName(name).
		SetType(marketplaceitem.TypeConnector).
		SetTitle(name + " connector").
		SetProvider("Open Source").
		SetLatestVersion("1.0.0").
		SetStatus(marketplaceitem.StatusPublished).
		Save(context.Background())
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	return item
}

// TestInstallConnector_ProvisionFailureMarksInstallationFailed 锁定「不再伪装成功」：
// 缺凭据导致 provision 失败时，安装记录必须是 failed + error_message，
// 而不是旧行为的 active（市场页显示已安装，实际一条消息都发不出去）。
func TestInstallConnector_ProvisionFailureMarksInstallationFailed(t *testing.T) {
	svc, client, manager := newClosureFixture(t, "stubfail", &stubConnector{rejectEmptyToken: true})
	ctx := context.Background()
	item := createConnectorItem(t, client, "stubfail")

	installation, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	if !errors.Is(err, ErrMarketplaceConnectorNotReady) {
		t.Fatalf("expected ErrMarketplaceConnectorNotReady, got %v", err)
	}
	if installation == nil {
		t.Fatal("installation record must be returned so the caller can show why it failed")
	}
	if installation.Status != tenantinstallation.StatusFailed {
		t.Fatalf("expected status failed, got %s", installation.Status)
	}
	if !strings.Contains(installation.ErrorMessage, "credentials.token") {
		t.Fatalf("error_message must carry the real cause, got %q", installation.ErrorMessage)
	}

	if _, ok := manager.Get(1, "stubfail"); ok {
		t.Fatal("a failed provisioning must not leave a live connector instance")
	}
}

// TestInstallConnector_SuccessActivates 成功路径：运行态起来了才置 active。
func TestInstallConnector_SuccessActivates(t *testing.T) {
	svc, client, manager := newClosureFixture(t, "stubok", &stubConnector{})
	ctx := context.Background()
	item := createConnectorItem(t, client, "stubok")

	installation, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if installation.Status != tenantinstallation.StatusActive {
		t.Fatalf("expected active, got %s", installation.Status)
	}
	if installation.ErrorMessage != "" {
		t.Fatalf("active installation must not carry error message, got %q", installation.ErrorMessage)
	}
	if _, ok := manager.Get(1, "stubok"); !ok {
		t.Fatal("active installation must have a live connector instance")
	}
}

// TestInstallConnector_ReinstallRetriesFailed 把「安装」当重试入口：
// 上次 failed 的记录不应被幂等短路，而应重新 provision 并转 active。
func TestInstallConnector_ReinstallRetriesFailed(t *testing.T) {
	stub := &stubConnector{rejectEmptyToken: true}
	svc, client, manager := newClosureFixture(t, "stubretry", stub)
	ctx := context.Background()
	item := createConnectorItem(t, client, "stubretry")

	failed, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	if !errors.Is(err, ErrMarketplaceConnectorNotReady) {
		t.Fatalf("expected first install to fail, got %v", err)
	}

	// 桩改成「凭据不再必需」，模拟运维已经把凭据配好
	stub.rejectEmptyToken = false

	retried, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	if err != nil {
		t.Fatalf("retry install: %v", err)
	}
	if retried.ID != failed.ID {
		t.Fatalf("retry must reuse the same installation record, got %d vs %d", retried.ID, failed.ID)
	}
	if retried.Status != tenantinstallation.StatusActive {
		t.Fatalf("expected active after retry, got %s", retried.Status)
	}
	if _, ok := manager.Get(1, "stubretry"); !ok {
		t.Fatal("retried installation must have a live connector instance")
	}
}

// TestUpdateConfig_PersistsCredentialsAndActivatesFailedInstallation 覆盖生产主路径：
// 安装（失败）→ 填凭据（PUT config）→ 运行态起来、状态转 active。
func TestUpdateConfig_PersistsCredentialsAndActivatesFailedInstallation(t *testing.T) {
	svc, client, manager := newClosureFixture(t, "stubcfg", &stubConnector{rejectEmptyToken: true})
	ctx := context.Background()
	item := createConnectorItem(t, client, "stubcfg")

	if _, err := svc.InstallItem(ctx, 1, item.ID, "1001"); !errors.Is(err, ErrMarketplaceConnectorNotReady) {
		t.Fatalf("expected failed install, got %v", err)
	}

	updated, err := svc.UpdateInstallationConfig(ctx, 1, item.ID, map[string]interface{}{
		"credentials": map[string]interface{}{"token": "real-token"},
	})
	if err != nil {
		t.Fatalf("update config: %v", err)
	}
	if updated.Status != tenantinstallation.StatusActive {
		t.Fatalf("expected active after config filled, got %s", updated.Status)
	}
	if updated.ErrorMessage != "" {
		t.Fatalf("error message must be cleared on success, got %q", updated.ErrorMessage)
	}
	if _, ok := manager.Get(1, "stubcfg"); !ok {
		t.Fatal("connector must be running after credentials are provided")
	}
}

// TestUpdateConfig_ProvisionFailureKeepsPreviousConfig 是「DB 与运行态不许分裂」的核心锁：
// 新配置让 Init 失败时，DB 必须保留上一份可用配置，且运行态仍是旧凭据。
func TestUpdateConfig_ProvisionFailureKeepsPreviousConfig(t *testing.T) {
	stub := &stubConnector{rejectEmptyToken: true}
	svc, client, manager := newClosureFixture(t, "stubkeep", stub)
	ctx := context.Background()
	item := createConnectorItem(t, client, "stubkeep")

	// 先安装（此时无凭据，记录停 failed），再用一份可用凭据把它配置起来
	if _, err := svc.InstallItem(ctx, 1, item.ID, "1001"); !errors.Is(err, ErrMarketplaceConnectorNotReady) {
		t.Fatalf("expected install to fail without credentials, got %v", err)
	}
	if _, err := svc.UpdateInstallationConfig(ctx, 1, item.ID, map[string]interface{}{
		"credentials": map[string]interface{}{"token": "old-token"},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	if _, ok := manager.Get(1, "stubkeep"); !ok {
		t.Fatal("seed config must bring the connector up")
	}

	// 让下一次 Init 必然失败，模拟新凭据被第三方拒绝
	stub.initErr = errors.New("upstream rejected token")

	_, err := svc.UpdateInstallationConfig(ctx, 1, item.ID, map[string]interface{}{
		"credentials": map[string]interface{}{"token": "bad-token"},
	})
	if !errors.Is(err, ErrMarketplaceConnectorNotReady) {
		t.Fatalf("expected ErrMarketplaceConnectorNotReady, got %v", err)
	}

	installation, err := svc.GetInstallation(ctx, 1, item.ID)
	if err != nil {
		t.Fatalf("get installation: %v", err)
	}
	creds, _ := installation.Config["credentials"].(map[string]interface{})
	if creds["token"] != "old-token" {
		t.Fatalf("db config must keep the last working credentials, got %v", creds["token"])
	}
	if installation.ErrorMessage == "" {
		t.Fatal("failure reason must be recorded for the operator")
	}

	// 运行态仍是旧实例（Init 失败时 Manager 不替换实例）
	if _, ok := manager.Get(1, "stubkeep"); !ok {
		t.Fatal("previous connector instance must survive a failed re-provision")
	}
}

// TestReinstallAfterUninstall_StillGoesThroughProvisioning 锁住历史记录复用路径：
// 卸载后再装会复用同一条 tenant_installations 行，旧实现直接置 active 且不
// provision。这里让凭据缺失，重装必须如实变成 failed，而不是假装装好了。
func TestReinstallAfterUninstall_StillGoesThroughProvisioning(t *testing.T) {
	svc, client, manager := newClosureFixture(t, "stubhistory", &stubConnector{rejectEmptyToken: true})
	ctx := context.Background()
	item := createConnectorItem(t, client, "stubhistory")

	// 造一条已卸载的历史记录
	if _, err := client.TenantInstallation.Create().
		SetTenantID(1).
		SetItemID(item.ID).
		SetInstalledVersion("0.9.0").
		SetStatus(tenantinstallation.StatusUninstalled).
		SetInstalledBy("1001").
		Save(ctx); err != nil {
		t.Fatalf("seed history: %v", err)
	}

	installation, err := svc.InstallItem(ctx, 1, item.ID, "2002")
	if !errors.Is(err, ErrMarketplaceConnectorNotReady) {
		t.Fatalf("expected ErrMarketplaceConnectorNotReady, got %v", err)
	}
	if installation.Status != tenantinstallation.StatusFailed {
		t.Fatalf("reactivated installation must reflect provisioning failure, got %s", installation.Status)
	}
	if _, ok := manager.Get(1, "stubhistory"); ok {
		t.Fatal("failed reinstall must not leave a live connector instance")
	}
}

// TestUninstall_RevokesRuntimeAndMarksUninstalled 正常卸载：运行态消失 + 状态 uninstalled。
func TestUninstall_RevokesRuntimeAndMarksUninstalled(t *testing.T) {
	svc, client, manager := newClosureFixture(t, "stubuninstall", &stubConnector{})
	ctx := context.Background()
	item := createConnectorItem(t, client, "stubuninstall")

	if _, err := svc.InstallItem(ctx, 1, item.ID, "1001"); err != nil {
		t.Fatalf("install: %v", err)
	}
	if err := svc.UninstallItem(ctx, 1, item.ID); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if _, ok := manager.Get(1, "stubuninstall"); ok {
		t.Fatal("connector instance must be gone after uninstall")
	}

	installation, err := svc.GetInstallation(ctx, 1, item.ID)
	if !errors.Is(err, ErrMarketplaceInstallationAbsent) {
		t.Fatalf("expected installation absent after uninstall, got %v (installation=%v)", err, installation)
	}
}

// TestUninstall_FailsClosedWhenInstanceSurvives 锁住「假卸载」：
// Revoke 以 (tenant,name,provider) 为键，provider 不一致时实例删不掉。
// 此时安装记录绝不能变成 uninstalled——否则界面显示已卸载，连接器还在用活凭据收发消息。
func TestUninstall_FailsClosedWhenInstanceSurvives(t *testing.T) {
	svc, client, manager := newClosureFixture(t, "stubresidual", &stubConnector{})
	ctx := context.Background()
	item := createConnectorItem(t, client, "stubresidual")

	// 用不同 provider 注册同名实例，复现「键不匹配导致 Revoke 落空」的历史数据形态
	if err := manager.Provision(ctx, connector.Config{
		TenantID: 1,
		Name:     "stubresidual",
		Provider: "legacy-provider",
		Enabled:  true,
	}); err != nil {
		t.Fatalf("provision residual instance: %v", err)
	}
	if _, err := svc.InstallItem(ctx, 1, item.ID, "1001"); err != nil {
		t.Fatalf("install: %v", err)
	}

	err := svc.UninstallItem(ctx, 1, item.ID)
	if !errors.Is(err, ErrMarketplaceConnectorStillRunning) {
		t.Fatalf("expected ErrMarketplaceConnectorStillRunning, got %v", err)
	}

	installation, err := svc.GetInstallation(ctx, 1, item.ID)
	if err != nil {
		t.Fatalf("installation must remain available for retry: %v", err)
	}
	if installation.Status == tenantinstallation.StatusUninstalled {
		t.Fatal("installation must not be marked uninstalled while a connector instance is still running")
	}
}
