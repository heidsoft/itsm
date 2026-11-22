'use strict';

const assert = require('node:assert/strict');
const { spawnSync } = require('node:child_process');
const { join } = require('node:path');
const test = require('node:test');

function extract(tag, changelog) {
  return spawnSync('bash', [join(__dirname, '../release-notes.sh'), tag, '/dev/stdin'], {
    input: changelog,
    encoding: 'utf8',
  });
}

test('publishes only the public notes for the exact tag, preserving Markdown spacing', () => {
  const result = extract('v1.6.12', [
    '## [Unreleased]', '### 发布说明', '- 下个版本',
    '## [1x6x12]', '- 近似版本号',
    '## [1.6.12] - 2026-11-21', '### 发布说明', '- 新增筛选',
    '', '升级后检查列表。', '### Fixed', '- 内部测试细节',
    '## [1.6.11]', '- 上个版本', '',
  ].join('\n'));
  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.stdout, '- 新增筛选\n\n升级后检查列表。\n');
});

test('uses the historical changelog entry when there is no public section', () => {
  const result = extract('v1.6.11', '## [1.6.11]\n### Fixed\n\n- 修复筛选\n\n---\n');
  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.stdout, '### Fixed\n\n- 修复筛选\n');
});

test('rejects missing versions and empty public notes', () => {
  const changelog = '## [1.6.12]\n### 发布说明\n\n### Fixed\n- 内部记录\n';
  for (const tag of ['v1.6.12', 'v1.6.12-rc.1']) {
    const result = extract(tag, changelog);
    assert.equal(result.status, 1);
    assert.equal(result.stdout, '');
    assert.ok(result.stderr.includes(tag));
  }
});
