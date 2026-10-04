#!/usr/bin/env bash
# P0-2: 占位文案零新增检测
# 防止"即将推出/敬请期待"等假功能入口泄漏到用户界面
# 用法: ./scripts/check-placeholder-text.sh

set -euo pipefail

cd "$(dirname "$0")/.."

echo "🔍 检查前端代码中的占位文案..."

# 搜索模式: 即将推出、敬请期待、coming soon、TBD 等
PATTERNS=(
    '即将推出'
    '敬请期待'
    'coming soon'
    'COMING SOON'
    '功能开发中'
    '暂未开放'
)

HITS=""
for PATTERN in "${PATTERNS[@]}"; do
    # 排除:
    # - 注释 (// 或 *)
    # - i18n 翻译文件 (translations.ts 是源头，不是渲染)
    # - throw new Error (内部错误，不直接展示给用户)
    RESULT=$(grep -rnF "$PATTERN" itsm-frontend/src/ 2>/dev/null \
        | grep -v '^\s*//' \
        | grep -v '^\s*\*' \
        | grep -v 'translations\.ts' \
        | grep -v 'throw new Error' \
        | grep -v 'console\.' \
        || true)
    if [ -n "$RESULT" ]; then
        HITS="${HITS}${RESULT}\n"
    fi
done

if [ -n "$HITS" ]; then
    echo "❌ 发现占位文案（假功能入口）："
    echo ""
    echo -e "$HITS"
    echo ""
    echo "这些文案会让用户看到未实现的功能入口。"
    echo "处理方式："
    echo "  1. 如果功能已实现：删除占位文案，接入真实数据"
    echo "  2. 如果功能未实现：从菜单/路由中移除入口"
    echo "  3. 如果是内部注释：确保不在 JSX 渲染路径中"
    exit 1
fi

echo "✅ 未发现占位文案"
exit 0
