package service_request

import (
	"context"
	"sync"
	"testing"


	"go.uber.org/zap"
)

// repairRepoMock 内存版 Repository，只实现 repair 用到的方法（其余 panic 兜底）。
type repairRepoMock struct {
	Repository // 未实现的接口方法直接 panic（嵌入 nil interface）

	mu          sync.Mutex
	requests    []*ServiceRequest
	approvals   map[int][]*ServiceRequestApproval // requestID -> approvals
	usersByRole map[string][]int                  // role -> userIDs
	userDept    map[int]string                    // userID -> department
	updated     []int                             // 被调用 UpdateApproval 的 approval.ID
}

func (m *repairRepoMock) ListPendingApprovals(ctx context.Context, tenantID, targetLevel int, requiredStatus, requesterDept string, page, size int) ([]*ServiceRequest, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*ServiceRequest, 0)
	for _, r := range m.requests {
		out = append(out, r)
	}
	return out, len(out), nil
}

func (m *repairRepoMock) GetWithApprovals(ctx context.Context, id, tenantID int) (*ServiceRequest, []*ServiceRequestApproval, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.requests {
		if r.ID == id {
			return r, m.approvals[id], nil
		}
	}
	return nil, nil, nil
}

func (m *repairRepoMock) GetUserContext(ctx context.Context, userID, tenantID int) (string, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.userDept[userID], "tester", nil
}

func (m *repairRepoMock) FindActiveUsersByRole(ctx context.Context, tenantID int, role, department string) ([]int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.usersByRole[role], nil
}

func (m *repairRepoMock) UpdateApproval(ctx context.Context, approval *ServiceRequestApproval) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updated = append(m.updated, approval.ID)
	// 写回内存
	for _, list := range m.approvals {
		for _, a := range list {
			if a.ID == approval.ID {
				a.Node = approval.Node
			}
		}
	}
	return nil
}

func newRepairFixture(t *testing.T) (*repairRepoMock, *PendingApprovalRepairer) {
	t.Helper()
	repo := &repairRepoMock{
		approvals: map[int][]*ServiceRequestApproval{},
		usersByRole: map[string][]int{
			"manager":        {2, 5}, // 同部门解析结果（mock 不模拟部门过滤）
			"it_admin":       {3},
			"security_admin": {4},
			"super_admin":    {1},
		},
		userDept: map[int]string{1: "IT部门"},
	}
	svc := NewPendingApprovalRepairer(repo, zap.NewNop().Sugar())
	return repo, svc
}

func intPtrSlice(ids []int) []interface{} {
	out := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		out = append(out, float64(id))
	}
	return out
}

// 已退化为 super_admin 独审的 pending 审批应被重算为正确的角色审批人。
func TestRepairRecalculatesDegradedApprover(t *testing.T) {
	repo, svc := newRepairFixture(t)
	repo.requests = []*ServiceRequest{{ID: 1, TenantID: 1, RequesterID: 1, Status: "submitted"}}
	repo.approvals[1] = []*ServiceRequestApproval{
		{ID: 11, Level: 1, Step: ApprovalStepManager, Status: "pending", Node: map[string]interface{}{"approver_ids": intPtrSlice([]int{1})}},
	}

	n, err := svc.RunOnce(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("repaired = %d, want 1", n)
	}
	if len(repo.updated) != 1 || repo.updated[0] != 11 {
		t.Fatalf("updated = %v, want [11]", repo.updated)
	}
	got := repo.approvals[1][0].Node["approver_ids"]
	list, ok := got.([]int)
	if !ok || len(list) != 2 || list[0] != 2 || list[1] != 5 {
		t.Errorf("approver_ids = %v (%T), want [2 5]", got, got)
	}
}

// 已有人工决策（非 pending）的审批记录不得被改动；幂等：重算结果相同不写回。
func TestRepairSkipsDecidedAndIdempotent(t *testing.T) {
	repo, svc := newRepairFixture(t)
	repo.requests = []*ServiceRequest{{ID: 2, TenantID: 1, RequesterID: 1, Status: "manager_approved"}}
	repo.approvals[2] = []*ServiceRequestApproval{
		{ID: 21, Level: 1, Step: ApprovalStepManager, Status: "approved", Node: map[string]interface{}{"approver_ids": intPtrSlice([]int{1})}},
		{ID: 22, Level: 2, Step: ApprovalStepIT, Status: "pending", Node: map[string]interface{}{"approver_ids": intPtrSlice([]int{3})}},
	}

	n, err := svc.RunOnce(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("repaired = %d, want 0（approved 不动 + pending 已正确）", n)
	}
	if len(repo.updated) != 0 {
		t.Fatalf("updated = %v, want 空", repo.updated)
	}
}

// 空 node（审批链未持久化 approver_ids）也触发重算。
func TestRepairsEmptyNode(t *testing.T) {
	repo, svc := newRepairFixture(t)
	repo.requests = []*ServiceRequest{{ID: 3, TenantID: 1, RequesterID: 1, Status: "submitted"}}
	repo.approvals[3] = []*ServiceRequestApproval{
		{ID: 31, Level: 1, Step: ApprovalStepSecurity, Status: "pending", Node: nil},
	}

	if _, err := svc.RunOnce(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	got := repo.approvals[3][0].Node["approver_ids"]
	list, ok := got.([]int)
	if !ok || len(list) != 1 || list[0] != 4 {
		t.Errorf("security approver_ids = %v, want [4]", got)
	}
}
