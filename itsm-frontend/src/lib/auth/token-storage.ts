/**
 * 统一的 Token / Tenant 存储读写与兼容迁移层
 *
 * 规范键名：
 * - access token:  access_token
 * - refresh token: refresh_token
 * - tenant code:   current_tenant_code
 * - tenant id:     current_tenant_id
 *
 * 兼容旧键名（只读并迁移到新键名）：
 * - auth_token / itsm_token / token
 * - tenantCode
 */
export const STORAGE_KEYS = {
  ACCESS_TOKEN: 'access_token',
  REFRESH_TOKEN: 'refresh_token',
  TENANT_CODE: 'current_tenant_code',
  TENANT_ID: 'current_tenant_id',

  // legacy keys
  LEGACY_AUTH_TOKEN: 'auth_token',
  LEGACY_ITSM_TOKEN: 'itsm_token',
  LEGACY_TOKEN: 'token',
  LEGACY_TENANT_CODE: 'tenantCode',
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
 * 在不破坏现有登录态的前提下，把旧键名迁移到新键名。
 * - 仅当新键名缺失时才迁移
 * - 迁移后会清理旧键名，避免多处读取产生“看似登录但接口不带token”的分裂态
 */
export function migrateLegacyAuthStorage(): void {
  if (typeof window === 'undefined') return;

  const currentAccess = safeGet(STORAGE_KEYS.ACCESS_TOKEN);
  if (!currentAccess) {
    const legacy =
      safeGet(STORAGE_KEYS.LEGACY_AUTH_TOKEN) ||
      safeGet(STORAGE_KEYS.LEGACY_ITSM_TOKEN) ||
      safeGet(STORAGE_KEYS.LEGACY_TOKEN);
    if (legacy) {
      safeSet(STORAGE_KEYS.ACCESS_TOKEN, legacy);
      safeRemove(STORAGE_KEYS.LEGACY_AUTH_TOKEN);
      safeRemove(STORAGE_KEYS.LEGACY_ITSM_TOKEN);
      safeRemove(STORAGE_KEYS.LEGACY_TOKEN);
    }
  }

  const currentTenantCode = safeGet(STORAGE_KEYS.TENANT_CODE);
  if (!currentTenantCode) {
    const legacyTenantCode = safeGet(STORAGE_KEYS.LEGACY_TENANT_CODE);
    if (legacyTenantCode) {
      safeSet(STORAGE_KEYS.TENANT_CODE, legacyTenantCode);
      safeRemove(STORAGE_KEYS.LEGACY_TENANT_CODE);
    }
  }
}

export function getAccessToken(): string | null {
  migrateLegacyAuthStorage();
  return safeGet(STORAGE_KEYS.ACCESS_TOKEN);
}

export function getRefreshToken(): string | null {
  migrateLegacyAuthStorage();
  return safeGet(STORAGE_KEYS.REFRESH_TOKEN);
}

export function getTenantCode(): string | null {
  migrateLegacyAuthStorage();
  return safeGet(STORAGE_KEYS.TENANT_CODE);
}

export function getTenantId(): string | null {
  migrateLegacyAuthStorage();
  return safeGet(STORAGE_KEYS.TENANT_ID);
}

export function setAccessToken(token: string): void {
  safeSet(STORAGE_KEYS.ACCESS_TOKEN, token);
}

export function setRefreshToken(token: string): void {
  safeSet(STORAGE_KEYS.REFRESH_TOKEN, token);
}

export function clearAuthStorage(): void {
  safeRemove(STORAGE_KEYS.ACCESS_TOKEN);
  safeRemove(STORAGE_KEYS.REFRESH_TOKEN);
  safeRemove(STORAGE_KEYS.TENANT_ID);
  safeRemove(STORAGE_KEYS.TENANT_CODE);
  // 清理历史键名，避免残留
  safeRemove(STORAGE_KEYS.LEGACY_AUTH_TOKEN);
  safeRemove(STORAGE_KEYS.LEGACY_ITSM_TOKEN);
  safeRemove(STORAGE_KEYS.LEGACY_TOKEN);
  safeRemove(STORAGE_KEYS.LEGACY_TENANT_CODE);
}
// End of file
