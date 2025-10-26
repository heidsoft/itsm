'use client';

import React, { useEffect, useState } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { Spin, Result, Button } from 'antd';
import { Lock, AlertTriangle } from 'lucide-react';
import { RoutePermissionChecker, type RouteConfig, type RoutePermission } from '../../lib/router/route-config';

interface User {
  id: number;
  username: string;
  email: string;
  roles: string[];
  permissions: RoutePermission[];
  isActive: boolean;
}

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
}

interface RouteGuardProps {
  children: React.ReactNode;
  route?: RouteConfig;
  fallback?: React.ReactNode;
}

// 模拟认证服务
class AuthService {
  private static readonly TOKEN_KEY = 'itsm_token';
  private static readonly USER_KEY = 'itsm_user';

  static getToken(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem(this.TOKEN_KEY);
  }

  static setToken(token: string): void {
    if (typeof window === 'undefined') return;
    localStorage.setItem(this.TOKEN_KEY, token);
  }

  static removeToken(): void {
    if (typeof window === 'undefined') return;
    localStorage.removeItem(this.TOKEN_KEY);
    localStorage.removeItem(this.USER_KEY);
  }

  static getUser(): User | null {
    if (typeof window === 'undefined') return null;
    const userStr = localStorage.getItem(this.USER_KEY);
    if (!userStr) return null;
    
    try {
      return JSON.parse(userStr);
    } catch {
      return null;
    }
  }

  static setUser(user: User): void {
    if (typeof window === 'undefined') return;
    localStorage.setItem(this.USER_KEY, JSON.stringify(user));
  }

  static async validateToken(_token: string): Promise<User | null> {
    try {
      // 模拟API调用验证token
      await new Promise(resolve => setTimeout(resolve, 500));
      
      // 模拟用户数据
      const mockUser: User = {
        id: 1,
        username: 'admin',
        email: 'admin@example.com',
        roles: ['admin', 'user'],
        permissions: [
          { resource: 'dashboard', action: 'read' },
          { resource: 'ticket', action: 'read' },
          { resource: 'ticket', action: 'create' },
          { resource: 'ticket', action: 'update' },
          { resource: 'incident', action: 'read' },
          { resource: 'incident', action: 'create' },
          { resource: 'problem', action: 'read' },
          { resource: 'change', action: 'read' },
          { resource: 'knowledge', action: 'read' },
          { resource: 'cmdb', action: 'read' },
          { resource: 'report', action: 'read' },
          { resource: 'admin', action: 'read' },
          { resource: 'user', action: 'manage' },
          { resource: 'role', action: 'manage' },
        ],
        isActive: true,
      };

      return mockUser;
    } catch {
      console.error('Token validation failed');
      return null;
    }
  }

  static async login(username: string, password: string): Promise<{ user: User; token: string } | null> {
    try {
      // 模拟登录API调用
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      if (username === 'admin' && password === 'admin123') {
        const token = 'mock_jwt_token_' + Date.now();
        const user = await this.validateToken(token);
        
        if (user) {
          this.setToken(token);
          this.setUser(user);
          return { user, token };
        }
      }
      
      return null;
    } catch (error) {
      console.error('Login failed:', error);
      return null;
    }
  }

  static logout(): void {
    this.removeToken();
  }
}

// 认证Hook
export const useAuth = () => {
  const [authState, setAuthState] = useState<AuthState>({
    user: null,
    token: null,
    isAuthenticated: false,
    isLoading: true,
  });

  const router = useRouter();

  useEffect(() => {
    const initAuth = async () => {
      const token = AuthService.getToken();
      const cachedUser = AuthService.getUser();

      if (token && cachedUser) {
        // 验证token是否仍然有效
        const user = await AuthService.validateToken(token);
        if (user) {
          setAuthState({
            user,
            token,
            isAuthenticated: true,
            isLoading: false,
          });
        } else {
          // Token无效，清除本地存储
          AuthService.logout();
          setAuthState({
            user: null,
            token: null,
            isAuthenticated: false,
            isLoading: false,
          });
        }
      } else {
        setAuthState({
          user: null,
          token: null,
          isAuthenticated: false,
          isLoading: false,
        });
      }
    };

    initAuth();
  }, []);

  const login = async (username: string, password: string): Promise<boolean> => {
    setAuthState(prev => ({ ...prev, isLoading: true }));
    
    const result = await AuthService.login(username, password);
    
    if (result) {
      setAuthState({
        user: result.user,
        token: result.token,
        isAuthenticated: true,
        isLoading: false,
      });
      return true;
    } else {
      setAuthState(prev => ({ ...prev, isLoading: false }));
      return false;
    }
  };

  const logout = () => {
    AuthService.logout();
    setAuthState({
      user: null,
      token: null,
      isAuthenticated: false,
      isLoading: false,
    });
    router.push('/login');
  };

  return {
    ...authState,
    login,
    logout,
  };
};

