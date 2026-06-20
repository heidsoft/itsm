#!/usr/bin/env python3
"""Stage 4 runner: Marketplace + AI"""
import sys
import json
import time
from pathlib import Path
from typing import Any, Dict

sys.path.insert(0, str(Path(__file__).parent))
from itsm_api_runner import (
    load_token, load_ids, save_id, api_call, run_test_case, run_stage
)

BASE = Path("/Users/heidsoft/Downloads/research/itsm")
CASES_FILE = BASE / "output/functional-test/cases_stage4.json"
RESULT_FILE = BASE / "output/functional-test/stage4.json"
IDS_FILE = BASE / "output/functional-test/ids_index.json"

def main():
    token = load_token()
    if not token:
        print("[ERROR] no token, please login first")
        return 1

    # 重新登录以避免 token 过期
    import urllib.request
    req = urllib.request.Request(
        "http://localhost:8090/api/v1/auth/login",
        data=json.dumps({"username":"admin","password":"admin123"}).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST"
    )
    with urllib.request.urlopen(req, timeout=10) as resp:
        d = json.loads(resp.read().decode("utf-8"))
        token = d.get("data", {}).get("access_token") or d.get("access_token") or d.get("token", "")
        if token:
            with open("/tmp/itsm_test_token.env", "w") as f:
                f.write(f"TOKEN={token}\n")

    with open(CASES_FILE, "r", encoding="utf-8") as f:
        cases = json.load(f)["cases"]

    ids = load_ids()
    print(f"[stage4] running {len(cases)} test cases")
    print(f"[stage4] existing ids: {list(ids.keys())}")

    # 注入 ID 占位符
    for tc in cases:
        path = tc.get("path", "")
        if "__item_id__" in path:
            tc["path"] = path.replace("__item_id__", str(ids.get("item_id", 1)))
        if "__install_id__" in path:
            tc["path"] = path.replace("__install_id__", str(ids.get("install_id", 1)))
        if "__ticket_id__" in path:
            tc["path"] = path.replace("__ticket_id__", str(ids.get("ticket_id", 1)))

    # 提取 item_id / install_id 从前面用例
    # 提前跑 install 然后获取 install_id
    summary = run_stage("stage4", cases)

    # 保存结果
    with open(RESULT_FILE, "w", encoding="utf-8") as f:
        json.dump(summary, f, ensure_ascii=False, indent=2)

    # 统计
    total = len(summary["results"])
    passed = sum(1 for r in summary["results"] if r["status"] == "PASS")
    print(f"\n[stage4] {passed}/{total} passed ({100*passed/total:.1f}%)")
    print(f"[stage4] result saved to {RESULT_FILE}")

    # 打印失败用例
    print("\n[stage4] FAILED CASES:")
    for r in summary["results"]:
        if r["status"] != "PASS":
            print(f"  - {r['id']} {r['name']}: {r.get('fail_reason', r.get('http_code'))}")

    return 0

if __name__ == "__main__":
    sys.exit(main())
