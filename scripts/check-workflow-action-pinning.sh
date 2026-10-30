#!/usr/bin/env bash
# =============================================================================
# GitHub Actions 引用固定守卫
#
# 背景（2026-10-31）：security.yml 里 gosec 与 trivy 长期用 `@master`、
# trufflehog 用 `@main`——浮动分支引用。后果不是「不安全」这种抽象说法，而是：
#   - 上游一次合并就能改变两次运行之间的扫描结果，报告无法复现；
#   - `uses: x@main` 无法回答「这次跑的是哪个版本」，出问题时无从回溯；
#   - 同一 workflow 里的 gosec 已因浮动引用被钉版本，另两处却漏了——说明
#     靠人工 review 记不住，需要机器守。
#
# 规则：远程 action 的 ref 必须是
#   - 版本 tag：`v1` / `v4` / `v0.36.0` / `3.99.7`（可带前导 v），或
#   - 40 位 commit SHA（最严格，tag 被移动时仍可复现）。
# 本地 action（`./...`）与 `docker://` 镜像不在本守卫范围。
#
# 退出码 0 = 通过；非 0 = 存在浮动引用。
# =============================================================================
set -uo pipefail

cd "$(dirname "$0")/.." || exit 2
ROOT="$(pwd)"

# 判定单个 ref 是否合规。返回 0 = 合规，1 = 浮动/不可复现。
is_pinned_ref() {
  local ref="$1"
  if [[ "$ref" =~ ^[0-9a-f]{40}$ ]]; then
    return 0
  fi
  # v1 / v4 / v0.36.0 / 3.99.7；不允许 -beta、branch 名、master/main/latest。
  if [[ "$ref" =~ ^v?[0-9]+(\.[0-9]+)*$ ]]; then
    return 0
  fi
  return 1
}

# --- 自检：守卫本身必须先证明能拦住浮动引用，否则它可能零命中地通过 ---------
self_test() {
  local failures=0
  local pinned=("v4" "v0.36.0" "3.99.7" "0123456789abcdef0123456789abcdef01234567")
  local floating=("main" "master" "HEAD" "latest" "release/v1" "v1-beta")

  local ref
  for ref in "${pinned[@]}"; do
    if ! is_pinned_ref "$ref"; then
      echo "✗ 自检失败：应放行的 ref 被判为浮动：$ref"
      failures=$((failures + 1))
    fi
  done
  for ref in "${floating[@]}"; do
    if is_pinned_ref "$ref"; then
      echo "✗ 自检失败：应拦截的浮动 ref 被判为合规：$ref"
      failures=$((failures + 1))
    fi
  done
  return "$failures"
}

if ! self_test; then
  echo "守卫自检未通过，拒绝继续（否则可能零命中地给出假绿）。"
  exit 2
fi

violations=0
checked=0

shopt -s nullglob
for wf in "$ROOT"/.github/workflows/*.yml "$ROOT"/.github/workflows/*.yaml; do
  [ -f "$wf" ] || continue
  rel="${wf#"$ROOT"/}"
  # 只匹配 YAML 的 uses 键（行首缩进后可选 "- "），注释行与 run: | 里的文本
  # 不会命中——否则一段示例脚本就能让守卫误报。
  while IFS= read -r match; do
    lineno="${match%%:*}"
    line="${match#*:}"
    value="$(printf '%s' "$line" | sed -E 's/^[[:space:]]*(-[[:space:]]+)?uses:[[:space:]]*//; s/[[:space:]]*#.*$//; s/[[:space:]]*$//')"
    [ -n "$value" ] || continue
    case "$value" in
      ./*) continue ;;          # 本地 action
      docker://*) continue ;;   # 镜像由 digest 策略单独管理
    esac
    checked=$((checked + 1))
    if [[ "$value" != *"@"* ]]; then
      echo "✗ ${rel}:${lineno} 缺少 ref（必须 @<version> 或 @<sha>）：$value"
      violations=$((violations + 1))
      continue
    fi
    ref="${value##*@}"
    if ! is_pinned_ref "$ref"; then
      echo "✗ ${rel}:${lineno} 浮动引用：$value"
      echo "    ref '${ref}' 不是版本 tag 也不是 40 位 SHA；上游改动会让扫描/构建结果不可复现。"
      violations=$((violations + 1))
    fi
  done < <(grep -nE '^[[:space:]]*(-[[:space:]]+)?uses:' "$wf")
done

if [ "$checked" -eq 0 ]; then
  echo "✗ 未扫描到任何远程 action 引用——守卫可能因路径错误而零命中。"
  exit 2
fi

if [ "$violations" -gt 0 ]; then
  echo "共 ${violations} 处浮动引用（已检查 ${checked} 处）。"
  exit 1
fi

echo "✓ Workflow action 引用全部固定（${checked} 处，均为版本 tag 或 SHA）。"
