import { renderHook } from '@testing-library/react';
import { useErrorHandler, handleGlobalError, handleErrorBoundary } from '../useErrorHandler';

// Mock antd message
const mockMessage = {
  success: jest.fn(),
  error: jest.fn(),
  warning: jest.fn(),
  info: jest.fn(),
};

jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    error: jest.fn(),
    warning: jest.fn(),
    info: jest.fn(),
  },
}));

describe('useErrorHandler', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    (console.error as jest.Mock).mockRestore();
  });

  it('should return error handler functions', () => {
    const { result } = renderHook(() => useErrorHandler());

    expect(result.current.handleError).toBeDefined();
    expect(result.current.handleAsyncError).toBeDefined();
    expect(result.current.handleValidationError).toBeDefined();
    expect(result.current.handleNetworkError).toBeDefined();
  });

  it('should handle basic Error', () => {
    const { result } = renderHook(() => useErrorHandler());

    const appError = result.current.handleError(new Error('Something went wrong'), 'test context');

    expect(appError.message).toBe('Something went wrong');
    expect(appError.context).toBe('test context');
    expect(console.error).toHaveBeenCalled();
  });

  it('should handle network errors', () => {
    const { result } = renderHook(() => useErrorHandler());

    const appError = result.current.handleError(new Error('Network Error'));

    expect(appError.message).toBe('网络连接失败，请检查网络设置');
  });

  it('should handle timeout errors', () => {
    const { result } = renderHook(() => useErrorHandler());

    const appError = result.current.handleError(new Error('Request timeout'));

    expect(appError.message).toBe('请求超时，请稍后重试');
  });

  it('should handle 401 unauthorized', () => {
    const { result } = renderHook(() => useErrorHandler());

    const appError = result.current.handleError(new Error('401 Unauthorized'));

    expect(appError.message).toBe('登录已过期，请重新登录');
  });

  it('should handle 403 forbidden', () => {
    const { result } = renderHook(() => useErrorHandler());

    const appError = result.current.handleError(new Error('403 Forbidden'));

    expect(appError.message).toBe('权限不足，无法执行此操作');
  });

  it('should handle async errors', async () => {
    const { result } = renderHook(() => useErrorHandler());

    const failingFn = () => Promise.reject(new Error('Async failure'));
    const returnedValue = await result.current.handleAsyncError(failingFn, 'async context');

    expect(returnedValue).toBeNull();
    expect(console.error).toHaveBeenCalled();
  });

  it('should return value on async success', async () => {
    const { result } = renderHook(() => useErrorHandler());

    const successFn = () => Promise.resolve('success data');
    const returnedValue = await result.current.handleAsyncError(successFn, 'async context');

    expect(returnedValue).toBe('success data');
  });

  it('should use custom message when provided', () => {
    const { result } = renderHook(() => useErrorHandler());

    const appError = result.current.handleError(
      new Error('some error'),
      'context',
      'Custom user message'
    );

    expect(appError.message).toBe('Custom user message');
  });
});

describe('handleGlobalError', () => {
  beforeEach(() => {
    jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    (console.error as jest.Mock).mockRestore();
  });

  it('should handle global errors', () => {
    const appError = handleGlobalError(new Error('Global error'), 'global context');

    expect(appError.message).toBe('Global error');
    expect(appError.context).toBe('global context');
  });
});

describe('handleErrorBoundary', () => {
  beforeEach(() => {
    jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    (console.error as jest.Mock).mockRestore();
  });

  it('should handle error boundary errors', () => {
    handleErrorBoundary(new Error('Render error'), { componentStack: '<div>' });

    expect(console.error).toHaveBeenCalledWith(
      'ErrorBoundary caught an error:',
      expect.any(Error),
      expect.any(Object)
    );
  });
});
