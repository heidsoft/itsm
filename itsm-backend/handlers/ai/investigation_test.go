package ai_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/aianalysisresult"
	"itsm-backend/ent/enttest"
	"itsm-backend/handlers/ai"
	"itsm-backend/handlers/skill"
	"itsm-backend/middleware"
)

// 工单调查编排回归：钉住「triage → KB → CMDB → LLM 综合」链的契约与降级语义。
//
// 关键不变量：
//   - 每个工具步骤经 ExecuteTool（Gate 2 RBAC + 审计）；权限拒绝只降级本步，
//     不中断整链，且 reason 如实标注（不得伪装成「无命中」）；
//   - LLM 综合失败/输出非法 JSON → 整体 degraded + 确定性拼接，不伪造根因；
//   - suggestedActions 白名单外的动作必须丢弃，且全部 RequiresApproval=true；
//   - 每次全量重跑都落 ai_analysis_results 历史 + investigation 决策底账。

const investigationSuccessJSON = `{
  "findings": ["主库磁盘空间不足", "知识库存在同类处理记录"],
  "suspectedRootCause": "日志写入量增长导致磁盘耗尽",
  "impactedServices": ["订单库"],
  "suggestedActions": [
    {"action": "create_ticket", "description": "为日志清理创建变更工单"},
    {"action": "reboot_server", "description": "不在白名单的动作，必须被丢弃"}
  ],
  "confidence": 0.87
}`

// investigationLLMStub 记录调用次数并返回固定回复/错误。
type investigationLLMStub struct {
	response string
	chatErr  error
	chats    int
}

func (s *investigationLLMStub) Chat(_ context.Context, _ string, _ []dto.LLMMessage) (string, error) {
	s.chats++
	if s.chatErr != nil {
		return "", s.chatErr
	}
	return s.response, nil
}

func (s *investigationLLMStub) ChatStream(_ context.Context, _ string, _ []dto.LLMMessage, _ func(string)) error {
	return errors.New("stub: streaming not supported")
}

func (s *investigationLLMStub) ChatStreamWithTools(_ context.Context, _ string, _ []dto.LLMMessage, _ []dto.LLMTool, _ func(string), _ func([]dto.LLMToolCall)) error {
	return errors.New("stub: streaming not supported")
}

func (s *investigationLLMStub) SupportsToolCalling() bool { return false }

func (s *investigationLLMStub) EffectiveModel(string) string { return "test-model-1" }

func (s *investigationLLMStub) ProviderName() string { return "stub" }

// investigationToolsStub 记录调用顺序与参数，按名分发固定结果。
type investigationToolsStub struct {
	calls     []string
	argsLog   map[string][]map[string]interface{}
	kbHits    []map[string]interface{}
	cis       []map[string]interface{}
	impact    map[string]interface{}
	executeEr error
}

func (s *investigationToolsStub) Execute(_ context.Context, _ int, name string, args map[string]interface{}) (interface{}, error) {
	s.calls = append(s.calls, name)
	if s.argsLog == nil {
		s.argsLog = map[string][]map[string]interface{}{}
	}
	s.argsLog[name] = append(s.argsLog[name], args)
	if s.executeEr != nil {
		return nil, s.executeEr
	}
	switch name {
	case "list_kb":
		return s.kbHits, nil
	case "list_cis":
		return s.cis, nil
	case "get_ci_impact":
		return s.impact, nil
	}
	return nil, fmt.Errorf("stub: unexpected tool %s", name)
}

func (s *investigationToolsStub) GetTool(name string) *dto.ToolDefinition {
	// 资源/动作显式声明：Gate 2 的完整性校验要求非空。
	return &dto.ToolDefinition{Name: name, Description: "stub", ReadOnly: true, Resource: "stub", Action: "read"}
}

func (s *investigationToolsStub) ListTools() []dto.ToolDefinition { return nil }

func (s *investigationToolsStub) ListToolsForTenant(_ context.Context, _ int) []dto.ToolDefinition { return nil }

// investigationTriageStub 固定分诊结论。
type investigationTriageStub struct {
	result dto.TriageResult
	calls  int
}

func (s *investigationTriageStub) SuggestForTenant(_ context.Context, _, _ string, _ int) dto.TriageResult {
	s.calls++
	return s.result
}

// investigationLedgerStub 记录底账 scenario/inputRef 并返回自增 ID。
type investigationLedgerStub struct {
	scenarios []string
	inputRefs []string
	nextID    int64
}

