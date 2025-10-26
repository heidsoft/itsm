'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/lib/store/auth-store';
import { AuthService } from '@/lib/services/auth-service';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';

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
  const { isAuthenticated, user, isLoading, hasPermission, hasRole } = useAuthStore();
  const [isInitializing, setIsInitializing] = useState(true);

  useEffect(() => {
    const initializeAuth = async () => {
      try {
        // 检查localStorage中是否有认证信息
        const token = typeof window !== 'undefined' ? localStorage.getItem('auth_token') : null;

        if (token) {
          // 如果有token，尝试恢复状态
          const userInfo = typeof window !== 'undefined' ? localStorage.getItem('user_info') : null;
          if (userInfo) {
            try {
              const user = JSON.parse(userInfo);
              // 更新store状态
              const { login } = useAuthStore.getState();
              login(user, token, { id: 1, name: '默认租户', code: 'default' });
            } catch (e) {
              console.error('Failed to restore user info:', e);
            }
          }
        }

        setIsInitializing(false);
      } catch (error) {
        console.error('Auth initialization failed:', error);
        setIsInitializing(false);
      }
    };

    initializeAuth();
  }, [isAuthenticated, user]);

  // 处理未认证重定向
  useEffect(() => {
    if (requireAuth && !isAuthenticated && !fallback) {
      router.push(redirectTo);
    }
  }, [requireAuth, isAuthenticated, fallback, router, redirectTo]);

  // 正在初始化或加载中
  if (isInitializing || isLoading) {
    return (
      <div className='flex items-center justify-center min-h-screen'>
        <LoadingSpinner size='lg' />
      </div>
    );
  }

  // 不需要认证，直接渲染子组件
  if (!requireAuth) {
    return <>{children}</>;
  }

  if (requireAuth && (!isAuthenticated || !user)) {
    if (fallback) {
      return <>{fallback}</>;
    }

    // 重定向处理
    return (
      <div className='flex items-center justify-center min-h-screen'>
        <LoadingSpinner size='lg' />
      </div>
    );
  }

  // 检查角色权限
  if (requiredRole && !hasRole(requiredRole)) {
    return (
      <div className='flex items-center justify-center min-h-screen'>
        <div className='text-center'>
          <h2 className='text-2xl font-bold text-gray-900 mb-4'>访问被拒绝</h2>
          <p className='text-gray-600 mb-6'>您没有访问此页面的权限</p>
          <button
            onClick={e => {
              e.preventDefault();
              setTimeout(() => router.back(), 0);
            }}
            className='px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700'
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
        <div className='flex items-center justify-center min-h-screen'>
          <div className='text-center'>
            <h2 className='text-2xl font-bold text-gray-900 mb-4'>权限不足</h2>
            <p className='text-gray-600 mb-6'>您没有执行此操作的权限</p>
            <button
              onClick={e => {
                e.preventDefault();
                setTimeout(() => router.back(), 0);
              }}
              className='px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700'
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
