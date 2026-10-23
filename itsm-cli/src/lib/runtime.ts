import { isLoggedIn, hasRefreshableSession } from './credentials.js';
import { setExternalTokenProvider, assertTransportSafety } from './api-client.js';
import { redactSecrets, readSecretFile } from './secrets.js';

/**
 * Runtime configuration for the CLI, resolved from (highest priority first):
 *   1. CLI flags (--base-url)
 *   2. Environment variables (ITSM_BASE_URL, ITSM_TOKEN[_FILE], ITSM_USERNAME,
 *      ITSM_PASSWORD[_FILE])
 *   3. Stored credentials file (~/.itsm/credentials or ITSM_CREDENTIALS_FILE)
 *
 * The agent/headless mode relies on env vars because a sandboxed or CI agent
 * must never depend on a prior interactive `itsm login` run. Secrets should be
 * supplied through the `*_FILE` variants where the scheduler supports it: env
 * values are visible to every process running as the same user.
 */

export interface RuntimeConfig {
  baseURL: string;
  /** Bearer token when provided statically via env; undefined to fall back to stored credentials. */
  token?: string;
  /** Username/password pair for non-interactive login; agent auto-login with token-less setups. */
  username?: string;
  password?: string;
  /** Source of the auth material, surfaced in `itsm auth status` and error hints. */
  authSource: 'env-token' | 'env-password' | 'stored-credentials' | 'none';
}

export const DEFAULT_BASE_URL = 'http://localhost:8090';

/**
 * Why the most recent ITSM_USERNAME/ITSM_PASSWORD auto-login failed.
 * Auto-login failures used to be swallowed, which surfaced as a misleading
 * "缺少认证token" 401 — in practice it was often the backend login rate limit
 * (code 2003, 10/min/IP). The runner now folds this into the error hint.
 */
export interface AutoLoginFailure {
  /** 0 means the request never reached the backend (network/DNS/policy). */
  httpStatus: number;
  businessCode?: number;
  message: string;
}

let lastAutoLoginFailure: AutoLoginFailure | undefined;

export function getAutoLoginFailure(): AutoLoginFailure | undefined {
  return lastAutoLoginFailure;
}

function envSecret(valueName: string, fileName: string): string | undefined {
  const direct = (process.env[valueName] || '').trim();
  if (direct) return direct;
  const file = (process.env[fileName] || '').trim();
  if (!file) return undefined;
  try {
    return readSecretFile(file, valueName).trim() || undefined;
  } catch (err) {
    // Fail closed: a configured-but-unreadable secret file must not silently
    // downgrade to "no credentials" and produce a misleading 401 later.
    lastAutoLoginFailure = {
      httpStatus: 0,
      message: redactSecrets(err instanceof Error ? err.message : String(err)),
    };
    throw err;
  }
}

/**
 * Headless token hydration: when only ITSM_USERNAME/ITSM_PASSWORD are set
 * (no ITSM_TOKEN, no stored credentials), perform one login against the
 * backend and keep the access token in process memory for all subsequent
 * requests. The failure reason is recorded (not silently dropped) so the
 * follow-up 401 carries an actionable hint.
 */
export async function hydrateAgentToken(cfg: RuntimeConfig): Promise<void> {
  lastAutoLoginFailure = undefined;
  if (cfg.token) {
    setExternalTokenProvider(() => cfg.token);
    return;
  }
  if (cfg.username && cfg.password) {
    try {
      // Refuse to transmit the password over plaintext HTTP to a remote host.
      assertTransportSafety(cfg.baseURL, { sendingSecret: true });
      const res = await fetch(`${cfg.baseURL}/api/v1/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: cfg.username, password: cfg.password }),
      });
      const cookies = res.headers.getSetCookie?.() ?? [res.headers.get('set-cookie') ?? ''];
      const token = cookies.find((c) => c.trim().startsWith('access_token='))?.split(';')[0]?.split('=')[1];
      if (token) {
        setExternalTokenProvider(() => token);
        return;
      }
      let businessCode: number | undefined;
      let message = `login returned HTTP ${res.status} without an access_token cookie`;
      try {
        const body = (await res.json()) as { code?: number; message?: string };
        if (typeof body.code === 'number') businessCode = body.code;
        if (body.message) message = body.message;
      } catch {
        // Non-JSON error body — keep the status-based message.
      }
      lastAutoLoginFailure = { httpStatus: res.status, businessCode, message: redactSecrets(message) };
    } catch (e) {
      lastAutoLoginFailure = { httpStatus: 0, message: redactSecrets(e instanceof Error ? e.message : String(e)) };
    }
    return;
  }
  if (cfg.authSource === 'stored-credentials') {
    // request() falls back to the credentials file on its own.
  }
}

export function resolveRuntimeConfig(flags: { baseUrl?: string } = {}): RuntimeConfig {
  const baseURL = flags.baseUrl || process.env['ITSM_BASE_URL'] || DEFAULT_BASE_URL;
  const token = envSecret('ITSM_TOKEN', 'ITSM_TOKEN_FILE');
  const username = (process.env['ITSM_USERNAME'] || '').trim() || undefined;
  const password = envSecret('ITSM_PASSWORD', 'ITSM_PASSWORD_FILE');

  if (token) return { baseURL, token, authSource: 'env-token' };

  // An expired stored token must not shadow a fresh env-password login:
  // replaying it just burns a request on a guaranteed 401. A stored refresh
  // token keeps the session alive though — request() renews on the first 401.
  if (isLoggedIn()) {
    return { baseURL, authSource: 'stored-credentials' };
  }
  if (hasRefreshableSession()) {
    return { baseURL, authSource: 'stored-credentials' };
  }

  if (username && password) return { baseURL, username, password, authSource: 'env-password' };

  return { baseURL, authSource: 'none' };
}
