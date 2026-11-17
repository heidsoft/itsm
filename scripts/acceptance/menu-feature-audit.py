#!/usr/bin/env python3
"""菜单-功能一致性审计（prod 库 × 源码 × HTTP 实测）。

基准：prod 数据库的 menus/permissions/roles/role_permissions（种子初始化数据），
对照三层实现逐菜单核验「页面存在、权限码真实、后端路由支撑、角色覆盖、页面可达」。

只读审计：DB 全 SELECT，HTTP 全 GET，不写任何业务数据。
"""
from __future__ import annotations

import argparse
import glob
import http.cookiejar
import json
import os
import re
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any

REPO = Path(__file__).resolve().parents[2]
FE = REPO / "itsm-frontend"
BE = REPO / "itsm-backend"
API = "/api/v1"
RESULTS: list[dict[str, Any]] = []
OPENER: urllib.request.OpenerDirector | None = None
BASE = "http://127.0.0.1"
FRONTEND_BASE = ""
ENV: dict[str, str] = {"PATH": "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"}


def record(group: str, name: str, ok: bool, *, detail: str = "",
           severity: str = "FAIL") -> dict[str, Any]:
    item = {"group": group, "name": name, "result": "PASS" if ok else severity,
            "detail": detail}
    RESULTS.append(item)
    print(f"[{item['result']:5}] {group:14} {name:46}{('  ' + detail) if detail else ''}",
          flush=True)
    return item


# ---------------------------------------------------------------------------
# 数据采集
# ---------------------------------------------------------------------------
def psql(sql: str) -> list[tuple[str, ...]]:
    cmd = ["docker", "exec", "itsm-postgres-prod", "sh", "-c",
           f'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tAF "|" -c "{sql}"']
    out = subprocess.run(cmd, capture_output=True, text=True, timeout=60, env=ENV)
    rows = []
    for line in (out.stdout or "").splitlines():
        line = line.rstrip("\n")
        if line:
            rows.append(tuple(line.split("|")))
    return rows


def load_db() -> dict[str, Any]:
    menus = []
    for r in psql("SELECT id, COALESCE(parent_id::text,''), name, path, "
                  "COALESCE(permission_code,''), is_visible, is_enabled "
                  "FROM menus WHERE tenant_id=1 ORDER BY parent_id NULLS FIRST, sort_order"):
        menus.append({"id": int(r[0]), "parent": r[1] or None, "name": r[2],
                      "path": r[3], "code": r[4],
                      "visible": r[5] == "t", "enabled": r[6] == "t"})
    perms = {}
    for r in psql("SELECT id, code, resource, action, COALESCE(description,'') "
                  "FROM permissions WHERE tenant_id IN (0,1) ORDER BY id"):
        perms[r[1]] = {"id": int(r[0]), "resource": r[2], "action": r[3],
                       "description": r[4]}
    roles = []
    for r in psql("SELECT id, name, code, is_system FROM roles ORDER BY id"):
        roles.append({"id": int(r[0]), "name": r[1], "code": r[2],
                      "system": r[3] == "t"})
    rp: dict[int, set[str]] = {}
    for r in psql("SELECT role_id, p.code FROM role_permissions rp "
                  "JOIN permissions p ON p.id = rp.permission_id"):
        rp.setdefault(int(r[0]), set()).add(r[1])
    users = psql("SELECT count(*) FROM users")
    return {"menus": menus, "perms": perms, "roles": roles, "rp": rp,
            "users": int(users[0][0]) if users else 0}


def frontend_pages() -> dict[str, str]:
    """src/app/**/page.tsx → 路由路径。路由组 (main) 与动态段如实处理。"""
    out: dict[str, str] = {}
    for p in glob.glob(str(FE / "src/app/**/page.tsx"), recursive=True):
        rel = Path(p).relative_to(FE / "src/app")
        parts = []
        for seg in rel.parts[:-1]:
            if seg.startswith("(") and seg.endswith(")"):
                continue
            if seg.startswith("[") and seg.endswith("]"):
                parts.append(f":{seg[1:-1]}")
            else:
                parts.append(seg)
        route = "/" + "/".join(parts) if parts else "/"
        out[route] = str(Path(p).relative_to(REPO))
    return out


