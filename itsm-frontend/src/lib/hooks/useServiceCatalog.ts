/**
 * 服务目录 React Query Hooks
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import type {
  ServiceQuery,
  ServiceRequestQuery,
} from '@/types/service-catalog';

export const SERVICE_CATALOG_KEYS = {
  all: ['service-catalog'] as const,
  services: () => [...SERVICE_CATALOG_KEYS.all, 'services'] as const,
  serviceList: (query?: ServiceQuery) =>
    [...SERVICE_CATALOG_KEYS.services(), 'list', query] as const,
  serviceDetail: (id: string) =>
    [...SERVICE_CATALOG_KEYS.services(), 'detail', id] as const,
  serviceRatings: (id: string) =>
    [...SERVICE_CATALOG_KEYS.services(), 'ratings', id] as const,
  serviceAnalytics: (id: string) =>
    [...SERVICE_CATALOG_KEYS.services(), 'analytics', id] as const,
  requests: () => [...SERVICE_CATALOG_KEYS.all, 'requests'] as const,
  requestList: (query?: ServiceRequestQuery) =>
    [...SERVICE_CATALOG_KEYS.requests(), 'list', query] as const,
  requestDetail: (id: number) =>
    [...SERVICE_CATALOG_KEYS.requests(), 'detail', id] as const,
  favorites: () => [...SERVICE_CATALOG_KEYS.all, 'favorites'] as const,
  portal: () => [...SERVICE_CATALOG_KEYS.all, 'portal'] as const,
  stats: () => [...SERVICE_CATALOG_KEYS.all, 'stats'] as const,
};

// Query Hooks
export function useServicesQuery(query?: ServiceQuery) {
  return useQuery({
    queryKey: SERVICE_CATALOG_KEYS.serviceList(query),
    queryFn: () => ServiceCatalogApi.getServices(query),
    staleTime: 300000,
  });
}

export function useServiceQuery(id: string, enabled = true) {
  return useQuery({
    queryKey: SERVICE_CATALOG_KEYS.serviceDetail(id),
    queryFn: () => ServiceCatalogApi.getService(id),
    enabled: enabled && !!id,
    staleTime: 300000,
  });
}

export function useServiceRatingsQuery(
  serviceId: string,
  params?: { page?: number; pageSize?: number },
  enabled = true
) {
  return useQuery({
    queryKey: [...SERVICE_CATALOG_KEYS.serviceRatings(serviceId), params],
    queryFn: () => ServiceCatalogApi.getServiceRatings(serviceId, params),
    enabled: enabled && !!serviceId,
    staleTime: 60000,
  });
}

export function useServiceAnalyticsQuery(
  serviceId: string,
  params?: { startDate?: string; endDate?: string },
  enabled = true
) {
  return useQuery({
    queryKey: [...SERVICE_CATALOG_KEYS.serviceAnalytics(serviceId), params],
    queryFn: () => ServiceCatalogApi.getServiceAnalytics(serviceId, params),
    enabled: enabled && !!serviceId,
    staleTime: 300000,
  });
}

export function useServiceRequestsQuery(query?: ServiceRequestQuery) {
  return useQuery({
    queryKey: SERVICE_CATALOG_KEYS.requestList(query),
    queryFn: () => ServiceCatalogApi.getServiceRequests(query),
    staleTime: 60000,
  });
}

export function useServiceRequestQuery(id: number, enabled = true) {
  return useQuery({
    queryKey: SERVICE_CATALOG_KEYS.requestDetail(id),
    queryFn: () => ServiceCatalogApi.getServiceRequest(id),
    enabled: enabled && !!id,
    staleTime: 30000,
  });
}

export function useFavoritesQuery() {
  return useQuery({
    queryKey: SERVICE_CATALOG_KEYS.favorites(),
    queryFn: () => ServiceCatalogApi.getFavorites(),
    staleTime: 300000,
  });
}

export function usePortalConfigQuery() {
  return useQuery({
    queryKey: SERVICE_CATALOG_KEYS.portal(),
    queryFn: () => ServiceCatalogApi.getPortalConfig(),
    staleTime: 600000,
  });
}

export function useCatalogStatsQuery() {
  return useQuery({
    queryKey: SERVICE_CATALOG_KEYS.stats(),
    queryFn: () => ServiceCatalogApi.getCatalogStats(),
    staleTime: 60000,
  });
}

// Mutation Hooks
export function useCreateServiceMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ServiceCatalogApi.createService,
    onSuccess: () => {
      message.success('服务已创建');
      queryClient.invalidateQueries({
        queryKey: SERVICE_CATALOG_KEYS.services(),
      });
      queryClient.invalidateQueries({ queryKey: SERVICE_CATALOG_KEYS.stats() });
    },
  });
}

export function useUpdateServiceMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: Parameters<typeof ServiceCatalogApi.updateService>[1];
    }) => ServiceCatalogApi.updateService(id, data),
    onSuccess: (_, variables) => {
      message.success('服务已更新');
      queryClient.invalidateQueries({
        queryKey: SERVICE_CATALOG_KEYS.serviceDetail(variables.id),
      });
      queryClient.invalidateQueries({
        queryKey: SERVICE_CATALOG_KEYS.services(),
      });
    },
  });
}

export function usePublishServiceMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => ServiceCatalogApi.publishService(id),
    onSuccess: (_, id) => {
      message.success('服务已发布');
      queryClient.invalidateQueries({
        queryKey: SERVICE_CATALOG_KEYS.serviceDetail(id),
      });
      queryClient.invalidateQueries({
        queryKey: SERVICE_CATALOG_KEYS.services(),
      });
    },
  });
}

export function useCreateServiceRequestMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ServiceCatalogApi.createServiceRequest,
    onSuccess: () => {
      message.success('服务请求已提交');
      queryClient.invalidateQueries({
        queryKey: SERVICE_CATALOG_KEYS.requests(),
      });
      queryClient.invalidateQueries({ queryKey: SERVICE_CATALOG_KEYS.stats() });
    },
  });
}

export function useApproveServiceRequestMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, comment }: { id: number; comment?: string }) =>
      ServiceCatalogApi.approveServiceRequest(id, comment),
    onSuccess: (_, variables) => {
      message.success('服务请求已批准');
      queryClient.invalidateQueries({
        queryKey: SERVICE_CATALOG_KEYS.requestDetail(variables.id),
      });
      queryClient.invalidateQueries({
        queryKey: SERVICE_CATALOG_KEYS.requests(),
      });
    },
  });
}

export function useRejectServiceRequestMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reason }: { id: number; reason: string }) =>
      ServiceCatalogApi.rejectServiceRequest(id, reason),
    onSuccess: (_, variables) => {
      message.success('服务请求已拒绝');
      queryClient.invalidateQueries({
        queryKey: SERVICE_CATALOG_KEYS.requestDetail(variables.id),
      });
      queryClient.invalidateQueries({
        queryKey: SERVICE_CATALOG_KEYS.requests(),
      });
    },
  });
}

export function useAddFavoriteMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (serviceId: string) => ServiceCatalogApi.addFavorite(serviceId),
    onSuccess: () => {
      message.success('已添加到收藏');
      queryClient.invalidateQueries({
        queryKey: SERVICE_CATALOG_KEYS.favorites(),
      });
    },
  });
}

export function useRemoveFavoriteMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (serviceId: string) => ServiceCatalogApi.removeFavorite(serviceId),
    onSuccess: () => {
      message.success('已取消收藏');
      queryClient.invalidateQueries({
        queryKey: SERVICE_CATALOG_KEYS.favorites(),
      });
    },
  });
}

export function useRateServiceMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      serviceId,
      rating,
      comment,
    }: {
      serviceId: string;
      rating: number;
      comment?: string;
    }) => ServiceCatalogApi.rateService(serviceId, rating, comment),
    onSuccess: (_, variables) => {
      message.success('评分已提交');
      queryClient.invalidateQueries({
        queryKey: SERVICE_CATALOG_KEYS.serviceRatings(variables.serviceId),
      });
      queryClient.invalidateQueries({
        queryKey: SERVICE_CATALOG_KEYS.serviceDetail(variables.serviceId),
      });
    },
  });
}

export default {
  useServicesQuery,
  useServiceQuery,
  useServiceRatingsQuery,
  useServiceAnalyticsQuery,
  useServiceRequestsQuery,
  useServiceRequestQuery,
  useFavoritesQuery,
  usePortalConfigQuery,
  useCatalogStatsQuery,
  useCreateServiceMutation,
  useUpdateServiceMutation,
  usePublishServiceMutation,
  useCreateServiceRequestMutation,
  useApproveServiceRequestMutation,
  useRejectServiceRequestMutation,
  useAddFavoriteMutation,
  useRemoveFavoriteMutation,
  useRateServiceMutation,
};

