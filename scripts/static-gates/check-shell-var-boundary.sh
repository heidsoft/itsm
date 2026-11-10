#!/usr/bin/env bash
#
# scripts/static-gates/check-shell-var-boundary.sh
#
# Stage 5.14 - 禁止 shell 里「未加花括号的 $VAR 紧跟非 ASCII 字符」。
#
# 根因（2026-11-10 本机 prod v1.6.50 重建验收实测）：
#   bash 允许变量名包含高位字节（非 ASCII）。在 set -u 下
#       fail "itsm-cli flow run $FLOW_SCENARIO（场景失败需人工判 failed/blocked）"
#   实际展开的是名为 `FLOW_SCENARIO（` 的变量，直接报
#       scripts/prod-post-deploy-verify.sh: line 136: FLOW_SCENARIO?: unbound variable
#   脚本崩在**失败分支**，真实失败原因被吞掉，--full 验收剧本 EXIT=5，
#   发布证据不可用（当时只能靠 /tmp 里的临时修补副本跑完验收）。
#   最小复现：
#       bash -c 'set -u; X=a; echo "$X（x）"'    -> EXIT=127
#       bash -c 'set -u; X=a; echo "${X}（x）"'  -> EXIT=0
#   不止全角标点：`echo "$X条"` 同样 unbound。任何紧邻的非 ASCII 字符都会被
#   并进变量名。修法统一用 ${VAR} 显式给出边界，输出文案零变化。
#
# 2026-11-10 一次性清理 18 处（5 个脚本），本守卫防回流。
# 守卫自身带自检：违规样本必须被命中、合规样本必须零命中、扫描文件数必须 >0，
# 任一不满足即 exit 2（守卫不可用视为失败，不静默放行）。
#
# 用法：bash scripts/static-gates/check-shell-var-boundary.sh
# 退出码：0 通过 / 1 发现命中 / 2 守卫自身不可用
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
  echo "ERROR: python3 not found; shell-var-boundary guard cannot run" >&2
  exit 2
fi

"${PYTHON_BIN}" - <<'PYEOF'
import pathlib
import re
import sys

ROOT = pathlib.Path.cwd()
SKIP_DIRS = {'.git', 'node_modules', '.next', 'dist', 'build', 'vendor', '.venv', '__pycache__'}

# Unbraced $VAR followed immediately by any non-ASCII byte. bash treats those
# bytes as part of the variable name, so `${VAR}` (or an ASCII separator) is required.
PATTERN = re.compile(r'(?<![$\{])\$([A-Za-z_][A-Za-z0-9_]*)(?=[^\x00-\x7f])')


def hits_in(line):
    # Pure comment lines never expand variables. (Shebang is covered too.)
    if line.lstrip().startswith('#'):
        return []
    return [m.group(0) for m in PATTERN.finditer(line)]


# --- guard self-check: a broken scanner must fail loudly, never pass silently ---
# Samples are assembled from fragments: this file is itself among the scanned
# *.sh files, so it must not contain the forbidden literal. First run of this
# guard did flag its own two sample lines - proof that the scan is real.
BAD_FRAGMENT = '$' + 'FLOW_SCENARIO' + '（场景失败）'
POSITIVE = '  fail "flow run ' + BAD_FRAGMENT + '"'
NEGATIVE_BRACED = '  fail "flow run ${' + 'FLOW_SCENARIO' + '}（场景失败）"'
NEGATIVE_ASCII = '  fail "flow run ' + '$' + 'FLOW_SCENARIO (failed)"'
NEGATIVE_COMMENT = '#  fail "flow run ' + BAD_FRAGMENT + '"'
if (len(hits_in(POSITIVE)) != 1 or hits_in(NEGATIVE_BRACED)
        or hits_in(NEGATIVE_ASCII) or hits_in(NEGATIVE_COMMENT)):
    print('ERROR: shell-var-boundary scanner self-check failed', file=sys.stderr)
    sys.exit(2)

files = sorted(
    p for p in ROOT.rglob('*.sh')
    if not any(part in SKIP_DIRS for part in p.parts)
)
if not files:
    print('ERROR: no shell files scanned; refusing to pass vacuously', file=sys.stderr)
    sys.exit(2)

hits = []
for path in files:
    try:
        lines = path.read_text(encoding='utf-8').splitlines()
    except UnicodeDecodeError:
        print(f'ERROR: cannot decode {path} as UTF-8', file=sys.stderr)
        sys.exit(2)
    for lineno, line in enumerate(lines, 1):
        for token in hits_in(line):
            hits.append((path.relative_to(ROOT), lineno, token, line.strip()[:160]))

if hits:
    print(f'FAIL: shell variable boundary - {len(hits)} hit(s) in {len(files)} .sh file(s)')
    for path, lineno, token, line in hits:
        print(f'  {path}:{lineno}: {token} | {line}')
    print('')
    print('bash treats the trailing non-ASCII byte as part of the variable name;')
    print('under set -u this aborts with "unbound variable" (usually on an error branch,')
    print('hiding the real failure). Fix: use ${VAR} before the CJK text/punctuation,')
    print('or insert an ASCII separator. Output text must stay unchanged.')
    sys.exit(1)

print(f'PASS: shell variable boundary - {len(files)} .sh files scanned, 0 hits')
PYEOF
