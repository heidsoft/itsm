/**
 * 工单模板 React Query Hooks
 * 提供完整的模板数据管理、缓存和状态同步
 */

import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
  type UseMutationOptions,
} from '@tanstack/react-query';
import { message } from 'antd';
import { TemplateApi } from '@/lib/api/template-api';
import type {
  TicketTemplate,
  CreateTemplateRequest,
  UpdateTemplateRequest,
  TemplateListQuery,
  TemplateListResponse,
  CreateTicketFromTemplateRequest,
  TemplateUsageStats,
  TemplateDuplicateRequest,
  TemplateExportFormat,
  TemplateImportRequest,
} from '@/types/template';
import type { Ticket } from '@/types/ticket';

// ==================== Query Keys ====================

export const templateKeys = {
  all: ['templates'] as const,
  lists: () => [...templateKeys.all, 'list'] as const,
  list: (filters: TemplateListQuery) =>
    [...templateKeys.lists(), { filters }] as const,
  details: () => [...templateKeys.all, 'detail'] as const,
  detail: (id: string) => [...templateKeys.details(), id] as const,
  stats: (id: string) => [...templateKeys.all, 'stats', id] as const,
  recent: () => [...templateKeys.all, 'recent'] as const,
  popular: () => [...templateKeys.all, 'popular'] as const,
  recommended: () => [...templateKeys.all, 'recommended'] as const,
  favorites: () => [...templateKeys.all, 'favorites'] as const,
  categories: () => ['template-categories'] as const,
  ratings: (id: string) => [...templateKeys.all, 'ratings', id] as const,
  versions: (id: string) => [...templateKeys.all, 'versions', id] as const,
};

// ==================== Query Hooks ====================

/**
 * 获取模板列表
 */
export function useTemplatesQuery(
  query?: TemplateListQuery,
  options?: Omit<UseQueryOptions<TemplateListResponse>, 'queryKey' | 'queryFn'>
) {
  return useQuery({
    queryKey: templateKeys.list(query || {}),
    queryFn: () => TemplateApi.getTemplates(query),
    staleTime: 30000, // 30秒
    ...options,
  });
}

/**
 * 获取模板详情
 */
export function useTemplateQuery(
  templateId: string,
  options?: Omit<UseQueryOptions<TicketTemplate>, 'queryKey' | 'queryFn'>
) {
  return useQuery({
    queryKey: templateKeys.detail(templateId),
    queryFn: () => TemplateApi.getTemplate(templateId),
    enabled: !!templateId,
    staleTime: 60000, // 1分钟
    ...options,
  });
}

/**
 * 获取模板使用统计
 */
export function useTemplateStatsQuery(
  templateId: string,
  options?: Omit<UseQueryOptions<TemplateUsageStats>, 'queryKey' | 'queryFn'>
) {
  return useQuery({
    queryKey: templateKeys.stats(templateId),
    queryFn: () => TemplateApi.getTemplateStats(templateId),
    enabled: !!templateId,
    staleTime: 60000, // 1分钟
    ...options,
  });
}

/**
 * 获取最近使用的模板
 */
export function useRecentTemplatesQuery(
  limit = 10,
  options?: Omit<UseQueryOptions<TicketTemplate[]>, 'queryKey' | 'queryFn'>
) {
  return useQuery({
    queryKey: [...templateKeys.recent(), limit],
    queryFn: () => TemplateApi.getRecentTemplates(limit),
    staleTime: 30000, // 30秒
    ...options,
  });
}

/**
 * 获取最受欢迎的模板
 */
export function usePopularTemplatesQuery(
  limit = 10,
  options?: Omit<UseQueryOptions<TicketTemplate[]>, 'queryKey' | 'queryFn'>
) {
  return useQuery({
    queryKey: [...templateKeys.popular(), limit],
    queryFn: () => TemplateApi.getPopularTemplates(limit),
    staleTime: 60000, // 1分钟
    ...options,
  });
}

/**
 * 获取推荐模板
 */
