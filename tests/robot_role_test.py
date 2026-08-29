"""
ITSM 机器人企业角色测试 - 最终版
模拟 5 类企业自动化集成场景
"""
import os
import requests
import json
from datetime import datetime

BASE = os.environ.get("ITSM_TEST_BASE_URL", "http://localhost:8090")
# 凭据一律从环境变量读取，禁止硬编码真实口令进版本库。
# 示例：export ITSM_ADMIN_PASSWORD=xxx ITSM_USER_PASSWORD=xxx
ADMIN_PASSWORD = os.environ.get("ITSM_ADMIN_PASSWORD", "")
USER_PASSWORD = os.environ.get("ITSM_USER_PASSWORD", "")
results = []

def log(msg):
    print(f"  {msg}", flush=True)

def record(scenario, action, ok, detail=""):
    results.append({"scenario": scenario, "action": action, "ok": ok, "detail": detail})
    icon = "✅" if ok else "❌"
    log(f"{icon} {action}: {detail}")

def login(username, password):
    r = requests.post(f"{BASE}/api/v1/auth/login", json={"username": username, "password": password}, timeout=10)
    return r.json()["data"]["accessToken"], r.json()["data"]["user"]

def H(token):
    return {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}

admin_tk, _ = login("admin", ADMIN_PASSWORD)
enduser_tk, _ = login("enduser1", USER_PASSWORD)
agent_tk, _ = login("agent1", USER_PASSWORD)
agent2_tk, _ = login("agent2", USER_PASSWORD)
manager_tk, _ = login("manager1", USER_PASSWORD)

# ============================================================
# 场景 1: Webhook 回调 - 外部监控系统
# ============================================================
print("\n[场景 1] Webhook 回调 (Zabbix/Prometheus)")
try:
    r = requests.post(f"{BASE}/api/v1/tickets", headers=H(admin_tk), json={
        "title": "[Zabbix] HIS-DB-01 数据库连接超时",
        "description": "Zabbix Trigger: HIS-DB-01\nSeverity: High\nTime: " + datetime.now().isoformat(),
        "priority": "high",
        "type": "incident",
        "source": "zabbix"
    }, timeout=15)
    d = r.json()
    zbx_id = d.get("data", {}).get("id")
    record("webhook", "Zabbix触发创建工单", d.get("code") == 0, f"工单 #{zbx_id}")
except Exception as e:
    record("webhook", "Zabbix触发创建工单", False, str(e)[:80])

if zbx_id:
    try:
        r = requests.patch(f"{BASE}/api/v1/tickets/{zbx_id}", headers=H(admin_tk), json={"status": "in_progress"}, timeout=10)
        d = r.json()
        record("webhook", "Webhook更新工单状态", d.get("code") == 0, f"状态变更为 in_progress")
    except Exception as e:
        record("webhook", "Webhook更新工单状态", False, str(e)[:80])

try:
    success = 0
    for i in range(3):
        r = requests.post(f"{BASE}/api/v1/tickets", headers=H(admin_tk), json={
            "title": f"[Prometheus] HIS-WEB-{i:02d} 响应时间过长",
            "description": f"P99 latency > 5s, Instance: his-web-{i:02d}",
            "priority": "medium",
            "type": "incident",
            "source": "prometheus"
        }, timeout=10)
        if r.json().get("code") == 0:
            success += 1
    record("webhook", "批量Prometheus告警建单", success == 3, f"{success}/3成功")
except Exception as e:
    record("webhook", "批量告警建单", False, str(e)[:80])

# ============================================================
# 场景 2: Connector - 飞书/钉钉
# ============================================================
print("\n[场景 2] Connector 集成 (飞书/钉钉机器人)")
try:
    r = requests.post(f"{BASE}/api/v1/tickets", headers=H(enduser_tk), json={
        "title": "[飞书Bot] 内科打印机故障",
        "description": "科室：内科\n工位：3F-12\n描述：HP LaserJet 错误 13.20.00\n来源：飞书群@IT机器人",
        "priority": "medium",
        "type": "incident",
        "source": "feishu_bot"
    }, timeout=15)
    d = r.json()
    fs_id = d.get("data", {}).get("id") if d.get("code") == 0 else None
    record("feishu", "飞书机器人建单", d.get("code") == 0, f"工单 #{fs_id}")
except Exception as e:
    record("feishu", "飞书机器人建单", False, str(e)[:80])

