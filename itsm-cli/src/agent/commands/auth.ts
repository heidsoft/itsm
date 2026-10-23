import { cmd } from '../spec.js';
import { stringFlag, booleanFlag, numberFlag } from '../helpers.js';
import { apiClient, setExternalTokenProvider, describeTransport } from '../../lib/api-client.js';
import {
  saveCredentials,
  clearCredentials,
  loadCredentials,
  isLoggedIn,
  hasRefreshableSession,
  describeCredentialStore,
} from '../../lib/credentials.js';
import { resolveRuntimeConfig } from '../../lib/runtime.js';
import { CliError, usageError, classifyError } from '../../lib/errors.js';
import { resolveSecret, redactObject } from '../../lib/secrets.js';

/**
 * Auth surface.
 *
 * Security invariants (see src/lib/credentials.ts, src/lib/secrets.ts):
 *   · passwords are read from stdin or a 0600 file, never required on argv;
 *   · access/refresh tokens are persisted but NEVER returned by a command;
 *   · logout revokes the refresh token server-side before deleting the store;
 *   · tenant switch preserves the rotated refresh token (otherwise silent
 *     renewal dies right after the switch);
 *   · credentials are refused over plaintext HTTP to a remote host.
 */

const ACCESS_TTL_SECONDS = 15 * 60;

function expiresAtFrom(seconds: number | undefined): string | undefined {
  return typeof seconds === 'number' && seconds > 0
    ? new Date(Date.now() + seconds * 1000).toISOString()
    : undefined;
}

function passwordFromFlags(flags: Record<string, string | boolean | number>): string {
  const password = resolveSecret({
    literal: stringFlag(flags, 'password', 'p'),
    stdin: booleanFlag(flags, 'password-stdin') === true,
    file: stringFlag(flags, 'password-file'),
    fileEnv: 'ITSM_PASSWORD_FILE',
    valueEnv: 'ITSM_PASSWORD',
    label: 'password',
  });
  if (!password) {
    throw usageError(
      'Missing password. Pipe it with --password-stdin, point --password-file at a 0600 file, or set ITSM_PASSWORD_FILE.'
    );
  }
  return password;
}

function credentialSummary() {
  return describeCredentialStore();
}

