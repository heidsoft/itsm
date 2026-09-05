#!/usr/bin/env bash
#
# scripts/check-handlers-hygiene.sh
#
# handlers/<domain> 垂直切片卫生检查（advisory / strict 两档）：
#
#   H.1 裸奔域警告：域目录存在非测试 .go 文件，但没有 service.go 或 repository.go。
#       -> 业务逻辑疑堆积在 handler.go，违反五件套分层（评审 P0）。
#   H.2 域间直接 import 警告：handlers/<domain> 生产代码直接 import 其他
#       handlers/<other>（common/shared 及其子包除外）。
#       -> 接口类型依赖合法（如 standard_change → change 仅用 DTO 类型），
#          但值得人工确认；具体实现依赖则属违规。
#   H.3 下划线包名冻结：新建域包名含下划线则提示改用短名（不阻断）。
#
# 用法：
#   ./scripts/check-handlers-hygiene.sh            # advisory（仅报告）
#   ./scripts/check-handlers-hygiene.sh --strict   # strict（WARN 计数>0 即失败）
#
# 与 docs-gate 一致：v1.x advisory，v2.0 起建议接入 CI 为 strict。
#

set -uo pipefail

ROOT_DIR="${HANDLERS_GATE_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
HANDLERS_DIR="${ROOT_DIR}/itsm-backend/handlers"
STRICT="${1:-}"

TOTAL_WARN=0

echo "== handlers 分层卫生检查 =="
if [ ! -d "${HANDLERS_DIR}" ]; then
  echo "handlers 目录不存在: ${HANDLERS_DIR}"
  exit 1
fi

for dir in "${HANDLERS_DIR}"/*/; do
  [ -d "${dir}" ] || continue
  domain="$(basename "${dir}")"
  # 跳过纯共享目录：common/shared 及其子包天然多域消费
  if [[ "${domain}" == "common" || "${domain}" == "shared" ]]; then
    continue
  fi

  has_prod_go=0
  for f in "${dir}"*.go; do
    [ -f "${f}" ] || continue
    base="$(basename "${f}")"
    [[ "${base}" == *_test.go || "${base}" == doc.go ]] && continue
    has_prod_go=1
    break
  done
  if [ "${has_prod_go}" -eq 0 ]; then
    continue
  fi

  # H.1 裸奔域：有生产 .go 但无 service.go / repository.go
  if [ ! -f "${dir}service.go" ] && [ ! -f "${dir}repository.go" ]; then
    echo "  [H.1] 裸奔域 ${domain}: 无 service.go / repository.go，业务逻辑可能堆积在 handler.go"
    TOTAL_WARN=$((TOTAL_WARN + 1))
  fi

  # H.2 域间直接 import（生产代码）
  prod_files=()
  for f in "${dir}"*.go; do
    [ -f "${f}" ] || continue
    base="$(basename "${f}")"
    [[ "${base}" == *_test.go ]] && continue
    prod_files+=("${f}")
  done
  if [ ${#prod_files[@]} -gt 0 ]; then
    imports="$(grep -hoE '"itsm-backend/handlers/[a-z0-9_/]+"' "${prod_files[@]}" 2>/dev/null | tr -d '"' | sort -u)"
    if [ -n "${imports}" ]; then
      while IFS= read -r imp; do
        # 跳过 common/shared 共享子包
        case "${imp}" in
          *handlers/common*|*handlers/shared*) continue ;;
        esac
        echo "  [H.2] ${domain} 直接依赖 ${imp}（确认是接口/DTO 类型依赖而非具体实现）"
        TOTAL_WARN=$((TOTAL_WARN + 1))
      done <<< "${imports}"
    fi
  fi

  # H.3 下划线包名（新建域提示）
  if [[ "${domain}" == *_* ]]; then
    echo "  [H.3] ${domain}: 包名含下划线，新建域请用短小写单词（存量不强制改）"
  fi
done

echo ""
if [ "${TOTAL_WARN}" -gt 0 ]; then
  echo "== 卫生检查: ${TOTAL_WARN} 条警告 =="
else
  echo "== 卫生检查: 无警告 =="
fi

if [ "${STRICT}" = "--strict" ] && [ "${TOTAL_WARN}" -gt 0 ]; then
  echo "strict 模式：存在警告，退出码 1"
  exit 1
fi
exit 0
