package sla

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
)

// 监控大屏回归（用户报告：合规率 96.5% 其实是工单解决率、活跃告警数量不对、
// 响应/解决达成率是伪造值、服务类型绩效缺少过滤条件）。
//
// 修复前 GetSLAMonitoring 忽略传入窗口且零样本时返回 complianceRate=1.0，
// 前端再自行乘以 100 并伪造 totalTickets/alerts。下面的数据集是确定性的，
// 每个比率都能在断言里手工核对样本数，因此任何“再伪造一次”的改动都会失败。

type slaMonitorFixture struct {
	tenantID      int
	start         time.Time
	end           time.Time
	incidentDefID int
	changeDefID   int
	overdueTicket *ent.Ticket
	littleTicket  *ent.Ticket
	unboundTicket *ent.Ticket
	outsideTicket *ent.Ticket
	requesterID   int
}

// seedSLAMonitoringFixture 创建窗口内 4 张工单 + 窗口外 1 张工单，以及
// 2 条未解决违约记录、1 条已解决违约记录、2 条未解决告警和 1 条已解决告警。
func seedSLAMonitoringFixture(t *testing.T, client *ent.Client, tenantID int) slaMonitorFixture {
	t.Helper()
	ctx := context.Background()
	uid := slaUniqueID()
	now := time.Now().UTC().Truncate(time.Second)

	f := slaMonitorFixture{
		tenantID: tenantID,
		start:    now.Add(-24 * time.Hour),
		end:      now.Add(24 * time.Hour),
	}

	user, err := client.User.Create().
		SetUsername("sla-mon-user-" + uid).
		SetEmail("sla-mon-" + uid + "@example.com").
		SetName("SLA Monitor User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	f.requesterID = user.ID

	incidentDef, err := client.SLADefinition.Create().
		SetName("监控-事件SLA-" + uid).
		SetServiceType("incident").
		SetPriority(common.PriorityHigh).
		SetResponseTime(30).
		SetResolutionTime(240).
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	f.incidentDefID = incidentDef.ID

	changeDef, err := client.SLADefinition.Create().
		SetName("监控-变更SLA-" + uid).
		SetServiceType("change").
		SetPriority(common.PriorityMedium).
		SetResponseTime(60).
		SetResolutionTime(480).
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	f.changeDefID = changeDef.ID

	// 1) 响应达标 + 解决达标 + 已解决
	_, err = client.Ticket.Create().
		SetTitle("监控-按时解决-" + uid).
		SetTicketNumber("TKT-MON-OK-" + uid).
		SetStatus(common.TicketStatusResolved).
		SetPriority(common.PriorityHigh).
		SetRequesterID(user.ID).
		SetTenantID(tenantID).
		SetSLADefinitionID(incidentDef.ID).
		SetCreatedAt(now.Add(-10 * time.Hour)).
		SetSLAResponseDeadline(now.Add(-9 * time.Hour)).
		SetFirstResponseAt(now.Add(-9*time.Hour - 30*time.Minute)).
		SetSLAResolutionDeadline(now.Add(-2 * time.Hour)).
		SetResolvedAt(now.Add(-3 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	// 2) 响应超时 + 未解决 + 有未解决违约 + 有活跃告警
	f.overdueTicket, err = client.Ticket.Create().
		SetTitle("监控-响应超时-" + uid).
		SetTicketNumber("TKT-MON-LATE-" + uid).
		SetStatus(common.TicketStatusOpen).
		SetPriority(common.PriorityHigh).
		SetRequesterID(user.ID).
		SetTenantID(tenantID).
		SetSLADefinitionID(incidentDef.ID).
		SetCreatedAt(now.Add(-5 * time.Hour)).
		SetSLAResponseDeadline(now.Add(-4 * time.Hour)).
		SetFirstResponseAt(now.Add(-3 * time.Hour)).
		SetSLAResolutionDeadline(now.Add(19 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	// 3) 无截止时间：不得计入任何达成率样本
	f.littleTicket, err = client.Ticket.Create().
		SetTitle("监控-未配置截止-" + uid).
		SetTicketNumber("TKT-MON-NODEADLINE-" + uid).
		SetStatus(common.TicketStatusNew).
		SetPriority(common.PriorityLow).
		SetRequesterID(user.ID).
		SetTenantID(tenantID).
		SetSLADefinitionID(changeDef.ID).
		SetCreatedAt(now.Add(-1 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	// 4) 未绑定 SLA 定义：serviceType 维度必须落到 unassigned 而不是丢弃
	f.unboundTicket, err = client.Ticket.Create().
		SetTitle("监控-未绑定SLA-" + uid).
		SetTicketNumber("TKT-MON-UNBOUND-" + uid).
		SetStatus(common.TicketStatusResolved).
		SetPriority(common.PriorityLow).
		SetRequesterID(user.ID).
		SetTenantID(tenantID).
		SetCreatedAt(now.Add(-20 * time.Hour)).
		SetSLAResolutionDeadline(now.Add(-5 * time.Hour)).
		SetResolvedAt(now.Add(-6 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	// 5) 窗口外工单：任何指标都不得统计它
	f.outsideTicket, err = client.Ticket.Create().
		SetTitle("监控-窗口外-" + uid).
		SetTicketNumber("TKT-MON-OUTSIDE-" + uid).
		SetStatus(common.TicketStatusResolved).
		SetPriority(common.PriorityHigh).
		SetRequesterID(user.ID).
		SetTenantID(tenantID).
		SetCreatedAt(now.Add(-40 * 24 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	repo := NewEntRepository(client)
	newViolation := func(ticketID, defID int, vType string, resolved bool) {
		_, err := repo.CreateViolation(ctx, &SLAViolation{
			TicketID:        ticketID,
			SLADefinitionID: defID,
			ViolationType:   vType,
			ViolationTime:   now.Add(-1 * time.Hour),
			Severity:        "high",
			IsResolved:      resolved,
			TenantID:        tenantID,
		})
		require.NoError(t, err)
	}
	newViolation(f.overdueTicket.ID, incidentDef.ID, "response", false)
	newViolation(f.overdueTicket.ID, incidentDef.ID, "resolution", false)
	newViolation(f.overdueTicket.ID, incidentDef.ID, "response", true)

	rule, err := client.SLAAlertRule.Create().
		SetSLADefinitionID(incidentDef.ID).
		SetName("监控-预警规则-" + uid).
		SetThresholdPercentage(80).
		SetAlertLevel("warning").
		SetNotificationChannels([]string{"email"}).
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	newAlert := func(created time.Time, resolved bool) {
		create := client.SLAAlertHistory.Create().
			SetTicketID(f.overdueTicket.ID).
			SetTicketNumber(f.overdueTicket.TicketNumber).
			SetTicketTitle(f.overdueTicket.Title).
			SetAlertRuleID(rule.ID).
			SetAlertRuleName(rule.Name).
			SetAlertLevel("warning").
			SetThresholdPercentage(80).
			SetActualPercentage(92.5).
			SetCreatedAt(created).
			SetTenantID(tenantID)
		if resolved {
			create = create.SetResolvedAt(now)
		}
		_, err := create.Save(ctx)
		require.NoError(t, err)
	}
	newAlert(now.Add(-30*time.Minute), false)
	newAlert(now.Add(-90*time.Minute), false)
	newAlert(now.Add(-3*time.Hour), true)

	return f
}

// slaEngineForTenant 复用真实 handler 构造指定租户上下文的路由，
// 路径与 router.go 中注册的 canonical 路径保持一致。
func slaEngineForTenant(t *testing.T, client *ent.Client, tenantID int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(NewEntRepository(client), zaptest.NewLogger(t).Sugar()))
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(slaTestAuth(tenantID, 1))
	r.POST("/api/v1/sla/monitoring", h.GetSLAMonitoring)
	r.GET("/api/v1/sla/performance", h.GetSLAPerformance)
	return r
}

func slaWindowRequest(f slaMonitorFixture) dto.SLAMonitoringRequest {
	return dto.SLAMonitoringRequest{
		StartTime: f.start.Format(time.RFC3339),
		EndTime:   f.end.Format(time.RFC3339),
	}
}

func TestSLAHandler_MonitoringMetricsAreHonestRegression(t *testing.T) {
	r, client, tenantID := setupSLAHandler(t)
	f := seedSLAMonitoringFixture(t, client, tenantID)

	resp := doSLAReq(t, r, http.MethodPost, "/api/v1/sla/monitor", slaWindowRequest(f), false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	data := resp.Data.(map[string]interface{})

	t.Run("窗口与样本口径", func(t *testing.T) {
		assert.Equal(t, f.start.Format(time.RFC3339), data["startTime"])
		assert.Equal(t, f.end.Format(time.RFC3339), data["endTime"])
		assert.Equal(t, false, data["truncated"])
		// 创建时间在 40 天前的工单必须被窗口排除：总数只含窗口内 4 张
		assert.Equal(t, float64(4), slaNum(t, data, "totalTickets"))
		assert.Equal(t, float64(2), slaNum(t, data, "resolvedTickets"))
		// 工单解决率 = 2/4，不再是历史里被当成合规率的数字
		assert.Equal(t, 50.0, slaNum(t, data, "resolutionRate"))
	})

	t.Run("合规率来自违约工单数而不是恒等于1", func(t *testing.T) {
		assert.Equal(t, float64(1), slaNum(t, data, "violatedTickets"))
		assert.Equal(t, float64(3), slaNum(t, data, "metSlaTickets"))
		assert.Equal(t, 75.0, slaNum(t, data, "complianceRate"))
		assert.Equal(t, 25.0, slaNum(t, data, "violationRate"))
	})

	t.Run("达成率携带样本数", func(t *testing.T) {
		assert.Equal(t, float64(2), slaNum(t, data, "responseTimeSamples"))
		assert.Equal(t, float64(1), slaNum(t, data, "responseTimeMet"))
		assert.Equal(t, 50.0, slaNum(t, data, "responseTimeCompliance"))
		assert.Equal(t, float64(2), slaNum(t, data, "resolutionTimeSamples"))
		assert.Equal(t, float64(2), slaNum(t, data, "resolutionTimeMet"))
		assert.Equal(t, 100.0, slaNum(t, data, "resolutionTimeCompliance"))
	})

	t.Run("平均时长以分钟返回", func(t *testing.T) {
		// (30 + 120) / 2 = 75，(420 + 840) / 2 = 630
		assert.Equal(t, 75.0, slaNum(t, data, "averageResponseMinutes"))
		assert.Equal(t, 630.0, slaNum(t, data, "averageResolutionMinutes"))
	})

	t.Run("风险工单只统计未完成且已超时的", func(t *testing.T) {
		assert.Equal(t, float64(1), slaNum(t, data, "atRiskTickets"))
	})

	t.Run("违约记录数与工单数口径分开", func(t *testing.T) {
		assert.Equal(t, float64(3), slaNum(t, data, "totalViolations"))
		assert.Equal(t, float64(1), slaNum(t, data, "resolvedViolations"))
		assert.Equal(t, float64(2), slaNum(t, data, "activeViolations"))
	})

	t.Run("活跃告警来自告警历史并补齐工单上下文", func(t *testing.T) {
		assert.Equal(t, float64(2), slaNum(t, data, "activeAlerts"))
		assert.Equal(t, float64(2), slaNum(t, data, "activeSlas"))
		assert.Equal(t, float64(1), slaNum(t, data, "activeAlertRules"))
		alerts, ok := data["alerts"].([]interface{})
		require.True(t, ok, "alerts 必须是数组: %v", data["alerts"])
		require.Len(t, alerts, 2)
		first := alerts[0].(map[string]interface{})
		assert.Equal(t, float64(f.overdueTicket.ID), slaNum(t, first, "ticketId"))
		assert.Equal(t, f.overdueTicket.TicketNumber, first["ticketNumber"])
		assert.Equal(t, f.overdueTicket.Title, first["ticketTitle"])
		assert.Equal(t, common.PriorityHigh, first["priority"])
		assert.NotEmpty(t, first["alertRuleName"])
		assert.NotNil(t, first["timeRemaining"], "有解决截止时间时必须返回剩余时间")
	})

	t.Run("契约只有一套 camelCase", func(t *testing.T) {
		assert.NotContains(t, data, "total_violations")
		assert.NotContains(t, data, "compliance_rate")
		raw, _ := json.Marshal(resp)
		assert.NotContains(t, string(raw), "sla_definition_id")
	})
}

// 零样本必须诚实为 0（前端按样本数渲染“暂无样本”），不得回退成 100%。
func TestSLAHandler_MonitoringWithoutSamplesIsNotFakeSuccess(t *testing.T) {
	r, _, _ := setupSLAHandler(t)
	resp := doSLAReq(t, r, http.MethodPost, "/api/v1/sla/monitoring", dto.SLAMonitoringRequest{}, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	data := resp.Data.(map[string]interface{})

	assert.Equal(t, float64(0), slaNum(t, data, "totalTickets"))
	assert.Equal(t, 0.0, slaNum(t, data, "complianceRate"))
	assert.Equal(t, 0.0, slaNum(t, data, "resolutionRate"))
	assert.Equal(t, 0.0, slaNum(t, data, "responseTimeCompliance"))
	assert.Equal(t, float64(0), slaNum(t, data, "responseTimeSamples"))
	assert.Equal(t, 0.0, slaNum(t, data, "resolutionTimeCompliance"))
	assert.Equal(t, float64(0), slaNum(t, data, "averageResponseMinutes"))
	assert.Equal(t, float64(0), slaNum(t, data, "activeAlerts"))
	assert.Empty(t, data["alerts"])
}

// 缺省窗口：不传时间体时套用最近 30 天，因此 40 天前的工单仍然被排除。
func TestSLAHandler_MonitoringDefaultWindow(t *testing.T) {
	r, client, tenantID := setupSLAHandler(t)
	seedSLAMonitoringFixture(t, client, tenantID)

	resp := doSLAReq(t, r, http.MethodPost, "/api/v1/sla/monitor", nil, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, float64(4), slaNum(t, data, "totalTickets"))
}

func TestSLAHandler_MonitoringValidatesInput(t *testing.T) {
	r, client, tenantID := setupSLAHandler(t)
	f := seedSLAMonitoringFixture(t, client, tenantID)

	cases := []struct {
		name string
		body dto.SLAMonitoringRequest
		code int
	}{
		{
			name: "伪时间值不再被静默接受",
			body: dto.SLAMonitoringRequest{StartTime: "30d", EndTime: "now"},
			code: common.ParamErrorCode,
		},
		{
			name: "结束时间必须晚于开始时间",
			body: dto.SLAMonitoringRequest{
				StartTime: f.end.Format(time.RFC3339),
				EndTime:   f.start.Format(time.RFC3339),
			},
			code: common.ParamErrorCode,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, status := doSLAReqRaw(t, r, http.MethodPost, "/api/v1/sla/monitor", tc.body)
			assert.Equal(t, tc.code, resp.Code, "body=%s", slaStr(resp))
			assert.Equal(t, http.StatusBadRequest, status)
		})
	}
}

func TestSLAHandler_MonitoringFailsClosedWithoutTenant(t *testing.T) {
	r, client, tenantID := setupSLAHandler(t)
	f := seedSLAMonitoringFixture(t, client, tenantID)

	resp, status := doSLAReqRaw(t, r, http.MethodPost, "/api/v1/sla/monitor", slaWindowRequest(f), true)
	assert.Equal(t, common.AuthFailedCode, resp.Code, "body=%s", slaStr(resp))
	assert.Equal(t, http.StatusUnauthorized, status)
}

// 跨租户读取必须 fail closed：租户 A 的指标不受租户 B 数据影响，反之亦然。
func TestSLAHandler_MonitoringTenantIsolation(t *testing.T) {
	_, client, tenantA := setupSLAHandler(t)
	f := seedSLAMonitoringFixture(t, client, tenantA)
	ctx := context.Background()
	uid := slaUniqueID()

	tenantB, err := client.Tenant.Create().
		SetName("监控租户B").
		SetCode("MONB" + uid).
		SetDomain("mon-b.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	userB, err := client.User.Create().
		SetUsername("sla-mon-b-" + uid).
		SetEmail("sla-mon-b-" + uid + "@example.com").
		SetName("Tenant B User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)
	defB, err := client.SLADefinition.Create().
		SetName("监控-B事件SLA-" + uid).
		SetServiceType("incident").
		SetResponseTime(30).
		SetResolutionTime(240).
		SetIsActive(true).
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)
	ticketB, err := client.Ticket.Create().
		SetTitle("监控-B工单-" + uid).
		SetTicketNumber("TKT-MON-B-" + uid).
		SetStatus(common.TicketStatusOpen).
		SetPriority(common.PriorityCritical).
		SetRequesterID(userB.ID).
		SetTenantID(tenantB.ID).
		SetSLADefinitionID(defB.ID).
		SetCreatedAt(time.Now().UTC().Add(-2 * time.Hour)).
		SetSLAResolutionDeadline(time.Now().UTC().Add(-time.Hour)).
		Save(ctx)
	require.NoError(t, err)
	ruleB, err := client.SLAAlertRule.Create().
		SetSLADefinitionID(defB.ID).
		SetName("监控-B规则-" + uid).
		SetThresholdPercentage(90).
		SetAlertLevel("critical").
		SetNotificationChannels([]string{"email"}).
		SetIsActive(true).
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.SLAAlertHistory.Create().
		SetTicketID(ticketB.ID).
		SetTicketNumber(ticketB.TicketNumber).
		SetTicketTitle(ticketB.Title).
		SetAlertRuleID(ruleB.ID).
		SetAlertRuleName(ruleB.Name).
		SetAlertLevel("critical").
		SetCreatedAt(time.Now().UTC().Add(-time.Hour)).
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)

	engineA := slaEngineForTenant(t, client, tenantA)
	respA := doSLAReq(t, engineA, http.MethodPost, "/api/v1/sla/monitoring", slaWindowRequest(f), false)
	require.Equal(t, common.SuccessCode, respA.Code, "body=%s", slaStr(respA))
	dataA := respA.Data.(map[string]interface{})
	assert.Equal(t, float64(4), slaNum(t, dataA, "totalTickets"))
	assert.Equal(t, float64(2), slaNum(t, dataA, "activeAlerts"))
	assert.Equal(t, 75.0, slaNum(t, dataA, "complianceRate"))
	for _, alert := range dataA["alerts"].([]interface{}) {
		assert.NotEqual(t, ticketB.ID, int(slaNum(t, alert.(map[string]interface{}), "ticketId")))
	}

	engineB := slaEngineForTenant(t, client, tenantB.ID)
	respB := doSLAReq(t, engineB, http.MethodPost, "/api/v1/sla/monitoring", slaWindowRequest(f), false)
	require.Equal(t, common.SuccessCode, respB.Code, "body=%s", slaStr(respB))
	dataB := respB.Data.(map[string]interface{})
	assert.Equal(t, float64(1), slaNum(t, dataB, "totalTickets"))
	assert.Equal(t, float64(1), slaNum(t, dataB, "activeAlerts"))
	assert.Equal(t, float64(0), slaNum(t, dataB, "totalViolations"))
	assert.Equal(t, 0.0, slaNum(t, dataB, "responseTimeCompliance"))
}

func TestSLAHandler_PerformanceByServiceType(t *testing.T) {
	_, client, tenantID := setupSLAHandler(t)
	f := seedSLAMonitoringFixture(t, client, tenantID)
	r := slaEngineForTenant(t, client, tenantID)

	url := fmt.Sprintf("/api/v1/sla/performance?dimension=serviceType&startDate=%s&endDate=%s&page=1&pageSize=20",
		f.start.Format(time.RFC3339), f.end.Format(time.RFC3339))
	resp := doSLAReq(t, r, http.MethodGet, url, nil, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	data := resp.Data.(map[string]interface{})

	t.Run("列表信封字段固定", func(t *testing.T) {
		assert.Equal(t, "serviceType", data["dimension"])
		assert.Equal(t, float64(3), slaNum(t, data, "total"))
		assert.Equal(t, float64(1), slaNum(t, data, "page"))
		assert.Equal(t, float64(20), slaNum(t, data, "pageSize"))
		assert.Equal(t, float64(1), slaNum(t, data, "totalPages"))
		assert.Equal(t, false, data["truncated"])
	})

	rows := slaRows(t, data)
	require.Len(t, rows, 3)
	// 工单数降序，同数按 key 升序
	assert.Equal(t, "incident", rows[0]["key"])
	assert.Equal(t, "change", rows[1]["key"])
	assert.Equal(t, SLAPerformanceUnassignedKey, rows[2]["key"])

	incident := rows[0]
	assert.Equal(t, float64(2), slaNum(t, incident, "totalTickets"))
	assert.Equal(t, float64(1), slaNum(t, incident, "resolvedTickets"))
	assert.Equal(t, 50.0, slaNum(t, incident, "resolutionRate"))
	assert.Equal(t, float64(1), slaNum(t, incident, "violatedTickets"))
	assert.Equal(t, 50.0, slaNum(t, incident, "complianceRate"))
	assert.Equal(t, float64(2), slaNum(t, incident, "responseSamples"))
	assert.Equal(t, 50.0, slaNum(t, incident, "responseAchievementRate"))
	assert.Equal(t, float64(1), slaNum(t, incident, "resolutionSamples"))
	assert.Equal(t, 100.0, slaNum(t, incident, "resolutionAchievementRate"))

	// 无样本分组诚实返回 0，不伪装成达标
	change := rows[1]
	assert.Equal(t, float64(1), slaNum(t, change, "totalTickets"))
	assert.Equal(t, float64(0), slaNum(t, change, "responseSamples"))
	assert.Equal(t, 0.0, slaNum(t, change, "responseAchievementRate"))
	assert.Equal(t, 0.0, slaNum(t, change, "resolutionAchievementRate"))
	assert.Equal(t, 100.0, slaNum(t, change, "complianceRate"))

	// 未绑定 SLA 定义的工单必须单独成行
	unbound := rows[2]
	assert.Equal(t, float64(1), slaNum(t, unbound, "totalTickets"))
	assert.Equal(t, float64(1), slaNum(t, unbound, "resolvedTickets"))
	assert.Equal(t, float64(1), slaNum(t, unbound, "resolutionSamples"))
}

func TestSLAHandler_PerformanceByPriorityAndFilters(t *testing.T) {
	_, client, tenantID := setupSLAHandler(t)
	f := seedSLAMonitoringFixture(t, client, tenantID)
	r := slaEngineForTenant(t, client, tenantID)
	window := fmt.Sprintf("startDate=%s&endDate=%s", f.start.Format(time.RFC3339), f.end.Format(time.RFC3339))

	t.Run("按优先级分组", func(t *testing.T) {
		resp := doSLAReq(t, r, http.MethodGet, "/api/v1/sla/performance?dimension=priority&"+window, nil, false)
		require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
		rows := slaRows(t, resp.Data.(map[string]interface{}))
		require.Len(t, rows, 2)
		assert.Equal(t, common.PriorityHigh, rows[0]["key"])
		assert.Equal(t, float64(2), slaNum(t, rows[0], "totalTickets"))
		assert.Equal(t, common.PriorityLow, rows[1]["key"])
		assert.Equal(t, float64(2), slaNum(t, rows[1], "totalTickets"))
	})

	t.Run("服务类型过滤在数据库层生效", func(t *testing.T) {
		resp := doSLAReq(t, r, http.MethodGet, "/api/v1/sla/performance?dimension=priority&serviceType=change&"+window, nil, false)
		require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
		rows := slaRows(t, resp.Data.(map[string]interface{}))
		require.Len(t, rows, 1)
		assert.Equal(t, common.PriorityLow, rows[0]["key"])
		assert.Equal(t, float64(1), slaNum(t, rows[0], "totalTickets"))
	})

	t.Run("unassigned 作为过滤值命中未绑定工单", func(t *testing.T) {
		resp := doSLAReq(t, r, http.MethodGet, "/api/v1/sla/performance?serviceType="+SLAPerformanceUnassignedKey+"&"+window, nil, false)
		require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
		rows := slaRows(t, resp.Data.(map[string]interface{}))
		require.Len(t, rows, 1)
		assert.Equal(t, SLAPerformanceUnassignedKey, rows[0]["key"])
	})

	t.Run("优先级过滤", func(t *testing.T) {
		resp := doSLAReq(t, r, http.MethodGet, "/api/v1/sla/performance?dimension=serviceType&priority="+common.PriorityHigh+"&"+window, nil, false)
		require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
		rows := slaRows(t, resp.Data.(map[string]interface{}))
		require.Len(t, rows, 1)
		assert.Equal(t, "incident", rows[0]["key"])
		assert.Equal(t, float64(2), slaNum(t, rows[0], "totalTickets"))
	})

	t.Run("未知服务类型返回空集合而不是全量", func(t *testing.T) {
		resp := doSLAReq(t, r, http.MethodGet, "/api/v1/sla/performance?serviceType=does-not-exist&"+window, nil, false)
		require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
		data := resp.Data.(map[string]interface{})
		assert.Equal(t, float64(0), slaNum(t, data, "total"))
		assert.Empty(t, data["items"])
	})

	t.Run("分页切片不影响 total", func(t *testing.T) {
		resp := doSLAReq(t, r, http.MethodGet, "/api/v1/sla/performance?dimension=serviceType&page=2&pageSize=2&"+window, nil, false)
		require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
		data := resp.Data.(map[string]interface{})
		rows := slaRows(t, data)
		require.Len(t, rows, 1)
		assert.Equal(t, float64(3), slaNum(t, data, "total"))
		assert.Equal(t, float64(2), slaNum(t, data, "totalPages"))
		assert.Equal(t, SLAPerformanceUnassignedKey, rows[0]["key"])
	})
}

func TestSLAHandler_PerformanceValidatesInput(t *testing.T) {
	_, client, tenantID := setupSLAHandler(t)
	f := seedSLAMonitoringFixture(t, client, tenantID)
	engine := slaEngineForTenant(t, client, tenantID)

	t.Run("不支持的维度返回参数错误", func(t *testing.T) {
		resp, status := doSLAReqRaw(t, engine, http.MethodGet, "/api/v1/sla/performance?dimension=assignee", nil)
		assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", slaStr(resp))
		assert.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("时间格式非法返回参数错误", func(t *testing.T) {
		resp, status := doSLAReqRaw(t, engine, http.MethodGet, "/api/v1/sla/performance?startDate="+f.start.Format("2006-01-02"), nil)
		assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", slaStr(resp))
		assert.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("pageSize 别名 size 不再被接受", func(t *testing.T) {
		resp, _ := doSLAReqRaw(t, engine, http.MethodGet, "/api/v1/sla/performance?size=5", nil)
		require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
		data := resp.Data.(map[string]interface{})
		// size 被忽略，走默认 pageSize=20，避免同一接口支持两套分页契约
		assert.Equal(t, float64(20), slaNum(t, data, "pageSize"))
	})

	t.Run("缺少租户 fail closed", func(t *testing.T) {
		resp, status := doSLAReqRaw(t, engine, http.MethodGet, "/api/v1/sla/performance", nil, true)
		assert.Equal(t, common.AuthFailedCode, resp.Code, "body=%s", slaStr(resp))
		assert.Equal(t, http.StatusUnauthorized, status)
	})
}

// ---- 测试助手 ----

func slaNum(t *testing.T, m map[string]interface{}, key string) float64 {
	t.Helper()
	v, ok := m[key]
	require.True(t, ok, "响应缺少字段 %s: %v", key, m)
	if v == nil {
		return 0
	}
	f, ok := v.(float64)
	require.True(t, ok, "字段 %s 不是数字: %T=%v", key, v, v)
	return f
}

func slaRows(t *testing.T, data map[string]interface{}) []map[string]interface{} {
	t.Helper()
	raw, ok := data["items"].([]interface{})
	require.True(t, ok, "items 必须是数组: %v", data["items"])
	rows := make([]map[string]interface{}, 0, len(raw))
	for _, it := range raw {
		m, ok := it.(map[string]interface{})
		require.True(t, ok, "items 元素必须是对象: %T", it)
		rows = append(rows, m)
	}
	return rows
}

func doSLAReqRaw(t *testing.T, r *gin.Engine, method, path string, body interface{}, skipAuth ...bool) (*common.Response, int) {
	t.Helper()
	var req *http.Request
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		req = httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if len(skipAuth) > 0 && skipAuth[0] {
		req.Header.Set("X-Skip-Auth", "1")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "status=%d body=%s", w.Code, w.Body.String())
	return &resp, w.Code
}
