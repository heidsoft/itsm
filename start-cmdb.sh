#!/bin/bash

# ITSM CMDB 启动脚本
# 用途: 快速启动ITSM CMDB系统

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# 检查Docker和Docker Compose
check_prerequisites() {
    log_info "检查系统依赖..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker未安装，请先安装Docker"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose未安装，请先安装Docker Compose"
        exit 1
    fi
    
    log_success "系统依赖检查通过"
}

# 创建必要的目录
create_directories() {
    log_info "创建必要的目录..."
    
    mkdir -p logs
    mkdir -p ssl
    mkdir -p monitoring
    mkdir -p nginx/conf.d
    mkdir -p scripts
    
    log_success "目录创建完成"
}

# 生成配置文件
generate_configs() {
    log_info "生成配置文件..."
    
    # 生成Nginx配置
    cat > nginx/nginx.conf << 'EOF'
user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';
    
    access_log /var/log/nginx/access.log main;
    
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;
    
    gzip on;
    gzip_vary on;
    gzip_min_length 10240;
    gzip_proxied expired no-cache no-store private must-revalidate auth;
    gzip_types text/plain text/css text/xml text/javascript application/x-javascript application/xml+rss application/json;
    
    include /etc/nginx/conf.d/*.conf;
}
EOF

    # 生成站点配置
    cat > nginx/conf.d/default.conf << 'EOF'
upstream frontend {
    server itsm-frontend:3000;
}

upstream backend {
    server itsm-cmdb-backend:8080;
}

server {
    listen 80;
    server_name localhost;
    
    # 前端路由
    location / {
        proxy_pass http://frontend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    # API路由
    location /api/ {
        proxy_pass http://backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    # 健康检查
    location /health {
        proxy_pass http://backend/health;
    }
}
EOF

    # 生成Prometheus配置
    cat > monitoring/prometheus.yml << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  # - "first_rules.yml"
  # - "second_rules.yml"

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'cmdb-backend'
    static_configs:
      - targets: ['itsm-cmdb-backend:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres:5432']
    scrape_interval: 30s

  - job_name: 'redis'
    static_configs:
      - targets: ['redis:6379']
    scrape_interval: 30s
EOF

    # 生成数据库初始化脚本
    cat > scripts/init-db.sql << 'EOF'
-- 创建CMDB数据库
CREATE DATABASE itsm_cmdb;

-- 创建用户和权限
CREATE USER cmdb_user WITH PASSWORD 'cmdb_password';
GRANT ALL PRIVILEGES ON DATABASE itsm_cmdb TO cmdb_user;

-- 连接到CMDB数据库
\c itsm_cmdb;

-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- 设置时区
SET timezone = 'Asia/Shanghai';
EOF

    log_success "配置文件生成完成"
}

# 构建和启动服务
start_services() {
    log_info "构建和启动服务..."
    
    # 构建镜像
    log_info "构建Docker镜像..."
    docker-compose build --no-cache
    
    # 启动服务
    log_info "启动服务..."
    docker-compose up -d
    
    # 等待服务启动
    log_info "等待服务启动..."
    sleep 30
    
    # 检查服务状态
    check_services
}

# 检查服务状态
check_services() {
    log_info "检查服务状态..."
    
    services=("postgres" "redis" "itsm-cmdb-backend" "itsm-frontend" "itsm-nginx")
    
    for service in "${services[@]}"; do
        if docker-compose ps | grep -q "$service.*Up"; then
            log_success "$service 服务运行正常"
        else
            log_error "$service 服务启动失败"
            docker-compose logs "$service"
        fi
    done
}

# 显示访问信息
show_access_info() {
    log_success "ITSM CMDB系统启动完成！"
    echo ""
    echo "访问地址:"
    echo "  前端界面: http://localhost"
    echo "  API接口:  http://localhost/api"
    echo "  健康检查: http://localhost/health"
    echo "  Grafana:  http://localhost:3001 (admin/admin)"
    echo "  Prometheus: http://localhost:9090"
    echo ""
    echo "数据库连接:"
    echo "  Host: localhost"
    echo "  Port: 5432"
    echo "  Database: itsm_cmdb"
    echo "  Username: postgres"
    echo "  Password: password"
    echo ""
    echo "Redis连接:"
    echo "  Host: localhost"
    echo "  Port: 6379"
    echo ""
    echo "日志查看:"
    echo "  docker-compose logs -f [service_name]"
    echo ""
    echo "停止服务:"
    echo "  docker-compose down"
}

# 停止服务
stop_services() {
    log_info "停止服务..."
    docker-compose down
    log_success "服务已停止"
}

# 清理数据
clean_data() {
    log_warning "这将删除所有数据，包括数据库数据！"
    read -p "确认继续？(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "清理数据..."
        docker-compose down -v
        docker system prune -f
        log_success "数据清理完成"
    else
        log_info "取消清理操作"
    fi
}

# 显示帮助信息
show_help() {
    echo "ITSM CMDB 启动脚本"
    echo ""
    echo "用法: $0 [命令]"
    echo ""
    echo "命令:"
    echo "  start     启动所有服务 (默认)"
    echo "  stop      停止所有服务"
    echo "  restart   重启所有服务"
    echo "  status    查看服务状态"
    echo "  logs      查看服务日志"
    echo "  clean     清理所有数据"
    echo "  help      显示帮助信息"
    echo ""
}

# 主函数
main() {
    case "${1:-start}" in
        "start")
            check_prerequisites
            create_directories
            generate_configs
            start_services
            show_access_info
            ;;
        "stop")
            stop_services
            ;;
        "restart")
            stop_services
            sleep 5
            start_services
            show_access_info
            ;;
        "status")
            docker-compose ps
            ;;
        "logs")
            docker-compose logs -f "${2:-}"
            ;;
        "clean")
            clean_data
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            log_error "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
