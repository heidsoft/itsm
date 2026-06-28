#!/usr/bin/env python3
"""verify-codeowners.py
验证 .github/CODEOWNERS 文件的正确性和覆盖率。

Usage:
    ./scripts/verify-codeowners.py                 # 默认（格式 + 覆盖率审计）
    ./scripts/verify-codeowners.py --check         # CI 模式（发现问题非零退出）
    ./scripts/verify-codeowners.py --audit         # 仅审计模式
    ./scripts/verify-codeowners.py --verify-owners # 调用 GitHub API 校验 owner
    ./scripts/verify-codeowners.py --verbose       # 详细输出

校验项：
    1. 格式：每行必须是 "pattern owner1 [owner2 ...]"、以 # 开头或空行
    2. 路径：相对仓库根的 pattern 必须合法（不包含 ..）
    3. 覆盖率：关键目录（itsm-backend/itsm-frontend/.github/scripts/docs）必须有 owner
    4. owner：必须是 @username 或 @org/team 格式

环境变量:
    GH_TOKEN           用于 --verify-owners（可选）
    GITHUB_REPOSITORY  目标仓库 owner/repo（默认从 git remote 推断）
"""
from __future__ import annotations

import argparse
import fnmatch
import json
import os
import re
import subprocess
import sys
import urllib.error
import urllib.request
from dataclasses import dataclass, field
from pathlib import Path
from typing import Iterable

# ---- 常量 -----------------------------------------------------------------

REPO_ROOT = Path(subprocess.check_output(
    ["git", "rev-parse", "--show-toplevel"], text=True
).strip()) if subprocess.run(["git", "rev-parse", "--show-toplevel"],
                             capture_output=True).returncode == 0 \
    else Path.cwd()

CODEOWNERS = REPO_ROOT / ".github" / "CODEOWNERS"

# 必须被 CODEOWNERS 显式或隐式覆盖
CRITICAL_DIRS = [
    "itsm-backend",
    "itsm-frontend",
    ".github/workflows",
    "scripts",
    "docs",
]

OWNER_PATTERN = re.compile(r"^@[A-Za-z0-9._-]+(/[A-Za-z0-9._-]+)?$")

# ---- 输出辅助 -------------------------------------------------------------


class Color:
    """ANSI 颜色代码，仅在 TTY 下启用。"""
    RED = "\033[0;31m"
    YELLOW = "\033[0;33m"
    GREEN = "\033[0;32m"
    BLUE = "\033[0;34m"
    RESET = "\033[0m"

    ENABLED = sys.stdout.isatty()

    @classmethod
    def wrap(cls, text: str, color: str) -> str:
        if cls.ENABLED:
            return f"{color}{text}{cls.RESET}"
        return text


def info(msg: str) -> None:
    print(Color.wrap(f"ℹ {msg}", Color.BLUE))


def ok(msg: str) -> None:
    print(Color.wrap(f"✓ {msg}", Color.GREEN))


def warn(msg: str) -> None:
    print(Color.wrap(f"⚠ {msg}", Color.YELLOW), file=sys.stderr)


def err(msg: str) -> None:
    print(Color.wrap(f"✗ {msg}", Color.RED), file=sys.stderr)


def verbose(msg: str, enabled: bool) -> None:
    if enabled:
        print(f"  · {msg}")


# ---- 数据模型 -------------------------------------------------------------


@dataclass
class CodeownerEntry:
    """CODEOWNERS 单条规则。"""
    line_no: int
    pattern: str
    owners: list[str]


@dataclass
class ValidationResult:
    """验证结果汇总。"""
    errors: int = 0
    warnings: int = 0
    entries: list[CodeownerEntry] = field(default_factory=list)
    unique_owners: set[str] = field(default_factory=set)

    def add_error(self, n: int = 1) -> None:
        self.errors += n

    def add_warning(self, n: int = 1) -> None:
        self.warnings += n


# ---- 阶段 1: 格式解析与校验 ------------------------------------------------


