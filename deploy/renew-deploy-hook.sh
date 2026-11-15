#!/usr/bin/env bash
# acme.sh deploy hook for itsm.cloudmesh.top
# Runs automatically after a successful renewal. Pushes the new cert/key to the
# ECS (which has no outbound network, so the Mac is the ACME client) and reloads
# the host nginx. Keep this file free of secrets — SSH key auth is assumed.
set -euo pipefail

DOMAIN="${DOMAIN:-itsm.cloudmesh.top}"
ECS="root@39.97.54.216"
REMOTE_DIR="/etc/letsencrypt/live/${DOMAIN}"

# Only act for the ITSM domain.
if [ "$DOMAIN" != "itsm.cloudmesh.top" ]; then
  echo "[deploy-hook] skipping domain $DOMAIN"
  exit 0
fi

: "${FULLCHAIN_FILE:?}"
: "${KEY_FILE:?}"

echo "[deploy-hook] pushing renewed cert to ${ECS}:${REMOTE_DIR}"
scp -o StrictHostKeyChecking=no "$FULLCHAIN_FILE" "$ECS:$REMOTE_DIR/fullchain.pem"
scp -o StrictHostKeyChecking=no "$KEY_FILE" "$ECS:$REMOTE_DIR/privkey.pem"

ssh -o StrictHostKeyChecking=no "$ECS" \
  "chmod 600 $REMOTE_DIR/privkey.pem; chmod 644 $REMOTE_DIR/fullchain.pem; nginx -t && nginx -s reload"

echo "[deploy-hook] done for $DOMAIN"
exit 0
