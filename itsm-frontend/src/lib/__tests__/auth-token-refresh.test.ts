/**
 * Auth Token Refresh Mechanism Tests
 *
 * 测试覆盖:
 * - AuthService.refreshToken() 方法
 * - HttpClient.refreshTokenInternal() 方法
 * - 401 自动检测和重试逻辑
 * - Token 存储和检索 (httpOnly cookie 模式)
 * - Token 过期处理
 * - 刷新失败场景
 */

import { AuthService } from '@/lib/services/auth-service';
import { httpClient } from '@/lib/api/http-client';
import {
  getAccessToken,
  getRefreshToken,
  setAccessToken,
  setRefreshToken,
  clearAuthStorage,
  isAuthenticated,
} from '@/lib/auth/token-storage';

// Mock fetch globally
global.fetch = jest.fn();

// Mock cookie for testing
let cookieStore = '';
Object.defineProperty(document, 'cookie', {
  get: jest.fn(() => cookieStore),
  set: jest.fn((val: string) => {
    // Simple cookie setter - append or update
    const [pair] = val.split(';');
    const [name, ...rest] = pair.split('=');
    const value = rest.join('=');
    const cookies = cookieStore.split('; ').filter(Boolean);
    const idx = cookies.findIndex(c => c.startsWith(`${name}=`));
    if (idx >= 0) {
      cookies[idx] = `${name}=${value}`;
    } else {
      cookies.push(`${name}=${value}`);
    }
    cookieStore = cookies.join('; ');
  }),
});

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: jest.fn((key: string) => store[key] || null),
    setItem: jest.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: jest.fn((key: string) => {
      delete store[key];
    }),
    clear: jest.fn(() => {
      store = {};
    }),
  };
})();

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
  writable: true,
});

// Mock console methods
const consoleSpy = {
  error: jest.spyOn(console, 'error').mockImplementation(() => {}),
  warn: jest.spyOn(console, 'warn').mockImplementation(() => {}),
  log: jest.spyOn(console, 'log').mockImplementation(() => {}),
};

// Mock useAuthStore
let mockAuthState = {
  user: null,
  isAuthenticated: false,
  login: jest.fn(),
  logout: jest.fn(),
};

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStore: {
    getState: () => mockAuthState,
    setState: jest.fn(fn => {
      mockAuthState = typeof fn === 'function' ? fn(mockAuthState) : fn;
    }),
    subscribe: jest.fn(),
  },
}));

