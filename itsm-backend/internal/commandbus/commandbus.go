package commandbus

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"itsm-backend/common/tenantctx"
	"itsm-backend/ent"
	"itsm-backend/ent/operationalcommand"

	"go.uber.org/zap"
)

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusSucceeded  = "succeeded"
	StatusDeadLetter = "dead_letter"

	CommandStartBPMN              = "workflow.start"
	CommandExecuteBPMNServiceTask = "workflow.service_task.execute"
	CommandDeliverNotification    = "notification.deliver"
	CommandProcessCMDBImport      = "cmdb.import.process"
	CommandProcessCMDBExport      = "cmdb.export.process"
	CommandRunCMDBCloudDiscovery  = "cmdb.cloud_discovery.run"
	CommandExecuteTicketRules     = "ticket.rules.execute"
	CommandSyncTicketFeishu       = "ticket.feishu.sync"
	CommandExecuteIncidentRules   = "incident.rules.execute"
	CommandSendIntakeEmail        = "email_intake.email.send"
	CommandProcessIntakeEmail     = "email_intake.message.process"
)

var ErrLeaseLost = errors.New("operational command lease lost")

type Handler func(context.Context, *ent.OperationalCommand) error

type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

func NewRegistry() *Registry { return &Registry{handlers: make(map[string]Handler)} }

func ValidateStorage(ctx context.Context, client *ent.Client) error {
	if client == nil {
		return fmt.Errorf("operational command storage client is required")
	}
	systemCtx := tenantctx.SystemContext(ctx, "commandbus:readiness", "verify durable command storage before serving traffic")
	if _, err := client.OperationalCommand.Query().Limit(1).All(systemCtx); err != nil {
		return fmt.Errorf("operational command storage is not ready: %w", err)
	}
	if _, err := client.NotificationDelivery.Query().Limit(1).All(systemCtx); err != nil {
		return fmt.Errorf("notification delivery audit storage is not ready: %w", err)
	}
	return nil
}

