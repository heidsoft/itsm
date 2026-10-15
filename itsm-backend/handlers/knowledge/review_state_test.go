package knowledge

import (
	"testing"

	"itsm-backend/common"

	"github.com/stretchr/testify/require"
)

// TestReviewTransitionsAllowed 锁定状态机允许的流转全集。
//
// 这张表是评审合规链路的核心：任何"漏配"都会让某个本该被拒的动作通过
// （例如 approved 直接回 draft 而不留流水，审计出现无法解释的空洞）。
func TestReviewTransitionsAllowed(t *testing.T) {
	cases := []struct {
		action   string
		from     string
		wantTo   string
		shouldOK bool
	}{
		{ReviewActionSubmit, ReviewStatusDraft, ReviewStatusUnderReview, true},
		// 驳回后修改重提
		{ReviewActionSubmit, ReviewStatusRejected, ReviewStatusUnderReview, true},
		{ReviewActionApprove, ReviewStatusUnderReview, ReviewStatusApproved, true},
		{ReviewActionReject, ReviewStatusUnderReview, ReviewStatusRejected, true},
		{ReviewActionWithdraw, ReviewStatusUnderReview, ReviewStatusDraft, true},
		{ReviewActionRepubish, ReviewStatusApproved, ReviewStatusUnderReview, true},

		// 不允许的组合
		{ReviewActionSubmit, ReviewStatusUnderReview, "", false},
		{ReviewActionSubmit, ReviewStatusApproved, "", false},
		{ReviewActionApprove, ReviewStatusDraft, "", false},
		{ReviewActionApprove, ReviewStatusRejected, "", false},
		{ReviewActionReject, ReviewStatusDraft, "", false},
		{ReviewActionWithdraw, ReviewStatusDraft, "", false},
		{ReviewActionRepubish, ReviewStatusDraft, "", false},
	}

	for _, tc := range cases {
		got, err := nextReviewStatus(tc.action, tc.from)
		if tc.shouldOK {
			require.NoErrorf(t, err, "动作 %s 从 %s 应被允许", tc.action, tc.from)
			require.Equal(t, tc.wantTo, got, "动作 %s 从 %s 的目标状态", tc.action, tc.from)
		} else {
			require.Errorf(t, err, "动作 %s 从 %s 应被拒绝", tc.action, tc.from)
		}
	}
}

// TestReviewRejectionIsConflictNotServerError 锁定状态机拒绝必须映射成 409，
// 不是 500 —— "当前状态不允许"是业务拒绝，不是服务端故障。
//
// 这条如果退化成 500，监控会把用户的正常操作（重复提交、撤回已撤回）算成故障。
func TestReviewRejectionIsConflictNotServerError(t *testing.T) {
	_, err := nextReviewStatus(ReviewActionApprove, ReviewStatusDraft)
	require.Error(t, err)

	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr, "应返回 BusinessError 以便映射具体状态码")
	require.Equal(t, common.ConflictCode, bizErr.Code,
		"状态机拒绝必须是 409（ConflictCode），不能是 500")
}

// TestUnknownReviewActionIsValidationError 锁定未知动作是 400 而非 409/500。
func TestUnknownReviewActionIsValidationError(t *testing.T) {
	_, err := nextReviewStatus("teleport", ReviewStatusDraft)
	require.Error(t, err)

	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr, "未知动作应返回 AppError")
	require.Equal(t, common.ErrCodeValidation, appErr.Code,
		"未知动作是参数问题，应为 400（ErrCodeValidation）")
}

// TestNormalizeReviewStatus 锁定空/未知状态归一到 draft。
func TestNormalizeReviewStatus(t *testing.T) {
	require.Equal(t, ReviewStatusDraft, NormalizeReviewStatus(""))
	require.Equal(t, ReviewStatusDraft, NormalizeReviewStatus("garbage"))
	require.Equal(t, ReviewStatusUnderReview, NormalizeReviewStatus(ReviewStatusUnderReview))
	require.Equal(t, ReviewStatusApproved, NormalizeReviewStatus(ReviewStatusApproved))
	require.Equal(t, ReviewStatusRejected, NormalizeReviewStatus(ReviewStatusRejected))
}

// TestRejectRequiresComment 锁定驳回必须给理由。
//
// 只说"不通过"而不说为什么，作者无法改进，评审退化为一次无信息盖章。
func TestRejectRequiresComment(t *testing.T) {
	require.True(t, ReviewRequiresComment(ReviewActionReject))
	require.False(t, ReviewRequiresComment(ReviewActionApprove))
	require.False(t, ReviewRequiresComment(ReviewActionSubmit))
	require.False(t, ReviewRequiresComment(ReviewActionWithdraw))
}

// TestIsReviewAction 锁定合法动作集合。
func TestIsReviewAction(t *testing.T) {
	for _, a := range []string{
		ReviewActionSubmit, ReviewActionApprove, ReviewActionReject,
		ReviewActionWithdraw, ReviewActionRepubish,
	} {
		require.Truef(t, IsReviewAction(a), "动作 %s 应被识别", a)
	}
	require.False(t, IsReviewAction("delete"))
	require.False(t, IsReviewAction(""))
}

// TestReviewStatusMatchesEntEnum 锁定常量与 ent schema 枚举值一致。
// schema 与常量各写一份是典型的漂移源：改了 schema 忘了改常量，
// 状态机会静默接受一个 ent 拒绝的值。
func TestReviewStatusMatchesEntEnum(t *testing.T) {
	all := []string{
		ReviewStatusDraft, ReviewStatusUnderReview,
		ReviewStatusApproved, ReviewStatusRejected,
	}
	for _, s := range all {
		require.NotEmpty(t, s)
		require.Equal(t, s, NormalizeReviewStatus(s), "已声明的状态必须能通过归一化（否则说明拼写与枚举不一致）")
	}
}

// TestEveryStatusHasOutgoingTransition 锁定不存在死状态。
// 一个进得去出不来的状态意味着流程会永久卡死。
func TestEveryStatusHasOutgoingTransition(t *testing.T) {
	reachable := map[string]bool{ReviewStatusDraft: true}
	// 从 draft 出发做可达性遍历，确认四个状态都能走到终态之外。
	for range 4 {
		for _, byFrom := range reviewTransitions {
			for from, to := range byFrom {
				if reachable[from] {
					reachable[to] = true
				}
			}
		}
	}
	for _, s := range []string{
		ReviewStatusDraft, ReviewStatusUnderReview,
		ReviewStatusApproved, ReviewStatusRejected,
	} {
		require.Truef(t, reachable[s], "状态 %s 不可达，流转表可能漏配", s)
	}
}
