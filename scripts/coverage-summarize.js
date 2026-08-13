#!/usr/bin/env node
/**
 * coverage-summarize.js
 *
 * Parse backend Go coverage (.out) and frontend Jest coverage (coverage-summary.json)
 * and emit a unified Markdown report at the project root's `coverage-summary.md`.
 *
 * Usage:
 *   node scripts/coverage-summarize.js \
 *     --go itsm-backend/coverage.out \
 *     --jest itsm-frontend/coverage/coverage-summary.json \
 *     --out coverage-summary.md
 *
 * If --go or --jest is missing, that side is reported as "n/a" instead of failing.
 *
 * Output sections:
 *   1. Combined totals table
 *   2. Backend (Go): top 10 worst packages by line coverage %
 *   3. Frontend (Jest): top 10 worst files by line coverage %
 *   4. Delta vs. baseline file (--baseline path), if exists
 */

'use strict';

const fs = require('node:fs');
const path = require('node:path');

function parseArgs(argv) {
  const out = { go: null, jest: null, out: null, baseline: null, projectRoot: process.cwd() };
  for (let i = 2; i < argv.length; i += 2) {
    const k = argv[i];
    const v = argv[i + 1];
    if (k === '--go') out.go = v;
    else if (k === '--jest') out.jest = v;
    else if (k === '--out') out.out = v;
    else if (k === '--baseline') out.baseline = v;
    else if (k === '--project-root') out.projectRoot = v;
  }
  return out;
}

/**
 * Parse a Go `go test -coverprofile` output file.
 * Format: lines like `package/path/file.go:start.col,end.col statements count`
 * We aggregate per package using:
 *   per-file statements = (covered) / (total) percentage isn't reliable because
 *   statement blocks are split; instead we approximate "line coverage %" by
 *   aggregating covered/total statement blocks across the package.
 */
function parseGoCoverage(filePath) {
  if (!filePath || !fs.existsSync(filePath)) return null;
  const raw = fs.readFileSync(filePath, 'utf8');
  const lines = raw.split('\n');
  const perFile = new Map(); // file -> {covered, total}
  for (const line of lines) {
    // Mode-prefixed profiles include "mode: set" header
    if (!line || line.startsWith('mode:')) continue;
    const colon = line.lastIndexOf(':');
    if (colon === -1) continue;
    const file = line.slice(0, colon);
    const rest = line.slice(colon + 1);
    // block: "start.col,end.col numStmt count" — only treat count > 0 as covered
    const parts = rest.trim().split(/\s+/);
    if (parts.length < 3) continue;
    const numStmt = Number(parts[parts.length - 2]);
    const count = Number(parts[parts.length - 1]);
    if (!Number.isFinite(numStmt) || numStmt <= 0) continue;
    const slot = perFile.get(file) || { covered: 0, total: 0 };
    slot.total += numStmt;
    if (count > 0) slot.covered += numStmt;
    perFile.set(file, slot);
  }
  // Aggregate by package: group files by directory (relative to itsm-backend root)
  const perPkg = new Map();
  for (const [file, agg] of perFile.entries()) {
    // Try to strip itsm-backend prefix, else take the path as-is
    let rel = file;
    const m = file.match(/(itsm-backend\/.+)$/);
    if (m) rel = m[1];
    const segments = rel.split('/');
    // For itsm-backend/foo/bar/baz.go -> pkg=itsm-backend/foo/bar (without extension)
    if (segments[0] === 'itsm-backend' && segments.length >= 3) {
      segments.pop(); // drop file
    }
    const pkg = segments.join('/');
    const slot = perPkg.get(pkg) || { covered: 0, total: 0, files: 0 };
    slot.covered += agg.covered;
    slot.total += agg.total;
    slot.files += 1;
    perPkg.set(pkg, slot);
  }
  // Flatten
  const pkgs = [];
  for (const [pkg, agg] of perPkg.entries()) {
    const pct = agg.total === 0 ? 0 : (agg.covered / agg.total) * 100;
    pkgs.push({ pkg, covered: agg.covered, total: agg.total, files: agg.files, pct });
  }
  // Overall
  let overallCovered = 0;
  let overallTotal = 0;
  for (const p of pkgs) {
    overallCovered += p.covered;
    overallTotal += p.total;
  }
  return {
    overall: overallTotal === 0 ? 0 : (overallCovered / overallTotal) * 100,
    overallCovered,
    overallTotal,
    pkgs: pkgs.sort((a, b) => b.pct - a.pct),
  };
}

/**
 * Parse Jest json-summary output.
 * Shape: { total: {lines|branches|functions|statements: {pct, total, covered}},
 *         "<file>": same }
 */
