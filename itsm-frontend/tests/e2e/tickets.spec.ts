/**
 * Ticket Management E2E Tests
 */

import { test, expect } from '@playwright/test';

const mockTickets = [
  {
    id: 1,
    title: 'Test Ticket 1',
    status: 'open',
    priority: 'high',
    category: 'Incident',
    created_at: new Date().toISOString(),
  },
  {
    id: 2,
    title: 'Test Ticket 2',
    status: 'in_progress',
    priority: 'medium',
    category: 'Service Request',
    created_at: new Date().toISOString(),
  },
];

test.describe('Ticket List Page', () => {
  test.beforeEach(async ({ page }) => {
    // Mock ticket list API
    await page.route('**/api/tickets**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          message: 'success',
          data: {
            tickets: mockTickets,
            total: mockTickets.length,
            page: 1,
            size: 10,
          },
        }),
      });
    });
    await page.goto('/tickets');
  });

  test('should display ticket list', async ({ page }) => {
    await expect(page.locator('table, [class*="table"], [class*="list"]')).toBeVisible();
  });

  test('should display ticket titles', async ({ page }) => {
    await expect(page.getByText('Test Ticket 1')).toBeVisible();
    await expect(page.getByText('Test Ticket 2')).toBeVisible();
  });

  test('should display status badges', async ({ page }) => {
    await expect(page.getByText('Open')).toBeVisible();
    await expect(page.getByText('In Progress')).toBeVisible();
  });

  test('should have search functionality', async ({ page }) => {
    const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="Search"]');
    await expect(searchInput).toBeVisible();
  });

  test('should have filter functionality', async ({ page }) => {
    const filterSelect = page.locator('[class*="filter"], select[class*="filter"]');
    await expect(filterSelect.first()).toBeVisible();
  });
});

test.describe('Ticket Detail Page', () => {
  test.beforeEach(async ({ page }) => {
    // Mock single ticket API
    await page.route('**/api/tickets/1**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          message: 'success',
          data: mockTickets[0],
        }),
      });
    });
    await page.goto('/tickets/1');
  });

  test('should display ticket title', async ({ page }) => {
    await expect(page.getByText('Test Ticket 1')).toBeVisible();
  });

  test('should display ticket details', async ({ page }) => {
    await expect(page.locator('[class*="detail"], [class*="info"]')).toBeVisible();
  });

  test('should have action buttons', async ({ page }) => {
    const actionButtons = page.locator('button[class*="action"], button[class*="btn"]');
    await expect(actionButtons.first()).toBeVisible();
  });
});

test.describe('Create Ticket Flow', () => {
  test('should display create ticket form', async ({ page }) => {
    await page.goto('/tickets/new');
    await expect(page.locator('form')).toBeVisible();
  });

  test('should have required fields', async ({ page }) => {
    await page.goto('/tickets/new');
    const titleInput = page.locator('input[id*="title"], input[name*="title"]');
    await expect(titleInput).toBeVisible();
  });

  test('should allow submitting new ticket', async ({ page }) => {
    await page.route('**/api/tickets**', async (route) => {
      if (route.request().method() === 'POST') {
        const { id, ...ticketData } = mockTickets[0];
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 0,
            message: 'success',
            data: { id: 3, ...ticketData },
          }),
        });
      }
    });

    await page.goto('/tickets/new');
    await page.locator('input[id*="title"]').fill('New Test Ticket');
    await page.locator('textarea[id*="description"]').fill('Test description');
    await page.locator('button[type="submit"]').click();

    // Should navigate away after successful submission
    await expect(page).not.toHaveURL(/\/new/);
  });
});
