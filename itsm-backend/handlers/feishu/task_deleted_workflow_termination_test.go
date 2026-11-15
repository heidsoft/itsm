package feishu

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/feishuticketsync"
	entTicket "itsm-backend/ent/ticket"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// recordingTerminator 记录被调用的事实，并可选地注入失败。
//
// 断言重点不是「有没有调用」，而是「调用时拿到的事务 client 是不是同一个事务」：
// 拿到 nil 就说明流程终止跑在事务外，工单状态回滚时实例却已终止——
// 这是与 F1 相反的另一种脏状态，同样必须被测试拦住。
type recordingTerminator struct {
	calls      int
	lastTx     *ent.Client
	lastTenant int
	lastTicket int
	lastReason string
	err        error
}

func (r *recordingTerminator) TerminateTicketWorkflowTx(
	ctx context.Context, txc *ent.Client, tenantID, actorUserID, ticketID int, reason string,
) error {
	r.calls++
	r.lastTx = txc
	r.lastTenant = tenantID
	r.lastTicket = ticketID
	r.lastReason = reason
	return r.err
}

// newCloseTestFixture 造一条「工单 + 飞书同步记录」，两者属于同一租户。
//
// taskGUID 必须与 markTaskDeleted 的查询键一致（feishuticketsync.FeishuTaskID），
// 否则查不到记录会直接返回 nil 而根本走不到关单逻辑。
func newCloseTestFixture(t *testing.T, name string) (*ent.Client, *FeishuSyncService, string, int, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:feishu_close_"+name+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })

	const tenantID = 1
	ctx := context.Background()

	// 必须真的建租户与用户：Ticket 的 requester 是必填边，sqlite 开了 _fk=1
	// 时只给 requester_id 数字会被外键约束拒绝（不是字段校验，是 FK）。
	// Tenant 主键由 ent 自增，不能 SetID；工单与同步记录都用它返回的 ID，
	// 否则租户过滤条件（TenantID(tenantID)）会把记录查成 not found。
	tenantRow, err := client.Tenant.Create().
		SetName("feishu-close-" + name).
		SetCode("feishu-close-" + name).
		SetDomain("feishu-close-" + name + ".com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	requester, err := client.User.Create().
		SetUsername("requester_" + name).
		SetEmail("requester_" + name + "@example.com").
		SetName("Requester").
		SetPasswordHash("hash").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenantRow.ID).
		Save(ctx)
	require.NoError(t, err)

	tk, err := client.Ticket.Create().
		SetTenantID(tenantRow.ID).
		SetTitle("飞书任务删除联动关闭").
		SetDescription("...").
		SetType("incident").
		SetTicketNumber(fmt.Sprintf("TKT-FS-%s", name)).
		SetStatus("resolved").
		SetPriority("normal").
		SetRequesterID(requester.ID).
		SetRequester(requester).
		Save(ctx)
	require.NoError(t, err)

	taskGUID := fmt.Sprintf("guid-%s", name)
	_, err = client.FeishuTicketSync.Create().
		SetTenantID(tenantID).
		SetTicketID(tk.ID).
		SetFeishuTaskID(taskGUID).
		SetFeishuTaskGUID(taskGUID).
		SetSyncStatus("synced").
		SetLastSyncDirection("itsm_to_feishu").
		Save(ctx)
	require.NoError(t, err)

	svc := NewFeishuSyncService(client, zap.NewNop().Sugar())
	return client, svc, taskGUID, tenantRow.ID, tk.ID
}

