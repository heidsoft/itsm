import { renderHook } from '@testing-library/react';
import { useErrorHandler, handleGlobalError, handleErrorBoundary } from '../useErrorHandler';

type MockApi = {
  success: jest.Mock;
  error: jest.Mock;
  warning: jest.Mock;
  info: jest.Mock;
};

// antd message：App 上下文实例与静态实例分开，用于断言优先使用上下文实例。
jest.mock('antd', () => {
  const newApi = (): MockApi => ({
    success: jest.fn(),
    error: jest.fn(),
    warning: jest.fn(),
    info: jest.fn(),
  });
  const contextMessage = newApi();
  const staticMessage = newApi();
  const state = { useAppMessage: contextMessage as MockApi | undefined };
  return {
    App: { useApp: () => ({ message: state.useAppMessage }) },
    message: staticMessage,
    mockContextMessage: contextMessage,
    mockStaticMessage: staticMessage,
    mockState: state,
  };
});

const antdMock = jest.requireMock('antd') as {
  mockContextMessage: MockApi;
  mockStaticMessage: MockApi;
  mockState: { useAppMessage: MockApi | undefined };
};
const mockMessage = antdMock.mockContextMessage;
const mockStaticMessage = antdMock.mockStaticMessage;

describe('useErrorHandler', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    antdMock.mockState.useAppMessage = mockMessage;
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

  it('动作标签与失败原因一起提示用户', () => {
    // 工单详情页分配失败历史上只弹「分配失败」，end_user 无法知道是 403 权限不足，
    // 只会反复点同一个按钮。
    const { result } = renderHook(() => useErrorHandler());

    result.current.handleError(new Error('权限不足'), 'assignTicket', '分配失败');

    expect(mockMessage.error).toHaveBeenCalledWith('分配失败：权限不足');
  });

  it('403 分类原因同样带出', () => {
    const { result } = renderHook(() => useErrorHandler());

    result.current.handleError(new Error('403 Forbidden'), 'loadRules', '加载失败');

    expect(mockMessage.error).toHaveBeenCalledWith('加载失败：权限不足，无法执行此操作');
  });

  it('优先使用 App 上下文的反馈实例', () => {
    // 主题/locale 由 App 上下文提供；静态实例无法继承，且在没有 holder 时不可见。
    const { result } = renderHook(() => useErrorHandler());

    result.current.handleError(new Error('boom'), 'ctx');

    expect(mockMessage.error).toHaveBeenCalledWith('boom');
    expect(mockStaticMessage.error).not.toHaveBeenCalled();
  });

  it('没有 App 上下文时退回静态实例而不是静默', () => {
    antdMock.mockState.useAppMessage = undefined;
    const { result } = renderHook(() => useErrorHandler());

    result.current.handleError(new Error('boom'), 'ctx');

    expect(mockStaticMessage.error).toHaveBeenCalledWith('boom');
  });

  it('showMessage=false 时不弹提示', () => {
    const { result } = renderHook(() => useErrorHandler({ showMessage: false }));

    result.current.handleError(new Error('boom'), 'ctx');

    expect(mockMessage.error).not.toHaveBeenCalled();
  });
});

describe('handleGlobalError', () => {
  beforeEach(() => {
    jest.clearAllMocks();
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
