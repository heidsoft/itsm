import { cmd } from '../spec.js';
import type { ParamSpec } from '../params.js';
import { apiClient } from '../../lib/api-client.js';
import type { ApiClient } from '../../lib/api-client.js';
import { resolveRuntimeConfig } from '../../lib/runtime.js';
import { classifyError } from '../../lib/errors.js';

/**
 * `itsm doctor` — one call that tells an agent where it is before it acts.
 *
 * An agent that starts cold has three unknowns: can I reach the backend, who am
 * I, and what am I allowed to read? Today that takes four commands and manual
 * interpretation (`infra health`, `infra version`, `infra me`, then guess at a
 * domain). Doctor answers all three in one envelope, in parallel, and always
 * exits 0 when it produced a report — a diagnosis is a result, not a failure.
 */

interface CheckResult {
  name: string;
  ok: boolean;
  ms: number;
  detail?: string;
  kind?: string;
  hint?: string;
}

const PROBES: Array<{ name: string; run: (client: ApiClient) => Promise<unknown> }> = [
  { name: 'read:tickets', run: (c) => c.listTickets({ page: 1, pageSize: 1 }) },
  { name: 'read:incidents', run: (c) => c.listIncidents({ page: 1, pageSize: 1 }) },
  { name: 'read:problems', run: (c) => c.listProblems({ page: 1, pageSize: 1 }) },
  { name: 'read:changes', run: (c) => c.listChanges({ page: 1, pageSize: 1 }) },
  { name: 'read:cmdb', run: (c) => c.listCIs({ page: 1, pageSize: 1 }) },
  { name: 'read:knowledge', run: (c) => c.listKnowledge({ page: 1, pageSize: 1 }) },
  { name: 'read:notifications', run: (c) => c.listNotifications({ page: 1, pageSize: 1 }) },
];

async function probe(name: string, fn: () => Promise<unknown>): Promise<CheckResult> {
  const started = Date.now();
  try {
    const value = await fn();
    const ms = Date.now() - started;
    const detail = summarize(value);
    return { name, ok: true, ms, ...(detail ? { detail } : {}) };
  } catch (err) {
    const { agentError } = classifyError(err);
    return {
      name,
      ok: false,
      ms: Date.now() - started,
      kind: agentError.kind,
      detail: agentError.message,
      ...(agentError.hint ? { hint: agentError.hint } : {}),
    };
  }
}

/** Compact "what came back" line so a passing probe still proves it returned data. */
function summarize(value: unknown): string {
  if (Array.isArray(value)) return `${value.length} item(s)`;
  if (value && typeof value === 'object') {
    const obj = value as Record<string, unknown>;
    if (typeof obj['total'] === 'number' && Array.isArray(obj['items'])) {
      return `${obj['items'].length} of ${obj['total']} item(s)`;
    }
    const keys = Object.keys(obj);
    return keys.length ? `keys: ${keys.slice(0, 6).join(', ')}${keys.length > 6 ? '…' : ''}` : 'empty object';
  }
  return String(value ?? '');
}

const PARAMS: ParamSpec[] = [
  {
    name: 'skipProbes',
    kind: 'flag',
    type: 'boolean',
    required: false,
    description: 'Skip per-domain read probes (connectivity + identity only).',
  },
];

export function doctorCommands() {
  return [
    cmd(
      'infra.doctor',
      'One-call orientation: reachability, version, identity, and per-domain read permission probes',
      'itsm doctor [--skipProbes]',
      [
        'GET /api/v1/health',
        'GET /api/v1/version',
        'GET /api/v1/auth/me',
        'GET /api/v1/tickets',
        'GET /api/v1/incidents',
        'GET /api/v1/problems',
        'GET /api/v1/changes',
        'GET /api/v1/cmdb/cis',
        'GET /api/v1/knowledge/articles',
        'GET /api/v1/notifications',
      ],
      async (ctx) => {
        const cfg = resolveRuntimeConfig();
        const client = apiClient;
        const skipProbes = ctx.args.flags['skipProbes'] === true || ctx.args.flags['skip-probes'] === true;

        let identityRaw: unknown = null;
        const [health, version, identity] = await Promise.all([
          probe('reachability', () => client.getHealth()),
          probe('version', () => client.getVersion()),
          probe('identity', async () => {
            identityRaw = await client.getUserInfo();
            return identityRaw;
          }),
        ]);

        const probes = skipProbes ? [] : await Promise.all(PROBES.map((p) => probe(p.name, () => p.run(client))));
        const checks = [health, version, identity, ...probes];

        const unreachable = !health.ok && health.kind === 'network';
        const unauthenticated = !identity.ok && identity.kind === 'not_authenticated';
        const failed = checks.filter((c) => !c.ok);

        const verdict = unreachable ? 'unreachable' : unauthenticated ? 'unauthenticated' : failed.length ? 'degraded' : 'healthy';

        return {
          baseUrl: cfg.baseURL,
          authSource: cfg.authSource,
          verdict,
          summary: {
            checks: checks.length,
            passed: checks.length - failed.length,
            failed: failed.length,
            ...(unreachable || unauthenticated ? { blockedOn: unreachable ? 'network' : 'authentication' } : {}),
          },
          identity: identity.ok ? identityRaw : null,
          checks,
          next:
            verdict === 'unreachable'
              ? 'Backend is not reachable. Verify ITSM_BASE_URL / --base-url, then retry.'
              : verdict === 'unauthenticated'
                ? 'Run `itsm --json auth login --username <u> --password <p>` or set ITSM_TOKEN.'
                : verdict === 'degraded'
                  ? 'Some domains are unavailable to this identity. Read each failed check hint before attempting that domain.'
                  : 'All checks passed. Continue with `itsm --json schema <command>` to plan the next call.',
        };
      },
      'normal',
      { params: PARAMS }
    ),
  ];
}
