package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/auditlog"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 云调用失败 ≠ 云侧什么都没发生：RunInstances 超时/响应丢失时实例可能已创建。
// 失败审计必须带对账证据（ClientToken/地域/账号），否则人工判定只能瞎猜。
func TestExecuteTaskFailureAuditRecordsReconciliationEvidence(t *testing.T) {
	fixture := newCloudDeliveryFixture(t, []string{"cn-hangzhou"}, false, func(ctx context.Context, client *ent.Client, tenantID int, ref string) {
		seedSingleAliyunAccount(t, client, ctx, tenantID, ref)
	})
	ctx := context.Background()

	provider := &fakeCloudProvider{
		requiresAccount: true,
		err:             fmt.Errorf("RunInstances timeout"),
	}
	resolver := &fakeSecretResolver{values: map[string]string{
		fixture.accountRef: `{"type":"ak","access_key_id":"ak-test","access_key_secret":"sk-test"}`,
	}}
	svc := NewProvisioningService(fixture.client, zaptest.NewLogger(t).Sugar())
	svc.SetProvider(provider)
	svc.SetCloudRuntime(resolver)

	_, err := svc.ExecuteTask(ctx, fixture.task.ID, fixture.tenantID, fixture.requester)
	require.Error(t, err)

	audits, err := fixture.client.AuditLog.Query().
		Where(auditlog.TenantID(fixture.tenantID), auditlog.ActionEQ("provision.execute.failed")).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, audits, 1)
	require.NotNil(t, audits[0].RequestBody)

	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(*audits[0].RequestBody), &body))
	require.Equal(t, fmt.Sprintf("itsm-provisioning-task-%d", fixture.task.ID), body["clientToken"],
		"失败审计必须带 ClientToken，云侧按它核对同一笔创建请求")
	require.Equal(t, true, body["possibleCloudResource"],
		"云调用失败必须标记可能残留云资源")
	require.Equal(t, "cn-hangzhou", body["region"])
	require.Equal(t, "1234567890123456", body["cloudAccountId"])
	require.NotContains(t, *audits[0].RequestBody, "ak-test", "审计不得落凭据")
	require.NotContains(t, *audits[0].RequestBody, "sk-test", "审计不得落凭据")
}

// 人工判定是高风险操作：判定人、判定结果、依据必须落审计行，
// 而不是只留进程日志（日志会滚动，审计表不会）。
func TestResolveTaskManuallyWritesAuditTrail(t *testing.T) {
	fixture := newCloudDeliveryFixture(t, []string{"cn-hangzhou"}, false, func(ctx context.Context, client *ent.Client, tenantID int, ref string) {
		seedSingleAliyunAccount(t, client, ctx, tenantID, ref)
	})
	ctx := context.Background()

	svc := NewProvisioningService(fixture.client, zaptest.NewLogger(t).Sugar())

	_, err := svc.ResolveTaskManually(ctx, fixture.task.ID, fixture.tenantID, fixture.requester, "delivered", "资源已在平台外核实创建")
	require.NoError(t, err)

	audits, err := fixture.client.AuditLog.Query().
		Where(auditlog.TenantID(fixture.tenantID), auditlog.ActionEQ("provision.resolve.manual")).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, audits, 1, "人工判定必须落审计行")
	require.NotNil(t, audits[0].RequestBody)
	require.Equal(t, fixture.requester, audits[0].UserID)

	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(*audits[0].RequestBody), &body))
	require.Equal(t, "delivered", body["outcome"])
	require.Equal(t, "资源已在平台外核实创建", body["note"])
	require.Equal(t, true, body["manual"])

	// 失败判定同样要留谱系：在同一库内再造一个 pending 任务。
	secondTask, err := fixture.client.ProvisioningTask.Create().
		SetTenantID(fixture.tenantID).
		SetServiceRequestID(fixture.request.ID).
		SetProvider("aliyun").
		SetResourceType("ecs").
		SetStatus("pending").
		SetPayload(map[string]any{}).
		Save(ctx)
	require.NoError(t, err)
	_, err = svc.ResolveTaskManually(ctx, secondTask.ID, fixture.tenantID, fixture.requester, "failed", "云侧核实未创建资源，凭据已吊销")
	require.NoError(t, err)

	audits, err = fixture.client.AuditLog.Query().
		Where(auditlog.TenantID(fixture.tenantID), auditlog.ActionEQ("provision.resolve.manual")).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, audits, 2, "delivered 与 failed 判定各留一条审计")

	var failedBody map[string]any
	require.NoError(t, json.Unmarshal([]byte(*audits[1].RequestBody), &failedBody))
	require.Equal(t, "failed", failedBody["outcome"])
	require.Equal(t, true, failedBody["manual"])
}