export function authCommands() {
  return [
    // ---------- login / identity ----------
    cmd(
      'auth.login',
      'Authenticate and store credentials (password via stdin/file; or --token-stdin for a pre-issued token)',
      'itsm auth login [--username <u>] [--password <p> | --password-stdin | --password-file <path>] [--token-stdin] [--tenant-code <code>] [--tenant-id <n>]',
      ['POST /api/v1/auth/login', 'GET /api/v1/auth/me'],
      async (ctx) => {
        const flags = ctx.args.flags;
        const tokenMode = booleanFlag(flags, 'token-stdin') === true;
        if (tokenMode) {
          const token = resolveSecret({ stdin: true, label: 'token' });
          if (!token) throw usageError('No token received on stdin.');
          saveCredentials({ token, expiresAt: expiresAtFrom(ACCESS_TTL_SECONDS) });
          // Prove the token is accepted before declaring the identity; clear the
          // store again if it is not, so a failed login never leaves a bad token.
          setExternalTokenProvider(() => token);
          try {
            const user = await apiClient.getUserInfo();
            const stored = loadCredentials();
            if (stored) saveCredentials({ ...stored, user, tenantId: user.tenantId });
            return {
              identity: 'user',
              source: 'token-stdin',
              user,
              tenantId: user.tenantId,
              expiresAt: expiresAtFrom(ACCESS_TTL_SECONDS),
              credentialsStored: true,
              refreshable: false,
              credentialStore: credentialSummary(),
            };
          } catch (err) {
            clearCredentials();
            throw err;
          }
        }

        const username = stringFlag(flags, 'username', 'u');
        if (!username) throw usageError('Missing --username (or pass a pre-issued token with --token-stdin).');
        const password = passwordFromFlags(flags);
        const tenantId = numberFlag(flags, 'tenant-id');
        const tenantCode = stringFlag(flags, 'tenant-code');
        const result = await apiClient.login({
          username,
          password,
          ...(tenantCode ? { tenantCode } : {}),
          ...(tenantId ? { tenantId } : {}),
        });
        if (!result.accessToken) {
          throw new CliError({
            kind: 'not_authenticated',
            message: 'Login succeeded but the response carried no access_token cookie',
            retryable: false,
            hint: 'Check that the backend is reachable directly (not behind a proxy that strips Set-Cookie) and that cookies are enabled.',
          });
        }
        const expiresAt = expiresAtFrom(result.expiresIn) ?? expiresAtFrom(ACCESS_TTL_SECONDS);
        saveCredentials({
          token: result.accessToken,
          // A missing refresh token is not fatal for this process, but it means
          // the session dies with the 15-minute access token. Report it.
          ...(result.refreshToken ? { refreshToken: result.refreshToken } : {}),
          user: result.user,
          tenantId: result.user.tenantId,
          ...(expiresAt ? { expiresAt } : {}),
        });
        return {
          identity: 'user',
          source: 'password',
          user: result.user,
          tenantId: result.user.tenantId,
          expiresIn: result.expiresIn,
          expiresAt,
          credentialsStored: true,
          refreshable: Boolean(result.refreshToken),
          credentialStore: credentialSummary(),
        };
      }
    ),
    cmd(
      'auth.refresh',
      'Rotate the stored refresh token and persist the new access token (never prints tokens)',
      'itsm auth refresh',
      ['POST /api/v1/auth/refresh'],
      async () => {
        const renewed = await apiClient.refreshStoredSession();
        return {
          ok: true,
          expiresAt: renewed.expiresAt,
          expiresIn: renewed.expiresIn,
          refreshRotated: true,
          identity: loadCredentials()?.user ?? null,
          credentialStore: credentialSummary(),
        };
      }
    ),
    cmd(
      'auth.status',
      'Identity, tenant, credential store, transport safety and expiry of the current session',
      'itsm auth status',
      ['GET /api/v1/auth/me', 'GET /api/v1/auth/tenants', 'GET /api/v1/auth/session'],
      async () => {
        const cfg = resolveRuntimeConfig();
        const stored = loadCredentials();
        const store = credentialSummary();
        const transport = describeTransport(cfg.baseURL);
        const warnings: string[] = [];
        if (store.exists && !store.secure && store.reason) warnings.push(store.reason);
        if (!transport.credentialSafe && transport.reason) warnings.push(transport.reason);
        if (stored?.token && !isLoggedIn() && stored.refreshToken) {
          warnings.push('access token expired; the next authenticated request will rotate the refresh token');
        }

        let user: unknown = null;
        let authError: string | undefined;
        try {
          user = await apiClient.getUserInfo();
        } catch (e) {
          authError = e instanceof Error ? e.message : String(e);
        }
        let tenants: unknown = null;
        try {
          tenants = await apiClient.getUserTenants();
        } catch {
          // Best effort: tenants listing may be blocked by RBAC; status stays valid.
        }
        let serverExpiresIn: number | undefined;
        try {
          const session = (await apiClient.getSession()) as { expiresIn?: unknown };
          if (typeof session?.expiresIn === 'number') serverExpiresIn = session.expiresIn;
        } catch {
          // Access token may already be expired; local expiry remains useful.
        }

        return {
          identity: stored?.user?.username ? 'user' : cfg.authSource === 'none' ? 'anonymous' : 'user',
          authSource: cfg.authSource,
          baseURL: cfg.baseURL,
          transport,
          expiresAt: stored?.expiresAt ?? null,
          serverExpiresIn: serverExpiresIn ?? null,
          refreshable: hasRefreshableSession(),
          user,
          tenants,
          credentialStore: store,
          warnings,
          authError,
        };
      }
    ),
    cmd(
      'auth.logout',
      'Revoke the session server-side (refresh token + access token) and delete the local store',
      'itsm auth logout [--local-only]',
      ['POST /api/v1/auth/logout'],
      async (ctx) => {
        const stored = loadCredentials();
        const localOnly = booleanFlag(ctx.args.flags, 'local-only') === true;
        let serverRevocation: 'ok' | 'failed' | 'skipped' = 'skipped';
        let warning: string | undefined;

        if (!localOnly && (stored?.token || stored?.refreshToken)) {
          try {
            await apiClient.logout(stored?.refreshToken);
            serverRevocation = 'ok';
          } catch (err) {
            const classified = classifyError(err);
            serverRevocation = 'failed';
            warning =
              `Local credentials were deleted, but server-side revocation failed (${classified.agentError.kind}: ${classified.agentError.message}). ` +
              'The refresh token may remain valid until it expires; an administrator can force the account offline.';
          }
        }
        clearCredentials();
        return { ok: true, credentialsCleared: true, serverRevocation, ...(warning ? { warning } : {}), credentialStore: credentialSummary() };
      }
    ),
    cmd(
      'auth.switch-tenant',
      'Switch active tenant context (MSP / multi-tenant operators); rotates access AND refresh tokens',
      'itsm auth switch-tenant <tenantId>',
      ['POST /api/v1/auth/switch-tenant'],
      async (ctx) => {
        const tenantId = ctx.args.positional[0];
        if (!tenantId || !Number.isInteger(Number(tenantId)) || Number(tenantId) <= 0) {
          throw usageError('Usage: itsm auth switch-tenant <tenantId> (positive integer)');
        }
        const current = loadCredentials();
        if (!current) {
          throw new CliError({
            kind: 'not_authenticated',
            message: 'No stored session to switch',
            retryable: false,
            hint: 'Run `itsm auth login --password-stdin` first.',
          });
        }
        const result = await apiClient.switchTenant(tenantId);
        if (!result.accessToken) {
          throw new CliError({
            kind: 'not_authenticated',
            message: 'Tenant switch returned no access token',
            retryable: false,
            hint: 'The previous session is unchanged. Re-run `itsm auth login` if the session is stale.',
          });
        }
        const expiresAt = expiresAtFrom(ACCESS_TTL_SECONDS);
        saveCredentials({
          ...current,
          token: result.accessToken,
          // Backend rotates both cookies. Preserve the old refresh token only
          // when the response genuinely omitted it (older deployment).
          ...(result.refreshToken ? { refreshToken: result.refreshToken } : {}),
          user: result.user,
          tenantId: result.user.tenantId,
          ...(expiresAt ? { expiresAt } : {}),
        });
        return {
          ok: true,
          tenantId: result.user.tenantId,
          tenantName: (result.tenant as { name?: string } | null | undefined)?.name,
          credentialsRotated: true,
          refreshRotated: Boolean(result.refreshToken),
          expiresAt,
          credentialStore: credentialSummary(),
        };
      }
    ),
    cmd(
      'auth.session',
      'Session metadata: identity, tenants, authoritative expiry, menus',
      'itsm auth session',
      ['GET /api/v1/auth/session', 'GET /api/v1/auth/menus'],
      async () => {
        const session = await apiClient.getSession().catch(() => null);
        const menus = await apiClient.getUserMenus().catch(() => null);
        return { session, menus };
      }
    ),

    // ---------- self-service account lifecycle ----------
    cmd(
      'auth.register',
      'Self-service registration (server assigns the end_user role and tenant; no role/tenant in the body)',
      'itsm auth register --username <u> --email <e> [--display-name <name>] [--phone <p>] [--company <c>] [--password-stdin]',
      ['POST /api/v1/auth/register'],
      async (ctx) => {
        const flags = ctx.args.flags;
        const username = stringFlag(flags, 'username', 'u');
        const email = stringFlag(flags, 'email');
        if (!username || !email) throw usageError('Missing --username and --email.');
        const result = await apiClient.register({
          username,
          email,
          password: passwordFromFlags(flags),
          ...(stringFlag(flags, 'display-name', 'displayName') ? { displayName: stringFlag(flags, 'display-name', 'displayName') } : {}),
          ...(stringFlag(flags, 'phone') ? { phone: stringFlag(flags, 'phone') } : {}),
          ...(stringFlag(flags, 'company') ? { company: stringFlag(flags, 'company') } : {}),
        });
        return redactObject(result);
      }
    ),
    cmd(
      'auth.forgot-password',
      'Request a password-reset token by email (same response whether or not the account exists)',
      'itsm auth forgot-password --email <e> [--tenant-code <code>]',
      ['POST /api/v1/auth/forgot-password'],
      async (ctx) => {
        const email = stringFlag(ctx.args.flags, 'email');
        if (!email) throw usageError('Missing --email.');
        const tenantCode = stringFlag(ctx.args.flags, 'tenant-code');
        return apiClient.forgotPassword({ email, ...(tenantCode ? { tenantCode } : {}) });
      }
    ),
    cmd(
      'auth.validate-reset-token',
      'Check a password-reset token before showing the reset form',
      'itsm auth validate-reset-token --email <e> --token <t>',
      ['POST /api/v1/auth/validate-reset-token'],
      async (ctx) => {
        const email = stringFlag(ctx.args.flags, 'email');
        const token = stringFlag(ctx.args.flags, 'token');
        if (!email || !token) throw usageError('Missing --email and --token.');
        return apiClient.validateResetToken({ email, token });
      }
    ),
    cmd(
      'auth.reset-password',
      'Complete a password reset (passwordConfirm is derived from the same secret)',
      'itsm auth reset-password --email <e> --token <t> [--password-stdin]',
      ['POST /api/v1/auth/reset-password'],
      async (ctx) => {
        const email = stringFlag(ctx.args.flags, 'email');
        const token = stringFlag(ctx.args.flags, 'token');
        if (!email || !token) throw usageError('Missing --email and --token.');
        const password = passwordFromFlags(ctx.args.flags);
        return apiClient.resetPassword({ email, token, password, passwordConfirm: password });
      }
    ),

    // ---------- SSO (OIDC) ----------
    cmd(
      'auth.sso-providers',
      'List enabled SSO providers (public; no secrets returned)',
      'itsm auth sso-providers',
      ['GET /api/v1/auth/sso/providers'],
      async () => apiClient.listSSOProviders()
    ),
    cmd(
      'auth.sso-initiate',
      'Start an OIDC login; returns the IdP redirect URL and state',
      'itsm auth sso-initiate --provider <code> [--tenant-code <code>]',
      ['POST /api/v1/auth/sso/initiate'],
      async (ctx) => {
        const provider = stringFlag(ctx.args.flags, 'provider');
        if (!provider) throw usageError('Missing --provider.');
        const tenantCode = stringFlag(ctx.args.flags, 'tenant-code');
        return apiClient.initiateSSO({ provider, ...(tenantCode ? { tenantCode } : {}) });
      }
    ),
    cmd(
      'auth.sso-identities',
      'List the current user’s SSO identity bindings',
      'itsm auth sso-identities',
      ['GET /api/v1/auth/sso/identities'],
      async () => apiClient.listSSOIdentities()
    ),
    cmd(
      'auth.sso-bind',
      'Bind an SSO identity to the current account (provider + code + state from the callback)',
      'itsm auth sso-bind --provider <code> --code <c> --state <s>',
      ['POST /api/v1/auth/sso/bind'],
      async (ctx) => {
        const provider = stringFlag(ctx.args.flags, 'provider');
        const code = stringFlag(ctx.args.flags, 'code');
        const state = stringFlag(ctx.args.flags, 'state');
        if (!provider || !code || !state) throw usageError('Missing --provider, --code or --state.');
        return apiClient.bindSSO({ provider, code, state });
      }
    ),
    cmd(
      'auth.sso-unbind',
      'Remove an SSO identity binding from the current account',
      'itsm auth sso-unbind --provider <code>',
      ['POST /api/v1/auth/sso/unbind'],
      async (ctx) => {
        const provider = stringFlag(ctx.args.flags, 'provider');
        if (!provider) throw usageError('Missing --provider.');
        return apiClient.unbindSSO({ provider });
      }
    ),
  ];
}
