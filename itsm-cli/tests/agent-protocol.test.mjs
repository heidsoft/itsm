/**
 * Agent protocol regression suite.
 *
 * The headless contract is what an LLM depends on: exactly one JSON envelope,
 * semantic exit codes, no destructive write without consent, bounded output.
 * Every assertion here runs the real `bin/itsm.js` against a stub backend — the
 * same production entrypoint an agent calls, not an in-process helper.
 *
 * Run: npm test   (builds first)
 */
import { test, before, after } from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';

const ROOT = path.resolve(import.meta.dirname, '..');
const BIN = path.join(ROOT, 'bin/itsm.js');
const HOME = mkdtempSync(path.join(tmpdir(), 'itsm-cli-test-'));
const TOKEN = 'test-token';

let server;
let baseUrl;
/** path → number of requests, for retry/dry-run/high-risk assertions. */
const hits = new Map();
const boomOnce = new Set(['/boom/503-once']);

const ok = (res, data) => json(res, 200, { code: 0, message: 'success', data });
function json(res, status, payload) {
  res.writeHead(status, { 'content-type': 'application/json' });
  res.end(JSON.stringify(payload));
}

function route(req, res) {
  const url = new URL(req.url, baseUrl);
  const p = url.pathname.replace(/^\/api\/v1/, '');
  hits.set(p, (hits.get(p) ?? 0) + 1);

  // Public routes
  if (p === '/health') return json(res, 200, { status: 'ok', timestamp: '2026-01-01T00:00:00Z' });
  if (p === '/version') return ok(res, { version: 'v9.9.9', build: 'test' });

  if (req.headers.authorization !== `Bearer ${TOKEN}`) {
    return json(res, 401, { code: 2001, message: '缺少认证token' });
  }

  if (p === '/auth/me') return ok(res, { id: 1, username: 'agent', role: 'admin', tenantId: 1 });

  if (p === '/tickets' && req.method === 'GET') {
    return ok(res, {
      items: [
        { id: 1, ticketNumber: 'TKT-1', title: 'a', status: 'open', priority: 'high' },
        { id: 2, ticketNumber: 'TKT-2', title: 'b', status: 'new', priority: 'low' },
        { id: 3, ticketNumber: 'TKT-3', title: 'c', status: 'closed', priority: 'medium' },
      ],
      total: 3,
      page: 1,
      pageSize: 20,
      totalPages: 1,
    });
  }
  if (/^\/tickets\/\d+$/.test(p) && req.method === 'DELETE') return ok(res, null);
  // Legacy backend shape: business code carried in an HTTP 200 body.
  if (p === '/tickets/999999' && req.method === 'GET') return json(res, 200, { code: 4004, message: 'Not Found' });
  if (p === '/tickets/999998' && req.method === 'GET') return json(res, 200, { code: 4290, message: '请求过于频繁，请稍后再试' });
  if (p === '/incidents' || p === '/problems' || p === '/changes' || p === '/cmdb/cis' ||
      p === '/knowledge/articles' || p === '/notifications') {
    if (req.method === 'GET') return ok(res, { items: [], total: 0, page: 1, pageSize: 1, totalPages: 0 });
  }

  if (p === '/boom/forbidden') return json(res, 403, { code: 2004, message: 'no permission' });
  if (p === '/boom/notfound') return json(res, 404, { code: 3001, message: 'not found' });
  if (p === '/boom/conflict') return json(res, 409, { code: 3002, message: 'version conflict' });
  if (p === '/boom/validation') return json(res, 400, { code: 1001, message: 'title is required' });
  if (p === '/boom/503-once') {
    if (boomOnce.has(p)) {
      boomOnce.delete(p);
      return json(res, 503, { code: 5001, message: 'temporarily unavailable' });
    }
    return ok(res, { recovered: true });
  }
  if (p === '/boom/slow') {
    return setTimeout(() => ok(res, { slow: true }), 400);
  }

  // Read-only command whose verb names a mutation (risk-derivation regression lock).
  if (p === '/changes/1/rollback-steps' && req.method === 'GET') return ok(res, { steps: [] });
  if (p === '/changes/1/rollback' && req.method === 'POST') return ok(res, { status: 'rolled_back' });
  if (p === '/changes/calendar' && req.method === 'GET') return ok(res, { items: [] });

  return json(res, 404, { code: 3001, message: `stub has no route for ${req.method} ${p}` });
}