func (r *Registry) Register(commandType string, handler Handler) error {
	if commandType == "" || handler == nil {
		return fmt.Errorf("command type and handler are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.handlers[commandType]; exists {
		return fmt.Errorf("handler already registered for %s", commandType)
	}
	r.handlers[commandType] = handler
	return nil
}

func (r *Registry) Get(commandType string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[commandType]
	return h, ok
}

type EnqueueRequest struct {
	TenantID       int
	CommandType    string
	AggregateType  string
	AggregateID    int
	IdempotencyKey string
	Payload        map[string]interface{}
	MaxAttempts    int
}

func Enqueue(ctx context.Context, client *ent.Client, req EnqueueRequest) (*ent.OperationalCommand, error) {
	return enqueue(ctx, client.OperationalCommand.Create(), req)
}

func EnqueueTx(ctx context.Context, tx *ent.Tx, req EnqueueRequest) (*ent.OperationalCommand, error) {
	return enqueue(ctx, tx.OperationalCommand.Create(), req)
}

// EnqueueSQLTx lets domain repositories that already own a database/sql
// transaction participate in the same durable outbox without opening a second
// Ent transaction. The business write and command therefore commit or roll
// back together.
func EnqueueSQLTx(ctx context.Context, tx *sql.Tx, req EnqueueRequest) error {
	if tx == nil {
		return fmt.Errorf("operational command SQL transaction is required")
	}
	if req.TenantID <= 0 || req.AggregateID <= 0 || req.CommandType == "" || req.AggregateType == "" || req.IdempotencyKey == "" {
		return fmt.Errorf("invalid operational command")
	}
	maxAttempts := req.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 8
	}
	payload, err := json.Marshal(req.Payload)
	if err != nil {
		return fmt.Errorf("marshal operational command payload: %w", err)
	}
	now := time.Now()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO operational_commands
			(tenant_id, command_type, aggregate_type, aggregate_id, idempotency_key, payload,
			 status, attempt, max_attempts, available_at, fencing_token, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending', 0, $7, $8, 0, $8, $8)
	`, req.TenantID, req.CommandType, req.AggregateType, req.AggregateID, req.IdempotencyKey, string(payload), maxAttempts, now)
	if err != nil {
		return fmt.Errorf("insert operational command: %w", err)
	}
	return nil
}

func enqueue(ctx context.Context, create *ent.OperationalCommandCreate, req EnqueueRequest) (*ent.OperationalCommand, error) {
	if req.TenantID <= 0 || req.AggregateID <= 0 || req.CommandType == "" || req.AggregateType == "" || req.IdempotencyKey == "" {
		return nil, fmt.Errorf("invalid operational command")
	}
	maxAttempts := req.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 8
	}
	return create.
		SetTenantID(req.TenantID).
		SetCommandType(req.CommandType).
		SetAggregateType(req.AggregateType).
		SetAggregateID(req.AggregateID).
		SetIdempotencyKey(req.IdempotencyKey).
		SetPayload(req.Payload).
		SetMaxAttempts(maxAttempts).
		Save(ctx)
}

type Worker struct {
	client       *ent.Client
	registry     *Registry
	logger       *zap.SugaredLogger
	owner        string
	leaseTTL     time.Duration
	pollInterval time.Duration
	now          func() time.Time
}

func NewWorker(client *ent.Client, registry *Registry, logger *zap.SugaredLogger, owner string) *Worker {
	return &Worker{
		client: client, registry: registry, logger: logger, owner: owner,
		leaseTTL: time.Minute, pollInterval: time.Second, now: time.Now,
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		processed, err := w.RunOnce(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			w.logger.Errorw("operational command worker iteration failed", "error", err, "owner", w.owner)
		}
		if processed {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) RunOnce(ctx context.Context) (bool, error) {
	now := w.now()
	systemCtx := tenantctx.SystemContext(ctx, "commandbus:claim", "claim one due command across tenants")
	candidate, err := w.client.OperationalCommand.Query().
		Where(
			operationalcommand.AvailableAtLTE(now),
			operationalcommand.Or(
				operationalcommand.StatusEQ(StatusPending),
				operationalcommand.And(
					operationalcommand.StatusEQ(StatusProcessing),
					operationalcommand.LeaseExpiresAtLT(now),
				),
			),
		).
		Order(ent.Asc(operationalcommand.FieldAvailableAt), ent.Asc(operationalcommand.FieldID)).
		First(systemCtx)
	if ent.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	claimed, err := w.client.OperationalCommand.UpdateOneID(candidate.ID).
		Where(
			operationalcommand.FencingTokenEQ(candidate.FencingToken),
			operationalcommand.Or(
				operationalcommand.StatusEQ(StatusPending),
				operationalcommand.And(operationalcommand.StatusEQ(StatusProcessing), operationalcommand.LeaseExpiresAtLT(now)),
			),
		).
		SetStatus(StatusProcessing).
		SetLeaseOwner(w.owner).
		SetLeaseExpiresAt(now.Add(w.leaseTTL)).
		AddFencingToken(1).
		AddAttempt(1).
		Save(systemCtx)
	if ent.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	handler, ok := w.registry.Get(claimed.CommandType)
	if !ok {
		return true, w.fail(systemCtx, claimed, fmt.Errorf("no handler registered for %s", claimed.CommandType))
	}
	handlerCtx, cancelHandler := context.WithCancel(tenantctx.WithTenantID(ctx, claimed.TenantID))
	heartbeatDone := make(chan error, 1)
	go w.heartbeat(handlerCtx, claimed, cancelHandler, heartbeatDone)
	handlerErr := handler(handlerCtx, claimed)
	cancelHandler()
	heartbeatErr := <-heartbeatDone
	if heartbeatErr != nil {
		return true, heartbeatErr
	}
	if handlerErr != nil {
		return true, w.fail(systemCtx, claimed, handlerErr)
	}
	_, err = w.client.OperationalCommand.UpdateOneID(claimed.ID).
		Where(operationalcommand.StatusEQ(StatusProcessing), operationalcommand.LeaseOwnerEQ(w.owner), operationalcommand.FencingTokenEQ(claimed.FencingToken)).
		SetStatus(StatusSucceeded).
		SetCompletedAt(w.now()).
		ClearLeaseOwner().
		ClearLeaseExpiresAt().
		ClearLastError().
		Save(systemCtx)
	if ent.IsNotFound(err) {
		return true, ErrLeaseLost
	}
	return true, err
}

func (w *Worker) heartbeat(ctx context.Context, cmd *ent.OperationalCommand, cancel context.CancelFunc, done chan<- error) {
	interval := w.leaseTTL / 3
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			done <- nil
			return
		case <-ticker.C:
			systemCtx := tenantctx.SystemContext(ctx, "commandbus:heartbeat", "extend an owned operational command lease")
			_, err := w.client.OperationalCommand.UpdateOneID(cmd.ID).
				Where(operationalcommand.StatusEQ(StatusProcessing), operationalcommand.LeaseOwnerEQ(w.owner), operationalcommand.FencingTokenEQ(cmd.FencingToken)).
				SetLeaseExpiresAt(w.now().Add(w.leaseTTL)).Save(systemCtx)
			if ent.IsNotFound(err) {
				cancel()
				done <- ErrLeaseLost
				return
			}
			if err != nil {
				cancel()
				done <- err
				return
			}
		}
	}
}

func (w *Worker) fail(ctx context.Context, cmd *ent.OperationalCommand, cause error) error {
	status := StatusPending
	if cmd.Attempt >= cmd.MaxAttempts {
		status = StatusDeadLetter
	}
	delay := time.Duration(math.Min(math.Pow(2, float64(cmd.Attempt)), 300)) * time.Second
	update := w.client.OperationalCommand.UpdateOneID(cmd.ID).
		Where(operationalcommand.StatusEQ(StatusProcessing), operationalcommand.LeaseOwnerEQ(w.owner), operationalcommand.FencingTokenEQ(cmd.FencingToken)).
		SetStatus(status).
		SetLastError(truncateError(cause)).
		ClearLeaseOwner().
		ClearLeaseExpiresAt()
	if status == StatusPending {
		update.SetAvailableAt(w.now().Add(delay))
	} else {
		update.SetCompletedAt(w.now())
	}
	if _, err := update.Save(ctx); ent.IsNotFound(err) {
		return ErrLeaseLost
	} else if err != nil {
		return err
	}
	w.logger.Warnw("operational command failed", "command_id", cmd.ID, "command_type", cmd.CommandType,
		"tenant_id", cmd.TenantID, "attempt", cmd.Attempt, "status", status, "error", cause)
	return nil
}

func truncateError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if len(message) > 2000 {
		return message[:2000]
	}
	return message
}
