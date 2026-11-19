#!/usr/bin/env python3
"""prod 新功能点专项黑盒测试。

覆盖面（2026-11 权限码运行期策略 / Gate 2.5 AI 主体零信任 / Skill 持久化 /
调查编排 / BPMN F1 关单终止实例）：
  ① 部署指纹：镜像 tag、seed 日志、skill_definitions 行数（DB 直查）
  ② 权限运行期策略：列表策略字段、meta、PUT 即时生效、ValidatePolicy 负例
  ③ Gate 2.5：写工具审批队列、ai.agent_tool 技能的 agent 主体默认 deny、
     显式授权后 pending_agent + 发起人自批拦截
  ④ Skill 持久化：自定义技能 CRUD 双写落库（DB 直查）、Invoke 决策底账
  ⑤ 调查编排：POST /ai/tickets/:id/investigate 结构化结果 + 重跑留痕
  ⑥ BPMN F1：POST /tickets/:id/close 同事务终止流程实例（DB 直查）

不硬编码凭据：ADMIN_PASSWORD 从 .env.prod 读取；DB 探针走 docker exec
psql（只读 SELECT）；写操作产生的工单/技能/策略改动在用例末尾清理或还原。
"""
from __future__ import annotations

import argparse
import datetime as dt
import http.cookiejar
import json
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any

REPO = Path(__file__).resolve().parents[2]
API = "/api/v1"
CONTAINER = "itsm-postgres-prod"
BACKEND_CONTAINER = "itsm-backend-prod"

RESULTS: list[dict[str, Any]] = []
OPENER: urllib.request.OpenerDirector | None = None
BASE = "http://127.0.0.1"
FRONTEND_BASE = ""
EXPECT_IMAGE = ""
TENANT_ID = 0
# 清理登记：策略还原快照 / 临时技能 code / 临时工单 id / 待审批 invocation
SAVED_POLICY: dict[str, Any] = {}
TEMP_SKILL = ""
TEMP_TICKET = 0


def record(group: str, name: str, ok: bool, *, method: str = "", path: str = "",
           http: int | None = None, code: Any = None, elapsed_ms: int = 0,
           message: str = "", severity: str = "FAIL") -> dict[str, Any]:
    item = {"group": group, "name": name, "method": method, "path": path,
            "http": http, "code": code, "elapsed_ms": elapsed_ms,
            "result": "PASS" if ok else severity, "message": message}
    RESULTS.append(item)
    print(f"[{item['result']:5}] {group:12} {name:44}"
          + (f" HTTP={http} code={code} {elapsed_ms}ms" if http is not None else "")
          + (f"  {message}" if message else ""), flush=True)
    return item


def env_password(env_file: Path) -> str:
    for line in env_file.read_text(encoding="utf-8").splitlines():
        if line.startswith("ADMIN_PASSWORD="):
            return line.split("=", 1)[1].strip()
    raise SystemExit("ADMIN_PASSWORD missing in .env.prod")


def call(method: str, path: str, body: Any = None, csrf: bool = False,
         timeout: float = 25.0) -> tuple[int, Any, float]:
    url = path if path.startswith("http") else BASE + path
    headers = {"Content-Type": "application/json"}
    if csrf:
        code, data, _ = call("GET", f"{API}/csrf-token", timeout=timeout)
        token = (data or {}).get("data", {}).get("csrfToken", "") if code == 200 else ""
        headers["X-CSRF-Token"] = token
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
    except Exception as e:  # noqa: BLE001
        return 0, {"error": str(e)}, time.monotonic() - started
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError:
        parsed = {"raw": raw[:300]}
    return status, parsed, time.monotonic() - started


def http_check(group: str, name: str, method: str, path: str, body: Any = None,
               csrf: bool = False, allow: tuple[int, ...] = (200, 201),
               biz_codes: tuple[int, ...] = (0,), severity: str = "FAIL") -> dict[str, Any]:
    status, data, elapsed = call(method, path, body, csrf=csrf)
    biz = data.get("code") if isinstance(data, dict) else None
    ok = status in allow and (biz is None or biz in biz_codes)
    return record(group, name, ok, method=method, path=path, http=status, code=biz,
                  elapsed_ms=round(elapsed * 1000),
                  message=("" if ok else str(data.get("message") if isinstance(data, dict) else data))[:200],
                  severity=severity)


def db(sql: str) -> str:
    """只读探针：docker exec psql -tAc。失败返回空串并记 WARN。"""
    cmd = ["docker", "exec", CONTAINER, "sh", "-c",
           f'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tAc "{sql}"']
    try:
        out = subprocess.run(cmd, capture_output=True, text=True, timeout=20,
                             env={"PATH": "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"})
        return (out.stdout or "").strip()
    except Exception as e:  # noqa: BLE001
        record("DB探针", "psql 可达性", False, message=str(e)[:160], severity="WARN")
        return ""


def prod_log_level() -> str:
    """prod .env.prod 的 LOG_LEVEL；用于解释日志断言的可见性。"""
    try:
        for line in (REPO / ".env.prod").read_text(encoding="utf-8").splitlines():
            if line.startswith("LOG_LEVEL="):
                return line.split("=", 1)[1].strip().lower()
    except Exception:  # noqa: BLE001
        pass
    return ""


