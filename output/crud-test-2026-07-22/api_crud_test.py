#!/usr/bin/env python3
"""ITSM CRUD API 层测试 —— fix2：修正 Problem/Catalog/Dept/Team/SLA 路径与必填"""
import json, time, urllib.request as ur, urllib.error as ue
BASE = "http://localhost/api/v1"
TOKEN = open("/tmp/itsm_token").read().strip()
H = {"Authorization": f"Bearer {TOKEN}", "Content-Type": "application/json"}
results = []

def http(method, path, body=None):
    url = BASE + path
    data = json.dumps(body).encode() if body else None
    req = ur.Request(url, data=data, headers=H, method=method)
    try:
        with ur.urlopen(req, timeout=10) as r:
            return r.status, r.read().decode()
    except ue.HTTPError as e:
        return e.code, e.read().decode()

def rec(module, action, method, path, body=None):
    code, txt = http(method, path, body)
    try: js = json.loads(txt) if txt else {}
    except: js = {"_raw": txt[:120]}
    ok = 200 <= code < 300 and js.get("code") in (0, None, 200)
    results.append({"module": module, "action": action, "method": method, "path": path,
                    "http": code, "api_code": js.get("code"), "msg": (js.get("message") or "")[:100], "ok": ok})
    return code, js

def extract_id(js, id_key="id"):
    d = js.get("data")
    if isinstance(d, dict):
        if id_key in d: return d[id_key]
        for v in d.values():
            if isinstance(v, dict) and id_key in v: return v[id_key]
    return None

def crud(name, base, create_body, update_body, update_path=None, delete_path=None):
    rec(name, "LIST", "GET", f"{base}?page=1&pageSize=5")
    code, js = rec(name, "CREATE", "POST", base, create_body)
    nid = extract_id(js) if js.get("code") == 0 else None
    if not nid:
        results[-1]["msg"] += " (no id)"
        return
    rec(name, "READ", "GET", f"{base}/{nid}")
    up = update_path or f"{base}/{nid}"
    rec(name, "UPDATE", "PUT", up, update_body)
    dp = delete_path or f"{base}/{nid}"
    rec(name, "DELETE", "DELETE", dp)

ts = int(time.time())

crud("Tickets", "/tickets",
    {"title": f"CRUD工单-{ts}", "description": "auto", "priority": "low", "type": "incident"},
    {"title": f"CRUD工单-{ts}(改)", "priority": "medium"})

crud("Incidents", "/incidents",
    {"title": f"CRUD事件-{ts}", "description": "auto", "priority": "low", "severity": "low"},
    {"title": f"CRUD事件-{ts}(改)"})

crud("Problems", "/problems",
    {"title": f"CRUD问题-{ts}", "description": "问题详情描述内容-auto", "priority": "low"},
    {"title": f"CRUD问题-{ts}(改)"})

crud("Changes", "/changes",
    {"title": f"CRUD变更-{ts}", "description": "auto", "type": "standard", "priority": "low", "riskLevel": "low"},
    {"title": f"CRUD变更-{ts}(改)"})

crud("CMDB-CIs", "/cmdb/cis",
    {"name": f"CRUD-CI-{ts}", "type": "Server", "ciTypeId": 1, "status": "active", "environment": "production", "criticality": "low", "attributes": {"ip": "10.0.0.1"}},
    {"name": f"CRUD-CI-{ts}(改)", "status": "inactive"})

crud("Knowledge", "/knowledge/articles",
    {"title": f"CRUD知识-{ts}", "content": "内容", "category": "test", "status": "draft"},
    {"title": f"CRUD知识-{ts}(改)", "content": "内容(改)"})

crud("ServiceCatalog", "/service-catalogs",
    {"name": f"CRUD目录-{ts}", "description": "auto", "category": "general", "status": "enabled"},
    {"name": f"CRUD目录-{ts}(改)"})

crud("Users", "/users",
    {"username": f"crud_{ts}", "password": "Crud123456!", "email": f"crud_{ts}@test.com", "name": "自动化用户", "phone": "13800000000"},
    {"name": "自动化用户(改)", "phone": "13900000000"})

crud("Roles", "/roles",
    {"name": f"crud_role_{ts}", "description": "auto"},
    {"description": "auto(改)"})

crud("Groups", "/groups",
    {"name": f"crud_group_{ts}", "description": "auto"},
    {"description": "auto(改)"})

# 修正：Departments 列表走 /departments，写走 /org/departments
def crud_split(name, list_path, write_base, create_body, update_body):
    rec(name, "LIST", "GET", f"{list_path}?page=1&pageSize=5")
    code, js = rec(name, "CREATE", "POST", write_base, create_body)
    nid = extract_id(js) if js.get("code") == 0 else None
    if not nid:
        results[-1]["msg"] += " (no id)"
        return
    # 详情 tenant-scoped 路径试探
    rec(name, "READ", "GET", f"{list_path}/{nid}")
    rec(name, "UPDATE", "PUT", f"{write_base}/{nid}", update_body)
    rec(name, "DELETE", "DELETE", f"{write_base}/{nid}")

crud_split("Departments", "/departments", "/org/departments",
    {"name": f"crud_dept_{ts}", "code": f"DEPT{ts}", "description": "auto"},
    {"description": "auto(改)"})

crud_split("Teams", "/teams", "/org/teams",
    {"name": f"crud_team_{ts}", "code": f"TEAM{ts}", "description": "auto"},
    {"description": "auto(改)"})

# SLA 走 /sla/definitions
crud("SLA-Definitions", "/sla/definitions",
    {"name": f"crud_sla_{ts}", "description": "auto", "targetHours": 4, "priority": "medium", "responseTime": 30, "resolutionTime": 240},
    {"description": "auto(改)"})

# 只读探测
rec("Workflows-BPMN", "LIST", "GET", "/bpmn/process-definitions")

out = {"summary": {"total": len(results), "pass": sum(1 for r in results if r["ok"]), "fail": sum(1 for r in results if not r["ok"])}, "details": results}
json.dump(out, open("/Users/heidsoft/Downloads/research/itsm/output/crud-test-2026-07-22/api-crud-report.json", "w"), ensure_ascii=False, indent=2)

print(f"\n=== {out['summary']['pass']}/{out['summary']['total']} 通过 ===\n")
for r in results:
    tag = "✅" if r["ok"] else "❌"
    print(f"{tag} {r['module']:18s} {r['action']:8s} {r['method']:6s} {r['path']:38s} http={r['http']:3d} code={r['api_code']} {r['msg']}")
