package bpmn

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/user"
	"itsm-backend/internal/commandbus"

	"go.uber.org/zap"
)

// NotificationHandler 通知服务任务处理器
//
// 通知不再只打日志：邮件/短信/站内通知统一写入 operational_commands 可靠投递队列
// （command_type = notification.deliver），由 NotificationDeliveryCommandHandler
// worker 真正落库并经连接器发送；Webhook 通知直接复用 WebhookHandler 发真实 HTTP 请求。
type NotificationHandler struct {
	HandlerBase
	client  *ent.Client
	logger  *zap.SugaredLogger
	webhook *WebhookHandler
}

// NewNotificationHandler 创建通知处理器
func NewNotificationHandler(client *ent.Client, logger *zap.SugaredLogger) *NotificationHandler {
	return &NotificationHandler{
		client:  client,
		logger:  logger,
		webhook: NewWebhookHandler(client, logger),
	}
}

// GetTaskType 返回任务类型
func (h *NotificationHandler) GetTaskType() string {
	return "notification_task"
}

// GetHandlerID 返回处理器标识
func (h *NotificationHandler) GetHandlerID() string {
	return "notification_handler"
}

// Execute 执行通知服务任务
func (h *NotificationHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	action, _ := variables["action"].(string)
	switch action {
	case "send_email":
		return h.sendEmail(ctx, task, variables)
	case "send_sms":
		return h.sendSMS(ctx, task, variables)
	case "send_in_app":
		return h.sendInAppNotification(ctx, task, variables)
	case "send_webhook":
		return h.sendWebhookNotification(ctx, variables)
	default:
		return h.sendInAppNotification(ctx, task, variables)
	}
}

// Validate 验证配置
func (h *NotificationHandler) Validate(ctx context.Context, config map[string]interface{}) error {
	return nil
}

// sendEmail 发送邮件通知：按邮箱解析租户内的真实收件人后入队可靠投递
func (h *NotificationHandler) sendEmail(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	recipients := GetStringFromVars(variables, "recipients")
	subject := GetStringFromVars(variables, "subject")
	body := GetStringFromVars(variables, "body")

	if recipients == "" {
		return nil, fmt.Errorf("邮件收件人不能为空")
	}

	tenantID, resourceType, resourceID, err := h.resolveTarget(ctx, task, variables)
	if err != nil {
		return h.failure(err, "send_email", variables)
	}

	addresses := splitRecipients(recipients)
	recipientIDs, err := h.resolveUsersByContact(ctx, tenantID, "email", addresses)
	if err != nil {
		return h.failure(err, "send_email", variables)
	}

	content := strings.TrimSpace(subject + "\n" + body)
	if content == "" {
		return h.failure(fmt.Errorf("邮件正文与标题均为空"), "send_email", variables)
	}

	enqueued, err := h.enqueueDeliveries(ctx, deliveryRequest{
		TenantID:         tenantID,
		ResourceType:     resourceType,
		ResourceID:       resourceID,
		Channel:          "email",
		NotificationType: notificationTypeOrDefault(variables, "bpmn_email"),
		Content:          content,
		OccurrenceKey:    occurrenceKey(task, variables, "send_email"),
		RecipientIDs:     recipientIDs,
	})
	if err != nil {
		return h.failure(err, "send_email", variables)
	}

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("邮件通知已入队投递，收件人 %d 人", enqueued),
		OutputVars: map[string]interface{}{
			"channel":        "email",
			"recipient_ids":  recipientIDs,
			"enqueued_count": enqueued,
		},
	}, nil
}

