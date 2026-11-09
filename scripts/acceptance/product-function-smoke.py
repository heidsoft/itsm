#!/usr/bin/env python3
"""ITSM 产品功能黑盒回归（prod 栈）。

用途：service 层迁移 / 工作流改动落地后，对真实 prod 栈做端到端功能验证。
不硬编码任何凭据：ADMIN_PASSWORD 从 .env.prod 读取；会话用内存 CookieJar。
写操作前重新获取 CSRF token（写后服务端会轮换 token）。

用法:
  python3 scripts/acceptance/product-function-smoke.py
  python3 scripts/acceptance/product-function-smoke.py --base http://127.0.0.1 --json /tmp/r.json --md /tmp/r.md
  # 迁移后独立 HEAD 实例：API 走新二进制，前端页面仍走现有 nginx；双重版本指纹
  python3 scripts/acceptance/product-function-smoke.py \
      --base http://127.0.0.1:8099 --frontend-base http://127.0.0.1 \
      --expect-head --expect-version head-a6cda5ead --migration-deep

输出：每项记录 HTTP / 业务码 / 耗时 / 结论；产出 JSON + Markdown 报告。
清理：创建的知识文章、工单在用例末尾自动删除；删除失败会在报告中留 ID。
"""
from __future__ import annotations

import argparse
import datetime as dt
import http.cookiejar
import json
import os
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any

REPO = Path(__file__).resolve().parents[2]
API = "/api/v1"

RESULTS: list[dict[str, Any]] = []
CLEANUP_IDS: dict[str, Any] = {}
OPENER: urllib.request.OpenerDirector | None = None
BASE = "http://127.0.0.1"
FRONTEND_BASE = ""
CSRF = ""


def env_password(env_file: Path) -> str:
    for line in env_file.read_text(encoding="utf-8").splitlines():
        if line.startswith("ADMIN_PASSWORD="):
            return line.split("=", 1)[1].strip()
    raise SystemExit("ADMIN_PASSWORD missing in .env.prod")


def call(method: str, path: str, body: Any = None, csrf: bool = False,
         timeout: float = 12.0) -> tuple[int, Any, float]:
    global CSRF
    # 允许传绝对 URL：前端页面探测可指向另一环境，而 API 始终走 BASE
    url = path if path.startswith("http") else BASE + path
    headers = {"Content-Type": "application/json"}
    if csrf:
        code, data, _ = call("GET", f"{API}/csrf-token", timeout=timeout)
        token = (data or {}).get("data", {}).get("csrfToken", "") if code == 200 else ""
        headers["X-CSRF-Token"] = token
        CSRF = token
    data = None
    if body is not None:
        data = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    started = time.monotonic()
    try:
        with OPENER.open(req, timeout=timeout) as resp:  # type: ignore[union-attr]
            raw = resp.read().decode("utf-8", "replace")
            status = resp.status
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", "replace")
        status = e.code
    except Exception as e:  # noqa: BLE001 - 网络异常也要落进报告
        return 0, {"error": str(e)}, time.monotonic() - started
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError:
        parsed = {"raw": raw[:300]}
    return status, parsed, time.monotonic() - started


def check(group: str, name: str, method: str, path: str, body: Any = None,
          csrf: bool = False, allow: tuple[int, ...] = (200, 201),
          biz_codes: tuple[int, ...] = (0,), severity: str = "FAIL") -> dict[str, Any]:
    status, data, elapsed = call(method, path, body, csrf=csrf)
    biz = data.get("code") if isinstance(data, dict) else None
    ok = status in allow and (biz is None or biz in biz_codes)
    item = {
        "group": group, "name": name, "method": method, "path": path,
        "http": status, "code": biz, "elapsed_ms": round(elapsed * 1000),
        "result": "PASS" if ok else severity,
        "message": (data.get("message") if isinstance(data, dict) else str(data)) if not ok else "",
    }
    RESULTS.append(item)
    print(f"[{item['result']:5}] {group:14} {name:38} HTTP={status} code={biz} {item['elapsed_ms']}ms"
          + (f"  {item['message']}" if item["message"] else ""), flush=True)
    return item


