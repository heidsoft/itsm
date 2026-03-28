'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/lib/store/auth-store';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { isAuthenticated as checkCookieAuth } from '@/lib/auth/token-storage';
import { httpClient } from '@/lib/api/http-client';

interface AuthGuardProps {
  children: React.ReactNode;
  requireAuth?: boolean;
  requiredPermissions?: string[];
  requiredRole?: string;
  fallback?: React.ReactNode;
  redirectTo?: string;
}

/**
 * 认证守卫组件
 * 用于保护需要认证的页面和组件
 */
export const AuthGuard: React.FC<AuthGuardProps> = ({
  children,
  requireAuth = true,
  requiredPermissions = [],
  requiredRole,
  fallback,
  redirectTo = '/login',
}) => {
  const router = useRouter();
  const { isAuthenticated: storeIsAuth, user, isLoading, hasPermission, hasRole } = useAuthStore();
  const [isInitializing, setIsInitializing] = useState(true);

  useEffect(() => {
    const initializeAuth = async () => {
      try {
        // 检查 httpOnly cookie 中是否有认证信息
        const hasAuth = checkCookieAuth();

        if (hasAuth) {
          // 从后端获取真实的用户和租户信息
          try {
            const [userResponse, tenantsResponse] = await Promise.all([
              httpClient.get<{
                id: number;
                username: string;
                email: string;
                name: string;
                role: string;
                department: string;
                tenant_id: number;
              }>('/api/v1/auth/me'),
              httpClient.get<{
                tenants: Array<{
                  id: number;
                  name: string;
                  code: string;
                  domain: string;
                  type: string;
                  status: string;
                }>;
              }>('/api/v1/auth/tenants'),
            ]);

            if (userResponse && tenantsResponse?.tenants?.length > 0) {
              const currentTenant = tenantsResponse.tenants[0];
              const { login } = useAuthStore.getState();
              login(
                {
                  id: userResponse.id,
                  username: userResponse.username,
                  email: userResponse.email,
                  name: userResponse.name,
                  role: userResponse.role,
                  department: userResponse.department,
                },
                'authenticated', // Token is in httpOnly cookie, not accessible here
                {
                  id: currentTenant.id,
                  name: currentTenant.name,
                  code: currentTenant.code,
                  type: currentTenant.type as 'standard' | 'trial' | 'enterprise',
                  status: currentTenant.status as 'active' | 'suspended' | 'expired',
                  createdAt: new Date().toISOString(),
                  updatedAt: new Date().toISOString(),
                }
              );
            }
          } catch (e) {
            console.error('Failed to restore user info from backend:', e);
          }
        }

        setIsInitializing(false);
      } catch (error) {
        console.error('Auth initialization failed:', error);
        setIsInitializing(false);
      }
    };

    initializeAuth();
  }, []);

  // 处理未认证重定向
  useEffect(() => {
    if (requireAuth && !storeIsAuth && !fallback) {
      router.push(redirectTo);
    }
  }, [requireAuth, storeIsAuth, fallback, router, redirectTo]);

  // 正在初始化或加载中
  if (isInitializing || isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  // 不需要认证，直接渲染子组件
  if (!requireAuth) {
    return <>{children}</>;
  }

  if (requireAuth && (!storeIsAuth || !user)) {
    if (fallback) {
      return <>{fallback}</>;
    }

    // 重定向处理
    return (
      <div className="flex items-center justify-center min-h-screen">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  // 检查角色权限
  if (requiredRole && !hasRole(requiredRole)) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">访问被拒绝</h2>
          <p className="text-gray-600 mb-6">您没有访问此页面的权限</p>
          <button
            onClick={e => {
              e.preventDefault();
              setTimeout(() => router.back(), 0);
            }}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
          >
            返回上一页
          </button>
        </div>
      </div>
    );
  }

  // 检查具体权限
  if (requiredPermissions.length > 0) {
    const hasAllPermissions = requiredPermissions.every(permission => hasPermission(permission));

    if (!hasAllPermissions) {
      return (
        <div className="flex items-center justify-center min-h-screen">
          <div className="text-center">
            <h2 className="text-2xl font-bold text-gray-900 mb-4">权限不足</h2>
            <p className="text-gray-600 mb-6">您没有执行此操作的权限</p>
            <button
              onClick={e => {
                e.preventDefault();
                setTimeout(() => router.back(), 0);
              }}
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
            >
              返回上一页
            </button>
          </div>
        </div>
      );
    }
  }

  // 通过所有检查，渲染子组件
  return <>{children}</>;
};

/**
 * 权限检查组件
 * 用于在组件内部进行权限控制
 */
interface PermissionGuardProps {
  children: React.ReactNode;
  permissions?: string[];
  role?: string;
  fallback?: React.ReactNode;
  requireAll?: boolean; // 是否需要所有权限，默认为 true
}

export const PermissionGuard: React.FC<PermissionGuardProps> = ({
  children,
  permissions = [],
  role,
  fallback = null,
  requireAll = true,
}) => {
  const { hasPermission, hasRole } = useAuthStore();

  // 检查角色
  if (role && !hasRole(role)) {
    return <>{fallback}</>;
  }

  // 检查权限
  if (permissions.length > 0) {
    const checkPermissions = requireAll
      ? permissions.every(permission => hasPermission(permission))
      : permissions.some(permission => hasPermission(permission));

    if (!checkPermissions) {
      return <>{fallback}</>;
    }
  }

  return <>{children}</>;
};

/**
 * 角色检查组件
 * 用于基于角色显示不同内容
 */
interface RoleGuardProps {
  children: React.ReactNode;
  roles: string[];
  fallback?: React.ReactNode;
  requireAll?: boolean;
}

export const RoleGuard: React.FC<RoleGuardProps> = ({
  children,
  roles,
  fallback = null,
  requireAll = false,
}) => {
  const { hasRole } = useAuthStore();

  const checkRoles = requireAll
    ? roles.every(role => hasRole(role))
    : roles.some(role => hasRole(role));

  if (!checkRoles) {
    return <>{fallback}</>;
  }

  return <>{children}</>;
};

/**
 * 高阶组件：为组件添加路由守卫
 */
export function withRouteGuard<P extends object>(
  Component: React.ComponentType<P>,
  options: {
    requireAuth?: boolean;
    requiredPermissions?: string[];
    requiredRole?: string;
  } = {}
) {
  const WrappedComponent = (props: P) => {
    return (
      <AuthGuard
        requireAuth={options.requireAuth}
        requiredPermissions={options.requiredPermissions}
        requiredRole={options.requiredRole}
      >
        <Component {...props} />
      </AuthGuard>
    );
  };

  WrappedComponent.displayName = `withRouteGuard(${Component.displayName || Component.name})`;

  return WrappedComponent;
}
