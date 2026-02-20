/**
 * TanStack Query Hooks 模板
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { httpClient } from '@/lib/api/http-client';

// 基础查询 Hook
export function useList<T>(
  key: string | string[],
  endpoint: string,
  params?: object
) {
  return useQuery({
    queryKey: Array.isArray(key) ? key : [key],
    queryFn: () => httpClient.get<{ list: T[]; total: number }>(endpoint, params),
  });
}

// 详情查询 Hook
export function useDetail<T>(key: string, id: number, endpoint: string) {
  return useQuery({
    queryKey: [key, id],
    queryFn: () => httpClient.get<T>(`${endpoint}/${id}`),
    enabled: !!id,
  });
}

// 创建 Mutation Hook
export function useCreate<T>(
  endpoint: string,
  queryKeyToInvalidate: string | string[]
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: T) => httpClient.post<T>(endpoint, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [queryKeyToInvalidate] });
    },
  });
}

// 更新 Mutation Hook
export function useUpdate<T>(endpoint: string) {
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<T> }) =>
      httpClient.put<T>(`${endpoint}/${id}`, data),
  });
}

// 删除 Mutation Hook
export function useDelete(endpoint: string, queryKeyToInvalidate: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => httpClient.delete(`${endpoint}/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [queryKeyToInvalidate] });
    },
  });
}