// TestMarkTaskDeletedTerminatesWorkflowInSameTx 锁住 F1 飞书侧分叉已闭合。
func TestMarkTaskDeletedTerminatesWorkflowInSameTx(t *testing.T) {
	client, svc, taskGUID, tenantID, ticketID := newCloseTestFixture(t, "same_tx")

	term := &recordingTerminator{}
	svc.SetWorkflowTerminator(term)

	resp, err := svc.markTaskDeleted(context.Background(), tenantID, taskGUID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// 必须真的终止了流程
	require.Equal(t, 1, term.calls, "飞书删任务关单必须触发流程终止；为 0 即 F1 未收口")
	require.Equal(t, ticketID, term.lastTicket)
	require.Equal(t, tenantID, term.lastTenant)
	require.Equal(t, feishuCloseReason, term.lastReason)

	// 必须拿到事务 client：nil 说明终止跑在事务外，失去原子性
	require.NotNil(t, term.lastTx, "流程终止必须拿到事务内 client，否则与状态写入不同生共死")

	// 工单确实被置 closed
	tk, err := client.Ticket.Query().Where(entTicket.ID(ticketID)).Only(context.Background())
	require.NoError(t, err)
	require.Equal(t, "closed", tk.Status, "缺少 StatusClosed 常量时的实际值断言")
}

// TestMarkTaskDeletedRollsBackWhenTerminationFails 锁住「终止失败则整体回滚」。
//
// 若只做「先关单、再尝试终止」，终止失败后工单停在 closed 而实例还活着——
// 比原缺陷更糟：用户看到工单已关，流程却继续派活。
func TestMarkTaskDeletedRollsBackWhenTerminationFails(t *testing.T) {
	client, svc, taskGUID, tenantID, ticketID := newCloseTestFixture(t, "rollback")

	term := &recordingTerminator{err: errors.New("审批节点待办未完成，拒绝代签")}
	svc.SetWorkflowTerminator(term)

	_, err := svc.markTaskDeleted(context.Background(), tenantID, taskGUID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "terminate ticket workflow")

	// 工单状态必须回滚（仍是 resolved）
	tk, qerr := client.Ticket.Query().Where(entTicket.ID(ticketID)).Only(context.Background())
	require.NoError(t, qerr)
	require.Equal(t, "resolved", tk.Status, "终止失败必须回滚工单状态，不能停在 closed")

	// 同步记录不能被标记成 deleted
	rec, qerr := client.FeishuTicketSync.Query().
		Where(feishuticketsync.FeishuTaskID(taskGUID)).Only(context.Background())
	require.NoError(t, qerr)
	require.NotEqual(t, "deleted", rec.SyncStatus, "整事务回滚后同步状态不应是 deleted")
}

// TestMarkTaskDeletedStillClosesWithoutTerminator 锁住「未接线不阻断关单」。
//
// 兼容性约定：组装根没注入终止器时必须退回旧行为（照常关单），但要有 Warnw。
// 这里只断言行为不变；日志内容不锁字面量，避免测试对文案过敏。
func TestMarkTaskDeletedStillClosesWithoutTerminator(t *testing.T) {
	client, svc, taskGUID, tenantID, ticketID := newCloseTestFixture(t, "no_terminator")

	resp, err := svc.markTaskDeleted(context.Background(), tenantID, taskGUID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	tk, err := client.Ticket.Query().Where(entTicket.ID(ticketID)).Only(context.Background())
	require.NoError(t, err)
	require.Equal(t, "closed", tk.Status, "未接线时仍应关单，不得阻断飞书联动")
}

// TestMarkTaskDeletedUnknownTaskIsNoop 锁住幂等：查不到同步记录时直接返回，不报错也不终止。
func TestMarkTaskDeletedUnknownTaskIsNoop(t *testing.T) {
	_, svc, _, tenantID, _ := newCloseTestFixture(t, "noop")

	term := &recordingTerminator{}
	svc.SetWorkflowTerminator(term)

	resp, err := svc.markTaskDeleted(context.Background(), tenantID, "guid-not-exists")
	require.NoError(t, err)
	require.Nil(t, resp, "查不到记录应返回 nil, nil")
	require.Equal(t, 0, term.calls, "没有工单可关时不应触碰流程")
}

// TestTerminateWorkflowTxForwardsTxClient 单独锁住「事务 client 透传」。
//
// 上一个用例里 calls==0 会在「没调用」和「调用但传 nil」两种情况下同时变红，
// 无法区分。这里直接调 terminateWorkflowTx 并断言 lastTx 非 nil——
// 若有人图省事把 tx.Client() 换成 nil（便于在事务外调用），这条会精确变红。
func TestTerminateWorkflowTxForwardsTxClient(t *testing.T) {
	client, svc, _, tenantID, ticketID := newCloseTestFixture(t, "forward_tx")

	term := &recordingTerminator{}
	svc.SetWorkflowTerminator(term)

	tx, err := client.Tx(context.Background())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	require.NoError(t, svc.terminateWorkflowTx(context.Background(), tx.Client(), tenantID, ticketID))
	require.Equal(t, 1, term.calls)
	require.NotNil(t, term.lastTx, "必须透传事务 client；传 nil 会让流程终止跑在事务外")
}
