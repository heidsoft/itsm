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
	"itsm-backend/internal/commandbus"

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

// IncidentAlertDeliveryCommandHandler 处理 incident_alert.deliver 命令：
// 在 worker 内重载 alert（带租户隔离），由 IncidentAlertingService.DeliverExternalAlert
// 实际触发外部渠道（email / sms / slack / webhook）。原 fire-and-forget goroutine
// 在请求结束后会被取消；现在交由 commandbus 调度，可重试到 max_attempts 并
// 由 operations.operations API 暴露的死信/重放兜底。
type IncidentAlertDeliveryCommandHandler struct {
	alerting *IncidentAlertingService
}

func NewIncidentAlertDeliveryCommandHandler(alerting *IncidentAlertingService) *IncidentAlertDeliveryCommandHandler {
	if alerting == nil {
		return nil
	}
	return &IncidentAlertDeliveryCommandHandler{alerting: alerting}
}

func (h *IncidentAlertDeliveryCommandHandler) Handle(ctx context.Context, cmd *ent.OperationalCommand) error {
	if h == nil || h.alerting == nil || cmd == nil {
		return fmt.Errorf("invalid incident alert delivery command")
	}
	if cmd.CommandType != commandbus.CommandDeliverIncidentAlert ||
		cmd.AggregateType != "incident_alert" ||
		cmd.AggregateID <= 0 ||
		cmd.TenantID <= 0 {
		return fmt.Errorf("invalid incident alert delivery command")
	}
	executionCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return h.alerting.DeliverExternalAlert(executionCtx, cmd.AggregateID, cmd.TenantID)
}

// ProvisioningTaskCommandHandler 处理 provisioning.task.execute 命令：
// 在 worker 内重载 ProvisioningTask，调 ProvisioningService.ExecuteTask 完成执行。
// 入箱时已用 task ID 作 idempotency_key，replay / 重试不会重复执行同一任务。
// 现有手动 POST /provisioning-tasks/:id/execute 端点保留，可作为运维强制重试入口。
type ProvisioningTaskCommandHandler struct {
	svc *ProvisioningService
}

func NewProvisioningTaskCommandHandler(svc *ProvisioningService) *ProvisioningTaskCommandHandler {
	if svc == nil {
		return nil
	}
	return &ProvisioningTaskCommandHandler{svc: svc}
}

func (h *ProvisioningTaskCommandHandler) Handle(ctx context.Context, cmd *ent.OperationalCommand) error {
	if h == nil || h.svc == nil || cmd == nil {
		return fmt.Errorf("invalid provisioning task command")
	}
	if cmd.CommandType != commandbus.CommandExecuteProvisioningTask ||
		cmd.AggregateType != "provisioning_task" ||
		cmd.AggregateID <= 0 ||
		cmd.TenantID <= 0 {
		return fmt.Errorf("invalid provisioning task command")
	}
	actorUserID, _ := cmd.Payload["actorUserId"].(float64)
	executionCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	_, err := h.svc.ExecuteTask(executionCtx, cmd.AggregateID, cmd.TenantID, int(actorUserID))
	return err
}
