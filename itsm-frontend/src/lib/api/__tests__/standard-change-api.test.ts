import { StandardChangeApi } from '../standard-change-api';
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
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;

describe('StandardChangeApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getTemplates', () => {
    it('should get templates without params', async () => {
      mockGet.mockResolvedValue({ total: 0, templates: [], page: 1, pageSize: 20 });
      await StandardChangeApi.getTemplates();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/standard-changes', undefined);
    });

    it('should get templates with params', async () => {
      mockGet.mockResolvedValue({ total: 1, templates: [{ id: 1 }], page: 1, pageSize: 10 });
      await StandardChangeApi.getTemplates({ page: 1, pageSize: 10, category: 'network', activeOnly: true });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/standard-changes', { page: 1, pageSize: 10, category: 'network', activeOnly: true });
    });
  });

  describe('getTemplate', () => {
    it('should get template by id', async () => {
      mockGet.mockResolvedValue({ id: 1, title: 'Standard Network Change' });
      const result = await StandardChangeApi.getTemplate(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/standard-changes/1');
      expect(result.title).toBe('Standard Network Change');
    });
  });

  describe('createTemplate', () => {
    it('should create template', async () => {
      const data = { title: 'New Template', implementationPlan: 'Plan', rollbackPlan: 'Rollback' };
      mockPost.mockResolvedValue({ id: 1, ...data });
      await StandardChangeApi.createTemplate(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/standard-changes', data);
    });
  });

  describe('updateTemplate', () => {
    it('should update template', async () => {
      const data = { title: 'Updated Title' };
      mockPut.mockResolvedValue({ id: 1, ...data });
      await StandardChangeApi.updateTemplate(1, data);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/standard-changes/1', data);
    });
  });

  describe('deleteTemplate', () => {
    it('should delete template', async () => {
      mockDelete.mockResolvedValue(undefined);
      await StandardChangeApi.deleteTemplate(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/standard-changes/1');
    });
  });

  describe('getCategories', () => {
    it('should get categories', async () => {
      mockGet.mockResolvedValue({ categories: ['network', 'server', 'application'] });
      const result = await StandardChangeApi.getCategories();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/standard-changes/categories');
      expect(result.categories).toContain('network');
    });
  });

  describe('instantiate', () => {
    it('should instantiate from template', async () => {
      mockPost.mockResolvedValue({ changeId: 42 });
      const result = await StandardChangeApi.instantiate(1, { title: 'My Change', plannedStartDate: '2024-02-01' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/standard-changes/1/instantiate', { title: 'My Change', plannedStartDate: '2024-02-01' });
      expect(result.changeId).toBe(42);
    });

    it('should instantiate without optional data', async () => {
      mockPost.mockResolvedValue({ changeId: 43 });
      await StandardChangeApi.instantiate(1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/standard-changes/1/instantiate', {});
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Template not found'));
      await expect(StandardChangeApi.getTemplate(999)).rejects.toThrow('Template not found');
    });
  });
});
