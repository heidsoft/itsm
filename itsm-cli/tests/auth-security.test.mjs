/**
 * Auth security regression suite.
 *
 * These are the invariants an enterprise buyer (and a security review) will
 * check first:
 *   · a password or reset token never has to appear in argv;
 *   · argv secrets are actively refused, not just discouraged;
 *   · secrets never reach stdout, even on success;
 *   · the credential file is 0600, written atomically, and never through a symlink;
 *   · plaintext HTTP to a remote host is refused before a secret is sent.
 *
 * Every assertion runs the real `bin/itsm.js` entrypoint against a stub backend.
 */
import { test, before, after } from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { mkdtempSync, writeFileSync, chmodSync, statSync, symlinkSync, readFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';

const ROOT = path.resolve(import.meta.dirname, '..');
const BIN = path.join(ROOT, 'bin/itsm.js');
const HOME = mkdtempSync(path.join(tmpdir(), 'itsm-cli-auth-'));
const ACCESS = 'access-token-must-not-leak';
const REFRESH = 'refresh-token-must-not-leak';

let server;
let baseUrl;
const hits = new Map();

function json(res, status, payload, headers = {}) {
  res.writeHead(status, { 'content-type': 'application/json', ...headers });
  res.end(JSON.stringify(payload));
}

function route(req, res) {
  const url = new URL(req.url, baseUrl);
  const p = url.pathname.replace(/^\/api\/v1/, '');
  hits.set(p, (hits.get(p) ?? 0) + 1);

  if (p === '/health') return json(res, 200, { status: 'ok' });
  if (p === '/auth/login' && req.method === 'POST') {
    return json(
      res,
      200,
      { code: 0, message: 'success', data: { user: { id: 1, username: 'admin', name: 'Admin', tenantId: 1 }, expiresIn: 900 } },
      { 'set-cookie': [`access_token=${ACCESS}; Path=/; HttpOnly`, `refresh_token=${REFRESH}; Path=/; HttpOnly`] }
    );
  }
  if (p === '/auth/me') return json(res, 200, { code: 0, message: 'success', data: { id: 1, username: 'admin', name: 'Admin', tenantId: 1 } });
  if (p === '/auth/logout') return json(res, 200, { code: 0, message: 'success', data: null });
  if (p === '/auth/refresh') {
    return json(
      res,
      200,
      { code: 0, message: 'success', data: null },
      { 'set-cookie': [`access_token=${ACCESS}; Path=/; HttpOnly`] }
    );
  }
  return json(res, 404, { code: 3001, message: `stub has no route for ${req.method} ${p}` });
}

before(async () => {
  server = createServer(route);
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  baseUrl = `http://127.0.0.1:${server.address().port}`;
});

after(() => server?.close());

function run(args, extraEnv = {}) {
  return new Promise((resolve) => {
    const child = spawn(process.execPath, [BIN, '--json', ...args], {
      env: {
        ...process.env,
        HOME,
        ITSM_BASE_URL: baseUrl,
        ITSM_TOKEN: '',
        ITSM_PASSWORD: '',
        ITSM_CREDENTIALS_FILE: path.join(HOME, 'credentials'),
        ...extraEnv,
      },
    });
    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (d) => (stdout += d));
    child.stderr.on('data', (d) => (stderr += d));
    child.on('close', (code) => resolve({ code, stdout, stderr }));
  });
}

function envelope(result) {
  const line = result.stdout.trim().split('\n').filter(Boolean).pop();
  assert.ok(line, `expected a JSON envelope, got stdout=${JSON.stringify(result.stdout)} stderr=${JSON.stringify(result.stderr)}`);
  return JSON.parse(line);
}

const count = (p) => hits.get(p) ?? 0;

test('a password passed on argv is refused before any request is sent', async () => {
  const before = count('/auth/login');
  const result = await run(['auth', 'login', '--username', 'admin', '--password', 'hunter2']);
  assert.equal(result.code, 2);
  const env = envelope(result);
  assert.equal(env.ok, false);
  assert.match(env.error.message, /Refusing '--password' on the command line/i);
  assert.equal(count('/auth/login'), before, 'must not attempt a login with an argv password');
  assert.ok(!result.stdout.includes('hunter2'));
});