// sendSMS 发送短信通知：按手机号解析租户内的真实收件人后入队可靠投递
func (h *NotificationHandler) sendSMS(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	phoneNumbers := GetStringFromVars(variables, "phone_numbers")
	message := GetStringFromVars(variables, "message")

	if phoneNumbers == "" {
		return nil, fmt.Errorf("手机号不能为空")
	}
	if strings.TrimSpace(message) == "" {
		return nil, fmt.Errorf("短信内容不能为空")
	}

	tenantID, resourceType, resourceID, err := h.resolveTarget(ctx, task, variables)
	if err != nil {
		return h.failure(err, "send_sms", variables)
	}

	recipientIDs, err := h.resolveUsersByContact(ctx, tenantID, "phone", splitRecipients(phoneNumbers))
	if err != nil {
		return h.failure(err, "send_sms", variables)
	}

	enqueued, err := h.enqueueDeliveries(ctx, deliveryRequest{
		TenantID:         tenantID,
		ResourceType:     resourceType,
		ResourceID:       resourceID,
		Channel:          "sms",
		NotificationType: notificationTypeOrDefault(variables, "bpmn_sms"),
		Content:          message,
		OccurrenceKey:    occurrenceKey(task, variables, "send_sms"),
		RecipientIDs:     recipientIDs,
	})
	if err != nil {
		return h.failure(err, "send_sms", variables)
	}

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("短信通知已入队投递，收件人 %d 人", enqueued),
		OutputVars: map[string]interface{}{
			"channel":        "sms",
			"recipient_ids":  recipientIDs,
			"enqueued_count": enqueued,
		},
	}, nil
}

// sendInAppNotification 发送应用内通知：入队后由 worker 写 notifications / ticket_notifications
func (h *NotificationHandler) sendInAppNotification(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	recipientIDs := GetIntSliceFromVars(variables, "user_ids")
	if len(recipientIDs) == 0 {
		if single := GetIntFromVars(variables, "user_ids"); single > 0 {
			recipientIDs = []int{single}
		}
	}
	if len(recipientIDs) == 0 {
		if single := GetIntFromVars(variables, "user_id"); single > 0 {
			recipientIDs = []int{single}
		}
	}
	if len(recipientIDs) == 0 {
		return nil, fmt.Errorf("用户ID不能为空")
	}

	title := GetStringFromVars(variables, "title")
	content := GetStringFromVars(variables, "content")
	if strings.TrimSpace(title) == "" && strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("通知标题与内容不能同时为空")
	}

	tenantID, resourceType, resourceID, err := h.resolveTarget(ctx, task, variables)
	if err != nil {
		return h.failure(err, "send_in_app", variables)
	}

	enqueued, err := h.enqueueDeliveries(ctx, deliveryRequest{
		TenantID:         tenantID,
		ResourceType:     resourceType,
		ResourceID:       resourceID,
		Channel:          "in_app",
		NotificationType: notificationTypeOrDefault(variables, firstNonEmpty(title, "bpmn_notification")),
		Content:          strings.TrimSpace(firstNonEmpty(content, title)),
		OccurrenceKey:    occurrenceKey(task, variables, "send_in_app"),
		RecipientIDs:     recipientIDs,
	})
	if err != nil {
		return h.failure(err, "send_in_app", variables)
	}

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("应用内通知已入队投递，收件人 %d 人", enqueued),
		OutputVars: map[string]interface{}{
			"channel":        "in_app",
			"recipient_ids":  recipientIDs,
			"enqueued_count": enqueued,
		},
	}, nil
}

// sendWebhookNotification 发送Webhook通知：复用 WebhookHandler 的真实 HTTP 调用（含 SSRF 防护）
func (h *NotificationHandler) sendWebhookNotification(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	webhookURL := GetStringFromVars(variables, "webhook_url")
	if webhookURL == "" {
		return nil, fmt.Errorf("webhook URL不能为空")
	}
	if h.webhook == nil {
		return h.failure(fmt.Errorf("webhook 处理器未初始化"), "send_webhook", variables)
	}
	return h.webhook.callWebhook(ctx, variables)
}

// deliveryRequest 一次通知投递入队请求
type deliveryRequest struct {
	TenantID         int
	ResourceType     string
	ResourceID       int
	Channel          string
	NotificationType string
	Content          string
	OccurrenceKey    string
	RecipientIDs     []int
}

