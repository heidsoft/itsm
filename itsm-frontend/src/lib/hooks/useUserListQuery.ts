'use client';

/**
 * 用户下拉数据 React Query 缓存
 *
 * 处理人/工单创建页/详情页都会拉取用户列表。直接用 useState + useEffect 会
 * 在每次组件挂载时重新请求，造成重复请求和抖动。集中到 React Query 后：
 *   - 默认 staleTime 5 分钟，期间重复挂载使用缓存
 *   - 所有调用方共享一份 in-flight Promise，不会并发触发
 *   - 业务方调用 invalidate 即可强制刷新
 */

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { UserApi, type ListUsersParams, type PagedUsersResponse } from '@/lib/api/user-api';

export const USER_LIST_KEY = ['users', 'list'] as const;

export const userListQueryKey = (params: ListUsersParams = {}) =>
  [...USER_LIST_KEY, params] as const;

export interface UseUserListOptions extends ListUsersParams {
  /** 是否启用缓存请求。默认 true */
  enabled?: boolean;
  /** 缓存过期时间（毫秒）。默认 5 分钟 */
  staleTime?: number;
}

/**
 * 获取用户列表（带缓存）。
 * 使用场景：工单分配人下拉、@提及、审批人选择等需要展示用户列表的地方。
 */
export function useUserListQuery(options: UseUserListOptions = {}) {
  const { enabled = true, staleTime = 5 * 60 * 1000, ...params } = options;
  return useQuery<PagedUsersResponse>({
    queryKey: userListQueryKey(params),
    queryFn: () => UserApi.getUsers(params),
    enabled,
    staleTime,
  });
}

/**
 * 主动失效用户列表缓存（创建/更新/停用用户后调用）。
 */
export function useInvalidateUserList() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: USER_LIST_KEY });
}