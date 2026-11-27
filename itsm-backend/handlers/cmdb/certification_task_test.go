package cmdb

// 数据认证任务（CMDB Health / Compliance）回归。
//
// 锁的四条不变式：
//  1. 同 CI 不能同时存在两条未决任务（否则同一份数据被反复催办、逾期计数翻倍）；
//  2. 确认必须「任务表 + CI 冗余字段」同一事务提交（任务 confirmed 但 CI 仍是旧
//     last_confirmed_at 会让治理看板显示陈旧，正是这个能力要解决的问题）；
//  3. 认证必须有操作人身份（actorID<=0 落 confirmed_by=0 等于伪造审计痕）；
//  4. 跨租户的 CI / 任务一律 fail-closed，不能靠 ID 猜测越租户读写。

import (
	"context"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type certFixture struct {
	svc      *CICertificationService
	ctx      context.Context
	tenantID int
	ciID     int
}

func newCertificationFixture(t *testing.T) *certFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:ci_certification?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	tenant, err := client.Tenant.Create().SetName("认证租户").SetCode("ci-cert").
		SetDomain("ci-cert.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	ci, err := client.ConfigurationItem.Create().SetName("cert-ci").SetCiTypeID(ciType.ID).
		SetTenantID(tenant.ID).SetOwnerID(7).Save(ctx)
	require.NoError(t, err)

	return &certFixture{
		svc:      NewCICertificationService(client, zap.NewNop().Sugar()),
		ctx:      ctx,
		tenantID: tenant.ID,
		ciID:     ci.ID,
	}
}

func TestCreateTask_PendingDeduplication(t *testing.T) {
	f := newCertificationFixture(t)

	first, err := f.svc.CreateTask(f.ctx, f.tenantID, 7, &dto.CreateCICertificationRequest{CIID: f.ciID})
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, "pending", first.Status)

	// due_at 缺省必须由服务端补齐：否则逾期统计恒为 0，认证体系形同无约束。
	assert.False(t, first.DueAt.IsZero(), "未指定截止时间时应由服务端给默认值")
	assert.True(t, first.DueAt.After(time.Now()), "默认截止时间应在未来")

	// 第二次必须在快速查询处拒绝，不能依赖默认 enttest 并不存在的唯一索引。
	f.svc.client.CICertificationTask.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			t.Error("已有 pending 时不应再次尝试插入")
			return next.Mutate(ctx, m)
		})
	})
	// 第二条同 CI 的未决任务必须被拒。
	_, err = f.svc.CreateTask(f.ctx, f.tenantID, 7, &dto.CreateCICertificationRequest{CIID: f.ciID})
	assert.ErrorIs(t, err, ErrCITaskAlreadyPending, "同一 CI 只能有一条未决认证任务")
}

func TestCreateTask_CrossTenantCIFailsClosed(t *testing.T) {
	f := newCertificationFixture(t)

	otherTenant, err := f.svc.client.Tenant.Create().SetName("别的租户").SetCode("ci-cert-other").
		SetDomain("ci-cert-other.test").SetStatus("active").Save(f.ctx)
	require.NoError(t, err)
	otherType, err := f.svc.client.CIType.Create().SetName("Server").SetTenantID(otherTenant.ID).Save(f.ctx)
	require.NoError(t, err)
	foreignCI, err := f.svc.client.ConfigurationItem.Create().SetName("foreign-ci").
		SetCiTypeID(otherType.ID).SetTenantID(otherTenant.ID).Save(f.ctx)
	require.NoError(t, err)

	_, err = f.svc.CreateTask(f.ctx, f.tenantID, 7, &dto.CreateCICertificationRequest{CIID: foreignCI.ID})
	assert.ErrorIs(t, err, ErrCINotOwned, "不能用本租户身份给别的租户的 CI 建认证任务")
}

func TestConfirmTask_SyncsCIComplianceFields(t *testing.T) {
	f := newCertificationFixture(t)

	task, err := f.svc.CreateTask(f.ctx, f.tenantID, 7, &dto.CreateCICertificationRequest{CIID: f.ciID})
	require.NoError(t, err)

	before := time.Now().Add(-time.Second)
	updated, err := f.svc.ConfirmTask(f.ctx, f.tenantID, 9, task.ID, &dto.ConfirmCICertificationRequest{
		Evidence: "已核对机架位置与序列号",
	})
	require.NoError(t, err)
	assert.Equal(t, "confirmed", updated.Status)
	assert.Equal(t, 9, updated.ConfirmedBy)
	assert.False(t, updated.ConfirmedAt.IsZero())

	// CI 冗余字段与任务同事务落地：这是治理看板「确认新鲜度」的唯一数据来源。
	ci, err := f.svc.client.ConfigurationItem.Get(f.ctx, f.ciID)
	require.NoError(t, err)
	// ent 的 Optional+Time 字段在 Go 侧是值类型，零值即「从未确认」。
	assert.False(t, ci.LastConfirmedAt.IsZero(), "确认后 CI 必须带上最近认证时间")
	assert.WithinDuration(t, updated.ConfirmedAt, ci.LastConfirmedAt, time.Second)
	assert.Equal(t, 9, ci.ConfirmedBy)
	assert.False(t, ci.LastConfirmedAt.Before(before))

	// 重复确认必须被拒（幂等拒绝），否则会反复冲刷 CI 上的确认时间。
	_, err = f.svc.ConfirmTask(f.ctx, f.tenantID, 9, task.ID, nil)
	assert.ErrorIs(t, err, ErrCITaskAlreadyConfirmed)
}

