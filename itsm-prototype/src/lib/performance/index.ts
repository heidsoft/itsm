/**
 * ITSM前端性能优化 - 统一入口
 * 
 * 统一导出所有性能优化相关的功能和组件
 */

// ==================== 渲染性能优化 ====================
export {
  usePerformanceMonitor,
  useDebounce,
  useThrottle,
  OptimizedSearchInput,
  OptimizedStatusTag,
  OptimizedTicketActions,
  OptimizedTicketFilters,
  OptimizedTicketList,
  OptimizedTicketManagementPage,
} from './render-optimization';

// ==================== 虚拟滚动 ====================
export {
  VirtualList,
  VariableVirtualList,
  VirtualTable,
  VirtualGrid,
  VirtualSelect,
  useVirtualScrollPerformance,
} from './virtual-scroll';

// ==================== 懒加载和代码分割 ====================
export {
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
} from './lazy-loading';

// ==================== React Query缓存策略 ====================
export {
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
} from './query-cache';

// ==================== 骨架屏和加载状态 ====================
export {
  TextSkeleton,
  TitleSkeleton,
  AvatarSkeleton,
  CardSkeleton,
  TableSkeleton,
  ListSkeleton,
  FormSkeleton,
  PageLoading,
  InlineLoading,
  ButtonLoading,
  ProgressLoader,
  StepLoader,
  ErrorState,
  EmptyState,
  LoadingStateManager,
  loadingStateManager,
  useSmartLoading,
  LoadingWrapper,
  DelayedLoading,
  ProgressiveLoading,
} from './skeleton-loading';

// ==================== PWA和离线支持 ====================
export {
  ServiceWorkerManager,
  serviceWorkerManager,
  CacheManager,
  cacheManager,
  NetworkManager,
  networkManager,
  PushNotificationManager,
  pushNotificationManager,
  OfflineStorageManager,
  offlineStorageManager,
  OfflineSyncManager,
  offlineSyncManager,
  usePWA,
} from './pwa-offline';

// ==================== 性能监控和报告 ====================
export {
  PerformanceMonitor,
  performanceMonitor,
  PerformanceAnalyzer,
  performanceAnalyzer,
  usePerformanceMonitoring,
  PerformanceDashboard,
} from './performance-monitoring';

// ==================== 类型定义 ====================
export type {
  PerformanceMetrics,
  PerformanceReport,
} from './performance-monitoring';

// ==================== 性能优化工具类 ====================

/**
 * 性能优化工具类
 */
export class PerformanceOptimizer {
  private static instance: PerformanceOptimizer;
  private isInitialized: boolean = false;

  private constructor() {}

  static getInstance(): PerformanceOptimizer {
    if (!PerformanceOptimizer.instance) {
      PerformanceOptimizer.instance = new PerformanceOptimizer();
    }
    return PerformanceOptimizer.instance;
  }

  /**
   * 初始化性能优化
   */
  async initialize(): Promise<void> {
    if (this.isInitialized) {
      return;
    }

    try {
      // 初始化Service Worker
      await serviceWorkerManager.register();

      // 初始化离线存储
      await offlineStorageManager.init();

      // 开始性能监控
      performanceMonitor.startMonitoring();

      // 设置网络状态监听
      networkManager.addListener((isOnline) => {
        if (isOnline) {
          // 网络恢复时同步离线数据
          offlineSyncManager.syncPendingActions();
        }
      });

      this.isInitialized = true;
      console.log('Performance optimization initialized successfully');
    } catch (error) {
      console.error('Failed to initialize performance optimization:', error);
    }
  }

  /**
   * 获取性能优化状态
   */
  getStatus(): {
    isInitialized: boolean;
    swSupported: boolean;
    cacheSupported: boolean;
    notificationSupported: boolean;
    isOnline: boolean;
  } {
    return {
      isInitialized: this.isInitialized,
      swSupported: serviceWorkerManager.isServiceWorkerSupported(),
      cacheSupported: 'caches' in window,
      notificationSupported: 'Notification' in window,
      isOnline: networkManager.isOnline(),
    };
  }

  /**
   * 获取性能报告
   */
  getPerformanceReport(): PerformanceReport {
    return performanceMonitor.getReport();
  }

  /**
   * 清理性能数据
   */
  cleanup(): void {
    performanceMonitor.clear();
    offlineSyncManager.clearPendingActions();
    cacheManager.cleanExpiredCache();
  }
}

export const performanceOptimizer = PerformanceOptimizer.getInstance();

// ==================== 性能优化配置 ====================

export const PERFORMANCE_CONFIG = {
  // 渲染优化配置
  RENDER: {
    DEBOUNCE_DELAY: 300,
    THROTTLE_DELAY: 100,
    MEMO_DEPENDENCIES: true,
  },
  
  // 虚拟滚动配置
  VIRTUAL_SCROLL: {
    ITEM_HEIGHT: 50,
    OVERSCAN: 10,
    CONTAINER_HEIGHT: 400,
  },
  
  // 懒加载配置
  LAZY_LOADING: {
    LOADING_DELAY: 200,
    RETRY_COUNT: 3,
    PRELOAD_DISTANCE: 100,
  },
  
  // 缓存配置
  CACHE: {
    STALE_TIME: 5 * 60 * 1000, // 5分钟
    CACHE_TIME: 10 * 60 * 1000, // 10分钟
    MAX_CACHE_SIZE: 50 * 1024 * 1024, // 50MB
  },
  
  // PWA配置
  PWA: {
    CACHE_NAME: 'itsm-cache-v1',
    OFFLINE_DB_NAME: 'itsm-offline',
    SYNC_INTERVAL: 30 * 1000, // 30秒
  },
  
  // 性能监控配置
  MONITORING: {
    METRICS_INTERVAL: 1000, // 1秒
    REPORT_INTERVAL: 60 * 1000, // 1分钟
    ERROR_THRESHOLD: 5,
  },
} as const;

// ==================== 性能优化Hook ====================

/**
 * 使用性能优化
 */
export function usePerformanceOptimization() {
  const [status, setStatus] = useState(performanceOptimizer.getStatus());
  const [isInitialized, setIsInitialized] = useState(false);

  useEffect(() => {
    const initialize = async () => {
      await performanceOptimizer.initialize();
      setIsInitialized(true);
      setStatus(performanceOptimizer.getStatus());
    };

    initialize();
  }, []);

  const getReport = useCallback(() => {
    return performanceOptimizer.getPerformanceReport();
  }, []);

  const cleanup = useCallback(() => {
    performanceOptimizer.cleanup();
  }, []);

  return {
    status,
    isInitialized,
    getReport,
    cleanup,
  };
}

// ==================== 性能优化组件 ====================

/**
 * 性能优化提供者组件
 */
export const PerformanceOptimizationProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isInitialized } = usePerformanceOptimization();

  if (!isInitialized) {
    return <PageLoading message="正在初始化性能优化..." />;
  }

  return <>{children}</>;
};

// ==================== 默认导出 ====================
export default {
  performanceOptimizer,
  PERFORMANCE_CONFIG,
  usePerformanceOptimization,
  PerformanceOptimizationProvider,
};
