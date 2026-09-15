package service

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/processinstance"
)

// ---------------------------------------------------------------------------
// timer_phase5_test.go — Timer Start Event（Phase 5）验证
//
// 覆盖三块：
//   A. timer_cron.go 纯函数：表达式分类、按租户时区求下次触发、循环次数语义
//   B. timer_start_schedule.go：发布/部署即注册、重发布整体替换、过期一次性跳过、停用即取消
//   C. timer_event_handler.go：触发即启动实例 + cron 重排，且重放不产生重复时间表
// ---------------------------------------------------------------------------

// ===========================================================================
// A. 表达式解析与时区语义（纯函数）
// ===========================================================================

func TestParseTimerExpression_ClassifiesForms(t *testing.T) {
	cases := []struct {
		name string
		expr string
		want ExpressionType
	}{
		{"ISO duration", "PT30M", ExprTypeDuration},
		{"ISO duration days", "P1D", ExprTypeDuration},
		{"RFC3339 date", "2026-12-31T23:59:59Z", ExprTypeDate},
		{"ISO cycle bounded", "R5/PT10M", ExprTypeCycle},
		{"ISO cycle unbounded", "R/PT1H", ExprTypeCycle},
		{"cron 5-field", "0 9 * * 1-5", ExprTypeCron},
		{"cron with step", "*/15 * * * *", ExprTypeCron},
		{"cron descriptor", "@every 5m", ExprTypeCron},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseTimerExpression(c.expr)
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestParseTimerExpression_RejectsGarbage(t *testing.T) {
	for _, expr := range []string{"", "   ", "不是表达式", "PT", "0 9 * *"} {
		_, err := ParseTimerExpression(expr)
		assert.Error(t, err, "表达式 %q 应被拒绝", expr)
	}
}

func TestNextFireAt_CronHonoursTimezone(t *testing.T) {
	shanghai := ResolveLocation("Asia/Shanghai")
	newYork := ResolveLocation("America/New_York")

	// 2026-09-15T00:00:00Z == 上海 08:00、纽约 20:00（前一天）
	now := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)

	fireSH, err := NextFireAt("0 9 * * *", ExprTypeCron, shanghai, now)
	require.NoError(t, err)
	fireNY, err := NextFireAt("0 9 * * *", ExprTypeCron, newYork, now)
	require.NoError(t, err)

	// 同一条 cron 在各自时区的"本地 09:00"触发
	assert.Equal(t, 9, fireSH.In(shanghai).Hour(), "上海：本地 9 点")
	assert.Equal(t, 9, fireNY.In(newYork).Hour(), "纽约：本地 9 点")

	// 上海无夏令时，恒为 +08 ⇒ UTC 01:00
	assert.Equal(t, 1, fireSH.In(time.UTC).Hour(), "上海 09:00 == UTC 01:00")

	// 时区不同 ⇒ 绝对时刻必须不同（这正是"按租户时区"的意义）
	assert.NotEqual(t, fireSH.UTC(), fireNY.UTC(), "不同时区不应得到同一绝对时刻")

	// cron 语义：下次触发严格晚于参考时刻
	assert.True(t, fireSH.After(now))
	assert.True(t, fireNY.After(now))
}

func TestNextFireAt_CronDefaultLocationWhenNil(t *testing.T) {
	now := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	fire, err := NextFireAt("0 9 * * *", ExprTypeCron, nil, now)
	require.NoError(t, err)
	assert.Equal(t, 9, fire.In(ResolveLocation("")).Hour(), "loc 为 nil 时回退默认时区")
}

func TestNextFireAt_DurationDateCycle(t *testing.T) {
	now := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	loc := ResolveLocation("Asia/Shanghai")

	// duration：after + d
	fire, err := NextFireAt("PT30M", ExprTypeDuration, loc, now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(30*time.Minute).Unix(), fire.Unix())

	// date：绝对时间，与 after 无关
	fire, err = NextFireAt("2027-01-01T00:00:00Z", ExprTypeDate, loc, now)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC).Unix(), fire.Unix())

	// cycle：after + 间隔（首跳）
	fire, err = NextFireAt("R5/PT10M", ExprTypeCycle, loc, now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(10*time.Minute).Unix(), fire.Unix())
}