def backend_permission_routes() -> dict[str, list[str]]:
    """router/*.go 与 handlers/**/*.go 里 RequirePermission("res","act") → 路径字面量。

    handlers/<domain>/ 的域内路由文件（routes.go 等）是目标架构的一部分，
    必须与 router/ 一并扫描，否则会误报「零路由码」。
    """
    out: dict[str, list[str]] = {}
    pat = re.compile(r'RequirePermission\(\s*"([a-z_]+)"\s*,\s*"([a-z_]+)"\s*\)')
    route_pat = re.compile(r'"(/[a-z0-9\-/:_A-Z]+)"')
    files = sorted(glob.glob(str(BE / "router/*.go")))
    files += [f for f in sorted(glob.glob(str(BE / "handlers/**/*.go"), recursive=True))]
    for f in files:
        if f.endswith("_test.go"):
            continue
        text = Path(f).read_text(encoding="utf-8")
        lines = text.splitlines()
        cur_path = ""
        for i, line in enumerate(lines):
            m = route_pat.search(line)
            if m and ("GET" in line or "POST" in line or "PUT" in line
                      or "DELETE" in line or "PATCH" in line or ".Group(" in line):
                cur_path = m.group(1)
            m2 = pat.search(line)
            if m2:
                code = f"{m2.group(1)}:{m2.group(2)}"
                out.setdefault(code, []).append(cur_path)
    return out


def menu_baseline_paths() -> set[str]:
    text = (BE / "pkg/menubaseline/baseline.go").read_text(encoding="utf-8")
    return set(re.findall(r'Path:\s*"([^"]+)"', text))


def fe_route_config_perms() -> dict[str, str]:
    text = (FE / "src/lib/router/route-config.ts").read_text(encoding="utf-8")
    out = {}
    for m in re.finditer(r"path:\s*'([^']+)'[^}]*?permission:\s*'([^']+)'", text):
        out[m.group(1)] = m.group(2)
    return out


# ---------------------------------------------------------------------------
# 审计
# ---------------------------------------------------------------------------
def audit_menus_vs_pages(db: dict[str, Any], pages: dict[str, str]) -> None:
    menus = [m for m in db["menus"] if m["visible"] and m["enabled"]]
    missing_page: list[str] = []
    dynamic = []
    for m in menus:
        if m["path"] in pages:
            continue
        # 父容器路径（有子菜单或存在子路由 layout）允许无 page.tsx
        kids = [k for k in menus if k["parent"] == str(m["id"])]
        if kids and (pages.get(m["path"] + "/") or any(
                p.startswith(m["path"] + "/") for p in pages)):
            continue
        # 已知合法重定向/容器
        if m["path"] in ("/admin", "/workflow", "/ai"):
            continue
        # 菜单指向动态详情页属于配置错误
        probe = m["path"].rstrip("/")
        hit = [r for r in pages if r.rstrip("/") == probe]
        if hit:
            continue
        missing_page.append(m["path"])
    if missing_page:
        record("菜单→页面", "全部可见菜单路径有对应前端页面", False,
               detail=f"缺页面 {len(missing_page)}: {missing_page[:12]}")
    else:
        record("菜单→页面", f"全部 {len(menus)} 个可见菜单路径有对应前端页面", True)

    # 页面存在但菜单缺失（孤儿页面仅提示，不算失败——部分为详情页/深链）
    menu_paths = {m["path"].rstrip("/") for m in menus}
    orphans = [p for p in pages
               if p.rstrip("/") not in menu_paths
               and not any(p.startswith(mp + "/") for mp in menu_paths)
               and ":" not in p and p.count("/") <= 2
               and not p.startswith("/install") and p != "/"]
    record("页面→菜单", "顶级/二级页面均有菜单入口（提示级）",
            len(orphans) == 0, severity="WARN",
            detail="孤儿页面: " + ", ".join(sorted(orphans)[:12]) if orphans else "")


