/**
 * 本地存储的租户上下文与历史键名清理
 *
 * 这里只存放非敏感的租户标识；凭证一律由后端 httpOnly cookie 管理，JS 读不到也写不了。
 * 「我是否已登录」不在这个文件里判断——唯一来源是 lib/api/session-api.ts 的会话端点。
 *
 * 规范键名：
 * - tenant code:   current_tenant_code
 * - tenant id:     current_tenant_id
 */
export const STORAGE_KEYS = {
  // Tenant 信息存储在 localStorage
  TENANT_CODE: 'current_tenant_code',
  TENANT_ID: 'current_tenant_id',

  // legacy keys
  LEGACY_TENANT_CODE: 'tenantCode',
  LEGACY_AUTH_TOKEN: 'auth_token',
  LEGACY_ITSM_TOKEN: 'itsm_token',
  LEGACY_TOKEN: 'token',
} as const;

function safeGet(key: string): string | null {
  if (typeof window === 'undefined') return null;
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function safeSet(key: string, value: string): void {
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem(key, value);
  } catch {
    // ignore
  }
}

function safeRemove(key: string): void {
  if (typeof window === 'undefined') return;
  try {
    localStorage.removeItem(key);
  } catch {
    // ignore
  }
}

/**
 * 迁移旧键名到新键名
 * 只迁移租户信息，不涉及 token
 */
export function migrateLegacyAuthStorage(): void {
  if (typeof window === 'undefined') return;

  // 迁移租户代码
  const currentTenantCode = safeGet(STORAGE_KEYS.TENANT_CODE);
  if (!currentTenantCode) {
    const legacyTenantCode = safeGet(STORAGE_KEYS.LEGACY_TENANT_CODE);
    if (legacyTenantCode) {
      safeSet(STORAGE_KEYS.TENANT_CODE, legacyTenantCode);
      safeRemove(STORAGE_KEYS.LEGACY_TENANT_CODE);
    }
  }
}

export function getTenantCode(): string | null {
  migrateLegacyAuthStorage();
  return safeGet(STORAGE_KEYS.TENANT_CODE);
}

export function getTenantId(): string | null {
  migrateLegacyAuthStorage();
  return safeGet(STORAGE_KEYS.TENANT_ID);
}

export function clearAuthStorage(): void {
  safeRemove(STORAGE_KEYS.TENANT_ID);
  safeRemove(STORAGE_KEYS.TENANT_CODE);
  safeRemove(STORAGE_KEYS.LEGACY_AUTH_TOKEN);
  safeRemove(STORAGE_KEYS.LEGACY_ITSM_TOKEN);
  safeRemove(STORAGE_KEYS.LEGACY_TOKEN);
  safeRemove(STORAGE_KEYS.LEGACY_TENANT_CODE);
  // 清理 Zustand persist 的 auth-storage key（避免跨用户/跨租户残留 user 信息）
  safeRemove('auth-storage');
}
