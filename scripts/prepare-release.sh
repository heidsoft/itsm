#!/usr/bin/env bash
# prepare-release.sh — 发布准备脚本
#
# 用法:
#   scripts/prepare-release.sh <version>          # 完整流程：校验 → 打 tag → push → 创建 Release
#   scripts/prepare-release.sh <version> --dry-run # 只校验，不执行写操作
#   scripts/prepare-release.sh <version> --tag-only # 只打 tag 并 push，不创建 GitHub Release
#
# 示例:
#   scripts/prepare-release.sh v1.6.12
#   scripts/prepare-release.sh v1.6.12 --dry-run
#
# 前置条件:
#   - gh CLI 已登录（gh auth status）
#   - 工作目录干净（无未提交变更）
#   - CHANGELOG.md 包含对应版本的条目

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

VERSION="${1:-}"
DRY_RUN=false
TAG_ONLY=false

shift || true
for arg in "$@"; do
  case "$arg" in
    --dry-run)  DRY_RUN=true ;;
    --tag-only) TAG_ONLY=true ;;
    *)          error "未知参数: $arg"; exit 1 ;;
  esac
done

if [[ -z "$VERSION" ]]; then
  error "用法: $0 <version> [--dry-run] [--tag-only]"
  echo "  示例: $0 v1.6.12"
  exit 1
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
  error "版本号格式不合法: $VERSION（期望 v1.2.3 或 v1.2.3-rc.1）"
  exit 1
fi

# ── Step 1: 前置检查 ──────────────────────────────────────────────────────────
info "Step 1/6: 前置检查"

if ! command -v gh &>/dev/null; then
  error "gh CLI 未安装，请先安装: https://cli.github.com/"
  exit 1
fi

if ! gh auth status &>/dev/null 2>&1; then
  error "gh CLI 未登录，请先运行: gh auth login"
  exit 1
fi

# 检查未提交变更
if ! git diff --quiet || ! git diff --cached --quiet; then
  error "工作目录有未提交变更，请先提交或 stash"
  git status --short
  exit 1
fi

# 检查 tag 是否已存在
if git rev-parse "$VERSION" &>/dev/null 2>&1; then
  error "tag $VERSION 已存在: $(git rev-parse "$VERSION")"
  exit 1
fi

info "  gh CLI: OK"
info "  工作目录: 干净"
info "  tag $VERSION: 不存在（可创建）"

# ── Step 2: 校验 CHANGELOG ────────────────────────────────────────────────────
info "Step 2/6: 校验 CHANGELOG.md"

CHANGELOG="CHANGELOG.md"
if [[ ! -f "$CHANGELOG" ]]; then
  error "$CHANGELOG 不存在"
  exit 1
fi

# 提取版本日期（期望格式: ## [1.6.12] - 2026-10-03）
VERSION_NO_V="${VERSION#v}"
TODAY=$(date +%Y-%m-%d)
CHANGELOG_HEADER="## [${VERSION_NO_V}]"

if ! grep -qF "$CHANGELOG_HEADER" "$CHANGELOG"; then
  error "CHANGELOG.md 未找到 $CHANGELOG_HEADER 条目"
  echo "  请先在 CHANGELOG.md 中添加 $VERSION_NO_V 的发布说明"
  exit 1
fi

# 检查日期
if grep -qF "${CHANGELOG_HEADER}] - ${TODAY}" "$CHANGELOG"; then
  info "  CHANGELOG 条目: OK（日期 $TODAY）"
else
  EXPECTED_LINE=$(grep -nF "$CHANGELOG_HEADER" "$CHANGELOG" | head -1)
  warn "  CHANGELOG 日期可能不是今天（今天=$TODAY）"
  warn "  找到: $EXPECTED_LINE"
fi

# ── Step 3: 提取 Release Notes ────────────────────────────────────────────────
info "Step 3/6: 提取 Release Notes"

# 从 CHANGELOG 提取当前版本的条目（从 ## [version] 到下一个 ## [ 或文件末尾）
extract_changelog_section() {
  local version="$1"
  local file="$2"
  local in_section=false
  local section=""

  while IFS= read -r line; do
    if [[ "$line" == "## [${version}]"* ]]; then
      in_section=true
      continue
    fi
    if $in_section; then
      if [[ "$line" == "## ["* ]]; then
        break
      fi
      section+="$line"$'\n'
    fi
  done < "$file"

  echo "$section" | sed '/^$/d' | head -100
}

