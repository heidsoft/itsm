"use client";

import { useEffect, useRef, useState } from 'react';

// 性能监控接口
interface PerformanceMetrics {
  renderTime: number;
  componentMountTime: number;
  memoryUsage?: number;
  rerenderCount: number;
}

// 性能监控Hook
export const usePerformance = (componentName: string) => {
  const [metrics, setMetrics] = useState<PerformanceMetrics>({
    renderTime: 0,
    componentMountTime: 0,
    rerenderCount: 0,
  });
  
  const mountTimeRef = useRef<number>(Date.now());
  const renderCountRef = useRef<number>(0);
  const lastRenderTimeRef = useRef<number>(Date.now());

  useEffect(() => {
    const mountTime = Date.now() - mountTimeRef.current;
    
    // 记录组件挂载时间
    setMetrics(prev => ({
      ...prev,
      componentMountTime: mountTime,
    }));

    // 开发环境下输出性能指标
    if (process.env.NODE_ENV === 'development' && process.env.NEXT_PUBLIC_ENABLE_PERFORMANCE_MONITORING === 'true') {
      console.log(`[Performance] ${componentName} mounted in ${mountTime}ms`);
    }
  }, [componentName]);

  useEffect(() => {
    renderCountRef.current += 1;
    const renderTime = Date.now() - lastRenderTimeRef.current;
    lastRenderTimeRef.current = Date.now();

    setMetrics(prev => ({
      ...prev,
      renderTime,
      rerenderCount: renderCountRef.current,
    }));

    // 检测内存使用情况（仅在支持的浏览器中）
    if ('memory' in performance) {
      const memoryInfo = (performance as Performance & { memory?: { usedJSHeapSize: number } }).memory;
      if (memoryInfo) {
        setMetrics(prev => ({
          ...prev,
          memoryUsage: memoryInfo.usedJSHeapSize,
        }));
      }
    }
  });

  // 性能警告
  useEffect(() => {
    if (process.env.NODE_ENV === 'development') {
      if (metrics.rerenderCount > 10) {
        console.warn(`[Performance Warning] ${componentName} has re-rendered ${metrics.rerenderCount} times`);
      }
      
      if (metrics.renderTime > 100) {
        console.warn(`[Performance Warning] ${componentName} render time: ${metrics.renderTime}ms`);
      }
    }
  }, [componentName, metrics.rerenderCount, metrics.renderTime]);

  return metrics;
};

// 防抖Hook
export const useDebounce = <T>(value: T, delay: number): T => {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(handler);
    };
  }, [value, delay]);

  return debouncedValue;
};

// 节流Hook
export const useThrottle = <T>(value: T, limit: number): T => {
  const [throttledValue, setThrottledValue] = useState<T>(value);
  const lastRan = useRef<number>(Date.now());

  useEffect(() => {
    const handler = setTimeout(() => {
      if (Date.now() - lastRan.current >= limit) {
        setThrottledValue(value);
        lastRan.current = Date.now();
      }
    }, limit - (Date.now() - lastRan.current));

    return () => {
      clearTimeout(handler);
    };
  }, [value, limit]);

  return throttledValue;
};

// 本地存储Hook
export const useLocalStorage = <T>(key: string, initialValue: T) => {
  const [storedValue, setStoredValue] = useState<T>(() => {
    if (typeof window === 'undefined') {
      return initialValue;
    }
    
    try {
      const item = window.localStorage.getItem(key);
      return item ? JSON.parse(item) : initialValue;
    } catch (error) {
      console.error(`Error reading localStorage key "${key}":`, error);
      return initialValue;
    }
  });

  const setValue = (value: T | ((val: T) => T)) => {
    try {
      const valueToStore = value instanceof Function ? value(storedValue) : value;
      setStoredValue(valueToStore);
      
      if (typeof window !== 'undefined') {
        window.localStorage.setItem(key, JSON.stringify(valueToStore));
      }
    } catch (error) {
      console.error(`Error setting localStorage key "${key}":`, error);
    }
  };

  const removeValue = () => {
    try {
      setStoredValue(initialValue);
      if (typeof window !== 'undefined') {
        window.localStorage.removeItem(key);
      }
    } catch (error) {
      console.error(`Error removing localStorage key "${key}":`, error);
    }
  };

  return [storedValue, setValue, removeValue] as const;
};

// 会话存储Hook
export const useSessionStorage = <T>(key: string, initialValue: T) => {
  const [storedValue, setStoredValue] = useState<T>(() => {
    if (typeof window === 'undefined') {
      return initialValue;
    }
    
    try {
      const item = window.sessionStorage.getItem(key);
      return item ? JSON.parse(item) : initialValue;
    } catch (error) {
      console.error(`Error reading sessionStorage key "${key}":`, error);
      return initialValue;
    }
  });

  const setValue = (value: T | ((val: T) => T)) => {
    try {
      const valueToStore = value instanceof Function ? value(storedValue) : value;
      setStoredValue(valueToStore);
      
      if (typeof window !== 'undefined') {
        window.sessionStorage.setItem(key, JSON.stringify(valueToStore));
      }
    } catch (error) {
      console.error(`Error setting sessionStorage key "${key}":`, error);
    }
  };

  const removeValue = () => {
    try {
      setStoredValue(initialValue);
      if (typeof window !== 'undefined') {
        window.sessionStorage.removeItem(key);
      }
    } catch (error) {
      console.error(`Error removing sessionStorage key "${key}":`, error);
    }
  };

  return [storedValue, setValue, removeValue] as const;
};