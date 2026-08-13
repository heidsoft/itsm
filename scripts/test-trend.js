#!/usr/bin/env node
/**
 * scripts/test-trend.js
 *
 * Companion to PR-0.5 (Test Improvement Plan). Reads the canonical
 * test artifacts produced by backend-ci, frontend-ci and ga-gate,
 * diffs each against the baseline snapshot in
 * `docs/testing/coverage-baseline.json`, and emits a single Markdown
 * trend that the CI can post as a PR comment.
 *
 * Inputs (any of which may be missing — the script degrades gracefully):
 *   --junit <path>          junit.xml (Playwright or Jest). May be repeated.
 *   --go-coverage <path>    coverage.out (Go coverprofile)
 *   --jest-coverage <path>  coverage-summary.json (Jest)
 *   --baseline <path>       baseline JSON (default: docs/testing/coverage-baseline.json)
 *   --out <path>            output Markdown (default: stdout)
 *   --pr <url>              optional PR URL to embed
 *
 * Output:
 *   Markdown with three sections — Backend, Frontend (Jest), Frontend (E2E) —
 *   each rendered as a table that includes pass / fail / skip / coverage %
 *   and a delta vs the baseline.
 *
 * Used by:
 *   - .github/workflows/backend-ci.yml     (Go tests + coverage.out trend)
 *   - .github/workflows/frontend-ci.yml    (Jest results + junit trend)
 *   - .github/workflows/ga-gate.yml        (smoke + e2e junit trend)
 */

'use strict';

const fs = require('fs');
const path = require('path');

// ------------------------------------------------------------------
// argv
// ------------------------------------------------------------------
function parseArgs(argv) {
  const out = {
    junit: [],
    goCoverage: null,
    jestCoverage: null,
    baseline: path.resolve(
      'docs/testing/coverage-baseline.json'
    ),
    out: null,
    pr: '',
  };
  const flag = (name) => {
    const i = argv.indexOf(name);
    return i >= 0 ? argv[i + 1] : null;
  };
  let i = 0;
  while (i < argv.length) {
    const a = argv[i];
    if (a === '--junit') {
      out.junit.push(argv[i + 1]);
      i += 2;
      continue;
    }
    if (a === '--go-coverage') {
      out.goCoverage = argv[i + 1];
    } else if (a === '--jest-coverage') {
      out.jestCoverage = argv[i + 1];
    } else if (a === '--baseline') {
      out.baseline = argv[i + 1];
    } else if (a === '--out') {
      out.out = argv[i + 1];
    } else if (a === '--pr') {
      out.pr = argv[i + 1];
    }
    i += 1;
  }
  return out;
}

const args = parseArgs(process.argv.slice(2));

// ------------------------------------------------------------------
// helpers
// ------------------------------------------------------------------
function exists(p) {
  try {
    fs.accessSync(p, fs.constants.R_OK);
    return true;
  } catch (_e) {
    return false;
  }
}

function readJson(p) {
  try {
    return JSON.parse(fs.readFileSync(p, 'utf8'));
  } catch (e) {
    throw new Error(`failed to read JSON: ${p}: ${e.message}`);
  }
}

function readText(p) {
  return fs.readFileSync(p, 'utf8');
}

function fmtDelta(curr, base) {
  if (typeof base !== 'number') return '—';
  const d = +(curr - base).toFixed(2);
  if (Math.abs(d) < 0.01) return '⚪ ±0';
  return d > 0 ? `🟢 ↑${d}` : `🔴 ↓${Math.abs(d)}`;
}

function pct(x) {
  if (typeof x !== 'number') return '—';
  return x.toFixed(2) + '%';
}

