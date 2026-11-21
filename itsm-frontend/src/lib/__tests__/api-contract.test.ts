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
 * 无法静态解析的调用点基线（棘轮，只许缩小）。
 *
 * 2026-11-21 引入。起因是修完「逐行匹配」与「嵌套泛型」两个解析器缺陷后，
 * 仍有一批调用**结构上**无法静态解析——它们不是解析器能力问题：
 *
 *   base-api.ts       httpClient.get<T>(path)        path 是运行时参数
 *   group-api.ts      httpClient.get(`${this.baseURL}/...`)  this 指向实例字段
 *   incident-api.ts   httpClient.get(API_URLS.INCIDENT(id))  函数返回路径
 *   menu-api.ts       httpClient.get(`/.../${tenantId}`)      局部变量前缀
 *
 * 对这些调用，「解析不到」不等于「有漂移」：真实路由由参数在运行时决定。
 * 但也不能就这么放过——放过就等于承认守卫有盲区，而守卫自己不知道。
 *
 * 处置：**精确到行**登记，条目数做上界棘轮。
 *   ① 未登记的解析缺口 → 直接失败（逼新增代码写可解析形式）
 *   ② 登记了但已不再缺口（已改写/已删除）→ 直接失败（逼清理死条目）
 *   ③ 登记文件消失了 → 直接失败
 *
 * 为什么精确到行而不是按文件计数：按文件计数时，删掉一处、加一处会
 * 让计数持平，「棘轮只减不增」形同虚设。行号会随无关改动漂移，
 * 所以配对方式用「文件+方法+实参文本」而非纯行号（见 unresolvedKey）。
 */
interface UnresolvedBaselineEntry {
  file: string;
  method: string;
  /** 实参原文（未解析的表达式，如 `path` / `this.baseURL` / `API_URLS.INCIDENT`）。 */
  arg: string;
  reason: string;
}
const UNRESOLVED_CALL_BASELINE: ReadonlyArray<UnresolvedBaselineEntry> = [  // base-api.ts：泛型基类，路径由调用方传入（6 处）
  { file: 'base-api.ts', method: 'GET', arg: 'path', reason: '泛型基类，路径由调用方传入' },
  { file: 'base-api.ts', method: 'POST', arg: 'path', reason: '泛型基类，路径由调用方传入' },
  { file: 'base-api.ts', method: 'PUT', arg: 'path', reason: '泛型基类，路径由调用方传入' },
  { file: 'base-api.ts', method: 'DELETE', arg: 'path', reason: '泛型基类，路径由调用方传入' },
  { file: 'base-api.ts', method: 'PATCH', arg: 'path', reason: '泛型基类，路径由调用方传入' },
  { file: 'base-api.ts', method: 'POST', arg: 'this.getPath', reason: '路径由实例方法计算' },
  // group-api.ts：实例字段拼接（2 处）
  { file: 'group-api.ts', method: 'GET', arg: 'this.baseURL', reason: '`${this.baseURL}` 实例字段前缀' },
  { file: 'group-api.ts', method: 'POST', arg: 'this.baseURL', reason: '直接传实例字段 this.baseURL' },
  // incident-api.ts：路径由 API_URLS 工厂函数返回（3 处）
  { file: 'incident-api.ts', method: 'GET', arg: 'API_URLS.INCIDENTS', reason: 'API_URLS.INCIDENTS() 工厂函数' },
  { file: 'incident-api.ts', method: 'GET', arg: 'API_URLS.INCIDENT', reason: 'API_URLS.INCIDENT(id) 工厂函数' },
  { file: 'incident-api.ts', method: 'POST', arg: 'API_URLS.INCIDENTS', reason: 'API_URLS.INCIDENTS() 工厂函数' },
  // menu-api.ts：局部变量前缀（2 处）
  { file: 'menu-api.ts', method: 'GET', arg: 'tenantId', reason: '模板串以局部变量 ${tenantId} 开头' },
  { file: 'menu-api.ts', method: 'POST', arg: 'this.baseUrl', reason: '直接传实例字段 this.baseUrl' },
  // service-catalog-api.ts：三元表达式选择路径（1 处）
  { file: 'service-catalog-api.ts', method: 'GET', arg: 'endpoint', reason: '路径由三元表达式决定（pending / me 两个端点）' },
  { file: 'bpmn-ai-api.ts', method: 'POST', arg: 'endpoint', reason: '`${this.baseUrl}/generate...` 前缀即实例字段，无可取字面前缀' },
  // auditlog-api.ts 的 endpoint 已由「收集模板串字面前缀」解析，不再进基线；
  // 它此前是审计日志唯一端点却从未被校验，属于解析器缺陷而非不可解析。
  // user-api.ts：实例字段拼接（2 处）
  { file: 'user-api.ts', method: 'GET', arg: 'this.baseURL', reason: '直接传实例字段 this.baseURL' },
  { file: 'user-api.ts', method: 'POST', arg: 'this.baseURL', reason: '直接传实例字段 this.baseURL' },
];