def login() -> None:
    global OPENER
    jar = http.cookiejar.CookieJar()
    OPENER = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
    pwd = env_password(REPO / ".env.prod")
    status, data, elapsed = call("POST", f"{API}/auth/login",
                                 {"username": "admin", "password": pwd})
    RESULTS.append({"group": "平台", "name": "管理员登录（httpOnly cookie）",
                    "method": "POST", "path": f"{API}/auth/login", "http": status,
                    "code": data.get("code"), "elapsed_ms": round(elapsed * 1000),
                    "result": "PASS" if status == 200 and data.get("code") == 0 else "FAIL",
                    "message": "" if data.get("code") == 0 else str(data.get("message"))})
    if data.get("code") != 0:
        raise SystemExit("login failed, abort")


def read_surface() -> None:
    probes = [
        ("工单", "tickets"), ("事件", "incidents"), ("问题", "problems"),
        ("变更", "changes"), ("知识文章", "knowledge/articles"), ("用户", "users"),
        ("角色", "roles"), ("权限", "permissions"), ("服务请求", "service-requests"),
        ("CMDB 配置项", "cmdb/cis"), ("SLA 策略", "sla/policies"),
        ("BPMN 定义", "bpmn/process-definitions"), ("BPMN 实例", "bpmn/process-instances"),
        ("报表", "reports/service-quality"), ("通知投递", "notification-deliveries"),
        ("运维命令", "admin/operations/commands"), ("连接器", "connectors"),
        ("发布", "releases"), ("租户", "tenants"), ("审批待办", "approvals"),
    ]
    for name, p in probes:
        check("核心读面", name, "GET", f"{API}/{p}?page=1&pageSize=1")


def migration_surface() -> None:
    # knowledge 域：批次 2 抽出的 RAG 窄契约在此装配
    for name, p in [
        ("知识分类", "knowledge/categories"), ("知识统计", "knowledge/stats"),
        ("最近文章", "knowledge/recent"), ("推荐文章", "knowledge/recommendations"),
        ("受限分类", "knowledge/categories/restricted"),
    ]:
        check("迁移·知识域", name, "GET", f"{API}/{p}", severity="WARN")
    check("迁移·知识域", "知识检索（RAG 读面）", "POST", f"{API}/knowledge/search",
          {"query": "ITSM", "limit": 3}, csrf=True, severity="WARN")
    # rbac 域：批次 2 把 role_service 归位到 handlers/rbac
    for name, p in [
        ("角色清单", "roles"), ("权限清单", "permissions"),
        ("用户清单", "users"), ("菜单权限码", "auth/menus"),
        ("能力成熟度", "capabilities"),
    ]:
        check("迁移·RBAC", name, "GET", f"{API}/{p}?page=1&pageSize=1")
    # 批次 1 退役 8 个实现后，相关业务读面不得 500 / 404
    for name, p in [
        ("部门清单", "departments"), ("部门树", "departments/tree"),
        ("CMDB 标签", "cmdb/tags"), ("服务目录", "service-catalog"),
        ("工单标签", "ticket-tags"), ("已知错误库", "known-errors"),
        ("服务目录模板", "service-catalogs"), ("MSP 分配", "msp/allocations"),
    ]:
        check("迁移·退役面", name, "GET", f"{API}/{p}?page=1&pageSize=1", severity="WARN")


def version_fingerprint() -> None:
    """行为级证据：能力矩阵必须反映 HEAD 源码事实，而不是旧镜像里的过期文案。

    --expect-head 时启用。旧镜像会把「问题域无 workflow.start 生产者 /
    ServiceTask 分事务 / 无补偿模板」写成现存缺口；HEAD 已闭环这些，只余
    真实租户跨域联调人工签字。这项断言比镜像 CreatedAt 更能证明测的是新代码。
    """
    status, data, elapsed = call("GET", f"{API}/capabilities")
    payload = data.get("data") if isinstance(data, dict) else None
    items = (payload or {}).get("items", []) if isinstance(payload, dict) else []
    wf = next((c for c in items if (c.get("key") or c.get("name")) == "workflow"), {})
    reason = str(wf.get("pilotReason") or "")
    closed = [
        ("问题域起流生产者已闭环", "问题域创建不落 workflow.start"),
        ("ServiceTask 同事务已闭环", "ServiceTask 外部副作用与流程完成分属两个事务"),
        ("补偿内置模板已闭环", "补偿只支持结束事件驱动且无内置模板"),
    ]
    for name, stale in closed:
        RESULTS.append({"group": "版本指纹", "name": name, "method": "GET",
                        "path": f"{API}/capabilities", "http": status, "code": data.get("code"),
                        "elapsed_ms": round(elapsed * 1000),
                        "result": "PASS" if reason and stale not in reason else "FAIL",
                        "message": "" if reason and stale not in reason else f"仍含旧缺口：{stale}"})
    for phrase in ("签字", "真实租户跨域联调"):
        RESULTS.append({"group": "版本指纹", "name": f"保留剩余阻断事实「{phrase}」",
                        "method": "GET", "path": f"{API}/capabilities", "http": status,
                        "code": data.get("code"), "elapsed_ms": 0,
                        "result": "PASS" if phrase in reason else "FAIL",
                        "message": "" if phrase in reason else "文案未反映当前验收状态"})
    RESULTS.append({"group": "版本指纹", "name": "workflow 未谎报 GA",
                    "method": "GET", "path": f"{API}/capabilities", "http": status,
                    "code": data.get("code"), "elapsed_ms": 0,
                    "result": "PASS" if wf.get("maturity") == "pilot" else "FAIL",
                    "message": "" if wf.get("maturity") == "pilot" else f"maturity={wf.get('maturity')}"})


