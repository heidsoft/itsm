'use client';

import React, { useEffect, useState } from 'react';
import { authService, User, PERMISSIONS, ROLES } from '../../auth';

// 认证守卫属性
export interface AuthGuardProps {
  children: React.ReactNode;
  fallback?: React.ReactNode;
  redirectTo?: string;
  requireAuth?: boolean;
  requiredRoles?: string[];
  requiredPermissions?: string[];
  requireAll?: boolean; // true: 需要所有权限, false: 需要任一权限
}

// 加载组件
const LoadingSpinner: React.FC = () => (
  <div className="min-h-screen flex items-center justify-center">
    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
  </div>
);

// 未认证组件
const UnauthorizedAccess: React.FC<{ message?: string; onLogin?: () => void }> = ({ 
  message = '您需要登录才能访问此页面', 
  onLogin 
}) => (
  <div className="min-h-screen flex items-center justify-center bg-gray-50">
    <div className="max-w-md w-full bg-white shadow-lg rounded-lg p-6 text-center">
      <div className="mb-4">
        <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
        </svg>
      </div>
      <h2 className="text-xl font-semibold text-gray-900 mb-2">访问受限</h2>
      <p className="text-gray-600 mb-4">{message}</p>
      {onLogin && (
        <button
          onClick={onLogin}
          className="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
        >
          立即登录
        </button>
      )}
    </div>
  </div>
);

// 权限不足组件
const InsufficientPermissions: React.FC<{ message?: string }> = ({ 
  message = '您没有足够的权限访问此页面' 
}) => (
  <div className="min-h-screen flex items-center justify-center bg-gray-50">
    <div className="max-w-md w-full bg-white shadow-lg rounded-lg p-6 text-center">
      <div className="mb-4">
        <svg className="mx-auto h-12 w-12 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728L5.636 5.636m12.728 12.728L18.364 5.636M5.636 18.364l12.728-12.728" />
        </svg>
      </div>
      <h2 className="text-xl font-semibold text-gray-900 mb-2">权限不足</h2>
      <p className="text-gray-600 mb-4">{message}</p>
      <button
        onClick={() => window.history.back()}
        className="w-full bg-gray-600 text-white py-2 px-4 rounded-md hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2"
      >
        返回上一页
      </button>
    </div>
  </div>
);

// 认证守卫组件
export const AuthGuard: React.FC<AuthGuardProps> = ({
  children,
  fallback,
  redirectTo = '/login',
  requireAuth = true,
  requiredRoles = [],
  requiredPermissions = [],
  requireAll = true,
}) => {
  const [authState, setAuthState] = useState({
    isAuthenticated: false,
    user: null as User | null,
    loading: true,
    error: null as string | null,
  });

  useEffect(() => {
    // 监听认证状态变化
    const unsubscribe = authService.addListener((state) => {
      setAuthState(state);
    });

    // 初始化认证状态
    setAuthState({
      isAuthenticated: authService.isAuthenticated(),
      user: authService.getUser(),
      loading: false,
      error: null,
    });

    return unsubscribe;
  }, []);

  // 处理登录跳转
  const handleLogin = () => {
    if (typeof window !== 'undefined') {
      window.location.href = redirectTo;
    }
  };

  // 检查角色权限
  const checkRolePermissions = (): boolean => {
    if (!authState.user) return false;

    // 检查角色
    if (requiredRoles.length > 0) {
      const hasRole = requireAll
        ? requiredRoles.every(role => authState.user!.roles.includes(role))
        : requiredRoles.some(role => authState.user!.roles.includes(role));
      
      if (!hasRole) return false;
    }

    // 检查权限
    if (requiredPermissions.length > 0) {
      const hasPermission = requireAll
        ? authService.hasAllPermissions(requiredPermissions)
        : authService.hasAnyPermission(requiredPermissions);
      
      if (!hasPermission) return false;
    }

    return true;
  };

  // 加载中状态
  if (authState.loading) {
    return fallback || <LoadingSpinner />;
  }

  // 不需要认证，直接渲染
  if (!requireAuth) {
    return <>{children}</>;
  }

  // 需要认证但未登录
  if (!authState.isAuthenticated) {
    return fallback || <UnauthorizedAccess onLogin={handleLogin} />;
  }

  // 已登录但权限不足
  if (!checkRolePermissions()) {
    const missingRoles = requiredRoles.filter(role => !authState.user!.roles.includes(role));
    const missingPermissions = requiredPermissions.filter(permission => !authService.hasPermission(permission));
    
    let message = '您没有足够的权限访问此页面。';
    if (missingRoles.length > 0) {
      message += `\n缺少角色: ${missingRoles.join(', ')}`;
    }
    if (missingPermissions.length > 0) {
      message += `\n缺少权限: ${missingPermissions.join(', ')}`;
    }

    return fallback || <InsufficientPermissions message={message} />;
  }

  // 通过所有检查，渲染子组件
  return <>{children}</>;
};

