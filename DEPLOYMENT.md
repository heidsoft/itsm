# ITSM 系统部署指南

## 环境要求

### 后端
- Go 1.21+
- PostgreSQL 14+ (推荐)
- Redis (可选，用于缓存)

### 前端
- Node.js 18+
- npm 9+

### 数据库
- 向量功能需要 pgvector 扩展

---

## 1. 数据库配置

### 1.1 创建数据库和用户

```bash
# 连接 PostgreSQL
psql -h localhost -U postgres

# 或使用本地有超级权限的用户
psql -h localhost -U heidsoft -d postgres
```

```sql
-- 创建数据库用户（如果需要）
CREATE ROLE dev WITH LOGIN PASSWORD '123456!@#$%^';
ALTER ROLE dev CREATEDB;

-- 创建数据库
CREATE DATABASE itsm OWNER dev;
GRANT ALL PRIVILEGES ON DATABASE itsm TO dev;
```

### 1.2 启用 pgvector 扩展（可选，用于 AI/RAG 功能）

```bash
psql -h localhost -U heidsoft -d itsm -c "CREATE EXTENSION IF NOT EXISTS vector;"
```

---

## 2. 环境配置

### 2.1 配置文件 `.env`

在 `itsm-backend` 目录下创建或编辑 `.env` 文件：

```bash
# 日志级别：debug, info, warn, error
LOG_LEVEL=info

# 数据库密码
DB_PASSWORD=你的数据库密码

# JWT 密钥（生产环境请使用强密钥）
JWT_SECRET=itsm-dev-jwt-secret-2024

# 可选：管理员密码
ADMIN_PASSWORD=admin123
```

### 2.2 配置文件 `config.yaml`

```yaml
database:
  host: localhost
  port: 5432
  user: dev
  password: "${DB_PASSWORD}"  # 从环境变量读取
  dbname: itsm
  sslmode: disable

server:
  port: 8090
  mode: release  # development, test, release

jwt:
  secret: "${JWT_SECRET}"
  expire_time: 900        # 15分钟
  refresh_expire_time: 604800  # 7天

log:
  level: "${LOG_LEVEL:info}"  # 默认 info
  path: ./logs
  max_size: 200
  max_backups: 20
  max_age: 90
  compress: true
  development: false

# 向量数据库（可选）
vector:
  enabled: true   # 需要先安装 pgvector 扩展
  dimension: 1536

llm:
  provider: openai
  model: gpt-4o-mini
  api_key: "${OPENAI_API_KEY:}"
```

---

## 3. 编译和运行

### 3.1 后端

```bash
cd itsm-backend

# 安装依赖
go mod download

# 编译
go build -o itsm-backend main.go

# 运行
./itsm-backend
```

或者直接运行：

```bash
cd itsm-backend
go run main.go
```

### 3.2 前端

```bash
cd itsm-frontend

# 安装依赖
npm install

# 开发模式
npm run dev

# 生产构建
npm run build
```

---

## 4. 验证部署

### 4.1 检查服务状态

```bash
# 健康检查
curl http://localhost:8090/api/v1/health

# 返回: {"status":"ok","timestamp":"2026-03-05T..."}
```

### 4.2 登录测试

```bash
curl -X POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# 返回包含 access_token 的 JSON
```

### 4.3 Swagger 文档

访问: http://localhost:8090/swagger/index.html

---

## 5. 常用 API 路径

| 功能 | 路径 | 方法 |
|------|------|------|
| 健康检查 | `/api/v1/health` | GET |
| 登录 | `/api/v1/auth/login` | POST |
| 用户详情 | `/api/v1/users/:id` | GET |
| 工单列表 | `/api/v1/tickets` | GET |
| 创建工单 | `/api/v1/tickets` | POST |
| 工单统计 | `/api/v1/tickets/stats` | GET |
| 服务目录 | `/api/v1/service-catalogs` | GET |
| 团队管理 | `/api/v1/org/teams` | GET |
| 部门管理 | `/api/v1/org/departments/tree` | GET |
| 审批工作流 | `/api/v1/approval-workflows` | GET |
| 系统配置 | `/api/v1/system-configs` | GET |
| 租户管理 | `/api/v1/tenants` | GET |
| 知识库 | `/api/v1/knowledge/articles` | GET |
| CMDB | `/api/v1/cmdb/cis` | GET |
| SLA | `/api/v1/sla/definitions` | GET |
| 角色权限 | `/api/v1/roles` | GET |

---

## 6. 默认账号

| 账号 | 密码 | 角色 |
|------|------|------|
| admin | admin123 | super_admin |

---

## 7. 常见问题

### 7.1 日志文件没有生成

检查 `.env` 文件中的 `LOG_LEVEL` 是否设置为 `info` 或更高级别：

```bash
LOG_LEVEL=info
```

### 7.2 pgvector 扩展权限不足

需要超级用户权限来创建扩展：

```bash
# 使用有超级权限的用户
psql -h localhost -U heidsoft -d itsm -c "CREATE EXTENSION IF NOT EXISTS vector;"
```

### 7.3 端口被占用

```bash
# 查找占用端口的进程
lsof -ti:8090

# 杀死进程
kill -9 <PID>
```

### 7.4 数据库连接失败

检查 `config.yaml` 中的数据库配置，确保用户名和密码正确。

### 7.5 权限不足错误

默认 admin 用户使用 `super_admin` 角色，拥有所有权限。如果遇到权限问题，检查用户角色是否正确。

---

## 8. Docker 部署（可选）

### 8.1 构建镜像

```bash
cd itsm-backend
docker build -t itsm-backend:latest .
```

### 8.2 运行容器

```bash
docker run -d \
  --name itsm-backend \
  -p 8090:8090 \
  -e DB_PASSWORD=你的密码 \
  -e JWT_SECRET=你的密钥 \
  -e LOG_LEVEL=info \
  itsm-backend:latest
```

---

## 9. 生产环境建议

1. **修改 JWT_SECRET** - 使用强随机字符串
2. **启用 HTTPS** - 使用反向代理（如 Nginx）
3. **配置防火墙** - 只开放必要端口
4. **定期备份数据库**
5. **监控日志** - 查看 `logs/itsm.log`
6. **配置 Redis** - 用于会话和缓存

---

## 10. 目录结构

```
itsm/
├── itsm-backend/          # 后端服务
│   ├── config/           # 配置文件
│   ├── config.yaml       # 主配置
│   ├── .env              # 环境变量
│   ├── logs/             # 日志目录
│   ├── service/          # 业务逻辑
│   ├── controller/       # HTTP 控制器
│   ├── ent/              # 数据库 ORM
│   └── main.go          # 入口
│
├── itsm-frontend/        # 前端应用
│   ├── src/
│   └── package.json
│
└── README.md
```

---

## 11. 后续步骤

- 配置 SMTP 邮件通知
- 配置 SMS 短信通知（阿里云/腾讯云）
- 配置 LDAP/AD 集成
- 配置企业微信/钉钉集成
- 定期更新安全补丁
