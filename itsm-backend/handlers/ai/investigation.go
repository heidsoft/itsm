package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
)

// 工单调查编排（AI-Native 多步链，决策支持而非自主授权）。
//
// 把既有原语串成一条确定性的调查链：triage（Guidance→LLM→keyword 三级分诊）
// → list_kb（知识库检索）→ list_cis + get_ci_impact（CMDB 影响分析）→ LLM
// 综合成结构化结论，全程：
//   - 每个工具步骤经 ExecuteTool：Gate 2 工具级 RBAC + 审计（approval_state=auto）；
//     权限不足/注册表缺失只降级该步骤，不中断整条链；
//   - 综合结论携带 promptVersion/model/confidence 溯源，落服务端决策底账
//     （scenario="investigation"）与 ai_analysis_results（analysis_type=
//     "investigation"，每次全量重跑并落历史——调查对 CMDB/知识库新鲜度敏感，
//     不做按工单幂等复用，与分诊的 cached 语义刻意不同）；
//   - suggestedActions 只出**建议**且必须命中工具白名单，写操作由操作者经
//     POST /agent/tools/execute 走既有审批流（Gate 3），编排器绝不静默执行。
//
// LLM 综合失败/未装配时降级为确定性拼接（degraded=true），不伪造根因。

const (
	investigationAnalysisType  = "investigation"
	investigationPromptVersion = "investigation-v1"
	investigationKBLimit       = 5
	investigationCISearchLimit = 3
	investigationImpactHops    = 2
)

// investigationActionWhitelist 是 suggestedActions 允许引用的写工具。
// 白名单外的动作一律丢弃——综合模型自由发挥出的未知动作不得进入契约。
var investigationActionWhitelist = map[string]bool{
	"create_ticket":    true,
	"update_ticket":    true,
	"link_ticket_ci":   true,
	"create_ci_relationship": true,
	"delete_ci_relationship": true,
}

// InvestigationStep 调查链单步的溯源信息。
type InvestigationStep struct {
	Step           string      `json:"step"` // triage | knowledge | cmdb | synthesis
	Tool           string      `json:"tool,omitempty"`
	Degraded       bool        `json:"degraded,omitempty"`
	DegradedReason string      `json:"degradedReason,omitempty"`
	HitCount       int         `json:"hitCount,omitempty"`
	Detail         interface{} `json:"detail,omitempty"`
}

// InvestigationSuggestedAction 综合结论里的后续动作建议。
// 只是建议：执行必须由操作者显式发起并走写工具审批流。
type InvestigationSuggestedAction struct {
	Action           string `json:"action"`
	Description      string `json:"description"`
	Tool             string `json:"tool,omitempty"`
	RequiresApproval bool   `json:"requiresApproval"`
}

// TicketInvestigationResponse 调查编排的响应契约（camelCase）。
type TicketInvestigationResponse struct {
	TicketID           int                            `json:"ticketId"`
	TicketNumber       string                         `json:"ticketNumber,omitempty"`
	Status             string                         `json:"status"` // completed | degraded
	Degraded           bool                           `json:"degraded"`
	DegradedReason     string                         `json:"degradedReason,omitempty"`
	PromptVersion      string                         `json:"promptVersion"`
	Model              string                         `json:"model,omitempty"`
	Confidence         float64                        `json:"confidence"`
	LatencyMs          int64                          `json:"latencyMs,omitempty"`
	Steps              []InvestigationStep            `json:"steps"`
	Findings           []string                       `json:"findings,omitempty"`
	SuspectedRootCause string                         `json:"suspectedRootCause,omitempty"`
	ImpactedServices   []string                       `json:"impactedServices,omitempty"`
	SuggestedActions   []InvestigationSuggestedAction `json:"suggestedActions,omitempty"`
	AuditID            int64                          `json:"auditId,omitempty"`
}

// ticketBrief 调查链的工单输入（最小字段集）。
type ticketBrief struct {
	ID           int
	TicketNumber string
	Title        string
	Description  string
	Status       string
	Priority     string
	AssigneeID   int
}

// loadTicketBrief 读取工单关键字段。租户谓词强制；entClient 缺失是装配缺口，
// 用显式错误暴露（handler 映射 503），不是静默空结果。
func (s *Service) loadTicketBrief(ctx context.Context, ticketID, tenantID int) (*ticketBrief, error) {
	if s.entClient == nil {
		return nil, ErrInvestigationUnavailable
	}
	tk, err := s.entClient.Ticket.Query().
		Where(ticket.IDEQ(ticketID), ticket.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		// NotFound 由 handler 映射 404；跨租户访问在租户谓词下同样表现为
		// NotFound，不泄漏存在性。
		if ent.IsNotFound(err) {
			return nil, ErrInvestigationTicketNotFound
		}
		return nil, err
	}
	return &ticketBrief{
		ID:           tk.ID,
		TicketNumber: tk.TicketNumber,
		Title:        tk.Title,
		Description:  tk.Description,
		Status:       tk.Status,
		Priority:     tk.Priority,
		AssigneeID:   tk.AssigneeID,
	}, nil
}

