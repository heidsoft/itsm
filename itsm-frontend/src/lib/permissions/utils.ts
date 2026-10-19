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
 * - 动作通配: "*:read" 匹配所有资源的 read
 * - 资源管理员: "ticket:admin" 覆盖 ticket 下的具体业务动作
 * - 全局通配: "*" 或 "*:*" 匹配所有权限
 *
 * 分支顺序与后端 middleware/rbac.go 的 checkPermissionMatch 一一对应。
 * 前端少认一条规则，就会把一个后端允许的操作显示成不可用（租管持 ticket:admin
 * 却看不到指派按钮），所以两侧必须同集合。
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
  if (permissions.has(`${resource}:admin`)) return true;

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
