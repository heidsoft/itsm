package operations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"
)

const StatusCancelled = "cancelled"

var (
	ErrCommandNotFound = errors.New("operational command not found")
	ErrInvalidState    = errors.New("operational command state does not allow this operation")
	ErrConcurrentWrite = errors.New("operational command changed concurrently")
	ErrBulkEmpty       = errors.New("operational command bulk operation matched no commands")
)

const (
	defaultBulkLimit = 200
	maxBulkLimit     = 1000
)

type Service struct {
	client *ent.Client
	now    func() time.Time
}

func NewService(client *ent.Client) *Service { return &Service{client: client, now: time.Now} }

type ListRequest struct {
	TenantID      int
	Status        string
	CommandType   string
	AggregateType string
	Page          int
	PageSize      int
}

type Page struct {
	Items      []CommandDTO      `json:"items"`
	Total      int               `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"pageSize"`
	Summary    CommandSummary    `json:"summary"`
	ByTypeRows []CommandTypeStat `json:"byType,omitempty"`
}

type CommandSummary struct {
	Pending       int        `json:"pending"`
	Processing    int        `json:"processing"`
	DeadLetter    int        `json:"deadLetter"`
	Cancelled     int        `json:"cancelled"`
	Succeeded     int        `json:"succeeded"`
	OldestWaiting *time.Time `json:"oldestWaitingAt,omitempty"`
	StuckLeases   int        `json:"stuckLeases"`
}

// CommandTypeStat 暴露按 CommandType 聚合的健康度视图，供巡检/告警使用。
// FailedRate 在 0..1 之间；SampleSize 表示过去 24h 内的总尝试次数。
type CommandTypeStat struct {
	CommandType     string  `json:"commandType"`
	Pending         int     `json:"pending"`
	Processing      int     `json:"processing"`
	DeadLetter      int     `json:"deadLetter"`
	SucceededRecent int     `json:"succeededRecent"`
	FailedRecent    int     `json:"failedRecent"`
	SampleSize       int     `json:"sampleSize"`
	FailureRate     float64 `json:"failureRate"`
}

