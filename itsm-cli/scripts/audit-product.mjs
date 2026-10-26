#!/usr/bin/env node
/**
 * Product surface audit — live instance, read-only.
 *
 * `doctor` proves reachability and `check:route-drift` proves the endpoints
 * exist; neither proves the *payload* the product actually ships. This script
 * drives the real `bin/itsm.js --json` entrypoint over a table of read-only
 * probes and then checks the responses against the API contract the repository
 * declares for itself:
 *
 *   C1  field naming   — no PascalCase and no snake_case key may escape the DTO
 *   C2  unset time     — `0001-01-01T00:00:00Z` must not be rendered to callers
 *   C3  list envelope  — paginated lists must be {items,total,page,pageSize,totalPages};
 *                        non-paginated collections are checked by `collection` probes,
 *                        aggregate payloads (stats/trend/reconciliation) by `aggregate`
 *   C4  success shape  — `ok:true` must carry an explicit `data` key
 *   C5  identity refs  — an ID field must not be 0 while a *Name is populated
 *   C6  error mapping  — errors must carry a stable kind + honest httpStatus
 *   C7  risk gate      — a high-risk write must stop at exit 10 before the wire
 *   C8  shaping        — --fields/--limit must reshape, never silently pass
 *
 * plus negative probes (missing resources, dead identity) that assert the
 * failure path rather than the happy path.
 *
 * Read-only by construction: every probe is a GET, `--dry-run`, or a write that
 * is expected to be refused before it reaches the network.
 *
 *   node scripts/audit-product.mjs               # report to stdout (+ file)
 *   node scripts/audit-product.mjs --strict      # exit 1 when violations exist
 *   node scripts/audit-product.mjs --json        # machine-readable findings
 *
 * Exit: 0 run completed (violations are the product of the report), 1 under
 * --strict with violations, 2 probe/registry mismatch (the audit itself is wrong).
 */
import { spawn } from 'node:child_process';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, '..');
const bin = path.join(root, 'bin/itsm.js');

const strict = process.argv.includes('--strict');
const asJson = process.argv.includes('--json');

const { registry } = await import(path.join(root, 'dist/agent/registry.js'));
const specs = new Map(registry().map((s) => [s.name, s]));

// ---------------------------------------------------------------------------
// Probe table
// ---------------------------------------------------------------------------

/** Seeds: list/read calls whose first item supplies ids for later probes. */
const SEEDS = [
  ['tickets', 'ticket.list', ['--pageSize', '3']],
  ['incidents', 'incident.list', ['--pageSize', '3']],
  ['problems', 'problem.list', ['--pageSize', '3']],
  ['changes', 'change.list', ['--pageSize', '3']],
  ['releases', 'release.list', ['--pageSize', '3']],
  ['requests', 'request.list', ['--pageSize', '3']],
  ['catalogs', 'catalog.list', ['--pageSize', '3']],
  ['cis', 'cmdb.list', ['--pageSize', '3']],
  ['cis2', 'cmdb.list', ['--pageSize', '3']],
  ['knowledge', 'knowledge.list', ['--pageSize', '3']],
  ['notifications', 'notification.list', ['--pageSize', '3']],
  ['deliveries', 'notification.deliveries', ['--pageSize', '3']],
  ['workflowDefs', 'workflow.definitions', ['--pageSize', '3']],
  ['workflowInstances', 'workflow.instances', ['--pageSize', '3']],
  ['workflowInstanceKey', 'workflow.instances', ['--pageSize', '3'], 'instanceId'],
  ['connectors', 'connector.list', []],
  ['users', 'admin.users', ['--pageSize', '3']],
  ['tenants', 'admin.tenants', ['--pageSize', '3']],
  ['commands', 'admin.commands', ['--limit', '3']],
];

/**
 * Probes. `list:true` additionally asserts the standard list envelope,
 * `list:true, paged:false` asserts an array/object but not the page contract.
 */
