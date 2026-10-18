package knowledge

import (
	"time"

	"itsm-backend/common"
)

// 评审状态。与 ent schema KnowledgeArticle.review_status 的枚举值一一对应。
const (
	ReviewStatusDraft       = "draft"        // 草稿：作者可编辑，未提交
	ReviewStatusUnderReview = "under_review" // 评审中：等待评审人决策
	ReviewStatusApproved    = "approved"     // 已通过：评审人批准，可发布
	ReviewStatusRejected    = "rejected"     // 已驳回：作者可修改后重新提交
)

// 评审动作。
const (
	ReviewActionSubmit   = "submit"    // 作者提交评审
	ReviewActionApprove  = "approve"   // 评审人通过
	ReviewActionReject   = "reject"    // 评审人驳回
	ReviewActionWithdraw = "withdraw"  // 作者撤回（评审中反悔）
	ReviewActionRepubish = "republish" // 已通过内容被修改后重新送审
)

// reviewTransitions 定义状态机允许的流转。
//
// 为什么用显式白名单而不是"任意状态可改"：知识评审是合规链路，
// 「approved 直接回 draft 而不留流水」这类跳转会让审计出现无法解释的空洞。
// 显式声明也让新增动作时必须同步更新这里的映射，漏了会在编译期或测试期暴露。
var reviewTransitions = map[string]map[string]string{
	// action → {fromStatus: toStatus}
	ReviewActionSubmit: {
		ReviewStatusDraft:    ReviewStatusUnderReview,
		ReviewStatusRejected: ReviewStatusUnderReview, // 驳回后修改重提
	},
	ReviewActionApprove: {
		ReviewStatusUnderReview: ReviewStatusApproved,
	},
	ReviewActionReject: {
		ReviewStatusUnderReview: ReviewStatusRejected,
	},
	ReviewActionWithdraw: {
		ReviewStatusUnderReview: ReviewStatusDraft,
	},
	// 内容在 approved 之后被作者改动，需要重新走一遍评审才能再次发布。
	ReviewActionRepubish: {
		ReviewStatusApproved: ReviewStatusUnderReview,
	},
}

// ErrReviewTransition 表示状态流转不被允许（当前状态不支持该动作）。
// 用独立错误而非泛化 error，handler 才能把它映射成 409 而不是 500 ——
// 「当前状态不允许」是业务拒绝，不是服务端故障。
var ErrReviewTransition = common.NewBusinessError(
	common.ConflictCode, "当前评审状态不允许该操作", "",
)

// NormalizeReviewStatus 把空/未知状态归一到合法值。
// 存量数据与测试可能不带该字段，统一按 draft 处理，
// 避免每个调用点都要判空导致漏判。
func NormalizeReviewStatus(s string) string {
	switch s {
	case ReviewStatusDraft, ReviewStatusUnderReview, ReviewStatusApproved, ReviewStatusRejected:
		return s
	default:
		return ReviewStatusDraft
	}
}

// nextReviewStatus 计算动作后的目标状态。
// 返回 ErrReviewTransition 表示当前状态不允许该动作。
func nextReviewStatus(action, current string) (string, error) {
	byFrom, ok := reviewTransitions[action]
	if !ok {
		return "", common.NewValidationError("未知的评审动作: "+action, nil)
	}
	current = NormalizeReviewStatus(current)
	to, ok := byFrom[current]
	if !ok {
		return "", ErrReviewTransition
	}
	return to, nil
}

// IsReviewAction 判定动作是否为合法的评审动作。
func IsReviewAction(action string) bool {
	_, ok := reviewTransitions[action]
	return ok
}

// ReviewRequiresComment 判定该动作是否必须填写意见。
// 驳回必须给理由：只说"不通过"而不说为什么，作者无法改进，
// 评审就退化成了一次无信息的盖章。
func ReviewRequiresComment(action string) bool {
	return action == ReviewActionReject
}

// ReviewRecord 是一次评审动作的审计流水。
type ReviewRecord struct {
	ID         int       `json:"id"`
	ArticleID  int       `json:"articleId"`
	TenantID   int       `json:"tenantId"`
	FromStatus string    `json:"fromStatus"`
	ToStatus   string    `json:"toStatus"`
	Action     string    `json:"action"`
	OperatorID int       `json:"operatorId"`
	Comment    string    `json:"comment,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}