if fs_id:
    try:
        r = requests.post(f"{BASE}/api/v1/tickets/{fs_id}/comments", headers=H(agent_tk), json={
            "content": "[飞书Bot同步] 李工程师已派单，30分钟内现场处理",
            "internal": False
        }, timeout=10)
        d = r.json()
        record("feishu", "工程师通过飞书Bot回复", d.get("code") == 0, f"{d.get('message')}")
    except Exception as e:
        record("feishu", "工程师回复", False, str(e)[:80])
    
    try:
        r = requests.patch(f"{BASE}/api/v1/tickets/{fs_id}", headers=H(agent2_tk), json={"status": "in_progress"}, timeout=10)
        d = r.json()
        record("feishu", "工程师接单变更状态", d.get("code") == 0, f"{d.get('message')}")
    except Exception as e:
        record("feishu", "工程师接单", False, str(e)[:80])

try:
    r = requests.post(f"{BASE}/api/v1/tickets", headers=H(enduser_tk), json={
        "title": "[钉钉Bot] 护士站电脑蓝屏",
        "description": "科室：外科护士站\n描述：Windows 11 蓝屏\n来源：钉钉群",
        "priority": "high",
        "type": "incident",
        "source": "dingtalk_bot"
    }, timeout=15)
    d = r.json()
    record("dingtalk", "钉钉机器人建单", d.get("code") == 0, f"工单 #{d.get('data',{}).get('id')}")
except Exception as e:
    record("dingtalk", "钉钉机器人建单", False, str(e)[:80])

# ============================================================
# 场景 3: AI 自动化
# ============================================================
print("\n[场景 3] AI 自动化")
if zbx_id:
    try:
        r = requests.post(f"{BASE}/api/v1/ai/tickets/{zbx_id}/analyze", headers=H(admin_tk), timeout=30)
        if r.status_code == 200:
            d = r.json()
            if d.get("code") == 0:
                data = d.get("data", {})
                record("ai", "AI工单分析", True, f"分类={data.get('category','-')} 优先级={data.get('priority','-')}")
            else:
                record("ai", "AI工单分析", False, f"code={d.get('code')}")
        else:
            record("ai", "AI工单分析", False, f"HTTP {r.status_code}")
    except Exception as e:
        record("ai", "AI工单分析", False, str(e)[:80])

try:
    r = requests.post(f"{BASE}/api/v1/knowledge/search", headers=H(admin_tk), json={
        "query": "HIS 系统升级操作流程", "topK": 3, "threshold": 0.6
    }, timeout=20)
    d = r.json()
    if d.get("code") == 0:
        items = d.get("data", {}).get("items", []) if isinstance(d.get("data"), dict) else []
        record("ai", "RAG语义检索", True, f"返回 {len(items)} 条结果")
    else:
        record("ai", "RAG语义检索", False, f"code={d.get('code')}")
except Exception as e:
    record("ai", "RAG语义检索", False, str(e)[:80])

try:
    r = requests.get(f"{BASE}/api/v1/knowledge/articles?isPublished=true&page=1&pageSize=10", headers=H(admin_tk), timeout=10)
    d = r.json()
    if d.get("code") == 0:
        items = d.get("data", {}).get("items", [])
        record("ai", "知识库已发布文章", True, f"{len(items)} 篇可用文章")
    else:
        record("ai", "知识库文章列表", False, f"code={d.get('code')}")
except Exception as e:
    record("ai", "知识库列表", False, str(e)[:80])

# ============================================================
# 场景 4: 多租户 MSP（契约对齐：使用 Router 真实注册路径；
# 私有部署模式下 MSP 路由未注册，应返回结构化能力禁用 404 而非裸 HTML）
# ============================================================
print("\n[场景 4] MSP 多租户")
for ep in ["/api/v1/msp/status", "/api/v1/msp/context", "/api/v1/msp/customers", "/api/v1/msp/allocations"]:
    try:
        r = requests.get(f"{BASE}{ep}", headers=H(admin_tk), timeout=10)
        try:
            d = r.json()
            if d.get("code") == 0:
                record("msp", f"MSP {ep.split('/')[-1]}", True, "可用")
            elif d.get("code") == 404 and "disabled" in str(d.get("message", "")):
                record("msp", f"MSP {ep.split('/')[-1]}", True, "能力禁用（private 模式，结构化 404 ✓）")
            elif d.get("code") == 2001:
                record("msp", f"MSP {ep.split('/')[-1]}", False, "需要MSP管理员")
            else:
                record("msp", f"MSP {ep.split('/')[-1]}", False, f"code={d.get('code')} {d.get('message','')[:60]}")
        except Exception:
            record("msp", f"MSP {ep.split('/')[-1]}", False, f"非JSON响应 HTTP {r.status_code}")
    except Exception as e:
        record("msp", f"MSP {ep.split('/')[-1]}", False, str(e)[:60])

