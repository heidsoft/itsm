/**
 * 角色权限 e2e 的夹具：cookie 会话 + 按需创建并复用的夹具用户。
 *
 * 用例侧契约保持不变（`loginAs(role)` 返回句柄，`apiGet/apiPost(句柄, path)` 发请求），
 * 但句柄不再是 JWT：它只是本 worker 内一个独立 cookie 罐的标识。
 * 之所以必须换掉令牌：登录响应自 v1.6.3 起不返回 access_token（只在 HttpOnly cookie 里），
 * 旧夹具读 `data.accessToken` 只会拿到 undefined，于是所有权限用例都在断言
 * `Authorization: Bearer undefined` 的 401，而不是真正的权限边界。
 *
 * 夹具用户按运行创建，原因有两条：
 * - 后端基线 seed 禁止固定口令演示账号（`pkg/seeder/seeder_test.go`）；
 * - 公共注册接口不应接受自选角色（自助声明 admin 等于提权），
 *   所以权限身份必须由已认证的管理员通过 `POST /api/v1/users` 授予，
 *   且权限走 `roleIds`（DBOnly 模式下只设主角色拿不到任何权限）。
 *
 * 创建结果与会话 cookie 都缓存在环回临时目录：登录接口是 10 次/分钟/IP 的进程内限流，
 * 每个测试文件都重新供给一次必然把后续用例挡在 403/2003，那不是产品结论。
 */
import { test as base, expect, type APIRequestContext } from '@playwright/test';

import {
  adminPassword,
  apiDelete as harnessDelete,
  apiGet as harnessGet,
  apiPatch as harnessPatch,
  apiPost as harnessPost,
  apiPut as harnessPut,
  cachedSession,
  dropFixtureUser,
  getFixtureUser,
  isolatedBaseURL,
  openSession,
  provisionRoleUser,
  RateLimitedError,
  storeFixtureUser,
  type ApiResponse,
} from '../harness';

/** 角色夹具：username 只是前缀，真实用户名带运行时后缀，口令随机不落仓库。 */
const ROLE_FIXTURES = {
  admin: { role: 'admin', usernamePrefix: 'e2e_admin' },
  user1: { role: 'end_user', usernamePrefix: 'e2e_user' },
  security1: { role: 'security', usernamePrefix: 'e2e_sec' },
  engineer1: { role: 'technician', usernamePrefix: 'e2e_eng' },
  manager1: { role: 'manager', usernamePrefix: 'e2e_mgr' },
  tenant1admin: { role: 'admin', usernamePrefix: 'e2e_t1admin' },
} as const;

export type TestRole = keyof typeof ROLE_FIXTURES;

/** 兼容仍按名字读取账号表的用例；这里只暴露角色，不含任何口令。 */
export const TEST_ACCOUNTS: Record<TestRole, { username: string; role: string }> = Object.fromEntries(
  (Object.keys(ROLE_FIXTURES) as TestRole[]).map(role => [
    role,
    { username: ROLE_FIXTURES[role].usernamePrefix, role: ROLE_FIXTURES[role].role },
  ])
) as Record<TestRole, { username: string; role: string }>;

// 句柄 -> cookie 罐。同一 worker 内按角色复用，跨 worker 由磁盘缓存接力。
const sessions = new Map<string, APIRequestContext>();

async function adminSession(): Promise<APIRequestContext> {
  return cachedSession('admin', () => openSession('admin', adminPassword()));
}

/**
 * 非 admin 角色：会话同样走磁盘缓存，未命中才在锁内供给并落盘。
 *
 * 这里必须套 `cachedSession`：登录限流是 10 次/分钟/IP 的进程内计数，只缓存 admin
 * 的话每个角色文件仍会各自重新登录（2026-10-02 实测：会话目录里只有 admin 的
 * state 文件，5 个夹具账号都建好了却每个 worker 各登录一次，仍在 403/2003 上成片飘红）。
 *
 * 账号口令只在缓存缺失时才读盘复用；限流（RateLimitedError）不算账号失效——
 * 丢弃它只会让下一次运行再撞一次限流并把账号白白重建成第二个用户。
 */
async function openRoleSession(role: Exclude<TestRole, 'admin'>): Promise<APIRequestContext> {
  return cachedSession(role, async () => {
    const spec = ROLE_FIXTURES[role];
    const stored = getFixtureUser(role);
    if (stored) {
      try {
        return await openSession(stored.username, stored.password);
      } catch (error) {
        if (error instanceof RateLimitedError) throw error;
        dropFixtureUser(role);
      }
    }

    const controller = await adminSession();
    const provisioned = await provisionRoleUser(controller, spec.role, spec.usernamePrefix);
    storeFixtureUser(role, {
      username: provisioned.username,
      password: provisioned.password,
      roleCode: provisioned.roleCode,
    });
    return openSession(provisioned.username, provisioned.password);
  });
}

async function sessionFor(role: TestRole): Promise<APIRequestContext> {
  const existing = sessions.get(role);
  if (existing) return existing;
  const created =
    role === 'admin' ? await adminSession() : await openRoleSession(role as Exclude<TestRole, 'admin'>);
  sessions.set(role, created);
  return created;
}

interface TestFixtures {
  loginAs: (role: TestRole) => Promise<TestRole>;
  apiGet: (session: TestRole, path: string) => Promise<ApiResponse>;
  apiPost: (session: TestRole, path: string, body?: unknown) => Promise<ApiResponse>;
  apiPut: (session: TestRole, path: string, body?: unknown) => Promise<ApiResponse>;
  apiPatch: (session: TestRole, path: string, body?: unknown) => Promise<ApiResponse>;
  apiDelete: (session: TestRole, path: string) => Promise<ApiResponse>;
}

export const test = base.extend<TestFixtures>({
  // 会话是 worker 级资源：跨用例复用同一 cookie 罐，进程退出时由 Playwright 回收。
  // 但 baseURL 必须在第一个夹具求值前就通过一次性栈守卫，所以这里显式调用一次。
  loginAs: async ({}, use) => {
    isolatedBaseURL();
    await use(async (role: TestRole) => {
      // 句柄就是角色键：真正的凭证只在那个 cookie 罐里，不出现在任何变量或仓库里。
      await sessionFor(role);
      return role;
    });
  },

  apiGet: async ({}, use) => {
    await use(async (session: TestRole, path: string) =>
      harnessGet(await sessionFor(session), path, session)
    );
  },

  apiPost: async ({}, use) => {
    await use(async (session: TestRole, path: string, body?: unknown) =>
      harnessPost(await sessionFor(session), path, body, session)
    );
  },

  apiPut: async ({}, use) => {
    await use(async (session: TestRole, path: string, body?: unknown) =>
      harnessPut(await sessionFor(session), path, body, session)
    );
  },

  apiPatch: async ({}, use) => {
    await use(async (session: TestRole, path: string, body?: unknown) =>
      harnessPatch(await sessionFor(session), path, body, session)
    );
  },

  apiDelete: async ({}, use) => {
    await use(async (session: TestRole, path: string) =>
      harnessDelete(await sessionFor(session), path, session)
    );
  },
});

export { expect };
