/**
 * HTTP Client 测试
 */

// 测试前需要模拟环境
const mockCookie = jest.fn();
const mockLocalStorage = jest.fn();

Object.defineProperty(document, 'cookie', {
  get: mockCookie,
  set: mockCookie,
});

Object.defineProperty(global, 'localStorage', {
  value: {
    getItem: mockLocalStorage,
    setItem: mockLocalStorage,
    removeItem: mockLocalStorage,
  },
});

// Mock security module
jest.mock('@/lib/security', () => ({
  security: {
    csrf: {
      getToken: jest.fn().mockResolvedValue('mock-csrf-token'),
    },
    network: {
      getSecureHeaders: jest.fn().mockReturnValue({
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest',
      }),
    },
  },
}));

// Mock env module
jest.mock('@/lib/env', () => ({
  logger: {
    debug: jest.fn(),
    info: jest.fn(),
    error: jest.fn(),
    warn: jest.fn(),
  },
}));

// Mock tenant-context module
jest.mock('@/lib/auth/tenant-context', () => ({
  getTenantId: jest.fn().mockReturnValue(null),
  getTenantCode: jest.fn().mockReturnValue(null),
  setTenantId: jest.fn(),
  setTenantCode: jest.fn(),
  clearTenantContext: jest.fn(),
}));

// Mock auth/token-storage
jest.mock('@/lib/auth/token-storage', () => ({
  getAccessToken: jest.fn().mockReturnValue(null),
  getRefreshToken: jest.fn().mockReturnValue(null),
  setAccessToken: jest.fn(),
  setRefreshToken: jest.fn(),
  clearAuthStorage: jest.fn(),
  isAuthenticated: jest.fn().mockReturnValue(false),
  STORAGE_KEYS: {},
}));

// 延迟导入以确保 mock 生效
import HttpClient from '../api/http-client';

describe('HttpClient', () => {
  let client: HttpClient;

  beforeEach(() => {
    jest.clearAllMocks();
    mockCookie.mockReturnValue('');
    mockLocalStorage.mockReturnValue(null);
    client = new HttpClient('http://localhost:8080');
  });

  describe('constructor', () => {
    it('should create client with default base URL', () => {
      const defaultClient = new HttpClient();
      expect(defaultClient.getBaseURL()).toBeDefined();
    });

    it('should create client with custom base URL', () => {
      const customClient = new HttpClient('http://custom-api:9000');
      expect(customClient.getBaseURL()).toBe('http://custom-api:9000');
    });
  });

  describe('token management', () => {
    it('should set and get token', () => {
      client.setToken('test-token');
      expect(client.getAuthToken()).toBe('test-token');
    });

    it('should clear token', () => {
      client.setToken('test-token');
      client.clearToken();
      expect(client.getAuthToken()).toBeNull();
    });
  });

  describe('tenant management', () => {
    it('should set and get tenant ID (deprecated — delegates to TenantContext)', () => {
      // setTenantId is now a no-op (deprecated); getTenantId reads from TenantContext
      client.setTenantId(123);
      expect(client.getTenantId()).toBeNull(); // TenantContext not initialized in test
    });

    it('should set and get tenant code (deprecated — delegates to TenantContext)', () => {
      client.setTenantCode('acme');
      expect(client.getTenantCode()).toBeNull(); // TenantContext not initialized in test
    });

    it('should clear tenant ID (deprecated — no-op)', () => {
      client.setTenantId(123);
      client.setTenantId(null);
      expect(client.getTenantId()).toBeNull();
    });
  });

  describe('getAuthToken', () => {
    it('should return null when no token', () => {
      expect(client.getAuthToken()).toBeNull();
    });

    it('should return token after setting', () => {
      client.setToken('abc123');
      expect(client.getAuthToken()).toBe('abc123');
    });
  });

  describe('getBaseURL', () => {
    it('should return base URL', () => {
      expect(client.getBaseURL()).toBe('http://localhost:8080');
    });
  });

  describe('getToken (backward compatibility)', () => {
    it('should return same as getAuthToken', () => {
      client.setToken('compat-token');
      expect(client.getToken()).toBe(client.getAuthToken());
    });
  });
});

describe('HttpClient methods', () => {
  let client: HttpClient;

  beforeEach(() => {
    jest.clearAllMocks();
    mockCookie.mockReturnValue('');
    mockLocalStorage.mockReturnValue(null);
    client = new HttpClient('http://localhost:8080');
  });

  describe('token from cookie (via setToken)', () => {
    it('should accept token via setToken', () => {
      client.setToken('my-token');
      expect(client.getAuthToken()).toBe('my-token');
    });
  });

  describe('clearToken', () => {
    it('should clear token', () => {
      client.setToken('temp-token');
      client.clearToken();
      expect(client.getAuthToken()).toBeNull();
    });
  });
});