# ============================================================
# 场景 5: 批量自动化
# ============================================================
print("\n[场景 5] 自动化批量操作")
for path, name in [
    ("/api/v1/cmdb/cis?page=1&pageSize=20", "CMDB CI列表"),
    ("/api/v1/sla/policies?page=1&pageSize=20", "SLA策略列表"),
    ("/api/v1/dashboard/overview", "Dashboard概览"),
    ("/api/v1/users?page=1&pageSize=20", "用户列表"),
    ("/api/v1/notifications?page=1&pageSize=10", "通知列表"),
]:
    try:
        r = requests.get(f"{BASE}{path}", headers=H(admin_tk), timeout=10)
        d = r.json()
        if d.get("code") == 0:
            data = d.get("data", {})
            total = data.get("total", 0) if isinstance(data, dict) else len(data) if isinstance(data, list) else "OK"
            record("batch", name, True, f"total={total}")
        else:
            record("batch", name, False, f"code={d.get('code')}")
    except Exception as e:
        record("batch", name, False, str(e)[:60])

# ============================================================
# 场景 6: 端到端跨角色流程
# ============================================================
print("\n[场景 6] 端到端跨角色流程 (端用户→工程师→经理)")
try:
    # 端用户提交
    r = requests.post(f"{BASE}/api/v1/tickets", headers=H(enduser_tk), json={
        "title": "[E2E] 影像系统PACS不能上传",
        "description": "放射科 PACS 上传 DICOM 失败",
        "priority": "high",
        "type": "incident"
    }, timeout=15)
    d = r.json()
    e2e_id = d.get("data", {}).get("id") if d.get("code") == 0 else None
    record("e2e", "端用户提交工单", d.get("code") == 0, f"工单 #{e2e_id}")
except Exception as e:
    record("e2e", "端用户提交", False, str(e)[:60])

if e2e_id:
    # 工程师接单
    try:
        r = requests.post(f"{BASE}/api/v1/tickets/{e2e_id}/assign", headers=H(manager_tk), json={"assigneeId": 4}, timeout=10)
        d = r.json()
        record("e2e", "经理分配工程师", d.get("code") == 0, f"{d.get('message')}")
    except Exception as e:
        record("e2e", "经理分配", False, str(e)[:60])
    
    try:
        r = requests.patch(f"{BASE}/api/v1/tickets/{e2e_id}", headers=H(agent_tk), json={"status": "in_progress"}, timeout=10)
        d = r.json()
        record("e2e", "工程师开始处理", d.get("code") == 0, f"{d.get('message')}")
    except Exception as e:
        record("e2e", "工程师处理", False, str(e)[:60])

    # 评论
    try:
        r = requests.post(f"{BASE}/api/v1/tickets/{e2e_id}/comments", headers=H(agent_tk), json={"content": "已检查 PACS 服务，重启后正常", "internal": False}, timeout=10)
        d = r.json()
        record("e2e", "工程师添加处理记录", d.get("code") == 0, f"{d.get('message')}")
    except Exception as e:
        record("e2e", "工程师评论", False, str(e)[:60])

    # 解决
    try:
        r = requests.post(f"{BASE}/api/v1/tickets/{e2e_id}/resolve", headers=H(agent_tk), json={"resolution": "已重启 PACS 服务并验证上传功能正常"}, timeout=10)
        d = r.json()
        record("e2e", "工程师标记已解决", d.get("code") == 0, f"{d.get('message')}")
    except Exception as e:
        record("e2e", "工程师解决", False, str(e)[:60])

# ============================================================
# 总结
# ============================================================
print("\n" + "="*60)
print("测试总结")
print("="*60)
total = len(results)
passed = sum(1 for r in results if r["ok"])
print(f"\n总测试数: {total} | 通过: {passed} | 失败: {total - passed} | 通过率: {passed/total*100:.1f}%\n")

from collections import defaultdict
groups = defaultdict(lambda: {"pass": 0, "fail": 0})
for r in results:
    if r["ok"]:
        groups[r["scenario"]]["pass"] += 1
    else:
        groups[r["scenario"]]["fail"] += 1

for scenario, g in groups.items():
    status = "✅" if g["fail"] == 0 else "⚠️ "
    print(f"  {status} {scenario:20s} {g['pass']}/{g['pass']+g['fail']} 通过")

with open("/Users/heidsoft/Downloads/research/itsm/test-results/robot-role/results_v2.json", "w") as f:
    json.dump({"total": total, "passed": passed, "failed": total - passed, "results": results}, f, ensure_ascii=False, indent=2)

print(f"\n详细结果: /Users/heidsoft/Downloads/research/itsm/test-results/robot-role/results_v2.json")