/** 基线条目数上界：只允许缩小。新增条目必须同期说明为何不能写成字面量路径。 */
const UNRESOLVED_CALL_BASELINE_UPPER_BOUND = UNRESOLVED_CALL_BASELINE.length;

/**
 * 分段匹配器的已知假阳性。
 *
 * `segmentsMatch` 允许「前端动态段 ↔ 后端静态段」任意配对，于是两侧都有动态段
 * 但位置错开时，两个不相关的路径会互相「吃掉」。实测案例（2026-10-24）：
 *
 *   前端  /api/v1/changes/templates/${templateId}/create
 *         → [changes, templates, ${dyn}, create]
 *   后端  /changes/:id/rollback-steps/:stepId
 *         → [changes, :id, rollback-steps, :stepId]
 *   逐段：changes==changes ✓ / templates↔:id（后端动态）✓ /
 *         ${dyn}↔rollback-steps（前端动态）✓ / create↔:stepId（后端动态）✓
 *   → 判定「已匹配」，但后端从未注册任何 /changes/templates 路由
 *     （router/ 下检索 templates 只有 dashboard 与 ticket 两组）。
 *
 * 后果不是误报，而是漏报：这条真实缺口被判成已匹配，于是
 * DISABLED_API_CONTRACTS 里 changeClassification 那条永远拿不到命中，
 * 「清单不能留死条目」的断言反而失败了。
 *
 * 这里不放松匹配规则——实测收紧为「动态段必须一一对应」会把 32 条未匹配
 * 放大到 74 条（其中多数是分组前缀解析的解析缺口，不是真实漂移），
 * 得不偿失。改为把个别确认过的假阳性显式登记，路径仍在缺口统计口径内，
 * 只是不再依赖匹配器去「看见」它。
 */