def backend_log_has(needle: str) -> bool:
    try:
        out = subprocess.run(
            ["docker", "logs", BACKEND_CONTAINER],
            capture_output=True, text=True, timeout=30,
            env={"PATH": "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"})
        blob = (out.stdout or "") + (out.stderr or "")
        return needle in blob
    except Exception:  # noqa: BLE001
        return False


def running_image_tag() -> str:
    try:
        out = subprocess.run(
            ["docker", "inspect", BACKEND_CONTAINER, "--format", "{{.Config.Image}}"],
            capture_output=True, text=True, timeout=20,
            env={"PATH": "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"})
        return (out.stdout or "").strip()
    except Exception:  # noqa: BLE001
        return ""


def login() -> None:
    global OPENER, TENANT_ID
    jar = http.cookiejar.CookieJar()
    OPENER = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
    pwd = env_password(REPO / ".env.prod")
    status, data, elapsed = call("POST", f"{API}/auth/login",
                                 {"username": "admin", "password": pwd})
    ok = status == 200 and data.get("code") == 0
    record("平台", "管理员登录（httpOnly cookie）", ok, method="POST",
           path=f"{API}/auth/login", http=status, code=data.get("code"),
           elapsed_ms=round(elapsed * 1000))
    if not ok:
        raise SystemExit("login failed, abort")
    d = data.get("data") or {}
    # 登录响应把身份挂在 data.user 下（data 顶层只有 expiresIn/user）。
    user = d.get("user") or {}
    TENANT_ID = int(user.get("tenantId") or d.get("tenantId") or 0)
    if TENANT_ID <= 0:
        record("平台", "解析租户上下文", False,
               message=f"tenantId 缺失: data={list(d.keys())[:8]} user={list(user.keys())[:8]}")


# ---------------------------------------------------------------------------
# ① 部署指纹
# ---------------------------------------------------------------------------
def deploy_fingerprint() -> None:
    tag = running_image_tag()
    record("部署指纹", "运行镜像 tag 记录", True, message=tag or "docker inspect 失败", severity="WARN")
    if EXPECT_IMAGE:
        record("部署指纹", "镜像与期望版本一致", tag == EXPECT_IMAGE,
               message=f"running={tag} expect={EXPECT_IMAGE}",
               severity="FAIL" if tag else "WARN")
    rows = db("SELECT count(*) FROM skill_definitions")
    n = int(rows or 0)
    record("部署指纹", "skill_definitions 种子行数 > 0（批次 C seed 已执行）", n > 0,
           method="DB", path="skill_definitions", message=f"rows={rows}")
    # seed 成功日志是 Info 级：prod 用 LOG_LEVEL=error 会主动吞掉它。
    # 这不是部署缺陷——此时以 skill_definitions 行数作为部署证据，标记 WARN 说明原因。
    log_level = prod_log_level()
    quiet_levels = {"error", "fatal", "panic"}
    seed_logged = backend_log_has("builtin skills seeded")
    if seed_logged:
        record("部署指纹", "启动日志含 builtin skills seeded", True,
               method="logs", path=BACKEND_CONTAINER)
    elif log_level in quiet_levels and n > 0:
        record("部署指纹", "启动日志含 builtin skills seeded", False,
               method="logs", path=BACKEND_CONTAINER, severity="WARN",
               message=f"LOG_LEVEL={log_level} 抑制 Info 级日志；以 DB rows={rows} 作为 seed 已执行证据")
    else:
        record("部署指纹", "启动日志含 builtin skills seeded", False,
               method="logs", path=BACKEND_CONTAINER,
               message=f"未见 seed 日志且 LOG_LEVEL={log_level or '-'} rows={rows}")
    record("部署指纹", "启动日志无 seed failed", not backend_log_has("builtin skill seed failed"),
           method="logs", path=BACKEND_CONTAINER)
    http_check("部署指纹", "健康检查", "GET", f"{API}/health")


# ---------------------------------------------------------------------------
# ② 权限运行期策略
# ---------------------------------------------------------------------------
POLICY_KEYS = ("riskLevel", "allowedPrincipals", "requiresApproval",
               "minApprovals", "isActive", "isSystem", "configured")


def put_policy(perm_id: int, item: dict[str, Any], *, risk: str, principals: list[str],
               requires: bool, min_appr: int) -> tuple[int, dict[str, Any], float]:
    payload = {"policy": {
        "riskLevel": risk,
        "allowedPrincipals": principals,
        "requiresApproval": requires,
        "minApprovals": min_appr,
        "category": item.get("category") or "core",
        "displayName": item.get("displayName") or "",
        "isActive": bool(item.get("isActive", True)),
    }}
    return call("PUT", f"{API}/permissions/{perm_id}", payload, csrf=True)


