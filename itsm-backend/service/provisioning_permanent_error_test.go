package service

import (
	"context"
	"fmt"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/auditlog"
	"itsm-backend/infrastructure/cloud"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 云侧业务拒绝（余额不足/配额封禁）重试期望收益为零：必须翻译成「永久失败」
// 哨兵，让命令总线停止自动重投；否则同一笔注定失败的调用会被重投数次，
// 刷出成串假告警，还把「去充值」这个真正动作埋在重试记录里。
func TestExecuteTaskPermanentCloudErrorStopsRetries(t *testing.T) {
	fixture := newCloudDeliveryFixture(t, []string{"cn-hangzhou"}, false, func(ctx context.Context, client *ent.Client, tenantID int, ref string) {
		seedSingleAliyunAccount(t, client, ctx, tenantID, ref)
	})
	ctx := context.Background()

	provider := &fakeCloudProvider{
		requiresAccount: true,
		err:             fmt.Errorf("%w: 阿里云 RunInstances 失败: code=InvalidAccountStatus.NotEnoughBalance", cloud.ErrPermanent),
	}
	resolver := &fakeSecretResolver{values: map[string]string{
		fixture.accountRef: `{"type":"ak","access_key_id":"ak-test","access_key_secret":"sk-test"}`,
	}}
	svc := NewProvisioningService(fixture.client, zaptest.NewLogger(t).Sugar())
	svc.SetProvider(provider)
	svc.SetCloudRuntime(resolver)

	_, err := svc.ExecuteTask(ctx, fixture.task.ID, fixture.tenantID, fixture.requester)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProvisioningPermanentlyFailed,
		"永久错误必须翻译成哨兵，命令层据此不重投")

	failed, err := fixture.client.ProvisioningTask.Get(ctx, fixture.task.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", failed.Status)
	require.Contains(t, failed.ErrorMessage, "NotEnoughBalance", "错误文案必须保留云侧原始码供运维定位")
	require.Equal(t, 1, failed.AttemptCount, "首次执行只消耗一次尝试")

	// 对账证据仍要落审计：失败 ≠ 云侧什么都没发生。
	audits, err := fixture.client.AuditLog.Query().
		Where(auditlog.TenantID(fixture.tenantID), auditlog.ActionEQ("provision.execute.failed")).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, audits, 1)
	require.Contains(t, *audits[0].RequestBody, "clientToken")

	// 运维修好账号后仍可人工重试（自动重投停止不等于永久锁死）。
	_, err = svc.RetryTask(ctx, fixture.task.ID, fixture.tenantID, fixture.requester, "账号已充值，人工重试")
	require.NoError(t, err)
}