RELEASE_NOTES=$(extract_changelog_section "$VERSION_NO_V" "$CHANGELOG")

if [[ -z "$RELEASE_NOTES" ]]; then
  error "无法从 CHANGELOG.md 提取 $VERSION_NO_V 的发布说明"
  exit 1
fi

NOTES_LINE_COUNT=$(echo "$RELEASE_NOTES" | wc -l | tr -d ' ')
info "  提取 $NOTES_LINE_COUNT 行 Release Notes"

if $DRY_RUN; then
  info "  [DRY-RUN] Release Notes 预览（前 20 行）:"
  echo "$RELEASE_NOTES" | head -20
  echo "  ..."
fi

# ── Step 4: 检查远程同步 ─────────────────────────────────────────────────────
info "Step 4/6: 检查远程同步"

CURRENT_BRANCH=$(git branch --show-current)
info "  当前分支: $CURRENT_BRANCH"

# 检查是否有未推送的 commit
UNPUSHED=$(git log @{u}..HEAD --oneline 2>/dev/null | wc -l | tr -d ' ' || echo "?")
if [[ "$UNPUSHED" != "0" ]]; then
  warn "  有 $UNPUSHED 个未推送的 commit（tag 会包含这些 commit）"
else
  info "  远程同步: OK"
fi

# ── Step 5: 创建 Tag ─────────────────────────────────────────────────────────
info "Step 5/6: 创建 annotated tag"

if $DRY_RUN; then
  info "  [DRY-RUN] 将创建 tag: $VERSION"
  info "  [DRY-RUN] 命令: git tag -a $VERSION -m \"Release $VERSION\""
else
  git tag -a "$VERSION" -m "Release $VERSION"
  info "  tag $VERSION 已创建"

  info "  推送 tag 到远程..."
  git push origin "$VERSION"
  info "  tag 已推送"
fi

# ── Step 6: 创建 GitHub Release ──────────────────────────────────────────────
if $TAG_ONLY; then
  info "Step 6/6: 跳过 GitHub Release（--tag-only 模式）"
else
  info "Step 6/6: 创建 GitHub Release"

  # 生成完整的 release notes（包含 CHANGELOG 内容 + 自动对比链接）
  FULL_NOTES="$RELEASE_NOTES"
  PREV_TAG=$(git tag -l 'v*' --sort=-version:refname | grep -v "^${VERSION}$" | head -1 || true)
  if [[ -n "$PREV_TAG" ]]; then
    FULL_NOTES+=$'\n'
    FULL_NOTES+="**Full Changelog**: https://github.com/$(gh repo view --json nameWithOwner -q .nameWithOwner)/compare/${PREV_TAG}...${VERSION}"
  fi

  if $DRY_RUN; then
    info "  [DRY-RUN] 将创建 GitHub Release: $VERSION"
    info "  [DRY-RUN] Release Notes 完整预览:"
    echo "---"
    echo "$FULL_NOTES"
    echo "---"
  else
    # 写入临时文件避免 shell 转义问题
    NOTES_FILE=$(mktemp)
    echo "$FULL_NOTES" > "$NOTES_FILE"

    gh release create "$VERSION" \
      --title "$VERSION" \
      --notes-file "$NOTES_FILE" \
      --latest

    rm -f "$NOTES_FILE"
    info "  GitHub Release $VERSION 已创建"
  fi
fi

# ── 完成 ─────────────────────────────────────────────────────────────────────
echo ""
info "========================================="
if $DRY_RUN; then
  info "DRY-RUN 完成，未执行任何写操作"
else
  info "Release $VERSION 发布完成！"
  echo ""
  info "后续步骤:"
  info "  1. CI workflow 已自动触发（release.yml）"
  info "     查看: https://github.com/$(gh repo view --json nameWithOwner -q .nameWithOwner)/actions"
  info "  2. Docker 镜像将在 ~8 分钟后推送到 GHCR"
  info "  3. 验证: gh run list --workflow=release.yml --limit 1"
fi
info "========================================="
