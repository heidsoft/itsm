import { MSPAPI } from '../msp-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;

describe('MSPAPI', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getAllocations', () => {
    it('should get allocations without params', async () => {
      mockGet.mockResolvedValue({ data: { items: [], total: 0 } });
      await MSPAPI.getAllocations();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/msp/allocations', undefined);
    });

    it('should get allocations with params', async () => {
      mockGet.mockResolvedValue({ data: { items: [], total: 0 } });
      await MSPAPI.getAllocations({ page: 1, pageSize: 10 } as any);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/msp/allocations', { page: 1, pageSize: 10 });
    });
  });

  describe('createAllocation', () => {
    it('should create allocation', async () => {
      const data = { mspUserId: 1, customerTenantId: 2 };
      mockPost.mockResolvedValue({ data: { id: 1 } });
      await MSPAPI.createAllocation(data as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/msp/allocations', data);
    });
  });

  describe('deallocate', () => {
    it('should deallocate with reason', async () => {
      mockPost.mockResolvedValue({ data: null });
      await MSPAPI.deallocate(1, 2, 'Contract ended');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/msp/allocations/deallocate', {
        mspUserId: 1, customerTenantId: 2, reason: 'Contract ended',
      });
    });

    it('should deallocate without reason', async () => {
      mockPost.mockResolvedValue({ data: null });
      await MSPAPI.deallocate(1, 2);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/msp/allocations/deallocate', {
        mspUserId: 1, customerTenantId: 2, reason: undefined,
      });
    });
  });

  describe('getCustomers', () => {
    it('should get customers', async () => {
      mockGet.mockResolvedValue({ data: { customers: [] } });
      await MSPAPI.getCustomers({ page: 1 } as any);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/msp/customers', { page: 1 });
    });
  });

  describe('getCustomerTickets', () => {
    it('should get customer tickets', async () => {
      mockGet.mockResolvedValue({ data: { tickets: [] } });
      await MSPAPI.getCustomerTickets(5, { status: 'open', page: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/msp/customers/5/tickets', { status: 'open', page: 1 });
    });

    it('should get customer tickets without params', async () => {
      mockGet.mockResolvedValue({ data: { tickets: [] } });
      await MSPAPI.getCustomerTickets(5);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/msp/customers/5/tickets', undefined);
    });
  });

  describe('assignTechnician', () => {
    it('should assign technician', async () => {
      mockPost.mockResolvedValue({ data: { id: 1, status: 'assigned' } });
      await MSPAPI.assignTechnician(10, 2, 3);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/msp/tickets/10/assign', {
        customerTenantId: 2, assignerUserId: 3,
      });
    });

    it('should assign technician without assigner', async () => {
      mockPost.mockResolvedValue({ data: { id: 1, status: 'assigned' } });
      await MSPAPI.assignTechnician(10, 2);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/msp/tickets/10/assign', {
        customerTenantId: 2, assignerUserId: undefined,
      });
    });
  });

  describe('getCustomerReports', () => {
    it('should get customer reports', async () => {
      mockGet.mockResolvedValue({ data: [] });
      await MSPAPI.getCustomerReports({ startDate: '2024-01-01', endDate: '2024-01-31' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/msp/reports/customers', { startDate: '2024-01-01', endDate: '2024-01-31' });
    });

    it('should get customer reports with customer filter', async () => {
      mockGet.mockResolvedValue({ data: [] });
      await MSPAPI.getCustomerReports({ startDate: '2024-01-01', endDate: '2024-01-31', customerTenantId: 5 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/msp/reports/customers', { startDate: '2024-01-01', endDate: '2024-01-31', customerTenantId: 5 });
    });
  });

  describe('getMSPPerformanceReports', () => {
    it('should get performance reports', async () => {
      mockGet.mockResolvedValue({ data: [] });
      await MSPAPI.getMSPPerformanceReports({ startDate: '2024-01-01', endDate: '2024-01-31' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/msp/reports/performance', { startDate: '2024-01-01', endDate: '2024-01-31' });
    });
  });

  describe('isMSPUser', () => {
    it('should return MSP status', async () => {
      mockGet.mockResolvedValue({ data: { isMsp: true, isAdmin: true } });
      const result = await MSPAPI.isMSPUser();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/msp/status');
      expect(result).toEqual({ isMSP: true, isAdmin: true });
    });

    it('should return false on error', async () => {
      mockGet.mockRejectedValue(new Error('Network error'));
      const result = await MSPAPI.isMSPUser();
      expect(result).toEqual({ isMSP: false, isAdmin: false });
    });

    it('should handle missing data', async () => {
      mockGet.mockResolvedValue({ data: null });
      const result = await MSPAPI.isMSPUser();
      expect(result).toEqual({ isMSP: false, isAdmin: false });
    });
  });

  describe('getMSPContext', () => {
    it('should get MSP context', async () => {
      mockGet.mockResolvedValue({ data: { tenantId: 1, role: 'manager' } });
      await MSPAPI.getMSPContext();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/msp/context');
    });
  });

  describe('getAllocationHistory', () => {
    it('should get allocation history', async () => {
      mockGet.mockResolvedValue({ data: [] });
      await MSPAPI.getAllocationHistory({ mspUserId: 1, startDate: '2024-01-01', endDate: '2024-01-31' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/msp/allocations/history', { mspUserId: 1, startDate: '2024-01-01', endDate: '2024-01-31' });
    });
  });

  describe('error propagation', () => {
    it('should propagate errors from getAllocations', async () => {
      mockGet.mockRejectedValue(new Error('Server error'));
      await expect(MSPAPI.getAllocations()).rejects.toThrow('Server error');
    });
  });
});
