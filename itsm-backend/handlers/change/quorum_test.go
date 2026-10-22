package change

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== 会签求值（纯函数） ====================
//
// 这些用例锁的是"语义本身"：层级聚合、阈值、跨层不串票、缺省必需层。
// 生产路径的行为回归在 approval_quorum_gate_test.go（走 TransitionStatus）。

func chainItem(level, approverID int, required bool, approvalType string, threshold int) *ApprovalChain {
	return &ApprovalChain{
		Level: level, ApproverID: approverID, IsRequired: required,
		ApprovalType: approvalType, Threshold: threshold,
	}
}

func decision(approverID int, levels []int, status string) *ApprovalRecord {
	return &ApprovalRecord{ApproverID: approverID, Levels: levels, Status: status}
}

func TestEvaluateApprovalQuorum_EmptyChainIsPreAuthorized(t *testing.T) {
	out := evaluateApprovalQuorum(nil, nil)
	assert.True(t, out.ChainEmpty)
	assert.True(t, out.AllRequiredMet, "未配置审批链＝免审路径，不挡批准")
	assert.False(t, out.Rejected)
}

func TestEvaluateApprovalQuorum_ParallelNeedsEveryone(t *testing.T) {
	chain := []*ApprovalChain{
		chainItem(1, 10, true, "parallel", 2),
		chainItem(1, 11, true, "parallel", 2),
	}

	one := evaluateApprovalQuorum(chain, []*ApprovalRecord{decision(10, []int{1}, "approved")})
	assert.False(t, one.AllRequiredMet, "会签少一人不得通过")
	assert.Equal(t, 1, one.ApprovedVotes)
	assert.Equal(t, 2, one.RequiredVotes)
	assert.Equal(t, []int{1}, one.MissingLevels)

	two := evaluateApprovalQuorum(chain, []*ApprovalRecord{
		decision(10, []int{1}, "approved"),
		decision(11, []int{1}, "approved"),
	})
	assert.True(t, two.AllRequiredMet)
}

func TestEvaluateApprovalQuorum_OrTakesOneVote(t *testing.T) {
	chain := []*ApprovalChain{
		chainItem(1, 10, true, "or", 1),
		chainItem(1, 11, true, "or", 1),
	}
	out := evaluateApprovalQuorum(chain, []*ApprovalRecord{decision(10, []int{1}, "approved")})
	assert.True(t, out.AllRequiredMet, "或签任意一票即通过")
}

func TestEvaluateApprovalQuorum_VoteDoesNotLeakAcrossLevels(t *testing.T) {
	// 审批人 10 只在第 1 层；他在第 2 层的"批准"不能计入第 2 层。
	chain := []*ApprovalChain{
		chainItem(1, 10, true, "serial", 1),
		chainItem(2, 11, true, "serial", 1),
	}
	out := evaluateApprovalQuorum(chain, []*ApprovalRecord{
		decision(10, []int{1, 2}, "approved"), // 声明命中两层，但第 2 层成员不是他
	})
	assert.False(t, out.AllRequiredMet, "第 2 层未投票，不得整体通过")
	assert.Equal(t, []int{2}, out.MissingLevels)
}

func TestEvaluateApprovalQuorum_MultiLevelAllRequired(t *testing.T) {
	chain := []*ApprovalChain{
		chainItem(1, 10, true, "serial", 1),
		chainItem(2, 11, true, "serial", 1),
	}
	partial := evaluateApprovalQuorum(chain, []*ApprovalRecord{decision(10, []int{1}, "approved")})
	assert.False(t, partial.AllRequiredMet, "第 2 级未批不得整体通过")

	full := evaluateApprovalQuorum(chain, []*ApprovalRecord{
		decision(10, []int{1}, "approved"),
		decision(11, []int{2}, "approved"),
	})
	assert.True(t, full.AllRequiredMet)
}

func TestEvaluateApprovalQuorum_RejectionWins(t *testing.T) {
	chain := []*ApprovalChain{
		chainItem(1, 10, true, "parallel", 2),
		chainItem(1, 11, true, "parallel", 2),
	}
	out := evaluateApprovalQuorum(chain, []*ApprovalRecord{
		decision(10, []int{1}, "approved"),
		decision(11, []int{1}, "rejected"),
	})
	assert.True(t, out.Rejected)
	assert.False(t, out.AllRequiredMet)
}

// 审批链存在但没有任何层标记 required 时，不能按字面"无需审批"放行——
// 那等于审批链形同虚设；按全部层必需处理（fail-closed）。
func TestEvaluateApprovalQuorum_NoRequiredLevelFallsBackToAll(t *testing.T) {
	chain := []*ApprovalChain{
		chainItem(1, 10, false, "serial", 1),
		chainItem(2, 11, false, "serial", 1),
	}
	out := evaluateApprovalQuorum(chain, []*ApprovalRecord{decision(10, []int{1}, "approved")})
	assert.False(t, out.AllRequiredMet, "未标记必需层时按全部层必需处理")
	assert.Equal(t, 2, out.RequiredVotes)

	full := evaluateApprovalQuorum(chain, []*ApprovalRecord{
		decision(10, []int{1}, "approved"),
		decision(11, []int{2}, "approved"),
	})
	assert.True(t, full.AllRequiredMet)
}

func TestApplyChainLevels_FillsMissingLevels(t *testing.T) {
	chain := []*ApprovalChain{chainItem(1, 10, true, "serial", 1), chainItem(2, 10, true, "serial", 1)}
	decisions := []*ApprovalRecord{{ApproverID: 10, Status: "approved"}}
	applyChainLevels(chain, decisions)
	assert.Equal(t, []int{1, 2}, decisions[0].Levels, "层级必须由审批链派生")
}