func TestNextFireAt_RejectsMalformedExpression(t *testing.T) {
	now := time.Now()
	loc := ResolveLocation("Asia/Shanghai")

	_, err := NextFireAt("PTxxX", ExprTypeDuration, loc, now)
	assert.Error(t, err, "非法 duration")

	_, err = NextFireAt("not-a-date", ExprTypeDate, loc, now)
	assert.Error(t, err, "非法 date")

	_, err = NextFireAt("R5", ExprTypeCycle, loc, now)
	assert.Error(t, err, "非法 cycle")

	_, err = NextFireAt("0 9 * *", ExprTypeCron, loc, now)
	assert.Error(t, err, "字段不足的 cron")
}

func TestResolveLocation_FallsBackToDefault(t *testing.T) {
	def := ResolveLocation("")
	require.NotNil(t, def)
	assert.Equal(t, DefaultTimerTimezone, def.String())

	// 非法时区名不 panic、不返回 nil，而是回退
	invalid := ResolveLocation("Not/AZone")
	require.NotNil(t, invalid)
	assert.Equal(t, DefaultTimerTimezone, invalid.String())

	valid := ResolveLocation("America/New_York")
	assert.Equal(t, "America/New_York", valid.String())
}

func TestCycleRemaining_Semantics(t *testing.T) {
	// 有限次数
	n, bounded, ok := CycleRemaining("R5/PT10M")
	assert.True(t, ok)
	assert.True(t, bounded)
	assert.Equal(t, 5, n)

	// 无限重复
	n, bounded, ok = CycleRemaining("R/PT1H")
	assert.True(t, ok)
	assert.False(t, bounded)
	assert.Equal(t, -1, n)

	// 无 R 前缀按一次性
	n, bounded, ok = CycleRemaining("PT10M")
	assert.True(t, ok)
	assert.True(t, bounded)
	assert.Equal(t, 1, n)

	// 非法：有 '/' 但前缀不是 R
	_, _, ok = CycleRemaining("X5/PT10M")
	assert.False(t, ok)

	// 无 '/' 输入按"一次性"处理（既定语义，调用方需按 exprType 区分，勿据此判定 cron）
	n, bounded, ok = CycleRemaining("0 9 * * *")
	assert.True(t, ok)
	assert.True(t, bounded)
	assert.Equal(t, 1, n, "无 '/' 的字符串被判为一次性——cron 必须靠 exprType 区分，不能走此分支")
}

func TestIsRecurring(t *testing.T) {
	assert.True(t, IsRecurring("0 9 * * *", ExprTypeCron))
	assert.True(t, IsRecurring("R5/PT10M", ExprTypeCycle))
	assert.False(t, IsRecurring("PT30M", ExprTypeDuration))
	assert.False(t, IsRecurring("2026-12-31T00:00:00Z", ExprTypeDate))
}

// ===========================================================================
// B. 时间表同步（timer_start_schedule.go）
// ===========================================================================

const phase5CronStartBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="cron_proc" name="Nightly Cron Process" isExecutable="true">
    <bpmn:startEvent id="start_cron" name="Every day 09:00">
      <bpmn:timerEventDefinition>
        <bpmn:timeCycle>0 9 * * *</bpmn:timeCycle>
      </bpmn:timerEventDefinition>
    </bpmn:startEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start_cron" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

const phase5ExpiredDateStartBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="once_proc" name="One-off In The Past" isExecutable="true">
    <bpmn:startEvent id="start_once" name="Once in the past">
      <bpmn:timerEventDefinition>
        <bpmn:timeDate>2020-01-01T00:00:00Z</bpmn:timeDate>
      </bpmn:timerEventDefinition>
    </bpmn:startEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start_once" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

const phase5PlainStartEndBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="cron_proc" name="Cron Target Process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

