package service

import (
	"context"
	"fmt"
	"testing"

	"entgo.io/ent/dialect"
	"strconv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent/enttest"
)

type versionFixture struct {
	svc      *BPMNVersionService
	client   interface{ Close() error }
	tenantID int
}

func setupVersionFixture(t *testing.T) versionFixture {
	t.Helper()
	client := enttest.Open(t, dialect.SQLite, testDSN())
	logger := zaptest.NewLogger(t).Sugar()
	svc := &BPMNVersionService{client: client, logger: logger}
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Ver Tenant").SetCode("ver-test").SetDomain("ver.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// changelog.created_by 有指向 users 的外键，测试用 CreatedBy="1" 需要真实用户
	_, err = client.User.Create().
		SetUsername("version-admin").SetEmail("version-admin@test.com").SetName("版本管理员").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return versionFixture{svc: svc, client: client, tenantID: tenant.ID}
}

func seedDefinition(t *testing.T, f versionFixture, key, version string) int {
	t.Helper()
	ctx := context.Background()
	deployment, err := f.svc.client.ProcessDeployment.Create().
		SetDeploymentID(fmt.Sprintf("DEP-%s-%s", key, version)).
		SetDeploymentName(fmt.Sprintf("%s v%s", key, version)).
		SetIsActive(true).SetTenantID(f.tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := f.svc.client.ProcessDefinition.Create().
		SetKey(key).SetName(fmt.Sprintf("%s v%s", key, version)).
		SetVersion(version).
		SetIsLatest(version == "1").SetIsActive(version == "1").
		SetBpmnXML([]byte("<definitions><process id=\"p1\" name=\"proc\"></process></definitions>")).
		SetDeploymentID(deployment.ID).SetTenantID(f.tenantID).
		Save(ctx)
	require.NoError(t, err)
	return def.ID
}

func TestBPMNVersion_CreateVersion_AutoIncrements(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	seedDefinition(t, f, "order_flow", "1")

	got, err := f.svc.CreateVersion(ctx, &CreateVersionRequest{
		ProcessDefinitionKey: "order_flow",
		Name:                 "Order Flow V2",
		Description:          "second version",
		BPMNXML:              "<definitions><process id=\"p2\" name=\"proc2\"><userTask id=\"t1\" name=\"task1\"/></process></definitions>",
		ChangeLog:            "added task1",
		TenantID:             f.tenantID,
		CreatedBy:            "1",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, got.Version)
	assert.Equal(t, "order_flow", got.ProcessDefinitionKey)
	assert.Equal(t, "Order Flow V2", got.Name)
	assert.False(t, got.IsActive, "新版本默认不激活")
	assert.False(t, got.IsActive)

	got2, err := f.svc.CreateVersion(ctx, &CreateVersionRequest{
		ProcessDefinitionKey: "order_flow",
		Name:                 "Order Flow V3",
		BPMNXML:              "<definitions/>",
		TenantID:             f.tenantID,
		CreatedBy:            "1",
	})
	require.NoError(t, err)
	assert.Equal(t, 3, got2.Version)
}

func TestBPMNVersion_CreateVersion_NoExistingDefinitionFails(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	_, err := f.svc.CreateVersion(ctx, &CreateVersionRequest{
		ProcessDefinitionKey: "nonexistent_key",
		Name:                 "First",
		BPMNXML:              "<definitions/>",
		TenantID:             f.tenantID,
	})
	require.Error(t, err, "没有已有定义时 CreateVersion 应失败")
	assert.Contains(t, err.Error(), "获取当前版本失败")
}

func TestBPMNVersion_GetVersion(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	defID := seedDefinition(t, f, "get_test", "1")
	_ = defID

	got, err := f.svc.GetVersion(ctx, "get_test", 1, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, got.Version)
	assert.Equal(t, "get_test", got.ProcessDefinitionKey)
	assert.Equal(t, f.tenantID, got.TenantID)
}

func TestBPMNVersion_GetVersion_NotFound(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	_, err := f.svc.GetVersion(ctx, "no_such_key", 1, f.tenantID)
	require.Error(t, err)
}

func TestBPMNVersion_GetVersion_TenantIsolation(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	seedDefinition(t, f, "tenant_test", "1")

	_, err := f.svc.GetVersion(ctx, "tenant_test", 1, f.tenantID+999)
	require.Error(t, err, "跨租户不应能读取版本")
}

func TestBPMNVersion_ListVersions(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	seedDefinition(t, f, "list_test", "1")

	for i := 0; i < 2; i++ {
		_, err := f.svc.CreateVersion(ctx, &CreateVersionRequest{
			ProcessDefinitionKey: "list_test",
			Name:                 fmt.Sprintf("list_test v%d", i+2),
			BPMNXML:              fmt.Sprintf("<definitions id=\"%d\"/>", i+2),
			TenantID:             f.tenantID,
			CreatedBy:            "1",
		})
		require.NoError(t, err)
	}

	versions, err := f.svc.ListVersions(ctx, "list_test", f.tenantID)
	require.NoError(t, err)
	assert.Len(t, versions, 3)

	assert.Greater(t, versions[0].Version, versions[1].Version, "应按版本号降序排列")
	assert.Greater(t, versions[1].Version, versions[2].Version)
}

func TestBPMNVersion_ListVersions_TenantIsolation(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	seedDefinition(t, f, "list_tenant", "1")

	versions, err := f.svc.ListVersions(ctx, "list_tenant", f.tenantID+999)
	require.NoError(t, err)
	assert.Empty(t, versions)
}

func TestBPMNVersion_ActivateVersion(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	seedDefinition(t, f, "activate_test", "1")

	for i := 0; i < 2; i++ {
		_, err := f.svc.CreateVersion(ctx, &CreateVersionRequest{
			ProcessDefinitionKey: "activate_test",
			Name:                 fmt.Sprintf("activate v%d", i+2),
			BPMNXML:              fmt.Sprintf("<definitions id=\"%d\"/>", i+2),
			TenantID:             f.tenantID,
			CreatedBy:            "1",
		})
		require.NoError(t, err)
	}

	err := f.svc.ActivateVersion(ctx, "activate_test", 3, f.tenantID)
	require.NoError(t, err)

	activated, err := f.svc.GetVersion(ctx, "activate_test", 3, f.tenantID)
	require.NoError(t, err)
	assert.True(t, activated.IsActive)

	v1, err := f.svc.GetVersion(ctx, "activate_test", 1, f.tenantID)
	require.NoError(t, err)
	assert.False(t, v1.IsActive, "旧版本应被停用")

	v2, err := f.svc.GetVersion(ctx, "activate_test", 2, f.tenantID)
	require.NoError(t, err)
	assert.False(t, v2.IsActive, "其他版本也应被停用")
}

func TestBPMNVersion_CompareVersions_Identical(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	xml := "<definitions><process id=\"p1\" name=\"proc\"></process></definitions>"
	seedDefinitionWithXML(t, f, "cmp_test", "1", xml)

	_, err := f.svc.CreateVersion(ctx, &CreateVersionRequest{
		ProcessDefinitionKey: "cmp_test",
		Name:                 "cmp v2",
		BPMNXML:              xml,
		TenantID:             f.tenantID,
		CreatedBy:            "1",
	})
	require.NoError(t, err)

	comparison, err := f.svc.CompareVersions(ctx, "cmp_test", 1, 2, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, "identical", comparison.Compatibility)
	assert.Empty(t, comparison.Changes)
}

func TestBPMNVersion_CompareVersions_TaskCountChanged(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	xml1 := "<definitions><process id=\"p1\" name=\"proc\"><userTask id=\"t1\" name=\"task1\"/></process></definitions>"
	xml2 := "<definitions><process id=\"p1\" name=\"proc\"><userTask id=\"t1\" name=\"task1\"/><userTask id=\"t2\" name=\"task2\"/></process></definitions>"

	seedDefinitionWithXML(t, f, "cmp_tasks", "1", xml1)

	_, err := f.svc.CreateVersion(ctx, &CreateVersionRequest{
		ProcessDefinitionKey: "cmp_tasks",
		Name:                 "cmp_tasks v2",
		BPMNXML:              xml2,
		TenantID:             f.tenantID,
		CreatedBy:            "1",
	})
	require.NoError(t, err)

	comparison, err := f.svc.CompareVersions(ctx, "cmp_tasks", 1, 2, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, "risky", comparison.Compatibility, "task 数量变化应为 high impact → risky")
	assert.NotEmpty(t, comparison.Changes)
	assert.Contains(t, comparison.Changes[0].Description, "Task count changed")
}

func TestBPMNVersion_CompareVersions_ContentChangedNoStructuralDiff(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	xml1 := "<definitions><process id=\"p1\" name=\"proc\"></process></definitions>"
	xml2 := "<definitions><process id=\"p1\" name=\"different_name\"></process></definitions>"

	seedDefinitionWithXML(t, f, "cmp_content", "1", xml1)

	_, err := f.svc.CreateVersion(ctx, &CreateVersionRequest{
		ProcessDefinitionKey: "cmp_content",
		Name:                 "cmp_content v2",
		BPMNXML:              xml2,
		TenantID:             f.tenantID,
		CreatedBy:            "1",
	})
	require.NoError(t, err)

	comparison, err := f.svc.CompareVersions(ctx, "cmp_content", 1, 2, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, "compatible", comparison.Compatibility, "内容变化但 task 数不变 → medium impact → compatible")
	assert.Len(t, comparison.Changes, 1)
	assert.Equal(t, "medium", comparison.Changes[0].Impact)
}

func TestBPMNVersion_RollbackToVersion(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	xml1 := "<definitions><process id=\"p1\" name=\"original\"></process></definitions>"
	seedDefinitionWithXML(t, f, "rollback_test", "1", xml1)

	for i := 0; i < 2; i++ {
		_, err := f.svc.CreateVersion(ctx, &CreateVersionRequest{
			ProcessDefinitionKey: "rollback_test",
			Name:                 fmt.Sprintf("rollback v%d", i+2),
			BPMNXML:              fmt.Sprintf("<definitions><process id=\"p%d\" name=\"v%d\"></process></definitions>", i+2, i+2),
			TenantID:             f.tenantID,
			CreatedBy:            "1",
		})
		require.NoError(t, err)
	}

	err := f.svc.RollbackToVersion(ctx, "rollback_test", 1, f.tenantID, "production bug")
	require.NoError(t, err)

	versions, err := f.svc.ListVersions(ctx, "rollback_test", f.tenantID)
	require.NoError(t, err)
	assert.Len(t, versions, 4, "回滚应创建新版本")

	latest := versions[0]
	assert.Equal(t, 4, latest.Version)
	assert.True(t, latest.IsActive, "回滚版本应被激活")
	assert.Contains(t, latest.BPMNXML, "original", "回滚版本应复制目标版本的 XML")
}

func TestBPMNVersion_RollbackToVersion_TargetNotExist(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	seedDefinition(t, f, "rollback_miss", "1")

	err := f.svc.RollbackToVersion(ctx, "rollback_miss", 99, f.tenantID, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "目标版本不存在")
}

func TestBPMNVersion_GetChangeLogsByProcessKey(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	seedDefinition(t, f, "changelog_test", "1")

	_, err := f.svc.CreateVersion(ctx, &CreateVersionRequest{
		ProcessDefinitionKey: "changelog_test",
		Name:                 "changelog v2",
		BPMNXML:              "<definitions id=\"2\"/>",
		ChangeLog:            "added new gateway",
		TenantID:             f.tenantID,
		CreatedBy:            "1",
	})
	require.NoError(t, err)

	logs, err := f.svc.GetChangeLogsByProcessKey(ctx, "changelog_test", f.tenantID)
	require.NoError(t, err)
	assert.NotEmpty(t, logs)
	assert.Equal(t, "added new gateway", logs[0].ChangeLog)
}

func TestBPMNVersion_GetChangeLogsByProcessKey_NotFound(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	_, err := f.svc.GetChangeLogsByProcessKey(ctx, "no_such_key", f.tenantID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "流程定义不存在")
}

func TestBPMNVersion_GetChangeLogsByProcessDefinitionID(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	seedDefinition(t, f, "changelog_id", "1")

	created, err := f.svc.CreateVersion(ctx, &CreateVersionRequest{
		ProcessDefinitionKey: "changelog_id",
		Name:                 "changelog_id v2",
		BPMNXML:              "<definitions id=\"2\"/>",
		ChangeLog:            "second version change",
		TenantID:             f.tenantID,
		CreatedBy:            "1",
	})
	require.NoError(t, err)

	// 变更日志记录在新建版本的定义 ID 上，而非原版本
	createdDefID, err := strconv.Atoi(created.ID)
	require.NoError(t, err)
	logs, err := f.svc.GetChangeLogsByProcessDefinitionID(ctx, createdDefID)
	require.NoError(t, err)
	assert.NotEmpty(t, logs)
}

func TestBPMNVersion_AssessCompatibility(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()

	tests := []struct {
		name            string
		changes         []ChangeDetail
		breakingChanges []string
		want            string
	}{
		{"no breaking, no changes", nil, nil, "identical"},
		{"breaking changes", []ChangeDetail{{Impact: "low"}}, []string{"removed field"}, "incompatible"},
		{"high impact", []ChangeDetail{{Impact: "high"}}, nil, "risky"},
		{"critical impact", []ChangeDetail{{Impact: "critical"}}, nil, "risky"},
		{"medium impact only", []ChangeDetail{{Impact: "medium"}}, nil, "compatible"},
		{"low impact only", []ChangeDetail{{Impact: "low"}}, nil, "compatible"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.svc.assessCompatibility(tt.changes, tt.breakingChanges)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBPMNVersion_CompareBPMNXML(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()

	t.Run("identical XML", func(t *testing.T) {
		xml := "<definitions><process id=\"p1\"/></definitions>"
		changes, breaking := f.svc.compareBPMNXML(xml, xml)
		assert.Empty(t, changes)
		assert.Empty(t, breaking)
	})

	t.Run("task count differs", func(t *testing.T) {
		base := "<definitions><process id=\"p1\" name=\"proc\"><userTask id=\"t1\" name=\"a\"/></process></definitions>"
		target := "<definitions><process id=\"p1\" name=\"proc\"><userTask id=\"t1\" name=\"a\"/><userTask id=\"t2\" name=\"b\"/></process></definitions>"
		changes, _ := f.svc.compareBPMNXML(base, target)
		require.Len(t, changes, 1)
		assert.Equal(t, "high", changes[0].Impact)
		assert.Contains(t, changes[0].Description, "Task count changed from 1 to 2")
	})

	t.Run("content differs but same task count", func(t *testing.T) {
		base := "<definitions><process id=\"p1\" name=\"short\"/></definitions>"
		target := "<definitions><process id=\"p1\" name=\"much_longer_name\"/></definitions>"
		changes, _ := f.svc.compareBPMNXML(base, target)
		require.Len(t, changes, 1)
		assert.Equal(t, "medium", changes[0].Impact)
		assert.Equal(t, "BPMN XML content has changed", changes[0].Description)
	})

	t.Run("invalid XML falls back to generic change", func(t *testing.T) {
		changes, _ := f.svc.compareBPMNXML("not-xml", "<also bad")
		require.Len(t, changes, 1)
		assert.Equal(t, "medium", changes[0].Impact)
	})
}

func seedDefinitionWithXML(t *testing.T, f versionFixture, key, version, bpmnXML string) int {
	t.Helper()
	ctx := context.Background()
	deployment, err := f.svc.client.ProcessDeployment.Create().
		SetDeploymentID(fmt.Sprintf("DEP-%s-%s", key, version)).
		SetDeploymentName(fmt.Sprintf("%s v%s", key, version)).
		SetIsActive(true).SetTenantID(f.tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := f.svc.client.ProcessDefinition.Create().
		SetKey(key).SetName(fmt.Sprintf("%s v%s", key, version)).
		SetVersion(version).
		SetIsLatest(version == "1").SetIsActive(version == "1").
		SetBpmnXML([]byte(bpmnXML)).
		SetDeploymentID(deployment.ID).SetTenantID(f.tenantID).
		Save(ctx)
	require.NoError(t, err)
	return def.ID
}
