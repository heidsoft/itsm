#!/usr/bin/env bash
# P1 前端质量守卫集合
# 防止契约漂移、冗长空态、假数据渲染
# 用法: ./scripts/check-frontend-p1-guards.sh

set -euo pipefail

cd "$(dirname "$0")/.."

echo "========================================="
echo "P1 质量守卫"
echo "========================================="
echo ""

FAILED=0

echo "▶ [1/4] 列表信封契约测试"
if ! node scripts/check-list-envelope-contract.js; then
    FAILED=1
fi
echo ""

echo "▶ [2/4] snake_case 零新增扫描"
if ! bash scripts/check-snake-case-zero-new.sh; then
    FAILED=1
fi
echo ""

echo "▶ [3/4] 空状态文案守卫"
if ! bash scripts/check-empty-state-verbosity.sh; then
    FAILED=1
fi
echo ""

echo "▶ [4/4] 假数据渲染检测"
if ! node scripts/check-fake-data-rendering.js; then
    FAILED=1
fi
echo ""

echo "========================================="
if [ $FAILED -eq 0 ]; then
    echo "✅ P1 守卫全部通过"
    exit 0
else
    echo "❌ P1 守卫存在失败项"
    echo ""
    echo "说明: P1 守卫发现的违规可能是存量问题，需要逐步修复。"
    echo "新增代码必须遵守规则，存量问题记录到 issue 跟踪。"
    exit 1
fi
