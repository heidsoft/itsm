'use strict';
/**
 * Tests for scripts/test-trend.js — PR-0.5.
 *
 * Run from repo root: `node --test scripts/__tests__/test-trend.test.js`
 */

const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const test = require('node:test');

const root = path.resolve(__dirname, '..', '..');

function tmpFile(name, body) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'itsm-trend-'));
  const p = path.join(dir, name);
  fs.writeFileSync(p, body);
  return { dir, p };
}

function run(args) {
  return spawnSync('node', ['scripts/test-trend.js', ...args], {
    cwd: root,
    encoding: 'utf8',
  });
}

test('emits a Markdown header even with no inputs', () => {
  const result = run(['--baseline', 'docs/testing/coverage-baseline.json']);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /^## 📈 Test Trend/m);
  assert.match(result.stdout, /Backend \(Go\)/m);
  assert.match(result.stdout, /Frontend \(Jest\)/m);
});

test('aggregates two junit files', () => {
  const xmlA = `<?xml version="1.0"?>
<testsuites>
  <testsuite name="a" tests="5" failures="1" errors="0" skipped="1" time="0.5"/>
</testsuites>`;
  const xmlB = `<?xml version="1.0"?>
<testsuites>
  <testsuite name="b" tests="3" failures="0" errors="1" skipped="0" time="0.7"/>
</testsuites>`;
  const a = tmpFile('a.xml', xmlA);
  const b = tmpFile('b.xml', xmlB);

  const result = run([
    '--junit', a.p,
    '--junit', b.p,
    '--baseline', 'docs/testing/coverage-baseline.json',
  ]);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /\| tests \| 8 \|/);
  assert.match(result.stdout, /\| failures \| 1 \|/);
  assert.match(result.stdout, /\| errors \| 1 \|/);
  assert.match(result.stdout, /\| skipped \| 1 \|/);
  assert.match(result.stdout, /\| duration \(s\) \| 1\.20 \|/);

  // cleanup
  fs.rmSync(a.dir, { recursive: true, force: true });
  fs.rmSync(b.dir, { recursive: true, force: true });
});

test('parses a Go coverage.out fragment with mixed counts', () => {
  // Two foo blocks and one bar block:
  //   foo:1 = 3 stmts all hit (count=1 → "covered")
  //   foo:2 = 5 stmts none hit (count=0 → "uncovered")
  //   bar   = 4 stmts all hit
  // Total: 12 stmts, 7 hit → 7/12 ≈ 58.33%
  const cov = [
    'mode: set',
    'itsm-backend/service/foo.go:10.5,12.1 3 1',
    'itsm-backend/service/foo.go:20.3,22.1 5 0',
    'itsm-backend/service/bar.go:1.1,3.1 4 4',
  ].join('\n');
  const f = tmpFile('cov.out', cov);

  const result = run([
    '--go-coverage', f.p,
    '--baseline', 'docs/testing/coverage-baseline.json',
  ]);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /line coverage.*58\.33%/);
  // Both packages should make it into the "bottom 5 packages" roll-up.
  assert.match(result.stdout, /service/);

  fs.rmSync(f.dir, { recursive: true, force: true });
});

test('missing baseline does not crash', () => {
  const xml = `<?xml version="1.0"?>
<testsuites>
  <testsuite name="x" tests="4" failures="0" errors="0" skipped="0" time="0.1"/>
</testsuites>`;
  const f = tmpFile('x.xml', xml);

  const result = run([
    '--baseline', '/nonexistent/baseline.json',
    '--junit', f.p,
  ]);
  assert.equal(result.status, 0, result.stderr);
  // Banner wording should reflect "no baseline" rather than a captured ts.
  assert.match(result.stdout, /baseline unknown captured/);
  // And the aggregate junit row should still render.
  assert.match(result.stdout, /\| tests \| 4 \|/);

  fs.rmSync(f.dir, { recursive: true, force: true });
});

test('Go coverage renders a delta cell when baseline is present', () => {
  const cov = [
    'mode: set',
    'itsm-backend/service/foo.go:1.1,3.1 10 10',  // 100%
  ].join('\n');
  const covFile = tmpFile('cov.out', cov);

  const baselineDir = fs.mkdtempSync(
    path.join(os.tmpdir(), 'itsm-trend-baseline-')
  );
  const baselineFile = path.join(baselineDir, 'baseline.json');
  fs.writeFileSync(
    baselineFile,
    JSON.stringify({ go: 50, jest: 0, ts: '2026-06-28T00:00:00Z' })
  );

  const result = run([
    '--go-coverage', covFile.p,
    '--baseline', baselineFile,
  ]);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /100\.00%/);
  assert.match(result.stdout, /🟢 ↑50/);

  fs.rmSync(covFile.dir, { recursive: true, force: true });
  fs.rmSync(baselineDir, { recursive: true, force: true });
});