def permission_policy() -> None:
    status, data, elapsed = call("GET", f"{API}/permissions?pageSize=500")
    items = ((data.get("data") or {}).get("items") or []) if isinstance(data, dict) else []
    ok = status == 200 and data.get("code") == 0 and len(items) > 0
    record("权限策略", "GET /permissions 列表非空", ok, method="GET",
           path=f"{API}/permissions", http=status, code=data.get("code"),
           elapsed_ms=round(elapsed * 1000))
    if not items:
        return
    missing = {k for it in items for k in POLICY_KEYS if k not in it}
    record("权限策略", "全部 items 携带运行期策略字段（7 键）", not missing,
           message=f"缺失键={sorted(missing)}" if missing else f"覆盖 {len(items)} 条")

    meta_status, meta, _ = call("GET", f"{API}/permissions/meta")
    mmax = (meta.get("data") or {}).get("minApprovalsMax") if isinstance(meta, dict) else None
    record("权限策略", "GET /permissions/meta 返回 minApprovalsMax",
           meta_status == 200 and meta.get("code") == 0 and isinstance(mmax, int) and mmax >= 2,
           method="GET", path=f"{API}/permissions/meta", http=meta_status,
           code=meta.get("code"), message=f"minApprovalsMax={mmax}")

    # 选一条低风险读码做正向 PUT + 即时生效 + 还原
    target = next((it for it in items if it.get("code") == "knowledge:read"), None)
    if target is None:
        record("权限策略", "定位 knowledge:read", False, message="权限目录无该码", severity="BLOCKED")
        return
    saved = {k: target.get(k) for k in
             ("riskLevel", "allowedPrincipals", "requiresApproval", "minApprovals",
              "category", "displayName", "isActive")}
    SAVED_POLICY.update({"id": target["id"], "item": saved})
    pid = target["id"]
    st, d, el = put_policy(pid, target, risk="low", principals=["human"],
                           requires=False, min_appr=0)
    ok = st in (200, 201) and d.get("code") == 0
    record("权限策略", "PUT 合法策略（low/human/无审批）", ok, method="PUT",
           path=f"{API}/permissions/{pid}", http=st, code=d.get("code"),
           elapsed_ms=round(el * 1000))
    if ok:
        st2, d2, _ = call("GET", f"{API}/permissions?pageSize=500")
        fresh = next((it for it in ((d2.get("data") or {}).get("items") or [])
                      if it.get("id") == pid), {})
        record("权限策略", "PUT 后回读即时生效（TTL 缓存同实例失效）",
               fresh.get("allowedPrincipals") == ["human"] and fresh.get("requiresApproval") is False,
               message=f"allowedPrincipals={fresh.get('allowedPrincipals')} requiresApproval={fresh.get('requiresApproval')}")

    # ValidatePolicy 负例（闭集规则，必须被拒且不入库）
    neg: list[tuple[str, int, Any]] = []
    st, d, _ = put_policy(pid, target, risk="low", principals=["human"],
                          requires=True, min_appr=1)
    neg.append(("低风险强制审批", st, d.get("code")))
    record("权限策略", "负例：低风险不得强制人工审批", d.get("code") != 0,
           method="PUT", path=f"{API}/permissions/{pid}", http=st, code=d.get("code"),
           message=("" if d.get("code") != 0 else "被放行——闭集校验失效"))
    st, d, _ = put_policy(pid, target, risk="banana", principals=["human"],
                          requires=False, min_appr=0)
    neg.append(("非法 riskLevel", st, d.get("code")))
    record("权限策略", "负例：非法 riskLevel 被拒", d.get("code") != 0,
           method="PUT", path=f"{API}/permissions/{pid}", http=st, code=d.get("code"),
           message=("" if d.get("code") != 0 else "非法枚举被放行"))
    st, d, _ = put_policy(pid, target, risk="medium", principals=["human", "agent"],
                          requires=False, min_appr=0)
    neg.append(("AI 中风险未开审批", st, d.get("code")))
    record("权限策略", "负例：AI 执行中风险必须开审批", d.get("code") != 0,
           method="PUT", path=f"{API}/permissions/{pid}", http=st, code=d.get("code"),
           message=("" if d.get("code") != 0 else "零信任门禁失效"))
    # 拒绝语义：闭集校验失败是调用方参数问题，应为 400/1001；
    # 当前 handler 用 FailWithErr 兜底成 500/5001，前端无法区分「填错」与「服务故障」。
    sem_ok = all(s == 400 and c == 1001 for _, s, c in neg)
    record("权限策略", "负例拒绝语义为 400/1001（参数错误）而非 500/5001", sem_ok,
           method="PUT", path=f"{API}/permissions/{pid}",
           message="" if sem_ok else "; ".join(f"{n}: http={s} code={c}" for n, s, c in neg))


def restore_policy() -> None:
    if "id" not in SAVED_POLICY:
        return
    pid = SAVED_POLICY["id"]
    saved = SAVED_POLICY["item"]
    payload = {"policy": {
        "riskLevel": saved["riskLevel"],
        "allowedPrincipals": saved["allowedPrincipals"],
        "requiresApproval": bool(saved["requiresApproval"]),
        "minApprovals": int(saved["minApprovals"] or 0),
        "category": saved["category"] or "core",
        "displayName": saved.get("displayName") or "",
        "isActive": bool(saved["isActive"]),
    }}
    st, d, _ = call("PUT", f"{API}/permissions/{pid}", payload, csrf=True)
    record("清理", "还原权限策略快照", st in (200, 201) and d.get("code") == 0,
           method="PUT", path=f"{API}/permissions/{pid}", http=st, code=d.get("code"),
           severity="WARN")


