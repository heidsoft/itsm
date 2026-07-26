#!/usr/bin/env python3
"""Iteratively fix swagger comments that reference undefined types."""
from __future__ import annotations

import os
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path("/Users/heidsoft/Downloads/research/itsm/itsm-backend")
CONTROLLERS = ROOT / "controller"
DTO = ROOT / "dto"
HANDLERS = ROOT / "handlers"

SWAG_CMD = [
    "go",
    "run",
    "github.com/swaggo/swag/cmd/swag@v1.16.6",
    "init",
    "--parseDependency",
    "--parseInternal",
    "-g",
    "main.go",
    "-o",
    "docs",
]


def collect_defined_types() -> set:
    types = set()
    pattern = re.compile(r"^type\s+([A-Z]\w*)\s+")
    for d in (DTO, HANDLERS):
        if not d.exists():
            continue
        for f in d.rglob("*.go"):
            text = f.read_text(encoding="utf-8", errors="ignore")
            for match in pattern.finditer(text):
                types.add(match.group(1))
    return types


OBJ_RE = re.compile(
    r"@Success\s+\d+\s+\{object\}\s+((?:dto|common|handlers)\.)?(\w+)"
)
DATA_RE = re.compile(
    r"data=(\[\])?((?:dto|common|handlers)\.)?(\w+)"
)
PARSE_ERR_RE = re.compile(
    r"cannot find type definition:\s+((?:dto|common|handlers)\.)?(\w+)"
)


def fix_missing_types(missing):
    if not missing:
        return 0
    files_changed = 0
    # Also include handlers directory for comments.
    search_dirs = [CONTROLLERS]
    if HANDLERS.exists():
        search_dirs.append(HANDLERS)
    for search_root in search_dirs:
        for f in search_root.rglob("*.go"):
            text = f.read_text(encoding="utf-8", errors="ignore")
            original = text
            for m in OBJ_RE.finditer(text):
                pkg = m.group(1) or ""
                t = m.group(2)
                if t in missing:
                    old = "@Success 200 {object} " + pkg + t
                    new = "@Success 200 {object} object"
                    text = text.replace(old, new)
                    text = text.replace(
                        "@Success 201 {object} " + pkg + t,
                        "@Success 201 {object} object",
                    )
            for m in DATA_RE.finditer(text):
                array_prefix = m.group(1) or ""
                pkg = m.group(2) or ""
                t = m.group(3)
                if t in missing:
                    old = "data=" + array_prefix + pkg + t
                    new = "data=" + array_prefix + "object"
                    text = text.replace(old, new)
            if text != original:
                f.write_text(text, encoding="utf-8")
                files_changed += 1
    return files_changed


def run_swag():
    result = subprocess.run(
        SWAG_CMD,
        cwd=str(ROOT),
        capture_output=True,
        text=True,
    )
    output = result.stdout + result.stderr
    return result.returncode, output


def main():
    print("Defined types:", len(collect_defined_types()))
    for iteration in range(1, 200):
        rc, output = run_swag()
        missing = set()
        for m in PARSE_ERR_RE.finditer(output):
            missing.add(m.group(2))
        if not missing:
            print("swag init succeeded after iteration " + str(iteration - 1))
            return 0
        print("[iter " + str(iteration) + "] rc=" + str(rc) + ", missing=" + str(sorted(missing)))
        changed = fix_missing_types(missing)
        print("  files changed: " + str(changed))
        if changed == 0:
            print("No more files to change but errors remain.")
            for line in output.splitlines():
                if "cannot find type" in line or "ParseComment" in line:
                    print("  " + line)
            return 1
    print("Exceeded max iterations")
    return 1


if __name__ == "__main__":
    sys.exit(main())