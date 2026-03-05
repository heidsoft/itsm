# ITSM 项目部署指南

**版本**: 1.0  
**更新日期**: 2026-03-01

---

## 📋 部署方式

### 方式一：Docker Compose（推荐）⭐

**适用场景**: 快速部署、开发环境、测试环境

**步骤**:

```bash
cd itsm

# 1. 启动所有服务
docker-compose up -d

# 2. 查看日志
docker-compose logs -f

# 3. 检查服务状态
docker-compose ps

# 4. 停止服务
docker-compose down
```

**访问地址**:
- 前端：http://localhost:3000
- 后端 API: http://localhost:8090/api/v1
- PostgreSQL: localhost:5432
- Redis: localhost:6379

---

### 方式二：手动安装（生产环境）

#### 1. 安装 PostgreSQL 17

**自动化脚本**（推荐）:

```bash
cd itsm/scripts
chmod +x install-postgresql.sh
sudo ./install-postgresql.sh
```

**手动安装**:

```bash
# 1. 安装 PGDG 仓库
sudo dnf install -y https://download.postgresql.org/pub/repos/yum/reporpms/EL-10-x86_64/pgdg-redhat-repo-latest.noarch.rpm

# 2. 禁用默认 PostgreSQL 模块
sudo dnf -qy module disable postgresql

# 3. 清理并重建缓存
sudo dnf clean all
sudo dnf makecache

# 4. 安装 PostgreSQL 17
sudo dnf install -y postgresql17-server postgresql17-contrib

# 5. 初始化数据库
sudo /usr/pgsql-17/bin/postgresql-17-setup initdb

# 6. 启动服务
sudo systemctl enable postgresql-17
sudo systemctl start postgresql-17

# 7. 创建数据库和用户
sudo -u postgres psql -c "CREATE USER itsm WITH ENCRYPTED PASSWORD 'itsm_password_2026';"
sudo -u postgres psql -c "CREATE DATABASE itsm OWNER itsm;"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE itsm TO itsm;"

# 8. 初始化数据库结构
sudo -u itsm psql -d itsm -f scripts/init-db.sql
```

**验证安装**:

```bash
# 检查服务状态
sudo systemctl status postgresql-17

# 连接测试
psql -U itsm -d itsm -h localhost

# 查看数据库
\l

# 查看表
\dt
```

---

#### 2. 安装 Redis

```bash
# 安装 Redis
sudo dnf install -y redis

# 启动服务
sudo systemctl enable redis
sudo systemctl start redis

# 验证
redis-cli ping
# 应返回：PONG
```

---

#### 3. 部署后端

```bash
cd itsm-backend

# 1. 安装 Go 依赖
go mod download

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env 文件，配置数据库连接

# 3. 编译
go build -o itsm-backend cmd/server/main.go

# 4. 启动服务
./itsm-backend

# 或使用 systemd 管理
sudo cp itsm-backend.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable itsm-backend
sudo systemctl start itsm-backend
```

---

#### 4. 部署前端

```bash
cd itsm-frontend

# 1. 安装 Node.js 依赖
npm install

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env 文件，配置 API 地址

# 3. 构建
npm run build

# 4. 部署到 Nginx
sudo cp -r dist/* /usr/share/nginx/html/

# 5. 重启 Nginx
sudo systemctl restart nginx
```

---

## 🔧 配置说明

### 后端配置 (.env)

```bash
# 服务器配置
PORT=8080
GIN_MODE=release

# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=itsm
DB_PASSWORD=itsm_password_2026
DB_NAME=itsm

# Redis 配置
REDIS_HOST=localhost
REDIS_PORT=6379

# 阿里云配置
ALIYUN_ACCESS_KEY_ID=LTAI5t...
ALIYUN_ACCESS_KEY_SECRET=...
ALIYUN_REGION_ID=cn-shanghai
SECURITY_GROUP_ID=sg-xxx

# JWT 配置
JWT_SECRET=your_jwt_secret
JWT_EXPIRE=24h
```

### 前端配置 (.env)

```bash
# API 地址
REACT_APP_API_URL=http://localhost:8090/api/v1

# 其他配置
REACT_APP_ENV=production
```

---

## 📊 数据库管理

### 备份

```bash
# 完整备份
pg_dump -U itsm -h localhost itsm > backup_$(date +%Y%m%d).sql

# 仅备份结构
pg_dump -U itsm -h localhost --schema-only itsm > schema.sql

# 压缩备份
pg_dump -U itsm -h localhost itsm | gzip > backup_$(date +%Y%m%d).sql.gz
```