const PROBES = [
  // ---- domain read surfaces -------------------------------------------------
  { cmd: 'ticket.get', args: ['{{tickets}}'] },
  { cmd: 'ticket.list', args: [], list: true },
  { cmd: 'ticket.stats', args: [] },
  { cmd: 'ticket.overdue', args: [], list: true },
  { cmd: 'ticket.search', args: ['打印'], list: true },
  { cmd: 'ticket.sla', args: ['{{tickets}}'] },
  { cmd: 'ticket.history', args: ['{{tickets}}'] },
  { cmd: 'ticket.comments.list', args: ['{{tickets}}'] },
  { cmd: 'ticket.approval-records', args: ['--ticketId', '{{tickets}}'] },
  { cmd: 'ticket.relations', args: ['{{tickets}}'] },
  { cmd: 'ticket.relations-stats', args: ['{{tickets}}'] },
  { cmd: 'ticket.configuration-items', args: ['{{tickets}}'] },
  { cmd: 'ticket.workflow-history', args: ['{{tickets}}'] },
  { cmd: 'ticket.subtasks.list', args: ['{{tickets}}'] },
  { cmd: 'ticket.notifications', args: ['{{tickets}}'] },
  { cmd: 'ticket.rating-stats', args: [], aggregate: true },

  { cmd: 'incident.get', args: ['{{incidents}}'] },
  { cmd: 'incident.list', args: [], list: true },
  { cmd: 'incident.stats', args: [] },
  { cmd: 'incident.report', args: [] },
  { cmd: 'incident.events', args: ['{{incidents}}'] },
  { cmd: 'incident.alerts', args: ['{{incidents}}'] },
  { cmd: 'incident.metrics', args: ['{{incidents}}'] },
  { cmd: 'incident.classification', args: ['{{incidents}}'] },
  { cmd: 'incident.impact-assessment', args: ['{{incidents}}'] },
  { cmd: 'incident.root-cause-get', args: ['{{incidents}}'] },
  { cmd: 'incident.comments.list', args: ['{{incidents}}'] },

  { cmd: 'problem.get', args: ['{{problems}}'] },
  { cmd: 'problem.list', args: [], list: true },
  { cmd: 'problem.stats', args: [] },
  { cmd: 'problem.trend', args: [], aggregate: true },
  { cmd: 'problem.hotspots', args: [], aggregate: true },
  { cmd: 'problem.known-errors.list', args: [], list: true },
  { cmd: 'problem.comments.list', args: ['{{problems}}'] },

  { cmd: 'change.get', args: ['{{changes}}'] },
  { cmd: 'change.list', args: [], list: true },
  { cmd: 'change.stats', args: [] },
  { cmd: 'change.calendar', args: ['--startDate', '2026-10-01', '--endDate', '2026-10-31'] },
  { cmd: 'change.approval-summary', args: ['{{changes}}'] },
  { cmd: 'change.approvals', args: ['{{changes}}'] },
  { cmd: 'change.risk-assessment', args: ['{{changes}}'] },
  { cmd: 'change.cmdb-impact', args: ['{{changes}}'] },
  { cmd: 'change.rollback-steps', args: ['{{changes}}'] },
  { cmd: 'change.pir-list', args: [], list: true },

  { cmd: 'release.list', args: [], list: true },
  { cmd: 'release.stats', args: [] },
  { cmd: 'request.list', args: [], list: true },
  { cmd: 'request.provisioning-tasks', args: ['{{requests}}'] },
  { cmd: 'catalog.list', args: [], list: true },

  { cmd: 'cmdb.get', args: ['{{cis}}'] },
  { cmd: 'cmdb.list', args: [], list: true },
  { cmd: 'cmdb.stats', args: [] },
  { cmd: 'cmdb.types', args: [], list: true },
  { cmd: 'cmdb.type.get', args: ['1'] },
  { cmd: 'cmdb.relationship-types', args: [] },
  { cmd: 'cmdb.relationships.list', args: [], list: true },
  { cmd: 'cmdb.relationships.for-ci', args: ['{{cis}}'] },
  { cmd: 'cmdb.topology', args: ['{{cis}}'] },
  { cmd: 'cmdb.impact', args: ['{{cis}}'] },
  { cmd: 'cmdb.ontology', args: [] },
  { cmd: 'cmdb.capabilities', args: [] },
  { cmd: 'cmdb.changes-history', args: ['{{cis}}'] },
  { cmd: 'cmdb.search', args: ['网关'], list: true },
  { cmd: 'cmdb.reconciliation', args: [], aggregate: true },

  { cmd: 'knowledge.list', args: [], list: true },
  { cmd: 'knowledge.stats', args: [] },
  { cmd: 'knowledge.categories', args: [] },
  { cmd: 'knowledge.search', args: ['变更'], collection: true },
  { cmd: 'knowledge.reviews', args: ['{{knowledge}}'] },
  { cmd: 'knowledge.versions', args: ['{{knowledge}}'] },

  { cmd: 'sla.list', args: [], list: true },
  { cmd: 'sla.stats', args: [] },
  { cmd: 'sla.performance', args: [] },
  { cmd: 'sla.monitoring', args: [] },
  { cmd: 'sla.violations', args: [], list: true },
  { cmd: 'sla.compliance-report', args: ['--startDate', '2026-10-01T00:00:00Z', '--endDate', '2026-10-24T00:00:00Z'], dateOk: true },
  { cmd: 'sla.alert-history', args: [], list: true },

  { cmd: 'workflow.definitions', args: [], list: true },
  { cmd: 'workflow.instances', args: [], list: true },
  { cmd: 'workflow.tasks', args: [], list: true },
  { cmd: 'workflow.instance', args: ['{{workflowInstanceKey}}'] },
  // A list row's own `id` must be fetchable by the detail endpoint; today
  // GET /bpmn/process-instances/:id only resolves `instanceId` (PI-…).
  { cmd: 'workflow.instance', args: ['{{workflowInstances}}'], addressable: true },
  // Cross-endpoint date contract: change.calendar accepts YYYY-MM-DD while
  // sla.compliance-report demands RFC 3339 for the same conceptual field.
  { cmd: 'sla.compliance-report', args: ['--startDate', '2026-10-01', '--endDate', '2026-10-24'], dateInconsistent: true },

  { cmd: 'notification.list', args: [], list: true },
  { cmd: 'notification.deliveries', args: [], list: true },
  { cmd: 'notification.delivery-stats', args: [] },
  { cmd: 'notification.unread-count', args: [] },

  { cmd: 'connector.list', args: [], collection: true },
  { cmd: 'connector.health', args: [] },

  // ---- admin / reliability --------------------------------------------------
  { cmd: 'admin.overview', args: [] },
  { cmd: 'admin.users', args: [], list: true },
  { cmd: 'admin.tenants', args: [], list: true },
  { cmd: 'admin.roles', args: [], list: true },
  { cmd: 'admin.permissions', args: [], collection: true },
  { cmd: 'admin.audit-logs', args: [], list: true },
  { cmd: 'admin.commands', args: ['--limit', '2'], list: true },
  { cmd: 'admin.timers', args: [], list: true },
  { cmd: 'admin.timer-stats', args: [] },
  { cmd: 'admin.msp-status', args: [], capabilityDisabled: true },
  { cmd: 'infra.capabilities', args: [] },
  { cmd: 'infra.ready', args: [] },
  { cmd: 'auth.status', args: [] },
  { cmd: 'auth.session', args: [] },

  // ---- negative probes ------------------------------------------------------
  { cmd: 'ticket.get', args: ['999999'], negative: true, expect: 'not_found' },
  { cmd: 'incident.get', args: ['999999'], negative: true, expect: 'not_found' },
  { cmd: 'problem.get', args: ['999999'], negative: true, expect: 'not_found' },
  { cmd: 'change.get', args: ['999999'], negative: true, expect: 'not_found' },
  { cmd: 'release.get', args: ['999999'], negative: true, expect: 'not_found' },
  { cmd: 'request.get', args: ['999999'], negative: true, expect: 'not_found' },
  { cmd: 'cmdb.get', args: ['999999'], negative: true, expect: 'not_found' },
  { cmd: 'knowledge.get', args: ['999999'], negative: true, expect: 'not_found' },
  { cmd: 'connector.configs', args: ['nonexistent'], negative: true, expect: 'usage' },
  { cmd: 'admin.user', args: ['999999'], negative: true, expect: 'not_found' },

  // ---- protocol assertions (no backend write) --------------------------------
  { cmd: 'ticket.delete', args: ['{{tickets}}'], gate: true },
  // A list payload must be projected AND report meta.projected.
  { cmd: 'ticket.list', args: ['--limit', '2', '--fields', 'id,status,notAField'], shaping: true },
  // A single-object payload must not pretend the flag applied.
  { cmd: 'ticket.get', args: ['{{tickets}}', '--fields', 'id,status'], shapingObject: true },
  { cmd: 'change.delete', args: ['{{changes}}', '--dry-run'], dryRun: true },
  { cmd: 'doesnotexist.thing', args: [], unknown: true },
];