export function useRecommendedTemplatesQuery(
  userId?: string,
  options?: Omit<UseQueryOptions<TicketTemplate[]>, 'queryKey' | 'queryFn'>
) {
  return useQuery({
    queryKey: [...templateKeys.recommended(), userId],
    queryFn: () => TemplateApi.getRecommendedTemplates(userId),
    staleTime: 60000, // 1分钟
    ...options,
  });
}

/**
 * 获取收藏的模板
 */
export function useFavoriteTemplatesQuery(
  options?: Omit<UseQueryOptions<TicketTemplate[]>, 'queryKey' | 'queryFn'>
) {
  return useQuery({
    queryKey: templateKeys.favorites(),
    queryFn: () => TemplateApi.getFavoriteTemplates(),
    staleTime: 30000, // 30秒
    ...options,
  });
}

/**
 * 获取模板分类列表
 */
export function useTemplateCategoriesQuery(
  options?: Omit<UseQueryOptions<any[]>, 'queryKey' | 'queryFn'>
) {
  return useQuery({
    queryKey: templateKeys.categories(),
    queryFn: () => TemplateApi.getCategories(),
    staleTime: 300000, // 5分钟
    ...options,
  });
}

/**
 * 获取模板评分列表
 */
export function useTemplateRatingsQuery(
  templateId: string,
  options?: Omit<UseQueryOptions<any[]>, 'queryKey' | 'queryFn'>
) {
  return useQuery({
    queryKey: templateKeys.ratings(templateId),
    queryFn: () => TemplateApi.getTemplateRatings(templateId),
    enabled: !!templateId,
    staleTime: 60000, // 1分钟
    ...options,
  });
}

/**
 * 获取模板版本历史
 */
export function useTemplateVersionsQuery(
  templateId: string,
  options?: Omit<UseQueryOptions<any[]>, 'queryKey' | 'queryFn'>
) {
  return useQuery({
    queryKey: templateKeys.versions(templateId),
    queryFn: () => TemplateApi.getTemplateVersions(templateId),
    enabled: !!templateId,
    staleTime: 60000, // 1分钟
    ...options,
  });
}

// ==================== Mutation Hooks ====================

/**
 * 创建模板
 */
export function useCreateTemplateMutation(
  options?: UseMutationOptions<
    TicketTemplate,
    Error,
    CreateTemplateRequest
  >
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateTemplateRequest) =>
      TemplateApi.createTemplate(data),
    onSuccess: (data, variables, context) => {
      // 刷新模板列表
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
      
      message.success('模板创建成功！');
      
      options?.onSuccess?.(data, variables, context, undefined as any);
    },
    onError: (error, variables, context) => {
      message.error(`创建失败：${error.message}`);
      options?.onError?.(error, variables, context, undefined as any);
    },
    ...options,
  });
}

/**
 * 更新模板
 */
export function useUpdateTemplateMutation(
  options?: UseMutationOptions<
    TicketTemplate,
    Error,
    { id: string; data: UpdateTemplateRequest }
  >
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateTemplateRequest }) =>
      TemplateApi.updateTemplate(id, data),
    onSuccess: (data, variables, context) => {
      // 更新缓存
      queryClient.setQueryData(templateKeys.detail(variables.id), data);
      
      // 刷新列表
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
      
      message.success('模板更新成功！');
      
      options?.onSuccess?.(data, variables, context, undefined as any);
    },
    onError: (error, variables, context) => {
      message.error(`更新失败：${error.message}`);
      options?.onError?.(error, variables, context, undefined as any);
    },
    ...options,
  });
}

/**
 * 删除模板
 */
export function useDeleteTemplateMutation(
  options?: UseMutationOptions<void, Error, string>
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (templateId: string) => TemplateApi.deleteTemplate(templateId),
    onSuccess: (data, templateId, context) => {
      // 移除详情缓存
      queryClient.removeQueries({ queryKey: templateKeys.detail(templateId) });
      
      // 刷新列表
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
      
      message.success('模板删除成功！');
      
      options?.onSuccess?.(data, templateId, context, undefined as any);
    },
    onError: (error, variables, context) => {
      message.error(`删除失败：${error.message}`);
      options?.onError?.(error, variables, context, undefined as any);
    },
    ...options,
  });
}

