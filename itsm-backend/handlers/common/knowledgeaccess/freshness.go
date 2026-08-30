package knowledgeaccess

import (
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/predicate"
)

// 本文件实现「知识可引用性 L1：时效性」。
//
// # 为什么它是正确性问题而不是排序问题
//
// RAG 把检索到的片段直接喂给 LLM 作为回答依据。一条 2023 年的报销标准、
// 一份已下线系统的操作手册，只要还留在库里且被召回，模型就会用它一本正经地
// 给出错误答案——而且因为带着引用来源，比模型自己编造更有迷惑性。
// 这类错误无法靠排序缓解：只要它还在候选集里，就可能在别的 query 上排第一。
// 因此时效逾期必须走**过滤**（准入判定），而不是降权。
//
// # 与 L0（分类可见性）的分工
//
// L0 回答「这个人能不能看这篇」，L1 回答「这篇现在还能不能被引用」。
// 两者正交，RAG 检索必须同时通过。
//
// # 判定维度
//
//   - 硬时效（valid_from / valid_until）：管理员显式声明的生命周期，无争议，
//     到达即失效。可在 SQL 层过滤，limit 语义精确。
//   - 复核逾期（last_reviewed_at + review_interval_days）：声明了复核周期却
//     逾期未完成复核。涉及列间运算（now - last_reviewed_at > interval），
//     ent 谓词无法表达，只能后置过滤。
//
// # 向后兼容
//
// 三个时间字段均为 nullable 且默认不填，review_interval_days 默认 0。
// 未填时效信息的文章一律视为「长期有效、不设复核」，行为与本次改动前完全一致——
// 只有管理员显式声明了时效的知识才进入管控，存量知识不会被静默屏蔽。

// FreshnessVerdict 单篇知识文章的时效判定结果。
type FreshnessVerdict int

const (
	// FreshOK 处于有效期内，可被 RAG 引用。
	FreshOK FreshnessVerdict = iota
	// FreshNotYetEffective 已定稿但尚未到达生效时间（预定发布的知识）。
	FreshNotYetEffective
	// FreshExpired 已越过失效时间，内容不再适用。
	FreshExpired
	// FreshReviewOverdue 声明了复核周期但逾期未完成复核，内容未经确认。
	FreshReviewOverdue
)

// String 返回判定结果的可读名称，用于日志与审计。
func (v FreshnessVerdict) String() string {
	switch v {
	case FreshOK:
		return "ok"
	case FreshNotYetEffective:
		return "not_yet_effective"
	case FreshExpired:
		return "expired"
	case FreshReviewOverdue:
		return "review_overdue"
	default:
		return "unknown"
	}
}

// Citable 该文章当前是否可被 RAG 引用。
func (v FreshnessVerdict) Citable() bool { return v == FreshOK }

// FreshnessPolicy 时效性管控策略。
//
// ExcludeNotYetEffective / ExcludeExpired 建议恒为 true：它们表达的是
// 管理员显式声明的生命周期，放行等于违背声明。
//
// ExcludeReviewOverdue 是本项目主动选择的严格默认值——用户明确把「防错误引用」
// 列为正确性问题优先于召回量。它带来的代价是：设了复核周期的知识一旦逾期会从
// RAG 结果中消失，直到管理员复核（调用 MarkArticleReviewed）为止。
// 若某租户的知识运营跟不上复核节奏，可关掉这一项退回宽松模式。
type FreshnessPolicy struct {
	// ExcludeNotYetEffective 是否排除尚未生效的文章。
	ExcludeNotYetEffective bool
	// ExcludeExpired 是否排除已失效的文章。
	ExcludeExpired bool
	// ExcludeReviewOverdue 是否排除复核逾期的文章。
	ExcludeReviewOverdue bool
	// ReviewGraceDays 复核宽限期（天）。逾期未超过宽限期仍视为有效，
	// 用于吸收节假日、复核人休假等正常抖动，避免误伤。
	ReviewGraceDays int
}

// DefaultFreshnessPolicy 返回默认策略：三类逾期全部排除，不给宽限期。
func DefaultFreshnessPolicy() FreshnessPolicy {
	return FreshnessPolicy{
		ExcludeNotYetEffective: true,
		ExcludeExpired:         true,
		ExcludeReviewOverdue:   true,
		ReviewGraceDays:        0,
	}
}

// PermissiveFreshnessPolicy 返回宽松策略：只排除管理员显式声明失效的文章。
// 适用于知识复核流程尚未建立、但又想避免引用已废止制度的租户。
func PermissiveFreshnessPolicy() FreshnessPolicy {
	return FreshnessPolicy{
		ExcludeNotYetEffective: true,
		ExcludeExpired:         true,
		ExcludeReviewOverdue:   false,
		ReviewGraceDays:        0,
	}
}

// NeedsPostFilter 该策略是否存在无法在 SQL 层表达、必须后置过滤的判定项。
// 复核逾期涉及「当前时间 - 上次复核时间 > 复核周期」的列间运算，
// ent 谓词体系无法表达，只能在 Go 侧对已取回的文章逐一判定。
// 调用方据此决定是否需要多取候选（over-fetch）以保证 limit 语义。
func (p FreshnessPolicy) NeedsPostFilter() bool { return p.ExcludeReviewOverdue }

// FreshnessFields 时效判定所需的字段快照。
// 单独成结构是为了让判定逻辑不依赖 ent 实体，便于纯单元测试。
type FreshnessFields struct {
	ValidFrom          *time.Time
	ValidUntil         *time.Time
	LastReviewedAt     *time.Time
	ReviewIntervalDays int
}