before(async () => {
  server = createServer(route);
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  baseUrl = `http://127.0.0.1:${server.address().port}`;
});

after(() => server?.close());

/**
 * Run the real CLI as an agent would.
 * Async on purpose: the stub backend lives in this process, so a blocking
 * spawnSync would deadlock the server that has to answer the child.
 */
function run(args, extraEnv = {}) {
  return new Promise((resolve) => {
    const child = spawn(process.execPath, [BIN, ...args], {
      env: {
        ...process.env,
        HOME,
        ITSM_BASE_URL: baseUrl,
        ITSM_TOKEN: TOKEN,
        ITSM_USERNAME: '',
        ITSM_PASSWORD: '',
        ...extraEnv,
      },
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (chunk) => { stdout += chunk; });
    child.stderr.on('data', (chunk) => { stderr += chunk; });
    const timer = setTimeout(() => child.kill('SIGKILL'), 20_000);
    child.on('close', (code) => {
      clearTimeout(timer);
      resolve({ code, stdout, stderr });
    });
  });
}

function envelope(result) {
  const lines = result.stdout.trimEnd().split('\n').filter(Boolean);
  assert.equal(lines.length, 1, `expected exactly one stdout line, got:\n${result.stdout}`);
  return JSON.parse(lines[0]);
}

const api = (...args) => run(['--json', 'infra', 'api', ...args]);

// ---------------------------------------------------------------------------
// Output contract
// ---------------------------------------------------------------------------

test('success is exactly one JSON envelope with the backend data unwrapped', async () => {
  const r = await run(['--json', 'infra', 'version']);
  assert.equal(r.code, 0);
  const env = envelope(r);
  assert.equal(env.ok, true);
  assert.equal(env.data.version, 'v9.9.9');
  assert.equal(env.error, undefined);
});

test('--limit and --fields shrink list payloads and report meta', async () => {
  const r = await run(['--json', 'ticket', 'list', '--limit', '2', '--fields', 'id,status']);
  assert.equal(r.code, 0);
  const env = envelope(r);
  assert.equal(env.data.items.length, 2);
  assert.deepEqual(Object.keys(env.data.items[0]).sort(), ['id', 'status']);
  assert.equal(env.meta.returned, 2);
  assert.equal(env.meta.total, 3);
  assert.equal(env.meta.truncated, true);
});

test('shaping flags on a non-list payload are reported, not silently ignored', async () => {
  const r = await api('GET', '/auth/me', '--fields', 'id');
  assert.equal(r.code, 0);
  const env = envelope(r);
  assert.equal(env.data.username, 'agent');
  assert.match(env.meta.shapingSkipped, /--fields\/--limit do not apply/);
});

test('--format ndjson streams one object per line and ends with _meta', async () => {
  const r = await run(['--json', 'ticket', 'list', '--limit', '3', '--fields', 'id', '--format', 'ndjson']);
  assert.equal(r.code, 0);
  const lines = r.stdout.trimEnd().split('\n').map((l) => JSON.parse(l));
  assert.equal(lines.length, 4);
  assert.deepEqual(lines.slice(0, 3).map((l) => l.id), [1, 2, 3]);
  assert.equal(lines[3]._meta.returned, 3);
  assert.equal(lines[3]._meta.total, 3);
});

// ---------------------------------------------------------------------------
// Error taxonomy → exit codes
// ---------------------------------------------------------------------------

test('unknown command exits 2 with a usage envelope', async () => {
  const r = await run(['--json', 'ticket', 'gett', '1']);
  assert.equal(r.code, 2);
  const env = envelope(r);
  assert.equal(env.ok, false);
  assert.equal(env.error.kind, 'usage');
  assert.match(env.error.message, /Unknown command/);
});

test('401 exits 3, 403 exits 4, 404/409/400 exit 5 with distinct kinds', async () => {
  const auth = envelope(await run(['--json', 'ticket', 'list'], { ITSM_TOKEN: 'wrong-token' }));
  assert.equal(auth.error.kind, 'not_authenticated');

  const cases = [
    ['/boom/forbidden', 4, 'forbidden'],
    ['/boom/notfound', 5, 'not_found'],
    ['/boom/conflict', 5, 'conflict'],
    ['/boom/validation', 5, 'validation'],
  ];
  for (const [route, expectedCode, expectedKind] of cases) {
    const r = await api('GET', route);
    assert.equal(r.code, expectedCode, `${route} should exit ${expectedCode}, got ${r.code}`);
    const env = envelope(r);
    assert.equal(env.error.kind, expectedKind);
    assert.equal(env.error.retryable, false);
    assert.ok(env.error.hint, `${route} must carry an actionable hint`);
  }
});

test('a business code in a 200 body is classified by the code, not the status', async () => {
  const notFound = await api('GET', '/tickets/999999');
  assert.equal(notFound.code, 5);
  const nf = envelope(notFound);
  assert.equal(nf.error.kind, 'not_found');
  assert.equal(nf.error.businessCode, 4004);
  assert.equal(nf.error.retryable, false);

  const limited = await api('GET', '/tickets/999998');
  assert.equal(limited.code, 5);
  const rl = envelope(limited);
  assert.equal(rl.error.kind, 'unavailable');
  assert.equal(rl.error.businessCode, 4290);
  assert.equal(rl.error.retryable, true, 'a rate limit is worth retrying with backoff');
});

test('unreachable backend exits 1 and is marked retryable', async () => {
  const r = await run(['--json', 'infra', 'health'], { ITSM_BASE_URL: 'http://127.0.0.1:1' });
  assert.equal(r.code, 1);
  const env = envelope(r);
  assert.equal(env.error.kind, 'network');
  assert.equal(env.error.retryable, true);
});

test('--timeout bounds a slow request and explains the fix', async () => {
  const r = await api('GET', '/boom/slow', '--timeout', '50');
  assert.equal(r.code, 1);
  const env = envelope(r);
  assert.equal(env.error.kind, 'network');
  assert.match(env.error.message, /timed out after 50ms/);
  assert.match(env.error.hint, /--timeout/);
});

test('a single 503 is retried transparently for an idempotent GET', async () => {
  const before = hits.get('/boom/503-once') ?? 0;
  const r = await api('GET', '/boom/503-once');
  assert.equal(r.code, 0, r.stdout);
  const env = envelope(r);
  assert.equal(env.data.recovered, true);
  assert.equal((hits.get('/boom/503-once') ?? 0) - before, 2, 'expected exactly one retry');
});

// ---------------------------------------------------------------------------
// Confirmation gate (exit 10)
// ---------------------------------------------------------------------------

test('high-risk write without --yes exits 10, sends nothing, and explains the risk', async () => {
  const before = hits.get('/tickets/1') ?? 0;
  const r = await run(['--json', 'ticket', 'delete', '1']);
  assert.equal(r.code, 10);
  const env = envelope(r);
  assert.equal(env.error.kind, 'confirmation_required');
  assert.equal(env.error.risk.command, 'ticket.delete');
  assert.equal(env.error.risk.irreversible, true);
  assert.match(env.error.hint, /Never retry automatically/);
  assert.equal(hits.get('/tickets/1') ?? 0, before, 'no request may be sent before consent');
});

test('high-risk write with --yes executes', async () => {
  const r = await run(['--json', 'ticket', 'delete', '1', '--yes']);
  assert.equal(r.code, 0, r.stdout);
  assert.equal(envelope(r).ok, true);
});

test('--dry-run previews a high-risk command without triggering the gate or the request', async () => {
  const before = hits.get('/tickets/1') ?? 0;
  const r = await run(['--json', 'ticket', 'delete', '1', '--dry-run']);
  assert.equal(r.code, 0);
  const env = envelope(r);
  assert.equal(env.data.dryRun, true);
  assert.equal(env.data.requiresConfirmation, true);
  assert.equal(hits.get('/tickets/1') ?? 0, before);
});

test('--help short-circuits the risk gate', async () => {
  const r = await run(['--json', 'ticket', 'delete', '--help']);
  assert.equal(r.code, 0);
  const env = envelope(r);
  assert.equal(env.data.command, 'ticket.delete');
  assert.equal(env.data.requiresConfirmation, true);
  assert.ok(env.data.input.params.some((p) => p.name === 'id' && p.required && p.kind === 'positional'));
});

test('a read-only command is never gated, even when its verb names a mutation', async () => {
  // Regression: risk was derived from the verb alone, so change.rollback-steps
  // (GET /changes/:id/rollback-steps) matched /rollback/ and demanded --yes just to
  // list a checklist. HTTP method now wins over the verb regex.
  const before = hits.get('/changes/1/rollback-steps') ?? 0;
  const r = await run(['--json', 'change', 'rollback-steps', '1']);
  assert.equal(r.code, 0, r.stdout);
  assert.equal(envelope(r).ok, true);
  assert.equal(hits.get('/changes/1/rollback-steps') ?? 0, before + 1);

  // The guard itself must stay intact for real mutations.
  const gated = await run(['--json', 'change', 'rollback', '1']);
  assert.equal(gated.code, 10);
});

test('a command with backend-required query params fails locally with exit 2', async () => {
  // dto.ChangeCalendarRequest binds startDate/endDate as required; sending them
  // costs one round trip to learn that, and page/pageSize do not exist here.
  const before = hits.get('/changes/calendar') ?? 0;
  const r = await run(['--json', 'change', 'calendar']);
  assert.equal(r.code, 2);
  assert.equal(envelope(r).error.kind, 'usage');
  assert.match(envelope(r).error.message, /startDate/);
  assert.equal(hits.get('/changes/calendar') ?? 0, before, 'no request may be sent for a usage error');
});

// ---------------------------------------------------------------------------
// Discovery surfaces
// ---------------------------------------------------------------------------

test('schema exposes required params and backend enum values', async () => {
  const r = await run(['--json', 'schema', 'ticket.create']);
  assert.equal(r.code, 0);
  const env = envelope(r);
  assert.deepEqual(env.data.input.required, ['title', 'description', 'priority']);
  const priority = env.data.input.params.find((p) => p.name === 'priority');
  assert.equal(priority.type, 'enum');
  assert.ok(priority.values.includes('urgent'));
});

test('schema --as tools emits function-calling definitions with a CLI mapping', async () => {
  const r = await run(['--json', 'schema', '--as', 'tools', '--domain', 'ticket']);
  assert.equal(r.code, 0);
  const env = envelope(r);
  assert.equal(env.data.format, 'tools');
  const create = env.data.tools.find((t) => t.name === 'itsm_ticket_create');
  assert.ok(create, 'expected a itsm_ticket_create tool');
  assert.deepEqual(create.input_schema.required.sort(), ['description', 'priority', 'title']);
  assert.equal(create['x-itsm'].command, 'ticket.create');
  assert.equal(create['x-itsm'].positional[0], 'title');
});

test('manifest advertises the protocol and output controls', async () => {
  const env = envelope(await run(['--json', 'manifest']));
  assert.equal(env.data.agentProtocol, true);
  assert.equal(env.data.exitCodes['10'], 'confirmation_required (high-risk write; show risk, get consent, then retry with --yes)');
  assert.ok(env.data.outputControls['--fields id,title']);
  assert.ok(env.data.commands > 100);
});

test('doctor reports reachability, identity and per-domain probes in one call', async () => {
  const r = await run(['--json', 'doctor']);
  assert.equal(r.code, 0);
  const env = envelope(r);
  assert.equal(env.data.verdict, 'healthy');
  assert.equal(env.data.identity.username, 'agent');
  const names = env.data.checks.map((c) => c.name);
  for (const expected of ['reachability', 'version', 'identity', 'read:tickets', 'read:changes']) {
    assert.ok(names.includes(expected), `doctor must probe ${expected}`);
  }
});
