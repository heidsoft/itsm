'use strict';

/**
 * Regression + observation tests for scripts/test-coverage-guard.js.
 *
 * Two things are locked here (2026-11-14):
 *
 *  1. Frontend glob candidates (`<name>-*.test.ts`) were dead code. The old
 *     match expression was `tests.includes(c) || fileExists(c)`, where
 *     fileExists() calls fs.existsSync() on the LITERAL path containing `*`.
 *     A real sibling test such as `ticket_api-helper.test.ts` therefore never
 *     satisfied the guard and the source was reported as missing a test.
 *
 *  2. `--strict-diff` is an OBSERVATION mode. It must report "no candidate
 *     test was modified in this diff" while keeping the exit code governed
 *     exclusively by the normal (on-disk) rule. Promoting it to a hard gate
 *     is a project decision, not something this test may silently do.
 *
 * Reverse verification is isolated inside the temporary repository: the
 * production guard is copied there and can be patched back to the old
 * implementation before running. The real source tree is never mutated.
 */

const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const test = require('node:test');

const root = path.resolve(__dirname, '..', '..');
const GUARD_SRC = path.join(root, 'scripts', 'test-coverage-guard.js');

const GIT_IDENTITY = [
  '-c', 'user.email=guard-strict-test@example.invalid',
  '-c', 'user.name=Guard Strict Test',
  '-c', 'commit.gpgsign=false',
];

function git(dir, args) {
  const r = spawnSync('git', [...GIT_IDENTITY, ...args], {
    cwd: dir,
    encoding: 'utf8',
    env: { ...process.env, GIT_CONFIG_NOSYSTEM: '1' },
  });
  assert.equal(r.status, 0, `git ${args.join(' ')} failed: ${r.stderr}`);
  return r.stdout;
}

function writeFiles(dir, files) {
  for (const [rel, content] of Object.entries(files)) {
    const abs = path.join(dir, rel);
    fs.mkdirSync(path.dirname(abs), { recursive: true });
    fs.writeFileSync(abs, content);
  }
}

/**
 * Build <base commit> -> <change commit>. `patchGuard` optionally rewrites the
 * copied guard so we can prove the assertion distinguishes old/new behaviour.
 */
function makeRepo({ baseFiles = {}, changes = {}, patchGuard = null } = {}) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'itsm-guard-strict-'));
  fs.mkdirSync(path.join(dir, 'scripts'), { recursive: true });
  let guard = fs.readFileSync(GUARD_SRC, 'utf8');
  if (patchGuard) {
    assert.ok(
      guard.includes(patchGuard.from),
      'reverse-verification anchor disappeared from the guard source'
    );
    guard = guard.replace(patchGuard.from, patchGuard.to);
  }
  fs.writeFileSync(path.join(dir, 'scripts', 'test-coverage-guard.js'), guard);

  git(dir, ['init', '-q']);
  writeFiles(dir, baseFiles);
  git(dir, ['add', '--', 'scripts/test-coverage-guard.js', ...Object.keys(baseFiles)]);
  git(dir, ['commit', '-q', '--allow-empty', '-m', 'base']);

  writeFiles(dir, changes);
  git(dir, ['add', '--', ...Object.keys(changes)]);
  git(dir, ['commit', '-q', '--allow-empty', '-m', 'change']);
  return dir;
}

function runGuard(dir, extraArgs = [], json = true) {
  const args = [
    'scripts/test-coverage-guard.js',
    '--base', 'HEAD~1',
    '--head', 'HEAD',
    ...(json ? ['--json'] : []),
    ...extraArgs,
  ];
  const r = spawnSync(process.execPath, args, { cwd: dir, encoding: 'utf8' });
  let report = null;
  if (json) {
    try {
      report = JSON.parse(r.stdout);
    } catch {
      // Left null; assertions below surface stdout/stderr on failure.
    }
  }
  return { status: r.status, report, stdout: r.stdout, stderr: r.stderr };
}

function cleanup(dir) {
  fs.rmSync(dir, { recursive: true, force: true });
}

const SRC_TS = 'export const fetchTicket = () => 1;\n';
const TEST_TS = "test('placeholder', () => expect(1).toBe(1));\n";
const SRC_GO = 'package services\n\nfunc NewThing() {}\n';
const TEST_GO = 'package services\n\nimport "testing"\n\nfunc TestNewThing(t *testing.T) {}\n';

const FE_SRC = 'itsm-frontend/src/lib/api/ticket_api.ts';
const FE_TEST_DIR = 'itsm-frontend/src/lib/api/__tests__/';
const OLD_MATCH = 'tests.includes(c) || fileExists(c)';
const NEW_MATCH = 'candidateSatisfied(c, tests)';

// ---------- 1. frontend glob regression ----------

