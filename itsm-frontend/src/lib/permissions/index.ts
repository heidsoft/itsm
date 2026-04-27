/**
 * 权限模块入口
 */

export type { Permission, PermissionString } from './types';
export {
  parsePermissions,
  hasWildcardPermission,
  matchesPermission,
  hasAllPermissions,
  hasAnyPermission,
} from './utils';
