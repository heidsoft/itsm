package cmdb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/service"
)

// DiscoveryScheduler 消费 `discovery_sources.schedule`。
//
// 背景：`schedule` 字段从建表起就存在（「调度表达式」），但全仓没有任何消费者——
// 运维在来源里填了 `0 2 * * *`，系统既不会在 2 点跑，也不会提示这个配置没生效。
// 云发现因此只有「手动点一次」这一条路径，巡检语义完全缺失；这也是该能力
// 维持 Pilot 的阻断项之一（另两条是仅阿里云 ECS 一种适配器、发现结果未回写 CI）。
//
// 设计约束：
//   - **复用创建链路**：调度不自己写 discovery_jobs，而是走
//     Service.CreateDiscoveryJob——job 落库与命令入队的事务边界只有一份实现，
//     调度器不可能产出「有 job 没命令」的僵尸任务。
//   - **到期判定以「上一次入队」为锚点**，而不是 last_success_at：失败的任务
//     也算「跑过了」，否则一个持续失败的来源会被每分钟重试一次。锚点取该来源
//     最近一条 job 的 queued_at；从未跑过时用「now - 宽限窗口」，只在宽限窗口
//     内确实有触发点时才算到期（避免首次启动就把所有来源全部补跑一遍）。
//   - **在途守卫**：该来源还有非终态 job（queued/discovering/discovered/
//     reconciling）时跳过——cron 每分钟触发一次，而一次发现可能跑很久，
//     不守卫会让任务堆积成队列。
//   - **幂等键按触发点派生**：`sched-<sourceID>-<fireAt unix>`。多副本同时扫描
//     同一触发点时算出同一个键，后到的那个拿 409（视为「已被别的副本排上」，
//     不算失败）。这是本仓库唯一能跨副本去重的手段——discovery_jobs 上
//     (tenant, operation, source, idempotency_key) 只是普通索引，服务层
//     先查后插存在窗口，所以幂等键必须确定性，不能带随机数/纳秒时间戳。
type DiscoveryScheduler struct {
	repo    Repository
	service *Service
	logger  *zap.SugaredLogger
	now     func() time.Time
	// locationFor 决定 cron 的时区解释。cron 的语义是「租户本地时间 2 点」，
	// 用 UTC 解释会让所有国内租户的 2 点变成 10 点。
	locationFor func(context.Context, int) *time.Location
	// grace 是「从未跑过」的来源的判定窗口，等于调度器的扫描间隔量级。
	grace time.Duration
	// warnCooldown 抑制同一个来源的重复告警（账户停用这类错误会一直成立，
	// 每分钟一条会把日志淹没）。进程内状态，重启后重新计一次，可接受。
	warnCooldown time.Duration
	lastWarn     map[string]time.Time
}

// DiscoveryScheduleResult 是一次租户级扫描的计数，供日志与回归断言使用。
type DiscoveryScheduleResult struct {
	Scanned int // 配置了 schedule 且启用的来源数
	Created int // 本次排上的任务数
	Skipped int // 未到期 / 在途 / 已被其他副本排上
	Failed  int // 创建失败（账户停用、来源失效等），不影响其他来源
}

func NewDiscoveryScheduler(repo Repository, svc *Service, logger *zap.SugaredLogger) *DiscoveryScheduler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &DiscoveryScheduler{
		repo:         repo,
		service:      svc,
		logger:       logger,
		now:          time.Now,
		locationFor:  func(context.Context, int) *time.Location { return service.ResolveLocation("") },
		grace:        2 * time.Minute,
		warnCooldown: 10 * time.Minute,
		lastWarn:     make(map[string]time.Time),
	}
}

// SetLocationResolver 注入租户时区解析（生产实现读 tenants.timezone）。
func (s *DiscoveryScheduler) SetLocationResolver(fn func(context.Context, int) *time.Location) {
	if fn != nil {
		s.locationFor = fn
	}
}

// discoveryRunPlan 是单个来源的到期判定结果。
type discoveryRunPlan struct {
	Due    bool
	FireAt time.Time // 本次应触发的时点（due 时有效）
	Anchor time.Time // 判定基准（上次入队时间，或 now-grace）
	Reason string    // 未到期/跳过的人话原因，写日志用
}

// planDiscoveryRun 是纯函数：给定表达式、上次入队时间与在途状态，判断此刻是否该跑。
//
// 拆成纯函数是为了让「跨天/补跑/在途/非法表达式」这些边界能在毫秒级单测里覆盖，
// 不必为每个时点起一个 sqlite 库。
func planDiscoveryRun(schedule string, lastQueued *time.Time, lastStatus string, now time.Time, loc *time.Location, grace time.Duration) (discoveryRunPlan, error) {
	expr := strings.TrimSpace(schedule)
	if expr == "" {
		return discoveryRunPlan{Reason: "来源未配置调度表达式"}, nil
	}
	if lastStatus != "" && !isTerminalDiscoveryJobStatus(lastStatus) {
		return discoveryRunPlan{Reason: fmt.Sprintf("上一次任务仍在执行（%s）", lastStatus)}, nil
	}

	exprType, err := service.ParseTimerExpression(expr)
	if err != nil {
		return discoveryRunPlan{}, err
	}
	if grace <= 0 {
		grace = time.Minute
	}

	anchor := now.Add(-grace - time.Second)
	if lastQueued != nil && !lastQueued.IsZero() {
		anchor = *lastQueued
	}

	fireAt, err := service.NextFireAt(expr, exprType, loc, anchor)
	if err != nil {
		return discoveryRunPlan{}, err
	}
	if fireAt.IsZero() {
		return discoveryRunPlan{Reason: "调度表达式没有下一次触发时间"}, nil
	}
	if fireAt.After(now) {
		return discoveryRunPlan{Anchor: anchor, FireAt: fireAt, Reason: "未到触发时间"}, nil
	}
	return discoveryRunPlan{Due: true, Anchor: anchor, FireAt: fireAt}, nil
}

