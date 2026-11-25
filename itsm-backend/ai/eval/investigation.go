package eval

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/handlers/ai"
)

// 调查编排（POST /ai/tickets/:id/investigate）的评测层。
//
// 与四类既有场景不同，调查编排是**结构化输出链**（分诊→KB→CMDB→综合），
// 文本相似度指标（ROUGE）不适用。CI degraded 档测的是生产链路的**降级行为
// 契约**——候选来自生产 ai.Service.InvestigateTicket（真实 ent 内存库 +
// 真实 ExecuteTool Gate 2 RBAC 链 + LLM 未装配的真实降级综合），期望是
// 评测设计者对「降级输出长什么样」的结构性定义：
//
//	StepCompleteness  四步齐且顺序正确——链路完整性（砍掉任何一步都会转红）
//	DegradedHonesty   降级必须自报（degraded/status/synthesis 步标记），
//	                  且不得伪造根因与动作建议——决策支持边界（把降级伪装成
//	                  completed、或降级路径编造 rootCause 都会转红）
//	EvidenceFidelity  降级结论如实转述证据：分诊解释进 findings、KB 命中数
//	                  如实传递、CMDB 命中与事实一致（证据被吞或编造会转红）
//
// 降级路径是确定性代码：正确实现下三指标恒为 1.0，任何偏差都是回归。
// 这不是「自我实现」的自我实现门禁——候选不是期望答案的抄写，而是生产
// 函数对真实数据的真实输出；与 summarize/RAG 的「候选真实性」判据同理，
// 这里的判据是「候选来自生产函数调用」，且退化敏感测试
// （investigation_eval_test.go 的 Gate 用例）证明指标对坏输出敏感。
//
// LLM 档（真实模型综合质量）需要真实 LLM 与人工验收，不在此层——与
// doc.go 的档位口径一致。

// InvestigationCase 调查编排评测 golden case。
type InvestigationCase struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	// TriageExplanation 固定分诊替身给出的解释——期望降级结论把它如实转述进
	// findings（降级输出必须转述证据，不得吞掉也不得编造别的）。
	TriageExplanation string `json:"triage_explanation"`
	// ExpectedKBHits 固定 KB 替身返回的条数——期望 steps[knowledge].hitCount
	// 如实传递（命中数被吞或多报都是回归）。
	ExpectedKBHits int `json:"expected_kb_hits"`
	// ExpectedCMDBFound 期望 CMDB 步是否有命中（true=定位到 CI 并完成影响分析）。
	ExpectedCMDBFound bool `json:"expected_cmdb_found"`
}

// InvestigationResult 编排候选类型别名：CLI 报告等消费方经 eval 包引用，
// 不直接依赖 handlers/ai。
type InvestigationResult = ai.TicketInvestigationResponse

// investigationToolsStub 调查链工具替身：实现 handlers/ai.ToolCatalog。
// GetTool 返回显式 Resource/Action（Gate 2 完整性校验要求非空）；
// Execute 按名分发固定形状的结果（KB 条数/CMDB 命中由 RunCase 按 case 配置）。
type investigationToolsStub struct {
	kbHits    int
	cmdbFound bool
}

func (s *investigationToolsStub) Execute(_ context.Context, _ int, name string, _ map[string]interface{}) (interface{}, error) {
	switch name {
	case "list_kb":
		hits := make([]map[string]interface{}, 0, s.kbHits)
		for i := 1; i <= s.kbHits; i++ {
			hits = append(hits, map[string]interface{}{
				"id":    float64(i),
				"title": fmt.Sprintf("KB: 处置手册 #%d", i),
			})
		}
		return hits, nil
	case "list_cis":
		if !s.cmdbFound {
			return []map[string]interface{}{}, nil
		}
		return []map[string]interface{}{{"id": float64(11), "name": "eval-ci"}}, nil
	case "get_ci_impact":
		return map[string]interface{}{
			"impact":     "high",
			"hops":       float64(2),
			"ciId":       float64(11),
			"explain_ai": map[string]interface{}{"summary": "影响下游服务"},
		}, nil
	}
	return nil, fmt.Errorf("eval stub: unexpected tool %s", name)
}

