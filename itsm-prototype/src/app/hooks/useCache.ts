import { useState, useEffect, useCallback } from 'react';

interface CacheOptions {
  ttl?: number; // 缓存时间（毫秒）
  staleWhileRevalidate?: boolean; // 返回过期数据同时重新验证
}

interface CacheEntry<T> {
  data: T;
  timestamp: number;
  ttl: number;
}

class CacheManager {
  private cache = new Map<string, CacheEntry<unknown>>();
  private maxSize = 100;

  set<T>(key: string, data: T, ttl: number = 5 * 60 * 1000) {
    // 如果缓存已满，删除最旧的条目
    if (this.cache.size >= this.maxSize) {
      const firstKey = this.cache.keys().next().value;
      this.cache.delete(firstKey);
    }

    this.cache.set(key, {
      data,
      timestamp: Date.now(),
      ttl
    });
  }

  get<T>(key: string): T | null {
    const entry = this.cache.get(key);
    if (!entry) return null;

    const isExpired = Date.now() - entry.timestamp > entry.ttl;
    if (isExpired) {
      this.cache.delete(key);
      return null;
    }

    return entry.data;
  }

  getStale<T>(key: string): T | null {
    const entry = this.cache.get(key);
    return entry ? entry.data : null;
  }

  isStale(key: string): boolean {
    const entry = this.cache.get(key);
    if (!entry) return true;
    return Date.now() - entry.timestamp > entry.ttl;
  }

  delete(key: string) {
    this.cache.delete(key);
  }

  clear() {
    this.cache.clear();
  }
}

const cacheManager = new CacheManager();

export function useCache<T>(
  key: string,
  fetcher: () => Promise<T>,
  options: CacheOptions = {}
) {
  const { ttl = 5 * 60 * 1000, staleWhileRevalidate = true } = options;
  
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const fetchData = useCallback(async (forceRefresh = false) => {
    try {
      setError(null);
      
      // 检查缓存
      if (!forceRefresh) {
        const cachedData = cacheManager.get<T>(key);
        if (cachedData) {
          setData(cachedData);
          return cachedData;
        }

        // 如果启用了 staleWhileRevalidate，先返回过期数据
        if (staleWhileRevalidate) {
          const staleData = cacheManager.getStale<T>(key);
          if (staleData) {
            setData(staleData);
          }
        }
      }

      setLoading(true);
      const result = await fetcher();
      
      // 缓存新数据
      cacheManager.set(key, result, ttl);
      setData(result);
      
      return result;
    } catch (err) {
      setError(err as Error);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [key, fetcher, ttl, staleWhileRevalidate]);

  const mutate = useCallback((newData: T) => {
    cacheManager.set(key, newData, ttl);
    setData(newData);
  }, [key, ttl]);

  const invalidate = useCallback(() => {
    cacheManager.delete(key);
  }, [key]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return {
    data,
    loading,
    error,
    refetch: () => fetchData(true),
    mutate,
    invalidate
  };
}

export { cacheManager };