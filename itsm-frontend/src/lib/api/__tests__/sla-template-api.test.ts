import { SLATemplateApi } from '../sla-template-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;

describe('SLATemplateApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('listTemplates', () => {
    it('should list templates from the standard list response', async () => {
      const templates = [
        {
          key: 'incident_p1_critical',
          name: '事件 P1 紧急',
          recommended: true,
          installed: true,
          slaDefinitionId: 7,
          businessHours: { timeZone: 'Asia/Shanghai', alwaysOn: true, workDays: [1, 2, 3, 4, 5] },
          escalationLevels: [{ level: 1, thresholdMinutes: 15, notifyRoles: ['manager'] }],
        },
      ];
      mockGet.mockResolvedValue({ items: templates, total: 1 });

      const result = await SLATemplateApi.listTemplates();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/templates');
      expect(result).toEqual(templates);
      expect(result[0].installed).toBe(true);
      expect(result[0].businessHours?.timeZone).toBe('Asia/Shanghai');
      expect(result[0].escalationLevels?.[0].thresholdMinutes).toBe(15);
    });
  });

  describe('getTemplate', () => {
    it('should get template by key', async () => {
      mockGet.mockResolvedValue({ key: 'gold', name: 'Gold SLA', installed: false });
      const result = await SLATemplateApi.getTemplate('gold');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/templates/gold');
      expect(result.key).toBe('gold');
    });

    it('should encode key with special characters', async () => {
      mockGet.mockResolvedValue({ key: 'special/key', name: 'Test', installed: false });
      await SLATemplateApi.getTemplate('special/key');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/templates/special%2Fkey');
    });
  });

  describe('installTemplate', () => {
    it('should install template', async () => {
      mockPost.mockResolvedValue({
        templateKey: 'gold',
        slaDefinitionId: 1,
        created: true,
        wasAlreadyExist: false,
        message: 'Installed',
      });
      const result = await SLATemplateApi.installTemplate('gold');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/sla/templates/gold/install', {});
      expect(result.created).toBe(true);
    });

    it('should encode key for install', async () => {
      mockPost.mockResolvedValue({ templateKey: 'key/1', created: false, wasAlreadyExist: true });
      await SLATemplateApi.installTemplate('key/1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/sla/templates/key%2F1/install', {});
    });
  });

  describe('installAllRecommended', () => {
    it('should install only recommended templates', async () => {
      const templates = [
        { key: 'gold', name: 'Gold', recommended: true, installed: false },
        { key: 'silver', name: 'Silver', recommended: true, installed: false },
        { key: 'basic', name: 'Basic', recommended: false, installed: false },
      ];
      mockGet.mockResolvedValue({ items: templates, total: templates.length });
      mockPost.mockResolvedValue({ templateKey: 'gold', created: true, wasAlreadyExist: false });

      const summary = await SLATemplateApi.installAllRecommended();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/templates');
      expect(mockPost).toHaveBeenCalledTimes(2);
      expect(summary.results).toHaveLength(2);
      expect(summary.failures).toEqual([]);
    });

    it('should report per-template failures instead of swallowing them', async () => {
      const templates = [
        { key: 'gold', name: 'Gold', recommended: true, installed: false },
        { key: 'silver', name: 'Silver', recommended: true, installed: false },
      ];
      mockGet.mockResolvedValue({ items: templates, total: templates.length });
      mockPost
        .mockRejectedValueOnce(new Error('安装 SLA 模板失败'))
        .mockResolvedValueOnce({ templateKey: 'silver', created: true, wasAlreadyExist: false });

      const summary = await SLATemplateApi.installAllRecommended();

      expect(summary.results).toHaveLength(1);
      expect(summary.results[0].templateKey).toBe('silver');
      expect(summary.failures).toEqual([{ templateKey: 'gold', reason: '安装 SLA 模板失败' }]);
    });

    it('should return empty summary when no recommended templates', async () => {
      mockGet.mockResolvedValue({
        items: [{ key: 'basic', name: 'Basic', recommended: false, installed: false }],
        total: 1,
      });

      const summary = await SLATemplateApi.installAllRecommended();

      expect(mockPost).not.toHaveBeenCalled();
      expect(summary).toEqual({ results: [], failures: [] });
    });
  });

  describe('error propagation', () => {
    it('should propagate errors from listTemplates', async () => {
      mockGet.mockRejectedValue(new Error('Server error'));
      await expect(SLATemplateApi.listTemplates()).rejects.toThrow('Server error');
    });
  });
});
