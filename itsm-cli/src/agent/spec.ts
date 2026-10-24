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

const RISK_SKIP = new Set(['read', 'list', 'get', 'view', 'search', 'stats', 'health']);

/** Read-only HTTP methods can never be gated, whatever the verb is named. */
const READ_METHODS = new Set(['GET', 'HEAD', 'OPTIONS']);

/**
 * Derive risk from the verb when a spec doesn't declare it.
 *
 * `methods` is the set of HTTP methods the command's endpoints declare. The verb
 * regex alone mis-classified read-only commands whose names happen to contain a
 * mutating word — `change.rollback-steps` (GET /changes/:id/rollback-steps) was
 * flagged high-risk and demanded --yes just to list a checklist. Method wins.
 */
export function deriveRisk(verb: string, methods?: ReadonlySet<string>): RiskLevel {
  const onlyReads = methods && methods.size > 0 && [...methods].every((m) => READ_METHODS.has(m));
  if (onlyReads) return 'normal';
  if (RISK_SKIP.has(verb)) return 'normal';
  if (/delete|revoke|terminate|cancel|close|bulk|batch|rollback|replay|archive|install/.test(verb)) return 'high-risk-write';
  if (/create|update|patch|assign|resolve|approve|reject|submit|complete|claim|start|schedule|publish|send|provision|enable|disable|restore|reopen|escalate|withdraw|pause|resume|unpublish|status|accept|forward|convert/.test(verb)) return 'write';
  return 'normal';
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
    risk: risk ?? deriveRisk(verb, methodsOf(endpoints)),
    examples: meta?.examples,
    input: { params, ...(derived.body ? { body: derived.body } : {}) },
    bodyFields: meta?.bodyFields,
    ...(meta?.denyFlags ? { denyFlags: meta.denyFlags } : {}),
    run,
  };
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
    requiresConfirmation: spec.risk === 'high-risk-write',
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

