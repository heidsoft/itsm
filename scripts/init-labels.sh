#!/usr/bin/env bash
# init-labels.sh
# 幂等初始化 GitHub labels：已存在跳过，不存在则创建
# 用法：./scripts/init-labels.sh [--dry-run]
#
# 前置：需要 GH_TOKEN 或 gh CLI 已认证
#   export GH_TOKEN=ghp_xxxx
#   或 gh auth login

set -euo pipefail

REPO="${GITHUB_REPOSITORY:-}"
DRY_RUN="false"

# 解析参数
for arg in "$@"; do
  case "$arg" in
    --dry-run) DRY_RUN="true" ;;
    *) echo "Unknown arg: $arg"; exit 1 ;;
  esac
done

# 检测仓库
if [[ -z "$REPO" ]]; then
  if command -v gh >/dev/null 2>&1; then
    REPO=$(gh repo view --json nameWithOwner -q '.nameWithOwner' 2>/dev/null || true)
  fi
fi

if [[ -z "$REPO" ]]; then
  echo "❌ 无法自动检测仓库，请设置 GITHUB_REPOSITORY 环境变量"
  echo "   export GITHUB_REPOSITORY=owner/repo"
  exit 1
fi

echo "📦 Repository: $REPO"
echo "🔍 Dry run: $DRY_RUN"
echo

# 颜色 + 描述定义
# 格式：<name>|<color>|<description>
LABELS=(
# area/*
"area/backend|#1D76DB|Backend Go code (controller / service / middleware)"
"area/backend-entity|#1D76DB|Backend Ent ORM schema"
"area/backend-test|#1D76DB|Backend Go tests"
"area/frontend|#5319E7|Frontend Next.js / TypeScript"
"area/frontend-test|#5319E7|Frontend tests (Jest / Playwright)"
"area/ai|#FBCA04|AI / RAG / LLM features"
"area/cli|#0E8A16|itsm-cli tool"
"area/agent|#0E8A16|itsm-agent worker"
"area/skill|#0E8A16|itsm-skill plugins"
"area/infra|#BFD4F2|Docker / Nginx / monitoring"
"area/ci|#BFD4F2|GitHub Actions / workflows"
"area/docs|#0075CA|Documentation only"
"area/scripts|#BFD4F2|Helper scripts / Makefile"
# kind/*
"kind/feature|#5319E7|New feature"
"kind/bugfix|#D93F0B|Bug fix"
"kind/refactor|#5319E7|Code refactor (no behavior change)"
"kind/tests|#5319E7|Test-only changes"
"kind/docs|#0075CA|Documentation-only changes"
"kind/ci|#BFD4F2|CI / pipeline changes"
"kind/build|#BFD4F2|Build system / dependency changes"
"kind/dependencies|#0366D6|Dependabot dependency update"
# priority/*
"priority/p0|#B60205|Critical: 4h SLA"
"priority/p1|#D93F0B|High: 1d SLA"
"priority/p2|#FBCA04|Medium: 1w SLA"
"priority/p3|#0E8A16|Low: 2w+ SLA"
# status/*
"status/needs-triage|#FFFFFF|Awaiting triage"
"status/triaged|#0E8A16|Triaged, ready for development"
"status/in-progress|#FBCA04|In development"
"status/blocked|#B60205|Blocked by external dependency"
"status/duplicate|#CCCCCC|Duplicate of existing issue"
"status/wontfix|#FFFFFF|Will not be addressed"
"status/ready-to-merge|#0E8A16|Approved, awaiting merge"
# size/*
"size/XS|#C5DEF5|Extra small (< 10 LOC)"
"size/S|#C5DEF5|Small (10-49 LOC)"
"size/M|#BFD4F2|Medium (50-199 LOC)"
"size/L|#FBCA04|Large (200-499 LOC, consider splitting)"
"size/XL|#B60205|Extra large (500+ LOC, MUST split)"
# special
"breaking-change|#B60205|Breaking change (semver major bump)"
"dependencies|#0366D6|Pull requests that update a dependency"
"needs-triage|#FFFFFF|Needs maintainer triage"
"needs-review|#1D76DB|Awaiting reviewer"
"needs-design|#5319E7|Awaiting design proposal"
"needs-docs|#0075CA|Needs documentation update"
"security|#B60205|Security issue (private disclosure)"
"first-time-contributor|#7057FF|First contribution from this user"
"pinned|#FFFFFF|Prevents stale bot from closing"
"epic|#5319E7|Tracks a multi-PR feature"
)

CREATED=0
SKIPPED=0
FAILED=0

for entry in "${LABELS[@]}"; do
  IFS='|' read -r name color desc <<< "$entry"

  if [[ "$DRY_RUN" == "true" ]]; then
    echo "[DRY-RUN] would ensure label: $name"
    continue
  fi

  # GitHub API：POST 创建（已存在返回 422），PUT 更新
  http_code=$(curl -sS -o /dev/null -w "%{http_code}" -X POST \
    -H "Authorization: Bearer ${GH_TOKEN:-}" \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    "https://api.github.com/repos/${REPO}/labels" \
    -d "$(printf '{"name":"%s","color":"%s","description":"%s"}' "$name" "$color" "$desc")" 2>/dev/null || echo "000")

  case "$http_code" in
    201)
      echo "✅ created: $name"
      CREATED=$((CREATED+1))
      ;;
    422)
      # 已存在 — 通过 PATCH 更新描述和颜色
      patch_code=$(curl -sS -o /dev/null -w "%{http_code}" -X PATCH \
        -H "Authorization: Bearer ${GH_TOKEN:-}" \
        -H "Accept: application/vnd.github+json" \
        -H "X-GitHub-Api-Version: 2022-11-28" \
        "https://api.github.com/repos/${REPO}/labels/${name}" \
        -d "$(printf '{"color":"%s","description":"%s"}' "$color" "$desc")" 2>/dev/null || echo "000")
      if [[ "$patch_code" == "200" ]]; then
        echo "♻️  updated: $name"
        SKIPPED=$((SKIPPED+1))
      else
        echo "⚠️  exists but update failed (HTTP $patch_code): $name"
        SKIPPED=$((SKIPPED+1))
      fi
      ;;
    401|403)
      echo "❌ auth failed (HTTP $http_code). Check GH_TOKEN."
      exit 1
      ;;
    *)
      echo "❌ failed (HTTP $http_code): $name"
      FAILED=$((FAILED+1))
      ;;
  esac
done

echo
echo "===================="
echo "Created:  $CREATED"
echo "Skipped:  $SKIPPED (already existed, metadata updated)"
echo "Failed:   $FAILED"
echo "===================="

if [[ "$DRY_RUN" == "true" ]]; then
  echo "(Dry run, no changes made)"
fi

exit 0
