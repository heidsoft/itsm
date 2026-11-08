/**
 * Agent command surface specification.
 *
 * A command is identified by `domain verb` (e.g. `ticket.create`, `bpmn.task-complete`).
 * Every spec carries its usage string and endpoint list so `itsm commands --json`
 * (capability discovery) and the route-drift check can derive everything from
 * the registry — command docs and endpoint snapshots must not be hand-maintained
 * in two places.
 */

import { deriveParams, mergeParams, extractEnums, applyEnums } from './params.js';
import type { CommandInputSchema, ParamSpec } from './params.js';

export type RiskLevel = 'normal' | 'write' | 'high-risk-write';

/** Optional richness a usage string cannot express (enums, defaults, examples). */
export interface CommandMeta {
  params?: ParamSpec[];
  examples?: string[];
  /** Names of the backend DTO fields `--data` accepts, when worth advertising. */
  bodyFields?: string[];
  /**
   * Flags that must be REJECTED with a specific explanation instead of being
   * reported as merely unknown. Used for secrets: `--password` on argv is a
   * security defect, and the caller deserves the reason plus the safe channel.
   * Map of flag → reason shown to the user.
   */
  denyFlags?: Record<string, string>;
  /**
   * Risk that depends on runtime input, not just the command name: values of
   * the FIRST positional argument that trigger the confirmation gate even
   * though the spec's static risk level is lower. `infra.api` is the user:
   * its METHOD positional decides whether the call is a read (`GET`) or an
   * unrecoverable delete. Enforcement lives inside spec.run at execution
   * time; this field exists so dry-run/preview/describe advertise the same
   * fact instead of a misleading "no confirmation needed".
   */
  confirmMethods?: string[];
}

export interface CommandSpec {
  name: string;
  summary: string;
  usage: string;
  /** Backend endpoint(s) this command hits, as `METHOD /api/v1/...`. Used by route-drift checks. */
  endpoints: string[];
  risk: RiskLevel;
  /** Examples rendered in help output. */
  examples?: string[];
  /** Machine-readable input contract: positionals, flags, body. */
  input: CommandInputSchema;
  /** Backend DTO fields accepted in `--data`, when declared. */
  bodyFields?: string[];
  /** Flags rejected by policy (secrets on argv), with the reason. */
  denyFlags?: Record<string, string>;
  /** First-positional values that turn this command's gate on (see CommandMeta). */
  confirmMethods?: string[];
  run: (ctx: CommandContext) => Promise<unknown>;
}

export interface CommandContext {
  args: {
    positional: string[];
    flags: Record<string, string | boolean | number>;
    yes: boolean;
    dryRun: boolean;
  };
}

const RISK_SKIP = new Set(['list', 'view', 'search', 'stats', 'health']);

/** Read-only HTTP methods can never be gated, whatever the verb is named. */
const READ_METHODS = new Set(['GET', 'HEAD', 'OPTIONS']);

/**
 * POST/PUT used as pure transport for a read or a session operation. These are
 * the only non-GET commands that must stay `normal`: analysis, session
 * management, and client-level search-by-body never mutate domain state. Every
 * other non-GET command defaults to `write` — an unknown mutating verb must
 * never be advertised read-only to a host (MCP hosts annotate `readOnlyHint`
 * from this level and then happily batch "read-only" calls).
 *
 * Reviewed per registry entry 2026-11 (auth.* session ops, AI decision-support
 * analysis, report-style aggregation, connection test, isListish search bodies).
 */
const READ_BY_POST = new Set([
  // Session management: reads/writes the caller's own token, not tenant data.
  'auth.login',
  'auth.refresh',
  'auth.logout',
  'auth.switch-tenant',
  'auth.register',
  'auth.forgot-password',
  'auth.reset-password',
  'auth.validate-reset-token',
  'auth.sso-initiate',
  'auth.sso-bind',
  'auth.sso-unbind',
  // AI decision support: returns a suggestion, changes nothing.
  'incident.triage',
  'ticket.ai-analyze',
  // Report/aggregation reads that take a POST body.
  'sla.monitoring',
  'sla.check-compliance',
  // POST search bodies (also RISK_SKIP verbs, listed explicitly for clarity).
  'cmdb.search',
  'knowledge.search',
  // Live connection check against the upstream — no connector state change.
  'connector.test',
]);

/**
 * Derive risk from the verb when a spec doesn't declare it.
 *
 * `methods` is the set of HTTP methods the command's endpoints declare. The verb
 * regex alone mis-classified read-only commands whose names happen to contain a
 * mutating word — `change.rollback-steps` (GET /changes/:id/rollback-steps) was
 * flagged high-risk and demanded --yes just to list a checklist. Method wins,
 * and conversely: a mutating method with an unknown verb must not fall back to
 * `normal` — it defaults to `write` (fail-safe), because 40+ registry commands
 * (comments.add, acknowledge, root-cause, lifecycle, suspend, tags-add …) had
 * verbs that were missing from the write regex and were advertised as reads.
 */
