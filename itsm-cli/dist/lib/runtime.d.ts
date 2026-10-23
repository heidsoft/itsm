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
export declare const DEFAULT_BASE_URL = "http://localhost:8090";
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
export declare function getAutoLoginFailure(): AutoLoginFailure | undefined;
/**
 * Headless token hydration: when only ITSM_USERNAME/ITSM_PASSWORD are set
 * (no ITSM_TOKEN, no stored credentials), perform one login against the
 * backend and keep the access token in process memory for all subsequent
 * requests. The failure reason is recorded (not silently dropped) so the
 * follow-up 401 carries an actionable hint.
 */
export declare function hydrateAgentToken(cfg: RuntimeConfig): Promise<void>;
export declare function resolveRuntimeConfig(flags?: {
    baseUrl?: string;
}): RuntimeConfig;
