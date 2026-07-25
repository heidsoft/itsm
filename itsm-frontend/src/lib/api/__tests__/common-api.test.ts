import { CommonApi } from '../common-api';
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

describe('CommonApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getUsers', () => {
    it('should get users without params', async () => {
      mockGet.mockResolvedValue([{ id: 1, name: 'User1' }]);
      const result = await CommonApi.getUsers();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/users', undefined);
      expect(result).toEqual([{ id: 1, name: 'User1' }]);
    });

    it('should get users with params', async () => {
      mockGet.mockResolvedValue([]);
      await CommonApi.getUsers({ page: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/users', { page: 1 });
    });
  });

  describe('getUser', () => {
    it('should get user by id', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'User1' });
      const result = await CommonApi.getUser(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/users/1');
      expect(result).toEqual({ id: 1, name: 'User1' });
    });
  });

  describe('listUsers', () => {
    it('should delegate to getUsers', async () => {
      mockGet.mockResolvedValue([{ id: 1 }]);
      const result = await CommonApi.listUsers({ active: true });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/users', { active: true });
      expect(result).toEqual([{ id: 1 }]);
    });
  });

  describe('getDepartments', () => {
    it('should get departments', async () => {
      mockGet.mockResolvedValue([{ id: 1, name: 'IT' }]);
      const result = await CommonApi.getDepartments();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/departments');
      expect(result).toEqual([{ id: 1, name: 'IT' }]);
    });
  });

  describe('getDepartmentTree', () => {
    it('should get department tree', async () => {
      mockGet.mockResolvedValue([{ id: 1, children: [] }]);
      const result = await CommonApi.getDepartmentTree();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/departments/tree');
      expect(result).toEqual([{ id: 1, children: [] }]);
    });
  });

  describe('getTeams', () => {
    it('should get teams', async () => {
      mockGet.mockResolvedValue([{ id: 1, name: 'Team A' }]);
      const result = await CommonApi.getTeams();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/teams');
      expect(result).toEqual([{ id: 1, name: 'Team A' }]);
    });
  });

  describe('getTags', () => {
    it('should get tags', async () => {
      mockGet.mockResolvedValue([{ id: 1, name: 'urgent' }]);
      const result = await CommonApi.getTags();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tags');
      expect(result).toEqual([{ id: 1, name: 'urgent' }]);
    });
  });

  describe('getAuditLogs', () => {
    it('should get audit logs without params', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0 });
      const result = await CommonApi.getAuditLogs();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/audit-logs', undefined);
      expect(result).toEqual({ items: [], total: 0 });
    });

    it('should get audit logs with params', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0 });
      await CommonApi.getAuditLogs({ page: 1, action: 'create' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/audit-logs', { page: 1, action: 'create' });
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Unauthorized'));
      await expect(CommonApi.getUsers()).rejects.toThrow('Unauthorized');
    });
  });
});
