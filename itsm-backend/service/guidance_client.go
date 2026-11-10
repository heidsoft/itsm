package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
)

// GuidanceClient calls the Python Guidance Sidecar for constrained LLM generation
type GuidanceClient struct {
	baseURL string
	client  *http.Client
	logger  *zap.SugaredLogger
	timeout time.Duration
	// A4：服务间共享令牌。sidecar 配置 AI_SERVICE_TOKEN 后要求 X-Service-Token，
	// 未配置时该头为空，不发送（向后兼容）。令牌只从运行环境读取，不落配置文件。
	serviceToken string
}

// NewGuidanceClient creates a new Guidance sidecar client
func NewGuidanceClient(baseURL string, logger *zap.SugaredLogger) *GuidanceClient {
	if baseURL == "" {
		baseURL = "http://localhost:8091"
	}
	return &GuidanceClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		logger:       logger,
		timeout:      120 * time.Second,
		serviceToken: os.Getenv("AI_SERVICE_TOKEN"),
	}
}

// GuidanceTriageRequest matches the Python sidecar's TriageRequest
type GuidanceTriageRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	TenantID    int    `json:"tenantId"`
}

// GuidanceTriageResponse matches the Python sidecar's TriageResponse
type GuidanceTriageResponse struct {
	Category     string  `json:"category"`
	Priority     string  `json:"priority"`
	Confidence   float64 `json:"confidence"`
	Explanation  string  `json:"explanation"`
	SuggestedFix string  `json:"suggestedFix,omitempty"`
	AssigneeID   int     `json:"assigneeId"`
	Method       string  `json:"method"`
	LatencyMs    float64 `json:"latencyMs"`
	// Degraded 是 sidecar 自报的降级位：LLM 不可用时会返回关键词结果
	// （method="guidance-fallback"）。此前这个字段（以及 Method）定义在这里
	// 却零消费，降级结果被当成模型决策消费掉了。
	Degraded bool `json:"degraded"`
}

// Triage calls the Guidance sidecar for constrained JSON output
func (c *GuidanceClient) Triage(ctx context.Context, title, description string, tenantID int) (*GuidanceTriageResponse, error) {
	reqBody := GuidanceTriageRequest{
		Title:       title,
		Description: description,
		TenantID:    tenantID,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/triage", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.serviceToken != "" {
		req.Header.Set("X-Service-Token", c.serviceToken)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("guidance sidecar call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("guidance sidecar error: %v", errResp)
	}

	var result GuidanceTriageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// HealthCheck verifies the Guidance sidecar is available
func (c *GuidanceClient) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("guidance sidecar unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("guidance sidecar unhealthy: status %d", resp.StatusCode)
	}

	// A5：sidecar 的 /health 保持 200（否则 compose healthcheck 会触发
	// restart: always 的重启循环），把「凭据缺失」放在响应体里。
	// 这里必须真正读懂它，否则配置错误依旧被当成健康。
	// 用指针兼容旧版 sidecar（没有 llm.configured 字段时不误报）。
	var health struct {
		Status string `json:"status"`
		LLM    *struct {
			Configured *bool  `json:"configured"`
			Reason     string `json:"reason"`
		} `json:"llm"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return fmt.Errorf("guidance sidecar health response unreadable: %w", err)
	}
	if health.LLM != nil && health.LLM.Configured != nil && !*health.LLM.Configured {
		return fmt.Errorf("guidance sidecar LLM is not configured: %s", health.LLM.Reason)
	}
	if health.Status != "" && health.Status != "healthy" {
		return fmt.Errorf("guidance sidecar degraded: %s", health.Status)
	}

	return nil
}
