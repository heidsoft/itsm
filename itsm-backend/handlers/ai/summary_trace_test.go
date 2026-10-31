package ai_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/ent/enttest"
	_ "itsm-backend/ent/runtime"
	"itsm-backend/handlers/ai"
	"itsm-backend/service"

	_ "github.com/mattn/go-sqlite3"
)

// 本文件守卫「AI 摘要的产出方式必须可溯源」这条性质。
//
// 背景：SummarizeTicket 此前只委托 rca，而 rca 的默认产物是**字段拼接**
// （method=fallback）；同时 SetSummarizeService 的注释声称「优先走
// SummarizeService」——装配与实现不一致，summarizeService 被注入后从未被调用。
// 后果是界面上标着「AI 摘要」的东西，很可能是原文直出或字段拼接，而响应里
// 没有任何字段能区分。采纳率、有用率这些指标因此从根上就是脏的。
//
// 三态必须如实区分，且每一态都要落底账：
//
//	llm      —— 模型真的生成了（model 为真实模型名）
//	direct   —— 原文够短直接返回（不是摘要；model 留空是实情）
//	fallback —— 模型不可用/失败后截断（不是摘要；model 留空是实情）
//
// 「不是模型产出」的摘要恰恰最需要被记录：否则它会被当成 AI 表现统计进去。

// stubModelProvider 能自报模型的 LLM provider（用于 llm 路径）。
type stubModelProvider struct {
	model    string
	response string
}

func (p *stubModelProvider) Chat(_ context.Context, _ string, _ []service.LLMMessage) (string, error) {
	return p.response, nil
}

func (p *stubModelProvider) Model() string { return p.model }

// newSummaryServiceWithClient 构造带内存库与摘要服务的 AI Service。
//
// 返回 (svc, client, ticketID)：ticketID 是预置工单的 ID（由 ent 自增分配，
// 不同用例各自建库因此不会撞号）。
func newSummaryServiceWithClient(t *testing.T, dsn string, gateway *service.LLMGateway, ledger ai.AuditLedgerWriter, longText bool) (*ai.Service, int) {
	t.Helper()
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { _ = client.Close() })

	// 工单所属租户必须先存在（Ticket.tenant_id 有外键）。
	tenant := client.Tenant.Create().SetName("t").SetCode("t").SaveX(ctx)
	// requester_id 同样有外键：先建报单人。
	requester := client.User.Create().
		SetUsername("requester").
		SetName("requester").
		SetEmail("requester@example.com").
		SetPasswordHash("x").
		SetTenantID(tenant.ID).
		SaveX(ctx)

	b := client.Ticket.Create().
		SetTicketNumber("TK-9001").
		SetTitle("数据库连接超时").
		SetStatus("open").
		SetPriority("high").
		SetTenantID(tenant.ID).
		SetRequesterID(requester.ID)
	if longText {
		// 超过摘要上限（200）→ 触发模型生成路径。
		b = b.SetDescription("应用无法连接 PostgreSQL 实例，报 connection refused，影响订单服务下单与查询，" +
			"已持续二十分钟，期间订单成功率下降到六成，客服收到多起用户反馈，需要尽快定位是实例网络策略、" +
			"连接数打满还是凭据轮换导致的鉴权失败，并同步给出临时规避方案。")
	}
	ticket := b.SaveX(ctx)

	svc := ai.NewService(nil, zap.NewNop().Sugar(), nil, nil, nil, nil, nil, nil, nil, nil)
	svc.SetEntClient(client)
	if gateway != nil {
		svc.SetSummarizeService(service.NewSummarizeService(gateway, zap.NewNop()))
	}
	if ledger != nil {
		svc.SetAuditLedger(ledger)
	}
	return svc, ticket.ID
}

