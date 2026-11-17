'use strict';

/**
 * Injection tests for scripts/test-coverage-guard.js.
 *
 * Why a temporary git repo instead of stubbing `execSync`: the guard's whole
 * contract is "for every source file in a diff range, a candidate test must
 * exist". Mocking the diff would only test our mock. Each case below creates a
 * real two-commit repository, copies the production guard into it (the guard
 * resolves REPO_ROOT from its own __dirname), and asserts on the real exit
 * code / JSON report.
 *
 * Regression locked here (2026-11-14): the service layer moved to
 * internal/services/ (337 production files) but the guard still classified
 * only handlers/** and service/**, so core business-logic changes were
 * completely outside the "source changed => test exists" gate.
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
  '-c', 'user.email=guard-test@example.invalid',
  '-c', 'user.name=Guard Test',
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
 * Build <empty base commit> -> <commit containing `changes`>.
 * `baseFiles` land in the base commit so they are on disk but NOT in the diff.
 */
function makeRepo({ baseFiles = {}, changes = {} } = {}) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'itsm-coverage-guard-'));
  fs.mkdirSync(path.join(dir, 'scripts'), { recursive: true });
  fs.copyFileSync(GUARD_SRC, path.join(dir, 'scripts', 'test-coverage-guard.js'));

  git(dir, ['init', '-q']);
  writeFiles(dir, baseFiles);
  git(dir, ['add', '--', 'scripts/test-coverage-guard.js', ...Object.keys(baseFiles)]);
  git(dir, ['commit', '-q', '--allow-empty', '-m', 'base']);

  writeFiles(dir, changes);
  git(dir, ['add', '--', ...Object.keys(changes)]);
  git(dir, ['commit', '-q', '--allow-empty', '-m', 'change']);
  return dir;
}

function runGuard(dir) {
  const r = spawnSync(
    process.execPath,
    ['scripts/test-coverage-guard.js', '--base', 'HEAD~1', '--head', 'HEAD', '--json'],
    { cwd: dir, encoding: 'utf8' }
  );
  let report = null;
  try {
    report = JSON.parse(r.stdout);
  } catch {
    // Kept as null; the assertion below will surface stdout/stderr on failure.
  }
  return { status: r.status, report, stdout: r.stdout, stderr: r.stderr };
}

function cleanup(dir) {
  fs.rmSync(dir, { recursive: true, force: true });
}

const SRC = 'package services\n\nfunc NewThing() {}\n';
const TEST = 'package services\n\nimport "testing"\n\nfunc TestNewThing(t *testing.T) {}\n';

test('internal/services source change without any test is rejected', () => {
  const dir = makeRepo({
    changes: { 'itsm-backend/internal/services/new_thing.go': SRC },
  });
  try {
    const { status, report, stdout, stderr } = runGuard(dir);
    assert.equal(status, 1, `${stdout}\n${stderr}`);
    assert.deepEqual(
      report.missing.map((m) => m.src),
      ['itsm-backend/internal/services/new_thing.go']
    );
  } finally {
    cleanup(dir);
  }
});

test('internal/services sub-package source change is also rejected', () => {
  const dir = makeRepo({
    changes: { 'itsm-backend/internal/services/billing/new_thing.go': SRC },
  });
  try {
    const { status, report, stdout, stderr } = runGuard(dir);
    assert.equal(status, 1, `${stdout}\n${stderr}`);
    assert.equal(report.missing.length, 1);
    assert.match(report.missing[0].src, /^itsm-backend\/internal\/services\//);
  } finally {
    cleanup(dir);
  }
});

test('internal/services change with a matching test in the diff passes', () => {
  const dir = makeRepo({
    changes: {
      'itsm-backend/internal/services/new_thing.go': SRC,
      'itsm-backend/internal/services/new_thing_test.go': TEST,
    },
  });
  try {
    const { status, report, stdout, stderr } = runGuard(dir);
    assert.equal(status, 0, `${stdout}\n${stderr}`);
    assert.equal(report.missing.length, 0);
  } finally {
    cleanup(dir);
  }
});

test('internal/services change is satisfied by an on-disk sibling test (documented weak-gate semantics)', () => {
  // This is intentionally NOT a strict "test must be in the diff" rule.
  // backendCandidates() accepts any *_test.go in the same package. Locking the
  // behaviour here makes the looseness explicit and prevents accidentally
  // advertising this guard as a strong per-symbol coverage check.
  const dir = makeRepo({
    baseFiles: { 'itsm-backend/internal/services/unrelated_test.go': TEST },
    changes: { 'itsm-backend/internal/services/new_thing.go': SRC },
  });
  try {
    const { status, report, stdout, stderr } = runGuard(dir);
    assert.equal(status, 0, `${stdout}\n${stderr}`);
    assert.equal(report.missing.length, 0);
  } finally {
    cleanup(dir);
  }
});

test('explicit skip marker still exempts an internal/services source file', () => {
  const dir = makeRepo({
    changes: {
      'itsm-backend/internal/services/generated_bridge.go':
        '// test-coverage-guard: skip — generated adapter, exercised by E2E\n' + SRC,
    },
  });
  try {
    const { status, report, stdout, stderr } = runGuard(dir);
    assert.equal(status, 0, `${stdout}\n${stderr}`);
    assert.equal(report.missing.length, 0);
  } finally {
    cleanup(dir);
  }
});

test('legacy jurisdiction is unchanged: handler change without test is rejected', () => {
  const dir = makeRepo({
    changes: { 'itsm-backend/handlers/ticket/handler.go': SRC },
  });
  try {
    const { status, report, stdout, stderr } = runGuard(dir);
    assert.equal(status, 1, `${stdout}\n${stderr}`);
    assert.equal(report.missing[0].src, 'itsm-backend/handlers/ticket/handler.go');
  } finally {
    cleanup(dir);
  }
});

test('legacy jurisdiction is unchanged: frontend lib change without test is rejected', () => {
  const dir = makeRepo({
    changes: { 'itsm-frontend/src/lib/api/new_api.ts': 'export const x = 1;\n' },
  });
  try {
    const { status, report, stdout, stderr } = runGuard(dir);
    assert.equal(status, 1, `${stdout}\n${stderr}`);
    assert.equal(report.missing[0].src, 'itsm-frontend/src/lib/api/new_api.ts');
  } finally {
    cleanup(dir);
  }
});

test('unrelated internal/ trees are outside this guard (no false positives)', () => {
  const dir = makeRepo({
    changes: { 'itsm-backend/internal/some_other_pkg/new_thing.go': SRC },
  });
  try {
    const { status, report, stdout, stderr } = runGuard(dir);
    assert.equal(status, 0, `${stdout}\n${stderr}`);
    assert.equal(report.sourceFiles, 0);
    assert.equal(report.missing.length, 0);
  } finally {
    cleanup(dir);
  }
});
