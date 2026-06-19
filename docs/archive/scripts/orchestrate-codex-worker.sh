#!/usr/bin/env bash

set -euo pipefail

TASK_FILE="${1:?task file required}"
HANDOFF_FILE="${2:?handoff file required}"
STATUS_FILE="${3:?status file required}"

mkdir -p "$(dirname "$HANDOFF_FILE")" "$(dirname "$STATUS_FILE")"

timestamp() {
  date +"%Y-%m-%d %H:%M:%S %z"
}

cat >"$STATUS_FILE" <<EOF
state: in_progress
updated_at: $(timestamp)
current_focus: $(basename "$TASK_FILE")
blockers:
EOF

cat >"$HANDOFF_FILE" <<EOF
## Summary

Worker started at $(timestamp).

## Files Changed

-

## Verification

-

## Risks

-

## Next Step

-
EOF

echo "Task file: $TASK_FILE"
echo "Handoff file: $HANDOFF_FILE"
echo "Status file: $STATUS_FILE"
echo
echo "Worker prompt:"
echo "--------------------"
cat "$TASK_FILE"
