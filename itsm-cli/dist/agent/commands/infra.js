import { cmd } from '../spec.js';
import { requirePositional, jsonFlag } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
import { resolveRuntimeConfig } from '../../lib/runtime.js';
import { usageError } from '../../lib/errors.js';
/**
 * Infra tier: the first commands an agent runs to establish trust.
 * These work without login for health/version/readyz (public routes).
 */
export function infraCommands() {
    return [
        cmd('infra.health', 'Backend reachability smoke test (no auth required)', 'itsm infra health', ['GET /api/v1/health'], async () => apiClient.getHealth()),
        cmd('infra.ready', 'Initialization readiness (public)', 'itsm infra ready', ['GET /api/v1/readyz'], async () => apiClient.getReady()),
        cmd('infra.version', 'Backend version and build info', 'itsm infra version', ['GET /api/v1/version'], async () => apiClient.getVersion()),
        cmd('infra.capabilities', 'Platform capability probe (auth)', 'itsm infra capabilities', ['GET /api/v1/capabilities'], async () => apiClient.getCapabilities()),
        cmd('infra.me', 'Current authenticated identity, tenant and config source', 'itsm infra me', ['GET /api/v1/auth/me'], async () => {
            const cfg = resolveRuntimeConfig();
            const me = await apiClient.getUserInfo();
            return { authSource: cfg.authSource, baseURL: cfg.baseURL, user: me };
        }),
        cmd('infra.search', 'Global search across tickets/incidents/CI/knowledge', 'itsm infra search <query> [--page N]', ['GET /api/v1/global-search'], async (ctx) => {
            const q = requirePositional(ctx.args, 0, 'query');
            return apiClient.globalSearch(q);
        }),
        cmd('infra.api', 'Raw HTTP escape hatch for any /api/v1 endpoint (debugging flows not covered by first-class commands)', 'itsm infra api <METHOD> <path-relative-to-/api/v1> [--data <json>]', ['ANY /api/v1/*'], async (ctx) => {
            const method = requirePositional(ctx.args, 0, 'METHOD').toUpperCase();
            const relPath = requirePositional(ctx.args, 1, 'path');
            if (!/^\/[a-z0-9]/i.test(relPath))
                throw usageError('Path must start with / (relative to /api/v1)');
            const body = jsonFlag(ctx.args.flags);
            return apiClient.requestRaw(method, `/api/v1${relPath}`, body);
        }, 'write'),
    ];
}
