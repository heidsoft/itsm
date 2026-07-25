/**
 * Tests for env.ts
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

import { getEnvironment, env, logger, performance as perfUtil, errorHandler, devTools } from '../env';

describe('Environment Utilities', () => {
  beforeEach(() => {
    jest.spyOn(console, 'error').mockImplementation(() => {});
    jest.spyOn(console, 'warn').mockImplementation(() => {});
    jest.spyOn(console, 'info').mockImplementation(() => {});
    jest.spyOn(console, 'time').mockImplementation(() => {});
    jest.spyOn(console, 'timeEnd').mockImplementation(() => {});
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  describe('getEnvironment', () => {
    it('should return current environment', () => {
      const result = getEnvironment();
      expect(['development', 'production', 'test']).toContain(result);
    });
  });

  describe('env', () => {
    it('should have environment flags', () => {
      expect(typeof env.isDevelopment).toBe('boolean');
      expect(typeof env.isProduction).toBe('boolean');
      expect(typeof env.isTest).toBe('boolean');
    });

    it('should have feature flags', () => {
      expect(typeof env.features.debugMode).toBe('boolean');
      expect(typeof env.features.performanceMonitoring).toBe('boolean');
      expect(typeof env.features.errorReporting).toBe('boolean');
    });

    it('should have API config', () => {
      expect(env.api.timeout).toBeGreaterThan(0);
      expect(env.api.retryCount).toBeGreaterThan(0);
    });

    it('should have app config', () => {
      expect(env.app.name).toBe('AI-Native ITSM');
      expect(env.app.version).toBeDefined();
    });
  });

  describe('logger', () => {
    it('should have debug method', () => {
      expect(() => logger.debug('test')).not.toThrow();
    });

    it('should have info method', () => {
      expect(() => logger.info('test')).not.toThrow();
    });

    it('should have warn method', () => {
      expect(() => logger.warn('test')).not.toThrow();
    });

    it('should have error method that always logs', () => {
      logger.error('test error');
      expect(console.error).toHaveBeenCalledWith('[ERROR]', 'test error');
    });

    it('should have performance method', () => {
      expect(() => logger.performance('test', { duration: '100ms' })).not.toThrow();
    });

    it('should have security method', () => {
      expect(() => logger.security('event', { user: 'test' })).not.toThrow();
    });
  });

  describe('performance utilities', () => {
    it('should have start method', () => {
      const result = perfUtil.start('test');
      // In test env performanceMonitoring is false
      expect(result === null || result === 'test').toBe(true);
    });

    it('should have end method', () => {
      expect(() => perfUtil.end('test')).not.toThrow();
    });

    it('should measure function execution', () => {
      const result = perfUtil.measure('test', () => 42);
      expect(result).toBe(42);
    });

    it('should measure async function execution', async () => {
      const result = await perfUtil.measureAsync('test', async () => 'async result');
      expect(result).toBe('async result');
    });
  });

  describe('errorHandler', () => {
    it('should handle API errors', () => {
      const result = errorHandler.handleApiError(new Error('API failed'), 'test');
      expect(result.success).toBe(false);
      expect(result.error).toBe('API failed');
    });

    it('should handle non-Error API errors', () => {
      const result = errorHandler.handleApiError('string error', 'ctx');
      expect(result.success).toBe(false);
      expect(result.error).toBe('string error');
    });

    it('should handle validation errors', () => {
      const result = errorHandler.handleValidationError({ name: ['required'] });
      expect(result.success).toBe(false);
      expect(result.message).toBe('表单验证失败');
    });

    it('should handle network errors', () => {
      const result = errorHandler.handleNetworkError(new Error('Network Error'));
      expect(result.success).toBe(false);
      expect(result.type).toBe('network');
    });

    it('should handle Failed to fetch errors', () => {
      const result = errorHandler.handleNetworkError(new Error('Failed to fetch'));
      expect(result.type).toBe('network');
    });

    it('should handle non-network errors', () => {
      const result = errorHandler.handleNetworkError(new Error('Some other error'));
      expect(result.type).toBe('unknown');
    });
  });

  describe('devTools', () => {
    it('should execute onlyInDev based on environment', () => {
      const fn = jest.fn().mockReturnValue('result');
      const result = devTools.onlyInDev(fn, 'fallback');
      // In test env, isDevelopment is false
      if (env.isDevelopment) {
        expect(result).toBe('result');
      } else {
        expect(result).toBe('fallback');
      }
    });

    it('should call debugInfo without throwing', () => {
      expect(() => devTools.debugInfo('Component', { data: 'test' })).not.toThrow();
    });

    it('should call mark without throwing', () => {
      expect(() => devTools.mark('test-mark')).not.toThrow();
    });

    it('should call measure without throwing', () => {
      expect(() => devTools.measure('test', 'start', 'end')).not.toThrow();
    });
  });
});
