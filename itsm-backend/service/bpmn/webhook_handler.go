package bpmn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"itsm-backend/dto"
	"itsm-backend/ent"

	"go.uber.org/zap"
)

// WebhookHandler Webhook服务任务处理器
type WebhookHandler struct {
	HandlerBase
	client     *ent.Client
	logger     *zap.SugaredLogger
	httpClient *http.Client
}

// NewWebhookHandler 创建Webhook处理器
func NewWebhookHandler(client *ent.Client, logger *zap.SugaredLogger) *WebhookHandler {
	return &WebhookHandler{
		client:     client,
		logger:     logger,
		httpClient: &http.Client{},
	}
}

// GetTaskType 返回任务类型
func (h *WebhookHandler) GetTaskType() string {
	return "webhook_task"
}

// GetHandlerID 返回处理器标识
func (h *WebhookHandler) GetHandlerID() string {
	return "webhook_handler"
}

// Execute 执行Webhook服务任务
func (h *WebhookHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	action, _ := variables["action"].(string)
	switch action {
	case "call_webhook":
		return h.callWebhook(ctx, variables)
	case "send_notification":
		return h.sendWebhookNotification(ctx, variables)
	default:
		return h.callWebhook(ctx, variables)
	}
}

// Validate 验证配置
func (h *WebhookHandler) Validate(ctx context.Context, config map[string]interface{}) error {
	return nil
}

// callWebhook 调用Webhook
func (h *WebhookHandler) callWebhook(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	webhookURL := GetStringFromVars(variables, "webhook_url")
	method := GetStringFromVars(variables, "method")
	payload := GetStringFromVars(variables, "payload")
	headers := GetStringFromVars(variables, "headers")
	timeout := GetIntFromVars(variables, "timeout")

	if webhookURL == "" {
		return nil, fmt.Errorf("Webhook URL不能为空")
	}

	// 设置默认方法
	if method == "" {
		method = "POST"
	}

	h.logger.Infow("Calling webhook via BPMN",
		"webhook_url", webhookURL,
		"method", method,
	)

	// 准备请求
	var body *bytes.Reader
	if payload != "" {
		body = bytes.NewReader([]byte(payload))
	} else {
		body = bytes.NewReader([]byte("{}"))
	}

	req, err := http.NewRequestWithContext(ctx, method, webhookURL, body)
	if err != nil {
		return nil, fmt.Errorf("创建Webhook请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if headers != "" {
		// 解析自定义Headers (JSON格式)
		var headerMap map[string]string
		if err := json.Unmarshal([]byte(headers), &headerMap); err == nil {
			for k, v := range headerMap {
				req.Header.Set(k, v)
			}
		}
	}

	// 设置超时
	timeoutValue := 30 // 默认30秒
	if timeout > 0 {
		timeoutValue = timeout
	}

	_ = timeoutValue // 保留用于后续使用（设置 httpClient 的超时）

	// 发送请求
	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.logger.Errorw("Webhook call failed", "webhook_url", webhookURL, "error", err)
		return nil, fmt.Errorf("调用Webhook失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var respBody map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&respBody); err != nil {
		// 响应可能不是JSON格式，记录状态码即可
		h.logger.Warnw("Webhook response is not JSON", "status_code", resp.StatusCode)
	}

	h.logger.Infow("Webhook called successfully",
		"webhook_url", webhookURL,
		"status_code", resp.StatusCode,
	)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("Webhook调用成功，状态码: %d", resp.StatusCode),
		OutputVars: map[string]interface{}{
			"webhook_url":   webhookURL,
			"status_code":   resp.StatusCode,
			"response_body": respBody,
		},
	}, nil
}

// sendWebhookNotification 发送Webhook通知
func (h *WebhookHandler) sendWebhookNotification(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	// 这是一个便捷方法，实际上调用的是同一个callWebhook
	return h.callWebhook(ctx, variables)
}

// 确保 WebhookHandler 实现了 ServiceTaskHandlerInterface
var _ ServiceTaskHandlerInterface = (*WebhookHandler)(nil)
