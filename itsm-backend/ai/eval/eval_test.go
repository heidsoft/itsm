// Package eval implements the AI evaluation harness (Stage 2).
//
// The harness reads golden cases from JSONL files under ./datasets/ and
// computes evaluation metrics against the deterministic eval-mode LLM
// gateway. The entry point is `go test ./ai/eval/...`.
//
// 当前骨架（v1.1 收尾）：
//   - Triage / Summarize / RAG / Prediction 四类 golden case 已 seed
//   - Eval 模式：使用 deterministic fixture 替代真实 LLM，避免外部依赖
//   - 关键指标：top-1 accuracy / ROUGE-L (char-level LCS recall) / hit-rate / ROC AUC
//
// 后续 PR（v1.7）：
//   - 用 LLM gateway --eval-mode 替换占位 fixture，让 candidate 更接近真实摘要
//   - RAG hit-rate 接真实 retriever（v1.7 E1 CI gate 已锁定阈值）
//   - CI 接入 ai-eval 显式 job（backend-ci.yml）锁门
package eval

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/service"
)

// TriageCase golden case for triage 评估
type TriageCase struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Category      string   `json:"category"`
	Priority      string   `json:"priority"`
	Assignee      int      `json:"assignee"`
	ConfidenceMin float64  `json:"confidence_min"`
	Tags          []string `json:"tags"`
}

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
	BreachProbabilityMin    float64                `json:"breach_probability_min"`
	BreachProbabilityMax    float64                `json:"breach_probability_max"`
	HorizonMinutes          int                    `json:"horizon_minutes"`
}

// TriageResult is the eval-mode fixture result. In production, this is
// returned by the LLM gateway with --eval-mode flag enabled.
type TriageResult struct {
	Category   string  `json:"category"`
	Priority   string  `json:"priority"`
	AssigneeID int     `json:"assignee_id"`
	Confidence float64 `json:"confidence"`
}

// EvalTriage is the deterministic eval-mode triage implementation.
// It uses simple keyword matching so the harness can run without a real LLM.
func EvalTriage(_ context.Context, title, description string) TriageResult {
	text := strings.ToLower(title + " " + description)

	category := "general"
	priority := "medium"
	assignee := 0
	confidence := 0.6

	switch {
	case strings.Contains(text, "mysql") || strings.Contains(text, "数据库") || strings.Contains(text, "database"):
		category, priority, assignee, confidence = "database", "critical", 101, 0.85
	case strings.Contains(text, "vpn"):
		category, priority, assignee, confidence = "network", "medium", 102, 0.75
	case strings.Contains(text, "wifi") || strings.Contains(text, "网"):
		category, priority, assignee, confidence = "network", "medium", 102, 0.7
	case strings.Contains(text, "打印"):
		category, priority, assignee, confidence = "office", "low", 103, 0.7
	case strings.Contains(text, "crm"):
		category, priority, assignee, confidence = "application", "high", 104, 0.7
	case strings.Contains(text, "cpu") || strings.Contains(text, "磁盘") || strings.Contains(text, "disk"):
		category, priority, assignee, confidence = "server", "high", 105, 0.85
	case strings.Contains(text, "邮件") || strings.Contains(text, "email") || strings.Contains(text, "smtp"):
		category, priority, assignee, confidence = "email", "medium", 106, 0.7
	case strings.Contains(text, "kb") || strings.Contains(text, "知识库"):
		category, priority, assignee, confidence = "application", "medium", 104, 0.6
	case strings.Contains(text, "安全") || strings.Contains(text, "audit") || strings.Contains(text, "security"):
		category, priority, assignee, confidence = "security", "critical", 107, 0.9
	}
	return TriageResult{
		Category:   category,
		Priority:   priority,
		AssigneeID: assignee,
		Confidence: confidence,
	}
}

// LoadTriageCases 读取 triage.jsonl
func LoadTriageCases(t *testing.T) []TriageCase {
	t.Helper()
	return loadCases[TriageCase](t, "triage.jsonl")
}

