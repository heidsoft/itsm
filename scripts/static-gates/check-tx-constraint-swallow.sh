#!/usr/bin/env bash
#
# scripts/static-gates/check-tx-constraint-swallow.sh
#
# Stage 5.7 — 禁止在事务内吞掉约束错误后继续执行。
#
# 背景（为什么必须有机器守卫）：
#   PostgreSQL 遇到唯一约束冲突会把**整条事务**标记为 aborted(SQLSTATE 25P02)，
#   此后该事务内的任何语句（包括 COMMIT）都会失败。因此形如
#
#       if ent.IsConstraintError(err) { continue }        // 错
#       if err != nil && !ent.IsConstraintError(err) { }  // 错
#
#   的写法是错的：它把"事务已废"当成"幂等命中"，让同事务内的业务写入连坐回滚。
#   internal/commandbus/commandbus.go 的注释已完整记录该语义，并提供了
#   EnqueueTxIdempotent（事务内先查后插）。但知识停在注释里就会漂移——
#   2026-10-13 的实测命中 4 处（含工单创建主链路 CreateWithTx），本守卫即为堵这个口。
#
# 正确的两种写法：
#   A. 先回滚再吞（幂等语义）：
#          if rbErr := tx.Rollback(); rbErr != nil { ... }
#          if ent.IsConstraintError(err) { return nil }
#      —— 见 service/sla_monitor_service.go、handlers/email_intake/orchestrator.go
#   B. 直接上抛，让调用方回滚：
#          if err != nil { return err }
#      —— 事务内入箱一律改用 commandbus.EnqueueTxIdempotent 规避冲突本身
#
# 用法：
#   ./scripts/static-gates/check-tx-constraint-swallow.sh
# 退出码：
#   0  通过（无命中）
#   1  发现疑似命中（需人工确认）
#

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT_DIR}"

PYTHON_BIN="${PYTHON_BIN:-}"
if [ -z "${PYTHON_BIN}" ]; then
  for candidate in \
    "/Users/heidsoft/.workbuddy/binaries/python/versions/3.13.12/bin/python3" \
    "$(command -v python3 || true)"; do
    if [ -n "${candidate}" ] && [ -x "${candidate}" ]; then
      PYTHON_BIN="${candidate}"
      break
    fi
  done
fi

if [ -z "${PYTHON_BIN}" ]; then
  echo "未找到 python3，跳过检查"
  exit 0
fi

"${PYTHON_BIN}" - <<'PYEOF'
import os
import re
import sys

SCAN_DIRS = [
    "itsm-backend/handlers",
    "itsm-backend/service",
    "itsm-backend/repository",
]

IF_START = re.compile(r"\bif\b")
IS_CONSTRAINT = re.compile(r"IsConstraintError\s*\(")
ROLLBACK = re.compile(r"Rollback\s*\(")
# 吞错：continue，或返回全空值（错误信息被丢掉）
SWALLOW_CONTINUE = re.compile(r"^\s*continue\s*$")
SWALLOW_RETURN = re.compile(r"^\s*return\s+(nil|0|true|false|0\.0)(\s*,\s*(nil|0|true|false))?\s*$")
# 传播错误：return err / return fmt.Errorf / return ..., err
PROPAGATE = re.compile(r"\berr\b|Errorf|Wrapf")

TX_CONTEXT = re.compile(r"\.Tx\s*\(\s*ctx\s*\)|tx\s+\*ent\.Tx|Tx\s+\*ent\.Tx")


def find_func_start(lines, idx):
    """从 idx 向上找最近的顶层 func 声明行号。"""
    depth = 0
    for i in range(idx, -1, -1):
        line = lines[i]
        depth += line.count("}") - line.count("{")
        if depth > 0:
            continue
        if re.match(r"^func\s", line):
            return i
    return 0


def extract_if_block(lines, idx):
    """从 if 语句起始行提取 then 块（else 之前的完整花括号内容）。"""
    text = ""
    i = idx
    brace_depth = 0
    started = False
    while i < len(lines):
        line = lines[i]
        for ch_i, ch in enumerate(line):
            if ch == "{":
                if not started:
                    started = True
                brace_depth += 1
            elif ch == "}":
                brace_depth -= 1
                if started and brace_depth == 0:
                    # 返回 then 块主体（去掉首尾花括号）
                    body = text
                    return body, line
            if started:
                text += ch
        text += "\n"
        i += 1
        if i - idx > 120:
            break
    return None, None


def then_block(body):
    """从已累积的含花括号文本中取出最外层 then 块。"""
    depth = 0
    out = []
    for ch in body:
        if ch == "{":
            depth += 1
            if depth == 1:
                continue
        elif ch == "}":
            depth -= 1
            if depth == 0:
                break
        if depth >= 1:
            out.append(ch)
    return "".join(out)


def strip_else(then):
    """截断 else / else if 部分，只保留 then 主体。"""
    depth = 0
    for i, ch in enumerate(then):
        if ch in "([{":
            depth += 1
        elif ch in ")]}":
            depth -= 1
        elif depth == 0 and then[i:i + 4] == "else":
            return then[:i]
    return then


def check_file(path):
    findings = []
    try:
        with open(path, encoding="utf-8") as fh:
            lines = fh.read().split("\n")
    except (OSError, UnicodeDecodeError):
        return findings

    for idx, line in enumerate(lines):
        if not IS_CONSTRAINT.search(line):
            continue
        # 只关心 if 条件里用到 IsConstraintError 的行
        if not IF_START.search(line):
            continue
        body, _ = extract_if_block(lines, idx)
        if body is None:
            continue
        then = strip_else(then_block(body))
        if not then.strip():
            continue
        # A 模式：先回滚再吞 —— 正确
        if ROLLBACK.search(then):
            continue
        # A 模式变体：rollback 写在外层 if 块内、本 if 之前若干行，例如
        #     if err != nil {
        #         if rbErr := tx.Rollback(); rbErr != nil { ... }
        #         if ent.IsConstraintError(err) { return nil }
        #     }
        # 向前回溯同一语句块，出现 Rollback 即视为已回滚。
        prelude = "\n".join(lines[max(0, idx - 12):idx])
        if ROLLBACK.search(prelude):
            continue
        # B 模式：向上传播 —— 正确
        if PROPAGATE.search(then):
            continue
        # 吞错判定
        for tline in then.split("\n"):
            if SWALLOW_CONTINUE.match(tline) or SWALLOW_RETURN.match(tline):
                findings.append((path, idx + 1, line.strip()))
                break
    return findings


all_findings = []
for root in SCAN_DIRS:
    if not os.path.isdir(root):
        continue
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in ("__pycache__", "node_modules")]
        for name in sorted(filenames):
            if not name.endswith(".go") or name.endswith("_test.go"):
                continue
            all_findings.extend(check_file(os.path.join(dirpath, name)))

if not all_findings:
    print("PASS: 未发现事务内吞掉约束错误的写法")
    sys.exit(0)

print("FAIL: 发现 %d 处疑似「事务内吞掉约束错误」" % len(all_findings))
print("")
print("约束冲突会把整条事务标记为 aborted(25P02)，事务内重试或跳过都必然失败，")
print("并会让同事务内的业务写入连坐回滚。正确写法二选一：")
print("  A. 先 tx.Rollback() 再按幂等处理")
print("  B. 直接 return err 让调用方回滚（入箱一律改用 commandbus.EnqueueTxIdempotent）")
print("")
for path, lineno, snippet in all_findings:
    print("  %s:%d" % (path, lineno))
    print("      %s" % snippet[:140])
sys.exit(1)
PYEOF
