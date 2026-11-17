'use strict';

/**
 * CI wiring contracts for the test stack.
 *
 * These are static contract locks, not re-implementations of CI. They exist
 * because the failure modes they guard are all silent-green failures:
 *   - `go test ... | tee` under GitHub Actions' default `bash -e` (no
 *     pipefail) hides a failing AI-evaluator gate behind tee's exit code.
 *   - A guard without a matching `paths:` entry never runs on the PR that
 *     changed the thing it guards.
 *   - A script test without a runner is not a test.
 *
 * Assertions lock semantics (exit-code capture, whole-directory runner,
 * trigger paths) rather than incidental line numbers or wording.
 */

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const root = path.resolve(__dirname, '..', '..');
const read = (rel) => fs.readFileSync(path.join(root, rel), 'utf8');

const backendCi = read('.github/workflows/backend-ci.yml');
const docsGate = read('.github/workflows/docs-gate.yml');
const coverageGuard = read('.github/workflows/test-coverage-guard.yml');
const makefile = read('Makefile');

function section(yaml, start, end) {
  const re = new RegExp(`^  ${start}:[\\s\\S]*?(?=^  ${end}:)`, 'm');
  const m = yaml.match(re);
  assert.ok(m, `could not locate "${start}:" section in workflow YAML`);
  return m[0];
}

function assertExitCodeCapture(yaml) {
  // Strip YAML/run-block comment lines first: the explanatory comment itself
  // mentions PIPESTATUS and pipefail, and matching against the raw file would
  // let a comment keep the assertion green after the real code was removed.
  const code = yaml.split('\n').filter((line) => !/^\s*#/.test(line)).join('\n');
  // The pipeline must still tee its log (observability) AND explicitly read
  // PIPESTATUS[0] (or enable pipefail) so a failing go test cannot be masked.
  assert.match(code, /go test \.\/ai\/eval\/\.\.\.[^\n]*\| tee ai-eval\.log/);
  assert.match(
    code,
    /(?:PIPESTATUS\[0\]|set -o pipefail)/,
    'AI-eval pipeline needs pipefail or an explicit PIPESTATUS check in executable code'
  );
  assert.match(code, /AI_EVAL_STATUS[^\n]*-ne 0/);
}

test('backend-ci captures the real exit code of the AI-eval pipeline', () => {
  assertExitCodeCapture(backendCi);
});

test('backend-ci main go test has a timeout so a hung package fails instead of stalling the job', () => {
  assert.match(backendCi, /go test \$TESTABLE_PKGS[^\n]*-timeout\s+\d+m/);
});

test('the exit-code assertion actually distinguishes a swallowed failure', () => {
  // Reverse check: emulate the old implementation (plain `go test | tee`,
  // no capture) and re-run the SAME assertion function. If it still passes,
  // the contract test is a tautology and must be tightened.
  const broken = backendCi
    .split('\n')
    .filter((line) => !/PIPESTATUS|pipefail|AI_EVAL_STATUS/.test(line))
    .join('\n');
  assert.notEqual(broken, backendCi);
  assert.throws(() => assertExitCodeCapture(broken));
});

test('docs-gate runs the whole script test directory, not a single file', () => {
  assert.match(docsGate, /node --test scripts\/__tests__\/\*\.test\.js/);
  assert.doesNotMatch(docsGate, /node --test scripts\/__tests__\/docs-gate\.test\.js\s*$/m);
});

test('docs-gate triggers on every workflow and script the contract tests read', () => {
  const required = [
    '.github/workflows/backend-ci.yml',
    '.github/workflows/test-coverage-guard.yml',
    '.github/workflows/frontend-ci.yml',
    'scripts/test-coverage-guard.js',
    'scripts/__tests__/**',
    'itsm-backend/internal/services/**',
    'itsm-frontend/tests/e2e/**',
    '.gitignore',
    // github-ci-parity.test.js 的被测对象（2026-11-14 远端红灯复盘）
    '.github/workflows/star-history.yml',
    '.github/workflows/api-contract-check.yml',
    'itsm-backend/docs/**',
    'scripts/check-ci-parity.sh',
    'itsm-frontend/src/**/*.test.ts',
    'itsm-frontend/src/**/*.test.tsx',
  ];
  for (const name of ['pull_request', 'push']) {
    const trigger = section(docsGate, name, name === 'pull_request' ? 'push' : 'workflow_call');
    for (const p of required) {
      assert.ok(
        trigger.includes(`- '${p}'`),
        `docs-gate ${name} trigger is missing ${p} — contract tests would never re-run`
      );
    }
  }
});

test('test-coverage-guard triggers on both legacy and post-migration service trees', () => {
  const required = [
    'itsm-backend/handlers/**',
    'itsm-backend/service/**',
    'itsm-backend/internal/services/**',
  ];
  const pushTrigger = section(coverageGuard, 'push', 'pull_request');
  const prTrigger = section(coverageGuard, 'pull_request', 'workflow_dispatch');
  for (const p of required) {
    assert.ok(pushTrigger.includes(`- '${p}'`), `guard push trigger missing ${p}`);
    assert.ok(prTrigger.includes(`- '${p}'`), `guard PR trigger missing ${p}`);
  }
});

function assertStrictDiffObservation(yaml) {
  // Strip comment lines: the explanatory comment itself mentions
  // --strict-diff and must not be able to keep this assertion green.
  const code = yaml.split('\n').filter((line) => !/^\s*#/.test(line)).join('\n');
  assert.match(code, /--strict-diff[^\n]*\|\| true/, 'observation must be explicitly non-fatal');
  assert.match(code, /if:\s*always\(\)/, 'observation must run even when the hard gate failed');
  assert.match(code, /GITHUB_STEP_SUMMARY/, 'observation data must be visible to reviewers');
}

test('coverage guard wires a non-blocking strict-diff observation', () => {
  assertStrictDiffObservation(coverageGuard);
  // Reverse check against a workflow with the observation removed.
  const broken = coverageGuard
    .split('\n')
    .filter((line) => !/--strict-diff|GITHUB_STEP_SUMMARY|if: always\(\)/.test(line))
    .join('\n');
  assert.notEqual(broken, coverageGuard);
  assert.throws(() => assertStrictDiffObservation(broken));
});

test('make test includes the script guard self-tests', () => {
  assert.match(makefile, /^test-scripts:/m);
  assert.match(makefile, /^test:\s+[^\n]*test-scripts/m);
  assert.match(
    makefile,
    /^test-scripts:[\s\S]*?node --test scripts\/__tests__\/\*\.test\.js/m
  );
});
