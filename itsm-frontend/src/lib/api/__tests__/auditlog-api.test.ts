import { listAuditLogs } from '@/lib/api/auditlog-api';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;

describe('listAuditLogs', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  it('should query audit logs with all params', async () => {
    const expected = { logs: [], total: 0, page: 1, pageSize: 20 };
    mockGet.mockResolvedValue(expected);
    const res = await listAuditLogs({
      page: 1,
      pageSize: 20,
      userId: 5,
      resource: 'ticket',
      action: 'create',
      method: 'POST',
      statusCode: 200,
      path: '/api/v1/tickets',
      requestId: 'req-123',
      from: '2024-01-01T00:00:00Z',
      to: '2024-01-31T23:59:59Z',
    });
    expect(mockGet).toHaveBeenCalledWith(
      '/api/v1/audit-logs?page=1&pageSize=20&userId=5&resource=ticket&action=create&method=POST&statusCode=200&path=%2Fapi%2Fv1%2Ftickets&requestId=req-123&from=2024-01-01T00%3A00%3A00Z&to=2024-01-31T23%3A59%3A59Z'
    );
    expect(res).toEqual(expected);
  });

  it('should query with no params', async () => {
    const expected = { logs: [], total: 0, page: 1, pageSize: 20 };
    mockGet.mockResolvedValue(expected);
    const res = await listAuditLogs({});
    expect(mockGet).toHaveBeenCalledWith('/api/v1/audit-logs');
    expect(res).toEqual(expected);
  });

  it('should handle partial params', async () => {
    const expected = { logs: [{ id: 1 }], total: 1, page: 1, pageSize: 10 };
    mockGet.mockResolvedValue(expected);
    const res = await listAuditLogs({ page: 1, resource: 'user' });
    expect(mockGet).toHaveBeenCalledWith(expect.stringContaining('page=1'));
    expect(mockGet).toHaveBeenCalledWith(expect.stringContaining('resource=user'));
    expect(res).toEqual(expected);
  });
});
