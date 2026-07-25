import { cloudAccountApi, cloudServiceApi, cloudResourceApi } from '../cloud-api';
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
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;

describe('Cloud API', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('cloudAccountApi', () => {
    it('list should get accounts', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0 });
      await cloudAccountApi.list({ page: 1 } as any);
      expect(mockGet).toHaveBeenCalledWith('/cloud/accounts', { params: { page: 1 } });
    });

    it('get should get account by id', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'AWS' });
      const result = await cloudAccountApi.get(1);
      expect(mockGet).toHaveBeenCalledWith('/cloud/accounts/1');
      expect(result.name).toBe('AWS');
    });

    it('create should create account', async () => {
      const data = { name: 'AWS', provider: 'aws' };
      mockPost.mockResolvedValue({ id: 1, ...data });
      await cloudAccountApi.create(data as any);
      expect(mockPost).toHaveBeenCalledWith('/cloud/accounts', data);
    });

    it('update should update account', async () => {
      mockPut.mockResolvedValue({ id: 1, name: 'Updated' });
      await cloudAccountApi.update(1, { name: 'Updated' } as any);
      expect(mockPut).toHaveBeenCalledWith('/cloud/accounts/1', { name: 'Updated' });
    });

    it('delete should delete account', async () => {
      mockDelete.mockResolvedValue(undefined);
      await cloudAccountApi.delete(1);
      expect(mockDelete).toHaveBeenCalledWith('/cloud/accounts/1');
    });
  });

  describe('cloudServiceApi', () => {
    it('list should get services', async () => {
      mockGet.mockResolvedValue({ items: [] });
      await cloudServiceApi.list();
      expect(mockGet).toHaveBeenCalledWith('/cloud/services', { params: undefined });
    });

    it('get should get service by id', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'EC2' });
      await cloudServiceApi.get(1);
      expect(mockGet).toHaveBeenCalledWith('/cloud/services/1');
    });

    it('create should create service', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await cloudServiceApi.create({ name: 'S3' } as any);
      expect(mockPost).toHaveBeenCalledWith('/cloud/services', { name: 'S3' });
    });

    it('update should update service', async () => {
      mockPut.mockResolvedValue({ id: 1 });
      await cloudServiceApi.update(1, { name: 'Updated' } as any);
      expect(mockPut).toHaveBeenCalledWith('/cloud/services/1', { name: 'Updated' });
    });

    it('delete should delete service', async () => {
      mockDelete.mockResolvedValue(undefined);
      await cloudServiceApi.delete(1);
      expect(mockDelete).toHaveBeenCalledWith('/cloud/services/1');
    });
  });

  describe('cloudResourceApi', () => {
    it('list should get resources', async () => {
      mockGet.mockResolvedValue({ items: [] });
      await cloudResourceApi.list();
      expect(mockGet).toHaveBeenCalledWith('/cloud/resources', { params: undefined });
    });

    it('get should get resource by id', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'instance-1' });
      await cloudResourceApi.get(1);
      expect(mockGet).toHaveBeenCalledWith('/cloud/resources/1');
    });

    it('create should create resource', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await cloudResourceApi.create({ name: 'vm-1' } as any);
      expect(mockPost).toHaveBeenCalledWith('/cloud/resources', { name: 'vm-1' });
    });

    it('update should update resource', async () => {
      mockPut.mockResolvedValue({ id: 1 });
      await cloudResourceApi.update(1, { name: 'Updated' } as any);
      expect(mockPut).toHaveBeenCalledWith('/cloud/resources/1', { name: 'Updated' });
    });

    it('delete should delete resource', async () => {
      mockDelete.mockResolvedValue(undefined);
      await cloudResourceApi.delete(1);
      expect(mockDelete).toHaveBeenCalledWith('/cloud/resources/1');
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Network error'));
      await expect(cloudAccountApi.get(999)).rejects.toThrow('Network error');
    });
  });
});
