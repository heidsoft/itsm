#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="/Users/heidsoft/Downloads/research/itsm"
CODEX_BIN="${CODEX_BIN:-/usr/local/bin/codex}"
LOG_DIR="$REPO_DIR/logs"
REPORT_DIR="$LOG_DIR/codex-review"
LOCK_DIR="$LOG_DIR/.codex-review.lock"

export PATH="/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin:$PATH"

mkdir -p "$LOG_DIR" "$REPORT_DIR"

log() {
  printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S %z')" "$*"
}

cleanup() {
  rmdir "$LOCK_DIR" 2>/dev/null || true
}

if ! mkdir "$LOCK_DIR" 2>/dev/null; then
  log "another codex review is already running; skip"
  exit 0
fi
trap cleanup EXIT

cd "$REPO_DIR"

if [ ! -x "$CODEX_BIN" ]; then
  log "codex binary not found or not executable: $CODEX_BIN"
  exit 1
fi

timestamp="$(date '+%Y%m%d-%H%M%S')"
report_file="$REPORT_DIR/review-$timestamp.md"

review_prompt='请以资深工程师代码审查视角 review 这个 AI Native ITSM 项目的代码变更。重点关注：
1. 后端 Go/Gin/Ent API 是否违反 DTO 返回规范、租户隔离、鉴权、错误处理和日志规范。
2. 前端 Next.js/TypeScript 是否存在类型错误、API 字段命名不一致、Ant Design 组件用法错误、交互或状态问题。
3. ITIL v3 流程、CMDB、BPMN 工作流、SLA、RAG/AI 能力相关代码是否存在业务回归风险。
4. 安全问题、数据泄露、权限绕过、并发问题、数据库迁移风险和缺失测试。
请按严重程度输出 findings，包含文件路径、行号、影响和建议修复方式；没有问题时明确说明。'

{
  echo "# Codex Scheduled Code Review"
  echo
  echo "- Repository: $REPO_DIR"
  echo "- Started At: $(date '+%Y-%m-%d %H:%M:%S %z')"
  echo "- Codex: $("$CODEX_BIN" --version 2>/dev/null || echo "$CODEX_BIN")"
  echo
} > "$report_file"

run_review() {
  local title="$1"
  shift

  {
    echo
    echo "## $title"
    echo
    "$CODEX_BIN" review "$@" "$review_prompt"
  } >> "$report_file" 2>&1
}

select_base() {
  if git rev-parse --verify --quiet origin/main >/dev/null; then
    echo "origin/main"
    return
  fi

  if git rev-parse --verify --quiet origin/master >/dev/null; then
    echo "origin/master"
    return
  fi

  if git rev-parse --verify --quiet main >/dev/null; then
    echo "main"
    return
  fi

  if git rev-parse --verify --quiet master >/dev/null; then
    echo "master"
    return
  fi
}

base_ref="$(select_base || true)"
ran_review=0

log "scheduled codex review started"
log "report file: $report_file"

if [ -n "$base_ref" ] && ! git diff --quiet "$base_ref"...HEAD; then
  log "reviewing committed changes against $base_ref"
  run_review "Committed Changes Against $base_ref" --base "$base_ref"
  ran_review=1
else
  {
    echo
    echo "## Committed Changes"
    echo
    if [ -n "$base_ref" ]; then
      echo "No committed diff found against \`$base_ref\`."
    else
      echo "No base branch found. Skipped committed-change review."
    fi
  } >> "$report_file"
fi

if [ -n "$(git status --porcelain)" ]; then
  log "reviewing uncommitted changes"
  run_review "Uncommitted Changes" --uncommitted
  ran_review=1
else
  {
    echo
    echo "## Uncommitted Changes"
    echo
    echo "Working tree is clean."
  } >> "$report_file"
fi

{
  echo
  echo "- Finished At: $(date '+%Y-%m-%d %H:%M:%S %z')"
} >> "$report_file"

if [ "$ran_review" -eq 0 ]; then
  log "no code changes detected; report recorded without running codex review"
else
  log "scheduled codex review completed"
fi
