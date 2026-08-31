import { httpClient } from '@/lib/api/http-client';
import { TicketWorkflowStateApi } from '@/lib/api/ticket-workflow-state-api';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn() },
}));

const mockGet = httpClient.get as jest.Mock;

describe('TicketWorkflowStateApi', () => {
  beforeEach(() => {
    mockGet.mockReset();
  });

  it('unwraps bpmnProcessState from the workflow-state DTO', async () => {
    const bpmnProcessState = {
      processInstanceId: 'process-1',
      processDefinitionKey: 'ticket-approval',
      processDefinitionName: '工单审批',
      bpmnStatus: 'running',
    };
    mockGet.mockResolvedValue({
      ticketId: 42,
      currentStatus: 'in_progress',
      availableActions: [],
      bpmnProcessState,
    });

    await expect(TicketWorkflowStateApi.getStateV2(42)).resolves.toEqual(bpmnProcessState);
    expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/42/workflow/state-v2');
  });

  it('returns null from the safe wrapper when BPMN state is unavailable', async () => {
    mockGet.mockResolvedValue({
      ticketId: 42,
      currentStatus: 'open',
      availableActions: [],
    });

    await expect(TicketWorkflowStateApi.tryGetStateV2(42)).resolves.toBeNull();
  });
});
