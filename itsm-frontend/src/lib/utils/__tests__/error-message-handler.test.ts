import {
  ErrorMessageHandler,
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
  getFriendlyErrorMessage,
} from '../error-message-handler';

describe('ErrorMessageHandler.formatApiError', () => {
  it('formats Error instances', () => {
    const err = new Error('boom');
    const info = ErrorMessageHandler.formatApiError(err);
    expect(info.message).toBe('boom');
    expect(info.details).toBeDefined();
  });

  it('formats string errors', () => {
    expect(ErrorMessageHandler.formatApiError('oops').message).toBe('oops');
  });

  it('formats plain objects', () => {
    const info = ErrorMessageHandler.formatApiError({ code: 404, message: 'not found' });
    expect(info.code).toBe(404);
    expect(info.message).toBe('not found');
  });

  it('handles unknown types', () => {
    expect(ErrorMessageHandler.formatApiError(123).message).toBe('未知错误，请稍后重试');
  });
});

describe('ErrorMessageHandler.formatNetworkError', () => {
  it('detects network keyword', () => {
    const info = ErrorMessageHandler.formatNetworkError(new Error('network failure'));
    expect(info.message).toContain('网络连接');
    expect(info.actionText).toBe('重试');
  });

  it('detects timeout keyword', () => {
    const info = ErrorMessageHandler.formatNetworkError(new Error('request timeout'));
    expect(info.message).toContain('超时');
  });
});

describe('getErrorMessage', () => {
  it('extracts message from Error', () => {
    expect(getErrorMessage(new Error('test'))).toBe('test');
  });
  it('returns string directly', () => {
    expect(getErrorMessage('hello')).toBe('hello');
  });
  it('returns fallback for unknown', () => {
    expect(getErrorMessage(42)).toBe('An unknown error occurred');
  });
});

describe('isRetryableError', () => {
  it('returns true for 503 status', () => {
    expect(isRetryableError({ status: 503 })).toBe(true);
  });
  it('returns true for TIMEOUT code', () => {
    expect(isRetryableError({ code: 'TIMEOUT' })).toBe(true);
  });
  it('returns false for non-object', () => {
    expect(isRetryableError('err')).toBe(false);
  });
  it('returns false for 404', () => {
    expect(isRetryableError({ status: 404 })).toBe(false);
  });
});

describe('getErrorSeverity', () => {
  it('returns critical for 500+', () => {
    expect(getErrorSeverity({ status: 500 })).toBe('critical');
  });
  it('returns high for 400-499', () => {
    expect(getErrorSeverity({ status: 403 })).toBe('high');
  });
  it('returns high for AUTH code', () => {
    expect(getErrorSeverity({ code: 'AUTH_FAILED' })).toBe('high');
  });
  it('returns medium for non-object', () => {
    expect(getErrorSeverity(null)).toBe('medium');
  });
});

describe('parseErrorCode', () => {
  it('uses code field when present', () => {
    expect(parseErrorCode({ code: 'CUSTOM' })).toBe('CUSTOM');
  });
  it('maps status 401 to UNAUTHORIZED', () => {
    expect(parseErrorCode({ status: 401 })).toBe('UNAUTHORIZED');
  });
  it('returns UNKNOWN_ERROR for primitives', () => {
    expect(parseErrorCode('x')).toBe('UNKNOWN_ERROR');
  });
});

describe('getErrorMessageByCode', () => {
  it('returns known message', () => {
    expect(getErrorMessageByCode('NOT_FOUND')).toBe('请求的资源不存在');
  });
  it('returns default for unknown code', () => {
    expect(getErrorMessageByCode('SOMETHING')).toBe('发生了一个错误');
  });
});

describe('logError', () => {
  it('calls console.error', () => {
    const spy = jest.spyOn(console, 'error').mockImplementation(() => {});
    logError(new Error('test'), { page: 'home' });
    expect(spy).toHaveBeenCalled();
    spy.mockRestore();
  });
});

describe('mapErrorToAction', () => {
  it('maps UNAUTHORIZED to login', () => {
    expect(mapErrorToAction({ status: 401 })).toBe('login');
  });
  it('maps TIMEOUT code to retry', () => {
    expect(mapErrorToAction({ code: 'TIMEOUT' })).toBe('retry');
  });
});

describe('validateErrorMessage', () => {
  it('returns false for null', () => expect(validateErrorMessage(null)).toBe(false));
  it('returns true for non-empty string', () => expect(validateErrorMessage('x')).toBe(true));
  it('returns false for empty string', () => expect(validateErrorMessage('')).toBe(false));
});