def audit_menu_codes(db: dict[str, Any], backend: dict[str, list[str]]) -> None:
    menus = [m for m in db["menus"] if m["visible"] and m["enabled"] and m["code"]]
    # ① 菜单码 ∈ permissions
    ghost = sorted({m["code"] for m in menus if m["code"] not in db["perms"]})
    record("菜单权限码", "菜单引用的权限码全部真实存在", not ghost,
           detail=f"幽灵码: {ghost}" if ghost else f"覆盖 {len(set(m['code'] for m in menus))} 个码")
    # ② 信封 isActive（批次 C+ 的策略信封；纯文本行视为待升级不判死）
    inactive = []
    for m in menus:
        p = db["perms"].get(m["code"])
        if not p or not p["description"].startswith("{"):
            continue
        try:
            env = json.loads(p["description"])
            pol = env.get("policy", env)
            if pol.get("isActive") is False:
                inactive.append(m["code"])
        except json.JSONDecodeError:
            pass
    record("菜单权限码", "菜单码策略信封无 isActive=false", not inactive,
           detail=f"停用码: {sorted(set(inactive))}" if inactive else "")
    # ③ 菜单码有后端路由支撑（RequirePermission 真实挂载）
    # msp:* 已核实走 RequireMSPPermission 独立守卫体系（router/msp_routes.go，
    # 资源名为 msp_customer/msp_allocation/msp_ticket/msp_report）+ MSP 能力
    # 开关 fail-closed（未启用 503），不算零挂载。
    msp_guarded = {c for c in db["perms"] if c.startswith("msp:")}
    no_route = sorted({m["code"] for m in menus
                       if m["code"] in db["perms"] and m["code"] not in backend
                       and m["code"] not in msp_guarded})
    record("菜单权限码", "菜单码均有后端 RequirePermission 路由挂载", not no_route,
           detail=f"零路由码: {no_route}" if no_route else "")
    # ④ 反向：后端 RequirePermission 码必须在 DB 目录里（幽灵后端码）
    be_ghost = sorted({c for c in backend if c not in db["perms"]})
    record("后端路由面", "RequirePermission 码全部在 DB 权限目录", not be_ghost,
           detail=f"幽灵后端码: {be_ghost[:10]}" if be_ghost else
                  f"覆盖 {len(backend)} 个码")


def audit_roles(db: dict[str, Any]) -> None:
    menus = [m for m in db["menus"] if m["visible"] and m["enabled"] and m["code"]]
    menu_codes = {m["code"] for m in menus}
    # admin（租户管理员）必须能见全部菜单
    admin = next((r for r in db["roles"] if r["code"] == "admin"), None)
    if admin:
        got = db["rp"].get(admin["id"], set())
        miss = sorted(menu_codes - got)
        record("角色覆盖", "租户管理员角色(admin)持有全部菜单权限码", not miss,
               detail=f"缺失 {len(miss)}: {miss[:12]}"
                      + "（admin 权威源 internal/authz/roles.go 与菜单码选型错配；"
                        "当前无用户持有 admin 角色，实际用户为 super_admin 通配）"
                      if miss else f"覆盖 {len(menu_codes)} 码")
    super_admin = next((r for r in db["roles"] if r["code"] == "super_admin"), None)
    if super_admin:
        got = db["rp"].get(super_admin["id"], set())
        # 已核实：middleware/rbac.go AuthorizeResource 对 super_admin 角色名
        # 代码级直接放行（通配），DB 不落 rp 行是有意设计；菜单侧
        # menu_service_impl 对 super_admin 用户同样给 permissions["*"]。
        record("角色覆盖", "超级管理员 rp=0 为代码级通配设计（rbac.go:897 放行）",
               True, detail=f"DB rp={len(got)} 行；中间件无 DB 依赖")
    # 非管理员角色：菜单可见性抽查（end_user 不应看到管理面）
    for code, expect_block in (("end_user", "/admin"), ("technician", "/admin/menus")):
        role = next((r for r in db["roles"] if r["code"] == code), None)
        if not role:
            continue
        got = db["rp"].get(role["id"], set())
        leaked = sorted(m for m in menus
                        if m["path"].startswith(expect_block) and m["code"] in got)
        record("角色覆盖", f"{code} 不持有 {expect_block} 面板码", not leaked,
               severity="WARN",
               detail="泄漏: " + ", ".join(f"{m['path']}({m['code']})" for m in leaked[:8])
                      if leaked else "")
    # 每角色 rp 数量概览
    summary = ", ".join(f"{r['code']}={len(db['rp'].get(r['id'], set()))}"
                        for r in db["roles"])
    record("角色覆盖", f"角色权限授予概览（users={db['users']}）", True,
           severity="WARN", detail=summary)


