#!/usr/bin/env bash
set -euo pipefail

# Simple static check: ensure protected routes in router/router.go include required middlewares
FILE="$(dirname "$0")/../router/router.go"
REQS=("AuthMiddleware" "TenantMiddleware" "RequestIDMiddleware" "AuditMiddleware")

missing=0
for req in "${REQS[@]}"; do
  if ! grep -q "$req" "$FILE"; then
    echo "[ERROR] $req not found in router/router.go"
    missing=1
  fi
done

if [[ $missing -eq 1 ]]; then
  echo "Required middlewares missing."
  exit 1
fi

echo "Middleware check passed."