/**
 * 发布模板
 */
export function usePublishTemplateMutation(
  options?: UseMutationOptions<
    TicketTemplate,
    Error,
    { templateId: string; changelog?: string }
  >
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ templateId, changelog }) =>
      TemplateApi.publishTemplate(templateId, changelog),
    onSuccess: (data, variables, context) => {
      // 更新缓存
      queryClient.setQueryData(templateKeys.detail(variables.templateId), data);
      
      // 刷新版本历史
      queryClient.invalidateQueries({
        queryKey: templateKeys.versions(variables.templateId),
      });
      
      message.success('模板发布成功！');
      
      options?.onSuccess?.(data, variables, context, undefined as any);
    },
    onError: (error, variables, context) => {
      message.error(`发布失败：${error.message}`);
      options?.onError?.(error, variables, context, undefined as any);
    },
    ...options,
  });
}

/**
 * 复制模板
 */
export function useDuplicateTemplateMutation(
  options?: UseMutationOptions<
    TicketTemplate,
    Error,
    TemplateDuplicateRequest
  >
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: TemplateDuplicateRequest) =>
      TemplateApi.duplicateTemplate(data),
    onSuccess: (data, variables, context) => {
      // 刷新列表
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
      
      message.success('模板复制成功！');
      
      options?.onSuccess?.(data, variables, context, undefined as any);
    },
    onError: (error, variables, context) => {
      message.error(`复制失败：${error.message}`);
      options?.onError?.(error, variables, context, undefined as any);
    },
    ...options,
  });
}

/**
 * 从模板创建工单
 */
export function useCreateTicketFromTemplateMutation(
  options?: UseMutationOptions<
    Ticket,
    Error,
    CreateTicketFromTemplateRequest
  >
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateTicketFromTemplateRequest) =>
      TemplateApi.createTicketFromTemplate(data),
    onSuccess: (data, variables, context) => {
      // 记录模板使用
      TemplateApi.recordTemplateUsage(variables.templateId);
      
      // 刷新模板统计
      queryClient.invalidateQueries({
        queryKey: templateKeys.stats(variables.templateId),
      });
      
      // 刷新最近使用
      queryClient.invalidateQueries({ queryKey: templateKeys.recent() });
      
      message.success('工单创建成功！');
      
      options?.onSuccess?.(data, variables, context, undefined as any);
    },
    onError: (error, variables, context) => {
      message.error(`创建工单失败：${error.message}`);
      options?.onError?.(error, variables, context, undefined as any);
    },
    ...options,
  });
}

/**
 * 为模板评分
 */
export function useRateTemplateMutation(
  options?: UseMutationOptions<
    any,
    Error,
    { templateId: string; rating: number; comment?: string }
  >
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ templateId, rating, comment }) =>
      TemplateApi.rateTemplate(templateId, rating, comment),
    onSuccess: (data, variables, context) => {
      // 刷新模板详情（包含新的评分）
      queryClient.invalidateQueries({
        queryKey: templateKeys.detail(variables.templateId),
      });
      
      // 刷新评分列表
      queryClient.invalidateQueries({
        queryKey: templateKeys.ratings(variables.templateId),
      });
      
      message.success('评分成功！感谢您的反馈');
      
      options?.onSuccess?.(data, variables, context, undefined as any);
    },
    onError: (error, variables, context) => {
      message.error(`评分失败：${error.message}`);
      options?.onError?.(error, variables, context, undefined as any);
    },
    ...options,
  });
}

/**
 * 收藏模板
 */
