/**
 * Auth API Unit Tests
 * 认证API单元测试
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
};
vi.stubGlobal('localStorage', localStorageMock);

// Mock fetch
const fetchMock = vi.fn();
vi.stubGlobal('fetch', fetchMock);

describe('Auth API Tests', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Login', () => {
    it('should login successfully with valid credentials', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          token: 'mock-jwt-token',
          refresh_token: 'mock-refresh-token',
          user: {
            id: 1,
            username: 'admin',
            email: 'admin@example.com',
          },
        },
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      });

      // Import and call login
      const { AuthApi } = await import('@/lib/api/auth-api');
      const result = await AuthApi.login('admin', 'password123');

      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/auth/login'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ username: 'admin', password: 'password123' }),
        })
      );

      expect(result.token).toBe('mock-jwt-token');
      expect(result.user.username).toBe('admin');
    });

    it('should fail login with invalid credentials', async () => {
      const mockResponse = {
        code: 2001,
        message: '用户名或密码错误',
      };

      fetchMock.mockResolvedValueOnce({
        ok: false,
        json: async () => mockResponse,
      });

      const { AuthApi } = await import('@/lib/api/auth-api');

      await expect(AuthApi.login('admin', 'wrongpassword')).rejects.toThrow();
    });

    it('should handle network error', async () => {
      fetchMock.mockRejectedValueOnce(new Error('Network error'));

      const { AuthApi } = await import('@/lib/api/auth-api');

      await expect(AuthApi.login('admin', 'password')).rejects.toThrow('Network error');
    });
  });

  describe('Token Management', () => {
    it('should store token in localStorage', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          token: 'mock-jwt-token',
          refresh_token: 'mock-refresh-token',
        },
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      });

      const { AuthApi } = await import('@/lib/api/auth-api');
      await AuthApi.login('admin', 'password');

      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        'itsm_token',
        'mock-jwt-token'
      );
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        'itsm_refresh_token',
        'mock-refresh-token'
      );
    });

    it('should retrieve token from localStorage', async () => {
      localStorageMock.getItem.mockReturnValue('mock-jwt-token');

      const { AuthApi } = await import('@/lib/api/auth-api');
      const token = AuthApi.getToken();

      expect(token).toBe('mock-jwt-token');
    });

    it('should return null when no token exists', async () => {
      localStorageMock.getItem.mockReturnValue(null);

      const { AuthApi } = await import('@/lib/api/auth-api');
      const token = AuthApi.getToken();

      expect(token).toBeNull();
    });

    it('should remove tokens on logout', async () => {
      const { AuthApi } = await import('@/lib/api/auth-api');
      AuthApi.logout();

      expect(localStorageMock.removeItem).toHaveBeenCalledWith('itsm_token');
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('itsm_refresh_token');
    });
  });

  describe('Refresh Token', () => {
    it('should refresh token successfully', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          token: 'new-mock-token',
          refresh_token: 'new-refresh-token',
        },
      };

      localStorageMock.getItem.mockReturnValue('mock-refresh-token');

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      });

      const { AuthApi } = await import('@/lib/api/auth-api');
      const result = await AuthApi.refreshToken();

      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/auth/refresh'),
        expect.objectContaining({
          method: 'POST',
        })
      );

      expect(result.token).toBe('new-mock-token');
    });

    it('should handle refresh token failure', async () => {
      localStorageMock.getItem.mockReturnValue('expired-refresh-token');

      const mockResponse = {
        code: 2002,
        message: 'Refresh token已过期',
      };

      fetchMock.mockResolvedValueOnce({
        ok: false,
        json: async () => mockResponse,
      });

      const { AuthApi } = await import('@/lib/api/auth-api');

      await expect(AuthApi.refreshToken()).rejects.toThrow();
    });
  });

  describe('Get Current User', () => {
    it('should get current user info', async () => {
      const mockUser = {
        id: 1,
        username: 'admin',
        email: 'admin@example.com',
        role: 'admin',
      };

      const mockResponse = {
        code: 0,
        message: 'success',
        data: mockUser,
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      });

      const { AuthApi } = await import('@/lib/api/auth-api');
      const user = await AuthApi.getCurrentUser();

      expect(user.username).toBe('admin');
      expect(user.email).toBe('admin@example.com');
    });

    it('should handle unauthorized error', async () => {
      fetchMock.mockResolvedValueOnce({
        ok: false,
        status: 401,
      });

      const { AuthApi } = await import('@/lib/api/auth-api');

      await expect(AuthApi.getCurrentUser()).rejects.toThrow();
    });
  });

  describe('Password Reset', () => {
    it('should request password reset', async () => {
      const mockResponse = {
        code: 0,
        message: '邮件已发送',
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      });

      const { AuthApi } = await import('@/lib/api/auth-api');
      const result = await AuthApi.requestPasswordReset('user@example.com');

      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/auth/password/reset'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ email: 'user@example.com' }),
        })
      );
    });

    it('should reset password with token', async () => {
      const mockResponse = {
        code: 0,
        message: '密码重置成功',
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      });

      const { AuthApi } = await import('@/lib/api/auth-api');
      const result = await AuthApi.resetPassword('reset-token', 'newpassword123');

      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/auth/password/confirm'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            token: 'reset-token',
            password: 'newpassword123',
          }),
        })
      );
    });
  });
});

describe('Auth Store Tests', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should initialize with default state', async () => {
    const { useAuthStore } = await import('@/lib/store/auth-store');
    const state = useAuthStore.getState();

    expect(state.isAuthenticated).toBe(false);
    expect(state.user).toBeNull();
    expect(state.token).toBeNull();
  });

  it('should set user on login', async () => {
    const { useAuthStore } = await import('@/lib/store/auth-store');

    const mockUser = {
      id: 1,
      username: 'admin',
      email: 'admin@example.com',
    };

    useAuthStore.getState().setUser(mockUser, 'mock-token');

    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(true);
    expect(state.user?.username).toBe('admin');
    expect(state.token).toBe('mock-token');
  });

  it('should clear user on logout', async () => {
    const { useAuthStore } = await import('@/lib/store/auth-store');

    const mockUser = { id: 1, username: 'admin' };
    useAuthStore.getState().setUser(mockUser, 'token');
    useAuthStore.getState().logout();

    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(false);
    expect(state.user).toBeNull();
    expect(state.token).toBeNull();
  });

  it('should update user profile', async () => {
    const { useAuthStore } = await import('@/lib/store/auth-store');

    const mockUser = { id: 1, username: 'admin', email: 'admin@example.com' };
    useAuthStore.getState().setUser(mockUser, 'token');

    useAuthStore.getState().updateProfile({ nickname: '管理员' });

    const state = useAuthStore.getState();
    expect(state.user?.nickname).toBe('管理员');
  });
});
