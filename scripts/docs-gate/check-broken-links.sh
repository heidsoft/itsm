#!/usr/bin/env bash
#
# scripts/docs-gate/check-broken-links.sh
#
# Gate C.3 — 内部 markdown 链接失效检测（advisory only）。
#
# 策略：
#   - 扫描 .md / .mdx 里的内部相对链接与路径锚点：裸写的 `中文名.md`、`scripts/foo.sh`
#     同样校验，不要求 `./` 前缀。
#   - 不验证外部链接（http/https/mailto 等带 scheme 的，避免外部站点波动误伤）。
#   - `file:///...` 指向写文档那台机器的绝对路径，别人 clone 后必然失效，单独报
#     non-portable / broken 两类。
#   - 使用 grep + awk 抽取 [text](path)，逐条验证 path 是否存在。
#
# 已知限制（advisory）：
#   - 跨 repo 链接（GitHub issue / PR）跳过。
#   - 文档锚点 (#section) 仅做存在性检查，不解析 heading。
#   - `-template.md`、`-guide.md` 结尾的文档不校验（其中链接是样例，目标未必存在）。
#
# 当前阶段（v1.5）advisory：仅警告，不阻断。
# v2.0 起升级为 hard。
#
# 用法：
#   ./scripts/docs-gate/check-broken-links.sh [--strict]
#

set -uo pipefail

ROOT_DIR="${DOCS_GATE_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}"
cd "${ROOT_DIR}"

STRICT="${1:-}"
VIOLATIONS=0

echo "########################################"
echo "# Gate C.3 — 内部 markdown 链接失效检测（advisory）"
echo "########################################"

# 抽取内部链接，验证存在性
check_links_in_file() {
  local file="$1"
  local dir
  dir="$(dirname "${file}")"

  # 匹配 [text](path)。只排除带 scheme 的外部链接与纯锚点，
  # 其余都按仓库内相对路径解析——裸写的 `中文名.md`、`foo.sh` 同样要校验。
  grep -oE '\[[^]]+\]\(([^)]+)\)' "${file}" \
    | sed -E 's/^\[[^]]+\]\(//; s/\)$//' \
    | while IFS= read -r link; do
        # 去掉 #anchor
        path="${link%%#*}"
        if [ -z "${path}" ]; then
          continue
        fi
        case "${path}" in
          http://* | https://* | ftp://* | mailto:* | tel:* | data:* | "www."*) continue ;;
        esac
        # file:// 链接只在写它的那台机器上成立，仓库文档里必须用相对路径
        case "${path}" in
          file://*)
            target="${path#file://}"
            if [[ "${target}" == "${ROOT_DIR}"/* && -e "${target}" ]]; then
              echo "  - ${file}: non-portable link '${link}' (指向本机绝对路径，须改为仓库相对路径)"
            else
              echo "  - ${file}: broken link '${link}' -> ${target} (missing)"
            fi
            continue ;;
        esac
        # 只校验看起来像文件/目录的链接，跳过说明性文字
        if [[ "${path}" != */* && ! "${path}" =~ \.[A-Za-z0-9]{1,5}$ ]]; then
          continue
        fi
        # 解析相对路径
        if [[ "${path}" == /* ]]; then
          if [[ "${path}" != "${ROOT_DIR}"/* ]]; then
            echo "  - ${file}: broken link '${link}' -> ${path} (指向仓库外绝对路径)"
            continue
          fi
          full_norm="${ROOT_DIR}${path}"
        else
          parent="${ROOT_DIR}/${dir}/$(dirname "${path}")"
          if [ -d "${parent}" ]; then
            full_norm="$(cd "${parent}" && pwd)/$(basename "${path}")"
          else
            full_norm="${ROOT_DIR}/${dir}/${path}"
          fi
        fi
        if [ ! -e "${full_norm}" ]; then
          echo "  - ${file}: link '${link}' -> ${full_norm} (missing)"
        fi
      done
}

while IFS= read -r -d '' f; do
  # 模板/指南文件中的"链接"是样例，目标文件不一定存在
  [[ "${f}" =~ (-template\.md$|-guide\.md$) ]] && continue
  HITS="$(check_links_in_file "${f}" 2>/dev/null || true)"
  if [ -n "${HITS}" ]; then
    echo "${HITS}"
    VIOLATIONS=$((VIOLATIONS + $(echo "${HITS}" | wc -l | tr -d ' ')))
  fi
done < <(git ls-files -z '*.md' '*.mdx')

echo ""
echo "########################################"
echo "# Gate C.3 Summary: ${VIOLATIONS} broken link(s)"
echo "########################################"

if [ "${STRICT}" = "--strict" ]; then
  if [ "${VIOLATIONS}" -gt 0 ]; then
    echo "::error::Broken internal links detected. Fix paths or convert to external URL."
    exit 1
  fi
fi

if [ "${VIOLATIONS}" -gt 0 ]; then
  echo "[advisory] ${VIOLATIONS} potentially broken internal link(s). Review and fix."
fi
exit 0
