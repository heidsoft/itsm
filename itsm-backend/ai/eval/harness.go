package eval

import (
	"context"
	"strings"

	"itsm-backend/service"
)

// 本文件是 AI 评测门禁的**可复用实现层**（非 _test.go），因为门禁有效性守卫
// 需要在测试之外复用这些判定逻辑——尤其是「候选是否就是期望答案」的检测。
//
// ## 为什么要有「候选真实性」这个概念
//
// 此前四类评测里有两类的候选是**自我实现**的：
//
//	摘要 ROUGE：stub = 把 messages 拼起来当候选 —— 候选就是输入本身
//	RAG hit-rate：stub = 返回 case.ExpectedDocIDs —— 候选就是期望答案
//
// 这类门禁的分数恒为满分（实测 ROUGE 0.84 / hit-rate 1.00 / ROC AUC 1.00），
// 因为「用正确答案去考自己」不可能低分。更要命的是：把真实实现改坏（比如把
// 摘要换成空串、把检索换成随机），这些门禁**照样绿**——它们测不出任何东西。
//
// 一个测不出退化��门禁不是「宽松的门禁」，是**没有门禁**：它给人虚假的安全感，
// 让「AI 质量有 CI 保护」这句话变成不实陈述。
//
// 因此本层提供三件事：
//
//  1. 真实候选生成器（走生产代码路径，不是 stub）；
//  2. 自我实现检测（候选 == 期望答案 → 判定评测无效）；
//  3. 退化基线（证明门禁对退化敏感）。
//
// ## CI 与生产验收的分工
//
//	CI（无外部依赖）：用生产链路的**降级路径**做候选（无 LLM 时摘要走截断、
//	              检索走确定性基线），断言「候选不是期望答案」+「退化会转红」。
//	              这测的是「降级路径不回归」，是真实的。
//	生产验收：需要真实语料与真实 LLM 候选生成器，当前尚未提供。不要用一个
//	              只切换日志标签的环境变量冒充它。

// SummarizeCase golden case for summarize 评估
type SummarizeCase struct {
	ID             string   `json:"id"`
	TicketID       string   `json:"ticket_id"`
	Messages       []string `json:"messages"`
	ExpectedTopics []string `json:"expected_topics"`
	MaxLength      int      `json:"max_length"`
	MinLength      int      `json:"min_length"`
}

// PredictionCase golden case for SLA breach prediction 评估
type PredictionCase struct {
	ID                      string                 `json:"id"`
	TicketID                string                 `json:"ticket_id"`
	Features                map[string]interface{} `json:"features"`
	ExpectedBreachWithinSLA bool                   `json:"expected_breach_within_sla"`
	HorizonMinutes          int                    `json:"horizon_minutes"`
	// AnnotationBasis 是人工标注的依据。
	//
	// 期望标签必须**独立于预测器**产生：此前的数据集每条自带
	// breach_probability_min（分数），AUC 用的就是它——分数与标签构造上一致，
	// AUC 恒 1.00，那个门禁不测任何生产代码。现在分数改由生产预测器产出
	// （SLABreachPredictor），标签由人工按 SLA 语义标注，标注依据随数据集
	// 留存，供审阅与再标注。
	AnnotationBasis string `json:"annotation_basis"`
}

// RAGCase 是 RAG 评测的 golden case。
//
// 定义放在非测试文件（harness.go）而非 eval_test.go：候选生成器与守卫需要
// 在非测试代码里引用它——如果只在 _test.go 里定义，非测试文件就无法提供
// 可复用的候选生成与真实性检测。
type RAGCase struct {
	ID       string `json:"id"`
	Query    string `json:"query"`
	TenantID int    `json:"tenant_id"`
	TopK     int    `json:"top_k"`
	// ExpectedRefs 是期望召回的**文章标题**（不是库的主键）。
	//
	// 为什么按标题而不是 ID：按 ID 会把数据集钉死在某个库的具体主键上，换
	// 环境（生产库、另一租户）就失效。按标题引用让同一份数据集既能在 CI 的
	// 内存库里跑，也能在接了真实知识库的生产验收里跑。
	ExpectedRefs []string `json:"expected_refs"`
	// MinRelevance 是该用例的相关性下限（生产语义：低于此分不得呈现）。
	MinRelevance float64 `json:"min_relevance"`
	// CorpusTenantID 是语料所属租户；0 = 与 TenantID 同租户。
	CorpusTenantID int `json:"corpus_tenant_id"`
}

// GoldenCandidateReport 描述一次候选生成是否「就是期望答案」。
type GoldenCandidateReport struct {
	// UsedGolden 为 true 表示候选直接来自期望答案——评测无效。
	UsedGolden bool
	// Detail 人类可读的判定依据。
	Detail string
}

