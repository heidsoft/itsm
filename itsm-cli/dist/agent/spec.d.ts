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
/**
 * Derive risk from the verb when a spec doesn't declare it.
 *
 * `methods` is the set of HTTP methods the command's endpoints declare. The verb
 * regex alone mis-classified read-only commands whose names happen to contain a
 * mutating word — `change.rollback-steps` (GET /changes/:id/rollback-steps) was
 * flagged high-risk and demanded --yes just to list a checklist. Method wins.
 */
export declare function deriveRisk(verb: string, methods?: ReadonlySet<string>): RiskLevel;
/** Compact builder for command specs — keeps domain modules declarative. */
export declare function cmd(name: string, summary: string, usage: string, endpoints: string[], run: (ctx: CommandContext) => Promise<unknown>, risk?: RiskLevel, meta?: CommandMeta): CommandSpec;
/**
 * Whether THIS rendered invocation needs the confirmation gate — the single
 * fact shared by dry-run (CLI), preview (MCP) and describe:
 *   · static: high-risk-write always gates;
 *   · method-derived: confirmMethods compares against the first positional's
 *     value (infra.api's METHOD). With no/unknown value the gate cannot fire —
 *     describe-level callers just see the declared confirmMethods.
 * Execution-time enforcement stays in spec.run; this must never be weaker.
 */
export declare function needsConfirmation(spec: CommandSpec, firstPositional?: string): boolean;
/** One command rendered for agents: what it does, what it needs, what it costs. */
export declare function describeCommand(spec: CommandSpec): Record<string, unknown>;
