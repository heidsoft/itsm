package service

import (
	"context"
	"fmt"
	"time"

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
	TimerStatusCancelled TimerStatus = "cancelled"
	TimerStatusFailed    TimerStatus = "failed"
)

type TimerType string

const (
	TimerTypeStart         TimerType = "start"
	TimerTypeIntermediate  TimerType = "intermediate"
	TimerTypeBoundary      TimerType = "boundary"
	// TimerTypeTaskDue 任务截止定时器（Phase 4）：BPMN userTask 配置 dueDate
	// 属性时注册，到期由 TimerEventHandler 分发 TimeoutScanner 四动作；
	// TimeoutScanner 轮询降级为恢复兜底（claim-once 保护下双路径安全）。
	TimerTypeTaskDue TimerType = "task_due"
)

type ExpressionType string

const (
	ExprTypeDuration ExpressionType = "duration"
	ExprTypeCron     ExpressionType = "cron"
	ExprTypeDate     ExpressionType = "date"
	// ExprTypeCycle ISO 8601 循环表达式（R5/PT10M）或重复 cron 的通用别名。
	ExprTypeCycle ExpressionType = "cycle"
)

type PauseState string

const (
	PauseStateRunning PauseState = "running"
	PauseStatePaused  PauseState = "paused"
)

type CreateTimerRequest struct {
	TimerType          TimerType
	ProcessDefinitionKey string
	ProcessInstanceID  *int
	ActivityID         string
	TimerExpression    string
	ExpressionType     ExpressionType
	FireAt             time.Time
	ContextVariables   map[string]interface{}
	TotalDurationSeconds *float64
	TenantID           int
}

type TimerStore interface {
	Create(ctx context.Context, req *CreateTimerRequest) (*ent.ProcessTimer, error)
	GetByTimerID(ctx context.Context, timerID string) (*ent.ProcessTimer, error)
	FindPendingDue(ctx context.Context, tenantID int, now time.Time) ([]*ent.ProcessTimer, error)
	FindPendingFuture(ctx context.Context, tenantID int, now time.Time) ([]*ent.ProcessTimer, error)
	FindFiredStale(ctx context.Context, threshold time.Time) ([]*ent.ProcessTimer, error)
	FindFailedRetryable(ctx context.Context, tenantID int) ([]*ent.ProcessTimer, error)
	CASFire(ctx context.Context, timerID string, firedAt time.Time) (*ent.ProcessTimer, error)
	CASFail(ctx context.Context, timerID string, reason string, retryCount int, nextFireAt *time.Time) (*ent.ProcessTimer, error)
	CancelByProcessInstance(ctx context.Context, tenantID, processInstanceID int) (int, error)
	CancelByTimerID(ctx context.Context, timerID string) error
	List(ctx context.Context, filter TimerListFilter) ([]*ent.ProcessTimer, int, error)
	Stats(ctx context.Context, tenantID int) (*TimerStats, error)
}

type TimerListFilter struct {
	TenantID           int
	Status             string
	TimerType          string
	ProcessInstanceID  *int
	ProcessDefinitionKey string
	Page               int
	PageSize           int
}

type TimerStats struct {
	Pending   int `json:"pending"`
	Fired     int `json:"fired"`
	Cancelled int `json:"cancelled"`
	Failed    int `json:"failed"`
	Paused    int `json:"paused"`
	Total     int `json:"total"`
}

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

func (s *DBTimerStore) FindFailedRetryable(ctx context.Context, tenantID int) ([]*ent.ProcessTimer, error) {
	preds := []predicate.ProcessTimer{
		processtimer.StatusEQ(string(TimerStatusFailed)),
	}
	if tenantID > 0 {
		preds = append(preds, processtimer.TenantID(tenantID))
	}
	return s.client.ProcessTimer.Query().
		Where(preds...).
		All(ctx)
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

// CancelPendingStartTimers 取消指定流程定义 Key 下全部 pending 的 start timer（Phase 5）。
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
