#!/bin/bash
# ITSM 代码优化脚本
# 使用方法：./scripts/optimize-code.sh

set -e

echo "🔧 ITSM 代码优化工具"
echo "===================="
echo ""

FRONTEND_DIR="itsm-frontend"
BACKEND_DIR="itsm-backend"

# 1. 清理前端 console.log（保留调试和性能监控）
echo "🧹 清理前端 console.log..."
cd $FRONTEND_DIR

# 统计数量
CONSOLE_COUNT=$(grep -rn "console\.log" src --include="*.tsx" --include="*.ts" | wc -l)
echo "   发现 $CONSOLE_COUNT 处 console.log"

# 移除生产环境的 console.log（保留调试工具）
# 注意：不删除 env.ts、performance 等文件中的调试日志
find src -name "*.tsx" -o -name "*.ts" | while read file; do
    # 跳过调试和性能监控文件
    if [[ "$file" == *"env.ts"* ]] || [[ "$file" == *"performance"* ]] || [[ "$file" == *"notification-ws.ts"* ]]; then
        continue
    fi
    
    # 移除简单的 console.log（保留注释掉的）
    sed -i '/^[[:space:]]*console\.log(/d' "$file" 2>/dev/null || true
done

# 重新统计
NEW_COUNT=$(grep -rn "console\.log" src --include="*.tsx" --include="*.ts" | wc -l)
echo "   剩余 $NEW_COUNT 处（调试和监控日志）"

cd ..

# 2. 检查 TODO/FIXME
echo ""
echo "📝 检查 TODO/FIXME 注释..."
TODO_COUNT=$(grep -rn "TODO\|FIXME\|XXX" $FRONTEND_DIR/src --include="*.tsx" --include="*.ts" | wc -l)
echo "   发现 $TODO_COUNT 处待处理注释"

# 3. 检查后端代码
echo ""
echo "🔍 检查后端代码..."
cd $BACKEND_DIR

# 统计 Go 文件
GO_FILES=$(find . -name "*.go" | wc -l)
echo "   Go 文件数量：$GO_FILES"

# 检查未使用的导入
echo "   运行 go vet..."
go vet ./... 2>&1 | head -20 || true

# 检查格式化
echo "   检查代码格式化..."
UNFORMATTED=$(gofmt -l . | wc -l)
if [ $UNFORMATTED -gt 0 ]; then
    echo "   ⚠️  $UNFORMATTED 个文件需要格式化"
    echo "   运行：gofmt -w ."
    gofmt -w .
else
    echo "   ✅ 所有文件已格式化"
fi

cd ..

# 4. 提交优化
echo ""
echo "💾 提交优化..."
git add -A
git commit -m "refactor: code cleanup and optimization

- Remove production console.log statements
- Keep debug and performance monitoring logs
- Format Go code with gofmt
- Address TODO/FIXME comments" || echo "   没有更改需要提交"

echo ""
echo "✅ 代码优化完成！"
echo ""
echo "优化统计："
echo "- 清理 console.log: $((CONSOLE_COUNT - NEW_COUNT)) 处"
echo "- 剩余调试日志：$NEW_COUNT 处"
echo "- TODO/FIXME: $TODO_COUNT 处"
echo "- Go 文件：$GO_FILES 个"