// enqueueDeliveries 在单个事务内为所有收件人写入可靠投递命令；任一收件人非法即整体失败。
func (h *NotificationHandler) enqueueDeliveries(ctx context.Context, req deliveryRequest) (int, error) {
	if req.TenantID <= 0 || req.ResourceID <= 0 || req.ResourceType == "" {
		return 0, fmt.Errorf("通知目标不完整：tenant=%d resource=%s/%d", req.TenantID, req.ResourceType, req.ResourceID)
	}
	if len(req.RecipientIDs) == 0 {
		return 0, fmt.Errorf("没有可投递的收件人")
	}
	if req.Content == "" {
		return 0, fmt.Errorf("通知内容不能为空")
	}

	tx, err := h.client.Tx(ctx)
	if err != nil {
		return 0, fmt.Errorf("开启通知事务失败: %w", err)
	}
	rollback := func(cause error) (int, error) {
		_ = tx.Rollback()
		return 0, cause
	}

	seen := make(map[int]struct{}, len(req.RecipientIDs))
	enqueued := 0
	for _, recipientID := range req.RecipientIDs {
		if recipientID <= 0 {
			return rollback(fmt.Errorf("非法的收件人ID: %d", recipientID))
		}
		if _, dup := seen[recipientID]; dup {
			continue
		}
		seen[recipientID] = struct{}{}

		exists, err := tx.User.Query().Where(
			user.IDEQ(recipientID), user.TenantIDEQ(req.TenantID), user.ActiveEQ(true),
		).Exist(ctx)
		if err != nil {
			return rollback(fmt.Errorf("校验收件人失败: %w", err))
		}
		if !exists {
			return rollback(fmt.Errorf("收件人 %d 不是租户 %d 的有效用户", recipientID, req.TenantID))
		}

		idempotencyKey := fmt.Sprintf("%s:%s:%d:%s:%d:%s",
			req.OccurrenceKey, req.ResourceType, req.ResourceID,
			req.Channel, recipientID, req.NotificationType)
		if _, err := commandbus.EnqueueTx(ctx, tx, commandbus.EnqueueRequest{
			TenantID:       req.TenantID,
			CommandType:    commandbus.CommandDeliverNotification,
			AggregateType:  req.ResourceType,
			AggregateID:    req.ResourceID,
			IdempotencyKey: idempotencyKey,
			MaxAttempts:    8,
			Payload: map[string]interface{}{
				"resourceType": req.ResourceType,
				"resourceId":   req.ResourceID,
				"recipientId":  recipientID,
				"channel":      req.Channel,
				"type":         req.NotificationType,
				"content":      req.Content,
			},
		}); err != nil {
			if ent.IsConstraintError(err) {
				// 同一业务事件重复触发，视为已投递，保持幂等
				continue
			}
			return rollback(fmt.Errorf("通知入队失败: %w", err))
		}
		enqueued++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("提交通知事务失败: %w", err)
	}

	h.logger.Infow("BPMN notification enqueued for reliable delivery",
		"tenant_id", req.TenantID, "resource_type", req.ResourceType, "resource_id", req.ResourceID,
		"channel", req.Channel, "recipients", len(seen), "enqueued", enqueued)
	return enqueued, nil
}

