#!/usr/bin/env bash
# P0-1: Unicode 非标点检测
# 防止智能引号、全角标点等混入 TSX/TS 导致编译失败
# 用法: ./scripts/check-unicode-punctuation.sh

set -euo pipefail

cd "$(dirname "$0")/.."

echo "🔍 检查前端代码中的 Unicode 非标点字符..."

# 使用 grep -P (Perl regex) 匹配实际 Unicode 字符
# U+201C/U+201D: 左右双引号 ""
# U+2018/U+2019: 左右单引号 ''
# U+3001/U+3002: 中文逗号、句号
# U+FF0C: 全角逗号
# U+FF1A: 全角冒号
# U+FF1B: 全角分号

HITS=$(grep -rnP '[\x{201c}\x{201d}\x{2018}\x{2019}\x{3001}\x{3002}\x{ff0c}\x{ff1a}\x{ff1b}]' itsm-frontend/src/ 2>/dev/null || true)

if [ -n "$HITS" ]; then
    echo "❌ 发现 Unicode 非标点字符（智能引号、全角标点等）："
    echo ""
    echo "$HITS"
    echo ""
    echo "这些字符会导致 TypeScript 编译失败。"
    echo "请替换为 ASCII 等价字符："
    echo "  \" \" → \" (U+0022)"
    echo "  ' ' → ' (U+0027)"
    echo "  ，→ , (U+002C)"
    echo "  。→ . (U+002E)"
    echo "  ：→ : (U+003A)"
    echo "  ；→ ; (U+003B)"
    exit 1
fi

echo "✅ 未发现 Unicode 非标点字符"
exit 0
