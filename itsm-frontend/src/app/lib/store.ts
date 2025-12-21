/**
 * @deprecated 此文件已被弃用
 * 请使用 @/lib/store/unified-auth-store
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
} from '@/lib/store/unified-auth-store';

// 为了兼容性，也导出 Tenant 类型 from api-config
export type { Tenant as TenantType } from './api-config';