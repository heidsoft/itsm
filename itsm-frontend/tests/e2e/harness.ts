/**
 * e2e 会话底座：cookie 认证 + 跨 worker 会话缓存 + 一次性栈守卫。
 *
 * 为什么必须存在这一层（2026-10-02 实测）：
 * 1. 凭证自 v1.6.3 起只在 HttpOnly cookie 里，登录响应不再返回 access_token
 *    （`dto/auth_dto.go` 的 `AccessToken` 是 `json:"-"`）。任何
 *    `data.access_token` + `Authorization: Bearer …` 的写法只会发出
 *    `Bearer undefined`，断言的是「未认证」而不是权限边界。
 * 2. 夹具此前把后端地址写成 `NEXT_PUBLIC_API_URL || 'http://localhost'`，
 *    本机 nginx 监听 :80 并反代生产后端，于是 `npm run test:e2e:roles`
 *    直接在生产环境登录成功并 `POST /api/v1/auth/register` 建了用户。
 *    目标地址因此不允许有默认值，必须显式声明是可丢弃环境。
 * 3. 权限边界类用例的身份不能依赖固定口令演示账号——后端基线 seed 明确禁止
 *    （`pkg/seeder/seeder_test.go`），夹具用户改为运行时创建、口令随机。
 * 4. `middleware.LoginRateLimiter` 是 10 次/分钟/IP 的**进程内**计数，而 e2e 全部
 *    经同一个环回地址出去。7 个角色文件各自登录 admin + 夹具用户就要 14 次，
 *    必然把后面的用例挡在 403/2003 上——那既不是权限结论也不是产品缺陷。
 *    所以会话按角色缓存在环回临时目录，worker 之间复用同一枚 cookie 罐。
 */

