package ticket

import (
	"context"
	"testing"

	"itsm-backend/ent"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// F1 回归锁：工单进入终态时必须终止关联 BPMN 实例。
//
// 背景（浏览器 QA 实测）：详情页走 `POST /tickets/workflow/close`，那里在自家事务里
// 终止实例并取消待办；但列表页走 `POST /tickets/:id/close`，`PUT /tickets/:id/status`
// 也写 closed，这两条落到 handlers/ticket，原先只 `UpdateStatus("closed")` ——
// 完全不碰流程。结果是工单已关闭、ProcessInstance 仍 running、待办常驻；
// 用户把待办完成掉，引擎沿网关继续 createUserTask（ticket_general_flow 的
// 「升级处理」是回环），于是"关掉的工单不断冒出新待办"。
//
// 本文件锁的是**收敛点必须在 service 层**：只改某一个 handler 会让另一条路径继续
// 漏，测试因此对 CloseTicket 与 UpdateStatus(closed) 两条入口都断言同一条行为。

// txCloserStub 同时实现 Repository 与 TxTicketCloser，用来验证「支持事务钩子」分支。
//
// 刻意不嵌入 mockRepository：嵌入会让它隐式满足 Repository 全部方法，
// 于是「是否走了 hook 路径」只能靠副作用标志位判断，而不是靠类型能力本身。
type txCloserStub struct {
	*mockRepository
	// hookCalls 记录 hook 被调用的次数（应为 1）
	hookCalls int
	// hookErr 让测试注入「流程终止失败」，验证整体回滚而不是只吞掉错误
	hookErr error
	// terminatorCalls 记录注入的终止器被调用次数
	terminatorCalls int
	// lastTerminateTx 记录终止器收到的事务 client；必须非 nil，
	// 否则终止与状态写入不在同一事务，原子性无从谈起
	lastTerminateTx *ent.Client
	lastTerminateArg struct {
		tenantID, actorID, ticketID int
		reason                      string
	}
}

func (s *txCloserStub) CloseTicketWithTxHook(
	ctx context.Context, ticketID int, version int, tenantID int, hook TxCloseHook,
) (*Ticket, error) {
	s.hookCalls++
	tx := &ent.Client{} // 非 nil 即可：本用例只断言「事务 client 被传给终止器」
	if hook != nil {
		if err := hook(ctx, tx); err != nil {
			return nil, err
		}
	}
	return s.mockRepository.CloseTicket(ctx, ticketID, tenantID)
}

// recordingTerminator 实现 TicketWorkflowTerminator。
type recordingTerminator struct {
	stub *txCloserStub
}

func (t recordingTerminator) TerminateTicketWorkflowTx(
	_ context.Context, txc *ent.Client, tenantID, actorUserID, ticketID int, reason string,
) error {
	t.stub.terminatorCalls++
	t.stub.lastTerminateTx = txc
	t.stub.lastTerminateArg.tenantID = tenantID
	t.stub.lastTerminateArg.actorID = actorUserID
	t.stub.lastTerminateArg.ticketID = ticketID
	t.stub.lastTerminateArg.reason = reason
	return t.stub.hookErr
}

// seedClosableTicket 放一张 resolved 工单（状态机只允许 resolved → closed）。
func seedClosableTicket(t *testing.T, repo *mockRepository) *Ticket {
	t.Helper()
	tk := &Ticket{
		ID: 1, TicketNumber: "T-1", Title: "closable", Status: "resolved",
		Priority: "medium", Type: "incident", RequesterID: 7, TenantID: 1,
	}
	repo.tickets[tk.ID] = tk
	return tk
}

// 修复后必须绿：两个入口都触发流程终止，且拿到事务 client。
func TestCloseTicketTerminatesWorkflowInSameTransaction(t *testing.T) {
	base := newMockRepository()
	stub := &txCloserStub{mockRepository: base}
	seedClosableTicket(t, base)

	svc := NewService(stub, nil, nil, zap.NewNop().Sugar()).
		SetWorkflowTerminator(recordingTerminator{stub: stub})

	_, err := svc.CloseTicket(context.Background(), 1, 1, 100, "admin")
	require.NoError(t, err)

	require.Equal(t, 1, stub.hookCalls, "关闭必须经过事务钩子路径")
	require.Equal(t, 1, stub.terminatorCalls, "必须终止关联 BPMN 实例")
	require.NotNil(t, stub.lastTerminateTx,
		"终止器必须收到事务 client；传 nil 意味着与状态写入不在同一事务")
	require.Equal(t, 1, stub.lastTerminateArg.tenantID)
	require.Equal(t, 1, stub.lastTerminateArg.ticketID)
	require.Equal(t, 100, stub.lastTerminateArg.actorID, "终止需记录实际操作人")
	require.NotEmpty(t, stub.lastTerminateArg.reason, "终止原因必须留给审计")
}

// PUT /tickets/:id/status 写 closed 时必须走同一条收敛路径。
// 只给 /close 补逻辑的话，本用例会红——这正是"逐个 handler 补必漏"的锁。
func TestUpdateStatusClosedTerminatesWorkflow(t *testing.T) {
	base := newMockRepository()
	stub := &txCloserStub{mockRepository: base}
	seedClosableTicket(t, base)

	svc := NewService(stub, nil, nil, zap.NewNop().Sugar()).
		SetWorkflowTerminator(recordingTerminator{stub: stub})

	_, err := svc.UpdateStatus(context.Background(), 1, "closed", 1, 100, "admin")
	require.NoError(t, err)

	require.Equal(t, 1, stub.hookCalls, "status=closed 也必须经过事务钩子路径")
	require.Equal(t, 1, stub.terminatorCalls)
	require.NotNil(t, stub.lastTerminateTx)
}

// 非终态流转不得终止流程：把 in_progress 也收口会误杀在跑的流程。
func TestUpdateStatusNonClosedDoesNotTerminateWorkflow(t *testing.T) {
	base := newMockRepository()
	stub := &txCloserStub{mockRepository: base}
	base.tickets[1] = &Ticket{
		ID: 1, TicketNumber: "T-1", Title: "x", Status: "open",
		Priority: "medium", Type: "incident", RequesterID: 7, TenantID: 1,
	}

	svc := NewService(stub, nil, nil, zap.NewNop().Sugar()).
		SetWorkflowTerminator(recordingTerminator{stub: stub})

	_, err := svc.UpdateStatus(context.Background(), 1, "in_progress", 1, 100, "admin")
	require.NoError(t, err)
	require.Equal(t, 0, stub.terminatorCalls, "非终态不得终止流程实例")
	require.Equal(t, 0, stub.hookCalls, "非终态不该走关闭事务钩子")
}

// 流程终止失败必须整体失败：吞掉错误会让工单显示"已关闭"而流程仍在跑。
func TestCloseTicketFailsWhenWorkflowTerminationFails(t *testing.T) {
	base := newMockRepository()
	stub := &txCloserStub{mockRepository: base}
	seedClosableTicket(t, base)
	stub.hookErr = assertAnError{}

	svc := NewService(stub, nil, nil, zap.NewNop().Sugar()).
		SetWorkflowTerminator(recordingTerminator{stub: stub})

	_, err := svc.CloseTicket(context.Background(), 1, 1, 100, "admin")
	require.Error(t, err, "终止失败必须上抛，不能静默吞掉")
}

// 状态机拦住的非法流转不应触碰流程（哪怕传了 closed）。
func TestCloseTicketRejectedByStateMachineBeforeWorkflow(t *testing.T) {
	base := newMockRepository()
	stub := &txCloserStub{mockRepository: base}
	base.tickets[1] = &Ticket{
		ID: 1, TicketNumber: "T-1", Title: "x", Status: "in_progress",
		Priority: "medium", Type: "incident", RequesterID: 7, TenantID: 1,
	}

	svc := NewService(stub, nil, nil, zap.NewNop().Sugar()).
		SetWorkflowTerminator(recordingTerminator{stub: stub})

	_, err := svc.CloseTicket(context.Background(), 1, 1, 100, "admin")
	require.Error(t, err, "in_progress 不允许直接 closed")
	require.Equal(t, 0, stub.terminatorCalls, "被状态机拒绝时不得终止流程")
	require.Equal(t, 0, stub.hookCalls)
}

// assertAnError 是测试用的哨兵错误。
type assertAnError struct{}

func (assertAnError) Error() string { return "workflow termination exploded" }
