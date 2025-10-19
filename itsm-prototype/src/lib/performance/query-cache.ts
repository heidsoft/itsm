/**
 * ITSM前端性能优化 - React Query缓存策略优化
 * 
 * 实现智能缓存、预取、乐观更新和错误回滚
 */

import { QueryClient, QueryClientProvider, useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { ReactNode, useCallback, useMemo } from 'react';

// ==================== 缓存配置 ====================

/**
 * 缓存时间配置
 */
export const CACHE_TIMES = {
  // 静态数据 - 长期缓存
  STATIC: 30 * 60 * 1000, // 30分钟
  
  // 用户数据 - 中期缓存
  USER: 10 * 60 * 1000, // 10分钟
  
  // 工单数据 - 短期缓存
  TICKET: 5 * 60 * 1000, // 5分钟
  
  // 实时数据 - 极短期缓存
  REALTIME: 1 * 60 * 1000, // 1分钟
  
  // 临时数据 - 不缓存
  TEMPORARY: 0,
} as const;

/**
 * 过期时间配置
 */
export const STALE_TIMES = {
  // 静态数据 - 长期有效
  STATIC: 15 * 60 * 1000, // 15分钟
  
  // 用户数据 - 中期有效
  USER: 5 * 60 * 1000, // 5分钟
  
  // 工单数据 - 短期有效
  TICKET: 2 * 60 * 1000, // 2分钟
  
  // 实时数据 - 极短期有效
  REALTIME: 30 * 1000, // 30秒
  
  // 临时数据 - 立即过期
  TEMPORARY: 0,
} as const;

// ==================== 查询键管理 ====================

/**
 * 查询键工厂
 */
export class QueryKeyFactory {
  private static readonly PREFIXES = {
    TICKETS: 'tickets',
    USERS: 'users',
    INCIDENTS: 'incidents',
    PROBLEMS: 'problems',
    CHANGES: 'changes',
    CMDB: 'cmdb',
    KNOWLEDGE: 'knowledge',
    SERVICES: 'services',
    ANALYTICS: 'analytics',
    WORKFLOWS: 'workflows',
    NOTIFICATIONS: 'notifications',
  } as const;
  
  // 工单相关查询键
  static tickets = {
    all: [this.PREFIXES.TICKETS] as const,
    lists: () => [...this.tickets.all, 'list'] as const,
    list: (filters?: any) => [...this.tickets.lists(), filters] as const,
    details: () => [...this.tickets.all, 'detail'] as const,
    detail: (id: number) => [...this.tickets.details(), id] as const,
    comments: (id: number) => [...this.tickets.detail(id), 'comments'] as const,
    attachments: (id: number) => [...this.tickets.detail(id), 'attachments'] as const,
    activities: (id: number) => [...this.tickets.detail(id), 'activities'] as const,
  };
  
  // 用户相关查询键
  static users = {
    all: [this.PREFIXES.USERS] as const,
    lists: () => [...this.users.all, 'list'] as const,
    list: (filters?: any) => [...this.users.lists(), filters] as const,
    details: () => [...this.users.all, 'detail'] as const,
    detail: (id: number) => [...this.users.details(), id] as const,
    profile: () => [...this.users.all, 'profile'] as const,
    permissions: (id: number) => [...this.users.detail(id), 'permissions'] as const,
  };
  
  // 事件相关查询键
  static incidents = {
    all: [this.PREFIXES.INCIDENTS] as const,
    lists: () => [...this.incidents.all, 'list'] as const,
    list: (filters?: any) => [...this.incidents.lists(), filters] as const,
    details: () => [...this.incidents.all, 'detail'] as const,
    detail: (id: number) => [...this.incidents.details(), id] as const,
    events: (id: number) => [...this.incidents.detail(id), 'events'] as const,
    alerts: (id: number) => [...this.incidents.detail(id), 'alerts'] as const,
    metrics: (id: number) => [...this.incidents.detail(id), 'metrics'] as const,
  };
  
  // 问题相关查询键
  static problems = {
    all: [this.PREFIXES.PROBLEMS] as const,
    lists: () => [...this.problems.all, 'list'] as const,
    list: (filters?: any) => [...this.problems.lists(), filters] as const,
    details: () => [...this.problems.all, 'detail'] as const,
    detail: (id: number) => [...this.problems.details(), id] as const,
    investigations: (id: number) => [...this.problems.detail(id), 'investigations'] as const,
  };
  
  // 变更相关查询键
  static changes = {
    all: [this.PREFIXES.CHANGES] as const,
    lists: () => [...this.changes.all, 'list'] as const,
    list: (filters?: any) => [...this.changes.lists(), filters] as const,
    details: () => [...this.changes.all, 'detail'] as const,
    detail: (id: number) => [...this.changes.details(), id] as const,
    approvals: (id: number) => [...this.changes.detail(id), 'approvals'] as const,
  };
  
  // CMDB相关查询键
  static cmdb = {
    all: [this.PREFIXES.CMDB] as const,
    assets: () => [...this.cmdb.all, 'assets'] as const,
    asset: (id: number) => [...this.cmdb.assets(), id] as const,
    configurations: () => [...this.cmdb.all, 'configurations'] as const,
    configuration: (id: number) => [...this.cmdb.configurations(), id] as const,
    relationships: (id: number) => [...this.cmdb.asset(id), 'relationships'] as const,
  };
  
  // 知识库相关查询键
  static knowledge = {
    all: [this.PREFIXES.KNOWLEDGE] as const,
    articles: () => [...this.knowledge.all, 'articles'] as const,
    article: (id: number) => [...this.knowledge.articles(), id] as const,
    categories: () => [...this.knowledge.all, 'categories'] as const,
    category: (id: number) => [...this.knowledge.categories(), id] as const,
    search: (query: string) => [...this.knowledge.all, 'search', query] as const,
  };
  
  // 服务相关查询键
  static services = {
    all: [this.PREFIXES.SERVICES] as const,
    catalogs: () => [...this.services.all, 'catalogs'] as const,
    catalog: (id: number) => [...this.services.catalogs(), id] as const,
    requests: () => [...this.services.all, 'requests'] as const,
    request: (id: number) => [...this.services.requests(), id] as const,
  };
  
  // 分析相关查询键
  static analytics = {
    all: [this.PREFIXES.ANALYTICS] as const,
    dashboards: () => [...this.analytics.all, 'dashboards'] as const,
    dashboard: (id: number) => [...this.analytics.dashboards(), id] as const,
    reports: () => [...this.analytics.all, 'reports'] as const,
    report: (id: number) => [...this.analytics.reports(), id] as const,
    metrics: (type: string) => [...this.analytics.all, 'metrics', type] as const,
  };
  
  // 工作流相关查询键
  static workflows = {
    all: [this.PREFIXES.WORKFLOWS] as const,
    definitions: () => [...this.workflows.all, 'definitions'] as const,
    definition: (id: number) => [...this.workflows.definitions(), id] as const,
    instances: () => [...this.workflows.all, 'instances'] as const,
    instance: (id: number) => [...this.workflows.instances(), id] as const,
    tasks: (id: number) => [...this.workflows.instance(id), 'tasks'] as const,
  };
  
  // 通知相关查询键
  static notifications = {
    all: [this.PREFIXES.NOTIFICATIONS] as const,
    lists: () => [...this.notifications.all, 'list'] as const,
    list: (filters?: any) => [...this.notifications.lists(), filters] as const,
    details: () => [...this.notifications.all, 'detail'] as const,
    detail: (id: number) => [...this.notifications.details(), id] as const,
    templates: () => [...this.notifications.all, 'templates'] as const,
    template: (id: number) => [...this.notifications.templates(), id] as const,
  };
}

// ==================== 智能缓存管理器 ====================

class SmartCacheManager {
  private queryClient: QueryClient;
  private cacheMetrics = new Map<string, { hits: number; misses: number; lastAccess: number }>();
  
  constructor(queryClient: QueryClient) {
    this.queryClient = queryClient;
  }
  
  /**
   * 智能预取
   */
  async smartPrefetch<T>(
    queryKey: string[],
    queryFn: () => Promise<T>,
    options?: {
      staleTime?: number;
      cacheTime?: number;
      priority?: 'high' | 'medium' | 'low';
    }
  ): Promise<void> {
    const key = queryKey.join('.');
    const metrics = this.cacheMetrics.get(key) || { hits: 0, misses: 0, lastAccess: 0 };
    
    // 检查是否需要预取
    if (this.shouldPrefetch(key, metrics)) {
      await this.queryClient.prefetchQuery({
        queryKey,
        queryFn,
        staleTime: options?.staleTime || STALE_TIMES.TICKET,
        cacheTime: options?.cacheTime || CACHE_TIMES.TICKET,
      });
      
      metrics.lastAccess = Date.now();
      this.cacheMetrics.set(key, metrics);
    }
  }
  
  /**
   * 批量预取
   */
  async batchPrefetch<T>(
    queries: Array<{
      queryKey: string[];
      queryFn: () => Promise<T>;
      options?: { staleTime?: number; cacheTime?: number };
    }>
  ): Promise<void> {
    const promises = queries.map(({ queryKey, queryFn, options }) =>
      this.smartPrefetch(queryKey, queryFn, options)
    );
    
    await Promise.allSettled(promises);
  }
  
  /**
   * 智能缓存清理
   */
  smartCleanup(): void {
    const now = Date.now();
    const cleanupThreshold = 24 * 60 * 60 * 1000; // 24小时
    
    this.cacheMetrics.forEach((metrics, key) => {
      if (now - metrics.lastAccess > cleanupThreshold) {
        const queryKey = key.split('.');
        this.queryClient.removeQueries({ queryKey });
        this.cacheMetrics.delete(key);
      }
    });
  }
  
  /**
   * 获取缓存统计
   */
  getCacheStats(): {
    totalQueries: number;
    hitRate: number;
    memoryUsage: number;
    topQueries: Array<{ key: string; hits: number; misses: number }>;
  } {
    let totalHits = 0;
    let totalMisses = 0;
    const topQueries: Array<{ key: string; hits: number; misses: number }> = [];
    
    this.cacheMetrics.forEach((metrics, key) => {
      totalHits += metrics.hits;
      totalMisses += metrics.misses;
      topQueries.push({ key, ...metrics });
    });
    
    topQueries.sort((a, b) => b.hits - a.hits);
    
    return {
      totalQueries: this.cacheMetrics.size,
      hitRate: totalHits + totalMisses > 0 ? totalHits / (totalHits + totalMisses) : 0,
      memoryUsage: (performance as any).memory?.usedJSHeapSize || 0,
      topQueries: topQueries.slice(0, 10),
    };
  }
  
  /**
   * 判断是否需要预取
   */
  private shouldPrefetch(key: string, metrics: { hits: number; misses: number; lastAccess: number }): boolean {
    const now = Date.now();
    const timeSinceLastAccess = now - metrics.lastAccess;
    
    // 如果最近访问过，不需要预取
    if (timeSinceLastAccess < 5 * 60 * 1000) { // 5分钟
      return false;
    }
    
    // 如果命中率低，不需要预取
    const hitRate = metrics.hits / (metrics.hits + metrics.misses);
    if (hitRate < 0.3) {
      return false;
    }
    
    return true;
  }
}

// ==================== 乐观更新管理器 ====================

class OptimisticUpdateManager {
  private queryClient: QueryClient;
  private rollbackData = new Map<string, any>();
  
  constructor(queryClient: QueryClient) {
    this.queryClient = queryClient;
  }
  
  /**
   * 执行乐观更新
   */
  async executeOptimisticUpdate<T>(
    queryKey: string[],
    updateFn: (oldData: T) => T,
    mutationFn: () => Promise<T>,
    options?: {
      onError?: (error: Error) => void;
      onSuccess?: (data: T) => void;
    }
  ): Promise<T> {
    const key = queryKey.join('.');
    
    // 保存原始数据用于回滚
    const originalData = this.queryClient.getQueryData(queryKey);
    this.rollbackData.set(key, originalData);
    
    // 执行乐观更新
    this.queryClient.setQueryData(queryKey, updateFn);
    
    try {
      // 执行实际更新
      const result = await mutationFn();
      
      // 更新成功，更新缓存
      this.queryClient.setQueryData(queryKey, result);
      this.rollbackData.delete(key);
      
      options?.onSuccess?.(result);
      return result;
    } catch (error) {
      // 更新失败，回滚数据
      this.rollbackData.delete(key);
      this.queryClient.setQueryData(queryKey, originalData);
      
      options?.onError?.(error as Error);
      throw error;
    }
  }
  
  /**
   * 批量乐观更新
   */
  async executeBatchOptimisticUpdate<T>(
    updates: Array<{
      queryKey: string[];
      updateFn: (oldData: T) => T;
      mutationFn: () => Promise<T>;
    }>,
    options?: {
      onError?: (error: Error) => void;
      onSuccess?: (results: T[]) => void;
    }
  ): Promise<T[]> {
    const rollbackData = new Map<string, any>();
    const results: T[] = [];
    
    try {
      // 保存所有原始数据
      updates.forEach(({ queryKey }) => {
        const key = queryKey.join('.');
        const originalData = this.queryClient.getQueryData(queryKey);
        rollbackData.set(key, originalData);
      });
      
      // 执行所有乐观更新
      updates.forEach(({ queryKey, updateFn }) => {
        this.queryClient.setQueryData(queryKey, updateFn);
      });
      
      // 执行所有实际更新
      for (const { mutationFn } of updates) {
        const result = await mutationFn();
        results.push(result);
      }
      
      // 更新所有缓存
      updates.forEach(({ queryKey }, index) => {
        this.queryClient.setQueryData(queryKey, results[index]);
      });
      
      options?.onSuccess?.(results);
      return results;
    } catch (error) {
      // 回滚所有数据
      rollbackData.forEach((originalData, key) => {
        const queryKey = key.split('.');
        this.queryClient.setQueryData(queryKey, originalData);
      });
      
      options?.onError?.(error as Error);
      throw error;
    }
  }
  
  /**
   * 手动回滚
   */
  rollback(queryKey: string[]): void {
    const key = queryKey.join('.');
    const originalData = this.rollbackData.get(key);
    
    if (originalData !== undefined) {
      this.queryClient.setQueryData(queryKey, originalData);
      this.rollbackData.delete(key);
    }
  }
  
  /**
   * 清理回滚数据
   */
  cleanup(): void {
    this.rollbackData.clear();
  }
}

// ==================== 查询客户端配置 ====================

/**
 * 创建优化的查询客户端
 */
export function createOptimizedQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // 默认缓存时间
        cacheTime: CACHE_TIMES.TICKET,
        // 默认过期时间
        staleTime: STALE_TIMES.TICKET,
        // 重试次数
        retry: (failureCount, error) => {
          // 4xx错误不重试
          if (error instanceof Error && 'status' in error) {
            const status = (error as any).status;
            if (status >= 400 && status < 500) {
              return false;
            }
          }
          return failureCount < 3;
        },
        // 重试延迟
        retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
        // 窗口聚焦时不重新获取
        refetchOnWindowFocus: false,
        // 网络重连时重新获取
        refetchOnReconnect: true,
        // 挂载时不重新获取
        refetchOnMount: true,
      },
      mutations: {
        // 重试次数
        retry: 1,
        // 重试延迟
        retryDelay: 1000,
      },
    },
  });
}

