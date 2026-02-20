/**
 * 环境配置测试
 */

import { getEnvironment, env, logger, performance, errorHandler, devTools } from '@/lib/env';

describe('Environment Utilities', () => {
  describe('getEnvironment', () => {
    it('should be defined', () => {
      expect(getEnvironment).toBeDefined();
    });

    it('should be a function', () => {
      expect(typeof getEnvironment).toBe('function');
    });

    it('should return a valid environment', () => {
      const result = getEnvironment();
      expect(['development', 'production', 'test']).toContain(result);
    });
  });

  describe('env', () => {
    it('should be defined', () => {
      expect(env).toBeDefined();
    });

    it('should have isDevelopment property', () => {
      expect(env.isDevelopment).toBeDefined();
    });

    it('should have isProduction property', () => {
      expect(env.isProduction).toBeDefined();
    });

    it('should have isTest property', () => {
      expect(env.isTest).toBeDefined();
    });

    it('should have features object', () => {
      expect(env.features).toBeDefined();
    });

    it('should have api object', () => {
      expect(env.api).toBeDefined();
      expect(env.api.baseUrl).toBeDefined();
      expect(env.api.timeout).toBeDefined();
    });

    it('should have app object', () => {
      expect(env.app).toBeDefined();
      expect(env.app.name).toBe('ITSM Platform');
    });
  });

  describe('logger', () => {
    it('should be defined', () => {
      expect(logger).toBeDefined();
    });

    it('should have debug function', () => {
      expect(typeof logger.debug).toBe('function');
    });

    it('should have info function', () => {
      expect(typeof logger.info).toBe('function');
    });

    it('should have warn function', () => {
      expect(typeof logger.warn).toBe('function');
    });

    it('should have error function', () => {
      expect(typeof logger.error).toBe('function');
    });

    it('should have performance function', () => {
      expect(typeof logger.performance).toBe('function');
    });

    it('should have security function', () => {
      expect(typeof logger.security).toBe('function');
    });
  });

  describe('performance', () => {
    it('should be defined', () => {
      expect(performance).toBeDefined();
    });

    it('should have start function', () => {
      expect(typeof performance.start).toBe('function');
    });

    it('should have end function', () => {
      expect(typeof performance.end).toBe('function');
    });

    it('should have measure function', () => {
      expect(typeof performance.measure).toBe('function');
    });

    it('should have measureAsync function', () => {
      expect(typeof performance.measureAsync).toBe('function');
    });
  });

  describe('errorHandler', () => {
    it('should be defined', () => {
      expect(errorHandler).toBeDefined();
    });

    it('should have handleApiError function', () => {
      expect(typeof errorHandler.handleApiError).toBe('function');
    });

    it('should have handleValidationError function', () => {
      expect(typeof errorHandler.handleValidationError).toBe('function');
    });

    it('should have handleNetworkError function', () => {
      expect(typeof errorHandler.handleNetworkError).toBe('function');
    });
  });

  describe('devTools', () => {
    it('should be defined', () => {
      expect(devTools).toBeDefined();
    });

    it('should have onlyInDev function', () => {
      expect(typeof devTools.onlyInDev).toBe('function');
    });

    it('should have debugInfo function', () => {
      expect(typeof devTools.debugInfo).toBe('function');
    });

    it('should have mark function', () => {
      expect(typeof devTools.mark).toBe('function');
    });

    it('should have measure function', () => {
      expect(typeof devTools.measure).toBe('function');
    });
  });
});