### 恢复

```bash
# 从备份恢复
psql -U itsm -h localhost itsm < backup_20260301.sql

# 从压缩备份恢复
gunzip -c backup_20260301.sql.gz | psql -U itsm -h localhost itsm
```

### 日常维护

```bash
# 分析表
psql -U itsm -h localhost itsm -c "ANALYZE;"

# 清理死元组
psql -U itsm -h localhost itsm -c "VACUUM;"

# 完全清理
psql -U itsm -h localhost itsm -c "VACUUM FULL;"
```

---

## 🔒 安全配置

### 防火墙配置

```bash
# 开放 PostgreSQL 端口
sudo firewall-cmd --permanent --add-port=5432/tcp
sudo firewall-cmd --reload

# 开放 Redis 端口
sudo firewall-cmd --permanent --add-port=6379/tcp
sudo firewall-cmd --reload

# 开放后端 API 端口
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload

# 开放前端端口
sudo firewall-cmd --permanent --add-port=3000/tcp
sudo firewall-cmd --reload
```

### PostgreSQL 安全

```bash
# 编辑 pg_hba.conf
sudo vim /var/lib/pgsql/17/data/pg_hba.conf

# 限制只允许本地连接
# host    itsm    itsm    127.0.0.1/32    scram-sha-256
# host    itsm    itsm    ::1/128         scram-sha-256

# 重启 PostgreSQL
sudo systemctl restart postgresql-17
```

---

## 📈 监控与日志

### 查看日志

```bash
# PostgreSQL 日志
sudo tail -f /var/log/pgsql/postgresql-17-main.log

# 后端日志
sudo journalctl -u itsm-backend -f

# Nginx 日志
sudo tail -f /var/log/nginx/access.log
sudo tail -f /var/log/nginx/error.log
```

### 性能监控

```bash
# PostgreSQL 连接数
psql -U itsm -h localhost itsm -c "SELECT count(*) FROM pg_stat_activity;"

# 数据库大小
psql -U itsm -h localhost itsm -c "SELECT pg_size_pretty(pg_database_size('itsm'));"

# 慢查询
psql -U itsm -h localhost itsm -c "SELECT * FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10;"
```

---

## 🚨 故障排查

### PostgreSQL 无法启动

```bash
# 检查日志
sudo journalctl -u postgresql-17 -n 50

# 检查端口占用
sudo netstat -tlnp | grep 5432

# 检查权限
sudo ls -la /var/lib/pgsql/17/data/
```

### 连接失败

```bash
# 测试连接
psql -U itsm -h localhost itsm -c "SELECT 1;"

# 检查防火墙
sudo firewall-cmd --list-all

# 检查 PostgreSQL 配置
sudo cat /var/lib/pgsql/17/data/postgresql.conf | grep listen_addresses
```

### 性能问题

```bash
# 检查慢查询
psql -U itsm -h localhost itsm -c "SELECT * FROM pg_stat_statements WHERE total_time > 1000 ORDER BY total_time DESC;"

# 检查锁
psql -U itsm -h localhost itsm -c "SELECT * FROM pg_locks WHERE NOT granted;"

# 检查表大小
psql -U itsm -h localhost itsm -c "SELECT table_name, pg_size_pretty(pg_total_relation_size(table_name)) FROM information_schema.tables WHERE table_schema = 'public' ORDER BY pg_total_relation_size(table_name) DESC;"
```

---

## 📝 快速参考

### 服务管理

```bash
# PostgreSQL
sudo systemctl start|stop|restart|status postgresql-17

# Redis
sudo systemctl start|stop|restart|status redis

# 后端
sudo systemctl start|stop|restart|status itsm-backend

# Nginx
sudo systemctl start|stop|restart|status nginx
```

### 数据库操作

```bash
# 连接数据库
psql -U itsm -h localhost itsm

# 备份
pg_dump -U itsm -h localhost itsm > backup.sql

# 恢复
psql -U itsm -h localhost itsm < backup.sql
```

---

## 🌐 访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| 前端 | http://localhost:3000 | Web 界面 |
| 后端 API | http://localhost:8090/api/v1 | REST API |
| PostgreSQL | localhost:5432 | 数据库 |
| Redis | localhost:6379 | 缓存 |
| 健康检查 | http://localhost:8090/health | 健康检查 |

---

**部署完成！** 🎉

**维护者**: 运维团队  
**最后更新**: 2026-03-01  
**下次审查**: 2026-04-01