func (l *investigationLedgerStub) RecordDecisionAudit(_ context.Context, _, _ int, _, scenario, inputRef, _, _ string, _ float64, _ map[string]interface{}) (int64, error) {
	l.scenarios = append(l.scenarios, scenario)
	l.inputRefs = append(l.inputRefs, inputRef)
	l.nextID++
	return l.nextID, nil
}

// investigationEnv 一次测试的完整装配（真实 ent 仓储 + stub 依赖）。
type investigationEnv struct {
	client   *ent.Client
	svc      *ai.Service
	tools    *investigationToolsStub
	llm      *investigationLLMStub
	triage   *investigationTriageStub
	ledger   *investigationLedgerStub
	tenantID int
	userID   int
	ticketID int
}

func newInvestigationEnv(t *testing.T) *investigationEnv {
	t.Helper()
	dsn := fmt.Sprintf("file:ai_investigation_%s?mode=memory&cache=shared&_fk=1",
		strings.NewReplacer("/", "_", "#", "_").Replace(t.Name()))
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	suffix := fmt.Sprintf("%d", client.Ticket.Query().CountX(ctx)+client.User.Query().CountX(ctx))
	tenant := client.Tenant.Create().
		SetName("Inv Tenant").
		SetCode("inv-" + suffix).
		SetDomain("inv.example.com").
		SaveX(ctx)
	user := client.User.Create().
		SetUsername("inv-op-" + suffix).
		SetEmail("inv-op-" + suffix + "@example.com").
		SetName("Investigation Operator").
		SetPasswordHash("hash").
		SetTenantID(tenant.ID).
		SaveX(ctx)
	tk := client.Ticket.Create().
		SetTitle("数据库磁盘告警").
		SetDescription("主库磁盘使用率 96%，告警持续 15 分钟").
		SetTicketNumber("TK-INV-" + suffix).
		SetRequesterID(user.ID).
		SetTenantID(tenant.ID).
		SetStatus("open").
		SetPriority("high").
		SaveX(ctx)

	tools := &investigationToolsStub{
		kbHits: []map[string]interface{}{
			{"id": 1, "title": "KB: 磁盘空间清理手册"},
			{"id": 2, "title": "KB: 日志轮转配置指南"},
		},
		cis: []map[string]interface{}{
			{"id": 11, "ciNumber": "CI-001", "name": "db-server-01"},
		},
		impact: map[string]interface{}{
			"impact":     "high",
			"hops":       2,
			"ciId":       11,
			"explain_ai": map[string]interface{}{"summary": "影响下游订单库"},
		},
	}
	llm := &investigationLLMStub{response: investigationSuccessJSON}
	triage := &investigationTriageStub{result: dto.TriageResult{
		Category:      "storage",
		Priority:      "high",
		Confidence:    0.82,
		Explanation:   "关键词命中存储",
		Model:         "triage-model",
		PromptVersion: "v1",
	}}
	ledger := &investigationLedgerStub{}

	svc := ai.NewService(ai.NewEntRepository(client), zap.NewNop().Sugar(), nil, tools, nil, nil, nil, triage, nil, nil)
	svc.SetEntClient(client)
	svc.SetLLMGateway(llm)
	svc.SetAuditLedger(ledger)

	return &investigationEnv{
		client:   client,
		svc:      svc,
		tools:    tools,
		llm:      llm,
		triage:   triage,
		ledger:   ledger,
		tenantID: tenant.ID,
		userID:   user.ID,
		ticketID: tk.ID,
	}
}

