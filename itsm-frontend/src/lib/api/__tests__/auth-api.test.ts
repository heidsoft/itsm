import { authApiClient, AuthAPI } from '@/lib/api/auth-api';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
  },
}));

jest.mock('@/lib/auth/token-storage', () => ({
  clearAuthStorage: jest.fn(),
}));

// Mock fetch globally
const mockFetch = jest.fn();
global.fetch = mockFetch;

describe('AuthApiClient', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  describe('getCsrfToken', () => {
    it('should fetch CSRF token', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ data: { csrf_token: 'token123' } }),
      });
      const token = await authApiClient.getCsrfToken();
      expect(token).toBe('token123');
      expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('/api/v1/csrf-token'), expect.any(Object));
    });

    it('should throw on non-ok response', async () => {
      mockFetch.mockResolvedValue({ ok: false });
      await expect(authApiClient.getCsrfToken()).rejects.toThrow('Failed to get CSRF token');
    });
  });

  describe('login', () => {
    it('should login successfully', async () => {
      mockFetch
        .mockResolvedValueOnce({ ok: true, json: async () => ({ data: { csrf_token: 'csrf1' } }) })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ data: { user: { id: '1', username: 'admin' } } }) });
      const res = await authApiClient.login({ username: 'admin', password: 'pass' });
      expect(res.success).toBe(true);
    });

    it('should return error on failed login', async () => {
      mockFetch
        .mockResolvedValueOnce({ ok: true, json: async () => ({ data: { csrf_token: 'csrf1' } }) })
        .mockResolvedValueOnce({ ok: false, json: async () => ({ message: 'Invalid credentials' }) });
      const res = await authApiClient.login({ username: 'admin', password: 'wrong' });
      expect(res.success).toBe(false);
      expect(res.error).toBe('Invalid credentials');
    });

    it('should handle network error', async () => {
      mockFetch.mockRejectedValue(new Error('Network down'));
      const res = await authApiClient.login({ username: 'admin', password: 'pass', csrfToken: 'x' });
      expect(res.success).toBe(false);
      expect(res.error).toBe('Network down');
    });
  });

  describe('refreshToken / validateToken 已从废弃客户端移除', () => {
    it('不再保留第二套续签与 token 校验入口', () => {
      const client = authApiClient as unknown as Record<string, unknown>;
      expect(client.refreshToken).toBeUndefined();
      expect(client.validateToken).toBeUndefined();
    });
  });

  describe('logout', () => {
    it('should logout successfully', async () => {
      mockFetch.mockResolvedValue({ ok: true });
      const res = await authApiClient.logout();
      expect(res.success).toBe(true);
    });

    it('should clear storage even on failure', async () => {
      mockFetch.mockRejectedValue(new Error('timeout'));
      const res = await authApiClient.logout();
      expect(res.success).toBe(false);
    });
  });

  describe('getWebAuthnChallenge', () => {
    it('should get challenge', async () => {
      mockFetch.mockResolvedValue({ ok: true, json: async () => ({ challenge: 'abc123' }) });
      const res = await authApiClient.getWebAuthnChallenge('user1');
      expect(res.success).toBe(true);
      expect(res.challenge).toBe('abc123');
    });
  });

  describe('initiateSSOLogin', () => {
    it('should initiate SSO', async () => {
      mockFetch.mockResolvedValue({ ok: true, json: async () => ({ redirectUrl: 'https://sso.example.com' }) });
      const res = await authApiClient.initiateSSOLogin('default');
      expect(res.success).toBe(true);
      expect(res.redirectUrl).toBe('https://sso.example.com');
    });
  });

  describe('AuthAPI convenience object', () => {
    it('只保留废弃客户端仍支持的入口，续签与校验走会话端点', () => {
      expect(AuthAPI.login).toBeDefined();
      expect(AuthAPI.logout).toBeDefined();
      expect(AuthAPI.getCsrfToken).toBeDefined();
      expect(AuthAPI.getWebAuthnChallenge).toBeDefined();
      expect(AuthAPI.verifyWebAuthn).toBeDefined();
      expect(AuthAPI.initiateSSOLogin).toBeDefined();
      expect((AuthAPI as unknown as Record<string, unknown>).refreshToken).toBeUndefined();
      expect((AuthAPI as unknown as Record<string, unknown>).validateToken).toBeUndefined();
    });
  });
});
