import { RoleAPI } from '@/lib/api/role-api';
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

describe('RoleAPI', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getRoles', () => {
    it('should get roles with params', async () => {
      const expected = { roles: [{ id: 1, name: 'Admin', permissions: ['read'] }], total: 1 };
      mockGet.mockResolvedValue(expected);
      const res = await RoleAPI.getRoles({ page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/roles', expect.objectContaining({ page: 1, pageSize: 10 }));
      expect(res.roles).toHaveLength(1);
      expect(res.roles[0].permissions).toEqual(['read']);
    });

    it('should normalize object permissions', async () => {
      const expected = { roles: [{ id: 1, name: 'Admin', permissions: [{ code: 'read:all' }] }], total: 1 };
      mockGet.mockResolvedValue(expected);
      const res = await RoleAPI.getRoles();
      expect(res.roles[0].permissions).toEqual(['read:all']);
    });
  });

  describe('getRole', () => {
    it('should get role by id', async () => {
      const expected = { id: 1, name: 'Admin', permissions: ['read'] };
      mockGet.mockResolvedValue(expected);
      const res = await RoleAPI.getRole(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/roles/1');
      expect(res.name).toBe('Admin');
    });
  });

  describe('createRole', () => {
    it('should create role', async () => {
      const data = { name: 'New Role', permissions: ['read'] };
      const expected = { id: 2, name: 'New Role', permissions: ['read'] };
      mockPost.mockResolvedValue(expected);
      const res = await RoleAPI.createRole(data as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/roles', data);
      expect(res.name).toBe('New Role');
    });
  });

  describe('updateRole', () => {
    it('should update role', async () => {
      const data = { name: 'Updated' };
      const expected = { id: 1, name: 'Updated', permissions: [] };
      mockPut.mockResolvedValue(expected);
      const res = await RoleAPI.updateRole(1, data as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/roles/1', data);
      expect(res.name).toBe('Updated');
    });
  });

  describe('deleteRole', () => {
    it('should delete role', async () => {
      mockDelete.mockResolvedValue(undefined);
      await RoleAPI.deleteRole(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/roles/1');
    });
  });

  describe('getPermissions', () => {
    it('should get permissions list', async () => {
      mockGet.mockResolvedValue({ permissions: ['read:tickets', 'write:tickets'] });
      const res = await RoleAPI.getPermissions();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/permissions');
      expect(res).toEqual(['read:tickets', 'write:tickets']);
    });
  });

  describe('getPermissionCatalog', () => {
    it('should get permission catalog with items', async () => {
      const items = [{ id: 1, code: 'read:tickets', name: 'Read Tickets', resource: 'tickets', action: 'read' }];
      mockGet.mockResolvedValue({ permissions: [], items });
      const res = await RoleAPI.getPermissionCatalog();
      expect(res).toEqual(items);
    });

    it('should fallback to parsing permissions when items missing', async () => {
      mockGet.mockResolvedValue({ permissions: ['tickets:read'] });
      const res = await RoleAPI.getPermissionCatalog();
      expect(res[0].code).toBe('tickets:read');
      expect(res[0].resource).toBe('tickets');
      expect(res[0].action).toBe('read');
    });
  });

  describe('initDefaultPermissions', () => {
    it('should init default permissions', async () => {
      mockPost.mockResolvedValue(undefined);
      await RoleAPI.initDefaultPermissions();
      expect(mockPost).toHaveBeenCalledWith('/api/v1/permissions/init');
    });
  });

  describe('assignPermissions', () => {
    it('should assign permissions to role', async () => {
      mockPost.mockResolvedValue(undefined);
      await RoleAPI.assignPermissions(1, [10, 20]);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/roles/1/permissions', { permissionIds: [10, 20] });
    });
  });
});
