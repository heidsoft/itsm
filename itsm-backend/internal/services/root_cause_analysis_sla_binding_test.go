package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/slastate"
	slaEngine "itsm-backend/service/sla"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 事件转问题必须与「直接在本域新建问题」共用同一条 SLA 绑定路径。
//
// 缺陷形态（P1-1）：handlers/problem 的 bindSLAOnCreate 是包内私有方法，
// RCA 在共享层无法调用；CreateProblemFromIncident 的事务里也没有任何 SLA
// 写入。结果是「问题管理页新建问题」有 sla_states 行，而「事件详情页转问题」
// 建出来的问题 GET /problems/:id/sla 恒 bound=false —— 同一实体两套事实。
//
// 本组测试锁四件事，缺一不可：
//  1. 配了 problem 类型定义 → 转换后必须有 sla_state 行（主判据，修复前红）
//  2. 没配 problem 定义 → 必须不绑（锁「问题域不套默认 SLA」语义；
//     CalculateSLADeadlineFromRequest 在无匹配时会返回 SLADefinitionID=0
//     并塞上 60/480 分钟默认值，若漏判 ID 就会把隐藏默认值绑到存量问题上，
//     下一轮 watcher 让它们集体违规）
//  3. slaSvc 为 nil → 不 panic 不绑（组合根漏注入时创建流程不得被 SLA 拖垮）
//  4. 绑定失败不得让已提交的问题消失（问题本身是业务成果，SLA 只是附加计时）
//
// 绑定必须发生在 tx.Commit() 之后：sla.Store 持有独立 s.client 句柄而非
// 事务句柄，放进事务内会绕开事务提交（SQLite 下还可能把表锁住）。

// seedRCAConversionFixture 建租户/用户/事件三件套，并按需写入 problem 类型的 SLA 定义。
func seedRCAConversionFixture(
	t *testing.T, ctx context.Context, client *ent.Client, suffix string, withProblemDefinition bool,
) (tenant *ent.Tenant, userEntity *ent.User, incidentEntity *ent.Incident, definitionID int) {
	t.Helper()
	tenant, err := createIncidentTestTenant(ctx, client, suffix)
	require.NoError(t, err)
	userEntity, err = createIncidentTestUser(ctx, client, tenant.ID, suffix)
	require.NoError(t, err)
	incidentEntity, err = client.Incident.Create().
		SetTitle("Repeated API outage " + suffix).
		SetDescription("API repeatedly becomes unavailable").
		SetStatus("resolved").
		SetPriority("critical").
		SetSeverity("critical").
		SetCategory("application").
		SetIncidentNumber("INC-SLA-" + suffix).
		SetReporterID(userEntity.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	if !withProblemDefinition {
		return tenant, userEntity, incidentEntity, 0
	}
	definition, err := client.SLADefinition.Create().
		SetName("Problem SLA " + suffix).
		SetServiceType("problem").
		SetPriority("critical").
		SetResponseTime(30).
		SetResolutionTime(240).
		SetIsActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	return tenant, userEntity, incidentEntity, definition.ID
}

// newRCAWithSLA 组装与组合根同构的 RCA：真实 TicketSLAService + 真实 sla.Store。
func newRCAWithSLA(t *testing.T, client *ent.Client) *RootCauseAnalysisService {
	t.Helper()
	logger := zaptest.NewLogger(t).Sugar()
	rca := NewRootCauseAnalysisService(client)
	rca.SetLogger(logger)
	ticketSLAService := NewTicketSLAService(client, logger)
	ticketSLAService.SetSLAStore(slaEngine.NewStore(client, slaEngine.NewEngine(nil)))
	rca.SetSLAService(ticketSLAService)
	return rca
}

func TestCreateProblemFromIncident_BindsSLADefinitionForProblem(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()

	tenant, userEntity, incidentEntity, definitionID := seedRCAConversionFixture(t, ctx, client, "rca_sla_bound", true)
	rca := newRCAWithSLA(t, client)

	created, err := rca.CreateProblemFromIncident(
		ctx, incidentEntity.ID, userEntity.ID, tenant.ID,
		&dto.ConvertIncidentToProblemRequest{Title: "API stability problem"},
	)
	require.NoError(t, err)

	state, err := client.SLAState.Query().
		Where(
			slastate.TenantID(tenant.ID),
			slastate.AggregateType("problem"),
			slastate.AggregateID(created.ID),
		).
		Only(ctx)
	require.NoError(t, err, "转问题后必须有 problem 类型的 sla_state 行（修复前此处 NotFound）")
	assert.Equal(t, definitionID, state.SLADefinitionID)
	assert.Equal(t, "active", state.Status)
	assert.False(t, state.ResponseDeadline.IsZero(), "响应截止时间必须被计算")
	assert.False(t, state.ResolutionDeadline.IsZero(), "解决截止时间必须被计算")
}

// 截止时间差锁「用的是配置值而非默认兜底」——定义 response=30 / resolution=240
// 时差 210 分钟；若误走 60/480 默认兜底则差 420 分钟。用差值而非绝对值比较，
// 免得 nowFunc 取当前时刻带来的毫秒漂移造成偶发红。
func TestCreateProblemFromIncident_SLADeadlineMatchesDefinition(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()

	tenant, userEntity, incidentEntity, definitionID := seedRCAConversionFixture(t, ctx, client, "rca_sla_deadline", true)
	rca := newRCAWithSLA(t, client)

	created, err := rca.CreateProblemFromIncident(
		ctx, incidentEntity.ID, userEntity.ID, tenant.ID,
		&dto.ConvertIncidentToProblemRequest{},
	)
	require.NoError(t, err)

	state, err := client.SLAState.Query().
		Where(
			slastate.TenantID(tenant.ID),
			slastate.AggregateType("problem"),
			slastate.AggregateID(created.ID),
		).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, definitionID, state.SLADefinitionID)
	delta := state.ResolutionDeadline.Sub(state.ResponseDeadline)
	assert.InDelta(t, 210*60, delta.Seconds(), 1.0,
		"截止时间差应为 (240-30)=210 分钟；若走默认兜底会是 (480-60)=420 分钟")
}

