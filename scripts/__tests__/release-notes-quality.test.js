'use strict';

const assert = require('node:assert/strict');
const { mkdtempSync, writeFileSync, rmSync } = require('node:fs');
const { tmpdir } = require('node:os');
const { join, resolve } = require('node:path');
const { spawnSync } = require('node:child_process');
const test = require('node:test');

const root = resolve(__dirname, '..', '..');
const gate = join(root, 'scripts', 'check-release-notes-quality.sh');

function check(content) {
  const dir = mkdtempSync(join(tmpdir(), 'itsm-release-notes-quality-'));
  try {
    const file = join(dir, 'notes.md');
    writeFileSync(file, content, 'utf8');
    return spawnSync('bash', [gate, file], { encoding: 'utf8' });
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
}

test('accepts user-value oriented release notes', () => {
  const result = check([
    '工单列表支持按处理人筛选，值班组可以更快找到待跟进的工单。',
    '升级前请备份数据库；迁移命令与验证方法见升级文档。',
    '验证范围：go test ./handlers/ticket/，CI fresh-install gate。',
  ].join('\n'));
  assert.equal(result.status, 0, result.stderr);
});

test('rejects empty notes', () => {
  const result = check('');
  assert.equal(result.status, 1);
  assert.match(result.stderr, /发布说明为空/);
});

test('reverse: each banned phrase makes the gate fail', () => {
  for (const phrase of [
    '史诗级升级', '全面赋能', '革命性突破', '企业级能力全面增强',
    '全部通过', '立即上线', 'production ready', 'all green',
  ]) {
    const result = check(`本次${phrase}，请放心升级。`);
    assert.equal(result.status, 1, `phrase=${phrase} 必须被拒绝：${result.stdout}`);
    assert.match(result.stderr, /无证据的宣传语/);
  }
});