test('a reset token passed on argv is refused', async () => {
  const result = await run(['auth', 'reset-password', '--email', 'a@b.test', '--token', 'reset-secret', '--password-stdin'], {});
  assert.equal(result.code, 2);
  assert.match(envelope(result).error.message, /Refusing '--token' on the command line/i);
  assert.ok(!result.stdout.includes('reset-secret'));
});

test('a group/world-readable password file is rejected', async () => {
  const file = path.join(HOME, 'pw-loose');
  writeFileSync(file, 'hunter2');
  chmodSync(file, 0o644);
  const result = await run(['auth', 'login', '--username', 'admin', '--password-file', file]);
  assert.equal(result.code, 2);
  assert.match(envelope(result).error.message, /accessible by group\/other/i);
});

test('a 0600 password file works, the envelope carries no token, and the store is 0600', async () => {
  const file = path.join(HOME, 'pw-tight');
  writeFileSync(file, 'hunter2');
  chmodSync(file, 0o600);
  const store = path.join(HOME, 'credentials-login');
  const result = await run(['auth', 'login', '--username', 'admin', '--password-file', file], { ITSM_CREDENTIALS_FILE: store });
  assert.equal(result.code, 0, result.stderr);
  assert.ok(!result.stdout.includes(ACCESS), 'access token must never be printed');
  assert.ok(!result.stdout.includes(REFRESH), 'refresh token must never be printed');
  const env = envelope(result);
  assert.equal(env.data.source, 'password');
  assert.equal(env.data.credentialsStored, true);

  const saved = JSON.parse(readFileSync(store, 'utf-8'));
  assert.equal(saved.token, ACCESS, 'token must be persisted for later calls');
  assert.equal(saved.refreshToken, REFRESH);
  const mode = statSync(store).mode & 0o777;
  assert.equal(mode, 0o600);
});

test('writing credentials through a planted symlink fails closed', async () => {
  const target = path.join(HOME, 'attacker-controlled');
  writeFileSync(target, 'not-json');
  const link = path.join(HOME, 'credentials-symlink');
  symlinkSync(target, link);
  const file = path.join(HOME, 'pw-tight');
  const result = await run(['auth', 'login', '--username', 'admin', '--password-file', file], { ITSM_CREDENTIALS_FILE: link });
  assert.equal(result.code, 6);
  assert.match(envelope(result).error.message, /symlink/i);
  assert.equal(readFileSync(target, 'utf-8'), 'not-json', 'must not follow the symlink');
});

test('plaintext HTTP to a remote host is refused before the secret is sent', async () => {
  const file = path.join(HOME, 'pw-tight');
  const result = await run(['auth', 'login', '--username', 'admin', '--password-file', file], {
    ITSM_BASE_URL: 'http://192.0.2.10:8090',
  });
  assert.equal(result.code, 2);
  const env = envelope(result);
  assert.match(env.error.message, /Refusing to send credentials/i);
  assert.match(env.error.hint ?? '', /ITSM_ALLOW_INSECURE_HTTP=1/);
});

test('ITSM_ALLOW_INSECURE_HTTP=1 is an explicit opt-in to legacy intranet HTTP', async () => {
  // The opt-in must reach the transport guard; it will still fail on the
  // unreachable TEST-NET host, but with a network error instead of a refusal.
  const file = path.join(HOME, 'pw-tight');
  const result = await run(['auth', 'login', '--username', 'admin', '--password-file', file], {
    ITSM_BASE_URL: 'http://192.0.2.10:1',
    ITSM_ALLOW_INSECURE_HTTP: '1',
  });
  assert.equal(result.code, 1, 'opted-in plaintext should proceed to the network layer');
  assert.doesNotMatch(envelope(result).error.message, /Refusing to send credentials/i);
});

test('manifest lets an agent discover the argv policy and the safe secret channels', async () => {
  const result = await run(['manifest']);
  assert.equal(result.code, 0);
  const manifest = envelope(result).data;
  assert.match(manifest.authSecurity.argvSecrets, /exit 2/);
  assert.match(manifest.authSecurity.argvSecrets, /--password-stdin/);
  for (const key of ['ITSM_TOKEN_FILE', 'ITSM_PASSWORD_FILE', 'ITSM_RESET_TOKEN_FILE', 'ITSM_CREDENTIALS_FILE', 'ITSM_ALLOW_INSECURE_HTTP']) {
    assert.ok(manifest.env[key], `manifest must advertise ${key}`);
  }
});
