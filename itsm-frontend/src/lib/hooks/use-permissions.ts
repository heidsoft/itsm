import { useMemo } from 'react';
import { useAuthStore } from '../../app/lib/store';
import { RoutePermission } from '../router/route-config';

/**
 * 根据角色获取权限列表
 */
const getRolePermissions = (role: string): RoutePermission[] => {
  const rolePermissionMap: Record<string, RoutePermission[]> = {
    'super_admin': [
      { resource: 'ticket', action: 'read' },
      { resource: 'ticket', action: 'create' },
      { resource: 'ticket', action: 'update' },
      { resource: 'ticket', action: 'delete' },
      { resource: 'ticket', action: 'assign' },
      { resource: 'ticket', action: 'escalate' },
      { resource: 'ticket', action: 'resolve' },
      { resource: 'ticket', action: 'close' },
      { resource: 'ticket', action: 'reopen' },
      { resource: 'ticket', action: 'batch_delete' },
      { resource: 'ticket', action: 'export' },
      { resource: 'incident', action: 'read' },
      { resource: 'incident', action: 'create' },
      { resource: 'incident', action: 'update' },
      { resource: 'incident', action: 'delete' },
      { resource: 'incident', action: 'assign' },
      { resource: 'incident', action: 'escalate' },
      { resource: 'incident', action: 'resolve' },
      { resource: 'incident', action: 'close' },
      { resource: 'incident', action: 'declare_major' },
      { resource: 'problem', action: 'read' },
      { resource: 'problem', action: 'create' },
      { resource: 'problem', action: 'update' },
      { resource: 'problem', action: 'delete' },
      { resource: 'problem', action: 'assign' },
      { resource: 'problem', action: 'resolve' },
      { resource: 'problem', action: 'close' },
      { resource: 'change', action: 'read' },
      { resource: 'change', action: 'create' },
      { resource: 'change', action: 'update' },
      { resource: 'change', action: 'delete' },
      { resource: 'change', action: 'approve' },
      { resource: 'change', action: 'reject' },
      { resource: 'change', action: 'implement' },
      { resource: 'change', action: 'review' },
      { resource: 'knowledge', action: 'read' },
      { resource: 'knowledge', action: 'create' },
      { resource: 'knowledge', action: 'update' },
      { resource: 'knowledge', action: 'delete' },
      { resource: 'knowledge', action: 'publish' },
      { resource: 'knowledge', action: 'archive' },
      { resource: 'cmdb', action: 'read' },
      { resource: 'cmdb', action: 'create' },
      { resource: 'cmdb', action: 'update' },
      { resource: 'cmdb', action: 'delete' },
      { resource: 'cmdb', action: 'manage' },
      { resource: 'cmdb', action: 'import' },
      { resource: 'cmdb', action: 'export' },
      { resource: 'user', action: 'read' },
      { resource: 'user', action: 'create' },
      { resource: 'user', action: 'update' },
      { resource: 'user', action: 'delete' },
      { resource: 'user', action: 'manage' },
      { resource: 'user', action: 'reset_password' },
      { resource: 'role', action: 'read' },
      { resource: 'role', action: 'create' },
      { resource: 'role', action: 'update' },
      { resource: 'role', action: 'delete' },
      { resource: 'role', action: 'manage' },
      { resource: 'system', action: 'read' },
      { resource: 'system', action: 'update' },
      { resource: 'system', action: 'manage' },
      { resource: 'system', action: 'view_logs' },
      { resource: 'system', action: 'backup' },
      { resource: 'report', action: 'read' },
      { resource: 'report', action: 'export' },
      { resource: 'report', action: 'create' },
      { resource: 'report', action: 'schedule' },
    ],
    'admin': [
      { resource: 'ticket', action: 'read' },
      { resource: 'ticket', action: 'create' },
      { resource: 'ticket', action: 'update' },
      { resource: 'ticket', action: 'delete' },
      { resource: 'ticket', action: 'assign' },
      { resource: 'ticket', action: 'escalate' },
      { resource: 'ticket', action: 'resolve' },
      { resource: 'ticket', action: 'close' },
      { resource: 'ticket', action: 'reopen' },
      { resource: 'ticket', action: 'export' },
      { resource: 'incident', action: 'read' },
      { resource: 'incident', action: 'create' },
      { resource: 'incident', action: 'update' },
      { resource: 'incident', action: 'assign' },
      { resource: 'incident', action: 'escalate' },
      { resource: 'incident', action: 'resolve' },
      { resource: 'incident', action: 'close' },
      { resource: 'incident', action: 'declare_major' },
      { resource: 'problem', action: 'read' },
      { resource: 'problem', action: 'create' },
      { resource: 'problem', action: 'update' },
      { resource: 'problem', action: 'assign' },
      { resource: 'problem', action: 'resolve' },
      { resource: 'problem', action: 'close' },
      { resource: 'change', action: 'read' },
      { resource: 'change', action: 'create' },
      { resource: 'change', action: 'update' },
      { resource: 'change', action: 'approve' },
      { resource: 'change', action: 'reject' },
      { resource: 'change', action: 'implement' },
      { resource: 'change', action: 'review' },
      { resource: 'knowledge', action: 'read' },
      { resource: 'knowledge', action: 'create' },
      { resource: 'knowledge', action: 'update' },
      { resource: 'knowledge', action: 'publish' },
      { resource: 'cmdb', action: 'read' },
      { resource: 'cmdb', action: 'create' },
      { resource: 'cmdb', action: 'update' },
      { resource: 'cmdb', action: 'manage' },
      { resource: 'user', action: 'read' },
      { resource: 'user', action: 'create' },
      { resource: 'user', action: 'update' },
      { resource: 'user', action: 'manage' },
      { resource: 'report', action: 'read' },
      { resource: 'report', action: 'export' },
      { resource: 'report', action: 'create' },
    ],
    'manager': [
      { resource: 'ticket', action: 'read' },
      { resource: 'ticket', action: 'create' },
      { resource: 'ticket', action: 'update' },
      { resource: 'ticket', action: 'assign' },
      { resource: 'ticket', action: 'escalate' },
      { resource: 'ticket', action: 'resolve' },
      { resource: 'ticket', action: 'close' },
      { resource: 'ticket', action: 'export' },
      { resource: 'incident', action: 'read' },
      { resource: 'incident', action: 'create' },
      { resource: 'incident', action: 'update' },
      { resource: 'incident', action: 'assign' },
      { resource: 'incident', action: 'escalate' },
      { resource: 'incident', action: 'resolve' },
      { resource: 'incident', action: 'close' },
      { resource: 'problem', action: 'read' },
      { resource: 'problem', action: 'create' },
      { resource: 'problem', action: 'update' },
      { resource: 'problem', action: 'assign' },
      { resource: 'problem', action: 'resolve' },
      { resource: 'problem', action: 'close' },
      { resource: 'change', action: 'read' },
      { resource: 'change', action: 'create' },
      { resource: 'change', action: 'update' },
      { resource: 'change', action: 'approve' },
      { resource: 'change', action: 'review' },
      { resource: 'knowledge', action: 'read' },
      { resource: 'knowledge', action: 'create' },
      { resource: 'knowledge', action: 'update' },
      { resource: 'cmdb', action: 'read' },
      { resource: 'cmdb', action: 'create' },
      { resource: 'cmdb', action: 'update' },
      { resource: 'report', action: 'read' },
      { resource: 'report', action: 'export' },
    ],
    'agent': [
      { resource: 'ticket', action: 'read' },
      { resource: 'ticket', action: 'create' },
      { resource: 'ticket', action: 'update' },
      { resource: 'ticket', action: 'resolve' },
      { resource: 'ticket', action: 'close' },
      { resource: 'incident', action: 'read' },
      { resource: 'incident', action: 'create' },
      { resource: 'incident', action: 'update' },
      { resource: 'incident', action: 'resolve' },
      { resource: 'incident', action: 'close' },
      { resource: 'problem', action: 'read' },
      { resource: 'problem', action: 'create' },
      { resource: 'problem', action: 'update' },
      { resource: 'change', action: 'read' },
      { resource: 'change', action: 'create' },
      { resource: 'change', action: 'update' },
      { resource: 'knowledge', action: 'read' },
      { resource: 'knowledge', action: 'create' },
      { resource: 'knowledge', action: 'update' },
      { resource: 'cmdb', action: 'read' },
      { resource: 'report', action: 'read' },
    ],
    'user': [
      { resource: 'ticket', action: 'read' },
      { resource: 'ticket', action: 'create' },
      { resource: 'incident', action: 'read' },
      { resource: 'incident', action: 'create' },
      { resource: 'knowledge', action: 'read' },
      { resource: 'cmdb', action: 'read' },
    ],
  };

  return rolePermissionMap[role] || [];
};

