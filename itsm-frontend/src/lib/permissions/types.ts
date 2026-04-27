/**
 * 权限类型定义
 * resource 和 action 由后端定义，前端不限制
 */

/**
 * 权限对象
 */
export interface Permission {
  resource: string;
  action: string;
}

/**
 * 权限字符串格式
 * 格式: "resource:action"
 */
export type PermissionString = string;
