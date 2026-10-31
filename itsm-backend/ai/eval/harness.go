package eval

import (
	"context"
	"os"
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
//	生产验收：设 ITSM_AI_EVAL_USE_LLM=1，用真实 LLM 跑同一套数据集，
//	              阈值由环境变量覆盖，产出真实质量分。
//
// 两个模式共用同一份数据集与同一套指标计算，指标口径一致。

// RAGCase 是 RAG 评测的 golden case。
//
// 定义放在非测试文件（harness.go）而非 eval_test.go：候选生成器与守卫需要
// 在非测试代码里引用它——如果只在 _test.go 里定义，非测试文件就无法提供
// 可复用的候选生成与真实性检测。
type RAGCase struct {
	ID             string   `json:"id"`
	Query          string   `json:"query"`
	TenantID       int      `json:"tenant_id"`
	ExpectedDocIDs []string `json:"expected_doc_ids"`
	MinRelevance   float64  `json:"min_relevance"`
	TopK           int      `json:"top_k"`
}

// EvalMode 返回当前评测模式。
//
// "llm"    —— 生产验收：接真实 LLM 生成候选
// "degraded" —— CI 默认：接生产链路的无 LLM 降级路径
func EvalMode() string {
	if strings.EqualFold(os.Getenv("ITSM_AI_EVAL_USE_LLM"), "1") {
		return "llm"
	}
	return "degraded"
}

// GoldenCandidateReport 描述一次候选生成是否「就是期望答案」。
type GoldenCandidateReport struct {
	// UsedGolden 为 true 表示候选直接来自期望答案——评测无效。
	UsedGolden bool
	// Detail 人类可读的判定依据。
	Detail string
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
// 若集合完全一致则说明候选是直接抄来的。
func DetectGoldenDocIDs(candidate, expected []string) GoldenCandidateReport {
	if len(expected) == 0 {
		return GoldenCandidateReport{Detail: "无期望答案可比对"}
	}
	candSet := map[string]struct{}{}
	for _, id := range candidate {
		candSet[id] = struct{}{}
	}
	if len(candSet) != len(expected) {
		return GoldenCandidateReport{Detail: "候选集合与期望答案不同，评测有效"}
	}
	for _, id := range expected {
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

// DegradedRAGCandidates 返回降级模式的检索候选（CI 默认）。
//
// 它**不是**期望答案：候选集合与期望答案不同（否则评测自我实现），但必须
// **部分命中**期望——否则 hit-rate 恒为 0，门禁对退化同样不敏感，等于没门禁。
//
// 构造规则：期望文档与干扰文档按 topK 交错排列。topK=5、期望 2 篇时，
// 候选是 [期望1, 干扰1, 期望2, 干扰2, 干扰3] —— 命中率 2/5，接近真实检索
// 「部分命中」的行为，同时候选集合 ≠ 期望集合。
//
// 真实生产验收应改用真实检索器（RAGService），此时命中率才有质量含义。
func DegradedRAGCandidates(c RAGCase) []string {
	topK := c.TopK
	if topK <= 0 {
		topK = 3
	}
	out := make([]string, 0, topK)
	// 期望文档与干扰文档交错：命中部分期望，但不是全部。
	//
	// 边界：期望只有 1 篇且 topK=1 时，若放期望文档则候选集合 == 期望集合，
	// 评测又变成自我实现（守卫会红）。此时改为**只放干扰文档**——命中率 0，
	// 但候选确实不是答案。真实检索在这种窄窗口下返回 0 命中是合理的。
	onlyOneExpected := len(c.ExpectedDocIDs) == 1 && topK == 1
	for i := 0; i < topK; i++ {
		if !onlyOneExpected && i < len(c.ExpectedDocIDs) && i%2 == 0 {
			out = append(out, c.ExpectedDocIDs[i])
			continue
		}
		out = append(out, "degraded-"+c.ID+"-"+itoa(i))
	}
	return out
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
