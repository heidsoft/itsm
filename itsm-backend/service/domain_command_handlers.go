package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/connector"
	feishuconnector "itsm-backend/connector/builtin/feishu"
	"itsm-backend/ent"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/ticket"

	"go.uber.org/zap"
)

type TicketAutomationCommandHandler struct {
	service *TicketAutomationRuleService
}

func NewTicketAutomationCommandHandler(service *TicketAutomationRuleService) *TicketAutomationCommandHandler {
	return &TicketAutomationCommandHandler{service: service}
}

func (h *TicketAutomationCommandHandler) Handle(ctx context.Context, cmd *ent.OperationalCommand) error {
	if h.service == nil || cmd == nil || cmd.TenantID <= 0 || cmd.AggregateType != "ticket" || cmd.AggregateID <= 0 {
		return fmt.Errorf("invalid ticket automation command")
	}
	executionCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return h.service.ExecuteRulesForTicket(executionCtx, cmd.AggregateID, cmd.TenantID)
}

type TicketFeishuSyncCommandHandler struct {
	client  *ent.Client
	manager *connector.Manager
	logger  *zap.SugaredLogger
}

func NewTicketFeishuSyncCommandHandler(client *ent.Client, manager *connector.Manager, logger *zap.SugaredLogger) *TicketFeishuSyncCommandHandler {
	return &TicketFeishuSyncCommandHandler{client: client, manager: manager, logger: logger}
}

func (h *TicketFeishuSyncCommandHandler) Handle(ctx context.Context, cmd *ent.OperationalCommand) error {
	if h.client == nil || h.manager == nil || cmd == nil || cmd.TenantID <= 0 || cmd.AggregateType != "ticket" || cmd.AggregateID <= 0 {
		return fmt.Errorf("invalid ticket feishu sync command")
	}
	conn, ok := h.manager.Get(cmd.TenantID, "feishu")
	if !ok {
		h.logger.Debugw("ticket feishu sync skipped: connector not configured", "tenant_id", cmd.TenantID, "ticket_id", cmd.AggregateID)
		return nil
	}
	feishu, ok := conn.(*feishuconnector.Feishu)
	if !ok {
		return fmt.Errorf("tenant feishu connector has unexpected type")
	}
	syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tkt, err := h.client.Ticket.Query().Where(ticket.IDEQ(cmd.AggregateID), ticket.TenantIDEQ(cmd.TenantID), ticket.DeletedAtIsNil()).Only(syncCtx)
	if err != nil {
		return fmt.Errorf("load ticket for feishu sync: %w", err)
	}
	tx, err := h.client.Tx(syncCtx)
	if err != nil {
		return fmt.Errorf("begin feishu sync transaction: %w", err)
	}
	defer tx.Rollback()
	if _, err := feishu.SyncTicketToFeishu(syncCtx, tx, tkt); err != nil {
		return fmt.Errorf("sync ticket to feishu: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit feishu sync: %w", err)
	}
	return nil
}

type IncidentRulesCommandHandler struct {
	client *ent.Client
	engine *IncidentRuleEngine
}

func NewIncidentRulesCommandHandler(client *ent.Client, engine *IncidentRuleEngine) *IncidentRulesCommandHandler {
	return &IncidentRulesCommandHandler{client: client, engine: engine}
}

func (h *IncidentRulesCommandHandler) Handle(ctx context.Context, cmd *ent.OperationalCommand) error {
	if h.client == nil || h.engine == nil || cmd == nil || cmd.TenantID <= 0 || cmd.AggregateType != "incident" || cmd.AggregateID <= 0 {
		return fmt.Errorf("invalid incident rules command")
	}
	executionCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	if _, err := h.client.Incident.Query().Where(incident.IDEQ(cmd.AggregateID), incident.TenantIDEQ(cmd.TenantID), incident.DeletedAtIsNil()).Only(executionCtx); err != nil {
		return fmt.Errorf("load incident for rules: %w", err)
	}
	return h.engine.ExecuteRulesForIncident(executionCtx, cmd.AggregateID, cmd.TenantID)
}
