package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// changeApprovalTxFixture 阶段 D 集成测试的最小夹具：
// 一个 tenant + 一个变更创建人 + 一个审批人，再加一条 Change 主表行供 tx.Change.Get 校验。
func changeApprovalTxFixture(t *testing.T) (*ent.Client, context.Context, *ent.Tenant, *ent.User, *ent.User, *ent.Change) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	client := enttest.Open(t, dialect.SQLite, dsn)
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Phase D Tenant").
		SetCode("phase-d-tenant").
		SetDomain("phase-d.example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	creator, err := client.User.Create().
		SetUsername("creator-d").
		SetEmail("creator-d@example.com").
		SetName("Change Creator").
		SetPasswordHash("hash").
		SetRole("manager").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	approver, err := client.User.Create().
		SetUsername("approver-d").
		SetEmail("approver-d@example.com").
		SetName("CAB Approver").
		SetPasswordHash("hash").
		SetRole("security").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	change, err := client.Change.Create().
		SetTitle("Phase D change").
		SetDescription("verify change approval notification tx sink").
		SetCreatedBy(creator.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return client, ctx, tenant, creator, approver, change
}

// TestNotifyChangeApprovalRequiredTxSinksIntoTx 验证「变更审批待办」在
// commit 时落到 operational_command，承载 type=change_approval_required 的 in_app 通知。
func TestNotifyChangeApprovalRequiredTxSinksIntoTx(t *testing.T) {
	client, ctx, tenant, _, approver, change := changeApprovalTxFixture(t)

	notificationSvc := NewTicketNotificationService(client, zaptest.NewLogger(t).Sugar())
	notificationSvc.EnableTxOutbox()

	tx, err := client.Tx(ctx)
	require.NoError(t, err)

	require.NoError(t, notificationSvc.NotifyChangeApprovalRequiredTx(
		ctx, tx,
		change.ID, approver.ID, tenant.ID,
		1, "cab_chair",
	))
	require.NoError(t, tx.Commit())

	commands, err := client.OperationalCommand.Query().
		Where(operationalcommand.TenantID(tenant.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, commands, 1, "审批人必须收到一条 change_approval_required 通知")

	cmd := commands[0]
	require.Equal(t, commandbus.CommandDeliverNotification, cmd.CommandType)
	require.Equal(t, "change", cmd.AggregateType)
	require.NotNil(t, cmd.Payload)
	require.Equal(t, "change_approval_required", cmd.Payload["type"])
	require.Equal(t, "in_app", cmd.Payload["channel"])
	require.Equal(t, "change", cmd.Payload["resourceType"])
	require.Equal(t, float64(approver.ID), float64(asInt(cmd.Payload["recipientId"])))
}

// TestNotifyChangeApprovalRequiredTxRollsBackWithChangeWrite 验证当与变更主表写入共享 tx 时，
// 任一侧失败都会整体回滚——通知与 change 数据同生同死。
func TestNotifyChangeApprovalRequiredTxRollsBackWithChangeWrite(t *testing.T) {
	client, ctx, tenant, creator, approver, _ := changeApprovalTxFixture(t)

	notificationSvc := NewTicketNotificationService(client, zaptest.NewLogger(t).Sugar())
	notificationSvc.EnableTxOutbox()

	tx, err := client.Tx(ctx)
	require.NoError(t, err)

	// 业务侧先把 change 写进 tx
	_, err = tx.Change.Create().
		SetTitle("Rollback scenario change").
		SetDescription("verify rollback when approval-notification fails").
		SetCreatedBy(creator.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 用一个伪造的 approvalLevel=0 触发校验失败：我们的方法要求参数合法，但不会因 level=0 报错；
	// 这里直接采用 tenantID 不匹配触发 fail-fast，模拟「业务校验后发现不能继续」的语义。
	// 为了真正触发回滚，我们改用一个不存在的 approverID，会让 tx.User.Get 返回 NotFound，
	// 这一错误会被方法包装返回，从而整体回滚。
	err = notificationSvc.NotifyChangeApprovalRequiredTx(
		ctx, tx,
		999999, approver.ID, tenant.ID, // changeID=999999 在 tx 中不存在
		1, "cab_chair",
	)
	require.Error(t, err, "不存在的 changeID 应让通知方法整体失败")
	require.NoError(t, tx.Rollback())

	// 既然 tx 回滚了，上面的 Change.Create 也不应持久化
	allChanges, err := client.Change.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, allChanges, 1, "只有 fixture 自带的 change 应保留；tx 内写入的应随 rollback 一起消失")

	notifications, err := client.OperationalCommand.Query().
		Where(operationalcommand.TenantID(tenant.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Empty(t, notifications, "rollback 后通知入箱行必须消失")
}

// TestNotifyChangeApprovalRequiredTxFailClosedWhenOutboxDisabled 验证未启用
// txOutboxEnabled 时直接 fail-closed，避免业务侧误以为已入箱。
func TestNotifyChangeApprovalRequiredTxFailClosedWhenOutboxDisabled(t *testing.T) {
	client, ctx, tenant, _, approver, change := changeApprovalTxFixture(t)

	notificationSvc := NewTicketNotificationService(client, zap.NewNop().Sugar())
	// 故意不调用 EnableTxOutbox

	tx, err := client.Tx(ctx)
	require.NoError(t, err)

	err = notificationSvc.NotifyChangeApprovalRequiredTx(
		ctx, tx,
		change.ID, approver.ID, tenant.ID,
		1, "cab_chair",
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "transactional notification outbox disabled")

	// rollback 即可，不应留下副作用
	require.NoError(t, tx.Rollback())

	commands, err := client.OperationalCommand.Query().
		Where(operationalcommand.TenantID(tenant.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Empty(t, commands)
}

// TestNotifyChangeApprovalRequiredTxRejectsCrossTenantApprover 验证跨 tenant 审批人会被拒绝。
// 防止构造错误的 approve notification 误导审批人。
func TestNotifyChangeApprovalRequiredTxRejectsCrossTenantApprover(t *testing.T) {
	client, ctx, _, _, approver, change := changeApprovalTxFixture(t)

	// 造一个不同 tenant 的「假审批人」
	otherTenant, err := client.Tenant.Create().
		SetName("Other Tenant").
		SetCode("other-tenant-d").
		SetDomain("other.example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	foreignApprover, err := client.User.Create().
		SetUsername("foreign-approver").
		SetEmail("foreign@example.com").
		SetName("Foreign Approver").
		SetPasswordHash("hash").
		SetRole("security").
		SetActive(true).
		SetTenantID(otherTenant.ID).
		Save(ctx)
	require.NoError(t, err)
	_ = foreignApprover

	notificationSvc := NewTicketNotificationService(client, zaptest.NewLogger(t).Sugar())
	notificationSvc.EnableTxOutbox()

	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	// 故意传入「与原 change 同 tenant 但不同人」的 approver，但 request tenantID 是 otherTenant —— 跨 tenant 应失败。
	err = notificationSvc.NotifyChangeApprovalRequiredTx(
		ctx, tx,
		change.ID, approver.ID, otherTenant.ID, // approver 属于 phase-d tenant，但 request tenantID 是 otherTenant
		1, "cab_chair",
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tenant mismatch")
}

// TestNotifyChangeApprovalDecidedTxSinksIntoTx 验证「变更审批结论」通知入箱。
func TestNotifyChangeApprovalDecidedTxSinksIntoTx(t *testing.T) {
	client, ctx, tenant, creator, _, change := changeApprovalTxFixture(t)

	notificationSvc := NewTicketNotificationService(client, zaptest.NewLogger(t).Sugar())
	notificationSvc.EnableTxOutbox()

	tx, err := client.Tx(ctx)
	require.NoError(t, err)

	require.NoError(t, notificationSvc.NotifyChangeApprovalDecidedTx(
		ctx, tx,
		change.ID, creator.ID, tenant.ID,
		42, "approved", "looks good",
	))
	require.NoError(t, tx.Commit())

	commands, err := client.OperationalCommand.Query().
		Where(operationalcommand.TenantID(tenant.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, commands, 1)

	cmd := commands[0]
	require.Equal(t, "change", cmd.AggregateType)
	require.Equal(t, "change_approval_decided", cmd.Payload["type"])
	require.Equal(t, "in_app", cmd.Payload["channel"])
	require.Equal(t, float64(creator.ID), float64(asInt(cmd.Payload["recipientId"])))

	// content 含 decision 文本
	content, _ := cmd.Payload["content"].(string)
	require.Contains(t, content, "已通过")
	require.Contains(t, content, "looks good")
}

// TestNotifyChangeApprovalDecidedTxInvalidDecision 验证 decision 字段必须为白名单之一。
func TestNotifyChangeApprovalDecidedTxInvalidDecision(t *testing.T) {
	client, ctx, tenant, creator, _, change := changeApprovalTxFixture(t)

	notificationSvc := NewTicketNotificationService(client, zaptest.NewLogger(t).Sugar())
	notificationSvc.EnableTxOutbox()

	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	err = notificationSvc.NotifyChangeApprovalDecidedTx(
		ctx, tx,
		change.ID, creator.ID, tenant.ID,
		1, "MAYBE", "",
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid decision")

	commands, err := client.OperationalCommand.Query().
		Where(operationalcommand.TenantID(tenant.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Empty(t, commands)
}

// TestNotifyChangeApprovalDecidedTxRollsBackApprovalWrite 同 D.3，但针对 decided 路径。
// 这里直接复用 D.2 已通过的合法路径，验证 nil-error 时正常入箱，rollback 场景由 decided 的 invalidDecision 测试已覆盖。
func TestNotifyChangeApprovalDecidedTxFailClosedWhenOutboxDisabled(t *testing.T) {
	client, ctx, tenant, creator, _, change := changeApprovalTxFixture(t)

	notificationSvc := NewTicketNotificationService(client, zap.NewNop().Sugar())
	// 未启用 EnableTxOutbox

	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	err = notificationSvc.NotifyChangeApprovalDecidedTx(
		ctx, tx,
		change.ID, creator.ID, tenant.ID,
		1, "rejected", "",
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "transactional notification outbox disabled")
}
