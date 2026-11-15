import { useMemo } from 'react';
import { useAuthStore } from '@/lib/store/auth-store';
import { matchesPermission } from '@/lib/permissions/utils';
import { PERM, type PermissionCode } from '@/lib/permissions';
import type { RoutePermission } from '../router/route-config';

/**
 * 权限管理 Hook
 *
 * 契约（2026-11-14 权限码单点化批次）：
 * - 后端下发的 user.permissions 是唯一事实来源，缺权限一律 fail-closed；
 * - hasPermission 只接受生成的权限码（PERM.* / PermissionCode），
 *   引用后端码空间不存在的码会在 TypeScript 编译期直接失败（幽灵码零容忍）；
 * - 码空间唯一来源：itsm-frontend/src/lib/permissions/catalog.gen.ts
 *   （由后端 internal/authz/catalog.go 经 cmd/authz-gen 生成，禁手改）。
 */
export const usePermissions = () => {
  const { user } = useAuthStore();

  // 后端返回的权限是唯一事实来源；无用户或无权限时保持 fail-closed。
  const userPermissions = useMemo<RoutePermission[]>(() => {
    if (!user?.permissions?.length) return [];

    return user.permissions.flatMap(permission => {
      const separatorIndex = permission.indexOf(':');
      if (separatorIndex <= 0 || separatorIndex === permission.length - 1) return [];

      return [{
        resource: permission.slice(0, separatorIndex),
        action: permission.slice(separatorIndex + 1),
      }];
    });
  }, [user?.permissions]);

  // 用户角色列表
  const userRoles = useMemo<string[]>(() => {
    if (!user?.role) return [];
    return [user.role];
  }, [user?.role]);

  // 判定共用 permissions/utils，与后端 checkPermissionMatch 同一套通配语义。
  const permissionSet = useMemo(
    () => new Set(user?.permissions ?? []),
    [user?.permissions],
  );

  /**
   * 检查是否有指定权限（接受生成的权限码，如 PERM.ticket.read）
   */
  const hasPermission = (code: PermissionCode): boolean => {
    const separatorIndex = code.indexOf(':');
    if (separatorIndex <= 0 || separatorIndex === code.length - 1) return false;
    return matchesPermission(
      permissionSet,
      code.slice(0, separatorIndex),
      code.slice(separatorIndex + 1),
    );
  };

  /**
   * 低层判定：权限 (resource, action) 由运行时动态拼出的场景使用
   * （例如批量操作菜单的权限元组）。静态权限码请一律用 hasPermission(PERM.x.y)。
   */
  const hasResourcePermission = (resource: string, action: string): boolean => {
    return matchesPermission(permissionSet, resource, action);
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
  const canAccessRoute = (
    routePermissions: RoutePermission[],
    requiredRoles?: string[]
  ): boolean => {
    // 检查角色权限
    if (requiredRoles && requiredRoles.length > 0) {
      if (!hasAnyRole(requiredRoles)) {
        return false;
      }
    }

    // 检查资源权限
    if (routePermissions && routePermissions.length > 0) {
      return routePermissions.every(requiredPermission =>
        hasResourcePermission(requiredPermission.resource, requiredPermission.action)
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
    if (!user?.permissions?.length) return [];

    const prefix = `${resource}:`;
    return user.permissions
      .filter(permission => permission.startsWith(prefix) && permission.length > prefix.length)
      .map(permission => permission.slice(prefix.length));
  };

  return {
    userPermissions,
    userRoles,
    hasPermission,
    hasResourcePermission,
    hasRole,
    hasAnyRole,
    hasAllRoles,
    canAccessRoute,
    isAdmin,
    isSuperAdmin,
    getAvailableActions,
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
  const checkRoutePermission = (
    routePermissions?: RoutePermission[],
    requiredRoles?: string[]
  ): boolean => {
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
 *
 * 口径对齐（与后端 router/ 的 RequirePermission 声明逐条核对，2026-11-14）：
 * - 后端 checkPermissionMatch 没有 write⇒create/update 派生，前端也不再做；
 * - 每个操作挂的是后端路由实际要求的那个码（例：reopen 路由要求
 *   ticket:close，resolve/close 路由要求 ticket:update），不再引用
 *   码空间不存在的臆造码（ticket:reopen / incident:declare_major 等）；
 * - 无后端路由支撑的「能力」（system:backup、report:schedule 等）直接删除，
 *   不保留永远为假的摆设判定。
 */
export const useOperationPermissions = () => {
  const { hasPermission } = usePermissions();

  // 工单操作权限（码 = 路由真实声明，见 router/ticket_routes.go）
  const ticketPermissions = {
    canView: () => hasPermission(PERM.ticket.read),
    canCreate: () => hasPermission(PERM.ticket.create),
    canUpdate: () => hasPermission(PERM.ticket.update),
    canDelete: () => hasPermission(PERM.ticket.delete),
    canAssign: () => hasPermission(PERM.ticket.assign),
    canEscalate: () => hasPermission(PERM.ticket.escalate),
    canApprove: () => hasPermission(PERM.ticket.approve),
    // resolve/close 有两套路由族：/tickets/:id/* 挂 ticket:update，
    // /tickets/workflow/* 挂 ticket:resolve / ticket:close —— 任一命中即可操作
    canResolve: () => hasPermission(PERM.ticket.update) || hasPermission(PERM.ticket.resolve),
    canClose: () => hasPermission(PERM.ticket.update) || hasPermission(PERM.ticket.close),
    // reopen 路由声明的码是 ticket:close
    canReopen: () => hasPermission(PERM.ticket.close),
    // batch-delete 路由声明的码是 ticket:delete
    canBatchDelete: () => hasPermission(PERM.ticket.delete),
    canExport: () => hasPermission(PERM.ticket.export),
  };

  // 事件操作权限（incident 域全部写路由统一挂 incident:write）
  const incidentPermissions = {
    canView: () => hasPermission(PERM.incident.read),
    canCreate: () => hasPermission(PERM.incident.write),
    canUpdate: () => hasPermission(PERM.incident.write),
    canDelete: () => hasPermission(PERM.incident.delete),
    canAssign: () => hasPermission(PERM.incident.write),
    canEscalate: () => hasPermission(PERM.incident.write),
    canResolve: () => hasPermission(PERM.incident.write),
    canClose: () => hasPermission(PERM.incident.write),
    canReopen: () => hasPermission(PERM.incident.write),
    canDeclareAsMajor: () => hasPermission(PERM.incident.write),
  };

  // 问题操作权限（problem 域写路由统一挂 problem:write）
  const problemPermissions = {
    canView: () => hasPermission(PERM.problem.read),
    canCreate: () => hasPermission(PERM.problem.write),
    canUpdate: () => hasPermission(PERM.problem.write),
    canDelete: () => hasPermission(PERM.problem.delete),
    canAssign: () => hasPermission(PERM.problem.write),
    canResolve: () => hasPermission(PERM.problem.write),
    canClose: () => hasPermission(PERM.problem.write),
  };

  // 变更操作权限（approve 路由同时承载批准与驳回）
  const changePermissions = {
    canView: () => hasPermission(PERM.change.read),
    canCreate: () => hasPermission(PERM.change.write),
    canUpdate: () => hasPermission(PERM.change.write),
    canDelete: () => hasPermission(PERM.change.delete),
    canApprove: () => hasPermission(PERM.change.approve),
    canReject: () => hasPermission(PERM.change.approve),
    canImplement: () => hasPermission(PERM.change.write),
    canReview: () => hasPermission(PERM.change.read),
  };

  // 知识库操作权限（publish/unpublish 路由挂 knowledge:write）
  const knowledgePermissions = {
    canView: () => hasPermission(PERM.knowledge.read),
    canCreate: () => hasPermission(PERM.knowledge.write),
    canUpdate: () => hasPermission(PERM.knowledge.write),
    canDelete: () => hasPermission(PERM.knowledge.delete),
    canPublish: () => hasPermission(PERM.knowledge.write),
    canArchive: () => hasPermission(PERM.knowledge.write),
  };

  // CMDB操作权限（import/export 的 POST 均挂 cmdb:write）
  const cmdbPermissions = {
    canView: () => hasPermission(PERM.cmdb.read),
    canCreate: () => hasPermission(PERM.cmdb.write),
    canUpdate: () => hasPermission(PERM.cmdb.write),
    canDelete: () => hasPermission(PERM.cmdb.delete),
    canManage: () => hasPermission(PERM.cmdb.write),
    canImport: () => hasPermission(PERM.cmdb.write),
    canExport: () => hasPermission(PERM.cmdb.write),
  };

  // 用户管理权限（CRUD/重置密码全部挂 user:write）
  const userPermissions = {
    canView: () => hasPermission(PERM.user.read),
    canCreate: () => hasPermission(PERM.user.write),
    canUpdate: () => hasPermission(PERM.user.write),
    canDelete: () => hasPermission(PERM.user.delete),
    canManage: () => hasPermission(PERM.user.write),
    canResetPassword: () => hasPermission(PERM.user.write),
  };

  // 角色管理权限（CRUD/授权全部挂 role:write）
  const rolePermissions = {
    canView: () => hasPermission(PERM.role.read),
    canCreate: () => hasPermission(PERM.role.write),
    canUpdate: () => hasPermission(PERM.role.write),
    canDelete: () => hasPermission(PERM.role.delete),
    canManage: () => hasPermission(PERM.role.write),
  };

  // 系统管理权限（设置页挂 system_config:*，操作日志挂 audit:read）
  const systemPermissions = {
    canViewSettings: () => hasPermission(PERM.system_config.read),
    canUpdateSettings: () => hasPermission(PERM.system_config.write),
    canManage: () => hasPermission(PERM.system_config.write),
    canViewLogs: () => hasPermission(PERM.audit.read),
  };

  // 报表权限（导出路由挂 report:read；create/schedule 无后端路由支撑，已删除）
  const reportPermissions = {
    canView: () => hasPermission(PERM.report.read),
    canExport: () => hasPermission(PERM.report.read),
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
   * 操作权限守卫（接受生成的权限码）
   */
  const guardOperation = (code: PermissionCode, onDenied?: () => void): boolean => {
    const hasAccess = hasPermission(code);

    if (!hasAccess && onDenied) {
      onDenied();
    }

    return hasAccess;
  };

  /**
   * 角色权限守卫
   */
  const guardRole = (requiredRoles: string[], onDenied?: () => void): boolean => {
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
