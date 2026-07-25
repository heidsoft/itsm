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
    it('should get user service requests', async () => {
      mockSuccessResponse({ requests: [{ id: 1 }], total: 1, page: 1, size: 10 });
      const result = await serviceRequestAPI.getUserServiceRequests({ page: 1, size: 10 });
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/service-requests/me'),
        expect.any(Object)
      );
      expect(result.requests).toHaveLength(1);
    });
  });

  describe('getPendingApprovals', () => {
    it('should get pending approvals', async () => {
      mockSuccessResponse({ requests: [], total: 0, page: 1, size: 10 });
      const result = await serviceRequestAPI.getPendingApprovals({ page: 1 });
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/service-requests/approvals/pending'),
        expect.any(Object)
      );
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
