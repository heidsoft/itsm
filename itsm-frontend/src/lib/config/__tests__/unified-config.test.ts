/**
 * Tests for Unified Config
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

import {
  ENV_CONFIG,
  API_CONFIG,
  APP_CONFIG,
  DEV_CONFIG,
  PROD_CONFIG,
  ROUTE_CONFIG,
  PERMISSION_CONFIG,
  I18N_CONFIG,
  LOG_CONFIG,
  validateConfig,
  getCurrentConfig,
  initConfig,
  getConfig,
} from '../unified-config';

describe('Unified Config', () => {
  beforeEach(() => {
    jest.spyOn(console, 'error').mockImplementation(() => {});
    jest.spyOn(console, 'group').mockImplementation(() => {});
    jest.spyOn(console, 'groupEnd').mockImplementation(() => {});
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  describe('ENV_CONFIG', () => {
    it('should have NODE_ENV', () => {
      expect(ENV_CONFIG.NODE_ENV).toBeDefined();
    });

    it('should have API URL fields', () => {
      expect(ENV_CONFIG).toHaveProperty('NEXT_PUBLIC_API_URL');
      expect(ENV_CONFIG).toHaveProperty('ITSM_BACKEND_URL');
    });

    it('should have boolean flags', () => {
      expect(typeof ENV_CONFIG.NEXT_PUBLIC_ENABLE_ANALYTICS).toBe('boolean');
      expect(typeof ENV_CONFIG.NEXT_PUBLIC_ENABLE_DEBUG).toBe('boolean');
      expect(typeof ENV_CONFIG.NEXT_PUBLIC_ENABLE_MOCK).toBe('boolean');
    });
  });

  describe('API_CONFIG', () => {
    it('should have timeout', () => {
      expect(API_CONFIG.TIMEOUT).toBe(30000);
    });

    it('should have retry config', () => {
      expect(API_CONFIG.RETRY_ATTEMPTS).toBe(3);
      expect(API_CONFIG.RETRY_DELAY).toBe(1000);
    });

    it('should have endpoints', () => {
      expect(API_CONFIG.ENDPOINTS.AUTH).toBe('/auth');
      expect(API_CONFIG.ENDPOINTS.TICKETS).toBe('/tickets');
    });
  });

  describe('APP_CONFIG', () => {
    it('should have feature flags', () => {
      expect(APP_CONFIG.FEATURES.MULTI_TENANT).toBe(true);
      expect(APP_CONFIG.FEATURES.RBAC).toBe(true);
    });

    it('should have pagination config', () => {
      expect(APP_CONFIG.PAGINATION.DEFAULT_PAGE_SIZE).toBe(20);
      expect(APP_CONFIG.PAGINATION.MAX_PAGE_SIZE).toBe(100);
    });

    it('should have upload config', () => {
      expect(APP_CONFIG.UPLOAD.MAX_FILE_SIZE).toBe(50 * 1024 * 1024);
      expect(APP_CONFIG.UPLOAD.MAX_FILE_COUNT).toBe(10);
    });

    it('should have cache config', () => {
      expect(APP_CONFIG.CACHE.API_CACHE_TTL).toBe(5 * 60 * 1000);
    });

    it('should have storage keys', () => {
      expect(APP_CONFIG.STORAGE.TOKEN_KEY).toBe('itsm_token');
    });

    it('should have theme config', () => {
      expect(APP_CONFIG.THEME.DEFAULT_MODE).toBe('light');
      expect(APP_CONFIG.THEME.LAYOUT.SIDEBAR_WIDTH).toBe(240);
    });
  });

  describe('ROUTE_CONFIG', () => {
    it('should have public routes', () => {
      expect(ROUTE_CONFIG.PUBLIC_ROUTES).toContain('/login');
    });

    it('should have admin routes', () => {
      expect(ROUTE_CONFIG.ADMIN_ROUTES.length).toBeGreaterThan(0);
    });

    it('should have default routes', () => {
      expect(ROUTE_CONFIG.DEFAULT_ROUTE).toBe('/dashboard');
      expect(ROUTE_CONFIG.LOGIN_ROUTE).toBe('/login');
    });
  });

  describe('PERMISSION_CONFIG', () => {
    it('should have role definitions', () => {
      expect(PERMISSION_CONFIG.ROLES.SUPER_ADMIN).toBe('super_admin');
      expect(PERMISSION_CONFIG.ROLES.USER).toBe('user');
    });

    it('should have permission definitions', () => {
      expect(PERMISSION_CONFIG.PERMISSIONS.TICKET_CREATE).toBe('ticket:create');
    });

    it('should have role permissions mapping', () => {
      expect(PERMISSION_CONFIG.ROLE_PERMISSIONS.SUPER_ADMIN).toContain('*');
    });
  });

  describe('I18N_CONFIG', () => {
    it('should have default locale', () => {
      expect(I18N_CONFIG.DEFAULT_LOCALE).toBe('zh-CN');
    });

    it('should have supported locales', () => {
      expect(I18N_CONFIG.SUPPORTED_LOCALES).toContain('zh-CN');
      expect(I18N_CONFIG.SUPPORTED_LOCALES).toContain('en-US');
    });

    it('should have datetime config', () => {
      expect(I18N_CONFIG.DATETIME.DATE_FORMAT).toBe('YYYY-MM-DD');
    });
  });

  describe('LOG_CONFIG', () => {
    it('should have level map', () => {
      expect(LOG_CONFIG.LEVEL_MAP.DEBUG).toBe(0);
      expect(LOG_CONFIG.LEVEL_MAP.ERROR).toBe(3);
    });

    it('should have colors', () => {
      expect(LOG_CONFIG.COLORS.ERROR).toBe('#FF4D4F');
    });
  });

  describe('validateConfig', () => {
    it('should return validation result', () => {
      const result = validateConfig();
      expect(result).toHaveProperty('isValid');
      expect(result).toHaveProperty('errors');
    });
  });

  describe('getCurrentConfig', () => {
    it('should return config object', () => {
      const config = getCurrentConfig();
      expect(config.ENV_CONFIG).toBeDefined();
      expect(config.API_CONFIG).toBeDefined();
      expect(config.APP_CONFIG).toBeDefined();
      expect(config.isTest).toBe(true);
    });
  });

  describe('initConfig', () => {
    it('should return current config', () => {
      const config = initConfig();
      expect(config).toBeDefined();
      expect(config.ENV_CONFIG).toBeDefined();
    });
  });

  describe('getConfig', () => {
    it('should return full config when no path', () => {
      const config = getConfig();
      expect(config).toBeDefined();
    });

    it('should return nested value by path', () => {
      const result = getConfig('API_CONFIG.TIMEOUT');
      expect(result).toBe(30000);
    });

    it('should return undefined for invalid path', () => {
      const result = getConfig('NONEXISTENT.PATH');
      expect(result).toBeUndefined();
    });
  });
});