// 登录页面组件
const LoginPage: React.FC<{ onLogin: (username: string, password: string) => Promise<boolean> }> = ({ onLogin }) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    const formData = new FormData(e.currentTarget);
    const username = formData.get('username') as string;
    const password = formData.get('password') as string;

    try {
      const success = await onLogin(username, password);
      if (!success) {
        setError('用户名或密码错误');
      }
    } catch (error) {
      setError('登录失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full space-y-8">
        <div className="text-center">
          <Lock size={48} className="mx-auto text-blue-600" />
          <h2 className="mt-6 text-3xl font-extrabold text-gray-900">
            ITSM 系统登录
          </h2>
          <p className="mt-2 text-sm text-gray-600">
            请输入您的账号和密码
          </p>
        </div>
        
        <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
          <div className="space-y-4">
            <div>
              <label htmlFor="username" className="block text-sm font-medium text-gray-700">
                用户名
              </label>
              <input
                id="username"
                name="username"
                type="text"
                required
                className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm placeholder-gray-400 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                placeholder="请输入用户名"
                defaultValue="admin"
              />
            </div>
            
            <div>
              <label htmlFor="password" className="block text-sm font-medium text-gray-700">
                密码
              </label>
              <input
                id="password"
                name="password"
                type="password"
                required
                className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm placeholder-gray-400 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                placeholder="请输入密码"
                defaultValue="admin123"
              />
            </div>
          </div>

          {error && (
            <div className="text-red-600 text-sm text-center">
              {error}
            </div>
          )}

          <div>
            <button
              type="submit"
              disabled={loading}
              className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? (
                <Spin size="small" className="mr-2" />
              ) : null}
              {loading ? '登录中...' : '登录'}
            </button>
          </div>

          <div className="text-xs text-gray-500 text-center">
            <p>测试账号: admin / admin123</p>
          </div>
        </form>
      </div>
    </div>
  );
};

// 权限检查组件
const PermissionGuard: React.FC<{
  children: React.ReactNode;
  route: RouteConfig;
  user: User;
}> = ({ children, route, user }) => {
  const hasPermission = RoutePermissionChecker.hasRoutePermission(
    route,
    user.permissions,
    user.roles
  );

  if (!hasPermission) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Result
          status="403"
          title="403"
          subTitle="抱歉，您没有权限访问此页面"
          icon={<AlertTriangle size={64} className="text-red-500" />}
          extra={
            <Button type="primary" onClick={() => window.history.back()}>
              返回上一页
            </Button>
          }
        />
      </div>
    );
  }

  return <>{children}</>;
};

// 主路由守卫组件
export const RouteGuard: React.FC<RouteGuardProps> = ({ 
  children, 
  route,
  fallback 
}) => {
  const { user, isAuthenticated, isLoading, login } = useAuth();
  const pathname = usePathname();

  // 公共路由，不需要认证
  const publicRoutes = ['/login', '/register', '/forgot-password'];
  const isPublicRoute = publicRoutes.includes(pathname);

  // 加载状态
  if (isLoading) {
    return (
      fallback || (
        <div className="min-h-screen flex items-center justify-center">
          <Spin size="large" />
        </div>
      )
    );
  }

  // 未认证且不是公共路由
  if (!isAuthenticated && !isPublicRoute) {
    return <LoginPage onLogin={login} />;
  }

  // 已认证但访问登录页面，重定向到首页
  if (isAuthenticated && pathname === '/login') {
    window.location.href = '/';
    return null;
  }

  // 需要权限检查的路由
  if (route && user && route.meta?.requireAuth) {
    return (
      <PermissionGuard route={route} user={user}>
        {children}
      </PermissionGuard>
    );
  }

  return <>{children}</>;
};

// 页面级路由守卫HOC
export const withRouteGuard = <P extends object>(
  Component: React.ComponentType<P>,
  route?: RouteConfig
) => {
  const GuardedComponent: React.FC<P> = (props) => {
    return (
      <RouteGuard route={route}>
        <Component {...props} />
      </RouteGuard>
    );
  };

  GuardedComponent.displayName = `withRouteGuard(${Component.displayName || Component.name})`;
  
  return GuardedComponent;
};

export default RouteGuard;