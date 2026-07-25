import { ApiError, getFriendlyErrorMessage, ApiHandler, handleApiRequest, handleBatchApiRequests, retryApiRequest } from '../base-api-handler';
import { message } from 'antd';

jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    error: jest.fn(),
    warning: jest.fn(),
  },
}));

describe('base-api-handler', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('ApiError', () => {
    it('should create with message, code, details, requestId', () => {
      const err = new ApiError('Not found', 404, { id: 1 }, 'req-123');
      expect(err.message).toBe('Not found');
      expect(err.code).toBe(404);
      expect(err.details).toEqual({ id: 1 });
      expect(err.requestId).toBe('req-123');
      expect(err.name).toBe('ApiError');
      expect(err instanceof Error).toBe(true);
    });

    it('should work with minimal params', () => {
      const err = new ApiError('Error', 500);
      expect(err.code).toBe(500);
      expect(err.details).toBeUndefined();
    });
  });

  describe('getFriendlyErrorMessage', () => {
    it('should return ApiError message directly', () => {
      const err = new ApiError('Custom message', 400);
      expect(getFriendlyErrorMessage(err)).toBe('Custom message');
    });

    it('should map Network Error', () => {
      const err = new Error('Network Error');
      expect(getFriendlyErrorMessage(err)).toBe('网络连接失败，请检查您的网络');
    });

    it('should map timeout', () => {
      const err = new Error('timeout');
      expect(getFriendlyErrorMessage(err)).toBe('请求超时，请稍后重试');
    });

    it('should map HTTP status codes', () => {
      expect(getFriendlyErrorMessage(new Error('401'))).toBe('未授权，请重新登录');
      expect(getFriendlyErrorMessage(new Error('403'))).toBe('没有权限执行此操作');
      expect(getFriendlyErrorMessage(new Error('404'))).toBe('请求的资源不存在');
      expect(getFriendlyErrorMessage(new Error('500'))).toBe('服务器内部错误');
    });

    it('should return error message for unmapped errors', () => {
      expect(getFriendlyErrorMessage(new Error('Some custom error'))).toBe('Some custom error');
    });

    it('should handle non-Error values', () => {
      expect(getFriendlyErrorMessage('string error')).toBe('操作失败，请稍后重试');
      expect(getFriendlyErrorMessage(null)).toBe('操作失败，请稍后重试');
      expect(getFriendlyErrorMessage(undefined)).toBe('操作失败，请稍后重试');
    });
  });

  describe('ApiHandler.handleRequest', () => {
    it('should return result on success', async () => {
      const result = await ApiHandler.handleRequest(Promise.resolve({ id: 1 }));
      expect(result).toEqual({ id: 1 });
    });

    it('should show success message when configured', async () => {
      await ApiHandler.handleRequest(Promise.resolve('ok'), { showSuccess: true, successMessage: 'Done!' });
      expect(message.success).toHaveBeenCalledWith('Done!');
    });

    it('should not show success message when silent', async () => {
      await ApiHandler.handleRequest(Promise.resolve('ok'), { showSuccess: true, successMessage: 'Done!', silent: true });
      expect(message.success).not.toHaveBeenCalled();
    });

    it('should show error message on failure', async () => {
      await expect(ApiHandler.handleRequest(Promise.reject(new Error('Network Error')))).rejects.toThrow();
      expect(message.error).toHaveBeenCalledWith('网络连接失败，请检查您的网络');
    });

    it('should show custom error message when provided', async () => {
      await expect(ApiHandler.handleRequest(Promise.reject(new Error('fail')), { errorMessage: 'Custom error' })).rejects.toThrow();
      expect(message.error).toHaveBeenCalledWith('Custom error');
    });

    it('should not show error message when showError is false', async () => {
      await expect(ApiHandler.handleRequest(Promise.reject(new Error('fail')), { showError: false })).rejects.toThrow();
      expect(message.error).not.toHaveBeenCalled();
    });

    it('should not show error message when silent', async () => {
      await expect(ApiHandler.handleRequest(Promise.reject(new Error('fail')), { silent: true })).rejects.toThrow();
      expect(message.error).not.toHaveBeenCalled();
    });

    it('should re-throw the original error', async () => {
      const err = new Error('Original');
      await expect(ApiHandler.handleRequest(Promise.reject(err))).rejects.toBe(err);
    });
  });

  describe('ApiHandler.handleBatchRequests', () => {
    it('should resolve all requests with Promise.all by default', async () => {
      const results = await ApiHandler.handleBatchRequests([Promise.resolve(1), Promise.resolve(2)]);
      expect(results).toEqual([1, 2]);
    });

    it('should show error on failure when not continueOnError', async () => {
      await expect(ApiHandler.handleBatchRequests([Promise.reject(new Error('fail'))])).rejects.toThrow();
      expect(message.error).toHaveBeenCalledWith('批量操作失败');
    });

    it('should use custom error message', async () => {
      await expect(ApiHandler.handleBatchRequests([Promise.reject(new Error('x'))], { errorMessage: 'Custom' })).rejects.toThrow();
      expect(message.error).toHaveBeenCalledWith('Custom');
    });

    it('should continue on error when configured', async () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
      const results = await ApiHandler.handleBatchRequests(
        [Promise.resolve(1), Promise.reject(new Error('fail')), Promise.resolve(3)],
        { continueOnError: true }
      );
      expect(results).toEqual([1, 3]);
      expect(message.warning).toHaveBeenCalled();
      consoleSpy.mockRestore();
    });

    it('should not show warning when no failures in continueOnError mode', async () => {
      const results = await ApiHandler.handleBatchRequests(
        [Promise.resolve(1), Promise.resolve(2)],
        { continueOnError: true }
      );
      expect(results).toEqual([1, 2]);
      expect(message.warning).not.toHaveBeenCalled();
    });
  });

  describe('ApiHandler.retryRequest', () => {
    it('should return result on first success', async () => {
      const fn = jest.fn().mockResolvedValue('ok');
      const result = await ApiHandler.retryRequest(fn, { maxRetries: 3, delay: 10 });
      expect(result).toBe('ok');
      expect(fn).toHaveBeenCalledTimes(1);
    });

    it('should retry on failure and succeed', async () => {
      const fn = jest.fn()
        .mockRejectedValueOnce(new Error('fail1'))
        .mockResolvedValueOnce('ok');
      const result = await ApiHandler.retryRequest(fn, { maxRetries: 3, delay: 10 });
      expect(result).toBe('ok');
      expect(fn).toHaveBeenCalledTimes(2);
    });

    it('should throw after max retries', async () => {
      const fn = jest.fn().mockRejectedValue(new Error('always fail'));
      await expect(ApiHandler.retryRequest(fn, { maxRetries: 2, delay: 10 })).rejects.toThrow('always fail');
      expect(fn).toHaveBeenCalledTimes(2);
    });

    it('should use linear delay when backoff is false', async () => {
      const fn = jest.fn()
        .mockRejectedValueOnce(new Error('fail'))
        .mockResolvedValueOnce('ok');
      await ApiHandler.retryRequest(fn, { maxRetries: 3, delay: 10, backoff: false });
      expect(fn).toHaveBeenCalledTimes(2);
    });
  });

  describe('exported convenience functions', () => {
    it('handleApiRequest should be ApiHandler.handleRequest', () => {
      expect(handleApiRequest).toBe(ApiHandler.handleRequest);
    });

    it('handleBatchApiRequests should be ApiHandler.handleBatchRequests', () => {
      expect(handleBatchApiRequests).toBe(ApiHandler.handleBatchRequests);
    });

    it('retryApiRequest should be ApiHandler.retryRequest', () => {
      expect(retryApiRequest).toBe(ApiHandler.retryRequest);
    });
  });
});
