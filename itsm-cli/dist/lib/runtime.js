import { isLoggedIn, hasRefreshableSession } from './credentials.js';
import { setExternalTokenProvider, assertTransportSafety } from './api-client.js';
import { redactSecrets, readSecretFile } from './secrets.js';
export const DEFAULT_BASE_URL = 'http://localhost:8090';
let lastAutoLoginFailure;
export function getAutoLoginFailure() {
    return lastAutoLoginFailure;
}
function envSecret(valueName, fileName) {
    const direct = (process.env[valueName] || '').trim();
    if (direct)
        return direct;
    const file = (process.env[fileName] || '').trim();
    if (!file)
        return undefined;
    try {
        return readSecretFile(file, valueName).trim() || undefined;
    }
    catch (err) {
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
export async function hydrateAgentToken(cfg) {
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
            let businessCode;
            let message = `login returned HTTP ${res.status} without an access_token cookie`;
            try {
                const body = (await res.json());
                if (typeof body.code === 'number')
                    businessCode = body.code;
                if (body.message)
                    message = body.message;
            }
            catch {
                // Non-JSON error body — keep the status-based message.
            }
            lastAutoLoginFailure = { httpStatus: res.status, businessCode, message: redactSecrets(message) };
        }
        catch (e) {
            lastAutoLoginFailure = { httpStatus: 0, message: redactSecrets(e instanceof Error ? e.message : String(e)) };
        }
        return;
    }
    if (cfg.authSource === 'stored-credentials') {
        // request() falls back to the credentials file on its own.
    }
}
export function resolveRuntimeConfig(flags = {}) {
    const baseURL = flags.baseUrl || process.env['ITSM_BASE_URL'] || DEFAULT_BASE_URL;
    const token = envSecret('ITSM_TOKEN', 'ITSM_TOKEN_FILE');
    const username = (process.env['ITSM_USERNAME'] || '').trim() || undefined;
    const password = envSecret('ITSM_PASSWORD', 'ITSM_PASSWORD_FILE');
    if (token)
        return { baseURL, token, authSource: 'env-token' };
    // An expired stored token must not shadow a fresh env-password login:
    // replaying it just burns a request on a guaranteed 401. A stored refresh
    // token keeps the session alive though — request() renews on the first 401.
    if (isLoggedIn()) {
        return { baseURL, authSource: 'stored-credentials' };
    }
    if (hasRefreshableSession()) {
        return { baseURL, authSource: 'stored-credentials' };
    }
    if (username && password)
        return { baseURL, username, password, authSource: 'env-password' };
    return { baseURL, authSource: 'none' };
}
