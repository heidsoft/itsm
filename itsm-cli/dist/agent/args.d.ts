/**
 * Minimal argv parser for the headless agent layer. Agent mode intentionally
 * does not reuse meow (the Ink TUI parser): meow needs flags predeclared and
 * rejects/merges unknown flags unpredictably, which is fragile for a protocol
 * surface. This parser is strict and transparent: every token maps to either
 * a positional argument or a typed flag.
 */
export interface ParsedArgs {
    positional: string[];
    flags: Record<string, string | boolean | number>;
    /** Global agent-protocol flags, extracted even before command dispatch. */
    json: boolean;
    yes: boolean;
    dryRun: boolean;
    baseUrl?: string;
    /** Per-request timeout in milliseconds (default 30s). */
    timeoutMs?: number;
    /** Transport retry attempts for idempotent requests (default 2). */
    retries?: number;
    /** Output projection: only these top-level fields of each object survive. */
    fields?: string[];
    /** Output cap: at most N items of a list payload survive. */
    limit?: number;
    /** `json` = one envelope (default); `ndjson` = one JSON object per line for list payloads. */
    format: 'json' | 'ndjson';
}
/** The subset every command implementation receives ({positional, flags}). */
export interface ArgContext {
    positional: string[];
    flags: Record<string, string | boolean | number>;
}
export declare function parseArgs(argv: string[]): ParsedArgs;
/** String flag accessors used by command implementations. */
export declare function stringFlag(flags: ParsedArgs['flags'], ...names: string[]): string | undefined;
export declare function booleanFlag(flags: ParsedArgs['flags'], ...names: string[]): boolean | undefined;
export declare function numberFlag(flags: ParsedArgs['flags'], ...names: string[]): number | undefined;
/**
 * Comma-separated integer list (`--ids 1,2,3`).
 *
 * numberFlag cannot serve this: Number("1,2,3") is NaN, so a bulk flag would be
 * silently reported as missing and the command would exit 2 on a valid call.
 * Returns undefined when absent or when any segment is not a finite integer —
 * a partially-valid list is worse than none, since it would silently update the
 * wrong subset of records.
 */
export declare function numberListFlag(flags: ParsedArgs['flags'], ...names: string[]): number[] | undefined;
export declare function requirePositional(args: ArgContext, index: number, label: string): string;
/** Parse a JSON argument supplied inline (e.g. --data '{"summary":"..."}'). */
export declare function jsonFlag(flags: ParsedArgs['flags'], names?: string[]): Record<string, unknown> | undefined;
