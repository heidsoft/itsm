package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"itsm-backend/connector"
	"itsm-backend/ent"
	"itsm-backend/ent/notificationdelivery"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

type NotificationDeliveryCommandHandler struct {
	client     *ent.Client
	connectors *connector.Manager
	logger     *zap.SugaredLogger
}

func NewNotificationDeliveryCommandHandler(client *ent.Client, connectors *connector.Manager, logger *zap.SugaredLogger) *NotificationDeliveryCommandHandler {
	return &NotificationDeliveryCommandHandler{client: client, connectors: connectors, logger: logger}
}

func (h *NotificationDeliveryCommandHandler) Handle(ctx context.Context, cmd *ent.OperationalCommand) error {
	if cmd == nil {
		return fmt.Errorf("notification command is required")
	}
	existing, err := h.client.NotificationDelivery.Query().
		Where(notificationdelivery.OperationalCommandIDEQ(cmd.ID), notificationdelivery.TenantIDEQ(cmd.TenantID)).
		Only(ctx)
	if err == nil && existing.Status == "sent" {
		return nil
	}
	if err != nil && !ent.IsNotFound(err) {
		return fmt.Errorf("query delivery audit: %w", err)
	}

	ticketID, err := payloadInt(cmd.Payload, "ticketId")
	if err != nil {
		return err
	}
	recipientID, err := payloadInt(cmd.Payload, "recipientId")
	if err != nil {
		return err
	}
	channel := payloadString(cmd.Payload, "channel")
	notificationType := payloadString(cmd.Payload, "type")
	content := payloadString(cmd.Payload, "content")
	if channel == "" || notificationType == "" || content == "" {
		return fmt.Errorf("notification channel, type and content are required")
	}

	tk, err := h.client.Ticket.Query().Where(ticket.IDEQ(ticketID), ticket.TenantIDEQ(cmd.TenantID)).Only(ctx)
	if err != nil {
		return fmt.Errorf("load notification ticket: %w", err)
	}
	recipient, err := h.client.User.Query().Where(user.IDEQ(recipientID), user.TenantIDEQ(cmd.TenantID), user.ActiveEQ(true)).Only(ctx)
	if err != nil {
		return fmt.Errorf("load notification recipient: %w", err)
	}

	if channel == "in_app" {
		return h.deliverInApp(ctx, cmd, tk, recipient, notificationType, content)
	}
	return h.deliverConnector(ctx, cmd, tk, recipient, channel, notificationType, content, existing)
}

func (h *NotificationDeliveryCommandHandler) deliverInApp(ctx context.Context, cmd *ent.OperationalCommand, tk *ent.Ticket, recipient *ent.User, notificationType, content string) error {
	tx, err := h.client.Tx(ctx)
	if err != nil {
		return err
	}
	rollback := func(cause error) error { _ = tx.Rollback(); return cause }
	now := time.Now()
	ticketNotification, err := tx.TicketNotification.Create().
		SetTicketID(tk.ID).SetUserID(recipient.ID).SetType(notificationType).SetChannel("in_app").
		SetContent(content).SetTenantID(cmd.TenantID).SetStatus("sent").SetSentAt(now).Save(ctx)
	if err != nil {
		return rollback(err)
	}
	_, err = tx.Notification.Create().SetTitle(notificationType).SetMessage(content).SetType("info").
		SetUserID(recipient.ID).SetTenantID(cmd.TenantID).SetActionURL(fmt.Sprintf("/tickets/%d", tk.ID)).SetActionText("查看工单").Save(ctx)
	if err != nil {
		return rollback(err)
	}
	_, err = tx.NotificationDelivery.Create().SetTenantID(cmd.TenantID).SetOperationalCommandID(cmd.ID).
		SetTicketID(tk.ID).SetTicketNotificationID(ticketNotification.ID).SetRecipientID(recipient.ID).
		SetChannel("in_app").SetTargetMasked(fmt.Sprintf("user:%d", recipient.ID)).SetStatus("sent").
		SetAttempt(cmd.Attempt).SetSentAt(now).Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			_ = tx.Rollback()
			return nil
		}
		return rollback(err)
	}
	return tx.Commit()
}

