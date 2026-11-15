package eval

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/handlers/ai"
)

// 调查编排评测：CI degraded 档锁定生产**降级路径**的结构契约。
//
// 候选真实性判据：候选 = 生产 ai.Service.InvestigateTicket 对真实 ent 内存库
// 的真实输出（真实 ExecuteTool Gate 2 链 + 真实降级综合），不是期望答案的
// 抄写——本文件末尾的 Gate 用例证明三个指标对坏输出全部敏感，把生产代码
// 改坏（砍步骤/伪造 completed/吞证据）必然转红。

// LoadInvestigationCases 读取 investigation.jsonl（复用包内 loadCases 泛型加载，
// 含路径白名单防穿越）。
func LoadInvestigationCases(t *testing.T) []InvestigationCase {
	t.Helper()
	cases := loadCases[InvestigationCase](t, "investigation.jsonl")
	require.NotEmpty(t, cases, "investigation 数据集不得为空")
	return cases
}

// runInvestigationDataset 用生产降级链跑完整数据集，返回 case.ID → 响应。
func runInvestigationDataset(t *testing.T, cases []InvestigationCase) map[string]*ai.TicketInvestigationResponse {
	t.Helper()
	h, cleanup, err := NewInvestigationHarness(
		"file:aieval_inv?mode=memory&cache=shared&_fk=1", cases)
	require.NoError(t, err)
	t.Cleanup(cleanup)

	results := map[string]*ai.TicketInvestigationResponse{}
	for _, c := range cases {
		r, err := h.RunCase(context.Background(), c)
		require.NoErrorf(t, err, "case %s 调查失败", c.ID)
		require.NotNil(t, r)
		results[c.ID] = r
	}
	return results
}

// TestEval_Investigation_DegradedPathContract 锁定降级路径三个结构指标的
// 下界。降级路径是确定性代码：正确实现恒为 1.0，任何偏差都是回归——
// 这与 summarize/RAG 的「下界 + 可能波动」不同，是「恒等式」门禁。
func TestEval_Investigation_DegradedPathContract(t *testing.T) {
	cases := LoadInvestigationCases(t)
	results := runInvestigationDataset(t, cases)

	completeness := InvestigationStepCompleteness(cases, results)
	honesty := InvestigationDegradedHonesty(cases, results)
	fidelity := InvestigationEvidenceFidelity(cases, results)
	t.Logf("investigation degraded: completeness=%.2f honesty=%.2f fidelity=%.2f (cases=%d)",
		completeness, honesty, fidelity, len(cases))

	// 链路完整性：triage → knowledge → cmdb → synthesis 四步齐。
	require.GreaterOrEqual(t, completeness, 1.0, "步骤完整率应 = 1.0（确定性路径），实际 %.2f", completeness)
	// 决策支持边界：降级自报、不伪造根因与动作。
	require.GreaterOrEqual(t, honesty, 1.0, "降级诚实率应 = 1.0，实际 %.2f", honesty)
	// 证据转述：分诊解释进 findings、KB 命中数如实、CMDB 命中与事实一致。
	require.GreaterOrEqual(t, fidelity, 1.0, "证据保真率应 = 1.0，实际 %.2f", fidelity)

	// 候选真实性（机制性判据，与 RAG 的 query-count 同思路）：
	// 生产响应必须带溯源字段且与替身事实对得上——候选不是静态期望的抄写。
	for _, c := range cases {
		r := results[c.ID]
		require.Equal(t, "investigation-v1", r.PromptVersion, "候选必须来自生产链路（promptVersion 溯源）")
		require.Equal(t, "degraded", r.Status, "LLM 未装配时生产链路必须自报 degraded")
	}
}

// TestGate_InvestigationMetricsDetectDegradation 退化敏感证明（对齐
// gate_guard_test.go 的门禁有效性原则）：构造三类坏响应，证明三个指标
// 分别会把它们判为不达标——门禁测得出退化，不是恒满分摆设。
func TestGate_InvestigationMetricsDetectDegradation(t *testing.T) {
	cases := []InvestigationCase{{
		ID: "c1", Title: "t", Description: "d",
		TriageExplanation: "关键词命中存储", ExpectedKBHits: 2, ExpectedCMDBFound: true,
	}}
	good := &ai.TicketInvestigationResponse{
		Status: "degraded", Degraded: true, PromptVersion: "investigation-v1",
		Steps: []ai.InvestigationStep{
			{Step: "triage"},
			{Step: "knowledge", HitCount: 2},
			{Step: "cmdb", HitCount: 1},
			{Step: "synthesis", Degraded: true},
		},
		Findings: []string{"Triage: 关键词命中存储"},
	}
	results := map[string]*ai.TicketInvestigationResponse{"c1": good}

	// 良性基线：三指标满分。
	assert.InDelta(t, 1.0, InvestigationStepCompleteness(cases, results), 1e-9)
	assert.InDelta(t, 1.0, InvestigationDegradedHonesty(cases, results), 1e-9)
	assert.InDelta(t, 1.0, InvestigationEvidenceFidelity(cases, results), 1e-9)

	// 退化 1：砍掉 synthesis 步 → 完整率转红。
	missing := *good
	missing.Steps = good.Steps[:3]
	assert.Less(t, InvestigationStepCompleteness(cases, map[string]*ai.TicketInvestigationResponse{"c1": &missing}), 1.0,
		"砍掉 synthesis 步必须让完整率下降——门禁必须对链路残缺敏感")

	// 退化 2：降级伪装 completed + 编造根因与动作 + 综合步不再自报降级 → 诚实率转红。
	faked := *good
	faked.Status = "completed"
	faked.Degraded = false
	faked.SuspectedRootCause = "编造的根因"
	faked.Steps = []ai.InvestigationStep{
		{Step: "triage"}, {Step: "knowledge", HitCount: 2},
		{Step: "cmdb", HitCount: 1}, {Step: "synthesis"},
	}
	assert.Less(t, InvestigationDegradedHonesty(cases, map[string]*ai.TicketInvestigationResponse{"c1": &faked}), 1.0,
		"降级伪装 completed 或编造根因必须让诚实率下降")

	// 退化 3：吞证据（KB 命中数丢失）→ 保真率转红。
	muted := *good
	muted.Steps = []ai.InvestigationStep{
		{Step: "triage"},
		{Step: "knowledge", HitCount: 0},
		{Step: "cmdb", HitCount: 1},
		{Step: "synthesis", Degraded: true},
	}
	assert.Less(t, InvestigationEvidenceFidelity(cases, map[string]*ai.TicketInvestigationResponse{"c1": &muted}), 1.0,
		"KB 命中数被吞必须让保真率下降——证据转述不得失真")
}