// 权限检查Hook
export const usePermissions = () => {
  const [authState, setAuthState] = useState({
    isAuthenticated: false,
    user: null as User | null,
    loading: true,
  });

  useEffect(() => {
    const unsubscribe = authService.addListener((state) => {
      setAuthState({
        isAuthenticated: state.isAuthenticated,
        user: state.user,
        loading: state.loading,
      });
    });

    setAuthState({
      isAuthenticated: authService.isAuthenticated(),
      user: authService.getUser(),
      loading: false,
    });

    return unsubscribe;
  }, []);

  return {
    ...authState,
    hasRole: (role: string) => authService.hasRole(role),
    hasAllPermissions: (permissions: string[]) => authService.hasAllPermissions(permissions),
    hasAnyPermission: (permissions: string[]) => authService.hasAnyPermission(permissions),
    checkPermission: (permission: string) => authService.checkPermission(permission),
  };
};

// 权限组件 - 根据权限条件渲染内容
export interface PermissionProps {
  children: React.ReactNode;
  fallback?: React.ReactNode;
  roles?: string[];
  permissions?: string[];
  requireAll?: boolean;
}

export const Permission: React.FC<PermissionProps> = ({
  children,
  fallback = null,
  roles = [],
  permissions = [],
  requireAll = true,
}) => {
  const { hasRole, hasAllPermissions, hasAnyPermission } = usePermissions();

  // 检查角色
  const hasRequiredRoles = roles.length === 0 || (
    requireAll
      ? roles.every(role => hasRole(role))
      : roles.some(role => hasRole(role))
  );

  // 检查权限
  const hasRequiredPermissions = permissions.length === 0 || (
    requireAll
      ? hasAllPermissions(permissions)
      : hasAnyPermission(permissions)
  );

  // 如果有任何条件不满足，显示fallback
  if (!hasRequiredRoles || !hasRequiredPermissions) {
    return <>{fallback}</>;
  }

  return <>{children}</>;
};

// 预定义的权限组件
export const AdminOnly: React.FC<{ children: React.ReactNode; fallback?: React.ReactNode }> = ({ 
  children, 
  fallback 
}) => (
  <Permission roles={[ROLES.ADMIN, ROLES.SUPER_ADMIN]} requireAll={false} fallback={fallback}>
    {children}
  </Permission>
);

export const ManagerOnly: React.FC<{ children: React.ReactNode; fallback?: React.ReactNode }> = ({ 
  children, 
  fallback 
}) => (
  <Permission roles={[ROLES.MANAGER, ROLES.ADMIN, ROLES.SUPER_ADMIN]} requireAll={false} fallback={fallback}>
    {children}
  </Permission>
);

export const AgentOnly: React.FC<{ children: React.ReactNode; fallback?: React.ReactNode }> = ({ 
  children, 
  fallback 
}) => (
  <Permission roles={[ROLES.AGENT, ROLES.MANAGER, ROLES.ADMIN, ROLES.SUPER_ADMIN]} requireAll={false} fallback={fallback}>
    {children}
  </Permission>
);

// 事件管理权限组件
export const CanViewIncidents: React.FC<{ children: React.ReactNode; fallback?: React.ReactNode }> = ({ 
  children, 
  fallback 
}) => (
  <Permission permissions={[PERMISSIONS.INCIDENT_VIEW]} fallback={fallback}>
    {children}
  </Permission>
);

export const CanCreateIncidents: React.FC<{ children: React.ReactNode; fallback?: React.ReactNode }> = ({ 
  children, 
  fallback 
}) => (
  <Permission permissions={[PERMISSIONS.INCIDENT_CREATE]} fallback={fallback}>
    {children}
  </Permission>
);

export const CanManageIncidents: React.FC<{ children: React.ReactNode; fallback?: React.ReactNode }> = ({ 
  children, 
  fallback 
}) => (
  <Permission 
    permissions={[PERMISSIONS.INCIDENT_UPDATE, PERMISSIONS.INCIDENT_DELETE, PERMISSIONS.INCIDENT_ASSIGN]} 
    requireAll={false} 
    fallback={fallback}
  >
    {children}
  </Permission>
);