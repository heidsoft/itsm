package cmdb

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"itsm-backend/ent/discoveryjob"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"

	_ "github.com/mattn/go-sqlite3"
)

// 云发现调度（cron 消费者）回归锁。
//
// 背景：`discovery_sources.schedule` 自建表起就存在，但全仓没有任何消费者——
// 运维填了 `0 2 * * *` 既不生效也无任何反馈。本文件锁两件事：
//  1. 到期判定（planDiscoveryRun 纯函数）：补跑、不重复、在途守卫、时区、非法表达式；
//  2. 端到端（sqlite）：到期的来源真的产出 queued job + pending 命令，
//     且第二轮不会重复排。

func TestPlanDiscoveryRun(t *testing.T) {
	cst := time.FixedZone("CST", 8*3600)
	// 2026-10-23 02:00:30（租户本地时间），即 cron 触发点刚过 30 秒。
	now := time.Date(2026, 10, 23, 2, 0, 30, 0, cst)
	grace := 2 * time.Minute

	t.Run("从未跑过且窗口内有触发点则到期", func(t *testing.T) {
		plan, err := planDiscoveryRun("0 2 * * *", nil, "", now, cst, grace)
		require.NoError(t, err)
		require.True(t, plan.Due)
		require.Equal(t, time.Date(2026, 10, 23, 2, 0, 0, 0, cst).UTC(), plan.FireAt.UTC())
	})

	t.Run("窗口内没有触发点则不到期", func(t *testing.T) {
		// 02:10 扫描，"0 2 * * *" 的下一次是明天 02:00。
		later := now.Add(10 * time.Minute)
		plan, err := planDiscoveryRun("0 2 * * *", nil, "", later, cst, grace)
		require.NoError(t, err)
		require.False(t, plan.Due)
		require.Equal(t, time.Date(2026, 10, 24, 2, 0, 0, 0, cst).UTC(), plan.FireAt.UTC())
	})

	t.Run("刚为同一触发点排过任务则不重复", func(t *testing.T) {
		queued := time.Date(2026, 10, 23, 2, 0, 5, 0, cst)
		plan, err := planDiscoveryRun("0 2 * * *", &queued, DiscoveryJobQueued, now, cst, grace)
		require.NoError(t, err)
		// 在途守卫先命中，原因要说清楚，否则日志里只看到「跳过了」。
		require.False(t, plan.Due)
		require.Contains(t, plan.Reason, "仍在执行")

		// 即便上一次已终态，同一触发点也不该重排（锚点已推进到 02:00:05）。
		plan, err = planDiscoveryRun("0 2 * * *", &queued, DiscoveryJobSucceeded, now, cst, grace)
		require.NoError(t, err)
		require.False(t, plan.Due)
		require.Equal(t, time.Date(2026, 10, 24, 2, 0, 0, 0, cst).UTC(), plan.FireAt.UTC())
	})

	t.Run("停机两天后补跑一次而不是补跑两天", func(t *testing.T) {
		stale := now.Add(-48 * time.Hour)
		plan, err := planDiscoveryRun("0 2 * * *", &stale, DiscoveryJobSucceeded, now, cst, grace)
		require.NoError(t, err)
		require.True(t, plan.Due)
		// 补跑取的是「锚点之后最近的一个触发点」，不是逐个补。
		require.Equal(t, time.Date(2026, 10, 22, 2, 0, 0, 0, cst).UTC(), plan.FireAt.UTC())
	})

	t.Run("失败任务也算跑过不触发每分钟重试", func(t *testing.T) {
		failedAt := now.Add(-30 * time.Second)
		plan, err := planDiscoveryRun("0 2 * * *", &failedAt, DiscoveryJobFailed, now, cst, grace)
		require.NoError(t, err)
		require.False(t, plan.Due)
	})

	t.Run("在途状态全部拦截终态全部放行", func(t *testing.T) {
		old := now.Add(-72 * time.Hour)
		for _, status := range []string{DiscoveryJobQueued, DiscoveryJobDiscovering, DiscoveryJobDiscovered, DiscoveryJobReconciling} {
			plan, err := planDiscoveryRun("0 2 * * *", &old, status, now, cst, grace)
			require.NoError(t, err)
			require.Falsef(t, plan.Due, "在途状态 %s 不应触发新任务", status)
		}
		for _, status := range []string{DiscoveryJobSucceeded, DiscoveryJobPartialFailed, DiscoveryJobFailed, DiscoveryJobCancelled} {
			plan, err := planDiscoveryRun("0 2 * * *", &old, status, now, cst, grace)
			require.NoError(t, err)
			require.Truef(t, plan.Due, "终态 %s 之后到期应触发", status)
		}
	})

	t.Run("空表达式不调度", func(t *testing.T) {
		plan, err := planDiscoveryRun("   ", nil, "", now, cst, grace)
		require.NoError(t, err)
		require.False(t, plan.Due)
	})

	t.Run("非法表达式返回错误而不是静默不跑", func(t *testing.T) {
		// 6 段带秒的 cron 是常见误配：BPMN 与发现调度都只接受 5 段。
		_, err := planDiscoveryRun("0 2 * * * *", nil, "", now, cst, grace)
		require.Error(t, err)
		_, err = planDiscoveryRun("每天凌晨两点", nil, "", now, cst, grace)
		require.Error(t, err)
	})

	t.Run("cron 按租户时区解释", func(t *testing.T) {
		// 同一瞬间：UTC 02:00:30 = CST 10:00:30。
		// 用 UTC 解释时下一次是 10-23 02:00 UTC；用 CST 解释时下一次是
		// 10-24 02:00 CST（= 10-23 18:00 UTC），差 8 小时。
		sameInstant := time.Date(2026, 10, 23, 2, 0, 30, 0, time.UTC)
		planUTC, err := planDiscoveryRun("0 2 * * *", nil, "", sameInstant, time.UTC, grace)
		require.NoError(t, err)
		require.True(t, planUTC.Due)
		require.Equal(t, time.Date(2026, 10, 23, 2, 0, 0, 0, time.UTC), planUTC.FireAt.UTC())

		planCST, err := planDiscoveryRun("0 2 * * *", nil, "", sameInstant.In(cst), cst, grace)
		require.NoError(t, err)
		require.False(t, planCST.Due)
		require.Equal(t, time.Date(2026, 10, 24, 2, 0, 0, 0, cst).UTC(), planCST.FireAt.UTC())
	})

	t.Run("间隔表达式按上次入队时间推进", func(t *testing.T) {
		last := now.Add(-25 * time.Hour)
		plan, err := planDiscoveryRun("@every 24h", &last, DiscoveryJobSucceeded, now, cst, grace)
		require.NoError(t, err)
		require.True(t, plan.Due)
		require.Equal(t, last.Add(24*time.Hour).UTC(), plan.FireAt.UTC())
	})
}

