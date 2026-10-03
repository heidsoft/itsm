import type { Tenant, User } from '@/lib/api/api-config';
import { API_BASE_URL } from '@/lib/api/api-config';
import { useAuthStore } from '@/lib/store/auth-store';
import { loadSession, logoutSession, refreshSession, type SessionOutcome } from '@/lib/api/session-api';

/**
 * 认证入口。凭证只存在于后端下发的 httpOnly cookie 里，前端既不读取也不写入：
 * 登录态的唯一真相是 GET /api/v1/auth/session（见 lib/api/session-api.ts）。
 * 因此这里不存在 getAccessToken/setTokens 之类的 helper —— 它们在 JS 侧永远拿不到值。
 */
export class AuthService {
  private constructor() {}

  /**
   * 把后端会话写进 store。tenants[0] 是当前上下文租户；缺失时留空，
   * 由租户切换接口决定，不再伪造「默认租户」。
   */
  private static applySession(user: User, tenants: Tenant[]): void {
    const currentTenant = tenants[0];
    useAuthStore.getState().login(user, currentTenant);
  }

  static async thirdPartyLogin(provider: string, code: string, state?: string | null): Promise<void> {
    // 例外：第三方 OAuth 回调路径（/api/auth/:provider/callback）不同于 SSO callback，无法走 AuthAPI
    // eslint-disable-next-line no-restricted-syntax
    const response = await fetch(`/api/auth/${provider}/callback`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ code, state }),
    });

    if (!response.ok) {
      throw new Error('登录失败');
    }

    // 回调只负责让后端签发会话；身份与租户范围一律回后端会话端点读取。
    const outcome = await loadSession();
    if (outcome.state !== 'authenticated') {
      throw new Error('第三方登录已完成但会话不可用，请重新登录');
    }
    AuthService.applySession(outcome.user, outcome.tenants);
  }

  static getCurrentUser(): User | null {
    return useAuthStore.getState().user;
  }

  /**
   * 读取并向 store 同步后端会话真相，返回三态结果供调用方决定界面：
   * authenticated 填充 store；unauthenticated 清空本地状态；
   * unavailable 不触碰 store，避免一次网络抖动把有效会话变成假未登录。
   */
  static async syncSession(signal?: AbortSignal): Promise<SessionOutcome> {
    const outcome = await loadSession(signal);
    if (outcome.state === 'authenticated') {
      AuthService.applySession(outcome.user, outcome.tenants);
    }
    if (outcome.state === 'unauthenticated') {
      useAuthStore.getState().logout();
    }
    return outcome;
  }

  /** store 里最近一次后端确认过的会话，仅用于界面渲染；授权永远以后端为准。 */
  static isAuthenticated(): boolean {
    return useAuthStore.getState().isAuthenticated;
  }

  // 直接使用fetch进行HTTP请求，避免循环依赖
  private static async makeRequest<T>(endpoint: string, options: RequestInit): Promise<T> {
    const url = `${API_BASE_URL}${endpoint}`;

    // eslint-disable-next-line no-restricted-syntax
    const response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      credentials: options.credentials || 'include', // 默认包含cookies
      ...options,
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const responseData = (await response.json()) as {
      code: number;
      message: string;
      data: T | { retryAfterSeconds?: number } | null;
    };

    // 检查响应码
    if (responseData.code !== 0) {
      // P0-2（2026-09-06 UAT 修复）：登录限流响应 data.retryAfterSeconds 一并带出，
      // 调用方（LoginForm）据此展示倒计时。
      const data = (responseData.data ?? null) as { retryAfterSeconds?: number } | null;
      const err = new Error(responseData.message || '请求失败');
      if (data && typeof data.retryAfterSeconds === 'number') {
        (err as Error & { retryAfterSeconds?: number }).retryAfterSeconds = data.retryAfterSeconds;
      }
      throw err;
    }

    return responseData.data as T;
  }

  /** 主动续签；返回是否拿到新会话。并发调用共享一次请求（refresh token 单次使用）。 */
  static async refreshToken(): Promise<boolean> {
    const outcome = await refreshSession();
    if (outcome.state !== 'refreshed') {
      if (outcome.state === 'expired') {
        useAuthStore.getState().logout();
      }
      return false;
    }
    return true;
  }

  /**
   * 登出：先清本地状态（界面立刻不可用），再让后端清 cookie 并吊销两类凭证。
   * 后端吊销失败必须可见——cookie 清了但 7 天 refresh 凭证仍有效是运维需要知道的状态。
   */
  static logout(): void {
    useAuthStore.getState().logout();
    void logoutSession().then(result => {
      if (!result.revoked) {
        console.warn('[auth] 服务端会话吊销未完成:', result.reason);
      }
    });
  }

  /**
   * 登录。响应只给出 `{user, expiresIn}`（凭证走 httpOnly cookie），
   * 因此成功后回到会话端点确认后端真正接受了它下发的 cookie，并取得租户范围与权限。
   */
  static async login(
    username: string,
    password: string,
    tenantCode?: string
  ): Promise<boolean> {
    try {
      await this.makeRequest<{ user: User; expiresIn: number }>('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({
          username,
          password,
          tenantCode: tenantCode,
        }),
      });

      const outcome = await loadSession();
      if (outcome.state === 'authenticated') {
        AuthService.applySession(outcome.user, outcome.tenants);
        return true;
      }
      if (outcome.state === 'unavailable') {
        // 后端已接受凭据，但会话查询失败不能谎报「登录成功」，也不能说成密码错误。
        throw new Error('登录已接受，但暂时无法确认会话，请稍后重试');
      }
      return false;
    } catch (error) {
      // P0-2（2026-09-06 UAT 修复）：限流响应 data.retryAfterSeconds 由 makeRequest
      // 附加到 Error 上，LoginForm 据此展示按钮倒计时。仅对限流错误 rethrow，
      // 其他登录失败（凭证错误、网络错误）按调用方契约返回 false。
      const e = error as Error & { retryAfterSeconds?: number };
      if (typeof e.retryAfterSeconds === 'number' && e.retryAfterSeconds > 0) {
        throw e;
      }
      if (e.message.includes('暂时无法确认会话')) {
        throw e;
      }
      console.error('Login failed:', error);
      return false;
    }
  }

  // 注册（角色由服务端固定指派 end_user，不接受前端指定）
  static async register(params: {
    username: string;
    email: string;
    password: string;
    fullName: string;
    phone?: string;
    company?: string;
  }): Promise<boolean> {
    try {
      await this.makeRequest<{ id: number; username: string; email: string; message: string }>(
        '/api/v1/auth/register',
        {
          method: 'POST',
          body: JSON.stringify({
            username: params.username,
            email: params.email,
            password: params.password,
            fullName: params.fullName,
            phone: params.phone,
            company: params.company,
          }),
        }
      );

      return true;
    } catch (error) {
      console.error('Registration failed:', error);
      return false;
    }
  }

  // 发送密码重置邮件
  static async forgotPassword(email: string, tenantCode?: string): Promise<{ ok: true } | { ok: false; message: string }> {
    try {
      await this.makeRequest<{ message: string }>('/api/v1/auth/forgot-password', {
        method: 'POST',
        body: JSON.stringify({
          email,
          tenantCode: tenantCode,
        }),
      });

      return { ok: true };
    } catch (error) {
      console.error('Forgot password request failed:', error);
      const msg = error instanceof Error ? error.message : '请求失败';
      return { ok: false, message: msg };
    }
  }

  // 重置密码
  static async resetPassword(params: {
    token: string;
    email: string;
    password: string;
    passwordConfirm: string;
  }): Promise<boolean> {
    try {
      await this.makeRequest<{ message: string }>('/api/v1/auth/reset-password', {
        method: 'POST',
        body: JSON.stringify({
          token: params.token,
          email: params.email,
          password: params.password,
          passwordConfirm: params.passwordConfirm,
        }),
      });

      return true;
    } catch (error) {
      console.error('Reset password failed:', error);
      return false;
    }
  }

  // 验证重置令牌
  static async validateResetToken(token: string, email: string): Promise<boolean> {
    try {
      const result = await this.makeRequest<{ valid: boolean; email: string }>(
        '/api/v1/auth/validate-reset-token',
        {
          method: 'POST',
          body: JSON.stringify({
            token,
            email,
          }),
        }
      );

      return result.valid;
    } catch (error) {
      console.error('Validate reset token failed:', error);
      return false;
    }
  }
}

export default AuthService;
