# ITSM 配置参考指南

**最后更新**: 2026-03-04
**版本**: v1.0

---

## 📋 目录

- [环境变量概述](#环境变量概述)
- [后端配置](#后端配置)
- [前端配置](#前端配置)
- [数据库配置](#数据库配置)
- [Redis 配置](#redis-配置)
- [AI 服务配置](#ai-服务配置)
- [存储配置](#存储配置)
- [监控配置](#监控配置)
- [功能开关](#功能开关)

---

## 环境变量概述

ITSM 系统使用环境变量进行配置，支持以下配置方式（优先级从高到低）：

1. **环境变量文件** (`.env`, `.env.local`, `.env.prod`)
2. **命令行参数**
3. **配置文件** (`config.yaml`)

### 配置文件位置

```
itsm-backend/
├── .env                 # 开发环境配置（本地覆盖）
├── .env.example        # 环境变量示例
├── config.yaml         # 配置文件（可选）
└── config/
    └── config.go       # 配置加载逻辑
```

---

## 后端配置

### 必需配置

| 变量名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `SERVER_PORT` | int | 8090 | 服务端口 |
| `SERVER_ENV` | string | development | 运行环境 |
| `DB_HOST` | string | localhost | 数据库主机 |
| `DB_PORT` | int | 5432 | 数据库端口 |
| `DB_USER` | string | - | 数据库用户名 |
| `DB_PASSWORD` | string | - | 数据库密码 |
| `DB_NAME` | string | - | 数据库名称 |

### 日志配置

| 变量名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `LOG_LEVEL` | string | debug | 日志级别 |
| `LOG_PATH` | string | ./logs | 日志目录 |
| `LOG_MAX_SIZE` | int | 100 | 单文件最大 MB |
| `LOG_MAX_AGE` | int | 30 | 文件保留天数 |
| `LOG_COMPRESS` | bool | true | 是否压缩 |

### 日志级别说明

| 级别 | 使用场景 |
|------|----------|
| `debug` | 开发调试 |
| `info` | 正常运行时 |
| `warn` | 警告信息 |
| `error` | 错误信息（生产推荐） |

### JWT 配置

| 变量名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `JWT_SECRET` | string | - | JWT 密钥（必填） |
| `JWT_EXPIRE` | int | 86400 | Token 有效期（秒） |
| `JWT_REFRESH_EXPIRE` | int | 604800 | 刷新 Token 有效期 |

### 服务配置

| 变量名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `GIN_MODE` | string | debug | Gin 运行模式 |
| `READ_TIMEOUT` | int | 60 | 读取超时（秒） |
| `WRITE_TIMEOUT` | int | 60 | 写入超时（秒） |
| `MAX_HEADER_BYTES` | int | 1048576 | 最大头大小 |

---

## 前端配置

### 环境变量

> **注意**: 前端环境变量必须以 `NEXT_PUBLIC_` 开头才能在浏览器中访问

| 变量名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `NEXT_PUBLIC_API_URL` | string | http://localhost:8090 | API 基础地址 |
| `NEXT_PUBLIC_API_TIMEOUT` | int | 30000 | 请求超时（毫秒） |
| `NEXT_PUBLIC_ENABLE_AI` | boolean | true | 启用 AI 功能 |
| `NEXT_PUBLIC_ENABLE_BPMN` | boolean | true | 启用工作流 |
| `NEXT_PUBLIC_MAX_UPLOAD_SIZE` | int | 10485760 | 上传文件大小限制 |

---

## 数据库配置

### PostgreSQL 配置

| 变量名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `DB_HOST` | string | localhost | 数据库主机 |
| `DB_PORT` | int | 5432 | 数据库端口 |
| `DB_USER` | string | - | 用户名 |
| `DB_PASSWORD` | string | - | 密码 |
| `DB_NAME` | string | - | 数据库名 |
| `DB_SSLMODE` | string | disable | SSL 模式 |
| `DB_MAX_IDLE_CONNS` | int | 10 | 最大空闲连接 |
| `DB_MAX_OPEN_CONNS` | int | 100 | 最大开放连接 |
| `DB_CONN_MAX_LIFETIME` | int | 3600 | 连接生命周期（秒） |

### SSL 模式

| 值 | 说明 |
|---|------|
| `disable` | 不使用 SSL（开发环境） |
| `require` | 使用 SSL（生产推荐） |
| `verify-full` | 验证证书 |

### 连接字符串格式

```env
# 标准格式
DATABASE_URL=postgres://user:password@host:5432/dbname?sslmode=disable

# 或使用单独配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=itsm
DB_PASSWORD=your_password
DB_NAME=itsm_db
DB_SSLMODE=disable
```

---

## Redis 配置

| 变量名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `REDIS_HOST` | string | localhost | Redis 主机 |
| `REDIS_PORT` | int | 6379 | Redis 端口 |
| `REDIS_PASSWORD` | string | - | Redis 密码 |
| `REDIS_DB` | int | 0 | 数据库编号 |
| `REDIS_POOL_SIZE` | int | 10 | 连接池大小 |

### 连接格式

```env
# 标准格式
REDIS_URL=redis://:password@host:6379/0

# 或使用单独配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

---

## AI 服务配置

### LLM 提供商配置

| 变量名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `LLM_PROVIDER` | string | openai | LLM 提供商 |
| `LLM_MODEL` | string | gpt-4 | 模型名称 |
| `OPENAI_API_KEY` | string | - | OpenAI API Key |
| `OPENAI_ORG_ID` | string | - | OpenAI 组织 ID |
| `OLLAMA_BASE_URL` | string | http://localhost:11434 | Ollama 地址 |

### 提供商选项

| 值 | 说明 |
|---|------|
| `openai` | OpenAI GPT 模型 |
| `azure` | Azure OpenAI |
| `local` | 本地 Ollama |
| `anthropic` | Anthropic Claude |

### 向量存储配置

| 变量名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `EMBEDDING_PROVIDER` | string | openai | 向量嵌入提供商 |
| `EMBEDDING_MODEL` | string | text-embedding-ada-002 | 嵌入模型 |
| `VECTOR_DB` | string | chroma | 向量数据库 |
| `CHROMA_PATH` | string | ./data/chroma | Chroma 数据目录 |

---

## 存储配置

### MinIO 对象存储

| 变量名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `MINIO_ENDPOINT` | string | localhost:9000 | MinIO 端点 |
| `MINIO_ACCESS_KEY` | string | minioadmin | 访问密钥 |
| `MINIO_SECRET_KEY` | string | minioadmin123 | 秘密密钥 |
| `MINIO_BUCKET` | string | itsm | 存储桶名称 |
| `MINIO_USE_SSL` | bool | false | 使用 SSL |

### 文件上传

| 变量名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `UPLOAD_PATH` | string | ./uploads | 上传文件目录 |
| `MAX_UPLOAD_SIZE` | int | 10485760 | 最大上传大小（字节） |
| `ALLOWED_EXTENSIONS` | string | jpg,jpeg,png,pdf,doc,docx | 允许的扩展名 |

---

## 监控配置

### 指标收集

| 变量名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `ENABLE_METRICS` | bool | true | 启用指标收集 |
| `METRICS_PORT` | int | 9090 | 指标端口 |
| `PROMETHEUS_ENABLED` | bool | true | Prometheus 格式 |

### 健康检查

| 变量名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `HEALTH_CHECK_PATH` | string | /health | 健康检查路径 |
| `HEALTH_CHECK_INTERVAL` | int | 30 | 检查间隔（秒） |

---

## 功能开关

| 变量名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `ENABLE_SWAGGER` | bool | true | 启用 Swagger 文档 |
| `ENABLE_CORS` | bool | true | 启用 CORS |
| `ENABLE_AI` | bool | true | 启用 AI 功能 |
| `ENABLE_RAG` | bool | true | 启用 RAG 知识库 |
| `ENABLE_SLA` | bool | true | 启用 SLA 监控 |
| `ENABLE_ESCALATION` | bool | true | 启用升级机制 |

---

## 配置示例

### 开发环境 (.env)

```bash
# ========== 服务器 ==========
SERVER_PORT=8090
SERVER_ENV=development
GIN_MODE=debug
LOG_LEVEL=debug

# ========== 数据库 ==========
DB_HOST=localhost
DB_PORT=5432
DB_USER=itsm
DB_PASSWORD=itsm_dev_password
DB_NAME=itsm
DB_SSLMODE=disable

# ========== Redis ==========
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# ========== JWT ==========
JWT_SECRET=dev-secret-key-change-in-production
JWT_EXPIRE=86400

# ========== 功能开关 ==========
ENABLE_SWAGGER=true
ENABLE_CORS=true
ENABLE_AI=true
```

### 生产环境 (.env.prod)

```bash
# ========== 服务器 ==========
SERVER_PORT=8090
SERVER_ENV=production
GIN_MODE=release
LOG_LEVEL=error

# ========== 数据库 ==========
DB_HOST=postgres.internal
DB_PORT=5432
DB_USER=itsm_prod
DB_PASSWORD=VeryStrongPassword!@#$%
DB_NAME=itsm_prod
DB_SSLMODE=require

# ========== Redis ==========
REDIS_HOST=redis.internal
REDIS_PORT=6379
REDIS_PASSWORD=VeryStrongRedisPassword!@#$%
REDIS_DB=0

# ========== JWT ==========
JWT_SECRET=VeryLongRandomSecretKeyMinimum32Characters!
JWT_EXPIRE=3600

# ========== 功能开关 ==========
ENABLE_SWAGGER=false
ENABLE_CORS=true
ENABLE_AI=true
```

---

## 配置验证

启动前验证配置：

```bash
# 后端配置验证
cd itsm-backend
go run main.go --verify-config

# 或检查环境变量
go run main.go env list
```

---

**文档维护**: ITSM 开发团队
**最后更新**: 2026-03-04
