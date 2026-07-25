import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
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
const mockRequest = (httpClient as any).request as jest.Mock;

describe('KnowledgeBaseApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getArticles', () => {
    it('should get articles', async () => {
      mockGet.mockResolvedValue({ articles: [{ id: '1', title: 'Guide' }], total: 1 });
      const result = await KnowledgeBaseApi.getArticles({ page: 1 } as any);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/knowledge/articles', { page: 1 });
      expect(result.articles).toHaveLength(1);
    });
  });

  describe('getArticle', () => {
    it('should get article by id', async () => {
      mockGet.mockResolvedValue({ id: '1', title: 'Guide' });
      const result = await KnowledgeBaseApi.getArticle('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/knowledge/articles/1');
    });
  });

  describe('createArticle', () => {
    it('should create an article', async () => {
      mockPost.mockResolvedValue({ id: '2', title: 'New' });
      await KnowledgeBaseApi.createArticle({ title: 'New' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles', { title: 'New' });
    });
  });

  describe('updateArticle', () => {
    it('should update an article', async () => {
      mockPut.mockResolvedValue({ id: '1', title: 'Updated' });
      await KnowledgeBaseApi.updateArticle('1', { title: 'Updated' } as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/knowledge/articles/1', { title: 'Updated' });
    });
  });

  describe('deleteArticle', () => {
    it('should delete an article', async () => {
      mockDelete.mockResolvedValue(undefined);
      await KnowledgeBaseApi.deleteArticle('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/knowledge/articles/1');
    });
  });

  describe('publishArticle', () => {
    it('should publish an article', async () => {
      mockPost.mockResolvedValue({ id: '1', status: 'published' });
      await KnowledgeBaseApi.publishArticle('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/publish', undefined);
    });
  });

  describe('getCategories', () => {
    it('should get categories', async () => {
      mockGet.mockResolvedValue([{ id: '1', name: 'FAQ' }]);
      const result = await KnowledgeBaseApi.getCategories();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/knowledge/categories');
      expect(result).toHaveLength(1);
    });
  });

  describe('getTags', () => {
    it('should get tags', async () => {
      mockGet.mockResolvedValue([{ id: '1', name: 'howto' }]);
      const result = await KnowledgeBaseApi.getTags();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/knowledge/tags');
    });
  });

  describe('search', () => {
    it('should search knowledge base', async () => {
      mockPost.mockResolvedValue({ results: [], total: 0 });
      await KnowledgeBaseApi.search({ query: 'test' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/search', { query: 'test' });
    });
  });

  describe('likeArticle', () => {
    it('should like an article', async () => {
      mockPost.mockResolvedValue(undefined);
      await KnowledgeBaseApi.likeArticle('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/like');
    });
  });

  describe('unlikeArticle', () => {
    it('should unlike an article', async () => {
      mockDelete.mockResolvedValue(undefined);
      await KnowledgeBaseApi.unlikeArticle('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/like');
    });
  });

  describe('getStats', () => {
    it('should get stats', async () => {
      mockGet.mockResolvedValue({ totalArticles: 100 });
      const result = await KnowledgeBaseApi.getStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/knowledge/stats');
    });
  });

  describe('getRecommendations', () => {
    it('should get recommendations', async () => {
      mockGet.mockResolvedValue([]);
      await KnowledgeBaseApi.getRecommendations('1', 5);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/knowledge/recommendations', { articleId: '1', limit: 5 });
    });
  });
});