// ==================== 性能优化Hook ====================

/**
 * 使用优化的查询
 */
export function useOptimizedQuery<T>(
  queryKey: string[],
  queryFn: () => Promise<T>,
  options?: {
    staleTime?: number;
    cacheTime?: number;
    enabled?: boolean;
    refetchInterval?: number;
    refetchIntervalInBackground?: boolean;
  }
) {
  return useQuery({
    queryKey,
    queryFn,
    staleTime: options?.staleTime || STALE_TIMES.TICKET,
    cacheTime: options?.cacheTime || CACHE_TIMES.TICKET,
    enabled: options?.enabled,
    refetchInterval: options?.refetchInterval,
    refetchIntervalInBackground: options?.refetchIntervalInBackground,
  });
}

/**
 * 使用优化的变更
 */
export function useOptimizedMutation<T, V>(
  mutationFn: (variables: V) => Promise<T>,
  options?: {
    onSuccess?: (data: T, variables: V) => void;
    onError?: (error: Error, variables: V) => void;
    onSettled?: (data: T | undefined, error: Error | null, variables: V) => void;
  }
) {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn,
    onSuccess: (data, variables) => {
      options?.onSuccess?.(data, variables);
    },
    onError: (error, variables) => {
      options?.onError?.(error as Error, variables);
    },
    onSettled: (data, error, variables) => {
      options?.onSettled?.(data, error as Error, variables);
    },
  });
}

