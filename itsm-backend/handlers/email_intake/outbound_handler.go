package email_intake

import (
	"context"
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
	_, _ = h.client.EmailOutboundMessage.UpdateOneID(outbound.ID).SetStatus("SENDING").AddAttempts(1).Save(ctx)
	err = h.manager.Send(ctx, command.TenantID, "email", &connector.Message{Channel: outbound.ToAddress, Type: "text", Title: outbound.Subject, Content: outbound.BodyText, ReplyTo: outbound.InReplyTo, ThreadID: fmt.Sprintf("%d", outbound.ConversationID)})
	if err != nil {
		_, _ = h.client.EmailOutboundMessage.UpdateOneID(outbound.ID).SetStatus("FAILED").SetLastError(err.Error()).Save(ctx)
		return err
	}
	_, err = h.client.EmailOutboundMessage.UpdateOneID(outbound.ID).SetStatus("SENT").SetSentAt(time.Now()).ClearLastError().Save(ctx)
	return err
}
