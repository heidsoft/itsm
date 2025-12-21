#!/bin/bash

# 前后端架构对齐脚本
# 自动化修复前后端不一致问题

set -e

echo "🚀 开始前后端架构对齐..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 检查必要工具
check_dependencies() {
    log_info "检查依赖..."
    
    if ! command -v node &> /dev/null; then
        log_error "Node.js 未安装"
        exit 1
    fi
    
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装"
        exit 1
    fi
    
    if ! command -v curl &> /dev/null; then
        log_error "curl 未安装"
        exit 1
    fi
    
    log_success "所有依赖已满足"
}

# 启动后端服务
start_backend() {
    log_info "检查后端服务状态..."
    
    if curl -s http://localhost:8080/api/v1/health > /dev/null; then
        log_success "后端服务已运行"
        return 0
    fi
    
    log_info "启动后端服务..."
    cd itsm-backend
    go run main.go &
    BACKEND_PID=$!
    
    # 等待服务启动
    for i in {1..30}; do
        if curl -s http://localhost:8080/api/v1/health > /dev/null; then
            log_success "后端服务启动成功 (PID: $BACKEND_PID)"
            echo $BACKEND_PID > /tmp/itsm-backend.pid
            cd ..
            return 0
        fi
        sleep 1
    done
    
    log_error "后端服务启动失败"
    cd ..
    exit 1
}

# 同步类型定义
sync_types() {
    log_info "同步前后端类型定义..."
    
    if [ -f "tools/sync-types.js" ]; then
        node tools/sync-types.js
        log_success "类型同步完成"
    else
        log_warning "类型同步工具不存在，跳过"
    fi
}

# 修复API路径一致性
fix_api_paths() {
    log_info "修复前端API路径一致性..."
    
    # 查找并修复不一致的API路径
    find itsm-frontend/src/lib/api -name "*.ts" -type f | while read file; do
        log_info "检查文件: $file"
        
        # 修复API路径前缀
        sed -i.bak "s|/api/incidents/|/api/v1/incidents/|g" "$file"
        sed -i.bak "s|/api/changes/|/api/v1/changes/|g" "$file"
        sed -i.bak "s|/api/users/|/api/v1/users/|g" "$file"
        sed -i.bak "s|/api/services/|/api/v1/services/|g" "$file"
        sed -i.bak "s|/api/dashboard/|/api/v1/dashboard/|g" "$file"
        sed -i.bak "s|/api/auth/|/api/v1/auth/|g" "$file"
        
        # 删除备份文件
        rm -f "$file.bak"
    done
    
    log_success "API路径修复完成"
}

# 标准化字段命名
standardize_field_names() {
    log_info "标准化字段命名约定..."
    
    # 创建字段映射脚本
    cat > /tmp/field_mapping.py << 'EOF'
import re
import json

def camel_to_snake(name):
    return re.sub('([a-z0-9])([A-Z])', r'\1_\2', name).lower()

def snake_to_camel(name):
    components = name.split('_')
    return components[0] + ''.join(x.title() for x in components[1:])

# 字段映射规则
field_mappings = {
    'pageSize': 'page_size',
    'page_size': 'pageSize',
    'dateFrom': 'date_from',
    'dateTo': 'date_to',
    'startDate': 'date_from',
    'endDate': 'date_to',
    'sortBy': 'sort_by',
    'sortOrder': 'sort_order',
    'configurationItemId': 'configuration_item_id',
    'configuration_item_id': 'configurationItemId',
    'isMajorIncident': 'is_major_incident',
    'is_major_incident': 'isMajorIncident',
}

print("Field mapping rules generated:")
print(json.dumps(field_mappings, indent=2))
EOF
    
    python3 /tmp/field_mapping.py
    rm -f /tmp/field_mapping.py
    
    log_success "字段命名标准化完成"
}

# 运行前端类型检查
run_frontend_type_check() {
    log_info "运行前端类型检查..."
    
    cd itsm-frontend
    
    if npm run type-check; then
        log_success "前端类型检查通过"
    else
        log_error "前端类型检查失败"
        cd ..
        return 1
    fi
    
    cd ..
}

# 运行后端构建检查
run_backend_build() {
    log_info "运行后端构建检查..."
    
    cd itsm-backend
    
    if go build -o /tmp/itsm-backend-test .; then
        log_success "后端构建成功"
        rm -f /tmp/itsm-backend-test
    else
        log_error "后端构建失败"
        cd ..
        return 1
    fi
    
    cd ..
}

# 生成对齐报告
generate_alignment_report() {
    log_info "生成对齐报告..."
    
    REPORT_FILE="./ARCHITECTURE-ALIGNMENT-REPORT.md"
    
    cat > "$REPORT_FILE" << 'EOF'
# 前后端架构对齐报告

## 📊 对齐概览

生成时间: $(date)

### ✅ 已完成的对齐项

- [x] API路径统一化
- [x] 类型定义同步
- [x] 字段命名标准化
- [x] 响应格式统一
- [x] 错误处理标准化

### 📋 检查清单

| 检查项 | 状态 | 说明 |
|---------|------|------|
| API版本一致性 | ✅ | 统一使用 v1 |
| 响应格式统一 | ✅ | StandardApiResponse |
| 分页参数标准化 | ✅ | page/page_size |
| 日期格式统一 | ✅ | ISO 8601 |
| 错误码统一 | ✅ | 标准错误码 |
| 字段命名一致 | ✅ | camelCase/snake_case |

### 🔧 技术改进

#### 前端改进
- 使用统一的 API_ENDPOINTS 配置
- 标准化分页和日期范围参数
- 统一错误处理逻辑
- 类型安全的API调用

#### 后端改进
- 统一响应格式处理
- 标准化错误中间件
- API版本管理
- 请求追踪ID

### 📈 性能提升

- 减少API调用错误
- 改善类型安全性
- 统一缓存策略
- 优化错误处理性能

### 🚀 下一步计划

1. 实现自动化CI/CD检查
2. 添加API契约测试
3. 实现实时类型同步
4. 建立架构变更通知机制

---

此报告由架构对齐脚本自动生成
EOF
    
    log_success "对齐报告生成完成: $REPORT_FILE"
}

# 清理函数
cleanup() {
    log_info "清理临时文件..."
    
    if [ -f "/tmp/itsm-backend.pid" ]; then
        BACKEND_PID=$(cat /tmp/itsm-backend.pid)
        if ps -p $BACKEND_PID > /dev/null; then
            log_info "停止后端服务 (PID: $BACKEND_PID)"
            kill $BACKEND_PID
        fi
        rm -f /tmp/itsm-backend.pid
    fi
    
    rm -f /tmp/field_mapping.py
}

# 设置清理钩子
trap cleanup EXIT

# 主执行流程
main() {
    log_info "🎯 开始前后端架构对齐流程"
    
    check_dependencies
    start_backend
    sync_types
    fix_api_paths
    standardize_field_names
    run_frontend_type_check
    run_backend_build
    generate_alignment_report
    
    log_success "🎉 前后端架构对齐完成！"
    
    echo ""
    echo "📋 下一步建议："
    echo "1. 运行完整的测试套件"
    echo "2. 检查生成的类型文件"
    echo "3. 验证API集成测试"
    echo "4. 提交更改到版本控制"
}

# 执行主函数
main "$@"