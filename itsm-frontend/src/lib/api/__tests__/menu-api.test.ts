import { MenuAdminAPI, getUserMenus } from '@/lib/api/menu-api';
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

describe('MenuAdminAPI', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('list', () => {
    it('should get menu list', async () => {
      const expected = { menus: [], total: 0 };
      mockGet.mockResolvedValue(expected);
      const res = await MenuAdminAPI.list();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/menus');
      expect(res).toEqual(expected);
    });
  });

  describe('get', () => {
    it('should get menu by id', async () => {
      const expected = { id: 1, name: 'Dashboard', path: '/dashboard', sortOrder: 1, tenantId: 1, isVisible: true, isEnabled: true };
      mockGet.mockResolvedValue(expected);
      const res = await MenuAdminAPI.get(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/menus/1');
      expect(res).toEqual(expected);
    });
  });

  describe('create', () => {
    it('should create menu', async () => {
      const payload = { name: 'Settings', path: '/settings' };
      const expected = { id: 2, ...payload, sortOrder: 0, tenantId: 1, isVisible: true, isEnabled: true };
      mockPost.mockResolvedValue(expected);
      const res = await MenuAdminAPI.create(payload);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/menus', payload);
      expect(res).toEqual(expected);
    });
  });

  describe('update', () => {
    it('should update menu', async () => {
      const payload = { name: 'Updated Menu' };
      const expected = { id: 1, name: 'Updated Menu', path: '/dashboard', sortOrder: 1, tenantId: 1, isVisible: true, isEnabled: true };
      mockPut.mockResolvedValue(expected);
      const res = await MenuAdminAPI.update(1, payload);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/menus/1', payload);
      expect(res).toEqual(expected);
    });
  });

  describe('remove', () => {
    it('should delete menu', async () => {
      mockDelete.mockResolvedValue(undefined);
      await MenuAdminAPI.remove(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/menus/1');
    });
  });

  describe('initDefaults', () => {
    it('should init default menus', async () => {
      const expected = { message: 'ok', count: 5 };
      mockPost.mockResolvedValue(expected);
      const res = await MenuAdminAPI.initDefaults();
      expect(mockPost).toHaveBeenCalledWith('/api/v1/menus/init', {});
      expect(res).toEqual(expected);
    });
  });
});

describe('getUserMenus', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  it('should get user menus', async () => {
    const expected = { main: [], admin: [] };
    mockGet.mockResolvedValue(expected);
    const res = await getUserMenus();
    expect(mockGet).toHaveBeenCalledWith('/api/v1/auth/menus');
    expect(res).toEqual(expected);
  });
});
