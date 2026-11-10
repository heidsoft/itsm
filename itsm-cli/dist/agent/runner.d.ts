import type { ParsedArgs } from './args.js';
import type { CommandSpec } from './spec.js';
export declare function runAgent(argv: string[]): Promise<number>;
/**
 * Strict flag validation at dispatch.
 *
 * `parseArgs` is deliberately permissive (it cannot know the command yet), so
 * without this check an unknown flag lands in `flags`, is never read by the
 * command, and the call silently succeeds without the requested effect — a
 * wrong answer, not an error. The declared contract is `spec.input` (derived
 * from the usage string plus explicit params), which is the SAME source the
 * MCP/agent schemas advertise, so a flag accepted here is always visible in
 * `itsm schema` output and vice versa.
 *
 * Single-character names (`-p`) are allowed through: they are legacy aliases
 * read via `stringFlag(flags, 'password', 'p')` and never advertised.
 */
export declare function validateFlags(spec: CommandSpec, args: ParsedArgs): void;
/**
 * Strict positional validation at dispatch — the mirror of `validateFlags`.
 *
 * Only *extra* positionals are rejected (a declared-but-unused one is harmless).
 * Without this, `itsm connector configs garbage` resolves to `connector.configs`,
 * never reads the trailing token, and returns ok:true — the agent believes a
 * per-connector config was fetched when the argument was silently dropped.
 * Under-declared positionals are caught statically by `audit-positionals.mjs`.
 */
export declare function validatePositionals(spec: CommandSpec, rest: string[]): void;
