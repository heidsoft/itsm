/**
 * 权限模块入口
 *
 * 码空间唯一来源：./catalog.gen.ts（由后端 internal/authz/catalog.go 经
 * cmd/authz-gen 生成，禁手改）。组件代码不得手写 "resource:action" 字符串，
 * 一律引用 PERM.* 常量；引用码空间不存在的码会在 TypeScript 编译期失败。
 */

export type { Permission, PermissionString } from './types';
export {
  parsePermissions,
  hasWildcardPermission,
  matchesPermission,
  hasAllPermissions,
  hasAnyPermission,
} from './utils';

export {
  PERMISSION_CATALOG,
  PERM,
} from './catalog.gen';
export type {
  PermissionCode,
  PermissionCatalogEntry,
} from './catalog.gen';
