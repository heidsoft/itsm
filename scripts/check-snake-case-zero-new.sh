#!/usr/bin/env bash
# P1-6: snake_case 零新增扫描
# 防止后端 DTO 新增 snake_case JSON tag
# 用法: ./scripts/check-snake-case-zero-new.sh
#
# 规则: 新增的 Go 代码不得在 json/form/query tag 中使用 snake_case
# 例外: 数据库列名、SQL、Ent schema storage key、外部协议适配器

set -euo pipefail

cd "$(dirname "$0")/.."

echo "🔍 检查后端代码是否新增 snake_case 字段..."

# 检查 git diff 中新增的行
# 匹配模式: json:"xxx_yyy" 或 form:"xxx_yyy" 或 query:"xxx_yyy"
# 注意: 只在有 git 变更时检查，避免全量扫描
if git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
    if ! git diff --unified=0 -- 'itsm-backend/**/*.go' ':!itsm-backend/ent/generated/**' 2>/dev/null | \
        grep -E '^\+.*(json|form|query):\"[a-z0-9]+_[a-z0-9]+\"' | \
        grep -v '^\+\+\+' | \
        grep -v '//.*snake_case.*allowed' > /tmp/snake_case_hits.txt 2>/dev/null; then
        # grep returns 1 when no matches, which is success for us
        echo "✅ 未发现新增 snake_case 字段"
        rm -f /tmp/snake_case_hits.txt
        exit 0
    fi
else
    echo "⚠️  非 git 仓库，跳过 snake_case 检查"
    exit 0
fi

if [ -s /tmp/snake_case_hits.txt ]; then
    echo "❌ 发现新增 snake_case 字段："
    echo ""
    cat /tmp/snake_case_hits.txt
    echo ""
    echo "后端 DTO 字段应使用 camelCase："
    echo "  ❌ json:\"assignee_id\""
    echo "  ✅ json:\"assigneeId\""
    echo ""
    echo "例外情况（需在代码旁注明）："
    echo "  - 数据库列名、SQL、Ent schema storage key"
    echo "  - 外部协议适配器（如 Feishu/DingTalk webhook）"
    echo "  - 在行尾添加注释: // snake_case allowed: <原因>"
    rm -f /tmp/snake_case_hits.txt
    exit 1
fi

rm -f /tmp/snake_case_hits.txt
echo "✅ 未发现新增 snake_case 字段"
exit 0
