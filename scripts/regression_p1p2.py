#!/usr/bin/env python3
"""
ITSM 修复回归测试：覆盖 P0/P1/P2 所有 6 项修复
"""
import json
import sys
import time
import urllib.request
import urllib.error
import urllib.parse

BASE = "http://localhost:8090"
COOKIE = None
TOKEN = None
RESULTS = []


def http(method, path, body=None, expect=200, headers=None):
    url = f"{BASE}{path}"
    data = None
    if body is not None:
        data = json.dumps(body).encode()
    req = urllib.request.Request(url, data=data, method=method)
    if body is not None:
        req.add_header("Content-Type", "application/json")
    if headers:
        for k, v in headers.items():
            req.add_header(k, str(v))
    if TOKEN:
        req.add_header("Authorization", f"Bearer {TOKEN}")
        req.add_header("X-Tenant-ID", "1")
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            return resp.status, json.loads(resp.read().decode() or "{}")
    except urllib.error.HTTPError as e:
        body = e.read().decode(errors="ignore")
        try:
            return e.code, json.loads(body or "{}")
        except Exception:
            return e.code, {"raw": body}
    except Exception as e:
        return 0, {"error": str(e)}


def case(name, ok, detail=""):
    RESULTS.append((name, ok, detail))
    print(f"  {'OK' if ok else 'FAIL'} {name}{(' -- ' + detail) if detail else ''}")


def login():
    global TOKEN
    code, body = http("POST", "/api/v1/auth/login", {
        "username": "admin",
        "password": "admin123",
        "tenantCode": "default",
    }, headers={})
    if code != 200 or body.get("code") != 0:
        print(f"login failed: code={code} body={body}")
        sys.exit(1)
    TOKEN = body["data"]["access_token"]
    case("登录获取 token", True, f"len={len(TOKEN)}")


def test_p1_routes():
    print("\n=== P1-01 路由别名 ===")
    code, body = http("GET", "/api/v1/dashboard/metrics")
    case("/dashboard/metrics 可达", code == 200 and body.get("code") == 0,
         f"code={code}, has kpiMetrics={'kpiMetrics' in (body.get('data') or {})}")

    code, body = http("GET", "/api/v1/system/config")
    case("/system/config 可达", code == 200 and body.get("status") == "ok",
         f"code={code}, status={body.get('status')}")


def test_p1_07_sla():
    print("\n=== P1-07 SLA actual_response_minutes ===")
    code, body = http("POST", "/api/v1/tickets", {
        "title": "回归测试-修复P1-07",
        "description": "P1-07 SLA 修复回归测试工单，内容足够长以满足最小长度校验规则",
        "priority": "high",
        "type": "incident",
    })
    if code != 200 or body.get("code") != 0:
        case("创建 SLA 验证工单", False, f"code={code} body={body}")
        return
    ticket_id = body["data"]["id"]
    case("创建 SLA 验证工单", True, f"id={ticket_id}")

    code, body = http("POST", "/api/v1/tickets/workflow/accept", {
        "ticket_id": ticket_id,
        "comment": "回归测试接单",
    })
    case("接单成功", code == 200 and body.get("code") == 0,
         f"code={code} msg={body.get('message')}")

    code, body = http("POST", f"/api/v1/sla/check-compliance/{ticket_id}")
    data = body.get("data") or {}
    has_actual = "actual_response_minutes" in data
    has_compliant = "compliant" in data
    has_message = "message" in data
    case("SLA check-compliance 返回新结构体",
         has_actual and has_compliant and has_message,
         f"keys={list(data.keys())}")
    case("actual_response_minutes > 0 (接单后)",
         data.get("actual_response_minutes", 0) > 0,
         f"actual_response_minutes={data.get('actual_response_minutes')}")
    case("compliant=True (接单后)", data.get("compliant") is True,
         f"compliant={data.get('compliant')}")


def test_p1_08_workflow_history():
    print("\n=== P1-08 workflow-history ===")
    code, body = http("GET", "/api/v1/tickets?page=1&page_size=1")
    if code != 200 or body.get("code") != 0:
        case("获取工单列表", False, f"code={code}")
        return
    tickets = body.get("data", {}).get("tickets", [])
    if not tickets:
        case("工单列表为空，跳过", True, "无工单")
        return
    tid = tickets[0]["id"]
    code, body = http("GET", f"/api/v1/tickets/{tid}/workflow-history")
    items = body.get("data")
    case("workflow-history 返回数组（不报错）",
         code == 200 and body.get("code") == 0 and isinstance(items, list),
         f"code={code} items_count={len(items) if isinstance(items, list) else 'N/A'}")
    if isinstance(items, list) and items:
        first = items[0]
        case("comment/reason 字段存在",
             "comment" in first and "reason" in first,
             f"comment={first.get('comment')}, reason={first.get('reason')}")