func TestSchedulePreview(t *testing.T) {
	cst := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 10, 23, 10, 0, 0, 0, cst)

	next, exprErr := schedulePreview("0 2 * * *", now, cst)
	require.Empty(t, exprErr)
	require.NotNil(t, next)
	require.Equal(t, time.Date(2026, 10, 24, 2, 0, 0, 0, cst).UTC(), next.UTC())

	// 配置错误必须回显：这正是「填了 cron 没反应」最难排查的形态。
	next, exprErr = schedulePreview("0 2 * * * *", now, cst)
	require.Nil(t, next)
	require.NotEmpty(t, exprErr)

	next, exprErr = schedulePreview("", now, cst)
	require.Nil(t, next)
	require.Empty(t, exprErr)
}

func TestDiscoverySchedulerRejectsMissingTenant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_sched_notenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	repo := NewEntRepository(client)
	sched := NewDiscoveryScheduler(repo, NewService(repo, nil, nopLogger()), nopLogger())

	_, err := sched.RunTenant(context.Background(), 0, time.Now())
	require.Error(t, err)
}

// seedScheduledSource 建一个「启用 + 配了每分钟调度」的来源，返回租户与来源 ID。
func seedScheduledSource(t *testing.T, repo *EntRepository, code, schedule string, enabled bool) (int, string) {
	t.Helper()
	ctx := context.Background()
	tenant, err := repo.client.Tenant.Create().SetName("T-" + code).SetCode(code).
		SetDomain(code + ".example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	source, err := repo.CreateDiscoverySource(ctx, &DiscoverySource{
		ID: "ds-" + code, Name: "调度源", SourceType: "api", Provider: "aliyun",
		ServiceCodes: []string{"ecs"}, Regions: []string{"cn-hangzhou"},
		Schedule: schedule, ReconcilePolicy: "manual", StaleThreshold: 2,
		IsActive: enabled, TenantID: tenant.ID,
	})
	require.NoError(t, err)
	return tenant.ID, source.ID
}

