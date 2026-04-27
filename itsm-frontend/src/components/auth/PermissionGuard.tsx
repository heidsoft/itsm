'use client';

import React from 'react';
import { usePermissions } from '@/lib/hooks/use-permissions';
import type { Permission } from '@/lib/permissions';

/**
 * 权限守卫组件属性
 */
interface PermissionGuardProps {
  children: React.ReactNode;
  /** 权限检查（AND - 需要全部拥有） */
  permissions: Permission[];
  /** 无权限时显示的内容 */
  fallback?: React.ReactNode;
}

/**
 * 权限守卫组件
 *
 * @example
 * // 单个权限检查
 * <PermissionGuard permissions={[{ resource: 'ticket', action: 'create' }]}>
 *   <Button>创建工单</Button>
 * </PermissionGuard>
 *
 * @example
 * // 多个权限检查（AND - 需要全部拥有）
 * <PermissionGuard permissions={[
 *   { resource: 'ticket', action: 'read' },
 *   { resource: 'ticket', action: 'update' }
 * ]}>
 *   <Button>编辑工单</Button>
 * </PermissionGuard>
 */
export const PermissionGuard: React.FC<PermissionGuardProps> = ({
  children,
  permissions,
  fallback = null,
}) => {
  const { hasPermission } = usePermissions();

  // Check if user has ALL the permissions (AND logic)
  const hasAllPermissions = permissions.every(p => hasPermission(p.resource, p.action));

  if (!hasAllPermissions) {
    return <>{fallback}</>;
  }

  return <>{children}</>;
};

/**
 * 角色守卫组件属性
 */
interface RoleGuardProps {
  children: React.ReactNode;
  /** 允许的角色列表（OR - 拥有任意一个即可） */
  roles: string[];
  /** 无权限时显示的内容 */
  fallback?: React.ReactNode;
}

/**
 * 角色守卫组件
 *
 * @example
 * <RoleGuard roles={['admin', 'manager']}>
 *   <Button>管理员操作</Button>
 * </RoleGuard>
 */
export const RoleGuard: React.FC<RoleGuardProps> = ({ children, roles, fallback = null }) => {
  const { hasRole } = usePermissions();

  if (!roles.some(r => hasRole(r))) {
    return <>{fallback}</>;
  }

  return <>{children}</>;
};

/**
 * 管理员守卫组件属性
 */
interface AdminGuardProps {
  children: React.ReactNode;
  /** 无权限时显示的内容 */
  fallback?: React.ReactNode;
}

/**
 * 管理员守卫组件
 *
 * @example
 * <AdminGuard>
 *   <Button>管理员操作</Button>
 * </AdminGuard>
 */
export const AdminGuard: React.FC<AdminGuardProps> = ({ children, fallback = null }) => {
  const { isAdmin } = usePermissions();

  if (!isAdmin()) {
    return <>{fallback}</>;
  }

  return <>{children}</>;
};

/**
 * 任意权限守卫组件属性
 */
interface AnyPermissionGuardProps {
  children: React.ReactNode;
  /** 权限检查（OR - 拥有任意一个即可） */
  permissions: Permission[];
  /** 无权限时显示的内容 */
  fallback?: React.ReactNode;
}

/**
 * 任意权限守卫组件
 *
 * @example
 * <AnyPermissionGuard permissions={[
 *   { resource: 'ticket', action: 'delete' },
 *   { resource: 'ticket', action: 'admin' }
 * ]}>
 *   <Button danger>删除工单</Button>
 * </AnyPermissionGuard>
 */
export const AnyPermissionGuard: React.FC<AnyPermissionGuardProps> = ({
  children,
  permissions,
  fallback = null,
}) => {
  const { hasPermission } = usePermissions();

  // Check if user has ANY of the permissions (OR logic)
  const hasAnyPermission = permissions.some(p => hasPermission(p.resource, p.action));

  if (!hasAnyPermission) {
    return <>{fallback}</>;
  }

  return <>{children}</>;
};

export default PermissionGuard;
