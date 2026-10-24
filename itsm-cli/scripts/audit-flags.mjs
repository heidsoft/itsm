#!/usr/bin/env node
/**
 * Flag audit: for every command spec, compare the flags the implementation
 * READS against the flags the usage string DECLARES.
 *
 * Why this matters: `deriveParams(usage)` is the only source of the MCP/agent
 * input schema. A flag the implementation reads but usage does not declare is
 * invisible to schema-driven clients, and the CLI parser silently ignores
 * unknown flags — so the gap never surfaces as an error. This audit closes the
 * loop the other direction: it finds "read but not declared" before strict
 * runtime validation turns the gap into user-visible failures.
 *
 * Static, heuristic, transitive: it resolves flags read inside module-level
 * helpers called from a command's run closure. It errs toward over-reporting
 * (superset), which is safe here — every report is either a real gap or a
 * command that should not read that flag at all.
 *
 * Usage: node scripts/audit-flags.mjs [--fix-hints]
 */
import { readFileSync, readdirSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const CMD_DIR = join(ROOT, 'src', 'agent', 'commands');

/**
 * Flags an implementation deliberately reads but the usage string must NOT
 * advertise. Each entry needs a justification, or it is a schema gap.
 *   - skipProbes: legacy camelCase alias kept for clients built on the old
 *     schema; the canonical kebab-case `--skip-probes` is advertised.
 *   - payload: alternative body flag read by `body()`/`jsonFlag()` wherever
 *     `--data` is declared; the runtime strict check accepts it as an alias.
 */
const DECLARED_ALIASES = new Set(['skipProbes', 'payload']);

// ---------- parse each command file ----------

const FLAG_READ_PATTERNS = [
  /\b(?:optionalString|optionalNumber|stringFlag|numberFlag|booleanFlag|numberListFlag)\(\s*(?:ctx\.args\.flags|args\.flags|flags)\s*,\s*'([^']+)'/g,
  /\bcsvNumbers\(\s*ctx\s*,\s*'([^']+)'/g,
  /\bcsvStrings\(\s*ctx\s*,\s*'([^']+)'/g,
  /\bargs\.flags\['([^']+)'\]/g,
  /\bflags\['([^']+)'\]/g,
];

const HELPER_CALL_PATTERNS = [
  /\bwithPagination\(/g, // contributes page, pageSize
  /\bjsonFlag\(/g, // contributes data, payload
];

/** Extract `function name(...) {...}` bodies at module top level (brace-matched). */
function extractFunctions(src) {
  const out = new Map();
  const re = /(?:export\s+)?function\s+([A-Za-z_$][\w$]*)\s*\(/g;
  let m;
  while ((m = re.exec(src))) {
    const name = m[1];
    let i = src.indexOf('{', m.index + m[0].length - 1);
    if (i < 0) continue;
    let depth = 0;
    const start = i;
    for (; i < src.length; i++) {
      if (src[i] === '{') depth++;
      else if (src[i] === '}') {
        depth--;
        if (depth === 0) break;
      }
    }
    out.set(name, src.slice(start, i + 1));
  }
  return out;
}

function flagsReadIn(body, fileHelpers, seen = new Set()) {
  const flags = new Set();
  const helpers = new Set();
  for (const re of FLAG_READ_PATTERNS) {
    for (const m of body.matchAll(re)) flags.add(m[1]);
  }
  const PAGINATION = HELPER_CALL_PATTERNS[0];
  const BODY = HELPER_CALL_PATTERNS[1];
  for (const re of HELPER_CALL_PATTERNS) {
    if (re === PAGINATION) {
      // A global regex carries lastIndex across matchAll calls; reset per body
      // or later commands silently miss their matches.
      re.lastIndex = 0;
      for (const _m of body.matchAll(re)) {
        flags.add('page');
        flags.add('pageSize');
      }
    } else if (re === BODY) {
      re.lastIndex = 0;
      for (const _m of body.matchAll(re)) {
        flags.add('data');
        flags.add('payload');
      }
    }
  }
  // Transitively descend into module-level helpers called from this body.
  for (const name of fileHelpers.keys()) {
    if (seen.has(name)) continue;
    if (new RegExp(`\\b${name}\\s*\\(`).test(body)) {
      seen.add(name);
      for (const f of flagsReadIn(fileHelpers.get(name), fileHelpers, seen)) flags.add(f);
    }
  }
  return flags;
}

/** Flags declared in a usage string: every `--name` token. */
function flagsDeclared(usage) {
  const out = new Set();
  for (const m of usage.matchAll(/--([A-Za-z][\w-]*)/g)) out.add(m[1]);
  return out;
}

// ---------- per-file audit ----------

const problems = [];
for (const file of readdirSync(CMD_DIR).filter((f) => f.endsWith('.ts')).sort()) {
  const src = readFileSync(join(CMD_DIR, file), 'utf8');
  const helpers = extractFunctions(src);
  // Boundary: literal `cmd('name',` AND spread templates `cmd(`change.${action}`,`
  // (the latter would otherwise be attributed to the preceding command).
  const cmdRe = /cmd\(\s*['`]([\w.${}-]*)['`]/g;
  const starts = [];
  let m;
  while ((m = cmdRe.exec(src))) starts.push({ name: m[1], index: m.index });
  for (let k = 0; k < starts.length; k++) {
    const { name } = starts[k];
    const end = k + 1 < starts.length ? starts[k + 1].index : src.length;
    const closure = src.slice(starts[k].index, end);

    const usageMatch = closure.match(/(['`])itsm [^'`]*\1/);
    const usage = usageMatch ? usageMatch[0].slice(1, -1) : '';
    const declared = flagsDeclared(usage);

    const runIdx = closure.indexOf('async (ctx)');
    // The last cmd of a file would otherwise swallow every helper defined
    // after the command array; stop at the array terminator.
    const scoped = closure.slice(runIdx >= 0 ? runIdx : 0);
    const arrayEnd = scoped.search(/^\s*\];/m);
    const runBody = arrayEnd >= 0 ? scoped.slice(0, arrayEnd) : scoped;
    const read = flagsReadIn(runBody, helpers);

    // `denyFlags` are read only to reject them with a security explanation
    // (secrets must not travel on argv). They are intentionally absent from the
    // usage string, so subtract them instead of reporting a schema gap.
    for (const block of closure.matchAll(/denyFlags:\s*\{([^}]*)\}/g)) {
      for (const key of block[1].matchAll(/([A-Za-z][\w-]*)\s*:/g)) read.delete(key[1]);
    }

    const undeclared = [...read]
      .filter((f) => !declared.has(f) && !DECLARED_ALIASES.has(f))
      .sort();
    if (undeclared.length) {
      problems.push({ file, command: name, undeclared, usage });
    }
  }
}

if (!problems.length) {
  console.log('audit clean: every flag read by an implementation is declared in its usage string');
} else {
  console.log(`READ-BUT-NOT-DECLARED (${problems.length} commands):\n`);
  for (const p of problems) {
    console.log(`  ${p.command}  (${p.file})`);
    console.log(`    reads: --${p.undeclared.join(', --')}`);
    console.log(`    usage: ${p.usage || '(none)'}`);
  }
  process.exitCode = 1;
}
