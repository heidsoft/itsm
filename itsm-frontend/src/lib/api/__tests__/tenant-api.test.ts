import { TenantAPI } from '@/lib/api/tenant-api';
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
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;

describe('TenantAPI', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getTenants', () => {
    it('should get tenants with params', async () => {
      const expected = { tenants: [], total: 0 };
      mockGet.mockResolvedValue(expected);
      const res = await TenantAPI.getTenants({ page: 1, pageSize: 10 } as any);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tenants', { page: 1, pageSize: 10 });
      expect(res).toEqual(expected);
    });
  });

  describe('getTenant', () => {
    it('should get tenant by id', async () => {
      const expected = { id: 1, name: 'Tenant 1' };
      mockGet.mockResolvedValue(expected);
      const res = await TenantAPI.getTenant(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tenants/1');
      expect(res).toEqual(expected);
    });
  });

  describe('createTenant', () => {
    it('should create tenant', async () => {
      const data = { name: 'New Tenant', code: 'new-tenant' };
      const expected = { id: 2, ...data };
      mockPost.mockResolvedValue(expected);
      const res = await TenantAPI.createTenant(data as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tenants', data);
      expect(res).toEqual(expected);
    });
  });

  describe('updateTenant', () => {
    it('should update tenant', async () => {
      const data = { name: 'Updated Tenant' };
      const expected = { id: 1, name: 'Updated Tenant' };
      mockPut.mockResolvedValue(expected);
      const res = await TenantAPI.updateTenant(1, data as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/tenants/1', data);
      expect(res).toEqual(expected);
    });
  });

  describe('deleteTenant', () => {
    it('should delete tenant', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TenantAPI.deleteTenant(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tenants/1');
    });
  });

  describe('getCurrentTenant', () => {
    it('should get current tenant', async () => {
      const expected = { id: 1, name: 'Current' };
      mockGet.mockResolvedValue(expected);
      const res = await TenantAPI.getCurrentTenant();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tenants/current');
      expect(res).toEqual(expected);
    });
  });

  describe('switchTenant', () => {
    it('should switch tenant', async () => {
      mockPost.mockResolvedValue(undefined);
      await TenantAPI.switchTenant(2);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tenants/switch', { tenantId: 2 });
    });
  });
});
