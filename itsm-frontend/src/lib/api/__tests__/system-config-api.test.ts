import { SystemConfigAPI } from '@/lib/api/system-config-api';
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
const mockPut = httpClient.put as jest.Mock;

describe('SystemConfigAPI', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getConfigs', () => {
    it('should get configs with params', async () => {
      const expected = { configs: [], total: 0 };
      mockGet.mockResolvedValue(expected);
      const res = await SystemConfigAPI.getConfigs({ page: 1, pageSize: 10 } as any);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/system-configs', { page: 1, pageSize: 10 });
      expect(res).toEqual(expected);
    });

    it('should get configs without params', async () => {
      const expected = { configs: [], total: 0 };
      mockGet.mockResolvedValue(expected);
      const res = await SystemConfigAPI.getConfigs();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/system-configs', undefined);
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
      const data = { value: 'New ITSM' };
      const expected = { id: 1, key: 'site_name', value: 'New ITSM' };
      mockPut.mockResolvedValue(expected);
      const res = await SystemConfigAPI.updateConfig(1, data as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/system-configs/1', data);
      expect(res).toEqual(expected);
    });
  });

  describe('updateConfigs', () => {
    it('should batch update configs', async () => {
      const data = [{ id: 1, value: 'A' }, { id: 2, value: 'B' }] as any;
      const expected = [{ id: 1, value: 'A' }, { id: 2, value: 'B' }];
      mockPut.mockResolvedValue(expected);
      const res = await SystemConfigAPI.updateConfigs(data);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/system-configs/batch', data);
      expect(res).toEqual(expected);
    });
  });

  describe('getSystemStatus', () => {
    it('should get system status', async () => {
      const expected = { version: '1.0.0', uptime: 3600 };
      mockGet.mockResolvedValue(expected);
      const res = await SystemConfigAPI.getSystemStatus();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/system-configs/status');
      expect(res).toEqual(expected);
    });
  });
});