/**
 * 使用乐观更新
 */
export function useOptimisticUpdate<T, V>(
  queryKey: string[],
  mutationFn: (variables: V) => Promise<T>,
  options?: {
    onSuccess?: (data: T, variables: V) => void;
    onError?: (error: Error, variables: V) => void;
  }
) {
  const queryClient = useQueryClient();
  const optimisticManager = useMemo(() => new OptimisticUpdateManager(queryClient), [queryClient]);
  
  return useMutation({
    mutationFn,
    onMutate: async (variables: V) => {
      // 取消正在进行的查询
      await queryClient.cancelQueries({ queryKey });
      
      // 保存原始数据
      const previousData = queryClient.getQueryData(queryKey);
      
      // 执行乐观更新
      const updateFn = (oldData: T) => {
        // 这里需要根据具体的更新逻辑来实现
        return oldData;
      };
      
      await optimisticManager.executeOptimisticUpdate(
        queryKey,
        updateFn,
        () => mutationFn(variables),
        {
          onSuccess: (data) => options?.onSuccess?.(data, variables),
          onError: (error) => options?.onError?.(error, variables),
        }
      );
      
      return { previousData };
    },
    onError: (error, variables, context) => {
      // 回滚数据
      if (context?.previousData) {
        queryClient.setQueryData(queryKey, context.previousData);
      }
      options?.onError?.(error as Error, variables);
    },
    onSettled: () => {
      // 重新获取数据
      queryClient.invalidateQueries({ queryKey });
    },
  });
}

// ==================== 缓存性能监控 ====================

/**
 * 缓存性能监控Hook
 */
export function useCachePerformance() {
  const queryClient = useQueryClient();
  const [metrics, setMetrics] = useState({
    cacheSize: 0,
    hitRate: 0,
    memoryUsage: 0,
    queryCount: 0,
  });
  
  useEffect(() => {
    const updateMetrics = () => {
      const cache = queryClient.getQueryCache();
      const queries = cache.getAll();
      
      setMetrics({
        cacheSize: queries.length,
        hitRate: 0, // 需要从缓存管理器获取
        memoryUsage: (performance as any).memory?.usedJSHeapSize || 0,
        queryCount: queries.length,
      });
    };
    
    updateMetrics();
    const interval = setInterval(updateMetrics, 5000);
    
    return () => clearInterval(interval);
  }, [queryClient]);
  
  return metrics;
}

// ==================== 导出 ====================

export default {
  CACHE_TIMES,
  STALE_TIMES,
  QueryKeyFactory,
  SmartCacheManager,
  OptimisticUpdateManager,
  createOptimizedQueryClient,
  useOptimizedQuery,
  useOptimizedMutation,
  useOptimisticUpdate,
  useCachePerformance,
};
