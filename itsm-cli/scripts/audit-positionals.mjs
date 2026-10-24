#!/usr/bin/env node
/**
 * Positional audit: for every command, compare the positional indexes the
 * implementation READS against the positionals the usage string DECLARES.
 *
 * The mirror image of audit-flags.mjs. A command that reads `<ctx.args.positional>[2]`
 * while usage only advertises one positional is unreachable from the schema; a
 * command whose usage advertises more than it reads silently swallows an
 * argument — and the runtime strict-positional check turns that into a hard
 * error, so the two sides must agree before it ships.
 *
 * Static, heuristic, transitive (same resolution strategy as audit-flags.mjs).
 */
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const dir = join(root, 'src/agent/commands');

const { registry } = await import(join(root, 'dist/agent/registry.js'));
const specs = new Map(registry().map((s) => [s.name, s]));

let bad = 0;

for (const file of readdirSync(dir).filter((f) => f.endsWith('.ts'))) {
  const src = readFileSync(join(dir, file), 'utf8');

  // Module-level helpers, keyed by name: a helper's reads only belong to the
  // commands that actually CALL it, otherwise one shared `id(ctx)` helper
  // appears to be read by every command in the file.
  const helpers = new Map(
    [...src.matchAll(/function\s+(\w+)\s*\([^)]*\)\s*\{[\s\S]*?\n\}/g)].map((m) => [m[1], m[0]])
  );

  const marks = [...src.matchAll(/cmd\(\s*'([^']+)'/g)];
  marks.forEach((m, i) => {
    const spec = specs.get(m[1]);
    if (!spec) return;
    const closure = src.slice(m.index, i + 1 < marks.length ? marks[i + 1].index : src.length);
    const runIdx = closure.indexOf('async (');
    const scoped = closure.slice(runIdx >= 0 ? runIdx : 0);
    const arrayEnd = scoped.search(/^\s*\];/m);
    const ownBody = arrayEnd >= 0 ? scoped.slice(0, arrayEnd) : scoped;
    const called = [...ownBody.matchAll(/(?:^|[^.\w])(\w+)\s*\(/g)].map((c) => c[1]);
    const runBody = ownBody + [...new Set(called)].map((n) => helpers.get(n) ?? '').join('\n');

    const declared = spec.input.params.filter((p) => p.kind === 'positional').length;
    // requirePositional(ctx.args, N, 'name') — the index is what matters.
    const reads = [...runBody.matchAll(/requirePositional\(\s*[^,]+,\s*(\d+)\s*,\s*'([^']*)'/g)].map((r) => ({ index: Number(r[1]), name: r[2] }));
    const maxRead = reads.length ? Math.max(...reads.map((r) => r.index)) : -1;
    if (maxRead >= declared) {
      bad++;
      console.log(`✗ ${m[1]} reads positional[${maxRead}] (${reads.find((r) => r.index === maxRead)?.name}) but usage declares ${declared}`);
      console.log(`    usage: ${spec.usage}`);
    }
  });
}

if (bad === 0) console.log('audit clean: every positional read by an implementation is declared in its usage');
else process.exit(1);