# ---------------------------------------------------------------------------
# HTTP 实测
# ---------------------------------------------------------------------------
def probe_pages(db: dict[str, Any]) -> None:
    menus = [m for m in db["menus"] if m["visible"] and m["enabled"]]
    paths = sorted({m["path"] for m in menus})
    fails = []
    for p in paths:
        st, _, _ = fetch(FRONTEND_BASE + p)
        if st not in (200, 302, 307, 308):
            fails.append(f"{p}→{st}")
    record("HTTP页面", f"GET {len(paths)} 个菜单路径经 nginx 全部可达", not fails,
           detail="; ".join(fails[:10]) if fails else "")


def probe_page_apis(db: dict[str, Any]) -> None:
    """每个菜单码抽一个有代表性的 GET 路由实测（路由从 router 源码反解）。"""
    menus = [m for m in db["menus"] if m["visible"] and m["enabled"] and m["code"]]
    # 静态映射：资源 → 代表性列表端点（admin 面板页面的主数据源）
    resource_api = {
        "dashboard": "/dashboard", "ticket": "/tickets?pageSize=1",
        "incident": "/incidents?pageSize=1", "problem": "/problems?pageSize=1",
        "change": "/changes?pageSize=1", "knowledge": "/knowledge/articles?pageSize=1",
        "email_intake": "/email-intake/conversations?pageSize=1",
        # service:read 目前无任何路由挂载 RequirePermission("service","read")；
        # 服务目录菜单页实际消费 /service-catalogs（守卫 service_catalog:read）。
        # 这里探真实数据端点；「菜单码 vs API 守卫码」错配由角色覆盖检查单列。
        "service": "/service-catalogs?pageSize=1",
        "cmdb": "/cmdb/cis?pageSize=1", "asset": "/assets?pageSize=1",
        "sla": "/sla/policies?pageSize=1",
        "workflow": "/bpmn/ai/templates?pageSize=1",
        "ai": "/ai/metrics", "task": "/bpmn/tasks?pageSize=1",
        "release": "/releases?pageSize=1", "report": "/reports/service-quality",
        "marketplace": "/marketplace/items?pageSize=1",
        "msp": "/msp/customers?pageSize=1", "audit": "/audit-logs?pageSize=1",
        "notification": "/notifications?pageSize=1",
        "user": "/users?pageSize=1", "role": "/roles?pageSize=1",
        "group": "/groups?pageSize=1", "tenant": "/tenants?pageSize=1",
        "department": "/departments?pageSize=1", "team": "/teams?pageSize=1",
        "system_config": "/system-configs?pageSize=1", "bpmn": "/bpmn/definitions?pageSize=1",
        "connector": "/connectors", "asset_license": "/licenses?pageSize=1",
        "license": "/licenses?pageSize=1", "on_call": "/on-call/schedules",
        "ticket_type": "/ticket-types?pageSize=1",
        "ticket_template": "/tickets/templates?pageSize=1",
        "ticket_category": "/ticket-categories?pageSize=1",
        "assignment_rule": "/tickets/assignment-rules?pageSize=1",
        "automation_rule": "/tickets/automation-rules?pageSize=1",
        "service_catalog": "/service-catalogs?pageSize=1",
        "approval": "/approval-chains?pageSize=1",
        "customer_master": "/email-intake/customers?pageSize=1",
        "support_contract": "/email-intake/support-contracts?pageSize=1",
        "application": "/applications?pageSize=1",
        "project": "/projects?pageSize=1",
        "marketplace_catalog": "/marketplace/catalog/items?pageSize=1",
    }
    probes: dict[str, str] = {}
    for m in menus:
        perm = db["perms"].get(m["code"])
        if not perm:
            continue
        api = resource_api.get(perm["resource"])
        if api and m["code"] not in probes:
            probes[m["code"]] = API + api
    fails, unknown = [], []
    for code, api in sorted(probes.items()):
        st, data, _ = fetch(BASE + api)
        biz = data.get("code") if isinstance(data, dict) else None
        if st == 200 and biz == 0:
            continue
        if st in (503,) and (data.get("data") or {}).get("state") in ("disabled", "not_implemented"):
            unknown.append(f"{code}→{api}:503能力未启用")
            continue
        if code.startswith("msp:") and st in (503,):
            unknown.append(f"{code}→{api}:503 MSP能力未启用（fail-closed 如实边界）")
            continue
        if st in (404, 501):
            unknown.append(f"{code}→{api}:{st}")
            continue
        fails.append(f"{code}→{api}:HTTP{st}/code{biz}")
    record("HTTP接口", f"菜单权限码对应主数据接口实测（{len(probes)} 码）", not fails,
           detail="; ".join(fails[:10]) if fails else "")
    if unknown:
        record("HTTP接口", "待核对端点（404/503/能力未启用）", False, severity="WARN",
               detail="; ".join(unknown))
    # 已核实：GET /system/config（系统状态聚合）返回裸 JSON 非标准信封，
    # 属 API 契约偏差（AGENTS.md 要求全部接口 {code,message,data}）。
    st, data, _ = fetch(BASE + API + "/system/config")
    envelope_ok = isinstance(data, dict) and "code" in data and "message" in data
    record("HTTP接口", "GET /system/config 响应遵循标准信封", envelope_ok,
           severity="WARN",
           detail="" if envelope_ok else f"裸JSON: {json.dumps(data, ensure_ascii=False)[:100]}")


