import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useArticlesQuery, useArticleQuery, useCreateArticleMutation, KNOWLEDGE_BASE_KEYS } from '../useKnowledgeBase';

// Mock antd
jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    error: jest.fn(),
  },
}));

// Mock KnowledgeBaseApi
jest.mock('@/lib/api/knowledge-base-api', () => ({
  KnowledgeBaseApi: {
    getArticles: jest.fn(),
    getArticle: jest.fn(),
    getArticleBySlug: jest.fn(),
    createArticle: jest.fn(),
    updateArticle: jest.fn(),
    deleteArticle: jest.fn(),
    publishArticle: jest.fn(),
    getArticleVersions: jest.fn(),
    getComments: jest.fn(),
    addComment: jest.fn(),
    getArticleAnalytics: jest.fn(),
    getCategories: jest.fn(),
    getTags: jest.fn(),
    search: jest.fn(),
    getRecommendations: jest.fn(),
    getPopularArticles: jest.fn(),
    getRecentArticles: jest.fn(),
    getStats: jest.fn(),
    likeArticle: jest.fn(),
    bookmarkArticle: jest.fn(),
    uploadImage: jest.fn(),
    submitForReview: jest.fn(),
    reviewArticle: jest.fn(),
  },
}));

import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
const mockKnowledgeBaseApi = KnowledgeBaseApi as jest.Mocked<typeof KnowledgeBaseApi>;

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('useKnowledgeBase hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('KNOWLEDGE_BASE_KEYS', () => {
    it('should generate correct query keys', () => {
      expect(KNOWLEDGE_BASE_KEYS.all).toEqual(['knowledge']);
      expect(KNOWLEDGE_BASE_KEYS.articles()).toEqual(['knowledge', 'articles']);
      expect(KNOWLEDGE_BASE_KEYS.articleDetail('art-1')).toEqual(['knowledge', 'articles', 'detail', 'art-1']);
    });
  });

  describe('useArticlesQuery', () => {
    it('should fetch articles', async () => {
      mockKnowledgeBaseApi.getArticles.mockResolvedValue({ items: [], total: 0 });

      const { result } = renderHook(() => useArticlesQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockKnowledgeBaseApi.getArticles).toHaveBeenCalled();
    });
  });

  describe('useArticleQuery', () => {
    it('should fetch a single article', async () => {
      mockKnowledgeBaseApi.getArticle.mockResolvedValue({ id: 'art-1', title: 'Test Article' });

      const { result } = renderHook(() => useArticleQuery('art-1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockKnowledgeBaseApi.getArticle).toHaveBeenCalledWith('art-1');
    });

    it('should not fetch when disabled', () => {
      renderHook(() => useArticleQuery('art-1', false), {
        wrapper: createWrapper(),
      });

      expect(mockKnowledgeBaseApi.getArticle).not.toHaveBeenCalled();
    });
  });

  describe('useCreateArticleMutation', () => {
    it('should create an article', async () => {
      mockKnowledgeBaseApi.createArticle.mockResolvedValue({ id: 'new-art' });

      const { result } = renderHook(() => useCreateArticleMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ title: 'New Article', content: 'content' } as any);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockKnowledgeBaseApi.createArticle).toHaveBeenCalledWith({ title: 'New Article', content: 'content' }, expect.anything());
    });
  });
});
