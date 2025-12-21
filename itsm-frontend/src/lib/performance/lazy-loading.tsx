/**
 * ITSM前端性能优化 - 组件懒加载和代码分割
 *
 * 实现组件懒加载、代码分割和动态导入
 */

import React, { Suspense, lazy, ComponentType, ReactNode } from 'react';
import { Spin, Skeleton } from 'antd';
import { LoadingOutlined } from 'lucide-react';

// ==================== 加载状态组件 ====================

/**
 * 默认加载组件
 */
export const DefaultLoadingComponent: React.FC = () => (
  <div
    style={{
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center',
      height: '200px',
    }}
  >
    <Spin indicator={<LoadingOutlined style={{ fontSize: 24 }} spin />} />
  </div>
);

/**
 * 骨架屏加载组件
 */
export const SkeletonLoadingComponent: React.FC<{ rows?: number }> = ({ rows = 3 }) => (
  <div style={{ padding: '16px' }}>
    {Array.from({ length: rows }).map((_, index) => (
      <Skeleton key={index} active paragraph={{ rows: 1 }} style={{ marginBottom: '16px' }} />
    ))}
  </div>
);

/**
 * 表格骨架屏
 */
export const TableSkeletonComponent: React.FC = () => (
  <div style={{ padding: '16px' }}>
    <Skeleton active paragraph={{ rows: 0 }} />
    {Array.from({ length: 5 }).map((_, index) => (
      <Skeleton key={index} active paragraph={{ rows: 1 }} style={{ marginBottom: '8px' }} />
    ))}
  </div>
);

/**
 * 卡片骨架屏
 */
export const CardSkeletonComponent: React.FC = () => (
  <div style={{ padding: '16px' }}>
    <Skeleton active avatar paragraph={{ rows: 2 }} />
  </div>
);

// ==================== 懒加载工具函数 ====================

/**
 * 创建懒加载组件
 */
export function createLazyComponent<T extends ComponentType<any>>(
  importFunc: () => Promise<{ default: T }>,
  fallback?: ReactNode
): React.LazyExoticComponent<T> {
  return lazy(importFunc);
}

/**
 * 创建带重试的懒加载组件
 */
export function createLazyComponentWithRetry<T extends ComponentType<any>>(
  importFunc: () => Promise<{ default: T }>,
  retries: number = 3,
  fallback?: ReactNode
): React.LazyExoticComponent<T> {
  const retryImport = async (retryCount: number = 0): Promise<{ default: T }> => {
    try {
      return await importFunc();
    } catch (error) {
      if (retryCount < retries) {
        console.warn(`Failed to load component, retrying... (${retryCount + 1}/${retries})`);
        await new Promise(resolve => setTimeout(resolve, 1000 * (retryCount + 1)));
        return retryImport(retryCount + 1);
      }
      throw error;
    }
  };

  return lazy(() => retryImport());
}

/**
 * 创建预加载的懒加载组件
 */
export function createPreloadableLazyComponent<T extends ComponentType<any>>(
  importFunc: () => Promise<{ default: T }>,
  fallback?: ReactNode
): React.LazyExoticComponent<T> & { preload: () => Promise<void> } {
  const LazyComponent = lazy(importFunc);

  const preload = async () => {
    try {
      await importFunc();
    } catch (error) {
      console.warn('Failed to preload component:', error);
    }
  };

  return Object.assign(LazyComponent, { preload });
}

// ==================== 懒加载包装器组件 ====================

interface LazyWrapperProps {
  children: ReactNode;
  fallback?: ReactNode;
  errorBoundary?: boolean;
}

/**
 * 懒加载包装器
 */
export const LazyWrapper: React.FC<LazyWrapperProps> = ({
  children,
  fallback = <DefaultLoadingComponent />,
  errorBoundary = true,
}) => {
  if (errorBoundary) {
    return (
      <ErrorBoundary fallback={<div>组件加载失败</div>}>
        <Suspense fallback={fallback}>{children}</Suspense>
      </ErrorBoundary>
    );
  }

  return <Suspense fallback={fallback}>{children}</Suspense>;
};

// ==================== 错误边界组件 ====================

interface ErrorBoundaryState {
  hasError: boolean;
  error?: Error;
}

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
}

export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('ErrorBoundary caught an error:', error, errorInfo);
    this.props.onError?.(error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return (
        this.props.fallback || (
          <div
            style={{
              padding: '20px',
              textAlign: 'center',
              color: '#ff4d4f',
            }}
          >
            <h3>组件加载失败</h3>
            <p>请刷新页面重试</p>
            <button
              onClick={() => window.location.reload()}
              style={{
                padding: '8px 16px',
                backgroundColor: '#1890ff',
                color: 'white',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer',
              }}
            >
              刷新页面
            </button>
          </div>
        )
      );
    }

    return this.props.children;
  }
}

// ==================== 路由懒加载组件 ====================

/**
 * 路由懒加载组件
 */
