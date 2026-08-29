package change

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGetApprovalHistory_DialectSafeLevels 回归测试（2026-08-29 生产 P0-1）：
// 旧实现在 SELECT 中使用 string_agg(integer, ',')，PostgreSQL 对聚合参数不做
// 隐式 int->text 转换，直接报 `function string_agg(integer, unknown) does not exist`，
// 导致 POST /changes/:id/approve 返回 500 "failed to get approval history"。
// 修复后 levels 由独立查询派生并在 Go 侧拼接，PG/SQLite 双方言均可运行。
// 本用例锁定新实现的行为契约：历史记录正确返回且 levels 按审批人正确派生。
func TestGetApprovalHistory_DialectSafeLevels(t *testing.T) {
	svc, client, _, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	approver1 := mkChangeUser(t, client, tenantID, "manager")
	approver2 := mkChangeUser(t, client, tenantID, "security")
	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)

	db := svc.repo.(*EntRepository).db
	_, err := db.ExecContext(ctx, `
		INSERT INTO change_approvals (change_id, tenant_id, approver_id, status, comment, approved_at, created_at, updated_at)
		VALUES ($1, $2, $3, 'pending', '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ($1, $2, $4, 'approved', '同意', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, changeID, tenantID, approver1, approver2)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO change_approval_chains
			(change_id, tenant_id, level, approver_id, role, status, is_required, approval_type, threshold, created_at)
		VALUES ($1, $2, 2, $3, 'manager', 'pending', 1, 'serial', 1, CURRENT_TIMESTAMP),
		       ($1, $2, 1, $3, 'manager', 'pending', 1, 'serial', 1, CURRENT_TIMESTAMP),
		       ($1, $2, 1, $4, 'security', 'approved', 1, 'serial', 1, CURRENT_TIMESTAMP)
	`, changeID, tenantID, approver1, approver2)
	require.NoError(t, err)

	records, err := svc.repo.GetApprovalHistory(ctx, changeID, tenantID)
	require.NoError(t, err)
	require.Len(t, records, 2)

	byApprover := map[int]*ApprovalRecord{}
	for _, r := range records {
		byApprover[r.ApproverID] = r
	}
	require.Contains(t, byApprover, approver1)
	require.Contains(t, byApprover, approver2)
	require.Equal(t, "pending", byApprover[approver1].Status)
	require.Equal(t, "approved", byApprover[approver2].Status)
	require.NotNil(t, byApprover[approver2].ApprovedAt)
	// levels 必须按升序派生（乱序插入 2,1 仍应得到 [1,2]）
	require.Equal(t, []int{1, 2}, byApprover[approver1].Levels)
	require.Equal(t, []int{1}, byApprover[approver2].Levels)

	// 租户隔离：其他租户查询不得返回记录
	other, err := svc.repo.GetApprovalHistory(ctx, changeID, tenantID+999)
	require.NoError(t, err)
	require.Empty(t, other)
}