// ErrInvestigationUnavailable 调查编排的依赖未装配（ent client 等）。
var ErrInvestigationUnavailable = errors.New("ai investigation dependencies unavailable")

// ErrInvestigationTicketNotFound 目标工单不存在（含跨租户访问——租户谓词下
// 同样表现为 not found，不泄漏存在性）。handler 映射 404。
var ErrInvestigationTicketNotFound = errors.New("investigation ticket not found")

// InvestigateTicket 执行工单调查编排，返回结构化结论。
func (s *Service) InvestigateTicket(ctx context.Context, ticketID, tenantID, userID int, role string) (*TicketInvestigationResponse, error) {
	started := time.Now()

	brief, err := s.loadTicketBrief(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}

	resp := &TicketInvestigationResponse{
		TicketID:      brief.ID,
		TicketNumber:  brief.TicketNumber,
		PromptVersion: investigationPromptVersion,
		Steps:         []InvestigationStep{},
	}

	query := strings.TrimSpace(brief.Title + " " + brief.Description)

	// Step 1: 分诊（Guidance → LLM → keyword 三级链，自带 Degraded 标注）。
	var triageResult *dto.TriageResult
	if s.triageService != nil {
		tr := s.triageService.SuggestForTenant(ctx, brief.Title, brief.Description, tenantID)
		triageResult = &tr
		resp.Steps = append(resp.Steps, InvestigationStep{
			Step:           "triage",
			Degraded:       tr.Degraded,
			DegradedReason: tr.DegradedReason,
			Detail: map[string]interface{}{
				"category":    tr.Category,
				"priority":    tr.Priority,
				"confidence":  tr.Confidence,
				"explanation": tr.Explanation,
			},
		})
	} else {
		resp.Steps = append(resp.Steps, InvestigationStep{
			Step:           "triage",
			Degraded:       true,
			DegradedReason: "triage service unavailable",
		})
	}

	// Step 2: 知识库检索（经 ExecuteTool：RBAC + 审计；失败降级本步）。
	kbEvidence := ""
	kbHits := 0
	if s.tools != nil {
		res, _, err := s.ExecuteTool(ctx, userID, tenantID, role, "list_kb", map[string]interface{}{
			"q":     query,
			"limit": float64(investigationKBLimit),
		})
		if err != nil {
			resp.Steps = append(resp.Steps, s.degradedToolStep("knowledge", "list_kb", err))
		} else {
			hits := toLooseList(res)
			kbHits = len(hits)
			kbEvidence = compactEvidence(hits, "id,title", investigationEvidenceChars)
			resp.Steps = append(resp.Steps, InvestigationStep{
				Step:     "knowledge",
				Tool:     "list_kb",
				HitCount: kbHits,
				Detail:   compactEvidenceList(hits, "id", "title", investigationKBDetailHits),
			})
		}
	} else {
		resp.Steps = append(resp.Steps, InvestigationStep{
			Step:           "knowledge",
			Degraded:       true,
			DegradedReason: "tool registry unavailable",
		})
	}

	// Step 3: CMDB 影响分析。先按标题检索 CI，命中即取第一个做影响扩散
	//（get_ci_impact 自带 AI 解释层，fail-open）。无命中是事实不是错误。
	ciFound := false
	ciEvidence := ""
	if s.tools != nil {
		res, _, err := s.ExecuteTool(ctx, userID, tenantID, role, "list_cis", map[string]interface{}{
			"search": brief.Title,
			"limit":  float64(investigationCISearchLimit),
		})
		if err != nil {
			resp.Steps = append(resp.Steps, s.degradedToolStep("cmdb", "list_cis", err))
		} else {
			items := toLooseList(res)
			if len(items) == 0 {
				resp.Steps = append(resp.Steps, InvestigationStep{
					Step: "cmdb",
					Tool: "list_cis",
					Detail: map[string]interface{}{
						"found": false,
						"note":  "no matching CI for ticket title",
					},
				})
			} else {
				ciID := looseInt(items[0], "id")
				impactRes, _, err := s.ExecuteTool(ctx, userID, tenantID, role, "get_ci_impact", map[string]interface{}{
					"ciId": float64(ciID),
					"hops": float64(investigationImpactHops),
				})
				if err != nil {
					resp.Steps = append(resp.Steps, s.degradedToolStep("cmdb", "get_ci_impact", err))
				} else {
					ciFound = true
					impactMap, _ := impactRes.(map[string]interface{})
					ciEvidence = compactEvidence([]map[string]interface{}{impactMap}, "impact,explain_ai", investigationEvidenceChars)
					explain, _ := impactMap["explain_ai"].(map[string]interface{})
					resp.Steps = append(resp.Steps, InvestigationStep{
						Step:     "cmdb",
						Tool:     "get_ci_impact",
						HitCount: 1,
						Detail: map[string]interface{}{
							"found":   true,
							"ciId":    ciID,
							"hops":    looseInt(impactMap, "hops"),
							"explain": explain,
						},
					})
				}
			}
		}
	} else {
		resp.Steps = append(resp.Steps, InvestigationStep{
			Step:           "cmdb",
			Degraded:       true,
			DegradedReason: "tool registry unavailable",
		})
	}

	// Step 4: 综合。LLM 优先，失败/未装配降级确定性拼接。
	synthesisStep := s.synthesizeInvestigation(ctx, brief, triageResult, kbEvidence, kbHits, ciFound, ciEvidence, resp)
	resp.Steps = append(resp.Steps, synthesisStep)

	if resp.Degraded {
		resp.Status = "degraded"
	} else {
		resp.Status = "completed"
	}
	resp.LatencyMs = time.Since(started).Milliseconds()

	// 服务端决策底账 + 历史落库（均 best-effort，不阻塞结论返回）。
	resp.AuditID = s.recordInvestigationAudit(ctx, tenantID, userID, brief, resp)
	s.persistInvestigation(ctx, tenantID, userID, brief, resp)

	return resp, nil
}

