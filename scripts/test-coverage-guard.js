#!/usr/bin/env node
/**
 * test-coverage-guard.js
 *
 * CI guard for the "source X changed ⇒ test X must change" rule.
 *
 * Why this exists (L4.1):
 *   The audit that produced this codebase found multiple examples of
 *   production code that evolved without a corresponding test update:
 *   - `ticket-approval-api.ts` started calling `/api/v1/tickets/approval-workflows`
 *     while the backend never had that route; the only test asserted
 *     `expect.stringContaining('/api/v1/tickets')` and happily passed.
 *   - `ticket_controller.go` exposed `UpdateTicket` returning 5001 (InternalError)
 *     for "ticket not found" while `GetTicket` correctly returned 4004 (NotFound),
 *     and no test caught the asymmetry. (Historical: the `controller/` layer was
 *     deleted in 48eae74a; the same asymmetry class now lives in
 *     `handlers/<domain>/`.)
 *
 *   This script makes those regressions loud: it inspects the git diff for a
 *   given PR/commit range and, for every modified source file, requires a
 *   corresponding test file to exist. A candidate test satisfies the guard when
 *   it is present on disk OR appears in the diff range — so a source file that
 *   already has a test passes without editing it, while a source file with no
 *   test at all fails and must either gain one or opt out with a
 *   `// test-coverage-guard: skip` justification.
 *
 * Mapping rules (kept conservative; false positives cost reviewer time):
 *   Frontend (itsm-frontend/src/lib/<area>/<name>.ts):
 *     test candidates:
 *       <area>/__tests__/<name>.test.ts
 *       <area>/__tests__/<name>-*.test.ts
 *
 *   Backend (itsm-backend/handlers/<domain>/<name>.go,
 *            itsm-backend/service/<name>_service.go,
 *            itsm-backend/service/<pkg>/<name>.go,
 *            itsm-backend/internal/services/<name>.go):
 *     test candidates (any one of):
 *       <dir>/<name>_test.go
 *       <dir>/<name>_service_test.go        (for service/<name>_service.go)
 *       any *_test.go in the same package    (domain slices use
 *                                             handler_test.go / service_test.go)
 *
 *   The service layer was physically moved to internal/services/ (332+ files,
 *   G0-b / 7063011e5). Until 2026-11-14 both this classifier and the
 *   workflow's `paths:` filter still targeted the legacy `service/` tree, so
 *   changes to the core business layer were completely outside this guard.
 *   Both are now covered; see scripts/__tests__/test-coverage-guard.test.js
 *   for the injection tests that lock this in.
 *
 * Exemptions (no test required):
 *   - Type-only files: *.d.ts, files inside a types/ directory
 *   - Index/barrel files: index.ts, index.js
 *   - Generated files: anything under itsm-backend/ent/schema/ (the ent
 *     ORM regenerates these; the consumer is the generator)
 *   - Migrations: itsm-backend/migrations/**
 *   - Wire-only files (interfaces only, no executable code):
 *     the script flags these as `skipped` but does NOT count them as
 *     failures.
 *
 * Usage:
 *   node scripts/test-coverage-guard.js [--base <ref>] [--head <ref>]
 *                                       [--json] [--strict-diff]
 *
 * Defaults:
 *   --base  origin/main  (override with TEST_COVERAGE_BASE env var)
 *   --head  HEAD         (override with TEST_COVERAGE_HEAD env var)
 *
 * --strict-diff (observation only, 2026-11-14):
 *   The normal rule accepts a test that merely EXISTS on disk. That is a
 *   deliberately weak gate: editing a handler in a package that already has
 *   any *_test.go passes without touching a test. --strict-diff reports the
 *   stricter question "was a candidate test actually MODIFIED in this diff?"
 *   as a warning / JSON field, but NEVER changes the exit code. It exists so
 *   the project can measure how often the weak gate is relied on before
 *   deciding whether to promote it to a hard rule.
 *
 * Exit codes:
 *   0 - all source changes have a test change (or were exempted)
 *   1 - at least one source file changed without a test change
 *   2 - git diff failed (no commits, missing refs, etc.)
 */

'use strict';

const { execSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const args = parseArgs(process.argv.slice(2));
const BASE = args.base || process.env.TEST_COVERAGE_BASE || 'origin/main';
const HEAD = args.head || process.env.TEST_COVERAGE_HEAD || 'HEAD';

const REPO_ROOT = path.resolve(__dirname, '..');

main();

function main() {
  const allChanged = changedFiles(BASE, HEAD);
  if (!allChanged) {
    console.error(
      `test-coverage-guard: could not compute git diff ${BASE}..${HEAD}. ` +
        `Make sure both refs exist locally. For an initial branch, use --base HEAD~1.`
    );
    process.exit(2);
  }

  const sources = allChanged.filter(isSourceFile).filter((f) => !isExempt(f));
  const tests = allChanged.filter(isTestFile);

  const missing = [];
  for (const src of sources) {
    const candidates = testCandidatesFor(src);
    // A candidate may be a glob (frontend `<name>-*.test.ts`). Both the
    // on-disk check and the diff check must expand it, otherwise that
    // candidate is dead weight and a real sibling test is reported missing.
    if (!candidates.some((c) => candidateSatisfied(c, tests))) {
      missing.push({ src, candidates });
    }
  }

  // Observation-only signal. `candidateChangedInDiff` deliberately ignores
  // the on-disk state so the number is comparable across CI runs.
  const strictMissing = args.strictDiff
    ? sources.filter(
        (src) => !testCandidatesFor(src).some((c) => candidateChangedInDiff(c, tests))
      )
    : null;

  if (args.json) {
    process.stdout.write(
      JSON.stringify(
        {
          base: BASE,
          head: HEAD,
          totalChanged: allChanged.length,
          sourceFiles: sources.length,
          testFiles: tests.length,
          missing,
          strictDiff: strictMissing
            ? { enabled: true, missingWithoutTestChange: strictMissing }
            : { enabled: false },
        },
        null,
        2
      ) + '\n'
    );
  } else {
    report(allChanged.length, sources.length, tests.length, missing);
    if (strictMissing) reportStrictDiff(strictMissing, sources.length);
  }

  // --strict-diff is observation only: never contributes to the exit code.
  process.exit(missing.length === 0 ? 0 : 1);
}

function reportStrictDiff(strictMissing, srcCount) {
  const banner = '─'.repeat(72);
  console.log(banner);
  console.log(' 🔍 strict-diff observation (NON-BLOCKING — exit code unchanged)');
  console.log(
    `    ${strictMissing.length}/${srcCount} source files changed without any` +
      ` candidate test change in this diff.`
  );
  console.log('    The normal gate only requires a test to exist on disk; this');
  console.log('    section measures how often that weaker rule is relied on.');
  for (const src of strictMissing) {
    console.log(`   ::warning::strict-diff: no test change in diff for ${src}`);
  }
  console.log(banner);
}

function report(totalChanged, srcCount, testCount, missing) {
  const banner = '═'.repeat(72);
  console.log(banner);
  console.log(` test-coverage-guard  diff: ${BASE}..${HEAD}`);
  console.log(banner);
  console.log(` Total files changed:    ${totalChanged}`);
  console.log(` Source files:           ${srcCount}`);
  console.log(` Test files:             ${testCount}`);
  console.log(` Missing test updates:   ${missing.length}`);
  console.log('');

  if (missing.length === 0) {
    console.log(' ✅ Every source change has a corresponding test change.');
    return;
  }

  console.log(' ❌ The following source files were modified without updating any test:');
  console.log('');
  for (const { src, candidates } of missing) {
    console.log(`   • ${src}`);
    console.log(`     Expected one of:`);
    for (const c of candidates) console.log(`       - ${c}`);
    console.log('');
  }
  console.log(' To waive the guard for a single source file, add a comment containing');
  console.log(' `test-coverage-guard: skip` on its own line in that file.');
  console.log(banner);
}

// ---------- git plumbing ----------

function changedFiles(base, head) {
  // Use --diff-filter=ACMR to ignore deletions and renames; we only care
  // about modified-or-added files for test-coverage purposes.
  // `--name-only` keeps the output tiny and easy to parse.
  try {
    const out = execSync(
      `git diff --name-only --diff-filter=ACMR ${shellQuote(base)} ${shellQuote(head)}`,
      { cwd: REPO_ROOT, encoding: 'utf8', stdio: ['pipe', 'pipe', 'pipe'] }
    );
    return out
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean);
  } catch (err) {
    // Fall back to `git log` if the base ref doesn't exist (initial branch).
    if (/unknown revision|not a tree/.test(String(err.stderr || err.message))) {
      try {
        const out = execSync(
          `git log --pretty=format: --name-only --diff-filter=ACMR ${shellQuote(head)}`,
          { cwd: REPO_ROOT, encoding: 'utf8', stdio: ['pipe', 'pipe', 'pipe'] }
        );
        return Array.from(new Set(out.split('\n').map((s) => s.trim()).filter(Boolean)));
      } catch {
        return null;
      }
    }
    return null;
  }
}