export function useFavoriteTemplateMutation(
  options?: UseMutationOptions<void, Error, string>
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (templateId: string) => TemplateApi.favoriteTemplate(templateId),
    onSuccess: (data, templateId, context) => {
      // 刷新收藏列表
      queryClient.invalidateQueries({ queryKey: templateKeys.favorites() });
      
      message.success('已添加到收藏');
      
      options?.onSuccess?.(data, templateId, context, undefined as any);
    },
    onError: (error, variables, context) => {
      message.error(`收藏失败：${error.message}`);
      options?.onError?.(error, variables, context, undefined as any);
    },
    ...options,
  });
}

/**
 * 取消收藏模板
 */
export function useUnfavoriteTemplateMutation(
  options?: UseMutationOptions<void, Error, string>
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (templateId: string) =>
      TemplateApi.unfavoriteTemplate(templateId),
    onSuccess: (data, templateId, context) => {
      // 刷新收藏列表
      queryClient.invalidateQueries({ queryKey: templateKeys.favorites() });
      
      message.success('已取消收藏');
      
      options?.onSuccess?.(data, templateId, context, undefined as any);
    },
    onError: (error, variables, context) => {
      message.error(`取消收藏失败：${error.message}`);
      options?.onError?.(error, variables, context, undefined as any);
    },
    ...options,
  });
}

/**
 * 导入模板
 */
export function useImportTemplateMutation(
  options?: UseMutationOptions<any, Error, TemplateImportRequest>
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: TemplateImportRequest) =>
      TemplateApi.importTemplate(data),
    onSuccess: (data, variables, context) => {
      // 刷新模板列表
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
      
      message.success('模板导入成功！');
      
      options?.onSuccess?.(data, variables, context, undefined as any);
    },
    onError: (error, variables, context) => {
      message.error(`导入失败：${error.message}`);
      options?.onError?.(error, variables, context, undefined as any);
    },
    ...options,
  });
}

/**
 * 归档模板
 */
export function useArchiveTemplateMutation(
  options?: UseMutationOptions<TicketTemplate, Error, string>
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (templateId: string) => TemplateApi.archiveTemplate(templateId),
    onSuccess: (data, templateId, context) => {
      // 更新缓存
      queryClient.setQueryData(templateKeys.detail(templateId), data);
      
      // 刷新列表
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
      
      message.success('模板已归档');
      
      options?.onSuccess?.(data, templateId, context, undefined as any);
    },
    onError: (error, variables, context) => {
      message.error(`归档失败：${error.message}`);
      options?.onError?.(error, variables, context, undefined as any);
    },
    ...options,
  });
}

/**
 * 批量删除模板
 */
export function useBatchDeleteTemplatesMutation(
  options?: UseMutationOptions<any, Error, string[]>
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (templateIds: string[]) =>
      TemplateApi.batchDeleteTemplates(templateIds),
    onSuccess: (data, variables, context) => {
      // 刷新列表
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
      
      message.success(`成功删除 ${data.success} 个模板`);
      
      if (data.failed > 0) {
        message.warning(`有 ${data.failed} 个模板删除失败`);
      }
      
      options?.onSuccess?.(data, variables, context, undefined as any);
    },
    onError: (error, variables, context) => {
      message.error(`批量删除失败：${error.message}`);
      options?.onError?.(error, variables, context, undefined as any);
    },
    ...options,
  });
}

/**
 * 批量启用/禁用模板
 */
export function useBatchToggleTemplatesMutation(
  options?: UseMutationOptions<
    any,
    Error,
    { templateIds: string[]; isActive: boolean }
  >
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ templateIds, isActive }) =>
      TemplateApi.batchToggleTemplates(templateIds, isActive),
    onSuccess: (data, variables, context) => {
      // 刷新列表
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
      
      const action = variables.isActive ? '启用' : '禁用';
      message.success(`成功${action} ${data.success} 个模板`);
      
      if (data.failed > 0) {
        message.warning(`有 ${data.failed} 个模板${action}失败`);
      }
      
      options?.onSuccess?.(data, variables, context, undefined as any);
    },
    onError: (error, variables, context) => {
      message.error(`批量操作失败：${error.message}`);
      options?.onError?.(error, variables, context, undefined as any);
    },
    ...options,
  });
}

