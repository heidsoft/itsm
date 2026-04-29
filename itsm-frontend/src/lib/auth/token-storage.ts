/**
 * 统一的 Token / Tenant 存储读写与兼容迁移层
 *
 * 安全说明：
 * - Access token 和 Refresh token 存储在 httpOnly cookies 中（由后端设置）
 * - 前端无法直接读取 token 值，只能通过检查 cookie 是否存在来判断认证状态
 * - Tenant 信息存储在 localStorage 中（非敏感数据）
 *
 * 重要：Token 永远不存储在 localStorage 中！所有认证检查依赖 httpOnly cookie。
 *
 * 规范键名：
 * - tenant code:   current_tenant_code
 * - tenant id:     current_tenant_id
 */
export const STORAGE_KEYS = {
  // Token 存储在 httpOnly cookies，以下常量仅用于兼容性检查
  ACCESS_TOKEN: 'access_token',
  REFRESH_TOKEN: 'refresh_token',

  // Tenant 信息存储在 localStorage
  TENANT_CODE: 'current_tenant_code',
  TENANT_ID: 'current_tenant_id',

  // legacy keys
  LEGACY_TENANT_CODE: 'tenantCode',
  LEGACY_AUTH_TOKEN: 'auth_token',
  LEGACY_ITSM_TOKEN: 'itsm_token',
  LEGACY_TOKEN: 'token',
} as const;

// 检查 auth-token cookie 是否存在（唯一有效的认证状态检查）
function hasAuthCookie(): boolean {
  if (typeof document === 'undefined') return false;
  return document.cookie.split(';').some(cookie => {
    const [name] = cookie.trim().split('=');
    return name === 'auth-token';
  });
}

// 检查 access_token cookie 是否存在（兼容旧后端）
function hasAccessTokenCookie(): boolean {
  if (typeof document === 'undefined') return false;
  return document.cookie.split(';').some(cookie => {
    const [name] = cookie.trim().split('=');
    return name === 'access_token';
  });
}

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

  const currentAccessToken = safeGet(STORAGE_KEYS.ACCESS_TOKEN);
  if (!currentAccessToken) {
    const legacyAuthToken = safeGet(STORAGE_KEYS.LEGACY_AUTH_TOKEN);
    const legacyItsmToken = safeGet(STORAGE_KEYS.LEGACY_ITSM_TOKEN);
    const legacyToken = safeGet(STORAGE_KEYS.LEGACY_TOKEN);
    const legacy = legacyAuthToken || legacyItsmToken || legacyToken;
    if (legacy) {
      safeSet(STORAGE_KEYS.ACCESS_TOKEN, legacy);
    }
  }

  safeRemove(STORAGE_KEYS.LEGACY_AUTH_TOKEN);
  safeRemove(STORAGE_KEYS.LEGACY_ITSM_TOKEN);
  safeRemove(STORAGE_KEYS.LEGACY_TOKEN);

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

/**
 * 检查用户是否已认证
 * 只检查 httpOnly cookie，不读取 localStorage（localStorage 不存储 token）
 */
export function isAuthenticated(): boolean {
  migrateLegacyAuthStorage();
  return hasAuthCookie() || hasAccessTokenCookie() || !!getAccessToken();
}

/**
 * 获取 Access Token
 * 注意：Token 存储在 httpOnly cookie 中，JavaScript 无法读取。
 * 始终返回 null，前端依赖 cookie 进行认证。
 * @deprecated Token 在 httpOnly cookie 中，前端不应读取
 */
export function getAccessToken(): string | null {
  migrateLegacyAuthStorage();
  return safeGet(STORAGE_KEYS.ACCESS_TOKEN);
}

/**
 * 获取 Refresh Token
 * @deprecated Token 在 httpOnly cookie 中，前端无法读取
 */
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
  safeRemove(STORAGE_KEYS.LEGACY_AUTH_TOKEN);
  safeRemove(STORAGE_KEYS.LEGACY_ITSM_TOKEN);
  safeRemove(STORAGE_KEYS.LEGACY_TOKEN);
  safeRemove(STORAGE_KEYS.LEGACY_TENANT_CODE);
}
