/**
 * Login Page E2E Tests
 */

import { test, expect } from '@playwright/test';

test.describe('Login Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
  });

  test('should display login form', async ({ page }) => {
    await expect(page.locator('form')).toBeVisible();
  });

  test('should display username field', async ({ page }) => {
    await expect(page.locator('input[type="text"], input[id*="username"], input[id*="email"]')).toBeVisible();
  });

  test('should display password field', async ({ page }) => {
    await expect(page.locator('input[type="password"]')).toBeVisible();
  });

  test('should display login button', async ({ page }) => {
    const loginButton = page.locator('button[type="submit"]');
    await expect(loginButton).toBeVisible();
  });

  test('should show error when submitting empty form', async ({ page }) => {
    await page.locator('button[type="submit"]').click();
    // Should show validation error or prevent submission
    await expect(page).toHaveURL(/\/login/);
  });

  test('should navigate to dashboard on successful login', async ({ page }) => {
    // Mock successful login
    await page.route('**/api/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          message: 'success',
          data: {
            token: 'mock-token',
            user: {
              id: 1,
              username: 'testuser',
              email: 'test@example.com',
            },
          },
        }),
      });
    });

    await page.locator('input[type="text"], input[id*="username"]').fill('testuser');
    await page.locator('input[type="password"]').fill('password123');
    await page.locator('button[type="submit"]').click();

    await expect(page).not.toHaveURL(/\/login/);
  });
});

test.describe('Dashboard Page', () => {
  test('should display dashboard on home page', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('should display navigation menu', async ({ page }) => {
    await page.goto('/');
    // Check for common navigation elements
    const nav = page.locator('nav, header, [class*="nav"]');
    await expect(nav.first()).toBeVisible();
  });
});