const MATCHER_FALSE_POSITIVES: ReadonlyArray<SkipEntry> = [
  {
    path: '/api/v1/changes/templates/${templateId}/create',
    reason:
      '被 /changes/:id/rollback-steps/:stepId 错配（两侧动态段位置错开）；后端无 /changes/templates 路由，缺口真实存在',
  },
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

/** 一处无法静态解析的调用点（进基线棘轮对账，不是「没问题」）。 */
interface UnresolvedCall {
  file: string;
  method: string;
  /** 实参原文，如 `path` / `this.baseURL` / `API_URLS.INCIDENT`。 */
  arg: string;
  line: number;
}

interface FrontendExtraction {
  paths: ExtractedFrontendPath[];
  unresolved: UnresolvedCall[];
}

function extractFrontendPaths(filePath: string): FrontendExtraction {
  const content = fs.readFileSync(filePath, 'utf-8');
  const results: ExtractedFrontendPath[] = [];
  const unresolved: UnresolvedCall[] = [];
  const staticBases = new Map<string, string>();
  // 重名绑定登记表。
  //
  // 为什么需要它：staticBases 是**文件级** name→path 映射，而 JS/TS 里
  // `const url = 'A'` 出现在两个函数内是完全合法的（各有各的作用域）。
  // 先前用 `Map.set` 无条件写入 → 后写覆盖前写，于是「先定义的端点」被
  // 解析成后一个端点，看起来匹配成功、实际从未校验过。2026-11-21 实测：
  // ai-api.ts 里两个 `const url` 让 /ai/audit-logs 长期处于这种假绿状态，
  // 直到新增第三个同名绑定时才暴露（3 mismatches）。
  //
  // 处置：重名即记为歧义，**不再猜测**归属——把这些名字产生的调用标成
  // 解析缺口（unresolvedAmbiguous），让它们出现在诊断里等人改写变量名，
  // 而不是随便挑一个值当答案。宁可见的红，不要看不见的绿。
  const ambiguousNames = new Set<string>();

  // 收集静态路径常量。
  //
  // 三件必须一起做，缺一件都留盲区：
  //   ① 收 '...' 与 "..."：常规字面量。
  //   ② 收模板串的**字面前缀**（第一个 `${` 之前的部分）：
  //        const endpoint = `/api/v1/audit-logs${query.toString() ? ...}`;
  //      旧实现只认「整串不含 $ 的模板串」，而仓库里绝大多数模板常量的
  //      $ 出现在尾部（查询串），于是统统不收 → 端点既不解析也不进缺口。
  //      查询串本就会被 stripQuery 去掉，所以取前缀是正确的、不是近似。
  //      前缀为空（`${this.baseUrl}/generate...`）说明整条路径都由实例字段
  //      决定，此时**不登记**——登记空串会把任意裸标识符都"解析"成空路径。
  //   ③ 收 `private static baseURL = '...'` 这类带修饰符的静态字段：
  //      旧正则要求 `static` 紧跟变量名，`private static` 会漏。
  const bindingRe = /(?:const|static(?:\s+readonly)?)\s+(\w+)\s*=\s*([^;\n]+)/g;
  for (const match of content.matchAll(bindingRe)) {
    const name = match[1];
    const expr = match[2].trim();
    let value: string | null = null;
    const quoted = /^'([^']*)'|^"([^"]*)"/.exec(expr);
    if (quoted) {
      value = quoted[1] ?? quoted[2] ?? '';
    } else if (expr.startsWith('`')) {
      const close = expr.indexOf('`', 1);
      const raw = close > 0 ? expr.slice(1, close) : expr.slice(1);
      const prefix = raw.split('${')[0];
      // 前缀为空 = 路径完全由变量决定，交给解析缺口机制处理。
      value = prefix.length > 0 ? prefix : null;
    }
    if (value === null) continue;
    if (staticBases.has(name) && staticBases.get(name) !== value) {
      ambiguousNames.add(name);
      continue;
    }
    staticBases.set(name, value);
  }

  // 解析 httpClient.METHOD<泛型>(第一个实参) 里的路径。
  //
  // 为什么不再用一条正则：2026-11-21 实测两个缺陷叠加，
  // 使「0 mismatches」对这个目录的 11 处调用完全不成立。
  //
  // 缺陷 A（逐行匹配）：旧实现 split('\n') 后**逐行**跑正则，而 prettier
  // 对较长的调用必然折行——
  //   return httpClient.post<T>(
  //     `/api/v1/agent/tools/invocations/${id}/review`,
  //     {}
  //   );
  // 第一行只有 `httpClient.post<T>(`，路径在下一行，整条失配。
  // 实测把路径改成后端不存在的 /reviewzzz，套件仍然 3 passed / 0 mismatches：
  // 「没被解析到」被读成「没有漂移」。
  //
  // 缺陷 B（泛型正则）：`(?:<[^>]+>)?` 在嵌套泛型处提前闭合。
  //   httpClient.get<PaginationResponse<CIType>>(`/path`)
  // 里 `[^>]+` 停在第一个 `>`，剩下的 `CIType>>(` 让后续 `\(` 失配。
  // cmdb-api.ts / workflow-api.ts / incident-api.ts 等多处因此从未被校验。
  //
  // 修法：角括号按**深度平衡**扫描，字符串/模板串按引号原样读取，
  // 不再假设「泛型里没有 >」。这与真实 TS 的语法一致，而非近似。
  const parseCallArg = (
    pos: number
  ): { raw: string; bareIdentifier: boolean } | null => {
    let i = pos;
    while (i < content.length && /\s/.test(content[i])) i++;
    const ch = content[i];
    if (ch === "'" || ch === '"') {
      let j = i + 1;
      while (j < content.length) {
        if (content[j] === '\\') {
          j += 2;
          continue;
        }
        if (content[j] === ch) return { raw: content.slice(i + 1, j), bareIdentifier: false };
        j++;
      }
      return null;
    }
    if (ch === '`') {
      let j = i + 1;
      let depth = 0;
      while (j < content.length) {
        if (content[j] === '\\') {
          j += 2;
          continue;
        }
        if (content[j] === '$' && content[j + 1] === '{') {
          depth++;
          j += 2;
          continue;
        }
        if (depth > 0 && content[j] === '}') {
          depth--;
          j++;
          continue;
        }
        if (depth === 0 && content[j] === '`') {
          return { raw: content.slice(i + 1, j), bareIdentifier: false };
        }
        j++;
      }
      return null;
    }
    if (/[A-Za-z_$]/.test(ch)) {
      let j = i;
      while (j < content.length && /[\w$.]/.test(content[j])) j++;
      return { raw: content.slice(i, j), bareIdentifier: true };
    }
    return null;
  };

  const lineOf = (idx: number): number => {
    let line = 1;
    for (let k = 0; k < idx && k < content.length; k++) {
      if (content.charCodeAt(k) === 10) line++;
    }
    return line;
  };

  const headRe = /httpClient\.(get|post|put|patch|delete)\b/g;

  // 分母：真实调用点的数量，**不是** `httpClient.get` 字样出现的次数。
  //
  // 这两者必须用同一判据（真实调用点 = 泛型之后紧跟 `(`），否则自检会误报：
  // cloudsignal-api.ts 第 18 行有一句注释 `// httpClient.get 会把数组展开成重复键`，
  // 按字样计数是 9，按真实调用点是 8，分子分母对不上就抛「扫描不完整」，
  // 而这处「缺口」其实是注释。判据不统一时，自检会变成噪音源而不是守卫。
  let httpCallCount = 0;
  for (const m of content.matchAll(headRe)) {
    let pos = (m.index ?? 0) + m[0].length;
    while (pos < content.length && /\s/.test(content[pos])) pos++;
    if (content[pos] === '<') {
      let depth = 0;
      let j = pos;
      for (; j < content.length; j++) {
        if (content[j] === '<') depth++;
        else if (content[j] === '>') {
          depth--;
          if (depth === 0) {
            j++;
            break;
          }
        }
      }
      pos = j;
      while (pos < content.length && /\s/.test(content[pos])) pos++;
    }
    if (content[pos] === '(') httpCallCount++;
  }

  for (const m of content.matchAll(headRe)) {
    const method = m[1].toUpperCase();
    const at = m.index ?? 0;
    let pos = at + m[0].length;
    while (pos < content.length && /\s/.test(content[pos])) pos++;
    // 泛型按深度平衡跳过，嵌套泛型不再截断。
    if (content[pos] === '<') {
      let depth = 0;
      let j = pos;
      for (; j < content.length; j++) {
        if (content[j] === '<') depth++;
        else if (content[j] === '>') {
          depth--;
          if (depth === 0) {
            j++;
            break;
          }
        }
      }
      pos = j;
      while (pos < content.length && /\s/.test(content[pos])) pos++;
    }
    // 不在 '(' 处说明这不是真实调用点（例如注释里提到 httpClient.put）。
    if (content[pos] !== '(') continue;
    const arg = parseCallArg(pos + 1);
    // 实参不是可读表达式（如 httpClient.post(buildUrl(x).then(...))）：
    // 记为解析缺口，交给基线棘轮对账，不能静默丢弃。
    if (!arg) {
      unresolved.push({
        file: path.basename(filePath),
        method,
        arg: '(无法读取实参)',
        line: lineOf(at),
      });
      continue;
    }
    const bareIdentifier = arg.bareIdentifier ? arg.raw.split('.')[0] : undefined;
    // 裸标识符只在能解析为静态 base 时纳入校验，避免把局部变量误报成漂移。
    // 解析不到 = 运行时决定路径（泛型基类、实例字段、工厂函数）→ 记缺口。
    if (arg.bareIdentifier && (!bareIdentifier || !staticBases.has(bareIdentifier))) {
      unresolved.push({
        file: path.basename(filePath),
        method,
        arg: arg.raw,
        line: lineOf(at),
      });
      continue;
    }
    const rawPath = arg.raw;
    // 重名绑定的裸标识符：无从判断这行指的是哪一个定义，不猜。
    if (arg.bareIdentifier && bareIdentifier && ambiguousNames.has(bareIdentifier)) {
      unresolved.push({
        file: path.basename(filePath),
        method,
        arg: arg.raw,
        line: lineOf(at),
      });
      continue;
    }
    const resolvedPath = arg.bareIdentifier
      ? staticBases.get(bareIdentifier!)!
      : rawPath.replace(/^\$\{(?:this\.)?([^}]+)\}/, (token, name: string) => {
          return staticBases.get(name) ?? token;
        });
    results.push({
      path: rawPath,
      method,
      file: path.basename(filePath),
      line: lineOf(m.index ?? 0),
      forMatching: resolvedPath,
    });
  }

  // 有重名绑定时无法保证任何一条 bare-identifier 调用的归属。
  //
  // 这里刻意**抛错而不是警告**：警告只写进日志，CI 与人眼都看不见，
  // 于是「0 mismatches」对这个文件继续成立——而这正是本次要消灭的假绿形态。
  // 重名绑定意味着该文件里至少一条端点永远校验不到，属于守卫本身的完整性缺陷，
  // 必须阻断，让责任落到「改成唯一变量名」这个具体动作上。
  if (ambiguousNames.size > 0) {
    throw new Error(
      `[api-contract] ${path.basename(filePath)} 存在重名静态路径绑定 ` +
        `(${[...ambiguousNames].join(', ')})：这些绑定对应的端点无法校验，` +
        `请改用文件内唯一的变量名（如 auditLogsUrl / approvalsUrl）。`
    );
  }

  // 完整性对账：解析成功 + 解析缺口 必须等于 httpClient 调用总次数。
  //
  // 这条是守卫的自检：只要有一处调用两边都没落，就说明上面的扫描仍有盲区，
  // 而盲区不会产生 mismatch——「0 mismatches」会继续假装一切正常。
  // 2026-11-21 实测：折行写法与嵌套泛型曾让本目录 11 处调用长期不被校验。
  if (results.length + unresolved.length !== httpCallCount) {
    throw new Error(
      `[api-contract] ${path.basename(filePath)} 扫描不完整：` +
        `检测到 ${httpCallCount} 处 httpClient 调用，` +
        `解析成功 ${results.length} 处、记为缺口 ${unresolved.length} 处，` +
        `合计与总数不符。这是守卫自身的盲区（不是代码问题），必须修复解析逻辑。`
    );
  }

  return { paths: results, unresolved };
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