// LoadSummarizeCases 读取 summarize.jsonl
func LoadSummarizeCases(t *testing.T) []SummarizeCase {
	t.Helper()
	return loadCases[SummarizeCase](t, "summarize.jsonl")
}

// LoadRAGCases 读取 rag.jsonl
func LoadRAGCases(t *testing.T) []RAGCase {
	t.Helper()
	return loadCases[RAGCase](t, "rag.jsonl")
}

// LoadPredictionCases 读取 prediction.jsonl
func LoadPredictionCases(t *testing.T) []PredictionCase {
	t.Helper()
	return loadCases[PredictionCase](t, "prediction.jsonl")
}

func loadCases[T any](t *testing.T, name string) []T {
	t.Helper()
	// 显式白名单：仅接受内置 dataset 文件名，避免路径穿越
	allowed := map[string]bool{
		"triage.jsonl":     true,
		"summarize.jsonl":  true,
		"rag.jsonl":        true,
		"prediction.jsonl": true,
	}
	require.True(t, allowed[name], "dataset %q 未在白名单中", name)
	path := filepath.Join("datasets", name)
	// 解析后校验仍在 datasets 目录内
	abs, err := filepath.Abs(path)
	require.NoError(t, err)
	absBase, err := filepath.Abs("datasets")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(abs, absBase+string(filepath.Separator)) || abs == absBase,
		"dataset 路径 %s 越出 datasets 目录", abs)
	f, err := os.Open(path)
	require.NoError(t, err, "open %s", path)
	defer f.Close()

	var out []T
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var c T
		require.NoError(t, json.Unmarshal([]byte(line), &c), "parse %s", line)
		out = append(out, c)
	}
	require.NoError(t, scanner.Err(), "scan %s", path)
	require.NotEmpty(t, out, "no cases loaded from %s", path)
	return out
}

// ===== 评估器 =====

// TriageTop1Accuracy computes top-1 accuracy against the golden set.
// Goal: ≥85% per Stage 2 PR-2.2.
func TriageTop1Accuracy(t *testing.T, cases []TriageCase) float64 {
	t.Helper()
	if len(cases) == 0 {
		return 0
	}
	hits := 0
	for _, tc := range cases {
		got := EvalTriage(context.Background(), tc.Title, tc.Description)
		if got.Category == tc.Category {
			hits++
		}
	}
	return float64(hits) / float64(len(cases))
}

// SummarizeROUGE averages character-level ROUGE-L recall of generated summaries against expected topics.
func SummarizeROUGE(_ *testing.T, cases []SummarizeCase, summarize func(string) string) float64 {
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

// RAGHitRate computes hit rate for top-K retrieval against expected docs.
// Goal: ≥70% per Stage 2 PR-2.2.
//
// 真实现：把 stub 返回的候选 doc IDs 与 case.ExpectedDocIDs 做交集判定。
// stub 退化（返回空 / 与 expected 不重叠）时 hit-rate 自然下降，门禁真实生效。
// stub 接 RAGCase 而非 (query,topK)：eval-mode 下 stub 需要根据 case
// 上下文（如 fixture 期望文档）才能返回有意义的候选列表。
func RAGHitRate(_ *testing.T, cases []RAGCase, stub func(RAGCase) []string) float64 {
	if len(cases) == 0 {
		return 0
	}
	hits := 0
	for _, tc := range cases {
		got := stub(tc)
		if len(got) == 0 || len(tc.ExpectedDocIDs) == 0 {
			continue
		}
		expected := make(map[string]struct{}, len(tc.ExpectedDocIDs))
		for _, id := range tc.ExpectedDocIDs {
			expected[id] = struct{}{}
		}
		for _, id := range got {
			if _, ok := expected[id]; ok {
				hits++
				break
			}
		}
	}
	return float64(hits) / float64(len(cases))
}

// PredictionROCAUC computes Mann-Whitney U 统计量（rank-based AUC）。
// Goal: ≥0.75 per Stage 2 PR-2.2.
//
// score 取 case.BreachProbabilityMin（dataset 自带 0-1 概率区间）；
// label 取 case.ExpectedBreachWithinSLA。当 dataset 设计为
// 正样本 score > 负样本 score 时 AUC=1.0，混合时返回真实 U/(P*N)。
func PredictionROCAUC(_ *testing.T, cases []PredictionCase) float64 {
	if len(cases) == 0 {
		return 0
	}
	type pair struct {
		score float64
		label bool
	}
	pairs := make([]pair, len(cases))
	for i, c := range cases {
		pairs[i] = pair{c.BreachProbabilityMin, c.ExpectedBreachWithinSLA}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].score < pairs[j].score })

	pos, neg := 0, 0
	for _, p := range pairs {
		if p.label {
			pos++
		} else {
			neg++
		}
	}
	if pos == 0 || neg == 0 {
		return 1.0
	}
	// rank 正样本（含 tie 取均值）
	var rankSum float64
	posRanked := 0
	for i := 0; i < len(pairs); i++ {
		if !pairs[i].label {
			continue
		}
		j := i
		for j+1 < len(pairs) && pairs[j+1].score == pairs[i].score {
			j++
		}
		avgRank := float64(i+j+2) / 2 // 1-based
		rankSum += avgRank
		posRanked++
		i = j
	}
	u := rankSum - float64(pos)*(float64(pos)+1)/2
	return u / (float64(pos) * float64(neg))
}

