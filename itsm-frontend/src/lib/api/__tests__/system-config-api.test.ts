import { SystemConfigAPI } from '@/lib/api/system-config-api';
import { httpClient } from '@/lib/api/http-client';
import type { SystemStats } from '@/types/dashboard';

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
const mockPut = httpClient.put as jest.Mock;

describe('SystemConfigAPI', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getConfigs', () => {
    it('should get configs with params', async () => {
      const expected = { items: [], total: 0, page: 1, pageSize: 10, totalPages: 0 };
      mockGet.mockResolvedValue(expected);
      const res = await SystemConfigAPI.getConfigs({ page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/system-configs', { page: 1, pageSize: 10 });
      expect(res).toEqual(expected);
    });

    it('should get configs without params', async () => {
      const expected = { items: [], total: 0, page: 1, pageSize: 10, totalPages: 0 };
      mockGet.mockResolvedValue(expected);
      const res = await SystemConfigAPI.getConfigs();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/system-configs', { pageSize: 1000 });
      expect(res).toEqual(expected);
    });
  });

  describe('getConfig', () => {
    it('should get config by id', async () => {
      const expected = { id: 1, key: 'site_name', value: 'ITSM' };
      mockGet.mockResolvedValue(expected);
      const res = await SystemConfigAPI.getConfig(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/system-configs/1');
      expect(res).toEqual(expected);
    });
  });

  describe('getConfigByKey', () => {
    it('should get config by key', async () => {
      const expected = { id: 1, key: 'site_name', value: 'ITSM' };
      mockGet.mockResolvedValue(expected);
      const res = await SystemConfigAPI.getConfigByKey('site_name');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/system-configs/key/site_name');
      expect(res).toEqual(expected);
    });
  });

  describe('updateConfig', () => {
    it('should update config', async () => {
      const data = { key: 'site_name', value: 'New ITSM' };
      const expected = { id: 1, key: 'site_name', value: 'New ITSM' };
      mockPut.mockResolvedValue(expected);
      const res = await SystemConfigAPI.updateConfig(1, data);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/system-configs/1', data);
      expect(res).toEqual(expected);
    });
  });

  describe('updateConfigs', () => {
    it('should batch update configs', async () => {
      const data = [{ key: 'siteName', value: 'A' }, { key: 'siteDescription', value: 'B' }];
      const expected = [{ id: 1, value: 'A' }, { id: 2, value: 'B' }];
      mockPut.mockResolvedValue(expected);
      const res = await SystemConfigAPI.updateConfigs(data);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/system-configs/batch', data);
      expect(res).toEqual(expected);
    });
  });

  describe('getSystemStatus', () => {
    it('返回与系统健康卡同一份运行时 DTO（含 availability 与 goroutines）', async () => {
      const expected: SystemStats = {
        uptime: 3600,
        cpuUsage: 12.5,
        memoryUsage: 30,
        diskUsage: 40,
        avgResponseTime: 25,
        requestsPerSecond: 4.2,
        errorRate: 0.5,
        metricsWindowSeconds: 30,
        dbConnections: 8,
        dbSize: 1024,
        cacheHitRate: 95,
        cores: 8,
        goroutines: 42,
        sampledAt: '2026-10-15T02:00:00.000Z',
        availability: {
          cpu: 'measured',
          memory: 'measured',
          disk: 'measured',
          http: 'measured',
          dbSize: 'measured',
          cache: 'measured',
        },
      };
      mockGet.mockResolvedValue(expected);
      const res = await SystemConfigAPI.getSystemStatus();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/system-configs/status');
      expect(res.goroutines).toBe(42);
      expect(res.availability.memory).toBe('measured');
    });
  });
});
