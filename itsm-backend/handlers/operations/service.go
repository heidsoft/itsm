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
)

type Service struct {
	client *ent.Client
	now    func() time.Time
}

func NewService(client *ent.Client) *Service { return &Service{client: client, now: time.Now} }

type ListRequest struct {
	TenantID int
	Status   string
	Page     int
	PageSize int
}

type Page struct {
	Items    []CommandDTO   `json:"items"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
	Summary  CommandSummary `json:"summary"`
}

type CommandSummary struct {
	Pending       int        `json:"pending"`
	Processing    int        `json:"processing"`
	DeadLetter    int        `json:"deadLetter"`
	Cancelled     int        `json:"cancelled"`
	OldestWaiting *time.Time `json:"oldestWaitingAt,omitempty"`
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

func (s *Service) List(ctx context.Context, request ListRequest) (*Page, error) {
	query := s.client.OperationalCommand.Query().Where(operationalcommand.TenantIDEQ(request.TenantID))
	if request.Status != "" {
		query = query.Where(operationalcommand.StatusEQ(request.Status))
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
	return &Page{Items: items, Total: total, Page: request.Page, PageSize: request.PageSize, Summary: summary}, nil
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
