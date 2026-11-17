'use strict';

/**
 * Repository hygiene contracts.
 *
 * Two run artifacts were committed by accident and lived in the index for
 * months even though .gitignore already covered them:
 *   - test-coverage-guard.json            (a failed-run diagnostic from the
 *                                          coverage guard, rewritten in CI)
 *   - itsm-frontend/test-results/.last-run.json
 *                                        (Playwright's last-failed-test state)
 *
 * Both were removed from the index on 2026-11-14 while the on-disk copies
 * were kept. `git rm --cached` alone is easy to undo by a careless
 * `git add -A`, so these tests make a re-introduction fail in CI.
 */

const assert = require('node:assert/strict');
const { spawnSync } = require('node:child_process');
const path = require('node:path');
const test = require('node:test');

const root = path.resolve(__dirname, '..', '..');

const ARTIFACTS = [
  'test-coverage-guard.json',
  'itsm-frontend/test-results/.last-run.json',
];

function git(args) {
  const r = spawnSync('git', args, { cwd: root, encoding: 'utf8' });
  assert.equal(r.status, 0, `git ${args.join(' ')} failed: ${r.stderr}`);
  return r.stdout.trim();
}

function trackedSet() {
  return new Set(git(['ls-files']).split('\n').filter(Boolean));
}

function isIgnored(rel) {
  const r = spawnSync('git', ['check-ignore', '-q', '--', rel], { cwd: root });
  return r.status === 0;
}

function assertArtifactsClean(tracked, ignored) {
  for (const rel of ARTIFACTS) {
    assert.ok(
      !tracked.has(rel),
      `${rel} is tracked again — run "git rm --cached -- ${rel}" (keep the local file)`
    );
    assert.ok(
      ignored[rel],
      `${rel} is not covered by .gitignore — removing it from the index is not enough`
    );
  }
}

test('run artifacts are not tracked and are covered by .gitignore', () => {
  const ignored = Object.fromEntries(ARTIFACTS.map((rel) => [rel, isIgnored(rel)]));
  assertArtifactsClean(trackedSet(), ignored);
});

test('reverse: re-adding an artifact to the index makes the check fail', () => {
  const ignored = Object.fromEntries(ARTIFACTS.map((rel) => [rel, true]));
  const tracked = trackedSet();
  tracked.add(ARTIFACTS[0]);
  assert.throws(
    () => assertArtifactsClean(tracked, ignored),
    /is tracked again/
  );
});

test('reverse: dropping the ignore rule makes the check fail', () => {
  const ignored = Object.fromEntries(ARTIFACTS.map((rel) => [rel, rel !== ARTIFACTS[1]]));
  assert.throws(
    () => assertArtifactsClean(trackedSet(), ignored),
    /not covered by \.gitignore/
  );
});

test('no other well-known run artifacts leaked into the index', () => {
  const offenders = [
    'junit.xml',
    'coverage.out',
    'service_coverage.out',
    'test-results/.last-run.json',
  ];
  const tracked = trackedSet();
  for (const rel of offenders) {
    assert.ok(!tracked.has(rel), `${rel} is a run artifact and must not be tracked`);
  }
});
