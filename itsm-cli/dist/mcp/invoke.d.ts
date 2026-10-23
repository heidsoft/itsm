/**
 * The single execution gateway for MCP tool calls.
 *
 * Every path — curated tool, `itsm_call`, future resources — funnels through
 * `callCommand`, so the invariants are enforced once:
 *
 *   · the command exists in the CLI registry (no second endpoint table),
 *   · arguments are validated against the declared contract before any request,
 *   · high-risk writes never execute without an explicit `confirm`,
 *   · failures keep their classification (`forbidden` ≠ `not_found`) and hint,
 *   · list payloads are bounded so one call cannot flood the model's context.
 */
import type { CommandSpec } from '../agent/spec.js';
export interface CallOptions {
    confirm?: boolean;
    preview?: boolean;
    limit?: number;
    fields?: string[];
}
export interface CallResult {
    isError: boolean;
    /** Envelope-shaped payload, serialised to the tool's text content. */
    payload: Record<string, unknown>;
    structured: Record<string, unknown>;
}
interface Invocation {
    positional: string[];
    flags: Record<string, string | boolean | number>;
}
/**
 * Map MCP arguments onto the CLI's (positional, flags) shape using the
 * command's own declared parameters — the only place that knows what an
 * argument is called and whether it takes a value.
 */
export declare function buildInvocation(spec: CommandSpec, params: Record<string, unknown>): Invocation;
export declare function callCommand(command: string, params: Record<string, unknown> | undefined, opts?: CallOptions): Promise<CallResult>;
export {};
