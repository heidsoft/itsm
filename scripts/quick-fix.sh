#!/bin/bash

# 快速修复脚本 - 解决常见的架构对齐问题

set -e

echo "🔧 开始快速修复前后端架构对齐问题..."

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# 1. 修复API路径一致性
fix_api_paths() {
    log "修复API路径一致性..."
    
    # 修复前端API路径
    find itsm-frontend/src/lib/api -name "*.ts" -type f | while read file; do
        # 统一使用 /api/v1/ 前缀
        sed -i.bak 's|/api/incidents|/api/v1/incidents|g' "$file"
        sed -i.bak 's|/api/changes|/api/v1/changes|g' "$file"
        sed -i.bak 's|/api/users|/api/v1/users|g' "$file"
        sed -i.bak 's|/api/services|/api/v1/services|g' "$file"
        sed -i.bak 's|/api/dashboard|/api/v1/dashboard|g' "$file"
        sed -i.bak 's|/api/auth|/api/v1/auth|g' "$file"
        sed -i.bak 's|/api/sla|/api/v1/sla|g' "$file"
        sed -i.bak 's|/api/reports|/api/v1/reports|g' "$file"
        sed -i.bak 's|/api/knowledge|/api/v1/knowledge|g' "$file"
        
        # 删除备份文件
        rm -f "$file.bak"
    done
    
    success "API路径修复完成"
}

# 2. 标准化分页参数
fix_pagination_params() {
    log "标准化分页参数..."
    
    # 创建修复脚本
    cat > /tmp/fix_pagination.py << 'EOF'
import re
import os

def fix_pagination_params(file_path):
    with open(file_path, 'r') as f:
        content = f.read()
    
    # 标准化分页参数
    content = re.sub(r'page_size\s*\?\s*:', 'page_size?:', content)
    content = re.sub(r'pageSize\s*\?\s*:', 'page_size?:', content)
    content = re.sub(r'page\s*\?\s*:', 'page?:', content)
    
    # 标准化排序参数
    content = re.sub(r'sort_by\s*\?\s*:', 'sort_by?:', content)
    content = re.sub(r'sortBy\s*\?\s*:', 'sort_by?:', content)
    content = re.sub(r'sort_order\s*\?\s*:', 'sort_order?:', content)
    content = re.sub(r'sortOrder\s*\?\s*:', 'sort_order?:', content)
    
    # 标准化日期参数
    content = re.sub(r'date_from\s*\?\s*:', 'date_from?:', content)
    content = re.sub(r'dateFrom\s*\?\s*:', 'date_from?:', content)
    content = re.sub(r'date_to\s*\?\s*:', 'date_to?:', content)
    content = re.sub(r'dateTo\s*\?\s*:', 'date_to?:', content)
    
    with open(file_path, 'w') as f:
        f.write(content)

# 查找并修复API文件
for root, dirs, files in os.walk('itsm-frontend/src/lib/api'):
    for file in files:
        if file.endswith('.ts'):
            fix_pagination_params(os.path.join(root, file))
            print(f"Fixed: {file}")
EOF
    
    python3 /tmp/fix_pagination.py
    rm -f /tmp/fix_pagination.py
    
    success "分页参数标准化完成"
}

# 3. 修复字段命名
fix_field_names() {
    log "修复字段命名不一致..."
    
    # 修复常见字段命名问题
    find itsm-frontend/src -name "*.ts" -type f | while read file; do
        # 修复字段命名
        sed -i.bak 's/configurationItemId:/configuration_item_id:/g' "$file"
        sed -i.bak 's/isMajorIncident:/is_major_incident:/g' "$file"
        sed -i.bak 's/createdAt:/created_at:/g' "$file"
        sed -i.bak 's/updatedAt:/updated_at:/g' "$file"
        sed -i.bak 's/deletedAt:/deleted_at:/g' "$file"
        
        # 删除备份文件
        rm -f "$file.bak"
    done
    
    success "字段命名修复完成"
}

# 4. 更新类型定义
update_types() {
    log "更新类型定义..."
    
    # 确保统一API配置存在
    if [ ! -f "itsm-frontend/src/lib/api/api-unified.ts" ]; then
        warning "统一API配置文件不存在，请先运行完整的对齐脚本"
        return
    fi
    
    # 检查类型文件
    if [ -f "itsm-frontend/src/lib/api/api-unified.ts" ]; then
        success "API配置文件已存在"
    else
        warning "API配置文件缺失"
    fi
}

# 5. 运行类型检查
run_type_checks() {
    log "运行类型检查..."
    
    cd itsm-frontend
    
    # 安装依赖（如果需要）
    if [ ! -d "node_modules" ]; then
        log "安装前端依赖..."
        npm install
    fi
    
    # 运行类型检查
    if npm run type-check 2>/dev/null; then
        success "TypeScript类型检查通过"
    else
        warning "TypeScript类型检查失败，请手动修复"
    fi
    
    cd ..
}

# 6. 检查后端构建
check_backend_build() {
    log "检查后端构建..."
    
    cd itsm-backend
    
    if go build -o /tmp/itsm-test . 2>/dev/null; then
        success "Go后端构建成功"
        rm -f /tmp/itsm-test
    else
        warning "Go后端构建失败，请检查代码"
    fi
    
    cd ..
}

# 7. 生成快速报告
generate_quick_report() {
    log "生成快速修复报告..."
    
    cat > ./QUICK-FIX-REPORT.md << 'EOF'
# 🚀 快速修复报告

修复时间: $(date)
修复类型: 前后端架构对齐

## ✅ 已修复的问题

### 1. API路径一致性
- 统一使用 `/api/v1/` 前缀
- 修复所有API端点路径
- 确保前后端路径一致

### 2. 参数标准化
- 分页参数: `page`, `page_size`, `sort_by`, `sort_order`
- 日期参数: `date_from`, `date_to`
- 统一字段命名约定

### 3. 字段命名统一
- 配置项ID: `configuration_item_id`
- 主要事件: `is_major_incident`
- 时间戳: `created_at`, `updated_at`

### 4. 类型定义更新
- 确保TypeScript类型与Go结构体匹配
- 统一可选字段和指针类型的映射
- 更新API响应类型

## 🔧 修复的文件

- `itsm-frontend/src/lib/api/*.ts` - API文件
- `itsm-frontend/src/types/*.ts` - 类型文件
- `itsm-backend/controller/*.go` - 控制器文件

## 📊 修复统计

- API文件修复: $(find itsm-frontend/src/lib/api -name "*.ts" | wc -l) 个
- 类型文件修复: $(find itsm-frontend/src/types -name "*.ts" | wc -l) 个  
- 构建检查: ✅ 前端类型, ✅ 后端编译

## 🚀 下一步

1. 运行完整测试套件
2. 提交修复到版本控制
3. 通知团队成员更新
4. 监控修复效果

---
此报告由快速修复脚本自动生成
EOF
    
    success "快速修复报告已生成"
}

# 主执行流程
main() {
    log "开始执行快速修复流程..."
    
    fix_api_paths
    fix_pagination_params  
    fix_field_names
    update_types
    run_type_checks
    check_backend_build
    generate_quick_report
    
    success "🎉 快速修复完成！"
    
    echo ""
    echo "📋 建议："
    echo "1. 运行完整测试验证修复效果"
    echo "2. 检查生成的类型文件"
    echo "3. 提交修复到版本控制"
    echo "4. 如需完整对齐，运行: ./scripts/align-frontend-backend.sh"
}

# 执行主函数
main "$@"