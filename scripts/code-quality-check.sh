#!/bin/bash

# ITSM系统代码质量检查脚本
# 检查前端、后端代码质量，包括规范、性能、安全性等方面

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_section() {
    echo -e "${PURPLE}=== $1 ===${NC}"
}

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# 检查结果统计
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0

# 检查函数
run_check() {
    local check_name="$1"
    local check_command="$2"
    
    log_info "Running: $check_name"
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    if eval "$check_command" >/dev/null 2>&1; then
        log_success "✓ $check_name"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        return 0
    else
        log_error "✗ $check_name"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        return 1
    fi
}

# 显示检查结果
show_summary() {
    echo
    log_section "代码质量检查总结"
    echo "总检查项: $TOTAL_CHECKS"
    echo -e "通过: ${GREEN}$PASSED_CHECKS${NC}"
    echo -e "失败: ${RED}$FAILED_CHECKS${NC}"
    
    local success_rate=$((PASSED_CHECKS * 100 / TOTAL_CHECKS))
    echo -e "成功率: ${CYAN}$success_rate%${NC}"
    
    if [ $FAILED_CHECKS -eq 0 ]; then
        log_success "🎉 所有检查项都通过了！"
        return 0
    else
        log_warning "⚠️  有 $FAILED_CHECKS 个检查项未通过，请修复后重试"
        return 1
    fi
}

# 前端代码质量检查
check_frontend_code() {
    log_section "前端代码质量检查"
    
    cd "$PROJECT_ROOT/itsm-frontend"
    
    # 检查TypeScript类型
    log_info "检查TypeScript类型..."
    if npm run type-check 2>/dev/null; then
        log_success "✓ TypeScript类型检查通过"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    else
        log_error "✗ TypeScript类型检查失败"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
    fi
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    # ESLint检查
    log_info "运行ESLint检查..."
    if npm run lint 2>/dev/null; then
        log_success "✓ ESLint检查通过"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    else
        log_error "✗ ESLint检查失败"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
    fi
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    # Prettier格式检查
    log_info "检查代码格式..."
    if npm run format:check 2>/dev/null; then
        log_success "✓ 代码格式检查通过"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    else
        log_error "✗ 代码格式检查失败"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
    fi
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    # 检查依赖安全漏洞
    log_info "检查依赖安全漏洞..."
    if npm audit --audit-level moderate 2>/dev/null | grep -q "found 0 vulnerabilities"; then
        log_success "✓ 依赖安全检查通过"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    else
        log_warning "⚠ 发现依赖安全问题，请运行 'npm audit fix' 修复"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
    fi
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    # 检查bundle大小
    log_info "检查bundle大小..."
    if npm run build:analyze 2>/dev/null; then
        log_success "✓ Bundle分析完成"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    else
        log_warning "⚠ Bundle分析失败"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
    fi
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
}