/**
 * 权限管理Hook
 */
export const usePermissions = () => {
  const { user } = useAuthStore();

  // 用户权限列表 - 基于角色获取权限
  const userPermissions = useMemo<RoutePermission[]>(() => {
    if (!user?.role) return [];
    
    // 根据角色返回对应的权限
    const rolePermissions = getRolePermissions(user.role);
    return rolePermissions;
  }, [user?.role]);

  // 用户角色列表
  const userRoles = useMemo<string[]>(() => {
    if (!user?.role) return [];
    return [user.role];
  }, [user?.role]);

  /**
   * 检查是否有指定权限
   */
  const hasPermission = (resource: string, action: string): boolean => {
    return userPermissions.some(
      permission => permission.resource === resource && permission.action === action
    );
  };

  /**
   * 检查是否有指定角色
   */
  const hasRole = (roleName: string): boolean => {
    return userRoles.includes(roleName);
  };

  /**
   * 检查是否有任意一个角色
   */
  const hasAnyRole = (roleNames: string[]): boolean => {
    return roleNames.some(roleName => userRoles.includes(roleName));
  };

  /**
   * 检查是否有所有指定角色
   */
  const hasAllRoles = (roleNames: string[]): boolean => {
    return roleNames.every(roleName => userRoles.includes(roleName));
  };

  /**
   * 检查是否有访问路由的权限
   */
  const canAccessRoute = (routePermissions: RoutePermission[], requiredRoles?: string[]): boolean => {
    // 检查角色权限
    if (requiredRoles && requiredRoles.length > 0) {
      if (!hasAnyRole(requiredRoles)) {
        return false;
      }
    }

    // 检查资源权限
    if (routePermissions && routePermissions.length > 0) {
      return routePermissions.every(requiredPermission =>
        hasPermission(requiredPermission.resource, requiredPermission.action)
      );
    }

    return true;
  };

  /**
   * 检查是否是管理员
   */
  const isAdmin = (): boolean => {
    return hasAnyRole(['admin', 'super_admin']);
  };

  /**
   * 检查是否是超级管理员
   */
  const isSuperAdmin = (): boolean => {
    return hasRole('super_admin');
  };

  /**
   * 获取用户可执行的操作列表
   */
  const getAvailableActions = (resource: string): string[] => {
    return userPermissions
      .filter(permission => permission.resource === resource)
      .map(permission => permission.action);
  };

  /**
   * 检查是否可以执行批量操作
   */
  const canBatchOperate = (resource: string, action: string): boolean => {
    return hasPermission(resource, action) && hasPermission(resource, 'batch_' + action);
  };

  return {
    userPermissions,
    userRoles,
    hasPermission,
    hasRole,
    hasAnyRole,
    hasAllRoles,
    canAccessRoute,
    isAdmin,
    isSuperAdmin,
    getAvailableActions,
    canBatchOperate,
  };
};