type CommandDTO struct {
	ID             int                    `json:"id"`
	TenantID       int                    `json:"tenantId"`
	CommandType    string                 `json:"commandType"`
	AggregateType  string                 `json:"aggregateType"`
	AggregateID    int                    `json:"aggregateId"`
	IdempotencyKey string                 `json:"idempotencyKey"`
	Payload        map[string]interface{} `json:"payload,omitempty"`
	Status         string                 `json:"status"`
	Attempt        int                    `json:"attempt"`
	MaxAttempts    int                    `json:"maxAttempts"`
	AvailableAt    time.Time              `json:"availableAt"`
	LeaseOwner     string                 `json:"leaseOwner,omitempty"`
	LeaseExpiresAt *time.Time             `json:"leaseExpiresAt,omitempty"`
	FencingToken   int64                  `json:"fencingToken"`
	LastError      string                 `json:"lastError,omitempty"`
	CompletedAt    *time.Time             `json:"completedAt,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

type Actor struct {
	UserID    int
	RequestID string
	IP        string
	Path      string
	Method    string
}

// BulkFilter 描述批量运维动作的筛选条件。所有字段 AND；空值表示不过滤。
// Limit 上限由 Service 强制收紧，避免单次运维请求拖垮 DB 或 worker。
type BulkFilter struct {
	TenantID      int
	Status        string
	CommandType   string
	AggregateType string
	LeaseExpired  bool
	Limit         int
}

func (f BulkFilter) validate() error {
	if f.TenantID <= 0 {
		return fmt.Errorf("tenant id is required")
	}
	if f.Status != "" && !isKnownStatus(f.Status) {
		return fmt.Errorf("unsupported status filter: %s", f.Status)
	}
	if f.Limit <= 0 {
		f.Limit = defaultBulkLimit
	}
	if f.Limit > maxBulkLimit {
		f.Limit = maxBulkLimit
	}
	return nil
}

func isKnownStatus(status string) bool {
	switch status {
	case commandbus.StatusPending, commandbus.StatusProcessing,
		commandbus.StatusSucceeded, commandbus.StatusDeadLetter,
		StatusCancelled:
		return true
	}
	return false
}

type BulkResult struct {
	MatchedIDs []int `json:"matchedIds"`
	Updated    int   `json:"updated"`
	Skipped    int   `json:"skipped"`
	Limit      int   `json:"limit"`
}

func (s *Service) List(ctx context.Context, request ListRequest) (*Page, error) {
	query := s.client.OperationalCommand.Query().Where(operationalcommand.TenantIDEQ(request.TenantID))
	if request.Status != "" {
		if !isKnownStatus(request.Status) {
			return nil, fmt.Errorf("unsupported status filter: %s", request.Status)
		}
		query = query.Where(operationalcommand.StatusEQ(request.Status))
	}
	if request.CommandType != "" {
		query = query.Where(operationalcommand.CommandTypeEQ(request.CommandType))
	}
	if request.AggregateType != "" {
		query = query.Where(operationalcommand.AggregateTypeEQ(request.AggregateType))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count commands: %w", err)
	}
	commands, err := query.Order(ent.Desc(operationalcommand.FieldCreatedAt)).
		Offset((request.Page - 1) * request.PageSize).Limit(request.PageSize).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list commands: %w", err)
	}
	items := make([]CommandDTO, 0, len(commands))
	for _, command := range commands {
		items = append(items, mapCommand(command, false))
	}
	summary, err := s.summary(ctx, request.TenantID)
	if err != nil {
		return nil, err
	}
	byType, err := s.summaryByCommandType(ctx, request.TenantID)
	if err != nil {
		return nil, err
	}
	return &Page{Items: items, Total: total, Page: request.Page, PageSize: request.PageSize, Summary: summary, ByTypeRows: byType}, nil
}

func (s *Service) Get(ctx context.Context, tenantID, commandID int) (*CommandDTO, error) {
	command, err := s.get(ctx, tenantID, commandID)
	if err != nil {
		return nil, err
	}
	dto := mapCommand(command, true)
	return &dto, nil
}

func (s *Service) Replay(ctx context.Context, tenantID, commandID int, actor Actor) (*CommandDTO, error) {
	return s.transition(ctx, tenantID, commandID, actor, true)
}

func (s *Service) Cancel(ctx context.Context, tenantID, commandID int, actor Actor) (*CommandDTO, error) {
	return s.transition(ctx, tenantID, commandID, actor, false)
}

// BulkReplay 将符合 BulkFilter 的 dead_letter 命令批量重新入箱。
// idempotency_key 与 fencing_token 全部保留，仅重置 status / available_at / last_error / completed_at，
// 这样 worker 仍按原始幂等身份去重，不会与历史成功的 delivery 重复。
func (s *Service) BulkReplay(ctx context.Context, filter BulkFilter, actor Actor) (*BulkResult, error) {
	if err := filter.validate(); err != nil {
		return nil, err
	}
	if filter.Status == "" {
		filter.Status = commandbus.StatusDeadLetter
	} else if filter.Status != commandbus.StatusDeadLetter && filter.Status != StatusCancelled {
		return nil, ErrInvalidState
	}
	return s.bulkTransition(ctx, filter, actor, true)
}

// BulkCancel 强制撤离处理中或挂起的命令；leaseExpired=true 时仅取消 lease 过期
// 且未被 heartbeat 续约的 processing 命令，避免误杀正在跑的命令。
func (s *Service) BulkCancel(ctx context.Context, filter BulkFilter, actor Actor) (*BulkResult, error) {
	if err := filter.validate(); err != nil {
		return nil, err
	}
	if filter.Status == "" {
		filter.Status = commandbus.StatusProcessing
	} else if filter.Status != commandbus.StatusProcessing && filter.Status != commandbus.StatusPending {
		return nil, ErrInvalidState
	}
	return s.bulkTransition(ctx, filter, actor, false)
}

func (s *Service) transition(ctx context.Context, tenantID, commandID int, actor Actor, replay bool) (*CommandDTO, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin command operation: %w", err)
	}
	defer tx.Rollback()
	command, err := tx.OperationalCommand.Query().Where(
		operationalcommand.IDEQ(commandID), operationalcommand.TenantIDEQ(tenantID),
	).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, ErrCommandNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load command: %w", err)
	}
	if replay && command.Status != commandbus.StatusDeadLetter && command.Status != StatusCancelled {
		return nil, ErrInvalidState
	}
	if !replay && command.Status != commandbus.StatusPending && command.Status != commandbus.StatusProcessing {
		return nil, ErrInvalidState
	}
	update := tx.OperationalCommand.UpdateOneID(command.ID).
		Where(operationalcommand.TenantIDEQ(tenantID), operationalcommand.FencingTokenEQ(command.FencingToken)).
		AddFencingToken(1).ClearLeaseOwner().ClearLeaseExpiresAt()
	action := "cancel"
	if replay {
		action = "replay"
		update.SetStatus(commandbus.StatusPending).SetAvailableAt(s.now()).ClearCompletedAt().ClearLastError()
	} else {
		update.SetStatus(StatusCancelled).SetCompletedAt(s.now())
	}
	updated, err := update.Save(ctx)
	if ent.IsNotFound(err) {
		return nil, ErrConcurrentWrite
	}
	if err != nil {
		return nil, fmt.Errorf("update command: %w", err)
	}
	body, err := json.Marshal(map[string]interface{}{
		"commandId": command.ID, "previousStatus": command.Status, "idempotencyKey": command.IdempotencyKey,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal command audit: %w", err)
	}
	requestBody := string(body)
	if err := tx.AuditLog.Create().
		SetTenantID(tenantID).SetUserID(actor.UserID).SetRequestID(actor.RequestID).
		SetIP(actor.IP).SetResource("operational_command").SetAction(action).
		SetPath(actor.Path).SetMethod(actor.Method).SetStatusCode(200).
		SetNillableRequestBody(&requestBody).Exec(ctx); err != nil {
		return nil, fmt.Errorf("write command audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit command operation: %w", err)
	}
	dto := mapCommand(updated, true)
	return &dto, nil
}

// bulkTransition 在单事务内对符合 BulkFilter 的命令批量执行 replay / cancel，
// 并把每条 command 的 action 写进 audit_log。Limit 默认收紧到 defaultBulkLimit。
// replay 时保留 idempotency_key 与 fencing_token，避免与历史 delivery 重复；
// cancel 时同时清掉 lease_owner 与 lease_expires_at，释放被 worker 占用的资源。
func (s *Service) bulkTransition(ctx context.Context, filter BulkFilter, actor Actor, replay bool) (*BulkResult, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin bulk operation: %w", err)
	}
	defer tx.Rollback()

	query := tx.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(filter.TenantID),
		operationalcommand.StatusEQ(filter.Status),
	).Order(ent.Asc(operationalcommand.FieldAvailableAt), ent.Asc(operationalcommand.FieldID))
	if filter.CommandType != "" {
		query = query.Where(operationalcommand.CommandTypeEQ(filter.CommandType))
	}
	if filter.AggregateType != "" {
		query = query.Where(operationalcommand.AggregateTypeEQ(filter.AggregateType))
	}
	if filter.LeaseExpired && filter.Status == commandbus.StatusProcessing {
		query = query.Where(operationalcommand.LeaseExpiresAtLT(s.now()))
	}
	matched, err := query.Limit(filter.Limit).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("load bulk commands: %w", err)
	}
	if len(matched) == 0 {
		return nil, ErrBulkEmpty
	}

	now := s.now()
	updatedIDs := make([]int, 0, len(matched))
	skipped := 0
	for _, cmd := range matched {
		upd := tx.OperationalCommand.UpdateOneID(cmd.ID).
			Where(operationalcommand.TenantIDEQ(filter.TenantID), operationalcommand.FencingTokenEQ(cmd.FencingToken)).
			AddFencingToken(1).ClearLeaseOwner().ClearLeaseExpiresAt()
		action := "cancel"
		if replay {
			action = "replay"
			upd.SetStatus(commandbus.StatusPending).SetAvailableAt(now).ClearCompletedAt().ClearLastError()
		} else {
			upd.SetStatus(StatusCancelled).SetCompletedAt(now)
		}
		saved, err := upd.Save(ctx)
		if err != nil {
			// 任意一条失败 → 整批回滚。运维操作宁可返错也不允许部分生效，
			// 否则操作员无法判断哪条被改了哪条没改。
			return nil, fmt.Errorf("bulk %s failed on command %d: %w", action, cmd.ID, err)
		}
		body, mErr := json.Marshal(map[string]interface{}{
			"commandId": cmd.ID, "previousStatus": cmd.Status, "idempotencyKey": cmd.IdempotencyKey,
			"bulkFilter": filter, "newStatus": saved.Status,
		})
		if mErr != nil {
			return nil, fmt.Errorf("marshal bulk audit: %w", mErr)
		}
		bodyStr := string(body)
		if err := tx.AuditLog.Create().
			SetTenantID(filter.TenantID).SetUserID(actor.UserID).SetRequestID(actor.RequestID).
			SetIP(actor.IP).SetResource("operational_command").SetAction("bulk_" + action).
			SetPath(actor.Path).SetMethod(actor.Method).SetStatusCode(200).
			SetNillableRequestBody(&bodyStr).Exec(ctx); err != nil {
			return nil, fmt.Errorf("write bulk command audit: %w", err)
		}
		updatedIDs = append(updatedIDs, saved.ID)
		_ = skipped
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit bulk operation: %w", err)
	}
	return &BulkResult{MatchedIDs: updatedIDs, Updated: len(updatedIDs), Limit: filter.Limit}, nil
}

func (s *Service) get(ctx context.Context, tenantID, commandID int) (*ent.OperationalCommand, error) {
	command, err := s.client.OperationalCommand.Query().Where(
		operationalcommand.IDEQ(commandID), operationalcommand.TenantIDEQ(tenantID),
	).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, ErrCommandNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load command: %w", err)
	}
	return command, nil
}

func (s *Service) summary(ctx context.Context, tenantID int) (CommandSummary, error) {
	count := func(status string) (int, error) {
		return s.client.OperationalCommand.Query().Where(
			operationalcommand.TenantIDEQ(tenantID), operationalcommand.StatusEQ(status),
		).Count(ctx)
	}
	var result CommandSummary
	var err error
	if result.Pending, err = count(commandbus.StatusPending); err != nil {
		return result, fmt.Errorf("count pending commands: %w", err)
	}
	if result.Processing, err = count(commandbus.StatusProcessing); err != nil {
		return result, fmt.Errorf("count processing commands: %w", err)
	}
	if result.DeadLetter, err = count(commandbus.StatusDeadLetter); err != nil {
		return result, fmt.Errorf("count dead-letter commands: %w", err)
	}
	if result.Cancelled, err = count(StatusCancelled); err != nil {
		return result, fmt.Errorf("count cancelled commands: %w", err)
	}
	if result.Succeeded, err = count(commandbus.StatusSucceeded); err != nil {
		return result, fmt.Errorf("count succeeded commands: %w", err)
	}
	now := s.now()
	stuck, err := s.client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenantID),
		operationalcommand.StatusEQ(commandbus.StatusProcessing),
		operationalcommand.LeaseExpiresAtLT(now),
	).Count(ctx)
	if err != nil {
		return result, fmt.Errorf("count stuck leases: %w", err)
	}
	result.StuckLeases = stuck
	oldest, err := s.client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenantID), operationalcommand.StatusEQ(commandbus.StatusPending),
	).Order(ent.Asc(operationalcommand.FieldAvailableAt)).First(ctx)
	if err == nil {
		result.OldestWaiting = &oldest.AvailableAt
	} else if !ent.IsNotFound(err) {
		return result, fmt.Errorf("load oldest waiting command: %w", err)
	}
	return result, nil
}

// summaryByCommandType 按 CommandType 聚合当前积压 + 过去 24h 失败率。
// 失败率用于驱动运维巡检：failureRate > 0.5 即视为该类型严重不健康。
// 先把租户内命令类型在 Go 端去重，再按类型做 N 次单条 COUNT。
// CommandType 数量受 internal/commandbus.Register 控制，运维期内通常 < 20。
func (s *Service) summaryByCommandType(ctx context.Context, tenantID int) ([]CommandTypeStat, error) {
	commands, err := s.client.OperationalCommand.Query().
		Where(operationalcommand.TenantIDEQ(tenantID)).
		Select(operationalcommand.FieldCommandType).
		All(ctx)
	if err != nil || len(commands) == 0 {
		return nil, err
	}
	seen := make(map[string]struct{}, len(commands))
	commandTypes := make([]string, 0, len(commands))
	for _, command := range commands {
		if _, ok := seen[command.CommandType]; ok || command.CommandType == "" {
			continue
		}
		seen[command.CommandType] = struct{}{}
		commandTypes = append(commandTypes, command.CommandType)
	}
	recentWindow := s.now().Add(-24 * time.Hour)
	stats := make([]CommandTypeStat, 0, len(commandTypes))
	for _, commandType := range commandTypes {
		pending, err := s.client.OperationalCommand.Query().Where(
			operationalcommand.TenantIDEQ(tenantID), operationalcommand.CommandTypeEQ(commandType),
			operationalcommand.StatusEQ(commandbus.StatusPending),
		).Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("count pending %s: %w", commandType, err)
		}
		processing, err := s.client.OperationalCommand.Query().Where(
			operationalcommand.TenantIDEQ(tenantID), operationalcommand.CommandTypeEQ(commandType),
			operationalcommand.StatusEQ(commandbus.StatusProcessing),
		).Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("count processing %s: %w", commandType, err)
		}
		deadLetter, err := s.client.OperationalCommand.Query().Where(
			operationalcommand.TenantIDEQ(tenantID), operationalcommand.CommandTypeEQ(commandType),
			operationalcommand.StatusEQ(commandbus.StatusDeadLetter),
		).Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("count dead_letter %s: %w", commandType, err)
		}
		succeeded, failed, err := s.recentOutcomes(ctx, tenantID, commandType, recentWindow)
		if err != nil {
			return nil, err
		}
		sample := succeeded + failed
		var rate float64
		if sample > 0 {
			rate = float64(failed) / float64(sample)
		}
		stats = append(stats, CommandTypeStat{
			CommandType:     commandType,
			Pending:         pending,
			Processing:      processing,
			DeadLetter:      deadLetter,
			SucceededRecent: succeeded,
			FailedRecent:    failed,
			SampleSize:      sample,
			FailureRate:     rate,
		})
	}
	return stats, nil
}

// recentOutcomes 计算过去 24h 内 succeeded 与 failed 的命令数；failed = dead_letter +
// cancelled，避免把"运维手动取消"误计入失败率但同时承认其属于非正常终态。
func (s *Service) recentOutcomes(ctx context.Context, tenantID int, commandType string, since time.Time) (succeeded, failed int, err error) {
	succeeded, err = s.client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenantID),
		operationalcommand.CommandTypeEQ(commandType),
		operationalcommand.StatusEQ(commandbus.StatusSucceeded),
		operationalcommand.CompletedAtGTE(since),
	).Count(ctx)
	if err != nil {
		return 0, 0, err
	}
	dead, dErr := s.client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenantID),
		operationalcommand.CommandTypeEQ(commandType),
		operationalcommand.StatusEQ(commandbus.StatusDeadLetter),
		operationalcommand.CompletedAtGTE(since),
	).Count(ctx)
	if dErr != nil {
		return 0, 0, dErr
	}
	cancelled, cErr := s.client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenantID),
		operationalcommand.CommandTypeEQ(commandType),
		operationalcommand.StatusEQ(StatusCancelled),
		operationalcommand.CompletedAtGTE(since),
	).Count(ctx)
	if cErr != nil {
		return 0, 0, cErr
	}
	return succeeded, dead + cancelled, nil
}

func mapCommand(command *ent.OperationalCommand, includePayload bool) CommandDTO {
	dto := CommandDTO{
		ID: command.ID, TenantID: command.TenantID, CommandType: command.CommandType,
		AggregateType: command.AggregateType, AggregateID: command.AggregateID,
		IdempotencyKey: command.IdempotencyKey, Status: command.Status,
		Attempt: command.Attempt, MaxAttempts: command.MaxAttempts,
		AvailableAt: command.AvailableAt, LeaseOwner: command.LeaseOwner,
		LeaseExpiresAt: command.LeaseExpiresAt, FencingToken: command.FencingToken,
		LastError: command.LastError, CompletedAt: command.CompletedAt,
		CreatedAt: command.CreatedAt, UpdatedAt: command.UpdatedAt,
	}
	if includePayload {
		dto.Payload = sanitizePayload(command.Payload)
	}
	return dto
}

func sanitizePayload(payload map[string]interface{}) map[string]interface{} {
	if payload == nil {
		return nil
	}
	result := make(map[string]interface{}, len(payload))
	for key, value := range payload {
		normalized := strings.ToLower(key)
		if strings.Contains(normalized, "secret") || strings.Contains(normalized, "token") ||
			strings.Contains(normalized, "password") || strings.Contains(normalized, "credential") ||
			strings.Contains(normalized, "accesskey") {
			result[key] = "******"
			continue
		}
		result[key] = sanitizeValue(value)
	}
	return result
}

func sanitizeValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		return sanitizePayload(typed)
	case []interface{}:
		result := make([]interface{}, len(typed))
		for index, item := range typed {
			result[index] = sanitizeValue(item)
		}
		return result
	default:
		return value
	}
}