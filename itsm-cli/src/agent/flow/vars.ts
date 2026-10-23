/**
 * Small helpers for reading captured flow variables.
 *
 * Scenarios capture ids from responses as `unknown`; these helpers turn a
 * missing/wrong-typed variable into an immediate, readable failure instead of a
 * downstream `undefined` request URL that reports as a confusing 404.
 */

import { CliError } from '../../lib/errors.js';
import type { FlowContext } from './types.js';

export function requireVar<T>(ctx: FlowContext, key: string, check: (value: unknown) => value is T, expected: string): T {
  const value = ctx.vars[key];
  if (!check(value)) {
    throw new CliError({
      kind: 'internal',
      message: `Flow variable '${key}' is ${value === undefined ? 'missing' : `not ${expected}`}; an earlier capture step did not run or returned an unexpected shape.`,
      retryable: false,
    });
  }
  return value;
}

export function requireNumber(ctx: FlowContext, key: string): number {
  return requireVar(ctx, key, (v): v is number => typeof v === 'number' && Number.isFinite(v), 'a number');
}

export function requireString(ctx: FlowContext, key: string): string {
  return requireVar(ctx, key, (v): v is string => typeof v === 'string' && v.length > 0, 'a non-empty string');
}

/** Marks a scenario as unrunnable in the current environment (verdict: blocked). */
export function blocked(reason: string): never {
  throw new CliError({
    kind: 'unavailable',
    message: reason,
    retryable: false,
    hint: 'This is an environment/configuration gap, not a business-flow failure. Provision the dependency and re-run.',
  });
}

/** Read `id` out of a create response without casting the whole payload to `any`. */
export function idOf(result: unknown, label: string): number {
  const id = (result as { id?: unknown } | null)?.id;
  if (typeof id !== 'number') {
    throw new CliError({
      kind: 'internal',
      message: `${label} response has no numeric 'id' (got ${JSON.stringify(id)}); the create contract changed.`,
      retryable: false,
    });
  }
  return id;
}

/** Version field used by optimistic-lock DTOs. */
export function versionOf(result: unknown, label: string): number {
  const version = (result as { version?: unknown } | null)?.version;
  if (typeof version !== 'number') {
    throw new CliError({
      kind: 'internal',
      message: `${label} response has no numeric 'version' (got ${JSON.stringify(version)}); cannot build an optimistic-lock update.`,
      retryable: false,
    });
  }
  return version;
}