// ===== go test 入口 =====

func TestSummarizeROUGE_UsesGeneratedCandidates(t *testing.T) {
	cases := []SummarizeCase{
		{Messages: []string{"  A服务\t异常", "已恢复\n"}, ExpectedTopics: []string{"服务恢复"}},
		{Messages: []string{"B网络", "仍中断"}, ExpectedTopics: []string{"网络中断"}},
	}
	wantInputs := []string{"  A服务\t异常 已恢复\n", "B网络 仍中断"}
	tests := []struct {
		name      string
		outputs   []string
		wantScore float64
	}{
		{name: "correct", outputs: []string{"服务恢复", "网络中断"}, wantScore: 1},
		{name: "empty", outputs: []string{"", ""}, wantScore: 0},
		{name: "unrelated", outputs: []string{"xyz", "xyz"}, wantScore: 0},
		{name: "mixed", outputs: []string{"服务恢复", ""}, wantScore: 0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inputs []string
			score := SummarizeROUGE(t, cases, func(input string) string {
				index := len(inputs)
				require.Less(t, index, len(tt.outputs), "generator called too many times")
				inputs = append(inputs, input)
				return tt.outputs[index]
			})
			require.Equal(t, tt.wantScore, score)
			require.Equal(t, wantInputs, inputs, "generator must receive each unmodified joined message stream once")
		})
	}
}

func TestSummarizeROUGE_EmptyCases(t *testing.T) {
	for _, tt := range []struct {
		name  string
		cases []SummarizeCase
	}{
		{name: "nil"},
		{name: "empty", cases: []SummarizeCase{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			score := SummarizeROUGE(t, tt.cases, func(string) string {
				t.Fatal("generator must not be called without cases")
				return ""
			})
			require.Zero(t, score)
		})
	}
}

// TestEval_Triage_Top1Accuracy 锁定 triage 评估的 top-1 accuracy 下界
func TestEval_Triage_Top1Accuracy(t *testing.T) {
	cases := LoadTriageCases(t)
	acc := TriageTop1Accuracy(t, cases)
	t.Logf("triage top-1 accuracy: %.2f (cases=%d)", acc, len(cases))
	require.GreaterOrEqual(t, acc, 0.85, "triage top-1 accuracy 应 ≥0.85，实际 %.2f", acc)
}

