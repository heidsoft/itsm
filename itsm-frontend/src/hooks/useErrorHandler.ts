/**
 * 统一错误处理 Hook
 * 提供一致的错误处理模式：日志 + 用户提示 + 状态设置
 */
import { App } from 'antd';
import { AxiosError } from 'axios';
import { useState } from 'react';

interface ErrorHandlerOptions {
  /** 错误时执行的回调（如重置 loading 状态） */
  onError?: () => void;
  /** 自定义错误消息，不提供则使用默认消息 */
  customMessage?: string;
  /** 是否显示用户提示，默认 true */
  showMessage?: boolean;
}

/**
 * 统一的错误处理函数
 * 自动从 AxiosError 提取有意义的错误信息
 */
export const getErrorMessage = (error: unknown, fallback: string = '操作失败，请稍后重试'): string => {
  if (error instanceof AxiosError) {
    // API 返回的业务错误
    if (error.response?.data?.message) {
      return error.response.data.message;
    }
    // 网络错误
    if (error.code === 'ECONNABORTED') {
      return '请求超时，请稍后重试';
    }
    if (error.code === 'ERR_NETWORK') {
      return '网络连接失败，请检查网络';
    }
    // 服务端错误
    if (error.response?.status && error.response.status >= 500) {
      return '服务器错误，请稍后重试';
    }
    // 未授权
    if (error.response?.status === 401) {
      return '登录已过期，请重新登录';
    }
    // 无权限
    if (error.response?.status === 403) {
      return '您没有权限执行此操作';
    }
  }
  // 其他错误
  if (error instanceof Error) {
    return error.message || fallback;
  }
  return fallback;
};

/**
 * 错误处理 Hook
 * 使用方式：
 * const { handleError } = useErrorHandler();
 * // 或
 * const handleError = useErrorHandler().handleError;
 *
 * // 在 catch 中使用
 * try {
 *   await apiCall();
 * } catch (error) {
 *   handleError(error);
 * }
 *
 * // 带回调
 * handleError(error, {
 *   onError: () => setLoading(false),
 *   customMessage: '获取数据失败',
 *   showMessage: true
 * });
 */
export const useErrorHandler = () => {
  const { message } = App.useApp();

  const handleError = (error: unknown, options: ErrorHandlerOptions = {}) => {
    const {
      onError,
      customMessage,
      showMessage = true,
    } = options;

    // 1. 记录日志
    console.error('Error occurred:', error);

    // 2. 执行回调
    onError?.();

    // 3. 显示用户提示
    if (showMessage) {
      const errorMsg = customMessage || getErrorMessage(error);
      message.error(errorMsg);
    }
  };

  /**
   * 仅记录日志，不显示用户提示（静默失败）
   * 用于预期内的错误，如网络超时重试
   */
  const silentError = (error: unknown, onError?: () => void) => {
    console.warn('Silent error:', error);
    onError?.();
  };

  /**
   * 成功提示
   */
  const handleSuccess = (msg: string = '操作成功') => {
    message.success(msg);
  };

  /**
   * 警告提示
   */
  const handleWarning = (msg: string) => {
    message.warning(msg);
  };

  /**
   * 信息提示
   */
  const handleInfo = (msg: string) => {
    message.info(msg);
  };

  return {
    handleError,
    silentError,
    handleSuccess,
    handleWarning,
    handleInfo,
    getErrorMessage,
  };
};

/**
 * 异步操作包装器
 * 自动处理 loading 状态和错误
 *
 * 使用方式：
 * const { exec, loading } = useAsyncOperation();
 *
 * const handleSubmit = async () => {
 *   await exec(
 *     () => api.create(data),
 *     {
 *       onSuccess: () => { message.success('创建成功'); router.push('/list'); },
 *       errorMessage: '创建失败'
 *     }
 *   );
 * };
 */
export const useAsyncOperation = (defaultOptions?: ErrorHandlerOptions) => {
  const { handleError, handleSuccess, handleInfo, getErrorMessage } = useErrorHandler();
  const [loading, setLoading] = useState(false);

  const exec = async <T>(
    fn: () => Promise<T>,
    options: {
      onSuccess?: (data: T) => void;
      onError?: (error: unknown) => void;
      successMessage?: string;
      errorMessage?: string;
      completeMessage?: string; // 不管成功失败都显示的消息
    } = {}
  ): Promise<T | undefined> => {
    const {
      onSuccess,
      onError,
      successMessage,
      errorMessage,
      completeMessage,
    } = options;

    setLoading(true);
    try {
      const result = await fn();
      if (successMessage) {
        handleSuccess(successMessage);
      }
      onSuccess?.(result);
      return result;
    } catch (error) {
      const msg = errorMessage || getErrorMessage(error);
      handleError(error, { customMessage: msg, ...defaultOptions });
      onError?.(error);
      return undefined;
    } finally {
      setLoading(false);
      if (completeMessage) {
        handleInfo(completeMessage);
      }
    }
  };

  return {
    exec,
    loading,
    setLoading,
  };
};

export default useErrorHandler;