# 后端代码质量检查
check_backend_code() {
    log_section "后端代码质量检查"
    
    cd "$PROJECT_ROOT/itsm-backend"
    
    # Go格式检查
    run_check "Go格式检查" "gofmt -l . | wc -l | grep -q '^0$'"
    
    # Go vet检查
    run_check "Go vet检查" "go vet ./..."
    
    # Go lint检查
    if command -v golangci-lint >/dev/null 2>&1; then
        run_check "golangci-lint检查" "golangci-lint run"
    else
        log_warning "golangci-lint未安装，跳过此项检查"
    fi
    
    # 测试覆盖率检查
    log_info "运行测试覆盖率检查..."
    if go test -v -coverprofile=coverage.out ./... >/dev/null 2>&1; then
        coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        if (( $(echo "$coverage >= 60" | bc -l) )); then
            log_success "✓ 测试覆盖率 $coverage% (≥60%)"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
        else
            log_error "✗ 测试覆盖率 $coverage% (<60%)"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
        fi
        TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
        rm -f coverage.out
    else
        log_error "✗ 测试运行失败"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    fi
    
    # 检查Go依赖
    if command -v go-mod-tidy >/dev/null 2>&1; then
        run_check "Go依赖检查" "go-mod-tidy --check"
    fi
    
    # 安全检查
    if command -v gosec >/dev/null 2>&1; then
        log_info "运行安全检查..."
        if gosec ./... 2>/dev/null; then
            log_success "✓ 安全检查通过"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
        else
            log_warning "⚠ 发现安全问题，请检查输出"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
        fi
        TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    fi
    
    # P2-06: 大文件检查 (>500 行警告)
    log_info "检查大文件 (>500 行警告)..."
    local large_files=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./ent/*" | while read f; do
        lines=$(wc -l < "$f")
        if [ "$lines" -gt 500 ]; then
            echo "$f ($lines lines)"
        fi
    done)
    if [ -z "$large_files" ]; then
        log_success "✓ 无超大文件 (>500 行)"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    else
        log_warning "⚠ 发现超大文件 (>500 行)，建议拆分:\n$large_files"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
    fi
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    # P2-07: 函数长度检查 (>50 行警告)
    log_info "检查长函数 (>50 行警告)..."
    local long_funcs=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./ent/*" -not -path "./.git/*" | while read f; do
        awk '
        /^[[:space:]]*func[[:space:]]+/ {
            func_name = $0
            gsub(/^[[:space:]]*func[[:space:]]+/, "", func_name)
            gsub(/\(.*/, "", func_name)
            gsub(/\{.*/, "", func_name)
            gsub(/[[:space:]]+$/, "", func_name)
            start_line = NR
            brace_count = 0
            in_func = 1
        }
        in_func && /\{/ { brace_count += gsub(/\{/, "{") }
        in_func && /\}/ { brace_count -= gsub(/\}/, "}") }
        in_func && brace_count == 0 {
            func_len = NR - start_line + 1
            if (func_len > 50) {
                printf "%s:%s (%d lines)\n", FILENAME, func_name, func_len
            }
            in_func = 0
        }
        ' "$f"
    done | head -20)
    if [ -z "$long_funcs" ]; then
        log_success "✓ 无超长函数 (>50 行)"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    else
        log_warning "⚠ 发现超长函数 (>50 行)，建议拆分:\n$long_funcs"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
    fi
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
}

