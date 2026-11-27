#!/usr/bin/env python3
"""Unit + CLI tests for scripts/parallel/run-manifest.py (stdlib only)."""

from __future__ import annotations

import copy
import importlib.util
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPTS = Path(__file__).resolve().parents[1]
SCRIPT = SCRIPTS / "parallel" / "run-manifest.py"
SCHEMA = SCRIPTS / "parallel" / "run-manifest.schema.json"

spec = importlib.util.spec_from_file_location("run_manifest", SCRIPT)
assert spec and spec.loader
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


def base_manifest() -> dict:
    return {
        "schema_version": "1.0",
        "run_id": "cm-auto-test-01",
        "generated_at": "2026-11-26T00:00:00+08:00",
        "status": "partial",
        "repository": "heidsoft/itsm",
        "base_commit": "d8654786534febbabf91305b553a347eaba64498",
        "branch": "feat/test",
        "commits": ["56b59334"],
        "pull_request": {"number": 131, "url": "https://github.com/heidsoft/itsm/pull/131", "head": "56b59334"},
        "task": {
            "id": "AUDIT-070",
            "title": "写路径行级范围",
            "rationale": "高价值低风险",
            "evidence": [{"path": "internal/services/tool_registry.go", "lines": "776", "fact": "passed 0"}],
        },
        "tests": [{"command": "go test ./internal/services", "exit_code": 0, "note": "green"}],
        "ci": [{"name": "Lint", "status": "pass", "run_id": "123"}],
        "security": {"policy_changed": False, "db_migration": False, "deploy": False},
        "approvals": {"required": False, "pending": []},
        "risks": ["fail-closed"],
        "blockers": ["baseline scan"],
        "deployment": {"performed": False},
        "rollback": "revert commits",
    }


BACKLOG = """# 整改 Backlog

## P1

| ID | 问题 | 状态 | 处置 | 审批要求 |
|:---|:---|:---|:---|:---|
| AUDIT-070 | 工具越权 | `confirmed` | 待修 | 低风险 |
"""


class ValidationTests(unittest.TestCase):
    def setUp(self) -> None:
        self.schema = json.loads(SCHEMA.read_text(encoding="utf-8"))

    def test_accepts_complete_manifest(self) -> None:
        self.assertEqual(module.validate_manifest(base_manifest(), self.schema), [])

    def test_rejects_missing_exit_code(self) -> None:
        manifest = base_manifest()
        del manifest["tests"][0]["exit_code"]
        errors = module.validate_manifest(manifest, self.schema)
        self.assertTrue(any("exit_code" in error for error in errors), errors)

    def test_rejects_unknown_status(self) -> None:
        manifest = base_manifest()
        manifest["status"] = "unknown"
        errors = module.validate_manifest(manifest, self.schema)
        self.assertTrue(any("enum" in error for error in errors), errors)

    def test_rejects_bool_as_integer_exit_code(self) -> None:
        manifest = base_manifest()
        manifest["tests"][0]["exit_code"] = True
        errors = module.validate_manifest(manifest, self.schema)
        self.assertTrue(any("expected integer" in error for error in errors), errors)


class WritebackTests(unittest.TestCase):
    def test_writeback_is_idempotent_and_upserts_by_run_id(self) -> None:
        manifest = base_manifest()
        manifest["backlog_updates"] = [
            {
                "id": "AUDIT-070",
                "status": "resolved",
                "note": "PR #131 red/green",
                "evidence": "exit 1 -> 0",
            }
        ]
        first_backlog, first_ledger, errors = module.compute_writeback(manifest, BACKLOG, "")
        self.assertEqual(errors, [])
        self.assertIn("`resolved`（Run `cm-auto-test-01`）", first_backlog)
        self.assertEqual(len(first_ledger.strip().splitlines()), 1)

        second_backlog, second_ledger, errors = module.compute_writeback(manifest, first_backlog, first_ledger)
        self.assertEqual(errors, [])
        self.assertEqual(second_backlog, first_backlog)
        self.assertEqual(second_ledger, first_ledger)

    def test_unknown_backlog_id_is_refused(self) -> None:
        manifest = base_manifest()
        manifest["backlog_updates"] = [
            {"id": "AUDIT-999", "status": "resolved", "note": "x", "evidence": "y"}
        ]
        _, _, errors = module.compute_writeback(manifest, BACKLOG, "")
        self.assertTrue(any("AUDIT-999" in error for error in errors), errors)

    def test_malformed_ledger_is_refused(self) -> None:
        _, ledger, errors = module.compute_writeback(base_manifest(), BACKLOG, "not-json\n")
        self.assertEqual(ledger, "not-json\n")
        self.assertTrue(any("not valid JSON" in error for error in errors), errors)


class CliTests(unittest.TestCase):
    def test_cli_validate_dry_run_and_apply_guards(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            manifest = base_manifest()
            manifest["backlog_updates"] = [
                {"id": "AUDIT-070", "status": "resolved", "note": "n", "evidence": "e"}
            ]
            manifest_path = root / "manifest.json"
            manifest_path.write_text(json.dumps(manifest, ensure_ascii=False), encoding="utf-8")
            backlog_path = root / "backlog.md"
            backlog_path.write_text(BACKLOG, encoding="utf-8")
            ledger_path = root / "ledger.jsonl"

            def run(*extra: str) -> subprocess.CompletedProcess:
                return subprocess.run(
                    [sys.executable, str(SCRIPT), *extra],
                    capture_output=True,
                    text=True,
                    check=False,
                )

            result = run("validate", str(manifest_path))
            self.assertEqual(result.returncode, 0, result.stderr)
            result = run(
                "writeback",
                str(manifest_path),
                "--backlog",
                str(backlog_path),
                "--ledger",
                str(ledger_path),
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("DRY-RUN", result.stdout)
            self.assertNotIn("`resolved`", backlog_path.read_text(encoding="utf-8"))

            result = run(
                "writeback",
                str(manifest_path),
                "--backlog",
                str(backlog_path),
                "--ledger",
                str(ledger_path),
                "--apply",
            )
            self.assertEqual(result.returncode, 1)
            self.assertIn("backlog_snapshot_sha256", result.stderr)

            manifest["backlog_snapshot_sha256"] = module.sha256_text(BACKLOG)
            manifest_path.write_text(json.dumps(manifest, ensure_ascii=False), encoding="utf-8")
            result = run(
                "writeback",
                str(manifest_path),
                "--backlog",
                str(backlog_path),
                "--ledger",
                str(ledger_path),
                "--apply",
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("`resolved`", backlog_path.read_text(encoding="utf-8"))
            self.assertEqual(len(ledger_path.read_text(encoding="utf-8").strip().splitlines()), 1)


if __name__ == "__main__":
    unittest.main()