// TestSummarize_LLMPathCarriesRealModelAndAudit 锁住 llm 路径：
// 摘要确实是模型生成的，model 必须是真实模型名，且落了底账。
func TestSummarize_LLMPathCarriesRealModelAndAudit(t *testing.T) {
	const configured = "deepseek-chat"
	gw := service.NewLLMGateway(&stubModelProvider{model: configured, response: "订单服务因 PostgreSQL 连接被拒而不可用，需检查实例网络与凭据。"},
		nil, nil, "deepseek")
	ledger := &mockAuditLedger{id: 77}
	svc, ticketID := newSummaryServiceWithClient(t, "file:sum_llm?mode=memory&cache=shared&_fk=1", gw, ledger, true)

	out, err := svc.SummarizeTicket(context.Background(), ticketID, 1, 7)
	require.NoError(t, err)

	resp, ok := out.(map[string]interface{})
	require.True(t, ok, "摘要应返回带溯源字段的 map")
	assert.Equal(t, "llm", resp["method"])
	assert.Equal(t, service.SummarizePromptVersion, resp["promptVersion"])
	assert.Equal(t, configured, resp["model"], "model 必须是 provider 实际使用的模型，不是硬编码值")
	assert.Equal(t, int64(77), resp["auditId"])

	require.Len(t, ledger.calls, 1)
	call := ledger.calls[0]
	assert.Equal(t, "summary", call.Scenario)
	assert.Equal(t, 7, call.UserID)
	assert.Equal(t, service.SummarizePromptVersion, call.PromptVersion)
	assert.Equal(t, configured, call.Model)
	// 摘要是生成类产出，没有自评置信度：底账如实记 0 并标注不适用。
	assert.Equal(t, 0.0, call.Confidence)
	assert.Equal(t, true, call.Suggestion["confidenceNotApplicable"])
	assert.Equal(t, "llm", call.Suggestion["method"])
}

// TestSummarize_DirectPathStillAudited 锁住最容易漏的一态：原文够短时
// method=direct（不是模型产出的摘要），此时 model 留空是**实情**，但底账照落——
// 恰恰是这种「不是 AI 产出」的摘要最需要被记录，否则会被当成 AI 表现统计进去。
func TestSummarize_DirectPathStillAudited(t *testing.T) {
	gw := service.NewLLMGateway(&stubModelProvider{model: "deepseek-chat", response: "ignored"}, nil, nil, "deepseek")
	ledger := &mockAuditLedger{id: 78}
	// longText=false：工单文本短于摘要上限 → SummarizeService 走 direct。
	svc, ticketID := newSummaryServiceWithClient(t, "file:sum_direct?mode=memory&cache=shared&_fk=1", gw, ledger, false)

	out, err := svc.SummarizeTicket(context.Background(), ticketID, 1, 7)
	require.NoError(t, err)

	resp, ok := out.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "direct", resp["method"], "原文够短时应如实标注 direct，而不是假装是 llm 产出")
	assert.Equal(t, "", resp["model"], "无模型参与时 model 留空是实情——填任何值都是伪造溯源")
	assert.Equal(t, service.SummarizePromptVersion, resp["promptVersion"])

	// 关键：direct 也必须落底账，否则这条「非 AI 产出」不会被计入采纳率的分母。
	require.Len(t, ledger.calls, 1, "非模型产出的摘要同样要落底账")
	assert.Equal(t, "direct", ledger.calls[0].Suggestion["method"])
}

// TestSummarize_NoLedgerOmitsAuditId：未装配底账时摘要照常返回，只是没有 auditId。
func TestSummarize_NoLedgerOmitsAuditId(t *testing.T) {
	gw := service.NewLLMGateway(&stubModelProvider{model: "deepseek-chat", response: "摘要内容"}, nil, nil, "deepseek")
	svc, ticketID := newSummaryServiceWithClient(t, "file:sum_noledger?mode=memory&cache=shared&_fk=1", gw, nil, true)

	out, err := svc.SummarizeTicket(context.Background(), ticketID, 1, 7)
	require.NoError(t, err)
	resp, ok := out.(map[string]interface{})
	require.True(t, ok)
	assert.NotContains(t, resp, "auditId")
	assert.Contains(t, resp, "summary")
	assert.Contains(t, resp, "method")
}
