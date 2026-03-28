/**
 * 统一的 Token / Tenant 存储读写与兼容迁移层
 *
 * 安全说明：
 * - Access token 和 Refresh token 现在存储在 httpOnly cookies 中（由后端设置）
 * - 前端无法直接读取 token 值，只能通过检查 cookie 是否存在来判断认证状态
 * - Tenant 信息仍然存储在 localStorage 中（非敏感数据）
 *
 * 规范键名：
 * - tenant code:   current_tenant_code
 * - tenant id:     current_tenant_id
 */
export const STORAGE_KEYS = {
  // Token 现在存储在 httpOnly cookies 中，以下常量仅用于兼容性检查
  // 不再用于 localStorage 存储
  ACCESS_TOKEN: 'access_token',
  REFRESH_TOKEN: 'refresh_token',

  // Tenant 信息仍然存储在 localStorage
  TENANT_CODE: 'current_tenant_code',
  TENANT_ID: 'current_tenant_id',

  // legacy keys
  LEGACY_TENANT_CODE: 'tenantCode',
  LEGACY_AUTH_TOKEN: 'auth_token',
  LEGACY_ITSM_TOKEN: 'itsm_token',
  LEGACY_TOKEN: 'token',
} as const;

// Token 现在存储在 httpOnly cookies 中，无法从 JavaScript 读取
// 使用此函数检查 token cookie 是否存在（用于判断是否已登录）
function hasTokenCookie(): boolean {
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

  // 清理旧的 token 键名（token 现在在 httpOnly cookie 中）
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
 * 通过检查 httpOnly cookie 是否存在来判断
 */
export function isAuthenticated(): boolean {
  migrateLegacyAuthStorage();
  return hasTokenCookie();
}

/**
 * 获取 Access Token
 * 注意：由于 token 现在存储在 httpOnly cookie 中，无法从 JavaScript 直接读取
 * 此函数仅用于向后兼容，实际返回 null
 * @deprecated Use httpOnly cookies instead
 */
export function getAccessToken(): string | null {
  migrateLegacyAuthStorage();
  // Token 在 httpOnly cookie 中，JavaScript 无法读取
  // 但我们保留此函数以避免破坏依赖它的代码
  return hasTokenCookie() ? 'httpOnly' : null;
}

/**
 * 获取 Refresh Token
 * @deprecated Use httpOnly cookies instead
 */
export function getRefreshToken(): string | null {
  migrateLegacyAuthStorage();
  // Token 在 httpOnly cookie 中，JavaScript 无法读取
  return null;
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
  // 不再存储 token - 它在 httpOnly cookie 中
  // 仅保留此函数以避免破坏依赖它的代码
}

export function setRefreshToken(token: string): void {
  // 不再存储 token - 它在 httpOnly cookie 中
}

export function clearAuthStorage(): void {
  // 只清理租户信息，token 通过 Set-Cookie: max-age=0 由后端清除
  safeRemove(STORAGE_KEYS.TENANT_ID);
  safeRemove(STORAGE_KEYS.TENANT_CODE);
  safeRemove(STORAGE_KEYS.LEGACY_AUTH_TOKEN);
  safeRemove(STORAGE_KEYS.LEGACY_ITSM_TOKEN);
  safeRemove(STORAGE_KEYS.LEGACY_TOKEN);
  safeRemove(STORAGE_KEYS.LEGACY_TENANT_CODE);
}
