import { marketplaceService } from '../marketplace-service';
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

describe('MarketplaceService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('listItems', () => {
    it('should list items with default pagination', async () => {
      const response = { items: [{ id: 1, name: 'plugin-a', title: 'Plugin A', type: 'plugin', provider: 'acme' }], total: 1, page: 1, pageSize: 100 };
      mockGet.mockResolvedValue(response);

      const result = await marketplaceService.listItems();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/marketplace/items', {
        type: undefined,
        category: undefined,
        search: undefined,
        isOfficial: undefined,
        page: 1,
        pageSize: 100,
      });
      expect(result.items).toHaveLength(1);
      expect(result.total).toBe(1);
    });

    it('should list items with filters', async () => {
      const response = { items: [], total: 0, page: 1, pageSize: 10 };
      mockGet.mockResolvedValue(response);

      await marketplaceService.listItems({ type: 'connector', category: 'im', search: 'slack', page: 2, pageSize: 10 });

      expect(mockGet).toHaveBeenCalledWith('/api/v1/marketplace/items', {
        type: 'connector',
        category: 'im',
        search: 'slack',
        isOfficial: undefined,
        page: 2,
        pageSize: 10,
      });
    });
  });

  describe('getItem', () => {
    it('should get a single marketplace item', async () => {
      const item = { id: 1, name: 'plugin-a', title: 'Plugin A', type: 'plugin', provider: 'acme' };
      mockGet.mockResolvedValue(item);

      const result = await marketplaceService.getItem(1);

      expect(mockGet).toHaveBeenCalledWith('/api/v1/marketplace/items/1');
      expect(result.name).toBe('plugin-a');
    });
  });

  describe('installItem', () => {
    it('should install an item', async () => {
      const installation = { id: 10, tenantId: 1, itemId: 1, installedVersion: '1.0.0', status: 'active', installedAt: '2024-01-01' };
      mockPost.mockResolvedValue(installation);

      const result = await marketplaceService.installItem(1);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/marketplace/items/1/install');
      expect(result.status).toBe('active');
    });
  });

  describe('uninstallItem', () => {
    it('should uninstall an item', async () => {
      mockPost.mockResolvedValue(undefined);

      await marketplaceService.uninstallItem(1);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/marketplace/items/1/uninstall');
    });
  });

  describe('listInstallations', () => {
    it('should list installations without status filter', async () => {
      const installations = [{ id: 10, tenantId: 1, itemId: 1, installedVersion: '1.0.0', status: 'active', installedAt: '2024-01-01' }];
      mockGet.mockResolvedValue(installations);

      const result = await marketplaceService.listInstallations();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/marketplace/installations', undefined);
      expect(result).toHaveLength(1);
    });

    it('should list installations with status filter', async () => {
      mockGet.mockResolvedValue([]);

      await marketplaceService.listInstallations('active');

      expect(mockGet).toHaveBeenCalledWith('/api/v1/marketplace/installations', { status: 'active' });
    });
  });

  describe('getInstallation', () => {
    it('should get a specific installation', async () => {
      const installation = { id: 10, tenantId: 1, itemId: 5, installedVersion: '2.0.0', status: 'active', installedAt: '2024-01-01' };
      mockGet.mockResolvedValue(installation);

      const result = await marketplaceService.getInstallation(5);

      expect(mockGet).toHaveBeenCalledWith('/api/v1/marketplace/installations/5');
      expect(result?.itemId).toBe(5);
    });

    it('should return null when installation not found', async () => {
      mockGet.mockRejectedValue(new Error('Not found'));

      const result = await marketplaceService.getInstallation(999);

      expect(result).toBeNull();
    });
  });

  describe('updateInstallationConfig', () => {
    it('should update installation config', async () => {
      const config = { apiKey: 'new-key', enabled: true };
      const response = { id: 10, tenantId: 1, itemId: 5, installedVersion: '2.0.0', status: 'active', config, installedAt: '2024-01-01' };
      mockPut.mockResolvedValue(response);

      const result = await marketplaceService.updateInstallationConfig(5, config);

      expect(mockPut).toHaveBeenCalledWith('/api/v1/marketplace/installations/5/config', config);
      expect(result.config).toEqual(config);
    });
  });
});
