#!/usr/bin/env bash
# =============================================================================
# Go 工具链版本一致性守卫
#
# 背景（2026-10-30 实证）：把 go.mod 的 go 指令从 1.25.13 升到 1.26.10 后，
# 四处脱钩、三个工作流变红：
#   1. backend-ci 的 lint 缓存键只按 go.mod 哈希，restore-keys 前缀复活了用
#      go1.25.13 构建的 staticcheck 二进制 → 无法分析 1.26 标准库；
#   2. dev/prod Dockerfile 钉 golang:1.25.13-alpine 且 GOTOOLCHAIN=local
#      → go mod download 拒载；
#   3. pg-disaster-recovery 与 rls-golden-path 硬编码 go-version: '1.25'
#      → go test 拒载（go.mod requires go >= 1.26.10）；
#   4. govulncheck 对 1.26.0 报出 40 处标准库漏洞（最高修复版本 1.26.10）。
#
# 本守卫固定「Go 版本只有 go.mod 一个真相源」这条不变量：
#   - 所有工作流的 setup-go 必须用 go-version-file: itsm-backend/go.mod，
#     不得硬编码版本号（硬编码值必然随 go.mod 升级而脱钩）；
#   - 所有 Dockerfile 的 golang 基础镜像 tag 必须与 go.mod 的 go 指令一致；
#   - lint 工具缓存键必须包含解析出的 Go 版本（否则工具链升级会复活陈旧二进制）。
#
# 退出码 0 = 通过；非 0 = 有脱钩点。
# =============================================================================
set -uo pipefail

cd "$(dirname "$0")/.." || exit 2
ROOT="$(pwd)"

GO_MOD="${ROOT}/itsm-backend/go.mod"
[ -f "$GO_MOD" ] || { echo "找不到 $GO_MOD"; exit 2; }

# go.mod 的 go 指令形如 "go 1.26.10"；取 major.minor.patch
GO_VERSION="$(awk '$1 == "go" { print $2; exit }' "$GO_MOD")"
[ -n "$GO_VERSION" ] || { echo "无法从 go.mod 解析 go 指令"; exit 2; }
GO_MAJOR="${GO_VERSION%%.*}"
GO_REST="${GO_VERSION#*.}"
GO_MINOR="${GO_REST%%.*}"

echo "go.mod 声明的 Go 版本: ${GO_VERSION}"

violations=0

# -----------------------------------------------------------------------------
# 1) 工作流不得硬编码 go-version（必须用 go-version-file 跟随 go.mod）
# -----------------------------------------------------------------------------
echo "=== 检查工作流的 setup-go 版本来源 ==="
while IFS= read -r wf; do
  # 只看 setup-go 步骤附近的 go-version 行（排除 go-version-file）
  hardcoded="$(awk '
    /uses: actions\/setup-go/ { in_setup = 1; next }
    in_setup && /go-version:/ { print FILENAME ":" FNR ": " $0; in_setup = 0 }
    in_setup && /^      - name:/ { in_setup = 0 }
  ' "$wf")"
  if [ -n "$hardcoded" ]; then
    echo "$hardcoded" | while IFS= read -r line; do
      echo "  FAIL: $line"
    done
    violations=$((violations + 1))
  fi
done < <(find "${ROOT}/.github/workflows" -name "*.yml" -type f | sort)

# -----------------------------------------------------------------------------
# 2) Dockerfile 的 golang 基础镜像必须与 go.mod 一致
# -----------------------------------------------------------------------------
echo "=== 检查 Dockerfile 的 golang 基础镜像 ==="
while IFS= read -r df; do
  tag="$(awk '/^FROM .*golang:/ { for (i = 1; i <= NF; i++) if ($i ~ /^golang:/) { print $i; exit } }' "$df")"
  [ -n "$tag" ] || continue
  image_ver="${tag#golang:}"
  image_ver="${image_ver%%-*}"   # 去掉 -alpine 之类后缀
  if [ "$image_ver" != "$GO_VERSION" ]; then
    echo "  FAIL: $(basename "$df") 用 golang:${image_ver}，与 go.mod 的 ${GO_VERSION} 不一致"
    violations=$((violations + 1))
  else
    echo "  OK: $(basename "$df") → golang:${image_ver}"
  fi
done < <(find "${ROOT}/itsm-backend" -maxdepth 1 -name "Dockerfile*" -type f | sort)

# -----------------------------------------------------------------------------
# 3) lint 工具缓存键必须绑定实际 Go 版本
# -----------------------------------------------------------------------------
echo "=== 检查 lint 工具缓存键 ==="
CACHE_WF="${ROOT}/.github/workflows/backend-ci.yml"
if [ -f "$CACHE_WF" ]; then
  if grep -q "go-lint-tools-" "$CACHE_WF"; then
    if grep -q "go-lint-tools-.*setup-go.outputs.go-version" "$CACHE_WF"; then
      echo "  OK: 缓存键已绑定 setup-go 解析出的 Go 版本"
    else
      echo "  FAIL: lint 工具缓存键未绑定 setup-go.outputs.go-version——工具链升级会复活陈旧二进制"
      violations=$((violations + 1))
    fi
  fi
fi

echo
if [ "$violations" -gt 0 ]; then
  echo "工具链一致性检查失败：${violations} 处脱钩。"
  echo "Go 版本只有一个真相源——itsm-backend/go.mod 的 go 指令。所有引用点必须跟随它。"
  exit 1
fi

echo "工具链一致性检查通过（Go ${GO_VERSION}）。"
exit 0
