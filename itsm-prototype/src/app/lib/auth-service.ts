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
    const token = this.getToken();
    if (!token) {
      return false;
    }
    
    // 检查token是否为demo token，如果是则清除
    if (token === 'demo-token-12345') {
      this.clearToken();
      return false;
    }
    
    // 可以添加token格式验证或过期检查
    try {
      // 简单的JWT格式检查
      const parts = token.split('.');
      if (parts.length !== 3) {
        this.clearToken();
        return false;
      }
      return true;
    } catch (error) {
      this.clearToken();
      return false;
    }
  }

  // 添加登录方法
  static async login(username: string, password: string): Promise<boolean> {
    try {
      const response = await httpClient.post<{token: string, user: any}>('/api/login', {
        username,
        password
      });
      
      if (response.code === 0) {
        this.setToken(response.data.token);
        return true;
      }
      return false;
    } catch (error) {
      console.error('Login failed:', error);
      return false;
    }
  }

  // 移除默认token设置
  // static setDefaultToken() {
  //   this.setToken('demo-token-12345');
  // }
}

// 移除自动设置默认token
// if (typeof window !== 'undefined') {
//   AuthService.setDefaultToken();
// }

export default AuthService;