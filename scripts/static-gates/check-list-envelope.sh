#!/usr/bin/env bash
#
# scripts/static-gates/check-list-envelope.sh
#
# 门禁 5.10（硬）：handlers/** 里带分页字段（page/pageSize/totalPages）的 gin.H 响应体
# 必须把集合放在 items 键下。列表不得再放在 tickets/reports/logs/... 等领域键
# 下（标准信封见 itsm-backend/common/pagination.go 与 AGENTS.md「API 响应格式」）。
#
# 用法：
#   ./scripts/static-gates/check-list-envelope.sh
#

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT_DIR}/itsm-backend" || exit 1

if [[ ! -d handlers ]]; then
  echo "SKIP: handlers 目录不存在。"
  exit 0
fi

# 提取每个 gin.H{...} 字面量块，输出 "file:line<TAB>逗号分隔的键列表"。
# 键 = 块内出现的双引号字符串字面量（含值），判定只看是否命中下列约定，
# 因此误报只会来自同名值，不会漏报。
BLOCKS="$(
  find handlers -name '*.go' -not -name '*_test.go' -print0 | xargs -0 awk '
    function emit() {
      keys = ""
      n = split(blk, parts, "\"")
      for (i = 2; i <= n; i += 2) {
        if (parts[i] ~ /^[a-z][a-zA-Z0-9]*$/) keys = keys "," parts[i]
      }
      sub(/^,/, "", keys)
      if (keys != "") printf "%s:%d\t%s\n", FILENAME, startline, keys
    }
    FNR == 1 { blk = ""; start = 0 }
    {
      if (start == 0) {
        if (index($0, "gin.H{") == 0) next
        start = 1; startline = FNR; blk = $0
      } else {
        blk = blk " " $0
      }
      depth = gsub(/\{/, "{", blk) - gsub(/\}/, "}", blk)
      if (depth <= 0) { emit(); start = 0 }
    }
  '
)"

# 列表数据键黑名单：承载列表时必须改名为 items。
DOMAIN_LIST_KEYS="tickets incidents problems changes releases reports templates
notifications instances logs records allocations customers users tasks comments"

VIOLATIONS=0
REPORT=""

while IFS=$'\t' read -r loc keys; do
  [[ -z "${loc}" ]] && continue

  has_page=0
  has_items=0
  IFS=',' read -r -a key_arr <<<"${keys}"
  for k in "${key_arr[@]}"; do
    case "${k}" in
      page | pageSize | totalPages) has_page=1 ;;
      items) has_items=1 ;;
    esac
  done

  if [[ "${has_page}" -eq 1 && "${has_items}" -eq 0 ]]; then
    VIOLATIONS=$((VIOLATIONS + 1))
    REPORT="${REPORT}${loc} 分页信封缺少 items 键: [${keys}]"$'\n'
  fi

  for bad in ${DOMAIN_LIST_KEYS}; do
    for k in "${key_arr[@]}"; do
      [[ "${k}" == "${bad}" ]] || continue
      # 与 page/pageSize/totalPages 同时出现 => 该键承载的是分页列表
      if [[ "${has_page}" -eq 1 ]]; then
        VIOLATIONS=$((VIOLATIONS + 1))
        REPORT="${REPORT}${loc} 列表放在领域键 \"${k}\" 下，应改为 items: [${keys}]"$'\n'
      fi
    done
  done
done <<<"${BLOCKS}"

echo "==== Static Gate: handler list envelope (hard) ===="
if [[ "${VIOLATIONS}" -gt 0 ]]; then
  echo "FAIL: ${VIOLATIONS} 处 handler 响应体违反标准列表信封契约。"
  printf '%s' "${REPORT}"
  echo ""
  echo "修复：使用 common.SuccessWithList(c, items, total, page, pageSize)；"
  echo "      不分页的列表只带 {items,total}，不得伪造 page/pageSize。"
  exit 1
fi

echo "PASS: handlers 内所有分页 gin.H 响应体均使用 items 键。"
exit 0