// createTimerTenantWithTZ 建租户并显式指定时区（Phase 5 的 cron 语义依赖它）。
func createTimerTenantWithTZ(t *testing.T, client *ent.Client, suffix, tz string) int {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName("Timer TZ Tenant "+suffix).
		SetCode("timer-tz-"+suffix).
		SetDomain("timer-tz-" + suffix + ".example.com").
		SetStatus("active").
		SetTimezone(tz).
		Save(context.Background())
	require.NoError(t, err)
	return tenant.ID
}

func TestSyncStartTimers_RegistersCronInTenantTimezone(t *testing.T) {
	client := newTimerTestClient(t, "sync-start-tz")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenantWithTZ(t, client, "sync", "Asia/Shanghai")
	ctx := context.Background()

	registered, err := SyncStartTimers(ctx, client, store, []byte(phase5CronStartBPMN), "cron_proc", tenantID)
	require.NoError(t, err)
	require.Equal(t, 1, registered, "应注册 1 个 start timer")

	timers, err := ListActiveStartTimers(ctx, client, tenantID, "cron_proc")
	require.NoError(t, err)
	require.Len(t, timers, 1)

	timer := timers[0]
	assert.Equal(t, string(TimerTypeStart), timer.TimerType)
	assert.Equal(t, "start_cron", timer.ActivityID)
	assert.Equal(t, string(ExprTypeCron), timer.ExpressionType)
	assert.Equal(t, "0 9 * * *", timer.TimerExpression)

	// 落库为 UTC，但按租户时区应读作本地 09:00
	loc := ResolveLocation("Asia/Shanghai")
	assert.Equal(t, 9, timer.FireAt.In(loc).Hour(), "fire_at 应按租户时区落在本地 09:00")
	assert.True(t, timer.FireAt.After(time.Now()), "下次触发应在未来")
}

func TestSyncStartTimers_ReplacesOldScheduleOnRedeploy(t *testing.T) {
	client := newTimerTestClient(t, "sync-start-dedup")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenantWithTZ(t, client, "dedup", "Asia/Shanghai")
	ctx := context.Background()

	// 连续两次同步（模拟重复发布/部署）
	for i := 0; i < 2; i++ {
		registered, err := SyncStartTimers(ctx, client, store, []byte(phase5CronStartBPMN), "cron_proc", tenantID)
		require.NoError(t, err)
		require.Equal(t, 1, registered, "第 %d 次同步应仍只注册 1 个", i+1)
	}

	pending, err := ListActiveStartTimers(ctx, client, tenantID, "cron_proc")
	require.NoError(t, err)
	assert.Len(t, pending, 1, "重发布必须整体替换旧时间表，不能累积")
}

func TestSyncStartTimers_SkipsExpiredOneOff(t *testing.T) {
	client := newTimerTestClient(t, "sync-start-expired")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenantWithTZ(t, client, "expired", "Asia/Shanghai")
	ctx := context.Background()

	registered, err := SyncStartTimers(ctx, client, store, []byte(phase5ExpiredDateStartBPMN), "once_proc", tenantID)
	require.NoError(t, err)
	assert.Equal(t, 0, registered, "已过期的一次性定时启动不应注册")

	pending, err := ListActiveStartTimers(ctx, client, tenantID, "once_proc")
	require.NoError(t, err)
	assert.Empty(t, pending)
}

func TestSyncStartTimers_NoStoreIsNoop(t *testing.T) {
	client := newTimerTestClient(t, "sync-start-nostore")
	ctx := context.Background()
	registered, err := SyncStartTimers(ctx, client, nil, []byte(phase5CronStartBPMN), "cron_proc", 1)
	require.NoError(t, err)
	assert.Equal(t, 0, registered)
}

