package change

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestStep4_GetBPMNApprovalDecisions verifies that GetBPMNApprovalDecisions
// queries ProcessApprovalDecision table and maps to ApprovalRecord format.
func TestStep4_GetBPMNApprovalDecisions(t *testing.T) {
	svc, client, _, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	approver1 := mkChangeUser(t, client, tenantID, "manager")
	approver2 := mkChangeUser(t, client, tenantID, "security")
	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)

	// Insert ProcessApprovalDecision records (simulating BPMN bridge writes)
	now := time.Now()
	_, err := client.ProcessApprovalDecision.Create().
		SetProcessInstanceID(1).
		SetProcessTaskID(1).
		SetProcessInstanceKey("change:normal:1").
		SetTaskID("task_approve_1").
		SetProcessDefinitionKey("change_normal_flow").
		SetNodeKey("Activity_Approve").
		SetBusinessType("change").
		SetBusinessID("1").
		SetActorID(approver1).
		SetActorName("Manager 1").
		SetAction("approve").
		SetDecision("approved").
		SetComment("同意").
		SetTenantID(tenantID).
		SetCreatedAt(now).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ProcessApprovalDecision.Create().
		SetProcessInstanceID(1).
		SetProcessTaskID(2).
		SetProcessInstanceKey("change:normal:1").
		SetTaskID("task_approve_2").
		SetProcessDefinitionKey("change_normal_flow").
		SetNodeKey("Activity_Approve").
		SetBusinessType("change").
		SetBusinessID("1").
		SetActorID(approver2).
		SetActorName("Manager 2").
		SetAction("reject").
		SetDecision("rejected").
		SetComment("风险太高").
		SetTenantID(tenantID).
		SetCreatedAt(now.Add(time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	decisions, err := svc.GetBPMNApprovalDecisions(ctx, changeID, tenantID)
	require.NoError(t, err)
	require.Len(t, decisions, 2, "Should return 2 decisions")

	// Verify first decision (approved)
	require.Equal(t, approver1, decisions[0].ApproverID)
	require.Equal(t, "Manager 1", decisions[0].ApproverName)
	require.Equal(t, "approved", decisions[0].Status)
	require.NotNil(t, decisions[0].Comment)
	require.Equal(t, "同意", *decisions[0].Comment)
	require.Equal(t, changeID, decisions[0].ChangeID)
	require.Equal(t, tenantID, decisions[0].TenantID)

	// Verify second decision (rejected)
	require.Equal(t, approver2, decisions[1].ApproverID)
	require.Equal(t, "Manager 2", decisions[1].ApproverName)
	require.Equal(t, "rejected", decisions[1].Status)
	require.NotNil(t, decisions[1].Comment)
	require.Equal(t, "风险太高", *decisions[1].Comment)
}

// TestStep4_GetApprovalHistory_UsesBPMNDecisions verifies that GetApprovalHistory
// now reads from ProcessApprovalDecision and populates Levels from approval chain.
func TestStep4_GetApprovalHistory_UsesBPMNDecisions(t *testing.T) {
	svc, client, _, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	approver1 := mkChangeUser(t, client, tenantID, "manager")
	approver2 := mkChangeUser(t, client, tenantID, "security")
	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)

	// Insert approval chain (plan metadata)
	db := svc.repo.(*EntRepository).db
	_, err := db.ExecContext(ctx, `
		INSERT INTO change_approval_chains
			(change_id, tenant_id, level, approver_id, role, status, is_required, approval_type, threshold, created_at)
		VALUES ($1, $2, 1, $3, 'manager', 'pending', 1, 'serial', 1, CURRENT_TIMESTAMP),
		       ($1, $2, 2, $4, 'security', 'pending', 1, 'serial', 1, CURRENT_TIMESTAMP)
	`, changeID, tenantID, approver1, approver2)
	require.NoError(t, err)

	// Insert ProcessApprovalDecision (only for approver1)
	now := time.Now()
	_, err = client.ProcessApprovalDecision.Create().
		SetProcessInstanceID(1).
		SetProcessTaskID(1).
		SetProcessInstanceKey("change:normal:1").
		SetTaskID("task_approve_1").
		SetProcessDefinitionKey("change_normal_flow").
		SetNodeKey("Activity_Approve").
		SetBusinessType("change").
		SetBusinessID("1").
		SetActorID(approver1).
		SetActorName("Manager 1").
		SetAction("approve").
		SetDecision("approved").
		SetComment("同意").
		SetTenantID(tenantID).
		SetCreatedAt(now).
		Save(ctx)
	require.NoError(t, err)

	history, err := svc.GetApprovalHistory(ctx, changeID, tenantID)
	require.NoError(t, err)
	require.Len(t, history, 1, "Should return 1 decision (only approver1 has decided)")

	// Verify Levels field is populated from approval chain
	require.Equal(t, approver1, history[0].ApproverID)
	require.Contains(t, history[0].Levels, 1, "Approver1 should be in level 1")
}

