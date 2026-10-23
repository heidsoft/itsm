/**
 * Small helpers for reading captured flow variables.
 *
 * Scenarios capture ids from responses as `unknown`; these helpers turn a
 * missing/wrong-typed variable into an immediate, readable failure instead of a
 * downstream `undefined` request URL that reports as a confusing 404.
 */
import type { FlowContext } from './types.js';
export declare function requireVar<T>(ctx: FlowContext, key: string, check: (value: unknown) => value is T, expected: string): T;
export declare function requireNumber(ctx: FlowContext, key: string): number;
export declare function requireString(ctx: FlowContext, key: string): string;
/** Marks a scenario as unrunnable in the current environment (verdict: blocked). */
export declare function blocked(reason: string): never;
/** Read `id` out of a create response without casting the whole payload to `any`. */
export declare function idOf(result: unknown, label: string): number;
/** Version field used by optimistic-lock DTOs. */
export declare function versionOf(result: unknown, label: string): number;