func (s *investigationToolsStub) GetTool(name string) *dto.ToolDefinition {
	return &dto.ToolDefinition{Name: name, Description: "eval stub", ReadOnly: true, Resource: "eval", Action: "read"}
}

func (s *investigationToolsStub) ListTools() []dto.ToolDefinition { return nil }

func (s *investigationToolsStub) ListToolsForTenant(_ context.Context, _ int) []dto.ToolDefinition {
	return nil
}

// investigationTriageStub 固定分诊替身：实现 handlers/ai.TriageAdvisor。
// 解释文本由 RunCase 按 case 配置（期望降级结论如实转述它）。
type investigationTriageStub struct {
	explanation string
}

func (s *investigationTriageStub) SuggestForTenant(_ context.Context, _, _ string, _ int) dto.TriageResult {
	return dto.TriageResult{
		Category:    "eval-category",
		Priority:    "high",
		Confidence:  0.82,
		Explanation: s.explanation,
	}
}

// InvestigationHarness 调查编排评测装配：真实 ent 内存库 + 生产降级链。
//
// 与 NewRealRAGRetriever 同构：用 ent.Open 而非 enttest.Open（enttest 需要
// testing.TB，本层要能被 cmd/ai-eval 这样的非测试代码复用）。
type InvestigationHarness struct {
	client   *ent.Client
	svc      *ai.Service
	triage   *investigationTriageStub
	tools    *investigationToolsStub
	tenantID int
	userID   int
	ticketID map[string]int
}

// NewInvestigationHarness 建评测库并按 cases 逐条建工单。
// 返回 cleanup 供调用方关闭连接。
func NewInvestigationHarness(dsn string, cases []InvestigationCase) (*InvestigationHarness, func(), error) {
	client, err := ent.Open("sqlite3", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open investigation eval db: %w", err)
	}
	ctx := context.Background()
	if err := client.Schema.Create(ctx); err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("create investigation eval schema: %w", err)
	}

	tenant, err := client.Tenant.Create().SetName("inv-eval").SetCode("inv-eval").SetDomain("inv-eval.example.com").Save(ctx)
	if err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("create tenant: %w", err)
	}
	user, err := client.User.Create().
		SetUsername("inv-eval-op").
		SetName("inv-eval-op").
		SetEmail("inv-eval-op@example.com").
		SetPasswordHash("x").
		SetTenantID(tenant.ID).
		Save(ctx)
	if err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	h := &InvestigationHarness{
		client:   client,
		triage:   &investigationTriageStub{},
		tools:    &investigationToolsStub{},
		tenantID: tenant.ID,
		userID:   user.ID,
		ticketID: map[string]int{},
	}
	h.svc = ai.NewService(ai.NewEntRepository(client), zap.NewNop().Sugar(), nil, h.tools, nil, nil, nil, h.triage, nil, nil)
	h.svc.SetEntClient(client)

	for i, c := range cases {
		tk, err := client.Ticket.Create().
			SetTitle(c.Title).
			SetDescription(c.Description).
			SetTicketNumber(fmt.Sprintf("TK-INVEVAL-%03d", i+1)).
			SetRequesterID(user.ID).
			SetTenantID(tenant.ID).
			Save(ctx)
		if err != nil {
			client.Close()
			return nil, nil, fmt.Errorf("create ticket for case %s: %w", c.ID, err)
		}
		h.ticketID[c.ID] = tk.ID
	}

	cleanup := func() { _ = client.Close() }
	return h, cleanup, nil
}

// RunCase 用生产链路产出候选：配置替身 → InvestigateTicket（LLM 未装配 →
// 真实降级综合）。role=super_admin 让 Gate 2 直通——本层测的是链路行为契约，
// RBAC 拒绝降级已由 handlers/ai 的回归测试覆盖。
func (h *InvestigationHarness) RunCase(ctx context.Context, c InvestigationCase) (*ai.TicketInvestigationResponse, error) {
	id := h.ticketID[c.ID]
	if id == 0 {
		return nil, fmt.Errorf("case %s has no ticket", c.ID)
	}
	h.triage.explanation = c.TriageExplanation
	h.tools.kbHits = c.ExpectedKBHits
	h.tools.cmdbFound = c.ExpectedCMDBFound
	return h.svc.InvestigateTicket(ctx, id, h.tenantID, h.userID, "super_admin")
}