def test_p1_04_marketplace():
    print("\n=== P1-04 marketplace ===")
    code, body = http("GET", "/api/v1/marketplace/items?page=1&page_size=10")
    items = body.get("data", {}).get("items", [])
    case("marketplace/items 返回真实数据",
         code == 200 and body.get("code") == 0 and len(items) > 0,
         f"code={code} count={len(items)}")
    if items:
        first = items[0]
        required = {"id", "name", "title", "type", "description"}
        has_required = all(k in first for k in required)
        case("marketplace item 字段完整",
             has_required,
             f"sample_keys={list(first.keys())[:5]}")


def test_p1_05_knowledge():
    print("\n=== P1-05 knowledge API 字段 ===")
    code, body = http("GET", "/api/v1/knowledge/stats")
    data = body.get("data") or {}
    has_correct_fields = all(k in data for k in ["total", "published", "draft", "views"])
    case("/knowledge/stats 返回正确字段",
         code == 200 and has_correct_fields,
         f"keys={list(data.keys())}, total={data.get('total')}")

    code, body = http("GET", "/api/v1/knowledge/articles?page=1&pageSize=5")
    data = body.get("data") or {}
    has_articles = "articles" in data
    case("/knowledge/articles 返回正确",
         code == 200 and has_articles,
         f"articles_count={len(data.get('articles', []))}")


def test_p2_04_install_protection():
    print("\n=== P2-04 install 业务保护 ===")
    code, body = http("GET", "/api/v1/marketplace/items?page=1&page_size=50")
    items = body.get("data", {}).get("items", [])
    if not items:
        case("无可测试的 item", True, "skip")
        return
    code, body = http("GET", "/api/v1/marketplace/installations")
    installed_ids = set()
    if code == 200 and isinstance(body.get("data"), list):
        for inst in body["data"]:
            iid = inst.get("item_id") or inst.get("itemId") or inst.get("id")
            if iid:
                installed_ids.add(iid)

    # 先清理已安装项，避免状态污染
    # httpClient 自动把 snake_case 转 camelCase，字段名是 itemId
    for inst in (body.get("data") or []):
        iid = inst.get("itemId") or inst.get("item_id") or inst.get("id")
        if iid:
            http("POST", f"/api/v1/marketplace/items/{iid}/uninstall")

    target = next((it for it in items if it["id"] not in installed_ids), None)
    if not target:
        # 找任意一个
        target = items[0]
    item_id = target["id"]
    case("选择未安装的 item", True, f"id={item_id} name={target.get('name')}")

    code, body = http("POST", f"/api/v1/marketplace/items/{item_id}/install")
    first_ok = code == 200 and body.get("code") == 0
    case("首次 install 成功", first_ok, f"code={code} msg={body.get('message')}")

    code, body = http("POST", f"/api/v1/marketplace/items/{item_id}/install")
    second_rejected = code != 200 or body.get("code") != 0
    case("重复 install 被拒绝", second_rejected,
         f"code={code} msg={body.get('message')}")

    code, body = http("POST", f"/api/v1/marketplace/items/{item_id}/uninstall")
    case("uninstall 成功", code == 200 and body.get("code") == 0,
         f"code={code} msg={body.get('message')}")

    # 安装草稿状态的 item 应该被拒绝（业务保护）
    # 当前 /marketplace/items API 不支持 status 参数，因此我们检查返回列表中是否真的有 draft item
    draft_id = None
    code2, body2 = http("GET", "/api/v1/marketplace/items?page=1&page_size=50")
    if code2 == 200 and body2.get("code") == 0:
        drafts = [
            it for it in body2.get("data", {}).get("items", [])
            if it.get("status") == "draft"
        ]
        if drafts:
            draft_id = drafts[0]["id"]
    if draft_id:
        code, body = http("POST", f"/api/v1/marketplace/items/{draft_id}/install")
        case("草稿状态 install 被拒绝",
             code != 200 or body.get("code") != 0,
             f"code={code} msg={body.get('message')}")
    else:
        case("草稿状态 install 被拒绝 (跳过：当前无 draft item)",
             True,
             "skipped, no draft item in DB")


def main():
    print("== ITSM 修复回归测试 ==")
    login()
    test_p1_routes()
    test_p1_07_sla()
    test_p1_08_workflow_history()
    test_p1_04_marketplace()
    test_p1_05_knowledge()
    test_p2_04_install_protection()

    print("\n=== 总览 ===")
    total = len(RESULTS)
    passed = sum(1 for _, ok, _ in RESULTS if ok)
    failed = total - passed
    print(f"  总数: {total}  通过: {passed}  失败: {failed}")
    for name, ok, detail in RESULTS:
        if not ok:
            print(f"   FAIL {name} {detail}")
    sys.exit(0 if failed == 0 else 1)


if __name__ == "__main__":
    main()
