package bpmn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

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
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, fmt.Errorf("invalid webhook address: %w", err)
			}
			ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
			if err != nil {
				return nil, fmt.Errorf("resolve webhook host: %w", err)
			}
			for _, ip := range ips {
				if isPrivateWebhookIP(ip) {
					return nil, fmt.Errorf("webhook target resolves to a private or reserved address")
				}
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("webhook host has no address")
			}
			return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		},
	}
	return &WebhookHandler{
		client: client,
		logger: logger,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many webhook redirects")
				}
				return validateWebhookURL(req.URL.String())
			},
		},
	}
}

func validateWebhookURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return fmt.Errorf("invalid webhook URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("webhook URL must use http or https")
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("webhook target is not allowed")
	}
	if ip, err := netip.ParseAddr(host); err == nil && isPrivateWebhookIP(ip) {
		return fmt.Errorf("webhook target is not allowed")
	}
	return nil
}

func isPrivateWebhookIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() ||
		!ip.IsGlobalUnicast()
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
		return nil, fmt.Errorf("webhook URL不能为空")
	}
	if err := validateWebhookURL(webhookURL); err != nil {
		return nil, err
	}

	// 设置默认方法
	if method == "" {
		method = "POST"
	}

	h.logger.Infow(
		"Calling webhook via BPMN",
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
	if timeoutValue > 120 {
		timeoutValue = 120
	}
	requestCtx, cancel := context.WithTimeout(req.Context(), time.Duration(timeoutValue)*time.Second)
	defer cancel()
	req = req.WithContext(requestCtx)

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

	h.logger.Infow(
		"Webhook called successfully",
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
