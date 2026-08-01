/**
 * Tests for src/lib/auth module
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

describe('token-storage', () => {
  let tokenStorage: typeof import('../token-storage');

  beforeEach(() => {
    jest.resetModules();
    localStorage.clear();
    // Clear cookies
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: '',
    });
  });

  it('isAuthenticated returns false when no auth cookie', async () => {
    tokenStorage = await import('../token-storage');
    expect(tokenStorage.isAuthenticated()).toBe(false);
  });

  it('isAuthenticated returns true when auth-token cookie exists', async () => {
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: 'auth-token=abc123; other=val',
    });
    tokenStorage = await import('../token-storage');
    expect(tokenStorage.isAuthenticated()).toBe(true);
  });

  it('isAuthenticated returns true when access_token cookie exists', async () => {
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: 'access_token=xyz',
    });
    tokenStorage = await import('../token-storage');
    expect(tokenStorage.isAuthenticated()).toBe(true);
  });

  it('getAccessToken always returns null (httpOnly cookie)', async () => {
    tokenStorage = await import('../token-storage');
    expect(tokenStorage.getAccessToken()).toBeNull();
  });

  it('getRefreshToken always returns null', async () => {
    tokenStorage = await import('../token-storage');
    expect(tokenStorage.getRefreshToken()).toBeNull();
  });

  it('setAccessToken is a no-op', async () => {
    tokenStorage = await import('../token-storage');
    tokenStorage.setAccessToken('token123');
    // no error, no storage
    expect(localStorage.getItem('access_token')).toBeNull();
  });

  it('setRefreshToken is a no-op', async () => {
    tokenStorage = await import('../token-storage');
    tokenStorage.setRefreshToken('refresh123');
    expect(localStorage.getItem('refresh_token')).toBeNull();
  });

  it('getTenantCode returns stored value', async () => {
    localStorage.setItem('current_tenant_code', 'TENANT_A');
    tokenStorage = await import('../token-storage');
    expect(tokenStorage.getTenantCode()).toBe('TENANT_A');
  });

  it('getTenantId returns stored value', async () => {
    localStorage.setItem('current_tenant_id', '42');
    tokenStorage = await import('../token-storage');
    expect(tokenStorage.getTenantId()).toBe('42');
  });

  it('migrateLegacyAuthStorage migrates tenantCode', async () => {
    localStorage.setItem('tenantCode', 'LEGACY_TENANT');
    tokenStorage = await import('../token-storage');
    tokenStorage.migrateLegacyAuthStorage();
    expect(localStorage.getItem('current_tenant_code')).toBe('LEGACY_TENANT');
    expect(localStorage.getItem('tenantCode')).toBeNull();
  });

  it('migrateLegacyAuthStorage does not overwrite existing', async () => {
    localStorage.setItem('current_tenant_code', 'EXISTING');
    localStorage.setItem('tenantCode', 'LEGACY');
    tokenStorage = await import('../token-storage');
    tokenStorage.migrateLegacyAuthStorage();
    expect(localStorage.getItem('current_tenant_code')).toBe('EXISTING');
  });

  it('clearAuthStorage removes all keys', async () => {
    localStorage.setItem('current_tenant_id', '1');
    localStorage.setItem('current_tenant_code', 'X');
    localStorage.setItem('auth_token', 'old');
    localStorage.setItem('itsm_token', 'old2');
    localStorage.setItem('token', 'old3');
    localStorage.setItem('tenantCode', 'old4');
    tokenStorage = await import('../token-storage');
    tokenStorage.clearAuthStorage();
    expect(localStorage.getItem('current_tenant_id')).toBeNull();
    expect(localStorage.getItem('current_tenant_code')).toBeNull();
    expect(localStorage.getItem('auth_token')).toBeNull();
    expect(localStorage.getItem('itsm_token')).toBeNull();
    expect(localStorage.getItem('token')).toBeNull();
    expect(localStorage.getItem('tenantCode')).toBeNull();
  });

  it('STORAGE_KEYS are exported correctly', async () => {
    tokenStorage = await import('../token-storage');
    expect(tokenStorage.STORAGE_KEYS.ACCESS_TOKEN).toBe('access_token');
    expect(tokenStorage.STORAGE_KEYS.TENANT_CODE).toBe('current_tenant_code');
  });
});

describe('tenant-context', () => {
  let tenantContext: typeof import('../tenant-context');

  beforeEach(() => {
    jest.resetModules();
  });

  it('initial state is null', async () => {
    tenantContext = await import('../tenant-context');
    expect(tenantContext.getTenantId()).toBeNull();
    expect(tenantContext.getTenantCode()).toBeNull();
  });

  it('setTenantId updates tenant id', async () => {
    tenantContext = await import('../tenant-context');
    tenantContext.setTenantId(5);
    expect(tenantContext.getTenantId()).toBe(5);
  });

  it('setTenantCode updates tenant code', async () => {
    tenantContext = await import('../tenant-context');
    tenantContext.setTenantCode('ABC');
    expect(tenantContext.getTenantCode()).toBe('ABC');
  });

  it('setTenant updates both', async () => {
    tenantContext = await import('../tenant-context');
    tenantContext.setTenant(10, 'COMPANY');
    expect(tenantContext.getTenantId()).toBe(10);
    expect(tenantContext.getTenantCode()).toBe('COMPANY');
  });

  it('clearTenant resets state', async () => {
    tenantContext = await import('../tenant-context');
    tenantContext.setTenant(10, 'COMPANY');
    tenantContext.clearTenant();
    expect(tenantContext.getTenantId()).toBeNull();
    expect(tenantContext.getTenantCode()).toBeNull();
  });

  it('getState returns full state', async () => {
    tenantContext = await import('../tenant-context');
    tenantContext.setTenant(7, 'X');
    const state = tenantContext.getState();
    expect(state.tenantId).toBe(7);
    expect(state.tenantCode).toBe('X');
  });

  it('subscribe notifies on changes', async () => {
    tenantContext = await import('../tenant-context');
    const fn = jest.fn();
    const unsub = tenantContext.subscribe(fn);
    tenantContext.setTenantId(99);
    expect(fn).toHaveBeenCalledWith(expect.objectContaining({ tenantId: 99 }));
    unsub();
    tenantContext.setTenantId(100);
    expect(fn).toHaveBeenCalledTimes(1);
  });
});

describe('mock-auth-service', () => {
  let mockAuth: typeof import('./mock-auth-service');

  beforeEach(() => {
    jest.resetModules();
  });

  it('login succeeds for admin', async () => {
    mockAuth = await import('./mock-auth-service');
    const service = mockAuth.createMockAuthService();
    const result = await service.login('admin', 'pass');
    expect(result.success).toBe(true);
    expect(result.user?.username).toBe('admin');
    expect(result.token).toBeDefined();
  });

  it('login succeeds for test user', async () => {
    mockAuth = await import('./mock-auth-service');
    const service = mockAuth.createMockAuthService();
    const result = await service.login('test', 'pass');
    expect(result.success).toBe(true);
  });

  it('login fails for unknown user', async () => {
    mockAuth = await import('./mock-auth-service');
    const service = mockAuth.createMockAuthService();
    const result = await service.login('unknown', 'pass');
    expect(result.success).toBe(false);
    expect(result.error).toBe('Invalid credentials');
  });

  it('logout clears current user', async () => {
    mockAuth = await import('./mock-auth-service');
    const service = mockAuth.createMockAuthService();
    await service.login('admin', 'pass');
    await service.logout();
    expect(service.getCurrentUser()).toBeNull();
  });

  it('validateToken returns auth status', async () => {
    mockAuth = await import('./mock-auth-service');
    const service = mockAuth.createMockAuthService();
    expect(await service.validateToken('any')).toBe(true);
    await service.logout();
    expect(await service.validateToken('any')).toBe(false);
  });

  it('getCurrentUser returns default user initially', async () => {
    mockAuth = await import('./mock-auth-service');
    const service = mockAuth.createMockAuthService();
    const user = service.getCurrentUser();
    expect(user).not.toBeNull();
    expect(user?.role).toBe('admin');
  });

  it('exports singleton MockAuthService', async () => {
    mockAuth = await import('./mock-auth-service');
    expect(mockAuth.MockAuthService).toBeDefined();
    expect(mockAuth.MockAuthService.login).toBeDefined();
  });
});
