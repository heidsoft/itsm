/**
 * 知识库 React Query Hooks
 */

import { useMutation, useQuery, useQueryClient } from '@tantml:react-query';
import { message } from 'antd';
import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
import type {
  ArticleQuery,
  KnowledgeSearchRequest,
} from '@/types/knowledge-base';

export const KNOWLEDGE_BASE_KEYS = {
  all: ['knowledge'] as const,
  articles: () => [...KNOWLEDGE_BASE_KEYS.all, 'articles'] as const,
  articleList: (query?: ArticleQuery) =>
    [...KNOWLEDGE_BASE_KEYS.articles(), 'list', query] as const,
  articleDetail: (id: string) =>
    [...KNOWLEDGE_BASE_KEYS.articles(), 'detail', id] as const,
  articleVersions: (id: string) =>
    [...KNOWLEDGE_BASE_KEYS.articles(), 'versions', id] as const,
  articleComments: (id: string) =>
    [...KNOWLEDGE_BASE_KEYS.articles(), 'comments', id] as const,
  articleAnalytics: (id: string) =>
    [...KNOWLEDGE_BASE_KEYS.articles(), 'analytics', id] as const,
  categories: () => [...KNOWLEDGE_BASE_KEYS.all, 'categories'] as const,
  tags: () => [...KNOWLEDGE_BASE_KEYS.all, 'tags'] as const,
  search: (query: KnowledgeSearchRequest) =>
    [...KNOWLEDGE_BASE_KEYS.all, 'search', query] as const,
  recommendations: (articleId?: string) =>
    [...KNOWLEDGE_BASE_KEYS.all, 'recommendations', articleId] as const,
  popular: () => [...KNOWLEDGE_BASE_KEYS.all, 'popular'] as const,
  recent: () => [...KNOWLEDGE_BASE_KEYS.all, 'recent'] as const,
  stats: () => [...KNOWLEDGE_BASE_KEYS.all, 'stats'] as const,
};

// Query Hooks
export function useArticlesQuery(query?: ArticleQuery) {
  return useQuery({
    queryKey: KNOWLEDGE_BASE_KEYS.articleList(query),
    queryFn: () => KnowledgeBaseApi.getArticles(query),
    staleTime: 60000,
  });
}

export function useArticleQuery(id: string, enabled = true) {
  return useQuery({
    queryKey: KNOWLEDGE_BASE_KEYS.articleDetail(id),
    queryFn: () => KnowledgeBaseApi.getArticle(id),
    enabled: enabled && !!id,
    staleTime: 300000,
  });
}

export function useArticleBySlugQuery(slug: string, enabled = true) {
  return useQuery({
    queryKey: [...KNOWLEDGE_BASE_KEYS.articles(), 'slug', slug],
    queryFn: () => KnowledgeBaseApi.getArticleBySlug(slug),
    enabled: enabled && !!slug,
    staleTime: 300000,
  });
}

export function useArticleVersionsQuery(articleId: string, enabled = true) {
  return useQuery({
    queryKey: KNOWLEDGE_BASE_KEYS.articleVersions(articleId),
    queryFn: () => KnowledgeBaseApi.getArticleVersions(articleId),
    enabled: enabled && !!articleId,
    staleTime: 300000,
  });
}

export function useArticleCommentsQuery(
  articleId: string,
  params?: { page?: number; pageSize?: number },
  enabled = true
) {
  return useQuery({
    queryKey: [...KNOWLEDGE_BASE_KEYS.articleComments(articleId), params],
    queryFn: () => KnowledgeBaseApi.getComments(articleId, params),
    enabled: enabled && !!articleId,
    staleTime: 60000,
  });
}

export function useArticleAnalyticsQuery(
  articleId: string,
  params?: { startDate?: string; endDate?: string },
  enabled = true
) {
  return useQuery({
    queryKey: [...KNOWLEDGE_BASE_KEYS.articleAnalytics(articleId), params],
    queryFn: () => KnowledgeBaseApi.getArticleAnalytics(articleId, params),
    enabled: enabled && !!articleId,
    staleTime: 300000,
  });
}

export function useCategoriesQuery() {
  return useQuery({
    queryKey: KNOWLEDGE_BASE_KEYS.categories(),
    queryFn: () => KnowledgeBaseApi.getCategories(),
    staleTime: 600000,
  });
}

export function useTagsQuery() {
  return useQuery({
    queryKey: KNOWLEDGE_BASE_KEYS.tags(),
    queryFn: () => KnowledgeBaseApi.getTags(),
    staleTime: 600000,
  });
}

export function useKnowledgeSearchQuery(
  request: KnowledgeSearchRequest,
  enabled = true
) {
  return useQuery({
    queryKey: KNOWLEDGE_BASE_KEYS.search(request),
    queryFn: () => KnowledgeBaseApi.search(request),
    enabled: enabled && !!request.query,
    staleTime: 60000,
  });
}

export function useRecommendationsQuery(articleId?: string) {
  return useQuery({
    queryKey: KNOWLEDGE_BASE_KEYS.recommendations(articleId),
    queryFn: () => KnowledgeBaseApi.getRecommendations(articleId),
    staleTime: 300000,
  });
}

export function usePopularArticlesQuery(params?: {
  categoryId?: string;
  period?: 'day' | 'week' | 'month';
  limit?: number;
}) {
  return useQuery({
    queryKey: [...KNOWLEDGE_BASE_KEYS.popular(), params],
    queryFn: () => KnowledgeBaseApi.getPopularArticles(params),
    staleTime: 300000,
  });
}

