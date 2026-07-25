import { TemplateApi } from '../template-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
    request: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;
const mockPatch = httpClient.patch as jest.Mock;
const mockRequest = httpClient.request as jest.Mock;

describe('TemplateApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getTemplates', () => {
    it('should get templates', async () => {
      mockGet.mockResolvedValue({ templates: [], total: 0 });
      await TemplateApi.getTemplates({ page: 1, pageSize: 10 } as any);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates', { page: 1, pageSize: 10 });
    });
  });

  describe('getTemplate', () => {
    it('should get template by id', async () => {
      mockGet.mockResolvedValue({ id: '1', name: 'T1' });
      await TemplateApi.getTemplate('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/1');
    });
  });

  describe('createTemplate', () => {
    it('should create template', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await TemplateApi.createTemplate({ name: 'New' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates', { name: 'New' });
    });
  });

  describe('updateTemplate', () => {
    it('should update template', async () => {
      mockPatch.mockResolvedValue({ id: '1' });
      await TemplateApi.updateTemplate('1', { name: 'Updated' } as any);
      expect(mockPatch).toHaveBeenCalledWith('/api/v1/templates/1', { name: 'Updated' });
    });
  });

  describe('deleteTemplate', () => {
    it('should delete template', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TemplateApi.deleteTemplate('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/templates/1');
    });
  });

  describe('archiveTemplate', () => {
    it('should archive', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await TemplateApi.archiveTemplate('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/1/archive');
    });
  });

  describe('unarchiveTemplate', () => {
    it('should unarchive', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await TemplateApi.unarchiveTemplate('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/1/unarchive');
    });
  });

  describe('publishTemplate', () => {
    it('should publish', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await TemplateApi.publishTemplate('1', 'v1.0 release');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/1/publish', { changelog: 'v1.0 release' });
    });
  });

  describe('createDraft', () => {
    it('should create draft', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await TemplateApi.createDraft('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/1/draft');
    });
  });

  describe('getTemplateVersions', () => {
    it('should get versions', async () => {
      mockGet.mockResolvedValue([]);
      await TemplateApi.getTemplateVersions('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/1/versions');
    });
  });

  describe('rollbackToVersion', () => {
    it('should rollback', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await TemplateApi.rollbackToVersion('1', 'v2');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/1/rollback', { version: 'v2' });
    });
  });

  describe('compareVersions', () => {
    it('should compare versions', async () => {
      mockGet.mockResolvedValue({ diff: [] });
      await TemplateApi.compareVersions('1', 'v1', 'v2');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/1/compare', { versionA: 'v1', versionB: 'v2' });
    });
  });

  describe('createTicketFromTemplate', () => {
    it('should create ticket from template', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await TemplateApi.createTicketFromTemplate({ templateId: '1' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/create-ticket', { templateId: '1' });
    });
  });

  describe('getCategories', () => {
    it('should get categories', async () => {
      mockGet.mockResolvedValue([]);
      await TemplateApi.getCategories();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/template-categories');
    });
  });

  describe('createCategory', () => {
    it('should create category', async () => {
      mockPost.mockResolvedValue({ id: '1', name: 'IT' });
      await TemplateApi.createCategory({ name: 'IT' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/template-categories', { name: 'IT' });
    });
  });

  describe('updateCategory', () => {
    it('should update category', async () => {
      mockPatch.mockResolvedValue({ id: '1' });
      await TemplateApi.updateCategory('1', { name: 'Updated' });
      expect(mockPatch).toHaveBeenCalledWith('/api/v1/template-categories/1', { name: 'Updated' });
    });
  });

  describe('deleteCategory', () => {
    it('should delete category', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TemplateApi.deleteCategory('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/template-categories/1');
    });
  });

  describe('getTemplateStats', () => {
    it('should get stats', async () => {
      mockGet.mockResolvedValue({ usageCount: 10 });
      await TemplateApi.getTemplateStats('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/1/stats');
    });
  });

  describe('recordTemplateUsage', () => {
    it('should record usage', async () => {
      mockPost.mockResolvedValue(undefined);
      await TemplateApi.recordTemplateUsage('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/1/use');
    });
  });

  describe('getRecentTemplates', () => {
    it('should get recent templates', async () => {
      mockGet.mockResolvedValue([]);
      await TemplateApi.getRecentTemplates(5);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/recent', { limit: 5 });
    });
  });

  describe('getPopularTemplates', () => {
    it('should get popular templates', async () => {
      mockGet.mockResolvedValue([]);
      await TemplateApi.getPopularTemplates(5);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/popular', { limit: 5 });
    });
  });

  describe('rateTemplate', () => {
    it('should rate template', async () => {
      mockPost.mockResolvedValue({ rating: 5 });
      await TemplateApi.rateTemplate('1', 5, 'great');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/1/rate', { rating: 5, comment: 'great' });
    });
  });

  describe('getTemplateRatings', () => {
    it('should get ratings', async () => {
      mockGet.mockResolvedValue([]);
      await TemplateApi.getTemplateRatings('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/1/ratings');
    });
  });

  describe('duplicateTemplate', () => {
    it('should duplicate', async () => {
      mockPost.mockResolvedValue({ id: '2' });
      await TemplateApi.duplicateTemplate({ templateId: '1', name: 'Copy' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/duplicate', { templateId: '1', name: 'Copy' });
    });
  });

  describe('exportTemplate', () => {
    it('should export', async () => {
      const blob = new Blob(['data']);
      mockRequest.mockResolvedValue(blob);
      await TemplateApi.exportTemplate('1', { format: 'json' } as any);
      expect(mockRequest).toHaveBeenCalledWith(expect.objectContaining({ method: 'GET', url: '/api/v1/templates/1/export', responseType: 'blob' }));
    });
  });

  describe('validateTemplate', () => {
    it('should validate', async () => {
      mockPost.mockResolvedValue({ isValid: true });
      await TemplateApi.validateTemplate({ name: 'Test' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/validate', { name: 'Test' });
    });
  });

  describe('checkTemplateName', () => {
    it('should check name availability', async () => {
      mockGet.mockResolvedValue({ available: true });
      await TemplateApi.checkTemplateName('NewName', '1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/check-name', { name: 'NewName', excludeId: '1' });
    });
  });

  describe('batchToggleTemplates', () => {
    it('should batch toggle', async () => {
      mockPost.mockResolvedValue({ success: 2, failed: 0 });
      await TemplateApi.batchToggleTemplates(['1', '2'], true);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/batch/toggle', { templateIds: ['1', '2'], isActive: true });
    });
  });

  describe('batchDeleteTemplates', () => {
    it('should batch delete', async () => {
      mockRequest.mockResolvedValue({ success: 2, failed: 0 });
      await TemplateApi.batchDeleteTemplates(['1', '2']);
      expect(mockRequest).toHaveBeenCalledWith({ method: 'DELETE', url: '/api/v1/templates/batch', data: { templateIds: ['1', '2'] } });
    });
  });

  describe('searchTemplates', () => {
    it('should search', async () => {
      mockGet.mockResolvedValue({ templates: [], total: 0 });
      await TemplateApi.searchTemplates('test', { page: 1 } as any);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/search', { q: 'test', page: 1 });
    });
  });

  describe('getSmartRecommendations', () => {
    it('should get recommendations', async () => {
      mockPost.mockResolvedValue([]);
      await TemplateApi.getSmartRecommendations({ ticketType: 'incident' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/smart-recommend', { ticketType: 'incident' });
    });
  });

  describe('favoriteTemplate', () => {
    it('should favorite', async () => {
      mockPost.mockResolvedValue(undefined);
      await TemplateApi.favoriteTemplate('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/1/favorite');
    });
  });

  describe('unfavoriteTemplate', () => {
    it('should unfavorite', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TemplateApi.unfavoriteTemplate('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/templates/1/favorite');
    });
  });

  describe('getFavoriteTemplates', () => {
    it('should get favorites', async () => {
      mockGet.mockResolvedValue([]);
      await TemplateApi.getFavoriteTemplates();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/favorites');
    });
  });

  describe('isFavorite', () => {
    it('should check favorite status', async () => {
      mockGet.mockResolvedValue({ isFavorite: true });
      const result = await TemplateApi.isFavorite('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/1/favorite/status');
      expect(result).toBe(true);
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Not found'));
      await expect(TemplateApi.getTemplate('999')).rejects.toThrow('Not found');
    });
  });

  describe('previewTicketFromTemplate', () => {
    it('should preview ticket from template', async () => {
      mockPost.mockResolvedValue({ title: 'Preview', priority: 'high' });
      await TemplateApi.previewTicketFromTemplate({ templateId: '1' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/preview-ticket', { templateId: '1' });
    });
  });

  describe('getRecommendedTemplates', () => {
    it('should get recommended templates with userId', async () => {
      mockGet.mockResolvedValue([]);
      await TemplateApi.getRecommendedTemplates('user1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/recommended', { userId: 'user1' });
    });

    it('should get recommended templates without userId', async () => {
      mockGet.mockResolvedValue([]);
      await TemplateApi.getRecommendedTemplates();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/recommended', { userId: undefined });
    });
  });

  describe('getUserRating', () => {
    it('should get user rating', async () => {
      mockGet.mockResolvedValue({ rating: 4 });
      const result = await TemplateApi.getUserRating('1', 'user1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/1/ratings/user1');
    });
  });

  describe('exportTemplates', () => {
    it('should batch export templates', async () => {
      const blob = new Blob(['data']);
      mockRequest.mockResolvedValue(blob);
      await TemplateApi.exportTemplates(['1', '2'], { format: 'json' } as any);
      expect(mockRequest).toHaveBeenCalledWith(expect.objectContaining({ method: 'POST', url: '/api/v1/templates/export/batch' }));
    });
  });

  describe('importTemplate', () => {
    it('should import template with File', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      const file = new File(['{}'], 'template.json');
      await TemplateApi.importTemplate({ data: file, format: 'json', overwriteExisting: true, validateOnly: false });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/import', expect.any(FormData));
    });

    it('should import template with string data', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await TemplateApi.importTemplate({ data: '{"name":"test"}', format: 'json' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/import', expect.any(FormData));
    });
  });

  describe('batchArchiveTemplates', () => {
    it('should batch archive', async () => {
      mockPost.mockResolvedValue({ success: 2, failed: 0 });
      await TemplateApi.batchArchiveTemplates(['1', '2']);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/batch/archive', { templateIds: ['1', '2'] });
    });
  });

  describe('batchUpdateCategory', () => {
    it('should batch update category', async () => {
      mockPost.mockResolvedValue({ success: 2, failed: 0 });
      await TemplateApi.batchUpdateCategory(['1', '2'], 'cat1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/batch/update-category', { templateIds: ['1', '2'], categoryId: 'cat1' });
    });
  });

  describe('getFieldSuggestions', () => {
    it('should get field suggestions', async () => {
      mockGet.mockResolvedValue([{ name: 'title', type: 'string', label: 'Title' }]);
      await TemplateApi.getFieldSuggestions('cat1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/field-suggestions', { categoryId: 'cat1' });
    });
  });

  describe('getCommonFields', () => {
    it('should get common fields', async () => {
      mockGet.mockResolvedValue([]);
      await TemplateApi.getCommonFields();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/templates/common-fields');
    });
  });

  describe('generatePreview', () => {
    it('should generate preview', async () => {
      mockPost.mockResolvedValue({ html: '<div>Preview</div>' });
      await TemplateApi.generatePreview('1', { title: 'Test' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/1/preview', { sampleData: { title: 'Test' } });
    });
  });

  describe('testAutomation', () => {
    it('should test automation', async () => {
      mockPost.mockResolvedValue({ autoAssign: {}, autoNotify: {}, autoTag: {}, approvalWorkflow: {} });
      await TemplateApi.testAutomation('1', { title: 'Test' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/templates/1/test-automation', { testData: { title: 'Test' } });
    });
  });
});
