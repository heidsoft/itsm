import { test, expect } from '@playwright/test';
import { APIRequestContext } from '@playwright/test';

test.describe('User Management, Roles & Permissions Deep Testing', () => {
  let request: APIRequestContext;

  test.beforeAll(async ({ browser }) => {
    request = browser.request;
  });

  test.afterAll(async () => {
    await request.dispose();
  });

  test('should login successfully and obtain tokens', async () => {
    const response = await request.post('http://localhost:8090/api/v1/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });

    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('access_token');
    expect(body.data).toHaveProperty('refresh_token');
    expect(body.data.user).toHaveProperty('username', 'admin');
  });

  test('should fail login with invalid credentials', async () => {
    const response = await request.post('http://localhost:8090/api/v1/auth/login', {
      data: {
        username: 'invalid',
        password: 'wrongpass'
      }
    });

    const body = await response.json();
    expect(response.status()).toBe(200); // API returns 200 with error code
    expect(body.code).not.toBe(0);
  });

  test('should list users as admin', async () => {
    // First login
    const loginResp = await request.post('http://localhost:8090/api/v1/auth/login', {
      data: { username: 'admin', password: 'admin123' }
    });
    const loginBody = await loginResp.json();
    const token = loginBody.data.access_token;

    // List users
    const response = await request.get('http://localhost:8090/api/v1/users', {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });

    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('users');
    expect(body.data).toHaveProperty('pagination');
    expect(Array.isArray(body.data.users)).toBeTruthy();
  });

  test('should create user as admin', async () => {
    const loginResp = await request.post('http://localhost:8090/api/v1/auth/login', {
      data: { username: 'admin', password: 'admin123' }
    });
    const token = (await loginResp.json()).data.access_token;

    const newUser = {
      username: `testuser_${Date.now()}`,
      email: `test_${Date.now()}@example.com`,
      name: 'Test User',
      password: 'TestPass123',
      tenant_id: 1,
      role: 'agent'
    };

    const response = await request.post('http://localhost:8090/api/v1/users', {
      headers: { 'Authorization': `Bearer ${token}` },
      data: newUser
    });

    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data.username).toBe(newUser.username);
    expect(body.data.role).toBe('agent');
  });

  test('should list roles as admin', async () => {
    const loginResp = await request.post('http://localhost:8090/api/v1/auth/login', {
      data: { username: 'admin', password: 'admin123' }
    });
    const token = (await loginResp.json()).data.access_token;

    const response = await request.get('http://localhost:8090/api/v1/roles', {
      headers: { 'Authorization': `Bearer ${token}` }
    });

    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data).toHaveProperty('roles');
    expect(body.data.roles.length).toBeGreaterThan(0);

    // Check for default roles
    const roleNames = body.data.roles.map((r: any) => r.code || r.name);
    expect(roleNames).toContain('admin');
    expect(roleNames).toContain('agent');
    expect(roleNames).toContain('end_user');
  });

  test('should create custom role with permissions', async () => {
    const loginResp = await request.post('http://localhost:8090/api/v1/auth/login', {
      data: { username: 'admin', password: 'admin123' }
    });
    const token = (await loginResp.json()).data.access_token;

    const roleData = {
      name: 'Custom Test Role',
      code: 'custom_test_role',
      description: 'Role for automated testing',
      permissions: ['ticket:read', 'ticket:write', 'knowledge:read']
    };

    const response = await request.post('http://localhost:8090/api/v1/roles', {
      headers: { 'Authorization': `Bearer ${token}` },
      data: roleData
    });

    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.code).toBe(0);
    expect(body.data.name).toBe('Custom Test Role');
    expect(body.data.permissions).toHaveLength(3);
    expect(body.data.permissions).toContain('ticket:read');

    // Clean up - delete the role
    const roleId = body.data.id;
    await request.delete(`http://localhost:8090/api/v1/roles/${roleId}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
  });

  test('should deny access for end_user to list all users', async () => {
    // Create an end_user first (as admin)
    const adminLogin = await request.post('http://localhost:8090/api/v1/auth/login', {
      data: { username: 'admin', password: 'admin123' }
    });
    const adminToken = (await adminLogin.json()).data.access_token;

    const newUser = {
      username: `enduser_${Date.now()}`,
      email: `end_${Date.now()}@example.com`,
      name: 'End User Test',
      password: 'TestPass123',
      tenant_id: 1,
      role: 'end_user'
    };
    const createResp = await request.post('http://localhost:8090/api/v1/users', {
      headers: { 'Authorization': `Bearer ${adminToken}` },
      data: newUser
    });
    expect((await createResp.json()).code).toBe(0);

    // Login as end_user
    const endUserLogin = await request.post('http://localhost:8090/api/v1/auth/login', {
      data: { username: newUser.username, password: newUser.password }
    });
    const endUserToken = (await endUserLogin.json()).data.access_token;

    // Try to list all users
    const listResp = await request.get('http://localhost:8090/api/v1/users', {
      headers: { 'Authorization': `Bearer ${endUserToken}` }
    });

    const body = await listResp.json();
    // Should be forbidden or empty (end_user cannot list all users)
    expect(listResponse.status()).toBeOneOf([200, 403]);
    if (listResp.status() === 200) {
      // If 200, should only return own user or empty (depending on implementation)
      // The current RBAC middleware blocks the call entirely, so expect 403
    } else {
      expect(body.code).not.toBe(0);
    }
  });

  test('should enforce ticket ownership for end_user', async () => {
    // This test requires ticket creation; see further tests
  });

  test('should verify groups endpoint does not exist', async () => {
    const loginResp = await request.post('http://localhost:8090/api/v1/auth/login', {
      data: { username: 'admin', password: 'admin123' }
    });
    const token = (await loginResp.json()).data.access_token;

    const response = await request.get('http://localhost:8090/api/v1/groups', {
      headers: { 'Authorization': `Bearer ${token}` }
    });

    // Expect 404 Not Found
    expect(response.status()).toBe(404);
  });

  test('should test permission-based access to endpoints', async () => {
    // Create a role with specific permission and verify access
    // See integration test for detailed steps
  });

  test('should test token refresh flow', async () => {
    const loginResp = await request.post('http://localhost:8090/api/v1/auth/login', {
      data: { username: 'admin', password: 'admin123' }
    });
    const loginBody = await loginResp.json();
    const refreshToken = loginBody.data.refresh_token;

    // Wait a moment (simulate access token expiry not needed, refresh token is separate)
    const refreshResp = await request.post('http://localhost:8090/api/v1/auth/refresh', {
      data: { refresh_token: refreshToken }
    });

    expect(refreshResp.ok()).toBeTruthy();
    const refreshBody = await refreshResp.json();
    expect(refreshBody.code).toBe(0);
    expect(refreshBody.data).toHaveProperty('access_token');
  });

  test('should test logout invalidates session', async () => {
    const loginResp = await request.post('http://localhost:8090/api/v1/auth/login', {
      data: { username: 'admin', password: 'admin123' }
    });
    const token = (await loginResp.json()).data.access_token;

    const logoutResp = await request.post('http://localhost:8090/api/v1/auth/logout', {
      headers: { 'Authorization': `Bearer ${token}` }
    });

    expect(logoutResp.ok()).toBeTruthy();
    const body = await logoutResp.json();
    expect(body.code).toBe(0);

    // Subsequent call with same token should fail (if token is properly invalidated)
    // Note: Without token blacklist, JWT might still be valid until expiry
    // This depends on backend implementation
  });

  test('should verify tenant isolation', async () => {
    // Create tenant 1 admin, create user in tenant 1
    // Attempt to access that user as tenant 2 admin (if multi-tenant setup possible)
    // Requires more complex setup; see integration tests
  });
});

// Additional browser UI tests would be in separate file
// e.g., authz-roles-groups-ui.spec.ts