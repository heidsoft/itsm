'use client';

import React, { useEffect, useState } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { Spin, Result, Button } from 'antd';
import { AlertTriangle } from 'lucide-react';
import {
  RoutePermissionChecker,
  type RouteConfig,
  type RoutePermission,
} from '../../lib/router/route-config';
import { httpClient } from '@/lib/api/http-client';
import { AuthService as RealAuthService } from '@/lib/services/auth-service';

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
      try {
        const me = await httpClient.get<{
          id: number;
          username: string;
          email: string;
          role: string;
          active: boolean;
          permissions?: string[];
        }>('/api/v1/auth/me');

        const perms = Array.isArray(me?.permissions) ? me.permissions : [];
        const routePerms: RoutePermission[] = perms
          .map(p => {
            const [resource, action] = String(p).split(':');
            if (!resource || !action) return null;
            return { resource, action } as RoutePermission;
          })
          .filter(Boolean) as RoutePermission[];

        setAuthState({
          user: {
            id: Number(me?.id || 0),
            username: String(me?.username || ''),
            email: String(me?.email || ''),
            roles: [String(me?.role || '')].filter(Boolean),
            permissions: routePerms,
            isActive: Boolean(me?.active),
          },
          token: 'authenticated',
          isAuthenticated: true,
          isLoading: false,
        });
      } catch {
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

  const logout = () => {
    RealAuthService.logout();
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
    logout,
  };
};

// 权限检查组件
const PermissionGuard: React.FC<{
  children: React.ReactNode;
  route: RouteConfig;
  user: User;
}> = ({ children, route, user }) => {
  const router = useRouter();
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
            <Button type="primary" onClick={() => router.back()}>
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
export const RouteGuard: React.FC<RouteGuardProps> = ({ children, route, fallback }) => {
  const { user, isAuthenticated, isLoading } = useAuth();
  const pathname = usePathname();
  const router = useRouter();

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
    router.push(`/login?redirect=${encodeURIComponent(pathname || '/')}`);
    return null;
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
  const GuardedComponent: React.FC<P> = props => {
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
