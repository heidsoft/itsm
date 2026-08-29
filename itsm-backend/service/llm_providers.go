package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
)

// OpenAIProvider implements LLMProvider using OpenAI API
type OpenAIProvider struct {
	client    *openai.Client
	model     string
	maxTokens int
}

func NewOpenAIProvider(apiKey, endpoint, model string) *OpenAIProvider {
	config := openai.DefaultConfig(apiKey)
	if endpoint != "" {
		config.BaseURL = endpoint
	}
	client := openai.NewClientWithConfig(config)
	return &OpenAIProvider{
		client:    client,
		model:     model,
		maxTokens: 4096,
	}
}

// toOpenAIMessages 把内部 LLMMessage 映射为 OpenAI 消息，并收集声明的工具定义。
// 支持：Tools（请求侧工具声明）、ToolCalls（assistant 工具调用）、ToolCallID（tool 结果消息）。
func toOpenAIMessages(messages []LLMMessage) ([]openai.ChatCompletionMessage, []openai.Tool) {
	msgs := make([]openai.ChatCompletionMessage, 0, len(messages))
	var tools []openai.Tool
	for _, m := range messages {
		cm := openai.ChatCompletionMessage{Role: m.Role, Content: m.Content}
		if m.ToolCallID != "" {
			cm.ToolCallID = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			cm.ToolCalls = make([]openai.ToolCall, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				cm.ToolCalls = append(cm.ToolCalls, openai.ToolCall{
					Type:     openai.ToolTypeFunction,
					ID:       tc.ID,
					Function: openai.FunctionCall{Name: tc.Name, Arguments: tc.Arguments},
				})
			}
		}
		if len(m.Tools) > 0 {
			for _, td := range m.Tools {
				tools = append(tools, openai.Tool{
					Type: openai.ToolTypeFunction,
					Function: &openai.FunctionDefinition{
						Name:        td.Name,
						Description: td.Description,
						Parameters:  td.Parameters,
					},
				})
			}
		}
		msgs = append(msgs, cm)
	}
	return msgs, tools
}

func (p *OpenAIProvider) Chat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
	if model != "" {
		p.model = model
	}

	msgs, tools := toOpenAIMessages(messages)
	req := openai.ChatCompletionRequest{
		Model:       p.model,
		Messages:    msgs,
		MaxTokens:   p.maxTokens,
		Temperature: 0.3,
	}
	if len(tools) > 0 {
		req.Tools = tools
	}

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

// ChatStream streams tokens from OpenAI. It satisfies StreamingLLMProvider so
// that LLMGateway.ChatStream can deliver real-time deltas to the client.
func (p *OpenAIProvider) ChatStream(ctx context.Context, model string, messages []LLMMessage, callback func(string)) error {
	if model != "" {
		p.model = model
	}
	if callback == nil {
		callback = func(string) {}
	}

	msgs, tools := toOpenAIMessages(messages)
	req := openai.ChatCompletionRequest{
		Model:       p.model,
		Messages:    msgs,
		MaxTokens:   p.maxTokens,
		Temperature: 0.3,
	}
	if len(tools) > 0 {
		req.Tools = tools
	}

	stream, err := p.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return fmt.Errorf("OpenAI stream error: %w", err)
	}
	defer stream.Close()

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return nil
		}
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta.Content
			if delta != "" {
				callback(delta)
			}
		}
	}
}

