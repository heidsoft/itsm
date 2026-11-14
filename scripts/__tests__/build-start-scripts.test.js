'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const test = require('node:test');

const root = path.resolve(__dirname, '..', '..');

function fakeDockerEnvironment() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'itsm-build-script-'));
  const log = path.join(dir, 'docker.log');
  const docker = path.join(dir, 'docker');
  fs.writeFileSync(
    docker,
    `#!/bin/sh
if [ "$1" = "info" ]; then exit 0; fi
printf '%s\\n' "$*" >> "$DOCKER_TEST_LOG"
`,
    { mode: 0o755 }
  );
  return {
    env: {
      ...process.env,
      PATH: `${dir}:${process.env.PATH}`,
      DOCKER_TEST_LOG: log,
      NO_COLOR: '1',
    },
    log,
  };
}

test('image builder treats a lone version argument as a version, not a service filter', () => {
  const fixture = fakeDockerEnvironment();
  const result = spawnSync('bash', ['scripts/build-images.sh', 'v1.2.0'], {
    cwd: root,
    env: fixture.env,
    encoding: 'utf8',
  });

  assert.equal(result.status, 0, result.stderr || result.stdout);
  const builds = fs.readFileSync(fixture.log, 'utf8').trim().split('\n');
  assert.equal(builds.length, 4);
  assert.ok(builds.every(line => line.includes(':v1.2.0')));
});

test('image builder validates service filters before invoking docker build', () => {
  const fixture = fakeDockerEnvironment();
  const result = spawnSync(
    'bash',
    ['scripts/build-images.sh', 'latest', '', 'backend', 'unknown-service'],
    { cwd: root, env: fixture.env, encoding: 'utf8' }
  );

  assert.equal(result.status, 2);
  assert.match(result.stdout, /Unknown service/);
  assert.equal(fs.existsSync(fixture.log), false);
});

test('image builder validates tags and normalizes the registry separator', () => {
  const invalid = fakeDockerEnvironment();
  const invalidResult = spawnSync('bash', ['scripts/build-images.sh', 'bad tag'], {
    cwd: root,
    env: invalid.env,
    encoding: 'utf8',
  });
  assert.equal(invalidResult.status, 2);
  assert.match(invalidResult.stderr, /Invalid image version/);

  const valid = fakeDockerEnvironment();
  const validResult = spawnSync(
    'bash',
    ['scripts/build-images.sh', 'v2', 'registry.example.com/team', 'frontend'],
    { cwd: root, env: valid.env, encoding: 'utf8' }
  );
  assert.equal(validResult.status, 0, validResult.stderr || validResult.stdout);
  assert.match(fs.readFileSync(valid.log, 'utf8'), /registry\.example\.com\/team\/itsm-frontend:v2/);
});

test('standalone start fails with an actionable message when build output is absent', () => {
  const empty = fs.mkdtempSync(path.join(os.tmpdir(), 'itsm-standalone-start-'));
  const result = spawnSync(
    process.execPath,
    [path.join(root, 'itsm-frontend', 'scripts', 'start-standalone.mjs')],
    { cwd: empty, encoding: 'utf8' }
  );

  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Run "npm run build"/);
});

test('standalone preparation copies static and public assets into the runtime bundle', () => {
  const fixture = fs.mkdtempSync(path.join(os.tmpdir(), 'itsm-standalone-prepare-'));
  fs.mkdirSync(path.join(fixture, '.next', 'standalone'), { recursive: true });
  fs.mkdirSync(path.join(fixture, '.next', 'static'), { recursive: true });
  fs.mkdirSync(path.join(fixture, 'public'), { recursive: true });
  fs.writeFileSync(path.join(fixture, '.next', 'standalone', 'server.js'), '');
  fs.writeFileSync(path.join(fixture, '.next', 'static', 'asset.js'), 'static');
  fs.writeFileSync(path.join(fixture, 'public', 'health.txt'), 'public');

  const result = spawnSync(
    process.execPath,
    [path.join(root, 'itsm-frontend', 'scripts', 'prepare-standalone.mjs')],
    { cwd: fixture, encoding: 'utf8' }
  );

  assert.equal(result.status, 0, result.stderr);
  assert.equal(
    fs.readFileSync(path.join(fixture, '.next', 'standalone', '.next', 'static', 'asset.js'), 'utf8'),
    'static'
  );
  assert.equal(
    fs.readFileSync(path.join(fixture, '.next', 'standalone', 'public', 'health.txt'), 'utf8'),
    'public'
  );
});

