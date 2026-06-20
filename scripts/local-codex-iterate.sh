#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="/Users/heidsoft/Downloads/research/itsm"
CODEX_BIN="${CODEX_BIN:-/usr/local/bin/codex}"
LOG_DIR="$REPO_DIR/logs"
REPORT_DIR="$LOG_DIR/codex-iterate"
LOCK_DIR="$LOG_DIR/.codex-iterate.lock"
ALLOW_DIRTY_WORKTREE="${ALLOW_DIRTY_WORKTREE:-0}"

export PATH="/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin:$PATH"

mkdir -p "$LOG_DIR" "$REPORT_DIR"

log() {
  printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S %z')" "$*"
}

cleanup() {
  rmdir "$LOCK_DIR" 2>/dev/null || true
}

if ! mkdir "$LOCK_DIR" 2>/dev/null; then
  log "another codex iteration is already running; skip"
  exit 0
fi
trap cleanup EXIT

cd "$REPO_DIR"

if [ ! -x "$CODEX_BIN" ]; then
  log "codex binary not found or not executable: $CODEX_BIN"
  exit 1
fi

timestamp="$(date '+%Y%m%d-%H%M%S')"
report_file="$REPORT_DIR/iteration-$timestamp.md"
final_file="$REPORT_DIR/iteration-$timestamp-final.md"

log "scheduled codex iteration started"
log "report file: $report_file"

{
  echo "# Codex Scheduled Feature Iteration"
  echo
  echo "- Repository: $REPO_DIR"
  echo "- Started At: $(date '+%Y-%m-%d %H:%M:%S %z')"
  echo "- Codex: $("$CODEX_BIN" --version 2>/dev/null || echo "$CODEX_BIN")"
  echo
} > "$report_file"

if [ "$ALLOW_DIRTY_WORKTREE" != "1" ] && [ -n "$(git status --porcelain)" ]; then
  {
    echo "## Skipped"
    echo
    echo "Working tree has uncommitted changes. The scheduled iteration did not run to avoid stacking autonomous edits."
    echo
    echo '```'
    git status --short
    echo '```'
  } >> "$report_file"
  log "working tree is dirty; skip. Set ALLOW_DIRTY_WORKTREE=1 for a manual run if needed"
  exit 0
fi

iteration_prompt='你是这个开源 AI Native ITSM 项目的资深全栈工程师。请在本地仓库中推进一次小步、可审查的系统功能迭代开发。

项目定位：
- 国内企业级 ITSM，对标 ServiceNow。
- 覆盖 ITIL v3 流程、CMDB、工单、事件、问题、变更、服务目录、知识库、BPMN 工作流、SLA、RAG/AI。
- 目标是 AI Native ITSM，后续要支持飞书、企业微信、钉钉、连接器市场、skill 市场、插件市场和 CLI。

执行规则：
1. 先阅读 AGENTS.md、README、相关 specs/prd，以及当前代码结构，选择一个小而明确的功能迭代点。
2. 优先选择能提升核心产品能力或工程质量的改动，例如 DTO/API 一致性、权限/租户隔离、流程模板、CMDB 能力、连接器/插件扩展点、AI 辅助能力、CLI 能力、测试补齐。
3. 控制改动范围。不要做大规模重构，不要修改无关文件，不要回滚用户已有改动。
4. 不要自动 git commit、不要 push、不要创建远端分支。
5. 必须尽量运行相关验证命令；如果验证失败或环境缺失，请记录原因和下一步。
6. 最终输出必须包含：本次选择的迭代点、修改文件、验证结果、遗留风险、建议人工 review 的重点。'

"$CODEX_BIN" -a never exec \
  --cd "$REPO_DIR" \
  --sandbox workspace-write \
  --output-last-message "$final_file" \
  "$iteration_prompt" >> "$report_file" 2>&1

{
  echo
  echo "## Final Summary"
  echo
  if [ -f "$final_file" ]; then
    cat "$final_file"
  else
    echo "Codex did not write a final summary file."
  fi
  echo
  echo "- Finished At: $(date '+%Y-%m-%d %H:%M:%S %z')"
  echo
  echo "## Git Status After Iteration"
  echo
  echo '```'
  git status --short
  echo '```'
} >> "$report_file"

log "scheduled codex iteration completed"