# ---------------------------------------------------------------------------
# ③ Gate 2.5：AI 主体零信任
# ---------------------------------------------------------------------------
def grant_agent(code_id: int, item: dict[str, Any]) -> tuple[dict[str, Any], dict[str, Any]]:
    """把 agent 加进指定码的 allowedPrincipals（中风险+双人改单人按现值保守处理）。"""
    saved = {k: item.get(k) for k in
             ("riskLevel", "allowedPrincipals", "requiresApproval", "minApprovals",
              "category", "displayName", "isActive")}
    risk = saved["riskLevel"] or "medium"
    requires = True
    min_appr = int(saved["minApprovals"] or 0)
    if risk in ("high", "critical"):
        min_appr = max(min_appr, 2)
    elif min_appr < 1:
        min_appr = 1
    st, d, _ = put_policy(code_id, item, risk=risk,
                          principals=sorted(set((saved["allowedPrincipals"] or []) + ["agent"])),
                          requires=requires, min_appr=min_appr)
    return saved, {"status": st, "data": d}


def gate_agent_principal() -> None:
    # 定位写工具与对应权限码
    st, d, _ = call("GET", f"{API}/agent/tools")
    tools = ((d.get("data") or {}).get("items") or []) if isinstance(d, dict) else []
    write_tool = next((t for t in tools if t.get("name") == "link_ticket_ci"), None)
    if write_tool is None:
        write_tool = next((t for t in tools if t.get("readOnly") is False), None)
    if write_tool is None:
        record("Gate2.5", "定位写工具", False, message=f"可见工具 {len(tools)} 个中无写工具", severity="BLOCKED")
        return
    tool_name = write_tool["name"]
    resource = write_tool.get("resource") or ""
    action = write_tool.get("action") or ""
    code = f"{resource}:{action}"
    record("Gate2.5", "定位写工具（agent 主体测试靶）", True,
           message=f"tool={tool_name} code={code} risk={write_tool.get('riskLevel')}")

    # 1) 人发起写工具 → pending 审批队列（含策略摘要字段）
    st, d, el = call("POST", f"{API}/agent/tools/execute",
                     {"name": tool_name, "args": {}}, csrf=True)
    inv = (d.get("data") or {}) if isinstance(d, dict) else {}
    inv_id = inv.get("invocationId") or 0
    ok = st == 200 and d.get("code") == 0 and inv.get("status") == "pending" and inv_id > 0
    record("Gate2.5", "人发起写工具 → pending 审批队列", ok, method="POST",
           path=f"{API}/agent/tools/execute", http=st, code=d.get("code"),
           elapsed_ms=round(el * 1000),
           message="" if ok else str(d.get("message"))[:160])
    if inv_id > 0:
        st2, d2, _ = call("GET", f"{API}/agent/tools/invocations?state=pending")
        lst = ((d2.get("data") or {}).get("items") or []) if isinstance(d2, dict) else []
        found = next((x for x in lst if x.get("invocationId") == inv_id or x.get("id") == inv_id), None)
        record("Gate2.5", "invocations?state=pending 可见该调用", found is not None,
               method="GET", path=f"{API}/agent/tools/invocations?state=pending",
               http=st2, code=d2.get("code"))
        if found is not None:
            has_meta = "riskLevel" in found and "minApprovals" in found
            record("Gate2.5", "invocation 携带 riskLevel/minApprovals 策略摘要", has_meta,
                   message=f"riskLevel={found.get('riskLevel')} minApprovals={found.get('minApprovals')} approvalState={found.get('approvalState')}")

    # 2) agent 主体默认 deny：ai.agent_tool 技能调用写工具（未授权）
    st, d, el = call("POST", f"{API}/admin/skills/ai.agent_tool/invoke",
                     {"input": {"name": tool_name, "args": {}}}, csrf=True)
    denied = st >= 400 or (isinstance(d, dict) and d.get("code") not in (0, None))
    msg = str((d or {}).get("message") if isinstance(d, dict) else d)[:160]
    record("Gate2.5", "agent 主体默认零权限：技能调用写工具被拒", denied,
           method="POST", path=f"{API}/admin/skills/ai.agent_tool/invoke",
           http=st, code=d.get("code") if isinstance(d, dict) else None,
           elapsed_ms=round(el * 1000), message=msg if denied else "被放行——零信任失效")
    # 拒绝语义：权限拒绝应映射 forbidden(2003/403)，而不是内部错误 5001。
    # 当前实现把 registry 的 tool-permission-denied 归入 ErrSkillInvoke → 500，
    # fail-closed 安全目标达成，但操作者看不到「谁缺什么授权」。
    forbidden_semantics = st == 403 and (d.get("code") == 2003 if isinstance(d, dict) else False)
    record("Gate2.5", "拒绝语义映射 forbidden(2003/403) 而非 internal(5001)",
           forbidden_semantics, method="POST",
           path=f"{API}/admin/skills/ai.agent_tool/invoke", http=st,
           code=d.get("code") if isinstance(d, dict) else None,
           message=("" if forbidden_semantics else
                    f"拒绝已生效但返回 http={st} code={d.get('code') if isinstance(d, dict) else None}，"
                    "操作者无法区分「无授权」与「服务异常」"))

    # 3) 显式授权 agent → pending_agent 队列 + 发起人自批拦截
    st0, d0, _ = call("GET", f"{API}/permissions?pageSize=500")
    perm_item = next((it for it in ((d0.get("data") or {}).get("items") or [])
                      if it.get("code") == code), None)
    if perm_item is None:
        record("Gate2.5", "定位权限码", False, message=f"目录无 {code}", severity="BLOCKED")
        return
    saved, res = grant_agent(perm_item["id"], perm_item)
    granted = res["status"] in (200, 201) and res["data"].get("code") == 0
    SAVED_POLICY.setdefault("agent_grant", {"id": perm_item["id"], "item": saved})
    record("Gate2.5", f"显式授权 agent 执行 {code}", granted,
           method="PUT", path=f"{API}/permissions/{perm_item['id']}",
           http=res["status"], code=res["data"].get("code"),
           message=("" if granted else str(res["data"].get("message"))[:160]))
    if granted:
        st, d, el = call("POST", f"{API}/admin/skills/ai.agent_tool/invoke",
                         {"input": {"name": tool_name, "args": {}}}, csrf=True)
        # Skill 统一响应形状：{code, output, latencyMs, metricsTracked, isPilot}，
        # 工具调用结果在 output 里（不是顶层字段）。ai.agent_tool 命中审批时
        # output={"status":"approval_pending"}，invocationId 只在 /agent/tools 侧可见。
        inv = (d.get("data") or {}) if isinstance(d, dict) else {}
        output = inv.get("output") or {}
        st_state = output.get("status") if isinstance(output, dict) else None
        ok = st == 200 and d.get("code") == 0 and st_state in ("approval_pending", "pending")
        record("Gate2.5", "授权后 agent 调用进入审批（approval_pending）", ok,
               method="POST", path=f"{API}/admin/skills/ai.agent_tool/invoke",
               http=st, code=d.get("code"), elapsed_ms=round(el * 1000),
               message=("" if ok else
                        f"invoke code={d.get('code')} output={output} msg={str(d.get('message'))[:120]}"))
        # 从待审批队列按「本轮新增 + pending_agent」定位该调用（响应不带 id）。
        st2, d2, _ = call("GET", f"{API}/agent/tools/invocations?state=pending")
        lst = ((d2.get("data") or {}).get("items") or []) if isinstance(d2, dict) else []
        agent_rows = [x for x in lst
                      if str(x.get("approvalState", "")).startswith("pending_agent")
                      and int(x.get("id") or x.get("invocationId") or 0) > (inv_id or 0)]
        found = max(agent_rows, key=lambda x: int(x.get("id") or 0), default=None)
        agent_inv = int(found.get("id") or 0) if found else 0
        state_ok = found is not None
        record("Gate2.5", "审批状态机 pending_agent（非人类 pending）", state_ok,
               method="GET", path=f"{API}/agent/tools/invocations?state=pending",
               http=st2, code=d2.get("code") if isinstance(d2, dict) else None,
               message="" if state_ok else f"待审批队列中未见 pending_agent 行（items={len(lst)}）")
        if agent_inv > 0:
            if found is not None:
                has_meta = "riskLevel" in found and "minApprovals" in found
                record("Gate2.5", "agent invocation 携带 riskLevel/minApprovals 策略摘要", has_meta,
                       message=f"riskLevel={found.get('riskLevel')} minApprovals={found.get('minApprovals')} "
                               f"permissionReason={found.get('permissionReason')}")
            # 发起人自批拦截：admin 是发起人（Invoke 侧注入 userId）
            st3, d3, _ = call("POST", f"{API}/agent/tools/{agent_inv}/approve",
                              {"approve": True, "reason": "QA self-approval probe"}, csrf=True)
            blocked = st3 >= 400 or (isinstance(d3, dict) and d3.get("code") not in (0, None))
            record("Gate2.5", "发起人自批被拦截（fail-closed）", blocked,
                   method="POST", path=f"{API}/agent/tools/{agent_inv}/approve",
                   http=st3, code=d3.get("code") if isinstance(d3, dict) else None,
                   message=("" if blocked else "自批被放行——双人/自批门禁失效"))


