#!/usr/bin/env python3
"""
ITSM 通用 API 测试运行器
使用方法: python3 itsm_api_runner.py <stage_name> <test_cases_json>
或: python3 itsm_api_runner.py --interactive
"""
import json
import os
import subprocess
import sys
import time
from typing import Any, Dict, List, Optional

# 配置
BASE = os.environ.get("ITSM_BASE", "http://localhost:8090/api/v1")
TOKEN_FILE = "/tmp/itsm_test_token.env"
IDS_FILE = "/Users/heidsoft/Downloads/research/itsm/output/functional-test/ids_index.json"
OUT_DIR = "/Users/heidsoft/Downloads/research/itsm/output/functional-test"

# 加载 token
def load_token() -> str:
    if os.path.exists(TOKEN_FILE):
        with open(TOKEN_FILE) as f:
            for line in f:
                if line.startswith("TOKEN="):
                    return line.split("=", 1)[1].strip()
    raise RuntimeError(f"Token 文件 {TOKEN_FILE} 不存在")

# 加载/保存 ID 索引
def load_ids() -> Dict[str, Any]:
    if os.path.exists(IDS_FILE):
        with open(IDS_FILE) as f:
            return json.load(f)
    return {}

def save_ids(ids: Dict[str, Any]) -> None:
    with open(IDS_FILE, "w") as f:
        json.dump(ids, f, indent=2, ensure_ascii=False)

def save_id(key: str, value: Any) -> None:
    ids = load_ids()
    ids[key] = value
    save_ids(ids)

def get_id(key: str) -> Any:
    return load_ids().get(key)

# 通用 HTTP 调用（使用绝对路径避免 PATH 问题）
def _replace_in_dict(d: Any, placeholder_key: str, value: Any) -> Any:
    """递归替换字典中的占位符，保留原始类型"""
    if isinstance(d, dict):
        return {k: _replace_in_dict(v, placeholder_key, value) for k, v in d.items()}
    elif isinstance(d, list):
        return [_replace_in_dict(x, placeholder_key, value) for x in d]
    elif isinstance(d, str):
        if d == f"__{placeholder_key}__":
            return value  # 保留原始类型
        return d.replace(f"__{placeholder_key}__", str(value))
    return d


def api_call(method: str, path: str, data: Optional[str] = None,
             token: Optional[str] = None, timeout: int = 30) -> Dict[str, Any]:
    token = token or load_token()
    cmd = ["/usr/bin/curl", "-s", "-w", "\n%{http_code}", "-X", method,
           f"{BASE}{path}", "-H", f"Authorization: Bearer {token}",
           "-H", "Content-Type: application/json"]
    if data:
        cmd += ["-d", data]
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
        out = result.stdout
        # 最后一行是 http_code
        parts = out.rsplit("\n", 1)
        body = parts[0] if len(parts) == 2 else ""
        http_code = parts[1] if len(parts) == 2 else "0"
        try:
            body_json = json.loads(body) if body else None
        except json.JSONDecodeError:
            body_json = {"_raw": body[:500]}
        return {
            "http_code": http_code,
            "body": body_json,
            "raw": body[:1000],
        }
    except subprocess.TimeoutExpired:
        return {"http_code": "TIMEOUT", "body": None, "raw": "timeout"}
    except Exception as e:
        return {"http_code": "ERROR", "body": None, "raw": str(e)}

# 简化接口
def api_get(path: str, token: Optional[str] = None) -> Dict[str, Any]:
    return api_call("GET", path, token=token)

def api_post(path: str, data: Optional[Dict] = None, token: Optional[str] = None) -> Dict[str, Any]:
    body = json.dumps(data, ensure_ascii=False) if data else None
    return api_call("POST", path, data=body, token=token)

def api_put(path: str, data: Optional[Dict] = None, token: Optional[str] = None) -> Dict[str, Any]:
    body = json.dumps(data, ensure_ascii=False) if data else None
    return api_call("PUT", path, data=body, token=token)

def api_delete(path: str, token: Optional[str] = None) -> Dict[str, Any]:
    return api_call("DELETE", path, token=token)

# 测试用例结构：
# {"name": "...", "method": "GET/POST/...", "path": "...", "data": {...}, "expect": {"http": [200], "code": [0, ...]}, "save_id_key": "..."}

