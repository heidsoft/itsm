package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processdefinition"
)

func newTemplateUpgradeFixture(t *testing.T) (*ent.Client, int) {
	t.Helper()
	client := enttest.Open(t, dialect.SQLite, fmt.Sprintf("file:template_upgrade_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })

	unique := time.Now().UnixNano()
	tenant, err := client.Tenant.Create().
		SetName("Template Upgrade Tenant").
		SetCode(fmt.Sprintf("TPLU-%d", unique)).
		SetDomain(fmt.Sprintf("tplu-%d.example.com", unique)).
		SetStatus("active").
		Save(context.Background())
	require.NoError(t, err)
	return client, tenant.ID
}

func seedLatestTemplateDefinition(t *testing.T, client *ent.Client, tenantID int, key, version, xml string) *ent.ProcessDefinition {
	t.Helper()
	ctx := context.Background()
	unique := time.Now().UnixNano()
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID(fmt.Sprintf("seed-%s-%s-%d", key, version, unique)).
		SetDeploymentName("seed " + version).
		SetDeploymentTime(time.Now()).
		SetTenantID(tenantID).
		SetDeployedBy("test").
		Save(ctx)
	require.NoError(t, err)

	definition, err := client.ProcessDefinition.Create().
		SetKey(key).
		SetName("seed " + version).
		SetVersion(version).
		SetBpmnXML([]byte(xml)).
		SetIsActive(true).
		SetIsLatest(true).
		SetTenantID(tenantID).
		SetDeploymentID(deployment.ID).
		SetDeployedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	return definition
}

func latestPublishedDefinition(t *testing.T, client *ent.Client, tenantID int, key string) *ent.ProcessDefinition {
	t.Helper()
	definition, err := client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(key),
			processdefinition.TenantID(tenantID),
			processdefinition.IsLatest(true),
		).
		Only(context.Background())
	require.NoError(t, err)
	return definition
}

func TestDeclaredTemplateVersionReadsProcessMetaData(t *testing.T) {
	data, err := bpmnTemplates.ReadFile("bpmn/service_request_flow.bpmn")
	require.NoError(t, err)
	assert.Equal(t, "2.1.0", declaredTemplateVersion(data, "1.0.0"))
	assert.Equal(t, "1.0.0", declaredTemplateVersion([]byte(`<definitions/>`), "1.0.0"))
}

func TestLoadAndDeployTemplates_FreshTenantDeploysDeclaredVersion(t *testing.T) {
	client, tenantID := newTemplateUpgradeFixture(t)
	svc := NewBPMNTemplateService(client)

	_, err := svc.LoadAndDeployTemplates(context.Background(), tenantID)
	require.NoError(t, err)

	definition := latestPublishedDefinition(t, client, tenantID, "service_request_flow")
	assert.Equal(t, "2.1.0", definition.Version, "新租户应按模板 metaData 声明版本部署")

	count, err := client.ProcessDefinition.Query().
		Where(processdefinition.Key("service_request_flow"), processdefinition.TenantID(tenantID)).
		Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestLoadAndDeployTemplates_UpgradesExistingTenant(t *testing.T) {
	client, tenantID := newTemplateUpgradeFixture(t)
	seedLatestTemplateDefinition(t, client, tenantID, "service_request_flow", "1.0.0", `<definitions><!-- legacy --></definitions>`)
	svc := NewBPMNTemplateService(client)

	_, err := svc.LoadAndDeployTemplates(context.Background(), tenantID)
	require.NoError(t, err)

	latest := latestPublishedDefinition(t, client, tenantID, "service_request_flow")
	assert.Equal(t, "2.1.0", latest.Version, "已有 1.0.0 定义的租户必须升级到模板声明版本")

	latestCount, err := client.ProcessDefinition.Query().
		Where(
			processdefinition.Key("service_request_flow"),
			processdefinition.TenantID(tenantID),
			processdefinition.IsLatest(true),
		).
		Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, latestCount, "同一 key 只能有一个最新版本")

	legacy, err := client.ProcessDefinition.Query().
		Where(
			processdefinition.Key("service_request_flow"),
			processdefinition.TenantID(tenantID),
			processdefinition.Version("1.0.0"),
		).
		Only(context.Background())
	require.NoError(t, err)
	assert.False(t, legacy.IsLatest)
}

func TestLoadAndDeployTemplates_IsIdempotent(t *testing.T) {
	client, tenantID := newTemplateUpgradeFixture(t)
	seedLatestTemplateDefinition(t, client, tenantID, "service_request_flow", "1.0.0", `<definitions><!-- legacy --></definitions>`)
	svc := NewBPMNTemplateService(client)

	_, err := svc.LoadAndDeployTemplates(context.Background(), tenantID)
	require.NoError(t, err)
	_, err = svc.LoadAndDeployTemplates(context.Background(), tenantID)
	require.NoError(t, err)

	count, err := client.ProcessDefinition.Query().
		Where(processdefinition.Key("service_request_flow"), processdefinition.TenantID(tenantID)).
		Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, count, "重复启动不得重复创建同版本定义")
}