def restore_agent_grant() -> None:
    g = SAVED_POLICY.get("agent_grant")
    if not g:
        return
    saved = g["item"]
    payload = {"policy": {
        "riskLevel": saved["riskLevel"],
        "allowedPrincipals": saved["allowedPrincipals"],
        "requiresApproval": bool(saved["requiresApproval"]),
        "minApprovals": int(saved["minApprovals"] or 0),
        "category": saved["category"] or "core",
        "displayName": saved.get("displayName") or "",
        "isActive": bool(saved["isActive"]),
    }}
    st, d, _ = call("PUT", f"{API}/permissions/{g['id']}", payload, csrf=True)
    record("清理", f"还原 agent 授权策略（{saved['riskLevel']}）", st in (200, 201) and d.get("code") == 0,
           method="PUT", path=f"{API}/permissions/{g['id']}", http=st, code=d.get("code"),
           severity="WARN")


# ---------------------------------------------------------------------------
# ④ Skill 持久化
# ---------------------------------------------------------------------------
def skill_persistence() -> None:
    global TEMP_SKILL
    stamp = dt.datetime.now().strftime("%H%M%S")
    TEMP_SKILL = f"qa_custom_{stamp}"
    body = {"code": TEMP_SKILL, "version": "v1", "title": "QA 临时技能",
            "description": "prod 功能点测试临时数据", "category": "pilot",
            "requiredPermissions": ["knowledge:read"],
            "inputSchema": {"type": "object", "properties": {}}}
    st, d, el = call("POST", f"{API}/admin/skills", body, csrf=True)
    ok = st in (200, 201) and d.get("code") == 0
    record("Skill持久化", "注册自定义技能（CRUD 双写：先库后内存）", ok,
           method="POST", path=f"{API}/admin/skills", http=st, code=d.get("code"),
           elapsed_ms=round(el * 1000), message=("" if ok else str(d.get("message"))[:160]))
    if not ok:
        return
    rows = db(f"SELECT count(*) FROM skill_definitions WHERE code='{TEMP_SKILL}'")
    record("Skill持久化", "DB skill_definitions 立即落行", rows == "1",
           method="DB", path="skill_definitions", message=f"rows={rows}")
    st2, d2, _ = call("GET", f"{API}/admin/skills/{TEMP_SKILL}")
    record("Skill持久化", "GET /admin/skills/:code 可读回", st2 == 200 and d2.get("code") == 0,
           method="GET", path=f"{API}/admin/skills/{TEMP_SKILL}", http=st2, code=d2.get("code"))
    # 更新 version，验证 UPSERT（checksum 变化 → 更新而非新增）
    st3, d3, _ = call("PUT", f"{API}/admin/skills/{TEMP_SKILL}",
                      {**body, "version": "v2", "title": "QA 临时技能 v2"}, csrf=True)
    rows2 = db(f"SELECT version FROM skill_definitions WHERE code='{TEMP_SKILL}'")
    record("Skill持久化", "PUT 更新即时落库（UPSERT）", st3 in (200, 201) and d3.get("code") == 0 and rows2 == "v2",
           method="PUT", path=f"{API}/admin/skills/{TEMP_SKILL}", http=st3, code=d3.get("code"),
           message=f"db_version={rows2}")
    # 决策底账（批次 B 装配）：底账落在 ai_feedbacks（item_type='ai_audit'，
    # kind='skill:<code>'），不存在名为 ai_audit 的表。
    st4, d4, _ = call("POST", f"{API}/admin/skills/{TEMP_SKILL}/invoke", {"input": {}}, csrf=True)
    n_audit = db("SELECT count(*) FROM ai_feedbacks "
                 f"WHERE item_type='ai_audit' AND kind='skill:{TEMP_SKILL}'")
    record("Skill持久化", "Invoke 失败路径仍写决策底账（ai_feedbacks/ai_audit）",
           bool(n_audit) and int(n_audit) >= 1,
           method="DB", path="ai_feedbacks", message=f"rows={n_audit} invoke_http={st4}")