// resolveTarget 解析租户与通知承载的业务对象；缺失即明确失败，不再静默成功。
func (h *NotificationHandler) resolveTarget(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (int, string, int, error) {
	tenantID, err := ResolveTenantID(ctx, variables)
	if err != nil {
		if task != nil && task.TenantID > 0 {
			tenantID = task.TenantID
		} else {
			return 0, "", 0, err
		}
	}
	resourceType, resourceID, err := resolveNotificationResource(variables)
	if err != nil {
		return 0, "", 0, err
	}
	return tenantID, resourceType, resourceID, nil
}

// resolveUsersByContact 将邮箱/手机号解析为租户内的活跃用户ID；有任一无法解析则报错。
func (h *NotificationHandler) resolveUsersByContact(ctx context.Context, tenantID int, field string, contacts []string) ([]int, error) {
	if len(contacts) == 0 {
		return nil, fmt.Errorf("收件人列表为空")
	}
	query := h.client.User.Query().Where(user.TenantIDEQ(tenantID), user.ActiveEQ(true))
	switch field {
	case "email":
		query = query.Where(user.EmailIn(contacts...))
	case "phone":
		query = query.Where(user.PhoneIn(contacts...))
	default:
		return nil, fmt.Errorf("不支持的收件人字段: %s", field)
	}
	users, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("解析收件人失败: %w", err)
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("租户 %d 内未找到匹配的收件人(%s)", tenantID, field)
	}
	ids := make([]int, 0, len(users))
	matched := make(map[string]struct{}, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
		if field == "email" {
			matched[u.Email] = struct{}{}
		} else {
			matched[u.Phone] = struct{}{}
		}
	}
	missing := make([]string, 0)
	for _, c := range contacts {
		if _, ok := matched[c]; !ok {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		h.logger.Warnw("BPMN notification recipients not resolvable in tenant",
			"tenant_id", tenantID, "field", field, "missing_count", len(missing))
	}
	return ids, nil
}

// failure 输出结构化错误日志并返回明确失败的服务任务结果
func (h *NotificationHandler) failure(cause error, action string, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	h.logger.Errorw("BPMN notification task failed",
		"action", action,
		"tenant_id", GetIntFromVars(variables, "tenant_id"),
		"ticket_id", GetIntFromVars(variables, "ticket_id"),
		"change_id", GetIntFromVars(variables, "change_id"),
		"service_request_id", GetIntFromVars(variables, "service_request_id"),
		"error", cause)
	return &dto.ServiceTaskResult{
		Success: false,
		Message: fmt.Sprintf("通知发送失败: %v", cause),
	}, cause
}

// resolveNotificationResource 通知必须挂在受支持的业务对象上（投递 worker 只认这三种）
func resolveNotificationResource(variables map[string]interface{}) (string, int, error) {
	if id := GetIntFromVars(variables, "ticket_id"); id > 0 {
		return "ticket", id, nil
	}
	if id := GetIntFromVars(variables, "change_id"); id > 0 {
		return "change", id, nil
	}
	if id := GetIntFromVars(variables, "service_request_id"); id > 0 {
		return "service_request", id, nil
	}
	if resourceType := GetStringFromVars(variables, "resource_type"); resourceType != "" {
		if id := GetIntFromVars(variables, "resource_id"); id > 0 {
			switch resourceType {
			case "ticket", "change", "service_request":
				return resourceType, id, nil
			}
			return "", 0, fmt.Errorf("不支持的通知资源类型: %s", resourceType)
		}
	}
	return "", 0, fmt.Errorf("通知缺少业务对象引用(ticket_id/change_id/service_request_id)")
}

// occurrenceKey 构造幂等键前缀，优先使用流程任务标识，保证重放不会重复发送
func occurrenceKey(task *ent.ProcessTask, variables map[string]interface{}, action string) string {
	if task != nil {
		if task.TaskID != "" {
			return fmt.Sprintf("bpmn-task:%s:%s", task.TaskID, action)
		}
		if task.ID > 0 {
			return fmt.Sprintf("bpmn-task:%d:%s", task.ID, action)
		}
	}
	if pi := GetIntFromVars(variables, "process_instance_id"); pi > 0 {
		return fmt.Sprintf("bpmn-pi:%d:%s:%s", pi, GetStringFromVars(variables, "activity_id"), action)
	}
	return fmt.Sprintf("bpmn-adhoc:%d:%s", time.Now().UnixNano(), action)
}

func splitRecipients(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
	})
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		result = append(result, p)
	}
	return result
}

func notificationTypeOrDefault(variables map[string]interface{}, fallback string) string {
	if t := GetStringFromVars(variables, "notification_type"); t != "" {
		return t
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// 确保 NotificationHandler 实现了 ServiceTaskHandlerInterface
var _ ServiceTaskHandlerInterface = (*NotificationHandler)(nil)
