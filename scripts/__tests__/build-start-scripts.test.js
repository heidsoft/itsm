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
