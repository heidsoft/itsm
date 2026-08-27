package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// mockRCALLMProvider implements LLMProvider with a canned JSON response,
// mirroring the DeepSeek/OpenAI provider contract used by RootCauseService.
type mockRCALLMProvider struct {
	response string
	err      error
}

func (m mockRCALLMProvider) Chat(_ context.Context, _ string, _ []LLMMessage) (string, error) {
	return m.response, m.err
}

// helper: create tenants 1&2 (auto IDs), then a user + ticket in the requested
// tenant; return the ticket ID. Ticket.requester is a required FK edge, so the
// user must exist first. Two tenants are always created so that cross-tenant
// isolation tests can reference tenantID=2.
func createRCATicket(t *testing.T, client *ent.Client, tenantID int, priority string) int {
	t.Helper()
	ctx := context.Background()
	for i := 1; i <= 2; i++ {
		if _, err := client.Tenant.Create().
			SetName("RCA Tenant").
			SetCode(fmt.Sprintf("rca-%d", i)).
			SetDomain("rca.test").
			SetStatus("active").
			Save(ctx); err != nil {
			t.Fatalf("create tenant %d: %v", i, err)
		}
	}
	user, err := client.User.Create().
		SetUsername("rca-user").
		SetEmail("rca@test.com").
		SetName("RCA User").
		SetPasswordHash("hashed").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	ticketEntity, err := client.Ticket.Create().
		SetTitle("数据库连接超时").
		SetDescription("应用在高峰期频繁报数据库连接池耗尽").
		SetPriority(priority).
		SetStatus("open").
		SetTicketNumber("TCK-RCA-TEST").
		SetRequesterID(user.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return ticketEntity.ID
}

// 回归：LLM 网关注入后，AnalyzeTicket 必须消费 LLM 返回的 3 个根因，
// 绝不回退到启发式的“系统资源不足”模板。
func TestAnalyzeTicket_UsesLLM_ThreeRootCausesNoFallback(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	ticketID := createRCATicket(t, client, 1, "high")

	mock := mockRCALLMProvider{
		response: `{"root_causes": [
			{"title": "连接池配置过小", "description": "maxActive=10 不足以支撑高峰期并发", "confidence": 0.92, "category": "software"},
			{"title": "慢SQL拖长事务占用连接", "description": "部分查询未命中索引", "confidence": 0.78, "category": "software"},
			{"title": "数据库连接未及时释放", "description": "异常路径未归还连接", "confidence": 0.65, "category": "other"}
		]}`,
	}
	gateway := NewLLMGateway(mock, nil, nil, "test")

	svc := NewRootCauseService(client, zaptest.NewLogger(t).Sugar())
	svc.SetGateway(gateway)

	resp, err := svc.AnalyzeTicket(context.Background(), ticketID, 1)
	require.NoError(t, err)
	require.Len(t, resp.RootCauses, 3, "应返回 LLM 给出的 3 个根因")

	assert.Equal(t, "连接池配置过小", resp.RootCauses[0].Title)
	assert.Equal(t, "software", resp.RootCauses[0].Category, "category 应来自 LLM 响应")
	assert.Equal(t, 0.92, resp.RootCauses[0].Confidence)
	assert.Equal(t, "慢SQL拖长事务占用连接", resp.RootCauses[1].Title)
	assert.Equal(t, "数据库连接未及时释放", resp.RootCauses[2].Title)
	for _, rc := range resp.RootCauses {
		assert.NotEqual(t, "系统资源不足", rc.Title, "LLM 可用时不得回退到启发式模板")
		assert.Equal(t, "identified", rc.Status)
	}
}

// 回归：无 LLM 网关时按原启发式逻辑降级（高优先级 → 系统资源不足）。
func TestAnalyzeTicket_NoGateway_FallsBackToHeuristic(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	ticketID := createRCATicket(t, client, 1, "critical")

	svc := NewRootCauseService(client, zaptest.NewLogger(t).Sugar())
	// 不注入 gateway，走启发式分支

	resp, err := svc.AnalyzeTicket(context.Background(), ticketID, 1)
	require.NoError(t, err)
	require.Len(t, resp.RootCauses, 1)
	assert.Equal(t, "系统资源不足", resp.RootCauses[0].Title)
}

// 回归：租户隔离——跨租户查询必须失败，不允许返回其他租户的工单分析。
func TestAnalyzeTicket_TenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	// 工单属于租户 2
	ticketID := createRCATicket(t, client, 2, "medium")

	svc := NewRootCauseService(client, zaptest.NewLogger(t).Sugar())

	_, err := svc.AnalyzeTicket(context.Background(), ticketID, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ticket not found")
}

func TestAnalyzeIncident_UsesIncidentAggregateAndTenantScope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	for i := 1; i <= 2; i++ {
		client.Tenant.Create().SetName(fmt.Sprintf("Incident Tenant %d", i)).SetCode(fmt.Sprintf("incident-rca-%d", i)).SetStatus("active").SaveX(ctx)
	}
	user := client.User.Create().SetUsername("incident-rca-user").SetEmail("incident-rca@test.com").SetName("NOC").SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(2).SaveX(ctx)
	entity := client.Incident.Create().SetTitle("核心网络中断").SetDescription("监控显示核心交换机不可达").SetStatus("new").SetPriority("critical").SetSeverity("critical").SetImpact("high").SetUrgency("high").SetCategory("network").SetIncidentNumber("INC-AI-001").SetReporterID(user.ID).SetTenantID(2).SetDetectedAt(time.Now()).SaveX(ctx)

	gateway := NewLLMGateway(mockRCALLMProvider{response: `{"root_causes":[{"title":"核心交换机故障","description":"监控与故障现象一致","confidence":0.91,"category":"network"}]}`}, nil, nil, "test")
	svc := NewRootCauseService(client, zaptest.NewLogger(t).Sugar())
	svc.SetGateway(gateway)

	resp, err := svc.AnalyzeIncident(ctx, entity.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, entity.ID, resp.IncidentID)
	assert.Equal(t, "INC-AI-001", resp.IncidentNumber)
	assert.Equal(t, "llm", resp.AnalysisMethod)
	assert.False(t, resp.Degraded)
	require.Len(t, resp.RootCauses, 1)
	assert.Equal(t, "核心交换机故障", resp.RootCauses[0].Title)

	_, err = svc.AnalyzeIncident(ctx, entity.ID, 1)
	require.ErrorIs(t, err, ErrIncidentNotFound)
}

