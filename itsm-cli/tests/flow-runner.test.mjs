/**
 * Business-flow runner contract tests.
 *
 * The runner is the part an agent trusts to answer "did the product flow work?".
 * Its failure semantics matter more than its happy path: a negative test that
 * sees a 500 instead of a 404 must report `failed` (product defect), while a
 * governance restriction must report `blocked` (environment), because those two
 * verdicts lead a human to completely different fixes.
 *
 * These tests drive the runner directly with synthetic steps, so they assert
 * orchestration semantics without needing a backend.
 *
 * Run: npm test   (builds first)
 */
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { runScenario, makeRunTag } from '../dist/agent/flow/runner.js';
import { CliError } from '../dist/lib/errors.js';

const quiet = { log: () => {} };

function scenario(overrides) {
  return {
    name: 'synthetic-flow',
    summary: 'synthetic',
    domain: 'test',
    tags: ['test'],
    proves: [],
    steps: [],
    ...overrides,
  };
}

function failure(kind, message, extra = {}) {
  return new CliError({ kind, message, retryable: false, ...extra });
}

test('happy path captures context, runs teardown in reverse, and reports passed', async () => {
  const order = [];
  const report = await runScenario(
    scenario({
      steps: [
        {
          name: 'create',
          endpoint: 'POST /api/v1/things',
          run: async () => ({ id: 41, status: 'new' }),
          expect: [{ path: 'status', eq: 'new' }],
          capture: (result) => ({ thingId: result.id }),
        },
        {
          name: 'advance',
          endpoint: 'POST /api/v1/things/:id/advance',
          run: async (ctx) => {
            assert.equal(ctx.vars.thingId, 41, 'capture must be visible to later steps');
            return { status: 'open' };
          },
          expect: [{ path: 'status', eq: 'open' }],
        },
      ],
      teardown: [
        {
          name: 'delete thing',
          run: async () => {
            order.push('delete');
          },
        },
        {
          name: 'delete audit',
          run: async () => {
            order.push('audit');
          },
        },
      ],
    }),
    quiet
  );

  assert.equal(report.verdict, 'passed');
  assert.equal(report.steps.length, 2);
  assert.deepEqual(report.context, { thingId: 41 });
  assert.deepEqual(order, ['audit', 'delete'], 'teardown must run in reverse creation order');
  assert.equal(report.teardown.length, 2);
  assert.match(report.runTag, /^agent-\d{14}-[a-z0-9]{4}$/);
  assert.equal(report.failure, undefined);
  assert.match(report.next, /passed/);
});

test('an assertion failure stops later steps but still tears down', async () => {
  const ran = [];
  const report = await runScenario(
    scenario({
      steps: [
        {
          name: 'create',
          run: async () => ({ id: 7, status: 'new' }),
          capture: (result) => ({ thingId: result.id }),
        },
        {
          name: 'resolve',
          run: async () => {
            ran.push('resolve');
            return { status: 'resolved', resolution: undefined };
          },
          expect: [
            { path: 'status', eq: 'resolved' },
            { path: 'resolution', contains: 'fixed' },
          ],
        },
        {
          name: 'close',
          run: async () => {
            ran.push('close');
            return { status: 'closed' };
          },
        },
       ],
      teardown: [
        {
          name: 'cleanup',
          run: async () => {
            ran.push('cleanup');
          },
        },
      ],
    }),
    quiet
  );

  assert.equal(report.verdict, 'failed');
  assert.deepEqual(ran, ['resolve', 'cleanup'], 'steps after the failure must not run; teardown must');
  assert.equal(report.failure.step, 'resolve');
  assert.match(report.failure.message, /resolution contains/);
  assert.match(report.next, /Inspect step 'resolve'/);
  assert.equal(report.context.thingId, 7, 'captured ids survive for cleanup and reporting');
});

test('expectError passes only for the declared rejection', async () => {
  const report = await runScenario(
    scenario({
      steps: [
        {
          name: 're-resolve closed thing',
          run: async () => {
            throw failure('conflict', '当前工单状态不允许此操作', { httpStatus: 409 });
          },
          expectError: { kind: 'conflict', httpStatus: 409 },
        },
      ],
    }),
    quiet
  );

  assert.equal(report.verdict, 'passed');
  assert.equal(report.steps[0].ok, true);
});

test('a negative test that receives a 500 is failed, not blocked', async () => {
  const report = await runScenario(
    scenario({
      steps: [
        {
          name: 'relationship to a missing CI',
          run: async () => {
            throw failure('unavailable', '创建CI关系失败', { httpStatus: 500 });
          },
          expectError: { kind: 'not_found', httpStatus: 404 },
        },
      ],
    }),
    quiet
  );

  assert.equal(report.verdict, 'failed', 'a 5xx where a 404 was promised is a product defect');
  assert.equal(report.steps[0].error.kind, 'assertion_failed');
  assert.equal(report.steps[0].error.httpStatus, 500, 'the observed status stays in evidence');
  assert.match(report.failure.message, /did not fail as expected/);
  assert.match(report.failure.message, /unavailable\/500/);
});

test('blockedWhen downgrades a declared governance restriction to blocked', async () => {
  const report = await runScenario(
    scenario({
      steps: [
        {
          name: 'approve own article',
          run: async () => {
            throw failure('conflict', '不能评审自己撰写的文章', { httpStatus: 409 });
          },
          blockedWhen: { messageIncludes: '自己撰写', reason: 'needs a second reviewer identity' },
        },
      ],
    }),
    quiet
  );

  assert.equal(report.verdict, 'blocked');
  assert.equal(report.steps[0].error.kind, 'unavailable');
  assert.match(report.failure.message, /needs a second reviewer identity/);
  assert.match(report.next, /Environment is not ready/);
});

test('--keep skips teardown and warns how to find the artifacts', async () => {
  let tornDown = false;
  const report = await runScenario(
    scenario({
      steps: [{ name: 'create', run: async () => ({ id: 1 }) }],
      teardown: [
        {
          name: 'delete',
          run: async () => {
            tornDown = true;
          },
        },
      ],
    }),
    { ...quiet, keep: true, tag: 'agent-20260101000000-keep' }
  );

  assert.equal(report.verdict, 'passed');
  assert.equal(tornDown, false);
  assert.ok(report.warnings.some((w) => w.includes('agent-20260101000000-keep')));
});

test('a failing teardown is a warning and never masks the verdict', async () => {
  const report = await runScenario(
    scenario({
      steps: [{ name: 'create', run: async () => ({ id: 1 }) }],
      teardown: [
        {
          name: 'delete',
          run: async () => {
            throw failure('forbidden', 'denied', { httpStatus: 403 });
          },
        },
      ],
    }),
    quiet
  );

  assert.equal(report.verdict, 'passed');
  assert.equal(report.warnings.length, 1);
  assert.match(report.warnings[0], /teardown step 'delete' failed/);
});

test('makeRunTag is sortable and unique enough for artifact tracing', () => {
  const a = makeRunTag();
  const b = makeRunTag();
  assert.match(a, /^agent-\d{14}-[a-z0-9]{4}$/);
  assert.notEqual(a, b);
});
