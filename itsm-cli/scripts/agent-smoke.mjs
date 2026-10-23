#!/usr/bin/env node
/**
 * Agent-surface smoke test.
 *
 * Locks the contracts an external agent (WorkBuddy / CI / MCP bridge) depends on.
 * Local assertions need no backend; online assertions run `infra health` and, when
 * credentials are present, one authenticated read.
 *
 *   node scripts/agent-smoke.mjs             # local + online
 *   node scripts/agent-smoke.mjs --offline   # local protocol assertions only
 *
 * Exit 0 = all passed, 1 = at least one failed.
 */
import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const bin = path.join(here, '..', 'bin', 'itsm.js');
const offline = process.argv.includes('--offline');

let failures = 0;

function run(args) {
  const r = spawnSync(process.execPath, [bin, '--json', ...args], { encoding: 'utf8' });
  let json = null;
  try {
    json = JSON.parse(r.stdout.trim());
  } catch {
    // Leave null — the assertion below reports the raw output.
  }
  return { code: r.status, json, raw: (r.stdout + r.stderr).trim() };
}

function check(name, ok, detail = '') {
  if (ok) {
    console.log(`PASS  ${name}`);
  } else {
    failures++;
    console.log(`FAIL  ${name}${detail ? ` — ${detail}` : ''}`);
  }
}

// ---- protocol surface (no backend needed) ---------------------------------
const manifest = run(['manifest']);
check('manifest advertises agentProtocol', manifest.code === 0 && manifest.json?.data?.agentProtocol === true, manifest.raw.slice(0, 160));
check('manifest documents exit code 10', Boolean(manifest.json?.data?.exitCodes?.['10']));

const catalog = run(['commands']);
const domainCount = catalog.json?.data?.domains?.length ?? 0;
check('command catalog has the expected domains (>=15)', catalog.code === 0 && domainCount >= 15, `got ${domainCount}`);

const unknown = run(['ticket', 'no-such-verb']);
check('unknown command exits 2', unknown.code === 2, `exit ${unknown.code}`);

// --help must short-circuit before the high-risk exit-10 gate.
const help = run(['incident', 'close', '--help']);
check('verb --help exits 0 instead of 10', help.code === 0, `exit ${help.code}: ${help.raw.slice(0, 120)}`);
check('incident close help mentions --closeNotes', String(help.json?.data?.usage ?? '').includes('--closeNotes'));

// Flat-argument path for problem create (no hand-written JSON).
const problemHelp = run(['problem', 'create', '--help']);
check('problem create help mentions --description', String(problemHelp.json?.data?.usage ?? '').includes('--description'));
const problemMissing = run(['problem', 'create', 'only a title']);
check('problem create validates locally (exit 2, no request)', problemMissing.code === 2, `exit ${problemMissing.code}`);

// Dry-run previews a destructive write without the --yes gate.
const dry = run(['ticket', 'delete', '1', '--dry-run']);
check('--dry-run previews high-risk write (exit 0)', dry.code === 0 && dry.json?.data?.dryRun === true, `exit ${dry.code}`);
const gate = run(['ticket', 'delete', '1']);
check('high-risk write without --yes exits 10', gate.code === 10 && gate.json?.error?.kind === 'confirmation_required', `exit ${gate.code}`);

// ---- online surface --------------------------------------------------------
if (!offline) {
  const health = run(['infra', 'health']);
  const reachable = health.code === 0;
  check('infra health reachable', reachable, `exit ${health.code}: ${health.raw.slice(0, 160)}`);

  if (reachable) {
    const status = run(['auth', 'status']);
    const source = status.json?.data?.authSource;
    if (source && source !== 'none') {
      const stats = run(['ticket', 'stats']);
      check(`authenticated read works (authSource=${source})`, stats.code === 0, `exit ${stats.code}: ${stats.raw.slice(0, 160)}`);
    } else {
      console.log('SKIP  authenticated read — no credentials (run `itsm --json auth login ...` once)');
    }
  }
}

console.log(`\n${failures === 0 ? 'OK' : 'FAILED'}: ${failures} failure(s)`);
process.exit(failures === 0 ? 0 : 1);
