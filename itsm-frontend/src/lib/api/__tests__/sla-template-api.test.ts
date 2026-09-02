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
  beforeEach(() => { jest.clearAllMocks(); });

  describe('listTemplates', () => {
    it('should list templates from the standard list response', async () => {
      const templates = [{ key: 'gold', name: 'Gold SLA', recommended: true }];
      mockGet.mockResolvedValue({ items: templates, total: 1 });
      const result = await SLATemplateApi.listTemplates();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/templates');
      expect(result).toEqual(templates);
    });

  });

  describe('getTemplate', () => {
    it('should get template by key', async () => {
      mockGet.mockResolvedValue({ key: 'gold', name: 'Gold SLA' });
      const result = await SLATemplateApi.getTemplate('gold');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/templates/gold');
      expect(result.key).toBe('gold');
    });

    it('should encode key with special characters', async () => {
      mockGet.mockResolvedValue({ key: 'special/key', name: 'Test' });
      await SLATemplateApi.getTemplate('special/key');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/templates/special%2Fkey');
    });
  });

  describe('installTemplate', () => {
    it('should install template', async () => {
      mockPost.mockResolvedValue({ templateKey: 'gold', slaDefinitionId: 1, created: true, wasAlreadyExist: false, message: 'Installed' });
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
    it('should install all recommended templates', async () => {
      const templates = [
        { key: 'gold', name: 'Gold', recommended: true },
        { key: 'silver', name: 'Silver', recommended: true },
        { key: 'basic', name: 'Basic', recommended: false },
      ];
      mockGet.mockResolvedValue({ items: templates, total: templates.length });
      mockPost.mockResolvedValue({ templateKey: 'gold', created: true, wasAlreadyExist: false, message: 'ok' });

      const result = await SLATemplateApi.installAllRecommended();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/sla/templates');
      // Only gold and silver are recommended
      expect(mockPost).toHaveBeenCalledTimes(2);
      expect(result).toHaveLength(2);
    });

    it('should continue on individual failures', async () => {
      const templates = [
        { key: 'gold', name: 'Gold', recommended: true },
        { key: 'silver', name: 'Silver', recommended: true },
      ];
      mockGet.mockResolvedValue({ items: templates, total: templates.length });
      mockPost
        .mockRejectedValueOnce(new Error('Install failed'))
        .mockResolvedValueOnce({ templateKey: 'silver', created: true, wasAlreadyExist: false, message: 'ok' });

      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
      const result = await SLATemplateApi.installAllRecommended();
      expect(result).toHaveLength(1);
      consoleSpy.mockRestore();
    });

    it('should return empty array when no recommended templates', async () => {
      mockGet.mockResolvedValue({
        items: [{ key: 'basic', name: 'Basic', recommended: false }],
        total: 1,
      });
      const result = await SLATemplateApi.installAllRecommended();
      expect(mockPost).not.toHaveBeenCalled();
      expect(result).toEqual([]);
    });
  });

  describe('error propagation', () => {
    it('should propagate errors from listTemplates', async () => {
      mockGet.mockRejectedValue(new Error('Server error'));
      await expect(SLATemplateApi.listTemplates()).rejects.toThrow('Server error');
    });
  });
});