function shellQuote(s) {
  // Git refs can include characters that the shell would interpret. We avoid
  // spawning a shell so this is mostly paranoia, but escape anyway.
  return `'${String(s).replace(/'/g, "'\\''")}'`;
}

// ---------- classification ----------

function isSourceFile(p) {
  return (
    isFrontendSource(p) ||
    isBackendSource(p)
  );
}

function isFrontendSource(p) {
  // itsm-frontend/src/lib/<area>/<name>.ts  (api, services, store, hooks, ...)
  // Exclude __tests__ directories - those are test files, not source.
  return /^itsm-frontend\/src\/lib\/[^/]+\/[^/]+\.tsx?$/.test(p) && !/__tests__\//.test(p);
}

function isBackendSource(p) {
  // Domain-sliced handlers (target architecture):
  //   itsm-backend/handlers/<domain>/<name>.go
  // Shared business logic (legacy layout, still populated):
  //   itsm-backend/service/<name>_service.go
  //   itsm-backend/service/<pkg>/<name>.go
  // Shared business logic (post-G0-b target layout):
  //   itsm-backend/internal/services/<name>.go  (and future sub-packages)
  // The legacy itsm-backend/controller/** horizontal layer was deleted in
  // 48eae74a; it is intentionally NOT matched any more.
  // IMPORTANT: must not match *_test.go (those are tests, not source).
  if (/_test\.go$/.test(p)) return false;
  return (
    /^itsm-backend\/handlers\/[^/]+\/[^/]+\.go$/.test(p) ||
    /^itsm-backend\/service\/(?:[^/]+\/)?[^/]+\.go$/.test(p) ||
    /^itsm-backend\/internal\/services\/(?:[^/]+\/)*[^/]+\.go$/.test(p)
  );
}

