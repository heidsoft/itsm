/**
 * Agent command surface specification.
 *
 * A command is identified by `domain verb` (e.g. `ticket.create`, `bpmn.task-complete`).
 * Every spec carries its usage string and endpoint list so `itsm commands --json`
 * (capability discovery) and the route-drift check can derive everything from
 * the registry — command docs and endpoint snapshots must not be hand-maintained
 * in two places.
 */
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
/** Derive risk from the verb when a spec doesn't declare it. */
export declare function deriveRisk(verb: string): RiskLevel;
/** Compact builder for command specs — keeps domain modules declarative. */
export declare function cmd(name: string, summary: string, usage: string, endpoints: string[], run: (ctx: CommandContext) => Promise<unknown>, risk?: RiskLevel, meta?: CommandMeta): CommandSpec;
/** One command rendered for agents: what it does, what it needs, what it costs. */
export declare function describeCommand(spec: CommandSpec): Record<string, unknown>;