// degradedToolStep 把工具步骤错误转成可观测的降级步骤：
// 权限拒绝/未知工具如实标注，不把 RBAC 失败伪装成「无命中」。
func (s *Service) degradedToolStep(step, tool string, err error) InvestigationStep {
	reason := "tool execution failed"
	switch {
	case errors.Is(err, ErrToolPermissionDenied):
		reason = "tool permission denied"
	case errors.Is(err, ErrToolUnavailable):
		reason = "tool authorization dependencies unavailable"
	case errors.Is(err, ErrUnknownTool):
		reason = "unknown tool"
	}
	return InvestigationStep{Step: step, Tool: tool, Degraded: true, DegradedReason: reason}
}

// synthesizeInvestigation 综合各步证据产出结论；写入 resp 的
// Findings/RootCause/Impacted/SuggestedActions/Confidence/Degraded。
func (s *Service) synthesizeInvestigation(
	ctx context.Context,
	brief *ticketBrief,
	triage *dto.TriageResult,
	kbEvidence string,
	kbHits int,
	ciFound bool,
	ciEvidence string,
	resp *TicketInvestigationResponse,
) InvestigationStep {
	if s.llmGateway != nil {
		prompt := fmt.Sprintf(`You are an IT service management investigation assistant.
Given the ticket facts and investigation evidence below, produce STRICT JSON only (no markdown fences):

{"findings":["..."],"suspectedRootCause":"...","impactedServices":["..."],"suggestedActions":[{"action":"create_ticket|update_ticket|link_ticket_ci|create_ci_relationship|delete_ci_relationship","description":"..."}],"confidence":0.0}

Rules:
- findings must be grounded in the evidence; if evidence is insufficient, return empty arrays and low confidence.
- suspectedRootCause must be empty string if not determinable from evidence.
- suggestedActions only from the listed whitelist; actions requiring data changes will go through human approval.
- confidence in [0,1] reflects how well the evidence supports the conclusion.

Ticket:
Number: %s
Title: %s
Description: %s
Status: %s
Priority: %s

Triage: category=%s priority=%s confidence=%.2f explanation=%s

Knowledge base hits (%d):
%s

CMDB impact evidence (found=%v):
%s`,
			brief.TicketNumber, brief.Title, truncateForRef(brief.Description, 400), brief.Status, brief.Priority,
			triageCategory(triage), triagePriority(triage), triageConfidence(triage), triageExplanation(triage),
			kbHits, truncateForRef(kbEvidence, investigationEvidenceChars),
			ciFound, truncateForRef(ciEvidence, investigationEvidenceChars))

		messages := []dto.LLMMessage{
			{Role: "system", Content: "You are an IT service management assistant. Output strict JSON."},
			{Role: "user", Content: prompt},
		}
		out, err := s.llmGateway.Chat(ctx, "", messages)
		if err == nil {
			if _, ok := parseInvestigationSynthesis(out, resp); ok {
				resp.Model = s.llmGateway.EffectiveModel("")
				return InvestigationStep{
					Step:     "synthesis",
					Tool:     "llm",
					HitCount: len(resp.Findings),
				}
			}
			s.logger.Warnw("InvestigateTicket: synthesis JSON invalid, falling back",
				"ticket_id", brief.ID)
		} else {
			s.logger.Warnw("InvestigateTicket: LLM synthesis failed, falling back",
				"error", err, "ticket_id", brief.ID)
		}
	}
	deterministicInvestigation(brief, triage, kbHits, ciFound, resp)
	resp.Model = ""
	return InvestigationStep{
		Step:           "synthesis",
		Degraded:       true,
		DegradedReason: "llm synthesis unavailable, deterministic summary only",
	}
}

