/**
 * 数据获取Hook
 * Data Fetching Hook with enhanced features
 */

'use client';

import { useState, useEffect, useCallback, useRef } from 'react';

export interface UseFetchOptions<T> {
  // 初始数据
  initialData?: T;
  // 是否立即执行
  immediate?: boolean;
  // 依赖项
  dependencies?: any[];
  // 成功回调
  onSuccess?: (data: T) => void;
  // 错误回调
  onError?: (error: Error) => void;
  // 缓存时间（毫秒）
  cacheTime?: number;
  // 重试次数
  retryCount?: number;
  // 重试延迟（毫秒）
  retryDelay?: number;
}

export interface UseFetchResult<T> {
  data: T | undefined;
  error: Error | null;
  loading: boolean;
  refetch: () => Promise<void>;
  mutate: (newData: T) => void;
}

// 简单的内存缓存
const cache = new Map<string, { data: any; timestamp: number }>();

/**
 * 数据获取Hook
 */
export function useDataFetch<T = any>(
  fetchFn: () => Promise<T>,
  options: UseFetchOptions<T> = {}
): UseFetchResult<T> {
  const {
    initialData,
    immediate = true,
    dependencies = [],
    onSuccess,
    onError,
    cacheTime = 0,
    retryCount = 0,
    retryDelay = 1000,
  } = options;

  const [data, setData] = useState<T | undefined>(initialData);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState<boolean>(immediate);

  const cacheKey = useRef(fetchFn.toString());
  const retryCountRef = useRef(0);
  const mountedRef = useRef(true);

  const executeFetch = useCallback(async () => {
    // 检查缓存
    if (cacheTime > 0) {
      const cached = cache.get(cacheKey.current);
      if (cached && Date.now() - cached.timestamp < cacheTime) {
        setData(cached.data);
        setLoading(false);
        return;
      }
    }

    setLoading(true);
    setError(null);

    try {
      const result = await fetchFn();
      
      if (!mountedRef.current) return;

      setData(result);
      setError(null);
      retryCountRef.current = 0;

      // 更新缓存
      if (cacheTime > 0) {
        cache.set(cacheKey.current, {
          data: result,
          timestamp: Date.now(),
        });
      }

      onSuccess?.(result);
    } catch (err) {
      if (!mountedRef.current) return;

      const error = err instanceof Error ? err : new Error(String(err));
      
      // 重试逻辑
      if (retryCountRef.current < retryCount) {
        retryCountRef.current++;
        setTimeout(() => {
          executeFetch();
        }, retryDelay);
        return;
      }

      setError(error);
      onError?.(error);
    } finally {
      if (mountedRef.current) {
        setLoading(false);
      }
    }
  }, [fetchFn, cacheTime, retryCount, retryDelay, onSuccess, onError]);

  // 手动触发重新获取
  const refetch = useCallback(async () => {
    retryCountRef.current = 0;
    await executeFetch();
  }, [executeFetch]);

  // 手动更新数据
  const mutate = useCallback((newData: T) => {
    setData(newData);
    // 更新缓存
    if (cacheTime > 0) {
      cache.set(cacheKey.current, {
        data: newData,
        timestamp: Date.now(),
      });
    }
  }, [cacheTime]);

  useEffect(() => {
    mountedRef.current = true;

    if (immediate) {
      executeFetch();
    }

    return () => {
      mountedRef.current = false;
    };
  }, dependencies); // eslint-disable-line react-hooks/exhaustive-deps

  return {
    data,
    error,
    loading,
    refetch,
    mutate,
  };
}

/**
 * 分页数据获取Hook
 */
export interface UsePaginationOptions<T> extends Omit<UseFetchOptions<T>, 'dependencies'> {
  pageSize?: number;
  initialPage?: number;
}

export interface PaginatedData<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface UsePaginationResult<T> extends Omit<UseFetchResult<PaginatedData<T>>, 'mutate'> {
  page: number;
  pageSize: number;
  totalPages: number;
  nextPage: () => void;
  prevPage: () => void;
  goToPage: (page: number) => void;
  setPageSize: (size: number) => void;
}

export function usePagination<T = any>(
  fetchFn: (page: number, pageSize: number) => Promise<PaginatedData<T>>,
  options: UsePaginationOptions<PaginatedData<T>> = {}
): UsePaginationResult<T> {
  const { pageSize: initialPageSize = 10, initialPage = 1, ...fetchOptions } = options;

  const [page, setPage] = useState(initialPage);
  const [pageSize, setPageSize] = useState(initialPageSize);

  const { data, error, loading, refetch } = useDataFetch(
    () => fetchFn(page, pageSize),
    {
      ...fetchOptions,
      dependencies: [page, pageSize],
    }
  );

  const nextPage = useCallback(() => {
    if (data && page < data.totalPages) {
      setPage(p => p + 1);
    }
  }, [data, page]);

  const prevPage = useCallback(() => {
    if (page > 1) {
      setPage(p => p - 1);
    }
  }, [page]);

  const goToPage = useCallback((newPage: number) => {
    if (data && newPage >= 1 && newPage <= data.totalPages) {
      setPage(newPage);
    }
  }, [data]);

  const handleSetPageSize = useCallback((size: number) => {
    setPageSize(size);
    setPage(1); // 重置到第一页
  }, []);

  return {
    data,
    error,
    loading,
    refetch,
    page,
    pageSize,
    totalPages: data?.totalPages || 0,
    nextPage,
    prevPage,
    goToPage,
    setPageSize: handleSetPageSize,
  };
}

/**
 * 清除所有缓存
 */
export const clearCache = () => {
  cache.clear();
};

/**
 * 清除指定键的缓存
 */
export const clearCacheByKey = (key: string) => {
  cache.delete(key);
};

