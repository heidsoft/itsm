import { DomainConfigApi } from '../domain-config-api';
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

describe('DomainConfigApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('list', () => {
    it('should list configs without configType', async () => {
      mockGet.mockResolvedValue([{ id: 1, configKey: 'key1', configType: 'sla' }]);
      const result = await DomainConfigApi.list();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/domain-configs', undefined);
      expect(result[0].id).toBe(1);
      expect(result[0].configKey).toBe('key1');
    });

    it('should list configs with configType filter', async () => {
      mockGet.mockResolvedValue([{ id: 1, configType: 'workflow' }]);
      const result = await DomainConfigApi.list('workflow');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/domain-configs', { configType: 'workflow' });
      expect(result[0].configType).toBe('workflow');
    });

    it('should normalize response with defaults', async () => {
      mockGet.mockResolvedValue([{}]);
      const result = await DomainConfigApi.list();
      expect(result[0]).toEqual({
        id: 0,
        configKey: '',
        configType: '',
        configValue: {},
        inheritMode: 'inherit',
        tenantId: 0,
        departmentId: 0,
        teamId: 0,
        version: 1,
        isActive: true,
        description: undefined,
        createdAt: undefined,
        updatedAt: undefined,
      });
    });

    it('should handle null response', async () => {
      mockGet.mockResolvedValue(null);
      const result = await DomainConfigApi.list();
      expect(result).toEqual([]);
    });
  });

  describe('save', () => {
    it('should save config', async () => {
      mockPost.mockResolvedValue(undefined);
      const payload = { configType: 'sla', configKey: 'response_time', configValue: { value: 30 } };
      await DomainConfigApi.save(payload);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/domain-configs', payload);
    });
  });

  describe('getEffective', () => {
    it('should get effective config', async () => {
      mockGet.mockResolvedValue({ key: 'resp_time', value: { hours: 4 }, source: 'team', inheritMode: 'override', version: 2 });
      const result = await DomainConfigApi.getEffective({ configType: 'sla', configKey: 'resp_time' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/domain-configs/effective', { configType: 'sla', configKey: 'resp_time' });
      expect(result).toEqual({ key: 'resp_time', value: { hours: 4 }, source: 'team', inheritMode: 'override', version: 2 });
    });

    it('should include departmentId and teamId if provided', async () => {
      mockGet.mockResolvedValue({ key: 'k', value: {}, source: 'dept', inheritMode: 'inherit', version: 1 });
      await DomainConfigApi.getEffective({ configType: 'sla', configKey: 'k', departmentId: 5, teamId: 3 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/domain-configs/effective', { configType: 'sla', configKey: 'k', departmentId: 5, teamId: 3 });
    });

    it('should return null for null/undefined response', async () => {
      mockGet.mockResolvedValue(null);
      const result = await DomainConfigApi.getEffective({ configType: 'sla', configKey: 'k' });
      expect(result).toBeNull();
    });

    it('should normalize effective config with defaults', async () => {
      mockGet.mockResolvedValue({});
      const result = await DomainConfigApi.getEffective({ configType: 'sla', configKey: 'k' });
      expect(result).toEqual({ key: '', value: {}, source: '', inheritMode: '', version: 0 });
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Config not found'));
      await expect(DomainConfigApi.list()).rejects.toThrow('Config not found');
    });
  });
});