export const RouteLazyWrapper: React.FC<{
  children: ReactNode;
  routeName: string;
}> = ({ children, routeName }) => {
  const [isLoaded, setIsLoaded] = useState(false);

  useEffect(() => {
    // 标记路由已加载
    setIsLoaded(true);

    // 预加载下一个可能的路由
    const preloadNextRoute = () => {
      // 这里可以根据路由逻辑预加载下一个组件
    };

    const timer = setTimeout(preloadNextRoute, 1000);
    return () => clearTimeout(timer);
  }, [routeName]);

  return <LazyWrapper fallback={<SkeletonLoadingComponent rows={5} />}>{children}</LazyWrapper>;
};

// ==================== 预加载管理器 ====================

class PreloadManager {
  private preloadedComponents = new Set<string>();
  private preloadPromises = new Map<string, Promise<void>>();

  /**
   * 预加载组件
   */
  async preloadComponent(name: string, importFunc: () => Promise<any>): Promise<void> {
    if (this.preloadedComponents.has(name)) {
      return;
    }

    if (this.preloadPromises.has(name)) {
      return this.preloadPromises.get(name);
    }

    const promise = importFunc()
      .then(() => {
        this.preloadedComponents.add(name);
        this.preloadPromises.delete(name);
      })
      .catch(error => {
        console.warn(`Failed to preload component ${name}:`, error);
        this.preloadPromises.delete(name);
      });

    this.preloadPromises.set(name, promise);
    return promise;
  }

  /**
   * 批量预加载组件
   */
  async preloadComponents(
    components: Array<{ name: string; importFunc: () => Promise<any> }>
  ): Promise<void> {
    const promises = components.map(({ name, importFunc }) =>
      this.preloadComponent(name, importFunc)
    );

    await Promise.allSettled(promises);
  }

  /**
   * 检查组件是否已预加载
   */
  isPreloaded(name: string): boolean {
    return this.preloadedComponents.has(name);
  }

  /**
   * 获取预加载状态
   */
  getPreloadStatus(): { preloaded: string[]; loading: string[] } {
    return {
      preloaded: Array.from(this.preloadedComponents),
      loading: Array.from(this.preloadPromises.keys()),
    };
  }
}

export const preloadManager = new PreloadManager();

// ==================== 智能预加载Hook ====================

/**
 * 智能预加载Hook
 */
export function useSmartPreload() {
  const [preloadStatus, setPreloadStatus] = useState({
    preloaded: [] as string[],
    loading: [] as string[],
  });

  useEffect(() => {
    const updateStatus = () => {
      setPreloadStatus(preloadManager.getPreloadStatus());
    };

    updateStatus();
    const interval = setInterval(updateStatus, 1000);

    return () => clearInterval(interval);
  }, []);

  const preloadComponent = useCallback(async (name: string, importFunc: () => Promise<any>) => {
    await preloadManager.preloadComponent(name, importFunc);
    setPreloadStatus(preloadManager.getPreloadStatus());
  }, []);

  const preloadComponents = useCallback(
    async (components: Array<{ name: string; importFunc: () => Promise<any> }>) => {
      await preloadManager.preloadComponents(components);
      setPreloadStatus(preloadManager.getPreloadStatus());
    },
    []
  );

  return {
    preloadStatus,
    preloadComponent,
    preloadComponents,
    isPreloaded: preloadManager.isPreloaded.bind(preloadManager),
  };
}

// ==================== 代码分割工具 ====================

/**
 * 动态导入工具
 */
export const dynamicImport = {
  /**
   * 动态导入组件
   */
  async importComponent<T>(path: string): Promise<T> {
    try {
      const module = await import(path);
      return module.default || module;
    } catch (error) {
      console.error(`Failed to import component from ${path}:`, error);
      throw error;
    }
  },

  /**
   * 动态导入多个组件
   */
  async importComponents<T>(paths: string[]): Promise<T[]> {
    const promises = paths.map(path => this.importComponent<T>(path));
    return Promise.all(promises);
  },

  /**
   * 条件导入组件
   */
  async conditionalImport<T>(condition: boolean, truePath: string, falsePath: string): Promise<T> {
    const path = condition ? truePath : falsePath;
    return this.importComponent<T>(path);
  },
};

// ==================== 性能监控Hook ====================

/**
 * 组件加载性能监控Hook
 */
export function useComponentLoadPerformance(componentName: string) {
  const [metrics, setMetrics] = useState({
    loadTime: 0,
    renderTime: 0,
    memoryUsage: 0,
  });

  const startTimeRef = useRef<number>();

  useEffect(() => {
    startTimeRef.current = performance.now();

    return () => {
      if (startTimeRef.current) {
        const loadTime = performance.now() - startTimeRef.current;
        setMetrics(prev => ({
          ...prev,
          loadTime,
          memoryUsage: (performance as any).memory?.usedJSHeapSize || 0,
        }));

        if (process.env.NODE_ENV === 'development') {
          console.log(`[Performance] ${componentName} loaded in ${loadTime.toFixed(2)}ms`);
        }
      }
    };
  }, [componentName]);

  return metrics;
}

// ==================== 导出 ====================

export default {
  DefaultLoadingComponent,
  SkeletonLoadingComponent,
  TableSkeletonComponent,
  CardSkeletonComponent,
  createLazyComponent,
  createLazyComponentWithRetry,
  createPreloadableLazyComponent,
  LazyWrapper,
  ErrorBoundary,
  RouteLazyWrapper,
  preloadManager,
  useSmartPreload,
  dynamicImport,
  useComponentLoadPerformance,
};
