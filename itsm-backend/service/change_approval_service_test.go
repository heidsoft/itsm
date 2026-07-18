//go:build integration

package service

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// newChangeApprovalServiceForTest 仅当设置 ITSM_TEST_DB（Postgres DSN）时运行；
// 否则测试自动跳过。CI 中以 `go test -tags integration ./service/` 运行。
func newChangeApprovalServiceForTest(t *testing.T) (*ChangeApprovalService, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("ITSM_TEST_DB")
	if dsn == "" {
		t.Skip("set ITSM_TEST_DB (postgres DSN) to run integration tests")
	}

	client := enttest.Open(t, "postgres", dsn)
	require.NoError(t, client.Schema.Create(context.Background()))

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// 创建裸表（非 ent 管理，与生产迁移保持一致）
	_, err = db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS change_approval_chains (
			id BIGSERIAL PRIMARY KEY,
			change_id BIGINT NOT NULL,
			tenant_id BIGINT NOT NULL DEFAULT 0,
			level INT NOT NULL,
			approver_id BIGINT NOT NULL,
			role TEXT DEFAULT 'approver',
			status TEXT DEFAULT 'pending',
			is_required BOOLEAN DEFAULT true,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS change_approvals (
			id BIGSERIAL PRIMARY KEY,
			change_id BIGINT NOT NULL,
			approver_id BIGINT NOT NULL,
			tenant_id BIGINT NOT NULL DEFAULT 1,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			comment TEXT,
			approved_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err)

	svc := NewChangeApprovalService(client, db, zaptest.NewLogger(t).Sugar())
	return svc, db
}

func capStrPtr(s string) *string { return &s }

// TestChangeApprovalChainAdvancesToApproved 回归：修复前审批只写历史表，
// change_approval_chains 状态永不变更，导致变更状态永远停在 pending_approval。
func TestChangeApprovalChainAdvancesToApproved(t *testing.T) {
	ctx := context.Background()
	svc, db := newChangeApprovalServiceForTest(t)

	approverA := createUser(t, svc.client, "approver_a")
	approverB := createUser(t, svc.client, "approver_b")
	ch := createChange(t, svc.client, "pending_approval")

	_, err := db.ExecContext(ctx,
		`INSERT INTO change_approval_chains (change_id, tenant_id, level, approver_id, role, status, is_required, created_at) VALUES ($1,1,$2,$3,$4,$5,$6,NOW())`,
		ch.ID, 1, approverA.ID, "approver", "pending", true)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx,
		`INSERT INTO change_approval_chains (change_id, tenant_id, level, approver_id, role, status, is_required, created_at) VALUES ($1,1,$2,$3,$4,$5,$6,NOW())`,
		ch.ID, 2, approverB.ID, "approver", "pending", true)
	require.NoError(t, err)

	var histA, histB int
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO change_approvals (change_id, approver_id, tenant_id, status, created_at, updated_at) VALUES ($1,$2,$3,$4,NOW(),NOW()) RETURNING id`,
		ch.ID, approverA.ID, 1, "pending").Scan(&histA))
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO change_approvals (change_id, approver_id, tenant_id, status, created_at, updated_at) VALUES ($1,$2,$3,$4,NOW(),NOW()) RETURNING id`,
		ch.ID, approverB.ID, 1, "pending").Scan(&histB))

	// 审批人 A 批准
	_, err = svc.UpdateChangeApproval(ctx, histA, &dto.UpdateChangeApprovalRequest{
		Status:  dto.ChangeApprovalStatusApproved,
		Comment: capStrPtr("lgtm"),
	}, 1, approverA.ID)
	require.NoError(t, err)

	assertChainStatus(t, db, ch.ID, approverA.ID, "approved")
	assertChainStatus(t, db, ch.ID, approverB.ID, "pending")
	assertChangeStatus(t, svc, ch.ID, "pending_approval") // 仅一级批准，不应整体通过

	// 审批人 B 批准 -> 全部必填完成 -> 变更应为 approved
	_, err = svc.UpdateChangeApproval(ctx, histB, &dto.UpdateChangeApprovalRequest{
		Status:  dto.ChangeApprovalStatusApproved,
		Comment: capStrPtr("lgtm"),
	}, 1, approverB.ID)
	require.NoError(t, err)

	assertChainStatus(t, db, ch.ID, approverB.ID, "approved")
	assertChangeStatus(t, svc, ch.ID, "approved")
}

