#!/usr/bin/env python3
"""CloudMesh 自主研发闭环：Run Manifest 校验与 backlog 回写。

设计边界（只读观察 + 显式回写，不引入第二套审批引擎）：
  - validate : 按 scripts/parallel/run-manifest.schema.json 的 JSON Schema 子集校验清单；
  - writeback: 将清单中的 backlog_updates 回写到整改 backlog，并把本次 run 作为一条
    JSON 记录 upsert 到 docs/audits/autonomous-runs.jsonl（按 run_id 幂等）。
默认 dry-run；只有 --apply 才写文件。--apply 要求 backlog 快照摘要匹配，
防止覆盖并行会话的用户本地修改。不执行部署、数据库迁移、权限策略变更或合并。

用法：
  python3 scripts/parallel/run-manifest.py validate <manifest.json>
  python3 scripts/parallel/run-manifest.py writeback <manifest.json> [--apply]
"""

from __future__ import annotations

import argparse
import difflib
import hashlib
import json
import re
import sys
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[2]
DEFAULT_SCHEMA = Path(__file__).with_name("run-manifest.schema.json")
DEFAULT_BACKLOG = ROOT / "docs" / "audits" / "remediation-backlog.md"
DEFAULT_LEDGER = ROOT / "docs" / "audits" / "autonomous-runs.jsonl"


def load_json(path: Path) -> Any:
    with path.open("r", encoding="utf-8") as handle:
        return json.load(handle)


def sha256_text(text: str) -> str:
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def _type_ok(value: Any, expected: str) -> bool:
    if expected == "object":
        return isinstance(value, dict)
    if expected == "array":
        return isinstance(value, list)
    if expected == "string":
        return isinstance(value, str)
    if expected == "boolean":
        return isinstance(value, bool)
    if expected == "integer":
        return isinstance(value, int) and not isinstance(value, bool)
    if expected == "number":
        return isinstance(value, (int, float)) and not isinstance(value, bool)
    return True


def validate(value: Any, schema: dict[str, Any], path: str = "$") -> list[str]:
    """Validate with the documented JSON Schema subset. Returns error strings."""
    errors: list[str] = []
    expected = schema.get("type")
    if expected and not _type_ok(value, expected):
        return [f"{path}: expected {expected}, got {type(value).__name__}"]

    if "enum" in schema and value not in schema["enum"]:
        errors.append(f"{path}: value {value!r} not in enum {schema['enum']!r}")

    if isinstance(value, dict):
        for key in schema.get("required", []):
            if key not in value:
                errors.append(f"{path}: missing required field '{key}'")
        for key, child in value.items():
            child_schema = schema.get("properties", {}).get(key)
            if child_schema is not None:
                errors.extend(validate(child, child_schema, f"{path}.{key}"))
    elif isinstance(value, list):
        minimum = schema.get("minItems")
        if minimum is not None and len(value) < minimum:
            errors.append(f"{path}: expected at least {minimum} items, got {len(value)}")
        item_schema = schema.get("items")
        if isinstance(item_schema, dict):
            for index, item in enumerate(value):
                errors.extend(validate(item, item_schema, f"{path}[{index}]"))
    elif isinstance(value, str):
        minimum = schema.get("minLength")
        if minimum is not None and len(value) < minimum:
            errors.append(f"{path}: shorter than minLength {minimum}")
        pattern = schema.get("pattern")
        if pattern is not None and re.search(pattern, value) is None:
            errors.append(f"{path}: does not match pattern {pattern}")
    elif isinstance(value, int) and not isinstance(value, bool):
        minimum = schema.get("minimum")
        if minimum is not None and value < minimum:
            errors.append(f"{path}: value {value} is below minimum {minimum}")
    return errors


def validate_manifest(manifest: Any, schema: dict[str, Any]) -> list[str]:
    return validate(manifest, schema)


def ledger_record(manifest: dict[str, Any]) -> dict[str, Any]:
    """Build a deterministic run record; no wall-clock reads in the pure path."""
    ci_summary: dict[str, int] = {}
    for check in manifest.get("ci", []):
        status = check.get("status", "unknown")
        ci_summary[status] = ci_summary.get(status, 0) + 1
    task = manifest["task"]
    return {
        "run_id": manifest["run_id"],
        "generated_at": manifest["generated_at"],
        "status": manifest["status"],
        "repository": manifest["repository"],
        "base_commit": manifest["base_commit"],
        "branch": manifest["branch"],
        "commits": manifest.get("commits", []),
        "pull_request": manifest.get("pull_request"),
        "task": {"id": task["id"], "title": task["title"]},
        "tests": [
            {
                "command": item["command"],
                "exit_code": item["exit_code"],
                "note": item.get("note", ""),
            }
            for item in manifest.get("tests", [])
        ],
        "ci_summary": ci_summary,
        "security": manifest["security"],
        "approvals": manifest["approvals"],
        "blockers": manifest.get("blockers", []),
        "backlog_updates": manifest.get("backlog_updates", []),
    }


def _update_backlog(backlog_text: str, updates: list[dict[str, Any]], run_id: str) -> tuple[str, list[str]]:
    errors: list[str] = []
    lines = backlog_text.splitlines(keepends=True)
    for update in updates:
        item_id = update["id"]
        pattern = re.compile(r"^\|\s*" + re.escape(item_id) + r"\s*\|")
        index = next((i for i, line in enumerate(lines) if pattern.match(line)), None)
        if index is None:
            errors.append(f"backlog: ID {item_id} not found; refusing to create rows implicitly")
            continue
        line = lines[index]
        newline = "\n" if line.endswith("\n") else ""
        cells = [cell.strip() for cell in line.rstrip("\n").strip().strip("|").split("|")]
        if len(cells) < 3:
            errors.append(f"backlog: row {item_id} has fewer than 3 columns")
            continue
        cells[2] = f"`{update['status']}`（Run `{run_id}`）"
        marker = f"Run {run_id}"
        last = len(cells) - 1
        if marker not in cells[last]:
            cells[last] = (
                f"{cells[last]}；{marker}: {update['note']}（{update['evidence']}）"
            )
        lines[index] = "| " + " | ".join(cells) + " |" + newline
    return "".join(lines), errors