import { mkdirSync, readFileSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

import {
  expect,
  request as pwRequest,
  type APIRequestContext,
  type APIResponse,
} from '@playwright/test';

/** 后端统一响应信封；`data` 的具体结构由被测端点决定，由用例自己收窄。 */
export interface ApiEnvelope {
  code: number;
  message: string;
  data?: unknown;
}

/** 与既有用例保持同一返回形态：status 是传输层结果，data 是完整响应体。 */
export interface ApiResponse {
  status: number;
  data: ApiEnvelope | string;
}

/**
 * 允许写入测试数据的目标：只有环回地址上的前端 dev 端口。
 * 刻意不含 `http://localhost`（:80）与 :8090——在本机它们分别指向 nginx→生产后端
 * 和生产后端直连端口，e2e 会建用户、建工单，绝不能落到那里。
 */
const ALLOWED_TARGETS = new Set([
  'http://localhost:3000',
  'http://127.0.0.1:3000',
  'http://localhost:3001',
  'http://127.0.0.1:3001',
]);

/** `scripts/e2e-isolated-stack.sh up` 写出的证明文件结构。 */
interface StackProof {
  project: string;
  runId: string;
  baseURL: string;
  backendDirectURL: string;
  proxyUpstream: string;
  imageID: string;
  canaryUser: string;
  adminUsername: string;
  adminPassword: string;
  createdAt: string;
  expiresAtEpoch: number;
  proved: string;
}

const LOOPBACK_HOSTS = new Set(['localhost', '127.0.0.1', '::1', '[::1]']);

/**
 * 读并校验 proof：目标地址必须由「经前端代理写入、再进本栈数据库数出 1 行」的
 * 实测结果背书，而不是只看 URL 白名单。
 *
 * 为什么白名单不够（2026-10-02 实测）：`next.config.ts` 与
 * `src/app/api/[...path]/route.ts` 的代理 upstream 默认是 `http://localhost:8090`，
 * 而本机 127.0.0.1:8090 属于生产容器（`DB_NAME=itsm_prod`）。浏览器只打环回 :3000
 * 依然会把 POST /users 代理进生产后端。所以这里要求 proof 里记录的
 * proxyUpstream 必须是 compose 网络内的服务名，且目标与 proof 逐字符一致。
 */
function readStackProof(baseURL: string): StackProof {
  const proofPath = process.env.ITSM_E2E_STACK_PROOF;
  if (!proofPath) {
    throw new Error(
      '缺少 ITSM_E2E_STACK_PROOF：环回地址不能证明代理 upstream 指向一次性栈。' +
        '先运行 scripts/e2e-isolated-stack.sh up，再 set -a; . <proof目录>/itsm-e2e-*.env; set +a'
    );
  }

  let proof: StackProof;
  try {
    proof = JSON.parse(readFileSync(proofPath, 'utf8')) as StackProof;
  } catch (error) {
    throw new Error(
      `读取 e2e 隔离栈 proof 失败（${proofPath}）： ${(error as Error).message}；重新运行 scripts/e2e-isolated-stack.sh up`
    );
  }

  const nowSeconds = Math.floor(Date.now() / 1000);
  if (!Number.isFinite(proof.expiresAtEpoch) || proof.expiresAtEpoch < nowSeconds) {
    throw new Error(
      `e2e 隔离栈 proof 已过期（${proofPath}，project=${proof.project}）：栈可能已被拆除或被别的进程复用同一端口，重新 up`
    );
  }

  if (new URL(`${baseURL}/`).host !== new URL(`${proof.baseURL}/`).host) {
    throw new Error(
      `PLAYWRIGHT_BASE_URL=${baseURL} 与 proof 记录的 ${proof.baseURL} 不一致（project=${proof.project}）。` +
        '主机名不做别名换算：localhost 与 127.0.0.1 在本机可能解析到不同后端，必须逐字符使用 proof 里的地址'
    );
  }

  const upstreamHost = new URL(proof.proxyUpstream).hostname;
  if (LOOPBACK_HOSTS.has(upstreamHost)) {
    throw new Error(
      `proof 里的前端代理 upstream=${proof.proxyUpstream} 指向环回地址，` +
        `可能落到生产后端；一次性栈必须是 compose 网络内的服务名（如 http://backend:8090）`
    );
  }

  return proof;
}

let proofCache: { baseURL: string; proof: StackProof } | undefined;

/**
 * 一次性栈守卫：声明环境可丢弃 + 目标在环回白名单里 + 有 canary 实测 proof，
 * 否则拒绝发请求。与 `playwright.ticket-type.config.ts` 的
 * `ITSM_E2E_ISOLATED_STACK` 同一套契约，但那一套跑在 CI 的临时 runner 上，
 * 本机必须额外持有 proof。
 */
export function isolatedBaseURL(): string {
  if (process.env.ITSM_E2E_ISOLATED_STACK !== '1') {
    throw new Error(
      'e2e 会写业务数据，需要显式声明目标是一次性栈：ITSM_E2E_ISOLATED_STACK=1'
    );
  }
  const raw = process.env.PLAYWRIGHT_BASE_URL;
  if (!raw) {
    throw new Error(
      '缺少 PLAYWRIGHT_BASE_URL：一次性栈的目标地址必须由 scripts/e2e-isolated-stack.sh 给出，不允许有默认值'
    );
  }
  const normalized = raw.replace(/\/$/, '');
  const url = new URL(`${normalized}/`);
  const port = url.port || (url.protocol === 'https:' ? '443' : '80');
  if (!ALLOWED_TARGETS.has(normalized) || port === '80' || port === '8090') {
    throw new Error(
      `PLAYWRIGHT_BASE_URL=${normalized} 不在一次性栈白名单里（仅允许环回 :3000/:3001）：拒绝向未知目标写测试数据`
    );
  }
  if (proofCache?.baseURL === normalized) return normalized;
  const proof = readStackProof(normalized);
  proofCache = { baseURL: normalized, proof };
  return normalized;
}

/** 当前 proof（调用前会先走一遍 isolatedBaseURL 的全部校验）。 */
export function stackProof(): StackProof {
  const baseURL = isolatedBaseURL();
  if (!proofCache || proofCache.baseURL !== baseURL) {
    throw new Error(`内部错误：${baseURL} 没有缓存的 proof`);
  }
  return proofCache.proof;
}

/** 管理员口令只来自 proof 文件或环境，仓库里不得出现可用口令。 */
export function adminPassword(): string {
  const secret =
    process.env.E2E_ADMIN_PASSWORD || process.env.ADMIN_PASSWORD || stackProof().adminPassword;
  if (!secret) {
    throw new Error(
      '缺少 E2E_ADMIN_PASSWORD：e2e 不接受硬编码口令，也没有可用的 proof 文件（scripts/e2e-isolated-stack.sh up 生成）'
    );
  }
  return secret;
}

/** 命中登录限流：调用方据此区分「稍后再试」与「口令不对」，后者才该丢弃夹具账号。 */
export class RateLimitedError extends Error {
  constructor(readonly retryAfterSeconds: number, message: string) {
    super(message);
    this.name = 'RateLimitedError';
  }
}

export interface StoredFixtureUser {
  username: string;
  password: string;
  roleCode: string;
}

// ---------------------------------------------------------------------------
// 会话缓存：state 文件与夹具账号清单都只落在环回临时目录，不进仓库。
// ---------------------------------------------------------------------------

const SESSION_DIR = join(tmpdir(), 'itsm-e2e-sessions');
// 换目标（不同端口的一次性栈）就是另一批会话，不能互相冒用。
const targetSlug = () => isolatedBaseURL().replace(/[^a-zA-Z0-9]/g, '-');
const stateFile = (key: string) => join(SESSION_DIR, `${targetSlug()}--${key}.state.json`);
const usersFile = () => join(SESSION_DIR, `${targetSlug()}--users.json`);
const lockDir = (key: string) => join(SESSION_DIR, `${targetSlug()}--${key}.lock`);

function ensureSessionDir(): void {
  mkdirSync(SESSION_DIR, { recursive: true });
}

function readStoredUsers(): Record<string, StoredFixtureUser> {
  try {
    return JSON.parse(readFileSync(usersFile(), 'utf8')) as Record<string, StoredFixtureUser>;
  } catch {
    return {};
  }
}

/** 记住本角色的夹具账号：一次性栈里它长期有效，重复供给只会堆垃圾用户。 */
export function storeFixtureUser(key: string, user: StoredFixtureUser): void {
  ensureSessionDir();
  const users = readStoredUsers();
  users[key] = user;
  writeFileSync(usersFile(), `${JSON.stringify(users, null, 2)}\n`, 'utf8');
}

export function dropFixtureUser(key: string): void {
  const users = readStoredUsers();
  if (!(key in users)) return;
  delete users[key];
  writeFileSync(usersFile(), `${JSON.stringify(users, null, 2)}\n`, 'utf8');
}

export function getFixtureUser(key: string): StoredFixtureUser | undefined {
  return readStoredUsers()[key];
}

const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

/** mkdir 是原子操作，用它当跨进程锁；被 kill 的进程不会自己释放，所以超过 2 分钟就接管。 */
function acquireLock(key: string): boolean {
  try {
    mkdirSync(lockDir(key));
    return true;
  } catch {
    try {
      if (Date.now() - statSync(lockDir(key)).mtimeMs > 120_000) {
        rmSync(lockDir(key), { recursive: true, force: true });
        mkdirSync(lockDir(key));
        return true;
      }
    } catch {
      return false;
    }
    return false;
  }
}

async function withLock<T>(key: string, fn: () => Promise<T>): Promise<T> {
  const deadline = Date.now() + 60_000;
  while (!acquireLock(key)) {
    if (Date.now() > deadline) {
      throw new Error(`e2e 会话锁等待超时（${key}）：另一个 worker 未取得进展`);
    }
    await sleep(200);
  }
  try {
    return await fn();
  } finally {
    rmSync(lockDir(key), { recursive: true, force: true });
  }
}

/**
 * 复用已缓存的 cookie 罐；access token 过期就用 refresh token 续签并写回缓存。
 * 返回 undefined 表示必须重新登录（缓存不存在、栈已重建、或 refresh 也被拒）。
 */
async function restoreCachedSession(key: string): Promise<APIRequestContext | undefined> {
  const file = stateFile(key);
  let rq: APIRequestContext | undefined;
  try {
    rq = await pwRequest.newContext({ baseURL: isolatedBaseURL(), storageState: file });
  } catch {
    return undefined;
  }
  try {
    const session = await rq.get('/api/v1/auth/session', { maxRedirects: 0 });
    if (session.ok() && (await session.json()).code === 0) return rq;
    if ((await tryRefresh(rq)) && (await sessionAlive(rq))) {
      await rq.storageState({ path: file });
      return rq;
    }
    await rq.dispose();
    return undefined;
  } catch {
    await rq.dispose().catch(() => undefined);
    return undefined;
  }
}

/**
 * 按角色键取会话：命中缓存零次登录，未命中才在锁内供给并落盘。
 * `create` 只在真正需要时执行（可能包含一次登录），因此稳态跑整套 suites 不再触发限流。
 */
export async function cachedSession(
  key: string,
  create: () => Promise<APIRequestContext>
): Promise<APIRequestContext> {
  ensureSessionDir();
  const restored = await restoreCachedSession(key);
  if (restored) return restored;

  return withLock(key, async () => {
    // 等锁期间别的 worker 可能已经供给好同一角色。
    const raced = await restoreCachedSession(key);
    if (raced) return raced;
    const rq = await create();
    await rq.storageState({ path: stateFile(key) });
    return rq;
  });
}

async function sessionAlive(rq: APIRequestContext): Promise<boolean> {
  try {
    const response = await rq.get('/api/v1/auth/session', { maxRedirects: 0 });
    return response.ok() && (await response.json()).code === 0;
  } catch {
    return false;
  }
}

// ---------------------------------------------------------------------------
// 登录与请求通道
// ---------------------------------------------------------------------------

/** 打开一个独立的 cookie 罐并让后端确认会话，而不是只看登录接口是否 200。 */
export async function openSession(username: string, password: string): Promise<APIRequestContext> {
  const base = isolatedBaseURL();
  for (let attempt = 0; attempt < 2; attempt++) {
    const rq = await pwRequest.newContext({ baseURL: base });
    try {
      const response = await rq.post('/api/v1/auth/login', {
        data: { username, password },
        maxRedirects: 0,
      });
      if (response.ok()) {
        const cookies = (await rq.storageState()).cookies;
        const access = cookies.find(cookie => cookie.name === 'access_token');
        expect(access, '登录后必须拿到 httpOnly access_token cookie').toBeDefined();
        expect(access?.httpOnly, 'access_token 必须是 httpOnly，JS 读不到才算安全').toBe(true);
        const session = await readSession(rq);
        expect(session.user.username, '会话身份必须与登录用户一致').toBe(username);
        return rq;
      }

      const body = await response.json().catch(() => ({}) as ApiEnvelope);
      const retryAfter = Number((body as { data?: { retryAfterSeconds?: number } }).data?.retryAfterSeconds ?? 0);
      if (body.code === 2003 && retryAfter > 0) {
        // 只重试一次并明确上限：限流窗口是 60 秒，无限等待只会把用例拖成超时。
        await rq.dispose();
        if (attempt === 0 && retryAfter <= 20) {
          await sleep((retryAfter + 1) * 1000);
          continue;
        }
        throw new RateLimitedError(
          retryAfter,
          `e2e 登录被限流（${username}）：${retryAfter}s 后重试。会话应命中缓存，连续出现说明一次性栈被重建或角色供给过多`
        );
      }
      throw new Error(`e2e 登录失败 ${username}: ${response.status()} ${JSON.stringify(body)}`);
    } catch (error) {
      await rq.dispose().catch(() => undefined);
      throw error;
    }
  }
  throw new Error(`e2e 登录未返回结果：${username}`);
}

interface SessionPayload {
  user: { id: number; username: string; role: string; tenantId: number };
  tenants?: unknown[];
  expiresIn: number;
}

async function readSession(rq: APIRequestContext): Promise<SessionPayload> {
  return unwrap<SessionPayload>(await rq.get('/api/v1/auth/session', { maxRedirects: 0 }));
}

/** 解 {code,message,data} 信封；业务码非 0 直接失败，避免用例把错误当空结果。 */
export async function unwrap<T>(response: APIResponse): Promise<T> {
  const payload = (await response.json()) as ApiEnvelope;
  if (payload.code !== 0) {
    throw new Error(`API 业务码非 0：code=${payload.code} message=${payload.message}`);
  }
  return payload.data as T;
}

async function csrfToken(rq: APIRequestContext): Promise<string> {
  // CSRF 每次写操作后轮换，必须临写前再取，且取 cookie 而非猜测的 JSON 别名。
  const issued = await rq.get('/api/v1/csrf-token', { maxRedirects: 0 });
  expect(issued.status(), 'csrf-token 获取失败').toBe(200);
  const cookie = (await rq.storageState()).cookies.find(item => item.name === 'csrf_token');
  if (!cookie) throw new Error('csrf_token cookie 未下发：无法发起 cookie 认证的写请求');
  return decodeURIComponent(cookie.value);
}

async function send(
  rq: APIRequestContext,
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE',
  path: string,
  data?: unknown,
  withCsrf = false
): Promise<ApiResponse> {
  const headers: Record<string, string> = {};
  if (withCsrf) headers['X-CSRF-Token'] = await csrfToken(rq);
  const response = await rq.fetch(path, { method, data, headers, maxRedirects: 0 });
  const body: ApiEnvelope | string = response
    .headers()['content-type']
    ?.includes('json')
    ? await response.json()
    : await response.text();
  return { status: response.status(), data: body };
}

async function tryRefresh(rq: APIRequestContext): Promise<boolean> {
  try {
    const refreshed = await send(rq, 'POST', '/api/v1/auth/refresh', {}, true);
    return refreshed.status === 200;
  } catch {
    return false;
  }
}

/**
 * 401 不能直接判成「权限拒绝」：access token 只有 15 分钟，夹具活过一次续签窗口是常态。
 * 顺序是「先读盘（其他 worker 可能已续签）→ 再自己续签」，因为 refresh token 单次可用，
 * 直接重放会把另一 worker 刚换到的凭证作废，反而把可用会话判死。
 */
async function withSessionRetry(
  key: string,
  rq: APIRequestContext,
  call: (context: APIRequestContext) => Promise<ApiResponse>
): Promise<ApiResponse> {
  const first = await call(rq);
  if (first.status !== 401) return first;

  if (key) {
    const reloaded = await restoreCachedSession(key);
    if (reloaded && reloaded !== rq) {
      const retried = await call(reloaded);
      if (retried.status !== 401) return retried;
    }
  }
  if (!(await tryRefresh(rq))) return first;
  if (key) await rq.storageState({ path: stateFile(key) }).catch(() => undefined);
  return call(rq);
}

export async function apiGet(rq: APIRequestContext, path: string, cacheKey = ''): Promise<ApiResponse> {
  return withSessionRetry(cacheKey, rq, context => send(context, 'GET', path));
}

/**
 * 写操作（POST/PUT/PATCH/DELETE）与 POST 走同一套 cookie + CSRF 通道。
 * 每次重试都重新取 CSRF token，轮换后的令牌才不会失效。
 */
async function writeWithCsrf(
  rq: APIRequestContext,
  method: 'POST' | 'PUT' | 'PATCH' | 'DELETE',
  path: string,
  cacheKey: string,
  data?: unknown
): Promise<ApiResponse> {
  return withSessionRetry(cacheKey, rq, context => send(context, method, path, data, true));
}

export async function apiPost(
  rq: APIRequestContext,
  path: string,
  data?: unknown,
  cacheKey = ''
): Promise<ApiResponse> {
  return writeWithCsrf(rq, 'POST', path, cacheKey, data);
}

export async function apiPut(
  rq: APIRequestContext,
  path: string,
  data?: unknown,
  cacheKey = ''
): Promise<ApiResponse> {
  return writeWithCsrf(rq, 'PUT', path, cacheKey, data);
}

export async function apiPatch(
  rq: APIRequestContext,
  path: string,
  data?: unknown,
  cacheKey = ''
): Promise<ApiResponse> {
  return writeWithCsrf(rq, 'PATCH', path, cacheKey, data);
}

export async function apiDelete(
  rq: APIRequestContext,
  path: string,
  cacheKey = ''
): Promise<ApiResponse> {
  return writeWithCsrf(rq, 'DELETE', path, cacheKey);
}

export interface ProvisionedUser {
  username: string;
  password: string;
  roleId: number;
  roleCode: string;
}

interface RoleRow {
  id: number;
  code: string;
}

/**
 * 在一次性栈里按需创建一个带 RBAC 角色的夹具用户。
 *
 * 权限走 `roleIds`（user_roles M2M）：DBOnly 模式下只设 legacy 主角色拿不到任何权限。
 * 口令随机生成且满足后端策略（>=12，含大小写、数字、特殊字符）。
 */
export async function provisionRoleUser(
  admin: APIRequestContext,
  roleCode: string,
  usernamePrefix: string
): Promise<ProvisionedUser> {
  const rolesResponse = await apiGet(admin, '/api/v1/roles?page=1&pageSize=200', 'admin');
  if (rolesResponse.status !== 200) {
    throw new Error(
      `夹具无法读取角色：GET /api/v1/roles → ${rolesResponse.status} ${JSON.stringify(rolesResponse.data)}`
    );
  }
  // /api/v1/roles 的集合键当前是 roles（收敛进 items 前以真实 Router 为准）。
  const envelope = rolesResponse.data as ApiEnvelope;
  const payload = envelope.data as { roles?: RoleRow[]; total?: number };
  const roles = payload.roles ?? [];
  const role = roles.find(candidate => candidate.code === roleCode);
  if (!role) {
    throw new Error(
      `夹具角色 ${roleCode} 在一次性栈的 roles 表里不存在（读到 ${roles.length} 条：${roles
        .map(item => item.code)
        .join(', ')}）`
    );
  }

  const suffix = `${Date.now()}`.slice(-6);
  const username = `${usernamePrefix}_${suffix}`;
  const password = `E2e!${roleCode.slice(0, 4)}${suffix}s7`;

  const created = await apiPost(
    admin,
    '/api/v1/users',
    {
      username,
      email: `${username}@e2e.invalid`,
      name: `E2E ${roleCode}`,
      password,
      role: roleCode,
      roleIds: [role.id],
    },
    'admin'
  );
  if (created.status !== 200) {
    throw new Error(
      `夹具用户创建失败 ${username}(${roleCode})：${created.status} ${JSON.stringify(created.data)}`
    );
  }

  return { username, password, roleId: role.id, roleCode };
}

export { pwRequest };