// TestStep4_TransitionStatus_VerifiesApproverFromChain verifies that TransitionStatus
// checks approval chain and BPMN decisions instead of legacy change_approvals.
func TestStep4_TransitionStatus_VerifiesApproverFromChain(t *testing.T) {
	svc, client, _, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	approver1 := mkChangeUser(t, client, tenantID, "manager")
	nonApprover := mkChangeUser(t, client, tenantID, "agent")
	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)

	// Insert approval chain (only approver1 is in the chain)
	db := svc.repo.(*EntRepository).db
	_, err := db.ExecContext(ctx, `
		INSERT INTO change_approval_chains
			(change_id, tenant_id, level, approver_id, role, status, is_required, approval_type, threshold, created_at)
		VALUES ($1, $2, 1, $3, 'manager', 'pending', 1, 'serial', 1, CURRENT_TIMESTAMP)
	`, changeID, tenantID, approver1)
	require.NoError(t, err)

	// Promote to pending via direct SQL
	_, err = db.ExecContext(ctx, `UPDATE changes SET status = 'pending' WHERE id = $1 AND tenant_id = $2`, changeID, tenantID)
	require.NoError(t, err)

	// Non-approver should be rejected
	_, err = svc.TransitionStatus(ctx, changeID, tenantID, nonApprover, "approved", "同意", "manager")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not an approver", "Non-approver should be rejected")

	// Approver who hasn't decided should succeed (no BPMN decisions yet)
	_, err = svc.TransitionStatus(ctx, changeID, tenantID, approver1, "approved", "同意", "manager")
	// This will fail because we don't have a full BPMN setup, but the error should not be about approver verification
	if err != nil {
		require.NotContains(t, err.Error(), "not an approver", "Approver should pass verification")
	}
}

