#!/usr/bin/env bash
# P0-1: Unicode 智能引号检测
# 防止智能引号混入 TSX/TS 导致编译失败
# 用法: ./scripts/check-unicode-punctuation.sh
#
# 注意：中文标点（，。：；）在中文 UI 字符串中是正确的，不在检测范围内。
# 只检测智能引号（""''），这些字符在任何代码上下文中都是错误的。

set -euo pipefail

cd "$(dirname "$0")/.."

echo "🔍 检查前端代码中的智能引号字符..."

# 使用 grep -P (Perl regex) 匹配实际 Unicode 字符
# U+201C/U+201D: 左右双引号 ""
# U+2018/U+2019: 左右单引号 ''
# 这些字符通常是从 Word/网页复制时意外引入的，会导致 TypeScript 编译失败。

HITS=$(grep -rnP '[\x{201c}\x{201d}\x{2018}\x{2019}]' itsm-frontend/src/ 2>/dev/null || true)

if [ -n "$HITS" ]; then
    echo "❌ 发现智能引号字符（通常从 Word/网页复制时引入）："
    echo ""
    echo "$HITS"
    echo ""
    echo "请替换为 ASCII 等价字符："
    echo "  \" \" → \" (U+0022)"
    echo "  ' ' → ' (U+0027)"
    exit 1
fi

echo "✅ 未发现智能引号字符"
exit 0
