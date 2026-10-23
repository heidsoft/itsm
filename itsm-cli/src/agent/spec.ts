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

/** Derive risk from the verb when a spec doesn't declare it. */
export function deriveRisk(verb: string): RiskLevel {
  if (RISK_SKIP.has(verb)) return 'normal';
  if (/delete|revoke|terminate|cancel|close|bulk|batch|rollback|replay|archive|install/.test(verb)) return 'high-risk-write';
  if (/create|update|patch|assign|resolve|approve|reject|submit|complete|claim|start|schedule|publish|send|provision|enable|disable|restore|reopen|escalate|withdraw|pause|resume|unpublish|status|accept|forward|convert/.test(verb)) return 'write';
  return 'normal';
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
    risk: risk ?? deriveRisk(verb),
    examples: meta?.examples,
    input: { params, ...(derived.body ? { body: derived.body } : {}) },
    bodyFields: meta?.bodyFields,
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
      required: required.map((p) => p.name),
    },
    examples: spec.examples ?? [],
  };
}

