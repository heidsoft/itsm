#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
dev 环境业务流程深度测试（补充 prod_integration_test.py 未覆盖的生命周期流转）
覆盖：事件全生命周期、问题流转、服务请求提交、通知、仪表盘联动、AI 会话。
测试数据 IT_ 前缀，结束时清理。
"""
import json
import os
import subprocess
import sys
import time
import urllib.request

BASE = "http://localhost:8090/api/v1"
PG = ["docker", "exec", "itsm-postgres-dev", "psql", "-U", "itsm_user",
      "-d", "itsm", "-t", "-A"]
results = []
created = []


def req(method, path, body=None, token=None, timeout=30):
    url = BASE + path
    data = json.dumps(body).encode() if body is not None else None
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    r = urllib.request.Request(url, data=data, headers=headers, method=method)
    t0 = time.time()
    try:
        with urllib.request.urlopen(r, timeout=timeout) as resp:
            body = resp.read().decode()
            return resp.status, (json.loads(body) if body else {}), time.time() - t0
    except urllib.error.HTTPError as e:
        body = e.read().decode()
        try:
            return e.code, json.loads(body), time.time() - t0
        except Exception:
            return e.code, {"raw": body[:200]}, time.time() - t0


def db1(sql):
    r = subprocess.run(PG + ["-c", sql], capture_output=True, text=True)
    return r.stdout.strip()


def rec(mod, name, ok, detail=""):
    results.append((mod, name, ok, detail))
    print(f"  [{'PASS' if ok else 'FAIL'}] {name}" + (f" — {detail}" if detail else ""))


tok = None
st, b, _ = req("POST", "/auth/login", {"username": "admin", "password": "admin123"})
if st == 200:
    tok = b["data"]["accessToken"]
else:
    print("登录失败", st, b)
    sys.exit(1)


def try_flow(mod, steps):
    """steps: [(name, method, path, body, expect_codes), ...] 顺序执行，遇失败继续。"""
    for name, method, path, body, expects in steps:
        st, b, dt = req(method, path, body, tok)
        ok = st in expects
        detail = f"HTTP {st}" + (f" {dt*1000:.0f}ms" if ok else f" {json.dumps(b, ensure_ascii=False)[:160]}")
        rec(mod, name, ok, detail)
        yield st, b


print("[A] 事件 Incident 全生命周期")
tid = None
for st, b in try_flow("incident", [
    ("事件创建", "POST", "/incidents",
     {"title": "IT_flow_incident", "description": "dev 流程测试", "priority": "high",
      "impact": "medium", "urgency": "high"}, (200, 201)),
]):
    if st in (200, 201):
        tid = (b.get("data") or {}).get("id") or (b.get("data") or {}).get("ID")
        break
if tid:
    steps = [
        ("事件详情", "GET", f"/incidents/{tid}", None, (200,)),
        ("事件分配", "POST", f"/incidents/{tid}/assign", {"assigneeId": 1}, (200,)),
        ("事件确认(acknowledge)", "POST", f"/incidents/{tid}/acknowledge", None, (200,)),
        ("事件转处理中", "PUT", f"/incidents/{tid}", {"status": "in_progress"}, (200,)),
        ("事件备注", "POST", f"/incidents/{tid}/comments",
         {"content": "IT_flow_comment"}, (200, 201)),
        ("事件解决", "POST", f"/incidents/{tid}/resolve",
         {"resolution": "IT_flow_resolved"}, (200,)),
        ("事件关闭", "POST", f"/incidents/{tid}/close",
         {"closeNotes": "IT_flow_close_note"}, (200,)),
        ("清理事件", "DELETE", f"/incidents/{tid}", None, (200, 204, 400)),
    ]
    for st, b in try_flow("incident", steps):
        pass
else:
    rec("incident", "事件创建返回可用 id", False, "未获取到 id")

print("[B] 问题 Problem 流转")
pid = None
for st, b in try_flow("problem", [
    ("问题创建", "POST", "/problems",
     {"title": "IT_flow_problem", "description": "dev environment flow testing description", "priority": "medium", "category": "software"}, (200, 201)),
]):
    if st in (200, 201):
        pid = (b.get("data") or {}).get("id")
        break
if pid:
    for st, b in try_flow("problem", [
        ("问题详情", "GET", f"/problems/{pid}", None, (200,)),
        ("问题调查(investigate)", "POST", f"/problems/{pid}/investigate",
         {"notes": "IT_flow_investigate"}, (200,)),
        ("问题根因", "PUT", f"/problems/{pid}/root-cause",
         {"rootCause": "IT_flow_root_cause"}, (200,)),
        ("问题转已解决", "PUT", f"/problems/{pid}", {"status": "resolved"}, (200,)),
        ("问题关闭", "POST", f"/problems/{pid}/close",
         {"resolution": "IT_flow_resolution"}, (200,)),
        ("清理问题", "DELETE", f"/problems/{pid}", None, (200, 204, 400)),
    ]):
        pass
else:
    rec("problem", "问题创建返回可用 id", False, "未获取到 id")

print("[C] 服务请求 ServiceRequest")
srid = None
for st, b in try_flow("service-request", [
    ("服务目录列表(带模板)", "GET", "/service-catalogs", None, (200,)),
    ("服务请求提交", "POST", "/service-requests",
     {"title": "IT_flow_sr", "description": "dev 流程测试", "priority": "low"}, (200, 201)),
]):
    if st in (200, 201) and isinstance(b.get("data"), dict):
        srid = b["data"].get("id")
        break
if srid:
    for st, b in try_flow("service-request", [
        ("服务请求详情", "GET", f"/service-requests/{srid}", None, (200,)),
        ("服务请求取消/删除", "DELETE", f"/service-requests/{srid}", None, (200, 204, 400)),
    ]):
        pass

print("[D] 通知 / 仪表盘联动")
for st, b in try_flow("notification", [("通知列表", "GET", "/notifications", None, (200,))]):
    pass
st, b, _ = req("GET", "/dashboard/stats", token=tok)
if st == 200:
    d = b.get("data") or {}
    rec("dashboard", "仪表盘统计可读", True,
        f"keys={sorted(list(d.keys()))[:8]}")
else:
    rec("dashboard", "仪表盘统计可读", False, f"HTTP {st}")

print("[E] 变更拒绝路径（审批链负例）")
cid = None
for st, b in try_flow("change", [
    ("变更创建(负例)", "POST", "/changes",
     {"title": "IT_flow_chg_reject", "description": "dev", "change_type": "normal",
      "priority": "medium", "risk_level": "low"}, (200, 201)),
]):
    if st in (200, 201):
        cid = (b.get("data") or {}).get("id")
        break
if cid:
    for st, b in try_flow("change", [
        ("提交审批", "POST", f"/changes/{cid}/submit", None, (200,)),
        ("审批拒绝", "POST", f"/changes/{cid}/reject",
         {"comment": "IT_flow_reject"}, (200,)),
        ("DB 状态=rejected", "GET", f"/changes/{cid}", None, (200,)),
    ]):
        if "DB 状态" in str(b):
            pass
    st, b, _ = req("GET", f"/changes/{cid}", token=tok)
    status = ((b.get("data") or {}).get("status") or "")
    rec("change", "拒绝后 DB 状态校验", status in ("rejected", "rejected_pending"), f"status={status}")
    req("DELETE", f"/changes/{cid}", token=tok)

print("[F] 未授权写路径负例")
st, b, _ = req("POST", "/tickets", {"title": "IT_flow_forge", "description": "x"},
               token="forged-token-xyz")
rec("security", "伪造 token 写工单被拒", st in (401, 403), f"HTTP {st} {json.dumps(b, ensure_ascii=False)[:100]}")
st, b, _ = req("GET", "/auth/me", token="forged-token-xyz")
rec("security", "伪造 token 读用户信息被拒", st == 401, f"HTTP {st}")

print("[G] 状态机负例（investigating 直接 closed 应 409 而非 500）")
nid = None
st, b, _ = req("POST", "/problems",
               {"title": "IT_flow_neg", "description": "negative case for state machine",
                "priority": "low", "category": "software"}, tok)
if st == 200:
    nid = b["data"]["id"]
    req("POST", f"/problems/{nid}/investigate", {"notes": "x"}, tok)
    st, b, _ = req("POST", f"/problems/{nid}/close", {"resolution": "skip"}, tok)
    rec("problem", "非法状态转换返回 409(非500)", st == 409 and b.get("code") == 4090,
        f"HTTP {st} code={b.get('code')}")
    req("DELETE", f"/problems/{nid}", tok)
else:
    rec("problem", "负例问题创建", False, f"HTTP {st}")

print("[清理]")
for t, col in [("tickets", "title"), ("incidents", "title"), ("problems", "title"),
               ("changes", "title"), ("service_requests", "title")]:
    n = db1(f"SELECT count(*) FROM {t} WHERE {col} LIKE 'IT_flow%'")
    if n and n.isdigit() and int(n) > 0:
        db1(f"DELETE FROM {t} WHERE {col} LIKE 'IT_flow%'")
        print(f"  清理 {t}: {n} 条")

print("\n" + "=" * 70)
tp = sum(1 for _, _, o, _ in results if o)
tf = sum(1 for _, _, o, _ in results if not o)
print(f"深度流程测试: {tp} 通过 / {tf} 失败 (共 {len(results)})")
if tf:
    for m, n, o, d in results:
        if not o:
            print(f"  FAIL [{m}] {n}: {d}")
sys.exit(1 if tf else 0)