def cleanup_skill() -> None:
    global TEMP_SKILL
    if not TEMP_SKILL:
        return
    st, d, _ = call("DELETE", f"{API}/admin/skills/{TEMP_SKILL}", csrf=True)
    row = db(f"SELECT count(*) FROM skill_definitions WHERE code='{TEMP_SKILL}'")
    ok = st in (200, 201, 204) and d.get("code") in (0, None)
    record("清理", "删除临时技能（运行期 + 持久化）", ok, method="DELETE",
           path=f"{API}/admin/skills/{TEMP_SKILL}", http=st, code=d.get("code"), severity="WARN")
    if ok:
        # 实现在 handlers/skill/persistence.go DeleteDefinition 里是硬删
        # （DELETE FROM skill_definitions）。断言方式：先断言行为（DB 行已移除），
        # 再检查 handler 注释是否与行为一致——注释漂移属文档缺陷，不阻断功能。
        record("清理", "DB 行已移除（DELETE FROM skill_definitions）", row == "0",
               method="DB", path="skill_definitions", message=f"rows={row}")
        handler_src = REPO / "itsm-backend/handlers/skill/handler.go"
        doc_synced = False
        doc_msg = "未找到 handler 源码"
        if handler_src.exists():
            text = handler_src.read_text(encoding="utf-8")
            idx = text.find("func (h *Handler) Delete(c *gin.Context)")
            block = text[max(0, idx - 1200):idx] if idx > 0 else ""
            claims_soft = ("标记为 disabled" in block) or ("软删" in block)
            says_hard = ("硬删" in block) or ("硬删除" in block)
            doc_synced = says_hard and not claims_soft
            doc_msg = ("注释已声明硬删，与实现一致" if doc_synced
                       else "注释仍声称软删/disabled，与硬删实现不一致")
        record("清理", "handler.Delete 注释与实现一致（硬删，历史保留）", doc_synced,
               severity="PASS" if doc_synced else "WARN", message=doc_msg)
    TEMP_SKILL = ""


