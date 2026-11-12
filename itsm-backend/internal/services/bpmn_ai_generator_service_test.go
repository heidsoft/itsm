package service

import (
	"context"
	"testing"

	"itsm-backend/dto"

	"github.com/stretchr/testify/require"
)

type recordingBPMNLLMProvider struct {
	model string
}

func (p *recordingBPMNLLMProvider) Chat(_ context.Context, model string, _ []LLMMessage) (string, error) {
	p.model = model
	return `{}`, nil
}

func TestBPMNAIGeneratorUsesConfiguredProviderModel(t *testing.T) {
	provider := &recordingBPMNLLMProvider{}
	gateway := NewLLMGateway(provider, nil, NoopObserver{}, "test")
	svc := NewBPMNAIGeneratorService(gateway, nil)

	_, err := svc.PreviewBPMN(context.Background(), &dto.PreviewBPMNRequest{
		Requirement:    "VPN access approval",
		ProcessType:    "service_request",
		EnterpriseType: "enterprise",
	})

	require.NoError(t, err)
	require.Empty(t, provider.model, "the provider must retain its deployment-configured model")
}

func TestBPMNTemplateServiceSuggestsBuiltInTemplates(t *testing.T) {
	svc := NewBPMNTemplateService(nil)

	results, err := svc.SuggestTemplates(context.Background(), 7, "紧急", "incident")

	require.NoError(t, err)
	require.NotEmpty(t, results)
	require.Equal(t, "incident_emergency_flow", results[0].ID)
	require.Equal(t, "incident", results[0].ProcessType)
}
