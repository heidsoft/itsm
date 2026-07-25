import { renderHook, waitFor, act } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useArticlesQuery, useArticleQuery, useArticleBySlugQuery,
  useArticleVersionsQuery, useArticleCommentsQuery, useArticleAnalyticsQuery,
  useCategoriesQuery, useTagsQuery, useKnowledgeSearchQuery,
  useRecommendationsQuery, usePopularArticlesQuery, useRecentArticlesQuery,
  useKnowledgeStatsQuery,
  useCreateArticleMutation, useUpdateArticleMutation, useDeleteArticleMutation,
  usePublishArticleMutation, useAddCommentMutation, useLikeArticleMutation,
  useBookmarkArticleMutation, useUploadImageMutation,
  useSubmitForReviewMutation, useReviewArticleMutation,
  KNOWLEDGE_BASE_KEYS,
} from '../useKnowledgeBase';

jest.mock('antd', () => ({ message: { success: jest.fn(), error: jest.fn() } }));

jest.mock('@/lib/api/knowledge-base-api', () => ({
  KnowledgeBaseApi: {
    getArticles: jest.fn(), getArticle: jest.fn(), getArticleBySlug: jest.fn(),
    createArticle: jest.fn(), updateArticle: jest.fn(), deleteArticle: jest.fn(),
    publishArticle: jest.fn(), getArticleVersions: jest.fn(), getComments: jest.fn(),
    addComment: jest.fn(), getArticleAnalytics: jest.fn(), getCategories: jest.fn(),
    getTags: jest.fn(), search: jest.fn(), getRecommendations: jest.fn(),
    getPopularArticles: jest.fn(), getRecentArticles: jest.fn(), getStats: jest.fn(),
    likeArticle: jest.fn(), bookmarkArticle: jest.fn(), uploadImage: jest.fn(),
    submitForReview: jest.fn(), reviewArticle: jest.fn(),
  },
}));

import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
import { message } from 'antd';
const mockApi = KnowledgeBaseApi as jest.Mocked<typeof KnowledgeBaseApi>;

const createWrapper = () => {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: qc }, children);
};

