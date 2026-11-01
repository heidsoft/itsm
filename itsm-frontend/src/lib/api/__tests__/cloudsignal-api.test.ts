import { CloudSignalApi } from '@/lib/api/cloudsignal-api';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;

describe('CloudSignalApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('reads alert list from the cloud-signals route with camelCase params', async () => {
    mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20, totalPages: 0 });

    await CloudSignalApi.listAlerts({
      status: ['firing', 'acknowledged'],
      severity: ['P1'],
      cloudAccountId: 9,
      search: 'cpu',
      page: 2,
      pageSize: 20,
    });

    expect(mockGet).toHaveBeenCalledWith('/api/v1/cloud-signals/alerts', {
      status: ['firing', 'acknowledged'],
      severity: ['P1'],
      cloudAccountId: 9,
      search: 'cpu',
      page: 2,
      pageSize: 20,
    });
  });

  it('reads vulnerability list with instance and type filters', async () => {
    mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20, totalPages: 0 });

    await CloudSignalApi.listFindings({
      status: ['open'],
      severity: ['critical'],
      vulnType: 'cve',
      instanceId: 'i-bp1',
      cloudAccountId: 3,
      page: 1,
      pageSize: 50,
    });

    expect(mockGet).toHaveBeenCalledWith('/api/v1/cloud-signals/vulnerabilities', {
      status: ['open'],
      severity: ['critical'],
      vulnType: 'cve',
      instanceId: 'i-bp1',
      cloudAccountId: 3,
      page: 1,
      pageSize: 50,
    });
  });

  it('reads the unpatched vulnerability summary', async () => {
    mockGet.mockResolvedValue({ total: 4, bySeverity: {}, byStatus: {} });

    await CloudSignalApi.getFindingSummary();

    expect(mockGet).toHaveBeenCalledWith('/api/v1/cloud-signals/vulnerabilities/summary');
  });

  it('reads sync status from the list envelope route', async () => {
    mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 0, totalPages: 0 });

    await CloudSignalApi.getSyncStatus();

    expect(mockGet).toHaveBeenCalledWith('/api/v1/cloud-signals/sync-status');
  });

  it.each([
    ['acknowledgeAlert', 'acknowledge'],
    ['resolveAlert', 'resolve'],
  ] as const)('%s posts to the alert action route', async (method, action) => {
    mockPost.mockResolvedValue({ id: 7 });

    await CloudSignalApi[method](7);

    expect(mockPost).toHaveBeenCalledWith(`/api/v1/cloud-signals/alerts/7/${action}`);
  });

  it.each([
    ['ignoreFinding', 'ignore'],
    ['reopenFinding', 'reopen'],
  ] as const)('%s posts to the vulnerability action route', async (method, action) => {
    mockPost.mockResolvedValue({ id: 11 });

    await CloudSignalApi[method](11);

    expect(mockPost).toHaveBeenCalledWith(`/api/v1/cloud-signals/vulnerabilities/11/${action}`);
  });
});
