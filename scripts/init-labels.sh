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

# GitHub 的 label color 是不带 `#` 的 6 位 hex。数据表里为了可读性写成 `#1D76DB`，
# 但把 `#` 原样送进 API 会被判 `code: invalid, field: color` —— 同样是 HTTP 422，
# 与「名称已存在」的 422 无法靠状态码区分。历史上这里正是如此：422 一律当成
# 「已存在」→ 转 PATCH 更新 → 标签从没被创建过 → 404 → 打印 ⚠️ 后仍 exit 0，
# 47 个带斜杠/带前缀的标签从未真正同步，而工作流一直显示「成功」。
# 因此下面显式归一化颜色，并且 422 必须按响应体里的 errors[].code 分流。
strip_color() {
  local c="${1#"#"}"
  printf '%s' "$c"
}

for entry in "${LABELS[@]}"; do
  IFS='|' read -r name color desc <<< "$entry"
  color=$(strip_color "$color")

  if [[ "$DRY_RUN" == "true" ]]; then
    echo "[DRY-RUN] would ensure label: $name (color=$color)"
    continue
  fi

  # 标签名可能含 `/`（area/backend 等）。路径里的 `/` 会被当作分隔符，
  # 必须编码成 %2F，否则 PATCH/DELETE 都定位不到该资源（实测 404）。
  encoded_name=$(printf '%s' "$name" | sed 's|/|%2F|g')

  payload=$(printf '{"name":"%s","color":"%s","description":"%s"}' "$name" "$color" "$desc")
  resp=$(curl -sS -w '\n%{http_code}' -X POST \
    -H "Authorization: Bearer ${GH_TOKEN:-}" \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    "https://api.github.com/repos/${REPO}/labels" \
    -d "$payload" 2>/dev/null || printf '\n000')
  http_code=${resp##*$'\n'}
  body=${resp%$'\n'*}

  case "$http_code" in
    201)
      echo "✅ created: $name"
      CREATED=$((CREATED+1))
      ;;
    422)
      # 422 有两种含义，必须按 errors[].code 区分，不能一律当「已存在」：
      #   already_exists → 名称冲突，转到 PATCH 更新元数据
      #   其它（invalid/color 等）→ 是请求本身有问题，必须计失败并暴露
      if printf '%s' "$body" | grep -q '"code": *"already_exists"'; then
        patch_code=$(curl -sS -o /dev/null -w "%{http_code}" -X PATCH \
          -H "Authorization: Bearer ${GH_TOKEN:-}" \
          -H "Accept: application/vnd.github+json" \
          -H "X-GitHub-Api-Version: 2022-11-28" \
          "https://api.github.com/repos/${REPO}/labels/${encoded_name}" \
          -d "$(printf '{"color":"%s","description":"%s"}' "$color" "$desc")" 2>/dev/null || echo "000")
        if [[ "$patch_code" == "200" ]]; then
          echo "♻️  updated: $name"
          SKIPPED=$((SKIPPED+1))
        else
          echo "❌ exists but update failed (HTTP $patch_code): $name"
          FAILED=$((FAILED+1))
        fi
      else
        echo "❌ invalid payload (HTTP 422): $name — $(printf '%s' "$body" | tr -d '\n' | cut -c1-160)"
        FAILED=$((FAILED+1))
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
echo "Updated:  $SKIPPED (already existed, metadata refreshed)"
echo "Failed:   $FAILED"
echo "===================="

if [[ "$DRY_RUN" == "true" ]]; then
  echo "(Dry run, no changes made)"
fi

# 有失败必须以非零退出。此前无论失败多少都 exit 0，导致 label 同步长期静默
# 无效而工作流恒绿——「静默做错」比红更危险，这里改成有失败即失败。
if [[ "$FAILED" -gt 0 ]]; then
  echo "❌ $FAILED label(s) failed to sync"
  exit 1
fi

exit 0