// TestSyncStartTimers_ProductTemplate_IsSchedulable 把随包发布的产品模板
// 与 Phase 5 功能锁在一起：模板里的 Timer Start Event 必须能真正注册成时间表。
// 若模板被改动导致 cron 非法/被误删，本测试立即失败。
func TestSyncStartTimers_ProductTemplate_IsSchedulable(t *testing.T) {
	xmlBytes, err := os.ReadFile("bpmn/scheduled_inspection_flow.bpmn")
	require.NoError(t, err, "内置模板 scheduled_inspection_flow.bpmn 必须存在")

	client := newTimerTestClient(t, "sync-start-template")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenantWithTZ(t, client, "template", "Asia/Shanghai")
	ctx := context.Background()

	registered, err := SyncStartTimers(ctx, client, store, xmlBytes, "scheduled_inspection_flow", tenantID)
	require.NoError(t, err)
	require.Equal(t, 1, registered, "产品模板应注册 1 个定时启动")

	timers, err := ListActiveStartTimers(ctx, client, tenantID, "scheduled_inspection_flow")
	require.NoError(t, err)
	require.Len(t, timers, 1)
	assert.Equal(t, string(ExprTypeCron), timers[0].ExpressionType)
	assert.Equal(t, "StartEvent_Schedule", timers[0].ActivityID)

	loc := ResolveLocation("Asia/Shanghai")
	assert.Equal(t, 9, timers[0].FireAt.In(loc).Hour(), "模板 cron 0 9 * * * 应落在本地 09:00")
}

func TestCancelStartTimers(t *testing.T) {
	client := newTimerTestClient(t, "cancel-start")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenantWithTZ(t, client, "cancel", "Asia/Shanghai")
	ctx := context.Background()

	_, err := SyncStartTimers(ctx, client, store, []byte(phase5CronStartBPMN), "cron_proc", tenantID)
	require.NoError(t, err)

	cancelled, err := CancelStartTimers(ctx, client, "cron_proc", tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, cancelled)

	pending, err := ListActiveStartTimers(ctx, client, tenantID, "cron_proc")
	require.NoError(t, err)
	assert.Empty(t, pending, "取消后不应再有 pending 的 start timer")
}

// ===========================================================================
// C. 触发处理（timer_event_handler.go）
// ===========================================================================

func TestHandleStartTimer_LaunchesInstanceAndRearmsCron(t *testing.T) {
	client := newE2ETestClient(t, "phase5_start_launch")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "p5launch")

	deployE2EProcess(t, ctx, client, tenantID, "cron_proc", phase5PlainStartEndBPMN)

	store := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(store, nil)
	handler := NewTimerEventHandler(engine, logger)
	handler.SetTimerStore(store)

	timer := &TimerRecord{
		TimerID:              "timer-p5-launch",
		TimerType:            string(TimerTypeStart),
		ProcessDefinitionKey: "cron_proc",
		ActivityID:           "start_cron",
		FireAt:               time.Now(),
		TenantID:             tenantID,
		TimerExpression:      "0 9 * * *",
		ExpressionType:       string(ExprTypeCron),
	}

	require.NoError(t, handler.HandleTimerFire(ctx, timer))

	// 1) 流程实例已按幂等 businessKey 启动
	businessKey := "timer:timer-p5-launch:" + strconv.FormatInt(timer.FireAt.Unix(), 10)
	instances, err := client.ProcessInstance.Query().
		Where(
			processinstance.BusinessKey(businessKey),
			processinstance.TenantID(tenantID),
		).All(ctx)
	require.NoError(t, err)
	assert.Len(t, instances, 1, "start timer 应启动且仅启动一个流程实例")

	// 2) cron 已重排下一跳
	pending, total, err := store.List(ctx, TimerListFilter{
		TenantID:             tenantID,
		Status:               string(TimerStatusPending),
		TimerType:            string(TimerTypeStart),
		ProcessDefinitionKey: "cron_proc",
	})
	require.NoError(t, err)
	require.Equal(t, 1, total, "cron 触发后必须重排恰好一条下一跳")
	assert.Equal(t, string(ExprTypeCron), pending[0].ExpressionType)
	assert.True(t, pending[0].FireAt.After(time.Now()), "下一跳应在未来")
}

