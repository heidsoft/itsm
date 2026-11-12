package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/predicate"
	"itsm-backend/ent/processtimer"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TimerStatus string

const (
	TimerStatusPending   TimerStatus = "pending"
	TimerStatusFired     TimerStatus = "fired"
	TimerStatusCompleted TimerStatus = "completed"
	TimerStatusCancelled TimerStatus = "cancelled"
	TimerStatusFailed    TimerStatus = "failed"
)

type TimerType string

const (
	TimerTypeStart        TimerType = "start"
	TimerTypeIntermediate TimerType = "intermediate"
	TimerTypeBoundary     TimerType = "boundary"
	// TimerTypeTaskDue 任务截止定时器：BPMN userTask 配置 dueDate
	// 属性时注册，到期由 TimerEventHandler 分发 TimeoutScanner 四动作；
	// TimeoutScanner 轮询降级为恢复兜底（claim-once 保护下双路径安全）。
	TimerTypeTaskDue TimerType = "task_due"
)

// ExpressionType 的单一定义已下沉 dto；service 保留同名别名以兼容既有调用点。
type ExpressionType = dto.ExpressionType

const (
	ExprTypeDuration = dto.ExprTypeDuration
	ExprTypeCron     = dto.ExprTypeCron
	ExprTypeDate     = dto.ExprTypeDate
	// ExprTypeCycle ISO 8601 循环表达式（R5/PT10M）或重复 cron 的通用别名。
	ExprTypeCycle = dto.ExprTypeCycle
)

type PauseState string

const (
	PauseStateRunning PauseState = "running"
	PauseStatePaused  PauseState = "paused"
)

type CreateTimerRequest struct {
	TimerType            TimerType
	ProcessDefinitionKey string
	ProcessInstanceID    *int
	ActivityID           string
	TimerExpression      string
	ExpressionType       ExpressionType
	FireAt               time.Time
	ContextVariables     map[string]interface{}
	TotalDurationSeconds *float64
	TenantID             int
}

type TimerStore interface {
	Create(ctx context.Context, req *CreateTimerRequest) (*ent.ProcessTimer, error)
	GetByTimerID(ctx context.Context, timerID string) (*ent.ProcessTimer, error)
	FindPendingDue(ctx context.Context, tenantID int, now time.Time) ([]*ent.ProcessTimer, error)
	FindPendingFuture(ctx context.Context, tenantID int, now time.Time) ([]*ent.ProcessTimer, error)
	FindFiredStale(ctx context.Context, threshold time.Time) ([]*ent.ProcessTimer, error)
	CASFire(ctx context.Context, timerID string, firedAt time.Time) (*ent.ProcessTimer, error)
	CASComplete(ctx context.Context, timerID string) (*ent.ProcessTimer, error)
	CASReclaim(ctx context.Context, timerID string) (*ent.ProcessTimer, error)
	CASFail(ctx context.Context, timerID string, reason string, retryCount int, nextFireAt *time.Time) (*ent.ProcessTimer, error)
	Replay(ctx context.Context, tenantID int, timerID string) (*ent.ProcessTimer, error)
	CancelByProcessInstance(ctx context.Context, tenantID, processInstanceID int) (int, error)
	CancelByTimerID(ctx context.Context, timerID string) error
	List(ctx context.Context, filter TimerListFilter) ([]*ent.ProcessTimer, int, error)
	Stats(ctx context.Context, tenantID int) (*TimerStats, error)
}

// 两个值类型已下沉到 dto 包；别名是同一类型而非新定义，
// 保证 TimerStore.List/Stats 的方法签名与 handlers/timer 域内契约完全一致。
type TimerListFilter = dto.TimerListFilter
type TimerStats = dto.TimerStats

type DBTimerStore struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewDBTimerStore(client *ent.Client, logger *zap.SugaredLogger) *DBTimerStore {
	return &DBTimerStore{client: client, logger: logger}
}

func (s *DBTimerStore) Create(ctx context.Context, req *CreateTimerRequest) (*ent.ProcessTimer, error) {
	timerID := uuid.New().String()
	idempotencyKey := fmt.Sprintf("%s:%d:%d", timerID, req.FireAt.Unix(), req.TenantID)

	create := s.client.ProcessTimer.Create().
		SetTimerID(timerID).
		SetTimerType(string(req.TimerType)).
		SetProcessDefinitionKey(req.ProcessDefinitionKey).
		SetTimerExpression(req.TimerExpression).
		SetExpressionType(string(req.ExpressionType)).
		SetFireAt(req.FireAt).
		SetIdempotencyKey(idempotencyKey).
		SetTenantID(req.TenantID)

	if req.ProcessInstanceID != nil {
		create.SetProcessInstanceID(*req.ProcessInstanceID)
	}
	if req.ActivityID != "" {
		create.SetActivityID(req.ActivityID)
	}
	if req.ContextVariables != nil {
		create.SetContextVariables(req.ContextVariables)
	}
	if req.TotalDurationSeconds != nil {
		create.SetTotalDurationSeconds(*req.TotalDurationSeconds)
	}

	return create.Save(ctx)
}