// TestInvestigateTicket_LLMSynthesisSuccess 钉住成功路径：工具链按序执行、
// LLM 结论进入契约、白名单过滤生效、底账与历史落库都带 investigation 标识。
func TestInvestigateTicket_LLMSynthesisSuccess(t *testing.T) {
	env := newInvestigationEnv(t)
	ctx := context.Background()

	resp, err := env.svc.InvestigateTicket(ctx, env.ticketID, env.tenantID, env.userID, "super_admin")
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "completed", resp.Status)
	assert.False(t, resp.Degraded)
	assert.Equal(t, "investigation-v1", resp.PromptVersion)
	assert.Equal(t, "test-model-1", resp.Model)
	assert.InDelta(t, 0.87, resp.Confidence, 1e-9)

	// 工具链顺序与参数：KB 用 q、CMDB 先搜后取影响。
	assert.Equal(t, []string{"list_kb", "list_cis", "get_ci_impact"}, env.tools.calls)
	require.NotEmpty(t, env.tools.argsLog["list_kb"])
	assert.Contains(t, env.tools.argsLog["list_kb"][0]["q"], "数据库磁盘告警")
	require.NotEmpty(t, env.tools.argsLog["list_cis"])
	assert.Equal(t, "数据库磁盘告警", env.tools.argsLog["list_cis"][0]["search"])
	require.NotEmpty(t, env.tools.argsLog["get_ci_impact"])
	assert.Equal(t, float64(11), env.tools.argsLog["get_ci_impact"][0]["ciId"])

	// LLM 结论进契约；白名单外的 reboot_server 必须被丢弃。
	assert.Equal(t, []string{"主库磁盘空间不足", "知识库存在同类处理记录"}, resp.Findings)
	assert.Equal(t, "日志写入量增长导致磁盘耗尽", resp.SuspectedRootCause)
	assert.Equal(t, []string{"订单库"}, resp.ImpactedServices)
	require.Len(t, resp.SuggestedActions, 1)
	assert.Equal(t, "create_ticket", resp.SuggestedActions[0].Action)
	assert.True(t, resp.SuggestedActions[0].RequiresApproval,
		"写动作建议必须标注需审批，编排器绝不静默执行")

	// 步骤链完整且无降级。
	require.Len(t, resp.Steps, 4)
	assert.Equal(t, []string{"triage", "knowledge", "cmdb", "synthesis"},
		[]string{resp.Steps[0].Step, resp.Steps[1].Step, resp.Steps[2].Step, resp.Steps[3].Step})
	for _, st := range resp.Steps {
		assert.False(t, st.Degraded, "success path step %s must not degrade", st.Step)
	}

	// 决策底账：scenario=inputigation 固定、inputRef 指向工单。
	require.Len(t, env.ledger.scenarios, 1)
	assert.Equal(t, "investigation", env.ledger.scenarios[0])
	assert.Equal(t, fmt.Sprintf("ticket:%d", env.ticketID), env.ledger.inputRefs[0])
	assert.Equal(t, int64(1), resp.AuditID)

	// 历史落库：analysis_type=investigation，每次重跑新增一条。
	count := env.client.AIAnalysisResult.Query().
		Where(aianalysisresult.TenantIDEQ(env.tenantID), aianalysisresult.AnalysisTypeEQ("investigation")).
		CountX(ctx)
	assert.Equal(t, 1, count)

	// 二次重跑：不幂等复用（CMDB/KB 新鲜度敏感），新增历史记录并再次落底账。
	resp2, err := env.svc.InvestigateTicket(ctx, env.ticketID, env.tenantID, env.userID, "super_admin")
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp2.AuditID)
	count = env.client.AIAnalysisResult.Query().
		Where(aianalysisresult.TenantIDEQ(env.tenantID), aianalysisresult.AnalysisTypeEQ("investigation")).
		CountX(ctx)
	assert.Equal(t, 2, count, "每次调查都必须落一条历史，不做按工单幂等复用")
}

// TestInvestigateTicket_LLMBrokenJSONFallsBack 钉住降级路径：LLM 输出非法
// JSON 时整体 degraded，确定性拼接只如实汇总证据，不编造根因。
func TestInvestigateTicket_LLMBrokenJSONFallsBack(t *testing.T) {
	env := newInvestigationEnv(t)
	env.llm.response = "抱歉，我无法输出 JSON"

	resp, err := env.svc.InvestigateTicket(context.Background(), env.ticketID, env.tenantID, env.userID, "super_admin")
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "degraded", resp.Status)
	assert.True(t, resp.Degraded)
	assert.Empty(t, resp.Model)
	assert.Empty(t, resp.SuspectedRootCause, "降级路径不得编造根因")
	assert.Empty(t, resp.SuggestedActions, "降级路径不得给动作建议")
	assert.InDelta(t, 0.82, resp.Confidence, 1e-9, "降级置信度取分诊置信度")
	assert.Contains(t, resp.Findings, "Triage: 关键词命中存储")

	synthesis := resp.Steps[len(resp.Steps)-1]
	assert.Equal(t, "synthesis", synthesis.Step)
	assert.True(t, synthesis.Degraded)
	assert.NotEmpty(t, synthesis.DegradedReason)
}