def batch2_surface() -> None:
    """service 迁移批次 2 续批四个迁入域的定向黑盒用例（--migration-deep）。

    写操作都做成零副作用：CMDB 保存视图「创建后按 ID 删除」，仪表盘配置
    「读原值 → 原样回写 → 回读比对」，目标环境既有配置内容不变。
    """
    stamp = dt.datetime.now().strftime("%Y%m%d%H%M%S")
    group = "迁移·本批四域"

    def record(name: str, method: str, path: str, ok: bool, status: int,
               code: Any, message: str = "", severity: str = "FAIL") -> None:
        RESULTS.append({"group": group, "name": name, "method": method, "path": path,
                        "http": status, "code": code, "elapsed_ms": 0,
                        "result": "PASS" if ok else severity,
                        "message": "" if ok else message})

    # cmdb / saved_view_service.go：读面 + 写路径 + 清理
    check(group, "CMDB 保存视图列表", "GET", f"{API}/cmdb/views?page=1&pageSize=1")
    st, d, _ = call("POST", f"{API}/cmdb/views",
                    {"name": f"QA-MIGRATION-{stamp}", "description": "batch2 blackbox probe",
                     "filters": {"query": ""}, "isPublic": False}, csrf=True)
    view = (d.get("data") or {}) if isinstance(d, dict) else {}
    vid = view.get("id") or view.get("ID")
    record("创建 CMDB 保存视图", "POST", f"{API}/cmdb/views",
           st in (200, 201) and d.get("code") == 0 and bool(vid), st, d.get("code"),
           str(d.get("message")))
    if vid:
        CLEANUP_IDS["cmdbView"] = vid
        check(group, "读取新建保存视图", "GET", f"{API}/cmdb/views/{vid}")
        st2, d2, _ = call("DELETE", f"{API}/cmdb/views/{vid}", csrf=True)
        record("清理 CMDB 保存视图", "DELETE", f"{API}/cmdb/views/{vid}",
               d2.get("code") == 0, st2, d2.get("code"), str(d2.get("message")), "WARN")

    # dashboard / config_store.go：Load / Save 零变更回写
    st, d, _ = call("GET", f"{API}/dashboard/config")
    orig = d.get("data") if isinstance(d, dict) else None
    check(group, "仪表盘配置读取", "GET", f"{API}/dashboard/config")
    if isinstance(orig, dict):
        st2, d2, _ = call("POST", f"{API}/dashboard/config", orig, csrf=True)
        record("仪表盘配置零变更回写", "POST", f"{API}/dashboard/config",
               st2 in (200, 201) and d2.get("code") == 0, st2, d2.get("code"),
               str(d2.get("message")))
        st3, d3, _ = call("GET", f"{API}/dashboard/config")
        now = (d3.get("data") or {}) if isinstance(d3, dict) else {}
        same = (now.get("description") == orig.get("description")
                and len(now.get("widgets") or []) == len(orig.get("widgets") or []))
        record("仪表盘配置回读一致", "GET", f"{API}/dashboard/config",
               st3 == 200 and d3.get("code") == 0 and same, st3, d3.get("code"),
               f"description/widgets 回读不一致（same={same}）")
    else:
        record("仪表盘配置回读一致", "GET", f"{API}/dashboard/config", False, st,
               d.get("code"), "GET 未返回配置对象")

    # rbac / menu_service_impl.go
    check(group, "菜单管理列表", "GET", f"{API}/menus?page=1&pageSize=1")
    check(group, "用户可见菜单", "GET", f"{API}/auth/menus")
    # msp / allocation_service_impl.go：private 部署形态下 capability gate 会挡下，记 WARN
    check(group, "MSP 分配历史", "GET",
          f"{API}/msp/allocations/history?page=1&pageSize=1", severity="WARN")