func (s *DBTimerStore) GetByTimerID(ctx context.Context, timerID string) (*ent.ProcessTimer, error) {
	return s.client.ProcessTimer.Query().
		Where(processtimer.TimerID(timerID)).
		Only(ctx)
}

func (s *DBTimerStore) FindPendingDue(ctx context.Context, tenantID int, now time.Time) ([]*ent.ProcessTimer, error) {
	preds := []predicate.ProcessTimer{
		processtimer.StatusEQ(string(TimerStatusPending)),
		processtimer.FireAtLTE(now),
		processtimer.PauseStateEQ(string(PauseStateRunning)),
	}
	if tenantID > 0 {
		preds = append(preds, processtimer.TenantID(tenantID))
	}
	return s.client.ProcessTimer.Query().
		Where(preds...).
		Order(ent.Asc(processtimer.FieldFireAt)).
		All(ctx)
}

func (s *DBTimerStore) FindPendingFuture(ctx context.Context, tenantID int, now time.Time) ([]*ent.ProcessTimer, error) {
	preds := []predicate.ProcessTimer{
		processtimer.StatusEQ(string(TimerStatusPending)),
		processtimer.FireAtGT(now),
		processtimer.PauseStateEQ(string(PauseStateRunning)),
	}
	if tenantID > 0 {
		preds = append(preds, processtimer.TenantID(tenantID))
	}
	return s.client.ProcessTimer.Query().
		Where(preds...).
		Order(ent.Asc(processtimer.FieldFireAt)).
		All(ctx)
}

func (s *DBTimerStore) FindFiredStale(ctx context.Context, threshold time.Time) ([]*ent.ProcessTimer, error) {
	return s.client.ProcessTimer.Query().
		Where(
			processtimer.StatusEQ(string(TimerStatusFired)),
			processtimer.UpdatedAtLT(threshold),
		).
		All(ctx)
}

// CASComplete 把已触发（in-flight）的 timer 收敛为 completed。
// fired 只是「回调已启动」的中间态：回调成功后必须落 completed，
// 否则崩溃恢复无法区分「执行中」与「已完成」，stale 回收会造成重复触发。
func (s *DBTimerStore) CASComplete(ctx context.Context, timerID string) (*ent.ProcessTimer, error) {
	timer, err := s.GetByTimerID(ctx, timerID)
	if err != nil {
		return nil, fmt.Errorf("timer not found: %w", err)
	}
	if timer.Status != string(TimerStatusFired) {
		return nil, fmt.Errorf("timer %s status is %s, expected fired", timerID, timer.Status)
	}

	affected, err := s.client.ProcessTimer.Update().
		Where(
			processtimer.TimerID(timerID),
			processtimer.StatusEQ(string(TimerStatusFired)),
			processtimer.Version(timer.Version),
		).
		SetStatus(string(TimerStatusCompleted)).
		SetVersion(timer.Version + 1).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("CAS complete update failed: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("CAS complete conflict: timer %s version changed", timerID)
	}

	return s.GetByTimerID(ctx, timerID)
}

// CASReclaim 把滞留在 fired 的孤儿 timer 收回 pending，等待重新触发。
// 只回收 fired：正常完成会落 completed，不会进入回收视野；
// 回收后由调度器重新 CASFire → callback，业务效果按 at-least-once 重驱。
func (s *DBTimerStore) CASReclaim(ctx context.Context, timerID string) (*ent.ProcessTimer, error) {
	timer, err := s.GetByTimerID(ctx, timerID)
	if err != nil {
		return nil, fmt.Errorf("timer not found: %w", err)
	}
	if timer.Status != string(TimerStatusFired) {
		return nil, fmt.Errorf("timer %s status is %s, expected fired", timerID, timer.Status)
	}

	affected, err := s.client.ProcessTimer.Update().
		Where(
			processtimer.TimerID(timerID),
			processtimer.StatusEQ(string(TimerStatusFired)),
			processtimer.Version(timer.Version),
		).
		SetStatus(string(TimerStatusPending)).
		SetVersion(timer.Version + 1).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("CAS reclaim update failed: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("CAS reclaim conflict: timer %s version changed", timerID)
	}

	return s.GetByTimerID(ctx, timerID)
}

