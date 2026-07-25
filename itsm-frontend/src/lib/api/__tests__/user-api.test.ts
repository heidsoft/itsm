import { UserApi } from '../user-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
    request: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;
const mockRequest = httpClient.request as jest.Mock;

describe('UserApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getUsers', () => {
    it('should get users with params', async () => {
      const resp = { users: [], pagination: { page: 1, pageSize: 10, total: 0, totalPage: 0 } };
      mockGet.mockResolvedValue(resp);
      const result = await UserApi.getUsers({ page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/users', { page: 1, pageSize: 10 });
      expect(result).toEqual(resp);
    });

    it('should get users without params', async () => {
      mockGet.mockResolvedValue({ users: [], pagination: {} });
      await UserApi.getUsers();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/users', {});
    });
  });

  describe('createUser', () => {
    it('should create user', async () => {
      const data = { username: 'test', email: 'test@test.com', name: 'Test', department: 'IT', phone: '123', password: 'pass', tenantId: 1 };
      mockPost.mockResolvedValue({ id: 1, ...data });
      const result = await UserApi.createUser(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/users', data);
      expect(result.username).toBe('test');
    });
  });

  describe('getUserById', () => {
    it('should get user by id', async () => {
      mockGet.mockResolvedValue({ id: 1, username: 'test' });
      const result = await UserApi.getUserById(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/users/1');
      expect(result.id).toBe(1);
    });
  });

  describe('updateUser', () => {
    it('should update user', async () => {
      mockPut.mockResolvedValue({ id: 1, name: 'Updated' });
      const result = await UserApi.updateUser(1, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/users/1', { name: 'Updated' });
      expect(result.name).toBe('Updated');
    });
  });

  describe('deleteUser', () => {
    it('should delete user', async () => {
      mockDelete.mockResolvedValue(undefined);
      await UserApi.deleteUser(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/users/1');
    });
  });

  describe('changeUserStatus', () => {
    it('should change user status', async () => {
      mockPut.mockResolvedValue(undefined);
      await UserApi.changeUserStatus(1, false);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/users/1/status', { active: false });
    });
  });

  describe('resetPassword', () => {
    it('should reset password', async () => {
      mockPut.mockResolvedValue(undefined);
      await UserApi.resetPassword(1, 'newPass123');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/users/1/reset-password', { newPassword: 'newPass123' });
    });
  });

  describe('getUserStats', () => {
    it('should get stats without tenantId', async () => {
      mockGet.mockResolvedValue({ total: 10, active: 8, inactive: 2 });
      const result = await UserApi.getUserStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/users/stats', {});
      expect(result.total).toBe(10);
    });

    it('should get stats with tenantId', async () => {
      mockGet.mockResolvedValue({ total: 5 });
      await UserApi.getUserStats(2);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/users/stats', { tenantId: 2 });
    });
  });

  describe('batchUpdateUsers', () => {
    it('should batch update users', async () => {
      mockPut.mockResolvedValue(undefined);
      await UserApi.batchUpdateUsers({ userIds: [1, 2], action: 'activate' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/users/batch', { userIds: [1, 2], action: 'activate' });
    });
  });

  describe('searchUsers', () => {
    it('should search users', async () => {
      mockGet.mockResolvedValue([{ id: 1 }]);
      const result = await UserApi.searchUsers({ keyword: 'test' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/users/search', { keyword: 'test' });
      expect(result).toHaveLength(1);
    });
  });

  describe('exportUsers', () => {
    it('should export users', async () => {
      const blob = new Blob(['data']);
      mockRequest.mockResolvedValue(blob);
      const result = await UserApi.exportUsers({ department: 'IT' });
      expect(mockRequest).toHaveBeenCalledWith(expect.objectContaining({ method: 'GET', url: '/api/v1/users/export', responseType: 'blob' }));
      expect(result).toBe(blob);
    });
  });

  describe('importUsers', () => {
    it('should import users', async () => {
      const file = new File(['data'], 'users.csv');
      mockPost.mockResolvedValue({ success: [], failed: [], total: 1, processed: 1 });
      const result = await UserApi.importUsers(file);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/users/import', expect.any(FormData), expect.any(Object));
      expect(result.total).toBe(1);
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Not found'));
      await expect(UserApi.getUserById(999)).rejects.toThrow('Not found');
    });
  });
});