def _upsert_ledger(ledger_text: str, record: dict[str, Any]) -> tuple[str, list[str]]:
    encoded = json.dumps(record, ensure_ascii=False, sort_keys=True)
    output: list[str] = []
    replaced = False
    errors: list[str] = []
    for line in ledger_text.splitlines():
        if not line.strip():
            continue
        try:
            existing = json.loads(line)
        except json.JSONDecodeError:
            errors.append("ledger: existing line is not valid JSON; refusing to rewrite it")
            continue
        if isinstance(existing, dict) and existing.get("run_id") == record["run_id"]:
            output.append(encoded)
            replaced = True
        else:
            output.append(line)
    if errors:
        return ledger_text, errors
    if not replaced:
        output.append(encoded)
    return "\n".join(output) + "\n", []


def compute_writeback(
    manifest: dict[str, Any], backlog_text: str, ledger_text: str
) -> tuple[str, str, list[str]]:
    """Pure writeback planner. Returns (new_backlog, new_ledger, errors)."""
    updates = manifest.get("backlog_updates", [])
    new_backlog, errors = _update_backlog(backlog_text, updates, manifest["run_id"])
    new_ledger, ledger_errors = _upsert_ledger(ledger_text, ledger_record(manifest))
    return new_backlog, new_ledger, errors + ledger_errors


def _print_diff(label: str, old: str, new: str) -> None:
    diff = difflib.unified_diff(
        old.splitlines(keepends=True),
        new.splitlines(keepends=True),
        fromfile=f"a/{label}",
        tofile=f"b/{label}",
    )
    for line in diff:
        sys.stdout.write(line)


def _atomic_write(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temp = path.with_name(path.name + ".tmp")
    temp.write_text(text, encoding="utf-8")
    temp.replace(path)


def command_validate(args: argparse.Namespace) -> int:
    schema = load_json(Path(args.schema))
    manifest = load_json(Path(args.manifest))
    errors = validate_manifest(manifest, schema)
    if errors:
        for error in errors:
            print(f"INVALID {error}", file=sys.stderr)
        return 1
    print(f"VALID {args.manifest} ({len(manifest)} top-level fields)")
    return 0


def command_writeback(args: argparse.Namespace) -> int:
    schema = load_json(Path(args.schema))
    manifest = load_json(Path(args.manifest))
    errors = validate_manifest(manifest, schema)
    if errors:
        for error in errors:
            print(f"INVALID {error}", file=sys.stderr)
        return 1

    backlog_path = Path(args.backlog)
    ledger_path = Path(args.ledger)
    backlog_text = backlog_path.read_text(encoding="utf-8") if backlog_path.exists() else ""
    ledger_text = ledger_path.read_text(encoding="utf-8") if ledger_path.exists() else ""

    updates = manifest.get("backlog_updates", [])
    if updates and args.apply:
        expected = manifest.get("backlog_snapshot_sha256")
        if not expected:
            print("REFUSED: --apply with backlog_updates requires backlog_snapshot_sha256", file=sys.stderr)
            return 1
        if sha256_text(backlog_text) != expected:
            print("REFUSED: backlog changed since manifest snapshot; refusing to overwrite local work", file=sys.stderr)
            return 1
    expected_ledger = manifest.get("ledger_snapshot_sha256")
    if args.apply and expected_ledger and sha256_text(ledger_text) != expected_ledger:
        print("REFUSED: ledger changed since manifest snapshot; refusing to overwrite local work", file=sys.stderr)
        return 1

    new_backlog, new_ledger, errors = compute_writeback(manifest, backlog_text, ledger_text)
    if errors:
        for error in errors:
            print(f"REFUSED {error}", file=sys.stderr)
        return 1

    if not args.apply:
        _print_diff(str(backlog_path), backlog_text, new_backlog)
        _print_diff(str(ledger_path), ledger_text, new_ledger)
        print("DRY-RUN: no files written; re-run with --apply to commit the writeback.")
        return 0

    if new_backlog != backlog_text:
        _atomic_write(backlog_path, new_backlog)
    if new_ledger != ledger_text:
        _atomic_write(ledger_path, new_ledger)
    print(
        f"APPLIED run={manifest['run_id']} backlog_updates={len(updates)} "
        f"ledger={ledger_path}"
    )
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--schema", default=str(DEFAULT_SCHEMA), help="manifest schema path")
    sub = parser.add_subparsers(dest="command", required=True)

    validate = sub.add_parser("validate", help="validate a Run Manifest")
    validate.add_argument("manifest")
    validate.set_defaults(func=command_validate)

    writeback = sub.add_parser("writeback", help="write run record and backlog updates")
    writeback.add_argument("manifest")
    writeback.add_argument("--backlog", default=str(DEFAULT_BACKLOG))
    writeback.add_argument("--ledger", default=str(DEFAULT_LEDGER))
    writeback.add_argument("--apply", action="store_true", help="actually write files (default: dry-run)")
    writeback.set_defaults(func=command_writeback)
    return parser


def main() -> int:
    args = build_parser().parse_args()
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())