// Replay 运维显式重驱一个死信（failed）或滞留（fired）的 timer：
// 重置回 pending、清空重试预算与失败原因、fire_at 拉到当前时刻，
// 由调度器下一个同步周期拾起。pending/completed/cancelled 不可重放。
func (s *DBTimerStore) Replay(ctx context.Context, tenantID int, timerID string) (*ent.ProcessTimer, error) {
	timer, err := s.client.ProcessTimer.Query().
		Where(processtimer.TimerID(timerID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	if timer.TenantID != tenantID {
		return nil, &ent.NotFoundError{}
	}
	if timer.Status != string(TimerStatusFailed) && timer.Status != string(TimerStatusFired) {
		return nil, common.NewBusinessError(common.ConflictCode,
			fmt.Sprintf("定时器 %s 状态为 %s，仅 failed/fired 可重放", timerID, timer.Status), "")
	}

	fireAt := time.Now()
	affected, err := s.client.ProcessTimer.Update().
		Where(
			processtimer.TimerID(timerID),
			processtimer.TenantID(tenantID),
			processtimer.StatusIn(string(TimerStatusFailed), string(TimerStatusFired)),
			processtimer.Version(timer.Version),
		).
		SetStatus(string(TimerStatusPending)).
		SetFireAt(fireAt).
		SetRetryCount(0).
		ClearFailureReason().
		SetVersion(timer.Version + 1).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("replay update failed: %w", err)
	}
	if affected == 0 {
		return nil, common.NewBusinessError(common.ConflictCode,
			fmt.Sprintf("定时器 %s 状态已并发变更，重放失败，请刷新后重试", timerID), "")
	}

	return s.GetByTimerID(ctx, timerID)
}

func (s *DBTimerStore) CASFire(ctx context.Context, timerID string, firedAt time.Time) (*ent.ProcessTimer, error) {
	timer, err := s.GetByTimerID(ctx, timerID)
	if err != nil {
		return nil, fmt.Errorf("timer not found: %w", err)
	}
	if timer.Status != string(TimerStatusPending) {
		return nil, fmt.Errorf("timer %s status is %s, expected pending", timerID, timer.Status)
	}

	affected, err := s.client.ProcessTimer.Update().
		Where(
			processtimer.TimerID(timerID),
			processtimer.StatusEQ(string(TimerStatusPending)),
			processtimer.Version(timer.Version),
		).
		SetStatus(string(TimerStatusFired)).
		SetFiredAt(firedAt).
		SetVersion(timer.Version + 1).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("CAS fire update failed: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("CAS fire conflict: timer %s version changed", timerID)
	}

	return s.GetByTimerID(ctx, timerID)
}

func (s *DBTimerStore) CASFail(ctx context.Context, timerID string, reason string, retryCount int, nextFireAt *time.Time) (*ent.ProcessTimer, error) {
	timer, err := s.GetByTimerID(ctx, timerID)
	if err != nil {
		return nil, fmt.Errorf("timer not found: %w", err)
	}

	updater := s.client.ProcessTimer.Update().
		Where(
			processtimer.TimerID(timerID),
			processtimer.Version(timer.Version),
		).
		SetStatus(string(TimerStatusFailed)).
		SetFailureReason(reason).
		SetRetryCount(retryCount).
		SetLastFireAttempt(time.Now()).
		SetVersion(timer.Version + 1)

	if nextFireAt != nil && retryCount < timer.MaxRetries {
		updater.SetStatus(string(TimerStatusPending)).
			SetFireAt(*nextFireAt)
	}

	affected, err := updater.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("CAS fail update failed: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("CAS fail conflict: timer %s version changed", timerID)
	}

	return s.GetByTimerID(ctx, timerID)
}

func (s *DBTimerStore) CancelByProcessInstance(ctx context.Context, tenantID, processInstanceID int) (int, error) {
	affected, err := s.client.ProcessTimer.Update().
		Where(
			processtimer.ProcessInstanceID(processInstanceID),
			processtimer.TenantID(tenantID),
			processtimer.StatusEQ(string(TimerStatusPending)),
		).
		SetStatus(string(TimerStatusCancelled)).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("cancel by process instance failed: %w", err)
	}
	return affected, nil
}

func (s *DBTimerStore) CancelByTimerID(ctx context.Context, timerID string) error {
	affected, err := s.client.ProcessTimer.Update().
		Where(
			processtimer.TimerID(timerID),
			processtimer.StatusEQ(string(TimerStatusPending)),
		).
		SetStatus(string(TimerStatusCancelled)).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("cancel by timer ID failed: %w", err)
	}
	if affected == 0 {
		s.logger.Warnf("timer %s not cancelled: not found or not pending", timerID)
	}
	return nil
}

