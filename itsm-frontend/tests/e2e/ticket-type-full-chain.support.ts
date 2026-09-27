import { expect, type APIResponse, type Page } from '@playwright/test';

export interface SessionUser {
  id: number;
  username: string;
  tenantId: number;
}

// Single response contract only; individual steps assert the relevant DTO fields.
export async function success<T>(response: APIResponse): Promise<T> {
  expect(response.status(), response.url()).toBe(200);
  const payload: unknown = await response.json();
  expect(payload).toMatchObject({ code: 0, message: expect.any(String) });
  if (typeof payload !== 'object' || payload === null || !('data' in payload)) {
    throw new Error(`Missing response data: ${response.url()}`);
  }
  return payload.data as T;
}

export async function get<T>(page: Page, path: string): Promise<T> {
  return success<T>(await page.request.get(path, { maxRedirects: 0 }));
}

export async function mutate(
  page: Page,
  method: 'POST' | 'PUT',
  path: string,
  data: Record<string, unknown>
): Promise<APIResponse> {
  // The middleware rotates CSRF after each write. Read the current transport
  // cookie (not a guessed JSON alias) before every mutation. Auth cookies remain
  // HttpOnly in the shared browser/request jar, just as in auth-utils.ts.
  await get<unknown>(page, '/api/v1/csrf-token');
  const csrf = (await page.context().cookies()).find(cookie => cookie.name === 'csrf_token');
  if (!csrf) throw new Error('CSRF cookie missing from same-origin proxy response');
  return page.request.fetch(path, {
    method,
    data,
    maxRedirects: 0,
    headers: { 'X-CSRF-Token': decodeURIComponent(csrf.value) },
  });
}

export async function write<T>(
  page: Page,
  method: 'POST' | 'PUT',
  path: string,
  data: Record<string, unknown>
): Promise<T> {
  return success<T>(await mutate(page, method, path, data));
}

export async function login(page: Page, username: string, password: string): Promise<SessionUser> {
  await page.context().clearCookies();
  const data = await success<{ user: SessionUser }>(
    await page.request.post('/api/v1/auth/login', {
      data: { username, password },
      maxRedirects: 0,
    })
  );
  expect(data.user).toMatchObject({
    id: expect.any(Number),
    username,
    tenantId: expect.any(Number),
  });
  expect(data.user.id).toBeGreaterThan(0);
  expect(data.user.tenantId).toBeGreaterThan(0);
  expect(data).not.toHaveProperty('accessToken');
  expect(data).not.toHaveProperty('refreshToken');
  expect(data.user).not.toHaveProperty('passwordHash');
  const cookies = await page.context().cookies();
  // These are cookie transport names, not API DTO aliases.
  expect(cookies.find(cookie => cookie.name === 'access_token')).toMatchObject({ httpOnly: true });
  expect(await get<SessionUser>(page, '/api/v1/auth/me')).toMatchObject({
    id: data.user.id,
    username: data.user.username,
    tenantId: data.user.tenantId,
  });
  return data.user;
}

export function single<T>(items: T[], label: string): T {
  expect(items, label).toHaveLength(1);
  const item = items[0];
  if (item === undefined) throw new Error(`${label}: missing item`);
  return item;
}

export interface PageDTO<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export function assertPage<T>(page: PageDTO<T>): void {
  expect(page).toMatchObject({
    items: expect.any(Array),
    total: expect.any(Number),
    page: 1,
    pageSize: 20,
    totalPages: expect.any(Number),
  });
  // A fresh isolated stack fits on one page. Fail rather than silently ignoring
  // extra pages (the legacy BPMN query DTOs do not bind camelCase filters yet).
  expect(page.total).toBeLessThanOrEqual(page.pageSize);
  expect(page.items).toHaveLength(page.total);
  expect(page.totalPages).toBe(Math.ceil(page.total / page.pageSize));
}
