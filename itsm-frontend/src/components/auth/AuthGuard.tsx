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
          // 分别获取用户信息和租户信息，避免一个失败导致整体失败
          let userResponse = null;
          let tenantsResponse = null;

          try {
            userResponse = await httpClient.get<{
              id: number;
              username: string;
              email: string;
              name: string;
              role: string;
              department: string;
              tenant_id: number;
            }>('/api/v1/auth/me');
          } catch (e) {
            console.error('Failed to fetch user info:', e);
          }

          try {
            tenantsResponse = await httpClient.get<{
              tenants: Array<{
                id: number;
                name: string;
                code: string;
                domain: string;
                type: string;
                status: string;
              }>;
            }>('/api/v1/auth/tenants');
          } catch (e) {
            console.error('Failed to fetch tenants:', e);
          }

          // 如果两个都失败，则认为未认证
          if (!userResponse && !tenantsResponse) {
            setIsInitializing(false);
            return;
          }

          const currentTenant = tenantsResponse?.tenants?.[0];
          const { login } = useAuthStore.getState();
          login(
            {
              id: userResponse?.id || 0,
              username: userResponse?.username || '',
              email: userResponse?.email || '',
              name: userResponse?.name || '',
              role: userResponse?.role || 'end_user',
              department: userResponse?.department,
            },
            'authenticated', // Token is in httpOnly cookie, not accessible here
            currentTenant
              ? {
                  id: currentTenant.id,
                  name: currentTenant.name,
                  code: currentTenant.code,
                  type: currentTenant.type as 'standard' | 'trial' | 'enterprise',
                  status: currentTenant.status as 'active' | 'suspended' | 'expired',
                  createdAt: new Date().toISOString(),
                  updatedAt: new Date().toISOString(),
                }
              : undefined
          );
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

/**
 * 管理员守卫组件
 * 仅管理员可访问
 */
interface AdminGuardProps {
  children: React.ReactNode;
  fallback?: React.ReactNode;
  showFallback?: boolean;
}

export const AdminGuard: React.FC<AdminGuardProps> = ({
  children,
  fallback,
  showFallback = true,
}) => {
  const { hasRole } = useAuthStore();

  if (!hasRole('admin') && !hasRole('super_admin')) {
    if (showFallback) {
      return fallback || <AccessDenied />;
    }
    return null;
  }

  return <>{children}</>;
};

/**
 * 超级管理员守卫组件
 * 仅超级管理员可访问
 */
interface SuperAdminGuardProps {
  children: React.ReactNode;
  fallback?: React.ReactNode;
  showFallback?: boolean;
}

export const SuperAdminGuard: React.FC<SuperAdminGuardProps> = ({
  children,
  fallback,
  showFallback = true,
}) => {
  const { hasRole } = useAuthStore();

  if (!hasRole('super_admin')) {
    if (showFallback) {
      return fallback || <AccessDenied />;
    }
    return null;
  }

  return <>{children}</>;
};

/**
 * 条件守卫组件
 * 根据自定义条件显示内容
 */
interface ConditionalGuardProps {
  children: React.ReactNode;
  condition: boolean;
  fallback?: React.ReactNode;
  showFallback?: boolean;
}

export const ConditionalGuard: React.FC<ConditionalGuardProps> = ({
  children,
  condition,
  fallback,
  showFallback = false,
}) => {
  if (!condition) {
    if (showFallback && fallback) {
      return <>{fallback}</>;
    }
    return null;
  }

  return <>{children}</>;
};

// 默认的访问拒绝组件
const AccessDenied: React.FC = () => (
  <div className="min-h-screen flex items-center justify-center bg-gray-50">
    <div className="max-w-md w-full bg-white shadow-lg rounded-lg p-6 text-center">
      <div className="w-16 h-16 mx-auto mb-4 bg-red-100 rounded-full flex items-center justify-center">
        <svg className="w-8 h-8 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"
          />
        </svg>
      </div>
      <h2 className="text-xl font-semibold text-gray-900 mb-2">访问被拒绝</h2>
      <p className="text-gray-600 mb-4">您没有权限访问此页面或执行此操作。</p>
      <button
        onClick={() => window.history.back()}
        className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
      >
        返回上一页
      </button>
    </div>
  </div>
);

/**
 * 操作权限守卫组件
 * 用于保护特定的操作按钮或功能
 */
interface OperationGuardProps {
  children: React.ReactNode;
  resource: string;
  action: string;
  fallback?: React.ReactNode;
  showFallback?: boolean;
}

export const OperationGuard: React.FC<OperationGuardProps> = ({
  children,
  resource,
  action,
  fallback,
  showFallback = false,
}) => {
  const { hasPermission } = useAuthStore();

  // Combine resource and action into permission string (e.g., "ticket:read")
  const permission = `${resource}:${action}`;
  if (!hasPermission(permission)) {
    if (showFallback && fallback) {
      return <>{fallback}</>;
    }
    return null;
  }

  return <>{children}</>;
};

/**
 * 高阶组件：为组件添加认证
 */
export const withAuth = <P extends object>(
  Component: React.ComponentType<P>,
  options: {
    requireAuth?: boolean;
    requiredPermissions?: string[];
    requiredRole?: string;
  } = {}
) => {
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

  WrappedComponent.displayName = `withAuth(${Component.displayName || Component.name})`;
  return WrappedComponent;
};

/**
 * 高阶组件：为组件添加权限检查
 */
export const withPermission = <P extends object>(
  Component: React.ComponentType<P>,
  permissions: string[] = [],
  requireAll = true
) => {
  const WrappedComponent = (props: P) => {
    return (
      <PermissionGuard permissions={permissions} requireAll={requireAll}>
        <Component {...props} />
      </PermissionGuard>
    );
  };

  WrappedComponent.displayName = `withPermission(${Component.displayName || Component.name})`;
  return WrappedComponent;
};

/**
 * 高阶组件：为组件添加角色检查
 */
export const withRole = <P extends object>(
  Component: React.ComponentType<P>,
  roles: string[],
  requireAll = false
) => {
  const WrappedComponent = (props: P) => {
    return (
      <RoleGuard roles={roles} requireAll={requireAll}>
        <Component {...props} />
      </RoleGuard>
    );
  };

  WrappedComponent.displayName = `withRole(${Component.displayName || Component.name})`;
  return WrappedComponent;
};

/**
 * 高阶组件：为组件添加管理员检查
 */
export const withAdmin = <P extends object>(Component: React.ComponentType<P>) => {
  const WrappedComponent = (props: P) => {
    return (
      <AdminGuard>
        <Component {...props} />
      </AdminGuard>
    );
  };

  WrappedComponent.displayName = `withAdmin(${Component.displayName || Component.name})`;
  return WrappedComponent;
};
