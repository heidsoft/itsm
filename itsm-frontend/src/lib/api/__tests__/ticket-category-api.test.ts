import { TicketCategoryApi } from '@/lib/api/ticket-category-api';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
    getTenantId: jest.fn(() => 1),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;

describe('TicketCategoryApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getCategories', () => {
    it('should fetch categories with params', async () => {
      mockGet.mockResolvedValue({ categories: [{ id: 1, name: 'Bug' }], total: 1 });
      const result = await TicketCategoryApi.getCategories({ page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ticket-categories', { page: 1, pageSize: 10 });
      expect(result.total).toBe(1);
    });
  });

  describe('getCategoryTree', () => {
    it('should fetch category tree', async () => {
      mockGet.mockResolvedValue([{ id: 1, name: 'Root', children: [] }]);
      const result = await TicketCategoryApi.getCategoryTree();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ticket-categories/tree');
      expect(result).toHaveLength(1);
    });
  });

  describe('getCategory', () => {
    it('should fetch single category', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'Bug' });
      const result = await TicketCategoryApi.getCategory(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ticket-categories/1');
      expect(result.name).toBe('Bug');
    });
  });

  describe('createCategory', () => {
    it('should create a category with tenantId attached', async () => {
      const data = { name: 'Feature', code: 'FEATURE' };
      mockPost.mockResolvedValue({ id: 2, name: 'Feature' });
      const result = await TicketCategoryApi.createCategory(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ticket-categories', { ...data, tenantId: 1 });
      expect(result.id).toBe(2);
    });
  });

  describe('updateCategory', () => {
    it('should update a category', async () => {
      mockPut.mockResolvedValue({ id: 1, name: 'Updated' });
      const result = await TicketCategoryApi.updateCategory(1, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/ticket-categories/1', { name: 'Updated' });
      expect(result.name).toBe('Updated');
    });
  });

  describe('deleteCategory', () => {
    it('should delete a category', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketCategoryApi.deleteCategory(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/ticket-categories/1');
    });
  });

  describe('previewImport', () => {
    it('should preview import data', async () => {
      const formData = new FormData();
      mockPost.mockResolvedValue([{ name: 'Cat1', code: 'C1' }]);
      const result = await TicketCategoryApi.previewImport(formData);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ticket-categories/import/preview', formData);
      expect(result).toHaveLength(1);
    });
  });

  describe('executeImport', () => {
    it('should execute import', async () => {
      const formData = new FormData();
      mockPost.mockResolvedValue({ success: 5, failed: 0 });
      const result = await TicketCategoryApi.executeImport(formData);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ticket-categories/import', formData);
      expect(result.success).toBe(5);
    });
  });
});
