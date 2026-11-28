import importlib.util
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
MODULE_PATH = ROOT / "scripts" / "parallel" / "ci-attribution.py"
SPEC = importlib.util.spec_from_file_location("ci_attribution", MODULE_PATH)
ci = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(ci)

BACKLOG_TEXT = """# Backlog
| ID | 问题 | 状态 | 处置 |
|:---|:---|:---|:---|
| AUDIT-071 | Security Scan 基线红项回归观察 | `suspected` | 工具链升级需审批 |
"""

DEFAULT_RULES = [
    {
        "workflow": "Security Scan",
        "name_contains": "Trivy Gate",
        "backlog_id": "AUDIT-071",
        "status": "suspected",
        "note": "Trivy Gate 历史红项",
    }
]


def pr_check(name, state, bucket, workflow="Security Scan", link=""):
    return {
        "name": name,
        "state": state,
        "bucket": bucket,
        "workflow": workflow,
        "link": link,
    }


class CiAttributionTests(unittest.TestCase):
    def test_normalize_pr_checks_maps_gh_states_and_run_id(self):
        payload = [
            pr_check("A", "SUCCESS", "pass", link="https://github.com/o/r/actions/runs/123/job/1"),
            pr_check("B", "FAILURE", "fail"),
            pr_check("C", "CANCELLED", "cancel"),
            pr_check("D", "SKIPPED", "skipped"),
        ]
        checks = ci.normalize_checks(payload, "pr")
        self.assertEqual([c["status"] for c in checks], ["pass", "fail", "skipped", "skipped"])
        self.assertEqual(checks[0]["run_id"], "123")

    def test_normalize_base_api_check_runs(self):
        payload = {
            "total_count": 2,
            "check_runs": [
                {
                    "name": "Trivy Gate High/Critical",
                    "status": "completed",
                    "conclusion": "failure",
                    "html_url": "https://github.com/o/r/actions/runs/999/job/2",
                    "app": {"name": "Security Scan"},
                },
                {
                    "name": "Queued",
                    "status": "queued",
                    "conclusion": None,
                    "html_url": "",
                    "app": {"name": "backend-ci"},
                },
            ],
        }
        checks = ci.normalize_checks(payload, "base")
        self.assertEqual([c["status"] for c in checks], ["fail", "pending"])
        self.assertEqual(checks[0]["workflow"], "Security Scan")
        self.assertEqual(checks[0]["run_id"], "999")

    def test_baseline_failure_is_marked_and_mapped(self):
        pr = ci.normalize_checks(
            [
                pr_check(
                    "Trivy Gate High/Critical",
                    "FAILURE",
                    "fail",
                    link="https://github.com/o/r/actions/runs/123/job/4",
                )
            ],
            "pr",
        )
        base = ci.normalize_checks(
            [
                {
                    "name": "Trivy Gate High/Critical",
                    "status": "completed",
                    "conclusion": "failure",
                    "html_url": "https://github.com/o/r/actions/runs/99/job/1",
                    "app": {"name": "Security Scan"},
                }
            ],
            "base",
        )
        result = ci.classify(pr, base, DEFAULT_RULES, BACKLOG_TEXT, "abcdef0")
        self.assertTrue(result["ci"][0]["baseline"])
        self.assertEqual(result["ci"][0]["status"], "fail")
        self.assertEqual(len(result["backlog_updates"]), 1)
        self.assertEqual(result["backlog_updates"][0]["id"], "AUDIT-071")
        self.assertIn("base_sha=abcdef0", result["backlog_updates"][0]["evidence"])
        self.assertEqual(result["unattributed"], [])

    def test_new_failure_without_rule_stays_unattributed(self):
        pr = ci.normalize_checks([pr_check("Mystery Gate", "FAILURE", "fail", workflow="ops")], "pr")
        result = ci.classify(pr, [], [], BACKLOG_TEXT)
        self.assertFalse(result["ci"][0]["baseline"])
        self.assertEqual(result["backlog_updates"], [])
        self.assertEqual(result["unattributed"][0]["reason"], "no matching attribution rule")

    def test_missing_backlog_id_is_not_created_implicitly(self):
        rules = [
            {
                "workflow": "Security Scan",
                "name_contains": "Trivy",
                "backlog_id": "AUDIT-999",
                "status": "suspected",
            }
        ]
        pr = ci.normalize_checks([pr_check("Trivy Gate", "FAILURE", "fail")], "pr")
        result = ci.classify(pr, [], rules, BACKLOG_TEXT)
        self.assertEqual(result["backlog_updates"], [])
        self.assertIn("does not exist", result["unattributed"][0]["reason"])

    def test_merge_preserves_manifest_and_drops_stale_snapshots(self):
        manifest = {
            "run_id": "cm-auto-20261127-test",
            "status": "partial",
            "ci": [{"name": "Old", "status": "pass"}],
            "backlog_snapshot_sha256": "0" * 64,
            "ledger_snapshot_sha256": "1" * 64,
        }
        attribution = {
            "ci": [{"name": "New", "status": "fail", "baseline": True}],
            "backlog_updates": [
                {
                    "id": "AUDIT-071",
                    "status": "suspected",
                    "note": "n",
                    "evidence": "e",
                }
            ],
        }
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory)
            (base / "manifest.json").write_text(json.dumps(manifest), encoding="utf-8")
            (base / "attribution.json").write_text(json.dumps(attribution), encoding="utf-8")
            result = subprocess.run(
                [
                    sys.executable,
                    str(MODULE_PATH),
                    "merge",
                    "--manifest",
                    str(base / "manifest.json"),
                    "--attribution",
                    str(base / "attribution.json"),
                    "--output",
                    str(base / "merged.json"),
                ],
                text=True,
                capture_output=True,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            merged = json.loads((base / "merged.json").read_text(encoding="utf-8"))

        self.assertEqual(merged["run_id"], "cm-auto-20261127-test")
        self.assertEqual(merged["status"], "partial")
        self.assertEqual(merged["ci"], attribution["ci"])
        self.assertEqual(merged["backlog_updates"], attribution["backlog_updates"])
        self.assertNotIn("backlog_snapshot_sha256", merged)
        self.assertNotIn("ledger_snapshot_sha256", merged)

    def test_multiple_failures_mapping_to_same_id_are_deduplicated(self):
        pr = ci.normalize_checks(
            [
                pr_check("Trivy Gate High/Critical", "FAILURE", "fail", link="link-a"),
                pr_check("Trivy Gate Config", "FAILURE", "fail", link="link-b"),
            ],
            "pr",
        )
        result = ci.classify(pr, [], DEFAULT_RULES, BACKLOG_TEXT)
        self.assertEqual(len(result["backlog_updates"]), 1)
        self.assertIn("link-a", result["backlog_updates"][0]["evidence"])
        self.assertIn("link-b", result["backlog_updates"][0]["evidence"])

    def test_cli_strict_unattributed_exit_code(self):
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory)
            (base / "pr.json").write_text(
                json.dumps([pr_check("Unknown", "FAILURE", "fail", workflow="ops")]),
                encoding="utf-8",
            )
            (base / "base.json").write_text(json.dumps({"check_runs": []}), encoding="utf-8")
            (base / "config.json").write_text(json.dumps({"rules": []}), encoding="utf-8")
            (base / "backlog.md").write_text(BACKLOG_TEXT, encoding="utf-8")
            command = [
                sys.executable,
                str(MODULE_PATH),
                "--config",
                str(base / "config.json"),
                "attribute",
                "--pr-checks",
                str(base / "pr.json"),
                "--base-checks",
                str(base / "base.json"),
                "--backlog",
                str(base / "backlog.md"),
            ]
            strict = subprocess.run(command + ["--strict-unattributed"], text=True, capture_output=True)
            non_strict = subprocess.run(command, text=True, capture_output=True)

        self.assertEqual(strict.returncode, 1)
        self.assertIn("no verified attribution", strict.stderr)
        self.assertEqual(non_strict.returncode, 0)
        self.assertEqual(json.loads(non_strict.stdout)["unattributed"][0]["name"], "Unknown")


if __name__ == "__main__":
    unittest.main()
