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

// 节点命名规范必须写进生成与预览两套 prompt，否则 LLM 自由发挥会产出
// "AI分类/智能分类"、英文 key 直出、公文腔节点名（用户实测抱怨的来源）。
func TestBPMNAIGeneratorPromptsCarryNodeNamingRules(t *testing.T) {
	for name, prompt := range map[string]string{
		"generate": getSystemPrompt(),
		"preview":  getPreviewSystemPrompt(),
	} {
		t.Run(name, func(t *testing.T) {
			require.Contains(t, prompt, "节点命名要求", "必须含命名规范小节")
			require.Contains(t, prompt, "中文", "必须要求中文节点名")
			require.Contains(t, prompt, "禁止", "必须含禁止项")
			require.Contains(t, prompt, "智能", "必须明确禁止 AI/智能 字样")
			require.Contains(t, prompt, "工单分类", "必须给出自动化节点的业务命名示例")
		})
	}
}
