import { useCallback } from 'react';
import { message } from 'antd';

// 错误类型定义
export interface AppError {
  code?: string;
  message: string;
  details?: any;
  context?: string;
}

// 错误严重程度
export enum ErrorSeverity {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical',
}

// 错误处理配置
interface ErrorHandlerConfig {
  showMessage?: boolean;
  logToConsole?: boolean;
  reportToService?: boolean;
  fallbackMessage?: string;
}

// 默认配置
const defaultConfig: ErrorHandlerConfig = {
  showMessage: true,
  logToConsole: true,
  reportToService: true,
  fallbackMessage: '操作失败，请稍后重试',
};

// 获取用户友好的错误消息
const getUserFriendlyMessage = (error: any): string => {
  if (typeof error === 'string') {
    return error;
  }

  if (error?.message) {
    // 网络错误
    if (error.message.includes('Network Error')) {
      return '网络连接失败，请检查网络设置';
    }
    
    // 超时错误
    if (error.message.includes('timeout')) {
      return '请求超时，请稍后重试';
    }
    
    // 权限错误
    if (error.message.includes('401') || error.message.includes('Unauthorized')) {
      return '登录已过期，请重新登录';
    }
    
    // 权限不足
    if (error.message.includes('403') || error.message.includes('Forbidden')) {
      return '权限不足，无法执行此操作';
    }
    
    // 资源不存在
    if (error.message.includes('404') || error.message.includes('Not Found')) {
      return '请求的资源不存在';
    }
    
    // 服务器错误
    if (error.message.includes('500') || error.message.includes('Internal Server Error')) {
      return '服务器内部错误，请联系管理员';
    }
    
    return error.message;
  }

  return '未知错误';
};

// 错误上报服务
const reportError = (error: AppError, config: ErrorHandlerConfig) => {
  if (!config.reportToService) return;

  // 这里可以集成错误上报服务，如 Sentry、Bugsnag 等
  console.error('Error reported:', {
    code: error.code,
    message: error.message,
    details: error.details,
    context: error.context,
    timestamp: new Date().toISOString(),
    userAgent: navigator.userAgent,
    url: window.location.href,
  });
};

// 自定义错误处理Hook
export const useErrorHandler = (config: Partial<ErrorHandlerConfig> = {}) => {
  const mergedConfig = { ...defaultConfig, ...config };

  const handleError = useCallback((
    error: any,
    context?: string,
    customMessage?: string
  ) => {
    const appError: AppError = {
      code: error?.code,
      message: customMessage || getUserFriendlyMessage(error),
      details: error,
      context,
    };

    // 控制台日志
    if (mergedConfig.logToConsole) {
      console.error(`Error in ${context || 'unknown context'}:`, error);
    }

    // 显示用户消息
    if (mergedConfig.showMessage) {
      message.error(appError.message);
    }

    // 错误上报
    reportError(appError, mergedConfig);

    return appError;
  }, [mergedConfig]);

  // 处理异步操作错误
  const handleAsyncError = useCallback(async <T>(
    asyncFn: () => Promise<T>,
    context?: string,
    customMessage?: string
  ): Promise<T | null> => {
    try {
      return await asyncFn();
    } catch (error) {
      handleError(error, context, customMessage);
      return null;
    }
  }, [handleError]);

  // 处理表单验证错误
  const handleValidationError = useCallback((
    errors: any[],
    context?: string
  ) => {
    const errorMessages = errors.map(err => err.message || err).join('; ');
    handleError(new Error(errorMessages), context, '表单验证失败');
  }, [handleError]);

  // 处理网络错误
  const handleNetworkError = useCallback((
    error: any,
    context?: string
  ) => {
    let message = '网络连接失败';
    
    if (error?.code === 'NETWORK_ERROR') {
      message = '网络连接失败，请检查网络设置';
    } else if (error?.code === 'TIMEOUT') {
      message = '请求超时，请稍后重试';
    } else if (error?.code === 'SERVER_ERROR') {
      message = '服务器错误，请稍后重试';
    }

    handleError(error, context, message);
  }, [handleError]);

  return {
    handleError,
    handleAsyncError,
    handleValidationError,
    handleNetworkError,
  };
};

// 全局错误处理函数（用于非Hook环境）
export const handleGlobalError = (
  error: any,
  context?: string,
  customMessage?: string
) => {
  const appError: AppError = {
    code: error?.code,
    message: customMessage || getUserFriendlyMessage(error),
    details: error,
    context,
  };

  console.error(`Global error in ${context || 'unknown context'}:`, error);
  message.error(appError.message);
  reportError(appError, defaultConfig);

  return appError;
};

// 错误边界组件使用的错误处理
export const handleErrorBoundary = (error: Error, errorInfo: any) => {
  const appError: AppError = {
    code: 'COMPONENT_ERROR',
    message: '组件渲染错误',
    details: { error, errorInfo },
    context: 'ErrorBoundary',
  };

  console.error('ErrorBoundary caught an error:', error, errorInfo);
  reportError(appError, defaultConfig);
};

export default useErrorHandler;
