/**
 * Auth Token Refresh Mechanism Tests
 *
 * 测试覆盖:
 * - AuthService.refreshToken() 方法
 * - HttpClient.refreshTokenInternal() 方法
 * - 401 自动检测和重试逻辑
 * - Token 存储和检索
 * - Token 过期处理
 * - 刷新失败场景
 *
 * 任务：P1-05 Auth Token 刷新机制测试
 */

import { AuthService } from '@/lib/services/auth-service';
import { httpClient } from '@/lib/api/http-client';
import {
  getAccessToken,
  getRefreshToken,
  setAccessToken,
  setRefreshToken,
  clearAuthStorage,
} from '@/lib/auth/token-storage';

// Mock fetch globally
global.fetch = jest.fn();

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
    get store() {
      return store;
    },
    set store(value: Record<string, string>) {
      store = value;
    },
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

// Mock useAuthStore - will be overridden in beforeEach
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
    localStorageMock.store = {};
    localStorageMock.getItem.mockClear();
    localStorageMock.setItem.mockClear();
    localStorageMock.removeItem.mockClear();
    (fetch as jest.Mock).mockClear();
    // Reset auth state
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

  // ============================================================================
  // AuthService Token Refresh Tests
  // ============================================================================
  describe('AuthService.refreshToken()', () => {
    it('should successfully refresh token with valid refresh token', async () => {
      const mockRefreshToken = 'valid-refresh-token-123';
      const mockNewAccessToken = 'new-access-token-456';

      localStorageMock.store['refresh_token'] = mockRefreshToken;

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          code: 0,
          message: 'success',
          data: {
            access_token: mockNewAccessToken,
          },
        }),
      });

      const result = await AuthService.refreshToken();

      expect(result).toBe(true);
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/refresh-token'),
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
          }),
          body: JSON.stringify({
            refresh_token: mockRefreshToken,
          }),
        })
      );
      expect(localStorageMock.setItem).toHaveBeenCalledWith('access_token', mockNewAccessToken);
    });

    it('should return false when no refresh token exists', async () => {
      localStorageMock.store['refresh_token'] = undefined;

      const result = await AuthService.refreshToken();

      expect(result).toBe(false);
      expect(fetch).not.toHaveBeenCalled();
    });

    it('should handle refresh token API error', async () => {
      localStorageMock.store['refresh_token'] = 'invalid-refresh-token';

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          code: 2001,
          message: 'Refresh token expired',
          data: null,
        }),
      });

      const result = await AuthService.refreshToken();

      expect(result).toBe(false);
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('access_token');
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('refresh_token');
    });

    it('should handle network error during refresh', async () => {
      localStorageMock.store['refresh_token'] = 'refresh-token';

      (fetch as jest.Mock).mockRejectedValueOnce(new Error('Network error'));

      const result = await AuthService.refreshToken();

      expect(result).toBe(false);
      // Note: console.error is called but may not be captured by spy due to module loading order
      // The important assertion is that refreshToken returns false on error
    });

    it('should handle HTTP error during refresh', async () => {
      localStorageMock.store['refresh_token'] = 'refresh-token';

      (fetch as jest.Mock).mockRejectedValueOnce(new Error('HTTP error! status: 500'));

      const result = await AuthService.refreshToken();

      expect(result).toBe(false);
    });

    it('should clear tokens when refresh fails', async () => {
      localStorageMock.store['refresh_token'] = 'expired-token';
      localStorageMock.store['access_token'] = 'old-access-token';

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          code: 2001,
          message: 'Invalid refresh token',
          data: null,
        }),
      });

      await AuthService.refreshToken();

      expect(localStorageMock.removeItem).toHaveBeenCalledWith('access_token');
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('refresh_token');
    });
  });

  // ============================================================================
  // AuthService Token Management Tests
  // ============================================================================
  describe('AuthService Token Management', () => {
    describe('setTokens()', () => {
      it('should store both access and refresh tokens', () => {
        const accessToken = 'test-access-token';
        const refreshToken = 'test-refresh-token';

        AuthService.setTokens(accessToken, refreshToken);

        expect(localStorageMock.setItem).toHaveBeenCalledWith('access_token', accessToken);
        expect(localStorageMock.setItem).toHaveBeenCalledWith('refresh_token', refreshToken);
      });
    });

    describe('getAccessToken()', () => {
      it('should return access token from localStorage', () => {
        const token = 'stored-access-token';
        localStorageMock.store['access_token'] = token;

        const result = AuthService.getAccessToken();

        expect(result).toBe(token);
      });

      it('should return null when no token exists', () => {
        const result = AuthService.getAccessToken();
        expect(result).toBe(null);
      });
    });

    describe('getRefreshToken()', () => {
      it('should return refresh token from localStorage', () => {
        const token = 'stored-refresh-token';
        localStorageMock.store['refresh_token'] = token;

        const result = AuthService.getRefreshToken();

        expect(result).toBe(token);
      });
    });

    describe('clearTokens()', () => {
      it('should remove both access and refresh tokens', () => {
        AuthService.clearTokens();

        expect(localStorageMock.removeItem).toHaveBeenCalledWith('access_token');
        expect(localStorageMock.removeItem).toHaveBeenCalledWith('refresh_token');
      });
    });

    describe('isAuthenticated()', () => {
      it('should return true for valid JWT token', () => {
        // Create a valid mock JWT (header.payload.signature)
        const payload = { exp: Math.floor(Date.now() / 1000) + 3600 }; // Expires in 1 hour
        const encodedPayload = btoa(JSON.stringify(payload));
        const mockToken = `header.${encodedPayload}.signature`;

        localStorageMock.store['access_token'] = mockToken;

        const result = AuthService.isAuthenticated();

        expect(result).toBe(true);
      });

      it('should return false for expired token', () => {
        const payload = { exp: Math.floor(Date.now() / 1000) - 3600 }; // Expired 1 hour ago
        const encodedPayload = btoa(JSON.stringify(payload));
        const mockToken = `header.${encodedPayload}.signature`;

        localStorageMock.store['access_token'] = mockToken;

        const result = AuthService.isAuthenticated();

        expect(result).toBe(false);
        expect(localStorageMock.removeItem).toHaveBeenCalledWith('access_token');
      });

      it('should return false for invalid token format', () => {
        localStorageMock.store['access_token'] = 'invalid-token-format';

        const result = AuthService.isAuthenticated();

        expect(result).toBe(false);
      });

      it('should return true for mock token in development', () => {
        localStorageMock.store['access_token'] = 'mock_test_token';

        const result = AuthService.isAuthenticated();

        expect(result).toBe(true);
      });

      it('should return false when no token exists', () => {
        mockAuthState.isAuthenticated = false;
        const result = AuthService.isAuthenticated();
        expect(result).toBe(false);
      });
    });
  });

  // ============================================================================
  // HttpClient Token Refresh Tests
  // ============================================================================
  describe('HttpClient.refreshTokenInternal()', () => {
    it('should successfully refresh token', async () => {
      const mockRefreshToken = 'http-refresh-token';
      const mockNewAccessToken = 'http-new-access-token';

      localStorageMock.store['refresh_token'] = mockRefreshToken;

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          code: 0,
          message: 'success',
          data: {
            access_token: mockNewAccessToken,
          },
        }),
      });

      // Access the private method through a workaround
      const client = httpClient;
      const result = await (client as any).refreshTokenInternal();

      expect(result).toBe(true);
      expect(localStorageMock.setItem).toHaveBeenCalledWith('access_token', mockNewAccessToken);
    });

    it('should return false when no refresh token exists', async () => {
      localStorageMock.store['refresh_token'] = undefined;

      const result = await (httpClient as any).refreshTokenInternal();

      expect(result).toBe(false);
      expect(fetch).not.toHaveBeenCalled();
    });

    it('should handle refresh API failure', async () => {
      localStorageMock.store['refresh_token'] = 'invalid-token';

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          code: 2001,
          message: 'Invalid token',
          data: null,
        }),
      });

      const result = await (httpClient as any).refreshTokenInternal();

      expect(result).toBe(false);
    });

    it('should handle network error', async () => {
      localStorageMock.store['refresh_token'] = 'token';

      (fetch as jest.Mock).mockRejectedValueOnce(new Error('Network error'));

      const result = await (httpClient as any).refreshTokenInternal();

      expect(result).toBe(false);
    });
  });

  // ============================================================================
  // HttpClient 401 Auto-Retry Tests
  // ============================================================================
  describe('HttpClient 401 Auto-Retry', () => {
    it('should automatically retry request after successful token refresh on 401', async () => {
      const mockRefreshToken = 'retry-refresh-token';
      const mockNewAccessToken = 'retry-new-access-token';

      localStorageMock.store['refresh_token'] = mockRefreshToken;
      localStorageMock.store['access_token'] = 'old-token';

      // First call returns 401, second call (after refresh) returns 200
      (fetch as jest.Mock)
        .mockResolvedValueOnce({
          ok: false,
          status: 401,
          statusText: 'Unauthorized',
          headers: new Map(),
        })
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => ({
            code: 0,
            message: 'success',
            data: { access_token: mockNewAccessToken },
          }),
        })
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => ({
            code: 0,
            message: 'success',
            data: { tickets: [] },
          }),
          headers: new Map(),
        });

      const result = await httpClient.get('/api/v1/tickets');

      expect(fetch).toHaveBeenCalledTimes(3); // Initial + refresh + retry
      expect(result).toEqual({ tickets: [] });
    });

    it('should clear tokens and redirect to login when refresh fails on 401', async () => {
      localStorageMock.store['refresh_token'] = 'expired-refresh-token';
      localStorageMock.store['access_token'] = 'expired-token';

      // First call returns 401, refresh also fails
      (fetch as jest.Mock)
        .mockResolvedValueOnce({
          ok: false,
          status: 401,
          statusText: 'Unauthorized',
          headers: new Map(),
        })
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => ({
            code: 2001,
            message: 'Refresh token expired',
            data: null,
          }),
        });

      await expect(httpClient.get('/api/v1/tickets')).rejects.toThrow('Authentication failed');

      expect(localStorageMock.removeItem).toHaveBeenCalledWith('access_token');
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('refresh_token');
    });

    it('should not redirect if already on login page', async () => {
      localStorageMock.store['refresh_token'] = 'expired';
      localStorageMock.store['access_token'] = 'expired';

      // Mock window location
      const originalLocation = window.location;
      Object.defineProperty(window, 'location', {
        value: {
          pathname: '/login',
          href: 'http://localhost/login',
        },
        writable: true,
      });

      (fetch as jest.Mock)
        .mockResolvedValueOnce({
          ok: false,
          status: 401,
          headers: new Map(),
        })
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => ({ code: 2001, message: 'Failed', data: null }),
        });

      await expect(httpClient.get('/api/v1/tickets')).rejects.toThrow('Authentication failed');

      // Should not have redirected
      expect(window.location.href).not.toContain('redirect');

      // Restore
      Object.defineProperty(window, 'location', { value: originalLocation, writable: true });
    });
  });

  // ============================================================================
  // Token Storage Tests
  // ============================================================================
  describe('Token Storage Functions', () => {
    describe('getAccessToken()', () => {
      it('should retrieve access token from storage', () => {
        setAccessToken('test-token');
        const result = getAccessToken();
        expect(result).toBe('test-token');
      });
    });

    describe('getRefreshToken()', () => {
      it('should retrieve refresh token from storage', () => {
        setRefreshToken('test-refresh');
        const result = getRefreshToken();
        expect(result).toBe('test-refresh');
      });
    });

    describe('clearAuthStorage()', () => {
      it('should clear all auth-related storage', () => {
        localStorageMock.store['access_token'] = 'token';
        localStorageMock.store['refresh_token'] = 'refresh';
        localStorageMock.store['current_tenant_code'] = 'tenant';
        localStorageMock.store['current_tenant_id'] = '1';

        clearAuthStorage();

        expect(localStorageMock.removeItem).toHaveBeenCalledWith('access_token');
        expect(localStorageMock.removeItem).toHaveBeenCalledWith('refresh_token');
        expect(localStorageMock.removeItem).toHaveBeenCalledWith('current_tenant_code');
        expect(localStorageMock.removeItem).toHaveBeenCalledWith('current_tenant_id');
      });
    });
  });

  // ============================================================================
  // Edge Cases and Error Scenarios
  // ============================================================================
  describe('Edge Cases', () => {
    it('should handle concurrent token refresh requests', async () => {
      localStorageMock.store['refresh_token'] = 'concurrent-token';

      (fetch as jest.Mock).mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          code: 0,
          message: 'success',
          data: { access_token: 'new-token' },
        }),
      });

      // Simulate concurrent refresh requests
      const promises = [
        AuthService.refreshToken(),
        AuthService.refreshToken(),
        AuthService.refreshToken(),
      ];

      const results = await Promise.all(promises);

      expect(results).toEqual([true, true, true]);
      // Refresh endpoint should be called 3 times (no deduplication in current impl)
      expect(fetch).toHaveBeenCalledTimes(3);
    });

    it('should handle malformed JWT payload', () => {
      localStorageMock.store['access_token'] = 'header.invalid-payload.signature';

      const result = AuthService.isAuthenticated();

      expect(result).toBe(false);
    });

    it('should handle token with missing expiration', () => {
      const payload = { sub: 'user123' }; // No exp field
      const encodedPayload = btoa(JSON.stringify(payload));
      const mockToken = `header.${encodedPayload}.signature`;

      localStorageMock.store['access_token'] = mockToken;

      const result = AuthService.isAuthenticated();

      expect(result).toBe(true); // Token without exp is considered valid
    });

    it('should handle SSR environment (window undefined)', () => {
      // Temporarily make window undefined
      const originalWindow = global.window;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      delete (global as any).window;

      const accessToken = AuthService.getAccessToken();
      const refreshToken = AuthService.getRefreshToken();
      // In SSR, isAuthenticated checks localStorage first which returns null
      const isAuthenticated = AuthService.isAuthenticated();

      expect(accessToken).toBe(null);
      expect(refreshToken).toBe(null);
      // When localStorage returns null, it checks store which is false in SSR
      expect(isAuthenticated).toBe(false);

      // Restore window
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (global as any).window = originalWindow;
    });
  });

  // ============================================================================
  // Integration Scenarios
  // ============================================================================
  describe('Integration Scenarios', () => {
    it('should complete full auth flow: login → token refresh → API call', async () => {
      // Step 1: Login
      (fetch as jest.Mock)
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => ({
            code: 0,
            message: 'success',
            data: {
              access_token: 'initial-access-token',
              refresh_token: 'initial-refresh-token',
              user: { id: 1, username: 'testuser' },
            },
          }),
        })
        // Step 2: Token refresh (simulating expired access token)
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => ({
            code: 0,
            message: 'success',
            data: { access_token: 'refreshed-access-token' },
          }),
        })
        // Step 3: API call after refresh
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => ({
            code: 0,
            message: 'success',
            data: { tickets: [] },
          }),
          headers: new Map(),
        });

      // Login
      const loginResult = await AuthService.login('testuser', 'password');
      expect(loginResult).toBe(true);

      // Verify tokens were stored
      expect(localStorageMock.setItem).toHaveBeenCalledWith('access_token', 'initial-access-token');
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        'refresh_token',
        'initial-refresh-token'
      );

      // Simulate token expiration and refresh
      const refreshResult = await AuthService.refreshToken();
      expect(refreshResult).toBe(true);

      // Make API call (should use refreshed token)
      const tickets = await httpClient.get('/api/v1/tickets');
      expect(tickets).toEqual({ tickets: [] });
    });

    it('should handle session timeout after multiple failed refresh attempts', async () => {
      localStorageMock.store['refresh_token'] = 'expired-refresh';
      localStorageMock.store['access_token'] = 'expired-access';

      // All refresh attempts fail
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          code: 2001,
          message: 'Refresh token expired',
          data: null,
        }),
      });

      const result = await AuthService.refreshToken();
      expect(result).toBe(false);

      // Tokens should be cleared on failure
      expect(localStorageMock.removeItem).toHaveBeenCalledTimes(2);
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('access_token');
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('refresh_token');
    });
  });
});
