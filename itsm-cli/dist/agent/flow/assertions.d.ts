import type { Assertion, Expectation } from './types.js';
/**
 * Read a dotted path out of an unknown JSON value.
 * Supports array indices (`items.0.id`) and is deliberately unforgiving:
 * a missing key returns `undefined` so assertions can say "missing" instead of
 * silently passing on a wrongly-typed payload.
 */
export declare function readPath(value: unknown, path: string): unknown;
export declare function assertion(check: string, ok: boolean, expected?: unknown, actual?: unknown): Assertion;
/** Turn declarative expectations into named, evidence-bearing assertions. */
export declare function evaluateExpectations(expectations: Expectation[], response: unknown): Assertion[];
/** Collection helper used by verify() hooks: does any item match `field === value`? */
export declare function listContains(response: unknown, field: string, value: unknown): boolean;
/** Collection helper: how many items match? */
export declare function countMatching(response: unknown, predicate: (item: unknown) => boolean): number;