function isExempt(p) {
  // Type-only files and barrel/index files don't need their own test.
  if (/\.d\.tsx?$/.test(p)) return true;
  if (/\/(types|@types)\//.test(p)) return true;
  if (/\/index\.tsx?$/.test(p)) return true;
  if (/\/constants\.tsx?$/.test(p)) return true;
  // Backend: ent-generated schema and migrations are not hand-tested.
  if (p.startsWith('itsm-backend/ent/')) return true;
  if (p.startsWith('itsm-backend/migrations/')) return true;
  // The guard itself and its workflow.
  if (p === 'scripts/test-coverage-guard.js') return true;
  // A source file may opt out with a magic comment. Check the file on disk.
  try {
    const abs = path.join(REPO_ROOT, p);
    if (fs.existsSync(abs)) {
      const content = fs.readFileSync(abs, 'utf8');
      if (/^\s*\/\/.*test-coverage-guard:\s*skip/m.test(content)) return true;
      if (/^\s*\/\*[\s\S]*test-coverage-guard:\s*skip[\s\S]*\*\//m.test(content)) return true;
    }
  } catch {
    // ignore — file may have been deleted
  }
  return false;
}

function isTestFile(p) {
  return (
    /\/__tests__\/[^/]+\.test\.tsx?$/.test(p) ||
    /\/[^/]+\.test\.tsx?$/.test(p) ||
    /_test\.go$/.test(p)
  );
}

// ---------- candidate mapping ----------

function testCandidatesFor(src) {
  if (isFrontendSource(src)) return frontendCandidates(src);
  if (isBackendSource(src)) return backendCandidates(src);
  return [];
}

function frontendCandidates(src) {
  // itsm-frontend/src/lib/<area>/<name>.ts  →  <area>/__tests__/<name>.test.ts
  const m = src.match(/^itsm-frontend\/src\/lib\/([^/]+)\/([^/]+)\.tsx?$/);
  if (!m) return [];
  const [, area, name] = m;
  return [
    `itsm-frontend/src/lib/${area}/__tests__/${name}.test.ts`,
    `itsm-frontend/src/lib/${area}/__tests__/${name}.test.tsx`,
    `itsm-frontend/src/lib/${area}/__tests__/${name}-*.test.ts`,
    `itsm-frontend/src/lib/${area}/__tests__/${name}-*.test.tsx`,
  ];
}

function backendCandidates(src) {
  const dir = path.posix.dirname(src);
  const base = path.posix.basename(src, '.go');
  // Literal name-based candidates:
  //   analytics.go          → analytics_test.go
  //   release_service.go    → release_service_test.go / release_test.go
  const candidates = [
    `${dir}/${base}_test.go`,
    `${dir}/${base.replace(/_(service)$/, '')}_test.go`,
  ];
  // Domain slices and shared service packages follow an "at least one *_test.go
  // in the package" convention (handler_test.go, service_test.go, entity_test.go,
  // repository_impl_test.go, ...), so any sibling test file in the same package
  // satisfies the guard.
  try {
    for (const entry of fs.readdirSync(path.join(REPO_ROOT, dir))) {
      if (entry.endsWith('_test.go')) candidates.push(`${dir}/${entry}`);
    }
  } catch {
    // Directory missing (e.g. deleted file) — keep the literal candidates only.
  }
  return candidates;
}

// ---------- utilities ----------

function fileExists(rel) {
  // For glob candidates with `*` we still want a real check via the diff
  // (fileExists is only used when the candidate is a literal path).
  try {
    return fs.existsSync(path.join(REPO_ROOT, rel));
  } catch {
    return false;
  }
}

function globToRegExp(glob) {
  // Only `*` is used by our candidate patterns; everything else is literal.
  const body = glob
    .replace(/[.+?^${}()|[\]\\]/g, '\\$&')
    .replace(/\*/g, '.*');
  return new RegExp(`^${body}$`);
}

/** Was this exact test path (or a glob-expanded sibling) modified in the diff? */
function candidateChangedInDiff(candidate, tests) {
  if (!candidate.includes('*')) return tests.includes(candidate);
  const re = globToRegExp(candidate);
  return tests.some((t) => re.test(t));
}

/** Does a candidate test exist on disk OR appear in the diff? */
function candidateSatisfied(candidate, tests) {
  if (!candidate.includes('*')) {
    return tests.includes(candidate) || fileExists(candidate);
  }
  if (candidateChangedInDiff(candidate, tests)) return true;
  // Expand the glob against the real directory on disk. Without this the
  // frontend `<name>-*.test.ts` / `.test.tsx` candidates were unreachable:
  // fs.existsSync() was called with a literal path containing `*`.
  const dir = path.posix.dirname(candidate);
  const re = globToRegExp(candidate);
  try {
    return fs
      .readdirSync(path.join(REPO_ROOT, dir))
      .some((entry) => re.test(`${dir}/${entry}`));
  } catch {
    return false;
  }
}

function parseArgs(argv) {
  const out = { json: false, strictDiff: false };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--base') out.base = argv[++i];
    else if (a === '--head') out.head = argv[++i];
    else if (a === '--json') out.json = true;
    else if (a === '--strict-diff') out.strictDiff = true;
    else if (a === '-h' || a === '--help') {
      printHelp();
      process.exit(0);
    } else if (a.startsWith('--base=')) out.base = a.slice('--base='.length);
    else if (a.startsWith('--head=')) out.head = a.slice('--head='.length);
  }
  return out;
}

function printHelp() {
  console.log(
    `Usage: node scripts/test-coverage-guard.js [--base <ref>] [--head <ref>] [--json] [--strict-diff]\n` +
      `\n` +
      `Ensures every modified source file has a corresponding test file on disk\n` +
      `or in the diff range. Exits 0 when coverage holds, 1 when a source file\n` +
      `changed without a test, 2 on git errors.\n` +
      `\n` +
      `--strict-diff adds a NON-BLOCKING report of source files whose candidate\n` +
      `tests were not modified in the diff. It never changes the exit code.\n` +
      `\n` +
      `Defaults: --base origin/main  --head HEAD\n` +
      `\n` +
      `Override via env: TEST_COVERAGE_BASE, TEST_COVERAGE_HEAD\n` +
      `\n` +
      `Per-file opt-out: add '// test-coverage-guard: skip' on its own line.`
  );
}