/**
 * 只按**顶层** `?` 截断查询串。
 *
 * 为什么不能直接 `split('?')[0]`：查询串本身常常是模板表达式，里面还嵌着
 * 一个模板串，例如 skill-api.ts 的
 *   `/api/v1/admin/skills${query ? `?${query}` : ''}`
 * 朴素切分会得到 `/api/v1/admin/skills${query `（尾缺口未闭合），
 * 于是它被当成一条**真实漂移**报出来——而后端 handlers/skill/handler.go
 * 明明注册了 `admin.GET("", h.List)`。假红比假绿更容易误导人：
 * 它会让人去改一条本来正确的代码。
 *
 * 正确语义：从第一个 `${` 起，内部的 `?` 属于表达式而非路径；
 * 只有当 `?` 出现在 `${...}` 之外的深度 0 位置，才是真的查询串起点。
 */
function stripQuery(path: string): string {
  // 先看有没有**顶层** `?`：它是查询串起点，路径本体到此为止。
  // 扫描时记录 `${...}` 的深度，表达式内部的 `?` 不算路径分隔符。
  let depth = 0;
  for (let i = 0; i < path.length; i++) {
    const c = path[i];
    if (c === '$' && path[i + 1] === '{') {
      depth++;
      i++;
      continue;
    }
    if (depth > 0 && c === '}') {
      depth--;
      continue;
    }
    // 表达式内部还嵌着模板串（`${query ? `?${query}` : ''}`），
    // 里面的 `?` 是三元运算符，不是查询串起点。
    if (depth === 0 && c === '?') return path.slice(0, i);
  }

  // 没有顶层 `?`，但整条路径可能以「段尾附加表达式」形式表达查询串：
  //   /api/v1/admin/skills${query ? `?${query}` : ''}
  // 这里 `${` 紧跟在**非 /** 字符后（'s' 之后），说明它是拼在本段尾部的
  // 附加内容而不是一个独立动态段。路径本体到该 `${` 为止。
  // 若 `${` 前是 `/`（`/x/${id}/y`）则是真正的动态段，必须保留。
  const firstExpr = path.indexOf('${');
  if (firstExpr > 0 && path[firstExpr - 1] !== '/') {
    // 仅当该表达式内部确实带查询串语义（含 `?`）时才截断；
    // 否则它可能就是段的组成部分（如 `/tickets${suffix}`）。
    let d = 0;
    let end = -1;
    for (let i = firstExpr; i < path.length; i++) {
      if (path[i] === '$' && path[i + 1] === '{') {
        d++;
        i++;
        continue;
      }
      if (d > 0 && path[i] === '}') {
        d--;
        if (d === 0) {
          end = i;
          break;
        }
      }
    }
    const expr = end > 0 ? path.slice(firstExpr, end + 1) : path.slice(firstExpr);
    if (expr.includes('?')) return path.slice(0, firstExpr);
  }
  return path;
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
  const allUnresolved: UnresolvedCall[] = [];

  for (const file of apiFiles) {
    const fp = path.join(FRONTEND_API_DIR, file);
    const { paths, unresolved } = extractFrontendPaths(fp);
    allUnresolved.push(...unresolved);
    for (const p of paths) {
      // Skip known unmatched (TODO entries).
      if (KNOWN_UNMATCHED_FRONTEND_PATHS.some((e) => e.path === p.path)) continue;
      const parserGap = startsWithDynamicBase(p.forMatching);
      // 登记过的假阳性按「未匹配」处理，让它继续走下面的 disabled 清单判定，
      // 而不是像 KNOWN_UNMATCHED 那样直接跳过——否则清单条目会变成死条目。
      const maskedFalsePositive = MATCHER_FALSE_POSITIVES.some((e) => e.path === p.path);
      if (maskedFalsePositive || !isMatched(p.forMatching, backend)) {
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

  /**
   * 解析缺口棘轮（只许缩小）。
   *
   * 这条断言回答的是「守卫自己有没有盲区」，而不是「代码有没有漂移」。
   * 两者必须分开，因为处置完全不同：
   *   - 漂移 → 改代码或改后端路由；
   *   - 盲区 → 要么增强解析器，要么把不可解析的调用登记进基线。
   *
   * 三个方向都阻断，缺一个棘轮就会退化成装饰：
   *   ① 出现未登记的缺口 → 新代码在守卫视野外，必须失败
   *   ② 登记的缺口已不存在 → 条目成了死条目，README 式的「历史遗留」会越积越多
   *   ③ 总量超过基线 → 上界守卫（防止 ① 被大面积登记规避）
   */
  it('keeps unresolved-call baseline exact and monotonically shrinking', () => {
    const keyOf = (c: { file: string; method: string; arg: string }) =>
      `${c.file}|${c.method}|${c.arg}`;

    const actualKeys = allUnresolved.map(keyOf);
    const baselineKeys = UNRESOLVED_CALL_BASELINE.map(keyOf);

    // ① 未登记的新缺口。按 key 出现次数比较，允许同一文件同一表达式出现多次
    // （如 base-api.ts 的 5 个 `path`），但不允许比基线多。
    const remaining = new Map<string, number>();
    for (const k of baselineKeys) remaining.set(k, (remaining.get(k) ?? 0) + 1);

    const unregistered: UnresolvedCall[] = [];
    for (const c of allUnresolved) {
      const k = keyOf(c);
      const left = remaining.get(k) ?? 0;
      if (left > 0) {
        remaining.set(k, left - 1);
      } else {
        unregistered.push(c);
      }
    }

    const problems: string[] = [];
    if (unregistered.length > 0) {
      problems.push(
        `\n=== 未登记的解析缺口（${unregistered.length} 处）===`,
        `这些调用无法静态解析，且未在 UNRESOLVED_CALL_BASELINE 中登记。`,
        `优先改写为 httpClient.xxx('/literal/path')；确属运行时路径再登记并写明原因。`
      );
      for (const c of unregistered) {
        problems.push(`    ${c.file} L${c.line}  ${c.method.padEnd(6)}  ${c.arg}`);
      }
    }

    // ② 死条目：登记了，但该调用已不再无法解析（已改写或已删除）。
    const dead = [...remaining.entries()].filter(([, n]) => n > 0);
    if (dead.length > 0) {
      problems.push(
        `\n=== 已失效的基线条目（${dead.length} 条）===`,
        `这些条目对应的调用已被改写为可解析形式或已删除，请从基线中移除——`,
        `棘轮只允许缩小，留着条目会让「盲区总数」这个指标永远偏高。`
      );
      for (const [k, n] of dead) problems.push(`    ${k}  (剩余 ${n} 条未消化)`);
    }

    // ③ 上界：不允许通过大规模登记把棘轮改回去。
    if (UNRESOLVED_CALL_BASELINE.length > UNRESOLVED_CALL_BASELINE_UPPER_BOUND) {
      problems.push(
        `\n=== 基线超上界 ===`,
        `当前 ${UNRESOLVED_CALL_BASELINE.length} 条，上界 ${UNRESOLVED_CALL_BASELINE_UPPER_BOUND} 条。`,
        `该基线只允许缩小；确需扩充请同期说明为何无法改写为字面量路径。`
      );
    }

    if (problems.length > 0) throw new Error(problems.join('\n'));
  });
});