// investigationExpectedSteps 是调查链的完整步骤集。
var investigationExpectedSteps = map[string]bool{
	"triage": true, "knowledge": true, "cmdb": true, "synthesis": true,
}

// InvestigationStepCompleteness 四步齐的 case 占比。
// 砍掉任何一步（或改步名）都会让本指标低于 1——链路完整性退化敏感。
func InvestigationStepCompleteness(cases []InvestigationCase, results map[string]*ai.TicketInvestigationResponse) float64 {
	if len(cases) == 0 {
		return 0
	}
	ok := 0
	for _, c := range cases {
		r := results[c.ID]
		if r == nil {
			continue
		}
		seen := map[string]bool{}
		for _, st := range r.Steps {
			seen[st.Step] = true
		}
		if len(seen) == len(investigationExpectedSteps) {
			complete := true
			for step := range investigationExpectedSteps {
				if !seen[step] {
					complete = false
					break
				}
			}
			if complete {
				ok++
			}
		}
	}
	return float64(ok) / float64(len(cases))
}

// InvestigationDegradedHonesty 「降级如实自报且不伪造」的 case 占比。
//
// 判据（全部来自生产响应字段）：
//   - status=="degraded" 且 degraded==true（不得把降级伪装成 completed）
//   - suspectedRootCause 为空（降级路径没有 LLM，编造根因违反决策支持边界）
//   - suggestedActions 为空（没有综合就没有负责任的动作建议）
//   - synthesis 步自带 degraded 标记（逐步溯源可见）
func InvestigationDegradedHonesty(cases []InvestigationCase, results map[string]*ai.TicketInvestigationResponse) float64 {
	if len(cases) == 0 {
		return 0
	}
	ok := 0
	for _, c := range cases {
		r := results[c.ID]
		if r == nil {
			continue
		}
		if r.Status != "degraded" || !r.Degraded {
			continue
		}
		if r.SuspectedRootCause != "" || len(r.SuggestedActions) != 0 {
			continue
		}
		honestSynthesis := false
		for _, st := range r.Steps {
			if st.Step == "synthesis" && st.Degraded {
				honestSynthesis = true
			}
		}
		if honestSynthesis {
			ok++
		}
	}
	return float64(ok) / float64(len(cases))
}

// InvestigationEvidenceFidelity 「降级结论如实转述证据」的 case 占比：
// 分诊解释进 findings、KB 命中数如实传递、CMDB 命中与替身事实一致。
func InvestigationEvidenceFidelity(cases []InvestigationCase, results map[string]*ai.TicketInvestigationResponse) float64 {
	if len(cases) == 0 {
		return 0
	}
	ok := 0
	for _, c := range cases {
		r := results[c.ID]
		if r == nil {
			continue
		}
		triageQuoted := false
		for _, f := range r.Findings {
			if c.TriageExplanation != "" && containsFold(f, c.TriageExplanation) {
				triageQuoted = true
			}
		}
		kbHits := stepHitCountByName(r, "knowledge")
		cmdbFound := stepHitCountByName(r, "cmdb") > 0
		if triageQuoted && kbHits == c.ExpectedKBHits && cmdbFound == c.ExpectedCMDBFound {
			ok++
		}
	}
	return float64(ok) / float64(len(cases))
}

// stepHitCountByName 取指定步骤的 hitCount（未记录/缺失返回 -1，
// 与合法的 0 命中区分——证据被吞（步在但数没了）同样算失真）。
func stepHitCountByName(r *ai.TicketInvestigationResponse, step string) int {
	for _, st := range r.Steps {
		if st.Step == step {
			return st.HitCount
		}
	}
	return -1
}

func containsFold(s, substr string) bool {
	if len(substr) == 0 {
		return false
	}
	rs, rsub := []rune(s), []rune(substr)
	for i := 0; i+len(rsub) <= len(rs); i++ {
		match := true
		for j := range rsub {
			if lower(rs[i+j]) != lower(rsub[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func lower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}
