'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '../../app/lib/store';
import { usePermissions, useRoutePermissions } from '@/lib/hooks/use-permissions';
import { RoutePermission } from '@/lib/router/route-config';

interface AuthGuardProps {
  children: React.ReactNode;
  requiredPermissions?: RoutePermission[];
  requiredRoles?: string[];
  fallbackPath?: string;
  showFallback?: boolean;
}

/**
 * 认证守卫组件
 * 用于保护需要认证的路由和组件
 */
export const AuthGuard: React.FC<AuthGuardProps> = ({
  children,
  requiredPermissions,
  requiredRoles,
  fallbackPath = '/login',
  showFallback = false,
}) => {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const { checkRoutePermission } = useRoutePermissions();

  // 检查是否已认证
  if (!isAuthenticated) {
    if (showFallback) {
      return <LoginRequired onLogin={() => router.push(fallbackPath)} />;
    }
    router.push(fallbackPath);
    return null;
  }

  // 检查权限
  if (!checkRoutePermission(requiredPermissions, requiredRoles)) {
    if (showFallback) {
      return <AccessDenied />;
    }
    router.push('/unauthorized');
    return null;
  }

  return <>{children}</>;
};

/**
 * 权限守卫组件
 * 仅检查权限，不检查认证状态
 */
interface PermissionGuardProps {
  children: React.ReactNode;
  requiredPermissions?: RoutePermission[];
  requiredRoles?: string[];
  fallback?: React.ReactNode;
  showFallback?: boolean;
}

export const PermissionGuard: React.FC<PermissionGuardProps> = ({
  children,
  requiredPermissions,
  requiredRoles,
  fallback,
  showFallback = true,
}) => {
  const { checkRoutePermission } = useRoutePermissions();

  // 检查权限
  if (!checkRoutePermission(requiredPermissions, requiredRoles)) {
    if (showFallback) {
      return fallback || <AccessDenied />;
    }
    return null;
  }

  return <>{children}</>;
};

/**
 * 角色守卫组件
 * 仅检查角色权限
 */
interface RoleGuardProps {
  children: React.ReactNode;
  requiredRoles: string[];
  requireAll?: boolean; // 是否需要所有角色
  fallback?: React.ReactNode;
  showFallback?: boolean;
}

export const RoleGuard: React.FC<RoleGuardProps> = ({
  children,
  requiredRoles,
  requireAll = false,
  fallback,
  showFallback = true,
}) => {
  const { hasAnyRole, hasAllRoles } = usePermissions();

  // 检查角色权限
  const hasAccess = requireAll 
    ? hasAllRoles(requiredRoles)
    : hasAnyRole(requiredRoles);

  if (!hasAccess) {
    if (showFallback) {
      return fallback || <AccessDenied />;
    }
    return null;
  }

  return <>{children}</>;
};

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
  const { hasPermission } = usePermissions();

  if (!hasPermission(resource, action)) {
    if (showFallback && fallback) {
      return <>{fallback}</>;
    }
    return null;
  }

  return <>{children}</>;
};

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
  const { isAdmin } = usePermissions();

  if (!isAdmin()) {
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
  const { isSuperAdmin } = usePermissions();

  if (!isSuperAdmin()) {
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
        <svg
          className="w-8 h-8 text-red-600"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"
          />
        </svg>
      </div>
      <h2 className="text-xl font-semibold text-gray-900 mb-2">访问被拒绝</h2>
      <p className="text-gray-600 mb-4">
        您没有权限访问此页面或执行此操作。
      </p>
      <button
        onClick={() => window.history.back()}
        className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
      >
        返回上一页
      </button>
    </div>
  </div>
);

// 登录要求组件
interface LoginRequiredProps {
  onLogin: () => void;
}

const LoginRequired: React.FC<LoginRequiredProps> = ({ onLogin }) => (
  <div className="min-h-screen flex items-center justify-center bg-gray-50">
    <div className="max-w-md w-full bg-white shadow-lg rounded-lg p-6 text-center">
      <div className="w-16 h-16 mx-auto mb-4 bg-blue-100 rounded-full flex items-center justify-center">
        <svg
          className="w-8 h-8 text-blue-600"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
          />
        </svg>
      </div>
      <h2 className="text-xl font-semibold text-gray-900 mb-2">需要登录</h2>
      <p className="text-gray-600 mb-4">
        请先登录以访问此页面。
      </p>
      <button
        onClick={onLogin}
        className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
      >
        前往登录
      </button>
    </div>
  </div>
);

// 权限检查工具函数
export const withAuth = <P extends object>(
  Component: React.ComponentType<P>,
  requiredPermissions?: RoutePermission[],
  requiredRoles?: string[]
) => {
  const AuthenticatedComponent: React.FC<P> = (props) => (
    <AuthGuard
      requiredPermissions={requiredPermissions}
      requiredRoles={requiredRoles}
    >
      <Component {...props} />
    </AuthGuard>
  );

  AuthenticatedComponent.displayName = `withAuth(${Component.displayName || Component.name})`;
  return AuthenticatedComponent;
};

// 权限检查高阶组件
export const withPermission = <P extends object>(
  Component: React.ComponentType<P>,
  requiredPermissions?: RoutePermission[],
  requiredRoles?: string[]
) => {
  const PermissionComponent: React.FC<P> = (props) => (
    <PermissionGuard
      requiredPermissions={requiredPermissions}
      requiredRoles={requiredRoles}
    >
      <Component {...props} />
    </PermissionGuard>
  );

  PermissionComponent.displayName = `withPermission(${Component.displayName || Component.name})`;
  return PermissionComponent;
};

// 角色检查高阶组件
export const withRole = <P extends object>(
  Component: React.ComponentType<P>,
  requiredRoles: string[],
  requireAll = false
) => {
  const RoleComponent: React.FC<P> = (props) => (
    <RoleGuard
      requiredRoles={requiredRoles}
      requireAll={requireAll}
    >
      <Component {...props} />
    </RoleGuard>
  );

  RoleComponent.displayName = `withRole(${Component.displayName || Component.name})`;
  return RoleComponent;
};

// 管理员检查高阶组件
export const withAdmin = <P extends object>(
  Component: React.ComponentType<P>
) => {
  const AdminComponent: React.FC<P> = (props) => (
    <AdminGuard>
      <Component {...props} />
    </AdminGuard>
  );

  AdminComponent.displayName = `withAdmin(${Component.displayName || Component.name})`;
  return AdminComponent;
};