test('frontend source with only a <name>-*.test.ts sibling is accepted', () => {
  const dir = makeRepo({
    baseFiles: { [`${FE_TEST_DIR}ticket_api-helper.test.ts`]: TEST_TS },
    changes: { [FE_SRC]: SRC_TS },
  });
  try {
    const { status, report, stdout, stderr } = runGuard(dir);
    assert.equal(status, 0, `${stdout}\n${stderr}`);
    assert.equal(report.missing.length, 0);
  } finally {
    cleanup(dir);
  }
});

test('reverse: restoring the literal-only match makes that case fail', () => {
  const dir = makeRepo({
    baseFiles: { [`${FE_TEST_DIR}ticket_api-helper.test.ts`]: TEST_TS },
    changes: { [FE_SRC]: SRC_TS },
    patchGuard: { from: NEW_MATCH, to: OLD_MATCH },
  });
  try {
    const { status, report, stdout, stderr } = runGuard(dir);
    assert.equal(status, 1, `old implementation unexpectedly passed: ${stdout}\n${stderr}`);
    assert.equal(report.missing[0].src, FE_SRC);
  } finally {
    cleanup(dir);
  }
});

test('frontend source without any sibling test is still rejected', () => {
  const dir = makeRepo({ changes: { [FE_SRC]: SRC_TS } });
  try {
    const { status, report, stdout, stderr } = runGuard(dir);
    assert.equal(status, 1, `${stdout}\n${stderr}`);
    assert.equal(report.missing[0].src, FE_SRC);
  } finally {
    cleanup(dir);
  }
});

// ---------- 2. --strict-diff observation ----------

test('strict-diff reports a source whose on-disk sibling test was not in the diff', () => {
  const dir = makeRepo({
    baseFiles: { 'itsm-backend/internal/services/unrelated_test.go': TEST_GO },
    changes: { 'itsm-backend/internal/services/new_thing.go': SRC_GO },
  });
  try {
    const { status, report, stdout, stderr } = runGuard(dir, ['--strict-diff']);
    assert.equal(status, 0, `observation must not change exit code: ${stdout}\n${stderr}`);
    assert.equal(report.missing.length, 0);
    assert.equal(report.strictDiff.enabled, true);
    assert.deepEqual(report.strictDiff.missingWithoutTestChange, [
      'itsm-backend/internal/services/new_thing.go',
    ]);
  } finally {
    cleanup(dir);
  }
});

test('strict-diff is empty when a candidate test was modified in the diff', () => {
  const dir = makeRepo({
    changes: {
      'itsm-backend/internal/services/new_thing.go': SRC_GO,
      'itsm-backend/internal/services/new_thing_test.go': TEST_GO,
    },
  });
  try {
    const { status, report, stdout, stderr } = runGuard(dir, ['--strict-diff']);
    assert.equal(status, 0, `${stdout}\n${stderr}`);
    assert.equal(report.strictDiff.enabled, true);
    assert.deepEqual(report.strictDiff.missingWithoutTestChange, []);
  } finally {
    cleanup(dir);
  }
});

test('strict-diff expands frontend glob candidates against the diff too', () => {
  const dir = makeRepo({
    changes: {
      [FE_SRC]: SRC_TS,
      [`${FE_TEST_DIR}ticket_api-helper.test.ts`]: TEST_TS,
    },
  });
  try {
    const { status, report, stdout, stderr } = runGuard(dir, ['--strict-diff']);
    assert.equal(status, 0, `${stdout}\n${stderr}`);
    assert.deepEqual(report.strictDiff.missingWithoutTestChange, []);
  } finally {
    cleanup(dir);
  }
});

test('strict-diff does not rescue a genuinely missing test (exit code stays governed by normal rule)', () => {
  const dir = makeRepo({ changes: { [FE_SRC]: SRC_TS } });
  try {
    const { status, report, stdout, stderr } = runGuard(dir, ['--strict-diff']);
    assert.equal(status, 1, `${stdout}\n${stderr}`);
    assert.equal(report.missing.length, 1);
    assert.deepEqual(report.strictDiff.missingWithoutTestChange, [FE_SRC]);
  } finally {
    cleanup(dir);
  }
});

test('strict-diff is disabled unless explicitly requested', () => {
  const dir = makeRepo({ changes: { [FE_SRC]: SRC_TS } });
  try {
    const { report, stdout, stderr } = runGuard(dir);
    assert.ok(report, `${stdout}\n${stderr}`);
    assert.deepEqual(report.strictDiff, { enabled: false });
  } finally {
    cleanup(dir);
  }
});

test('human-readable strict-diff output is explicitly non-blocking', () => {
  const dir = makeRepo({
    baseFiles: { 'itsm-backend/internal/services/unrelated_test.go': TEST_GO },
    changes: { 'itsm-backend/internal/services/new_thing.go': SRC_GO },
  });
  try {
    const { status, stdout, stderr } = runGuard(dir, ['--strict-diff'], false);
    assert.equal(status, 0, `${stdout}\n${stderr}`);
    assert.match(stdout, /NON-BLOCKING/);
    assert.match(stdout, /::warning::strict-diff:/);
  } finally {
    cleanup(dir);
  }
});
