# ITSM 数据库配置指南

**最后更新**: 2026-03-04
**版本**: v1.0

---

## 📋 目录

- [数据库概述](#数据库概述)
- [数据库设置](#数据库设置)
- [连接配置](#连接配置)
- [数据库迁移](#数据库迁移)
- [数据初始化](#数据初始化)
- [数据库维护](#数据库维护)
- [备份与恢复](#备份与恢复)
- [性能优化](#性能优化)

---

## 数据库概述

ITSM 系统使用 **PostgreSQL** 作为主数据库，采用 **Ent ORM** 框架进行数据访问。

### 技术选型

| 特性 | 选择 | 原因 |
|------|------|------|
| 数据库 | PostgreSQL 17 | 可靠性高、功能丰富 |
| ORM | Ent | 类型安全、自动生成代码 |
| 迁移 | Ent Migrate | 与 Ent 深度集成 |
| 连接池 | pgxpool | 高性能连接池 |

### 数据库架构

```
┌─────────────────────────────────────────────┐
│              itsm 数据库                      │
├─────────────────────────────────────────────┤
│  Schema 表                                    │
│  ├─ users          # 用户表                  │
│  ├─ roles          # 角色表                  │
│  ├─ permissions    # 权限表                  │
│  ├─ tenants       # 租户表                  │
│  │                                          │
│  ├─ tickets        # 工单表                  │
│  ├─ incidents      # 事件表                  │
│  ├─ problems       # 问题表                  │
│  ├─ changes        # 变更表                  │
│  │                                          │
│  ├─ knowledge_articles  # 知识库文章         │
│  ├─ sla_definitions # SLA 定义              │
│  ├─ workflows      # 工作流定义              │
│  └─ ...            # 其他表                  │
└─────────────────────────────────────────────┘
```

---

## 数据库设置

### 方式一：Docker 部署（推荐）

```bash
# 使用 docker-compose 启动 PostgreSQL
docker compose up -d postgres

# 验证 PostgreSQL 运行状态
docker ps | grep postgres

# 查看日志
docker compose logs postgres
```

### 方式二：本地安装

#### macOS

```bash
# 安装 PostgreSQL
brew install postgresql@17
brew services start postgresql@17

# 创建数据库
createdb itsm

# 创建用户
createuser -s itsm

# 设置密码
psql -c "ALTER USER itsm PASSWORD 'itsm_password';"
```

#### Ubuntu/Debian

```bash
# 安装
sudo apt update
sudo apt install postgresql-17

# 启动服务
sudo systemctl start postgresql
sudo systemctl enable postgresql

# 创建用户和数据库
sudo -u postgres createuser -s itsm
sudo -u postgres psql -c "ALTER USER itsm PASSWORD 'itsm_password';"
sudo -u postgres createdb itsm
```

### 方式三：云数据库

#### 阿里云 RDS

```env
DB_HOST=rds-xxx.rds.aliyuncs.com
DB_PORT=5432
DB_USER=itsm
DB_PASSWORD=your_password
DB_NAME=itsm
DB_SSLMODE=require
```

#### AWS RDS

```env
DB_HOST=xxx.xxx.us-east-1.rds.amazonaws.com
DB_PORT=5432
DB_USER=itsm
DB_PASSWORD=your_password
DB_NAME=itsm
DB_SSLMODE=require
```

---

## 连接配置

### 环境变量配置

```bash
# 标准连接配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=itsm
DB_PASSWORD=itsm_password
DB_NAME=itsm
DB_SSLMODE=disable

# 或使用连接字符串
DATABASE_URL=postgres://itsm:itsm_password@localhost:5432/itsm?sslmode=disable
```

### 连接池配置

```env
# 连接池配置
DB_MAX_IDLE_CONNS=10      # 最大空闲连接
DB_MAX_OPEN_CONNS=100     # 最大开放连接
DB_CONN_MAX_LIFETIME=3600 # 连接生命周期（秒）
```

### 验证连接

```bash
# 测试数据库连接
psql -h localhost -U itsm -d itsm -c "SELECT version();"

# 或使用 Docker
docker exec -it itsm-postgres psql -U itsm -d itsm -c "SELECT version();"
```

---

## 数据库迁移

### 自动迁移

系统启动时自动执行迁移（开发环境）：

```bash
cd itsm-backend
go run main.go
```

迁移会自动创建/更新表结构。

### 手动迁移

```bash
# 运行迁移命令
cd itsm-backend
go run -tags migrate main.go

# 或使用 Make
make db-migrate
```

### 迁移参数

```bash
# 查看迁移帮助
go run main.go migrate -h

# 执行迁移
go run main.go migrate up

# 回滚最后一次迁移
go run main.go migrate down

# 重置数据库（⚠️ 危险！删除所有表）
go run main.go migrate reset

# 创建新迁移
go run main.go migrate create -name add_new_table
```

### 生成迁移文件

```bash
# 使用 Ent 生成迁移
cd itsm-backend/ent
go generate ./...

# 或使用 golang-migrate
migrate -path ./migrations -database "postgres://user:pass@localhost:5432/itsm" up
```

---

## 数据初始化

### 种子数据

系统首次启动时自动导入种子数据：

```bash
# 手动触发数据初始化
cd itsm-backend
go run main.go seed

# 或使用 Make
make db-seed
```

### 种子数据内容

| 数据类型 | 说明 |
|----------|------|
| 默认租户 | 系统默认租户 |
| 管理员用户 | admin/admin123 |
| 默认角色 | 管理员、服务台、终端用户 |
| 系统权限 | 完整权限列表 |
| 工单状态 | 新建、处理中、已解决、已关闭等 |
| 工单优先级 | 紧急、高、中、低 |
| SLA 模板 | 默认服务级别协议 |
| 示例工单 | 演示数据 |

### 自定义种子数据

编辑 `itsm-backend/pkg/seeder/seeder.go` 添加自定义数据：

```go
// 添加自定义角色
roles := []*ent.RoleCreate{
    client.Role.Create().
        SetName("custom_role").
        SetDescription("自定义角色").
        Save,
}
```

---

## 数据库维护

### 查看表结构

```sql
-- 查看所有表
\dt

-- 查看表结构
\d users

-- 查看索引
\di

-- 查看约束
\d users
```

### 常用维护操作

```sql
-- 清理连接
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE state = 'idle'
AND query_start < now() - interval '10 minutes';

-- 查看表大小
SELECT relname, pg_size_pretty(pg_total_relation_size(relid))
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC
LIMIT 10;

-- 查看索引使用情况
SELECT
    indexrelname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;
```

---

## 备份与恢复

### 手动备份

```bash
# 备份整个数据库
pg_dump -h localhost -U itsm -d itsm > backup_$(date +%Y%m%d_%H%M%S).sql

# 压缩备份
pg_dump -h localhost -U itsm -d itsm | gzip > backup_$(date +%Y%m%d).sql.gz

# 备份指定表
pg_dump -h localhost -U itsm -d itsm -t users -t tickets > tables_backup.sql
```

### Docker 环境备份

```bash
# 备份
docker exec itsm-postgres pg_dump -U itsm itsm > backup_$(date +%Y%m%d).sql

# 压缩备份
docker exec itsm-postgres pg_dump -U itsm itsm | gzip > backup_$(date +%Y%m%d).sql.gz
```

### 自动备份脚本

```bash
#!/bin/bash
# scripts/backup.sh

BACKUP_DIR="/var/backups/itsm"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

mkdir -p $BACKUP_DIR

# 备份数据库
docker exec itsm-postgres pg_dump -U itsm itsm | gzip > $BACKUP_DIR/db_$DATE.sql.gz

# 清理旧备份
find $BACKUP_DIR -name "*.sql.gz" -mtime +$RETENTION_DAYS -delete

echo "Backup completed: $DATE"
```

添加到 crontab：

```bash
# 每天凌晨 2 点执行
0 2 * * * /path/to/scripts/backup.sh >> /var/log/itsm-backup.log 2>&1
```

### 恢复数据

```bash
# 从备份文件恢复
psql -h localhost -U itsm -d itsm < backup_20260304.sql

# 从压缩文件恢复
gunzip -c backup_20260304.sql.gz | psql -h localhost -U itsm -d itsm
```

### 定期维护任务

```sql
-- 每周执行一次 VACUUM
VACUUM (VERBOSE, ANALYZE);

-- 重建索引（如果需要）
REINDEX TABLE tickets;
REINDEX TABLE users;
```

---

## 性能优化

### 索引优化

```sql
-- 创建常用索引
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_priority ON tickets(priority);
CREATE INDEX idx_tickets_created_at ON tickets(created_at DESC);
CREATE INDEX idx_tickets_assignee ON tickets(assignee_id);
CREATE INDEX idx_tickets_tenant ON tickets(tenant_id);

-- 创建复合索引
CREATE INDEX idx_tickets_status_priority ON tickets(status, priority);

-- 创建部分索引
CREATE INDEX idx_tickets_open ON tickets(created_at DESC)
WHERE status IN ('new', 'in_progress');
```

### PostgreSQL 配置优化

编辑 `postgresql.conf`：

```conf
# 内存配置
shared_buffers = 2GB                  # 25% 内存
effective_cache_size = 6GB            # 75% 内存
maintenance_work_mem = 512MB
work_mem = 64MB

# 写入配置
wal_buffers = 16MB
checkpoint_completion_target = 0.9
max_wal_size = 1GB
min_wal_size = 80MB

# 并发配置
max_connections = 200
effective_io_concurrency = 200

# 日志
log_min_duration_statement = 1000  # 记录慢查询
```

### 连接池配置

使用 PgBouncer：

```yaml
# docker-compose.yml
pgbouncer:
  image: pgbouncer/pgbouncer
  environment:
    DATABASE_URL: postgres://itsm:pass@postgres:5432/itsm
    POOL_MODE: transaction
    MAX_CLIENT_CONN: 200
    DEFAULT_POOL_SIZE: 20
```

---

## 常见问题

### 问题 1：连接被拒绝

```bash
# 检查 PostgreSQL 状态
docker ps | grep postgres

# 检查端口占用
lsof -i :5432

# 检查防火墙
sudo ufw status
```

### 问题 2：认证失败

```bash
# 修改密码
psql -h localhost -U postgres -c "ALTER USER itsm PASSWORD 'new_password';"
```

### 问题 3：数据库不存在

```bash
# 创建数据库
createdb -h localhost -U postgres itsm

# 或在 Docker 中
docker exec -it itsm-postgres psql -U postgres -c "CREATE DATABASE itsm;"
```

---

**文档维护**: ITSM 开发团队
**最后更新**: 2026-03-04