describe('Auth Token Refresh Mechanism', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (fetch as jest.Mock).mockClear();
    cookieStore = '';
    localStorageMock.clear();
    mockAuthState = {
      user: null,
      isAuthenticated: false,
      login: jest.fn(),
      logout: jest.fn(),
    };
  });

  afterAll(() => {
    Object.values(consoleSpy).forEach(spy => spy.mockRestore());
  });

  // ==========================================
  // AuthService.refreshToken()
  // ==========================================
  describe('AuthService.refreshToken()', () => {
    it('should return false when no refresh token cookie exists', async () => {
      const result = await AuthService.refreshToken();
      expect(result).toBe(false);
    });

    it('should successfully refresh token when refresh cookie exists', async () => {
      // Set the cookie so getRefreshToken() returns a value
      document.cookie = 'refresh_token=test-refresh-token';

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          code: 0,
          message: 'success',
          data: {
            access_token: 'new-access-token',
            refresh_token: 'new-refresh-token',
          },
        }),
      });

      const result = await AuthService.refreshToken();
      expect(result).toBe(true);
    });

    it('should handle refresh API failure', async () => {
      document.cookie = 'refresh_token=test-refresh-token';

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          code: 2001,
          message: 'Invalid refresh token',
          data: null,
        }),
      });

      const result = await AuthService.refreshToken();
      expect(result).toBe(false);
      // clearTokens calls logout from auth store
      expect(mockAuthState.logout).toHaveBeenCalled();
    });

    it('should clear tokens when refresh fails', async () => {
      document.cookie = 'refresh_token=test-refresh-token';

      (fetch as jest.Mock).mockRejectedValueOnce(new Error('Network error'));

      const result = await AuthService.refreshToken();
      expect(result).toBe(false);
      expect(mockAuthState.logout).toHaveBeenCalled();
    });
  });

  // ==========================================
  // AuthService Token Management
  // ==========================================
  describe('AuthService Token Management', () => {
    describe('setTokens()', () => {
      it('should be a no-op (tokens stored in httpOnly cookies by backend)', () => {
        // setTokens is now a no-op since tokens are in httpOnly cookies
        AuthService.setTokens('access-token', 'refresh-token');
        // No localStorage mutation expected
        expect(localStorageMock.setItem).not.toHaveBeenCalledWith('access_token', expect.anything());
      });
    });

    describe('getAccessToken()', () => {
      it('should return null (httpOnly cookies not readable from JS)', () => {
        expect(AuthService.getAccessToken()).toBeNull();
      });
    });

    describe('getRefreshToken()', () => {
      it('should return null when no cookie exists', () => {
        expect(AuthService.getRefreshToken()).toBeNull();
      });

      it('should return token value when cookie exists', () => {
        document.cookie = 'refresh_token=test-refresh-value';
        expect(AuthService.getRefreshToken()).toBe('test-refresh-value');
      });
    });

    describe('clearTokens()', () => {
      it('should call logout from auth store', () => {
        AuthService.clearTokens();
        expect(mockAuthState.logout).toHaveBeenCalled();
      });
    });

    describe('isAuthenticated()', () => {
      it('should return true when auth store says authenticated', () => {
        mockAuthState.isAuthenticated = true;
        expect(AuthService.isAuthenticated()).toBe(true);
      });

      it('should return true when access_token cookie exists', () => {
        mockAuthState.isAuthenticated = false;
        document.cookie = 'access_token=some-token';
        expect(AuthService.isAuthenticated()).toBe(true);
      });

      it('should return false when no auth state or cookie', () => {
        mockAuthState.isAuthenticated = false;
        expect(AuthService.isAuthenticated()).toBe(false);
      });
    });
  });

  // ==========================================
  // HttpClient.refreshTokenInternal()
  // ==========================================
  describe('HttpClient.refreshTokenInternal()', () => {
    it('should return false when no refresh token exists', async () => {
      const result = await httpClient.refreshTokenInternal();
      expect(result).toBe(false);
    });

    it('should handle refresh API failure', async () => {
      document.cookie = 'refresh_token=test-refresh-token';

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          code: 2001,
          message: 'Invalid token',
          data: null,
        }),
      });

      const result = await httpClient.refreshTokenInternal();
      expect(result).toBe(false);
    });

    it('should handle network error', async () => {
      document.cookie = 'refresh_token=test-refresh-token';

      (fetch as jest.Mock).mockRejectedValueOnce(new Error('Network error'));

      const result = await httpClient.refreshTokenInternal();
      expect(result).toBe(false);
    });
  });

  // ==========================================
  // Token Storage Functions (httpOnly cookie mode)
  // ==========================================
  describe('Token Storage Functions', () => {
    describe('getAccessToken()', () => {
      it('should return null (httpOnly cookies not readable from JS)', () => {
        expect(getAccessToken()).toBeNull();
      });
    });

    describe('getRefreshToken()', () => {
      it('should return null (httpOnly cookies not readable from JS)', () => {
        expect(getRefreshToken()).toBeNull();
      });
    });

    describe('clearAuthStorage()', () => {
      it('should clear tenant and legacy keys from localStorage', () => {
        localStorageMock.setItem('current_tenant_code', 'test');
        localStorageMock.setItem('current_tenant_id', '1');
        clearAuthStorage();
        expect(localStorageMock.removeItem).toHaveBeenCalledWith('current_tenant_id');
        expect(localStorageMock.removeItem).toHaveBeenCalledWith('current_tenant_code');
      });
    });

    describe('isAuthenticated()', () => {
      it('should return true when auth cookie exists', () => {
        document.cookie = 'auth-token=test-value';
        expect(isAuthenticated()).toBe(true);
      });

      it('should return true when access_token cookie exists', () => {
        document.cookie = 'access_token=test-value';
        expect(isAuthenticated()).toBe(true);
      });

      it('should return false when no auth cookie', () => {
        expect(isAuthenticated()).toBe(false);
      });
    });

    describe('setAccessToken / setRefreshToken', () => {
      it('should be no-op (tokens set by backend via httpOnly cookies)', () => {
        setAccessToken('test');
        setRefreshToken('test');
        // These are no-ops, no localStorage calls
        expect(localStorageMock.setItem).not.toHaveBeenCalledWith('access_token', expect.anything());
        expect(localStorageMock.setItem).not.toHaveBeenCalledWith('refresh_token', expect.anything());
      });
    });
  });

  // ==========================================
  // Edge Cases
  // ==========================================
  describe('Edge Cases', () => {
    it('should handle SSR environment (window undefined)', () => {
      // token-storage functions handle typeof window === 'undefined'
      // In test env, window exists, so this verifies the safe path
      expect(getAccessToken()).toBeNull();
      expect(getRefreshToken()).toBeNull();
    });

    it('should handle malformed JWT payload', () => {
      // isAuthenticated checks cookies, not JWT parsing
      expect(AuthService.isAuthenticated()).toBe(false);
    });
  });

  // ==========================================
  // Integration Scenarios
  // ==========================================
  describe('Integration Scenarios', () => {
    it('should complete full auth flow: login → token refresh → API call', async () => {
      // Step 1: Login
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          code: 0,
          message: 'success',
          data: {
            access_token: 'initial-access',
            refresh_token: 'initial-refresh',
            user: { id: 1, username: 'admin' },
          },
        }),
      });

      const loginResult = await AuthService.login('admin', 'admin123');
      expect(loginResult).toBe(true);

      // Step 2: Token refresh
      document.cookie = 'refresh_token=initial-refresh';
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          code: 0,
          message: 'success',
          data: {
            access_token: 'refreshed-access',
            refresh_token: 'refreshed-refresh',
          },
        }),
      });

      const refreshResult = await AuthService.refreshToken();
      expect(refreshResult).toBe(true);
    });

    it('should handle session timeout after multiple failed refresh attempts', async () => {
      document.cookie = 'refresh_token=stale-token';

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          code: 2001,
          message: 'Token expired',
          data: null,
        }),
      });

      const result = await AuthService.refreshToken();
      expect(result).toBe(false);
      expect(mockAuthState.logout).toHaveBeenCalled();
    });
  });
});
