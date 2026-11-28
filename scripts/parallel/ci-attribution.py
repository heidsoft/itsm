#!/usr/bin/env python3
"""CI 红项归因消费者：把 PR / base 的 CI 检查结果送入 Run Manifest 闭环。

设计边界（只观察与归并，不判定审批，不写 backlog）：
  - attribute : 读取 `gh pr checks --json` 与 base commit 的 check-runs JSON，
    按显式规则把失败项映射到已存在的 backlog ID；未命中的失败进入 unattributed，
    绝不猜测归属，也绝不自动新增 backlog 行；
  - merge     : 把归因结果合并进 Run Manifest 的 `ci` / `backlog_updates`。
    合并后清除旧的快照摘要，强制下一次 writeback 重新取快照。

随后仍由 scripts/parallel/run-manifest.py 负责校验与受保护的 writeback；
本工具不合并 PR、不部署、不迁移、不改权限策略，也不构成第二套审批引擎。

用法：
  python3 scripts/parallel/ci-attribution.py attribute \
    --pr-checks pr.json --base-checks base.json --base-sha <sha> > attribution.json
  python3 scripts/parallel/ci-attribution.py merge \
    --manifest manifest.json --attribution attribution.json --output manifest.new.json
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[2]
DEFAULT_CONFIG = Path(__file__).with_name("ci-attribution-map.json")
DEFAULT_BACKLOG = ROOT / "docs" / "audits" / "remediation-backlog.md"

PR_STATE_STATUS = {
    "SUCCESS": "pass",
    "FAILURE": "fail",
    "PENDING": "pending",
    "IN_PROGRESS": "pending",
    "QUEUED": "pending",
    "WAITING": "pending",
    "CANCELLED": "skipped",
    "SKIPPED": "skipped",
    "NEUTRAL": "skipped",
}
PR_BUCKET_STATUS = {
    "pass": "pass",
    "fail": "fail",
    "pending": "pending",
    "skipped": "skipped",
    "cancel": "skipped",
    "cancelled": "skipped",
    "canceled": "skipped",
}
API_CONCLUSION_STATUS = {
    "success": "pass",
    "failure": "fail",
    "timed_out": "fail",
    "cancelled": "skipped",
    "canceled": "skipped",
    "skipped": "skipped",
    "neutral": "skipped",
    "stale": "skipped",
    "action_required": "pending",
}
VALID_STATUS = {"pass", "fail", "pending", "skipped"}


def load_json(path: Path) -> Any:
    with path.open("r", encoding="utf-8") as handle:
        return json.load(handle)


def _pr_status(check: dict[str, Any]) -> str:
    bucket = str(check.get("bucket") or "").lower()
    if bucket in PR_BUCKET_STATUS:
        return PR_BUCKET_STATUS[bucket]
    state = str(check.get("state") or "").upper()
    if state in PR_STATE_STATUS:
        return PR_STATE_STATUS[state]
    return "pending"


def _api_status(check: dict[str, Any]) -> str:
    status = str(check.get("status") or "").lower()
    conclusion = str(check.get("conclusion") or "").lower()
    if status and status != "completed":
        return "pending"
    return API_CONCLUSION_STATUS.get(conclusion, "pending")


def _extract_run_id(link: str) -> str:
    match = re.search(r"/actions/runs/(\d+)", link)
    return match.group(1) if match else ""


def normalize_checks(payload: Any, source: str) -> list[dict[str, Any]]:
    """Normalize `gh pr checks --json` or GitHub check-runs API payloads."""
    if isinstance(payload, dict):
        raw = payload.get("check_runs", [])
        if not isinstance(raw, list):
            raise ValueError(f"{source}: check_runs must be an array")
    elif isinstance(payload, list):
        raw = payload
    else:
        raise ValueError(f"{source}: expected JSON object or array")

    checks: list[dict[str, Any]] = []
    for item in raw:
        if not isinstance(item, dict):
            raise ValueError(f"{source}: each check must be an object")
        if source == "base":
            status = _api_status(item)
            workflow = str((item.get("app") or {}).get("name") or "")
            link = str(item.get("html_url") or item.get("details_url") or "")
        else:
            status = _pr_status(item)
            workflow = str(item.get("workflow") or "")
            link = str(item.get("link") or "")
        if status not in VALID_STATUS:
            status = "pending"
        checks.append(
            {
                "name": str(item.get("name") or ""),
                "workflow": workflow,
                "status": status,
                "link": link,
                "run_id": _extract_run_id(link),
            }
        )
    return checks


def backlog_ids(backlog_text: str) -> set[str]:
    """Collect IDs of rows that already exist in the Markdown backlog."""
    return set(re.findall(r"^\|\s*([A-Z]+-\d+)\s*\|", backlog_text, re.MULTILINE))


def _rule_matches(rule: dict[str, Any], check: dict[str, Any]) -> bool:
    workflow = str(rule.get("workflow") or "")
    name_contains = str(rule.get("name_contains") or "")
    if not workflow and not name_contains:
        return False
    if workflow and workflow.lower() != check["workflow"].lower():
        return False
    if name_contains and name_contains.lower() not in check["name"].lower():
        return False
    return True


def classify(
    pr_checks: list[dict[str, Any]],
    base_checks: list[dict[str, Any]],
    rules: list[dict[str, Any]],
    backlog_text: str,
    base_sha: str = "",
) -> dict[str, Any]:
    """Pure classifier. Returns manifest-ready `ci` and `backlog_updates`."""
    existing_ids = backlog_ids(backlog_text)
    base_failed = {
        (check["name"].lower(), check["workflow"].lower())
        for check in base_checks
        if check["status"] == "fail"
    }

    ci_entries: list[dict[str, Any]] = []
    updates: list[dict[str, Any]] = []
    unattributed: list[dict[str, Any]] = []
    update_index: dict[str, int] = {}
    summary = {status: 0 for status in sorted(VALID_STATUS)}

    for check in pr_checks:
        status = check["status"]
        summary[status] += 1
        baseline = status == "fail" and (
            check["name"].lower(),
            check["workflow"].lower(),
        ) in base_failed
        entry: dict[str, Any] = {
            "name": check["name"],
            "status": status,
            "baseline": baseline,
        }
        if check["run_id"]:
            entry["run_id"] = check["run_id"]
        ci_entries.append(entry)

        if status != "fail":
            continue

        rule = next((candidate for candidate in rules if _rule_matches(candidate, check)), None)
        if rule is None:
            unattributed.append(
                {
                    "name": check["name"],
                    "workflow": check["workflow"],
                    "reason": "no matching attribution rule",
                    "link": check["link"],
                }
            )
            continue
        item_id = str(rule.get("backlog_id") or "")
        if item_id not in existing_ids:
            unattributed.append(
                {
                    "name": check["name"],
                    "workflow": check["workflow"],
                    "reason": f"backlog ID {item_id} does not exist",
                    "link": check["link"],
                }
            )
            continue

        evidence = (
            f"PR check {check['name']}: {check['link']}; "
            f"base_sha={base_sha or 'unknown'}; baseline={str(baseline).lower()}"
        )
        update = {
            "id": item_id,
            "status": str(rule.get("status") or ("confirmed" if baseline else "suspected")),
            "note": f"{rule.get('note') or 'CI 红项归因'}（{'基线复现' if baseline else '新增失败'}）",
            "evidence": evidence,
        }
        if item_id in update_index:
            prior = updates[update_index[item_id]]
            prior["evidence"] += "; " + evidence
        else:
            update_index[item_id] = len(updates)
            updates.append(update)

    return {
        "schema_version": "1.0",
        "summary": summary,
        "ci": ci_entries,
        "backlog_updates": updates,
        "unattributed": unattributed,
    }


def command_attribute(args: argparse.Namespace) -> int:
    pr_payload = load_json(Path(args.pr_checks))
    base_payload = load_json(Path(args.base_checks))
    config = load_json(Path(args.config))
    rules = config.get("rules", []) if isinstance(config, dict) else []
    if not isinstance(rules, list):
        print("INVALID config: rules must be an array", file=sys.stderr)
        return 1
    backlog_text = Path(args.backlog).read_text(encoding="utf-8")

    try:
        pr_checks = normalize_checks(pr_payload, "pr")
        base_checks = normalize_checks(base_payload, "base")
    except ValueError as exc:
        print(f"INVALID input: {exc}", file=sys.stderr)
        return 1

    result = classify(pr_checks, base_checks, rules, backlog_text, args.base_sha or "")
    rendered = json.dumps(result, ensure_ascii=False, indent=2, sort_keys=True)
    if args.output:
        Path(args.output).write_text(rendered + "\n", encoding="utf-8")
    else:
        print(rendered)

    if args.strict_unattributed and result["unattributed"]:
        print(
            f"REFUSED: {len(result['unattributed'])} failed check(s) have no verified attribution",
            file=sys.stderr,
        )
        return 1
    return 0


def command_merge(args: argparse.Namespace) -> int:
    manifest = load_json(Path(args.manifest))
    attribution = load_json(Path(args.attribution))
    if not isinstance(manifest, dict) or not isinstance(attribution, dict):
        print("INVALID: manifest and attribution must be JSON objects", file=sys.stderr)
        return 1
    if not isinstance(attribution.get("ci"), list):
        print("INVALID: attribution.ci must be an array", file=sys.stderr)
        return 1
    if not isinstance(attribution.get("backlog_updates", []), list):
        print("INVALID: attribution.backlog_updates must be an array", file=sys.stderr)
        return 1

    merged = dict(manifest)
    merged["ci"] = attribution["ci"]
    merged["backlog_updates"] = attribution.get("backlog_updates", [])
    # The merged manifest has different content; stale snapshots must be regenerated
    # by the caller before run-manifest.py --apply is allowed to write anything.
    merged.pop("backlog_snapshot_sha256", None)
    merged.pop("ledger_snapshot_sha256", None)

    rendered = json.dumps(merged, ensure_ascii=False, indent=2, sort_keys=True)
    if args.output:
        Path(args.output).write_text(rendered + "\n", encoding="utf-8")
    else:
        print(rendered)
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--config", default=str(DEFAULT_CONFIG), help="attribution rule config")
    sub = parser.add_subparsers(dest="command", required=True)

    attribute = sub.add_parser("attribute", help="classify PR CI checks against base checks")
    attribute.add_argument("--pr-checks", required=True, help="gh pr checks --json output")
    attribute.add_argument("--base-checks", required=True, help="base commit check-runs JSON")
    attribute.add_argument("--base-sha", default="", help="base commit SHA used as evidence")
    attribute.add_argument(
        "--backlog", default=str(DEFAULT_BACKLOG), help="backlog Markdown for ID existence checks"
    )
    attribute.add_argument("--output", help="write JSON to file instead of stdout")
    attribute.add_argument(
        "--strict-unattributed",
        action="store_true",
        help="exit 1 when any failed check has no verified attribution",
    )
    attribute.set_defaults(func=command_attribute)

    merge = sub.add_parser("merge", help="merge attribution into a Run Manifest")
    merge.add_argument("--manifest", required=True)
    merge.add_argument("--attribution", required=True)
    merge.add_argument("--output", help="write JSON to file instead of stdout")
    merge.set_defaults(func=command_merge)

    return parser


def main() -> int:
    args = build_parser().parse_args()
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())
