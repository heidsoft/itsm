#!/usr/bin/env python3
"""多会话共用一棵工作树时的会话账本与提交前检查。

子命令：
  enter   --id ID [--claim file ...]    记录进入时的 HEAD / 工作树 / 认领文件
  diff    --id ID                      对比进入快照与当前工作树，区分「我碰过」与「别人 WIP」
  check   --id ID [--commit]            提交前检查：认领冲突 / CHANGELOG 矛盾 / 敏感信息 / HEAD 漂移
  note    --id ID --commit <hash>       提交后写入账本与 HEAD 快照
  push-guard [--expect-head HASH]       push 前核对远端未被并行会话推进
  status                               列出所有会话账本与活跃认领

账本存放 .workbuddy/parallel/（已被 .gitignore 忽略），不入库。
设计原则：只观察、只记账；除账本文件外不碰任何源码。
"""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
LEDGER = ROOT / ".workbuddy" / "parallel"
SESSIONS = LEDGER / "sessions"

SENSITIVE_PATTERNS = [
    (re.compile(r"(?i)(adminprod|rlsadmin)"), "已知明文口令标识"),
    (re.compile(r"(?i)(password|passwd|secret|api[_-]?key|token|credential)\s*[:=]\s*[\"'][^\"']{6,}",), "疑似明文凭据"),
    (re.compile(r"-----BEGIN [A-Z ]*PRIVATE KEY-----"), "私钥块"),
]

# CHANGELOG 里「未触碰 X」这类声明会随并行会话推进而过期，提交时必须复核。
NEGATIVE_CLAIM = re.compile(r"(未触碰|不碰|没有改|未修改|不涉及|不改动)[^。；\n]{0,80}")


def git(*args: str, check: bool = False) -> str:
    r = subprocess.run(["git", *args], cwd=ROOT, text=True, capture_output=True)
    if check and r.returncode != 0:
        raise RuntimeError(f"git {' '.join(args)} 失败: {r.stderr.strip()}")
    return r.stdout


def now() -> str:
    return datetime.now(timezone.utc).astimezone().isoformat(timespec="seconds")


def session_path(session_id: str) -> Path:
    safe = re.sub(r"[^A-Za-z0-9._-]", "_", session_id)
    return SESSIONS / f"{safe}.json"


def load(session_id: str) -> dict:
    p = session_path(session_id)
    if not p.exists():
        sys.exit(f"[FATAL] 会话账本不存在：{p}\n请先运行 enter。")
    return json.loads(p.read_text(encoding="utf-8"))


def save(session_id: str, data: dict) -> Path:
    SESSIONS.mkdir(parents=True, exist_ok=True)
    p = session_path(session_id)
    p.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return p


def snapshot_worktree() -> dict:
    def lines(*args: str) -> list[str]:
        return [x for x in git(*args).splitlines() if x.strip()]

    return {
        "staged": lines("diff", "--cached", "--name-only"),
        "modified": lines("diff", "--name-only"),
        "untracked": lines("ls-files", "--others", "--exclude-standard"),
    }


def head_info() -> dict:
    return {
        "head": git("rev-parse", "HEAD", check=True).strip(),
        "subject": git("log", "-1", "--format=%s", check=True).strip(),
        "origin_main": git("rev-parse", "--verify", "origin/main").strip() or None,
    }


def file_hashes(paths: list[str]) -> dict[str, str]:
    out = {}
    for rel in paths:
        p = ROOT / rel
        if p.is_file():
            out[rel] = hashlib.sha256(p.read_bytes()).hexdigest()
    return out


def cmd_enter(args: argparse.Namespace) -> int:
    wt = snapshot_worktree()
    all_paths = sorted(set(wt["staged"]) | set(wt["modified"]) | set(wt["untracked"]))
    data = {
        "id": args.id,
        "entered_at": now(),
        "note": args.note or "",
        "claims": sorted(set(args.claim or [])),
        "baseline": head_info(),
        "worktree_at_enter": wt,
        "hashes_at_enter": file_hashes(all_paths),
        "commits": [],
    }
    p = save(args.id, data)
    total = len(all_paths)
    print(f"[enter] 会话 {args.id} 已记账 -> {p.relative_to(ROOT)}")
    print(f"  HEAD={data['baseline']['head'][:12]} {data['baseline']['subject']}")
    print(f"  origin/main={str(data['baseline']['origin_main'])[:12]}")
    print(f"  进入时工作树 WIP：{total} 个文件（暂存 {len(wt['staged'])} / 修改 {len(wt['modified'])} / 未跟踪 {len(wt['untracked'])}）")
    if data["claims"]:
        print(f"  认领文件：{', '.join(data['claims'])}")
    return 0