/**
 * 路由权限Hook
 */
export const useRoutePermissions = () => {
  const { userPermissions, userRoles, canAccessRoute } = usePermissions();

  /**
   * 检查路由权限
   */
  const checkRoutePermission = (routePermissions?: RoutePermission[], requiredRoles?: string[]): boolean => {
    if (!routePermissions && !requiredRoles) {
      return true;
    }

    return canAccessRoute(routePermissions || [], requiredRoles);
  };

  return {
    userPermissions,
    userRoles,
    checkRoutePermission,
  };
};

/**
 * 操作权限Hook
 */
export const useOperationPermissions = () => {
  const { hasPermission, isAdmin } = usePermissions();

  // 工单操作权限
  const ticketPermissions = {
    canView: () => hasPermission('ticket', 'read'),
    canCreate: () => hasPermission('ticket', 'create'),
    canUpdate: () => hasPermission('ticket', 'update'),
    canDelete: () => hasPermission('ticket', 'delete'),
    canAssign: () => hasPermission('ticket', 'assign'),
    canEscalate: () => hasPermission('ticket', 'escalate'),
    canResolve: () => hasPermission('ticket', 'resolve'),
    canClose: () => hasPermission('ticket', 'close'),
    canReopen: () => hasPermission('ticket', 'reopen'),
    canBatchDelete: () => hasPermission('ticket', 'batch_delete'),
    canExport: () => hasPermission('ticket', 'export'),
  };

  // 事件操作权限
  const incidentPermissions = {
    canView: () => hasPermission('incident', 'read'),
    canCreate: () => hasPermission('incident', 'create'),
    canUpdate: () => hasPermission('incident', 'update'),
    canDelete: () => hasPermission('incident', 'delete'),
    canAssign: () => hasPermission('incident', 'assign'),
    canEscalate: () => hasPermission('incident', 'escalate'),
    canResolve: () => hasPermission('incident', 'resolve'),
    canClose: () => hasPermission('incident', 'close'),
    canDeclareAsMajor: () => hasPermission('incident', 'declare_major'),
  };

  // 问题操作权限
  const problemPermissions = {
    canView: () => hasPermission('problem', 'read'),
    canCreate: () => hasPermission('problem', 'create'),
    canUpdate: () => hasPermission('problem', 'update'),
    canDelete: () => hasPermission('problem', 'delete'),
    canAssign: () => hasPermission('problem', 'assign'),
    canResolve: () => hasPermission('problem', 'resolve'),
    canClose: () => hasPermission('problem', 'close'),
  };

  // 变更操作权限
  const changePermissions = {
    canView: () => hasPermission('change', 'read'),
    canCreate: () => hasPermission('change', 'create'),
    canUpdate: () => hasPermission('change', 'update'),
    canDelete: () => hasPermission('change', 'delete'),
    canApprove: () => hasPermission('change', 'approve'),
    canReject: () => hasPermission('change', 'reject'),
    canImplement: () => hasPermission('change', 'implement'),
    canReview: () => hasPermission('change', 'review'),
  };

  // 知识库操作权限
  const knowledgePermissions = {
    canView: () => hasPermission('knowledge', 'read'),
    canCreate: () => hasPermission('knowledge', 'create'),
    canUpdate: () => hasPermission('knowledge', 'update'),
    canDelete: () => hasPermission('knowledge', 'delete'),
    canPublish: () => hasPermission('knowledge', 'publish'),
    canArchive: () => hasPermission('knowledge', 'archive'),
  };

  // CMDB操作权限
  const cmdbPermissions = {
    canView: () => hasPermission('cmdb', 'read'),
    canCreate: () => hasPermission('cmdb', 'create'),
    canUpdate: () => hasPermission('cmdb', 'update'),
    canDelete: () => hasPermission('cmdb', 'delete'),
    canManage: () => hasPermission('cmdb', 'manage'),
    canImport: () => hasPermission('cmdb', 'import'),
    canExport: () => hasPermission('cmdb', 'export'),
  };

  // 用户管理权限
  const userPermissions = {
    canView: () => hasPermission('user', 'read') || isAdmin(),
    canCreate: () => hasPermission('user', 'create') || isAdmin(),
    canUpdate: () => hasPermission('user', 'update') || isAdmin(),
    canDelete: () => hasPermission('user', 'delete') || isAdmin(),
    canManage: () => hasPermission('user', 'manage') || isAdmin(),
    canResetPassword: () => hasPermission('user', 'reset_password') || isAdmin(),
  };

  // 角色管理权限
  const rolePermissions = {
    canView: () => hasPermission('role', 'read') || isAdmin(),
    canCreate: () => hasPermission('role', 'create') || isAdmin(),
    canUpdate: () => hasPermission('role', 'update') || isAdmin(),
    canDelete: () => hasPermission('role', 'delete') || isAdmin(),
    canManage: () => hasPermission('role', 'manage') || isAdmin(),
  };

  // 系统管理权限
  const systemPermissions = {
    canViewSettings: () => hasPermission('system', 'read') || isAdmin(),
    canUpdateSettings: () => hasPermission('system', 'update') || isAdmin(),
    canManage: () => hasPermission('system', 'manage') || isAdmin(),
    canViewLogs: () => hasPermission('system', 'view_logs') || isAdmin(),
    canBackup: () => hasPermission('system', 'backup') || isAdmin(),
  };

  // 报表权限
  const reportPermissions = {
    canView: () => hasPermission('report', 'read'),
    canExport: () => hasPermission('report', 'export'),
    canCreate: () => hasPermission('report', 'create'),
    canSchedule: () => hasPermission('report', 'schedule'),
  };

  return {
    ticket: ticketPermissions,
    incident: incidentPermissions,
    problem: problemPermissions,
    change: changePermissions,
    knowledge: knowledgePermissions,
    cmdb: cmdbPermissions,
    user: userPermissions,
    role: rolePermissions,
    system: systemPermissions,
    report: reportPermissions,
  };
};