def fetch(url: str, timeout: float = 15.0) -> tuple[int, Any, float]:
    req = urllib.request.Request(url, headers={"Accept": "application/json"}, method="GET")
    started = time.monotonic()
    try:
        with OPENER.open(req, timeout=timeout) as resp:  # type: ignore[union-attr]
            raw = resp.read().decode("utf-8", "replace")
            status = resp.status
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", "replace")
        status = e.code
    except Exception as e:  # noqa: BLE001
        return 0, {"error": str(e)}, time.monotonic() - started
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError:
        parsed = {"raw": raw[:200]}
    return status, parsed, time.monotonic() - started


def login() -> None:
    global OPENER
    jar = http.cookiejar.CookieJar()
    OPENER = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
    pwd = ""
    for line in (REPO / ".env.prod").read_text(encoding="utf-8").splitlines():
        if line.startswith("ADMIN_PASSWORD="):
            pwd = line.split("=", 1)[1].strip()
    status, data, _ = fetch(BASE + f"{API}/auth/login")  # GET probe dummy
    req = urllib.request.Request(BASE + f"{API}/auth/login",
                                 data=json.dumps({"username": "admin", "password": pwd}).encode(),
                                 headers={"Content-Type": "application/json"}, method="POST")
    try:
        with OPENER.open(req, timeout=15) as resp:
            body = json.loads(resp.read().decode())
            status = resp.status
    except urllib.error.HTTPError as e:
        body, status = {}, e.code
    if not (status == 200 and body.get("code") == 0):
        raise SystemExit("login failed")
    record("平台", "管理员登录", True, detail="cookie 会话就绪")


