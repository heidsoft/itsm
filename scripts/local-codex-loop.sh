#!/usr/bin/env bash
set -euo pipefail

SOURCE_REPO="/Users/heidsoft/Downloads/research/itsm"
WORKTREE_DIR="${WORKTREE_DIR:-/Users/heidsoft/Downloads/research/itsm-codex-loop}"
BRANCH_NAME="${BRANCH_NAME:-codex/autonomous-loop}"
CODEX_BIN="${CODEX_BIN:-/usr/local/bin/codex}"
LOG_DIR="$SOURCE_REPO/logs"
REPORT_DIR="$LOG_DIR/codex-loop"
LOCK_DIR="$LOG_DIR/.codex-loop.lock"
STATE_DIR="$SOURCE_REPO/.codex-loop"
STOP_FILE="$STATE_DIR/STOP"
NOTES_FILE="$STATE_DIR/SHARED_TASK_NOTES.md"
MAX_RUNS="${MAX_RUNS:-6}"
SLEEP_SECONDS="${SLEEP_SECONDS:-1800}"
FAIL_SLEEP_SECONDS="${FAIL_SLEEP_SECONDS:-900}"
MAX_CONSECUTIVE_FAILURES="${MAX_CONSECUTIVE_FAILURES:-3}"
NOT_BEFORE_EPOCH="${NOT_BEFORE_EPOCH:-0}"

export PATH="/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin:$PATH"

mkdir -p "$LOG_DIR" "$REPORT_DIR" "$STATE_DIR"

log() {
  printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S %z')" "$*"
}

cleanup() {
  rmdir "$LOCK_DIR" 2>/dev/null || true
}

if ! mkdir "$LOCK_DIR" 2>/dev/null; then
  log "another codex loop is already running; skip"
  exit 0
fi
trap cleanup EXIT

if [ ! -x "$CODEX_BIN" ]; then
  log "codex binary not found or not executable: $CODEX_BIN"
  exit 1
fi

now_epoch="$(date '+%s')"
if [ "$NOT_BEFORE_EPOCH" -gt "$now_epoch" ]; then
  log "not before gate active until $(date -r "$NOT_BEFORE_EPOCH" '+%Y-%m-%d %H:%M:%S %z'); skip"
  exit 0
fi

if [ ! -f "$NOTES_FILE" ]; then
  cat > "$NOTES_FILE" <<'NOTES'
# Codex Autonomous Loop Notes

## Mission
- 持续推进 AI Native ITSM 的小步功能迭代。
- 优先补齐 ITIL v3、CMDB、工作流、SLA、AI/RAG、连接器、CLI、测试和工程质量。

## Rules
- 每轮只做一个小而可审查的改动。
- 每轮必须记录修改、验证和下一步。
- 不 push 到远端。
NOTES
fi

ensure_worktree() {
  cd "$SOURCE_REPO"

  if [ -d "$WORKTREE_DIR/.git" ] || git -C "$WORKTREE_DIR" rev-parse --git-dir >/dev/null 2>&1; then
    return
  fi

  if git show-ref --verify --quiet "refs/heads/$BRANCH_NAME"; then
    git worktree add "$WORKTREE_DIR" "$BRANCH_NAME"
  else
    git worktree add -b "$BRANCH_NAME" "$WORKTREE_DIR" HEAD
  fi
}

sync_notes_to_worktree() {
  mkdir -p "$WORKTREE_DIR/.codex-loop"
  cp "$NOTES_FILE" "$WORKTREE_DIR/.codex-loop/SHARED_TASK_NOTES.md"
}

sync_notes_from_worktree() {
  if [ -f "$WORKTREE_DIR/.codex-loop/SHARED_TASK_NOTES.md" ]; then
    cp "$WORKTREE_DIR/.codex-loop/SHARED_TASK_NOTES.md" "$NOTES_FILE"
  fi
}

run_codex_exec() {
  local title="$1"
  local prompt="$2"
  local final_file="$3"
  local report_file="$4"

  {
    echo
    echo "## $title"
    echo
  } >> "$report_file"

  "$CODEX_BIN" -a never exec \
    --cd "$WORKTREE_DIR" \
    --sandbox workspace-write \
    --output-last-message "$final_file" \
    "$prompt" >> "$report_file" 2>&1
}

commit_if_changed() {
  local report_file="$1"

  cd "$WORKTREE_DIR"

  if [ -z "$(git status --porcelain -- ':!.codex-loop')" ]; then
    echo "No file changes to commit." >> "$report_file"
    return 0
  fi

  git add -A -- . ':!.codex-loop'
  if git diff --cached --quiet; then
    echo "No staged file changes to commit after excluding loop state." >> "$report_file"
    return 0
  fi

  git commit -m "chore: codex autonomous iteration $(date '+%Y%m%d-%H%M%S')" >> "$report_file" 2>&1
}

ensure_worktree

consecutive_failures=0
run_count=0

log "codex autonomous loop started"
log "worktree: $WORKTREE_DIR"
log "branch: $BRANCH_NAME"

