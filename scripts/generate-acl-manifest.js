#!/usr/bin/env node
/**
 * generate-acl-manifest.js
 *
 * Parses itsm-backend/router/router.go and all *_routes.go files to extract
 * all route registrations with their HTTP method, path, and RequirePermission
 * declarations.  Outputs docs/acl-manifest.yaml.
 *
 * Usage:
 *   node scripts/generate-acl-manifest.js [--check] [--output <path>]
 *
 * --check  Exit 1 if any non-public route is missing RequirePermission middleware.
 */

"use strict";

const fs = require("fs");
const path = require("path");

const ROUTER_DIR = path.resolve(__dirname, "../itsm-backend/router");
const OUTPUT_FILE = path.resolve(__dirname, "../docs/acl-manifest.yaml");

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function extractPermission(arg) {
  const m = arg.match(/middleware\.RequirePermission\s*\(\s*["']([^"']+)["']\s*,\s*["']([^"']+)["']\s*\)/);
  if (m) return `${m[1]}.${m[2]}`;
  const m2 = arg.match(/RequirePermission\s*\(\s*["']([^"']+)["']\s*,\s*["']([^"']+)["']\s*\)/);
  if (m2) return `${m2[1]}.${m2[2]}`;
  return null;
}

function extractRouteCall(l) {
  // Matches:  tickets.GET("/path", ...)  or  GET("/path", ...)  (chain call)
  const m = l.match(/^\s*(?:(\w+)\s*\.)?(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s*\(\s*["']([^"']+)["']/);
  if (!m) return null;
  return { method: m[2], routePath: m[3], groupVar: m[1] || null };
}

function inferMenu(p) {
  if (!p) return "unknown";
  const seg = (p.split("/").filter(Boolean))[1] || "";
  const map = {
    tickets: "tickets", ticket: "tickets",
    incidents: "incidents", problems: "problems",
    changes: "changes", releases: "releases",
    assets: "assets", cmdb: "cmdb", configuration: "cmdb",
    knowledge: "knowledge", knowledge_articles: "knowledge",
    sla: "sla", ai: "ai", agent: "ai",
    users: "users", groups: "groups", roles: "roles",
    permissions: "permissions", menus: "menus",
    org: "org", projects: "projects", applications: "applications",
    services: "services", service_catalog: "service_catalog",
    service_requests: "service_requests",
    approvals: "approvals", approval_chains: "approval_chains",
    audit_logs: "audit_logs", notifications: "notifications",
    licenses: "licenses", vendors: "vendors",
    surveys: "surveys", cloud: "cloud",
    dashboard: "dashboard", reports: "reports",
    msp: "msp", connectors: "connectors",
    system_configs: "system_configs", system: "system",
    configs: "configs", ws: "ws",
    feedback: "feedback", tags: "tags",
    ticket_categories: "tickets", ticket_tags: "tickets",
    workflows: "workflows",
  };
  return map[seg] || seg || "system";
}

function inferDescription(method, p) {
  if (!p) return "";
  const M = {
    GET:  { "": "列表", "/stats": "统计", "/search": "搜索",
             "/tree": "树形结构", "/calendar": "日历视图",
             "/:id": "详情", "/me": "当前用户" },
    POST: { "": "创建", "/export": "导出", "/import": "导入",
             "/batch-delete": "批量删除" },
    PUT:  { "": "更新" },
    PATCH:{ "": "部分更新" },
    DELETE:{ "": "删除" },
  };
  const actions = M[method] || {};
  for (const [suffix, desc] of Object.entries(actions)) {
    if (p.endsWith(suffix)) return desc;
  }
  return "";
}

// ---------------------------------------------------------------------------
// Parser state
// ---------------------------------------------------------------------------

const routes = [];
let currentFile = "";

// prefixStack: chain of path prefixes for nested groups
// e.g. ["/api/v1", "/ticket-categories"] for /api/v1/ticket-categories/...
let prefixStack = ["/api/v1"];

// groupCloseDepth: for each prefix in prefixStack, the brace depth when it should be popped
// pop whenever currentBraceDepth < groupCloseDepth[stack.length - 1]
const groupCloseDepth = [];  // parallel to prefixStack

function currentPrefix() {
  // 栈内存的已经是完整前缀（pushPrefix 时已拼接），取栈顶即可。
  // 之前用 join("") 会把所有历史前缀重复拼接，每遇到一个 Group
  // 前缀长度翻倍，导致正则在超长字符串上回溯直至 OOM。
  return prefixStack[prefixStack.length - 1];
}

function pushPrefix(p, depth) {
  prefixStack.push(p);
  groupCloseDepth.push(depth);
}

function popToDepth(depth) {
  while (prefixStack.length > 1 && groupCloseDepth[groupCloseDepth.length - 1] > depth) {
    prefixStack.pop();
    groupCloseDepth.pop();
  }
}

// ---------------------------------------------------------------------------
// File parser
// ---------------------------------------------------------------------------

function parseFile(filePath) {
  currentFile = path.basename(filePath);
  prefixStack = ["/api/v1"];
  groupCloseDepth.length = 0;

  const lines = fs.readFileSync(filePath, "utf8").split("\n");

  // Named group variables → their prefix (for chained calls like tickets.GET)
  const namedGroups = {};

  for (let lineNum = 0; lineNum < lines.length; lineNum++) {
    const raw = lines[lineNum];
    const l = raw.trim();
    const lno = lineNum + 1;

    if (!l || l.startsWith("//") || l.startsWith("/*")) continue;

    // Update brace depth for the whole line
    for (const c of l) {
      if (c === '{') {
        // increment handled below via count
      } else if (c === '}') {
        popToDepth(0); // will be refined below
      }
    }

    // Count brace depth changes on this line for group-close tracking
    let bd = 0;
    for (const c of l) { if (c === '{') bd++; else if (c === '}') bd--; }

    // -------------------------------------------------------------------------
    // Named group declarations: auth := r.Group("/api/v1")
    //                           categories := tenant.Group("/ticket-categories")
    // -------------------------------------------------------------------------
    const groupDecl = l.match(/^\s*(\w+)\s*:=\s*(?:r|auth|tenant|msp|public)\s*\.\s*Group\s*\(\s*"([^"]+)"\s*\)/);
    if (groupDecl) {
      const name = groupDecl[1];
      const prefix = groupDecl[2];
      namedGroups[name] = prefix;

      // Compute full prefix for this new group
      const fullPrefix = (currentPrefix() + "/" + prefix).replace(/\/+/g, "/");

      // The group block closes at the matching brace depth.
      // We need to track what depth this block closes at.
      // Since we're INSIDE the Group() call's { ... } block already in the
      // source, we need to count braces from here.
      // A simpler approach: track cumulative depth and pop when we return to
      // the depth we were at BEFORE this line (which may be harder to track).
      // Instead: find the closing brace depth by counting braces from the start.
      // We already push "" as placeholder; we'll track depth-based closes below.
      pushPrefix(fullPrefix, 0); // depth placeholder, corrected below
      continue;
    }

    // Handle Group() calls that don't use := (inline chaining)
    // e.g.  tenant.(*gin.RouterGroup).Group("/tickets")
    const inlineGroup = l.match(/\.\s*Group\s*\(\s*"([^"]+)"\s*\)/);
    if (inlineGroup && !groupDecl) {
      const prefix = inlineGroup[1];
      const fullPrefix = (currentPrefix() + "/" + prefix).replace(/\/+/g, "/");
      pushPrefix(fullPrefix, 0);
      continue;
    }

    // -------------------------------------------------------------------------
    // Route registration: groupVar.METHOD("/path", ...) or METHOD("/path", ...)
    // -------------------------------------------------------------------------
    const routeInfo = extractRouteCall(l);
    if (routeInfo) {
      const { method, routePath, groupVar } = routeInfo;

      // Determine prefix: use named group variable if available, else current stack
      let prefix;
      if (groupVar && namedGroups[groupVar] !== undefined) {
        // Named group variable — use its assigned prefix
        prefix = namedGroups[groupVar];
      } else {
        prefix = currentPrefix();
      }

      const rp = routePath.startsWith("/") ? routePath : "/" + routePath;
      const fullPath = (prefix + rp).replace(/\/+/g, "/");

      // Extract RequirePermission from the arguments after the path
      const pathIdx = l.indexOf(routePath);
      const rest = l.slice(pathIdx + routePath.length + 1);
      // Split on commas that precede middleware/func/handler tokens
      const argsRaw = rest.split(/,(?=\s*(?:middleware|func|[A-Z]))/);
      let permission = null;
      for (const arg of argsRaw) {
        const p = extractPermission(arg.trim());
        if (p) { permission = p; break; }
      }

      routes.push({
        method,
        path: fullPath,
        permission,
        menu: inferMenu(fullPath),
        description: inferDescription(method, fullPath),
        file: currentFile,
        line: lno,
      });
      continue;
    }

    // -------------------------------------------------------------------------
    // Group.Use() that creates a new sub-group:  tenant := auth.Use(...)
    // (Gin .Use() returns *RouterGroup but same prefix)
    // -------------------------------------------------------------------------
    const useAssign = l.match(/^\s*(\w+)\s*:=\s*(\w+)\s*\.\s*Use\s*\(/);
    if (useAssign) {
      const name = useAssign[1];
      const baseGroup = useAssign[2];
      // .Use() returns the same group — same prefix
      namedGroups[name] = namedGroups[baseGroup] || currentPrefix();
      continue;
    }

    // -------------------------------------------------------------------------
    // Track brace depth changes to pop group scopes
    // pop when we close a block that was opened within the current group
    // -------------------------------------------------------------------------
    // Simple: for each '{' on this line, increment a "local depth"
    // for each '}', decrement and check if we should pop
    // We use a cumulative depth counter
  }
}

// ---------------------------------------------------------------------------
// Discover router files
// ---------------------------------------------------------------------------

function discoverRouterFiles(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  const files = [];
  for (const e of entries) {
    if (e.isFile() && (e.name.endsWith("_routes.go") || e.name === "router.go")) {
      files.push(path.join(dir, e.name));
    }
  }
  files.sort((a, b) => (a.endsWith("router.go") ? -1 : b.endsWith("router.go") ? 1 : 0));
  return files;
}

// ---------------------------------------------------------------------------
// Known public routes (no auth required)
// ---------------------------------------------------------------------------

const KNOWN_PUBLIC = new Set([
  "/api/v1/health", "/api/v1/healthz", "/api/v1/readyz",
  "/api/v1/version", "/api/v1/auth/login", "/api/v1/refresh-token",
  "/api/v1/csrf-token", "/api/v1/readiness/ga",
]);

// ---------------------------------------------------------------------------
// Path normalization
// ---------------------------------------------------------------------------

function normalizePath(p) {
  if (!p) return p;
  return p.replace(/\/api\/v1\/api\/v1\//g, "/api/v1/");
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

const args = process.argv.slice(2);
const checkMode = args.includes("--check");
const outIdx = args.indexOf("--output");
const outFile = outIdx !== -1 ? args[outIdx + 1] : OUTPUT_FILE;

console.log("Parsing router files...");
for (const f of discoverRouterFiles(ROUTER_DIR)) {
  parseFile(f);
}

let covered = 0;
let unprotected = [];
let pubCount = 0;

for (const r of routes) {
  r.path = normalizePath(r.path);
  if (KNOWN_PUBLIC.has(r.path)) {
    r.isPublic = true;
    r.permission = null;
    pubCount++;
    covered++;
    continue;
  }
  if (r.permission) {
    r.isProtected = true;
    covered++;
  } else {
    r.isProtected = false;
    unprotected.push(r);
  }
}

const total = routes.length;
const coveragePct = total > 0 ? ((covered / total) * 100).toFixed(2) : "0.00";

// ---------------------------------------------------------------------------
// Emit YAML
// ---------------------------------------------------------------------------

const yaml = [];
const y = (line) => yaml.push(line);

y(`version: "1.0"`);
y(`generated_at: "${new Date().toISOString()}"`);
y(`router_version: "1.0"`);
y(`total_routes: ${total}`);
y(`permission_coverage: "${coveragePct}%"`);
y(`routes:`);

for (const r of routes) {
  y(`  - method: ${r.method}`);
  y(`    path: ${r.path}`);
  if (r.permission) {
    y(`    permission: ${r.permission}`);
  } else if (r.isPublic) {
    y(`    permission: null  # PUBLIC`);
  } else {
    y(`    permission: null  # MISSING ACL`);
  }
  y(`    menu: ${r.menu}`);
  if (r.description) y(`    description: ${r.description}`);
  y(`    source: ${r.file}:${r.line}`);
}

fs.writeFileSync(outFile, yaml.join("\n") + "\n", "utf8");
console.log(`Wrote ${outFile} (${total} routes, ${coveragePct}% with permission)`);

// ---------------------------------------------------------------------------
// Check mode
// ---------------------------------------------------------------------------

if (checkMode) {
  console.log("\n=== ACL Coverage Check ===");
  if (unprotected.length > 0) {
    console.error(`\nFAIL: ${unprotected.length} non-public routes missing RequirePermission:`);
    for (const r of unprotected.slice(0, 50)) {
      console.error(`  ${r.method.padEnd(8)} ${r.path}  (${r.file}:${r.line})`);
    }
    if (unprotected.length > 50) {
      console.error(`  ... and ${unprotected.length - 50} more`);
    }
    process.exit(1);
  }
  console.log("PASS: All non-public routes have RequirePermission");
  process.exit(0);
}
