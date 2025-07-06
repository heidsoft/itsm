import { httpClient } from './http-client';

export class AuthService {
  // 设置认证token
  static setToken(token: string) {
    httpClient.setToken(token);
  }

  // 获取当前token
  static getToken(): string | null {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('auth_token');
    }
    return null;
  }

  // 清除token
  static clearToken() {
    httpClient.clearToken();
  }

  // 检查是否已认证
  static isAuthenticated(): boolean {
    return !!this.getToken();
  }

  // 临时设置默认token（用于演示）
  static setDefaultToken() {
    // 这里设置一个默认的token，实际项目中应该通过登录获取
    this.setToken('demo-token-12345');
  }
}

// 初始化时设置默认token
if (typeof window !== 'undefined') {
  AuthService.setDefaultToken();
}

export default AuthService;