while [ "$run_count" -lt "$MAX_RUNS" ]; do
  if [ -f "$STOP_FILE" ]; then
    log "stop file detected: $STOP_FILE"
    exit 0
  fi

  run_count=$((run_count + 1))
  timestamp="$(date '+%Y%m%d-%H%M%S')"
  report_file="$REPORT_DIR/loop-$timestamp.md"
  implement_final="$REPORT_DIR/loop-$timestamp-implement-final.md"
  cleanup_final="$REPORT_DIR/loop-$timestamp-cleanup-final.md"
  review_final="$REPORT_DIR/loop-$timestamp-review-final.md"

  sync_notes_to_worktree

  {
    echo "# Codex Autonomous Loop Iteration $run_count"
    echo
    echo "- Source Repo: $SOURCE_REPO"
    echo "- Worktree: $WORKTREE_DIR"
    echo "- Branch: $BRANCH_NAME"
    echo "- Started At: $(date '+%Y-%m-%d %H:%M:%S %z')"
    echo "- Codex: $("$CODEX_BIN" --version 2>/dev/null || echo "$CODEX_BIN")"
    echo
  } > "$report_file"

  log "loop iteration $run_count/$MAX_RUNS started: $report_file"

  set +e
  run_codex_exec "Implement" "你正在无人值守 loop 中推进 AI Native ITSM 项目功能迭代。

请先阅读 AGENTS.md、README.md、.codex-loop/SHARED_TASK_NOTES.md，并检查当前代码。

本轮目标：
1. 选择一个小而明确、能提升系统价值的迭代点。
2. 优先考虑 ITIL v3 流程、CMDB、BPMN 工作流、SLA、知识库/RAG、AI 辅助、连接器/插件/skill 市场、CLI、DTO/API 一致性、权限/租户隔离、测试补齐。
3. 实现代码和必要测试，范围必须可审查。
4. 不要 push，不要改远端配置，不要大规模重构。
5. 更新 .codex-loop/SHARED_TASK_NOTES.md，记录本轮完成内容、验证结果、下一轮建议。

最终输出：本轮选题、修改文件、验证命令与结果、遗留风险。" "$implement_final" "$report_file"
  implement_status=$?

  if [ "$implement_status" -eq 0 ]; then
    run_codex_exec "Cleanup And Verify" "请对当前 worktree 中本轮改动做 cleanup 和验证。

要求：
1. 删除无意义测试、冗余防御代码、console/debug 输出、无用注释。
2. 保留真实业务测试和必要错误处理。
3. 运行最相关的验证命令，例如 Go test、前端 type-check/lint/test，按改动范围选择。
4. 如果验证失败，优先修复；不能修复时把失败原因写入 .codex-loop/SHARED_TASK_NOTES.md。
5. 不要新增无关功能，不要 push。

最终输出：清理内容、验证结果、仍需人工关注的问题。" "$cleanup_final" "$report_file"
    cleanup_status=$?
  else
    cleanup_status=1
  fi

  if [ "$cleanup_status" -eq 0 ]; then
    run_codex_exec "Review Gate" "请以代码审查者身份 review 当前 worktree 的未提交改动。

重点检查：安全、租户隔离、鉴权、DTO/API 字段一致性、数据库迁移风险、并发、测试缺口、ITSM 业务回归。
如果发现必须修复的问题，请直接修复并运行相关验证。
如果没有阻断问题，请明确说明可以本地提交。
不要 push。

最终输出：findings、修复情况、验证结果。" "$review_final" "$report_file"
    review_status=$?
  else
    review_status=1
  fi
  set -e

  sync_notes_from_worktree

  {
    echo
    echo "## Git Status Before Commit"
    echo
    echo '```'
    git -C "$WORKTREE_DIR" status --short -- ':!.codex-loop'
    echo '```'
    echo
  } >> "$report_file"

  if [ "$implement_status" -eq 0 ] && [ "$cleanup_status" -eq 0 ] && [ "$review_status" -eq 0 ]; then
    commit_if_changed "$report_file"
    consecutive_failures=0
    log "loop iteration $run_count completed"
  else
    consecutive_failures=$((consecutive_failures + 1))
    log "loop iteration $run_count failed; consecutive failures: $consecutive_failures"
  fi

  {
    echo
    echo "- Finished At: $(date '+%Y-%m-%d %H:%M:%S %z')"
    echo "- Implement Status: $implement_status"
    echo "- Cleanup Status: $cleanup_status"
    echo "- Review Status: $review_status"
    echo
    echo "## Git Status After Iteration"
    echo
    echo '```'
    git -C "$WORKTREE_DIR" status --short -- ':!.codex-loop'
    echo '```'
  } >> "$report_file"

  if [ "$consecutive_failures" -ge "$MAX_CONSECUTIVE_FAILURES" ]; then
    log "max consecutive failures reached; stopping loop"
    exit 1
  fi

  if [ "$run_count" -lt "$MAX_RUNS" ]; then
    if [ "$implement_status" -eq 0 ] && [ "$cleanup_status" -eq 0 ] && [ "$review_status" -eq 0 ]; then
      sleep "$SLEEP_SECONDS"
    else
      sleep "$FAIL_SLEEP_SECONDS"
    fi
  fi
done

log "codex autonomous loop reached MAX_RUNS=$MAX_RUNS"
