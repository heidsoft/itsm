/**
 * 权限工具函数
 */

import type { Permission, PermissionString } from './types';

/**
 * 解析权限字符串列表为 Set（用于快速查找）
 */
export function parsePermissions(permissions: PermissionString[]): Set<string> {
  return new Set(permissions);
}

/**
 * 检查是否有通配符权限
 */
export function hasWildcardPermission(permissions: Set<string>): boolean {
  return permissions.has('*') || permissions.has('*:*');
}

/**
 * 检查单个权限是否匹配
 * 支持：
 * - 精确匹配: "ticket:read"
 * - 资源通配: "ticket:*" 匹配 ticket 的所有操作
 * - 全局通配: "*" 或 "*:*" 匹配所有权限
 */
export function matchesPermission(
  permissions: Set<string>,
  resource: string,
  action: string
): boolean {
  // 通配符匹配
  if (hasWildcardPermission(permissions)) return true;
  if (permissions.has(`${resource}:*`)) return true;
  if (permissions.has(`*:${action}`)) return true;

  // 精确匹配
  return permissions.has(`${resource}:${action}`);
}

/**
 * 批量检查权限（AND 关系 - 需要全部拥有）
 */
export function hasAllPermissions(
  permissions: Set<string>,
  required: Permission[]
): boolean {
  if (required.length === 0) return true;
  return required.every(p => matchesPermission(permissions, p.resource, p.action));
}

/**
 * 批量检查权限（OR 关系 - 拥有任意一个即可）
 */
export function hasAnyPermission(
  permissions: Set<string>,
  required: Permission[]
): boolean {
  if (required.length === 0) return false;
  return required.some(p => matchesPermission(permissions, p.resource, p.action));
}
