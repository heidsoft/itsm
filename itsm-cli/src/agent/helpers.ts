import {
  stringFlag,
  numberFlag,
  numberListFlag,
  booleanFlag,
  jsonFlag,
  requirePositional,
} from './args.js';
import type { ArgContext } from './args.js';
import { notAuthenticatedError, usageError } from '../lib/errors.js';
import { loadCredentials } from '../lib/credentials.js';

/** Shared helpers used by every domain command module. */

export { stringFlag, numberFlag, numberListFlag, booleanFlag, jsonFlag, jsonFlag as bodyFlag, requirePositional };

export interface PaginatedQuery extends Record<string, string | number | boolean | undefined> {}

/** Accept either {positional, flags} or a bare flags record. */
type ArgsLike = ArgContext | Record<string, string | boolean | number>;

function asArgContext(args: ArgsLike): ArgContext {
  return 'flags' in args ? (args as ArgContext) : { positional: [], flags: args as ArgContext['flags'] };
}

export function withPagination(argsLike: ArgsLike): PaginatedQuery {
  const args = asArgContext(argsLike);
  const out: PaginatedQuery = {};
  const page = numberFlag(args.flags, 'page');
  const pageSize = numberFlag(args.flags, 'pageSize');
  if (page !== undefined) out['page'] = page;
  if (pageSize !== undefined) out['pageSize'] = pageSize;
  return out;
}

export function optionalString(argsLike: ArgsLike, ...names: string[]): string | undefined {
  return stringFlag(asArgContext(argsLike).flags, ...names);
}

export function requiredString(argsLike: ArgsLike, name: string, usage: string): string {
  const value = stringFlag(asArgContext(argsLike).flags, name);
  if (!value) {
    throw usageError(`Missing required flag --${name}. Usage: ${usage}`);
  }
  return value;
}

export function optionalBool(argsLike: ArgsLike, ...names: string[]): boolean | undefined {
  return booleanFlag(asArgContext(argsLike).flags, ...names);
}

export function optionalNumber(argsLike: ArgsLike, ...names: string[]): number | undefined {
  return numberFlag(asArgContext(argsLike).flags, ...names);
}

/** Inline JSON body (--data '{"key":"value"}'); undefined when absent. */
export function body(argsLike: ArgsLike, names: string[] = ['data', 'payload']): Record<string, unknown> | undefined {
  return jsonFlag(asArgContext(argsLike).flags, names);
}

/** Standard pagination query from a list envelope response — pass through untouched. */
export function mergeQuery(...parts: Array<Record<string, string | number | boolean | undefined>>): PaginatedQuery {
  const out: PaginatedQuery = {};
  for (const part of parts) {
    for (const [k, v] of Object.entries(part)) {
      if (v !== undefined) out[k] = v;
    }
  }
  return out;
}

export function assertLoggedIn(): void {
  const token = (process.env['ITSM_TOKEN'] || '').trim() || loadCredentials()?.token;
  if (!token) throw notAuthenticatedError();
}
