'use strict';

const assert = require('node:assert/strict');
const { execFileSync, spawnSync } = require('node:child_process');
const { mkdtempSync, mkdirSync, writeFileSync, rmSync } = require('node:fs');
const { tmpdir } = require('node:os');
const { join, resolve } = require('node:path');
const test = require('node:test');

const gateDir = resolve(__dirname, '..', 'docs-gate');

function fixture(files) {
  const root = mkdtempSync(join(tmpdir(), 'itsm-docs-gate-'));
  for (const [name, content] of Object.entries(files)) {
    const path = join(root, name);
    mkdirSync(resolve(path, '..'), { recursive: true });
    writeFileSync(path, content);
  }
  execFileSync('git', ['init', '-q'], { cwd: root });
  execFileSync('git', ['add', '.'], { cwd: root });
  return root;
}

function run(root, script) {
  return spawnSync('bash', [join(gateDir, script), '--strict'], {
    cwd: root,
    env: { ...process.env, DOCS_GATE_ROOT: root },
    encoding: 'utf8',
  });
}

test('C.1 only blocks hardcoded credentials in production release surfaces', (t) => {
  const root = fixture({
    'docker-compose.dev.yml': 'ADMIN_PASSWORD=admin123\n',
    'docker-compose.prod.yml': 'ADMIN_PASSWORD=admin123\n',
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = run(root, 'check-hardcoded-passwords.sh');
  assert.equal(result.status, 1);
  assert.match(result.stdout, /docker-compose\.prod\.yml:1/);
  assert.doesNotMatch(result.stdout, /docker-compose\.dev\.yml:1/);
});

test('C.2 handles tracked markdown filenames containing spaces', (t) => {
  const root = fixture({
    'ROADMAP.md': '# canonical\n',
    'docs/release plan.md': '## Release Timeline\nv1.0 v1.1 v2.0\n',
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = run(root, 'check-duplicate-roadmap.sh');
  assert.equal(result.status, 1);
  assert.match(result.stdout, /docs\/release plan\.md/);
  assert.doesNotMatch(result.stderr, /cannot open/);
});

test('C.3 resolves links relative to each markdown file', (t) => {
  const root = fixture({
    'README.md': '# root\n',
    'docs/guide with spaces.md': '[root](../README.md)\n',
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = run(root, 'check-broken-links.sh');
  assert.equal(result.status, 0, result.stdout + result.stderr);
});

test('C.4 accepts an evidence anchor after a release claim', (t) => {
  const root = fixture({
    'docs/release/report.md': '## Result\n全部通过\ncontext\n证据日期：2026-09-05\n',
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = run(root, 'check-release-claims.sh');
  assert.equal(result.status, 0, result.stdout + result.stderr);
});