func TestConfirmTask_RequiresActorIdentity(t *testing.T) {
	f := newCertificationFixture(t)

	task, err := f.svc.CreateTask(f.ctx, f.tenantID, 7, &dto.CreateCICertificationRequest{CIID: f.ciID})
	require.NoError(t, err)

	// actorID=0（service 主体或上下文缺失）：必须拒绝，不能落 confirmed_by=0 的假审计痕。
	_, err = f.svc.ConfirmTask(f.ctx, f.tenantID, 0, task.ID, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCICertificationActorMissing)

	// 任务必须仍是 pending（拒绝不得留下半个 confirmed 状态）。
	got, err := f.svc.client.CICertificationTask.Get(f.ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "pending", got.Status, "拒绝确认时任务状态不得被改写")
}

func TestListTasks_DerivesOverdueFromDueAt(t *testing.T) {
	f := newCertificationFixture(t)

	// 一条已逾期（due_at 在过去），一条未逾期。
	past := time.Now().Add(-48 * time.Hour)
	_, err := f.svc.CreateTask(f.ctx, f.tenantID, 7, &dto.CreateCICertificationRequest{
		CIID: f.ciID, DueAt: &past,
	})
	require.NoError(t, err)

	_, total, pending, overdue, confirmed, err := f.svc.ListTasks(f.ctx, f.tenantID, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, 1, pending)
	assert.Equal(t, 1, overdue, "due_at 已过的 pending 必须计入逾期")
	assert.Equal(t, 0, confirmed)
}

func TestCertificationAuditAssigneeTenant(t *testing.T) {
	f := newCertificationFixture(t)
	other := f.svc.client.Tenant.Create().SetName("other").SetCode("other").SetDomain("other.test").SaveX(f.ctx)
	foreign := f.svc.client.User.Create().SetName("foreign").SetUsername("foreign").SetEmail("foreign@test.com").SetPasswordHash("test").SetTenantID(other.ID).SaveX(f.ctx)
	_, err := f.svc.CreateTask(f.ctx, f.tenantID, 7, &dto.CreateCICertificationRequest{CIID: f.ciID, AssigneeID: foreign.ID})
	require.Error(t, err)
	n, err := f.svc.client.CICertificationTask.Query().Count(f.ctx)
	require.NoError(t, err)
	require.Zero(t, n)
	own := f.svc.client.User.Create().SetName("own").SetUsername("own").SetEmail("own@test.com").SetPasswordHash("test").SetTenantID(f.tenantID).SaveX(f.ctx)
	task, err := f.svc.CreateTask(f.ctx, f.tenantID, 7, &dto.CreateCICertificationRequest{CIID: f.ciID, AssigneeID: own.ID})
	require.NoError(t, err)
	require.Equal(t, own.ID, task.AssigneeID)
}

func TestCertificationAuditCASPreservesFirstSignature(t *testing.T) {
	f := newCertificationFixture(t)
	task, err := f.svc.CreateTask(f.ctx, f.tenantID, 7, &dto.CreateCICertificationRequest{CIID: f.ciID})
	require.NoError(t, err)
	// Inject a winner after the losing request reads pending but before it starts its transaction.
	injected := false
	f.svc.client.CICertificationTask.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
			result, err := next.Query(ctx, q)
			if err == nil && !injected {
				injected = true
				_, writeErr := f.svc.client.CICertificationTask.UpdateOneID(task.ID).SetStatus("confirmed").SetConfirmedBy(11).SetEvidence("first signature").SetConfirmedAt(time.Now()).Save(ctx)
				require.NoError(t, writeErr)
			}
			return result, err
		})
	}))
	_, err = f.svc.ConfirmTask(f.ctx, f.tenantID, 12, task.ID, &dto.ConfirmCICertificationRequest{Evidence: "loser"})
	require.ErrorIs(t, err, ErrCITaskAlreadyConfirmed)
	saved, err := f.svc.client.CICertificationTask.Get(f.ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, 11, saved.ConfirmedBy)
	require.Equal(t, "first signature", saved.Evidence)
	ci, err := f.svc.client.ConfigurationItem.Get(f.ctx, f.ciID)
	require.NoError(t, err)
	require.Zero(t, ci.ConfirmedBy)
}
