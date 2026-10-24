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
import { BACKEND_MAX_PAGE_SIZE, isTruncated } from '@/lib/api/pagination';

export const USER_LIST_KEY = ['users', 'list'] as const;

/**
 * 归一化用户列表查询参数。
 *
 * 两个作用：
 *  1. **queryKey 稳定**。React Query 按 key 的值做哈希，同一份语义的查询若
 *     参数对象形状不同（缺省 pageSize / 多一个 `search: ''` / 键序不同）就会
 *     落进两个缓存槽，表现为「换个写法就重新请求一次」。这里把参数收敛成
 *     固定形状：空串与 undefined 一律丢弃，pageSize 缺失时补后端上限。
 *  2. **分页上限显式化**。不传 pageSize 时默认后端上限 100，而不是由各处
 *     自行写死字面量。
 */
function normalizeUserListParams(params: ListUsersParams): ListUsersParams {
  const out: ListUsersParams = {};
  if (params.page != null) out.page = params.page;
  out.pageSize = params.pageSize ?? BACKEND_MAX_PAGE_SIZE;
  if (params.tenantId != null) out.tenantId = params.tenantId;
  if (params.status) out.status = params.status;
  if (params.department) out.department = params.department;
  if (params.search) out.search = params.search;
  return out;
}

export const userListQueryKey = (params: ListUsersParams = {}) =>
  [...USER_LIST_KEY, normalizeUserListParams(params)] as const;

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
  const normalized = normalizeUserListParams(params);
  return useQuery<PagedUsersResponse>({
    queryKey: [...USER_LIST_KEY, normalized] as const,
    queryFn: () => UserApi.getUsers(normalized),
    enabled,
    staleTime,
  });
}

/**
 * 选人下拉专用的用户列表。
 *
 * 在 `useUserListQuery` 之上补齐三件下拉场景必需、但直接消费 `useQuery` 时
 * 容易漏掉的事：
 *   - `users`：空数据兜底，调用方不必再写 `data?.items ?? []`
 *   - `truncated`：后端按 100 静默截断（见 `@/lib/api/pagination`），下拉里
 *     少掉的人必须有显式提示，否则就是「找不到某个人」的静默故障
 *   - `errorMessage`：失败原因透出，而不是折叠成「获取用户列表失败」
 */
export function useUserPickerQuery(options: UseUserListOptions = {}) {
  const query = useUserListQuery(options);
  const items = query.data?.items ?? [];
  return {
    ...query,
    users: items,
    truncated: isTruncated(query.data?.total, items.length),
    total: query.data?.total ?? 0,
  };
}

/**
 * 主动失效用户列表缓存（创建/更新/停用用户后调用）。
 */
export function useInvalidateUserList() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: USER_LIST_KEY });
}