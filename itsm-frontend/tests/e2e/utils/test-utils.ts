/**
 * E2E Test Utilities
 * Provides helper functions for authentication, API calls, and test data management
 */

import { Page, expect } from '@playwright/test';

// Test user credentials from seed data (see itsm-backend/pkg/seeder/seeder.go)
export const TEST_USERS = {
  admin: {
    username: 'admin',
    password: 'admin123',
    role: 'admin',
  },
  end_user: {
    username: 'user1',
    password: 'user123',
    role: 'end_user',
  },
  security: {
    username: 'security1',
    password: 'security123',
    role: 'security',
  },
  // Agent uses security1 user (has agent-like permissions)
  agent: {
    username: 'security1',
    password: 'security123',
    role: 'agent',
  },
} as const;

export type TestUserRole = keyof typeof TEST_USERS;

/**
 * Login as a specific role
 */
export async function loginAs(page: Page, role: TestUserRole): Promise<void> {
  const user = TEST_USERS[role];

  // Navigate to login page
  await page.goto('/login');
  await page.waitForLoadState('domcontentloaded');

  // Wait for form inputs - Ant Design style
  await page.waitForSelector('input.ant-input', { timeout: 15000 });

  // Fill in login form - Ant Design uses input.ant-input
  const inputs = page.locator('input.ant-input');
  await inputs.nth(0).fill(user.username);
  await inputs.nth(1).fill(user.password);

  // Submit form - only click the submit button
  await page.locator('button[type="submit"]').click();

  // Wait to leave login page (any URL except login)
  await page.waitForURL('**/!(login)**', { timeout: 20000 }).catch(async () => {
    // Check for error message if still on login page
    const errorVisible = await page.locator('.ant-message-error, .ant-alert-error').isVisible().catch(() => false);
    if (errorVisible) {
      throw new Error(`Login failed for ${role}`);
    }
  });
}

/**
 * Login via API (more reliable for E2E tests)
 */
export async function loginViaApi(page: Page, role: TestUserRole): Promise<void> {
  const user = TEST_USERS[role];
  const baseUrl = process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3000';
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

  // Get CSRF token first
  const csrfResponse = await page.request.get(`${apiUrl}/api/v1/auth/csrf`, {
    headers: { 'X-Requested-With': 'XMLHttpRequest' },
  });

  let csrfToken = '';
  if (csrfResponse.ok()) {
    const csrfData = await csrfResponse.json();
    csrfToken = csrfData.token || '';
  }

  // Perform login
  const loginResponse = await page.request.post(`${apiUrl}/api/v1/auth/login`, {
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': csrfToken,
      'X-Requested-With': 'XMLHttpRequest',
    },
    data: {
      username: user.username,
      password: user.password,
    },
  });

  if (!loginResponse.ok()) {
    const errorData = await loginResponse.json();
    throw new Error(`Login failed for ${role}: ${errorData.message || loginResponse.statusText()}`);
  }

  const loginData = await loginResponse.json();

  // Set auth token in storage
  if (loginData.data?.access_token) {
    await page.request.get(baseUrl); // Initialize context
    await page.evaluate((token) => {
      localStorage.setItem('access_token', token);
      localStorage.setItem('auth_token', token);
      document.cookie = `auth-token=${token}; path=/; max-age=900`;
    }, loginData.data.access_token);
  }

  // Navigate to dashboard
  await page.goto('/dashboard');
  await page.waitForLoadState('networkidle');
}

/**
 * Logout current user
 */
export async function logout(page: Page): Promise<void> {
  // Ensure page is valid
  const currentUrl = page.url();
  if (!currentUrl || currentUrl === 'about:blank') {
    await page.goto('/login');
  }

  // Click user menu or logout button
  const logoutButton = page.locator(
    'button:has-text("退出"), button:has-text("注销"), [class*="logout"], [data-testid="logout"]'
  );

  if (await logoutButton.isVisible().catch(() => false)) {
    await logoutButton.click();
  }

  // Clear auth storage with error handling
  await page.evaluate(() => {
    try {
      localStorage.clear();
      sessionStorage.clear();
      document.cookie.split(';').forEach(c => {
        document.cookie = c.replace(/^ +/, '').replace(/=.*/, '=;expires=' + new Date().toUTCString() + ';path=/');
      });
    } catch {
      // Ignore storage errors
    }
  }).catch(() => {});
}

/**
 * Navigate to a page and wait for it to load
 */
export async function navigateTo(page: Page, path: string): Promise<void> {
  await page.goto(path);
  await page.waitForLoadState('networkidle');
}

/**
 * Wait for a table to load with data
 */
export async function waitForTable(page: Page, timeout = 10000): Promise<void> {
  await page.waitForSelector('table, [class*="table"], [class*="list"]', { timeout });
}

/**
 * Create a ticket via API
 */
export async function createTicketViaApi(
  page: Page,
  ticketData: {
    title: string;
    description?: string;
    priority?: string;
    category?: string;
  }
): Promise<{ id: number }> {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

  // Get current auth token
  const token = await page.evaluate(() => localStorage.getItem('access_token'));

  const response = await page.request.post(`${apiUrl}/api/v1/tickets`, {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    data: ticketData,
  });

  if (!response.ok()) {
    const error = await response.json();
    throw new Error(`Failed to create ticket: ${error.message}`);
  }

  const data = await response.json();
  return data.data;
}

/**
 * Get current user info
 */
export async function getCurrentUser(page: Page): Promise<{ username: string; role: string } | null> {
  return await page.evaluate(() => {
    const userStr = localStorage.getItem('user');
    if (userStr) {
      return JSON.parse(userStr);
    }
    return null;
  });
}

/**
 * Take a screenshot with name
 */
export async function takeScreenshot(page: Page, name: string): Promise<void> {
  await page.screenshot({ path: `test-results/screenshots/${name}-${Date.now()}.png`, fullPage: true });
}

/**
 * Fill and submit a form
 */
export async function fillForm(
  page: Page,
  fields: Record<string, string>
): Promise<void> {
  for (const [selector, value] of Object.entries(fields)) {
    const input = page.locator(selector);
    await input.fill(value);
  }
}

/**
 * Assert user is logged in
 */
export async function assertLoggedIn(page: Page): Promise<void> {
  // Check for user avatar/menu which indicates logged in state
  const userMenu = page.locator('[class*="user"], [class*="avatar"], button:has-text("admin")');
  await expect(userMenu.first()).toBeVisible({ timeout: 5000 });
}

/**
 * Assert user is on login page
 */
export async function assertOnLoginPage(page: Page): Promise<void> {
  await expect(page).toHaveURL(/login/);
  const loginButton = page.locator('button[type="submit"], button:has-text("登录")');
  await expect(loginButton).toBeVisible();
}

/**
 * Wait for notification/toast message
 */
export async function waitForNotification(page: Page, text?: string): Promise<void> {
  const notification = text
    ? page.locator(`.ant-message-success, .ant-message-error, [class*="notification"]:has-text("${text}")`)
    : page.locator('.ant-message-success, .ant-message-error, [class*="notification"]');
  await notification.first().waitFor({ timeout: 5000 });
}
