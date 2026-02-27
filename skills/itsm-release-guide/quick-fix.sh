#!/bin/bash
# ITSM 项目快速修复脚本
# 使用方法：./skills/itsm-release-guide/quick-fix.sh

set -e

echo "🔧 ITSM 项目快速修复工具"
echo "========================"
echo ""

# 1. 清理依赖
echo "📦 清理依赖..."
cd itsm-frontend
rm -rf node_modules package-lock.json
npm install --legacy-peer-deps
cd ..

# 2. 移除 @ts-nocheck
echo "📝 移除 @ts-nocheck..."
find itsm-frontend/src -name "*.tsx" -o -name "*.ts" | while read file; do
    sed -i '/@ts-nocheck/d' "$file"
done

# 3. 类型检查
echo "✅ 运行类型检查..."
cd itsm-frontend
npx tsc --noEmit || echo "⚠️  类型检查有错误，请手动修复"
cd ..

# 4. 提交更改
echo "💾 提交更改..."
git add -A
git commit -m "fix: apply quick fixes from itsm-release-guide" || echo "⚠️  没有更改需要提交"

echo ""
echo "✅ 快速修复完成！"
echo ""
echo "下一步："
echo "1. 检查类型错误：cd itsm-frontend && npx tsc --noEmit"
echo "2. 本地构建：npm run build"
echo "3. 推送代码：git push origin main"
echo "4. 创建 Release: git tag v1.0.0 && git push origin v1.0.0"
