#!/bin/bash

# ITSM PostgreSQL 17 安装脚本
# 适用于 AlmaLinux/Rocky Linux/CentOS 9+

set -e

echo "🚀 开始安装 PostgreSQL 17..."

# 1. 安装 PGDG 仓库
echo "📦 安装 PGDG 仓库..."
sudo dnf install -y https://download.postgresql.org/pub/repos/yum/reporpms/EL-10-x86_64/pgdg-redhat-repo-latest.noarch.rpm

# 2. 禁用默认 PostgreSQL 模块
echo "🔧 禁用默认 PostgreSQL 模块..."
sudo dnf -qy module disable postgresql

# 3. 清理并重建缓存
echo "🧹 清理并重建缓存..."
sudo dnf clean all
sudo dnf makecache

# 4. 安装 PostgreSQL 17
echo "📥 安装 PostgreSQL 17..."
sudo dnf install -y postgresql17-server postgresql17-contrib

# 5. 初始化数据库
echo "🗄️ 初始化数据库..."
sudo /usr/pgsql-17/bin/postgresql-17-setup initdb

# 6. 启动服务
echo "▶️ 启动 PostgreSQL 服务..."
sudo systemctl enable postgresql-17
sudo systemctl start postgresql-17

# 7. 检查状态
echo "📊 检查 PostgreSQL 状态..."
sudo systemctl status postgresql-17 --no-pager

# 8. 创建 ITSM 数据库和用户
echo "🔐 创建 ITSM 数据库和用户..."
sudo -u postgres psql -c "CREATE USER itsm WITH ENCRYPTED PASSWORD 'itsm_password_2026';"
sudo -u postgres psql -c "CREATE DATABASE itsm OWNER itsm;"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE itsm TO itsm;"
sudo -u postgres psql -c "ALTER DATABASE itsm OWNER TO itsm;"

# 9. 配置防火墙
echo "🔥 配置防火墙..."
if command -v firewall-cmd &> /dev/null; then
    sudo firewall-cmd --permanent --add-service=postgresql
    sudo firewall-cmd --reload
fi

# 10. 显示连接信息
echo ""
echo "✅ PostgreSQL 17 安装完成！"
echo ""
echo "📋 连接信息:"
echo "  主机：localhost"
echo "  端口：5432"
echo "  数据库：itsm"
echo "  用户：itsm"
echo "  密码：itsm_password_2026"
echo ""
echo "🔧 管理命令:"
echo "  启动：sudo systemctl start postgresql-17"
echo "  停止：sudo systemctl stop postgresql-17"
echo "  重启：sudo systemctl restart postgresql-17"
echo "  状态：sudo systemctl status postgresql-17"
echo ""
echo "📝 连接测试:"
echo "  psql -U itsm -d itsm -h localhost"
echo ""