def version_assert(expected: str) -> None:
    """硬版本指纹：/api/v1/version 的 APP_VERSION 必须由 HEAD 二进制注入。

    与 version_fingerprint() 互补：后者证明「行为文案是新源码」，本项证明
    「响应确实来自本次编译的实例」，两者都不能被镜像 CreatedAt 替代。
    """
    status, data, elapsed = call("GET", f"{API}/version")
    payload = data.get("data") if isinstance(data, dict) else None
    got = str((payload or {}).get("version") or "")
    ok = status == 200 and data.get("code") == 0 and got == expected
    RESULTS.append({"group": "版本指纹", "name": f"服务端实例版本 = {expected}",
                    "method": "GET", "path": f"{API}/version", "http": status,
                    "code": data.get("code"), "elapsed_ms": round(elapsed * 1000),
                    "result": "PASS" if ok else "FAIL",
                    "message": "" if ok else f"got={got!r} code={data.get('code')}"})


def write_flows() -> None:
    stamp = dt.datetime.now().strftime("%Y%m%d%H%M%S")
    title = f"QA-MIGRATION-{stamp}"
    # 知识文章：create -> read -> update -> search -> delete
    status, data, _ = call("POST", f"{API}/knowledge/articles",
                           {"title": title, "content": "service 迁移后功能回归临时数据",
                            "category": "operations", "tags": ["qa"]}, csrf=True)
    art = (data.get("data") or {}) if isinstance(data, dict) else {}
    art_id = art.get("id") or art.get("ID")
    ok = status in (200, 201) and data.get("code") == 0 and art_id
    RESULTS.append({"group": "写链路", "name": "创建知识文章", "method": "POST",
                    "path": f"{API}/knowledge/articles", "http": status,
                    "code": data.get("code"), "elapsed_ms": 0,
                    "result": "PASS" if ok else "FAIL",
                    "message": "" if ok else str(data.get("message"))})
    if ok:
        CLEANUP_IDS["article"] = art_id
        check("写链路", "读取新建文章", "GET", f"{API}/knowledge/articles/{art_id}")
        check("写链路", "更新文章", "PUT", f"{API}/knowledge/articles/{art_id}",
              {"title": title + "-updated"}, csrf=True)
        check("写链路", "文章版本", "GET", f"{API}/knowledge/articles/{art_id}/versions")
        check("写链路", "文章评论（已知未实现）", "GET",
              f"{API}/knowledge/articles/{art_id}/comments", severity="WARN")

    # 工单：create -> state-v2 -> 工作流动作 -> 状态回读
    status, data, _ = call("POST", f"{API}/tickets",
                           {"title": title, "description": "service 迁移后功能回归",
                            "priority": "medium", "type": "ticket"}, csrf=True)
    tk = (data.get("data") or {}) if isinstance(data, dict) else {}
    tk_id = tk.get("id") or tk.get("ID") or tk.get("ticketId")
    ok = status in (200, 201) and data.get("code") == 0 and tk_id
    RESULTS.append({"group": "写链路", "name": "创建工单", "method": "POST",
                    "path": f"{API}/tickets", "http": status, "code": data.get("code"),
                    "elapsed_ms": 0, "result": "PASS" if ok else "FAIL",
                    "message": "" if ok else str(data.get("message"))})
    if ok:
        CLEANUP_IDS["ticket"] = tk_id
        _, state, _ = call("GET", f"{API}/tickets/{tk_id}/workflow/state-v2")
        sdata = state.get("data") or {}
        actions = sdata.get("availableActions") or []
        check("写链路", "工作流状态机 state-v2", "GET",
              f"{API}/tickets/{tk_id}/workflow/state-v2")
        RESULTS.append({"group": "写链路", "name": "state-v2 返回可用动作",
                        "method": "GET", "path": f"{API}/tickets/{tk_id}/workflow/state-v2",
                        "http": 200, "code": state.get("code"), "elapsed_ms": 0,
                        "result": "PASS" if actions else "FAIL",
                        "message": "" if actions else "availableActions 为空，前端动作门控会 fail-closed"})
        check("写链路", "工作流状态 state", "GET",
              f"{API}/tickets/{tk_id}/workflow/state", severity="WARN")
        check("写链路", "工单详情", "GET", f"{API}/tickets/{tk_id}")

        # 负向门控：new 状态下 resolve 未开放，必须被拒绝（fail-closed）
        if "resolve" not in actions:
            s, d, _ = call("POST", f"{API}/tickets/workflow/resolve",
                           {"ticketId": tk_id, "comment": "QA gate probe"}, csrf=True)
            refused = d.get("code") != 0
            RESULTS.append({"group": "写链路", "name": "未开放动作 resolve 被拒绝",
                            "method": "POST", "path": f"{API}/tickets/workflow/resolve",
                            "http": s, "code": d.get("code"), "elapsed_ms": 0,
                            "result": "PASS" if refused else "FAIL",
                            "message": "" if refused else "门控失效：未开放动作被放行"})

        # 正向主链路：按 state-v2 的 availableActions 逐步推进（迁移改动面）
        for action in ("accept", "resolve", "close"):
            if action not in actions:
                RESULTS.append({"group": "写链路", "name": f"工作流动作 {action}",
                                "method": "POST", "path": f"{API}/tickets/workflow/{action}",
                                "http": 0, "code": None, "elapsed_ms": 0, "result": "BLOCKED",
                                "message": f"状态机未开放（availableActions={actions}）"})
                continue
            s, d, elapsed = call("POST", f"{API}/tickets/workflow/{action}",
                                 {"ticketId": tk_id, "comment": "QA regression"}, csrf=True)
            RESULTS.append({"group": "写链路", "name": f"工作流动作 {action}",
                            "method": "POST", "path": f"{API}/tickets/workflow/{action}",
                            "http": s, "code": d.get("code"),
                            "elapsed_ms": round(elapsed * 1000),
                            "result": "PASS" if d.get("code") == 0 else "FAIL",
                            "message": "" if d.get("code") == 0 else str(d.get("message"))})
            _, state, _ = call("GET", f"{API}/tickets/{tk_id}/workflow/state-v2")
            actions = (state.get("data") or {}).get("availableActions") or []

        _, final, _ = call("GET", f"{API}/tickets/{tk_id}")
        final_status = (final.get("data") or {}).get("status")
        RESULTS.append({"group": "写链路", "name": "工单终态回读",
                        "method": "GET", "path": f"{API}/tickets/{tk_id}",
                        "http": 200, "code": final.get("code"), "elapsed_ms": 0,
                        "result": "PASS" if final_status in ("resolved", "closed") else "BLOCKED",
                        "message": f"status={final_status}"})


