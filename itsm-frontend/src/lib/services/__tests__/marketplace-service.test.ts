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

  describe('upgradeItem', () => {
    it('省略目标版本时不发送请求体，语义是升到目录 latest', async () => {
      const response = { id: 10, tenantId: 1, itemId: 5, installedVersion: '1.1.0', status: 'active', installedAt: '2024-01-01' };
      mockPost.mockResolvedValue(response);

      const result = await marketplaceService.upgradeItem(5);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/marketplace/items/5/upgrade', undefined);
      expect(result.installedVersion).toBe('1.1.0');
    });

    it('点名版本时只发送 targetVersion（回滚走同一入口）', async () => {
      mockPost.mockResolvedValue({ id: 10, tenantId: 1, itemId: 5, installedVersion: '1.0.0', status: 'active', installedAt: '2024-01-01' });

      await marketplaceService.upgradeItem(5, '1.0.0');

      expect(mockPost).toHaveBeenCalledWith('/api/v1/marketplace/items/5/upgrade', { targetVersion: '1.0.0' });
    });
  });

  describe('setAutoUpgrade', () => {
    it.each([true, false])('必须显式发送布尔 %p，不能让后端把未传当成关闭', async autoUpgrade => {
      const mockPatch = httpClient.patch as jest.Mock;
      mockPatch.mockResolvedValue({ id: 10, tenantId: 1, itemId: 5, autoUpgrade, status: 'active', installedAt: '2024-01-01' });

      const result = await marketplaceService.setAutoUpgrade(5, autoUpgrade);

      expect(mockPatch).toHaveBeenCalledWith('/api/v1/marketplace/items/5/auto-upgrade', { autoUpgrade });
      expect(result.autoUpgrade).toBe(autoUpgrade);
    });
  });

  describe('dependencyStatus', () => {
    it('读取存量依赖报告，保留三态计数与明细', async () => {
      const report = {
        tenantId: 1,
        checkedAt: '2026-11-03T00:00:00Z',
        total: 2,
        satisfied: 0,
        unsatisfied: 1,
        declarationMissing: 1,
        findings: [
          { itemId: 7, itemName: 'dependent-skill', itemTitle: '依赖方', itemType: 'skill', status: 'unsatisfied', installedVersion: '1.0.0', installationStatus: 'active', problems: ['缺少前置 base-skill'] },
          { itemId: 8, itemName: 'orphan-skill', itemTitle: '无版本行', itemType: 'skill', status: 'declaration_missing', installedVersion: '9.9.9', installationStatus: 'active', problems: ['目录里没有 9.9.9 的已发布版本行'] },
        ],
      };
      mockGet.mockResolvedValue(report);

      const result = await marketplaceService.dependencyStatus();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/marketplace/dependency-status');
      expect(result.unsatisfied).toBe(1);
      expect(result.declarationMissing).toBe(1);
      expect(result.findings.map(finding => finding.status)).toEqual(['unsatisfied', 'declaration_missing']);
    });
  });
});
