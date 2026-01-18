package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

func (p *OpenAIProvider) Chat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
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

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       p.model,
		Messages:    msgs,
		MaxTokens:   p.maxTokens,
		Temperature: 0.3,
	})
	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
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
		return "", fmt.Errorf("Azure OpenAI API error: %w", err)
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
	timeout    time.Duration
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

// OllamaChatRequest for Ollama-compatible APIs
type OllamaChatRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	Stream  bool   `json:"stream"`
	Options struct {
		Temperature float64 `json:"temperature"`
		TopK        int     `json:"top_k"`
		TopP        float64 `json:"top_p"`
	} `json:"options,omitempty"`
}

// OllamaChatResponse for Ollama-compatible APIs
type OllamaChatResponse struct {
	Response      string   `json:"response"`
	Done          bool     `json:"done"`
	TotalDuration int64    `json:"total_duration,omitempty"`
	EvalCount     int      `json:"eval_count,omitempty"`
	StopReasons   []string `json:"stop_reason,omitempty"`
}

func (p *LocalProvider) Chat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
	req := OllamaChatRequest{
		Model: model,
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
	Provider    string
	Model       string
	APIKey      string
	Endpoint    string
	Deployment  string
	TokenCap    int
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

	switch cfg.Provider {
	case "openai":
		return NewOpenAIProvider(apiKey, cfg.Endpoint, cfg.Model)
	case "azure":
		return NewAzureProvider(apiKey, cfg.Endpoint, cfg.Deployment)
	case "local":
		return NewLocalProvider(cfg.Endpoint)
	default:
		// Default to OpenAI
		return NewOpenAIProvider(apiKey, cfg.Endpoint, cfg.Model)
	}
}
