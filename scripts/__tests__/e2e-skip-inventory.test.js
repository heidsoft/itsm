'use strict';

/**
 * Keeps docs/testing/e2e-skip-inventory.md honest.
 *
 * A skip inventory that is not mechanically tied to the source is just prose
 * that rots: the moment someone adds a `test.skip()` (or removes a spec) the
 * document silently becomes wrong. This test derives the real per-file skip
 * counts from the spec files and requires the table, the category subtotals
 * and the grand total to match.
 *
 * Reverse checks mutate a COPY of the parsed table in memory, so the real
 * repository is never touched.
 */

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const root = path.resolve(__dirname, '..', '..');
const E2E_DIR = path.join(root, 'itsm-frontend', 'tests', 'e2e');
const INVENTORY = path.join(root, 'docs', 'testing', 'e2e-skip-inventory.md');

// Matches test.skip(, test.fixme(, describe.skip(, describe.fixme(.
// \s* before `(` also covers the multi-line `test.skip(\n ...` form.
const SKIP_RE = /\b(?:test|describe)\.(?:skip|fixme)\s*\(/g;
const ROW_RE = /^\|\s*`([^`]+\.spec\.ts)`\s*\|\s*(\d+)\s*\|/gm;

function specFiles(dir = E2E_DIR, out = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const abs = path.join(dir, entry.name);
    if (entry.isDirectory()) specFiles(abs, out);
    else if (entry.name.endsWith('.spec.ts')) out.push(abs);
  }
  return out;
}

function actualSkipCounts() {
  const counts = new Map();
  for (const abs of specFiles()) {
    const text = fs.readFileSync(abs, 'utf8');
    const n = (text.match(SKIP_RE) || []).length;
    if (n > 0) {
      counts.set(path.relative(E2E_DIR, abs).split(path.sep).join('/'), n);
    }
  }
  return counts;
}

function documentedSkipCounts(markdown) {
  const counts = new Map();
  for (const m of markdown.matchAll(ROW_RE)) {
    counts.set(m[1], Number(m[2]));
  }
  return counts;
}

function categorySubtotals(markdown) {
  const m = markdown.match(
    /小计：`P`\s*=\s*(\d+)、`G`\s*=\s*(\d+)、`D`\s*=\s*(\d+)，合计\s*\*\*(\d+)\*\*/
  );
  assert.ok(m, 'inventory is missing the P/G/D subtotal line');
  return { p: Number(m[1]), g: Number(m[2]), d: Number(m[3]), total: Number(m[4]) };
}

function assertAligned(actual, documented, subtotals) {
  const actualFiles = [...actual.keys()].sort();
  const documentedFiles = [...documented.keys()].sort();
  assert.deepEqual(
    documentedFiles,
    actualFiles,
    'inventory file list differs from the spec files containing skips'
  );
  for (const [file, n] of actual) {
    assert.equal(
      documented.get(file),
      n,
      `inventory count for ${file} is wrong (source has ${n} skips)`
    );
  }
  const sum = [...actual.values()].reduce((a, b) => a + b, 0);
  assert.equal(subtotals.p + subtotals.g + subtotals.d, sum, 'P/G/D subtotals do not add up');
  assert.equal(subtotals.total, sum, 'grand total does not match the source');
}

const markdown = fs.readFileSync(INVENTORY, 'utf8');
const actual = actualSkipCounts();
const documented = documentedSkipCounts(markdown);
const subtotals = categorySubtotals(markdown);

test('skip inventory matches the real spec files, counts and subtotals', () => {
  assert.ok(actual.size > 0, 'no E2E skips found — scanner is probably broken');
  assertAligned(actual, documented, subtotals);
});

test('reverse: dropping a row from the inventory makes the check fail', () => {
  const broken = new Map(documented);
  broken.delete([...broken.keys()][0]);
  assert.throws(() => assertAligned(actual, broken, subtotals), /file list differs/);
});

test('reverse: a stale count in the inventory makes the check fail', () => {
  const first = [...documented.keys()][0];
  const broken = new Map(documented);
  broken.set(first, documented.get(first) + 1);
  assert.throws(() => assertAligned(actual, broken, subtotals), /is wrong/);
});

test('reverse: a stale grand total makes the check fail', () => {
  assert.throws(
    () => assertAligned(actual, documented, { ...subtotals, total: subtotals.total + 1 }),
    /grand total/
  );
});

test('reverse: inconsistent category subtotals make the check fail', () => {
  assert.throws(
    () => assertAligned(actual, documented, { ...subtotals, d: subtotals.d + 1 }),
    /subtotals/
  );
});
