#!/usr/bin/env bash
# P1-7: 前端空状态文案守卫
# 防止再出现"当前没有X数据，点击下方按钮创建第一个X"这类长文案
# 用法: ./scripts/check-empty-state-verbosity.sh

set -euo pipefail

cd "$(dirname "$0")/.."

echo "🔍 检查前端空状态文案是否过于冗长..."

# 搜索模式: 引导用户创建的长文案
PATTERNS=(
    '点击下方按钮'
    '创建第一个'
    '立即创建'
    '快去创建'
    '马上创建'
)

HITS=""
for PATTERN in "${PATTERNS[@]}"; do
    # 排除:
    # - 注释和文档
    # - i18n 翻译文件（源头，不是渲染）
    # - onboarding/wizard（有意引导）
    # - 测试文件
    RESULT=$(grep -rnF "$PATTERN" itsm-frontend/src/ 2>/dev/null \
        | grep -v '^\s*//' \
        | grep -v '^\s*\*' \
        | grep -v 'translations\.ts' \
        | grep -v 'onboarding' \
        | grep -v '__tests__' \
        | grep -v '\.test\.' \
        || true)
    if [ -n "$RESULT" ]; then
        HITS="${HITS}${RESULT}\n"
    fi
done

if [ -n "$HITS" ]; then
    echo "❌ 发现冗长的空状态文案："
    echo ""
    echo -e "$HITS"
    echo ""
    echo "这些文案信息密度低，应精简为'暂无X数据'。"
    echo "示例："
    echo "  ❌ 当前没有工单数据，点击下方按钮创建第一个工单"
    echo "  ✅ 暂无工单数据"
    exit 1
fi

echo "✅ 未发现冗长空状态文案"
exit 0
