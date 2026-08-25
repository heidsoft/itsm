package email_intake

import (
	"context"
	"errors"
	"fmt"
	"time"

	"itsm-backend/connector"
	"itsm-backend/ent"
	"itsm-backend/ent/emailoutboundmessage"
)

type OutboundCommandHandler struct {
	client  *ent.Client
	manager *connector.Manager
}

type IntakeProcessCommandHandler struct{ orchestrator *EmailIntakeOrchestrator }

func NewIntakeProcessCommandHandler(orchestrator *EmailIntakeOrchestrator) *IntakeProcessCommandHandler {
	return &IntakeProcessCommandHandler{orchestrator: orchestrator}
}

func (h *IntakeProcessCommandHandler) Handle(ctx context.Context, command *ent.OperationalCommand) error {
	if h == nil || h.orchestrator == nil || command.AggregateType != "inbound_email_message" {
		return errors.New("email intake process handler is not configured for aggregate")
	}
	return h.orchestrator.RetryMessage(ctx, command.TenantID, command.AggregateID)
}

func NewOutboundCommandHandler(client *ent.Client, manager *connector.Manager) *OutboundCommandHandler {
	return &OutboundCommandHandler{client: client, manager: manager}
}

func (h *OutboundCommandHandler) Handle(ctx context.Context, command *ent.OperationalCommand) error {
	if h == nil || h.client == nil || h.manager == nil {
		return fmt.Errorf("email outbound handler is not configured")
	}
	outbound, err := h.client.EmailOutboundMessage.Query().Where(emailoutboundmessage.IDEQ(command.AggregateID), emailoutboundmessage.TenantIDEQ(command.TenantID)).Only(ctx)
	if err != nil {
		return fmt.Errorf("load outbound email: %w", err)
	}
	if outbound.Status == "SENT" {
		return nil
	}
	deliveryToken := fmt.Sprintf("command-%d-attempt-%d", command.ID, command.Attempt)
	claimStatuses := []string{"PENDING", "FAILED"}
	if command.Attempt > 1 {
		claimStatuses = append(claimStatuses, "SENDING")
	}
	outbound, err = h.client.EmailOutboundMessage.UpdateOneID(outbound.ID).Where(
		emailoutboundmessage.TenantIDEQ(command.TenantID), emailoutboundmessage.StatusIn(claimStatuses...),
	// last_error doubles as the in-flight fencing token while status=SENDING;
	// terminal states replace/clear it, so a superseded worker cannot finalize.
	).SetStatus("SENDING").SetLastError(deliveryToken).AddAttempts(1).Save(ctx)
	if ent.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("claim outbound email: %w", err)
	}
	err = h.manager.Send(ctx, command.TenantID, "email", &connector.Message{ID: fmt.Sprintf("email-outbound-%d", outbound.ID), Channel: outbound.ToAddress, Type: "text", Title: outbound.Subject, Content: outbound.BodyText, ReplyTo: outbound.InReplyTo, ThreadID: fmt.Sprintf("%d", outbound.ConversationID)})
	if err != nil {
		if _, updateErr := h.client.EmailOutboundMessage.UpdateOneID(outbound.ID).Where(emailoutboundmessage.TenantIDEQ(command.TenantID), emailoutboundmessage.StatusEQ("SENDING"), emailoutboundmessage.LastErrorEQ(deliveryToken)).SetStatus("FAILED").SetLastError(safeOutboundError(err)).Save(ctx); updateErr != nil && !ent.IsNotFound(updateErr) {
			return fmt.Errorf("send outbound email: %v; persist failure: %w", err, updateErr)
		}
		return err
	}
	_, err = h.client.EmailOutboundMessage.UpdateOneID(outbound.ID).Where(emailoutboundmessage.TenantIDEQ(command.TenantID), emailoutboundmessage.StatusEQ("SENDING"), emailoutboundmessage.LastErrorEQ(deliveryToken)).SetStatus("SENT").SetSentAt(time.Now()).ClearLastError().Save(ctx)
	if ent.IsNotFound(err) {
		return nil
	}
	return err
}

func safeOutboundError(err error) string {
	if err == nil {
		return ""
	}
	return "email delivery failed"
}
