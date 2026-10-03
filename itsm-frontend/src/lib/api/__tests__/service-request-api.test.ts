import { serviceRequestAPI } from '@/lib/api/service-request-api';

jest.mock('@/lib/api/api-config', () => ({
  API_BASE_URL: 'http://localhost:8090',
}));

jest.mock('@/lib/auth/token-storage', () => ({
  getTenantCode: jest.fn().mockReturnValue('test-tenant'),
}));

// Mock global fetch
global.fetch = jest.fn();
const mockFetch = global.fetch as jest.Mock;

describe('ServiceRequestAPI', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  const mockSuccessResponse = (data: unknown) => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ code: 0, message: 'success', data }),
    });
  };

  describe('getUserServiceRequests', () => {
    it('解析五键信封并原样透传分页元数据', async () => {
      mockSuccessResponse({
        items: [{ id: 1, requestNumber: 'SR-202609-000001', status: 'submitted' }],
        total: 25,
        page: 2,
        pageSize: 10,
        totalPages: 3,
      });

      const result = await serviceRequestAPI.getUserServiceRequests({ page: 2, pageSize: 10 });

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8090/api/v1/service-requests/me?page=2&pageSize=10',
        expect.any(Object)
      );
      expect(result.items).toHaveLength(1);
      expect(result.total).toBe(25);
      expect(result.page).toBe(2);
      expect(result.pageSize).toBe(10);
      expect(result.totalPages).toBe(3);
    });

    it('只发送 page/pageSize，不再发送 size、userId 等后端不识别的别名', async () => {
      mockSuccessResponse({ items: [], total: 0, page: 1, pageSize: 20, totalPages: 0 });

      await serviceRequestAPI.getUserServiceRequests({ page: 1, status: 'delivered' });

      const [url] = mockFetch.mock.calls[0] as [string];
      expect(url).toBe('http://localhost:8090/api/v1/service-requests/me?page=1&status=delivered');
      expect(url).not.toContain('size=');
      expect(url).not.toContain('userId=');
    });
  });

  describe('getPendingApprovals', () => {
    it('待办收件箱同样按五键信封解析', async () => {
      mockSuccessResponse({
        items: [{ id: 7, requestNumber: 'SR-202609-000007', status: 'submitted' }],
        total: 1,
        page: 1,
        pageSize: 20,
        totalPages: 1,
      });

      const result = await serviceRequestAPI.getPendingApprovals({ page: 1 });

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8090/api/v1/service-requests/approvals/pending?page=1',
        expect.any(Object)
      );
      expect(result.items).toHaveLength(1);
      expect(result.totalPages).toBe(1);
    });
  });

  describe('getServiceRequestDetails', () => {
    it('should get service request details', async () => {
      mockSuccessResponse({ id: 1, catalogId: 2, requesterId: 3, status: 'submitted', version: 1, createdAt: '' });
      const result = await serviceRequestAPI.getServiceRequestDetails(1);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/service-requests/1'),
        expect.any(Object)
      );
      expect(result.id).toBe(1);
    });
  });

  describe('createServiceRequest', () => {
    it('should create a service request', async () => {
      const data = { catalogId: 1, complianceAck: true };
      mockSuccessResponse({ id: 10, ...data, status: 'submitted', version: 1, createdAt: '' });
      const result = await serviceRequestAPI.createServiceRequest(data);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/service-requests'),
        expect.objectContaining({ method: 'POST' })
      );
    });
  });

  describe('updateServiceRequestStatus', () => {
    it('should update status', async () => {
      mockSuccessResponse({ id: 1, status: 'delivered' });
      await serviceRequestAPI.updateServiceRequestStatus(1, 'delivered', 'Done');
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/service-requests/1/status'),
        expect.objectContaining({ method: 'PUT' })
      );
    });
  });

  describe('applyApprovalAction', () => {
    it('should apply approval action', async () => {
      mockSuccessResponse({ id: 1, status: 'security_approved' });
      await serviceRequestAPI.applyApprovalAction(1, { action: 'approve', comment: 'OK' });
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/service-requests/1/approvals'),
        expect.objectContaining({ method: 'POST' })
      );
    });
  });

  describe('startProvisioning', () => {
    it('should start provisioning', async () => {
      mockSuccessResponse({ task: { id: 1, status: 'pending' } });
      const result = await serviceRequestAPI.startProvisioning(1);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/service-requests/1/provision'),
        expect.objectContaining({ method: 'POST' })
      );
    });
  });

  describe('healthCheck', () => {
    it('should check health', async () => {
      mockSuccessResponse({ status: 'ok' });
      const result = await serviceRequestAPI.healthCheck();
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/health'),
        expect.any(Object)
      );
    });
  });
});
