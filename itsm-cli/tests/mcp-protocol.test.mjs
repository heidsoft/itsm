/**
 * MCP protocol regression suite.
 *
 * The MCP surface is what a host (WorkBuddy, Claude Desktop, any agent runtime)
 * depends on: a clean JSON-RPC handshake, tool schemas that match the CLI's own
 * contract, no high-risk write without consent, bounded payloads, and stdout
 * that carries nothing but protocol frames.
 *
 * Every assertion spawns the real `bin/itsm-mcp.js` against a stub backend —
 * the same entrypoint a host launches, not an in-process helper.
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
const BIN = path.join(ROOT, 'bin/itsm-mcp.js');
const HOME = mkdtempSync(path.join(tmpdir(), 'itsm-mcp-test-'));
const TOKEN = 'test-token';

let server;
let baseUrl;
/** "METHOD /path" → hit count, for "no request was sent" assertions. */
const hits = new Map();
/** "METHOD /path" → last JSON body, for "did the field name match the DTO" assertions. */
const bodies = new Map();
/** Query strings seen by GET /bpmn/tasks, in call order. */
const taskQueries = [];

/**
 * Refresh-token state for the single-use rotation stub.
 *
 * The backend revokes a refresh token the moment it is exchanged. `refreshState`
 * mirrors that: `current` is the only token that will be accepted, and every
 * successful exchange mints a new pair.
 */
const refreshState = { current: 'refresh-0', generation: 0, hits: 0 };

const ok = (res, data) => json(res, 200, { code: 0, message: 'success', data });
function json(res, status, payload) {
  res.writeHead(status, { 'content-type': 'application/json' });
  res.end(JSON.stringify(payload));
}

function readBody(req) {
  return new Promise((resolve) => {
    let raw = '';
    req.on('data', (c) => (raw += c));
    req.on('end', () => resolve(raw));
  });
}

async function route(req, res) {
  const url = new URL(req.url, baseUrl);
  const p = url.pathname.replace(/^\/api\/v1/, '');
  const key = `${req.method} ${p}`;
  hits.set(key, (hits.get(key) ?? 0) + 1);

  if (req.method === 'POST') {
    const raw = await readBody(req);
    bodies.set(key, raw ? JSON.parse(raw) : null);
  }

  if (p === '/version') return ok(res, { version: 'v9.9.9', build: 'test' });
  if (p === '/readyz') return ok(res, { status: 'ready' });

  // Refresh is called without an access token, so it is matched before the
  // bearer check. Rotation is single-use: the presented token dies here.
  if (p === '/auth/refresh' && req.method === 'POST') {
    refreshState.hits += 1;
    const body = bodies.get(key) ?? {};
    if (body.refreshToken !== refreshState.current) {
      return json(res, 401, { code: 2001, message: 'refresh token 已被轮换' });
    }
    refreshState.generation += 1;
    refreshState.current = `refresh-${refreshState.generation}`;
    const access = `rotated-${refreshState.generation}`;
    res.writeHead(200, {
      'content-type': 'application/json',
      'set-cookie': [
        `access_token=${access}; Path=/; HttpOnly`,
        `refresh_token=${refreshState.current}; Path=/; HttpOnly`,
      ],
    });
    return res.end(JSON.stringify({ code: 0, message: 'success', data: {} }));
  }

  const bearer = req.headers.authorization;
  const authorized = bearer === `Bearer ${TOKEN}` || bearer === `Bearer rotated-${refreshState.generation}`;
  if (!authorized) {
    return json(res, 401, { code: 2001, message: '缺少认证token' });
  }

  if (p === '/auth/me') return ok(res, { id: 1, username: 'agent', role: 'admin', tenantId: 1 });
  // "My tasks": returns every status unless `status` is filtered.
  if (p === '/bpmn/tasks' && req.method === 'GET') {
    taskQueries.push(url.searchParams);
    return ok(res, { items: [{ id: 1, taskName: '变更评估', status: url.searchParams.get('status') ?? 'unspecified' }], total: 1 });
  }
  // Live CI list endpoint: `search` is its keyword param.
  if (p === '/cmdb/cis' && req.method === 'GET') {
    const search = url.searchParams.get('search');
    return ok(res, {
      items: search ? [{ id: 1, name: 'web-server-test' }] : [],
      total: search ? 1 : 0,
      page: 1,
      pageSize: 20,
      totalPages: 1,
    });
  }
  // Knowledge search: the handler binds `query` with `binding:"required"` and a
  // `{q:...}` body is a 400, so the stub refuses anything else.
  if (p === '/knowledge/search' && req.method === 'POST') {
    const body = bodies.get(key) ?? {};
    if (typeof body.query !== 'string' || !body.query) {
      return json(res, 400, { code: 1001, message: '参数错误' });
    }
    return ok(res, { items: [], total: 0 });
  }
  if (p === '/tickets' && req.method === 'GET') {
    const items = Array.from({ length: 80 }, (_, i) => ({
      id: i + 1,
      ticketNumber: `TKT-${i + 1}`,
      title: `ticket ${i + 1}`,
      status: 'new',
      priority: 'medium',
      description: 'x'.repeat(200),
    }));
    return ok(res, { items, total: 80, page: 1, pageSize: 20, totalPages: 4 });
  }
  if (/^\/tickets\/\d+$/.test(p) && req.method === 'GET') {
    return ok(res, { id: Number(p.split('/')[2]), ticketNumber: 'TKT-1', title: 'a', status: 'new' });
  }
  if (/^\/cmdb\/cis\/\d+$/.test(p) && req.method === 'DELETE') return ok(res, null);

  return json(res, 404, { code: 4004, message: `no stub for ${key}` });
}