// FieldsOf 从 ent 实体提取时效字段。a 为 nil 时返回零值快照（判定为可引用，
// 因为 nil 不是一篇文章，调用方应自行保证非 nil）。
func FieldsOf(a *ent.KnowledgeArticle) FreshnessFields {
	if a == nil {
		return FreshnessFields{}
	}
	return FreshnessFields{
		ValidFrom:          a.ValidFrom,
		ValidUntil:         a.ValidUntil,
		LastReviewedAt:     a.LastReviewedAt,
		ReviewIntervalDays: a.ReviewIntervalDays,
	}
}

// FreshnessJudger 时效性判定器。无状态、不查库，可安全并发共享。
type FreshnessJudger struct {
	policy  FreshnessPolicy
	nowFunc func() time.Time
}

// NewFreshnessJudger 创建判定器。nowFunc 为 nil 时使用 time.Now，
// 测试可注入固定时钟。
func NewFreshnessJudger(p FreshnessPolicy, nowFunc func() time.Time) *FreshnessJudger {
	if nowFunc == nil {
		nowFunc = time.Now
	}
	return &FreshnessJudger{policy: p, nowFunc: nowFunc}
}

// Policy 返回当前策略（只读）。
func (j *FreshnessJudger) Policy() FreshnessPolicy { return j.policy }

// Now 返回判定器当前认定的时间。
func (j *FreshnessJudger) Now() time.Time { return j.nowFunc() }

// Judge 判定单篇文章的时效状态。
//
// 判定顺序是有意为之：先判管理员显式声明的失效（Expired），再判未生效，
// 最后才判复核逾期。前两者是确定事实，后者是流程信号，
// 当同一篇文章同时命中多个状态时，返回的事实性结论优先级更高，
// 便于运营从日志判断该走「废止」还是「催办复核」两条不同的处理路径。
func (j *FreshnessJudger) Judge(f FreshnessFields) FreshnessVerdict {
	now := j.nowFunc()

	if f.ValidUntil != nil && !now.Before(*f.ValidUntil) {
		return FreshExpired
	}
	if f.ValidFrom != nil && now.Before(*f.ValidFrom) {
		return FreshNotYetEffective
	}
	if f.ReviewIntervalDays > 0 {
		deadline := f.ReviewIntervalDays + j.policy.ReviewGraceDays
		if deadline < f.ReviewIntervalDays { // 整数溢出保护
			deadline = f.ReviewIntervalDays
		}
		switch {
		case f.LastReviewedAt == nil:
			// 声明了复核周期却从未复核：内容从未被确认过，按逾期处理。
			return FreshReviewOverdue
		case now.After(f.LastReviewedAt.AddDate(0, 0, deadline)):
			return FreshReviewOverdue
		}
	}
	return FreshOK
}

// Citable 判定文章当前是否可被 RAG 引用（综合策略与时效状态）。
func (j *FreshnessJudger) Citable(f FreshnessFields) bool {
	switch j.Judge(f) {
	case FreshOK:
		return true
	case FreshNotYetEffective:
		return !j.policy.ExcludeNotYetEffective
	case FreshExpired:
		return !j.policy.ExcludeExpired
	case FreshReviewOverdue:
		return !j.policy.ExcludeReviewOverdue
	default:
		return false
	}
}

// CitableArticle 直接判定 ent 实体，nil 一律视为不可引用（fail-closed）。
func (j *FreshnessJudger) CitableArticle(a *ent.KnowledgeArticle) bool {
	if a == nil {
		return false
	}
	return j.Citable(FieldsOf(a))
}

// SQLPredicate 生成可在数据库层执行的硬时效过滤谓词。
//
// 只覆盖 valid_from / valid_until（管理员显式声明的生命周期）。
// 复核逾期无法用 ent 谓词表达，必须由 FilterArticles 后置处理。
// 调用方应把本谓词的结果与 NeedsPostFilter 的后置过滤组合使用，
// 单独使用本谓词会漏掉复核逾期的文章。
func (j *FreshnessJudger) SQLPredicate() predicate.KnowledgeArticle {
	now := j.nowFunc()

	// 未设置失效时间，或尚未到达失效时间 → 保留
	notExpired := knowledgearticle.Or(
		knowledgearticle.ValidUntilIsNil(),
		knowledgearticle.ValidUntilGT(now),
	)
	// 未设置生效时间，或已到达生效时间 → 保留
	alreadyEffective := knowledgearticle.Or(
		knowledgearticle.ValidFromIsNil(),
		knowledgearticle.ValidFromLTE(now),
	)

	if j.policy.ExcludeExpired && j.policy.ExcludeNotYetEffective {
		return knowledgearticle.And(notExpired, alreadyEffective)
	}
	if j.policy.ExcludeExpired {
		return notExpired
	}
	if j.policy.ExcludeNotYetEffective {
		return alreadyEffective
	}
	// 两项都关闭时不加任何限制；返回永真谓词以免调用方特判 nil。
	return knowledgearticle.And()
}

// FilterArticles 后置过滤：剔除按当前策略不可引用的文章，保持原顺序。
//
// 返回保留的文章与剔除数量。请与 SQLPredicate 配合使用：
// SQL 负责硬时效（保证 limit 语义），本方法负责复核逾期。
func (j *FreshnessJudger) FilterArticles(articles []*ent.KnowledgeArticle) ([]*ent.KnowledgeArticle, int) {
	if len(articles) == 0 {
		return articles, 0
	}
	kept := make([]*ent.KnowledgeArticle, 0, len(articles))
	for _, a := range articles {
		if j.CitableArticle(a) {
			kept = append(kept, a)
		}
	}
	return kept, len(articles) - len(kept)
}