// RunTenant 扫描单个租户的到期来源并为其排上发现任务。
//
// 单个来源失败不中断整轮扫描：一个账户停用的来源不该让同租户其他来源停摆。
func (s *DiscoveryScheduler) RunTenant(ctx context.Context, tenantID int, now time.Time) (DiscoveryScheduleResult, error) {
	result := DiscoveryScheduleResult{}
	if s == nil || s.repo == nil || s.service == nil {
		return result, fmt.Errorf("discovery scheduler is not configured")
	}
	if tenantID <= 0 {
		return result, fmt.Errorf("tenant-scoped discovery scheduling requires a tenant id")
	}

	sources, err := s.repo.ListScheduledDiscoverySources(ctx, tenantID)
	if err != nil {
		return result, fmt.Errorf("list scheduled discovery sources: %w", err)
	}
	result.Scanned = len(sources)
	if len(sources) == 0 {
		return result, nil
	}

	loc := s.locationFor(ctx, tenantID)
	for _, source := range sources {
		if source == nil {
			continue
		}
		latest, err := s.repo.LatestDiscoveryJobForSource(ctx, tenantID, source.ID)
		if err != nil && !ent.IsNotFound(err) {
			s.warnOnce(source.ID, "load latest discovery job failed", err)
			result.Failed++
			continue
		}
		var lastQueued *time.Time
		lastStatus := ""
		if latest != nil {
			lastStatus = latest.Status
		}
		// cron 锚点只认自动调度任务。手动任务不应吞掉一个已经到期的触发点。
		scheduled, schedErr := s.repo.LatestScheduledDiscoveryJobForSource(ctx, tenantID, source.ID)
		if schedErr != nil && !ent.IsNotFound(schedErr) {
			s.warnOnce(source.ID, "load latest scheduled discovery job failed", schedErr)
			result.Failed++
			continue
		}
		if scheduled != nil {
			lastQueued = scheduled.QueuedAt
		}

		plan, err := planDiscoveryRun(source.Schedule, lastQueued, lastStatus, now, loc, s.grace)
		if err != nil {
			// 非法表达式是配置错误，不是运行时故障：告警一次，跳过该来源，
			// 其余来源照常调度。把它当 Failed 会让「配置写错」看起来像系统坏了。
			s.warnOnce(source.ID, "invalid discovery schedule expression: "+source.Schedule, err)
			result.Skipped++
			continue
		}
		if !plan.Due {
			result.Skipped++
			continue
		}

		job, err := s.service.CreateDiscoveryJob(ctx, tenantID, 0, &dto.DiscoveryJobRequest{
			SourceID:       source.ID,
			IdempotencyKey: scheduledDiscoveryIdempotencyKey(source.ID, plan.FireAt),
		})
		if err != nil {
			if isConflictError(err) {
				// 另一个副本先排上了同一触发点：这是去重生效，不是故障。
				result.Skipped++
				continue
			}
			s.warnOnce(source.ID, "scheduled discovery job creation failed", err)
			result.Failed++
			continue
		}
		s.clearWarn(source.ID)
		result.Created++
		s.logger.Infow("scheduled discovery job enqueued",
			"tenant_id", tenantID, "source_id", source.ID, "job_id", job.ID,
			"fire_at", plan.FireAt.UTC(), "schedule", source.Schedule)
	}
	return result, nil
}

// scheduledDiscoveryIdempotencyKey 由「来源 + 触发点」派生，跨副本可复现。
func scheduledDiscoveryIdempotencyKey(sourceID string, fireAt time.Time) string {
	return fmt.Sprintf("sched-%s-%d", sourceID, fireAt.UTC().Unix())
}

// schedulePreview 把来源的调度表达式换算成「下次运行时间」与「表达式错误」。
//
// 为什么需要回显：`schedule` 是个自由文本字段，写错了（少一个字段、用了
// 6 段带秒的格式）系统此前既不报错也不提示，运维只能观察到「到了点没跑」。
// 把下一次触发时间和解析错误一起回显，配置错误在保存那一刻就能看见。
func schedulePreview(schedule string, now time.Time, loc *time.Location) (next *time.Time, exprErr string) {
	expr := strings.TrimSpace(schedule)
	if expr == "" {
		return nil, ""
	}
	exprType, err := service.ParseTimerExpression(expr)
	if err != nil {
		return nil, err.Error()
	}
	nextAt, err := service.NextFireAt(expr, exprType, loc, now)
	if err != nil {
		return nil, err.Error()
	}
	return &nextAt, ""
}

func isConflictError(err error) bool {
	var appErr *common.AppError
	if errors.As(err, &appErr) {
		return appErr.Code == common.ErrCodeConflict
	}
	return false
}

// warnOnce 对同一来源的同类告警做冷却，避免「账户停用」这类长期成立的错误
// 每分钟刷一条日志。
func (s *DiscoveryScheduler) warnOnce(sourceID, message string, err error) {
	now := s.now()
	if s.lastWarn == nil {
		s.lastWarn = make(map[string]time.Time)
	}
	if at, ok := s.lastWarn[sourceID]; ok && now.Sub(at) < s.warnCooldown {
		return
	}
	s.lastWarn[sourceID] = now
	s.logger.Warnw(message, "source_id", sourceID, "error", err)
}

func (s *DiscoveryScheduler) clearWarn(sourceID string) {
	if s.lastWarn == nil {
		return
	}
	delete(s.lastWarn, sourceID)
}
