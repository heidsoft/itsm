/**
 * @deprecated 此文件已被弃用
 * 请使用 ./unified-auth-store.ts
 * 为了向后兼容，此文件重新导出统一的 store
 */

// 重新导出统一的 store
export {
  useAuthStore,
  useTenantStore,
  usePermissions,
  PERMISSIONS,
  ROLES,
  type User,
  type Tenant,
  type AuthState,
  type TenantState,
} from './unified-auth-store';