test('production dry-run exits before runtime verification and success reporting', () => {
  const script = fs.readFileSync(path.join(root, 'scripts', 'deploy-prod.sh'), 'utf8');
  const dryRunExit = script.indexOf('print_banner "Production Dry Run Complete"');
  const verification = script.indexOf('# Phase 5', dryRunExit);
  const deploymentSuccess = script.indexOf('print_banner "Deployment Successful!"', dryRunExit);

  assert.ok(dryRunExit >= 0, 'dry-run completion branch is missing');
  assert.ok(verification > dryRunExit, 'dry-run must exit before runtime verification');
  assert.ok(deploymentSuccess > dryRunExit, 'dry-run must exit before deployment success reporting');
  assert.match(
    script.slice(dryRunExit, verification),
    /return 0/,
    'dry-run branch must return before runtime verification'
  );
});

test('production deploy stops immediately when a compose start phase fails', () => {
  const script = fs.readFileSync(path.join(root, 'scripts', 'deploy-prod.sh'), 'utf8');

  // 锁语义而非字面：服务列表与 compose 参数（--env-file / -f）允许随部署演进调整，
  // 但每个启动阶段都必须是 `if ! run dc ... up -d ...; then ... return 1` 的 fail-fast 形态。
  const guarded = [...script.matchAll(/if ! run dc ([^\n]*?) up -d ([^\n;]+); then/g)];
  assert.ok(
    guarded.length >= 4,
    `expected at least 4 fail-guarded compose start phases, got ${guarded.length}`
  );
  for (const match of guarded) {
    const block = script.slice(match.index, match.index + 400);
    assert.match(
      block,
      /return 1/,
      `start phase [${match[2]}] must return 1 immediately on compose failure`
    );
  }
  for (const required of ['itsm-backend', 'itsm-frontend']) {
    assert.ok(
      guarded.some((match) => match[2].split(/\s+/).includes(required)),
      `missing fail-guarded start phase for ${required}`
    );
  }
});

test('production init and backend services share one immutable image contract', () => {
  const compose = fs.readFileSync(path.join(root, 'docker-compose.prod.yml'), 'utf8');
  const images = compose
    .split('\n')
    .filter((line) => /^\s*image:\s*itsm-backend:/.test(line))
    .map((line) => line.trim());

  assert.ok(
    images.length >= 2,
    `expected at least 2 itsm-backend image references (init + backend), got ${images.length}`
  );
  assert.equal(
    new Set(images).size,
    1,
    'itsm-init, itsm-backend and any other backend worker must pin the exact same image expression'
  );
  // VERSION 必须强制提供（:?），不能退回默认值（:-），否则会部署未打标签的 latest 镜像。
  assert.match(images[0], /\$\{VERSION:\?/, 'VERSION must be required, not defaulted');
  assert.doesNotMatch(images[0], /:-/, 'a default VERSION allows deploying an untagged image');
});

test('backend production image excludes local binaries and coverage artifacts', () => {
  const dockerignore = fs.readFileSync(
    path.join(root, 'itsm-backend', '.dockerignore'),
    'utf8'
  );
  const ignoredEntries = new Set(
    dockerignore.split(/\r?\n/).map(line => line.trim()).filter(Boolean)
  );

  for (const entry of ['bin', 'main', 'deploy', 'coverage.out', '*_coverage.out']) {
    assert.ok(
      ignoredEntries.has(entry),
      `${entry} must not be sent to the production Docker build context`
    );
  }
});

test('frontend production image excludes Jest cache artifacts', () => {
  const ignoredEntries = new Set(
    fs.readFileSync(path.join(root, 'itsm-frontend', '.dockerignore'), 'utf8')
      .split(/\r?\n/)
      .map(line => line.trim())
      .filter(Boolean)
  );

  assert.ok(
    ignoredEntries.has('.jest-cache'),
    '.jest-cache must not be sent to the production Docker build context'
  );
});