// 端到端：到期的来源真的产出 queued job + pending 命令；第二轮不重复；
// 在途/停用/未配调度都不进扫描集。
func TestDiscoveryScheduler_EndToEnd(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_sched_e2e?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	repo := NewEntRepository(client)

	// 每分钟触发，保证「此刻」一定落在触发窗口内（不依赖测试运行的具体时刻）。
	tenantID, sourceID := seedScheduledSource(t, repo, "sched-e2e", "* * * * *", true)
	// 停用与未配调度的来源不应进入扫描集。
	_, _ = seedScheduledSource(t, repo, "sched-off", "* * * * *", false)
	_, _ = seedScheduledSource(t, repo, "sched-noschedule", "", true)

	sched := NewDiscoveryScheduler(repo, NewService(repo, nil, nopLogger()), nopLogger())
	now := time.Now()

	result, err := sched.RunTenant(ctx, tenantID, now)
	require.NoError(t, err)
	require.Equal(t, 1, result.Scanned, "只有启用且配了调度的来源参与扫描")
	require.Equal(t, 1, result.Created)
	require.Zero(t, result.Failed)

	jobs, err := client.DiscoveryJob.Query().Where(discoveryjob.TenantID(tenantID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, DiscoveryJobQueued, jobs[0].Status)
	require.Equal(t, sourceID, jobs[0].SourceID)
	// 幂等键由「来源 + 触发点」派生，跨副本可复现（discovery_jobs 上
	// 幂等键只是普通索引，靠先查后插去重存在窗口，键必须确定性）。
	require.True(t, strings.HasPrefix(jobs[0].IdempotencyKey, "sched-"+sourceID+"-"),
		"unexpected idempotency key: %s", jobs[0].IdempotencyKey)

	cmd, err := client.OperationalCommand.Query().
		Where(operationalcommand.CommandType(commandbus.CommandRunCMDBCloudDiscovery)).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "pending", cmd.Status)
	require.EqualValues(t, jobs[0].ID, cmd.Payload["discoveryJobId"])

	// 第二轮：上一轮任务仍在途，不得再排一条。
	result, err = sched.RunTenant(ctx, tenantID, now.Add(time.Minute))
	require.NoError(t, err)
	require.Zero(t, result.Created)
	require.Equal(t, 1, result.Skipped)

	// 上一轮结束（终态）后，同一分钟窗口内也不该立刻重排：
	// 锚点已推进到「刚排过」的时刻，下一次是下一个分钟边界。
	_, err = client.DiscoveryJob.UpdateOneID(jobs[0].ID).SetStatus(DiscoveryJobSucceeded).Save(ctx)
	require.NoError(t, err)
	result, err = sched.RunTenant(ctx, tenantID, now.Add(2*time.Second))
	require.NoError(t, err)
	require.Zero(t, result.Created)

	// 一个完整周期之后必须再排上：调度不能只在第一次生效。
	result, err = sched.RunTenant(ctx, tenantID, now.Add(2*time.Minute))
	require.NoError(t, err)
	require.Equal(t, 1, result.Created)

	total, err := client.DiscoveryJob.Query().Where(discoveryjob.TenantID(tenantID)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, total)
}

func TestDiscoveryScheduler_ManualJobDoesNotSuppressScheduledOccurrence(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_sched_manual_anchor?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	repo := NewEntRepository(client)
	tenantID, sourceID := seedScheduledSource(t, repo, "sched-manual-anchor", "0 2 * * *", true)

	loc := time.FixedZone("CST", 8*3600)
	fire := time.Date(2026, 10, 23, 2, 0, 0, 0, loc)
	_, err := client.DiscoveryJob.Create().
		SetSourceID(sourceID).SetTenantID(tenantID).
		SetStatus(DiscoveryJobSucceeded).SetIdempotencyKey("manual-1").
		SetQueuedAt(fire.Add(time.Minute)).Save(ctx)
	require.NoError(t, err)

	sched := NewDiscoveryScheduler(repo, NewService(repo, nil, nopLogger()), nopLogger())
	sched.SetLocationResolver(func(context.Context, int) *time.Location { return loc })
	result, err := sched.RunTenant(ctx, tenantID, fire.Add(30*time.Second))
	require.NoError(t, err)
	require.Equal(t, 1, result.Created, "手动任务不得吞掉 02:00 的 scheduled occurrence")
}

