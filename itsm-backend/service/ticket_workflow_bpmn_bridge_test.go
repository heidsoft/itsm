package service

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticketworkflowrecord"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	_ "github.com/mattn/go-sqlite3"
)

// ==================== ApproveTicket ↔ BPMN 桥接集成测试（P0-1 阶段3） ====================

func createBridgeTestTicketWithApproval(t *testing.T, client *ent.Client, tenantID, approverID int, suffix string) (*ent.Ticket, *ent.TicketApproval) {
	t.Helper()
	ctx := context.Background()
	tk, err := client.Ticket.Create().
		SetTitle("Bridge Approval Ticket " + suffix).
		SetStatus("pending").
		SetPriority("medium").
		SetTicketNumber("BRG-TKT-" + suffix).
		SetRequesterID(approverID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	approval, err := client.TicketApproval.Create().
		SetTicketID(tk.ID).
		SetLevel(1).
		SetLevelName("一级审批").
		SetApproverID(approverID).
		SetStatus(string(dto.ApprovalStatusPending)).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return tk, approval
}

// TestApproveTicket_BridgesBPMNTask 工单审批端到端：
// 业务审批通过时应同时完成绑定的 BPMN 待办任务，并在流转记录中标记 bpmn_handled。
func TestApproveTicket_BridgesBPMNTask(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "ticket_bridge_e2e")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "tk-e2e")
	svc := NewTicketWorkflowService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	tk, approval := createBridgeTestTicketWithApproval(t, client, tenantID, actorID, "e2e")
	_, taskID := createBridgeProcessFixture(t, client, tenantID, "tk-e2e1",
		fmt.Sprintf("ticket:%d", tk.ID), actorID)

	err := svc.ApproveTicket(ctx, &dto.ApproveTicketRequest{
		TicketID:   tk.ID,
		ApprovalID: approval.ID,
		Action:     "approve",
		Comment:    "同意",
	}, actorID, tenantID)
	require.NoError(t, err)

	// BPMN 任务已完成
	task, err := client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", task.Status)

	// 业务审批与工单状态已更新
	updatedApproval, err := client.TicketApproval.Get(ctx, approval.ID)
	require.NoError(t, err)
	assert.Equal(t, string(dto.ApprovalStatusApproved), updatedApproval.Status)
	updatedTicket, err := client.Ticket.Get(ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "approved", updatedTicket.Status)

	// 流程审批决策带正确的业务上下文
	decisions, err := client.ProcessApprovalDecision.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, actorID, decisions[0].ActorID)
	assert.Equal(t, "ticket", decisions[0].BusinessType)
	assert.Equal(t, strconv.Itoa(tk.ID), decisions[0].BusinessID)

	// 流转记录标记 bpmn_handled=true
	record, err := client.TicketWorkflowRecord.Query().
		Where(ticketworkflowrecord.TicketID(tk.ID)).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, true, record.Metadata["bpmn_handled"])
}

// TestApproveTicket_BridgeFailClosed 失败关闭回归：
// 存在待办流程任务但操作人不是流程审批人时，业务审批必须整体中止，双轨状态均不变。
func TestApproveTicket_BridgeFailClosed(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "ticket_bridge_failclosed")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "tk-fc")
	svc := NewTicketWorkflowService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	tk, approval := createBridgeTestTicketWithApproval(t, client, tenantID, actorID, "fc")
	// 流程任务指派给其他人，业务审批人无权完成流程任务
	_, taskID := createBridgeProcessFixture(t, client, tenantID, "tk-fc1",
		fmt.Sprintf("ticket:%d", tk.ID), actorID+1000)

	err := svc.ApproveTicket(ctx, &dto.ApproveTicketRequest{
		TicketID:   tk.ID,
		ApprovalID: approval.ID,
		Action:     "approve",
		Comment:    "同意",
	}, actorID, tenantID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "同步流程审批任务失败")

	// 双轨状态均未被修改
	task, err := client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "assigned", task.Status)
	unchangedApproval, err := client.TicketApproval.Get(ctx, approval.ID)
	require.NoError(t, err)
	assert.Equal(t, string(dto.ApprovalStatusPending), unchangedApproval.Status)
	unchangedTicket, err := client.Ticket.Get(ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "pending", unchangedTicket.Status)

	// 未产生任何流程审批决策与流转记录
	decisionCount, err := client.ProcessApprovalDecision.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, decisionCount)
	recordCount, err := client.TicketWorkflowRecord.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, recordCount)
}

func TestApproveTicket_WithoutBPMNTaskDoesNotWrite(t *testing.T) {
	for _, action := range []string{"approve", "reject", "delegate"} {
		t.Run(action, func(t *testing.T) {
			client := newApprovalBridgeTestClient(t, "ticket_no_bpmn_"+action)
			tenantID, actorID := setupBridgeTenantAndActor(t, client, "no-bpmn-"+action)
			svc := NewTicketWorkflowService(client, zaptest.NewLogger(t).Sugar())
			ctx := context.Background()
			tk, approval := createBridgeTestTicketWithApproval(t, client, tenantID, actorID, action)
			for attempt := 0; attempt < 2; attempt++ {
				err := svc.ApproveTicket(ctx, &dto.ApproveTicketRequest{
					TicketID: tk.ID, ApprovalID: approval.ID, Action: action, DelegateToUserID: &actorID,
				}, actorID, tenantID)
				require.ErrorContains(t, err, "没有可处理的 BPMN 审批任务")
			}
			updatedApproval, err := client.TicketApproval.Get(ctx, approval.ID)
			require.NoError(t, err)
			assert.Equal(t, string(dto.ApprovalStatusPending), updatedApproval.Status)
			updatedTicket, err := client.Ticket.Get(ctx, tk.ID)
			require.NoError(t, err)
			assert.Equal(t, "pending", updatedTicket.Status)
			count, err := client.TicketApproval.Query().Count(ctx)
			require.NoError(t, err)
			assert.Equal(t, 1, count)
			count, err = client.TicketWorkflowRecord.Query().Count(ctx)
			require.NoError(t, err)
			assert.Zero(t, count)
			count, err = client.ProcessApprovalDecision.Query().Count(ctx)
			require.NoError(t, err)
			assert.Zero(t, count)
		})
	}
}

func TestApproveTicket_RejectBridgesBPMNTask(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "ticket_bridge_reject")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "tk-rej")
	svc := NewTicketWorkflowService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	tk, approval := createBridgeTestTicketWithApproval(t, client, tenantID, actorID, "rej")
	_, taskID := createBridgeProcessFixture(t, client, tenantID, "tk-rej1",
		fmt.Sprintf("ticket:%d", tk.ID), actorID)

	err := svc.ApproveTicket(ctx, &dto.ApproveTicketRequest{
		TicketID:   tk.ID,
		ApprovalID: approval.ID,
		Action:     "reject",
		Comment:    "不符合变更窗口",
	}, actorID, tenantID)
	require.NoError(t, err)

	task, err := client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", task.Status)

	updatedTicket, err := client.Ticket.Get(ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "rejected", updatedTicket.Status)

	decisions, err := client.ProcessApprovalDecision.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "rejected", decisions[0].Decision)
	assert.Equal(t, "不符合变更窗口", decisions[0].Comment)
}