func (s *DBTimerStore) List(ctx context.Context, filter TimerListFilter) ([]*ent.ProcessTimer, int, error) {
	preds := []predicate.ProcessTimer{}
	if filter.TenantID > 0 {
		preds = append(preds, processtimer.TenantID(filter.TenantID))
	}
	if filter.Status != "" {
		preds = append(preds, processtimer.StatusEQ(filter.Status))
	}
	if filter.TimerType != "" {
		preds = append(preds, processtimer.TimerTypeEQ(filter.TimerType))
	}
	if filter.ProcessInstanceID != nil {
		preds = append(preds, processtimer.ProcessInstanceIDEQ(*filter.ProcessInstanceID))
	}
	if filter.ProcessDefinitionKey != "" {
		preds = append(preds, processtimer.ProcessDefinitionKeyEQ(filter.ProcessDefinitionKey))
	}

	query := s.client.ProcessTimer.Query().Where(preds...)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count failed: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	items, err := query.
		Order(ent.Desc(processtimer.FieldCreatedAt)).
		Offset(offset).
		Limit(filter.PageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list failed: %w", err)
	}

	return items, total, nil
}

func (s *DBTimerStore) Stats(ctx context.Context, tenantID int) (*TimerStats, error) {
	preds := []predicate.ProcessTimer{}
	if tenantID > 0 {
		preds = append(preds, processtimer.TenantID(tenantID))
	}

	stats := &TimerStats{}

	query := s.client.ProcessTimer.Query().Where(preds...)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats total failed: %w", err)
	}
	stats.Total = total

	pendingCount, err := query.Clone().Where(processtimer.StatusEQ(string(TimerStatusPending))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats pending failed: %w", err)
	}
	stats.Pending = pendingCount

	firedCount, err := query.Clone().Where(processtimer.StatusEQ(string(TimerStatusFired))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats fired failed: %w", err)
	}
	stats.Fired = firedCount

	completedCount, err := query.Clone().Where(processtimer.StatusEQ(string(TimerStatusCompleted))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats completed failed: %w", err)
	}
	stats.Completed = completedCount

	cancelledCount, err := query.Clone().Where(processtimer.StatusEQ(string(TimerStatusCancelled))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats cancelled failed: %w", err)
	}
	stats.Cancelled = cancelledCount

	failedCount, err := query.Clone().Where(processtimer.StatusEQ(string(TimerStatusFailed))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats failed failed: %w", err)
	}
	stats.Failed = failedCount

	pausedCount, err := query.Clone().Where(processtimer.PauseStateEQ(string(PauseStatePaused))).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats paused failed: %w", err)
	}
	stats.Paused = pausedCount

	return stats, nil
}

// CancelPendingStartTimers 取消指定流程定义 Key 下全部 pending 的 start timer。
//
// 用于流程重部署：新定义的时间表整体替换旧时间表。若不取消，每次保存/发布流程都会
// 追加一份 start timer（timer_id 为新 uuid，幂等键不生效），导致定时任务被重复触发、
// 重复启动流程实例。
func CancelPendingStartTimers(ctx context.Context, client *ent.Client, tenantID int, processDefinitionKey string) (int, error) {
	if client == nil || tenantID <= 0 || processDefinitionKey == "" {
		return 0, nil
	}
	affected, err := client.ProcessTimer.Update().
		Where(
			processtimer.TenantID(tenantID),
			processtimer.ProcessDefinitionKey(processDefinitionKey),
			processtimer.TimerTypeEQ(string(TimerTypeStart)),
			processtimer.StatusEQ(string(TimerStatusPending)),
		).
		SetStatus(string(TimerStatusCancelled)).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("cancel pending start timers failed: %w", err)
	}
	return affected, nil
}

// ListActiveStartTimers 列出指定流程定义 Key 下仍处于 pending 的 start timer（管理面展示用）。
func ListActiveStartTimers(ctx context.Context, client *ent.Client, tenantID int, processDefinitionKey string) ([]*ent.ProcessTimer, error) {
	return client.ProcessTimer.Query().
		Where(
			processtimer.TenantID(tenantID),
			processtimer.ProcessDefinitionKey(processDefinitionKey),
			processtimer.TimerTypeEQ(string(TimerTypeStart)),
			processtimer.StatusEQ(string(TimerStatusPending)),
		).
		Order(ent.Asc(processtimer.FieldFireAt)).
		All(ctx)
}
