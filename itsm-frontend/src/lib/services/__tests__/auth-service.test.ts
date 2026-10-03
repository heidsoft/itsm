/**
 * Auth Service 测试套件。
 *
 * 关键约束：AuthService 不再持有任何凭证，也不再从 cookie/localStorage 推断登录态。
 * 所有登录态结论必须来自后端会话端点，因此这里的 fetch 序列固定为
 * 「业务请求 → GET /api/v1/auth/session」。
 */

import { AuthService } from '../auth-service';

// Mock Zustand store
const mockLogin = jest.fn();
const mockLogout = jest.fn();

const mockStore: {
  user: { id: number; username: string; email: string; name: string; role: string; tenantId: number } | null;
  isAuthenticated: boolean;
  login: typeof mockLogin;
  logout: typeof mockLogout;
} = {
  user: {
    id: 1,
    username: 'testuser',
    email: 'test@example.com',
    name: 'Test User',
    role: 'agent',
    tenantId: 1,
  },
  isAuthenticated: true,
  login: mockLogin,
  logout: mockLogout,
};

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStore: {
    getState: jest.fn(() => mockStore),
  },
}));

// Mock fetch
const mockFetch = jest.fn();
global.fetch = mockFetch;

const SESSION_OK = {
  ok: true,
  status: 200,
  json: () =>
    Promise.resolve({
      code: 0,
      message: 'success',
      data: {
        user: { id: 1, username: 'testuser', email: 'test@example.com', name: 'Test User', role: 'agent', tenantId: 1 },
        tenants: [{ id: 1, name: 'Test Tenant', code: 'test', type: 'standard', status: 'active' }],
        expiresIn: 900,
      },
    }),
};

const SESSION_UNAUTHENTICATED = { ok: false, status: 401, json: () => Promise.resolve({ code: 2001, message: 'unauthorized' }) };

const SESSION_UNAVAILABLE = { ok: false, status: 503, json: () => Promise.resolve({ code: 5001, message: 'db down' }) };

function loginAccepted() {
  return {
    ok: true,
    status: 200,
    json: () => Promise.resolve({ code: 0, message: 'success', data: { user: { id: 1 }, expiresIn: 900 } }),
  };
}

/** logout() 的吊销请求是 fire-and-forget，断言前需要让微任务队列跑完。 */
function flush(): Promise<void> {
  return new Promise(resolve => {
    setTimeout(resolve, 0);
  });
}

