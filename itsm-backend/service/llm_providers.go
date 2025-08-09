package service

import (
	"context"
)

// OpenAI-like provider stub (to be implemented with real SDK)
type OpenAIProvider struct{ apiKey, endpoint string }

func NewOpenAIProvider(apiKey, endpoint string) *OpenAIProvider {
	return &OpenAIProvider{apiKey: apiKey, endpoint: endpoint}
}
func (p *OpenAIProvider) Chat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
	// TODO: integrate real OpenAI/Azure SDKs
	return "(openai) not-implemented stub", nil
}

// Azure OpenAI stub
type AzureProvider struct{ apiKey, endpoint string }

func NewAzureProvider(apiKey, endpoint string) *AzureProvider {
	return &AzureProvider{apiKey: apiKey, endpoint: endpoint}
}
func (p *AzureProvider) Chat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
	return "(azure) not-implemented stub", nil
}

// Local model stub
type LocalProvider struct{}

func NewLocalProvider() *LocalProvider { return &LocalProvider{} }
func (p *LocalProvider) Chat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
	return "(local) not-implemented stub", nil
}