/**
 * 权限守卫Hook
 */
export const usePermissionGuard = () => {
  const { hasPermission, hasRole, canAccessRoute } = usePermissions();

  /**
   * 权限守卫函数
   */
  const guard = (
    requiredPermissions?: RoutePermission[],
    requiredRoles?: string[],
    fallback?: () => void
  ) => {
    const hasAccess = canAccessRoute(requiredPermissions || [], requiredRoles);
    
    if (!hasAccess && fallback) {
      fallback();
    }
    
    return hasAccess;
  };

  /**
   * 操作权限守卫
   */
  const guardOperation = (
    resource: string,
    action: string,
    onDenied?: () => void
  ): boolean => {
    const hasAccess = hasPermission(resource, action);
    
    if (!hasAccess && onDenied) {
      onDenied();
    }
    
    return hasAccess;
  };

  /**
   * 角色权限守卫
   */
  const guardRole = (
    requiredRoles: string[],
    onDenied?: () => void
  ): boolean => {
    const hasAccess = requiredRoles.some(role => hasRole(role));
    
    if (!hasAccess && onDenied) {
      onDenied();
    }
    
    return hasAccess;
  };

  return {
    guard,
    guardOperation,
    guardRole,
  };
};

// 权限常量定义
export const PERMISSIONS = {
  // 工单权限
  TICKET: {
    READ: { resource: 'ticket', action: 'read' },
    CREATE: { resource: 'ticket', action: 'create' },
    UPDATE: { resource: 'ticket', action: 'update' },
    DELETE: { resource: 'ticket', action: 'delete' },
    ASSIGN: { resource: 'ticket', action: 'assign' },
    ESCALATE: { resource: 'ticket', action: 'escalate' },
    RESOLVE: { resource: 'ticket', action: 'resolve' },
    CLOSE: { resource: 'ticket', action: 'close' },
    REOPEN: { resource: 'ticket', action: 'reopen' },
    BATCH_DELETE: { resource: 'ticket', action: 'batch_delete' },
    EXPORT: { resource: 'ticket', action: 'export' },
  },
  
  // 事件权限
  INCIDENT: {
    READ: { resource: 'incident', action: 'read' },
    CREATE: { resource: 'incident', action: 'create' },
    UPDATE: { resource: 'incident', action: 'update' },
    DELETE: { resource: 'incident', action: 'delete' },
    ASSIGN: { resource: 'incident', action: 'assign' },
    ESCALATE: { resource: 'incident', action: 'escalate' },
    RESOLVE: { resource: 'incident', action: 'resolve' },
    CLOSE: { resource: 'incident', action: 'close' },
    DECLARE_MAJOR: { resource: 'incident', action: 'declare_major' },
  },
  
  // 问题权限
  PROBLEM: {
    READ: { resource: 'problem', action: 'read' },
    CREATE: { resource: 'problem', action: 'create' },
    UPDATE: { resource: 'problem', action: 'update' },
    DELETE: { resource: 'problem', action: 'delete' },
    ASSIGN: { resource: 'problem', action: 'assign' },
    RESOLVE: { resource: 'problem', action: 'resolve' },
    CLOSE: { resource: 'problem', action: 'close' },
  },
  
  // 变更权限
  CHANGE: {
    READ: { resource: 'change', action: 'read' },
    CREATE: { resource: 'change', action: 'create' },
    UPDATE: { resource: 'change', action: 'update' },
    DELETE: { resource: 'change', action: 'delete' },
    APPROVE: { resource: 'change', action: 'approve' },
    REJECT: { resource: 'change', action: 'reject' },
    IMPLEMENT: { resource: 'change', action: 'implement' },
    REVIEW: { resource: 'change', action: 'review' },
  },
  
  // 系统权限
  SYSTEM: {
    READ: { resource: 'system', action: 'read' },
    UPDATE: { resource: 'system', action: 'update' },
    MANAGE: { resource: 'system', action: 'manage' },
    VIEW_LOGS: { resource: 'system', action: 'view_logs' },
    BACKUP: { resource: 'system', action: 'backup' },
  },
} as const;

// 角色常量定义
export const ROLES = {
  SUPER_ADMIN: 'super_admin',
  ADMIN: 'admin',
  MANAGER: 'manager',
  AGENT: 'agent',
  USER: 'user',
} as const;