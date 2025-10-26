/**
 * 统一错误处理系统
 * 提供一致的错误处理、日志记录和用户反馈
 */

import { message, notification } from 'antd';
import { ApiError, OperationResult } from '@/types/api';

// 错误类型枚举
export enum ErrorType {
  NETWORK = 'NETWORK',
  VALIDATION = 'VALIDATION',
  AUTHENTICATION = 'AUTHENTICATION',
  AUTHORIZATION = 'AUTHORIZATION',
  NOT_FOUND = 'NOT_FOUND',
  SERVER = 'SERVER',
  CLIENT = 'CLIENT',
  UNKNOWN = 'UNKNOWN',
}

// 错误严重程度
export enum ErrorSeverity {
  LOW = 'LOW',
  MEDIUM = 'MEDIUM',
  HIGH = 'HIGH',
  CRITICAL = 'CRITICAL',
}

// 错误信息接口
export interface ErrorInfo {
  type: ErrorType;
  severity: ErrorSeverity;
  message: string;
  code?: string | number;
  details?: Record<string, unknown>;
  stack?: string;
  timestamp: string;
  userId?: number;
  requestId?: string;
  url?: string;
}

// 错误处理器类
export class ErrorHandler {
  private static instance: ErrorHandler;
  private errorLog: ErrorInfo[] = [];
  private maxLogSize = 100;

  private constructor() {
    this.setupGlobalErrorHandlers();
  }

  public static getInstance(): ErrorHandler {
    if (!ErrorHandler.instance) {
      ErrorHandler.instance = new ErrorHandler();
    }
    return ErrorHandler.instance;
  }

  // 设置全局错误处理器
  private setupGlobalErrorHandlers(): void {
    // 检查是否在浏览器环境中
    if (typeof window === 'undefined') {
      return;
    }

    // 捕获未处理的Promise拒绝
    window.addEventListener('unhandledrejection', event => {
      this.handleError({
        type: ErrorType.UNKNOWN,
        severity: ErrorSeverity.MEDIUM,
        message: event.reason?.message || 'Unhandled Promise Rejection',
        details: { reason: event.reason },
        timestamp: new Date().toISOString(),
        url: window.location.href,
      });
    });

    // 捕获全局JavaScript错误
    window.addEventListener('error', event => {
      this.handleError({
        type: ErrorType.CLIENT,
        severity: ErrorSeverity.MEDIUM,
        message: event.message,
        details: {
          filename: event.filename,
          lineno: event.lineno,
          colno: event.colno,
        },
        stack: event.error?.stack,
        timestamp: new Date().toISOString(),
        url: window.location.href,
      });
    });
  }

  // 处理API错误
  public handleApiError(error: unknown, context?: string): ApiError {
    let errorInfo: ErrorInfo;

    if (this.isApiError(error)) {
      errorInfo = {
        type: this.mapHttpStatusToErrorType(error.status || 500),
        severity: this.mapErrorTypeToSeverity(this.mapHttpStatusToErrorType(error.status || 500)),
        message: error.message || 'API请求失败',
        code: error.status,
        details: { context, originalError: error },
        timestamp: new Date().toISOString(),
        url: window.location.href,
      };
    } else if (error instanceof Error) {
      errorInfo = {
        type: ErrorType.NETWORK,
        severity: ErrorSeverity.MEDIUM,
        message: error.message,
        details: { context, originalError: error },
        stack: error.stack,
        timestamp: new Date().toISOString(),
        url: window.location.href,
      };
    } else {
      errorInfo = {
        type: ErrorType.UNKNOWN,
        severity: ErrorSeverity.MEDIUM,
        message: '未知错误',
        details: { context, originalError: error },
        timestamp: new Date().toISOString(),
        url: window.location.href,
      };
    }

    this.handleError(errorInfo);
    return this.createApiError(errorInfo);
  }

  // 处理业务错误
  public handleBusinessError(error: Error, context?: string): void {
    const errorInfo: ErrorInfo = {
      type: ErrorType.CLIENT,
      severity: ErrorSeverity.MEDIUM,
      message: error.message,
      details: { context },
      stack: error.stack,
      timestamp: new Date().toISOString(),
      url: window.location.href,
    };

    this.handleError(errorInfo);
  }

  // 处理验证错误
  public handleValidationError(errors: Record<string, string[]>): void {
    const errorInfo: ErrorInfo = {
      type: ErrorType.VALIDATION,
      severity: ErrorSeverity.LOW,
      message: '表单验证失败',
      details: { validationErrors: errors },
      timestamp: new Date().toISOString(),
      url: window.location.href,
    };

    this.handleError(errorInfo);
    this.showValidationErrors(errors);
  }

  // 核心错误处理方法
  private handleError(errorInfo: ErrorInfo): void {
    // 记录错误日志
    this.logError(errorInfo);

    // 根据严重程度显示用户通知
    this.showUserNotification(errorInfo);

    // 发送错误报告到服务器（生产环境）
    if (process.env.NODE_ENV === 'production') {
      this.reportError(errorInfo);
    }
  }