function parseJestCoverage(filePath) {
  if (!filePath || !fs.existsSync(filePath)) return null;
  const raw = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  const total = raw.total || {};
  const files = [];
  for (const [file, agg] of Object.entries(raw)) {
    if (file === 'total') continue;
    const l = agg.lines || {};
    if (typeof l.pct !== 'number') continue;
    files.push({ file, pct: l.pct, covered: l.covered || 0, total: l.total || 0 });
  }
  files.sort((a, b) => a.pct - b.pct);
  return {
    overall: (total.lines && total.lines.pct) || 0,
    overallCovered: (total.lines && total.lines.covered) || 0,
    overallTotal: (total.lines && total.lines.total) || 0,
    files,
  };
}

function makeDelta(curr, baseline) {
  if (!baseline) return null;
  const delta = curr - baseline;
  const arrow = delta > 0.5 ? '🟢 ↑' : delta < -0.5 ? '🔴 ↓' : '⚪ ±';
  return `${arrow} ${delta >= 0 ? '+' : ''}${delta.toFixed(2)}%`;
}

function renderMarkdown(go, jest, baseline) {
  const lines = [];
  lines.push('# Test Coverage Summary');
  lines.push('');
  lines.push(`_Generated: ${new Date().toISOString()}_`);
  lines.push('');

  // Combined totals
  lines.push('## Combined Totals');
  lines.push('');
  lines.push('| Surface | Lines Covered | Total Lines | Coverage % | Delta vs baseline |');
  lines.push('|---|---:|---:|---:|---|');
  if (go) {
    const baseG = baseline && baseline.go;
    const dG = baseG != null ? makeDelta(go.overall, baseG) : '—';
    lines.push(`| Backend (Go) | ${go.overallCovered.toLocaleString()} | ${go.overallTotal.toLocaleString()} | **${go.overall.toFixed(2)}%** | ${dG} |`);
  } else {
    lines.push('| Backend (Go) | n/a | n/a | n/a | — |');
  }
  if (jest) {
    const baseJ = baseline && baseline.jest;
    const dJ = baseJ != null ? makeDelta(jest.overall, baseJ) : '—';
    lines.push(`| Frontend (Jest) | ${jest.overallCovered.toLocaleString()} | ${jest.overallTotal.toLocaleString()} | **${jest.overall.toFixed(2)}%** | ${dJ} |`);
  } else {
    lines.push('| Frontend (Jest) | n/a | n/a | n/a | — |');
  }
  lines.push('');
  lines.push('> Coverage thresholds (roadmap v1.5): Backend ≥40% / Frontend ≥30%.');
  lines.push('');

  // Backend worst packages
  lines.push('## Backend — Bottom 10 Packages (Go)');
  lines.push('');
  if (go && go.pkgs.length) {
    lines.push('| Package | Files | Lines Covered | Coverage % |');
    lines.push('|---|---:|---:|---:|');
    const worst = go.pkgs.slice().sort((a, b) => a.pct - b.pct).slice(0, 10);
    for (const p of worst) {
      lines.push(`| ${p.pkg} | ${p.files} | ${p.covered}/${p.total} | ${p.pct.toFixed(2)}% |`);
    }
  } else {
    lines.push('_No backend coverage data provided._');
  }
  lines.push('');

  // Frontend worst files
  lines.push('## Frontend — Bottom 10 Files (Jest)');
  lines.push('');
  if (jest && jest.files.length) {
    lines.push('| File | Lines Covered | Coverage % |');
    lines.push('|---|---:|---:|');
    for (const f of jest.files.slice(0, 10)) {
      lines.push(`| ${f.file} | ${f.covered}/${f.total} | ${f.pct.toFixed(2)}% |`);
    }
  } else {
    lines.push('_No frontend coverage data provided._');
  }
  lines.push('');

  lines.push('---');
  lines.push('');
  lines.push('_Auto-generated by `scripts/coverage-summarize.js`. Do not edit by hand._');
  lines.push('');
  return lines.join('\n');
}

function readBaseline(filePath) {
  if (!filePath || !fs.existsSync(filePath)) return null;
  try {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch (err) {
    return null;
  }
}

function main() {
  const args = parseArgs(process.argv);
  const go = parseGoCoverage(args.go);
  const jest = parseJestCoverage(args.jest);
  const baseline = readBaseline(args.baseline);
  const md = renderMarkdown(go, jest, baseline);
  if (args.out) {
    fs.mkdirSync(path.dirname(args.out), { recursive: true });
    fs.writeFileSync(args.out, md);
    console.log(`wrote ${args.out}`);
  } else {
    process.stdout.write(md);
  }
}

main();
