package knowledgeaccess

import (
	"sort"
	"time"
)

// 本文件实现「知识可引用性 L2：权威性排序」。
//
// # 为什么它是优化问题而不是正确性问题
//
// 时效与权限决定一篇知识「能不能被引用」（准入，L0/L1 的职责）；
// 权威性决定它「排在第几位」（质量）。两条都过的文章才进入本层，
// 因此这里永远不会把不该出现的内容排进来，也不会把该出现的挤出去--
// 只调整顺序。这就是它排在 L1 之后做的原因：先把错误引用堵死，再谈排得多好。
//
// # 排序信号的取舍
//
// 可用的信号有三类：
//
//   - authority_level：管理员显式标注的权威等级。可信度最高但粒度粗（0/10/20/30），
//     且同一等级内经常有多篇文章并列。
//   - 检索相关性分（vector 相似度 / keyword 匹配度）：区分度最高，
//     但只反映「文本像不像」，完全不知道内容靠不靠谱。
//   - 时间新鲜度（updated_at）：同样权威的文章里，越新维护的越可能反映现状。
//
// 单独用任何一个都会翻车：只看权威会退化成静态目录，只看相似度会被
// 「模板复制的伪官方文档」欺骗，只看时间会偏爱新但平庸的内容。
// 因此采用融合分：final = 相关性为主，权威等级做受控加成，时间做同分微调。
//
// # 受控加成的数值含义
//
// authorityBoost 把权威等级线性映射到 [0, 0.2] 的加成区间：
// 普通文章加 0，唯一真相源加 0.2。这个上界是刻意压低的--
// 相关性是「这篇文章在回答我的问题吗」，权威性是「这篇文章可信吗」，
// 前者的区分能力远高于后者。加成若超过 0.3，一篇权威但离题的文章
// 会稳定压过切题的普通文章，排序质量反而崩塌。
//
// 时间新鲜度仅在权威加成后仍同分时生效，用于打破平局，避免 slice
// 不稳定排序带来的结果抖动。t 的换算单位是 30 天记 1 分，
// 每年最多贡献约 12 分的 1/10000，量级上必须远小于权威加成。
//
// # 已知局限
//
//   - 权威等级由管理员手工标注，标注质量直接决定排序质量；
//     尚无基于引用、纠错反馈的自动权威度信号（那是 L3+ 的事）。
//   - keyword 路径的相关性分只有 0.5/0.9 两档，区分度低，
//     融合后权威加成的实际影响会比 vector 路径大，属可接受偏差。

// AuthorityLevel 常量：与 schema 注释保持一致。
const (
	// AuthorityNormal 普通知识（默认）。
	AuthorityNormal = 0
	// AuthorityRecommended 部门推荐：经部门内评审认可。
	AuthorityRecommended = 10
	// AuthorityOfficial 官方标准：组织级标准文档（制度、规范）。
	AuthorityOfficial = 20
	// AuthorityAuthoritative 唯一真相源：该主题的权威依据，与其他文章冲突时以此为准。
	AuthorityAuthoritative = 30
)

// maxAuthorityLevel 等级上限，用于归一化。
const maxAuthorityLevel = AuthorityAuthoritative

// RankInput 单条检索结果参与排序所需的信号。
type RankInput struct {
	// ArticleID 仅用于稳定排序与调试日志，不参与打分。
	ArticleID int
	// Relevance 检索相关性分，语义上应在 [0,1]。越界值会被钳制。
	Relevance float64
	// AuthorityLevel 管理员标注的权威等级（0/10/20/30）。
	AuthorityLevel int
	// UpdatedAt 最近更新时间，仅作同分平局裁决。
	UpdatedAt time.Time
}

// clamp 把数值钳制到 [lo, hi]。
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// authorityBoost 计算权威等级的受控加成，返回 [0, authorityBoostCap]。
func authorityBoost(level int) float64 {
	if level <= AuthorityNormal {
		return 0
	}
	if level >= maxAuthorityLevel {
		return authorityBoostCap
	}
	return authorityBoostCap * float64(level) / float64(maxAuthorityLevel)
}

// authorityBoostCap 权威加成的上限。
const authorityBoostCap = 0.2

// freshnessTiebreak 计算时间新鲜度平局分。
// 每 30 天扣 1 分、满分 12 分（一年前的文章归零），再折算进最终分（除以 10000），
// 保证量级只够打破平局，不足以影响任何有实际分差的排序。
func freshnessTiebreak(t time.Time, now time.Time) float64 {
	if t.IsZero() {
		return 0
	}
	days := now.Sub(t).Hours() / 24
	if days < 0 {
		days = 0 // 时钟偏移容错：未来时间按最新处理
	}
	score := clamp(12-days/30, 0, 12)
	return score / 10000
}

// FusionScore 计算融合分：相关性为主 + 权威受控加成 + 时间平局微调。
// 导出供调用方在自带排序容器（如需携带原始下标）时复用同一套打分口径，
// 避免包内外出现两套不一致的权威性权重。
func FusionScore(in RankInput, now time.Time) float64 {
	relevance := clamp(in.Relevance, 0, 1)
	return relevance + authorityBoost(in.AuthorityLevel) + freshnessTiebreak(in.UpdatedAt, now)
}

// RankedResult 排序后的结果。
type RankedResult struct {
	// Input 原样透传，调用方据此取回原始数据。
	Input RankInput
	// Score 融合分。
	Score float64
}

// RankRanking 对检索结果做权威性融合排序，返回降序结果。
//
// 稳定性保证：相关性、权威等级、更新时间全部相同的条目，保持输入顺序。
// 这让上游（向量召回顺序、SQL 返回顺序）的确定性得以保留，
// 不会因为一次排序在两次请求间产生肉眼可见的结果抖动。
//
// 排序发生在准入过滤（L0/L1）之后是本函数的隐含契约：
// 输入里不应再出现无权读取或时效逾期的文章。本函数不做二次校验，
// 因为准入判定需要查库（权限）与策略（时效），而排序是纯计算，
// 把两者搅在一起会让调用方付出双重查询代价。如果调用链上确有绕过
// 准入直接调用本函数的路径，那是调用方的 bug，应在接入层修复。
func RankRanking(inputs []RankInput, now time.Time) []RankedResult {
	results := make([]RankedResult, len(inputs))
	for i, in := range inputs {
		results[i] = RankedResult{Input: in, Score: FusionScore(in, now)}
	}
	sort.SliceStable(results, func(i, j int) bool {
		// 只按融合分比较。分毫必究的 ID 兜底比较会破坏稳定排序承诺：
		// 同分条目应保持输入序（上游召回顺序的确定性），
		// 引入 ID 强排序会让结果随主键分配顺序抖动，与文档语义矛盾。
		return results[i].Score > results[j].Score
	})
	return results
}

// OrderAuthorityRanked 对已按融合分排好序的结果，按原文顺序语义重排实体切片。
// 这是给「检索结果本身是 []*ent.KnowledgeArticle」的调用方准备的便捷方法：
// 它只做重排，不重新计算分数。
func OrderAuthorityRanked[T any](items []T, scores []float64) {
	if len(items) != len(scores) || len(items) < 2 {
		return
	}
	type pair struct {
		idx   int
		score float64
	}
	pairs := make([]pair, len(items))
	for i := range items {
		pairs[i] = pair{idx: i, score: scores[i]}
	}
	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].score > pairs[j].score })
	out := make([]T, len(items))
	for i, p := range pairs {
		out[i] = items[p.idx]
	}
	copy(items, out)
}
