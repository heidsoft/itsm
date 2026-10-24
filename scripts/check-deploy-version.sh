#!/bin/bash
# check-deploy-version.sh - 部署版本一致性门禁（P0-3）
#
# 背景：.env.prod 的 VERSION 是部署镜像标签，曾与 git tag 脱钩（v1.6.20/21/22/25
# 混用），导致 `git log v1.6.22..HEAD` 静默无输出、增量核验失据。本脚本把
# 「部署标签必须等于最近 git tag」固化为可执行检查。
#
# 用法:
#   ./scripts/check-deploy-version.sh                    # 校验 .env.prod VERSION == 最近 tag
#   ./scripts/check-deploy-version.sh --expect v1.6.20   # 显式指定期望版本（CI 用）
#   ./scripts/check-deploy-version.sh --env-file .env.staging
#
# 退出码: 0 一致 / 1 不一致 / 2 环境错误
#
# 建议：在 make build-parallel / 部署脚本入口调用，防漂移再发。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

ENV_FILE="${REPO_ROOT}/.env.prod"
EXPECT=""
while [ $# -gt 0 ]; do
  case "$1" in
    --expect) EXPECT="$2"; shift 2 ;;
    --env-file) ENV_FILE="$2"; shift 2 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

RED='\033[0;31m'; GREEN='\033[0;32m'; NC='\033[0m'
log_ok()  { echo -e "${GREEN}[VERSION]${NC} $1"; }
log_err() { echo -e "${RED}[VERSION]${NC} $1" >&2; }

# 1) 期望版本：显式指定优先，否则取最近 git tag
if [ -n "$EXPECT" ]; then
  EXPECTED="$EXPECT"
else
  EXPECTED="$(git -C "$REPO_ROOT" describe --tags --abbrev=0 2>/dev/null || true)"
  if [ -z "$EXPECTED" ]; then
    log_err "repo has no git tag; use --expect"
    exit 2
  fi
fi

# 2) 部署标签：从 env 文件读 VERSION
if [ ! -f "$ENV_FILE" ]; then
  log_err "env file not found: $ENV_FILE"
  exit 2
fi
DEPLOY_VERSION="$(grep -E '^VERSION=' "$ENV_FILE" | head -1 | cut -d= -f2 | tr -d '[:space:]')"
if [ -z "$DEPLOY_VERSION" ]; then
  log_err "VERSION not set in $ENV_FILE"
  exit 2
fi

# 3) 比对
if [ "$DEPLOY_VERSION" = "$EXPECTED" ]; then
  log_ok "deploy tag matches release tag: $DEPLOY_VERSION"
  exit 0
fi

log_err "version drift: $ENV_FILE VERSION=$DEPLOY_VERSION but latest git tag is $EXPECTED"
log_err "impact: incremental diff uses wrong base (git log v1.6.22..HEAD silently empty), release evidence broken"
log_err "fix: set VERSION=$EXPECTED in $ENV_FILE (deploy tag must come from git describe, never hand-incremented)"
exit 1
