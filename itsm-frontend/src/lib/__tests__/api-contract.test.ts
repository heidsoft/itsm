/**
 * API contract test: ensures every frontend path used in `src/lib/api/*.ts`
 * has a matching route registered in `itsm-backend/router/router.go`.
 *
 * Why this exists (L2.1):
 *   The bug we fixed in L1.1 (`/api/v1/tickets/approval-workflows` vs
 *   `/api/v1/approval-workflows`) was masked by `expect.stringContaining` in
 *   `ticket-approval.test.ts`. This test is a coarser but stricter guard: it
 *   scans all API files and asserts each declared path has a matching backend
 *   route. It runs as part of `npm test`, so a wrong path fails CI loudly.
 *
 * Strategy:
 *   1. Parse router.go to extract all registered routes, resolving Gin group
 *      prefixes by tracking `varName := parent.Group("/X")` and
 *      `varName := parent.Use(...)` declarations.
 *   2. Parse each `src/lib/api/*.ts` file to extract paths passed to
 *      `httpClient.{get,post,put,patch,delete}()`.
 *   3. Normalize template-literal placeholders (`${var}`) to a generic token.
 *   4. Match segment-by-segment: same length, static segments equal, and
 *      either side's dynamic token (frontend `${...}` or backend `:foo`)
 *      matches anything.
 *
 * Known limitations:
 *   - Concatenated paths like `'/api/v1/' + id + '/comments'` are skipped.
 *   - The parser is heuristic and may miss unusual Gin setups; documented
 *     false positives should be added to KNOWN_UNMATCHED_FRONTEND_PATHS below.
 */

import * as fs from 'fs';
import * as path from 'path';
import {
  DISABLED_API_CONTRACTS,
  PRODUCT_CAPABILITIES,
} from '@/config/product-capabilities';

// Workspace root = four levels up from src/lib/__tests__ → itsm-frontend/../
const WORKSPACE_ROOT = path.resolve(__dirname, '../../../..');
const BACKEND_DIR = path.join(WORKSPACE_ROOT, 'itsm-backend');
const FRONTEND_API_DIR = path.resolve(__dirname, '../api');

// Frontend paths that are intentionally not matched. Each entry must have a
// brief reason. Keep this list short — every entry represents an unfilled
// contract gap that should eventually be wired up.
interface SkipEntry {
  path: string;
  reason: string;
}
const KNOWN_UNMATCHED_FRONTEND_PATHS: ReadonlyArray<SkipEntry> = [
  // Add entries as: { path: '/api/v1/...', reason: 'TODO: backend not implemented' }
];

/**
 * Heuristic: a path that begins with a template-literal placeholder
 * (`${someBase}/...`) cannot be statically resolved without knowing
 * `someBase` at runtime. Examples found in the codebase:
 *   - `${CMDB_BASE}/...` (cmdb-api.ts, cmdb-advanced-api.ts)
 *   - `${this.baseUrl}/...` (bpmn-workflow-api.ts, knowledge-base-api.ts, ...)
 *   - `${API_BASE}/...` / `${TICKETS_BASE}/...` (various)
 *
 * We don't try to strip the prefix — that would mismatch backend routes
 * that legitimately have a `/api/v1/<scope>/...` shape. Instead, the
 * `isMatched` function returns false for these paths so they can be
 * tracked separately and audited manually. Tests that match ALL static
 * paths but fail only on dynamic-base paths are still useful signal.
 */
function startsWithDynamicBase(path: string): boolean {
  return /^\$\{[^}]+\}/.test(path);
}

// ---------- Backend path extraction ----------

interface ExtractedRoutes {
  /** Full paths like `/api/v1/tickets/:id` (group prefix + route). */
  fullPaths: Set<string>;
  /** All raw string literals found, for diagnostics. */
  literals: Set<string>;
}

