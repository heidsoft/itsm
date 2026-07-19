import { SLAApi } from '../sla-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    post: jest.fn(),
  },
}));

describe('SLAApi monitoring contract', () => {
  it('uses the registered /sla/monitor compatibility endpoint', async () => {
    const post = httpClient.post as jest.Mock;
    post.mockResolvedValue({
      totalViolations: 10,
      resolvedViolations: 8,
      activeViolations: 2,
      complianceRate: 0.8,
      activeSlas: 3,
      activeAlertRules: 1,
    });

    const result = await SLAApi.getSLAMonitoring({ startTime: '30d', endTime: 'now' });

    expect(post).toHaveBeenCalledWith('/api/v1/sla/monitor', {
      startTime: '30d',
      endTime: 'now',
    });
    expect(result).toMatchObject({
      totalTickets: 10,
      violatedTickets: 2,
      compliantTickets: 8,
      atRiskTickets: 2,
      complianceRate: 80,
      violationRate: 20,
    });
  });
});