export function useRecentArticlesQuery(params?: {
  categoryId?: string;
  limit?: number;
}) {
  return useQuery({
    queryKey: [...KNOWLEDGE_BASE_KEYS.recent(), params],
    queryFn: () => KnowledgeBaseApi.getRecentArticles(params),
    staleTime: 60000,
  });
}

export function useKnowledgeStatsQuery() {
  return useQuery({
    queryKey: KNOWLEDGE_BASE_KEYS.stats(),
    queryFn: () => KnowledgeBaseApi.getStats(),
    staleTime: 300000,
  });
}

// Mutation Hooks
export function useCreateArticleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: KnowledgeBaseApi.createArticle,
    onSuccess: () => {
      message.success('文章已创建');
      queryClient.invalidateQueries({
        queryKey: KNOWLEDGE_BASE_KEYS.articles(),
      });
      queryClient.invalidateQueries({
        queryKey: KNOWLEDGE_BASE_KEYS.stats(),
      });
    },
  });
}

export function useUpdateArticleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: Parameters<typeof KnowledgeBaseApi.updateArticle>[1];
    }) => KnowledgeBaseApi.updateArticle(id, data),
    onSuccess: (_, variables) => {
      message.success('文章已更新');
      queryClient.invalidateQueries({
        queryKey: KNOWLEDGE_BASE_KEYS.articleDetail(variables.id),
      });
      queryClient.invalidateQueries({
        queryKey: KNOWLEDGE_BASE_KEYS.articles(),
      });
    },
  });
}

export function useDeleteArticleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => KnowledgeBaseApi.deleteArticle(id),
    onSuccess: () => {
      message.success('文章已删除');
      queryClient.invalidateQueries({
        queryKey: KNOWLEDGE_BASE_KEYS.articles(),
      });
      queryClient.invalidateQueries({
        queryKey: KNOWLEDGE_BASE_KEYS.stats(),
      });
    },
  });
}

export function usePublishArticleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      request,
    }: {
      id: string;
      request?: Parameters<typeof KnowledgeBaseApi.publishArticle>[1];
    }) => KnowledgeBaseApi.publishArticle(id, request),
    onSuccess: (_, variables) => {
      message.success('文章已发布');
      queryClient.invalidateQueries({
        queryKey: KNOWLEDGE_BASE_KEYS.articleDetail(variables.id),
      });
      queryClient.invalidateQueries({
        queryKey: KNOWLEDGE_BASE_KEYS.articles(),
      });
    },
  });
}

export function useAddCommentMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      articleId,
      content,
      parentId,
    }: {
      articleId: string;
      content: string;
      parentId?: string;
    }) => KnowledgeBaseApi.addComment(articleId, content, parentId),
    onSuccess: (_, variables) => {
      message.success('评论已添加');
      queryClient.invalidateQueries({
        queryKey: KNOWLEDGE_BASE_KEYS.articleComments(variables.articleId),
      });
    },
  });
}

export function useLikeArticleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (articleId: string) => KnowledgeBaseApi.likeArticle(articleId),
    onSuccess: (_, articleId) => {
      queryClient.invalidateQueries({
        queryKey: KNOWLEDGE_BASE_KEYS.articleDetail(articleId),
      });
    },
  });
}

export function useBookmarkArticleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (articleId: string) =>
      KnowledgeBaseApi.bookmarkArticle(articleId),
    onSuccess: () => {
      message.success('已添加到收藏');
    },
  });
}

export function useUploadImageMutation() {
  return useMutation({
    mutationFn: (file: File) => KnowledgeBaseApi.uploadImage(file),
    onError: () => {
      message.error('图片上传失败');
    },
  });
}

export function useSubmitForReviewMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      articleId,
      reviewerId,
    }: {
      articleId: string;
      reviewerId?: number;
    }) => KnowledgeBaseApi.submitForReview(articleId, reviewerId),
    onSuccess: (_, variables) => {
      message.success('已提交审核');
      queryClient.invalidateQueries({
        queryKey: KNOWLEDGE_BASE_KEYS.articleDetail(variables.articleId),
      });
    },
  });
}

export function useReviewArticleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      articleId,
      request,
    }: {
      articleId: string;
      request: Parameters<typeof KnowledgeBaseApi.reviewArticle>[1];
    }) => KnowledgeBaseApi.reviewArticle(articleId, request),
    onSuccess: (_, variables) => {
      message.success('审核已完成');
      queryClient.invalidateQueries({
        queryKey: KNOWLEDGE_BASE_KEYS.articleDetail(variables.articleId),
      });
      queryClient.invalidateQueries({
        queryKey: KNOWLEDGE_BASE_KEYS.articles(),
      });
    },
  });
}

export default {
  useArticlesQuery,
  useArticleQuery,
  useArticleBySlugQuery,
  useArticleVersionsQuery,
  useArticleCommentsQuery,
  useArticleAnalyticsQuery,
  useCategoriesQuery,
  useTagsQuery,
  useKnowledgeSearchQuery,
  useRecommendationsQuery,
  usePopularArticlesQuery,
  useRecentArticlesQuery,
  useKnowledgeStatsQuery,
  useCreateArticleMutation,
  useUpdateArticleMutation,
  useDeleteArticleMutation,
  usePublishArticleMutation,
  useAddCommentMutation,
  useLikeArticleMutation,
  useBookmarkArticleMutation,
  useUploadImageMutation,
  useSubmitForReviewMutation,
  useReviewArticleMutation,
};

