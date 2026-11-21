'use strict';

const assert = require('node:assert/strict');
const { mkdtempSync, readdirSync, readFileSync, writeFileSync, rmSync } = require('node:fs');
const { tmpdir } = require('node:os');
const { join, resolve } = require('node:path');
const test = require('node:test');

const root = resolve(__dirname, '..', '..');

// /metrics is a production surface. docker-compose.prod.yml defaults to
// METRICS_REQUIRE_AUTH=true, and the router only skips AuthMiddleware when the
// value is exactly "false". A shipped example that sets false silently changes
// the security posture of every deployment that copies it.
function findDisabledMetricsAuth(content) {
  const hits = [];
  content.split(/\r?\n/).forEach((raw, index) => {
    const line = raw.trim();
    if (!line || line.startsWith('#')) return;
    const match = line.match(/^METRICS_REQUIRE_AUTH\s*=\s*"?([^"\s#]+)"?/);
    if (match && match[1].toLowerCase() === 'false') {
      hits.push({ line: index + 1, value: match[1] });
    }
  });
  return hits;
}

function exampleFiles(directory) {
  return readdirSync(directory, { withFileTypes: true })
    .filter((entry) => entry.isFile() && /\.env.*\.example$/.test(entry.name))
    .map((entry) => join(directory, entry.name));
}

test('shipped env examples never disable /metrics authentication', () => {
  const files = exampleFiles(root);
  assert.ok(files.length > 0, 'guard must inspect at least one shipped env example');

  const violations = [];
  for (const file of files) {
    for (const hit of findDisabledMetricsAuth(readFileSync(file, 'utf8'))) {
      violations.push(`${file}:${hit.line}`);
    }
  }
  assert.deepEqual(violations, [],
    `这些示例文件关闭了 /metrics 鉴权，请改为 true：${violations.join(', ')}`);
});

test('compose production default keeps /metrics authentication enabled', () => {
  const compose = readFileSync(join(root, 'docker-compose.prod.yml'), 'utf8');
  assert.match(compose, /METRICS_REQUIRE_AUTH=\$\{METRICS_REQUIRE_AUTH:-true\}/,
    '生产 compose 的 METRICS_REQUIRE_AUTH 默认值必须是 true');
});

test('reverse: the guard actually turns red on a disabled example', () => {
  const dir = mkdtempSync(join(tmpdir(), 'itsm-prod-env-guard-'));
  try {
    const file = join(dir, '.env.prod.example');
    writeFileSync(file, '# comment only\nMETRICS_REQUIRE_AUTH=false # bad\n');
    const hits = findDisabledMetricsAuth(readFileSync(file, 'utf8'));
    assert.equal(hits.length, 1, '守卫必须命中未注释的 false 值');
    assert.equal(hits[0].line, 2);

    writeFileSync(file, '# METRICS_REQUIRE_AUTH=false is documented, not applied\nMETRICS_REQUIRE_AUTH=true\n');
    assert.deepEqual(findDisabledMetricsAuth(readFileSync(file, 'utf8')), [],
      '注释中的 false 与显式 true 都不能误报');
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