// ------------------------------------------------------------------
// junit.xml parser (light, regex-based — works for Playwright & Jest)
// ------------------------------------------------------------------
function parseJunit(xmlText) {
  // Aggregate per file then sum. Avoid XML libs to keep the script
  // dependency-free (the repo is otherwise YAML/Yaml-only in CI).
  let tests = 0;
  let failures = 0;
  let errors = 0;
  let skipped = 0;
  let duration = 0;
  const re = /<testsuite\b([^>]*)>/g;
  const attr = (s, n) => {
    const m = new RegExp(`${n}="([^"]*)"`).exec(s);
    return m ? Number(m[1]) || m[1] : null;
  };
  let m;
  while ((m = re.exec(xmlText))) {
    const attrs = m[1];
    tests += Number(attr(attrs, 'tests')) || 0;
    failures += Number(attr(attrs, 'failures')) || 0;
    errors += Number(attr(attrs, 'errors')) || 0;
    skipped += Number(attr(attrs, 'skipped')) || 0;
    duration += Number(attr(attrs, 'time')) || 0;
  }
  return { tests, failures, errors, skipped, duration };
}

// ------------------------------------------------------------------
// go coverage parser  (matches scripts/coverage-summarize.js contract)
// ------------------------------------------------------------------
function parseGoCoverage(filePath) {
  if (!exists(filePath)) return { pct: null, packages: [] };
  const text = fs.readFileSync(filePath, 'utf8');
  const pkgMap = new Map(); // pkg -> {stmt, hit}
  for (const line of text.split('\n')) {
    if (!line || line.startsWith('mode:')) continue;
    // Use lastIndexOf(':') so paths containing ':' (rare but legal on
    // some platforms) still parse. Same trick as coverage-summarize.js.
    const colon = line.lastIndexOf(':');
    if (colon === -1) continue;
    const file = line.slice(0, colon);
    const rest = line.slice(colon + 1);
    const parts = rest.trim().split(/\s+/);
    if (parts.length < 3) continue;
    const stmt = Number(parts[parts.length - 2]);
    const count = Number(parts[parts.length - 1]);
    if (!Number.isFinite(stmt) || stmt <= 0) continue;
    // Strip itsm-backend prefix to align with coverage-summarize.js output
    let rel = file;
    const m = file.match(/(itsm-backend[\\/].+)$/);
    if (m) rel = m[1];
    const segments = rel.split(/[\\/]/);
    if (segments[0] === 'itsm-backend' && segments.length >= 3) {
      segments.pop();
    }
    const pkg = segments.join('/');
    if (!pkgMap.has(pkg)) {
      pkgMap.set(pkg, { stmt: 0, hit: 0 });
    }
    const acc = pkgMap.get(pkg);
    acc.stmt += stmt;
    if (count > 0) acc.hit += stmt;
  }
  let totalStmt = 0;
  let totalHit = 0;
  for (const v of pkgMap.values()) {
    totalStmt += v.stmt;
    totalHit += v.hit;
  }
  const pctNum = totalStmt === 0 ? null : (totalHit / totalStmt) * 100;
  return { pct: pctNum, packages: pkgMap };
}

// ------------------------------------------------------------------
// jest coverage parser
// ------------------------------------------------------------------
function parseJestCoverage(filePath) {
  if (!exists(filePath)) return null;
  const j = readJson(filePath);
  const t = j.total || {};
  return {
    lines: t.lines?.pct ?? null,
    branches: t.branches?.pct ?? null,
    functions: t.functions?.pct ?? null,
    statements: t.statements?.pct ?? null,
  };
}

// ------------------------------------------------------------------
// baseline
// ------------------------------------------------------------------
function readBaseline(p) {
  if (!exists(p)) {
    return { go: null, jest: null, ts: '' };
  }
  return readJson(p);
}

