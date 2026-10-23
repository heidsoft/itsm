import { stringFlag, numberFlag, numberListFlag, booleanFlag, jsonFlag, requirePositional } from './args.js';
import type { ArgContext } from './args.js';
/** Shared helpers used by every domain command module. */
export { stringFlag, numberFlag, numberListFlag, booleanFlag, jsonFlag, jsonFlag as bodyFlag, requirePositional };
export interface PaginatedQuery extends Record<string, string | number | boolean | undefined> {
}
/** Accept either {positional, flags} or a bare flags record. */
type ArgsLike = ArgContext | Record<string, string | boolean | number>;
export declare function withPagination(argsLike: ArgsLike): PaginatedQuery;
export declare function optionalString(argsLike: ArgsLike, ...names: string[]): string | undefined;
export declare function requiredString(argsLike: ArgsLike, name: string, usage: string): string;
export declare function optionalBool(argsLike: ArgsLike, ...names: string[]): boolean | undefined;
export declare function optionalNumber(argsLike: ArgsLike, ...names: string[]): number | undefined;
/** Inline JSON body (--data '{"key":"value"}'); undefined when absent. */
export declare function body(argsLike: ArgsLike, names?: string[]): Record<string, unknown> | undefined;
/** Standard pagination query from a list envelope response — pass through untouched. */
export declare function mergeQuery(...parts: Array<Record<string, string | number | boolean | undefined>>): PaginatedQuery;
export declare function assertLoggedIn(): void;