func (h *NotificationDeliveryCommandHandler) deliverConnector(ctx context.Context, cmd *ent.OperationalCommand, tk *ent.Ticket, recipient *ent.User, channel, notificationType, content string, existing *ent.NotificationDelivery) error {
	if h.connectors == nil {
		return fmt.Errorf("connector manager is not configured")
	}
	target := notificationTarget(recipient, channel)
	if target == "" {
		return fmt.Errorf("recipient has no configured target for %s", channel)
	}
	var delivery *ent.NotificationDelivery
	var ticketNotificationID int
	if existing == nil {
		tx, err := h.client.Tx(ctx)
		if err != nil {
			return err
		}
		tn, err := tx.TicketNotification.Create().SetTicketID(tk.ID).SetUserID(recipient.ID).
			SetType(notificationType).SetChannel(channel).SetContent(content).SetTenantID(cmd.TenantID).SetStatus("pending").Save(ctx)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		ticketNotificationID = tn.ID
		delivery, err = tx.NotificationDelivery.Create().SetTenantID(cmd.TenantID).SetOperationalCommandID(cmd.ID).
			SetTicketID(tk.ID).SetTicketNotificationID(tn.ID).SetRecipientID(recipient.ID).SetChannel(channel).
			SetTargetMasked(maskTarget(target)).SetStatus("pending").SetAttempt(cmd.Attempt).Save(ctx)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	} else {
		delivery = existing
		if existing.TicketNotificationID != nil {
			ticketNotificationID = *existing.TicketNotificationID
		}
	}
	messageID := cmd.IdempotencyKey
	err := h.connectors.Send(ctx, cmd.TenantID, channel, &connector.Message{
		ID: messageID, Channel: target, Type: "text", Title: notificationType, Content: content,
		Actions:  []connector.Action{{Type: "link", Text: "查看工单", URL: fmt.Sprintf("/tickets/%d", tk.ID)}},
		Metadata: map[string]interface{}{"ticket_id": tk.ID, "recipient_id": recipient.ID, "command_id": cmd.ID},
	})
	if err != nil {
		safeErr := fmt.Errorf("connector %s delivery failed", channel)
		_, _ = h.client.NotificationDelivery.UpdateOneID(delivery.ID).SetStatus("failed").SetAttempt(cmd.Attempt).
			SetErrorCode("connector_send_failed").SetErrorMessage(safeDeliveryError(safeErr)).Save(ctx)
		if ticketNotificationID > 0 {
			_, _ = h.client.TicketNotification.UpdateOneID(ticketNotificationID).SetStatus("failed").Save(ctx)
		}
		h.logger.Warnw("connector notification delivery failed", "channel", channel, "tenant_id", cmd.TenantID,
			"command_id", cmd.ID, "recipient_id", recipient.ID, "error_type", fmt.Sprintf("%T", err))
		return safeErr
	}
	now := time.Now()
	_, err = h.client.NotificationDelivery.UpdateOneID(delivery.ID).SetStatus("sent").SetAttempt(cmd.Attempt).
		SetProviderMessageID(messageID).ClearErrorCode().ClearErrorMessage().SetSentAt(now).Save(ctx)
	if err != nil {
		return err
	}
	if ticketNotificationID > 0 {
		_, err = h.client.TicketNotification.UpdateOneID(ticketNotificationID).SetStatus("sent").SetSentAt(now).Save(ctx)
	}
	return err
}

func payloadInt(payload map[string]interface{}, key string) (int, error) {
	v, ok := payload[key]
	if !ok {
		return 0, fmt.Errorf("notification payload %s is required", key)
	}
	switch value := v.(type) {
	case int:
		return value, nil
	case float64:
		return int(value), nil
	case string:
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed, nil
		}
	}
	return 0, fmt.Errorf("notification payload %s is invalid", key)
}

func payloadString(payload map[string]interface{}, key string) string {
	value, _ := payload[key].(string)
	return value
}

func notificationTarget(recipient *ent.User, channel string) string {
	switch channel {
	case "feishu":
		return recipient.FeishuOpenID
	case "dingtalk", "wecom":
		return recipient.Username
	case "email":
		return recipient.Email
	case "sms":
		return recipient.Phone
	case "webhook":
		return "tenant-default-webhook"
	default:
		return ""
	}
}

func maskTarget(target string) string {
	if len(target) <= 4 {
		return "****"
	}
	return target[:2] + "****" + target[len(target)-2:]
}

func safeDeliveryError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if len(message) > 1000 {
		message = message[:1000]
	}
	return message
}