def cmd_diff(args: argparse.Namespace) -> int:
    d = load(args.id)
    wt = snapshot_worktree()
    at_enter = d["worktree_at_enter"]
    old_hashes = d["hashes_at_enter"]

    current = sorted(set(wt["staged"]) | set(wt["modified"]) | set(wt["untracked"]))
    before = sorted(set(at_enter["staged"]) | set(at_enter["modified"]) | set(at_enter["untracked"]))
    now_hashes = file_hashes(current)

    touched = [f for f in current if f not in before or now_hashes.get(f) != old_hashes.get(f)]
    untouched = [f for f in current if f not in touched]
    disappeared = [f for f in before if f not in current]

    print(f"[diff] 会话 {args.id}：进入后工作树变化")
    print(f"  本会话碰过 / 新增（{len(touched)}）：")
    for f in touched:
        print(f"    + {f}")
    print(f"  进入时已有且未变（{len(untouched)}，默认视为他人 WIP）：")
    for f in untouched:
        print(f"    = {f}")
    if disappeared:
        print(f"  进入后消失（{len(disappeared)}，可能是他人提交/回滚）：")
        for f in disappeared:
            print(f"    - {f}")
    return 0


def active_claims(except_id: str) -> dict[str, list[str]]:
    result: dict[str, list[str]] = {}
    SESSIONS.mkdir(parents=True, exist_ok=True)
    for p in SESSIONS.glob("*.json"):
        try:
            d = json.loads(p.read_text(encoding="utf-8"))
        except (json.JSONDecodeError, OSError):
            continue
        if d.get("id") == except_id or not d.get("claims"):
            continue
        for f in d["claims"]:
            result.setdefault(f, []).append(d["id"])
    return result


def staged_diff_text() -> str:
    return git("diff", "--cached", check=True)


def changed_paths_from_diff(diff: str) -> set[str]:
    paths = set()
    for line in diff.splitlines():
        if line.startswith("+++ b/"):
            paths.add(line[6:].strip())
        elif line.startswith("--- a/"):
            paths.add(line[6:].strip())
    return {p for p in paths if p and p != "/dev/null"}


def cmd_check(args: argparse.Namespace) -> int:
    d = load(args.id)
    problems: list[str] = []
    warns: list[str] = []

    staged = git("diff", "--cached", "--name-only").splitlines()
    staged = [x for x in staged if x.strip()]
    if not staged:
        problems.append("暂存区为空；请先显式 git add 本次文件（禁 git add -A）")

    diff = staged_diff_text() if staged else ""
    touched_files = set(staged)

    # 1. 认领冲突：共享文件若被其他会话认领，必须人工确认。
    claims = active_claims(args.id)
    conflicts = {f: ids for f, ids in claims.items() if f in touched_files}
    if conflicts:
        problems.append("认领冲突：" + "; ".join(f"{f} <- {','.join(ids)}" for f, ids in conflicts.items()))

    # 2. CHANGELOG 矛盾：本次同时改了代码，但 changelog 里写着「未触碰 X」。
    changelog = ROOT / "CHANGELOG.md"
    if "CHANGELOG.md" in touched_files and changelog.exists():
        staged_text = "\n".join(
            line[1:] for line in diff.splitlines()
            if line.startswith("+") and not line.startswith("+++")
        )
        for m in NEGATIVE_CLAIM.finditer(staged_text):
            segment = m.group(0)
            for token in re.findall(r"[A-Za-z0-9_./-]+\.(?:go|ts|tsx|py|json|md|yml|yaml)", segment):
                name = Path(token).name
                if any(Path(f).name == name for f in touched_files if f != "CHANGELOG.md"):
                    problems.append(f"CHANGELOG 声明「{segment[:60]}」与本次提交文件冲突：{name}")

    # 3. 敏感信息：扫暂存区新增行。
    hits = []
    for line in diff.splitlines():
        if not line.startswith("+") or line.startswith("+++"):
            continue
        for pat, label in SENSITIVE_PATTERNS:
            if pat.search(line):
                hits.append(f"{label}: {line[1:][:120]}")
    if hits:
        problems.append("暂存区敏感信息命中：" + " | ".join(hits[:5]))

    # 4. 进入时已有且本次要提交的文件：可能是别人的 WIP。
    at_enter = d["worktree_at_enter"]
    prior = set(at_enter["staged"]) | set(at_enter["modified"]) | set(at_enter["untracked"])
    prior_now = [f for f in sorted(touched_files & prior) if f != "CHANGELOG.md"]
    if prior_now:
        warns.append("以下文件进入时已在 WIP 中，确认是否属于本次任务：" + ", ".join(prior_now))

    # 5. HEAD / origin 漂移。
    cur = head_info()
    if cur["head"] != d["baseline"]["head"]:
        warns.append(f"HEAD 已从 {d['baseline']['head'][:12]} 变为 {cur['head'][:12]}；提交后 note 会更新账本")
    if d["baseline"]["origin_main"] and cur["origin_main"] != d["baseline"]["origin_main"]:
        warns.append(
            f"origin/main 已从 {d['baseline']['origin_main'][:12]} 变为 {str(cur['origin_main'])[:12]}；"
            "push 前必须 pull --rebase 并核对语义"
        )

    # 6. 提交原子性提示：跨多个顶层目录/主题时提醒拆分。
    topics = set()
    for f in touched_files:
        parts = Path(f).parts
        topics.add("/".join(parts[:3]) if len(parts) > 2 else f)
    if len(touched_files) >= 20 and len(topics) >= 4:
        warns.append(f"本次暂存 {len(touched_files)} 文件、跨 {len(topics)} 个主题目录；建议按主题拆分为多个 commit")

    print(f"[check] 会话 {args.id}：{len(staged)} 个文件在暂存区")
    for w in warns:
        print(f"  WARN  {w}")
    for p in problems:
        print(f"  FAIL  {p}")
    if args.commit and not problems:
        print("  提交前检查 PASS；可执行 git commit（hook 不得跳过）")
    print(f"[check] 结论：{'FAIL(' + str(len(problems)) + ')' if problems else 'PASS'}")
    return 1 if problems else 0


