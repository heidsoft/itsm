'use strict';

const assert = require('node:assert/strict');
const { mkdtempSync, mkdirSync, writeFileSync, rmSync } = require('node:fs');
const { tmpdir } = require('node:os');
const { join, resolve } = require('node:path');
const { spawnSync } = require('node:child_process');
const test = require('node:test');

const ROOT = resolve(__dirname, '..', '..');
const CHECKER = join(ROOT, 'scripts', 'check-version-consistency.js');

function fixture(overrides = {}) {
  const dir = mkdtempSync(join(tmpdir(), 'itsm-version-guard-'));
  const files = {
    '.env.prod.example': 'VERSION=1.6.9\n',
    'itsm-backend/router/router.go': 'const defaultAppVersion = "1.6.19"\n',
    'itsm-backend/router/ga_readiness.go': 'Version: "1.6.8",\n',
    'itsm-frontend/package.json': '{ "version": "1.6.19" }\n',
    'docs/documentation-governance.md': '最新发布标签为 `v1.6.9`；工作树可能包含未发布变更\n',
    ...overrides,
  };
  for (const [name, content] of Object.entries(files)) {
    const target = join(dir, name);
    mkdirSync(resolve(target, '..'), { recursive: true });
    writeFileSync(target, content, 'utf8');
  }
  mkdirSync(join(dir, 'scripts'), { recursive: true });
  writeFileSync(join(dir, 'scripts', 'version-drift-baseline.json'), JSON.stringify({
    surfaces: {
      '.env.prod.example VERSION': '1.6.9',
      'backend defaultAppVersion': '1.6.19',
      'GA readiness version': '1.6.8',
      'frontend package version': '1.6.19',
      'docs release baseline': '1.6.9',
    },
  }), 'utf8');
  return dir;
}

function run(root, ...args) {
  return spawnSync(process.execPath, [CHECKER, `--canonical=${args[0] || 'v1.6.59'}`, ...args.slice(1)], {
    cwd: root,
    env: { ...process.env, VERSION_GUARD_ROOT: root },
    encoding: 'utf8',
  });
}

test('registered historical drift is warned but does not block', () => {
  const dir = fixture();
  try {
    const result = run(dir);
    assert.equal(result.status, 0, result.stderr);
    assert.equal((result.stdout.match(/WARN /g) || []).length, 5);
    assert.equal(result.stdout.includes('FAIL'), false);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

test('reverse: a new unregistered version on any surface turns red', () => {
  for (const [name, content] of [
    ['.env.prod.example', 'VERSION=9.9.9\n'],
    ['itsm-backend/router/router.go', 'const defaultAppVersion = "9.9.9"\n'],
    ['itsm-backend/router/ga_readiness.go', 'Version: "9.9.9",\n'],
    ['itsm-frontend/package.json', '{ "version": "9.9.9" }\n'],
    ['docs/documentation-governance.md', '最新发布标签为 `v9.9.9`\n'],
  ]) {
    const dir = fixture({ [name]: content });
    try {
      const result = run(dir);
      assert.equal(result.status, 1, `${name} 的新漂移必须失败：${result.stdout}`);
      assert.match(result.stdout + result.stderr, /FAIL /);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  }
});

test('strict mode rejects registered drift once alignment is complete', () => {
  const dir = fixture();
  try {
    const result = run(dir, 'v1.6.59', '--strict');
    assert.equal(result.status, 1);
    assert.match(result.stdout + result.stderr, /仍与 canonical/);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

test('missing surface file fails closed', () => {
  const dir = fixture();
  try {
    rmSync(join(dir, 'itsm-frontend/package.json'));
    const result = run(dir);
    assert.equal(result.status, 1);
    assert.match(result.stdout + result.stderr, /文件不存在/);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