// ChatStreamWithTools 声明工具并流式调用（ToolCallingStreamProvider）。
// 文本增量经 callback 下发；模型发起的工具调用在流结束后通过 onToolCalls 一次性返回。
// 若流中途断开，尽力返回已累积的工具调用。
func (p *OpenAIProvider) ChatStreamWithTools(ctx context.Context, model string, messages []LLMMessage, tools []LLMTool, callback func(string), onToolCalls func([]LLMToolCall)) error {
	if model != "" {
		p.model = model
	}
	if callback == nil {
		callback = func(string) {}
	}
	if onToolCalls == nil {
		onToolCalls = func([]LLMToolCall) {}
	}

	msgs, declared := toOpenAIMessages(messages)
	req := openai.ChatCompletionRequest{
		Model:       p.model,
		Messages:    msgs,
		MaxTokens:   p.maxTokens,
		Temperature: 0.3,
	}
	// 合并网关传入的工具声明
	for _, td := range tools {
		declared = append(declared, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        td.Name,
				Description: td.Description,
				Parameters:  td.Parameters,
			},
		})
	}
	if len(declared) > 0 {
		req.Tools = declared
	}

	stream, err := p.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return fmt.Errorf("OpenAI stream error: %w", err)
	}
	defer stream.Close()

	type toolCallAcc struct {
		id   string
		name string
		args strings.Builder
	}
	calls := map[int]*toolCallAcc{}
	var order []int

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			// 流中断：尽力返回已累积的工具调用
			break
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta
		if delta.Content != "" {
			callback(delta.Content)
		}
		for _, tc := range delta.ToolCalls {
			idx := 0
			if tc.Index != nil {
				idx = *tc.Index
			}
			acc := calls[idx]
			if acc == nil {
				acc = &toolCallAcc{}
				calls[idx] = acc
				order = append(order, idx)
			}
			if tc.ID != "" {
				acc.id = tc.ID
			}
			if tc.Function.Name != "" {
				acc.name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				acc.args.WriteString(tc.Function.Arguments)
			}
		}
	}

	if len(order) > 0 {
		result := make([]LLMToolCall, 0, len(order))
		for _, idx := range order {
			acc := calls[idx]
			result = append(result, LLMToolCall{
				ID:        acc.id,
				Name:      acc.name,
				Arguments: acc.args.String(),
			})
		}
		onToolCalls(result)
	}
	return nil
}

// StreamingOpenAIProvider supports streaming responses
type StreamingOpenAIProvider struct {
	client    *openai.Client
	model     string
	maxTokens int
}

func NewStreamingOpenAIProvider(apiKey, endpoint, model string) *StreamingOpenAIProvider {
	config := openai.DefaultConfig(apiKey)
	if endpoint != "" {
		config.BaseURL = endpoint
	}
	client := openai.NewClientWithConfig(config)
	return &StreamingOpenAIProvider{
		client:    client,
		model:     model,
		maxTokens: 4096,
	}
}

func (p *StreamingOpenAIProvider) ChatStream(ctx context.Context, model string, messages []LLMMessage, callback func(string)) error {
	if model != "" {
		p.model = model
	}

	msgs := make([]openai.ChatCompletionMessage, len(messages))
	for i, m := range messages {
		msgs[i] = openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	stream, err := p.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:       p.model,
		Messages:    msgs,
		MaxTokens:   p.maxTokens,
		Temperature: 0.3,
	})
	if err != nil {
		return fmt.Errorf("OpenAI stream error: %w", err)
	}
	defer stream.Close()

	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}
		if len(chunk.Choices) > 0 {
			callback(chunk.Choices[0].Delta.Content)
		}
	}
	return nil
}

// AzureProvider implements LLMProvider using Azure OpenAI Service
type AzureProvider struct {
	client       *openai.Client
	deploymentID string
	apiVersion   string
}

func NewAzureProvider(apiKey, endpoint, deploymentID string) *AzureProvider {
	config := openai.DefaultConfig(apiKey)
	// Azure endpoint format: https://{resource-name}.openai.azure.com/
	if endpoint != "" {
		config.BaseURL = fmt.Sprintf("%s/openai/v1", endpoint)
	}
	client := openai.NewClientWithConfig(config)
	return &AzureProvider{
		client:       client,
		deploymentID: deploymentID,
		apiVersion:   "2024-02-15-preview",
	}
}