def frontend_surface() -> None:
    for name, p in [("登录页", "/login"), ("仪表盘", "/dashboard"),
                    ("工单页", "/tickets"), ("知识库页", "/knowledge"),
                    ("能力矩阵页", "/admin/capabilities")]:
        status, data, elapsed = call("GET", FRONTEND_BASE + p)
        ok = status in (200, 302, 307)
        RESULTS.append({"group": "前端", "name": name, "method": "GET", "path": p,
                        "http": status, "code": None, "elapsed_ms": round(elapsed * 1000),
                        "result": "PASS" if ok else "FAIL",
                        "message": "" if ok else str(data)[:120]})


def cleanup() -> None:
    if "article" in CLEANUP_IDS:
        status, data, _ = call("DELETE", f"{API}/knowledge/articles/{CLEANUP_IDS['article']}",
                               csrf=True)
        RESULTS.append({"group": "清理", "name": "删除临时知识文章", "method": "DELETE",
                        "path": f"{API}/knowledge/articles/{CLEANUP_IDS['article']}",
                        "http": status, "code": data.get("code"), "elapsed_ms": 0,
                        "result": "PASS" if data.get("code") == 0 else "WARN",
                        "message": "" if data.get("code") == 0 else str(data.get("message"))})
    if "ticket" in CLEANUP_IDS:
        status, data, _ = call("DELETE", f"{API}/tickets/{CLEANUP_IDS['ticket']}", csrf=True)
        RESULTS.append({"group": "清理", "name": "删除临时工单", "method": "DELETE",
                        "path": f"{API}/tickets/{CLEANUP_IDS['ticket']}", "http": status,
                        "code": data.get("code"), "elapsed_ms": 0,
                        "result": "PASS" if data.get("code") == 0 else "WARN",
                        "message": "" if data.get("code") == 0 else "工单可能不提供删除端点，需人工清理"})


