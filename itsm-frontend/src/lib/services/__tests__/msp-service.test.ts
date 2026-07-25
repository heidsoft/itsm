import { MSPService } from '../msp-service';
import { MSPAPI } from '@/lib/api/msp-api';

jest.mock('@/lib/api/msp-api', () => ({
  MSPAPI: {
    getAllocations: jest.fn(),
    createAllocation: jest.fn(),
    deallocate: jest.fn(),
    getCustomers: jest.fn(),
    getCustomerTickets: jest.fn(),
    assignTechnician: jest.fn(),
    getCustomerReports: jest.fn(),
    isMSPUser: jest.fn(),
    getMSPContext: jest.fn(),
    getAllocationHistory: jest.fn(),
  },
}));

const mockGetAllocations = MSPAPI.getAllocations as jest.Mock;
const mockCreateAllocation = MSPAPI.createAllocation as jest.Mock;
const mockDeallocate = MSPAPI.deallocate as jest.Mock;
const mockGetCustomers = MSPAPI.getCustomers as jest.Mock;
const mockGetCustomerTickets = MSPAPI.getCustomerTickets as jest.Mock;
const mockAssignTechnician = MSPAPI.assignTechnician as jest.Mock;
const mockGetCustomerReports = MSPAPI.getCustomerReports as jest.Mock;
const mockIsMSPUser = MSPAPI.isMSPUser as jest.Mock;
const mockGetMSPContext = MSPAPI.getMSPContext as jest.Mock;
const mockGetAllocationHistory = MSPAPI.getAllocationHistory as jest.Mock;

