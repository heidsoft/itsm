// Single transport layer shared by the Ink TUI and the headless agent layer.
// Every endpoint path is static so the contract scanner (scripts/check-route-contract.mjs)
// can compare against itsm-backend/router/*.go registrations.
import { isLoggedIn, loadCredentials, saveCredentials } from './credentials.js';
import { httpStatusToAgentError, CliError } from './errors.js';
import { redactSecrets } from './secrets.js';
import type { AgentError, ErrorKind } from './errors.js';
import type {
  LoginRequest,
  LoginResponseAuth,
  SwitchTenantResponse,
  User,
  Ticket,
  TicketListResponse,
  CreateTicketRequest,
  PaginationParams,
  Incident,
  IncidentListResponse,
  Change,
  ChangeListResponse,
  CI,
  CIListResponse,
  KnowledgeArticle,
  KnowledgeListResponse,
  NotificationListResponse,
  ConnectorManifest,
  ProcessInstance,
  ProcessInstanceListResponse,
} from './types.js';

const DEFAULT_BASE_URL = 'http://localhost:8090';

/**
 * Transport policy for credentials.
 *
 * A bearer token sent over plaintext HTTP crosses every switch between the
 * CLI and the backend. Loopback is safe (it never leaves the host); a remote
 * `http://` deployment is not, so the CLI refuses to attach a credential
 * unless the operator opts in explicitly with ITSM_ALLOW_INSECURE_HTTP=1
 * (documented as a compatibility escape hatch for legacy intranets).
 *
 * URL userinfo (`https://user:pass@host`) is rejected outright: it leaks the
 * secret into argv, logs and shell history before the request is even built.
 */
const LOOPBACK_HOSTS = new Set(['localhost', '127.0.0.1', '::1', '[::1]', '0.0.0.0']);

export interface TransportDescription {
  baseURL: string;
  protocol: string;
  host: string;
  secure: boolean;
  loopback: boolean;
  /** Whether credentials may be sent to this base URL. */
  credentialSafe: boolean;
  reason?: string;
}

export function describeTransport(baseURL: string): TransportDescription {
  let url: URL;
  try {
    url = new URL(baseURL);
  } catch {
    return { baseURL, protocol: '', host: '', secure: false, loopback: false, credentialSafe: false, reason: 'invalid URL' };
  }
  const loopback = LOOPBACK_HOSTS.has(url.hostname) || url.hostname.endsWith('.localhost');
  const hasUserInfo = Boolean(url.username || url.password);
  const secure = url.protocol === 'https:' || loopback;
  let reason: string | undefined;
  if (hasUserInfo) reason = 'base URL embeds userinfo; remove it and use --password-stdin';
  else if (url.protocol !== 'http:' && url.protocol !== 'https:') reason = `unsupported scheme ${url.protocol}`;
  else if (!secure) reason = 'plaintext HTTP to a non-loopback host would expose the bearer token';
  return {
    baseURL,
    protocol: url.protocol.replace(':', ''),
    host: url.host,
    secure,
    loopback,
    credentialSafe: reason === undefined,
    ...(reason ? { reason } : {}),
  };
}

/** Fail closed before a secret is attached to a request. */
export function assertTransportSafety(baseURL: string, opts: { sendingSecret: boolean }): void {
  const transport = describeTransport(baseURL);
  if (!transport.credentialSafe) {
    const overridden = process.env['ITSM_ALLOW_INSECURE_HTTP'] === '1';
    if (transport.reason?.startsWith('plaintext') && !opts.sendingSecret) return;
    if (transport.reason?.startsWith('plaintext') && overridden) return;
    throw new CliError({
      kind: 'usage',
      message: `Refusing to send credentials to ${baseURL}: ${transport.reason}`,
      retryable: false,
      hint:
        transport.reason?.startsWith('plaintext')
          ? 'Use an https:// base URL (or a loopback address for local testing). For a legacy intranet over HTTP, set ITSM_ALLOW_INSECURE_HTTP=1 and accept the risk.'
          : 'Point --base-url / ITSM_BASE_URL at a clean http(s) URL without embedded credentials.',
    });
  }
}


/**
 * Process-wide base URL override, set by the agent runner from `--base-url`
 * before dispatch. Resolution order inside ApiClient:
 *   explicit constructor arg (withBaseURL) → this override → ITSM_BASE_URL → default.
 *
 * Without this the module-level `apiClient` used by every domain command was
 * pinned to localhost:8090, so agents pointing ITSM_BASE_URL at staging or a
 * customer deployment silently read the developer's local backend instead.
 */
let baseURLOverride: string | undefined;
export function configureBaseURL(url?: string): void {
  baseURLOverride = url;
}

/**
 * Transport policy. Agent runs are bounded: every request gets a deadline and
 * idempotent reads get transparent retries, so a flaky backend surfaces as a
 * retryable `network` error instead of an indefinite hang that burns the
 * agent's turn budget.
 */
interface TransportConfig {
  timeoutMs: number;
  retries: number;
}

const DEFAULT_TIMEOUT_MS = 30_000;
const DEFAULT_RETRIES = 2;

let transportConfig: TransportConfig = { timeoutMs: DEFAULT_TIMEOUT_MS, retries: DEFAULT_RETRIES };

export function configureTransport(patch: Partial<TransportConfig>): void {
  transportConfig = { ...transportConfig, ...patch };
}

