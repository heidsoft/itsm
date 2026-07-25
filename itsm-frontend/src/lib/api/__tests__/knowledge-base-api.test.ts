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

  describe('getArticleBySlug', () => {
    it('should get article by slug', async () => {
      mockGet.mockResolvedValue({ id: '1', slug: 'my-article' });
      await KnowledgeBaseApi.getArticleBySlug('my-article');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/knowledge/articles/slug/my-article');
    });
  });

  describe('unpublishArticle', () => {
    it('should unpublish article', async () => {
      mockPost.mockResolvedValue({ id: '1', status: 'draft' });
      await KnowledgeBaseApi.unpublishArticle('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/unpublish');
    });
  });

  describe('archiveArticle', () => {
    it('should archive article', async () => {
      mockPost.mockResolvedValue({ id: '1', status: 'archived' });
      await KnowledgeBaseApi.archiveArticle('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/archive');
    });
  });

  describe('cloneArticle', () => {
    it('should clone article', async () => {
      mockPost.mockResolvedValue({ id: '2', title: 'Clone' });
      await KnowledgeBaseApi.cloneArticle('1', 'Clone');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/clone', { title: 'Clone' });
    });
  });

  describe('batchOperation', () => {
    it('should perform batch operation', async () => {
      mockPost.mockResolvedValue({ success: 3, failed: 0 });
      await KnowledgeBaseApi.batchOperation({ action: 'publish', articleIds: ['1', '2', '3'] } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/batch', expect.any(Object));
    });
  });

  describe('getArticleVersions', () => {
    it('should get versions', async () => {
      mockGet.mockResolvedValue([{ version: 1 }, { version: 2 }]);
      const result = await KnowledgeBaseApi.getArticleVersions('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/versions');
      expect(result).toHaveLength(2);
    });
  });

  describe('restoreVersion', () => {
    it('should restore version', async () => {
      mockPost.mockResolvedValue({ id: '1', version: 1 });
      await KnowledgeBaseApi.restoreVersion('1', 1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/versions/1/restore');
    });
  });

  describe('compareVersions', () => {
    it('should compare versions', async () => {
      mockGet.mockResolvedValue({ diff: 'changes', changes: [] });
      await KnowledgeBaseApi.compareVersions('1', 1, 2);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/versions/compare', { from: 1, to: 2 });
    });
  });

  describe('createCategory', () => {
    it('should create category', async () => {
      mockPost.mockResolvedValue({ id: '1', name: 'New' });
      await KnowledgeBaseApi.createCategory({ name: 'New' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/categories', { name: 'New' });
    });
  });

  describe('updateCategory', () => {
    it('should update category', async () => {
      mockPut.mockResolvedValue({ id: '1', name: 'Updated' });
      await KnowledgeBaseApi.updateCategory('1', { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/knowledge/categories/1', { name: 'Updated' });
    });
  });

  describe('deleteCategory', () => {
    it('should delete category', async () => {
      mockDelete.mockResolvedValue(undefined);
      await KnowledgeBaseApi.deleteCategory('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/knowledge/categories/1');
    });
  });

  describe('createTag', () => {
    it('should create tag', async () => {
      mockPost.mockResolvedValue({ id: '1', name: 'howto' });
      await KnowledgeBaseApi.createTag({ name: 'howto' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/tags', { name: 'howto' });
    });
  });

  describe('getComments', () => {
    it('should get comments', async () => {
      mockGet.mockResolvedValue({ comments: [], total: 0 });
      await KnowledgeBaseApi.getComments('1', { page: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/comments', { page: 1 });
    });
  });

  describe('addComment', () => {
    it('should add comment', async () => {
      mockPost.mockResolvedValue({ id: '1', content: 'Great!' });
      await KnowledgeBaseApi.addComment('1', 'Great!', 'parent1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/comments', { content: 'Great!', parentId: 'parent1' });
    });
  });

  describe('deleteComment', () => {
    it('should delete comment', async () => {
      mockDelete.mockResolvedValue(undefined);
      await KnowledgeBaseApi.deleteComment('c1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/knowledge/articles/comments/c1');
    });
  });

  describe('markCommentHelpful', () => {
    it('should mark comment as helpful', async () => {
      mockPost.mockResolvedValue(undefined);
      await KnowledgeBaseApi.markCommentHelpful('c1', true);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/comments/c1/helpful', { helpful: true });
    });
  });

  describe('submitFeedback', () => {
    it('should submit feedback', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await KnowledgeBaseApi.submitFeedback('1', { rating: 5, comment: 'Helpful' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/feedback', { rating: 5, comment: 'Helpful' });
    });
  });

  describe('bookmarkArticle', () => {
    it('should bookmark', async () => {
      mockPost.mockResolvedValue(undefined);
      await KnowledgeBaseApi.bookmarkArticle('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/bookmark');
    });
  });

  describe('unbookmarkArticle', () => {
    it('should unbookmark', async () => {
      mockDelete.mockResolvedValue(undefined);
      await KnowledgeBaseApi.unbookmarkArticle('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/bookmark');
    });
  });

  describe('shareArticle', () => {
    it('should share article', async () => {
      mockPost.mockResolvedValue(undefined);
      await KnowledgeBaseApi.shareArticle('1', 'email');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/share', { platform: 'email' });
    });
  });

  describe('recordView', () => {
    it('should record view', async () => {
      mockPost.mockResolvedValue(undefined);
      await KnowledgeBaseApi.recordView('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/view');
    });
  });

  describe('getPopularArticles', () => {
    it('should get popular articles', async () => {
      mockGet.mockResolvedValue([]);
      await KnowledgeBaseApi.getPopularArticles({ period: 'week', limit: 10 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/knowledge/popular', { period: 'week', limit: 10 });
    });
  });

  describe('getRecentArticles', () => {
    it('should get recent articles', async () => {
      mockGet.mockResolvedValue([]);
      await KnowledgeBaseApi.getRecentArticles({ limit: 5 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/knowledge/recent', { limit: 5 });
    });
  });

  describe('uploadImage', () => {
    it('should upload image with FormData', async () => {
      mockPost.mockResolvedValue({ url: 'https://cdn.example.com/img.png' });
      const file = new File(['data'], 'test.png', { type: 'image/png' });
      await KnowledgeBaseApi.uploadImage(file);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/upload/image', expect.any(FormData), { headers: { 'Content-Type': 'multipart/form-data' } });
    });
  });

  describe('autoSaveDraft', () => {
    it('should auto-save draft', async () => {
      mockPost.mockResolvedValue(undefined);
      await KnowledgeBaseApi.autoSaveDraft('1', '<p>content</p>');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/autosave', { content: '<p>content</p>' });
    });
  });

  describe('submitForReview', () => {
    it('should submit for review', async () => {
      mockPost.mockResolvedValue({ id: '1', status: 'in_review' });
      await KnowledgeBaseApi.submitForReview('1', 5);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/review', { reviewerId: 5 });
    });
  });

  describe('reviewArticle', () => {
    it('should review article', async () => {
      mockPost.mockResolvedValue({ id: '1', status: 'published' });
      await KnowledgeBaseApi.reviewArticle('1', { approved: true } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/review/decision', { approved: true });
    });
  });

  describe('getArticleAnalytics', () => {
    it('should get analytics', async () => {
      mockGet.mockResolvedValue({ views: 100, likes: 10 });
      await KnowledgeBaseApi.getArticleAnalytics('1', { startDate: '2024-01-01' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/knowledge/articles/1/analytics', { startDate: '2024-01-01' });
    });
  });

  describe('exportKnowledgeBase', () => {
    it('should export knowledge base', async () => {
      const blob = new Blob(['pdf-data']);
      mockRequest.mockResolvedValue(blob);
      const result = await KnowledgeBaseApi.exportKnowledgeBase({ format: 'pdf' });
      expect(mockRequest).toHaveBeenCalledWith(expect.objectContaining({ method: 'POST', url: '/api/v1/knowledge/articles/export', responseType: 'blob' }));
      expect(result).toBe(blob);
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Not found'));
      await expect(KnowledgeBaseApi.getArticle('999')).rejects.toThrow('Not found');
    });
  });
});