func TestLoadAndDeployTemplates_DoesNotDowngrade(t *testing.T) {
	client, tenantID := newTemplateUpgradeFixture(t)
	seedLatestTemplateDefinition(t, client, tenantID, "service_request_flow", "9.9.9", `<definitions><!-- custom --></definitions>`)
	svc := NewBPMNTemplateService(client)

	_, err := svc.LoadAndDeployTemplates(context.Background(), tenantID)
	require.NoError(t, err)

	latest := latestPublishedDefinition(t, client, tenantID, "service_request_flow")
	assert.Equal(t, "9.9.9", latest.Version, "内置模板版本低于线上时不得自动降级")

	count, err := client.ProcessDefinition.Query().
		Where(processdefinition.Key("service_request_flow"), processdefinition.TenantID(tenantID)).
		Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestLoadAndDeployTemplates_ContentChangeWithoutVersionBumpDeploysPatch(t *testing.T) {
	client, tenantID := newTemplateUpgradeFixture(t)
	seedLatestTemplateDefinition(t, client, tenantID, "service_request_flow", "2.1.0", `<definitions><!-- stale 2.1.0 --></definitions>`)
	svc := NewBPMNTemplateService(client)

	_, err := svc.LoadAndDeployTemplates(context.Background(), tenantID)
	require.NoError(t, err)

	latest := latestPublishedDefinition(t, client, tenantID, "service_request_flow")
	assert.Equal(t, "2.1.1", latest.Version, "同版本内容变化应按补丁版本部署，避免与线上同版本唯一约束冲突")
}

// TestTranslateConstraintAsAlreadyDeployed 锁住 5.13 修复的核心契约：部署事务
// 内的约束冲突必须统一翻译成 sentinel，供外层按「内部事务已回滚的幂等命中」
// 处理；普通错误与 nil 原样透传，不得被误吞。
func TestTranslateConstraintAsAlreadyDeployed(t *testing.T) {
	constraint := &ent.ConstraintError{}
	require.True(t, ent.IsConstraintError(constraint))

	cases := map[string]error{
		"bare_constraint":    constraint,
		"pkg_errors_wrapped": pkgerrors.Wrap(constraint, "保存流程定义失败"),
		"fmt_wrapped":        fmt.Errorf("更新流程定义失败: %w", constraint),
	}
	for name, err := range cases {
		t.Run(name, func(t *testing.T) {
			require.ErrorIs(t, translateConstraintAsAlreadyDeployed(err), ErrTemplateAlreadyDeployed,
				"约束冲突必须翻译成幂等 sentinel")
		})
	}

	plain := pkgerrors.New("lint 门禁拦截")
	require.Equal(t, plain, translateConstraintAsAlreadyDeployed(plain), "普通错误必须原样透传")
	require.NoError(t, translateConstraintAsAlreadyDeployed(nil))
}

// TestLoadAndDeployTemplates_TreatsSentinelAsIdempotent 是 5.13 的回归锁：
// 旧写法在吞错点手摸 ent.IsConstraintError，对 sentinel 会判 false，于是把
// 并发幂等命中误记为部署失败并让整批返回 error —— 旧实现下本用例必红。
// 用包内钩子注入部署结果，避免依赖 SQLite 并发时序造成 flaky：顺序调用时
// 「先查后插」会挡住重复插入，根本撞不到唯一约束。
func TestLoadAndDeployTemplates_TreatsSentinelAsIdempotent(t *testing.T) {
	client, tenantID := newTemplateUpgradeFixture(t)
	svc := NewBPMNTemplateService(client)

	svc.deployTemplateFn = func(_ context.Context, _ *TemplateInfo, _ int, _ string, _ []byte) error {
		return fmt.Errorf("另一个实例已部署同一版本: %w", ErrTemplateAlreadyDeployed)
	}

	deployed, err := svc.LoadAndDeployTemplates(context.Background(), tenantID)
	require.NoError(t, err, "并发幂等命中不得让整批部署失败")
	templates, err := svc.listTemplates()
	require.NoError(t, err)
	require.Len(t, deployed, len(templates), "命中 sentinel 的模板应计入 deployed 而非 deployFailures")
}

// TestLoadAndDeployTemplates_DoesNotSwallowPlainErrors 防止修复被做成「任何
// 错误都吞」：只有 sentinel 才是幂等命中，lint 拦截等真实失败仍须上报。
func TestLoadAndDeployTemplates_DoesNotSwallowPlainErrors(t *testing.T) {
	client, tenantID := newTemplateUpgradeFixture(t)
	svc := NewBPMNTemplateService(client)

	svc.deployTemplateFn = func(_ context.Context, _ *TemplateInfo, _ int, _ string, _ []byte) error {
		return pkgerrors.New("lint 门禁拦截")
	}

	deployed, err := svc.LoadAndDeployTemplates(context.Background(), tenantID)
	require.Error(t, err, "非 sentinel 的部署失败必须让整批返回错误")
	require.Empty(t, deployed, "真实失败的模板不得计入 deployed")
}
