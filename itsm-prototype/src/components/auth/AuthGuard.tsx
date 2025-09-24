'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/lib/store/auth-store';
import { AuthService } from '@/lib/auth/auth-service';
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
        // 如果已经认证，跳过初始化
        if (isAuthenticated && user) {
          setIsInitializing(false);
          return;
        }

        // 尝试从本地存储恢复认证状态
        await AuthService.initialize();
      } catch (error) {
        console.error('Auth initialization failed:', error);
      } finally {
        setIsInitializing(false);
      }
    };

    initializeAuth();
  }, [isAuthenticated, user]);

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

  // 需要认证但未登录
  if (!isAuthenticated || !user) {
    if (fallback) {
      return <>{fallback}</>;
    }
    
    // 重定向到登录页
    router.push(redirectTo);
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
            onClick={() => router.back()}
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
    const hasAllPermissions = requiredPermissions.every((permission) =>
      hasPermission(permission)
    );

    if (!hasAllPermissions) {
      return (
        <div className="flex items-center justify-center min-h-screen">
          <div className="text-center">
            <h2 className="text-2xl font-bold text-gray-900 mb-4">权限不足</h2>
            <p className="text-gray-600 mb-6">您没有执行此操作的权限</p>
            <button
              onClick={() => router.back()}
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
      ? permissions.every((permission) => hasPermission(permission))
      : permissions.some((permission) => hasPermission(permission));

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
    ? roles.every((role) => hasRole(role))
    : roles.some((role) => hasRole(role));

  if (!checkRoles) {
    return <>{fallback}</>;
  }

  return <>{children}</>;
};