def main() -> int:
    global BASE, FRONTEND_BASE
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="http://127.0.0.1")
    ap.add_argument("--frontend-base", default="",
                    help="前端页面探测的独立基址（默认与 --base 相同）")
    ap.add_argument("--migration-deep", action="store_true",
                    help="追加本批迁移四域定向用例（含零副作用写回读与自动清理）")
    ap.add_argument("--expect-version", default="",
                    help="断言 /api/v1/version 的 APP_VERSION 等于该值（HEAD 实例硬指纹）")
    ap.add_argument("--json", default=str(REPO / "output" / "product-function-test.json"))
    ap.add_argument("--md", default=str(REPO / "output" / "product-function-test.md"))
    ap.add_argument("--expect-head", action="store_true",
                    help="启用能力矩阵行为指纹断言（仅在跑 HEAD 源码实例时使用）")
    args = ap.parse_args()
    BASE = args.base.rstrip("/")
    FRONTEND_BASE = (args.frontend_base or BASE).rstrip("/")

    login()
    check("平台", "健康检查", "GET", f"{API}/health")
    check("平台", "就绪检查 readyz", "GET", f"{API}/readyz", severity="WARN")
    check("平台", "当前用户", "GET", f"{API}/auth/me")
    check("平台", "菜单权限码", "GET", f"{API}/auth/menus")
    check("平台", "能力矩阵", "GET", f"{API}/capabilities")
    read_surface()
    migration_surface()
    if args.migration_deep:
        batch2_surface()
    if args.expect_head:
        version_fingerprint()
    if args.expect_version:
        version_assert(args.expect_version)
    write_flows()
    frontend_surface()
    cleanup()

    summary: dict[str, int] = {}
    for r in RESULTS:
        summary[r["result"]] = summary.get(r["result"], 0) + 1
    payload = {"generatedAt": dt.datetime.now().isoformat(timespec="seconds"),
               "base": BASE, "frontendBase": FRONTEND_BASE,
               "expectedVersion": args.expect_version, "expectHead": args.expect_head,
               "migrationDeep": args.migration_deep,
               "summary": summary, "cleanupIds": CLEANUP_IDS, "results": RESULTS}
    Path(args.json).parent.mkdir(parents=True, exist_ok=True)
    Path(args.json).write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")

    lines = ["# ITSM 产品功能回归报告", "",
             f"- 生成时间：{payload['generatedAt']}",
             f"- 目标环境：`{BASE}`（API / 后端）",
             f"- 前端探测：`{FRONTEND_BASE}`",
             f"- 版本指纹：`{args.expect_version or '未启用'}`"
             f" / 行为指纹：{'启用' if args.expect_head else '未启用'}"
             f" / 迁移深测：{'启用' if args.migration_deep else '未启用'}",
             f"- 结论：PASS {summary.get('PASS', 0)} / FAIL {summary.get('FAIL', 0)} / "
             f"WARN {summary.get('WARN', 0)} / BLOCKED {summary.get('BLOCKED', 0)}", ""]
    groups: dict[str, list[dict[str, Any]]] = {}
    for r in RESULTS:
        groups.setdefault(r["group"], []).append(r)
    for g, items in groups.items():
        lines += [f"## {g}", "", "| 用例 | 方法 | 路径 | HTTP | code | 耗时 | 结论 |",
                  "|---|---|---|---:|---:|---:|---|"]
        for r in items:
            lines.append(f"| {r['name']} | {r['method']} | `{r['path']}` | {r['http']} | "
                         f"{r['code'] if r['code'] is not None else '-'} | {r['elapsed_ms']}ms | "
                         f"{r['result']} |")
        lines.append("")
    fails = [r for r in RESULTS if r["result"] in ("FAIL", "WARN", "BLOCKED")]
    if fails:
        lines += ["## 异常明细", ""]
        for r in fails:
            lines.append(f"- **{r['result']}** {r['group']} / {r['name']}：`{r['path']}` "
                         f"HTTP={r['http']} code={r['code']} — {r['message']}")
    Path(args.md).write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(f"\nSUMMARY {summary}  json={args.json}  md={args.md}", flush=True)
    return 1 if summary.get("FAIL", 0) else 0


if __name__ == "__main__":
    sys.exit(main())
