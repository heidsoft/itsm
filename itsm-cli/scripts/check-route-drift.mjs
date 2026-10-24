#!/usr/bin/env node
/**
 * Route drift check: compares the agent command surface's declared endpoints
 * (src/agent/commands/*.ts, aggregated in the dist registry) against the
 * routes statically extracted from itsm-backend/router/*.go.
 *
 * The backend is the single source of truth for the URL contract; this script
 * fails when the CLI references a route that is not registered (or registered
 * with a different method).
 *
 * Usage: node scripts/check-route-drift.mjs [--quiet]
 */
import { existsSync, readFileSync, readdirSync } from 'fs';
import { fileURLToPath } from 'url';
import path from 'path';

const here = path.dirname(fileURLToPath(import.meta.url));
const cliRoot = path.resolve(here, '..');
const backendRouterDir = process.env['ITSM_ROUTER_DIR'] ?? path.resolve(cliRoot, '../itsm-backend/router');

// --- 1. CLI endpoints -------------------------------------------------------
const { registry } = await import(path.join(cliRoot, 'dist/agent/registry.js'));
const cliEndpoints = new Set(); // "METHOD /api/v1/path"
for (const spec of registry()) {
  for (const e of spec.endpoints) {
    const m = e.match(/^([A-Z]+) (.+)$/);
    if (!m) continue;
    const [, method, route] = m;
    cliEndpoints.add(`${method} ${normalize(route)}`);
  }
}

function normalize(route) {
  return route.replace(/\/:([A-Za-z0-9_]+)/g, '/{}');
}

// --- 2. Backend route extraction (variable → group prefix tracking) ---------
const backend = new Set(); // "METHOD /api/v1/..." (normalized, {} params)
const aliasesWanted = new Set(['GET /api/v1', 'ANY /api/v1/*']); // escape hatch sits outside the check

const verbToken = '(GET|POST|PUT|PATCH|DELETE)';
const scanDirs = [
  backendRouterDir,
  process.env['ITSM_HANDLERS_DIR'] ?? path.resolve(cliRoot, '../itsm-backend/handlers'),
];
const handlersGoFiles = [];
for (const dir of scanDirs) {
  const walk = (d) => {
    for (const entry of readdirSync(d, { withFileTypes: true })) {
      const p = path.join(d, entry.name);
      if (entry.isDirectory()) { walk(p); continue; }
      if (entry.name.endsWith('.go') && !entry.name.endsWith('_test.go')) handlersGoFiles.push(p);
    }
  };
  walk(dir);
}

for (const file of handlersGoFiles) {
  const lines = readFileSync(file, 'utf-8').split('\n');
  // variable → absolute prefix, chained through `v := base.Group("/x")`; base `r`/`router` root = ""
  const prefixes = new Map();
  const prefixOf = (v) => prefixes.get(v) ?? '';
  for (const line of lines) {
    const routeCall = line.trim().match(/^(\w+)\.(GET|POST|PUT|PATCH|DELETE|Any)\("([^"]*)"/);
    if (routeCall) {
      const [, v, method, routePath] = routeCall;
      const base = prefixOf(v);
      const full = `${base}${routePath}`.replace(/\/+/g, '/');
      const abs = full.startsWith('/api/v1') ? full : `/api/v1${full}`;
      if (!abs.startsWith('/api/v1/') && abs !== '/api/v1') continue;
      if (method === 'Any') continue; // websocket / any-handler: not part of the REST contract check
      backend.add(`${method.toUpperCase()} ${normalize(abs)}`);
      continue;
    }
    const group = line.trim().match(/^(\w+)\s*:?=\s*(\w+)\.Group\("([^"]*)"\)/);
    if (group) {
      const [, v, base, prefix] = group;
      prefixes.set(v, `${prefixOf(base)}${prefix}`);
    }
  }
}

// --- 3. Compare -------------------------------------------------------------
const missing = [];
for (const ep of cliEndpoints) {
  if (aliasesWanted.has(ep)) continue;
  if (backend.has(ep)) continue;
  missing.push(ep);
}

if (missing.length) {
  console.error(`✗ Route drift: ${missing.length} CLI endpoint(s) not found in itsm-backend/router:`);
  for (const m of missing.sort()) console.error(`  ${m}`);
  console.error('\nFix the CLI command endpoints (or the backend route), then re-run.');
  process.exit(1);
}
console.log(`✓ Route drift check passed: ${cliEndpoints.size} CLI endpoints all match backend registrations.`);
console.log(`  (backend scan covered ${backend.size} routes across ${handlersGoFiles.length} Go files)`);

// --- 4. Reverse direction: backend routes no command reaches -----------------
//
// The forward check can only fail when the CLI names a route the backend lacks
// — the opposite direction was never tested, so an entire backend family could
// ship with zero CLI coverage and every gate stayed green. The whole cloud /
// discovery surface (27 routes: cloud accounts, services, resources, discovery
// sources/jobs/results/providers/health) was exactly that: reachable from the
// web UI, invisible to `itsm` and to every MCP client.
//
// Absolute coverage is not the goal — 68% of routes are still uncovered and
// closing that is a product decision, not a build gate. What must never happen
// silently is *regression*, so the uncovered count is ratcheted: it may fall,
// and may only rise in a commit that raises this baseline and states why.
const baselineFile = process.env['ITSM_ROUTE_COVERAGE_BASELINE'] ?? path.join(here, 'route-coverage-baseline.txt');
const uncovered = [...backend].filter((ep) => !cliEndpoints.has(ep) && !aliasesWanted.has(ep)).sort();

const baselineRaw = existsSync(baselineFile) ? readFileSync(baselineFile, 'utf-8') : '';
const baselineMatch = baselineRaw.match(/^uncovered=(\d+)/m);
if (!baselineMatch) {
  console.error(`✗ 缺少或未解析到覆盖率基线：${baselineFile}`);
  console.error('  需要形如 `uncovered=<n>  # 理由` 的一行。空基线会让反向检查静默失效。');
  process.exit(1);
}
const want = Number(baselineMatch[1]);
if (!Number.isInteger(want)) {
  console.error(`✗ 基线 uncovered 不是整数：${baselineMatch[1]}`);
  process.exit(1);
}

const byFamily = new Map();
for (const ep of uncovered) {
  const family = ep.split(' ')[1].split('/').slice(0, 4).join('/');
  byFamily.set(family, (byFamily.get(family) ?? 0) + 1);
}
const quiet = process.argv.includes('--quiet');
if (!quiet) {
  console.log(`  未覆盖 ${uncovered.length} / ${backend.size} 条后端路由（基线 ${want}），按域：`);
  for (const [family, n] of [...byFamily.entries()].sort((a, b) => b[1] - a[1]).slice(0, 10)) {
    console.log(`    ${String(n).padStart(4)}  ${family}`);
  }
}

if (uncovered.length > want) {
  console.error(`✗ 路由覆盖率回退：未覆盖 ${uncovered.length} > 基线 ${want}（新增 +${uncovered.length - want}）`);
  console.error('  新增后端路由必须同时接入 CLI 命令；确属有意不接入时，在同一 commit 上调基线并写明理由。');
  for (const ep of uncovered.slice(0, 40)) console.error(`  ${ep}`);
  process.exit(1);
}
if (uncovered.length < want) {
  console.log(`  未覆盖数已下降（${uncovered.length} < 基线 ${want}），建议同步下调基线以保持棘轮有效。`);
}
console.log(`✓ Reverse coverage check passed: ${uncovered.length} uncovered ≤ baseline ${want}.`);