# ---------------------------------------------------------------------------
# ⑤ 调查编排
# ---------------------------------------------------------------------------
def investigation_orchestration() -> None:
    global TEMP_TICKET
    stamp = dt.datetime.now().strftime("%Y%m%d%H%M%S")
    st, d, _ = call("POST", f"{API}/tickets",
                    {"title": f"QA-INVESTIGATE-{stamp}", "description": "调查编排回归：磁盘告警后 IO 延迟升高",
                     "priority": "medium", "type": "ticket"}, csrf=True)
    tk = (d.get("data") or {}) if isinstance(d, dict) else {}
    tid = tk.get("id") or tk.get("ID") or 0
    ok = st in (200, 201) and d.get("code") == 0 and tid
    record("调查编排", "创建测试工单", ok, method="POST", path=f"{API}/tickets",
           http=st, code=d.get("code"))
    if not ok:
        return
    TEMP_TICKET = int(tid)
    st2, d2, el = call("POST", f"{API}/ai/tickets/{tid}/investigate", {}, csrf=True)
    res = (d2.get("data") or {}) if isinstance(d2, dict) else {}
    steps = res.get("steps") or []
    step_names = {s.get("step") for s in steps} if steps else set()
    ok2 = st2 == 200 and d2.get("code") == 0 and res.get("ticketId") == int(tid)
    record("调查编排", "POST investigate 返回结构化结果", ok2, method="POST",
           path=f"{API}/ai/tickets/{tid}/investigate", http=st2, code=d2.get("code"),
           elapsed_ms=round(el * 1000), message=("" if ok2 else str(d2.get("message"))[:160]))
    if ok2:
        record("调查编排", "步骤覆盖 triage/knowledge/cmdb/synthesis（或显式降级）",
               {"triage", "synthesis"} <= step_names,
               message=f"steps={sorted(step_names)} status={res.get('status')}")
        record("调查编排", "结果带 promptVersion/model 溯源",
               bool(res.get("promptVersion")),
               message=f"promptVersion={res.get('promptVersion')} model={res.get('model')}")
    # 重跑：第二次 investigate 应同样成功（编排幂等，不依赖上一次状态）
    st3, d3, el3 = call("POST", f"{API}/ai/tickets/{tid}/investigate", {}, csrf=True)
    record("调查编排", "重跑 investigate 再次成功", st3 == 200 and d3.get("code") == 0,
           method="POST", path=f"{API}/ai/tickets/{tid}/investigate", http=st3,
           code=d3.get("code"), elapsed_ms=round(el3 * 1000))


# ---------------------------------------------------------------------------
# ⑥ BPMN F1：关单终止实例
# ---------------------------------------------------------------------------
def bpmn_f1_close_termination() -> None:
    global TEMP_TICKET
    if not TEMP_TICKET:
        stamp = dt.datetime.now().strftime("%Y%m%d%H%M%S")
        st, d, _ = call("POST", f"{API}/tickets",
                        {"title": f"QA-F1-{stamp}", "description": "F1 关单终止实例回归",
                         "priority": "low", "type": "ticket"}, csrf=True)
        tk = (d.get("data") or {}) if isinstance(d, dict) else {}
        TEMP_TICKET = int(tk.get("id") or 0)
        if not TEMP_TICKET:
            record("BPMN F1", "创建测试工单", False, message=str(d.get("message"))[:160], severity="BLOCKED")
            return
    bk = f"ticket:{TEMP_TICKET}"
    # 状态机单一事实源：new 不能直达 resolved/closed。
    # 合法收敛链 new → in_progress → resolved，之后才允许 closed。
    st_s, d_s, _ = call("PUT", f"{API}/tickets/{TEMP_TICKET}/status",
                        {"status": "in_progress"}, csrf=True)
    advanced = st_s in (200, 201) and d_s.get("code") == 0
    record("BPMN F1", "前置：new → in_progress（状态机合法路径）", advanced,
           method="PUT", path=f"{API}/tickets/{TEMP_TICKET}/status", http=st_s,
           code=d_s.get("code") if isinstance(d_s, dict) else None,
           message="" if advanced else str(d_s.get("message"))[:160])
    if not advanced:
        return
    st_r, d_r, _ = call("POST", f"{API}/tickets/{TEMP_TICKET}/resolve",
                        {"resolution": "QA F1：关闭前先收敛到 resolved"}, csrf=True)
    resolved = st_r in (200, 201) and d_r.get("code") == 0
    record("BPMN F1", "前置：in_progress → resolved（closed 仅可从 resolved 迁入）", resolved,
           method="POST", path=f"{API}/tickets/{TEMP_TICKET}/resolve", http=st_r,
           code=d_r.get("code") if isinstance(d_r, dict) else None,
           message="" if resolved else str(d_r.get("message"))[:160])
    if not resolved:
        return
    before = db(f"SELECT count(*) FROM process_instances WHERE business_key='{bk}'")
    st, d, el = call("POST", f"{API}/tickets/{TEMP_TICKET}/close", {}, csrf=True)
    ok = st in (200, 201) and d.get("code") == 0
    record("BPMN F1", "POST /tickets/:id/close 关单成功", ok, method="POST",
           path=f"{API}/tickets/{TEMP_TICKET}/close", http=st, code=d.get("code"),
           elapsed_ms=round(el * 1000), message=("" if ok else str(d.get("message"))[:160]))
    if not ok:
        return
    time.sleep(1.0)  # 同事务终止；给读库一个可见窗口
    after = db(f"SELECT count(*) FROM process_instances WHERE business_key='{bk}' AND status NOT IN ('terminated','finished','completed','cancelled')")
    if before == "0":
        record("BPMN F1", "关单后无 running 实例残留", after == "0",
               method="DB", path="process_instances",
               message=f"建单未自动起流（before=0），断言退化但通过；after_running={after}",
               severity="WARN")
    else:
        record("BPMN F1", "关单同事务终止 BPMN 实例（F1 收口）", after == "0",
               method="DB", path="process_instances",
               message=f"before_instances={before} after_running={after}")


