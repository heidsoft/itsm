#!/usr/bin/env bash
# P0-1: Unicode 智能引号检测（代码上下文）
# 防止智能引号混入 TSX/TS 代码上下文导致编译失败
# 用法: ./scripts/check-unicode-punctuation.sh
#
# 中文文本中使用 ""'' 作为引号是标准排版，不是缺陷。
# 本脚本只检测不含 CJK 字符的行中出现的智能引号——
# 这些通常是 Word/网页复制时意外引入的，会导致 TypeScript 编译失败。

set -euo pipefail

cd "$(dirname "$0")/.."

echo "🔍 检查前端代码中的智能引号（排除中文文本上下文）..."

# 先匹配含智能引号的行，再排除含 CJK 字符的行（中文文本中的引号是合法的）
HITS=$(grep -rnP '[\x{201c}\x{201d}\x{2018}\x{2019}]' itsm-frontend/src/ 2>/dev/null \
    | grep -vP '[\x{4e00}-\x{9fff}\x{3400}-\x{4dbf}]' \
    || true)

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

echo "✅ 未发现代码上下文中的智能引号字符"
exit 0
