import { BPMNAIApi } from '@/lib/api/bpmn-ai-api';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    getTenantId: jest.fn().mockReturnValue(42),
  },
}));

describe('BPMNAIApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('generates BPMN XML through the backend AI endpoint', async () => {
    const mockResponse = {
      bpmnXml: '<xml />',
      processId: 'incident_flow',
      processName: 'Incident Flow',
      processDescription: 'Generated flow',
      version: '1.0.0',
      nodeCount: 6,
      complexity: 'medium',
      explanation: 'AI generated',
    };
    (httpClient.post as jest.Mock).mockResolvedValueOnce(mockResponse);

    const result = await BPMNAIApi.generateBPMN({
      requirement: 'Generate an incident management workflow',
      processType: 'incident',
      enterpriseType: 'cn_enterprise',
      includeSla: true,
      includeNotifications: true,
      includeApprovals: false,
    });

    expect(httpClient.post).toHaveBeenCalledWith('/api/v1/bpmn/ai/generate', {
      requirement: 'Generate an incident management workflow',
      processType: 'incident',
      enterpriseType: 'cn_enterprise',
      includeSla: true,
      includeNotifications: true,
      includeApprovals: false,
      tenantId: 42,
    });
    expect(result).toBe(mockResponse);
  });

  it('previews BPMN structure through the backend AI endpoint', async () => {
    (httpClient.post as jest.Mock).mockResolvedValueOnce({
      structureDescription: 'A simple flow',
      nodes: [],
      complexity: 'low',
      estimatedNodeCount: 3,
      useCases: 'Service request',
      suggestions: [],
    });

    await BPMNAIApi.previewBPMN({
      requirement: 'Preview service request workflow',
      processType: 'service_request',
      enterpriseType: 'cn_enterprise',
    });

    expect(httpClient.post).toHaveBeenCalledWith('/api/v1/bpmn/ai/preview', {
      requirement: 'Preview service request workflow',
      processType: 'service_request',
      enterpriseType: 'cn_enterprise',
    });
  });

  it('loads template suggestions from the backend AI endpoint', async () => {
    (httpClient.get as jest.Mock).mockResolvedValueOnce([]);

    await BPMNAIApi.getTemplateSuggestions({
      keyword: 'change approval',
      processType: 'change',
    });

    expect(httpClient.get).toHaveBeenCalledWith('/api/v1/bpmn/ai/templates/suggestions', {
      keyword: 'change approval',
      processType: 'change',
    });
  });
});
