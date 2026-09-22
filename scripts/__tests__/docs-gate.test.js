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
    'docs/delivery/report.md': '## Result\n全部通过\ncontext\n证据日期：2026-09-05\n',
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = run(root, 'check-release-claims.sh');
  assert.equal(result.status, 0, result.stdout + result.stderr);
});

// ---------------------------------------------------------------------------
// Gate C.6 — 产品口径漂移守卫
//
// C.6 是"口径不一致 = 构建失败"的守卫。它的价值完全取决于能否真的 FAIL，
// 所以每个断言都成对出现：先证明违规会阻断，再证明合规会放行。
// ---------------------------------------------------------------------------

function runDrift(root, env = {}) {
  return spawnSync('bash', [join(gateDir, 'check-product-drift.sh'), '--strict'], {
    cwd: root,
    env: { ...process.env, DOCS_GATE_ROOT: root, ...env },
    encoding: 'utf8',
  });
}

// 最小合规仓库：一张 README 成熟度表 + 对应能力契约表 + 一个已接线域包
function driftFixture(extra = {}) {
  return fixture({
    'README.md': [
      '# readme',
      '',
      '## 能力与成熟度',
      '',
      '| 能力域 | 状态 | 说明 |',
      '|:---|:---:|:---|',
      '| 工单与事件 | 可用 | ok |',
      '| 变更管理 | 预览 | ok |',
      '',
      '## 快速开始',
      '',
    ].join('\n'),
    'docs/product/itsm-commercial-capability-contract.md': [
      '# contract',
      '',
      '| 能力域 | 当前成熟度 | 已有业务证据 | 商业化缺口 |',
      '|---|---|---|---|',
      '| 工单/事件 | GA 候选 | e | g |',
      '| 变更 | Pilot | e | g |',
      '',
    ].join('\n'),
    'itsm-backend/handlers/incident/handler.go': 'package incident\n',
    'itsm-backend/router/router.go':
      'package router\n\nimport _ "itsm-backend/handlers/incident"\n',
    'baseline.txt': 'handlers_dirs=10\nfrontend_pages=10\n',
    'waivers.txt': '# no waivers\n',
    ...extra,
  });
}

function driftEnv(root, overrides = {}) {
  return {
    PRODUCT_DRIFT_BASELINE: join(root, 'baseline.txt'),
    PRODUCT_DRIFT_WAIVERS: join(root, 'waivers.txt'),
    PRODUCT_DRIFT_TODAY: '2026-09-22',
    ...overrides,
  };
}