// 停用来源后调度立即停止：扫描集按 enabled 过滤，不依赖 last_success_at。
func TestDiscoverySchedulerSkipsDisabledSource(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_sched_disabled?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	repo := NewEntRepository(client)

	tenantID, sourceID := seedScheduledSource(t, repo, "sched-disable", "* * * * *", true)
	sched := NewDiscoveryScheduler(repo, NewService(repo, nil, nopLogger()), nopLogger())
	now := time.Now()

	result, err := sched.RunTenant(ctx, tenantID, now)
	require.NoError(t, err)
	require.Equal(t, 1, result.Created)

	require.NoError(t, repo.client.DiscoverySource.UpdateOneID(sourceID).SetEnabled(false).Exec(ctx))
	result, err = sched.RunTenant(ctx, tenantID, now.Add(5*time.Minute))
	require.NoError(t, err)
	require.Zero(t, result.Scanned)
	require.Zero(t, result.Created)

	total, err := client.DiscoveryJob.Query().Where(discoveryjob.TenantID(tenantID)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, total)
}

// 非法表达式的来源只告警不阻断同租户其他来源。
func TestDiscoverySchedulerInvalidScheduleDoesNotBlockOthers(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_sched_invalid?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	repo := NewEntRepository(client)

	tenantID, _ := seedScheduledSource(t, repo, "sched-good", "* * * * *", true)
	// 同一租户下的坏表达式来源：手写 ent 直插，绕过 CreateDiscoverySource 的校验。
	_, err := repo.client.DiscoverySource.Create().
		SetID("ds-bad-cron").SetName("坏表达式").SetSourceType("api").SetProvider("aliyun").
		SetSchedule("0 2 * * * *").SetReconcilePolicy("manual").SetStaleThreshold(2).
		SetEnabled(true).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)

	sched := NewDiscoveryScheduler(repo, NewService(repo, nil, nopLogger()), nopLogger())
	result, err := sched.RunTenant(ctx, tenantID, time.Now())
	require.NoError(t, err)
	require.Equal(t, 2, result.Scanned)
	require.Equal(t, 1, result.Created, "坏表达式跳过，好来源照常调度")
	require.Equal(t, 1, result.Skipped)
	require.Zero(t, result.Failed, "配置错误不算运行时故障")
}

// 同一触发点被两个副本同时扫到，只能产生一条任务。
func TestDiscoverySchedulerDeduplicatesAcrossReplicas(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_sched_replicas?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	repo := NewEntRepository(client)

	tenantID, _ := seedScheduledSource(t, repo, "sched-replica", "* * * * *", true)
	now := time.Now()

	first := NewDiscoveryScheduler(repo, NewService(repo, nil, nopLogger()), nopLogger())
	second := NewDiscoveryScheduler(repo, NewService(repo, nil, nopLogger()), nopLogger())

	r1, err := first.RunTenant(ctx, tenantID, now)
	require.NoError(t, err)
	r2, err := second.RunTenant(ctx, tenantID, now)
	require.NoError(t, err)

	require.Equal(t, 1, r1.Created)
	require.Zero(t, r2.Created)
	total, err := client.DiscoveryJob.Query().Where(discoveryjob.TenantID(tenantID)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, total)
}

// 触发点不同则各自成任务（幂等键必须随触发点变化，否则第二天的调度会被判重）。
func TestScheduledIdempotencyKeyVariesByFireTime(t *testing.T) {
	fire := time.Date(2026, 10, 23, 2, 0, 0, 0, time.UTC)
	next := fire.Add(24 * time.Hour)
	require.NotEqual(t,
		scheduledDiscoveryIdempotencyKey("ds-1", fire),
		scheduledDiscoveryIdempotencyKey("ds-1", next))
	require.Equal(t,
		fmt.Sprintf("sched-ds-1-%d", fire.Unix()),
		scheduledDiscoveryIdempotencyKey("ds-1", fire))
}