// parseInvestigationSynthesis 解析综合模型的严格 JSON 输出；解析失败或字段
// 越界返回 false，调用方降级——半成品结论不得进入契约。
func parseInvestigationSynthesis(out string, resp *TicketInvestigationResponse) (*synthesisOutput, bool) {
	start := strings.Index(out, "{")
	end := strings.LastIndex(out, "}")
	if start < 0 || end <= start {
		return nil, false
	}
	var parsed synthesisOutput
	if err := json.Unmarshal([]byte(out[start:end+1]), &parsed); err != nil {
		return nil, false
	}
	if parsed.Confidence < 0 || parsed.Confidence > 1 {
		return nil, false
	}
	resp.Findings = parsed.Findings
	resp.SuspectedRootCause = parsed.SuspectedRootCause
	resp.ImpactedServices = parsed.ImpactedServices
	resp.Confidence = parsed.Confidence
	for _, a := range parsed.SuggestedActions {
		if !investigationActionWhitelist[a.Action] {
			continue
		}
		resp.SuggestedActions = append(resp.SuggestedActions, InvestigationSuggestedAction{
			Action:           a.Action,
			Description:      a.Description,
			Tool:             a.Action,
			RequiresApproval: true,
		})
	}
	resp.Degraded = false
	return &parsed, true
}

// synthesisOutput 综合模型的严格 JSON 目标结构（内部用，输出经白名单过滤）。
type synthesisOutput struct {
	Findings           []string `json:"findings"`
	SuspectedRootCause string   `json:"suspectedRootCause"`
	ImpactedServices   []string `json:"impactedServices"`
	SuggestedActions   []struct {
		Action      string `json:"action"`
		Description string `json:"description"`
	} `json:"suggestedActions"`
	Confidence float64 `json:"confidence"`
}

// deterministicInvestigation 是 LLM 综合不可用时的确定性降级：
// 只如实汇总证据数量与分诊解释，不编造根因/影响面/动作。
func deterministicInvestigation(brief *ticketBrief, triage *dto.TriageResult, kbHits int, ciFound bool, resp *TicketInvestigationResponse) {
	resp.Degraded = true
	resp.DegradedReason = "llm synthesis unavailable, deterministic summary only"
	resp.Confidence = 0
	if triage != nil {
		resp.Confidence = triage.Confidence
		if triage.Explanation != "" {
			resp.Findings = append(resp.Findings, "Triage: "+triage.Explanation)
		}
	}
	if kbHits > 0 {
		resp.Findings = append(resp.Findings, fmt.Sprintf("Knowledge base returned %d candidate articles (see steps[knowledge].detail)", kbHits))
	}
	if ciFound {
		resp.Findings = append(resp.Findings, "CMDB impact analysis completed (see steps[cmdb].detail)")
	}
}

// recordInvestigationAudit 落服务端决策底账（best-effort，失败仅告警）。
func (s *Service) recordInvestigationAudit(ctx context.Context, tenantID, userID int, brief *ticketBrief, resp *TicketInvestigationResponse) int64 {
	if s.auditLedger == nil {
		return 0
	}
	metadata := map[string]interface{}{
		"degraded":  resp.Degraded,
		"kbHits":    stepHitCount(resp, "knowledge"),
		"cmdbFound": stepHitCount(resp, "cmdb") > 0,
		"actions":   len(resp.SuggestedActions),
	}
	auditID, err := s.auditLedger.RecordDecisionAudit(ctx, tenantID, userID,
		ledgerRequestID(ctx, "investigation"),
		"investigation", fmt.Sprintf("ticket:%d", brief.ID),
		resp.PromptVersion, resp.Model, resp.Confidence, metadata)
	if err != nil {
		s.logger.Warnw("InvestigateTicket: audit ledger write failed (best-effort)", "error", err)
		return 0
	}
	return auditID
}

