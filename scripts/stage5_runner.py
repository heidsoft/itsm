#!/usr/bin/env python3
"""Stage 5 runner: Auxiliary modules"""
import sys
import json
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from itsm_api_runner import run_stage

BASE = Path("/Users/heidsoft/Downloads/research/itsm")

def main():
    # 重新登录
    import urllib.request
    req = urllib.request.Request(
        "http://localhost:8090/api/v1/auth/login",
        data=json.dumps({"username":"admin","password":"admin123"}).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST"
    )
    with urllib.request.urlopen(req, timeout=10) as resp:
        d = json.loads(resp.read().decode("utf-8"))
        token = d.get("data", {}).get("access_token") or d.get("access_token") or ""
        if token:
            with open("/tmp/itsm_test_token.env", "w") as f:
                f.write(f"TOKEN={token}\n")

    with open(BASE / "output/functional-test/cases_stage5.json") as f:
        cases = json.load(f)["cases"]

    print(f"[stage5] running {len(cases)} test cases")
    summary = run_stage("stage5", cases)
    return 0

if __name__ == "__main__":
    sys.exit(main())