# 性能检查
check_performance() {
    log_section "性能检查"
    
    # 检查前端性能
    cd "$PROJECT_ROOT/itsm-frontend"
    
    # Lighthouse检查
    if command -v lighthouse >/dev/null 2>&1; then
        log_info "运行Lighthouse性能检查..."
        if lighthouse http://localhost:3000 --output=json --output-path=./lighthouse-report.json --chrome-flags="--headless" >/dev/null 2>&1; then
            if [ -f "./lighthouse-report.json" ]; then
                performance=$(cat ./lighthouse-report.json | jq '.categories.performance.score * 100')
                if (( $(echo "$performance >= 80" | bc -l) )); then
                    log_success "✓ Lighthouse性能评分 $performance (≥80)"
                    PASSED_CHECKS=$((PASSED_CHECKS + 1))
                else
                    log_warning "⚠ Lighthouse性能评分 $performance (<80)"
                    FAILED_CHECKS=$((FAILED_CHECKS + 1))
                fi
                TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
                rm -f ./lighthouse-report.json
            fi
        fi
    else
        log_warning "Lighthouse未安装，跳过性能检查"
    fi
    
    # 检查后端性能
    cd "$PROJECT_ROOT/itsm-backend"
    
    # 基准测试
    if [ -d "./bench" ] || ls ./*_test.go 1> /dev/null 2>&1; then
        log_info "运行基准测试..."
        if go test -bench=. -benchmem ./... >/dev/null 2>&1; then
            log_success "✓ 基准测试完成"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
        else
            log_warning "⚠ 基准测试失败"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
        fi
        TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    fi
}

# 安全性检查
check_security() {
    log_section "安全性检查"
    
    # 检查敏感信息泄露
    log_info "检查敏感信息泄露..."
    
    # 检查前端
    cd "$PROJECT_ROOT/itsm-frontend"
    if grep -r "password\|secret\|token\|api_key" --include="*.ts" --include="*.tsx" --include="*.js" --include="*.jsx" src/ 2>/dev/null | grep -v "//.*password\|//.*secret\|//.*token\|//.*api_key" >/dev/null 2>&1; then
        log_warning "⚠ 前端代码中可能包含敏感信息"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
    else
        log_success "✓ 前端代码安全检查通过"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    fi
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    # 检查后端
    cd "$PROJECT_ROOT/itsm-backend"
    if grep -r "password\|secret\|token\|api_key" --include="*.go" . 2>/dev/null | grep -v "//.*password\|//.*secret\|//.*token\|//.*api_key" >/dev/null 2>&1; then
        log_warning "⚠ 后端代码中可能包含敏感信息"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
    else
        log_success "✓ 后端代码安全检查通过"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    fi
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    # 检查依赖漏洞
    log_info "检查依赖漏洞..."
    
    # 前端依赖检查
    cd "$PROJECT_ROOT/itsm-frontend"
    if npm audit --audit-level high 2>/dev/null | grep -q "found 0 vulnerabilities"; then
        log_success "✓ 前端依赖安全检查通过"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    else
        log_warning "⚠ 前端依赖存在高危漏洞"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
    fi
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    # 后端依赖检查
    cd "$PROJECT_ROOT/itsm-backend"
    if command -v govulncheck >/dev/null 2>&1; then
        if govulncheck ./... 2>/dev/null | grep -q "No vulnerabilities found"; then
            log_success "✓ 后端依赖安全检查通过"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
        else
            log_warning "⚠ 后端依赖存在漏洞"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
        fi
        TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    else
        log_warning "govulncheck未安装，跳过后端依赖安全检查"
    fi
}

# 文档检查
check_documentation() {
    log_section "文档检查"
    
    # 检查README文件
    if [ -f "$PROJECT_ROOT/README.md" ]; then
        run_check "README文件存在" "test -f $PROJECT_ROOT/README.md"
    else
        log_warning "⚠ 缺少README.md文件"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    fi
    
    # 检查API文档
    if [ -d "$PROJECT_ROOT/itsm-backend/docs" ] || [ -f "$PROJECT_ROOT/itsm-backend/docs/swagger.json" ]; then
        run_check "API文档存在" "test -d $PROJECT_ROOT/itsm-backend/docs -o -f $PROJECT_ROOT/itsm-backend/docs/swagger.json"
    else
        log_warning "⚠ 缺少API文档"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    fi
    
    # 检查代码注释覆盖率（Go）
    cd "$PROJECT_ROOT/itsm-backend"
    if command -v gocyclo >/dev/null 2>&1; then
        log_info "检查代码复杂度..."
        if gocyclo -over 15 . 2>/dev/null; then
            log_success "✓ 代码复杂度检查通过"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
        else
            log_warning "⚠ 发现高复杂度函数"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
        fi
        TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    fi
}

# Git提交规范检查
check_git_hooks() {
    log_section "Git规范检查"
    
    cd "$PROJECT_ROOT"
    
    # 检查commit-msg hook
    if [ -f ".git/hooks/commit-msg" ]; then
        run_check "Git commit-msg钩子已配置" "test -f .git/hooks/commit-msg"
    else
        log_warning "⚠ 建议配置commit-msg钩子"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    fi
    
    # 检查pre-commit hook
    if [ -f ".git/hooks/pre-commit" ]; then
        run_check "Git pre-commit钩子已配置" "test -f .git/hooks/pre-commit"
    else
        log_warning "⚠ 建议配置pre-commit钩子"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    fi
    
    # 检查最近的commit信息格式
    if git log --oneline -n 5 2>/dev/null | head -1 | grep -E "^(feat|fix|docs|style|refactor|test|chore)(\(.+\))?: " >/dev/null 2>&1; then
        log_success "✓ 最新commit格式符合规范"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    else
        log_warning "⚠ 建议遵循commit信息规范"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
    fi
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
}

# 主函数
main() {
    echo -e "${CYAN}"
    echo "  ____  _   _ ____  _____ ____     ____    _    _     _ "
    echo " |  _ \\| | | |  _ \\| ____|  _ \\   / ___|  / \\  | |   | |"
    echo " | |_) | | | | |_) |  _| | |_) | | |  _  / _ \\ | |   | |"
    echo " |  __/| |_| |  _ <| |___|  _ <  | |_| |/ ___ \\| |___|_|"
    echo " |_|    \\___/|_| \\_\\_____|_| \\_\\  \\____/_/   \\_\\_____| "
    echo "                                                       "
    echo -e "${NC}"
    
    log_info "开始ITSM系统代码质量检查..."
    echo "项目路径: $PROJECT_ROOT"
    echo
    
    # 检查前置条件
    if ! command -v node >/dev/null 2>&1; then
        log_error "Node.js未安装"
        exit 1
    fi
    
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go未安装"
        exit 1
    fi
    
    # 执行各项检查
    check_frontend_code
    check_backend_code
    check_performance
    check_security
    check_documentation
    check_git_hooks
    
    # 显示总结
    show_summary
}

# 如果直接运行此脚本
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi