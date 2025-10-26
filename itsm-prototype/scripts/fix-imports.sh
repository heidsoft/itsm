#!/bin/bash

# ITSM 模块导入路径批量修复脚本
# 用于将相对路径导入统一改为路径别名导入

echo "🔧 开始修复模块导入路径..."

# 定义需要修复的文件模式
FILES_TO_FIX=(
  "src/app/admin/**/*.tsx"
  "src/app/**/*.tsx"
  "src/components/**/*.tsx"
)

# 定义路径映射规则
declare -A PATH_MAPPINGS=(
  ["../../lib/"]="@/lib/"
  ["../../../lib/"]="@/lib/"
  ["../../components/"]="@/components/"
  ["../../../components/"]="@/components/"
  ["../components/"]="@/components/"
  ["../../lib/api/"]="@/lib/api/"
  ["../../../lib/api/"]="@/lib/api/"
  ["../lib/api/"]="@/lib/api/"
  ["../../lib/services/"]="@/lib/services/"
  ["../../../lib/services/"]="@/lib/services/"
  ["../lib/services/"]="@/lib/services/"
  ["../../lib/hooks/"]="@/lib/hooks/"
  ["../../../lib/hooks/"]="@/lib/hooks/"
  ["../lib/hooks/"]="@/lib/hooks/"
  ["../../types/"]="@/types/"
  ["../../../types/"]="@/types/"
  ["../types/"]="@/types/"
)

# 修复函数
fix_imports() {
  local file="$1"
  local temp_file="${file}.tmp"
  
  echo "修复文件: $file"
  
  # 复制文件到临时文件
  cp "$file" "$temp_file"
  
  # 应用路径映射
  for old_path in "${!PATH_MAPPINGS[@]}"; do
    new_path="${PATH_MAPPINGS[$old_path]}"
    sed -i '' "s|from ['\"]${old_path}|from '${new_path}|g" "$temp_file"
    sed -i '' "s|from [\"']${old_path}|from \"${new_path}|g" "$temp_file"
  done
  
  # 检查是否有变化
  if ! diff -q "$file" "$temp_file" > /dev/null; then
    mv "$temp_file" "$file"
    echo "✅ 已修复: $file"
  else
    rm "$temp_file"
    echo "⏭️  无需修复: $file"
  fi
}

# 查找并修复所有文件
for pattern in "${FILES_TO_FIX[@]}"; do
  for file in $pattern; do
    if [ -f "$file" ]; then
      fix_imports "$file"
    fi
  done
done

echo "🎉 模块导入路径修复完成！"

# 验证修复结果
echo "🔍 验证修复结果..."
echo "检查是否还有相对路径导入:"
grep -r "from ['\"][.][.]/" src/app/ src/components/ 2>/dev/null | head -10 || echo "✅ 未发现相对路径导入"

echo "检查路径别名使用情况:"
grep -r "from ['\"]@/" src/app/ src/components/ 2>/dev/null | wc -l | xargs echo "✅ 已使用路径别名的导入数量:"
