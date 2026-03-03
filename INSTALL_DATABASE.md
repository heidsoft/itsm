# ITSM 数据库安装和初始化指南

## 📋 前提条件

您需要先安装 PostgreSQL 数据库。

---

## 🔧 方案一：使用 Docker（推荐）

### 1. 安装 Docker
```bash
# CentOS/RHEL
sudo yum install -y docker
sudo systemctl start docker
sudo systemctl enable docker

# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y docker.io
sudo systemctl start docker
sudo systemctl enable docker
```

### 2. 启动 PostgreSQL
```bash
docker run -d \
  --name itsm-postgres \
  -e POSTGRES_DB=itsm \
  -e POSTGRES_USER=itsm \
  -e POSTGRES_PASSWORD=itsm_password_2026 \
  -p 5432:5432 \
  -v itsm_pgdata:/var/lib/postgresql/data \
  postgres:13-alpine
```

### 3. 验证运行
```bash
docker ps | grep itsm-postgres
```

---

## 🔧 方案二：直接安装 PostgreSQL

### CentOS/RHEL 7+
```bash
# 安装 PostgreSQL 13
sudo yum install -y https://download.postgresql.org/pub/repos/yum/reporpms/EL-7-x86_64/pgdg-redhat-repo-latest.noarch.rpm
sudo yum install -y postgresql13-server postgresql13-contrib

# 初始化数据库
sudo /usr/pgsql-13/bin/postgresql-13-setup initdb

# 启动服务
sudo systemctl enable postgresql-13
sudo systemctl start postgresql-13
```

### Ubuntu 20.04+
```bash
sudo apt-get update
sudo apt-get install -y postgresql postgresql-contrib

# 启动服务
sudo systemctl enable postgresql
sudo systemctl start postgresql
```

---

## 📝 数据库初始化

### 1. 连接数据库
```bash
# 使用 Docker
docker exec -it itsm-postgres psql -U itsm -d itsm

# 或直接安装
sudo -u postgres psql -h localhost -U itsm -d itsm
```

### 2. 创建数据库和用户（如果需要）
```sql
-- 创建数据库
CREATE DATABASE itsm;

-- 创建用户
CREATE USER itsm WITH PASSWORD 'itsm_password_2026';

-- 授权
GRANT ALL PRIVILEGES ON DATABASE itsm TO itsm;
```

### 3. 执行初始化脚本
```bash
# 方式一：使用提供的脚本
cd itsm-backend
./init_admin.sh

# 方式二：手动执行 SQL
psql -h localhost -U itsm -d itsm -f sql/init_data.sql
psql -h localhost -U itsm -d itsm -f sql/init_approval.sql
psql -h localhost -U itsm -d itsm -f sql/init_sla.sql
```

### 4. 创建默认管理员
```sql
-- 创建管理员账号（密码：admin123）
INSERT INTO users (id, username, email, password_hash, role, name, department, active, created_at, updated_at)
VALUES (
    'admin-001',
    'admin',
    'admin@itsm.com',
    crypt('admin123', gen_salt('bf')),
    'admin',
    '系统管理员',
    'IT 部门',
    true,
    NOW(),
    NOW()
)
ON CONFLICT (username) DO UPDATE SET
    password_hash = crypt('admin123', gen_salt('bf')),
    role = 'admin',
    updated_at = NOW();
```

---

## 🚀 运行后端

### 1. 设置环境变量
```bash
export DATABASE_URL="host=localhost user=itsm password=itsm_password_2026 dbname=itsm port=5432 sslmode=disable"
```

### 2. 执行数据库迁移
```bash
cd itsm-backend
go run -tags migrate main.go
```

### 3. 启动后端服务
```bash
go run main.go
# 或编译后运行
go build -o itsm-backend main.go
./itsm-backend
```

---

## 🔐 默认账号

```
用户名：admin
密码：admin123
```

⚠️ **建议首次登录后立即修改密码！**

---

## 📊 验证安装

### 1. 检查数据库连接
```bash
psql -h localhost -U itsm -d itsm -c "SELECT COUNT(*) FROM users;"
```

### 2. 检查后端服务
```bash
curl http://localhost:8080/health
# 应返回：{"status":"healthy"}
```

### 3. 检查 API
```bash
curl http://localhost:8080/api/v1/users/me
```

---

## 🐛 常见问题

### Q1: 无法连接数据库
**解决**:
```bash
# 检查 PostgreSQL 是否运行
docker ps | grep postgres
# 或
systemctl status postgresql

# 检查防火墙
sudo firewall-cmd --list-ports
sudo firewall-cmd --add-port=5432/tcp --permanent
sudo firewall-cmd --reload
```

### Q2: 权限错误
**解决**:
```bash
# 修改数据目录权限
sudo chown -R postgres:postgres /var/lib/postgresql
```

### Q3: 端口被占用
**解决**:
```bash
# 查找占用端口的进程
sudo netstat -tlnp | grep 5432
sudo lsof -i :5432

# 停止冲突的服务
sudo systemctl stop postgresql
# 或
docker stop itsm-postgres
```

---

## 📞 技术支持

如有问题，请联系技术支持团队。

---

**最后更新**: 2026-03-02