// TestInvestigateTicket_LLMErrorFallsBack LLM 网关报错同样降级且不中断。
func TestInvestigateTicket_LLMErrorFallsBack(t *testing.T) {
	env := newInvestigationEnv(t)
	env.llm.chatErr = errors.New("llm gateway down")

	resp, err := env.svc.InvestigateTicket(context.Background(), env.ticketID, env.tenantID, env.userID, "super_admin")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "degraded", resp.Status)
	assert.True(t, resp.Degraded)
	// 底账与历史仍然落（降级也是一次决策事实）。
	assert.Len(t, env.ledger.scenarios, 1)
}

// TestInvestigateTicket_ToolRBACDeniedDegradesSteps 钉住 RBAC 语义：无权限
// 角色下工具步骤降级且 reason 如实标注「权限拒绝」，不是「无命中」。
func TestInvestigateTicket_ToolRBACDeniedDegradesSteps(t *testing.T) {
	env := newInvestigationEnv(t)

	resp, err := env.svc.InvestigateTicket(context.Background(), env.ticketID, env.tenantID, env.userID, "restricted_role")
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Gate 2 拒绝后工具实现不得被触碰。
	assert.Empty(t, env.tools.calls, "RBAC 拒绝后不得执行任何工具")

	var knowledge, cmdb *ai.InvestigationStep
	for i := range resp.Steps {
		switch resp.Steps[i].Step {
		case "knowledge":
			knowledge = &resp.Steps[i]
		case "cmdb":
			cmdb = &resp.Steps[i]
		}
	}
	require.NotNil(t, knowledge)
	assert.True(t, knowledge.Degraded)
	assert.Contains(t, knowledge.DegradedReason, "permission denied")
	require.NotNil(t, cmdb)
	assert.True(t, cmdb.Degraded)
	assert.Contains(t, cmdb.DegradedReason, "permission denied")

	// 分诊不经过工具目录，不受影响。
	assert.False(t, resp.Steps[0].Degraded)
	assert.Equal(t, 1, env.triage.calls)
}

// TestInvestigateTicket_ToolRegistryUnavailableDegradesSteps 工具注册表未装配：
// knowledge/cmdb 降级、triage 与综合不受影响。
func TestInvestigateTicket_ToolRegistryUnavailableDegradesSteps(t *testing.T) {
	env := newInvestigationEnv(t)
	svc := ai.NewService(ai.NewEntRepository(env.client), zap.NewNop().Sugar(), nil, nil, nil, nil, nil, env.triage, nil, nil)
	svc.SetEntClient(env.client)
	svc.SetLLMGateway(env.llm)
	svc.SetAuditLedger(env.ledger)

	resp, err := svc.InvestigateTicket(context.Background(), env.ticketID, env.tenantID, env.userID, "super_admin")
	require.NoError(t, err)
	require.NotNil(t, resp)

	var degradedSteps []string
	for _, st := range resp.Steps {
		if st.Degraded {
			degradedSteps = append(degradedSteps, st.Step)
		}
	}
	assert.ElementsMatch(t, []string{"knowledge", "cmdb"}, degradedSteps)
	assert.Contains(t, resp.Steps[1].DegradedReason, "tool registry unavailable")
}

// TestInvestigateTicket_MissingEntClientUnavailable 装配缺口必须显式暴露，
// 不是静默空结果。
func TestInvestigateTicket_MissingEntClientUnavailable(t *testing.T) {
	env := newInvestigationEnv(t)
	svc := ai.NewService(ai.NewEntRepository(env.client), zap.NewNop().Sugar(), nil, env.tools, nil, nil, nil, env.triage, nil, nil)
	// 刻意不 SetEntClient。

	_, err := svc.InvestigateTicket(context.Background(), env.ticketID, env.tenantID, env.userID, "super_admin")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ai.ErrInvestigationUnavailable))
}

