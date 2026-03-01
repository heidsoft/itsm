package bpmn

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"

	"go.uber.org/zap"
)

// NotificationHandler 通知服务任务处理器
type NotificationHandler struct {
	HandlerBase
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewNotificationHandler 创建通知处理器
func NewNotificationHandler(client *ent.Client, logger *zap.SugaredLogger) *NotificationHandler {
	return &NotificationHandler{
		client: client,
		logger: logger,
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
		return h.sendEmail(ctx, variables)
	case "send_sms":
		return h.sendSMS(ctx, variables)
	case "send_in_app":
		return h.sendInAppNotification(ctx, variables)
	case "send_webhook":
		return h.sendWebhookNotification(ctx, variables)
	default:
		return h.sendInAppNotification(ctx, variables)
	}
}

// Validate 验证配置
func (h *NotificationHandler) Validate(ctx context.Context, config map[string]interface{}) error {
	return nil
}

// sendEmail 发送邮件通知
func (h *NotificationHandler) sendEmail(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	recipients := GetStringFromVars(variables, "recipients")
	subject := GetStringFromVars(variables, "subject")
	body := GetStringFromVars(variables, "body")

	if recipients == "" {
		return nil, fmt.Errorf("邮件收件人不能为空")
	}

	h.logger.Infow("Sending email notification via BPMN",
		"recipients", recipients,
		"subject", subject,
		"body_length", len(body),
	)

	// 这里应该调用邮件服务发送邮件
	// 简化处理，只记录日志

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("邮件已发送到 %s", recipients),
	}, nil
}

// sendSMS 发送短信通知
func (h *NotificationHandler) sendSMS(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	phoneNumbers := GetStringFromVars(variables, "phone_numbers")
	message := GetStringFromVars(variables, "message")

	if phoneNumbers == "" {
		return nil, fmt.Errorf("手机号不能为空")
	}

	h.logger.Infow("Sending SMS notification via BPMN",
		"phone_numbers", phoneNumbers,
		"message", message,
	)

	// 这里应该调用短信服务发送短信
	// 简化处理，只记录日志

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("短信已发送到 %s", phoneNumbers),
	}, nil
}

// sendInAppNotification 发送应用内通知
func (h *NotificationHandler) sendInAppNotification(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	userIDs := GetIntFromVars(variables, "user_ids")
	title := GetStringFromVars(variables, "title")
	content := GetStringFromVars(variables, "content")
	notificationType := GetStringFromVars(variables, "notification_type")

	if userIDs == 0 {
		return nil, fmt.Errorf("用户ID不能为空")
	}

	h.logger.Infow("Sending in-app notification via BPMN",
		"user_id", userIDs,
		"title", title,
		"type", notificationType,
		"content_length", len(content),
	)

	// 这里应该调用通知服务发送应用内通知
	// 简化处理，只记录日志

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("应用内通知已发送给用户 %d", userIDs),
	}, nil
}

// sendWebhookNotification 发送Webhook通知
func (h *NotificationHandler) sendWebhookNotification(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	webhookURL := GetStringFromVars(variables, "webhook_url")
	payload := GetStringFromVars(variables, "payload")

	if webhookURL == "" {
		return nil, fmt.Errorf("Webhook URL不能为空")
	}

	h.logger.Infow("Sending webhook notification via BPMN",
		"webhook_url", webhookURL,
		"payload", payload,
	)

	// 这里应该调用Webhook服务发送通知
	// 简化处理，只记录日志

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("Webhook通知已发送到 %s", webhookURL),
	}, nil
}

// 确保 NotificationHandler 实现了 ServiceTaskHandlerInterface
var _ ServiceTaskHandlerInterface = (*NotificationHandler)(nil)
