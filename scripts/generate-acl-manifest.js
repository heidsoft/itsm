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
  // RequirePermission / RequireMSPPermission 都是权限中间件
  const m = arg.match(/Require(?:MSP)?Permission\s*\(\s*["']([^"']+)["']\s*,\s*["']([^"']+)["']\s*\)/);
  if (m) return `${m[1]}.${m[2]}`;
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

// ---------------------------------------------------------------------------
// File parser
// ---------------------------------------------------------------------------

function parseFile(filePath) {
  currentFile = path.basename(filePath);

  const lines = fs.readFileSync(filePath, "utf8").split("\n");

  // 已知分组变量 → 完整路径前缀。
  // 基础变量约定（router.go 与各 Setup* 函数参数的实际挂载点）：
  //   r      → engine 根（""）
  //   auth / tenant / public → /api/v1（见 router.go SetupRoutes）
  const namedGroups = {
    r: "",
    auth: "/api/v1",
    tenant: "/api/v1",
    public: "/api/v1",
  };
  // 分组级 .Use(RequirePermission(...)) → 作为该分组下路由的默认权限
  const groupPermissions = {};

  function resolveVar(name) {
    if (namedGroups[name] !== undefined) return namedGroups[name];
    return "/api/v1"; // 未知变量保守回退
  }

  for (let lineNum = 0; lineNum < lines.length; lineNum++) {
    const raw = lines[lineNum];
    const l = raw.trim();
    const lno = lineNum + 1;

    if (!l || l.startsWith("//") || l.startsWith("/*")) continue;

    // -----------------------------------------------------------------------
    // 分组声明：name := parent.Group("/x") 或 name := parent.(*gin.RouterGroup).Group("/x")
    // -----------------------------------------------------------------------
    const groupDecl = l.match(/^(\w+)\s*:=\s*(\w+)(?:\.\(\*gin\.RouterGroup\))?\s*\.\s*Group\s*\(\s*"([^"]*)"\s*\)/);
    if (groupDecl) {
      const [, name, parent, prefix] = groupDecl;
      const base = parent === "r" ? "" : resolveVar(parent);
      namedGroups[name] = (base + (prefix ? "/" + prefix : "")).replace(/\/+/g, "/") || "/";
      // 继承父分组的分组级权限
      if (groupPermissions[parent]) groupPermissions[name] = groupPermissions[parent];
      continue;
    }

    // name := parent.Use(...) → 同前缀（gin 的 .Use 返回同一分组）
    const useAssign = l.match(/^(\w+)\s*:=\s*(\w+)(?:\.\(\*gin\.RouterGroup\))?\s*\.\s*(?:Group\s*\(\s*"([^"]*)"\s*\)\s*\.\s*)?Use\s*\(/);
    if (useAssign) {
      const [, name, parent, grpPrefix] = useAssign;
      const base = parent === "r" ? "" : resolveVar(parent);
      namedGroups[name] = (base + (grpPrefix ? "/" + grpPrefix : "")).replace(/\/+/g, "/") || base;
      const perm = extractPermission(l);
      if (perm) groupPermissions[name] = perm;
      continue;
    }

    // 分组级权限：name.Use(middleware.RequirePermission(...))
    const groupUse = l.match(/^(\w+)\s*\.\s*Use\s*\(/);
    if (groupUse && namedGroups[groupUse[1]] !== undefined) {
      const perm = extractPermission(l);
      if (perm) groupPermissions[groupUse[1]] = perm;
      continue;
    }

    // -----------------------------------------------------------------------
    // 路由注册：groupVar.METHOD("/path", ...)
    // -----------------------------------------------------------------------
    const routeInfo = extractRouteCall(l);
    if (routeInfo) {
      const { method, routePath, groupVar } = routeInfo;

      const prefix = groupVar ? resolveVar(groupVar) : "/api/v1";
      const rp = routePath === "" ? "" : (routePath.startsWith("/") ? routePath : "/" + routePath);
      const fullPath = (prefix + rp).replace(/\/+/g, "/");

      // 从路径后的参数中提取 RequirePermission；无则回退到分组级权限
      let permission = extractPermission(l.slice(l.indexOf(routePath) + routePath.length));
      if (!permission && groupVar && groupPermissions[groupVar]) {
        permission = groupPermissions[groupVar];
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
  // 外部系统回调：由独立签名/事件校验保护，无法要求登录态 RBAC
  "/api/v1/connectors/feishu/callback",
  "/api/v1/feishu/oauth/callback",
  "/api/v1/feishu/webhook",
]);

// 认证即可访问的身份/自服务类端点：登录后任意角色都需要，
// 无法绑定具体 RBAC 资源权限（否则新用户拿不到菜单/个人信息）。
// 变更此清单需评审：只允许身份、会话、WS 票据类端点。
const AUTH_ONLY = new Set([
  "/metrics",                    // Prometheus，JWT 认证后暴露
  "/api/v1/ws/ticket",           // WS 短期票据颁发（JWT 认证）
  "/api/v1/ws/notifications",    // WS 连接（一次性票据认证）
  "/api/v1/msp/status",          // MSP 基础状态（MSP 中间件已限制）
  "/api/v1/auth/me",
  "/api/v1/auth/tenants",
  "/api/v1/auth/logout",
  "/api/v1/auth/menus",
  "/api/v1/users/profile",
  "/api/v1/users/me",
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
  if (AUTH_ONLY.has(r.path)) {
    r.isAuthOnly = true;
    r.permission = null;
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
  } else if (r.isAuthOnly) {
    y(`    permission: null  # AUTH_ONLY (登录即可，身份/自服务端点)`);
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
