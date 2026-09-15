package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"
)

// 契约（2026-09-15 R3 修复）：process_approval_decisions 是只追加的审计事实表，
// 一个任务允许产生多条决策记录（delegate→受托人完成、add_approver、多次委托链）。
// 历史 UNIQUE(tenant_id, process_task_id) 已移除——它曾使"委托后完成审批"必然
// 撞唯一约束。防重由任务状态 CAS（completeTaskWithClient 事务内 updated!=1）保证。
func TestApprovalDecisionHistoryTenantIsolationAndMultiplicity(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:approval_decisions?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	create := func(tenantID, taskID int, key, action, decision string) error {
		_, err := client.ProcessApprovalDecision.Create().
			SetProcessInstanceID(10).SetProcessTaskID(taskID).SetProcessInstanceKey(key).
			SetTaskID("TASK").SetProcessDefinitionKey("change").SetNodeKey("manager").
			SetActorID(1).SetAction(action).SetDecision(decision).SetTenantID(tenantID).Save(ctx)
		return err
	}
	// 跨租户互不干扰
	if err := create(1, 100, "PI-1", "delegate", "delegated"); err != nil {
		t.Fatal(err)
	}
	if err := create(2, 200, "PI-1", "delegate", "delegated"); err != nil {
		t.Fatal(err)
	}
	// 同一任务允许多条审计事实：委托 → 受托人完成（回归 R3 唯一约束冲突）
	if err := create(1, 100, "PI-1", "approve", "approved"); err != nil {
		t.Fatalf("同任务多条决策（delegate+approve）必须合法: %v", err)
	}
	// 多次委托链同样合法
	if err := create(1, 100, "PI-1", "delegate", "delegated"); err != nil {
		t.Fatalf("多次委托链（同 action 多行）必须合法: %v", err)
	}

	svc := &bpmnTaskService{client: client}
	tenantOne := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, 1)
	history, err := svc.ListApprovalDecisions(tenantOne, "PI-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 3 {
		t.Fatalf("租户1应看到全部 3 条决策（2 delegate + 1 approve）: got %d", len(history))
	}
	for _, h := range history {
		if h.TenantID != 1 {
			t.Fatalf("租户隔离被破坏: %#v", h)
		}
	}
	// 按时间升序排列
	for i := 1; i < len(history); i++ {
		if history[i].CreatedAt.Before(history[i-1].CreatedAt) {
			t.Fatalf("历史应按 created_at 升序: %#v", history)
		}
	}
}
