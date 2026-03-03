# ITSM 系统快速启动指南

## 📋 前提条件

您需要先安装 PostgreSQL 数据库。

---

## 🔧 步骤 1: 安装 PostgreSQL

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

# 检查状态
sudo systemctl status postgresql-13
```

### Ubuntu 20.04+
```bash
sudo apt-get update
sudo apt-get install -y postgresql postgresql-contrib postgresql-client

# 启动服务
sudo systemctl enable postgresql
sudo systemctl start postgresql

# 检查状态
sudo systemctl status postgresql
```

---

## 🔧 步骤 2: 创建数据库和用户

```bash
# 切换到 postgres 用户
sudo -u postgres psql

# 在 PostgreSQL 命令行中执行:
CREATE DATABASE itsm;
CREATE USER dev WITH PASSWORD 'dev_password_2026';
GRANT ALL PRIVILEGES ON DATABASE itsm TO dev;
\q
```

---

## 🔧 步骤 3: 初始化数据库表结构

```bash
# 设置环境变量
export DB_PASSWORD=dev_password_2026
export PGPASSWORD=dev_password_2026

# 执行迁移
cd itsm-backend
go run -tags migrate main.go
```

---

## 🔧 步骤 4: 创建默认管理员账号

```bash
# 执行初始化脚本
cd itsm-backend
./init_database.sh
```

**默认管理员账号:**
```
用户名：admin
密码：admin123
```

---

## 🔧 步骤 5: 启动后端服务

```bash
cd itsm-backend

# 设置环境变量
export DB_PASSWORD=dev_password_2026

# 启动服务
go run main.go

# 或编译后运行
go build -o itsm-backend main.go
./itsm-backend
```

后端服务将在 `http://localhost:8080` 启动

---

## 🔧 步骤 6: 启动前端服务

```bash
cd itsm-frontend

# 安装依赖（首次运行）
npm install

# 启动开发服务器
npm run dev
```

前端服务将在 `http://localhost:5173` 启动

---

## 🌐 访问系统

### 部署管理系统
- **URL**: http://localhost:5173
- **账号**: admin / admin123

### 后端 API
- **URL**: http://localhost:8080
- **健康检查**: http://localhost:8080/health

---

## 🐛 常见问题

### Q1: PostgreSQL 服务无法启动
**解决**:
```bash
# 查看日志
sudo journalctl -u postgresql

# 检查端口占用
sudo netstat -tlnp | grep 5432
```

### Q2: 数据库连接失败
**解决**:
```bash
# 检查配置文件
sudo -u postgres psql -c "SHOW hba_file;"

# 允许本地连接
echo "host all all 127.0.0.1/32 md5" | sudo tee -a /var/lib/pgsql/13/data/pg_hba.conf
sudo systemctl restart postgresql-13
```

### Q3: 权限错误
**解决**:
```bash
# 修改数据目录权限
sudo chown -R postgres:postgres /var/lib/pgsql
```

---

## 📖 完整文档

- 数据库安装指南：`INSTALL_DATABASE.md`
- 部署文档：`docs/DEPLOYMENT_GUIDE.md`

---

**祝您使用愉快！** 🚀
