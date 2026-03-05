# PostgreSQL 详细使用指南

**最后更新**: 2026-03-04
**版本**: v1.0

---

## 📋 目录

- [PostgreSQL 简介](#postgresql-简介)
- [安装 PostgreSQL](#安装-postgresql)
- [创建用户和数据库](#创建用户和数据库)
- [用户权限管理](#用户权限管理)
- [数据库连接方式](#数据库连接方式)
- [数据库初始化结构](#数据库初始化结构)
- [表结构说明](#表结构说明)
- [数据导入导出](#数据导入导出)
- [日常运维](#日常运维)
- [性能监控](#性能监控)
- [安全配置](#安全配置)

---

## PostgreSQL 简介

PostgreSQL 是一个功能强大的开源对象关系型数据库系统，以其可靠性、鲁棒性和性能而闻名。

### 核心特性

| 特性 | 说明 |
|------|------|
| ACID 事务 | 原子性、一致性、隔离性、持久性 |
| 多版本并发控制 | MVCC 无锁读 |
| 复杂查询 | 支持 JOIN、子查询、窗口函数 |
| 扩展性 | 支持自定义函数、数据类型 |
| 复制 | 支持主从流复制 |
| JSON 支持 | 原生 JSON/JSONB 类型 |

---

## 安装 PostgreSQL

### macOS 安装

```bash
# 使用 Homebrew 安装
brew install postgresql@17

# 启动服务
brew services start postgresql@17

# 查看版本
pg_ctl --version

# 连接到默认数据库
psql -U $(whoami) -d postgres
```

### Ubuntu/Debian 安装

```bash
# 添加 PostgreSQL APT 源
sudo sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -
sudo apt-get update

# 安装 PostgreSQL 17
sudo apt-get install postgresql-17

# 启动服务
sudo systemctl start postgresql
sudo systemctl enable postgresql

# 查看状态
sudo systemctl status postgresql
```

### CentOS/RHEL 安装

```bash
# 安装 EPEL 仓库
sudo yum install epel-release

# 安装 PostgreSQL
sudo yum install postgresql17-server

# 初始化数据库
sudo /usr/pgsql-17/bin/postgresql-17-setup initdb

# 启动服务
sudo systemctl start postgresql-17
sudo systemctl enable postgresql-17
```

### Docker 安装

```bash
# 拉取官方镜像
docker pull postgres:17-alpine

# 运行 PostgreSQL 容器
docker run -d \
  --name itsm-postgres \
  -e POSTGRES_USER=itsm \
  -e POSTGRES_PASSWORD=itsm_password \
  -e POSTGRES_DB=itsm \
  -p 5432:5432 \
  -v postgres_data:/var/lib/postgresql/data \
  postgres:17-alpine

# 验证运行
docker ps | grep postgres
```

---

## 创建用户和数据库

### 1. 使用 psql 命令行

#### 连接到 PostgreSQL

```bash
# 本地连接（使用 postgres 系统用户）
sudo -u postgres psql

# 或使用当前用户
psql -U postgres -h localhost

# Docker 环境
docker exec -it itsm-postgres psql -U postgres
```

#### 创建用户（角色）

```sql
-- 创建用户
CREATE USER itsm_user WITH PASSWORD 'StrongPassword123!';

-- 创建超级用户（谨慎使用）
CREATE USER itsm_admin WITH PASSWORD 'StrongPassword123!' SUPERUSER;

-- 创建只读用户
CREATE USER itsm_readonly WITH PASSWORD 'StrongPassword123!' NOSUPERUSER;
```

#### 创建数据库

```sql
-- 创建数据库（指定所有者）
CREATE DATABASE itsm_db OWNER itsm_user;

-- 创建数据库（指定编码）
CREATE DATABASE itsm_db OWNER itsm_user ENCODING 'UTF8';

-- 创建数据库（指定表空间）
CREATE DATABASE itsm_db OWNER itsm_user TABLESPACE itsm_tablespace;
```

#### 删除数据库和用户

```sql
-- 删除数据库（必须先断开所有连接）
DROP DATABASE IF EXISTS itsm_db;

-- 删除用户
DROP USER IF EXISTS itsm_user;
```

### 2. 使用命令行工具

```bash
# 创建用户
createuser -U postgres -P itsm_user
# 系统会提示输入密码和确认

# 创建数据库
createdb -U postgres -O itsm_user itsm_db

# 删除数据库
dropdb -U postgres itsm_db

# 删除用户
dropuser -U postgres itsm_user
```

### 3. Docker 环境创建

```bash
# 方式一：环境变量自动创建
docker run -d \
  --name itsm-postgres \
  -e POSTGRES_USER=itsm_user \
  -e POSTGRES_PASSWORD=StrongPassword123! \
  -e POSTGRES_DB=itsm_db \
  -p 5432:5432 \
  postgres:17-alpine

# 方式二：进入容器创建
docker exec -it itsm-postgres psql -U postgres

# 在 psql 中执行
CREATE USER itsm_user WITH PASSWORD 'StrongPassword123!';
CREATE DATABASE itsm_db OWNER itsm_user;
GRANT ALL PRIVILEGES ON DATABASE itsm_db TO itsm_user;
```

### 4. 创建完整示例

```bash
#!/bin/bash
# scripts/init-postgres.sh

# PostgreSQL 连接参数
PG_HOST=${PG_HOST:-localhost}
PG_PORT=${PG_PORT:-5432}
PG_USER=${PG_USER:-postgres}
PG_DB=${PG_DB:-postgres}

# ITSM 数据库配置
ITSM_USER="itsm_user"
ITSM_DB="itsm_db"
ITSM_PASSWORD="StrongPassword123!"

echo "开始初始化 PostgreSQL..."

# 创建用户
psql -h $PG_HOST -p $PG_PORT -U $PG_USER -d $PG_DB << EOF
-- 创建用户
CREATE USER $ITSM_USER WITH PASSWORD '$ITSM_PASSWORD' NOSUPERUSER CREATEDB;

-- 创建数据库
CREATE DATABASE $ITSM_DB OWNER $ITSM_USER;

-- 授予权限
GRANT ALL PRIVILEGES ON DATABASE $ITSM_DB TO $ITSM_USER;

-- 授予架构权限
GRANT ALL ON SCHEMA public TO $ITSM_USER;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $ITSM_USER;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $ITSM_USER;

-- 设置默认权限
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $ITSM_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $ITSM_USER;
EOF

echo "PostgreSQL 初始化完成!"
echo "用户: $ITSM_USER"
echo "数据库: $ITSM_DB"
```

---

## 用户权限管理

### 权限级别

| 权限 | 说明 |
|------|------|
| `SELECT` | 读取数据 |
| `INSERT` | 插入数据 |
| `UPDATE` | 更新数据 |
| `DELETE` | 删除数据 |
| `TRUNCATE` | 清空表 |
| `REFERENCES` | 外键约束 |
| `TRIGGER` | 触发器 |
| `CREATE` | 创建对象 |
| `CONNECT` | 连接数据库 |
| `TEMPORARY` | 临时表 |

### 授予权限

```sql
-- 授予数据库级别权限
GRANT CONNECT ON DATABASE itsm_db TO itsm_user;
GRANT ALL PRIVILEGES ON DATABASE itsm_db TO itsm_user;

-- 授予表级别权限
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO itsm_user;

-- 授予序列权限
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO itsm_user;

-- 授予特定表权限
GRANT ALL PRIVILEGES ON TABLE users TO itsm_user;
GRANT SELECT ON TABLE tickets TO itsm_user;
```

### 角色管理

```sql
-- 创建角色
CREATE ROLE readonly;
CREATE ROLE readwrite;
CREATE ROLE admin;

-- 授予角色权限
GRANT readonly TO itsm_user;
GRANT readwrite TO itsm_user;

-- 撤销角色
REVOKE readonly FROM itsm_user;

-- 删除角色
DROP ROLE readonly;
```

### 行级安全策略

```sql
-- 启用行级安全
ALTER TABLE tickets ENABLE ROW LEVEL SECURITY;

-- 创建策略（租户隔离）
CREATE POLICY tenant_isolation_policy ON tickets
    USING (tenant_id = current_setting('app.tenant_id')::bigint);

-- 设置当前租户
SET app.tenant_id = '1';
```

---

## 数据库连接方式

### 1. 本地连接

```bash
# 使用 psql 直接连接
psql -U itsm_user -d itsm_db -h localhost

# 使用连接字符串
psql "postgresql://itsm_user:password@localhost:5432/itsm_db"
```

### 2. TCP/IP 连接

```bash
# 远程连接
psql -U itsm_user -d itsm_db -h 192.168.1.100 -p 5432

# SSL 连接
psql "postgresql://itsm_user:password@hostname:5432/itsm_db?sslmode=require"
```

### 3. 应用连接配置

```bash
# 环境变量
export DATABASE_URL="postgres://itsm_user:password@localhost:5432/itsm_db"

# Go 应用连接
# 格式: host=localhost port=5432 user=itsm_user password=password dbname=itsm_db sslmode=disable
```

### 4. pg_hba.conf 配置

编辑 `/etc/postgresql/17/main/pg_hba.conf`:

```conf
# 本地连接
local   all             all                                     peer

# IPv4 本地连接
host    all             all             127.0.0.1/32            scram-sha-256

# IPv6 本地连接
host    all             all             ::1/128                 scram-sha-256

# 远程连接（根据实际情况修改）
host    all             all             192.168.0.0/16          scram-sha-256

# 复制连接
host    replication     all             127.0.0.1/32            scram-sha-256
```

修改后重新加载：

```bash
# 重新加载配置
sudo -u postgres pg_ctl reload -D /var/lib/postgresql/17/main

# 或
SELECT pg_reload_conf();
```

---

## 数据库初始化结构

### 方式一：应用自动迁移

```bash
# 方式一：直接运行
cd itsm-backend
go run main.go

# 方式二：使用迁移标签
cd itsm-backend
go run -tags migrate main.go

# 方式三：使用 Make
make db-migrate
```

### 方式二：Ent ORM 自动生成

```bash
# 进入 ent 目录
cd itsm-backend/ent

# 生成迁移 SQL
go run ./cmd/migrate/main.go

# 或使用 Ent 工具
go generate ./...

# 查看生成的迁移
ls -la itsm-backend/ent/migrate/migrations/
```

### 方式三：手动执行 SQL

```bash
# 方式一：psql 命令行
psql -U itsm_user -d itsm_db -f init.sql

# 方式二：Docker 环境
docker exec -i itsm-postgres psql -U itsm_user -d itsm_db < init.sql

# 方式三：PG Admin 工具
# 使用 pgAdmin 图形界面执行 SQL 文件
```

### 方式四：初始化脚本

```bash
#!/bin/bash
# scripts/init-db.sh

set -e

echo "开始初始化数据库..."

# 等待 PostgreSQL 就绪
until PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -d postgres -c '\q'; do
  echo "等待数据库启动..."
  sleep 2
done

echo "数据库已就绪"

# 执行迁移
cd itsm-backend
go run -tags migrate main.go

# 执行种子数据
go run main.go seed

echo "数据库初始化完成!"
```

---

## 表结构说明

### 核心表结构

#### 用户表 (users)

```sql
CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    username        VARCHAR(50) UNIQUE NOT NULL,
    password        VARCHAR(255) NOT NULL,
    email           VARCHAR(255) UNIQUE NOT NULL,
    full_name       VARCHAR(100),
    phone           VARCHAR(20),
    avatar          VARCHAR(500),
    status          VARCHAR(20) DEFAULT 'active',
    tenant_id       BIGINT,
    role_id         BIGINT,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_status ON users(status);
```

#### 角色表 (roles)

```sql
CREATE TABLE roles (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(50) UNIQUE NOT NULL,
    code            VARCHAR(50) UNIQUE NOT NULL,
    description     TEXT,
    status          VARCHAR(20) DEFAULT 'active',
    tenant_id       BIGINT,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### 工单表 (tickets)

```sql
CREATE TABLE tickets (
    id              BIGSERIAL PRIMARY KEY,
    ticket_number   VARCHAR(50) UNIQUE NOT NULL,
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    status          VARCHAR(20) DEFAULT 'new',
    priority        VARCHAR(20) DEFAULT 'medium',
    category        VARCHAR(50),
    source          VARCHAR(20),
    requester_id    BIGINT NOT NULL,
    assignee_id     BIGINT,
    tenant_id       BIGINT,
    sla_id          BIGINT,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at     TIMESTAMP,
    closed_at       TIMESTAMP,
    deleted_at      TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_priority ON tickets(priority);
CREATE INDEX idx_tickets_requester ON tickets(requester_id);
CREATE INDEX idx_tickets_assignee ON tickets(assignee_id);
CREATE INDEX idx_tickets_created_at ON tickets(created_at DESC);
CREATE INDEX idx_tickets_tenant ON tickets(tenant_id);
```

#### 事件表 (incidents)

```sql
CREATE TABLE incidents (
    id              BIGSERIAL PRIMARY KEY,
    incident_number VARCHAR(50) UNIQUE NOT NULL,
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    status          VARCHAR(20) DEFAULT 'open',
    severity        VARCHAR(20) DEFAULT 'low',
    impact          VARCHAR(20),
    urgency         VARCHAR(20),
    category        VARCHAR(50),
    assignee_id     BIGINT,
    tenant_id       BIGINT,
    related_ticket_id BIGINT,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at     TIMESTAMP,
    closed_at       TIMESTAMP
);
```

### 查看表结构

```sql
-- 查看所有表
\dt

-- 查看表结构
\d users

-- 查看表详情
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_name = 'users'
ORDER BY ordinal_position;

-- 查看索引
SELECT indexname, indexdef
FROM pg_indexes
WHERE tablename = 'users';

-- 查看外键
SELECT
    tc.table_name,
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu
    ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_name = 'users';
```

---

## 数据导入导出

### 导出数据

```bash
# 导出整个数据库
pg_dump -h localhost -U itsm_user -d itsm_db > itsm_db.sql

# 导出特定表
pg_dump -h localhost -U itsm_user -d itsm_db -t users -t tickets > tables.sql

# 导出结构（不含数据）
pg_dump -h localhost -U itsm_user -d itsm_db --schema-only > schema.sql

# 导出数据（不含结构）
pg_dump -h localhost -U itsm_user -d itsm_db --data-only > data.sql

# 压缩导出
pg_dump -h localhost -U itsm_user -d itsm_db | gzip > itsm_db.sql.gz

# 导出为 CSV
COPY (SELECT * FROM users) TO '/tmp/users.csv' WITH CSV HEADER;
```

### 导入数据

```bash
# 导入整个数据库
psql -h localhost -U itsm_user -d itsm_db < itsm_db.sql

# 导入压缩文件
gunzip -c itsm_db.sql.gz | psql -h localhost -U itsm_user -d itsm_db

# 导入 CSV
COPY users(username, email, full_name) FROM '/tmp/users.csv' WITH CSV HEADER;

# 从远程导入
pg_dump -h remote-host -U remote-user -d remote_db | psql -h local-host -U local-user -d local_db
```

### 表级复制

```sql
-- 复制表结构
CREATE TABLE tickets_backup (LIKE tickets INCLUDING ALL);

-- 复制表数据
INSERT INTO tickets_backup SELECT * FROM tickets;

-- 带条件的复制
INSERT INTO tickets_backup SELECT * FROM tickets WHERE status = 'closed';
```

---

## 日常运维

### 服务管理

```bash
# macOS
brew services start postgresql@17
brew services stop postgresql@17
brew services restart postgresql@17

# Linux (systemd)
sudo systemctl start postgresql
sudo systemctl stop postgresql
sudo systemctl restart postgresql
sudo systemctl status postgresql

# Docker
docker start itsm-postgres
docker stop itsm-postgres
docker restart itsm-postgres
```

### 数据库连接管理

```sql
-- 查看当前连接
SELECT * FROM pg_stat_activity;

-- 查看特定数据库连接
SELECT * FROM pg_stat_activity WHERE datname = 'itsm_db';

-- 查看连接数统计
SELECT state, COUNT(*) FROM pg_stat_activity GROUP BY state;

-- 终止空闲连接
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE state = 'idle'
AND query_start < now() - interval '10 minutes';
```

### Vacuum 和 Analyze

```sql
-- 回收空间
VACUUM FULL users;

-- 更新统计信息
ANALYZE users;

-- 一起执行
VACUUM ANALYZE tickets;

-- 查看膨胀的表
SELECT
    schemaname,
    relname,
    pg_size_pretty(pg_total_relation_size(relid)) AS total_size,
    pg_size_pretty(pg_relation_size(relid)) AS table_size,
    pg_size_pretty(pg_total_relation_size(relid) - pg_relation_size(relid)) AS index_size
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC
LIMIT 10;
```

---

## 性能监控

### 慢查询日志

```sql
-- 启用慢查询日志
ALTER SYSTEM SET log_min_duration_statement = 1000;  -- 记录超过 1 秒的查询

-- 查看慢查询
SELECT
    query,
    calls,
    mean_time,
    total_time,
    rows
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;

-- 查看最近执行的查询
SELECT
    pid,
    now() - pg_stat_activity.query_start AS duration,
    query,
    state
FROM pg_stat_activity
WHERE (now() - pg_stat_activity.query_start) > interval '5 seconds'
AND state = 'active';
```

### 监控视图

```sql
-- 数据库大小
SELECT pg_database.datname, pg_size_pretty(pg_database_size(pg_database.datname))
FROM pg_database
WHERE datname = current_database();

-- 表大小排名
SELECT
    schemaname,
    relname,
    pg_size_pretty(pg_total_relation_size(relid))
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC
LIMIT 10;

-- 索引使用统计
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;

-- 缓存命中率
SELECT
    sum(heap_blks_read) as heap_read,
    sum(heap_blks_hit) as heap_hit,
    round(sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read)) * 100, 2) as ratio
FROM pg_statio_user_tables;
```

### 监控工具

```bash
# 使用 pgAdmin
# 下载地址: https://www.pgadmin.org/

# 使用 pg_top
sudo apt install ptop
pg_top

# 使用 pg_stat_statements
CREATE EXTENSION pg_stat_statements;
```

---

## 安全配置

### 密码策略

```sql
-- 设置密码有效期
ALTER USER itsm_user VALID UNTIL '2026-12-31';

-- 永不过期
ALTER USER itsm_user VALID UNTIL infinity;

-- 强制密码复杂度
-- 在 pg_hba.conf 中使用 scram-sha-256
```

### 连接限制

```sql
-- 限制用户连接数
ALTER USER itsm_user CONNECTION LIMIT 100;

-- 查看当前连接限制
SELECT rolname, rolconnlimit FROM pg_roles WHERE rolname = 'itsm_user';
```

### 审计日志

```sql
-- 启用审计日志
ALTER SYSTEM SET log_connections = on;
ALTER SYSTEM SET log_disconnections = on;
ALTER SYSTEM SET log_statement = 'all';

-- 查看连接日志
SELECT * FROM pg_log
WHERE message LIKE '%connection%'
ORDER BY log_time DESC;
```

### 数据加密

```bash
# 启用 SSL 连接
# 在连接字符串中添加 sslmode=require

# 启用表空间加密（需要文件系统支持）
CREATE TABLESPACE encrypted_tablespace LOCATION '/data/encrypted' WITH (encrypt = true);
```

---

## 常见问题

### 问题 1：连接数过多

```sql
-- 查看最大连接数
SHOW max_connections;

-- 终止异常连接
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE state = 'idle'
AND query_start < now() - interval '5 minutes';
```

### 问题 2：数据库启动失败

```bash
# 查看详细日志
tail -f /var/log/postgresql/postgresql-17-main.log

# 检查数据目录权限
ls -la /var/lib/postgresql/17/main/

# 修复权限
sudo chown -R postgres:postgres /var/lib/postgresql
```

### 问题 3：磁盘空间不足

```bash
# 清理 WAL 日志
pg_archivecleanup /var/lib/postgresql/17/main/pg_wal < oldest_wal>

# 清理临时文件
rm -rf /tmp/pgsql.*

# 清理大对象
VACUUM FULL;
```

---

**文档维护**: ITSM 开发团队
**最后更新**: 2026-03-04