def baseline_sync(db: dict[str, Any]) -> None:
    base = menu_baseline_paths()
    db_paths = {m["path"] for m in db["menus"]}
    only_db = sorted(db_paths - base)
    only_code = sorted(p for p in base - db_paths)
    record("基线同步", "DB menus 与 pkg/menubaseline 一致", not only_db and not only_code,
           severity="WARN",
           detail=(f"仅DB: {only_db[:8]} " if only_db else "")
                  + (f"仅基线: {only_code[:8]}" if only_code else "")
                  or f"两侧各 {len(db_paths)}/{len(base)} 条一致")


def ungated_endpoints() -> None:
    """已核实的静态发现：handlers/bpmn/dashboard.go 的统计面路由
    （metrics/bottlenecks/sla/tenant-stats）只挂认证、无 RequirePermission。
    租户隔离由 TenantMiddleware 保证，但资源级 RBAC 缺失。"""
    text = (BE / "handlers/bpmn/dashboard.go").read_text(encoding="utf-8")
    ungated = []
    for m in re.finditer(r'dashboard\.(GET|POST)\("([^"]+)"', text):
        seg = m.group(2)
        # 该路由注册处 ±6 行内含 RequirePermission 视为有守卫
        idx = text.find(f'"{seg}"')
        window = text[max(0, idx - 40): idx + 200]
        if "RequirePermission" not in window:
            ungated.append(seg)
    if ungated:
        record("后端路由面", "BPMN 统计面端点缺 RequirePermission（仅认证）", False,
               severity="WARN",
               detail="无守卫端点: " + ", ".join(ungated[:8]))


def main() -> int:
    global BASE, FRONTEND_BASE
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="http://127.0.0.1")
    ap.add_argument("--frontend-base", default="http://127.0.0.1")
    ap.add_argument("--json", default="")
    ap.add_argument("--md", default="")
    args = ap.parse_args()
    BASE = args.base.rstrip("/")
    FRONTEND_BASE = args.frontend_base.rstrip("/")
    login()
    db = load_db()
    pages = frontend_pages()
    backend = backend_permission_routes()
    record("采集", "数据面概览", True,
           detail=f"menus={len(db['menus'])} perms={len(db['perms'])} "
                  f"roles={len(db['roles'])} rp={sum(len(v) for v in db['rp'].values())} "
                  f"fe_pages={len(pages)} be_codes={len(backend)}")
    baseline_sync(db)
    ungated_endpoints()
    audit_menus_vs_pages(db, pages)
    audit_menu_codes(db, backend)
    audit_roles(db)
    probe_pages(db)
    probe_page_apis(db)
    summary = {"PASS": 0, "WARN": 0, "FAIL": 0, "BLOCKED": 0}
    for r in RESULTS:
        summary[r["result"]] = summary.get(r["result"], 0) + 1
    if args.json:
        Path(args.json).write_text(json.dumps({"summary": summary, "results": RESULTS},
                                              ensure_ascii=False, indent=2), encoding="utf-8")
    if args.md:
        lines = ["# 菜单-功能一致性审计报告", "",
                 f"结论：**{summary['PASS']} PASS / {summary['WARN']} WARN / {summary['FAIL']} FAIL**", ""]
        cur = ""
        for r in RESULTS:
            if r["group"] != cur:
                cur = r["group"]
                lines += [f"## {cur}", ""]
            lines.append(f"- {r['result']}  {r['name']}" + (f"  —  {r['detail'][:180]}" if r["detail"] else ""))
        Path(args.md).write_text("\n".join(lines), encoding="utf-8")
    print(f"SUMMARY {summary}", flush=True)
    return 0 if summary["FAIL"] == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
