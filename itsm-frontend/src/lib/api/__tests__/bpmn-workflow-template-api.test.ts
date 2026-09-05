import { BPMNWorkflowTemplateApi } from '@/lib/api/bpmn-workflow-template-api';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn() },
}));

describe('BPMNWorkflowTemplateApi', () => {
  beforeEach(() => jest.clearAllMocks());

  it('lists tenant-scoped templates through the canonical endpoint', async () => {
    const response = { items: [], total: 0, page: 1, pageSize: 20, totalPages: 0 };
    (httpClient.get as jest.Mock).mockResolvedValueOnce(response);

    await BPMNWorkflowTemplateApi.list({ status: 'draft', page: 1, pageSize: 20 });

    expect(httpClient.get).toHaveBeenCalledWith('/api/v1/bpmn/ai/templates', {
      status: 'draft', page: 1, pageSize: 20,
    });
  });

  it('publishes and archives a specific template key', async () => {
    (httpClient.post as jest.Mock).mockResolvedValue({});

    await BPMNWorkflowTemplateApi.publish('expense/approval');
    await BPMNWorkflowTemplateApi.archive('expense/approval');

    expect(httpClient.post).toHaveBeenNthCalledWith(1, '/api/v1/bpmn/ai/templates/expense%2Fapproval/publish');
    expect(httpClient.post).toHaveBeenNthCalledWith(2, '/api/v1/bpmn/ai/templates/expense%2Fapproval/archive');
  });
});