// TestChangeApprovalChainRejection 回归：任一必填审批人驳回，变更整体驳回。
func TestChangeApprovalChainRejection(t *testing.T) {
	ctx := context.Background()
	svc, db := newChangeApprovalServiceForTest(t)

	approverA := createUser(t, svc.client, "approver_a2")
	approverB := createUser(t, svc.client, "approver_b2")
	ch := createChange(t, svc.client, "pending_approval")

	_, err := db.ExecContext(ctx,
		`INSERT INTO change_approval_chains (change_id, tenant_id, level, approver_id, role, status, is_required, created_at) VALUES ($1,1,$2,$3,$4,$5,$6,NOW())`,
		ch.ID, 1, approverA.ID, "approver", "pending", true)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx,
		`INSERT INTO change_approval_chains (change_id, tenant_id, level, approver_id, role, status, is_required, created_at) VALUES ($1,1,$2,$3,$4,$5,$6,NOW())`,
		ch.ID, 2, approverB.ID, "approver", "pending", true)
	require.NoError(t, err)

	var histA, histB int
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO change_approvals (change_id, approver_id, tenant_id, status, created_at, updated_at) VALUES ($1,$2,$3,$4,NOW(),NOW()) RETURNING id`,
		ch.ID, approverA.ID, 1, "pending").Scan(&histA))
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO change_approvals (change_id, approver_id, tenant_id, status, created_at, updated_at) VALUES ($1,$2,$3,$4,NOW(),NOW()) RETURNING id`,
		ch.ID, approverB.ID, 1, "pending").Scan(&histB))

	// A 批准
	_, err = svc.UpdateChangeApproval(ctx, histA, &dto.UpdateChangeApprovalRequest{
		Status:  dto.ChangeApprovalStatusApproved,
		Comment: capStrPtr("ok"),
	}, 1, approverA.ID)
	require.NoError(t, err)
	assertChangeStatus(t, svc, ch.ID, "pending_approval")

	// B 驳回 -> 变更整体驳回
	_, err = svc.UpdateChangeApproval(ctx, histB, &dto.UpdateChangeApprovalRequest{
		Status:  dto.ChangeApprovalStatusRejected,
		Comment: capStrPtr("nope"),
	}, 1, approverB.ID)
	require.NoError(t, err)

	assertChainStatus(t, db, ch.ID, approverB.ID, "rejected")
	assertChangeStatus(t, svc, ch.ID, "rejected")
}

func createUser(t *testing.T, client *ent.Client, name string) *ent.User {
	u, err := client.User.Create().SetName(name).SetTenantID(1).Save(context.Background())
	require.NoError(t, err)
	return u
}

func createChange(t *testing.T, client *ent.Client, status string) *ent.Change {
	c, err := client.Change.Create().
		SetTitle("test change").
		SetTenantID(1).
		SetStatus(status).
		SetType("normal").
		SetPriority("medium").
		Save(context.Background())
	require.NoError(t, err)
	return c
}

func assertChainStatus(t *testing.T, db *sql.DB, changeID, approverID int, expect string) {
	var st string
	require.NoError(t, db.QueryRowContext(context.Background(),
		`SELECT status FROM change_approval_chains WHERE change_id=$1 AND approver_id=$2 ORDER BY level LIMIT 1`,
		changeID, approverID).Scan(&st))
	assert.Equal(t, expect, st)
}

func assertChangeStatus(t *testing.T, svc *ChangeApprovalService, changeID int, expect string) {
	ch, err := svc.client.Change.Get(context.Background(), changeID)
	require.NoError(t, err)
	assert.Equal(t, expect, ch.Status)
}

// TestCreateChangeApprovalWorkflowRejectsCrossTenantChange 阶段 A 验收：
// tenant B 的用户不能对 tenant A 的变更创建/覆盖审批链，防止 C1 类越权。
func TestCreateChangeApprovalWorkflowRejectsCrossTenantChange(t *testing.T) {
	ctx := context.Background()
	svc, _ := newChangeApprovalServiceForTest(t)

	// 创建 tenant A (id=1) 下的变更
	ch, err := svc.client.Change.Create().
		SetTitle("cross tenant target").
		SetTenantID(1).
		SetStatus("pending_approval").
		SetType("normal").
		SetPriority("medium").
		Save(ctx)
	require.NoError(t, err)

	approver := createUser(t, svc.client, "approver_cross")

	// tenant B (id=2) 尝试覆盖 tenant A 的审批链 -> 应被拒绝
	err = svc.CreateChangeApprovalWorkflow(ctx, &dto.ChangeApprovalWorkflowRequest{
		ChangeID: ch.ID,
		ApprovalChain: []dto.ChangeApprovalChainItem{
			{Level: 1, ApproverID: approver.ID, Role: "approver", IsRequired: true},
		},
	}, 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong to current tenant")
}

// TestCreateChangeApprovalWorkflowRejectsCrossTenantApprover 阶段 A 验收：
// 即使变更属于当前租户，也不能指定其他租户的用户作为审批人（防止 User.Get 枚举跨租户 ID）。
func TestCreateChangeApprovalWorkflowRejectsCrossTenantApprover(t *testing.T) {
	ctx := context.Background()
	svc, _ := newChangeApprovalServiceForTest(t)

	// tenant 1 的变更
	ch := createChange(t, svc.client, "pending_approval")

	// tenant 2 的用户
	otherTenantUser, err := svc.client.User.Create().
		SetName("foreign_approver").
		SetTenantID(2).
		Save(ctx)
	require.NoError(t, err)

	// tenant 1 尝试用 tenant 2 的用户当审批人 -> 应被拒绝
	err = svc.CreateChangeApprovalWorkflow(ctx, &dto.ChangeApprovalWorkflowRequest{
		ChangeID: ch.ID,
		ApprovalChain: []dto.ChangeApprovalChainItem{
			{Level: 1, ApproverID: otherTenantUser.ID, Role: "approver", IsRequired: true},
		},
	}, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in tenant")
}