export function getTransportConfig(): TransportConfig {
  return transportConfig;
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

function isIdempotent(method: string): boolean {
  return /^(GET|HEAD|OPTIONS)$/i.test(method);
}

/** fetch() with a hard deadline; an aborted request becomes a retryable network error. */
async function timedFetch(url: string, init: RequestInit): Promise<Response> {
  const { timeoutMs } = transportConfig;
  const controller = new AbortController();
  const timer = timeoutMs > 0 ? setTimeout(() => controller.abort(), timeoutMs) : undefined;
  try {
    return await fetch(url, { ...init, signal: controller.signal });
  } catch (err) {
    const aborted = err instanceof Error && (err.name === 'AbortError' || err.name === 'TimeoutError');
    if (aborted) {
      throw new CliError({
        kind: 'network',
        message: `Request timed out after ${timeoutMs}ms`,
        retryable: true,
        hint: 'Raise the budget with --timeout <ms>, or check backend health with `itsm --json infra health`.',
      });
    }
    throw err;
  } finally {
    if (timer) clearTimeout(timer);
  }
}

/**
 * Retry policy: only idempotent verbs are replayed automatically (a replayed
 * POST could double-create). 429/503 honour Retry-After, other transport
 * failures back off 250ms → 500ms.
 */
async function fetchWithRetry(url: string, init: RequestInit, method: string): Promise<Response> {
  const { retries } = transportConfig;
  const maxAttempts = isIdempotent(method) ? retries + 1 : 1;
  let attempt = 0;
  let lastErr: unknown;
  while (attempt < maxAttempts) {
    try {
      const res = await timedFetch(url, init);
      if (isIdempotent(method) && res.status >= 500 && attempt + 1 < maxAttempts) {
        await sleep(backoff(attempt, res.headers.get('retry-after')));
        attempt++;
        continue;
      }
      if (res.status === 429 && attempt + 1 < maxAttempts) {
        await sleep(backoff(attempt, res.headers.get('retry-after')));
        attempt++;
        continue;
      }
      return res;
    } catch (err) {
      lastErr = err;
      if (err instanceof CliError || attempt + 1 >= maxAttempts) throw err;
      await sleep(backoff(attempt));
      attempt++;
    }
  }
  throw lastErr ?? new Error('request failed');
}

function backoff(attempt: number, retryAfter: string | null = null): number {
  const seconds = Number(retryAfter);
  if (Number.isFinite(seconds) && seconds > 0) return Math.min(seconds * 1000, 5000);
  return 250 * 2 ** attempt;
}

/**
 * Token injected from outside the credentials file (`ITSM_TOKEN` env or the
 * agent runner's env-password auto-login). Priority: function result →
 * credentials file. Set by `runtime.hydrateAgentToken`.
 */
let externalTokenProvider: (() => string | undefined) | undefined;
export function setExternalTokenProvider(fn: () => string | undefined): void {
  externalTokenProvider = fn;
}

export interface TransportOperationError extends CliError {
  // Marker interface: all errors thrown from ApiClient are classified CliErrors.
}

export function transportError(
  status: number,
  message: string,
  endpoint: string,
  capability?: string,
  capabilityState?: string,
): AgentError {
  const err = httpStatusToAgentError(status, message || `request to ${endpoint} failed`);
  // 后端能力门控（HTTP 503 + {data:{capability,state}}）的显式状态透传：
  // 只有非空字符串才透传，凭空编造能力名和状态都属于伪造契约。
  if (capability) err.capability = capability;
  if (capabilityState) err.capabilityState = capabilityState;
  return err;
}

/** Access-token lifetime (mirrors middleware.AccessTokenTTL = 15m on the backend). */
const REFRESHED_ACCESS_TTL_MS = 15 * 60 * 1000;

/**
 * Silent re-auth: exchange the stored refresh token for a new access token.
 *
 * The access token lives only 15 minutes while the refresh token lives 7 days,
 * so without this any session longer than a quarter of an hour starts failing
 * with exit 3 — which is exactly what made the CLI unusable for long debugging
 * sessions. Returns null on any failure; the caller then surfaces the original
 * 401 unchanged.
 */
async function refreshSession(baseURL: string, refreshToken: string): Promise<{ accessToken: string; refreshToken?: string } | null> {
  // The refresh token is itself a 7-day bearer credential: never send it to a
  // plaintext remote endpoint, even when the caller's access token was absent.
  assertTransportSafety(baseURL, { sendingSecret: true });
  try {
    const res = await timedFetch(`${baseURL}/api/v1/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken }),
    });
    if (!res.ok) return null;
    const cookies = res.headers.getSetCookie?.() ?? [res.headers.get('set-cookie') ?? ''];
    const accessToken = extractCookie(cookies, 'access_token');
    if (!accessToken) return null;
    return { accessToken, refreshToken: extractCookie(cookies, 'refresh_token') };
  } catch (err) {
    if (err instanceof CliError) throw err;
    return null;
  }
}

/**
 * Single-flight token renewal.
 *
 * The backend rotates the refresh token atomically: the moment one exchange
 * succeeds, the token every other in-flight request is holding is revoked.
 * MCP hosts fire tool calls in parallel, so an expired access token used to
 * produce N concurrent 401s → N refreshes with the same refresh token → 1
 * success and N-1 failures reporting "the refresh token was rejected".
 * Measured on this machine: 8 parallel `itsm_whoami` calls, 1 passed.
 *
 * Worse than the failure rate is the message: it tells the agent the stored
 * credentials are dead and to log in again, when in fact the race is entirely
 * self-inflicted — re-logging in "fixes" it by accident and hides the cause.
 *
 * Every 401 now joins one shared exchange and replays with its result.
 */
let refreshInFlight: Promise<string | null> | undefined;

async function ensureRefreshedToken(baseURL: string, status: number): Promise<string | null> {
  if (refreshInFlight) return refreshInFlight;

  refreshInFlight = (async () => {
    const stored = loadCredentials();
    if (!stored?.refreshToken) return null;

    // Another request may have completed an exchange between our 401 and this
    // point; its token is already persisted, so reuse it instead of spending
    // (and invalidating) yet another refresh token.
    const latest = loadCredentials();
    if (latest?.token && latest.token !== stored.token && isLoggedIn()) {
      return latest.token;
    }

    const renewed = await refreshSession(baseURL, stored.refreshToken);
    if (!renewed) return null;
    // The backend rotates the refresh token atomically (the old one is
    // revoked). If the response carried no new refresh cookie, the stored
    // token is already dead — persisting it would make the next renewal fail
    // with a misleading "session expired".
    if (!renewed.refreshToken) {
      throw new CliError({
        kind: 'not_authenticated',
        message: 'Session refreshed but no rotated refresh token was returned',
        httpStatus: status,
        retryable: false,
        hint: 'Refusing to persist a consumed refresh token. Run `itsm auth login --password-stdin` once.',
      });
    }
    saveCredentials({
      ...stored,
      token: renewed.accessToken,
      refreshToken: renewed.refreshToken,
      expiresAt: new Date(Date.now() + REFRESHED_ACCESS_TTL_MS).toISOString(),
    });
    return renewed.accessToken;
  })().finally(() => {
    refreshInFlight = undefined;
  });

  return refreshInFlight;
}

async function request<T>(
  baseURL: string,
  method: string,
  endpoint: string,
  data?: unknown,
  params?: Record<string, string | number | boolean | undefined>,
  tokenOverride?: string,
  allowRefresh = true
): Promise<T> {
  const token = tokenOverride || externalTokenProvider?.() || loadCredentials()?.token;
  const storedForRefresh = allowRefresh ? loadCredentials() : null;
  assertTransportSafety(baseURL, { sendingSecret: Boolean(token || storedForRefresh?.refreshToken) });
  const url = new URL(`${baseURL}/api/v1${endpoint}`);
  if (params) {
    Object.entries(params).forEach(([k, v]) => {
      if (v !== undefined) url.searchParams.set(k, String(v));
    });
  }

  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const res: Response = await fetchWithRetry(url.toString(), {
    method,
    headers,
    body: data ? JSON.stringify(data) : undefined,
  }, method);

  if (!res.ok) {
    if (res.status === 401 && allowRefresh) {
      const renewedToken = await ensureRefreshedToken(baseURL, res.status);
      if (renewedToken) {
        return request(baseURL, method, endpoint, data, params, renewedToken, false);
      }
      // The refresh token itself is dead (single-use rotation already consumed,
      // or past its 7-day TTL). Replaying the request cannot help — say so
      // instead of surfacing a bare 401 that sends agents in circles.
      throw new CliError({
        kind: 'not_authenticated',
        message: 'Session expired and the refresh token was rejected',
        httpStatus: res.status,
        retryable: false,
        hint: 'Run `itsm auth login --username <u> --password-stdin` once. The stored refresh token is no longer valid (single-use rotation or 7-day expiry).',
      });
    }
    // 失败响应体只读一次：除 message 外同时取后端能力门控的显式状态
    // （HTTP 503 + {code:5003, data:{capability,state}}），供错误信封透传。
    const [message, capability, capabilityState] = await safeBodyDetails(res);
    throw new CliError(transportError(res.status, message, endpoint, capability, capabilityState));
  }

  const json = (await res.json()) as { code: number; message: string; data: T };
  if (json.code !== 0) {
    const kind = businessKind(json.code);
    // 部分门控以 HTTP 200 + 非零 code 返回（历史 handler 仍存在），data 里同样
    // 可能携带 {capability, state}——按同样契约透传，存在性由门控响应自己担保。
    const gate =
      json.data && typeof json.data === 'object'
        ? (json.data as { capability?: unknown; state?: unknown })
        : undefined;
    const capability = gate && typeof gate.capability === 'string' ? gate.capability : undefined;
    const capabilityState = gate && typeof gate.state === 'string' ? gate.state : undefined;
    throw new CliError({
      kind,
      message: (json.message || `business error ${json.code}`).trim(),
      httpStatus: res.status,
      businessCode: json.code,
      capability,
      capabilityState,
      // A declared outage/limit is worth retrying with backoff; a validation or
      // conflict is not (the same payload will fail the same way).
      retryable: kind === 'unavailable',
    });
  }
  return json.data;
}

/**
 * Backend business code → stable CLI error kind (HTTP 200 + code != 0 case).
 *
 * The backend is allowed to carry a business code in a 200 body (legacy handlers
 * still do), so the CLI must classify on the code, not on the HTTP status alone.
 * Otherwise a 404 arrives as `validation` and an agent retries a request that
 * can never succeed, or mistakes a rate limit for a permission error.
 */
function businessKind(code: number): ErrorKind {
  if (code === 2001 || code === 2002) return 'not_authenticated';
  if (code === 2003 || code === 2004) return 'forbidden';
  if (code === 4004 || code === 404) return 'not_found';
  if (code === 4090 || code === 409 || code === 4220 || code === 422) return 'conflict';
  if (code === 4290 || code === 429 || code === 5003 || code === 5010) return 'unavailable';
  if (code >= 5001) return 'internal';
  return 'validation';
}

async function safeBody(res: Response): Promise<string> {
  try {
    const text = await res.text();
    try {
      const json = JSON.parse(text) as { message?: string };
      return redactSecrets(json.message ?? text);
    } catch {
      return redactSecrets(text);
    }
  } catch {
    return '';
  }
}

/**
 * Read the failure body once and return (sanitized message, capability, state).
 *
 * The capability gate answers with `{code, message, data:{capability, state}}`;
 * dropping `data` on the floor turns "MSP is disabled in this deployment" into a
 * bare kind=unavailable, which callers cannot tell apart from a mere outage.
 */
async function safeBodyDetails(res: Response): Promise<[string, string | undefined, string | undefined]> {
  try {
    const text = await res.text();
    try {
      const json = JSON.parse(text) as {
        message?: string;
        data?: { capability?: string; state?: string };
      };
      const capability = typeof json.data?.capability === 'string' ? json.data.capability : undefined;
      const state = typeof json.data?.state === 'string' ? json.data.state : undefined;
      return [redactSecrets(json.message ?? text), capability, state];
    } catch {
      return [redactSecrets(text), undefined, undefined];
    }
  } catch {
    return ['', undefined, undefined];
  }
}

export class ApiClient {
  constructor(private explicitBaseURL?: string) {}

  /** Resolved per request so ITSM_BASE_URL / --base-url apply to every domain, not just infra. */
  private get baseURL(): string {
    return this.explicitBaseURL ?? baseURLOverride ?? process.env['ITSM_BASE_URL'] ?? DEFAULT_BASE_URL;
  }

  withBaseURL(baseURL: string): ApiClient {
    return new ApiClient(baseURL);
  }

  /** Generic raw request used by the agent escape hatch (`infra.api`). Full path required. */
  /**
   * Raw escape hatch (`itsm infra api`). Delegates to request() so the caller gets
   * the exact same auth chain (ITSM_TOKEN → stored credentials → silent refresh)
   * and error classification as first-class commands.
   *
   * It previously built its own fetch without an Authorization header, so every
   * authenticated endpoint answered 401 — which made the one surface meant for
   * debugging unreachable ones useless for exactly the cases it exists for.
   */
  async requestRaw(method: string, fullPath: string, body?: unknown): Promise<unknown> {
    const prefix = '/api/v1';
    const relPath = fullPath.startsWith(prefix) ? fullPath.slice(prefix.length) : fullPath;
    // fetch rejects a body on GET/HEAD; drop it instead of throwing.
    const sendBody = /^(GET|HEAD)$/i.test(method) ? undefined : body;
    return request<unknown>(this.baseURL, method, relPath, sendBody);
  }

  // ---------- Auth ----------
  async login(loginData: LoginRequest): Promise<LoginResponseAuth> {
    assertTransportSafety(this.baseURL, { sendingSecret: true });
    const url = `${this.baseURL}/api/v1/auth/login`;
    const res = await timedFetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(loginData),
    });
    if (!res.ok) throw new CliError(transportError(res.status, await safeBody(res), '/auth/login'));
    const json = (await res.json()) as { code: number; message: string; data: LoginResponseAuth };
    if (json.code !== 0) throw new CliError({ kind: 'not_authenticated', message: json.message || 'Login failed', httpStatus: res.status, businessCode: json.code, retryable: false });
    // Tokens travel in HttpOnly cookies (access_token/refresh_token); surface
    // them to the CLI caller so credentials can be persisted.
    const cookies = res.headers.getSetCookie?.() ?? [res.headers.get('set-cookie') ?? ''];
    const accessToken = extractCookie(cookies, 'access_token');
    const refreshToken = extractCookie(cookies, 'refresh_token');
    return { ...json.data, accessToken, refreshToken };
  }

  /**
   * Revoke the server-side session before clearing the local store.
   *
   * The backend reads the refresh token from the `refresh_token` cookie on
   * /auth/logout. A CLI has no cookie jar, so without this header `logout`
   * only deletes the local file and leaves a valid 7-day refresh token behind
   * — exactly the credential an attacker would keep after a file leak.
   */
  async logout(refreshToken?: string): Promise<void> {
    const endpoint = `${this.baseURL}/api/v1/auth/logout`;
    const token = externalTokenProvider?.() ?? loadCredentials()?.token;
    assertTransportSafety(this.baseURL, { sendingSecret: Boolean(token || refreshToken) });
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    if (token) headers['Authorization'] = `Bearer ${token}`;
    if (refreshToken) headers['Cookie'] = `refresh_token=${refreshToken}`;
    const res = await timedFetch(endpoint, { method: 'POST', headers });
    if (!res.ok) throw new CliError(transportError(res.status, await safeBody(res), '/auth/logout'));
    const json = (await res.json()) as { code: number; message: string };
    if (json.code !== 0) {
      throw new CliError({
        kind: businessKind(json.code),
        message: json.message || 'logout failed',
        httpStatus: res.status,
        businessCode: json.code,
        retryable: false,
      });
    }
  }

  /** Explicit rotation used by `auth refresh`; also used by the silent 401 path. */
  async refreshStoredSession(): Promise<{ accessToken: string; refreshToken: string; expiresAt: string; expiresIn: number }> {
    const stored = loadCredentials();
    if (!stored?.refreshToken) {
      throw new CliError({
        kind: 'not_authenticated',
        message: 'No refresh token is stored',
        retryable: false,
        hint: 'Run `itsm auth login --password-stdin` to establish a session first.',
      });
    }
    const renewed = await refreshSession(this.baseURL, stored.refreshToken);
    if (!renewed) {
      throw new CliError({
        kind: 'not_authenticated',
        message: 'Refresh token rejected or the refresh response carried no access token',
        retryable: false,
        hint: 'The refresh token is single-use and may have expired or already been rotated. Run `itsm auth login` again.',
      });
    }
    if (!renewed.refreshToken) {
      throw new CliError({
        kind: 'not_authenticated',
        message: 'Refresh response did not rotate the refresh token',
        retryable: false,
        hint: 'Refusing to persist a consumed refresh token. Run `itsm auth login` again.',
      });
    }
    const expiresAt = new Date(Date.now() + REFRESHED_ACCESS_TTL_MS).toISOString();
    saveCredentials({
      ...stored,
      token: renewed.accessToken,
      refreshToken: renewed.refreshToken,
      expiresAt,
    });
    return {
      accessToken: renewed.accessToken,
      refreshToken: renewed.refreshToken,
      expiresAt,
      expiresIn: Math.floor(REFRESHED_ACCESS_TTL_MS / 1000),
    };
  }

  async getUserInfo(): Promise<User> { return request<User>(this.baseURL, 'GET', '/auth/me'); }
  async getSession(): Promise<unknown> { return request(this.baseURL, 'GET', '/auth/session'); }
  async getUserTenants(): Promise<unknown> { return request(this.baseURL, 'GET', '/auth/tenants'); }
  async getUserMenus(): Promise<unknown> { return request(this.baseURL, 'GET', '/auth/menus'); }

  // Public self-service auth. These carry a password/token in the body, so the
  // transport guard is explicit: request() only knows about the auth header.
  async register(data: Record<string, unknown>): Promise<unknown> {
    assertTransportSafety(this.baseURL, { sendingSecret: true });
    return request(this.baseURL, 'POST', '/auth/register', data);
  }
  async forgotPassword(data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', '/auth/forgot-password', data);
  }
  async resetPassword(data: Record<string, unknown>): Promise<unknown> {
    assertTransportSafety(this.baseURL, { sendingSecret: true });
    return request(this.baseURL, 'POST', '/auth/reset-password', data);
  }
  async validateResetToken(data: Record<string, unknown>): Promise<unknown> {
    assertTransportSafety(this.baseURL, { sendingSecret: true });
    return request(this.baseURL, 'POST', '/auth/validate-reset-token', data);
  }
  async listSSOProviders(): Promise<unknown> { return request(this.baseURL, 'GET', '/auth/sso/providers'); }
  async initiateSSO(data: Record<string, unknown>): Promise<unknown> { return request(this.baseURL, 'POST', '/auth/sso/initiate', data); }
  async listSSOIdentities(): Promise<unknown> { return request(this.baseURL, 'GET', '/auth/sso/identities'); }
  async bindSSO(data: Record<string, unknown>): Promise<unknown> { return request(this.baseURL, 'POST', '/auth/sso/bind', data); }
  async unbindSSO(data: Record<string, unknown>): Promise<unknown> { return request(this.baseURL, 'POST', '/auth/sso/unbind', data); }
  async switchTenant(tenantId: string): Promise<SwitchTenantResponse & { accessToken?: string; refreshToken?: string }> {
    const url = new URL(`${this.baseURL}/api/v1/auth/switch-tenant`);
    const token = externalTokenProvider?.() ?? loadCredentials()?.token;
    assertTransportSafety(this.baseURL, { sendingSecret: Boolean(token) });
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    if (token) headers['Authorization'] = `Bearer ${token}`;
    const res = await timedFetch(url.toString(), {
      method: 'POST',
      headers,
      body: JSON.stringify({ tenantId: Number(tenantId) }),
    });
    if (!res.ok) throw new CliError(transportError(res.status, await safeBody(res), '/auth/switch-tenant'));
    const json = (await res.json()) as { code: number; message: string; data: SwitchTenantResponse };
    if (json.code !== 0) throw new CliError({ kind: businessKind(json.code), message: json.message || 'switch failed', httpStatus: res.status, businessCode: json.code, retryable: false });
    const cookies = res.headers.getSetCookie?.() ?? [res.headers.get('set-cookie') ?? ''];
    // Backend rotates BOTH cookies on tenant switch; dropping the refresh token
    // here used to break silent renewal right after a tenant switch.
    return {
      ...json.data,
      accessToken: extractCookie(cookies, 'access_token'),
      refreshToken: extractCookie(cookies, 'refresh_token'),
    };
  }

  // ---------- Platform / infra (used by agents as the first smoke test) ----------
  async getHealth(): Promise<unknown> {
    const res = await timedFetch(`${this.baseURL}/api/v1/health`, {});
    if (!res.ok) throw new CliError(transportError(res.status, '', '/health'));
    return unwrapBody(await res.json());
  }
  async getReady(): Promise<unknown> {
    const res = await timedFetch(`${this.baseURL}/api/v1/readyz`, {});
    if (!res.ok) throw new CliError(transportError(res.status, '', '/readyz'));
    return unwrapBody(await res.json());
  }
  async getVersion(): Promise<unknown> {
    const res = await timedFetch(`${this.baseURL}/api/v1/version`, {});
    if (!res.ok) throw new CliError(transportError(res.status, '', '/version'));
    return unwrapBody(await res.json());
  }
  async getCapabilities(): Promise<unknown> { return request<unknown>(this.baseURL, 'GET', '/capabilities'); }
  async globalSearch(q: string, page?: number): Promise<unknown> {
    return request<unknown>(this.baseURL, 'GET', '/global-search', undefined, { q, page });
  }

  // ---------- Tickets ----------
  async listTickets(params?: PaginationParams): Promise<TicketListResponse> {
    return request<TicketListResponse>(this.baseURL, 'GET', '/tickets', undefined, params as Record<string, string | number | undefined>);
  }
  async getTicket(id: number | string): Promise<Ticket> {
    return request<Ticket>(this.baseURL, 'GET', `/tickets/${id}`);
  }
  async createTicket(data: CreateTicketRequest): Promise<Ticket> {
    return request<Ticket>(this.baseURL, 'POST', '/tickets', data);
  }
  async updateTicket(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PATCH', `/tickets/${id}`, data);
  }
  async deleteTicket(id: number | string): Promise<void> {
    await request<void>(this.baseURL, 'DELETE', `/tickets/${id}`);
  }
  async searchTickets(query: string, params?: PaginationParams): Promise<TicketListResponse> {
    // The backend binds `c.Query("q")` (handlers/ticket/handler.go SearchTickets).
    // Sending `keyword` yields 400 搜索关键词不能为空 for every query.
    return request<TicketListResponse>(this.baseURL, 'GET', '/tickets/search', undefined, { ...params, q: query } as unknown as Record<string, string | number | boolean | undefined>);
  }
  async getOverdueTickets(): Promise<Ticket[]> {
    return request<Ticket[]>(this.baseURL, 'GET', '/tickets/overdue');
  }
  async getTicketStats(): Promise<unknown> { return request(this.baseURL, 'GET', '/tickets/stats'); }
  async getMyCC(): Promise<unknown> { return request(this.baseURL, 'GET', '/tickets/cc/my'); }
  async getApprovalRecords(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    // dto.GetApprovalRecordsRequest: ticketId / workflowId / status filters plus paging.
    return request(this.baseURL, 'GET', '/tickets/approval/records', undefined, params);
  }
  async assignTicket(id: number | string, assigneeId: number | string): Promise<unknown> {
    // The backend binds AssignTicketRequest.AssigneeID as int; sending the CLI's
    // positional string produced `{"assigneeId":"1"}` and a 400. Coerce here so
    // every caller (agent command, flow scenario) shares one correct body.
    return request(this.baseURL, 'POST', `/tickets/${id}/assign`, { assigneeId: Number(assigneeId) });
  }
  async resolveTicket(id: number | string, data?: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', `/tickets/${id}/resolve`, data);
  }
  async closeTicket(id: number | string, data?: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', `/tickets/${id}/close`, data);
  }
  async escalateTicket(id: number | string, data?: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', `/tickets/${id}/escalate`, data);
  }
  async getTicketHistory(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/tickets/${id}/history`);
  }
  async getTicketWorkflowState(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/tickets/${id}/workflow/state`);
  }
  async ticketWorkflowAction(action: 'accept' | 'reject' | 'withdraw' | 'forward' | 'approve' | 'resolve' | 'close' | 'reopen' | 'cc', data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', `/tickets/workflow/${action}`, data);
  }
  async getTicketSLA(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/tickets/${id}/sla`);
  }
  async aiSummarizeTicket(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/ai/tickets/${id}/summary`);
  }
  async aiAnalyzeTicket(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'POST', `/ai/tickets/${id}/analyze`);
  }
  async listTicketComments(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/tickets/${id}/comments`);
  }
  async createTicketComment(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', `/tickets/${id}/comments`, data);
  }
  async updateTicketStatus(id: number | string, status: string): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/tickets/${id}/status`, { status });
  }
  async pauseTicketSLA(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/tickets/${id}/sla/pause`);
  }
  async resumeTicketSLA(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/tickets/${id}/sla/resume`);
  }
  async listSubtasks(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/tickets/${id}/subtasks`);
  }
  async createSubtask(id: number | string, data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', `/tickets/${id}/subtasks`, data);
  }
  async updateSubtask(id: number | string, subtaskId: number | string, data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'PATCH', `/tickets/${id}/subtasks/${subtaskId}`, data);
  }
  async deleteSubtask(id: number | string, subtaskId: number | string): Promise<void> {
    await request<void>(this.baseURL, 'DELETE', `/tickets/${id}/subtasks/${subtaskId}`);
  }
  async getTicketRelations(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/tickets/${id}/relations`);
  }
  async updateTicketRelations(id: number | string, data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'PATCH', `/tickets/${id}/relations`, data);
  }
  async getTicketRelationStats(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/tickets/${id}/relations/stats`);
  }
  async getTicketCC(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/tickets/${id}/cc`);
  }
  async addTicketTags(id: number | string, data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', `/tickets/${id}/tags`, data);
  }
  async removeTicketTags(id: number | string, data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'DELETE', `/tickets/${id}/tags`, data);
  }
  async getTicketRating(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/tickets/${id}/rating`);
  }
  async submitTicketRating(id: number | string, data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', `/tickets/${id}/rating`, data);
  }
  async getRatingStats(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/tickets/rating-stats', undefined, params);
  }
  async getTicketConfigurationItems(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/tickets/${id}/configuration-items`);
  }
  async getTicketDependencies(id: number | string, params: { action: 'close' | 'delete' | 'change_status'; newStatus?: string }): Promise<unknown> {
    return request(this.baseURL, 'GET', `/tickets/${id}/dependencies`, undefined, params);
  }
  async listTicketNotifications(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/tickets/${id}/notifications`);
  }
  async sendTicketNotification(id: number | string, data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', `/tickets/${id}/notifications`, data);
  }
  async getTicketWorkflowHistory(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/tickets/${id}/workflow-history`);
  }
  async autoAssignTicket(id: number | string, data?: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', `/tickets/${id}/auto-assign`, data ?? {});
  }

  // ---------- Incidents ----------
  async listIncidents(params?: PaginationParams & { status?: string; priority?: string }): Promise<IncidentListResponse> {
    return request<IncidentListResponse>(this.baseURL, 'GET', '/incidents', undefined, params as Record<string, string | number | undefined>);
  }
  async getIncident(id: number | string): Promise<Incident> {
    return request<Incident>(this.baseURL, 'GET', `/incidents/${id}`);
  }
  async createIncident(data: unknown): Promise<unknown> { return request(this.baseURL, 'POST', '/incidents', data); }
  async deleteIncident(id: number | string): Promise<void> { await request<void>(this.baseURL, 'DELETE', `/incidents/${id}`); }
  async updateIncident(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/incidents/${id}`, data);
  }
  async incidentLifecycle(id: number | string, action: 'acknowledge' | 'resolve' | 'close' | 'reopen' | 'assign' | 'escalate' | 'convert-to-problem', data?: unknown): Promise<unknown> {
    // Always send a JSON body: the backend binds these with ShouldBindJSON, which
    // returns EOF (→ 400 "请求参数错误") when a body-less POST arrives.
    return request(this.baseURL, 'POST', `/incidents/${id}/${action}`, data ?? {});
  }
  async updateIncidentRootCause(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/incidents/${id}/root-cause`, data);
  }
  async listIncidentComments(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/incidents/${id}/comments`);
  }
  async createIncidentComment(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', `/incidents/${id}/comments`, data);
  }
  async updateIncidentStatus(id: number | string, data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/incidents/${id}/status`, data);
  }
  async getIncidentRootCause(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/incidents/${id}/root-cause`);
  }
  async getIncidentClassification(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/incidents/${id}/classification`);
  }
  async updateIncidentClassification(id: number | string, data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/incidents/${id}/classification`, data);
  }
  async getIncidentImpactAssessment(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/incidents/${id}/impact-assessment`);
  }
  async updateIncidentImpactAssessment(id: number | string, data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/incidents/${id}/impact-assessment`, data);
  }
  async escalateMajorIncident(id: number | string, data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', `/incidents/${id}/major-incident`, data);
  }
  async getIncidentEvents(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/incidents/${id}/events`);
  }
  async getIncidentAlerts(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/incidents/${id}/alerts`);
  }
  async getIncidentMetrics(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/incidents/${id}/metrics`);
  }
  async pauseIncidentSLA(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/incidents/${id}/sla/pause`);
  }
  async resumeIncidentSLA(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/incidents/${id}/sla/resume`);
  }
  async getIncidentStats(): Promise<unknown> { return request(this.baseURL, 'GET', '/incidents/stats'); }
  async incidentReport(params: { dateFrom?: string; dateTo?: string }): Promise<unknown> {
    return request(this.baseURL, 'GET', '/incidents/stats/report', undefined, params as Record<string, string | number | boolean | undefined>);
  }
  async aiTriage(payload: { title: string; description: string }): Promise<unknown> {
    return request<unknown>(this.baseURL, 'POST', '/ai/triage', payload);
  }

  // ---------- Problems ----------
  async listProblems(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/problems', undefined, params);
  }
  async getProblem(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/problems/${id}`);
  }
  async createProblem(data: unknown): Promise<unknown> { return request(this.baseURL, 'POST', '/problems', data); }
  async updateProblem(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/problems/${id}`, data);
  }
  async deleteProblem(id: number | string): Promise<void> { await request<void>(this.baseURL, 'DELETE', `/problems/${id}`); }
  async problemLifecycle(id: number | string, action: 'close' | 'investigate', data?: unknown): Promise<unknown> {
    // Same body-less POST pitfall as incidentLifecycle (ShouldBindJSON → EOF → 400).
    return request(this.baseURL, 'POST', `/problems/${id}/${action}`, data ?? {});
  }
  async updateProblemRootCause(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/problems/${id}/root-cause`, data);
  }
  async updateProblemSolution(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/problems/${id}/solution`, data);
  }
  async listProblemComments(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/problems/${id}/comments`);
  }
  async getProblemStats(): Promise<unknown> { return request(this.baseURL, 'GET', '/problems/stats'); }
  async getProblemTrends(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/problems/trend', undefined, params);
  }
  async getProblemHotspots(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/problems/hotspots', undefined, params);
  }
  async addProblemComment(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', `/problems/${id}/comments`, data);
  }
  async listKnownErrors(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/known-errors', undefined, params);
  }
  async getKnownError(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/known-errors/${id}`);
  }
  async searchKnownErrors(q: string): Promise<unknown> {
    return request(this.baseURL, 'GET', '/known-errors/search', undefined, { q });
  }

  // ---------- Changes ----------
  async listChanges(params?: PaginationParams & { status?: string }): Promise<ChangeListResponse> {
    return request<ChangeListResponse>(this.baseURL, 'GET', '/changes', undefined, params as Record<string, string | number | undefined>);
  }
  async getChange(id: number | string): Promise<Change> {
    return request<Change>(this.baseURL, 'GET', `/changes/${id}`);
  }
  async createChange(data: unknown): Promise<unknown> { return request(this.baseURL, 'POST', '/changes', data); }
  async updateChange(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/changes/${id}`, data);
  }
  async deleteChange(id: number | string): Promise<void> { await request<void>(this.baseURL, 'DELETE', `/changes/${id}`); }
  async changeLifecycle(id: number | string, action: 'submit' | 'assign' | 'approve' | 'reject' | 'schedule' | 'start' | 'complete' | 'fail' | 'close' | 'rollback' | 'cancel', data?: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', `/changes/${id}/${action}`, data);
  }
  async getChangeApprovals(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/changes/${id}/approvals`);
  }
  async getChangeRisk(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/changes/${id}/risk`);
  }
  async updateChangeRisk(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/changes/${id}/risk`, data);
  }
  async getChangePIR(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/changes/${id}/pir`);
  }
  async createChangePIR(id: number | string, data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', `/changes/${id}/pir`, data);
  }
  async updateChangePIR(pirId: number | string, data: Record<string, unknown>): Promise<unknown> {
    // Keyed by the PIR's own id, not the change id (dto/change_pir_dto.go).
    return request(this.baseURL, 'PUT', `/changes/pir/${pirId}`, data);
  }
  async getChangeApprovalSummary(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/changes/${id}/approval-summary`);
  }
  async createChangeApproval(id: number | string, data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', `/changes/${id}/approvals`, data);
  }
  async getChangeRiskAssessment(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/changes/${id}/risk-assessment`);
  }
  async getChangeCMDBImpact(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/changes/${id}/cmdb-impact`);
  }
  async getChangeRollbackSteps(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/changes/${id}/rollback-steps`);
  }
  async replaceChangeRollbackSteps(id: number | string, steps: string[]): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/changes/${id}/rollback-steps`, { steps });
  }
  async resolveChangeRollbackStep(id: number | string, stepId: number | string, data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', `/changes/${id}/rollback-steps/${stepId}`, data);
  }
  async listChangeCalendar(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/changes/calendar', undefined, params);
  }
  async listChangePIRs(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/changes/pirs', undefined, params);
  }
  async getChangeStats(): Promise<unknown> { return request(this.baseURL, 'GET', '/changes/stats'); }

  // ---------- Releases ----------
  async listReleases(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/releases', undefined, params);
  }
  async getRelease(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/releases/${id}`);
  }
  async createRelease(data: unknown): Promise<unknown> { return request(this.baseURL, 'POST', '/releases', data); }
  async updateRelease(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/releases/${id}`, data);
  }
  async updateReleaseStatus(id: number | string, status: string, comment?: string): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/releases/${id}/status`, comment ? { status, comment } : { status });
  }
  async releaseLifecycle(id: number | string, action: 'approve' | 'reject' | 'rollback', data?: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', `/releases/${id}/${action}`, data);
  }
  async deleteRelease(id: number | string): Promise<void> { await request<void>(this.baseURL, 'DELETE', `/releases/${id}`); }
  async getReleaseStats(): Promise<unknown> { return request(this.baseURL, 'GET', '/releases/stats'); }

  // ---------- Service Catalog / Requests ----------
  async listServiceCatalog(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/service-catalogs', undefined, params);
  }
  async getServiceCatalogItem(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/service-catalogs/${id}`);
  }
  async searchServiceCatalog(q: string): Promise<unknown> {
    return request(this.baseURL, 'GET', '/service-catalogs/search', undefined, { q });
  }
  async createServiceCatalogItem(data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', '/service-catalogs', data);
  }
  async updateServiceCatalogItem(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/service-catalogs/${id}`, data);
  }
  async deleteServiceCatalogItem(id: number | string): Promise<void> {
    await request<void>(this.baseURL, 'DELETE', `/service-catalogs/${id}`);
  }
  async listServiceRequests(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/service-requests', undefined, params);
  }
  async getServiceRequest(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/service-requests/${id}`);
  }
  async createServiceRequest(data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', '/service-requests', data);
  }
  async updateServiceRequest(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/service-requests/${id}`, data);
  }
  async updateServiceRequestStatus(id: number | string, status: string): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/service-requests/${id}/status`, { status });
  }
  async serviceRequestApproval(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', `/service-requests/${id}/approval`, data);
  }
  async startServiceRequestProvisioning(id: number | string, data?: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', `/service-requests/${id}/provision`, data);
  }
  async listProvisioningTasks(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/service-requests/${id}/provisioning-tasks`);
  }
  async getServiceRequestSLA(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/service-requests/${id}/sla`);
  }
  async listServiceRequestApprovals(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/service-requests/${id}/approvals`);
  }
  async deleteServiceRequest(id: number | string): Promise<void> {
    await request<void>(this.baseURL, 'DELETE', `/service-requests/${id}`);
  }
  /** Failed provisioning task dispositions: execute (retry now), retry (requeue), resolve (manual closure). */
  async provisioningTaskAction(id: number | string, action: 'execute' | 'retry' | 'resolve', data?: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', `/provisioning-tasks/${id}/${action}`, data);
  }

  // ---------- CMDB ----------
  async listCIs(params?: PaginationParams & { type?: string; status?: string }): Promise<CIListResponse> {
    return request<CIListResponse>(this.baseURL, 'GET', '/cmdb/cis', undefined, params as Record<string, string | number | undefined>);
  }
  async getCI(id: number | string): Promise<CI> {
    return request<CI>(this.baseURL, 'GET', `/cmdb/cis/${id}`);
  }
  async createCI(data: unknown): Promise<unknown> { return request(this.baseURL, 'POST', '/cmdb/cis', data); }
  async updateCI(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/cmdb/cis/${id}`, data);
  }
  async deleteCI(id: number | string): Promise<void> { await request<void>(this.baseURL, 'DELETE', `/cmdb/cis/${id}`); }
  /**
   * Keyword CI search. GET /cmdb/cis is the live endpoint (`search` is its
   * keyword param); POST /cmdb/cis/search is deprecated *and* only accepts a
   * structured envelope — a bare `{q}` there trips the `page: min=1` binding.
   */
  async searchCIsByKeyword(q: string, params?: PaginationParams): Promise<CIListResponse> {
    return request<CIListResponse>(this.baseURL, 'GET', '/cmdb/cis', undefined, { search: q, ...params });
  }
  /**
   * Structured CI search (filters/sort). `page` is defaulted because the
   * backend binds `min=1` while an omitted page arrives as 0 → 400, so a
   * filters-only body would be rejected for a field the caller never set.
   */
  async searchCIs(filters: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', '/cmdb/cis/search', { page: 1, pageSize: 20, ...filters });
  }
  async getCIStats(): Promise<unknown> { return request(this.baseURL, 'GET', '/cmdb/cis/stats'); }
  async listCITypes(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/cmdb/ci-types', undefined, params);
  }
  async getCITopology(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/cmdb/cis/${id}/topology`);
  }
  async getCIImpactAnalysis(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/cmdb/cis/${id}/impact-analysis`);
  }
  async getCIHistory(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/cmdb/cis/${id}/history`);
  }
  async updateCILifecycle(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/cmdb/cis/${id}/lifecycle`, data);
  }
  async listCIRelationships(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/cmdb/cis/relationships', undefined, params);
  }
  async createCIRelationship(data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', '/cmdb/cis/relationships', data);
  }
  async deleteCIRelationship(id: number | string): Promise<void> {
    // Canonical route is /cmdb/relationships/:id (the /cis/relationships path is
    // write/read only for compatibility, with no delete).
    await request<void>(this.baseURL, 'DELETE', `/cmdb/relationships/${id}`);
  }
  async getCIRelationship(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/cmdb/relationships/${id}`);
  }
  async updateCIRelationship(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/cmdb/relationships/${id}`, data);
  }
  async listCIRelationshipsByCI(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/cmdb/cis/${id}/relationships`);
  }
  async addCITags(id: number | string, tagIds: number[]): Promise<unknown> {
    return request(this.baseURL, 'POST', `/cmdb/cis/${id}/tags`, { tagIds });
  }
  async removeCITags(id: number | string, tagIds: number[]): Promise<unknown> {
    return request(this.baseURL, 'DELETE', `/cmdb/cis/${id}/tags`, { tagIds });
  }
  async getCILifecycleHistory(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/cmdb/cis/${id}/lifecycle/history`);
  }
  async revertCIVersion(id: number | string, version: number, remark?: string): Promise<unknown> {
    return request(this.baseURL, 'POST', `/cmdb/cis/${id}/revert`, remark ? { version, remark } : { version });
  }
  async createCIType(data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', '/cmdb/ci-types', data);
  }
  async updateCIType(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/cmdb/ci-types/${id}`, data);
  }
  async deleteCIType(id: number | string): Promise<void> {
    await request<void>(this.baseURL, 'DELETE', `/cmdb/ci-types/${id}`);
  }
  async getCIType(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/cmdb/ci-types/${id}`);
  }
  async listCITypeAttributes(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/cmdb/ci-types/${id}/attributes`);
  }
  async listRelationshipTypes(): Promise<unknown> {
    // Not paged: the handler returns a fixed type dictionary.
    return request(this.baseURL, 'GET', '/cmdb/relationship-types');
  }
  async getCMDBOntology(): Promise<unknown> { return request(this.baseURL, 'GET', '/cmdb/ontology'); }
  async getCMDBCapabilities(): Promise<unknown> { return request(this.baseURL, 'GET', '/cmdb/capabilities'); }
  async getCMDBReconciliation(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/cmdb/reconciliation', undefined, params);
  }
  async batchUpdateCIs(ids: number[], data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', '/cmdb/cis/batch', { ids, ...(data as object) });
  }
  async batchUpdateCILifecycle(ids: number[], data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', '/cmdb/cis/batch/lifecycle', { ids, ...(data as object) });
  }

  // ---------- Cloud / discovery ----------
  //
  // Cloud accounts / services / resources and the discovery pipeline live under
  // /cmdb/cloud-* and /cmdb/discovery/*. They were shipped with no CLI reach
  // (the drift check only fired when the CLI named a route the backend lacked,
  // never the reverse), so a whole product family was invisible to agents.
  async listCloudProviders(): Promise<unknown> {
    return request(this.baseURL, 'GET', '/cmdb/discovery/providers');
  }
  async getDiscoveryHealth(): Promise<unknown> {
    return request(this.baseURL, 'GET', '/cmdb/discovery/health');
  }
  // Accounts and services are *selector* endpoints: not paginated, returned whole
  // as the data source for CI-form cascades. Passing page/pageSize would be a
  // no-op at best and misleading at worst, so they are not accepted here.
  async listCloudAccounts(params?: { provider?: string }): Promise<unknown> {
    return request(this.baseURL, 'GET', '/cmdb/cloud-accounts', undefined, params as Record<string, string | undefined>);
  }
  async getCloudAccount(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/cmdb/cloud-accounts/${id}`);
  }
  async createCloudAccount(data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', '/cmdb/cloud-accounts', data);
  }
  async updateCloudAccount(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/cmdb/cloud-accounts/${id}`, data);
  }
  async deleteCloudAccount(id: number | string): Promise<void> {
    await request<void>(this.baseURL, 'DELETE', `/cmdb/cloud-accounts/${id}`);
  }
  async listCloudServices(params?: { provider?: string }): Promise<unknown> {
    return request(this.baseURL, 'GET', '/cmdb/cloud-services', undefined, params as Record<string, string | undefined>);
  }
  async getCloudService(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/cmdb/cloud-services/${id}`);
  }
  async createCloudService(data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', '/cmdb/cloud-services', data);
  }
  async updateCloudService(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/cmdb/cloud-services/${id}`, data);
  }
  async deleteCloudService(id: number | string): Promise<void> {
    await request<void>(this.baseURL, 'DELETE', `/cmdb/cloud-services/${id}`);
  }
  async listCloudResources(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/cmdb/cloud-resources', undefined, params);
  }
  async getCloudResource(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/cmdb/cloud-resources/${id}`);
  }
  async createCloudResource(data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', '/cmdb/cloud-resources', data);
  }
  async updateCloudResource(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/cmdb/cloud-resources/${id}`, data);
  }
  async deleteCloudResource(id: number | string): Promise<void> {
    await request<void>(this.baseURL, 'DELETE', `/cmdb/cloud-resources/${id}`);
  }
  async listDiscoverySources(): Promise<unknown> {
    return request(this.baseURL, 'GET', '/cmdb/discovery/sources');
  }
  async createDiscoverySource(data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', '/cmdb/discovery/sources', data);
  }
  async listDiscoveryJobs(params?: { status?: string; page?: number; pageSize?: number }): Promise<unknown> {
    return request(this.baseURL, 'GET', '/cmdb/discovery/jobs', undefined, params as Record<string, string | number | undefined>);
  }
  async getDiscoveryJob(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/cmdb/discovery/jobs/${id}`);
  }
  async createDiscoveryJob(data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', '/cmdb/discovery/jobs', data);
  }
  async cancelDiscoveryJob(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'POST', `/cmdb/discovery/jobs/${id}/cancel`, {});
  }
  async retryDiscoveryJob(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'POST', `/cmdb/discovery/jobs/${id}/retry`, {});
  }
  // `job_id` (snake) is what the handler reads off the query string; the DTO
  // field is an int, so an absent/unparseable value must be omitted rather than
  // sent as 0 — 0 would read as "filter by job 0" and return nothing.
  async listDiscoveryResults(params?: PaginationParams & { jobId?: number }): Promise<unknown> {
    const query: Record<string, string | number | undefined> = {};
    if (params?.jobId !== undefined) query['job_id'] = params.jobId;
    if (params?.page !== undefined) query['page'] = params.page;
    if (params?.pageSize !== undefined) query['pageSize'] = params.pageSize;
    return request(this.baseURL, 'GET', '/cmdb/discovery/results', undefined, query);
  }
  async applyDiscoveryResult(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'POST', `/cmdb/discovery/results/${id}/apply`, {});
  }
  async rejectDiscoveryResult(id: number | string, reason?: string): Promise<unknown> {
    return request(this.baseURL, 'POST', `/cmdb/discovery/results/${id}/reject`, reason ? { reason } : {});
  }

  /**
   * Discovery could not auto-create CIs for these resources (missing required
   * attributes); they wait for a human/agent to complete the record via
   * registerPendingDiscoveryResource. Standard `{items,total,page,pageSize,
   * totalPages}` list (query: page/pageSize).
   */
  async listPendingDiscoveryResources(params?: PaginationParams): Promise<unknown> {
    const query: Record<string, string | number | undefined> = {};
    if (params?.page !== undefined) query['page'] = params.page;
    if (params?.pageSize !== undefined) query['pageSize'] = params.pageSize;
    return request(this.baseURL, 'GET', '/cmdb/discovery/pending-resources', undefined, query);
  }
  /**
   * Turn one pending resource into a CI. Body is optional — omitting it keeps
   * the tag-inferred values; per-key `attributes` override same-named tags.
   * `environment` accepts exactly production | staging | development.
   * 409 when the resource is already linked to a CI.
   */
  async registerPendingDiscoveryResource(
    id: number | string,
    data?: { name?: string; environment?: 'production' | 'staging' | 'development'; assignedTo?: string; attributes?: Record<string, unknown> },
  ): Promise<unknown> {
    return request(this.baseURL, 'POST', `/cmdb/discovery/pending-resources/${id}/register`, data ?? {});
  }

  // ---------- Knowledge ----------
  async listKnowledge(params?: PaginationParams & { category?: string }): Promise<KnowledgeListResponse> {
    return request<KnowledgeListResponse>(this.baseURL, 'GET', '/knowledge/articles', undefined, params as Record<string, string | number | undefined>);
  }
  /**
   * The backend binds `query` with `binding:"required"` and reads paging from
   * the JSON body (not the query string), so `{q}` — or `pageSize` passed as a
   * query param — is a 400.
   */
  async searchKnowledge(query: string, opts?: { category?: string; pageSize?: number }): Promise<KnowledgeListResponse> {
    const payload: Record<string, unknown> = { query };
    if (opts?.category) payload['category'] = opts.category;
    if (opts?.pageSize !== undefined) payload['pageSize'] = opts.pageSize;
    return request<KnowledgeListResponse>(this.baseURL, 'POST', '/knowledge/search', payload);
  }
  async getKnowledgeArticle(id: number | string): Promise<KnowledgeArticle> {
    return request<KnowledgeArticle>(this.baseURL, 'GET', `/knowledge/articles/${id}`);
  }
  async createKnowledgeArticle(data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', '/knowledge/articles', data);
  }
  async updateKnowledgeArticle(id: number | string, data: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/knowledge/articles/${id}`, data);
  }
  async deleteKnowledgeArticle(id: number | string): Promise<void> {
    await request<void>(this.baseURL, 'DELETE', `/knowledge/articles/${id}`);
  }
  async knowledgeLifecycle(id: number | string, action: 'publish' | 'unpublish'): Promise<unknown> {
    return request(this.baseURL, 'POST', `/knowledge/articles/${id}/${action}`);
  }
  /** Editorial review workflow: submit → approve/reject, with withdraw/republish for revisions. */
  async knowledgeReviewAction(id: number | string, action: 'submit' | 'approve' | 'reject' | 'withdraw' | 'republish', data?: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', `/knowledge/articles/${id}/review/${action}`, data ?? {});
  }
  async listKnowledgeReviews(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/knowledge/articles/${id}/reviews`);
  }
  async listKnowledgeCategories(): Promise<unknown> {
    return request(this.baseURL, 'GET', '/knowledge/categories');
  }
  async listKnowledgeVersions(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/knowledge/articles/${id}/versions`);
  }
  async getKnowledgeStats(): Promise<unknown> { return request(this.baseURL, 'GET', '/knowledge/stats'); }

  // ---------- SLA ----------
  async getSLAStats(): Promise<unknown> { return request(this.baseURL, 'GET', '/sla/stats'); }
  async listSLADefinitions(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/sla/definitions', undefined, params);
  }
  async getSLADefinition(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/sla/definitions/${id}`);
  }
  async getSLAMetrics(): Promise<unknown> { return request(this.baseURL, 'GET', '/sla/metrics'); }
  async getSLAViolations(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/sla/violations', undefined, params);
  }
  async getSLAMonitoring(): Promise<unknown> { return request(this.baseURL, 'POST', '/sla/monitoring'); }
  async getSLAPerformance(): Promise<unknown> { return request(this.baseURL, 'GET', '/sla/performance'); }
  async getSLAAlertHistory(): Promise<unknown> { return request(this.baseURL, 'GET', '/sla/alert-history'); }
  async getSLAComplianceReport(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    // startDate/endDate are mandatory here (handlers/sla/handler.go:640) — unlike
    // most list endpoints, omitting them is a 400, not an empty default range.
    return request(this.baseURL, 'GET', '/sla/compliance-report', undefined, params);
  }
  async checkSLACompliance(ticketId: number | string): Promise<unknown> {
    return request(this.baseURL, 'POST', `/sla/check-compliance/${ticketId}`);
  }
  async listSLATemplates(): Promise<unknown> { return request(this.baseURL, 'GET', '/sla/templates'); }
  async installSLATemplate(key: string): Promise<unknown> {
    return request(this.baseURL, 'POST', `/sla/templates/${key}/install`);
  }

  // ---------- Workflow / BPMN ----------
  async listProcessInstances(params?: PaginationParams): Promise<ProcessInstanceListResponse> {
    return request<ProcessInstanceListResponse>(this.baseURL, 'GET', '/bpmn/process-instances', undefined, params as Record<string, string | number | undefined>);
  }
  async getProcessInstance(id: string): Promise<ProcessInstance> {
    return request<ProcessInstance>(this.baseURL, 'GET', `/bpmn/process-instances/${id}`);
  }
  async startProcess(data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', '/bpmn/process-instances', data);
  }
  async processInstanceAction(id: string, action: 'suspend' | 'resume' | 'terminate', data?: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/bpmn/process-instances/${id}/${action}`, data);
  }
  async listBPMNDefinitions(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/bpmn/process-definitions', undefined, params);
  }
  async getBPMNDefinition(key: string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/bpmn/process-definitions/${key}`);
  }
  /**
   * `GET /bpmn/tasks` is "my tasks" (assignee/candidate identity comes from the
   * token, never from the query). `status` is optional server-side and — when
   * omitted — returns completed tasks too, so a caller that means "待办" has to
   * say so.
   */
  async listUserTasks(params?: PaginationParams & { status?: string; processDefinitionKey?: string }): Promise<unknown> {
    return request(this.baseURL, 'GET', '/bpmn/tasks', undefined, params as Record<string, string | number | boolean | undefined>);
  }
  async getTask(id: string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/bpmn/tasks/${id}`);
  }
  async assignTask(id: string, assigneeId: number): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/bpmn/tasks/${id}/assign`, { assigneeId });
  }
  async decideTask(id: string, data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', `/bpmn/tasks/${id}/decisions`, data);
  }
  async cancelTask(id: string, reason: string): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/bpmn/tasks/${id}/cancel`, { reason });
  }
  async setProcessInstanceVariables(id: string, variables: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/bpmn/process-instances/${id}/variables`, { variables });
  }
  async completeTask(id: string, data?: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/bpmn/tasks/${id}/complete`, data ?? {});
  }
  async claimTask(id: string, data?: unknown): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/bpmn/tasks/${id}/claim`, data ?? {});
  }

  // ---------- Approval workflows ----------
  async listApprovalWorkflows(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/approval-workflows', undefined, params);
  }
  async getApprovalWorkflow(id: string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/approval-workflows/${id}`);
  }
  async listApprovalRecords(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/approval-records', undefined, params);
  }
  async submitApproval(data: unknown): Promise<unknown> {
    return request(this.baseURL, 'POST', '/approvals/submit', data);
  }

  // ---------- Notifications ----------
  async listNotifications(params?: PaginationParams & { unread?: boolean }): Promise<NotificationListResponse> {
    return request<NotificationListResponse>(this.baseURL, 'GET', '/notifications', undefined, params as unknown as Record<string, string | number | boolean | undefined>);
  }
  async getUnreadNotificationCount(): Promise<unknown> {
    return request(this.baseURL, 'GET', '/notifications/unread-count');
  }
  async markNotificationRead(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'PUT', `/notifications/${id}/read`);
  }
  async markAllNotificationsRead(): Promise<unknown> {
    return request(this.baseURL, 'PUT', '/notifications/read-all');
  }
  async deleteNotification(id: number | string): Promise<void> {
    await request<void>(this.baseURL, 'DELETE', `/notifications/${id}`);
  }

  // ---------- Connectors / IM / Marketplace ----------
  async listConnectors(): Promise<{ items: ConnectorManifest[]; total: number }> {
    return request<{ items: ConnectorManifest[]; total: number }>(this.baseURL, 'GET', '/connectors');
  }
  async listConnectorConfigs(): Promise<{ items: unknown[]; total: number }> {
    return request<{ items: unknown[]; total: number }>(this.baseURL, 'GET', '/connectors/configs');
  }
  async provisionConnector(cfg: { name: string; provider: string; enabled: boolean; credentials: Record<string, string>; settings: Record<string, unknown> }): Promise<unknown> {
    return request<unknown>(this.baseURL, 'POST', '/connectors/configs', cfg);
  }
  async revokeConnector(name: string, provider?: string): Promise<void> {
    await request<void>(this.baseURL, 'DELETE', `/connectors/configs/${name}`, undefined, provider ? { provider } : undefined);
  }
  async testConnector(name: string): Promise<unknown> {
    return request(this.baseURL, 'POST', `/connectors/${name}/test`);
  }
  async sendViaConnector(name: string, payload: { channel: string; type: string; title?: string; content: string; card?: unknown }): Promise<unknown> {
    return request(this.baseURL, 'POST', `/connectors/${name}/send`, payload);
  }
  async connectorHealth(): Promise<unknown> {
    return request(this.baseURL, 'GET', '/connectors/health');
  }

  // ---------- Notifications / deliveries ----------
  async listNotificationDeliveries(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/notification-deliveries', undefined, params);
  }
  async getDeliveryStats(): Promise<unknown> {
    return request(this.baseURL, 'GET', '/notification-deliveries/stats');
  }
  async retryDelivery(id: string): Promise<unknown> {
    return request(this.baseURL, 'POST', `/notification-deliveries/${id}/retry`);
  }

  // ---------- Admin / org / audit ----------
  async getDashboardOverview(): Promise<unknown> {
    return request(this.baseURL, 'GET', '/dashboard/overview');
  }
  async listUsers(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/users', undefined, params);
  }
  async getUser(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/users/${id}`);
  }
  async listTeams(): Promise<unknown> { return request(this.baseURL, 'GET', '/teams'); }
  async listDepartments(): Promise<unknown> { return request(this.baseURL, 'GET', '/departments'); }
  async listRoles(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/roles', undefined, params);
  }
  async listPermissions(): Promise<unknown> { return request(this.baseURL, 'GET', '/permissions'); }
  async listTenants(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/tenants', undefined, params);
  }
  async getTenant(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/tenants/${id}`);
  }
  async listAuditLogs(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/audit-logs', undefined, params);
  }
  async listOperationalCommands(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/admin/operations/commands', undefined, params);
  }
  async getOperationalCommand(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'GET', `/admin/operations/commands/${id}`);
  }
  async replayOperationalCommand(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'POST', `/admin/operations/commands/${id}/replay`);
  }
  async cancelOperationalCommand(id: number | string): Promise<unknown> {
    return request(this.baseURL, 'POST', `/admin/operations/commands/${id}/cancel`);
  }
  async bulkReplayOperationalCommands(data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', '/admin/operations/commands/bulk-replay', data);
  }
  async bulkCancelOperationalCommands(data: Record<string, unknown>): Promise<unknown> {
    return request(this.baseURL, 'POST', '/admin/operations/commands/bulk-cancel', data);
  }

  // ---------- Timers (SLA / escalation durable timers) ----------
  async listTimers(params?: Record<string, string | number | boolean | undefined>): Promise<unknown> {
    return request(this.baseURL, 'GET', '/timers', undefined, params);
  }
  async getTimerStats(): Promise<unknown> { return request(this.baseURL, 'GET', '/timers/stats'); }
  async getTimer(id: number | string): Promise<unknown> { return request(this.baseURL, 'GET', `/timers/${id}`); }

  // ---------- MSP (cross-tenant) ----------
  async getMSPStatus(): Promise<unknown> { return request(this.baseURL, 'GET', '/msp/status'); }
  async getMSPContext(): Promise<unknown> { return request(this.baseURL, 'GET', '/msp/context'); }
  async listMSPCustomers(): Promise<unknown> { return request(this.baseURL, 'GET', '/msp/customers'); }
  async listMSPAllocations(): Promise<unknown> { return request(this.baseURL, 'GET', '/msp/allocations'); }
}

export const apiClient = new ApiClient();

/** Some public routes ({status,timestamp}) return bare JSON; others wrap {code,message,data}. Handle both. */
function unwrapBody(json: unknown): unknown {
  if (json && typeof json === 'object' && 'code' in json && 'data' in json) {
    const j = json as { code: number; data?: unknown };
    if (j.code === 0) return j.data;
  }
  return json;
}

/** Pull one cookie value out of Set-Cookie header values. */
function extractCookie(cookies: string[], name: string): string | undefined {
  for (const raw of cookies) {
    for (const part of raw.split(/,(?=[A-Za-z0-9_\-]+=[^;=]+(?:;|$))/)) {
      const pair = part.trim().split(';')[0];
      const eq = pair.indexOf('=');
      if (eq > 0 && pair.slice(0, eq).trim() === name) {
        return pair.slice(eq + 1).trim() || undefined;
      }
    }
  }
  return undefined;
}
