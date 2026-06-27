package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// GuidanceClient calls the Python Guidance Sidecar for constrained LLM generation
type GuidanceClient struct {
	baseURL string
	client  *http.Client
	logger  *zap.SugaredLogger
	timeout time.Duration
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
		logger:  logger,
		timeout: 120 * time.Second,
	}
}

// GuidanceTriageRequest matches the Python sidecar's TriageRequest
type GuidanceTriageRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	TenantID    int    `json:"tenant_id"`
}

// GuidanceTriageResponse matches the Python sidecar's TriageResponse
type GuidanceTriageResponse struct {
	Category     string  `json:"category"`
	Priority     string  `json:"priority"`
	Confidence   float64 `json:"confidence"`
	Explanation  string  `json:"explanation"`
	SuggestedFix string  `json:"suggested_fix,omitempty"`
	AssigneeID   int     `json:"assignee_id"`
	Method       string  `json:"method"`
	LatencyMs    float64 `json:"latency_ms"`
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

	return nil
}
