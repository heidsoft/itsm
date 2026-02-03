// 环境配置工具

// 环境类型
export type Environment = 'development' | 'production' | 'test';

// 获取当前环境
export const getEnvironment = (): Environment => {
  if (typeof window !== 'undefined') {
    // 客户端环境
    return (process.env.NODE_ENV as Environment) || 'development';
  }
  // 服务端环境
  return (process.env.NODE_ENV as Environment) || 'development';
};

// 环境配置
export const env = {
  isDevelopment: getEnvironment() === 'development',
  isProduction: getEnvironment() === 'production',
  isTest: getEnvironment() === 'test',
  
  // 功能开关
  features: {
    debugMode: getEnvironment() === 'development',
    performanceMonitoring: getEnvironment() === 'production',
    errorReporting: getEnvironment() === 'production',
    consoleLogs: getEnvironment() === 'development',
  },
  
  // API配置
  api: {
    baseUrl: process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080',
    timeout: parseInt(process.env.NEXT_PUBLIC_API_TIMEOUT || '10000'),
    retryCount: parseInt(process.env.NEXT_PUBLIC_API_RETRY_COUNT || '3'),
  },
  
  // 应用配置
  app: {
    name: 'ITSM Platform',
    version: process.env.NEXT_PUBLIC_APP_VERSION || '1.0.0',
    buildTime: process.env.NEXT_PUBLIC_BUILD_TIME || '',
  },
};

// 日志工具
/* eslint-disable no-console */
export const logger = {
  // 开发环境日志
  debug: (...args: unknown[]) => {
    if (env.features.consoleLogs) {
      console.log('[DEBUG]', ...args);
    }
  },
  
  // 信息日志
  info: (...args: unknown[]) => {
    if (env.features.consoleLogs) {
      console.info('[INFO]', ...args);
    }
  },
  
  // 警告日志
  warn: (...args: unknown[]) => {
    if (env.features.consoleLogs) {
      console.warn('[WARN]', ...args);
    }
  },
  
  // 错误日志（生产环境也会输出）
  error: (...args: unknown[]) => {
    console.error('[ERROR]', ...args);
    
    // 生产环境错误上报
    if (env.features.errorReporting) {
      // 这里可以集成错误上报服务
      // reportError(args);
    }
  },
  
  // 性能日志
  performance: (label: string, data: Record<string, unknown>) => {
    if (env.features.performanceMonitoring) {
      console.log(`[PERFORMANCE] ${label}:`, data);
    }
  },
  
  // 安全日志
  security: (event: string, data: Record<string, unknown>) => {
    if (env.features.consoleLogs) {
      console.log('[SECURITY]', event, data);
    }
    
    // 生产环境安全事件上报
    if (env.features.errorReporting) {
      // reportSecurityEvent(event, data);
    }
  },
};

// 性能监控工具
export const performance = {
  // 开始计时
  start: (label: string) => {
    if (env.features.performanceMonitoring) {
      console.time(`[PERF] ${label}`);
      return label;
    }
    return null;
  },
  
  // 结束计时
  end: (label: string) => {
    if (env.features.performanceMonitoring) {
      console.timeEnd(`[PERF] ${label}`);
    }
  },
  
  // 测量函数执行时间
  measure: <T>(label: string, fn: () => T): T => {
    if (env.features.performanceMonitoring) {
      const start = Date.now();
      const result = fn();
      const end = Date.now();
      logger.performance(label, { duration: `${end - start}ms` });
      return result;
    }
    return fn();
  },
  
  // 异步函数性能测量
  measureAsync: async <T>(label: string, fn: () => Promise<T>): Promise<T> => {
    if (env.features.performanceMonitoring) {
      const start = Date.now();
      const result = await fn();
      const end = Date.now();
      logger.performance(label, { duration: `${end - start}ms` });
      return result;
    }
    return fn();
  },
};

// 错误处理工具
export const errorHandler = {
  // 处理API错误
  handleApiError: (error: unknown, context?: string) => {
    const errorMessage = error instanceof Error ? error.message : String(error);
    const errorContext = context ? `[${context}]` : '';
    
    logger.error(`${errorContext} API Error:`, errorMessage);
    
    // 生产环境错误上报
    if (env.features.errorReporting) {
      // reportApiError(error, context);
    }
    
    return {
      success: false,
      error: errorMessage,
      context,
    };
  },
  
  // 处理表单验证错误
  handleValidationError: (errors: Record<string, string[]>) => {
    logger.warn('Form validation errors:', errors);
    
    return {
      success: false,
      errors,
      message: '表单验证失败',
    };
  },
  
  // 处理网络错误
  handleNetworkError: (error: unknown) => {
    const isNetworkError = error instanceof Error && 
      (error.message.includes('Network Error') || 
       error.message.includes('Failed to fetch'));
    
    if (isNetworkError) {
      logger.error('Network error detected:', error);
      
      // 生产环境网络错误上报
      if (env.features.errorReporting) {
        // reportNetworkError(error);
      }
      
      return {
        success: false,
        error: '网络连接失败，请检查网络设置',
        type: 'network',
      };
    }
    
    return {
      success: false,
      error: '未知错误',
      type: 'unknown',
    };
  },
};

// 开发工具
export const devTools = {
  // 只在开发环境执行的函数
  onlyInDev: <T>(fn: () => T, fallback?: T): T | undefined => {
    if (env.isDevelopment) {
      return fn();
    }
    return fallback;
  },
  
  // 开发环境调试信息
  debugInfo: (component: string, data: unknown) => {
    if (env.isDevelopment) {
      logger.debug(`[${component}]`, data);
    }
  },
  
  // 开发环境性能标记
  mark: (name: string) => {
    if (env.isDevelopment && typeof window !== 'undefined' && window.performance) {
      window.performance.mark(name);
    }
  },
  
  // 开发环境性能测量
  measure: (name: string, startMark: string, endMark: string) => {
    if (env.isDevelopment && typeof window !== 'undefined' && window.performance) {
      try {
        const measure = window.performance.measure(name, startMark, endMark);
        logger.debug(`Performance measure [${name}]:`, measure.duration);
      } catch (error) {
        logger.warn(`Failed to measure performance [${name}]:`, error);
      }
    }
  },
};

// 导出默认配置
export default env;