// persistInvestigation 把调查结论落 ai_analysis_results（每次全量重跑都落
// 历史，不覆盖——调查轨迹本身就是审计素材）。
func (s *Service) persistInvestigation(ctx context.Context, tenantID, userID int, brief *ticketBrief, resp *TicketInvestigationResponse) {
	if s.repo == nil {
		return
	}
	payload, err := json.Marshal(resp)
	if err != nil {
		s.logger.Warnw("persistInvestigation: marshal failed", "error", err)
		return
	}
	record := &AIAnalysisResult{
		TenantID:        tenantID,
		UserID:          userID,
		AnalysisType:    investigationAnalysisType,
		TicketID:        brief.ID,
		TicketNumber:    brief.TicketNumber,
		TicketTitle:     brief.Title,
		RequestPrompt:   brief.Title,
		ResultJSON:      string(payload),
		Model:           resp.Model,
		LatencyMs:       int(resp.LatencyMs),
		ConfidenceScore: resp.Confidence,
		Degraded:        resp.Degraded,
	}
	if _, err := s.repo.SaveAIAnalysisResult(ctx, record); err != nil {
		s.logger.Warnw("persistInvestigation: save failed", "error", err)
	}
}

// ----------------------------------------------------------------------------
// 小工具：证据压缩与松散字段提取
// ----------------------------------------------------------------------------

const (
	investigationEvidenceChars = 800 // 喂给综合模型的单块证据截断长度
	investigationKBDetailHits  = 3   // 响应 steps[knowledge].detail 保留的命中条数
)

// toLooseList 把工具结果规整成 []map[string]any；非列表结果返回空。
func toLooseList(res interface{}) []map[string]interface{} {
	switch v := res.(type) {
	case []map[string]interface{}:
		return v
	case []interface{}:
		out := make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		// 工具结果可能是强类型切片（如 []dto.CIResponse）——经 JSON 往返
		// 统一成松散 map，调查编排只消费少量已知字段。
		raw, err := json.Marshal(res)
		if err != nil {
			return nil
		}
		var list []map[string]interface{}
		if json.Unmarshal(raw, &list) != nil {
			return nil
		}
		return list
	}
}

// looseInt 从松散 map 取 int 字段（JSON 数字是 float64）。
func looseInt(m map[string]interface{}, key string) int {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		n, _ := strconv.Atoi(v)
		return n
	}
	return 0
}

// compactEvidence 把证据条目的指定字段拼成有界的文本块（喂模型用）。
func compactEvidence(items []map[string]interface{}, fields string, maxChars int) string {
	keys := strings.Split(fields, ",")
	var b strings.Builder
	for i, item := range items {
		if b.Len() >= maxChars {
			break
		}
		b.WriteString(fmt.Sprintf("[%d] ", i+1))
		for j, k := range keys {
			k = strings.TrimSpace(k)
			if v, ok := item[k]; ok && v != nil {
				if j > 0 {
					b.WriteString(" | ")
				}
				b.WriteString(truncateForRef(fmt.Sprint(v), 200))
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

// compactEvidenceList 把证据条目压缩成响应里可读的 detail 列表。
func compactEvidenceList(items []map[string]interface{}, idKey, titleKey string, max int) interface{} {
	if len(items) == 0 {
		return nil
	}
	if len(items) > max {
		items = items[:max]
	}
	out := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		entry := map[string]interface{}{}
		if id := looseInt(item, idKey); id > 0 {
			entry["id"] = id
		}
		if title, ok := item[titleKey].(string); ok && title != "" {
			entry["title"] = title
		}
		out = append(out, entry)
	}
	return out
}

// stepHitCount 取某步骤的命中数（底账 metadata 用）。
func stepHitCount(resp *TicketInvestigationResponse, step string) int {
	for _, st := range resp.Steps {
		if st.Step == step {
			return st.HitCount
		}
	}
	return 0
}

func triageCategory(t *dto.TriageResult) string {
	if t == nil {
		return "unavailable"
	}
	return t.Category
}

func triagePriority(t *dto.TriageResult) string {
	if t == nil {
		return "unavailable"
	}
	return t.Priority
}

func triageConfidence(t *dto.TriageResult) float64 {
	if t == nil {
		return 0
	}
	return t.Confidence
}

func triageExplanation(t *dto.TriageResult) string {
	if t == nil {
		return ""
	}
	return t.Explanation
}