export function deriveRisk(name: string, verb: string, methods?: ReadonlySet<string>): RiskLevel {
  const onlyReads = methods && methods.size > 0 && [...methods].every((m) => READ_METHODS.has(m));
  if (onlyReads) return 'normal';
  if (READ_BY_POST.has(name)) return 'normal';
  if (RISK_SKIP.has(verb)) return 'normal';
  if (/delete|revoke|terminate|cancel|close|bulk|batch|rollback|revert|replay|archive|install/.test(verb)) return 'high-risk-write';
  if (/create|update|patch|assign|resolve|approve|reject|submit|complete|claim|start|schedule|publish|send|provision|enable|disable|restore|reopen|escalate|withdraw|pause|resume|unpublish|status|accept|forward|convert/.test(verb)) return 'write';
  // Fail-safe fallback: the endpoint method is a mutation the legacy verb list
  // never knew (comments.add, acknowledge, root-cause, investigate, solution,
  // fail, execute, retry, lifecycle, decision, cc, notify, remove …). Born as
  // 'normal', those tools read as read-only to hosts. Defaulting to `write`
  // keeps the gate honest for future verbs too.
  return 'write';
}

function methodsOf(endpoints: string[]): ReadonlySet<string> | undefined {
  const methods = new Set<string>();
  for (const e of endpoints) {
    const m = e.match(/^([A-Z]+)\s/);
    if (m) methods.add(m[1]);
  }
  return methods.size > 0 ? methods : undefined;
}

/** Compact builder for command specs — keeps domain modules declarative. */
export function cmd(
  name: string,
  summary: string,
  usage: string,
  endpoints: string[],
  run: (ctx: CommandContext) => Promise<unknown>,
  risk?: RiskLevel,
  meta?: CommandMeta
): CommandSpec {
  const verb = name.split('.').pop() ?? name;
  const derived = deriveParams(usage);
  const enums = extractEnums(`${summary} ${usage}`);
  const params = applyEnums(mergeParams(derived.params, meta?.params), enums);
  return {
    name,
    summary,
    usage,
    endpoints,
    risk: risk ?? deriveRisk(name, verb, methodsOf(endpoints)),
    examples: meta?.examples,
    input: { params, ...(derived.body ? { body: derived.body } : {}) },
    bodyFields: meta?.bodyFields,
    ...(meta?.denyFlags ? { denyFlags: meta.denyFlags } : {}),
    ...(meta?.confirmMethods ? { confirmMethods: meta.confirmMethods } : {}),
    run,
  };
}

/**
 * Whether THIS rendered invocation needs the confirmation gate — the single
 * fact shared by dry-run (CLI), preview (MCP) and describe:
 *   · static: high-risk-write always gates;
 *   · method-derived: confirmMethods compares against the first positional's
 *     value (infra.api's METHOD). With no/unknown value the gate cannot fire —
 *     describe-level callers just see the declared confirmMethods.
 * Execution-time enforcement stays in spec.run; this must never be weaker.
 */
export function needsConfirmation(spec: CommandSpec, firstPositional?: string): boolean {
  if (spec.risk === 'high-risk-write') return true;
  if (!spec.confirmMethods?.length) return false;
  if (!firstPositional) return false;
  // Case-insensitive: `infra.api` accepts `delete` as readily as `DELETE`, and
  // the gate must not depend on how the operator typed the verb.
  const normalized = firstPositional.toUpperCase();
  return spec.confirmMethods.some((method) => method.toUpperCase() === normalized);
}

/** One command rendered for agents: what it does, what it needs, what it costs. */
export function describeCommand(spec: CommandSpec): Record<string, unknown> {
  const required = spec.input.params.filter((p) => p.required);
  return {
    command: spec.name,
    summary: spec.summary,
    usage: spec.usage,
    risk: spec.risk,
    endpoints: spec.endpoints,
    requiresConfirmation: needsConfirmation(spec),
    ...(spec.confirmMethods ? { confirmMethods: spec.confirmMethods } : {}),
    input: {
      params: spec.input.params,
      ...(spec.input.body ? { body: spec.input.body } : {}),
      ...(spec.bodyFields?.length ? { bodyFields: spec.bodyFields } : {}),
      ...(spec.denyFlags ? { deniedFlags: spec.denyFlags } : {}),
      required: required.map((p) => p.name),
    },
    examples: spec.examples ?? [],
  };
}

