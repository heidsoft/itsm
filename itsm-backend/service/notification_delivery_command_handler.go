package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"itsm-backend/connector"
	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/notificationdelivery"
	"itsm-backend/ent/notificationpreference"
	"itsm-backend/ent/servicerequest"
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

	resourceType := cmd.AggregateType
	resourceID := cmd.AggregateID
	if payloadType := payloadString(cmd.Payload, "resourceType"); payloadType != "" {
		resourceType = payloadType
	}
	if _, ok := cmd.Payload["resourceId"]; ok {
		resourceID, err = payloadInt(cmd.Payload, "resourceId")
	} else if resourceType == "ticket" {
		resourceID, err = payloadInt(cmd.Payload, "ticketId")
	}
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

	var tk *ent.Ticket
	actionURL, actionText := "", ""
	switch resourceType {
	case "ticket":
		tk, err = h.client.Ticket.Query().Where(ticket.IDEQ(resourceID), ticket.TenantIDEQ(cmd.TenantID)).Only(ctx)
		actionURL, actionText = fmt.Sprintf("/tickets/%d", resourceID), "查看工单"
	case "change":
		_, err = h.client.Change.Query().Where(change.IDEQ(resourceID), change.TenantIDEQ(cmd.TenantID)).Only(ctx)
		actionURL, actionText = fmt.Sprintf("/changes/%d", resourceID), "查看变更"
	case "service_request":
		_, err = h.client.ServiceRequest.Query().Where(
			servicerequest.IDEQ(resourceID), servicerequest.TenantIDEQ(cmd.TenantID),
		).Only(ctx)
		actionURL, actionText = fmt.Sprintf("/service-requests/%d", resourceID), "查看服务请求"
	default:
		return fmt.Errorf("unsupported notification resource type %q", resourceType)
	}
	if err != nil {
		return fmt.Errorf("load notification %s: %w", resourceType, err)
	}
	recipient, err := h.client.User.Query().Where(user.IDEQ(recipientID), user.TenantIDEQ(cmd.TenantID), user.ActiveEQ(true)).Only(ctx)
	if err != nil {
		return fmt.Errorf("load notification recipient: %w", err)
	}

	// 消费 per-event 通知偏好（notification_preferences 表）。语义与同步路径
	// （TicketNotificationService.channelAllowedForEvent）一致：无偏好记录默认放行，
	// 查询失败留痕放行（best-effort，不阻塞投递主链路）。
	// 入箱载荷的 type 是短词表（created/assigned/sla_alert/…），偏好表是长词表
	// （ticket_created/sla_violated/…），必须先归一化再查询，否则永远匹配不上。
	eventType := normalizeNotificationEventType(notificationType)
	allowed, prefErr := h.channelAllowedForEvent(ctx, recipientID, cmd.TenantID, eventType, channel)
	if prefErr != nil {
		h.logger.Warnw("failed to load notification preference for delivery, allowing send",
			"recipient_id", recipientID, "tenant_id", cmd.TenantID,
			"event_type", eventType, "channel", channel, "error", prefErr)
		allowed = true
	}
	if !allowed {
		// 用户偏好显式关闭该渠道：语义是「按用户选择跳过」而非失败，
		// 标记命令成功且不写投递记录，避免 worker 无效重试到 dead_letter。
		h.logger.Infow("notification delivery skipped by user preference",
			"recipient_id", recipientID, "tenant_id", cmd.TenantID,
			"event_type", eventType, "channel", channel, "command_id", cmd.ID)
		return nil
	}

	if channel == "in_app" {
		return h.deliverInApp(ctx, cmd, tk, recipient, notificationType, content, actionURL, actionText)
	}
	return h.deliverConnector(ctx, cmd, tk, recipient, channel, notificationType, content, actionURL, actionText, resourceType, resourceID, existing)
}

// normalizeNotificationEventType 将入箱载荷的短 event_type 归一化为偏好表长词表。
// 词表对齐 dto.ListNotificationEventTypes 与前端 EVENT_TYPES；未识别的值原样返回
// （查不到偏好行 → 默认放行，与「无偏好=放行」语义收敛）。
func normalizeNotificationEventType(notificationType string) string {
	switch notificationType {
	case "created":
		return "ticket_created"
	case "assigned":
		return "ticket_assigned"
	case "updated", "status_changed":
		return "ticket_updated"
	case "commented":
		return "comment_added"
	case "resolved":
		return "ticket_resolved"
	case "closed":
		return "ticket_closed"
	case "sla_warning", "sla_alert":
		return "sla_warning"
	case "sla_breached":
		return "sla_violated"
	case "change_approval_required":
		return "approval_required"
	case "change_approval_decided":
		return "approval_completed"
	default:
		return notificationType
	}
}

