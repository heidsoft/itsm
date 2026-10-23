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
import { readFileSync, readdirSync } from 'fs';
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