describe('MSPService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    MSPService.refreshCache();
  });

  describe('getAllocations', () => {
    it('should return allocations when data exists', async () => {
      const allocations = [{ id: 1, mspUserId: 10, customerTenantId: 20 }];
      mockGetAllocations.mockResolvedValue({ data: { allocations, total: 1 } });

      const result = await MSPService.getAllocations({ page: 1, pageSize: 10 });

      expect(mockGetAllocations).toHaveBeenCalledWith({ page: 1, pageSize: 10 });
      expect(result.allocations).toEqual(allocations);
      expect(result.total).toBe(1);
    });

    it('should return empty when no data', async () => {
      mockGetAllocations.mockResolvedValue({ data: null });

      const result = await MSPService.getAllocations();

      expect(result).toEqual({ allocations: [], total: 0 });
    });
  });

  describe('createAllocation', () => {
    it('should create allocation successfully', async () => {
      const allocation = { id: 1, mspUserId: 10, customerTenantId: 20 };
      mockCreateAllocation.mockResolvedValue({ code: 0, data: allocation });

      const result = await MSPService.createAllocation({ mspUserId: 10, customerTenantId: 20 } as any);

      expect(result).toEqual(allocation);
    });

    it('should throw error on failure', async () => {
      mockCreateAllocation.mockResolvedValue({ code: 1, message: 'Allocation failed' });

      await expect(MSPService.createAllocation({} as any)).rejects.toThrow('Allocation failed');
    });
  });

  describe('deallocate', () => {
    it('should deallocate successfully', async () => {
      mockDeallocate.mockResolvedValue({ code: 0 });

      await MSPService.deallocate(10, 20, 'No longer needed');

      expect(mockDeallocate).toHaveBeenCalledWith(10, 20, 'No longer needed');
    });

    it('should throw error on failure', async () => {
      mockDeallocate.mockResolvedValue({ code: 1, message: 'Deallocate failed' });

      await expect(MSPService.deallocate(10, 20)).rejects.toThrow('Deallocate failed');
    });
  });

  describe('getCustomers', () => {
    it('should return customers when data exists', async () => {
      const customers = [{ id: 1, code: 'C1', name: 'Customer 1' }];
      mockGetCustomers.mockResolvedValue({ data: { customers, total: 1 } });

      const result = await MSPService.getCustomers();

      expect(result.customers).toEqual(customers);
      expect(result.total).toBe(1);
    });

    it('should return empty when no data', async () => {
      mockGetCustomers.mockResolvedValue({ data: null });

      const result = await MSPService.getCustomers();

      expect(result).toEqual({ customers: [], total: 0 });
    });
  });

  describe('getCustomerTickets', () => {
    it('should return customer tickets', async () => {
      const tickets = [{ id: 1, title: 'Ticket 1', status: 'open' }];
      mockGetCustomerTickets.mockResolvedValue({ data: { tickets, total: 1 } });

      const result = await MSPService.getCustomerTickets(20, { status: 'open' });

      expect(mockGetCustomerTickets).toHaveBeenCalledWith(20, { status: 'open' });
      expect(result.tickets).toEqual(tickets);
      expect(result.total).toBe(1);
    });

    it('should handle non-array tickets response', async () => {
      const ticket = { id: 1, title: 'Single Ticket', status: 'open' };
      mockGetCustomerTickets.mockResolvedValue({ data: { tickets: ticket, total: 1 } });

      const result = await MSPService.getCustomerTickets(20);

      expect(result.tickets).toEqual([ticket]);
    });

    it('should return empty when no data', async () => {
      mockGetCustomerTickets.mockResolvedValue({ data: null });

      const result = await MSPService.getCustomerTickets(20);

      expect(result).toEqual({ tickets: [], total: 0 });
    });
  });

  describe('assignTechnician', () => {
    it('should assign technician successfully', async () => {
      mockAssignTechnician.mockResolvedValue({ code: 0, data: { id: 1, status: 'assigned' } });

      const result = await MSPService.assignTechnician(100, 20, 5);

      expect(mockAssignTechnician).toHaveBeenCalledWith(100, 20, 5);
      expect(result).toEqual({ id: 1, status: 'assigned' });
    });

    it('should throw error on failure', async () => {
      mockAssignTechnician.mockResolvedValue({ code: 1, message: 'Assignment failed' });

      await expect(MSPService.assignTechnician(100, 20)).rejects.toThrow('Assignment failed');
    });
  });

  describe('getCustomerReports', () => {
    it('should return customer reports', async () => {
      const reports = [{ customerId: 1, ticketCount: 5 }];
      mockGetCustomerReports.mockResolvedValue({ data: reports });

      const result = await MSPService.getCustomerReports({ startDate: '2024-01-01', endDate: '2024-01-31' });

      expect(result).toEqual(reports);
    });

    it('should return empty array when no data', async () => {
      mockGetCustomerReports.mockResolvedValue({ data: null });

      const result = await MSPService.getCustomerReports({ startDate: '2024-01-01', endDate: '2024-01-31' });

      expect(result).toEqual([]);
    });
  });

  describe('isMSPUser', () => {
    it('should fetch and cache MSP user status', async () => {
      mockIsMSPUser.mockResolvedValue({ isMSP: true, isAdmin: false });

      const result = await MSPService.isMSPUser();

      expect(result).toEqual({ isMSP: true, isAdmin: false });
      expect(mockIsMSPUser).toHaveBeenCalledTimes(1);
    });

    it('should return cached result on subsequent calls', async () => {
      mockIsMSPUser.mockResolvedValue({ isMSP: true, isAdmin: true });

      await MSPService.isMSPUser();
      const result = await MSPService.isMSPUser();

      expect(mockIsMSPUser).toHaveBeenCalledTimes(1);
      expect(result).toEqual({ isMSP: true, isAdmin: true });
    });
  });

  describe('getMSPContext', () => {
    it('should fetch and cache MSP context', async () => {
      const context = { tenantId: 1, role: 'admin' };
      mockGetMSPContext.mockResolvedValue({ code: 0, data: context });

      const result = await MSPService.getMSPContext();

      expect(result).toEqual(context);
    });

    it('should return null when API returns error code', async () => {
      mockGetMSPContext.mockResolvedValue({ code: 1, data: null });

      const result = await MSPService.getMSPContext();

      expect(result).toBeNull();
    });

    it('should return cached context on subsequent calls', async () => {
      const context = { tenantId: 1, role: 'admin' };
      mockGetMSPContext.mockResolvedValue({ code: 0, data: context });

      await MSPService.getMSPContext();
      const result = await MSPService.getMSPContext();

      expect(mockGetMSPContext).toHaveBeenCalledTimes(1);
      expect(result).toEqual(context);
    });
  });

  describe('refreshCache', () => {
    it('should clear cached values', async () => {
      mockIsMSPUser.mockResolvedValue({ isMSP: true, isAdmin: false });
      await MSPService.isMSPUser();

      MSPService.refreshCache();

      mockIsMSPUser.mockResolvedValue({ isMSP: false, isAdmin: false });
      const result = await MSPService.isMSPUser();

      expect(mockIsMSPUser).toHaveBeenCalledTimes(2);
      expect(result).toEqual({ isMSP: false, isAdmin: false });
    });
  });

  describe('getAllocationHistory', () => {
    it('should return allocation history', async () => {
      const history = [{ id: 1, action: 'allocated' }];
      mockGetAllocationHistory.mockResolvedValue({ data: history });

      const result = await MSPService.getAllocationHistory({ mspUserId: 10 });

      expect(mockGetAllocationHistory).toHaveBeenCalledWith({ mspUserId: 10 });
      expect(result).toEqual(history);
    });

    it('should return empty array when no data', async () => {
      mockGetAllocationHistory.mockResolvedValue({ data: null });

      const result = await MSPService.getAllocationHistory({});

      expect(result).toEqual([]);
    });
  });
});