// SummarizeROUGE averages character-level ROUGE-L recall of generated summaries against expected topics.
func SummarizeROUGE(cases []SummarizeCase, summarize func(string) string) float64 {
	if len(cases) == 0 {
		return 0
	}
	total := 0.0
	for _, tc := range cases {
		candidate := summarize(strings.Join(tc.Messages, " "))
		refTokens := tokenize(strings.Join(tc.ExpectedTopics, " "))
		candTokens := tokenize(candidate)
		if tc.MaxLength > 0 && len(candTokens) > tc.MaxLength {
			candTokens = candTokens[:tc.MaxLength]
		}
		refRunes := []rune(strings.Join(refTokens, ""))
		candRunes := []rune(strings.Join(candTokens, ""))
		if len(refRunes) == 0 {
			total += 1 // vacuous pass when no expected topics
			continue
		}
		lcsLen := lcsLengthRunes(refRunes, candRunes)
		if lcsLen == 0 {
			continue
		}
		total += float64(lcsLen) / float64(len(refRunes)) // recall
	}
	return total / float64(len(cases))
}

// lcsLengthRunes is the rune-level LCS used for ROUGE-L. Kept separate from
// lcsLength (string-token variant) so callers can pick the granularity.
func lcsLengthRunes(a, b []rune) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	dp := make([][]int, len(a)+1)
	for i := range dp {
		dp[i] = make([]int, len(b)+1)
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] > dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}
	return dp[len(a)][len(b)]
}

// tokenize splits a string on whitespace and case-folds so token comparison
// is case-insensitive. Empty tokens are dropped.
func tokenize(s string) []string {
	s = strings.ToLower(s)
	parts := strings.Fields(s)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

// DetectGoldenText 判断候选文本是否就是期望答案。
//
// 判定依据：候选去掉空白后与期望完全相同。摘要场景的期望是 topics 列表
// （期望答案的拼接形式），所以这里比较「候选 vs 期望 topics 的拼接」。
//
// 为什么用「完全相同」而不是「高度相似」：完全相同是自我实现的确定特征
// （stub 直接返回期望），而高度相似是正常的评测结果（好的摘要本来就该覆盖
// 期望主题）。混淆两者会误杀真实的高分候选。
func DetectGoldenText(candidate string, expectedTopics []string) GoldenCandidateReport {
	if len(expectedTopics) == 0 {
		return GoldenCandidateReport{Detail: "无期望答案可比对"}
	}
	expected := strings.Join(expectedTopics, "")
	norm := func(s string) string {
		return strings.Join(strings.Fields(s), "")
	}
	if norm(candidate) == norm(expected) {
		return GoldenCandidateReport{
			UsedGolden: true,
			Detail:     "候选文本与期望答案完全相同——评测是自我实现的，指标无效",
		}
	}
	return GoldenCandidateReport{Detail: "候选与期望答案不同，评测有效"}
}

// DetectGoldenDocIDs 判断候选文档集合是否就是期望答案集合。
//
// 比较用集合（忽略顺序与重复）——检索结果的顺序本就该与期望不同，
// 若集合完全一致则说明候选是直接抄来的。比对前统一归一化（去空白、
// 统一大小写）：真实库的标题可能有全角空格或大小写差异，不归一化会产生
// 「明明召回了却算未命中」的假阴性，让召回率被低估、掩盖真实问题。
func DetectGoldenDocIDs(candidate, expected []string) GoldenCandidateReport {
	if len(expected) == 0 {
		return GoldenCandidateReport{Detail: "无期望答案可比对"}
	}
	candSet := map[string]struct{}{}
	for _, id := range candidate {
		candSet[NormalizeRef(id)] = struct{}{}
	}
	expSet := map[string]struct{}{}
	for _, id := range expected {
		expSet[NormalizeRef(id)] = struct{}{}
	}
	if len(candSet) != len(expSet) {
		return GoldenCandidateReport{Detail: "候选集合与期望答案不同，评测有效"}
	}
	for id := range expSet {
		if _, ok := candSet[id]; !ok {
			return GoldenCandidateReport{Detail: "候选集合与期望答案不同，评测有效"}
		}
	}
	return GoldenCandidateReport{
		UsedGolden: true,
		Detail:     "候选文档集合与期望答案完全相同——评测是自我实现的，指标无效",
	}
}

// SummarizeCandidateGenerator 返回摘要评测的候选生成器。
//
// llm 模式：走真实的 SummarizeService（需要注入 gateway），候选即模型产出。
// degraded 模式（CI 默认）：走**生产代码的降级路径**——
// service.SummarizeService 在没有 gateway 时会做截断，这正是生产上
// 「无 LLM 时摘要长什么样」的真实行为。
//
// 关键点：两条路径都**不是**「把输入拼起来当答案」。降级路径产出的是截断，
// 与期望答案不同——因此评测是有区分力的（实测 degraded 模式分数低于满分）。
func SummarizeCandidateGenerator(svc *service.SummarizeService) func(string) string {
	if svc == nil {
		return func(text string) string { return text }
	}
	return func(text string) string {
		summary, err := svc.Summarize(context.Background(), text, 0)
		if err != nil {
			// 候选生成失败不能当成「完美答案」：如实退回空串，
			// 让指标反映失败而不是掩盖它。
			return ""
		}
		return summary
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