def parse_codeowners(path: Path, result: ValidationResult,
                     verbose_flag: bool) -> None:
    """读取 CODEOWNERS，校验每行格式，返回解析后的条目列表。"""
    info(f"校验文件: {path.relative_to(REPO_ROOT)}")

    with path.open("r", encoding="utf-8") as f:
        for lineno, raw_line in enumerate(f, start=1):
            # 去除行尾回车与换行
            line = raw_line.rstrip("\r\n").strip()

            # 空行或注释行
            if not line or line.startswith("#"):
                continue

            # 拆分 pattern + owners（以首个空白为分隔）
            parts = line.split(maxsplit=1)
            if len(parts) < 2:
                err(f"L{lineno}: 格式错误 - 缺少 owner: {line!r}")
                result.add_error()
                continue
            pattern, owners_str = parts

            # 解析 owner 列表
            owners = [o for o in owners_str.split() if o]
            if not owners:
                err(f"L{lineno}: 至少需要一个 owner: {line!r}")
                result.add_error()
                continue

            # 校验每个 owner 格式
            for owner in owners:
                if not OWNER_PATTERN.match(owner):
                    err(f"L{lineno}: owner 格式无效 '{owner}' "
                        f"（应类似 @user 或 @org/team）")
                    result.add_error()

            # 校验 pattern 不允许 ..
            if ".." in pattern:
                err(f"L{lineno}: pattern 不允许包含 '..': {pattern}")
                result.add_error()

            # 路径存在性（绝对路径模式）
            if pattern.startswith("/"):
                rel = pattern.lstrip("/")
                if (REPO_ROOT / rel).exists():
                    verbose(f"L{lineno}: 路径 '{rel}' 存在", verbose_flag)
                else:
                    warn(f"L{lineno}: 路径 '{rel}' 不存在（可能是预留规则）")
                    result.add_warning()

            entry = CodeownerEntry(line_no=lineno, pattern=pattern,
                                   owners=owners)
            result.entries.append(entry)
            result.unique_owners.update(owners)

    info(f"=== 解析完成 ===")
    info(f"规则数: {len(result.entries)}")
    info(f"唯一 owner 数: {len(result.unique_owners)}")


# ---- 阶段 2: 路径匹配 ------------------------------------------------------


def match_pattern(pattern: str, target: str) -> bool:
    """简化版 GitHub CODEOWNERS 路径匹配。

    规则：
      - "*"                       匹配所有文件
      - "/path/"                  target 以 path/ 开头
      - "/path" 或 "/path/xxx"    target 以 path 开头
      - "*.ext"                   通配符匹配（仓库内任意位置）
    """
    if pattern == "*":
        return True

    p = pattern.lstrip("/")
    t = target

    # 通配符
    if "*" in p:
        # 把 pattern 当作 shell glob
        # 限制：fnmatch 不支持 **，此处仅支持简单的 * 和 ?
        # 若 pattern 不包含 /，fnmatch 会匹配任意位置；包含 / 时按整段匹配
        return fnmatch.fnmatch(t, p)

    # 目录匹配
    if pattern.endswith("/"):
        return t.startswith(p)

    # 精确 / 前缀匹配
    return t.startswith(p)


def find_owners(target: str, entries: Iterable[CodeownerEntry]) -> list[str]:
    """返回匹配 target 的所有 owner（去重）。"""
    seen = set()
    matched = []
    for entry in entries:
        if match_pattern(entry.pattern, target):
            for o in entry.owners:
                if o not in seen:
                    seen.add(o)
                    matched.append(o)
    return matched


# ---- 阶段 3: 覆盖率审计 ----------------------------------------------------


def audit_coverage(result: ValidationResult, verbose_flag: bool) -> None:
    """检查关键目录是否都有匹配的 owner。"""
    info("覆盖率审计（关键目录）：")
    print()
    print(f"  {'目录':<30} Owner(s)")
    print(f"  {'-' * 30} {'-' * 40}")

    for d in CRITICAL_DIRS:
        # 用一个代表性路径查询（<dir>/<first child>）
        target = d
        full_dir = REPO_ROOT / d
        if full_dir.is_dir():
            children = [p.name for p in full_dir.iterdir() if not p.name.startswith(".")]
            if children:
                target = f"{d}/{children[0]}"

        owners = find_owners(target, result.entries)
        if not owners:
            print(Color.wrap(f"  {d:<30} ❌ 未匹配任何 owner", Color.RED))
            result.add_error()
        else:
            line = f"  {d:<30} {', '.join(owners)}"
            print(Color.wrap(line, Color.GREEN))
    print()


