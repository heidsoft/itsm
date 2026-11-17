#!/usr/bin/env bash
# 本地复现 GitHub Actions 上出现过的红灯，避免「本地绿、远端红」。
#
# 背景（2026-11-14，来自 heidsoft/itsm 远端 Actions 日志）：
#   push 5806232b 后 backend-ci / Lint、api-contract-check / Swagger Docs
#   Freshness、frontend-ci / Unit tests 同时变红，而本地发布验收是绿的。
#   根因分别是 gofumpt 格式欠账、swagger 生成产物未提交、两个过期断言。
#
# 检查项：
#   1. gofumpt 版本必须与 backend-ci.yml 的 pin 一致（版本漂移会制造假绿/假红）
#   2. itsm-backend 下未格式化文件清单
#   3. OpenAPI 生成产物与已提交 docs 是否一致（与 CI 同一条 swag 命令）
#
# 注意：本工具检查的是「当前工作树源码」，包含未提交的 WIP；CI 检查的是
# 「提交态源码」。若本地有未提交且改过注解的 WIP，第 3 项出现差异属预期，
# 提交前请以纯 HEAD 隔离树复核一次。
#
# Usage:
#   bash scripts/check-ci-parity.sh            # 报告模式（默认，不阻断）
#   bash scripts/check-ci-parity.sh --strict  # 任一差异即 exit 1（适合 CI / pre-push）
#   make ci-parity / make ci-parity-strict

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT" || exit 2

MODE="report"
[ "${1:-}" = "--strict" ] && MODE="strict"

GOFUMPT_VERSION="v0.7.0"   # 必须与 .github/workflows/backend-ci.yml 的 Install gofumpt 步骤一致
CI_PIN="$(grep -oE 'go install mvdan\.cc/gofumpt@[v0-9.]+' .github/workflows/backend-ci.yml | head -1 | sed 's/.*@//')"

fail=0
note() { printf '  %s\n' "$*"; }
fail_msg() { printf '  [DIFF] %s\n' "$*"; fail=$((fail + 1)); }

echo "=== 1/3 gofumpt 版本一致性 ==="
if [ -z "$CI_PIN" ]; then
  fail_msg "无法从 backend-ci.yml 解析 gofumpt pin"
elif [ "$CI_PIN" != "$GOFUMPT_VERSION" ]; then
  fail_msg "CI pin=$CI_PIN，本工具=$GOFUMPT_VERSION；请同步两处后再信任结果"
else
  note "CI 与本地均为 gofumpt $GOFUMPT_VERSION"
fi

GOBIN="$(go env GOPATH)/bin"
if [ ! -x "$GOBIN/gofumpt" ]; then
  note "本地未安装 gofumpt，执行 go install mvdan.cc/gofumpt@$GOFUMPT_VERSION"
  GOTOOLCHAIN=auto go install "mvdan.cc/gofumpt@$GOFUMPT_VERSION" || exit 2
fi

echo "=== 2/3 后端格式（backend-ci / Lint 同款） ==="
GOFUMPT_OUT="$(mktemp)"
(cd itsm-backend && "$GOBIN/gofumpt" -l . > "$GOFUMPT_OUT")
COUNT="$(grep -c . "$GOFUMPT_OUT" || true)"
if [ "$COUNT" -gt 0 ]; then
  fail_msg "未格式化文件 $COUNT 个；修复：make fmt-backend"
  head -12 "$GOFUMPT_OUT" | sed 's/^/    /'
  [ "$COUNT" -gt 12 ] && note "    ... 其余见 make fmt-backend 后的 git diff"
else
  note "0 个未格式化文件"
fi
rm -f "$GOFUMPT_OUT"

echo "=== 3/3 OpenAPI 生成产物漂移（api-contract-check 同款命令） ==="
TMPDOCS="$(mktemp -d)/docs"   # 目录名必须是 docs：swag 用目录 basename 作为 docs.go 包名
mkdir -p "$TMPDOCS"
if (cd itsm-backend && GOTOOLCHAIN=auto go run github.com/swaggo/swag/cmd/swag init \
      -d . -g main.go -o "$TMPDOCS" --parseDependency --parseInternal) >/dev/null 2>&1; then
  swag_diff=0
  for f in docs.go swagger.json swagger.yaml; do
    if ! diff -q "itsm-backend/docs/$f" "$TMPDOCS/$f" >/dev/null 2>&1; then
      swag_diff=$((swag_diff + 1))
      fail_msg "itsm-backend/docs/$f 与注解不一致；修复：make swagger-gen"
    fi
  done
  [ "$swag_diff" -eq 0 ] && note "docs.go / swagger.json / swagger.yaml 与注解一致"
  if [ "$swag_diff" -ne 0 ]; then
    note "提示：本项包含未提交 WIP 的注解；判定「CI 提交态」须在纯 HEAD 隔离树复核"
    note "  git archive HEAD | tar -x -C /tmp/itsm-head && 在该树 itsm-backend/ 下重跑同一条 swag 命令"
  fi
else
  fail_msg "swag 生成失败（检查 GOTOOLCHAIN / 网络 / go.sum）"
fi
rm -rf "$(dirname "$TMPDOCS")"

echo "=== 未覆盖项（需单独运行） ==="
note "前端单测：cd itsm-frontend && npm test -- --runInBand --forceExit --ci"
note "E2E / RLS / 安全扫描：见 ga-gate.yml、rls-golden-path.yml、security.yml"

echo "=== Summary: $fail 项差异 (mode=$MODE) ==="
if [ "$MODE" = "strict" ] && [ "$fail" -ne 0 ]; then
  exit 1
fi
exit 0