// TestInvestigateTicket_SkillRegistered 钉住 ai.investigate 一等公民注册：
// 权限码、pilot 阶段与输入校验。
func TestInvestigateTicket_SkillRegistered(t *testing.T) {
	env := newInvestigationEnv(t)
	reg := skill.NewSkillRegistry()
	require.NoError(t, ai.RegisterBuiltinSkills(reg, env.svc, zap.NewNop().Sugar()))

	got, err := reg.Get("ai.investigate")
	require.NoError(t, err, "ai.investigate 必须注册为内置技能")
	m := got.Manifest()
	assert.Equal(t, "ai.investigate", m.Name)
	assert.Equal(t, "pilot", m.Category)
	require.NotEmpty(t, m.RequiredPermissions)
	assert.Contains(t, m.RequiredPermissions, "ai:read")
	assert.Contains(t, m.RequiredPermissions, "ticket:read")

	require.NoError(t, got.Validate(map[string]interface{}{"tenantId": 1, "ticketId": 2}))
	assert.Error(t, got.Validate(map[string]interface{}{"tenantId": "not-an-int"}))
	assert.Error(t, got.Validate("not-a-map"))
}

// investigationTestRoute 构造带租户/用户上下文的测试路由。
// ResolveTenantID 走 middleware.TenantContext（生产由 TenantMiddleware 写入），
// 测试里直接注入等价上下文，而不是裸 tenant_id。
func investigationTestRoute(env *investigationEnv, tenantID int) *gin.Engine {
	h := ai.NewHandler(env.svc)
	r := gin.New()
	r.POST("/api/v1/ai/tickets/:id/investigate", func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		c.Set("user_id", env.userID)
		c.Set("role", "super_admin")
		h.InvestigateTicket(c)
	})
	return r
}

// TestInvestigateTicket_Handler_CamelCaseContract HTTP 契约：camelCase 键、
// 无 snake_case 泄漏、成功语义 code=0。
func TestInvestigateTicket_Handler_CamelCaseContract(t *testing.T) {
	env := newInvestigationEnv(t)
	r := investigationTestRoute(env, env.tenantID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/ai/tickets/%d/investigate", env.ticketID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    map[string]any  `json:"data"`
	}
	require.NoError(t, mustUnmarshal(w.Body.Bytes(), &body))
	require.Zero(t, body.Code, body.Message)

	data := body.Data
	for _, key := range []string{"ticketId", "ticketNumber", "status", "degraded", "promptVersion", "confidence", "steps", "suggestedActions"} {
		require.Contains(t, data, key, "contract key %s missing", key)
	}
	assert.Equal(t, "completed", data["status"])
	assert.Equal(t, false, data["degraded"])

	for _, banned := range []string{"ticket_id", "ticket_number", "prompt_version", "suspected_root_cause", "suggested_actions", "latency_ms"} {
		assert.NotContains(t, data, banned, "snake_case contract key %s must not leak", banned)
	}
}

// TestInvestigateTicket_Handler_NotFound 未知工单 → 404 + code 4004。
func TestInvestigateTicket_Handler_NotFound(t *testing.T) {
	env := newInvestigationEnv(t)
	r := investigationTestRoute(env, env.tenantID)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/tickets/999999/investigate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, mustUnmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, common.NotFoundCode, body.Code)
}

// TestInvestigateTicket_Handler_CrossTenantRejected 跨租户访问：404 且响应
// 不泄漏工单存在性，也不产生任何落库副作用。
func TestInvestigateTicket_Handler_CrossTenantRejected(t *testing.T) {
	env := newInvestigationEnv(t)
	// 伪造另一租户上下文：tenant_id 与工单归属不同。
	r := investigationTestRoute(env, env.tenantID+990)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/ai/tickets/%d/investigate", env.ticketID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	assert.NotContains(t, w.Body.String(), "TK-INV", "跨租户拒绝响应不得泄漏工单编号")
	assert.NotContains(t, w.Body.String(), "数据库磁盘告警", "跨租户拒绝响应不得泄漏工单标题")

	// 失败调用不得留底账或历史。
	assert.Empty(t, env.ledger.scenarios)
	assert.Zero(t, env.client.AIAnalysisResult.Query().CountX(context.Background()))
}

// TestInvestigateTicket_Handler_Unavailable ent client 未装配 → 503。
func TestInvestigateTicket_Handler_Unavailable(t *testing.T) {
	env := newInvestigationEnv(t)
	svc := ai.NewService(ai.NewEntRepository(env.client), zap.NewNop().Sugar(), nil, env.tools, nil, nil, nil, env.triage, nil, nil)
	// 刻意不 SetEntClient：handler 必须映射 503 而不是 500。
	env.svc = svc
	r := investigationTestRoute(env, env.tenantID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/ai/tickets/%d/investigate", env.ticketID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
}

// mustUnmarshal 测试 JSON 解析（错误处理收敛在这里，让断言行干净）。
func mustUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