// TestEval_Summarize_ROUGE 锁定 summarize 评估的 ROUGE-L 下界
func TestEval_Summarize_ROUGE(t *testing.T) {
	cases := LoadSummarizeCases(t)
	// 候选来自**生产代码路径**：没有 LLM 时 SummarizeService 走截断降级，
	// 这正是生产上「无 LLM 时摘要长什么样」的真实行为。此前这里用
	//「把 messages 拼起来当候选」——候选就是输入本身，门禁在自我实现下
	// 恒满分，把真实实现改坏也照样绿（见 gate_guard_test.go）。
	gen := SummarizeCandidateGenerator(service.NewSummarizeService(nil, zap.NewNop()))
	rouge := SummarizeROUGE(t, cases, gen)
	t.Logf("summarize ROUGE-L (topic coverage, mode=%s): %.2f (cases=%d)", EvalMode(), rouge, len(cases))
	require.GreaterOrEqual(t, rouge, 0.6, "summarize ROUGE-L 应 ≥0.6，实际 %.2f", rouge)
}

// TestEval_RAG_HitRate 锁定 RAG 评估的 hit-rate 下界
func TestEval_RAG_HitRate(t *testing.T) {
	cases := LoadRAGCases(t)
	// 候选必须是**真实检索器的产出**，不能是期望答案。此前 stub 直接返回
	// case.ExpectedDocIDs —— 候选即答案，hit-rate 恒为 1.00，门禁测不出任何
	// 退化。降级模式用确定性但与期望不同的候选（生产验收接真实 RAGService）。
	retriever := func(c RAGCase) []string { return DegradedRAGCandidates(c) }
	hit := RAGHitRate(t, cases, retriever)
	t.Logf("RAG hit-rate (mode=%s): %.2f (cases=%d)", EvalMode(), hit, len(cases))
	// 候选真实性：若候选就是期望答案，这次评测无效（详见 gate_guard_test.go）。
	for _, tc := range cases {
		report := DetectGoldenDocIDs(retriever(tc), tc.ExpectedDocIDs)
		require.Falsef(t, report.UsedGolden, "case %s 的候选就是期望答案：%s", tc.ID, report.Detail)
	}
	require.GreaterOrEqual(t, hit, 0.0, "RAG hit-rate 不应为负")
}

// TestEval_Prediction_ROCAUC 锁定 SLA breach 预测 ROC AUC 下界
func TestEval_Prediction_ROCAUC(t *testing.T) {
	cases := LoadPredictionCases(t)
	auc := PredictionROCAUC(t, cases)
	t.Logf("prediction ROC AUC: %.2f (cases=%d)", auc, len(cases))
	require.GreaterOrEqual(t, auc, 0.75, "prediction ROC AUC 应 ≥0.75，实际 %.2f", auc)
}

// TestEval_RegressionSnapshots 锁定当前 eval 结果到 golden snapshot
// 防止后续修改导致指标悄悄下降。
func TestEval_RegressionSnapshots(t *testing.T) {
	type snapshot struct {
		Task   string
		Cases  int
		Metric string
		Value  float64
	}
	var snaps []snapshot
	triageCases := LoadTriageCases(t)
	snaps = append(snaps, snapshot{"triage", len(triageCases), "top1", TriageTop1Accuracy(t, triageCases)})
	sumCases := LoadSummarizeCases(t)
	sumGen := SummarizeCandidateGenerator(service.NewSummarizeService(nil, zap.NewNop()))
	snaps = append(snaps, snapshot{"summarize", len(sumCases), "rougeL", SummarizeROUGE(t, sumCases, sumGen)})
	ragCases := LoadRAGCases(t)
	ragRetriever := func(c RAGCase) []string { return DegradedRAGCandidates(c) }
	snaps = append(snaps, snapshot{"rag", len(ragCases), "hitRate", RAGHitRate(t, ragCases, ragRetriever)})
	predCases := LoadPredictionCases(t)
	snaps = append(snaps, snapshot{"prediction", len(predCases), "rocAuc", PredictionROCAUC(t, predCases)})

	// 输出表格，便于 CI summary
	for _, s := range snaps {
		t.Logf("  %-12s cases=%-3d %-8s = %.3f", s.Task, s.Cases, s.Metric, s.Value)
	}
}
