# ITSM 系统部署指南

本文档介绍如何部署和运行 ITSM 系统。

## 目录

- [前置要求](#前置要求)
- [快速开始](#快速开始)
- [Docker 部署](#docker-部署)
- [本地开发](#本地开发)
- [生产环境部署](#生产环境部署)
- [数据库管理](#数据库管理)
- [故障排查](#故障排查)

---

## 前置要求

### Docker 部署
- Docker 20.10+
- Docker Compose 2.0+

### 本地开发
- Go 1.21+
- Node.js 18+
- PostgreSQL 14+
- Redis 7+

---

## 快速开始

### 1. 克隆项目

```bash
git clone <repository-url>
cd itsm
```

### 2. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件，根据实际情况修改配置
```

### 3. 启动服务

#### Docker 方式（推荐）

```bash
./scripts/deploy.sh docker
```

#### 本地开发方式

```bash
./scripts/deploy.sh dev
```

### 4. 访问系统

- **后端 API**: http://localhost:8080
- **前端应用**: http://localhost:3000
- **API 文档**: http://localhost:8080/swagger/index.html
- **Grafana 监控**: http://localhost:3001 (admin/admin_password_changeme)

---

## Docker 部署

### 首次部署

```bash
# 1. 复制配置文件
cp .env.example .env

# 2. 启动所有服务
docker-compose up -d

# 3. 查看日志
docker-compose logs -f

# 4. 初始化数据库（首次运行）
./scripts/deploy.sh init
```

### 服务列表

| 服务名称 | 端口 | 说明 |
|---------|------|------|
| postgres | 5432 | PostgreSQL 数据库 |
| redis | 6379 | Redis 缓存 |
| minio | 9000, 9001 | 对象存储 |
| itsm-backend | 8080 | 后端 API 服务 |
| itsm-frontend | 3000 | 前端应用 |
| nginx | 80, 443 | 反向代理 |
| prometheus | 9090 | 监控指标收集 |
| grafana | 3001 | 监控可视化 |

### 常用命令

```bash
# 启动所有服务
docker-compose up -d

# 停止所有服务
docker-compose down

# 重启服务
docker-compose restart

# 查看日志
docker-compose logs -f [service-name]

# 查看服务状态
docker-compose ps

# 清理所有数据（谨慎使用！）
docker-compose down -v
```

---

## 本地开发

### 环境准备

```bash
# 1. 安装 Go 依赖
cd itsm-backend
go mod download

# 2. 安装前端依赖
cd ../itsm-frontend
npm install
```

### 启动数据库

```bash
# 使用 Docker 启动 PostgreSQL 和 Redis
docker-compose up -d postgres redis

# 或使用本地 PostgreSQL
# 确保本地 PostgreSQL 正在运行并配置好
```

### 初始化数据库

```bash
cd itsm-backend
go run -tags migrate main.go
```

### 启动后端服务

```bash
cd itsm-backend
go run main.go
```

### 启动前端服务

```bash
cd itsm-frontend
npm run dev
```

### 开发模式一键启动

```bash
./scripts/deploy.sh dev
```

---

## 生产环境部署

### 1. 环境配置

```bash
# 修改 .env 文件中的生产环境配置
SERVER_MODE=release
LOG_LEVEL=info
DB_PASSWORD=<强密码>
JWT_SECRET=<强随机字符串>
```

### 2. 使用构建产物部署

#### 后端构建

```bash
cd itsm-backend
go build -o itsm main.go
./itsm
```

#### 前端构建

```bash
cd itsm-frontend
npm run build
# 将 .next 目录部署到 Nginx 或 Node.js 服务器
```

### 3. 使用 Docker 部署

```bash
# 使用 release 镜像
docker-compose -f docker-compose.prod.yml up -d
```

### 4. 使用 systemd 管理（Linux）

创建 `/etc/systemd/system/itsm.service`:

```ini
[Unit]
Description=ITSM Backend Service
After=network.target postgresql.service

[Service]
Type=simple
User=itsm
WorkingDirectory=/opt/itsm
ExecStart=/opt/itsm/itsm
Restart=always
RestartSec=10
Environment="SERVER_MODE=release"
Environment="LOG_LEVEL=info"

[Install]
WantedBy=multi-user.target
```

```bash
systemctl enable itsm
systemctl start itsm
systemctl status itsm
```

---

## 数据库管理

### 运行迁移

```bash
cd itsm-backend

# 全新数据库迁移（删除旧数据库）
go run -tags migrate main.go

# 普通迁移（保留现有数据）
# 首次启动服务器时会自动运行
```

### 种子数据

种子数据在服务器启动时自动运行，包括：

- 默认租户 (Default Tenant)
- 默认用户 (admin/admin123, user1/user123, security1/security123)
- 默认部门 (IT部门, 运维部门, 客服部门)
- 默认团队 (一线支持, 二线支持)
- 默认角色 (管理员, 工程师, 用户)
- 示例资产 (笔记本电脑, 服务器, 网络设备等)
- 示例许可证 (IDE, 数据库等)
- 示例发布记录

### 手动执行 SQL

```bash
# 连接到数据库
psql -h localhost -p 5432 -U dev -d itsm_cmdb

# 执行 SQL 文件
\i sql/seed_test_data.sql
```

### 数据库备份

```bash
# 备份
pg_dump -h localhost -U dev itsm_cmdb > backup.sql

# 恢复
psql -h localhost -U dev itsm_cmdb < backup.sql
```

---

## 故障排查

### 数据库连接失败

1. 检查 PostgreSQL 是否运行
```bash
docker-compose ps postgres
```

2. 检查连接配置
```bash
# 确认 .env 中的 DB_HOST, DB_PORT, DB_USER, DB_PASSWORD 正确
```

3. 查看日志
```bash
docker-compose logs postgres
```

### 服务启动失败

1. 查看服务日志
```bash
docker-compose logs itsm-backend
```

2. 检查端口占用
```bash
lsof -i :8080
```

3. 检查环境变量
```bash
docker-compose config
```

### 前端无法访问后端

1. 确认后端服务运行
```bash
curl http://localhost:8080/health
```

2. 检查前端 API URL 配置
```bash
# 确认 .env 中的 NEXT_PUBLIC_API_URL 正确
```

### 权限问题

```bash
# 为部署脚本添加执行权限
chmod +x scripts/deploy.sh
```

---

## 监控和日志

### 查看应用日志

```bash
# Docker 方式
docker-compose logs -f itsm-backend

# 本地方式
tail -f itsm-backend/logs/app.log
```

### Prometheus 监控

访问 http://localhost:9090 查看 Prometheus 指标。

### Grafana 仪表板

访问 http://localhost:3001 查看 Grafana 仪表板。

默认账号: `admin`
默认密码: `admin_password_changeme`（请修改）

---

## 安全建议

1. **修改默认密码**: 修改数据库密码、JWT 密钥等敏感信息
2. **使用 HTTPS**: 生产环境配置 SSL/TLS 证书
3. **防火墙规则**: 只开放必要的端口
4. **定期备份**: 设置数据库定期备份
5. **更新依赖**: 定期更新系统依赖和安全补丁

---

## 联系方式

如有问题，请提交 Issue 或联系维护团队。
