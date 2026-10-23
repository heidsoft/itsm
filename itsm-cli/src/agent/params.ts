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

import { FIELD_ENUMS } from './enums.generated.js';

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
  body?: { flag: string; required: boolean; note: string };
}

const VALUE_PLACEHOLDER = /^<([^>]+)>$/;

/**
 * Bare placeholders: `[--status s]`, `[--page N]`, `--title X`.
 *
 * `[--status s]` used to be typed as a boolean flag because only `<angle>`
 * placeholders were recognised, so `itsm --json schema ticket.list` advertised
 * `status` as boolean. An agent that believed the schema sent `--status` with
 * no value; `stringFlag` then returned undefined and the filter was dropped
 * silently — a wrong answer instead of an error. A bare word immediately after
 * a flag is a value placeholder.
 *
 * Values that are themselves enumerations (`--action approve|reject`) contain
 * `|` and therefore do not match, keeping enum extraction in charge of them.
 */
const BARE_PLACEHOLDER = /^[A-Za-z0-9][A-Za-z0-9._@,-]*$/;

/** Split a usage string into top-level tokens, keeping bracketed groups whole. */
function tokenizeUsage(usage: string): Array<{ text: string; depth: number }> {
  const out: Array<{ text: string; depth: number }> = [];
  let depth = 0;
  let buf = '';
  let tokenDepth = 0;
  // `<new status>` is one placeholder, not two tokens: splitting on the space
  // inside the angle brackets lost the type and advertised a boolean flag.
  let inPlaceholder = false;
  const flush = () => {
    if (buf.trim()) out.push({ text: buf.trim(), depth: tokenDepth });
    buf = '';
  };
  for (const ch of usage) {
    if (ch === '<') inPlaceholder = true;
    if (ch === '>') inPlaceholder = false;
    if (!inPlaceholder && ch === '[') {
      if (!buf) tokenDepth = depth + 1;
      depth++;
      continue;
    }
    if (!inPlaceholder && ch === ']') {
      depth = Math.max(0, depth - 1);
      flush();
      tokenDepth = 0;
      continue;
    }
    if (!inPlaceholder && (ch === ' ' || ch === '\n' || ch === '\t')) {
      flush();
      tokenDepth = depth;
      continue;
    }
    if (!buf) tokenDepth = depth;
    buf += ch;
  }
  flush();
  return out;
}

/**
 * Flag typing.
 *
 * The placeholder is the strongest signal (`<n>` is a number, `<json>` is a
 * body, `<s>` is a string). Name heuristics only fill in names whose meaning is
 * unambiguous — they used to fire on any `*level` suffix, which typed
 * `riskLevel` as a number even though its values are low|medium|high.
 */
function inferFlagType(name: string, placeholder: string | undefined): ParamType {
  if (placeholder === undefined) return 'boolean';
  const p = placeholder.toLowerCase();
  if (p === 'json' || p === 'object') return 'json';
  if (/^(n|num|number|count|id|level|ids)$/.test(p)) return 'number';
  if (/(^|\.)(page|pageSize|limit|size)$/.test(name)) return 'number';
  if (/Id$/.test(name)) return 'number';
  return 'string';
}

/**
 * Parse a usage string such as
 *   `itsm ticket list [--status s] [--priority p] [--page N]`
 * into ParamSpecs. Bracket nesting defines required vs optional.
 */
export function deriveParams(usage: string): CommandInputSchema {
  const params: ParamSpec[] = [];
  const tokens = tokenizeUsage(usage.replace(/^itsm\s+/, ''));

  for (let i = 0; i < tokens.length; i++) {
    const { text, depth } = tokens[i];
    const required = depth === 0;

    // Positional placeholders: <id>, [<query>]
    const pos = text.match(VALUE_PLACEHOLDER);
    if (pos) {
      params.push({
        name: pos[1],
        kind: 'positional',
        type: 'string',
        required,
      });
      continue;
    }

    if (!text.startsWith('--')) continue;
    const flagToken = text === '--' ? '' : text.slice(2);
    // `--flag=value` form
    const [flagNameRaw, inlineVal] = flagToken.split('=');
    const flagName = flagNameRaw.replace(/[,|]$/, '');
    if (!flagName) continue;

    let placeholder: string | undefined = inlineVal;
    if (placeholder === undefined) {
      const next = tokens[i + 1];
      if (next && next.depth >= depth && VALUE_PLACEHOLDER.test(next.text)) {
        placeholder = next.text.match(VALUE_PLACEHOLDER)![1];
        i++;
      } else if (next && next.depth >= depth && BARE_PLACEHOLDER.test(next.text)) {
        placeholder = next.text;
        i++;
      }
    }

    if (flagName === 'data' || flagName === 'payload') {
      params.push({
        name: flagName,
        kind: 'body',
        type: 'json',
        required: required && !hasAlternativeBodySource(tokens),
        description: 'JSON request body; field names follow the backend DTO (camelCase).',
      });
      continue;
    }

    const type = inferFlagType(flagName, placeholder);
    params.push({
      name: flagName,
      kind: 'flag',
      type,
      required,
      ...(placeholder && type !== 'boolean' && type !== 'json'
        ? { description: `value: <${placeholder}>` }
        : {}),
    });
  }

  return {
    params,
    ...(params.some((p) => p.kind === 'body')
      ? {
          body: {
            flag: 'data',
            required: false,
            note: 'Field names follow the backend DTO (camelCase). Prefer flags where the command offers them.',
          },
        }
      : {}),
  };
}