test('C.6.1 blocks a README/contract maturity conflict', (t) => {
  // README 说"可用"，能力契约说"Pilot" —— 二者不可能同时为真
  const root = driftFixture({
    'README.md': [
      '## 能力与成熟度',
      '',
      '| 能力域 | 状态 | 说明 |',
      '|:---|:---:|:---|',
      '| 工单与事件 | 可用 | ok |',
      '| 变更管理 | 可用 | ok |',
      '',
      '## 快速开始',
      '',
    ].join('\n'),
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = runDrift(root, driftEnv(root));
  assert.equal(result.status, 1, result.stdout + result.stderr);
  assert.match(result.stdout, /口径冲突/);
  assert.match(result.stdout, /变更管理/);
});

test('C.6.1 blocks a contract domain missing from the README table', (t) => {
  const root = driftFixture({
    'README.md': [
      '## 能力与成熟度',
      '',
      '| 能力域 | 状态 | 说明 |',
      '|:---|:---:|:---|',
      '| 工单与事件 | 可用 | ok |',
      '',
      '## 快速开始',
      '',
    ].join('\n'),
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = runDrift(root, driftEnv(root));
  assert.equal(result.status, 1, result.stdout + result.stderr);
  assert.match(result.stdout, /对外承诺漏报/);
  assert.match(result.stdout, /变更/);
});

test('C.6.1 lets a registered waiver absorb the known conflict', (t) => {
  const root = driftFixture({
    'README.md': [
      '## 能力与成熟度',
      '',
      '| 能力域 | 状态 | 说明 |',
      '|:---|:---:|:---|',
      '| 工单与事件 | 可用 | ok |',
      '| 变更管理 | 可用 | ok |',
      '',
      '## 快速开始',
      '',
    ].join('\n'),
    'waivers.txt': 'C.6.1|变更管理|owner|2099-01-01|pending product decision\n',
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = runDrift(root, driftEnv(root));
  assert.equal(result.status, 0, result.stdout + result.stderr);
  assert.match(result.stdout, /WAIVED: C\.6\.1\/变更管理/);
});

test('C.6.1 re-blocks once a waiver expires', (t) => {
  const root = driftFixture({
    'README.md': [
      '## 能力与成熟度',
      '',
      '| 能力域 | 状态 | 说明 |',
      '|:---|:---:|:---|',
      '| 工单与事件 | 可用 | ok |',
      '| 变更管理 | 可用 | ok |',
      '',
      '## 快速开始',
      '',
    ].join('\n'),
    'waivers.txt': 'C.6.1|变更管理|owner|2020-01-01|expired\n',
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = runDrift(root, driftEnv(root));
  assert.equal(result.status, 1, result.stdout + result.stderr);
  assert.match(result.stdout, /豁免已过期/);
});

test('C.6.2 blocks a ghost domain in the AGENTS.md list', (t) => {
  const root = driftFixture({
    'AGENTS.md': '- **handlers/<domain>/** Existing domains: incident, cab.\n',
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = runDrift(root, driftEnv(root));
  assert.equal(result.status, 1, result.stdout + result.stderr);
  assert.match(result.stdout, /cab/);
  assert.match(result.stdout, /不存在于/);
});

test('C.6.3 blocks a handler package no production code imports', (t) => {
  const root = driftFixture({
    'itsm-backend/handlers/orphan/handler.go': 'package orphan\n',
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = runDrift(root, driftEnv(root));
  assert.equal(result.status, 1, result.stdout + result.stderr);
  assert.match(result.stdout, /零路由域包 handlers\/orphan/);
  // 已接线的 incident 不应被误判
  assert.doesNotMatch(result.stdout, /零路由域包 handlers\/incident/);
});

test('C.6.3 blocks an empty handler directory', (t) => {
  const root = driftFixture({
    'itsm-backend/handlers/ghost/.keep': '',
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = runDrift(root, driftEnv(root));
  assert.equal(result.status, 1, result.stdout + result.stderr);
  assert.match(result.stdout, /空域包 handlers\/ghost/);
});

test('C.6.4 blocks surface growth past the ratchet baseline', (t) => {
  const root = driftFixture({
    'baseline.txt': 'handlers_dirs=0\n',
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = runDrift(root, driftEnv(root));
  assert.equal(result.status, 1, result.stdout + result.stderr);
  assert.match(result.stdout, /表面增长：handlers_dirs/);
});

test('C.6.4 passes when the surface shrinks below baseline', (t) => {
  const root = driftFixture({
    'baseline.txt': 'handlers_dirs=99\n',
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = runDrift(root, driftEnv(root));
  assert.equal(result.status, 0, result.stdout + result.stderr);
  assert.match(result.stdout, /已下降/);
});

test('C.6.5 blocks an undisclosed frontend coverage scope', (t) => {
  const root = driftFixture({
    'itsm-frontend/jest.config.js': [
      'module.exports = {',
      "  collectCoverageFrom: ['src/lib/**/*.{ts,tsx}'],",
      '};',
      '',
    ].join('\n'),
    'ROADMAP.md': '## 📊 Key Metrics\n\n| Metric | v2.0 target |\n|:---|---:|\n| Frontend coverage | 60% |\n',
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = runDrift(root, driftEnv(root));
  assert.equal(result.status, 1, result.stdout + result.stderr);
  assert.match(result.stdout, /未披露该口径/);
});

test('C.6.5 passes once the scope is disclosed', (t) => {
  const root = driftFixture({
    'itsm-frontend/jest.config.js': [
      'module.exports = {',
      "  collectCoverageFrom: ['src/lib/**/*.{ts,tsx}'],",
      '};',
      '',
    ].join('\n'),
    'ROADMAP.md':
      '## 📊 Key Metrics\n\n仅统计 `src/lib/**`，不代表产品覆盖率。\n\n| Metric | v2.0 target |\n|:---|---:|\n',
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const result = runDrift(root, driftEnv(root));
  assert.equal(result.status, 0, result.stdout + result.stderr);
  assert.match(result.stdout, /已披露前端覆盖率口径/);
});
