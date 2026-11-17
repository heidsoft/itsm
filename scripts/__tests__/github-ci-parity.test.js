/**
 * GitHub Actions 远端红灯的回归锁（2026-11-14）。
 *
 * 证据来源：heidsoft/itsm 远端 Actions runs
 *   #43158704277 backend-ci / Lint            → gofumpt 格式欠账
 *   #43158704283 frontend-ci / Frontend       → 2 个过期断言
 *   #43158704335 api-contract-check           → Swagger 生成产物未提交
 *   #43401901147 Update Star History          → GH006 直推受保护 main 被拒
 *
 * 每个断言都配一个「改回旧实现必红」的反向验证，防止契约测试退化成
 * 只匹配注释关键字的同义反复。
 */
'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const ROOT = path.resolve(__dirname, '../..');
const read = (rel) => fs.readFileSync(path.join(ROOT, rel), 'utf8');

// GitHub Actions 的 run 块里常有大段解释性注释；断言只看代码行。
function codeOnly(text) {
  return text
    .split('\n')
    .filter((line) => !/^\s*#/.test(line))
    .join('\n');
}

const starHistory = read('.github/workflows/star-history.yml');
const backendCi = read('.github/workflows/backend-ci.yml');
const apiContract = read('.github/workflows/api-contract-check.yml');
const makefile = read('Makefile');
const parityScript = read('scripts/check-ci-parity.sh');
const swaggerJson = read('itsm-backend/docs/swagger.json');
const ticketApiTest = read('itsm-frontend/src/lib/api/__tests__/ticket-api.test.ts');
const instanceDetailTest = read(
  'itsm-frontend/src/components/workflow/__tests__/WorkflowInstanceDetail.test.tsx'
);

function assertNoProtectedBranchPush(yaml) {
  const code = codeOnly(yaml);
  // 旧实现的最后一行是裸 `git push`（默认推当前 main），被分支保护拒绝。
  assert.doesNotMatch(
    code,
    /^\s*git push\s*$/m,
    'star-history must not push directly to the protected default branch'
  );
  assert.match(code, /git push --force-with-lease origin "\$BOT_BRANCH"/);
  assert.match(code, /gh pr create/);
  assert.match(code, /pull-requests: write/);
  assert.match(code, /gh pr list --head "\$BOT_BRANCH" --state open/, 'open PR must be reused');
}

test('star-history publishes via a bot branch PR, never a direct push to main', () => {
  assertNoProtectedBranchPush(starHistory);
});

test('reverse: restoring the old bare git push makes the contract fail', () => {
  const broken = starHistory.replace(
    'git push --force-with-lease origin "$BOT_BRANCH"',
    'git push'
  );
  assert.notEqual(broken, starHistory);
  assert.throws(() => assertNoProtectedBranchPush(broken));
});

function assertGofumptPin(yaml, script) {
  const pinned = codeOnly(yaml).match(/go install mvdan\.cc\/gofumpt@(v[\d.]+)/);
  const local = codeOnly(script).match(/GOFUMPT_VERSION="(v[\d.]+)"/);
  assert.ok(pinned, 'backend-ci.yml must pin gofumpt');
  assert.ok(local, 'check-ci-parity.sh must declare the same pin');
  assert.equal(pinned[1], local[1], 'CI and local parity check must use the same gofumpt');
}

test('local parity check uses the exact gofumpt pin from backend-ci', () => {
  assertGofumptPin(backendCi, parityScript);
  assert.match(codeOnly(makefile), /gofumpt@v0\.7\.0/, 'make fmt-backend must use the CI pin');
});

test('reverse: drifting the CI pin away from the local pin fails', () => {
  assert.throws(() => assertGofumptPin(backendCi.replace('gofumpt@v0.7.0', 'gofumpt@v0.8.0'), parityScript));
});

// Makefile 用 $(GO) run，CI 用 go run；锁的是同一组 swag 参数，避免两边漂移。
const SWAG_ARGS =
  'github.com/swaggo/swag/cmd/swag init -d . -g main.go -o docs --parseDependency --parseInternal';

function assertSwaggerParity(yaml, mk) {
  assert.ok(
    codeOnly(yaml).includes(SWAG_ARGS),
    'api-contract-check must regenerate docs with the same swag arguments as make swagger-gen'
  );
  assert.ok(codeOnly(mk).includes(SWAG_ARGS), 'Makefile swagger-gen must keep the same arguments');
  // 服务层迁移后 dashboard.* 类型在 internal/services 中定义；改了它们也要触发
  // swagger freshness，否则「改了被测对象不触发门禁」。
  const pushSection = codeOnly(yaml).split('workflow_dispatch:')[0];
  const occurrences = (pushSection.match(/itsm-backend\/internal\/services\/\*\*/g) || []).length;
  assert.equal(occurrences, 2, 'both push and pull_request trigger sections must list internal/services/**');
}

test('swagger freshness uses one command and triggers on the migrated service layer', () => {
  assertSwaggerParity(apiContract, makefile);
});

test('reverse: dropping a trigger path makes the parity contract fail', () => {
  const broken = apiContract.replace("      - 'itsm-backend/internal/services/**'\n", '');
  assert.notEqual(broken, apiContract);
  assert.throws(() => assertSwaggerParity(broken, makefile));
});

function assertSwaggerRefreshed(json) {
  const doc = JSON.parse(json);
  const defs = Object.keys(doc.definitions || {});
  assert.ok(
    doc.paths?.['/api/v1/cmdb/export/{task_id}/download'],
    'missing CMDB export download path: generated docs are stale'
  );
  assert.ok(
    defs.some((d) => d.startsWith('dashboard.')),
    'missing dashboard.* definitions after the service-layer migration'
  );
  assert.equal(
    defs.filter((d) => d.startsWith('handlers.')).length,
    0,
    'stale handlers.* definitions survived regeneration'
  );
}

test('committed OpenAPI docs contain the newly annotated endpoints and types', () => {
  assertSwaggerRefreshed(swaggerJson);
});

test('reverse: stale docs without the new path fail the freshness lock', () => {
  const doc = JSON.parse(swaggerJson);
  delete doc.paths['/api/v1/cmdb/export/{task_id}/download'];
  assert.throws(() => assertSwaggerRefreshed(JSON.stringify(doc)));
});

function assertJestAssertions(ticket, detail) {
  // 两个用例曾让 frontend-ci 2 failed / 3092 total：断言停留在旧实现。
  assert.match(ticket, /toHaveBeenCalledWith\('\/api\/v1\/tickets\/stats', undefined\)/);
  assert.match(ticket, /toHaveBeenCalledWith\('\/api\/v1\/tickets\/stats', params\)/);
  assert.match(detail, /getByText\('流程已启动'\)/);
  assert.match(detail, /getByText\('CUSTOM ACTION'\)/);
  assert.doesNotMatch(detail, /getByText\('PROCESS STARTED'\)/);
}

test('the two formerly red Jest cases assert the current product contract', () => {
  assertJestAssertions(ticketApiTest, instanceDetailTest);
});

test('reverse: restoring either stale assertion makes the lock fail', () => {
  assert.throws(() =>
    assertJestAssertions(
      ticketApiTest.replace(
        "toHaveBeenCalledWith('/api/v1/tickets/stats', undefined)",
        "toHaveBeenCalledWith('/api/v1/tickets/stats')"
      ),
      instanceDetailTest
    )
  );
  assert.throws(() =>
    assertJestAssertions(
      ticketApiTest,
      instanceDetailTest
        .replace("getByText('流程已启动')", "getByText('PROCESS STARTED')")
        .replace("getByText('CUSTOM ACTION')", "getByText('流程已启动')")
    )
  );
});

test('make exposes local CI parity entry points', () => {
  const phony = makefile.split('\n').find((l) => l.startsWith('.PHONY:'));
  for (const target of ['fmt-backend', 'ci-parity', 'ci-parity-strict']) {
    assert.ok(phony.includes(target), `.PHONY missing ${target}`);
    assert.match(makefile, new RegExp(`^${target}:`, 'm'), `target ${target} missing`);
  }
  assert.match(makefile, /^ci-parity:.*\n\t@bash scripts\/check-ci-parity\.sh$/m);
});
