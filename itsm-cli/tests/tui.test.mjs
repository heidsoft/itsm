/**
 * TUI regression suite.
 *
 * The interactive surface used to be a hand-written switch with no test at
 * all: a route could disappear, a domain could be missing, or a high-risk
 * command could execute without its confirmation gate and nothing would fail.
 *
 * These tests render the real compiled components through
 * `ink-testing-library` and drive the real `GenericCommand` → agent registry
 * path against a stubbed `fetch`. That keeps the assertions on the same code
 * `bin/itsm.js` runs, not on a copy.
 *
 * Run: npm test   (builds first)
 */
import { test, after, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import React from 'react';
import { render, cleanup } from 'ink-testing-library';

const ROOT = path.resolve(import.meta.dirname, '..');
const HOME = mkdtempSync(path.join(tmpdir(), 'itsm-tui-test-'));

// Credentials must exist before any component renders: `App` checks
// `isLoggedIn()` synchronously and the API client reads the store per request.
const credentialsFile = path.join(HOME, 'credentials.json');
writeFileSync(
  credentialsFile,
  JSON.stringify({
    token: 'tui-test-token',
    expiresAt: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
    user: { id: 1, username: 'tui-tester', name: 'TUI Tester', role: 'admin' },
    version: 2,
  }),
  { mode: 0o600 },
);

process.env.ITSM_CREDENTIALS_FILE = credentialsFile;
process.env.ITSM_BASE_URL = 'http://127.0.0.1:45999';
process.env.ITSM_ALLOW_INSECURE_HTTP = '1';
process.env.NO_COLOR = '1';

/** Every request the components attempted, for "zero requests" assertions. */
let requests = [];

function jsonResponse(data, status = 200) {
  return new Response(JSON.stringify({ code: 0, message: 'success', data }), {
    status,
    headers: { 'content-type': 'application/json' },
  });
}

function route(url, method) {
  const parsed = new URL(url);
  const p = parsed.pathname.replace(/^\/api\/v1/, '');

  if (method === 'DELETE' && p === '/tickets/42') return jsonResponse(null);
  if (p === '/problems/stats') {
    return jsonResponse({ total: 3, open: 1, inProgress: 1, resolved: 1, closed: 0 });
  }
  if (p === '/problems') {
    return jsonResponse({
      items: [
        {
          id: 1,
          problemNumber: 'PRB-1',
          title: 'Printer jam',
          status: 'open',
          priority: 'high',
          assigneeName: 'Ada',
        },
      ],
      total: 1,
      page: 1,
      pageSize: 20,
      totalPages: 1,
    });
  }
  if (p.startsWith('/problems/')) {
    return jsonResponse({
      id: 1,
      problemNumber: 'PRB-1',
      title: 'Printer jam',
      status: 'open',
      priority: 'high',
      rootCause: 'Feeder roller worn',
    });
  }
  return jsonResponse({});
}

const originalFetch = globalThis.fetch;
globalThis.fetch = async (url, init = {}) => {
  const method = (init.method ?? 'GET').toUpperCase();
  requests.push({ method, url: String(url) });
  return route(String(url), method);
};

const stripAnsi = (value) => value.replace(/\u001b\[[0-9;]*m/g, '');

/** Poll `lastFrame()` until `predicate` passes; Ink paints on a timer, not synchronously. */
async function waitForFrame(getFrame, predicate, label, timeoutMs = 3000) {
  const deadline = Date.now() + timeoutMs;
  let frame = '';
  while (Date.now() < deadline) {
    frame = stripAnsi(getFrame() ?? '');
    if (predicate(frame)) return frame;
    await new Promise((resolve) => setTimeout(resolve, 10));
  }
  throw new Error(`Timed out waiting for ${label}. Last frame:\n${frame}`);
}

beforeEach(() => {
  requests = [];
});

afterEach(() => {
  cleanup();
});

after(() => {
  globalThis.fetch = originalFetch;
  cleanup();
});

test('curated routes cover problem/release/request list and detail', async () => {
  const { resolveTuiRoute } = await import('../dist/tui/routes.js');

  assert.equal(resolveTuiRoute(['problem', 'list'])?.id, 'problem.list');
  assert.equal(resolveTuiRoute(['problem'])?.id, 'problem.list');
  assert.equal(resolveTuiRoute(['problem', '7'])?.id, 'problem.get');
  assert.equal(resolveTuiRoute(['release', 'list'])?.id, 'release.list');
  assert.equal(resolveTuiRoute(['release', '12'])?.id, 'release.get');
  assert.equal(resolveTuiRoute(['request', 'list'])?.id, 'request.list');
  assert.equal(resolveTuiRoute(['request', '9'])?.id, 'request.get');

  // Verbs without a bespoke screen must fall through to GenericCommand, not
  // render an unknown-command error.
  assert.equal(resolveTuiRoute(['problem', 'stats']), undefined);
  assert.equal(resolveTuiRoute(['report']), undefined);
});

test('help screen lists the new domains', async () => {
  const { HelpScreen } = await import('../dist/tui/help.js');
  const { lastFrame } = render(React.createElement(HelpScreen));

  const frame = await waitForFrame(
    lastFrame,
    (value) => ['问题', '发布', '服务请求'].every((group) => value.includes(group)),
    'new help groups',
  );

  assert.match(frame, /itsm problem list/);
  assert.match(frame, /itsm release list/);
  assert.match(frame, /itsm request list/);
  assert.match(frame, /通用兜底/);
});

test('problem list renders rows and total from the standard envelope', async () => {
  const { ProblemListCommand } = await import('../dist/commands/problem.js');
  const { lastFrame } = render(React.createElement(ProblemListCommand, { status: 'open' }));

  const frame = await waitForFrame(lastFrame, (value) => value.includes('PRB-1'), 'problem row');

  assert.match(frame, /Problems \(1 total\)/);
  assert.match(frame, /Printer jam/);
  assert.match(frame, /Ada/);
  assert.equal(requests.length, 1);
  assert.match(requests[0].url, /\/api\/v1\/problems\?/);
  assert.match(requests[0].url, /status=open/);
});

test('GenericCommand executes a command that has no bespoke screen', async () => {
  const { GenericCommand } = await import('../dist/tui/GenericCommand.js');
  const { lastFrame } = render(React.createElement(GenericCommand, { argv: ['problem', 'stats'] }));

  const frame = await waitForFrame(lastFrame, (value) => value.includes('problem.stats'), 'generic result');

  assert.match(frame, /total/);
  assert.match(frame, /inProgress/);
  assert.equal(requests.length, 1);
  assert.match(requests[0].url, /\/api\/v1\/problems\/stats$/);
});

test('high-risk generic command stops at the confirmation gate with zero requests', async () => {
  const { GenericCommand } = await import('../dist/tui/GenericCommand.js');
  const { lastFrame } = render(React.createElement(GenericCommand, { argv: ['ticket', 'delete', '42'] }));

  const frame = await waitForFrame(lastFrame, (value) => value.includes('需要确认'), 'confirmation gate');

  assert.match(frame, /ticket\.delete/);
  assert.match(frame, /--yes/);
  assert.equal(requests.length, 0, 'the gate must not send the request before consent');
});

test('--yes executes the high-risk command exactly once', async () => {
  const { GenericCommand } = await import('../dist/tui/GenericCommand.js');
  const { lastFrame } = render(
    React.createElement(GenericCommand, { argv: ['ticket', 'delete', '42', '--yes'] }),
  );

  await waitForFrame(lastFrame, (value) => value.includes('ticket.delete'), 'generic completion');

  assert.equal(requests.length, 1);
  assert.equal(requests[0].method, 'DELETE');
  assert.match(requests[0].url, /\/api\/v1\/tickets\/42$/);
});

test('unknown command reports usage instead of silently succeeding', async () => {
  const { GenericCommand } = await import('../dist/tui/GenericCommand.js');
  const { lastFrame } = render(React.createElement(GenericCommand, { argv: ['bogus', 'thing'] }));

  const frame = await waitForFrame(lastFrame, (value) => value.includes('Unknown command'), 'unknown command');

  assert.match(frame, /bogus thing/);
  assert.match(frame, /itsm --json commands/);
  assert.equal(requests.length, 0);
});

test('bare domain help is derived from the live registry', async () => {
  const { GenericCommand } = await import('../dist/tui/GenericCommand.js');
  const { lastFrame } = render(React.createElement(GenericCommand, { argv: ['report'] }));

  const frame = await waitForFrame(lastFrame, (value) => value.includes('report ·'), 'domain help');

  assert.match(frame, /条命令/);
  assert.match(frame, /直接执行任意一条/);
  assert.equal(requests.length, 0);
});
