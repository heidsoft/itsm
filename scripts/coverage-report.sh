#!/usr/bin/env bash
#
# scripts/coverage-report.sh
#
# Unified coverage report entry point.
# Runs Go test coverage under itsm-backend/ and Jest coverage under itsm-frontend/,
# then invokes scripts/coverage-summarize.js to produce a single Markdown summary.
#
# Output:
#   - itsm-backend/coverage.out      (Go coverprofile)
#   - itsm-backend/coverage.html     (Go tool cover -html)
#   - itsm-frontend/coverage/        (Jest coverage artifacts; existing dir)
#   - coverage-summary.md            (unified Markdown summary)
#
# Usage:
#   ./scripts/coverage-report.sh
#
# Environment overrides:
#   BACKEND_TARGETS  — Go test targets (default: ./...)
#   FRONTEND_TARGETS — Jest runs if non-empty (default: run jest)
#   SKIP_BACKEND=1   — skip backend tests
#   SKIP_FRONTEND=1  — skip frontend tests
#   SKIP_SUMMARY=1   — skip summary generation
#   BASELINE_JSON    — path to baseline JSON for delta (optional)

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT_DIR}"

LOG_DIR="${ROOT_DIR}/output/coverage"
mkdir -p "${LOG_DIR}"

OUT_MD="${ROOT_DIR}/coverage-summary.md"
GO_OUT="${ROOT_DIR}/itsm-backend/coverage.out"
GO_HTML="${ROOT_DIR}/itsm-backend/coverage.html"
JEST_JSON="${ROOT_DIR}/itsm-frontend/coverage/coverage-summary.json"
BASELINE="${BASELINE_JSON:-${ROOT_DIR}/docs/testing/coverage-baseline.json}"

echo "[coverage-report] log dir: ${LOG_DIR}"
echo "[coverage-report] baseline: ${BASELINE}"

FAIL=0

# Backend (Go)
if [[ "${SKIP_BACKEND:-0}" != "1" ]]; then
  echo "[coverage-report] === backend: go test ./... -coverprofile ==="
  if [[ ! -f itsm-backend/go.mod ]]; then
    echo "[coverage-report] WARN itsm-backend/go.mod missing; skipping" >&2
  else
    pushd itsm-backend >/dev/null
      GOTOOLCHAIN="${GOTOOLCHAIN:-auto}" go test \
        ${BACKEND_TARGETS:-./...} \
        -coverprofile=coverage.out \
        -covermode=set \
        -count=1 \
        > "${LOG_DIR}/backend-coverage.log" 2>&1
      rc=$?
    popd >/dev/null
    if [[ $rc -ne 0 ]]; then
      echo "[coverage-report] backend go test FAILED (rc=$rc). Tail:"
      tail -40 "${LOG_DIR}/backend-coverage.log"
      FAIL=1
    else
      if [[ -f itsm-backend/coverage.out ]]; then
        (cd itsm-backend && go tool cover -func=coverage.out | tail -3 || true) \
          | tee "${LOG_DIR}/backend-coverage-summary.txt"
        (cd itsm-backend && go tool cover -html=coverage.out -o coverage.html) || true
        echo "[coverage-report] backend OK → ${GO_OUT}"
      else
        echo "[coverage-report] WARN backend go test produced no coverage.out" >&2
      fi
    fi
  fi
fi

# Frontend (Jest)
if [[ "${SKIP_FRONTEND:-0}" != "1" ]]; then
  echo "[coverage-report] === frontend: jest --ci --coverage ==="
  if [[ ! -d itsm-frontend ]]; then
    echo "[coverage-report] WARN itsm-frontend missing; skipping" >&2
  else
    pushd itsm-frontend >/dev/null
      if [[ ! -d node_modules ]]; then
        echo "[coverage-report] installing itsm-frontend deps first…"
        npm ci --no-audit --prefer-offline --no-progress >> "${LOG_DIR}/frontend-install.log" 2>&1 \
          || { echo "[coverage-report] WARN npm ci failed; jest may not run"; }
      fi
      npx --no-install jest --ci --coverage --watchAll=false --passWithNoTests \
        > "${LOG_DIR}/frontend-coverage.log" 2>&1
      rc=$?
    popd >/dev/null
    if [[ $rc -ne 0 ]]; then
      echo "[coverage-report] frontend jest FAILED (rc=$rc). Tail:"
      tail -40 "${LOG_DIR}/frontend-coverage.log"
      # Jest failure should not fail backend gate, but we still record it
      echo "[coverage-report] frontend reported failures, but coverage may still exist"
    else
      echo "[coverage-report] frontend OK → ${JEST_JSON}"
    fi
  fi
fi

# Summary
if [[ "${SKIP_SUMMARY:-0}" != "1" ]]; then
  echo "[coverage-report] === summary ==="
  node scripts/coverage-summarize.js \
    --go "${GO_OUT}" \
    --jest "${JEST_JSON}" \
    --out "${OUT_MD}" \
    --baseline "${BASELINE}" \
    || echo "[coverage-report] WARN summary generation failed"
  if [[ -f "${OUT_MD}" ]]; then
    head -30 "${OUT_MD}"
  fi
fi

if [[ $FAIL -ne 0 ]]; then
  echo "[coverage-report] OVERALL: FAIL (backend)" >&2
  exit 1
fi
echo "[coverage-report] OVERALL: OK"