# ---- 阶段 4: GitHub owner 校验 ---------------------------------------------


def verify_owners_on_github(result: ValidationResult,
                            token: str | None,
                            repo: str | None,
                            verbose_flag: bool) -> int:
    """调用 GitHub API 验证每个 owner 是否存在。返回错误数。"""
    info("=== GitHub owner 校验 ===")

    if not repo:
        # 从 git remote 推断
        try:
            remote = subprocess.check_output(
                ["git", "config", "--get", "remote.origin.url"],
                cwd=REPO_ROOT, text=True).strip()
        except subprocess.CalledProcessError:
            warn("无法推断 GITHUB_REPOSITORY，跳过 --verify-owners")
            return 0
        # 兼容 SSH/HTTPS 格式
        m = re.search(r"github\.com[/:]([^/]+/[^/.]+?)(\.git)?$", remote)
        repo = m.group(1) if m else None

    info(f"GitHub 仓库: {repo or '<未推断>'}")

    api_errors = 0
    headers = {"Accept": "application/vnd.github+json",
               "X-GitHub-Api-Version": "2022-11-28"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    else:
        warn("GH_TOKEN 未设置，使用匿名 API（速率限制 60/h）")

    import time

    for owner in sorted(result.unique_owners):
        # 构造 API URL
        if "/" in owner:  # @org/team
            org, team = owner.lstrip("@").split("/", 1)
            api_url = f"https://api.github.com/orgs/{org}/teams/{team}"
        else:
            api_url = f"https://api.github.com/users/{owner.lstrip('@')}"

        try:
            req = urllib.request.Request(api_url, headers=headers)
            with urllib.request.urlopen(req, timeout=10) as resp:
                code = resp.status
        except urllib.error.HTTPError as e:
            code = e.code
        except Exception as e:
            err(f"{owner} 网络错误: {e}")
            api_errors += 1
            continue

        if code in (200, 201):
            ok(f"{owner} ({api_url})")
        elif code == 404:
            err(f"{owner} 不存在 (404): {api_url}")
            api_errors += 1
        elif code == 403:
            warn(f"{owner} 权限不足或速率限制 (403): {api_url}")
            time.sleep(1)
        else:
            err(f"{owner} HTTP {code}: {api_url}")
            api_errors += 1

    return api_errors


# ---- 主流程 ----------------------------------------------------------------


def main() -> int:
    parser = argparse.ArgumentParser(
        description="验证 .github/CODEOWNERS 文件的正确性和覆盖率")
    parser.add_argument("--check", action="store_true",
                        help="CI 模式（发现问题非零退出）")
    parser.add_argument("--audit", action="store_true",
                        help="仅覆盖率审计")
    parser.add_argument("--verify-owners", action="store_true",
                        help="调用 GitHub API 校验 owner")
    parser.add_argument("--verbose", "-v", action="store_true",
                        help="详细输出")
    parser.add_argument("--quiet", action="store_true",
                        help="只输出错误")
    args = parser.parse_args()

    if args.quiet:
        # 抑制 info 输出
        global info
        info = lambda *a, **k: None  # noqa: E731

    if not CODEOWNERS.exists():
        err(f"找不到 CODEOWNERS 文件: {CODEOWNERS}")
        return 1

    info(f"运行模式: {'check' if args.check else 'audit' if args.audit else 'local'}")

    result = ValidationResult()
    parse_codeowners(CODEOWNERS, result, args.verbose)

    print()
    if args.verify_owners:
        api_errors = verify_owners_on_github(
            result,
            token=os.environ.get("GH_TOKEN"),
            repo=os.environ.get("GITHUB_REPOSITORY"),
            verbose_flag=args.verbose,
        )
        result.add_error(api_errors)

    if not args.verify_owners:
        # 默认 + --check + --audit 都做覆盖率审计
        audit_coverage(result, args.verbose)

    print()
    if result.errors > 0:
        err(f"校验失败：{result.errors} 个错误, {result.warnings} 个警告")
        return 1

    if result.warnings > 0:
        warn(f"校验通过但有警告：{result.warnings} 个")
    else:
        ok("校验全部通过 ✅")

    return 0


if __name__ == "__main__":
    sys.exit(main())