/**
 * 一次性栈守卫的离线回归：不需要真实栈，只验证「哪些目标会被拒绝」。
 *
 * 为什么要有这个文件（2026-10-02 实测）：本机 127.0.0.1:8090 是生产容器
 * （`DB_NAME=itsm_prod`），而 `[::1]:8090` 是另一个进程的本机二进制；前端代理
 * upstream 的默认值恰好是 `http://localhost:8090`。所以「地址在环回白名单里」
 * 并不等于安全，守卫必须同时校验 proof 记录的代理 upstream。这四条拒绝分支
 * 之前只能靠手工搭栈逐条点，锁不住后续改动。
 */
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

import { test, expect } from '@playwright/test';

import { isolatedBaseURL } from './harness';

const REAL_ENV = {
  ITSM_E2E_ISOLATED_STACK: process.env.ITSM_E2E_ISOLATED_STACK,
  ITSM_E2E_STACK_PROOF: process.env.ITSM_E2E_STACK_PROOF,
  PLAYWRIGHT_BASE_URL: process.env.PLAYWRIGHT_BASE_URL,
};

/** 每个用例用不同的合法环回地址，避免 harness 的 proof 缓存串到下一个用例。 */
const TARGETS = [
  'http://localhost:3000',
  'http://127.0.0.1:3000',
  'http://localhost:3001',
  'http://127.0.0.1:3001',
];

let dirIndex = 0;

function proofFile(overrides: Record<string, unknown>, baseURL: string): string {
  const dir = mkdtempSync(join(tmpdir(), `itsm-guard-proof-${dirIndex++}-`));
  const file = join(dir, 'proof.json');
  writeFileSync(
    file,
    JSON.stringify({
      project: 'itsm-e2e-guard',
      runId: 'guard',
      baseURL,
      backendDirectURL: 'http://127.0.0.1:18090',
      proxyUpstream: 'http://backend:8090',
      imageID: 'sha256:guard',
      canaryUser: 'e2e-canary-guard',
      adminUsername: 'admin',
      adminPassword: 'guard-only-fixture-password',
      createdAt: new Date().toISOString(),
      expiresAtEpoch: Math.floor(Date.now() / 1000) + 3600,
      proved: 'guard',
      ...overrides,
    })
  );
  return file;
}

test.afterEach(() => {
  for (const [key, value] of Object.entries(REAL_ENV)) {
    if (value === undefined) delete process.env[key];
    else process.env[key] = value;
  }
});

test('未声明一次性栈时拒绝运行', () => {
  delete process.env.ITSM_E2E_ISOLATED_STACK;
  expect(() => isolatedBaseURL()).toThrowError(/ITSM_E2E_ISOLATED_STACK=1/);
});

test('地址缺省时拒绝运行，不回落到宿主 :3000', () => {
  process.env.ITSM_E2E_ISOLATED_STACK = '1';
  delete process.env.PLAYWRIGHT_BASE_URL;
  expect(() => isolatedBaseURL()).toThrowError(/缺少 PLAYWRIGHT_BASE_URL/);
});

test('非白名单目标（含生产直连端口）拒绝运行', () => {
  process.env.ITSM_E2E_ISOLATED_STACK = '1';
  process.env.PLAYWRIGHT_BASE_URL = 'http://127.0.0.1:8090';
  expect(() => isolatedBaseURL()).toThrowError(/不在一次性栈白名单里/);
});

test('没有 proof 文件时拒绝运行', () => {
  process.env.ITSM_E2E_ISOLATED_STACK = '1';
  process.env.PLAYWRIGHT_BASE_URL = TARGETS[0];
  delete process.env.ITSM_E2E_STACK_PROOF;
  expect(() => isolatedBaseURL()).toThrowError(/缺少 ITSM_E2E_STACK_PROOF/);
});

test('proof 过期时拒绝运行', () => {
  const baseURL = TARGETS[1];
  process.env.ITSM_E2E_ISOLATED_STACK = '1';
  process.env.PLAYWRIGHT_BASE_URL = baseURL;
  process.env.ITSM_E2E_STACK_PROOF = proofFile({ expiresAtEpoch: 1_000_000_000 }, baseURL);
  expect(() => isolatedBaseURL()).toThrowError(/proof 已过期/);
  rmSync(join(process.env.ITSM_E2E_STACK_PROOF, '..'), { recursive: true, force: true });
});

test('地址与 proof 记录不一致时拒绝运行（localhost 不等于 127.0.0.1）', () => {
  const proofBaseURL = TARGETS[2];
  process.env.ITSM_E2E_ISOLATED_STACK = '1';
  process.env.PLAYWRIGHT_BASE_URL = TARGETS[3];
  process.env.ITSM_E2E_STACK_PROOF = proofFile({}, proofBaseURL);
  expect(() => isolatedBaseURL()).toThrowError(/与 proof 记录的/);
  rmSync(join(process.env.ITSM_E2E_STACK_PROOF, '..'), { recursive: true, force: true });
});

test('proof 里的代理 upstream 指向环回时拒绝运行', () => {
  const baseURL = TARGETS[0];
  process.env.ITSM_E2E_ISOLATED_STACK = '1';
  process.env.PLAYWRIGHT_BASE_URL = baseURL;
  process.env.ITSM_E2E_STACK_PROOF = proofFile({ proxyUpstream: 'http://localhost:8090' }, baseURL);
  expect(() => isolatedBaseURL()).toThrowError(/指向环回地址/);
  rmSync(join(process.env.ITSM_E2E_STACK_PROOF, '..'), { recursive: true, force: true });
});

test('proof 一致时放行并返回 proof 记录的地址', () => {
  const baseURL = TARGETS[1];
  process.env.ITSM_E2E_ISOLATED_STACK = '1';
  process.env.PLAYWRIGHT_BASE_URL = baseURL;
  process.env.ITSM_E2E_STACK_PROOF = proofFile({}, baseURL);
  expect(isolatedBaseURL()).toBe(baseURL);
  rmSync(join(process.env.ITSM_E2E_STACK_PROOF, '..'), { recursive: true, force: true });
});
