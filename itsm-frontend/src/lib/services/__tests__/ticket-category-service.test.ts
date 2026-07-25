/**
 * TicketCategoryService unit tests
 */
import { ticketCategoryService } from '../ticket-category-service';
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
const mockDelete = httpClient.delete as jest.Mock;

describe('TicketCategoryService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('listCategories', () => {
    it('should call GET /api/v1/ticket-categories with params', async () => {
      mockGet.mockResolvedValueOnce({ categories: [], total: 0 });
      await ticketCategoryService.listCategories({ page: 1, pageSize: 10, isActive: true });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ticket-categories', { page: 1, pageSize: 10, isActive: true });
    });

    it('should use empty params by default', async () => {
      mockGet.mockResolvedValueOnce({ categories: [], total: 0 });
      await ticketCategoryService.listCategories();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ticket-categories', {});
    });
  });

  describe('getCategoryTree', () => {
    it('should call GET /api/v1/ticket-categories/tree', async () => {
      const tree = [{ id: 1, name: 'Root', children: [] }];
      mockGet.mockResolvedValueOnce(tree);
      const result = await ticketCategoryService.getCategoryTree();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ticket-categories/tree');
      expect(result).toEqual(tree);
    });
  });

  describe('getCategory', () => {
    it('should call GET /api/v1/ticket-categories/:id', async () => {
      mockGet.mockResolvedValueOnce({ id: 5, name: 'Hardware' });
      const result = await ticketCategoryService.getCategory(5);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ticket-categories/5');
      expect(result.name).toBe('Hardware');
    });
  });

  describe('createCategory', () => {
    it('should call POST /api/v1/ticket-categories', async () => {
      const data = { name: 'Network', description: 'Network issues', code: 'NET', parentId: 0, sortOrder: 1, isActive: true };
      mockPost.mockResolvedValueOnce({ id: 10, ...data });
      const result = await ticketCategoryService.createCategory(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ticket-categories', data);
      expect(result.id).toBe(10);
    });
  });

  describe('updateCategory', () => {
    it('should call PUT /api/v1/ticket-categories/:id', async () => {
      mockPut.mockResolvedValueOnce({ id: 5, name: 'Updated' });
      const result = await ticketCategoryService.updateCategory(5, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/ticket-categories/5', { name: 'Updated' });
      expect(result.name).toBe('Updated');
    });
  });

  describe('deleteCategory', () => {
    it('should call DELETE /api/v1/ticket-categories/:id', async () => {
      mockDelete.mockResolvedValueOnce(undefined);
      await ticketCategoryService.deleteCategory(3);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/ticket-categories/3');
    });
  });

  describe('moveCategory', () => {
    it('should call PUT /api/v1/ticket-categories/:id/move', async () => {
      mockPut.mockResolvedValueOnce(undefined);
      await ticketCategoryService.moveCategory(5, { newParentId: 2, newSortOrder: 3 });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/ticket-categories/5/move', { newParentId: 2, newSortOrder: 3 });
    });
  });

  describe('batchUpdateCategories', () => {
    it('should call PUT /api/v1/ticket-categories/batch-update', async () => {
      const data = [{ id: 1, sortOrder: 1, level: 0 }, { id: 2, sortOrder: 2, level: 0 }];
      mockPut.mockResolvedValueOnce(undefined);
      await ticketCategoryService.batchUpdateCategories(data);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/ticket-categories/batch-update', data);
    });
  });
});
