package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/processinstance"

	"go.uber.org/zap"
)

// WorkflowStartCommandHandler starts BPMN processes idempotently for durable commands.
type WorkflowStartCommandHandler struct {
	client  *ent.Client
	trigger ProcessTriggerServiceInterface
	logger  *zap.SugaredLogger
}

func NewWorkflowStartCommandHandler(client *ent.Client, trigger ProcessTriggerServiceInterface, logger *zap.SugaredLogger) *WorkflowStartCommandHandler {
	return &WorkflowStartCommandHandler{client: client, trigger: trigger, logger: logger}
}

func (h *WorkflowStartCommandHandler) Handle(ctx context.Context, cmd *ent.OperationalCommand) error {
	if cmd == nil || cmd.TenantID <= 0 || cmd.AggregateID <= 0 {
		return fmt.Errorf("invalid workflow command")
	}
	businessKey := fmt.Sprintf("%s:%d", cmd.AggregateType, cmd.AggregateID)
	exists, err := h.client.ProcessInstance.Query().
		Where(processinstance.TenantIDEQ(cmd.TenantID), processinstance.BusinessKeyEQ(businessKey)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check existing workflow: %w", err)
	}
	if exists {
		h.logger.Infow("workflow start command already applied", "command_id", cmd.ID, "business_key", businessKey)
		return nil
	}

	var req *dto.ProcessTriggerRequest
	switch cmd.AggregateType {
	case "incident":
		inc, err := h.client.Incident.Query().Where(incident.IDEQ(cmd.AggregateID), incident.TenantIDEQ(cmd.TenantID), incident.DeletedAtIsNil()).Only(ctx)
		if err != nil {
			return fmt.Errorf("load incident for workflow: %w", err)
		}
		processKey := "incident_emergency_flow"
		req = &dto.ProcessTriggerRequest{
			BusinessType: dto.BusinessTypeIncident, BusinessID: inc.ID, ProcessDefinitionKey: processKey,
			Variables: map[string]interface{}{
				"incident_id": inc.ID, "incident_number": inc.IncidentNumber, "title": inc.Title,
				"description": inc.Description, "priority": inc.Priority, "severity": inc.Severity,
				"status": inc.Status, "category": inc.Category, "reporter_id": inc.ReporterID, "assignee_id": inc.AssigneeID,
			},
			TriggeredBy: fmt.Sprintf("%d", inc.ReporterID), TriggeredAt: time.Now(), TenantID: cmd.TenantID,
		}
	case "change":
		ch, err := h.client.Change.Query().Where(change.IDEQ(cmd.AggregateID), change.TenantIDEQ(cmd.TenantID)).Only(ctx)
		if err != nil {
			return fmt.Errorf("load change for workflow: %w", err)
		}
		processKey := "change_normal_flow"
		if ch.Type == "emergency" {
			processKey = "change_emergency_flow"
		}
		req = &dto.ProcessTriggerRequest{
			BusinessType: dto.BusinessTypeChange, BusinessID: ch.ID, ProcessDefinitionKey: processKey,
			Variables: map[string]interface{}{
				"change_id": ch.ID, "title": ch.Title, "description": ch.Description, "type": ch.Type,
				"priority": ch.Priority, "status": ch.Status, "impact_scope": ch.ImpactScope,
				"risk_level": ch.RiskLevel, "created_by": ch.CreatedBy, "assignee_id": ch.AssigneeID,
			},
			TriggeredBy: fmt.Sprintf("%d", ch.CreatedBy), TriggeredAt: time.Now(), TenantID: cmd.TenantID,
		}
	default:
		return fmt.Errorf("unsupported workflow aggregate type %q", cmd.AggregateType)
	}
	if _, err := h.trigger.TriggerProcess(ctx, req); err != nil {
		return fmt.Errorf("start %s workflow: %w", cmd.AggregateType, err)
	}
	return nil
}