def cmd_note(args: argparse.Namespace) -> int:
    d = load(args.id)
    commit = git("rev-parse", "--verify", args.commit, check=True).strip()
    d.setdefault("commits", []).append({
        "commit": commit,
        "noted_at": now(),
        "head_after": head_info(),
        "files": git("show", "--format=", "--name-only", commit).splitlines(),
    })
    p = save(args.id, d)
    print(f"[note] 会话 {args.id} 记录提交 {commit[:12]} -> {p.relative_to(ROOT)}")
    return 0


def cmd_push_guard(args: argparse.Namespace) -> int:
    cur = head_info()
    expect = args.expect_head
    if not expect:
        sessions = sorted(SESSIONS.glob("*.json")) if SESSIONS.exists() else []
        if sessions:
            newest = max(sessions, key=lambda p: p.stat().st_mtime)
            d = json.loads(newest.read_text(encoding="utf-8"))
            expect = d.get("baseline", {}).get("head")
    print(f"[push-guard] 本地 HEAD={cur['head'][:12]}  origin/main={str(cur['origin_main'])[:12]}")
    if expect and not cur["head"].startswith(expect) and not expect.startswith(cur["head"]):
        print(f"  WARN  会话进入时 HEAD={expect[:12]}，期间 HEAD 已移动；确认本地提交栈仍正确")
    ahead = git("rev-list", "--count", "origin/main..HEAD", check=True).strip()
    behind = git("rev-list", "--count", "HEAD..origin/main", check=True).strip()
    if behind != "0":
        print(f"  FAIL  origin/main 领先本地 {behind} 个提交（并行会话已推送）；先 pull --rebase 再 push")
        return 1
    print(f"  OK    本地领先 {ahead} 个提交，可 push")
    return 0


def cmd_status(args: argparse.Namespace) -> int:
    SESSIONS.mkdir(parents=True, exist_ok=True)
    files = sorted(SESSIONS.glob("*.json"))
    if not files:
        print("[status] 暂无会话账本")
        return 0
    print(f"[status] {len(files)} 个会话账本：")
    for p in files:
        d = json.loads(p.read_text(encoding="utf-8"))
        n = len(d.get("commits", []))
        print(f"  {d.get('id', p.stem):<28} enter={d.get('entered_at', '?')} "
              f"head={str(d.get('baseline', {}).get('head', '?'))[:12]} "
              f"claims={len(d.get('claims', []))} commits={n}")
    claims = active_claims("")
    if claims:
        print("  活跃认领：")
        for f, ids in sorted(claims.items()):
            print(f"    {f} <- {', '.join(ids)}")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="cmd", required=True)

    p = sub.add_parser("enter", help="记录会话进入基线")
    p.add_argument("--id", required=True, help="会话标识，如 2026-11-20-backend-audit")
    p.add_argument("--claim", action="append", default=[], help="认领文件（可重复）")
    p.add_argument("--note", help="本次任务说明")
    p.set_defaults(fn=cmd_enter)

    p = sub.add_parser("diff", help="对比进入快照与当前工作树")
    p.add_argument("--id", required=True)
    p.set_defaults(fn=cmd_diff)

    p = sub.add_parser("check", help="提交前检查")
    p.add_argument("--id", required=True)
    p.add_argument("--commit", action="store_true", help="确认这是提交前检查，通过时打印提示")
    p.set_defaults(fn=cmd_check)

    p = sub.add_parser("note", help="提交后记账")
    p.add_argument("--id", required=True)
    p.add_argument("--commit", required=True, help="提交 hash 或可解析的引用")
    p.set_defaults(fn=cmd_note)

    p = sub.add_parser("push-guard", help="push 前核对远端漂移")
    p.add_argument("--expect-head", help="期望的本地 HEAD（默认取最新会话账本）")
    p.set_defaults(fn=cmd_push_guard)

    p = sub.add_parser("status", help="列出会话账本")
    p.set_defaults(fn=cmd_status)

    args = parser.parse_args()
    return args.fn(args)


if __name__ == "__main__":
    sys.exit(main())
