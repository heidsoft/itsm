/**
 * error-message-handler - 错误消息处理工具测试
 */

import {
  getErrorMessage,
  formatApiError,
  isRetryableError,
  getErrorSeverity,
  logError,
  createUserFriendlyError,
  validateErrorMessage,
  parseErrorCode,
  getErrorMessageByCode,
  mapErrorToAction,
} from '../error-message-handler';

describe('error-message-handler', () => {
  describe('getErrorMessage', () => {
    it('应该从 Error 对象获取消息', () => {
      const error = new Error('Test error');
      expect(getErrorMessage(error)).toBe('Test error');
    });

    it('应该处理自定义错误', () => {
      const error = { message: 'Custom error', details: 'Additional info' };
      expect(getErrorMessage(error)).toBe('Custom error');
    });

    it('应该返回默认消息', () => {
      expect(getErrorMessage(null)).toBe('An unknown error occurred');
      expect(getErrorMessage(undefined)).toBe('An unknown error occurred');
    });
  });

  describe('formatApiError', () => {
    it('应该格式化 API 错误响应', () => {
      const apiError = {
        status: 404,
        data: { message: 'Not found', code: 'RESOURCE_NOT_FOUND' },
      };
      const result = formatApiError(apiError);
      expect(result).toContain('Not found');
      expect(result).toContain('404');
    });

    it('应该处理网络错误', () => {
      const networkError = { message: 'Network Error' };
      const result = formatApiError(networkError);
      expect(result).toContain('Network');
      expect(result).toContain('unable to connect');
    });

    it('应该处理无详细信息的错误', () => {
      const result = formatApiError({});
      expect(result).toBe('An error occurred');
    });
  });

  describe('isRetryableError', () => {
    it('应该识别可重试的错误', () => {
      expect(isRetryableError({ status: 502 })).toBe(true);
      expect(isRetryableError({ status: 503 })).toBe(true);
      expect(isRetryableError({ status: 504 })).toBe(true);
      expect(isRetryableError({ code: 'NETWORK_ERROR' })).toBe(true);
    });

    it('应该拒绝不可重试的错误', () => {
      expect(isRetryableError({ status: 400 })).toBe(false);
      expect(isRetryableError({ status: 401 })).toBe(false);
      expect(isRetryableError({ status: 403 })).toBe(false);
      expect(isRetryableError({ status: 404 })).toBe(false);
    });
  });

  describe('getErrorSeverity', () => {
    it('应该根据状态码返回严重程度', () => {
      expect(getErrorSeverity({ status: 500 })).toBe('critical');
      expect(getErrorSeverity({ status: 404 })).toBe('high');
      expect(getErrorSeverity({ status: 400 })).toBe('medium');
      expect(getErrorSeverity({ status: 200 })).toBe('low');
    });

    it('应该根据自定义代码返回严重程度', () => {
      expect(getErrorSeverity({ code: 'AUTH_FAILED' })).toBe('high');
      expect(getErrorSeverity({ code: 'VALIDATION_ERROR' })).toBe('medium');
    });
  });

  describe('createUserFriendlyError', () => {
    it('应该创建用户友好的错误消息', () => {
      const error = new Error('System failure');
      const result = createUserFriendlyError(error);
      expect(result).toContain('We encountered an issue');
      expect(result).not.toContain('System failure');
    });

    it('应该包含建议操作', () => {
      const error = { code: 'NETWORK_ERROR' };
      const result = createUserFriendlyError(error);
      expect(result).toContain('check your connection');
    });
  });

  describe('logError', () => {
    it('应该记录错误到控制台', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
      const error = new Error('Test error');

      logError(error, { context: 'test' });

      expect(consoleSpy).toHaveBeenCalled();
      consoleSpy.mockRestore();
    });
  });

  describe('validateErrorMessage', () => {
    it('应该验证错误消息格式', () => {
      const validError = { message: 'Error', code: 400 };
      expect(validateErrorMessage(validError)).toBe(true);
    });

    it('应该拒绝无效错误', () => {
      expect(validateErrorMessage(null)).toBe(false);
      expect(validateErrorMessage('string')).toBe(false);
      expect(validateErrorMessage({})).toBe(false);
    });
  });

  describe('parseErrorCode', () => {
    it('应该从错误中提取代码', () => {
      const error = { code: 'RESOURCE_NOT_FOUND' };
      expect(parseErrorCode(error)).toBe('RESOURCE_NOT_FOUND');
    });

    it('应该从状态码推断代码', () => {
      const error = { status: 404 };
      expect(parseErrorCode(error)).toBe('NOT_FOUND');
    });

    it('应该处理无代码错误', () => {
      expect(parseErrorCode({})).toBe('UNKNOWN_ERROR');
    });
  });

  describe('getErrorMessageByCode', () => {
    it('应该返回预定义消息', () => {
      const message = getErrorMessageByCode('NETWORK_ERROR');
      expect(message).toContain('network');
      expect(message).toContain('connect');
    });

    it('应该返回默认消息', () => {
      const message = getErrorMessageByCode('UNKNOWN_CODE');
      expect(message).toBe('An error occurred');
    });
  });

  describe('mapErrorToAction', () => {
    it('应该映射到重试操作', () => {
      const action = mapErrorToAction({ status: 503 });
      expect(action).toBe('retry');
    });

    it('应该映射到重新认证', () => {
      const action = mapErrorToAction({ status: 401 });
      expect(action).toBe('login');
    });

    it('应该映射到联系支持', () => {
      const action = mapErrorToAction({ status: 500 });
      expect(action).toBe('contact_support');
    });

    it('应该映射到无操作', () => {
      const action = mapErrorToAction({ status: 400 });
      expect(action).toBe('none');
    });
  });
});
