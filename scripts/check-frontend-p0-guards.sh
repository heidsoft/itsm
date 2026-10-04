#!/usr/bin/env bash
# P0 前端质量守卫集合
# 防止编译失败、假功能入口、冗余引导文案回潮
# 用法: ./scripts/check-frontend-p0-guards.sh

set -euo pipefail

cd "$(dirname "$0")/.."

echo "========================================="
echo "P0 前端质量守卫"
echo "========================================="
echo ""

FAILED=0

echo "▶ [1/3] Unicode 非标点检测"
if ! bash scripts/check-unicode-punctuation.sh; then
    FAILED=1
fi
echo ""

echo "▶ [2/3] 占位文案零新增检测"
if ! bash scripts/check-placeholder-text.sh; then
    FAILED=1
fi
echo ""

echo "▶ [3/3] Alert 冗余描述检测"
if ! node scripts/check-alert-verbosity.js; then
    FAILED=1
fi
echo ""

echo "========================================="
if [ $FAILED -eq 0 ]; then
    echo "✅ P0 守卫全部通过"
    exit 0
else
    echo "❌ P0 守卫存在失败项"
    exit 1
fi