function extractBackendRoutes(content: string): ExtractedRoutes {
  const fullPaths = new Set<string>();
  const literals = new Set<string>();
  const groupAbsolutePrefix = new Map<string, string>();
  groupAbsolutePrefix.set('r', '');

  // Parse in source order. Router setup intentionally reuses local names such
  // as `reports` and `templates`; a final name→prefix map loses earlier scopes.
  const statementRe =
    /(\w+)\s*:=\s*([\w.*()]+?)\.(Group|Use)\(\s*(?:"([^"]*)")?|\b(\w+)\.(GET|POST|PUT|PATCH|DELETE)\(\s*"([^"]*)"/g;
  let match: RegExpExecArray | null;
  while ((match = statementRe.exec(content)) !== null) {
    if (match[1]) {
      const [, variable, parentExpression, kind, relativePath] = match;
      const parent = parentExpression.match(/\w+/)?.[0];
      const parentPrefix = parent ? groupAbsolutePrefix.get(parent) ?? '' : '';
      groupAbsolutePrefix.set(
        variable,
        kind === 'Group' ? parentPrefix + (relativePath ?? '') : parentPrefix
      );
      continue;
    }

    const variable = match[5];
    const routePath = match[7];
    fullPaths.add((groupAbsolutePrefix.get(variable) ?? '') + routePath);
    literals.add(routePath);
  }

  return { fullPaths, literals };
}

// ---------- Frontend path extraction ----------

interface ExtractedFrontendPath {
  path: string;
  method: string;
  file: string;
  line: number;
  /** Template-literal placeholders preserved as `${...}` for matching. */
  forMatching: string;
}

function extractFrontendPaths(filePath: string): ExtractedFrontendPath[] {
  const content = fs.readFileSync(filePath, 'utf-8');
  const lines = content.split('\n');
  const results: ExtractedFrontendPath[] = [];
  const staticBases = new Map<string, string>();

  for (const match of content.matchAll(/(?:const|static(?:\s+readonly)?)\s+(\w+)\s*=\s*['"]([^'"]+)['"]/g)) {
    staticBases.set(match[1], match[2]);
  }

  // Match httpClient.METHOD("path", ...) | 'path' | `path` (with optional <T> generic).
  const callRe =
    /httpClient\.(get|post|put|patch|delete)\s*(?:<[^>]+>)?\s*\(\s*['"`]([^'"`]+)['"`]/g;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const re = new RegExp(callRe.source, 'g');
    let m: RegExpExecArray | null;
    while ((m = re.exec(line)) !== null) {
      const method = m[1].toUpperCase();
      const rawPath = m[2];
      const resolvedPath = rawPath.replace(/^\$\{(?:this\.)?([^}]+)\}/, (token, name: string) => {
        return staticBases.get(name) ?? token;
      });
      results.push({
        path: rawPath,
        method,
        file: path.basename(filePath),
        line: i + 1,
        forMatching: resolvedPath,
      });
    }
  }

  return results;
}

// ---------- Matching heuristic ----------

function isDynamicFrontendSegment(seg: string): boolean {
  // Template-literal placeholder: ${varName}
  return seg.startsWith('${') && seg.endsWith('}');
}

function isDynamicBackendSegment(seg: string): boolean {
  // Gin param: :id, :comment_id, :workflowId
  return seg.startsWith(':');
}

function isQueryOnly(path: string): boolean {
  // Paths like "/foo?bar=${baz}" should be reduced to "/foo".
  return path.includes('?');
}

function stripQuery(path: string): string {
  return path.split('?')[0];
}

function stripApiV1Prefix(segments: string[]): string[] {
  if (segments[0] === 'api' && segments[1] === 'v1') {
    return segments.slice(2);
  }
  return segments;
}

function segmentsMatch(feSegs: string[], beSegs: string[]): boolean {
  if (feSegs.length !== beSegs.length) return false;
  for (let i = 0; i < feSegs.length; i++) {
    const fe = feSegs[i];
    const be = beSegs[i];
    if (fe === be) continue;
    if (isDynamicFrontendSegment(fe)) continue;
    if (isDynamicBackendSegment(be)) continue;
    return false;
  }
  return true;
}

function isMatched(frontendPath: string, backend: ExtractedRoutes): boolean {
  // Parser limitation: paths starting with `${someBase}/` cannot be resolved
  // statically. Report as unmatched so they're visible in the diagnostic,
  // but tag them so reviewers can distinguish parser gaps from real drift.
  if (startsWithDynamicBase(stripQuery(frontendPath))) return false;
  // Strip query string — backend doesn't see those.
  const cleanPath = stripQuery(frontendPath);
  const feSegsAll = cleanPath.split('/').filter(Boolean);
  if (feSegsAll.length === 0) return false;

  // Build frontend segment candidates: with and without the /api/v1 prefix.
  const feCandidates: string[][] = [feSegsAll, stripApiV1Prefix(feSegsAll)];

  for (const bp of backend.fullPaths) {
    const beSegsAll = bp.split('/').filter(Boolean);
    // Build backend segment candidates: with and without the /api/v1 prefix.
    const beCandidates: string[][] = [beSegsAll, stripApiV1Prefix(beSegsAll)];
    for (const feSegs of feCandidates) {
      for (const beSegs of beCandidates) {
        if (segmentsMatch(feSegs, beSegs)) return true;
      }
    }
  }
  return false;
}

// ---------- The test ----------

describe('API contract: frontend ↔ backend routes', () => {
  const collectGoFiles = (directory: string): string[] =>
    fs.readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
      const entryPath = path.join(directory, entry.name);
      if (entry.isDirectory()) {
        if (['ent', 'vendor', 'migrations'].includes(entry.name)) return [];
        return collectGoFiles(entryPath);
      }
      return entry.name.endsWith('.go') && !entry.name.endsWith('_test.go') ? [entryPath] : [];
    });

  const backend = collectGoFiles(BACKEND_DIR)
    .map(file => extractBackendRoutes(fs.readFileSync(file, 'utf-8')))
    .reduce<ExtractedRoutes>(
      (combined, routes) => ({
        fullPaths: new Set([...combined.fullPaths, ...routes.fullPaths]),
        literals: new Set([...combined.literals, ...routes.literals]),
      }),
      { fullPaths: new Set(), literals: new Set() }
    );

  // Sanity: parser should extract a meaningful number of routes.
  it('extracts at least 100 backend routes (sanity)', () => {
    expect(backend.fullPaths.size).toBeGreaterThan(100);
  });

  const apiFiles = fs
    .readdirSync(FRONTEND_API_DIR)
    .filter((f) => f.endsWith('.ts') && !f.endsWith('.d.ts'));

  type Mismatch = { path: string; file: string; line: number; method: string; parserGap: boolean };
  const mismatches: Mismatch[] = [];
  const disabledContractHits = new Map<number, number>();

  for (const file of apiFiles) {
    const fp = path.join(FRONTEND_API_DIR, file);
    const paths = extractFrontendPaths(fp);
    for (const p of paths) {
      // Skip known unmatched (TODO entries).
      if (KNOWN_UNMATCHED_FRONTEND_PATHS.some((e) => e.path === p.path)) continue;
      const parserGap = startsWithDynamicBase(p.forMatching);
      if (!isMatched(p.forMatching, backend)) {
        const disabledIndex = DISABLED_API_CONTRACTS.findIndex(
          contract =>
            contract.file === file &&
            PRODUCT_CAPABILITIES[contract.capability] === false &&
            (!contract.path || contract.path.test(p.forMatching))
        );
        if (disabledIndex >= 0) {
          disabledContractHits.set(disabledIndex, (disabledContractHits.get(disabledIndex) ?? 0) + 1);
          continue;
        }
        mismatches.push({ path: p.path, file, line: p.line, method: p.method, parserGap });
      }
    }
  }

  // Single test that fails with a complete diagnostic when any path is
  // unmatched. Splitting into per-path `it()` blocks is tempting but creates
  // hundreds of failing tests that hide the root-cause signal.
  it(`every frontend API path matches a backend route (${mismatches.length} mismatches)`, () => {
    if (mismatches.length === 0) return;

    const byFile = new Map<string, Mismatch[]>();
    for (const m of mismatches) {
      if (!byFile.has(m.file)) byFile.set(m.file, []);
      byFile.get(m.file)!.push(m);
    }

    const realDrift = mismatches.filter((m) => !m.parserGap);
    const parserGaps = mismatches.filter((m) => m.parserGap);
    const lines: string[] = [];
    lines.push(`${mismatches.length} frontend path(s) do not match any backend route:`);
    lines.push(`  - ${realDrift.length} real contract drift`);
    lines.push(`  - ${parserGaps.length} parser gaps (dynamic-base paths starting with \`\${var}/\`)`);
    if (realDrift.length > 0) {
      lines.push(`\n=== REAL CONTRACT DRIFT (priority for fixes) ===`);
      for (const [file, items] of [...byFile.entries()].sort()) {
        const real = items.filter((m) => !m.parserGap);
        if (real.length === 0) continue;
        lines.push(`\n  ${file}:`);
        for (const m of real.slice(0, 30)) {
          lines.push(`    L${m.line}  ${m.method.padEnd(6)}  ${m.path}`);
        }
        if (real.length > 30) lines.push(`    ... and ${real.length - 30} more`);
      }
    }
    if (parserGaps.length > 0) {
      lines.push(`\n=== PARSER GAPS (reviewer should manually verify) ===`);
      for (const [file, items] of [...byFile.entries()].sort()) {
        const pg = items.filter((m) => m.parserGap);
        if (pg.length === 0) continue;
        lines.push(`\n  ${file}:`);
        for (const m of pg.slice(0, 20)) {
          lines.push(`    L${m.line}  ${m.method.padEnd(6)}  ${m.path}`);
        }
        if (pg.length > 20) lines.push(`    ... and ${pg.length - 20} more`);
      }
    }
    throw new Error(lines.join('\n'));
  });

  it('keeps every disabled API contract explicit and exercised by the audit', () => {
    const staleEntries = DISABLED_API_CONTRACTS.filter((_, index) => !disabledContractHits.has(index));
    expect(staleEntries).toEqual([]);
  });
});