// ---------------------------------------------------------------------------
// Runner
// ---------------------------------------------------------------------------

const hits = new Map();
/** Date-filter acceptance per format, for the cross-endpoint format consistency check. */
const dateFormats = new Map();

function run(argv, env = {}) {
  return new Promise((resolve) => {
    const child = spawn(process.execPath, [bin, ...argv], {
      env: { ...process.env, ITSM_CLI_AGENT_NO_GUARD: '', ...env },
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (c) => (stdout += c));
    child.stderr.on('data', (c) => (stderr += c));
    const timer = setTimeout(() => child.kill('SIGKILL'), 30_000);
    child.on('close', (code) => {
      clearTimeout(timer);
      resolve({ code, stdout, stderr });
    });
  });
}

function envelopeOf(result) {
  const lines = result.stdout.trimEnd().split('\n').filter(Boolean);
  if (lines.length !== 1) return { parseError: `expected 1 stdout line, got ${lines.length}` };
  try {
    return JSON.parse(lines[0]);
  } catch (err) {
    return { parseError: `unparsable: ${err.message}` };
  }
}

// ---------------------------------------------------------------------------
// Invariant checks
// ---------------------------------------------------------------------------

const findings = [];
function flag(severity, check, where, detail, repro) {
  findings.push({ severity, check, where, detail, repro });
}

const SNAKE = /^[a-z0-9]+(?:_[a-z0-9]+)+$/;
const PASCAL = /^[A-Z]/;

/** Recursively collect `{path,key,value}` for every object key in the payload. */
function walk(value, at, visit) {
  if (Array.isArray(value)) {
    value.forEach((v, i) => walk(v, `${at}[${i}]`, visit));
    return;
  }
  if (!value || typeof value !== 'object') return;
  for (const [k, v] of Object.entries(value)) {
    visit(`${at}.${k}`, k, v);
    walk(v, `${at}.${k}`, visit);
  }
}

/**
 * Where a key lives decides whether it is a contract break or data content.
 *
 * A DTO field must be camelCase (hard violation). Inside a *data bag* —
 * `formFields`, `variables`, `before/after` history maps, `typeDistribution` —
 * the keys are business content, so a snake_case key is a warning; except when
 * the key is a persistence column name (`created_at`, `tenant_i_d`, `ci_type_i_d`),
 * which means the storage schema escaped through an untyped map (hard violation:
 * the repository bans snake_case outside DB/adapter boundaries).
 */
const DATA_BAG = /(^|\.)(formFields|conditions|variables|taskVariables|before|after|attributes|metadata|settings|typeDistribution|payload|data|rule|rules|config|configs)(\.|$)/;
const PERSISTENCE_KEY = /(^|_)(at|id|i_d)$/;

function checkNaming(payload, repro, where) {
  const seen = new Set();
  walk(payload, '$', (p, k) => {
    const snake = SNAKE.test(k);
    const pascal = PASCAL.test(k);
    if (!snake && !pascal) return;
    const bag = DATA_BAG.test(p);
    const leak = bag && (PERSISTENCE_KEY.test(k) || /^is_/.test(k));
    const cls = pascal ? 'PascalCase' : 'snake_case';
    const dedupe = `${bag ? (leak ? 'schema-leak' : 'data-key') : cls}:${k}`;
    if (seen.has(dedupe)) return;
    seen.add(dedupe);
    if (leak) {
      flag('violation', 'schema-leak', `${where} @ ${p}`,
        `data map key "${k}" is a persistence column name — the storage schema leaked into the API through an untyped map`, repro);
    } else if (bag) {
      flag('warning', 'data-key', `${where} @ ${p}`,
        `data content key "${k}" is ${cls}; business keys should be normalized to camelCase at the adapter boundary`, repro);
    } else {
      flag('violation', cls, `${where} @ ${p}`, `field "${k}" uses ${cls}; API fields must be camelCase`, repro);
    }
  });
}

function checkZeroTimes(payload, repro, where) {
  const seen = new Set();
  walk(payload, '$', (p, k, v) => {
    if (typeof v === 'string' && /^0001-01-01T00:00:00/.test(v)) {
      const dedupe = k.replace(/^.*\./, '');
      if (seen.has(dedupe)) return;
      seen.add(dedupe);
      flag('violation', 'zero-time', `${where} @ ${p}`, `"${k}" renders the Go zero time "${v}"; unset times must be null or omitted`, repro);
    }
  });
}

const LIST_KEYS = ['items', 'total', 'page', 'pageSize', 'totalPages'];

/**
 * A non-paginated collection (`{items,total}` without page metadata) is a
 * consistency defect, not a contract break: the caller can still read the rows,
 * but cannot tell a truncated result from a complete one. Paginated endpoints
 * (list probes) must carry the full envelope; endpoints that only hand back
 * rows are checked here.
 */
function checkCollection(data, repro, where) {
  if (Array.isArray(data)) {
    flag('violation', 'list-envelope', where, 'collection returned a bare array — no `items`, no `total`, truncation undetectable', repro);
    return;
  }
  if (!data || typeof data !== 'object') {
    flag('violation', 'list-envelope', where, `collection returned ${JSON.stringify(data)}`, repro);
    return;
  }
  if (!Array.isArray(data.items)) {
    const alias = Object.keys(data).filter((k) => ['users', 'permissions', 'tickets', 'records', 'rows', 'list'].includes(k));
    flag('violation', 'list-envelope', where, `no \`items\` key; collection arrives as ${alias.join(', ') || Object.keys(data).join(', ')}`, repro);
    return;
  }
  const missing = LIST_KEYS.filter((k) => !(k in data));
  if (missing.length) {
    flag('warning', 'list-envelope', where, `collection carries only ${Object.keys(data).join(', ')} — missing ${missing.join(', ')} (caller cannot detect truncation)`, repro);
  }
}

/** Recognise the backend list envelope ({items,total,page,pageSize,totalPages}) or a bare array. */

function checkListEnvelope(data, repro, where) {
  if (Array.isArray(data)) {
    flag('violation', 'list-envelope', where, 'paginated list returned a bare array; contract is {items,total,page,pageSize,totalPages}', repro);
    return;
  }
  if (!data || typeof data !== 'object') {
    flag('violation', 'list-envelope', where, `paginated list returned ${JSON.stringify(data)}`, repro);
    return;
  }
  const missing = LIST_KEYS.filter((k) => !(k in data));
  const aliases = Object.keys(data).filter((k) => !LIST_KEYS.includes(k) && ['users', 'tickets', 'incidents', 'records', 'list', 'pagination', 'rows', 'data'].includes(k));
  if (missing.length) {
    flag('violation', 'list-envelope', where, `missing ${missing.join(', ')}; extra keys: ${Object.keys(data).join(', ') || '(none)'}`, repro);
  }
  if (aliases.length) {
    flag('violation', 'list-envelope', where, `uses non-standard collection keys: ${aliases.join(', ')}`, repro);
  }
  if (Array.isArray(data.items)) {
    // Identity reference sanity: an ID of 0 together with a populated name
    // means the join lost the row — callers cannot resolve the owner.
    data.items.forEach((it, i) => {
      if (!it || typeof it !== 'object') return;
      for (const [k, v] of Object.entries(it)) {
        if (!/Id$/.test(k) || typeof v !== 'number') continue;
        const nameKey = `${k.slice(0, -2)}Name`;
        if (v === 0 && typeof it[nameKey] === 'string' && it[nameKey].length) {
          flag('warning', 'identity-ref', `${where}.items[${i}].${k}`, `id 0 while "${nameKey}" = "${it[nameKey]}" — dangling owner reference`, repro);
        }
        if (v === 0 && k === 'createdBy') {
          flag('warning', 'identity-ref', `${where}.items[${i}].${k}`, 'createdBy 0 loses the audit actor', repro);
        }
      }
    });
  }
}

function checkError(env, code, expect, repro, where) {
  if (env.ok !== false) {
    flag('violation', 'negative-path', where, `expected the call to be refused, got ok:true (exit ${code})`, repro);
    return;
  }
  const kind = env.error?.kind;
  if (expect && kind !== expect) {
    flag('warning', 'error-mapping', where, `expected kind=${expect}, got kind=${kind} (http ${env.error?.httpStatus}, business ${env.error?.businessCode}) message=${JSON.stringify(env.error?.message)}`, repro);
  }
  if (!env.error?.message) flag('violation', 'error-mapping', where, 'error envelope without a message', repro);
}

// ---------------------------------------------------------------------------
// Run seeds
// ---------------------------------------------------------------------------

const ids = new Map();
const started = Date.now();

for (const [label, cmd, args, field = 'id'] of SEEDS) {
  if (!specs.has(cmd)) {
    flag('warning', 'probe', cmd, 'probe command not in registry (audit table is stale)', `itsm --json ${cmd}`);
    continue;
  }
  const argv = ['--json', ...cmd.split('.'), ...args];
  const res = await run(argv);
  const env = envelopeOf(res);
  if (env.parseError || env.ok !== true) {
    flag('warning', 'probe', cmd, `seed did not succeed: ${env.parseError ?? JSON.stringify(env.error)}`, `itsm --json ${argv.slice(1).join(' ')}`);
    continue;
  }
  const item = Array.isArray(env.data?.items) ? env.data.items[0] : undefined;
  const id = item && typeof item === 'object' ? (item[field] ?? (field === 'id' ? item.ID : undefined)) : undefined;
  if (id === undefined || id === null) {
    flag('warning', 'probe', cmd, 'seed returned no items — dependent probes skipped', `itsm --json ${argv.slice(1).join(' ')}`);
    continue;
  }
  ids.set(label, String(id));
}

function substitute(args) {
  const out = [];
  for (const a of args) {
    const m = /^\{\{(\w+)\}\}$/.exec(a);
    if (!m) {
      out.push(a);
      continue;
    }
    const v = ids.get(m[1]);
    if (v === undefined) return null;
    out.push(v);
  }
  return out;
}

// ---------------------------------------------------------------------------
// Run probes
// ---------------------------------------------------------------------------

let executed = 0;
let skipped = 0;

for (const probe of PROBES) {
  const name = probe.cmd;
  const repro = `itsm --json ${name.split('.').join(' ')} ${probe.args.join(' ')}`.trim();

  if (name === 'doesnotexist.thing') {
    const res = await run(['--json', 'doesnotexist', 'thing']);
    const env = envelopeOf(res);
    executed++;
    if (res.code !== 2 || env.ok !== false) {
      flag('violation', 'protocol', 'unknown command', `expected exit 2 + usage error, got exit ${res.code}`, 'itsm --json doesnotexist thing');
    }
    continue;
  }

  if (!specs.has(name)) {
    skipped++;
    flag('warning', 'probe', name, 'probe command not in registry (audit table is stale)', repro);
    continue;
  }

  const args = substitute(probe.args);
  if (args === null) {
    skipped++;
    continue;
  }

  const argv = ['--json', ...name.split('.'), ...args];
  const res = await run(argv);
  const env = envelopeOf(res);
  executed++;

  if (env.parseError) {
    flag('violation', 'protocol', name, env.parseError, repro);
    continue;
  }

  // ---- protocol / shaping ---------------------------------------------------
  if (probe.shaping) {
    const meta = env.meta ?? {};
    const items = env.data?.items;
    if (env.ok !== true || !Array.isArray(items) || !Array.isArray(meta.projected)) {
      flag('violation', 'shaping', name, `expected a projected list + meta.projected, got ${JSON.stringify(env).slice(0, 240)}`, repro);
    } else if (meta.projected.length !== 3) {
      flag('violation', 'shaping', name, `meta.projected must echo the requested fields, got ${JSON.stringify(meta.projected)}`, repro);
    } else if (items.length && !('notAField' in items[0])) {
      // An unknown field must surface as an explicit null, not vanish: a caller
      // asking for a field that does not exist must be able to see the miss.
      flag('violation', 'shaping', name, 'unknown requested field vanished instead of rendering null', repro);
    }
    continue;
  }
  if (probe.shapingObject) {
    const reason = env.meta?.shapingSkipped;
    if (env.ok !== true || typeof reason !== 'string' || !reason.length) {
      flag('violation', 'shaping', name, `a non-list payload must report meta.shapingSkipped, got meta=${JSON.stringify(env.meta)}`, repro);
    }
    continue;
  }
  if (probe.gate) {
    if (res.code !== 10 || env.error?.kind !== 'confirmation_required') {
      flag('violation', 'risk-gate', name, `high-risk write must stop at exit 10 (got ${res.code}, kind=${env.error?.kind})`, repro);
    }
    continue;
  }
  if (probe.dryRun) {
    if (env.ok !== true || env.data?.dryRun !== true) {
      flag('violation', 'dry-run', name, `--dry-run must preview without sending (got ${JSON.stringify(env).slice(0, 200)})`, repro);
    }
    continue;
  }

  // ---- contract probes -------------------------------------------------------
  if (probe.dateOk && env.ok === true) dateFormats.set('rfc3339', true);
  if (probe.addressable) {
    if (env.ok !== true) {
      flag('violation', 'addressable-id', name,
        `the id this list returns (${args[0]}) cannot be fetched by the detail endpoint: ${env.error?.message}` +
        ' — GET detail must accept the key the list hands out (or the list must only expose the addressable key)',
        repro);
    }
    continue;
  }
  if (probe.dateInconsistent) {
    dateFormats.set('dateOnly', env.ok === true ? true : (env.error?.message ?? 'rejected'));
    continue;
  }

  // ---- negative -------------------------------------------------------------
  if (probe.capabilityDisabled) {
    if (env.ok !== false) {
      flag('violation', 'capability-state', name, 'a capability disabled in this deployment must be refused, not returned as success', repro);
    } else if (!env.error.capability) {
      // 拒绝但没有给出「哪个能力、什么状态」——调用方只能从 kind 猜测，
      // 把「部署没启用」当成资源不存在或后端故障。
      flag('warning', 'capability-state', name, `disabled capability reported as kind=${env.error.kind} (http ${env.error.httpStatus}) without an explicit capability/state field — contract wants {capability, state} (disabled/unconfigured/unready), not just a kind`, repro);
    } // else: 错误信封显式携带 {capability, capabilityState}，符合契约
    continue;
  }
  if (probe.negative) {
    checkError(env, res.code, probe.expect, repro, name);
    continue;
  }

  // ---- positive -------------------------------------------------------------
  if (env.ok !== true) {
    flag(probe.list || probe.aggregate || probe.collection ? 'violation' : 'warning', 'read-failure', name, `expected ok:true, got ${JSON.stringify(env.error)}`, repro);
    continue;
  }

  checkNaming(env.data, repro, name);
  checkZeroTimes(env.data, repro, name);
  if (probe.list) checkListEnvelope(env.data, repro, name);
  if (probe.collection) checkCollection(env.data, repro, name);
  if (!('data' in env) || env.data === null || env.data === undefined) {
    // `ok:true` must always carry an explicit payload: `data:null` and a
    // missing `data` key are indistinguishable to a caller, and both read as
    // "nothing here" when the real answer may be "the handler bailed out".
    flag('violation', 'success-shape', name, 'ok:true without a data payload (key missing or null) — caller cannot tell an empty result from a bug', repro);
  }

  // HTTP/business status honesty on success paths: none of these should be
  // classified as an error by the backend (business code inside HTTP 200).
  if (env.meta?.httpStatus && env.meta.httpStatus >= 400) {
    flag('warning', 'error-mapping', name, `ok:true but httpStatus ${env.meta.httpStatus}`, repro);
  }
}

// ---- unauthenticated exit code ---------------------------------------------
{
  const res = await run(['--json', 'ticket', 'list', '--limit', '1'], {
    ITSM_CREDENTIALS_FILE: path.join(root, '.audit-no-creds'),
    ITSM_TOKEN: 'not-a-real-token',
  });
  const env = envelopeOf(res);
  executed++;
  if (res.code !== 3 || env.error?.kind !== 'not_authenticated') {
    flag('violation', 'protocol', 'unauthenticated request', `expected exit 3 + not_authenticated, got exit ${res.code} kind=${env.error?.kind}`, 'ITSM_TOKEN=bogus itsm --json ticket list --limit 1');
  }
}

// Cross-endpoint consistency: the same conceptual filter (a date range) must
// not require two different formats depending on which domain you call.
if (dateFormats.get('rfc3339') === true && typeof dateFormats.get('dateOnly') === 'string') {
  flag('warning', 'date-format', 'sla.compliance-report vs change.calendar',
    `date filters are format-incompatible across endpoints: this one rejects a plain date (${dateFormats.get('dateOnly')}) while change.calendar accepts the same "2026-10-01" value`,
    'itsm --json sla compliance-report --startDate 2026-10-01 --endDate 2026-10-24');
}

const elapsed = Date.now() - started;

// ---------------------------------------------------------------------------
// Report
// ---------------------------------------------------------------------------

const byCheck = {};
for (const f of findings) byCheck[f.check] = (byCheck[f.check] ?? 0) + 1;

const stamp = new Date().toISOString().replace(/[-:]/g, '').slice(0, 8);
const outDir = path.resolve(root, '..', 'output');
const outFile = path.join(outDir, `itsm-cli-product-audit-${stamp}.md`);

if (asJson) {
  console.log(JSON.stringify({ executed, skipped, elapsedMs: elapsed, findings, byCheck }, null, 2));
} else {
  const lines = [];
  lines.push(`# itsm-cli / itsm-mcp 产品功能点审查（${new Date().toISOString().slice(0, 19).replace('T', ' ')}）`);
  lines.push('');
  lines.push(`- 入口：\`bin/itsm.js --json\`（真实 agent 协议），后端 \`${process.env.ITSM_BASE_URL ?? 'http://localhost:8090'}\``);
  lines.push(`- 探测：执行 ${executed} 条（跳过 ${skipped} 条），耗时 ${(elapsed / 1000).toFixed(1)}s`);
  lines.push(`- 断言维度：${Object.keys(byCheck).join(' / ')}`);
  lines.push('');
  lines.push(`## 结论：${findings.filter((f) => f.severity === 'violation').length} 项违规 / ${findings.filter((f) => f.severity === 'warning').length} 项告警`);
  lines.push('');
  const order = ['violation', 'warning'];
  for (const sev of order) {
    const group = findings.filter((f) => f.severity === sev);
    if (!group.length) continue;
    lines.push(`## ${sev === 'violation' ? '违规（必须修）' : '告警（需判定）'}（${group.length}）`);
    lines.push('');
    const byName = new Map();
    for (const f of group) {
      const key = `${f.check}｜${f.detail}`;
      if (!byName.has(key)) byName.set(key, { ...f, where: [] });
      byName.get(key).where.push(f.where);
    }
    for (const { check, detail, where, repro } of byName.values()) {
      lines.push(`### [${check}] ${detail}`);
      lines.push('');
      lines.push(`- 位置：${[...new Set(where)].slice(0, 6).join(', ')}`);
      lines.push(`- 复现：\`${repro}\``);
      lines.push('');
    }
  }
  if (!findings.length) lines.push('全部断言通过。');
  const report = lines.join('\n');
  mkdirSync(outDir, { recursive: true });
  writeFileSync(outFile, report);
  console.log(report);
  console.log(`\n---\nreport: ${outFile}`);
}

if (strict && findings.some((f) => f.severity === 'violation')) process.exit(1);