func (p *AzureProvider) Chat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
	actualModel := p.deploymentID
	if model != "" {
		actualModel = model
	}

	msgs := make([]openai.ChatCompletionMessage, len(messages))
	for i, m := range messages {
		msgs[i] = openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       actualModel,
		Messages:    msgs,
		MaxTokens:   4096,
		Temperature: 0.3,
	})
	if err != nil {
		return "", fmt.Errorf("azure OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from Azure OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

// LocalProvider implements LLMProvider using local models (Ollama, LM Studio, etc.)
type LocalProvider struct {
	baseURL    string
	client     *http.Client
	stopTokens []string
}

func NewLocalProvider(baseURL string) *LocalProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &LocalProvider{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
		stopTokens: []string{"</s>", "\nUser:", "\nHuman:"},
	}
}

// MiniMaxProvider implements LLMProvider using MiniMax Anthropic-compatible API
type MiniMaxProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewMiniMaxProvider(apiKey, model string) *MiniMaxProvider {
	return &MiniMaxProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.minimaxi.com/anthropic/v1",
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// MiniMax Anthropic-format request
type MiniMaxAnthropicRequest struct {
	Model       string                    `json:"model"`
	MaxTokens   int                       `json:"maxTokens"`
	System      string                    `json:"system,omitempty"`
	Temperature float64                   `json:"temperature,omitempty"`
	Messages    []MiniMaxAnthropicMessage `json:"messages"`
}

// MiniMax Anthropic message format
type MiniMaxAnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MiniMax Anthropic response
type MiniMaxAnthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model      string `json:"model"`
	StopReason string `json:"stopReason"`
}

func (p *MiniMaxProvider) Chat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
	if model != "" {
		p.model = model
	}

	// Convert LLMMessage format to Anthropic format
	var systemPrompt string
	var anthropicMessages []MiniMaxAnthropicMessage

	for _, m := range messages {
		if m.Role == "system" {
			systemPrompt = m.Content
		} else {
			anthropicMessages = append(anthropicMessages, MiniMaxAnthropicMessage{Role: m.Role, Content: m.Content})
		}
	}

	reqBody := MiniMaxAnthropicRequest{
		Model:       p.model,
		MaxTokens:   4096,
		System:      systemPrompt,
		Temperature: 1.0,
		Messages:    anthropicMessages,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("MiniMax: failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/messages", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("MiniMax: failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("MiniMax API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errBody)
		errMsg := ""
		if em, ok := errBody["message"].(string); ok {
			errMsg = em
		}
		return "", fmt.Errorf("MiniMax API error: status %d, message: %s", resp.StatusCode, errMsg)
	}

	var anthropicResp MiniMaxAnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", fmt.Errorf("MiniMax: failed to decode response: %w", err)
	}

	// Extract text content from response
	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}

	return "", fmt.Errorf("MiniMax: no text content in response")
}

// OllamaChatRequest for Ollama-compatible APIs
type OllamaChatRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	Stream  bool `json:"stream"`
	Options struct {
		Temperature float64 `json:"temperature"`
		TopK        int     `json:"topK"`
		TopP        float64 `json:"topP"`
	} `json:"options,omitempty"`
}

// OllamaChatResponse for Ollama-compatible APIs
type OllamaChatResponse struct {
	Response      string   `json:"response"`
	Done          bool     `json:"done"`
	TotalDuration int64    `json:"totalDuration,omitempty"`
	EvalCount     int      `json:"evalCount,omitempty"`
	StopReasons   []string `json:"stopReason,omitempty"`
}

func (p *LocalProvider) Chat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
	req := OllamaChatRequest{
		Model:  model,
		Stream: false,
	}
	for _, m := range messages {
		req.Messages = append(req.Messages, struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("local LLM request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("local LLM returned status %d", resp.StatusCode)
	}

	var result OllamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Response, nil
}

// ProviderConfig holds LLM configuration
type ProviderConfig struct {
	Provider   string
	Model      string
	APIKey     string
	Endpoint   string
	Deployment string
	TokenCap   int
}

// LoadLLMConfig loads LLM configuration from viper
func LoadLLMConfig() ProviderConfig {
	return ProviderConfig{
		Provider:   viper.GetString("llm.provider"),
		Model:      viper.GetString("llm.model"),
		APIKey:     viper.GetString("llm.api_key"),
		Endpoint:   viper.GetString("llm.endpoint"),
		Deployment: viper.GetString("llm.deployment"),
		TokenCap:   viper.GetInt("llm.token_cap"),
	}
}

// NewProviderFromConfig creates the appropriate provider from config
func NewProviderFromConfig(cfg ProviderConfig) LLMProvider {
	apiKey := cfg.APIKey
	// Support environment variable override
	if envKey := os.Getenv("OPENAI_API_KEY"); envKey != "" && apiKey == "" {
		apiKey = envKey
	}
	if envKey := os.Getenv("AZURE_OPENAI_API_KEY"); envKey != "" && apiKey == "" {
		apiKey = envKey
	}
	if envKey := os.Getenv("MINIMAX_API_KEY"); envKey != "" && apiKey == "" {
		apiKey = envKey
	}

	switch cfg.Provider {
	case "openai":
		return NewOpenAIProvider(apiKey, cfg.Endpoint, cfg.Model)
	case "azure":
		return NewAzureProvider(apiKey, cfg.Endpoint, cfg.Deployment)
	case "local":
		return NewLocalProvider(cfg.Endpoint)
	case "minimax":
		return NewMiniMaxProvider(apiKey, cfg.Model)
	default:
		// Default to OpenAI
		return NewOpenAIProvider(apiKey, cfg.Endpoint, cfg.Model)
	}
}
