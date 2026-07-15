/**
 * E2E Test Utilities
 * Provides helper functions for authentication, API calls, and test data management
 */

import type { Page} from '@playwright/test';
import { expect } from '@playwright/test';
import { loginAndReturn } from '../auth-utils';

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
  await loginAndReturn(page, user.username, user.password);
}

/**
 * Login via API (more reliable for E2E tests)
 */
export async function loginViaApi(page: Page, role: TestUserRole): Promise<void> {
  await loginAs(page, role);
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