// ------------------------------------------------------------------
// markdown render
// ------------------------------------------------------------------
function render(baseline, goCov, jestCov, junits, pr, banner) {
  const lines = [];
  lines.push('## 📈 Test Trend');
  if (banner) lines.push(`> ${banner}`);
  lines.push('');
  lines.push(`_Generated: ${new Date().toISOString()}_`);
  if (pr) lines.push(`_PR: ${pr}_`);
  lines.push('');

  // Backend coverage
  lines.push('### Backend (Go)');
  if (typeof goCov.pct === 'number') {
    lines.push('');
    lines.push('| metric | value | Δ vs baseline |');
    lines.push('| --- | --- | --- |');
    lines.push(
      `| line coverage (svc+ctl) | ${pct(goCov.pct)} | ${fmtDelta(goCov.pct, baseline.go)} |`
    );
    if (goCov.packages && goCov.packages.size) {
      const sorted = [...goCov.packages.entries()]
        .map(([k, v]) => ({
          pkg: k,
          pct: v.stmt === 0 ? null : (v.hit / v.stmt) * 100,
        }))
        .filter((x) => typeof x.pct === 'number')
        .sort((a, b) => a.pct - b.pct)
        .slice(0, 5);
      if (sorted.length) {
        lines.push('');
        lines.push('**Bottom 5 packages (lowest coverage):**');
        for (const s of sorted) {
          lines.push(`- \`${s.pkg}\` — ${pct(s.pct)}`);
        }
      }
    }
  } else {
    lines.push('');
    lines.push('_no coverage profile found_');
  }
  lines.push('');

  // Frontend Jest coverage
  lines.push('### Frontend (Jest)');
  if (jestCov) {
    lines.push('');
    lines.push('| metric | value | Δ vs baseline |');
    lines.push('| --- | --- | --- |');
    const colMap = [
      ['lines', baseline.jest],
      ['branches', null],
      ['functions', null],
      ['statements', null],
    ];
    for (const [key, base] of colMap) {
      const v = jestCov[key];
      if (typeof v === 'number') {
        lines.push(
          `| ${key} | ${pct(v)} | ${key === 'lines' ? fmtDelta(v, base) : '—'} |`
        );
      }
    }
  } else {
    lines.push('');
    lines.push('_no Jest coverage profile found_');
  }
  lines.push('');

  // Junit aggregation
  lines.push('### Test suites (junit.xml)');
  if (junits.length === 0) {
    lines.push('');
    lines.push('_no junit.xml supplied_');
  } else {
    let totalT = 0;
    let totalF = 0;
    let totalE = 0;
    let totalS = 0;
    let totalD = 0;
    for (const j of junits) {
      totalT += j.tests;
      totalF += j.failures;
      totalE += j.errors;
      totalS += j.skipped;
      totalD += j.duration;
    }
    lines.push('');
    lines.push('| metric | value |');
    lines.push('| --- | --- |');
    lines.push(`| tests | ${totalT} |`);
    lines.push(`| failures | ${totalF} |`);
    lines.push(`| errors | ${totalE} |`);
    lines.push(`| skipped | ${totalS} |`);
    lines.push(`| duration (s) | ${totalD.toFixed(2)} |`);
  }

  return lines.join('\n');
}

// ------------------------------------------------------------------
// main
// ------------------------------------------------------------------
function main() {
  const junitReports = args.junit
    .filter(exists)
    .map((p) => parseJunit(readText(p)));
  const goCov = parseGoCoverage(args.goCoverage);
  const jestCov = parseJestCoverage(args.jestCoverage);
  const baseline = readBaseline(args.baseline);

  const banner =
    `baseline ${baseline.ts || 'unknown'} captured; ` +
    `delta = current - baseline`;

  const md = render(
    baseline,
    goCov,
    jestCov,
    junitReports,
    args.pr,
    banner
  );
  if (args.out) {
    fs.writeFileSync(args.out, md);
    console.error(`[test-trend] wrote ${args.out}`);
  } else {
    process.stdout.write(md);
  }
}

try {
  main();
} catch (e) {
  console.error(`[test-trend] ${e.message}`);
  // Degrade to non-zero exit so CI can detect a malformed trend
  process.exit(2);
}