# ---------------------------------------------------------------------------
# ⑦ 前端页面
# ---------------------------------------------------------------------------
def frontend_pages() -> None:
    for name, p in [("登录页", "/login"), ("仪表盘", "/dashboard"), ("工单页", "/tickets"),
                    ("权限管理页（AI 授权面板）", "/admin/permissions"),
                    ("AI 审批页（pending_agent 闭环）", "/ai/approval"),
                    ("技能管理页", "/admin/skills")]:
        st, data, el = call("GET", FRONTEND_BASE + p)
        ok = st in (200, 302, 307)
        record("前端", name, ok, method="GET", path=p, http=st,
               elapsed_ms=round(el * 1000), message=("" if ok else str(data)[:120]))


# ---------------------------------------------------------------------------
# 清理
# ---------------------------------------------------------------------------
def cleanup() -> None:
    restore_agent_grant()
    restore_policy()
    cleanup_skill()
    if TEMP_TICKET:
        st, d, _ = call("DELETE", f"{API}/tickets/{TEMP_TICKET}", csrf=True)
        record("清理", "删除临时工单", st in (200, 201, 204) and d.get("code") in (0, None),
               method="DELETE", path=f"{API}/tickets/{TEMP_TICKET}", http=st,
               code=d.get("code"), severity="WARN")


def write_report(json_path: Path | None, md_path: Path | None) -> dict[str, int]:
    summary = {"PASS": 0, "WARN": 0, "FAIL": 0, "BLOCKED": 0}
    for r in RESULTS:
        summary[r["result"]] = summary.get(r["result"], 0) + 1
    if json_path:
        json_path.write_text(json.dumps({"summary": summary, "results": RESULTS},
                                        ensure_ascii=False, indent=2), encoding="utf-8")
    if md_path:
        lines = ["# prod 新功能点专项测试报告",
                 "",
                 f"生成时间：{dt.datetime.now().isoformat(timespec='seconds')}  ",
                 f"结论：**{summary['PASS']} PASS / {summary['WARN']} WARN / {summary['FAIL']} FAIL / {summary['BLOCKED']} BLOCKED**",
                 ""]
        cur = ""
        for r in RESULTS:
            if r["group"] != cur:
                cur = r["group"]
                lines += [f"## {cur}", "", "| 结果 | 用例 | HTTP | 业务码 | 说明 |", "|---|---|---|---|---|"]
            lines.append(f"| {r['result']} | {r['name']} | {r['http']} | {r['code']} | {r['message'][:120]} |")
        lines.append("")
        md_path.write_text("\n".join(lines), encoding="utf-8")
    return summary


def main() -> int:
    global FRONTEND_BASE, EXPECT_IMAGE
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="http://127.0.0.1")
    ap.add_argument("--frontend-base", default="http://127.0.0.1")
    ap.add_argument("--expect-image", default="", help="期望的运行镜像 tag，如 itsm-backend:v1.6.60")
    ap.add_argument("--json", default="")
    ap.add_argument("--md", default="")
    args = ap.parse_args()
    BASE = args.base.rstrip("/")
    FRONTEND_BASE = args.frontend_base.rstrip("/")
    EXPECT_IMAGE = args.expect_image
    login()
    deploy_fingerprint()
    permission_policy()
    gate_agent_principal()
    skill_persistence()
    investigation_orchestration()
    bpmn_f1_close_termination()
    frontend_pages()
    cleanup()
    json_path = Path(args.json) if args.json else None
    md_path = Path(args.md) if args.md else None
    summary = write_report(json_path, md_path)
    print(f"SUMMARY {summary} json={args.json or '-'} md={args.md or '-'}", flush=True)
    return 0 if summary["FAIL"] == 0 and summary["BLOCKED"] == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
