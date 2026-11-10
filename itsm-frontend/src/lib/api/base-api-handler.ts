/**
 * API 基础处理器
 * 提供统一的错误处理和响应处理
 */

import { message } from 'antd';

// 错误消息映射
const ERROR_MESSAGES: Record<string, string> = {
  // 网络错误
  'Network Error': 'Network connection failed, please check your network',
  timeout: 'Request timed out, please try again later.',
  'Request timeout': 'Request timed out, please try again later.',

  // HTTP 状态码错误
  '400': 'Wrong request parameters',
  '401': 'Unauthorized, please log in again',
  '403': 'No permission to perform this operation',
  '404': 'The requested resource does not exist',
  '409': 'Resource conflict, please refresh and try again',
  '422': 'Data validation failed',
  '429': 'Requests are too frequent, please try again later.',
  '500': 'Server internal error',
  '502': 'Gateway error',
  '503': 'Service is temporarily unavailable',
  '504': 'Gateway timeout',
};

// API错误类
export class ApiError extends Error {
  code: number;
  details?: unknown;
  requestId?: string;

  constructor(message: string, code: number, details?: unknown, requestId?: string) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.details = details;
    this.requestId = requestId;
  }
}

/**
 * 获取友好的错误消息
 */
export const getFriendlyErrorMessage = (error: unknown): string => {
  if (error instanceof ApiError) {
    return error.message;
  }

  if (error instanceof Error) {
    // 检查是否有映射的错误消息
    for (const [key, msg] of Object.entries(ERROR_MESSAGES)) {
      if (error.message.includes(key)) {
        return msg;
      }
    }
    return error.message;
  }

  return 'Operation failed, please try again later.';
};

/**
 * API 请求包装器
 * 提供统一的错误处理和日志记录
 */
export class ApiHandler {
  /**
   * 包装 API 请求，添加统一的错误处理
   * @param request Promise 请求
   * @param options 选项
   * @returns Promise<T>
   */
  static async handleRequest<T>(
    request: Promise<T>,
    options?: {
      errorMessage?: string;
      showError?: boolean;
      showSuccess?: boolean;
      successMessage?: string;
      silent?: boolean;
    }
  ): Promise<T> {
    const {
      errorMessage,
      showError = true,
      showSuccess = false,
      successMessage,
      silent = false,
    } = options || {};

    try {
      const result = await request;

      // 显示成功消息
      if (showSuccess && successMessage && !silent) {
        message.success(successMessage);
      }

      return result;
    } catch (error) {
      // 获取友好的错误消息
      const friendlyMessage = errorMessage || getFriendlyErrorMessage(error);

      // 显示错误消息
      if (showError && !silent) {
        message.error(friendlyMessage);
      }

      // 记录错误到控制台（开发环境）
      if (process.env.NODE_ENV === 'development') {
        console.error('API Error:', {
          message: friendlyMessage,
          error,
          timestamp: new Date().toISOString(),
        });
      }

      // 重新抛出错误，让调用者可以进一步处理
      throw error;
    }
  }

  /**
   * 批量处理请求
   * @param requests Promise 数组
   * @param options 选项
   * @returns Promise<T[]>
   */
  static async handleBatchRequests<T>(
    requests: Promise<T>[],
    options?: {
      errorMessage?: string;
      showError?: boolean;
      continueOnError?: boolean;
    }
  ): Promise<T[]> {
    const {
      errorMessage = 'Batch operation failed',
      showError = true,
      continueOnError = false,
    } = options || {};

    try {
      if (continueOnError) {
        // 使用 Promise.allSettled 继续执行所有请求
        const results = await Promise.allSettled(requests);
        const successful: T[] = [];
        const failed: unknown[] = [];

        results.forEach((result, index) => {
          if (result.status === 'fulfilled') {
            successful.push(result.value);
          } else {
            failed.push(result.reason);
            console.error(`Request ${index} failed:`, result.reason);
          }
        });

        // 如果有失败的请求，显示警告
        if (failed.length > 0 && showError) {
          message.warning(`${successful.length} Operations successful,${failed.length} Operations failed`);
        }

        return successful;
      } else {
        // 使用 Promise.all，任何一个失败都会抛出错误
        return await Promise.all(requests);
      }
    } catch (error) {
      if (showError) {
        message.error(errorMessage);
      }
      throw error;
    }
  }

  /**
   * 延迟重试
   * @param fn 要重试的函数
   * @param options 重试选项
   * @returns Promise<T>
   */
  static async retryRequest<T>(
    fn: () => Promise<T>,
    options?: {
      maxRetries?: number;
      delay?: number;
      backoff?: boolean;
    }
  ): Promise<T> {
    const { maxRetries = 3, delay = 1000, backoff = true } = options || {};

    let lastError: unknown;

    for (let i = 0; i < maxRetries; i++) {
      try {
        return await fn();
      } catch (error) {
        lastError = error;

        if (i < maxRetries - 1) {
          // 计算延迟时间（如果启用了退避策略）
          const waitTime = backoff ? delay * Math.pow(2, i) : delay;
          await new Promise(resolve => setTimeout(resolve, waitTime));
        }
      }
    }

    throw lastError;
  }
}

/**
 * 便捷函数：处理请求
 */
export const handleApiRequest = ApiHandler.handleRequest;

/**
 * 便捷函数：批量处理请求
 */
export const handleBatchApiRequests = ApiHandler.handleBatchRequests;

/**
 * 便捷函数：重试请求
 */
export const retryApiRequest = ApiHandler.retryRequest;