func TestHandleStartTimer_ReentrantFireDoesNotDuplicateSchedule(t *testing.T) {
	client := newE2ETestClient(t, "phase5_start_reentrant")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "p5reent")

	deployE2EProcess(t, ctx, client, tenantID, "cron_proc", phase5PlainStartEndBPMN)

	store := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(store, nil)
	handler := NewTimerEventHandler(engine, logger)
	handler.SetTimerStore(store)

	timer := &TimerRecord{
		TimerID:              "timer-p5-reentrant",
		TimerType:            string(TimerTypeStart),
		ProcessDefinitionKey: "cron_proc",
		ActivityID:           "start_cron",
		FireAt:               time.Now(),
		TenantID:             tenantID,
		TimerExpression:      "0 9 * * *",
		ExpressionType:       string(ExprTypeCron),
	}

	// 崩溃恢复重放：同一 timer 记录被投递两次
	require.NoError(t, handler.HandleTimerFire(ctx, timer))
	require.NoError(t, handler.HandleTimerFire(ctx, timer))

	businessKey := "timer:timer-p5-reentrant:" + strconv.FormatInt(timer.FireAt.Unix(), 10)
	count, err := client.ProcessInstance.Query().
		Where(processinstance.BusinessKey(businessKey)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "幂等闸门：同一计划触发只允许一个实例")

	_, total, err := store.List(ctx, TimerListFilter{
		TenantID:             tenantID,
		Status:               string(TimerStatusPending),
		TimerType:            string(TimerTypeStart),
		ProcessDefinitionKey: "cron_proc",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total, "重排也必须幂等：重放不得把 cron 变成双份时间表")
}

func TestHandleStartTimer_BoundedCycleStopsAtLastRepetition(t *testing.T) {
	client := newE2ETestClient(t, "phase5_start_cycle_end")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "p5cycleend")

	deployE2EProcess(t, ctx, client, tenantID, "cron_proc", phase5PlainStartEndBPMN)

	store := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(store, nil)
	handler := NewTimerEventHandler(engine, logger)
	handler.SetTimerStore(store)

	// 剩余次数为 1：本次即最后一次，触发后不得再重排
	timer := &TimerRecord{
		TimerID:              "timer-p5-cycle-last",
		TimerType:            string(TimerTypeStart),
		ProcessDefinitionKey: "cron_proc",
		ActivityID:           "start_cron",
		FireAt:               time.Now(),
		TenantID:             tenantID,
		TimerExpression:      "R5/PT10M",
		ExpressionType:       string(ExprTypeCycle),
		ContextVariables:     map[string]interface{}{"remaining_repetitions": 1},
	}

	require.NoError(t, handler.HandleTimerFire(ctx, timer))

	_, total, err := store.List(ctx, TimerListFilter{
		TenantID:             tenantID,
		Status:               string(TimerStatusPending),
		TimerType:            string(TimerTypeStart),
		ProcessDefinitionKey: "cron_proc",
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total, "有限 cycle 用尽后不应再重排")
}

func TestHandleStartTimer_BoundedCycleRearmsWhileRemaining(t *testing.T) {
	client := newE2ETestClient(t, "phase5_start_cycle_more")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "p5cyclemore")

	deployE2EProcess(t, ctx, client, tenantID, "cron_proc", phase5PlainStartEndBPMN)

	store := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(store, nil)
	handler := NewTimerEventHandler(engine, logger)
	handler.SetTimerStore(store)

	timer := &TimerRecord{
		TimerID:              "timer-p5-cycle-more",
		TimerType:            string(TimerTypeStart),
		ProcessDefinitionKey: "cron_proc",
		ActivityID:           "start_cron",
		FireAt:               time.Now(),
		TenantID:             tenantID,
		TimerExpression:      "R5/PT10M",
		ExpressionType:       string(ExprTypeCycle),
		ContextVariables:     map[string]interface{}{"remaining_repetitions": 3},
	}

	require.NoError(t, handler.HandleTimerFire(ctx, timer))

	_, total, err := store.List(ctx, TimerListFilter{
		TenantID:             tenantID,
		Status:               string(TimerStatusPending),
		TimerType:            string(TimerTypeStart),
		ProcessDefinitionKey: "cron_proc",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total, "剩余次数 > 1 时应重排下一跳")
}