describe('useKnowledgeBase hooks', () => {
  beforeEach(() => jest.clearAllMocks());

  describe('KNOWLEDGE_BASE_KEYS', () => {
    it('generates all key shapes', () => {
      expect(KNOWLEDGE_BASE_KEYS.all).toEqual(['knowledge']);
      expect(KNOWLEDGE_BASE_KEYS.articles()).toEqual(['knowledge', 'articles']);
      expect(KNOWLEDGE_BASE_KEYS.articleDetail('a1')).toEqual(['knowledge', 'articles', 'detail', 'a1']);
      expect(KNOWLEDGE_BASE_KEYS.articleVersions('a1')).toEqual(['knowledge', 'articles', 'versions', 'a1']);
      expect(KNOWLEDGE_BASE_KEYS.articleComments('a1')).toEqual(['knowledge', 'articles', 'comments', 'a1']);
      expect(KNOWLEDGE_BASE_KEYS.articleAnalytics('a1')).toEqual(['knowledge', 'articles', 'analytics', 'a1']);
      expect(KNOWLEDGE_BASE_KEYS.categories()).toEqual(['knowledge', 'categories']);
      expect(KNOWLEDGE_BASE_KEYS.tags()).toEqual(['knowledge', 'tags']);
      expect(KNOWLEDGE_BASE_KEYS.popular()).toEqual(['knowledge', 'popular']);
      expect(KNOWLEDGE_BASE_KEYS.recent()).toEqual(['knowledge', 'recent']);
      expect(KNOWLEDGE_BASE_KEYS.stats()).toEqual(['knowledge', 'stats']);
    });
  });

  describe('useArticlesQuery', () => {
    it('fetches articles', async () => {
      mockApi.getArticles.mockResolvedValue({ items: [], total: 0 } as any);
      const { result } = renderHook(() => useArticlesQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useArticleQuery', () => {
    it('fetches article by id', async () => {
      mockApi.getArticle.mockResolvedValue({ id: 'a1' } as any);
      const { result } = renderHook(() => useArticleQuery('a1'), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useArticleQuery('a1', false), { wrapper: createWrapper() });
      expect(mockApi.getArticle).not.toHaveBeenCalled();
    });
  });

  describe('useArticleBySlugQuery', () => {
    it('fetches article by slug', async () => {
      mockApi.getArticleBySlug.mockResolvedValue({ slug: 'my-article' } as any);
      const { result } = renderHook(() => useArticleBySlugQuery('my-article'), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getArticleBySlug).toHaveBeenCalledWith('my-article');
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useArticleBySlugQuery('x', false), { wrapper: createWrapper() });
      expect(mockApi.getArticleBySlug).not.toHaveBeenCalled();
    });
  });

  describe('useArticleVersionsQuery', () => {
    it('fetches versions', async () => {
      mockApi.getArticleVersions.mockResolvedValue([] as any);
      const { result } = renderHook(() => useArticleVersionsQuery('a1'), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useArticleVersionsQuery('a1', false), { wrapper: createWrapper() });
      expect(mockApi.getArticleVersions).not.toHaveBeenCalled();
    });
  });

  describe('useArticleCommentsQuery', () => {
    it('fetches comments', async () => {
      mockApi.getComments.mockResolvedValue({ items: [] } as any);
      const { result } = renderHook(() => useArticleCommentsQuery('a1', { page: 1 }), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useArticleCommentsQuery('a1', undefined, false), { wrapper: createWrapper() });
      expect(mockApi.getComments).not.toHaveBeenCalled();
    });
  });

  describe('useArticleAnalyticsQuery', () => {
    it('fetches analytics', async () => {
      mockApi.getArticleAnalytics.mockResolvedValue({ views: 100 } as any);
      const { result } = renderHook(() => useArticleAnalyticsQuery('a1', { startDate: '2024-01-01' }), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useArticleAnalyticsQuery('a1', undefined, false), { wrapper: createWrapper() });
      expect(mockApi.getArticleAnalytics).not.toHaveBeenCalled();
    });
  });

  describe('useCategoriesQuery', () => {
    it('fetches categories', async () => {
      mockApi.getCategories.mockResolvedValue([] as any);
      const { result } = renderHook(() => useCategoriesQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useTagsQuery', () => {
    it('fetches tags', async () => {
      mockApi.getTags.mockResolvedValue([] as any);
      const { result } = renderHook(() => useTagsQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useKnowledgeSearchQuery', () => {
    it('searches when query is provided', async () => {
      mockApi.search.mockResolvedValue({ items: [] } as any);
      const { result } = renderHook(() => useKnowledgeSearchQuery({ query: 'test' } as any), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not search when disabled', () => {
      renderHook(() => useKnowledgeSearchQuery({ query: 'test' } as any, false), { wrapper: createWrapper() });
      expect(mockApi.search).not.toHaveBeenCalled();
    });
  });

  describe('useRecommendationsQuery', () => {
    it('fetches recommendations', async () => {
      mockApi.getRecommendations.mockResolvedValue([] as any);
      const { result } = renderHook(() => useRecommendationsQuery('a1'), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('usePopularArticlesQuery', () => {
    it('fetches popular articles', async () => {
      mockApi.getPopularArticles.mockResolvedValue([] as any);
      const { result } = renderHook(() => usePopularArticlesQuery({ period: 'week' }), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useRecentArticlesQuery', () => {
    it('fetches recent articles', async () => {
      mockApi.getRecentArticles.mockResolvedValue([] as any);
      const { result } = renderHook(() => useRecentArticlesQuery({ limit: 5 }), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useKnowledgeStatsQuery', () => {
    it('fetches stats', async () => {
      mockApi.getStats.mockResolvedValue({ total: 50 } as any);
      const { result } = renderHook(() => useKnowledgeStatsQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useCreateArticleMutation', () => {
    it('creates article on success', async () => {
      mockApi.createArticle.mockResolvedValue({ id: 'new' } as any);
      const { result } = renderHook(() => useCreateArticleMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ title: 'New', content: 'body' } as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useUpdateArticleMutation', () => {
    it('updates article on success', async () => {
      mockApi.updateArticle.mockResolvedValue({ id: 'a1' } as any);
      const { result } = renderHook(() => useUpdateArticleMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ id: 'a1', data: { title: 'Updated' } } as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useDeleteArticleMutation', () => {
    it('deletes article on success', async () => {
      mockApi.deleteArticle.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useDeleteArticleMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate('a1'); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('usePublishArticleMutation', () => {
    it('publishes article on success', async () => {
      mockApi.publishArticle.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => usePublishArticleMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ id: 'a1' }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useAddCommentMutation', () => {
    it('adds comment on success', async () => {
      mockApi.addComment.mockResolvedValue({ id: 'c1' } as any);
      const { result } = renderHook(() => useAddCommentMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ articleId: 'a1', content: 'Great!' }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useLikeArticleMutation', () => {
    it('likes article on success', async () => {
      mockApi.likeArticle.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useLikeArticleMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate('a1'); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useBookmarkArticleMutation', () => {
    it('bookmarks article on success', async () => {
      mockApi.bookmarkArticle.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useBookmarkArticleMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate('a1'); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useUploadImageMutation', () => {
    it('uploads image on success', async () => {
      mockApi.uploadImage.mockResolvedValue({ url: 'http://img.jpg' } as any);
      const { result } = renderHook(() => useUploadImageMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate(new File([''], 'test.png') as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('shows error on failure', async () => {
      mockApi.uploadImage.mockRejectedValue(new Error('upload fail'));
      const { result } = renderHook(() => useUploadImageMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate(new File([''], 'x.png') as any); });
      await waitFor(() => expect(result.current.isError).toBe(true));
      expect(message.error).toHaveBeenCalled();
    });
  });

  describe('useSubmitForReviewMutation', () => {
    it('submits for review on success', async () => {
      mockApi.submitForReview.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useSubmitForReviewMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ articleId: 'a1', reviewerId: 5 }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useReviewArticleMutation', () => {
    it('reviews article on success', async () => {
      mockApi.reviewArticle.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useReviewArticleMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ articleId: 'a1', request: { approved: true, comment: 'ok' } } as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });
});