/** `<a> ... | --data <json>` means the body is an alternative, not an extra requirement. */
function hasAlternativeBodySource(tokens: Array<{ text: string }>): boolean {
  return tokens.some((t) => t.text === '|');
}

/**
 * Merge derived params with explicit overrides. An explicit spec replaces the
 * derived entry of the same (name, kind) and appends anything new.
 */
export function mergeParams(derived: ParamSpec[], explicit?: ParamSpec[]): ParamSpec[] {
  if (!explicit?.length) return derived;
  const key = (p: ParamSpec) => `${p.kind}:${p.name}`;
  const merged = new Map<string, ParamSpec>();
  for (const p of derived) merged.set(key(p), p);
  for (const p of explicit) merged.set(key(p), { ...merged.get(key(p)), ...p });
  return [...merged.values()];
}

/**
 * Enum extraction.
 *
 * Command summaries already spell out legal values in a consistent shape
 * (`priority required: low|medium|high`, `--action approve|reject|delegate`).
 * Parsing them is what turns "call it and read the validation error" into
 * "the schema said so" — without a second hand-written table that would drift.
 */
export function extractEnums(text: string): Map<string, string[]> {
  const out = new Map<string, string[]>();
  const patterns: Array<[RegExp, (m: RegExpMatchArray) => { name: string; values: string }]> = [
    // `--action approve|reject`, `--action must be one of approve|reject`
    [/--([a-zA-Z][\w-]*)\s+(?:must be one of\s+|one of\s+|=|:)?([a-z][a-z0-9_]*(?:\s*\|\s*[a-z][a-z0-9_]*)+)/g, (m) => ({ name: m[1], values: m[2] })],
    // `priority required: low|medium|high`, `action values: approve|reject`
    [/\b([a-zA-Z][\w]*)\s+(?:is\s+)?(?:required|values?|one of)\s*:\s*([a-z][a-z0-9_]*(?:\s*\|\s*[a-z][a-z0-9_]*)+)/g, (m) => ({ name: m[1], values: m[2] })],
    // `(action approve|reject)`, `(type text|markdown|post|card)`
    [/\(([a-zA-Z][\w]*)\s+([a-z][a-z0-9_]*(?:\s*\|\s*[a-z][a-z0-9_]*)+)\)/g, (m) => ({ name: m[1], values: m[2] })],
    // `status open|pending|resolved` in usage text
    [/\b([a-z][a-z0-9_]*)\s+(open|new|pending|resolved|closed|approved|rejected|draft|published|active|inactive)(?:\s*\|\s*[a-z][a-z0-9_]*)+/g, (m) => ({ name: m[1], values: m.slice(2).join('|') })],
  ];

  for (const [regex, pick] of patterns) {
    for (const match of text.matchAll(regex)) {
      const { name, values } = pick(match);
      const parsed = values
        .split('|')
        .map((v) => v.trim())
        .filter((v) => /^[a-z][a-z0-9_]*$/.test(v));
      if (parsed.length < 2 || parsed.length > 8) continue;
      const key = name.replace(/^-+/, '').toLowerCase();
      if (!out.has(key)) out.set(key, [...new Set(parsed)]);
    }
  }
  return out;
}

/** Apply discovered enums onto matching params (case-insensitive, dash-insensitive). */
export function applyEnums(params: ParamSpec[], enums: Map<string, string[]>): ParamSpec[] {
  return params.map((p) => {
    if (p.type === 'enum' && p.values?.length) return p;
    const key = p.name.replace(/-/g, '').toLowerCase();
    // Command text wins (it is domain-specific); the generated backend table is
    // the fallback for fields whose meaning is consistent across domains.
    const values = enums.get(p.name.toLowerCase()) ?? enums.get(key) ?? FIELD_ENUMS[p.name] ?? FIELD_ENUMS[key];
    return values ? { ...p, type: 'enum', values } : p;
  });
}
export function paramToJsonSchema(p: ParamSpec): Record<string, unknown> {
  const base: Record<string, unknown> = { description: p.description ?? `${p.kind} ${p.name}` };
  switch (p.type) {
    case 'number':
      base['type'] = 'number';
      break;
    case 'boolean':
      base['type'] = 'boolean';
      break;
    case 'json':
      base['type'] = ['object', 'array'];
      break;
    case 'enum':
      base['type'] = 'string';
      if (p.values?.length) base['enum'] = p.values;
      break;
    default:
      base['type'] = 'string';
  }
  if (p.default !== undefined) base['default'] = p.default;
  base['x-itsm-kind'] = p.kind;
  return base;
}
