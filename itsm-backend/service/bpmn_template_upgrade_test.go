package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
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