def run_test_case(tc: Dict[str, Any]) -> Dict[str, Any]:
    """运行单个测试用例"""
    name = tc.get("name", tc.get("path", ""))
    method = tc.get("method", "GET").upper()
    path = tc.get("path", "")
    data = tc.get("data")
    expect = tc.get("expect", {})
    save_key = tc.get("save_id_key")
    save_path = tc.get("save_id_path")  # 从响应 body 中按路径提取

    # 占位符替换: __TICKET_ID__ -> ids_index.ticket_id (保留原始类型)
    ids = load_ids()
    for k, v in ids.items():
        if v is not None and v != "":
            path = path.replace(f"__{k.upper()}__", str(v))
            if isinstance(data, dict):
                data = _replace_in_dict(data, k.upper(), v)

    # 兼容 expect_status 字段（简写）
    if "expect_status" in tc and "http" not in expect:
        expect = {**expect, "http": tc["expect_status"]}
    if "expect_code" in tc and "code" not in expect:
        expect = {**expect, "code": tc["expect_code"]}

    if method == "GET":
        r = api_get(path)
    elif method == "POST":
        r = api_post(path, data)
    elif method == "PUT":
        r = api_put(path, data)
    elif method == "DELETE":
        r = api_delete(path)
    else:
        return {"name": name, "status": "ERROR", "msg": f"unknown method {method}"}

    # 评估期望
    http_code = r["http_code"]
    body_code = r.get("body", {}).get("code") if isinstance(r.get("body"), dict) else None

    passed = True
    fail_reason = []

    if "http" in expect:
        expected_http = expect["http"]
        if isinstance(expected_http, int):
            expected_http = [expected_http]
        if int(http_code) not in expected_http:
            passed = False
            fail_reason.append(f"http_code {http_code} not in {expected_http}")

    if "code" in expect:
        expected_code = expect["code"]
        if isinstance(expected_code, int):
            expected_code = [expected_code]
        if body_code not in expected_code:
            passed = False
            fail_reason.append(f"body.code {body_code} not in {expected_code}")

    # 提取并保存 ID
    saved_id = None
    if save_key and isinstance(r.get("body"), dict):
        body = r["body"]
        data_field = body.get("data")
        if save_path and isinstance(data_field, dict):
            # 按点路径提取
            parts = save_path.split(".")
            v = data_field
            for p in parts:
                if isinstance(v, dict):
                    v = v.get(p)
                elif isinstance(v, list) and p.isdigit():
                    v = v[int(p)] if int(p) < len(v) else None
                else:
                    v = None
                    break
            if v is not None:
                save_id(save_key, v)
                saved_id = v
        elif data_field is not None:
            save_id(save_key, data_field)
            saved_id = data_field

    return {
        "id": tc.get("id", ""),
        "name": name,
        "module": tc.get("module", ""),
        "method": method,
        "path": path,
        "http": http_code,
        "code": body_code,
        "status": "PASS" if passed else "FAIL",
        "msg": r.get("body", {}).get("message", "") if isinstance(r.get("body"), dict) else "",
        "passed": passed,
        "fail_reason": fail_reason,
        "saved_id": saved_id,
        "raw_preview": r["raw"][:200],
    }


def run_stage(stage_name: str, test_cases: List[Dict[str, Any]]) -> Dict[str, Any]:
    """运行整个阶段"""
    results = []
    pass_count = 0
    fail_count = 0

    print(f"\n{'='*60}")
    print(f"Stage: {stage_name}")
    print(f"{'='*60}\n")

    for i, tc in enumerate(test_cases, 1):
        result = run_test_case(tc)
        results.append(result)
        if result["passed"]:
            pass_count += 1
            icon = "✅"
        else:
            fail_count += 1
            icon = "❌"
        msg_short = result.get("msg", "")[:80] if not result["passed"] else ""
        saved = f" -> saved[{tc.get('save_id_key', '')}]={result['saved_id']}" if result.get('saved_id') is not None else ""
        print(f"  [{i:3d}] {icon} {result['method']:6s} {result['path']:50s} http={result['http']} code={result['code']}{saved}")
        if not result["passed"]:
            for fr in result["fail_reason"]:
                print(f"          FAIL: {fr}")
            print(f"          msg: {msg_short}")
            print(f"          raw: {result['raw_preview']}")

    summary = {
        "stage": stage_name,
        "total": len(test_cases),
        "passed": pass_count,
        "failed": fail_count,
        "pass_rate": round(pass_count / len(test_cases) * 100, 1) if test_cases else 0,
        "results": results,
    }

    out_file = os.path.join(OUT_DIR, f"{stage_name}.json")
    with open(out_file, "w") as f:
        json.dump(summary, f, indent=2, ensure_ascii=False)

    print(f"\n{'-'*60}")
    print(f"Stage {stage_name}: {pass_count}/{len(test_cases)} passed ({summary['pass_rate']}%)")
    print(f"  Output: {out_file}")
    print(f"{'-'*60}\n")

    return summary


if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: python3 itsm_api_runner.py <stage_name> <test_cases.json>")
        sys.exit(1)

    stage_name = sys.argv[1]
    cases_file = sys.argv[2]
    with open(cases_file) as f:
        cases = json.load(f)
    summary = run_stage(stage_name, cases)
    sys.exit(0 if summary["failed"] == 0 else 1)