func TestCreateProblemFromIncident_NoProblemDefinitionStaysUnbound(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()

	tenant, userEntity, incidentEntity, _ := seedRCAConversionFixture(t, ctx, client, "rca_sla_unbound", false)
	rca := newRCAWithSLA(t, client)

	created, err := rca.CreateProblemFromIncident(
		ctx, incidentEntity.ID, userEntity.ID, tenant.ID,
		&dto.ConvertIncidentToProblemRequest{},
	)
	require.NoError(t, err)

	count, err := client.SLAState.Query().
		Where(
			slastate.TenantID(tenant.ID),
			slastate.AggregateType("problem"),
			slastate.AggregateID(created.ID),
		).
		Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, count,
		"租户没配 problem 类型 SLA 定义时必须不绑定：CalculateSLADeadlineFromRequest "+
			"在无匹配时会返回 SLADefinitionID=0 并附带 60/480 分钟默认值，漏判 ID 就会把"+
			"隐藏默认值绑上去，下一轮 watcher 让存量问题集体误报违规")
}

func TestCreateProblemFromIncident_WithoutSLAServiceStillSucceeds(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()

	tenant, userEntity, incidentEntity, _ := seedRCAConversionFixture(t, ctx, client, "rca_sla_nilsvc", false)

	// 刻意不注入 SetSLAService：模拟组合根漏接线。
	rca := NewRootCauseAnalysisService(client)
	rca.SetLogger(zaptest.NewLogger(t).Sugar())

	created, err := rca.CreateProblemFromIncident(
		ctx, incidentEntity.ID, userEntity.ID, tenant.ID,
		&dto.ConvertIncidentToProblemRequest{},
	)
	require.NoError(t, err, "SLA 依赖缺失不得让事件转问题整体失败")
	assert.NotZero(t, created.ID)

	count, err := client.SLAState.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, count, "无 SLA 依赖时不应产生任何 sla_state 行")
}

func TestCreateProblemFromIncident_SLAFailureDoesNotLoseProblem(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()

	tenant, userEntity, incidentEntity, _ := seedRCAConversionFixture(t, ctx, client, "rca_sla_tolerate", false)
	rca := newRCAWithSLA(t, client)

	created, err := rca.CreateProblemFromIncident(
		ctx, incidentEntity.ID, userEntity.ID, tenant.ID,
		&dto.ConvertIncidentToProblemRequest{Title: "Tolerate SLA absence"},
	)
	require.NoError(t, err)
	assert.Equal(t, "Tolerate SLA absence", created.Title)
	assert.Equal(t, tenant.ID, created.TenantID)

	// 问题与事件关联边仍在（事务已提交，未被 SLA 环节拖回滚）。
	linked, err := client.Problem.Query().QueryIncidents().Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, incidentEntity.ID, linked.ID)
}