describe('AuthService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(console, 'error').mockImplementation(() => {});
    jest.spyOn(console, 'warn').mockImplementation(() => {});
  });

  describe('不再暴露凭证读写接口', () => {
    it('setTokens/getAccessToken/getRefreshToken/getToken/clearTokens 已全部移除', () => {
      const service = AuthService as unknown as Record<string, unknown>;
      expect(service.setTokens).toBeUndefined();
      expect(service.getAccessToken).toBeUndefined();
      expect(service.getRefreshToken).toBeUndefined();
      expect(service.getToken).toBeUndefined();
      expect(service.clearTokens).toBeUndefined();
    });
  });

  describe('Authentication Status', () => {
    describe('getCurrentUser', () => {
      it('should return current user from store', () => {
        const user = AuthService.getCurrentUser();
        expect(user).toBeDefined();
        expect(user?.id).toBe(1);
        expect(user?.username).toBe('testuser');
      });
    });

    describe('isAuthenticated', () => {
      it('只反映 store 中由会话端点确认过的状态', () => {
        expect(AuthService.isAuthenticated()).toBe(true);
      });

      it('store 未认证时即使浏览器里存在 cookie 也返回 false', () => {
        mockStore.isAuthenticated = false;
        try {
          expect(AuthService.isAuthenticated()).toBe(false);
        } finally {
          mockStore.isAuthenticated = true;
        }
      });
    });
  });

  describe('syncSession', () => {
    it('已认证时把后端身份与租户写入 store', async () => {
      mockFetch.mockResolvedValueOnce(SESSION_OK);

      const outcome = await AuthService.syncSession();

      expect(outcome.state).toBe('authenticated');
      expect(mockLogin).toHaveBeenCalledWith(
        expect.objectContaining({ id: 1 }),
        expect.objectContaining({ code: 'test' }),
      );
    });

    it('未认证时清空本地状态', async () => {
      mockFetch.mockResolvedValueOnce(SESSION_UNAUTHENTICATED);

      const outcome = await AuthService.syncSession();

      expect(outcome).toEqual({ state: 'unauthenticated' });
      expect(mockLogout).toHaveBeenCalledTimes(1);
      expect(mockLogin).not.toHaveBeenCalled();
    });

    it('服务故障时既不登出也不登录，把三态交给调用方', async () => {
      mockFetch.mockResolvedValueOnce(SESSION_UNAVAILABLE);

      const outcome = await AuthService.syncSession();

      expect(outcome.state).toBe('unavailable');
      expect(mockLogout).not.toHaveBeenCalled();
      expect(mockLogin).not.toHaveBeenCalled();
    });
  });

  describe('Login Functionality', () => {
    describe('login', () => {
      it('登录请求成功后回到会话端点确认，并取得租户范围', async () => {
        mockFetch.mockResolvedValueOnce(loginAccepted());
        mockFetch.mockResolvedValueOnce(SESSION_OK);

        const result = await AuthService.login('testuser', 'password123', 'test');

        expect(result).toBe(true);
        expect(mockFetch).toHaveBeenNthCalledWith(
          1,
          expect.stringContaining('/api/v1/auth/login'),
          expect.objectContaining({ method: 'POST', body: expect.stringContaining('testuser') }),
        );
        expect(mockFetch).toHaveBeenNthCalledWith(2, expect.stringContaining('/api/v1/auth/session'), expect.any(Object));
        expect(mockLogin).toHaveBeenCalled();
      });

      it('登录请求只返回用户与剩余时间，不接受前端写入的 rememberMe', async () => {
        mockFetch.mockResolvedValueOnce(loginAccepted());
        mockFetch.mockResolvedValueOnce(SESSION_OK);

        await AuthService.login('testuser', 'password123');

        const body = JSON.parse(mockFetch.mock.calls[0][1].body as string);
        expect(body).not.toHaveProperty('rememberMe');
      });

      it('后端接受凭证但会话查询失败时抛错，不谎报登录成功', async () => {
        mockFetch.mockResolvedValueOnce(loginAccepted());
        mockFetch.mockResolvedValueOnce(SESSION_UNAVAILABLE);

        await expect(AuthService.login('testuser', 'password123')).rejects.toThrow('暂时无法确认会话');
        expect(mockLogin).not.toHaveBeenCalled();
      });

      it('会话端点判定未登录时按登录失败返回', async () => {
        mockFetch.mockResolvedValueOnce(loginAccepted());
        mockFetch.mockResolvedValueOnce(SESSION_UNAUTHENTICATED);

        const result = await AuthService.login('testuser', 'password123');

        expect(result).toBe(false);
        expect(mockLogin).not.toHaveBeenCalled();
      });

      it('should return false on login failure', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: false,
          status: 401,
          json: () => Promise.resolve({ code: 2001, message: 'Invalid credentials' }),
        });

        const result = await AuthService.login('wronguser', 'wrongpass');

        expect(result).toBe(false);
      });

      it('should handle network errors gracefully', async () => {
        mockFetch.mockRejectedValueOnce(new Error('Network error'));

        const result = await AuthService.login('testuser', 'password');

        expect(result).toBe(false);
      });
    });
  });

  describe('Third Party Login', () => {
    it('回调成功后从会话端点建立登录态', async () => {
      mockFetch.mockResolvedValueOnce({ ok: true, status: 200, json: () => Promise.resolve({ code: 0 }) });
      mockFetch.mockResolvedValueOnce(SESSION_OK);

      await expect(AuthService.thirdPartyLogin('google', 'code123')).resolves.not.toThrow();
      expect(mockLogin).toHaveBeenCalled();
    });

    it('回调被接受但后端未建立会话时抛错', async () => {
      mockFetch.mockResolvedValueOnce({ ok: true, status: 200, json: () => Promise.resolve({ code: 0 }) });
      mockFetch.mockResolvedValueOnce(SESSION_UNAUTHENTICATED);

      await expect(AuthService.thirdPartyLogin('google', 'code123')).rejects.toThrow('会话不可用');
      expect(mockLogin).not.toHaveBeenCalled();
    });

    it('should throw error on failed third party login', async () => {
      mockFetch.mockResolvedValueOnce({ ok: false, status: 400, json: () => Promise.resolve({ code: 1001 }) });

      await expect(AuthService.thirdPartyLogin('google', 'invalid')).rejects.toThrow('登录失败');
    });
  });

  describe('Registration', () => {
    it('should register user successfully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () =>
          Promise.resolve({
            code: 0,
            message: 'success',
            data: { id: 2, username: 'newuser', email: 'new@example.com' },
          }),
      });

      const result = await AuthService.register({
        username: 'newuser',
        email: 'new@example.com',
        password: 'password123',
        fullName: 'New User',
        phone: '1234567890',
        company: 'Test Company',
      });

      expect(result).toBe(true);
      // P0-1：注册是未认证入口，角色与租户由服务端决定，请求体不得携带同名字段
      const body = JSON.parse(mockFetch.mock.calls[0][1].body as string);
      expect(body).not.toHaveProperty('role');
      expect(body).not.toHaveProperty('tenantCode');
    });

    it('should return false on registration failure', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Registration failed'));

      const result = await AuthService.register({
        username: 'newuser',
        email: 'new@example.com',
        password: 'password123',
        fullName: 'New User',
      });

      expect(result).toBe(false);
    });
  });

  describe('Password Reset', () => {
    describe('forgotPassword', () => {
      it('should send password reset email successfully', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: () => Promise.resolve({ code: 0, message: 'Reset email sent' }),
        });

        const result = await AuthService.forgotPassword('test@example.com', 'test');

        expect(result).toBe(true);
      });

      it('should return false on forgot password failure', async () => {
        mockFetch.mockRejectedValueOnce(new Error('Failed to send email'));

        const result = await AuthService.forgotPassword('test@example.com');

        expect(result).toBe(false);
      });
    });

    describe('resetPassword', () => {
      it('should reset password successfully', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: () => Promise.resolve({ code: 0, message: 'Password reset' }),
        });

        const result = await AuthService.resetPassword({
          token: 'reset-token',
          email: 'test@example.com',
          password: 'newpassword123',
          passwordConfirm: 'newpassword123',
        });

        expect(result).toBe(true);
      });

      it('should return false on reset password failure', async () => {
        mockFetch.mockRejectedValueOnce(new Error('Invalid token'));

        const result = await AuthService.resetPassword({
          token: 'invalid-token',
          email: 'test@example.com',
          password: 'newpass',
          passwordConfirm: 'newpass',
        });

        expect(result).toBe(false);
      });
    });

    describe('validateResetToken', () => {
      it('should validate reset token successfully', async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: () =>
            Promise.resolve({
              code: 0,
              message: 'success',
              data: { valid: true, email: 'test@example.com' },
            }),
        });

        const result = await AuthService.validateResetToken('valid-token', 'test@example.com');

        expect(result).toBe(true);
      });

      it('should return false for invalid token', async () => {
        mockFetch.mockRejectedValueOnce(new Error('Invalid token'));

        const result = await AuthService.validateResetToken('invalid-token', 'test@example.com');

        expect(result).toBe(false);
      });
    });
  });

  describe('Token Refresh', () => {
    it('走 canonical 续签端点，凭证由 httpOnly cookie 携带', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ code: 0, message: 'success', data: { expiresIn: 900 } }),
      });

      const result = await AuthService.refreshToken();

      expect(result).toBe(true);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/auth/refresh'),
        expect.objectContaining({ method: 'POST', credentials: 'include' }),
      );
    });

    it('后端判定过期时清本地状态并返回 false', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        json: () => Promise.resolve({ code: 2001, message: 'expired' }),
      });

      const result = await AuthService.refreshToken();

      expect(result).toBe(false);
      expect(mockLogout).toHaveBeenCalledTimes(1);
    });

    it('服务故障不等于过期：返回 false 但保留会话', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: () => Promise.resolve({ code: 5001, message: 'boom' }),
      });

      const result = await AuthService.refreshToken();

      expect(result).toBe(false);
      expect(mockLogout).not.toHaveBeenCalled();
    });
  });

  describe('Logout', () => {
    it('清本地状态并向后端吊销会话', async () => {
      mockFetch.mockResolvedValueOnce({ ok: true, status: 200, json: () => Promise.resolve({ code: 0, message: 'Logged out' }) });

      AuthService.logout();
      await flush();

      expect(mockLogout).toHaveBeenCalledTimes(1);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/auth/logout'),
        expect.objectContaining({ method: 'POST', keepalive: true }),
      );
      expect(console.warn).not.toHaveBeenCalled();
    });

    it('后端吊销失败时留下可观察告警', async () => {
      mockFetch.mockResolvedValueOnce({ ok: false, status: 500, json: () => Promise.resolve({ code: 5001, message: 'store down' }) });

      AuthService.logout();
      await flush();

      expect(mockLogout).toHaveBeenCalledTimes(1);
      expect(console.warn).toHaveBeenCalledWith(expect.stringContaining('吊销'), 'store down');
    });

    it('should handle logout network error gracefully', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      expect(() => AuthService.logout()).not.toThrow();
      expect(mockLogout).toHaveBeenCalledTimes(1);
    });
  });
});
