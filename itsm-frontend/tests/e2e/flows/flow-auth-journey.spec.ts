/**
 * Complete authentication journey: login -> RBAC menus -> route guard -> API ACL.
 *
 * The seeded "agent" persona uses security1 (role: security), matching the
 * existing role-agent E2E suite. Each persona gets its own Page (and therefore
 * its own browser storage) while the journey remains one serial E2E test.
 */
import { expect, test, type Page } from '@playwright/test';
import { loginAndReturn } from '../auth-utils';

type ApiMenu = {
  name: string;
  path: string;
  children?: ApiMenu[];
};

type Journey = {
  name: 'end_user' | 'agent' | 'admin';
  username: string;
  password: string;
  expectedRole: string;
  allowedPage: string;
  forbiddenPage?: string;
  forbiddenApi?: string;
};

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';

const journeys: Journey[] = [
  {
    name: 'end_user',
    username: 'user1',
    password: 'user123',
    expectedRole: 'end_user',
    allowedPage: '/tickets',
    forbiddenPage: '/admin/users',
    forbiddenApi: '/api/v1/users/999',
  },
  {
    name: 'agent',
    username: 'security1',
    password: 'security123',
    expectedRole: 'security',
    allowedPage: '/tickets',
    forbiddenPage: '/admin/users',
    forbiddenApi: '/api/v1/users/999',
  },
  {
    name: 'admin',
    username: 'admin',
    password: 'admin123',
    expectedRole: 'admin',
    allowedPage: '/admin/users',
  },
];

function flattenMenus(menus: ApiMenu[]): ApiMenu[] {
  return menus.flatMap(menu => [menu, ...flattenMenus(menu.children ?? [])]);
}

async function bearerToken(page: Page): Promise<string> {
  const token = await page.evaluate(() => localStorage.getItem('access_token'));
  expect(token, 'login should persist an access token for API assertions').toBeTruthy();
  return token as string;
}

test.describe.serial('FLOW-AUTH: login -> RBAC -> menus -> API access', () => {
  test('proves the complete auth journey for end_user, agent, and admin', async ({ page }) => {
    // The fixture page is intentionally not reused: every role starts with an
    // independent Page and browser storage, as in the multi-user flow tests.
    const context = page.context();
    await page.close();

    for (const journey of journeys) {
      await test.step(`${journey.name}: authenticate and enforce the same RBAC in UI and API`, async () => {
        const rolePage = await context.newPage();

        try {
          await loginAndReturn(
            rolePage,
            journey.username,
            journey.password,
            journey.allowedPage,
          );

          const token = await bearerToken(rolePage);
          const headers = { Authorization: `Bearer ${token}` };

          const menusResponse = await rolePage.request.get(`${API_URL}/api/v1/auth/menus`, {
            headers,
          });
          expect(menusResponse.status(), `${journey.name} can read its backend menus`).toBe(200);

          const menusEnvelope = await menusResponse.json();
          expect(menusEnvelope.code).toBe(0);
          const backendMenus = flattenMenus([
            ...(menusEnvelope.data.main ?? []),
            ...(menusEnvelope.data.admin ?? []),
          ] as ApiMenu[]);
          expect(backendMenus.length, `${journey.name} should have visible menus`).toBeGreaterThan(0);

          await expect(rolePage.locator('aside')).toBeVisible({ timeout: 15_000 });
          for (const menu of backendMenus) {
            await expect(
              rolePage.locator('aside .ant-menu').getByText(menu.name, { exact: true }).first(),
              `sidebar should render backend menu "${menu.name}" for ${journey.name}`,
            ).toBeAttached();
          }

          const renderedMenuNames = await rolePage
            .locator('aside .ant-menu .ant-menu-title-content .truncate')
            .allTextContents();
          expect(new Set(renderedMenuNames)).toEqual(new Set(backendMenus.map(menu => menu.name)));

          await expect(
            rolePage.locator('aside').getByText(journey.expectedRole, { exact: true }),
            `${journey.name} should display the role returned by login`,
          ).toBeVisible();

          const allowedNavigation = await rolePage.goto(journey.allowedPage, {
            waitUntil: 'domcontentloaded',
          });
          expect(allowedNavigation?.status()).toBe(200);
          await expect(rolePage).toHaveURL(new RegExp(`${journey.allowedPage.replace('/', '\\/')}(?:[/?#]|$)`));

          const allowedApi = await rolePage.request.get(`${API_URL}/api/v1/tickets`, { headers });
          expect(allowedApi.status(), `${journey.name} can read tickets`).toBe(200);

          if (journey.forbiddenPage) {
            const forbiddenNavigation = await rolePage.goto(journey.forbiddenPage, {
              waitUntil: 'domcontentloaded',
            });
            const accessDeniedVisible = await rolePage
              .getByText(/403|无权限|禁止访问|access denied/i)
              .first()
              .isVisible()
              .catch(() => false);
            expect(
              forbiddenNavigation?.status() === 403 ||
                !rolePage.url().includes(journey.forbiddenPage) ||
                accessDeniedVisible,
              `${journey.name} must not render ${journey.forbiddenPage}`,
            ).toBe(true);
          }

          if (journey.forbiddenApi) {
            const forbiddenApi = await rolePage.request.delete(`${API_URL}${journey.forbiddenApi}`, {
              headers,
            });
            expect(forbiddenApi.status(), `${journey.name} cannot delete users`).toBe(403);
          }
        } finally {
          await rolePage.close();
        }
      });
    }
  });
});
