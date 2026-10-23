/**
 * Machine-readable input contracts for agent commands.
 *
 * An LLM driving the CLI cannot read TypeScript. It needs, for every command:
 * which positional arguments exist, which flags are accepted, which are
 * required, and what enum values are legal — before it spends a turn guessing.
 *
 * Two sources feed this:
 *   1. `deriveParams(usage)` parses the human usage string every command already
 *      declares (`itsm ticket create <title> <description> [--priority <p>]`).
 *      That gives 100% coverage without a second hand-maintained table.
 *   2. Optional explicit `params` on the command spec for enums, defaults and
 *      descriptions that the usage string cannot express.
 *
 * Explicit params win; derived params fill the gaps. Neither is authoritative
 * for the HTTP contract — the backend DTOs remain that — but they remove the
 * "call it and read the validation error" loop.
 */
export type ParamKind = 'positional' | 'flag' | 'body';
export type ParamType = 'string' | 'number' | 'boolean' | 'json' | 'enum';
export interface ParamSpec {
    name: string;
    kind: ParamKind;
    type: ParamType;
    required: boolean;
    description?: string;
    /** Legal values when `type === 'enum'`. */
    values?: string[];
    default?: string | number | boolean;
}
/** Everything an agent needs to construct a valid invocation. */
export interface CommandInputSchema {
    params: ParamSpec[];
    /** True when the command accepts (or requires) a `--data '<json>'` body. */
    body?: {
        flag: string;
        required: boolean;
        note: string;
    };
}
/**
 * Parse a usage string such as
 *   `itsm ticket list [--status s] [--priority p] [--page N]`
 * into ParamSpecs. Bracket nesting defines required vs optional.
 */
export declare function deriveParams(usage: string): CommandInputSchema;
/**
 * Merge derived params with explicit overrides. An explicit spec replaces the
 * derived entry of the same (name, kind) and appends anything new.
 */
export declare function mergeParams(derived: ParamSpec[], explicit?: ParamSpec[]): ParamSpec[];
/**
 * Enum extraction.
 *
 * Command summaries already spell out legal values in a consistent shape
 * (`priority required: low|medium|high`, `--action approve|reject|delegate`).
 * Parsing them is what turns "call it and read the validation error" into
 * "the schema said so" — without a second hand-written table that would drift.
 */
export declare function extractEnums(text: string): Map<string, string[]>;
/** Apply discovered enums onto matching params (case-insensitive, dash-insensitive). */
export declare function applyEnums(params: ParamSpec[], enums: Map<string, string[]>): ParamSpec[];
export declare function paramToJsonSchema(p: ParamSpec): Record<string, unknown>;