before(async () => {
  server = createServer(route);
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  baseUrl = `http://127.0.0.1:${server.address().port}`;
});

after(async () => {
  await new Promise((resolve) => server.close(resolve));
});

/**
 * Send a batch of JSON-RPC frames, close stdin, and collect every frame the
 * server wrote back. Closing stdin right after the batch doubles as a regression
 * test for the drain path: in-flight calls must still be answered.
 */
/**
 * @param overrides keys set to `undefined` are dropped from the child env, so a
 * test can fall back from `ITSM_TOKEN` to the credential store.
 */
function session(frames, overrides = {}) {
  const env = {
    ...process.env,
    ITSM_BASE_URL: baseUrl,
    ITSM_TOKEN: TOKEN,
    ITSM_CREDENTIALS_FILE: path.join(HOME, 'credentials'),
    ITSM_REPO_ROOT: path.resolve(ROOT, '..'),
    ITSM_MCP_DEBUG: '0',
    ...overrides,
  };
  for (const [k, v] of Object.entries(env)) if (v === undefined) delete env[k];

  return new Promise((resolve, reject) => {
    const child = spawn(process.execPath, [BIN], {
      cwd: ROOT,
      env,
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (c) => (stdout += c));
    child.stderr.on('data', (c) => (stderr += c));

    child.on('error', reject);
    child.on('close', (code) => {
      const lines = stdout.split('\n').filter((l) => l.trim());
      const messages = [];
      for (const line of lines) {
        try {
          messages.push(JSON.parse(line));
        } catch {
          throw assert.fail(`stdout carried a non-JSON frame (breaks the protocol): ${line.slice(0, 120)}`);
        }
      }
      const byId = new Map(messages.filter((m) => m.id !== undefined && m.id !== null).map((m) => [m.id, m]));
      resolve({ code, messages, byId, stdout, stderr });
    });

    for (const frame of frames) child.stdin.write(`${JSON.stringify(frame)}\n`);
    child.stdin.end();
  });
}

const initialize = { jsonrpc: '2.0', id: 'init', method: 'initialize', params: { protocolVersion: '2025-06-18', capabilities: {} } };

test('initialize negotiates the protocol and advertises all three primitives', async () => {
  const { byId } = await session([initialize]);
  const result = byId.get('init').result;
  assert.equal(result.protocolVersion, '2025-06-18');
  assert.equal(result.serverInfo.name, 'itsm-mcp');
  assert.ok(result.capabilities.tools);
  assert.ok(result.capabilities.resources);
  assert.ok(result.capabilities.prompts);
  assert.match(result.instructions, /itsm_discover/);
});

test('an unknown protocol version falls back to a supported one instead of failing', async () => {
  const { byId } = await session([
    { jsonrpc: '2.0', id: 1, method: 'initialize', params: { protocolVersion: '1999-01-01', capabilities: {} } },
  ]);
  assert.ok(['2025-06-18', '2025-03-26', '2024-11-05'].includes(byId.get(1).result.protocolVersion));
});

test('tools/list: unique names, strict schemas, curated tools marked read-only', async () => {
  const { byId } = await session([initialize, { jsonrpc: '2.0', id: 2, method: 'tools/list' }]);
  const tools = byId.get(2).result.tools;
  assert.ok(tools.length >= 20, `expected a curated surface, got ${tools.length}`);

  const names = tools.map((t) => t.name);
  assert.deepEqual([...new Set(names)].sort(), names.slice().sort(), 'tool names must be unique');

  for (const tool of tools) {
    assert.equal(tool.inputSchema.type, 'object', `${tool.name} schema must be an object`);
    assert.equal(tool.inputSchema.additionalProperties, false, `${tool.name} must reject unknown arguments`);
    assert.ok(tool.description.length > 10, `${tool.name} needs a usable description`);
  }

  const readTool = tools.find((t) => t.name === 'itsm_ticket_list');
  assert.equal(readTool.annotations.readOnlyHint, true);

  const meta = tools.find((t) => t.name === 'itsm_call');
  assert.equal(meta.annotations.destructiveHint, true, 'itsm_call can reach destructive commands, so it must say so');
});

test('tools/list exposes the full input contract of a curated command', async () => {
  const { byId } = await session([initialize, { jsonrpc: '2.0', id: 3, method: 'tools/list' }]);
  const tools = byId.get(3).result.tools;
  const create = tools.find((t) => t.name === 'itsm_ticket_create');
  const props = create.inputSchema.properties;
  assert.deepEqual(create.inputSchema.required.sort(), ['description', 'priority', 'title']);
  assert.deepEqual(props.priority.enum, ['low', 'medium', 'high', 'critical', 'urgent']);
  assert.equal(props.id, undefined, 'create has no id positional');
});

test('a list call is bounded by default and reports total vs returned', async () => {
  const { byId } = await session([
    initialize,
    { jsonrpc: '2.0', id: 4, method: 'tools/call', params: { name: 'itsm_ticket_list', arguments: {} } },
  ]);
  const result = byId.get(4).result;
  assert.equal(result.isError, false);
  const payload = JSON.parse(result.content[0].text);
  assert.equal(payload.data.items.length, 50, 'default cap keeps one call from flooding the context');
  assert.equal(payload.meta.total, 80);
  assert.equal(payload.meta.truncated, true);
});

test('fields projection and explicit limit are applied together', async () => {
  const { byId } = await session([
    initialize,
    {
      jsonrpc: '2.0',
      id: 5,
      method: 'tools/call',
      params: { name: 'itsm_ticket_list', arguments: { limit: 2, fields: ['id', 'title'] } },
    },
  ]);
  const payload = JSON.parse(byId.get(5).result.content[0].text);
  assert.equal(payload.data.items.length, 2);
  assert.deepEqual(Object.keys(payload.data.items[0]).sort(), ['id', 'title']);
  assert.deepEqual(payload.meta.projected, ['id', 'title']);
});

test('a high-risk write is refused without confirm and never reaches the backend', async () => {
  hits.clear();
  const { byId } = await session([
    initialize,
    {
      jsonrpc: '2.0',
      id: 6,
      method: 'tools/call',
      params: { name: 'itsm_call', arguments: { command: 'cmdb.delete', params: { id: 7 } } },
    },
  ]);
  const result = byId.get(6).result;
  assert.equal(result.isError, true);
  const payload = JSON.parse(result.content[0].text);
  assert.equal(payload.error.kind, 'confirmation_required');
  assert.equal(payload.error.risk.irreversible, true);
  assert.match(payload.error.hint, /confirm/);
  assert.equal(hits.get('DELETE /cmdb/cis/7') ?? 0, 0, 'the destructive call must not be sent');
});

test('the same write executes once confirm is set', async () => {
  hits.clear();
  const { byId } = await session([
    initialize,
    {
      jsonrpc: '2.0',
      id: 7,
      method: 'tools/call',
      params: { name: 'itsm_call', arguments: { command: 'cmdb.delete', params: { id: 7 }, confirm: true } },
    },
  ]);
  assert.equal(byId.get(7).result.isError, false);
  assert.equal(hits.get('DELETE /cmdb/cis/7'), 1);
});

test('preview describes the call without sending it', async () => {
  hits.clear();
  const { byId } = await session([
    initialize,
    {
      jsonrpc: '2.0',
      id: 8,
      method: 'tools/call',
      params: { name: 'itsm_call', arguments: { command: 'ticket.get', params: { id: 1 }, preview: true } },
    },
  ]);
  const payload = JSON.parse(byId.get(8).result.content[0].text);
  assert.equal(payload.preview, true);
  assert.deepEqual(payload.endpoints, ['GET /api/v1/tickets/:id']);
  assert.equal(hits.get('GET /tickets/1') ?? 0, 0);
});

test('an undeclared argument is rejected with the allowed names, before any request', async () => {
  hits.clear();
  const { byId } = await session([
    initialize,
    {
      jsonrpc: '2.0',
      id: 9,
      method: 'tools/call',
      params: { name: 'itsm_ticket_get', arguments: { ticketId: 1 } },
    },
  ]);
  const result = byId.get(9).result;
  assert.equal(result.isError, true);
  const payload = JSON.parse(result.content[0].text);
  assert.equal(payload.error.kind, 'usage');
  assert.match(payload.error.message, /id/);
  assert.equal(hits.get('GET /tickets/1') ?? 0, 0);
});

test('discover works in Chinese and ranks name hits above summary hits', async () => {
  const { byId } = await session([
    initialize,
    { jsonrpc: '2.0', id: 10, method: 'tools/call', params: { name: 'itsm_discover', arguments: { query: '审批', limit: 6 } } },
  ]);
  const payload = JSON.parse(byId.get(10).result.content[0].text);
  assert.ok(payload.commands.length > 0, '中文查询必须命中');
  assert.match(payload.commands[0].command, /approv/, `top hit should be an approval command, got ${payload.commands[0].command}`);
});

test('describe returns the CLI contract, and misses carry a suggestion', async () => {
  const { byId } = await session([
    initialize,
    { jsonrpc: '2.0', id: 11, method: 'tools/call', params: { name: 'itsm_describe', arguments: { command: 'ticket.create' } } },
    { jsonrpc: '2.0', id: 12, method: 'tools/call', params: { name: 'itsm_describe', arguments: { command: 'ticket.creat' } } },
  ]);
  const described = JSON.parse(byId.get(11).result.content[0].text);
  assert.equal(described.command, 'ticket.create');
  assert.deepEqual(described.requiresConfirmation, false);
  assert.ok(described.params.some((p) => p.name === 'priority' && p.required));

  const miss = byId.get(12).result;
  assert.equal(miss.isError, true);
  assert.match(JSON.parse(miss.content[0].text).error.message, /ticket\.create/);
});

test('resources: capability map is live, unknown URIs are a protocol error', async () => {
  const { byId } = await session([
    initialize,
    { jsonrpc: '2.0', id: 13, method: 'resources/list' },
    { jsonrpc: '2.0', id: 14, method: 'resources/read', params: { uri: 'itsm://capabilities' } },
    { jsonrpc: '2.0', id: 15, method: 'resources/read', params: { uri: 'itsm://nope' } },
  ]);
  const listed = byId.get(13).result.resources.map((r) => r.uri);
  assert.ok(listed.includes('itsm://capabilities'));
  assert.ok(listed.some((u) => u.startsWith('itsm://docs/')));

  const caps = JSON.parse(byId.get(14).result.contents[0].text);
  assert.ok(caps.commands > 200, `expected the full registry, got ${caps.commands}`);
  assert.ok(caps.domains.some((d) => d.domain === 'ticket'));

  assert.equal(byId.get(15).error.code, -32002);
});

test('docs resources read the repository, not a bundled copy', async () => {
  const { byId } = await session([
    initialize,
    { jsonrpc: '2.0', id: 16, method: 'resources/read', params: { uri: 'itsm://docs/changelog' } },
  ]);
  const text = byId.get(16).result.contents[0].text;
  assert.match(text, /#/, 'CHANGELOG should be markdown');
  assert.equal(byId.get(16).result.contents[0].mimeType, 'text/markdown');
});

test('prompts: instructions render, missing required arguments are invalid params', async () => {
  const { byId } = await session([
    initialize,
    { jsonrpc: '2.0', id: 17, method: 'prompts/list' },
    { jsonrpc: '2.0', id: 18, method: 'prompts/get', params: { name: 'incident_postmortem', arguments: { incidentId: 'INC-9' } } },
    { jsonrpc: '2.0', id: 19, method: 'prompts/get', params: { name: 'incident_postmortem', arguments: {} } },
  ]);
  const names = byId.get(17).result.prompts.map((p) => p.name);
  assert.deepEqual(names.sort(), ['change_risk_review', 'incident_postmortem', 'triage_my_queue']);

  const rendered = byId.get(18).result.messages[0].content.text;
  assert.match(rendered, /INC-9/);
  assert.equal(byId.get(19).error.code, -32602);
});

test('unknown methods are MethodNotFound, and the server keeps serving afterwards', async () => {
  const { byId } = await session([
    initialize,
    { jsonrpc: '2.0', id: 20, method: 'tools/gimme' },
    { jsonrpc: '2.0', id: 21, method: 'ping' },
  ]);
  assert.equal(byId.get(20).error.code, -32601);
  assert.deepEqual(byId.get(21).result, {});
});

test('a malformed frame yields a parse error without killing the session', async () => {
  const child = spawn(process.execPath, [BIN], {
    cwd: ROOT,
    env: { ...process.env, ITSM_BASE_URL: baseUrl, ITSM_TOKEN: TOKEN, ITSM_CREDENTIALS_FILE: path.join(HOME, 'credentials') },
    stdio: ['pipe', 'pipe', 'pipe'],
  });

  const messages = await new Promise((resolve, reject) => {
    let out = '';
    child.stdout.on('data', (c) => (out += c));
    child.on('error', reject);
    child.on('close', () => resolve(out.split('\n').filter((l) => l.trim()).map((l) => JSON.parse(l))));
    child.stdin.write('{not json}\n');
    child.stdin.write(`${JSON.stringify({ jsonrpc: '2.0', id: 1, method: 'ping' })}\n`);
    child.stdin.end();
  });

  const parseError = messages.find((m) => m.error?.code === -32700);
  assert.ok(parseError, 'malformed input must be reported as -32700');
  assert.equal(parseError.id, null);
  assert.ok(messages.some((m) => m.id === 1 && m.result), 'the next request must still be answered');
});

test('every request in a batch gets an answer even though stdin closes immediately', async () => {
  const frames = [initialize];
  for (let i = 0; i < 6; i++) {
    frames.push({ jsonrpc: '2.0', id: `batch-${i}`, method: 'tools/call', params: { name: 'itsm_ticket_get', arguments: { id: i + 1 } } });
  }
  const { byId, code } = await session(frames);
  for (let i = 0; i < 6; i++) {
    assert.ok(byId.get(`batch-${i}`), `batch-${i} was dropped`);
  }
  assert.equal(code, 0);
});

test('CMDB keyword search goes to the live GET endpoint, not the deprecated POST body', async () => {
  const postBefore = hits.get('POST /cmdb/cis/search') ?? 0;
  const getBefore = hits.get('GET /cmdb/cis') ?? 0;

  const { byId } = await session([
    initialize,
    { jsonrpc: '2.0', id: 'cmdb', method: 'tools/call', params: { name: 'itsm_cmdb_search', arguments: { q: 'server' } } },
  ]);

  const payload = JSON.parse(byId.get('cmdb').result.content[0].text);
  assert.equal(payload.ok, true, payload.error?.message);
  assert.equal(hits.get('POST /cmdb/cis/search') ?? 0, postBefore, 'a keyword search must not use the structured POST endpoint');
  assert.ok((hits.get('GET /cmdb/cis') ?? 0) > getBefore, 'keyword search must hit GET /cmdb/cis');
});

test('knowledge search sends `query`, the field the backend binds as required', async () => {
  const { byId } = await session([
    initialize,
    { jsonrpc: '2.0', id: 'kb', method: 'tools/call', params: { name: 'itsm_knowledge_search', arguments: { q: 'VPN', limit: 5 } } },
  ]);

  const payload = JSON.parse(byId.get('kb').result.content[0].text);
  assert.equal(payload.ok, true, payload.error?.message);
  const body = bodies.get('POST /knowledge/search');
  assert.equal(body.query, 'VPN');
  assert.ok(!('q' in body), 'the backend binds `query`; a `q` body is rejected as a param error');
});

/**
 * The usage string is the only source of a command's parameter list, and the
 * CLI's own parser silently ignores undeclared flags. A flag the implementation
 * reads but the usage never advertises is therefore invisible to every
 * schema-driven client (this server, `itsm --json schema`) while looking
 * perfectly implemented — the exact shape of the earlier `--status` bug.
 */
test('every flag a command reads is declared in its usage', async () => {
  const { readFileSync, readdirSync } = await import('node:fs');
  const { registry } = await import('../dist/agent/registry.js');
  const dir = path.join(ROOT, 'src/agent/commands');

  const specs = new Map(registry().map((s) => [s.name, s]));
  const offenders = [];

  for (const file of readdirSync(dir).filter((f) => f.endsWith('.ts'))) {
    const src = readFileSync(path.join(dir, file), 'utf8');
    const marks = [...src.matchAll(/cmd\('([^']+)'/g)];
    marks.forEach((m, i) => {
      const spec = specs.get(m[1]);
      if (!spec) return;
      const block = src.slice(m.index, i + 1 < marks.length ? marks[i + 1].index : src.length);
      const declared = new Set(spec.input.params.map((p) => p.name));
      const read = [...block.matchAll(/(?:string|boolean|number)Flag\(ctx\.args\.flags,\s*'([A-Za-z-]+)'/g)].map((x) => x[1]);
      if (/withPagination\(/.test(block)) read.push('page', 'pageSize');
      const missing = [...new Set(read)].filter((f) => !declared.has(f));
      if (missing.length) offenders.push(`${spec.name}: --${missing.join(' --')}`);
    });
  }

  assert.deepEqual(offenders, [], `undeclared flags are unreachable from the schema:\n${offenders.join('\n')}`);
});

test('itsm_workflow_tasks defaults to pending work, and an explicit status wins', async () => {
  taskQueries.length = 0;
  const { byId } = await session([
    initialize,
    { jsonrpc: '2.0', id: 'wf-default', method: 'tools/call', params: { name: 'itsm_workflow_tasks', arguments: { limit: 5 } } },
    { jsonrpc: '2.0', id: 'wf-explicit', method: 'tools/call', params: { name: 'itsm_workflow_tasks', arguments: { status: 'completed', limit: 5 } } },
  ]);

  assert.equal(byId.get('wf-default').result.isError, false);
  assert.equal(byId.get('wf-explicit').result.isError, false);
  // GET /bpmn/tasks without a status filter returns completed tasks too, so the
  // tool titled 待办 must narrow it — otherwise "待办 36 条" is 31 finished ones.
  assert.equal(taskQueries[0].get('status'), 'created', 'default must be pending only');
  assert.equal(taskQueries[1].get('status'), 'completed', 'the caller must be able to override the default');
});

/**
 * The access token lives 15 minutes while the refresh token lives 7 days, and
 * the backend rotates the refresh token on every exchange. A host fires tool
 * calls in parallel, so one expiry used to produce N concurrent 401s → N
 * refreshes with the same refresh token → 1 success and N-1 failures that all
 * claimed "the refresh token was rejected" and told the agent to log in again.
 * Measured before the fix: 8 parallel calls, 1 passed. The message is the worse
 * half — the race is self-inflicted, so "log in again" is wrong advice.
 */
test('parallel calls that hit an expired token share one refresh instead of racing', async () => {
  const { writeFileSync } = await import('node:fs');
  const credFile = path.join(HOME, 'credentials-race');
  refreshState.current = 'refresh-0';
  refreshState.generation = 0;
  refreshState.hits = 0;

  writeFileSync(
    credFile,
    JSON.stringify({
      token: 'expired-access-token',
      refreshToken: refreshState.current,
      expiresAt: new Date(Date.now() - 60_000).toISOString(),
      user: 'agent',
      version: 2,
    }),
    { mode: 0o600 }
  );

  const frames = [
    initialize,
    ...Array.from({ length: 8 }, (_, i) => ({
      jsonrpc: '2.0',
      id: `race-${i}`,
      method: 'tools/call',
      params: { name: 'itsm_whoami', arguments: {} },
    })),
  ];

  // No ITSM_TOKEN: the call must fall back to the credential store and renew
  // through /auth/refresh, which is the path that raced.
  const { byId } = await session(frames, { ITSM_TOKEN: undefined, ITSM_CREDENTIALS_FILE: credFile });

  const failures = [];
  for (let i = 0; i < 8; i++) {
    const payload = JSON.parse(byId.get(`race-${i}`).result.content[0].text);
    if (!payload.ok) failures.push(payload.error?.message ?? 'unknown');
  }

  assert.deepEqual(failures, [], `every parallel call must succeed on a shared refresh:\\n${failures.join('\\n')}`);
  assert.equal(refreshState.hits, 1, `exactly one exchange for the whole batch, got ${refreshState.hits}`);
});

/**
 * "knowledge" is a substring of "acknowledge", so a plain substring score ranks
 * `incident.acknowledge` level with `knowledge.*` — and the alphabetical
 * tie-break then puts it first. A host reading the list top-down is told the
 * acknowledge command is the best answer to 知识.
 */
test('a query matching a command name boundary outranks a match buried in another word', async () => {
  const { byId } = await session([
    initialize,
    { jsonrpc: '2.0', id: 'kb-rank', method: 'tools/call', params: { name: 'itsm_discover', arguments: { query: '知识', limit: 5 } } },
  ]);

  const payload = byId.get('kb-rank').result.structuredContent;
  assert.ok(payload.commands.length > 0, '查询必须命中');
  assert.match(
    payload.commands[0].command,
    /^knowledge\./,
    `the top hit should be a knowledge command, got ${payload.commands[0].command}`
  );
});
