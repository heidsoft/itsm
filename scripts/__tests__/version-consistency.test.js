'use strict';

const assert = require('node:assert/strict');
const { mkdtempSync, mkdirSync, writeFileSync, rmSync } = require('node:fs');
const { tmpdir } = require('node:os');
const { join, resolve } = require('node:path');
const { spawnSync } = require('node:child_process');
const test = require('node:test');

const ROOT = resolve(__dirname, '..', '..');
const CHECKER = join(ROOT, 'scripts', 'check-version-consistency.js');

function fixture(overrides = {}, baselineOverrides = {}) {
  const dir = mkdtempSync(join(tmpdir(), 'itsm-version-guard-'));
  const files = {
    '.env.prod.example': 'VERSION=1.6.60\n',
    'itsm-backend/router/router.go': 'const defaultAppVersion = "1.6.60"\n',
    'itsm-backend/router/ga_readiness.go': 'Version: "1.6.60",\n',
    'itsm-frontend/package.json': '{ "version": "1.6.60" }\n',
    'docs/documentation-governance.md': '最新发布标签为 `v1.6.60`；工作树可能包含未发布变更\n',
    'itsm-backend/handlers/mcp/adapters.go': 'const defaultVersion = "1.6.60"\n',
    'itsm-backend/main.go': '// @version 1.6.60\n',
    'UPGRADE.md': '> 当前稳定版本：`1.6.60`（见 `CHANGELOG.md`）。\n',
    'deploy/k8s/20-backend.yaml': 'image: ghcr.io/heidsoft/itsm-backend:1.6.60\n',
    'deploy/k8s/30-frontend.yaml': 'image: ghcr.io/heidsoft/itsm-frontend:1.6.60\n',
    'itsm-backend/go.mod': 'module itsm-backend\n\ngo 1.26.10\n',
    'README.md': '[![Go](https://img.shields.io/badge/Go-1.26.10-00ADD8?logo=go)](https://go.dev/)\n',
    'README.en.md': '[![Go](https://img.shields.io/badge/Go-1.26.10-00ADD8?style=flat&logo=go)](https://golang.org)\n',
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
      '.env.prod.example VERSION': '1.6.60',
      'backend defaultAppVersion': '1.6.60',
      'GA readiness version': '1.6.60',
      'frontend package version': '1.6.60',
      'docs release baseline': '1.6.60',
      ...baselineOverrides,
    },
  }), 'utf8');
  return dir;
}

function run(root, ...args) {
  return spawnSync(process.execPath, [CHECKER, `--canonical=${args[0] || 'v1.6.60'}`, ...args.slice(1)], {
    cwd: root,
    env: { ...process.env, VERSION_GUARD_ROOT: root },
    encoding: 'utf8',
  });
}

test('aligned surfaces pass in both advisory and strict mode', () => {
  for (const args of [[], ['v1.6.60', '--strict']]) {
    const dir = fixture();
    try {
      const result = run(dir, ...args);
      assert.equal(result.status, 0, result.stdout + result.stderr);
      assert.equal(result.stdout.includes('FAIL'), false);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  }
});

test('reverse: a new unregistered version on any surface turns red', () => {
  for (const [name, content] of [
    ['.env.prod.example', 'VERSION=9.9.9\n'],
    ['itsm-backend/router/router.go', 'const defaultAppVersion = "9.9.9"\n'],
    ['itsm-backend/router/ga_readiness.go', 'Version: "9.9.9",\n'],
    ['itsm-frontend/package.json', '{ "version": "9.9.9" }\n'],
    ['docs/documentation-governance.md', '最新发布标签为 `v9.9.9`\n'],
    ['itsm-backend/handlers/mcp/adapters.go', 'const defaultVersion = "9.9.9"\n'],
    ['itsm-backend/main.go', '// @version 9.9.9\n'],
    ['UPGRADE.md', '> 当前稳定版本：`9.9.9`（见 `CHANGELOG.md`）。\n'],
    ['deploy/k8s/20-backend.yaml', 'image: ghcr.io/heidsoft/itsm-backend:9.9.9\n'],
    ['deploy/k8s/30-frontend.yaml', 'image: ghcr.io/heidsoft/itsm-frontend:9.9.9\n'],
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

test('registered historical drift is warned in advisory mode, blocked in strict mode', () => {
  const dir = fixture(
    { '.env.prod.example': 'VERSION=1.6.9\n' },
    { '.env.prod.example VERSION': '1.6.9' },
  );
  try {
    const advisory = run(dir);
    assert.equal(advisory.status, 0, advisory.stderr);
    assert.match(advisory.stdout, /WARN .*已登记的历史漂移/);

    const strict = run(dir, 'v1.6.60', '--strict');
    assert.equal(strict.status, 1);
    assert.match(strict.stdout + strict.stderr, /仍与 canonical/);
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

test('Go badge must match go.mod go directive', () => {
  const dir = fixture({ 'README.md': '[![Go](https://img.shields.io/badge/Go-1.25.13-00ADD8?logo=go)](https://go.dev/)\n' });
  try {
    const result = run(dir);
    assert.equal(result.status, 1);
    assert.match(result.stdout + result.stderr, /Go 徽章 1\.25\.13 与 itsm-backend\/go\.mod 的 1\.26\.10 不一致/);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
