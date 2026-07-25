/**
 * Tests for Error Handler
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

// Mock antd
jest.mock('antd', () => ({
  message: { error: jest.fn(), warning: jest.fn(), success: jest.fn(), info: jest.fn() },
  notification: { error: jest.fn(), success: jest.fn(), warning: jest.fn(), info: jest.fn() },
}));

// Mock CSS module
jest.mock('../error-handler.module.css', () => ({
  errorBoundary: 'errorBoundary',
  errorBoundaryContent: 'errorBoundaryContent',
  errorBoundaryTitle: 'errorBoundaryTitle',
  errorBoundaryMessage: 'errorBoundaryMessage',
  retryButton: 'retryButton',
}));

import { ErrorHandler, ErrorType, ErrorSeverity, handleError, handleBusinessError, handleValidationError, ErrorBoundary } from '../error-handler';
import { message, notification } from 'antd';
import React from 'react';

describe('ErrorHandler', () => {
  let handler: ErrorHandler;

  beforeEach(() => {
    jest.spyOn(console, 'error').mockImplementation(() => {});
    jest.spyOn(console, 'warn').mockImplementation(() => {});
    // Reset singleton for testing
    (ErrorHandler as any).instance = undefined;
    handler = ErrorHandler.getInstance();
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  describe('getInstance', () => {
    it('should return singleton instance', () => {
      const instance1 = ErrorHandler.getInstance();
      const instance2 = ErrorHandler.getInstance();
      expect(instance1).toBe(instance2);
    });
  });

  describe('handleApiError', () => {
    it('should handle API error with status', () => {
      const error = { status: 404, message: 'Not found' };
      const result = handler.handleApiError(error, 'test');
      expect(result.code).toBe(404);
      expect(result.message).toBe('Not found');
    });

    it('should handle Error instance', () => {
      const error = new Error('Network failure');
      const result = handler.handleApiError(error, 'test');
      expect(result.message).toBe('Network failure');
    });

    it('should handle unknown errors', () => {
      const result = handler.handleApiError('string error', 'test');
      expect(result.message).toBe('未知错误');
    });

    it('should handle 401 as authentication error', () => {
      const error = { status: 401, message: 'Unauthorized' };
      const result = handler.handleApiError(error);
      expect(result.code).toBe(401);
    });

    it('should handle 403 as authorization error', () => {
      const error = { status: 403, message: 'Forbidden' };
      const result = handler.handleApiError(error);
      expect(result.code).toBe(403);
    });

    it('should handle 500 as server error', () => {
      const error = { status: 500, message: 'Server Error' };
      handler.handleApiError(error);
      expect(notification.error).toHaveBeenCalled();
    });
  });

  describe('handleBusinessError', () => {
    it('should handle business errors', () => {
      const error = new Error('Business logic failed');
      handler.handleBusinessError(error, 'context');
      expect(message.error).toHaveBeenCalledWith('Business logic failed');
    });
  });

  describe('handleValidationError', () => {
    it('should show validation errors', () => {
      const errors = { name: ['Name is required'], email: ['Invalid email'] };
      handler.handleValidationError(errors);
      expect(message.error).toHaveBeenCalledTimes(2);
    });
  });

  describe('getErrorLog', () => {
    it('should return error log', () => {
      handler.handleApiError(new Error('test'), 'test');
      const log = handler.getErrorLog();
      expect(log.length).toBeGreaterThan(0);
      expect(log[0].message).toBe('test');
    });
  });

  describe('clearErrorLog', () => {
    it('should clear the log', () => {
      handler.handleApiError(new Error('test'), 'test');
      handler.clearErrorLog();
      expect(handler.getErrorLog()).toHaveLength(0);
    });
  });

  describe('getErrorStats', () => {
    it('should return stats by error type', () => {
      handler.handleApiError(new Error('test1'));
      handler.handleApiError({ status: 404, message: 'not found' });
      const stats = handler.getErrorStats();
      // Error instance maps to NETWORK, 404 maps to NOT_FOUND
      const totalErrors = Object.values(stats).reduce((a, b) => a + b, 0);
      expect(totalErrors).toBeGreaterThanOrEqual(2);
    });
  });

  describe('convenience functions', () => {
    it('handleError should work', () => {
      const result = handleError(new Error('test'));
      expect(result).toHaveProperty('message');
    });

    it('handleBusinessError should work', () => {
      handleBusinessError(new Error('business'));
      expect(message.error).toHaveBeenCalled();
    });

    it('handleValidationError should work', () => {
      handleValidationError({ field: ['error'] });
      expect(message.error).toHaveBeenCalled();
    });
  });

  describe('ErrorType enum', () => {
    it('should have correct values', () => {
      expect(ErrorType.NETWORK).toBe('NETWORK');
      expect(ErrorType.AUTHENTICATION).toBe('AUTHENTICATION');
      expect(ErrorType.SERVER).toBe('SERVER');
    });
  });

  describe('ErrorSeverity enum', () => {
    it('should have correct values', () => {
      expect(ErrorSeverity.LOW).toBe('LOW');
      expect(ErrorSeverity.CRITICAL).toBe('CRITICAL');
    });
  });
});
