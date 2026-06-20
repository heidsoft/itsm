#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="/Users/heidsoft/Downloads/research/itsm"
CC_SWITCH_DB="$HOME/.cc-switch/cc-switch.db"
CC_SWITCH_SETTINGS="$HOME/.cc-switch/settings.json"
CC_SWITCH_LOG="$HOME/.cc-switch/logs/cc-switch.log"
CC_SWITCH_APP="/Applications/CC Switch.app"
LOOP_PLIST="$HOME/Library/LaunchAgents/com.heidsoft.itsm.codex-loop.plist"
LOOP_LABEL="com.heidsoft.itsm.codex-loop"
LOG_DIR="$REPO_DIR/logs"
WATCHDOG_LOG="$LOG_DIR/codex-route-watchdog.log"
STATE_DIR="$REPO_DIR/.codex-loop"
STOP_FILE="$STATE_DIR/STOP"
OFFICIAL_PROVIDER="${OFFICIAL_PROVIDER:-codex-official}"
FALLBACK_PROVIDER="${FALLBACK_PROVIDER:-}"
ENABLE_AUTO_SWITCH="${ENABLE_AUTO_SWITCH:-1}"
OFFICIAL_RETRY_AFTER_SECONDS="${OFFICIAL_RETRY_AFTER_SECONDS:-21600}"
LOOP_NOT_BEFORE_EPOCH="${LOOP_NOT_BEFORE_EPOCH:-1781904254}"

export PATH="/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin:$PATH"

mkdir -p "$LOG_DIR" "$STATE_DIR"

log() {
  printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S %z')" "$*" >> "$WATCHDOG_LOG"
}

sqlite_value() {
  sqlite3 "$CC_SWITCH_DB" "$1" 2>/dev/null | head -1 || true
}

current_provider() {
  sqlite_value "select id from providers where app_type='codex' and is_current=1 limit 1;"
}

fallback_provider() {
  if [ -n "$FALLBACK_PROVIDER" ]; then
    echo "$FALLBACK_PROVIDER"
    return
  fi

  local candidate
  candidate="$(sqlite_value "select id from providers where app_type='codex' and in_failover_queue=1 and id <> '$OFFICIAL_PROVIDER' order by sort_index asc limit 1;")"
  if [ -n "$candidate" ]; then
    echo "$candidate"
    return
  fi

  sqlite_value "select id from providers where app_type='codex' and id <> '$OFFICIAL_PROVIDER' order by sort_index asc limit 1;"
}

provider_name() {
  sqlite_value "select name from providers where app_type='codex' and id='$1' limit 1;"
}

cc_switch_running() {
  pgrep -f "/Applications/CC Switch.app/Contents/MacOS/cc-switch" >/dev/null 2>&1
}

ensure_cc_switch_running() {
  if cc_switch_running; then
    return
  fi

  if [ -d "$CC_SWITCH_APP" ]; then
    log "starting CC Switch app"
    open -gja "CC Switch" || true
    sleep 8
  fi
}

quota_signal_seen() {
  local pattern='quota|rate.?limit|429|insufficient_quota|usage limit|token.*(low|exceed|exhaust)|exceeded your current quota|too many requests'

  {
    tail -300 "$CC_SWITCH_LOG" 2>/dev/null || true
    tail -300 "$LOG_DIR/codex-loop.log" 2>/dev/null || true
    find "$LOG_DIR/codex-loop" -type f -name 'loop-*.md' -mtime -1 -print0 2>/dev/null \
      | xargs -0 tail -120 2>/dev/null || true
  } | rg -i "$pattern" >/dev/null 2>&1
}

switch_provider() {
  local target="$1"
  local name
  name="$(provider_name "$target")"

  if [ -z "$target" ] || [ -z "$name" ]; then
    log "target provider not found: $target"
    return 1
  fi

  log "switching Codex provider to $name ($target)"
  sqlite3 "$CC_SWITCH_DB" "update providers set is_current=case when id='$target' and app_type='codex' then 1 else 0 end where app_type='codex';"

  if [ -f "$CC_SWITCH_SETTINGS" ]; then
    plutil -replace currentProviderCodex -string "$target" "$CC_SWITCH_SETTINGS" 2>/dev/null || true
    plutil -replace enableFailoverToggle -bool true "$CC_SWITCH_SETTINGS" 2>/dev/null || true
    plutil -replace enableLocalProxy -bool true "$CC_SWITCH_SETTINGS" 2>/dev/null || true
  fi

  sqlite3 "$CC_SWITCH_DB" "update proxy_config set proxy_enabled=1, enabled=1, auto_failover_enabled=1, updated_at=datetime('now') where app_type='codex';" 2>/dev/null || true
  ensure_cc_switch_running
}

loop_loaded() {
  launchctl print "gui/$(id -u)/$LOOP_LABEL" >/dev/null 2>&1
}

load_loop_if_needed() {
  if loop_loaded; then
    return
  fi

  if [ -f "$LOOP_PLIST" ]; then
    log "loading codex loop launch agent"
    launchctl bootstrap "gui/$(id -u)" "$LOOP_PLIST" 2>/dev/null || true
  fi
}

restart_loop_if_needed() {
  rm -f "$STOP_FILE"
  load_loop_if_needed
}

main() {
  if [ "$ENABLE_AUTO_SWITCH" != "1" ]; then
    log "auto switch disabled"
    exit 0
  fi

  if [ ! -f "$CC_SWITCH_DB" ]; then
    log "CC Switch DB not found: $CC_SWITCH_DB"
    exit 0
  fi

  local now current fallback official_name fallback_name
  now="$(date '+%s')"
  current="$(current_provider)"
  fallback="$(fallback_provider)"
  official_name="$(provider_name "$OFFICIAL_PROVIDER")"
  fallback_name="$(provider_name "$fallback")"

  log "route check: current=${current:-none}, official=${official_name:-missing}, fallback=${fallback_name:-missing} (${fallback:-none})"

  if quota_signal_seen; then
    if [ -n "$fallback" ] && [ "$current" != "$fallback" ]; then
      switch_provider "$fallback" || exit 0
    else
      ensure_cc_switch_running
    fi
    restart_loop_if_needed
    exit 0
  fi

  if [ "$now" -lt "$LOOP_NOT_BEFORE_EPOCH" ]; then
    log "official token recovery window not reached; keep loop gated until $(date -r "$LOOP_NOT_BEFORE_EPOCH" '+%Y-%m-%d %H:%M:%S %z')"
    exit 0
  fi

  if [ "$current" = "$OFFICIAL_PROVIDER" ]; then
    restart_loop_if_needed
    exit 0
  fi

  # After the recovery window, prefer official unless recent quota signals are still present.
  switch_provider "$OFFICIAL_PROVIDER" || true
  restart_loop_if_needed
}

main "$@"
