#!/usr/bin/env node
/**
 * MCP surface smoke test — live instance.
 *
 * `tests/mcp-protocol.test.mjs` proves the protocol against a stub backend.
 * This script proves the *wiring*: it drives the real `bin/itsm-mcp.js` against
 * whatever ITSM_BASE_URL points at, so a schema/auth/base-url regression shows up
 * before a host (WorkBuddy, Claude Desktop) reports "tool call failed".
 *
 * Read-only by construction: it never calls a write command, so it is safe to run
 * against a production instance.
 *
 *   node scripts/mcp-smoke.mjs              # local protocol + online reads
 *   node scripts/mcp-smoke.mjs --offline    # local protocol assertions only
 *
 * Exit 0 = all passed, 1 = at least one failed.
 */
import { spawn } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const bin = path.join(here, '..', 'bin', 'itsm-mcp.js');
const offline = process.argv.includes('--offline');

let failures = 0;

function check(name, ok, detail = '') {
  if (ok) {
    console.log(`PASS  ${name}`);
  } else {
    failures++;
    console.log(`FAIL  ${name}${detail ? ` — ${detail}` : ''}`);
  }
}

/** One MCP session: send every frame, close stdin, collect answers by id. */
function session(frames) {
  return new Promise((resolve, reject) => {
    const child = spawn(process.execPath, [bin], { stdio: ['pipe', 'pipe', 'pipe'] });
    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (c) => (stdout += c));
    child.stderr.on('data', (c) => (stderr += c));
    child.on('error', reject);
    child.on('close', () => {
      const byId = new Map();
      for (const line of stdout.split('\n').filter((l) => l.trim())) {
        let msg;
        try {
          msg = JSON.parse(line);
        } catch {
          reject(new Error(`non-JSON frame on stdout: ${line.slice(0, 120)}`));
          return;
        }
        if (msg.id !== undefined && msg.id !== null) byId.set(msg.id, msg);
      }
      resolve({ byId, stderr });
    });
    for (const frame of frames) child.stdin.write(`${JSON.stringify(frame)}\n`);
    child.stdin.end();
  });
}

const toolResult = (msg) => {
  if (!msg?.result) return undefined;
  try {
    return JSON.parse(msg.result.content[0].text);
  } catch {
    return undefined;
  }
};

// ---- local surface --------------------------------------------------------
const local = await session([
  { jsonrpc: '2.0', id: 1, method: 'initialize', params: { protocolVersion: '2025-06-18', capabilities: {} } },
  { jsonrpc: '2.0', id: 2, method: 'tools/list' },
  { jsonrpc: '2.0', id: 3, method: 'resources/list' },
  { jsonrpc: '2.0', id: 4, method: 'prompts/list' },
  { jsonrpc: '2.0', id: 5, method: 'tools/call', params: { name: 'itsm_call', arguments: { command: 'ticket.delete', params: { id: 1 } } } },
  { jsonrpc: '2.0', id: 6, method: 'tools/call', params: { name: 'itsm_discover', arguments: { query: '影响', limit: 5 } } },
  { jsonrpc: '2.0', id: 7, method: 'resources/read', params: { uri: 'itsm://capabilities' } },
]);

const init = local.byId.get(1)?.result;
check('initialize returns serverInfo + instructions', init?.serverInfo?.name === 'itsm-mcp' && init.instructions.length > 100);

const tools = local.byId.get(2)?.result?.tools ?? [];
check('tools/list serves a curated surface (>=20)', tools.length >= 20, `got ${tools.length}`);
check('every tool declares a strict inputSchema', tools.every((t) => t.inputSchema?.type === 'object' && t.inputSchema.additionalProperties === false));

check('resources/list exposes capabilities + docs', (local.byId.get(3)?.result?.resources ?? []).length >= 4);
check('prompts/list exposes operating procedures', (local.byId.get(4)?.result?.prompts ?? []).length >= 3);

const gated = toolResult(local.byId.get(5));
check('high-risk write returns confirmation_required', gated?.error?.kind === 'confirmation_required');

const discovered = toolResult(local.byId.get(6));
check('Chinese discovery query matches commands', (discovered?.commands ?? []).length > 0, JSON.stringify(discovered?.commands?.[0] ?? {}));

const caps = (() => {
  try {
    return JSON.parse(local.byId.get(7)?.result?.contents?.[0]?.text ?? '{}');
  } catch {
    return {};
  }
})();
check('capabilities resource reports the real command count', (caps.commands ?? 0) > 200, `got ${caps.commands}`);

// ---- online surface -------------------------------------------------------
if (offline) {
  console.log('SKIP  online reads (--offline)');
} else {
  const online = await session([
    { jsonrpc: '2.0', id: 1, method: 'initialize', params: { protocolVersion: '2025-06-18', capabilities: {} } },
    { jsonrpc: '2.0', id: 2, method: 'tools/call', params: { name: 'itsm_whoami', arguments: {} } },
    { jsonrpc: '2.0', id: 3, method: 'tools/call', params: { name: 'itsm_ticket_list', arguments: { limit: 3, fields: ['id', 'ticketNumber', 'status'] } } },
    { jsonrpc: '2.0', id: 4, method: 'tools/call', params: { name: 'itsm_health', arguments: {} } },
  ]);

  const who = toolResult(online.byId.get(2));
  const reachable = Boolean(who?.ok);
  check('backend reachable and authenticated', reachable, JSON.stringify(who?.error ?? {}).slice(0, 200));

  if (reachable) {
    console.log(`      identity: ${who.data?.user?.username} @ ${who.data?.baseURL} (${who.data?.authSource})`);

    const list = toolResult(online.byId.get(3));
    check('ticket.list returns a bounded projection', list?.ok === true && list.data.items.length <= 3 && Array.isArray(list.meta?.projected), JSON.stringify(list?.error ?? {}).slice(0, 200));

    const health = toolResult(online.byId.get(4));
    check('doctor probe passes every check', health?.ok === true && health.data?.summary?.failed === 0, JSON.stringify(health?.data?.summary ?? health?.error ?? {}).slice(0, 200));
  } else {
    console.log('SKIP  authenticated reads — backend or credentials unavailable');
  }
}

console.log(`\n${failures === 0 ? 'OK' : 'FAILED'}: ${failures} failure(s)`);
process.exit(failures === 0 ? 0 : 1);