// TestStep4_SubmitForApproval_NoLongerWritesChangeApprovals verifies that
// SubmitForApprovalWithWorkflow no longer writes to change_approvals table.
func TestStep4_SubmitForApproval_NoLongerWritesChangeApprovals(t *testing.T) {
	svc, client, _, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	approver1 := mkChangeUser(t, client, tenantID, "manager")
	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)

	plan := []ApprovalLevelPlan{
		{
			Level:        1,
			ApprovalType: "serial",
			Threshold:    1,
			Required:     true,
			ApproverIDs:  []int{approver1},
		},
	}

	err := svc.repo.SubmitForApproval(ctx, changeID, tenantID, plan, "请审批")
	require.NoError(t, err)

	// Verify change_approval_chains was written
	chain, err := svc.repo.GetApprovalChain(ctx, changeID, tenantID)
	require.NoError(t, err)
	require.Len(t, chain, 1, "Approval chain should be written")
	require.Equal(t, approver1, chain[0].ApproverID)

	// Verify change_approvals was NOT written (query directly from DB)
	db := svc.repo.(*EntRepository).db
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM change_approvals WHERE change_id = $1 AND tenant_id = $2`, changeID, tenantID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count, "change_approvals should not be written")
}

// TestStep4_QuorumEvaluation_UsesBPMNDecisions verifies that checkAndTransitionChange
// uses ProcessApprovalDecision for quorum evaluation.
func TestStep4_QuorumEvaluation_UsesBPMNDecisions(t *testing.T) {
	svc, client, _, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	approver1 := mkChangeUser(t, client, tenantID, "manager")
	approver2 := mkChangeUser(t, client, tenantID, "security")
	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)

	// Insert approval chain (2 approvers at level 1, serial/or → threshold=1)
	db := svc.repo.(*EntRepository).db
	_, err := db.ExecContext(ctx, `
		INSERT INTO change_approval_chains
			(change_id, tenant_id, level, approver_id, role, status, is_required, approval_type, threshold, created_at)
		VALUES ($1, $2, 1, $3, 'manager', 'pending', 1, 'serial', 1, CURRENT_TIMESTAMP),
		       ($1, $2, 1, $4, 'manager', 'pending', 1, 'serial', 1, CURRENT_TIMESTAMP)
	`, changeID, tenantID, approver1, approver2)
	require.NoError(t, err)

	// Promote to pending via direct SQL
	_, err = db.ExecContext(ctx, `UPDATE changes SET status = 'pending' WHERE id = $1 AND tenant_id = $2`, changeID, tenantID)
	require.NoError(t, err)

	// Insert one approval decision (threshold=1, so this should satisfy quorum)
	now := time.Now()
	_, err = client.ProcessApprovalDecision.Create().
		SetProcessInstanceID(1).
		SetProcessTaskID(1).
		SetProcessInstanceKey("change:normal:1").
		SetTaskID("task_approve_1").
		SetProcessDefinitionKey("change_normal_flow").
		SetNodeKey("Activity_Approve").
		SetBusinessType("change").
		SetBusinessID("1").
		SetActorID(approver1).
		SetActorName("Manager 1").
		SetAction("approve").
		SetDecision("approved").
		SetComment("同意").
		SetTenantID(tenantID).
		SetCreatedAt(now).
		Save(ctx)
	require.NoError(t, err)

	// checkAndTransitionChange should transition to approved (quorum satisfied)
	err = svc.checkAndTransitionChange(ctx, changeID, tenantID)
	require.NoError(t, err)

	// Verify change status was updated
	updated, err := svc.repo.Get(ctx, changeID, tenantID)
	require.NoError(t, err)
	require.Equal(t, "approved", updated.Status, "Change should be approved after quorum satisfied")
}

// TestStep4_TenantIsolation verifies that GetBPMNApprovalDecisions respects tenant isolation.
func TestStep4_TenantIsolation(t *testing.T) {
	svc, client, _, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	otherTenantID := 999
	approver := mkChangeUser(t, client, tenantID, "manager")
	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)

	// Insert decision for tenant A
	now := time.Now()
	_, err := client.ProcessApprovalDecision.Create().
		SetProcessInstanceID(1).
		SetProcessTaskID(1).
		SetProcessInstanceKey("change:normal:1").
		SetTaskID("task_approve_1").
		SetProcessDefinitionKey("change_normal_flow").
		SetNodeKey("Activity_Approve").
		SetBusinessType("change").
		SetBusinessID("1").
		SetActorID(approver).
		SetActorName("Manager").
		SetAction("approve").
		SetDecision("approved").
		SetTenantID(tenantID).
		SetCreatedAt(now).
		Save(ctx)
	require.NoError(t, err)

	// Query from tenant A should return the decision
	decisions, err := svc.GetBPMNApprovalDecisions(ctx, changeID, tenantID)
	require.NoError(t, err)
	require.Len(t, decisions, 1, "Tenant A should see its own decision")

	// Query from tenant B should return nothing (tenant isolation)
	decisions, err = svc.GetBPMNApprovalDecisions(ctx, changeID, otherTenantID)
	require.NoError(t, err)
	require.Len(t, decisions, 0, "Tenant B should not see tenant A's decisions")
}