func TestAnalyzeIncident_InvalidLLMOutputIsExplicitlyDegraded(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	tenant := client.Tenant.Create().SetName("Incident Tenant").SetCode("incident-degraded").SetStatus("active").SaveX(ctx)
	user := client.User.Create().SetUsername("incident-degraded-user").SetEmail("incident-degraded@test.com").SetName("NOC").SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).SaveX(ctx)
	entity := client.Incident.Create().SetTitle("应用故障").SetStatus("new").SetPriority("high").SetSeverity("high").SetCategory("software").SetIncidentNumber("INC-AI-002").SetReporterID(user.ID).SetTenantID(tenant.ID).SetDetectedAt(time.Now()).SaveX(ctx)

	gateway := NewLLMGateway(mockRCALLMProvider{response: `{"root_causes":[{"title":"越界结果","description":"bad","confidence":1.5,"category":"unknown"}]}`}, nil, nil, "test")
	svc := NewRootCauseService(client, zaptest.NewLogger(t).Sugar())
	svc.SetGateway(gateway)
	resp, err := svc.AnalyzeIncident(ctx, entity.ID, tenant.ID)
	require.NoError(t, err)
	assert.True(t, resp.Degraded)
	assert.Equal(t, "heuristic", resp.AnalysisMethod)
	assert.Equal(t, "llm_unavailable_or_invalid", resp.DegradedReason)
}