  // 记录错误日志
  private logError(errorInfo: ErrorInfo): void {
    this.errorLog.unshift(errorInfo);

    // 限制日志大小
    if (this.errorLog.length > this.maxLogSize) {
      this.errorLog = this.errorLog.slice(0, this.maxLogSize);
    }

    // 控制台输出
    console.error('ErrorHandler:', errorInfo);
  }

  // 显示用户通知
  private showUserNotification(errorInfo: ErrorInfo): void {
    const { type, severity, message } = errorInfo;

    // 根据严重程度选择通知方式
    switch (severity) {
      case ErrorSeverity.CRITICAL:
        notification.error({
          message: '系统错误',
          description: message,
          duration: 0, // 不自动关闭
        });
        break;
      case ErrorSeverity.HIGH:
        notification.error({
          message: '操作失败',
          description: message,
          duration: 5,
        });
        break;
      case ErrorSeverity.MEDIUM:
        message.error(message);
        break;
      case ErrorSeverity.LOW:
        message.warning(message);
        break;
    }
  }

  // 显示验证错误
  private showValidationErrors(errors: Record<string, string[]>): void {
    const errorMessages = Object.values(errors).flat();
    errorMessages.forEach(msg => message.error(msg));
  }

  // 发送错误报告到服务器
  private async reportError(errorInfo: ErrorInfo): Promise<void> {
    try {
      await fetch('/api/errors', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(errorInfo),
      });
    } catch (error) {
      console.error('Failed to report error:', error);
    }
  }

  // 工具方法
  private isApiError(error: unknown): error is { status?: number; message?: string } {
    return typeof error === 'object' && error !== null && ('status' in error || 'message' in error);
  }

  private mapHttpStatusToErrorType(status: number): ErrorType {
    if (status >= 200 && status < 300) return ErrorType.UNKNOWN;
    if (status === 401) return ErrorType.AUTHENTICATION;
    if (status === 403) return ErrorType.AUTHORIZATION;
    if (status === 404) return ErrorType.NOT_FOUND;
    if (status >= 400 && status < 500) return ErrorType.CLIENT;
    if (status >= 500) return ErrorType.SERVER;
    return ErrorType.NETWORK;
  }

  private mapErrorTypeToSeverity(type: ErrorType): ErrorSeverity {
    switch (type) {
      case ErrorType.AUTHENTICATION:
      case ErrorType.AUTHORIZATION:
        return ErrorSeverity.HIGH;
      case ErrorType.SERVER:
        return ErrorSeverity.CRITICAL;
      case ErrorType.VALIDATION:
        return ErrorSeverity.LOW;
      default:
        return ErrorSeverity.MEDIUM;
    }
  }

  private createApiError(errorInfo: ErrorInfo): ApiError {
    return {
      code: typeof errorInfo.code === 'number' ? errorInfo.code : 500,
      message: errorInfo.message,
      details: errorInfo.details,
    };
  }

  // 公共方法
  public getErrorLog(): ErrorInfo[] {
    return [...this.errorLog];
  }

  public clearErrorLog(): void {
    this.errorLog = [];
  }

  public getErrorStats(): Record<ErrorType, number> {
    const stats = {} as Record<ErrorType, number>;

    Object.values(ErrorType).forEach(type => {
      stats[type] = 0;
    });

    this.errorLog.forEach(error => {
      stats[error.type]++;
    });

    return stats;
  }
}

// 创建全局错误处理器实例
export const errorHandler = ErrorHandler.getInstance();

// 便捷的错误处理函数
export const handleError = (error: unknown, context?: string): ApiError => {
  return errorHandler.handleApiError(error, context);
};

export const handleBusinessError = (error: Error, context?: string): void => {
  errorHandler.handleBusinessError(error, context);
};

export const handleValidationError = (errors: Record<string, string[]>): void => {
  errorHandler.handleValidationError(errors);
};

// React Hook for error handling
export const useErrorHandler = () => {
  return {
    handleError,
    handleBusinessError,
    handleValidationError,
    getErrorLog: () => errorHandler.getErrorLog(),
    getErrorStats: () => errorHandler.getErrorStats(),
  };
};

// 错误边界组件
export class ErrorBoundary extends React.Component<
  { children: React.ReactNode; fallback?: React.ComponentType<{ error: Error }> },
  { hasError: boolean; error?: Error }
> {
  constructor(props: {
    children: React.ReactNode;
    fallback?: React.ComponentType<{ error: Error }>;
  }) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    errorHandler.handleBusinessError(error, `ErrorBoundary: ${errorInfo.componentStack}`);
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        const FallbackComponent = this.props.fallback;
        return <FallbackComponent error={this.state.error!} />;
      }

      return (
        <div className='flex items-center justify-center min-h-[400px] p-8'>
          <div className='text-center'>
            <h2 className='text-xl font-semibold text-red-600 mb-4'>出现错误</h2>
            <p className='text-gray-600 mb-4'>{this.state.error?.message}</p>
            <button
              onClick={() => this.setState({ hasError: false, error: undefined })}
              className='px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600'
            >
              重试
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