// channelAllowedForEvent 按 event_type+channel 查询用户偏好并判定渠道是否放行。
// 无偏好记录 → 放行；connector 渠道按 push_enabled 判定（与偏好模型一致）。
func (h *NotificationDeliveryCommandHandler) channelAllowedForEvent(ctx context.Context, userID, tenantID int, eventType, channel string) (bool, error) {
	pref, err := h.client.NotificationPreference.Query().
		Where(
			notificationpreference.UserID(userID),
			notificationpreference.TenantID(tenantID),
			notificationpreference.EventType(eventType),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return true, nil
		}
		return false, fmt.Errorf("query notification preference: %w", err)
	}
	switch channel {
	case "email":
		return pref.EmailEnabled, nil
	case "in_app":
		return pref.InAppEnabled, nil
	case "sms":
		return pref.SmsEnabled, nil
	default:
		// feishu / dingtalk / wecom / webhook / slack 等连接器渠道按推送开关判定
		return pref.PushEnabled, nil
	}
}

func (h *NotificationDeliveryCommandHandler) deliverInApp(ctx context.Context, cmd *ent.OperationalCommand, tk *ent.Ticket, recipient *ent.User, notificationType, content, actionURL, actionText string) error {
	tx, err := h.client.Tx(ctx)
	if err != nil {
		return err
	}
	rollback := func(cause error) error { _ = tx.Rollback(); return cause }
	now := time.Now()
	var ticketNotificationID *int
	if tk != nil {
		ticketNotification, err := tx.TicketNotification.Create().
			SetTicketID(tk.ID).SetUserID(recipient.ID).SetType(notificationType).SetChannel("in_app").
			SetContent(content).SetTenantID(cmd.TenantID).SetStatus("sent").SetSentAt(now).Save(ctx)
		if err != nil {
			return rollback(err)
		}
		ticketNotificationID = &ticketNotification.ID
	}
	_, err = tx.Notification.Create().SetTitle(notificationType).SetMessage(content).SetType("info").
		SetUserID(recipient.ID).SetTenantID(cmd.TenantID).SetActionURL(actionURL).SetActionText(actionText).Save(ctx)
	if err != nil {
		return rollback(err)
	}
	deliveryCreate := tx.NotificationDelivery.Create().SetTenantID(cmd.TenantID).SetOperationalCommandID(cmd.ID).
		SetNillableTicketNotificationID(ticketNotificationID).SetRecipientID(recipient.ID).
		SetChannel("in_app").SetTargetMasked(fmt.Sprintf("user:%d", recipient.ID)).SetStatus("sent").
		SetAttempt(cmd.Attempt).SetSentAt(now)
	if tk != nil {
		deliveryCreate.SetTicketID(tk.ID)
	}
	_, err = deliveryCreate.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			_ = tx.Rollback()
			return nil
		}
		return rollback(err)
	}
	return tx.Commit()
}

func (h *NotificationDeliveryCommandHandler) deliverConnector(ctx context.Context, cmd *ent.OperationalCommand, tk *ent.Ticket, recipient *ent.User, channel, notificationType, content, actionURL, actionText, resourceType string, resourceID int, existing *ent.NotificationDelivery) error {
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
		if tk != nil {
			tn, err := tx.TicketNotification.Create().SetTicketID(tk.ID).SetUserID(recipient.ID).
				SetType(notificationType).SetChannel(channel).SetContent(content).SetTenantID(cmd.TenantID).SetStatus("pending").Save(ctx)
			if err != nil {
				_ = tx.Rollback()
				return err
			}
			ticketNotificationID = tn.ID
		}
		deliveryCreate := tx.NotificationDelivery.Create().SetTenantID(cmd.TenantID).SetOperationalCommandID(cmd.ID).
			SetRecipientID(recipient.ID).SetChannel(channel).SetTargetMasked(maskTarget(target)).SetStatus("pending").SetAttempt(cmd.Attempt)
		if tk != nil {
			deliveryCreate.SetTicketID(tk.ID).SetTicketNotificationID(ticketNotificationID)
		}
		delivery, err = deliveryCreate.Save(ctx)
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
		Actions:  []connector.Action{{Type: "link", Text: actionText, URL: actionURL}},
		Metadata: map[string]interface{}{"resource_type": resourceType, "resource_id": resourceID, "recipient_id": recipient.ID, "command_id": cmd.ID},
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
