import { usageError } from '../lib/errors.js';

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

const GLOBAL_BOOLEAN_FLAGS = new Set(['json', 'yes', 'dry-run', 'no-retry']);

export function parseArgs(argv: string[]): ParsedArgs {
  const positional: string[] = [];
  const flags: Record<string, string | boolean | number> = {};
  let json = false;
  let yes = false;
  let dryRun = false;
  let baseUrl: string | undefined;
  let timeoutMs: number | undefined;
  let retries: number | undefined;
  let fields: string[] | undefined;
  let limit: number | undefined;
  let format: 'json' | 'ndjson' = 'json';

  for (let i = 0; i < argv.length; i++) {
    const token = argv[i];
    if (!token.startsWith('-') || token === '-') {
      positional.push(token);
      continue;
    }
    if (token === '--') {
      positional.push(...argv.slice(i + 1));
      break;
    }
    // Expand single-dash short flags (-p, -u) into their long form only when they map.
    const eq = token.indexOf('=');
    let name: string;
    let inlineValue: string | undefined;
    if (eq >= 0) {
      name = token.slice(token.startsWith('--') ? 2 : 1, eq);
      inlineValue = token.slice(eq + 1);
    } else {
      name = token.replace(/^--?/, '');
      inlineValue = undefined;
    }

    if (GLOBAL_BOOLEAN_FLAGS.has(name)) {
      if (name === 'json') json = true;
      else if (name === 'yes') yes = true;
      else if (name === 'dry-run') dryRun = true;
      else if (name === 'no-retry') retries = 0;
      continue;
    }
    if (name === 'base-url' || name === 'baseUrl') {
      baseUrl = inlineValue ?? argv[++i];
      continue;
    }
    if (name === 'timeout') {
      const raw = inlineValue ?? argv[++i];
      const n = Number(raw);
      if (!Number.isFinite(n) || n < 0) throw usageError('--timeout must be a non-negative number of milliseconds');
      timeoutMs = n;
      continue;
    }
    if (name === 'retries') {
      const raw = inlineValue ?? argv[++i];
      const n = Number(raw);
      if (!Number.isInteger(n) || n < 0) throw usageError('--retries must be a non-negative integer');
      retries = n;
      continue;
    }
    if (name === 'fields') {
      const raw = inlineValue ?? argv[++i];
      if (!raw) throw usageError('--fields requires a comma-separated field list, e.g. --fields id,title,status');
      fields = String(raw)
        .split(',')
        .map((f) => f.trim())
        .filter(Boolean);
      if (!fields.length) throw usageError('--fields requires at least one field name');
      continue;
    }
    if (name === 'limit') {
      const raw = inlineValue ?? argv[++i];
      const n = Number(raw);
      if (!Number.isInteger(n) || n < 0) throw usageError('--limit must be a non-negative integer');
      limit = n;
      continue;
    }
    if (name === 'format') {
      const raw = inlineValue ?? argv[++i];
      if (raw !== 'json' && raw !== 'ndjson') throw usageError("--format must be 'json' or 'ndjson'");
      format = raw;
      continue;
    }

    let value: string | boolean | number;
    if (inlineValue !== undefined) {
      value = inlineValue;
    } else if (i + 1 < argv.length && !argv[i + 1].startsWith('-')) {
      value = argv[++i];
    } else {
      value = true;
    }
    flags[name] = coerce(value);
  }

  return { positional, flags, json, yes, dryRun, baseUrl, timeoutMs, retries, fields, limit, format };
}

function coerce(value: string | boolean): string | boolean | number {
  if (typeof value === 'boolean') return value;
  if (value !== '' && !Number.isNaN(Number(value))) return Number(value);
  return value;
}

/** String flag accessors used by command implementations. */
export function stringFlag(flags: ParsedArgs['flags'], ...names: string[]): string | undefined {
  for (const n of names) {
    const v = flags[n];
    if (typeof v === 'string') return v;
    if (typeof v === 'number') return String(v);
  }
  return undefined;
}

export function booleanFlag(flags: ParsedArgs['flags'], ...names: string[]): boolean | undefined {
  for (const n of names) {
    const v = flags[n];
    if (typeof v === 'boolean') return v;
    if (v === 'true') return true;
    if (v === 'false') return false;
  }
  return undefined;
}

export function numberFlag(flags: ParsedArgs['flags'], ...names: string[]): number | undefined {
  for (const n of names) {
    const v = flags[n];
    if (typeof v === 'number') return v;
    if (typeof v === 'string' && v !== '' && !Number.isNaN(Number(v))) return Number(v);
  }
  return undefined;
}

/**
 * Comma-separated integer list (`--ids 1,2,3`).
 *
 * numberFlag cannot serve this: Number("1,2,3") is NaN, so a bulk flag would be
 * silently reported as missing and the command would exit 2 on a valid call.
 * Returns undefined when absent or when any segment is not a finite integer —
 * a partially-valid list is worse than none, since it would silently update the
 * wrong subset of records.
 */
export function numberListFlag(flags: ParsedArgs['flags'], ...names: string[]): number[] | undefined {
  for (const n of names) {
    const v = flags[n];
    const raw = Array.isArray(v) ? v.map(String) : typeof v === 'number' || typeof v === 'string' ? [String(v)] : undefined;
    if (!raw || raw.length === 0) continue;
    const parts = raw.flatMap((s) => s.split(',')).map((s) => s.trim()).filter((s) => s !== '');
    if (parts.length === 0) continue;
    const nums = parts.map(Number);
    if (nums.some((x) => !Number.isFinite(x))) return undefined;
    return nums;
  }
  return undefined;
}

export function requirePositional(args: ArgContext, index: number, label: string): string {
  const v = args.positional[index];
  if (v === undefined || v === '') {
    throw usageError(`Missing required argument <${label}>. See usage with --help.`);
  }
  return v;
}

/** Parse a JSON argument supplied inline (e.g. --data '{"summary":"..."}'). */
export function jsonFlag(flags: ParsedArgs['flags'], names: string[] = ['data', 'payload']): Record<string, unknown> | undefined {
  const raw = stringFlag(flags, ...names);
  if (raw === undefined) return undefined;
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
      throw new Error('must be a JSON object');
    }
    return parsed as Record<string, unknown>;
  } catch (e) {
    const detail = e instanceof Error ? e.message : String(e);
    throw usageError(`Invalid JSON in --${names[0]}: ${detail}`);
  }
}
