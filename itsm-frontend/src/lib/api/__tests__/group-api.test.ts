import { GroupAPI } from '../group-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
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

describe('GroupAPI', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getGroups', () => {
    it('should get groups with params', async () => {
      const resp = { groups: [], pagination: { page: 1, pageSize: 10, total: 0, totalPage: 0 } };
      mockGet.mockResolvedValue(resp);
      const result = await GroupAPI.getGroups({ page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/groups', { page: 1, pageSize: 10 });
      expect(result).toEqual(resp);
    });

    it('should get groups without params', async () => {
      mockGet.mockResolvedValue({ groups: [] });
      await GroupAPI.getGroups();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/groups', {});
    });
  });

  describe('getGroup', () => {
    it('should get group by id', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'Team A' });
      const result = await GroupAPI.getGroup(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/groups/1');
      expect(result.name).toBe('Team A');
    });
  });

  describe('createGroup', () => {
    it('should create group', async () => {
      const data = { name: 'New Group', description: 'desc' };
      mockPost.mockResolvedValue({ id: 1, ...data });
      const result = await GroupAPI.createGroup(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/groups', data);
      expect(result.name).toBe('New Group');
    });
  });

  describe('updateGroup', () => {
    it('should update group', async () => {
      mockPut.mockResolvedValue({ id: 1, name: 'Updated' });
      const result = await GroupAPI.updateGroup(1, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/groups/1', { name: 'Updated' });
      expect(result.name).toBe('Updated');
    });
  });

  describe('deleteGroup', () => {
    it('should delete group', async () => {
      mockDelete.mockResolvedValue(undefined);
      await GroupAPI.deleteGroup(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/groups/1');
    });
  });

  describe('getMembers', () => {
    it('should get members with params', async () => {
      mockGet.mockResolvedValue({ members: [], pagination: {} });
      await GroupAPI.getMembers(1, { page: 1, pageSize: 20 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/groups/1/members', { page: 1, pageSize: 20 });
    });

    it('should get members without params', async () => {
      mockGet.mockResolvedValue({ members: [] });
      await GroupAPI.getMembers(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/groups/1/members', {});
    });
  });

  describe('addMember', () => {
    it('should add member', async () => {
      mockPost.mockResolvedValue(undefined);
      await GroupAPI.addMember(1, 5);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/groups/1/members', { userId: 5 });
    });
  });

  describe('removeMember', () => {
    it('should remove member', async () => {
      mockDelete.mockResolvedValue(undefined);
      await GroupAPI.removeMember(1, 5);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/groups/1/members', { userId: 5 });
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Forbidden'));
      await expect(GroupAPI.getGroup(1)).rejects.toThrow('Forbidden');
    });
  });
});
