# OpenClaw 部署管理系统

完整的 Go + React 部署管理系统，支持阿里云 ECS 自动化部署。

## 🏗️ 技术栈

### 后端
- **语言**: Go 1.21+
- **框架**: Gin
- **数据库**: PostgreSQL + GORM
- **云 SDK**: 阿里云 ECS SDK
- **认证**: JWT

### 前端
- **框架**: React 18
- **路由**: React Router 6
- **HTTP**: Axios
- **构建**: Vite
- **UI**: 自定义组件 + Font Awesome

## 📦 项目结构

```
openclaw-deploy/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go          # 主程序入口
│   ├── internal/
│   │   ├── handlers/            # HTTP 处理器
│   │   ├── middleware/          # 中间件
│   │   └── models/              # 数据模型
│   ├── pkg/
│   │   ├── aliyun/              # 阿里云 SDK 封装
│   │   ├── database/            # 数据库操作
│   │   └── auth/                # 认证
│   ├── config/                  # 配置
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── components/          # React 组件
│   │   ├── pages/               # 页面
│   │   ├── services/            # API 服务
│   │   ├── hooks/               # 自定义 Hooks
│   │   └── styles/              # 样式
│   └── package.json
└── scripts/                     # 脚本工具
```

## 🚀 快速开始

### 1. 环境准备

**安装 Go**:
```bash
# 下载并安装 Go 1.21+
```

**安装 Node.js**:
```bash
# 下载并安装 Node.js 18+
```

**安装 PostgreSQL**:
```bash
# 安装 PostgreSQL 14+
createdb openclaw_deploy
```

### 2. 后端启动

```bash
cd openclaw-deploy/backend

# 安装依赖
go mod download

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件

# 启动服务
go run cmd/server/main.go
```

服务将在 http://localhost:8080 启动

### 3. 前端启动

```bash
cd openclaw-deploy/frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

前端将在 http://localhost:5173 启动

## 🔧 配置说明

### 环境变量 (.env)

```bash
# 服务器配置
PORT=8080
GIN_MODE=release

# 数据库配置
DB_DSN=host=localhost user=postgres password=password dbname=openclaw_deploy port=5432

# 阿里云配置
ALIYUN_ACCESS_KEY_ID=LTAI5t...
ALIYUN_ACCESS_KEY_SECRET=...
ALIYUN_REGION_ID=cn-shanghai
SECURITY_GROUP_ID=sg-xxx

# JWT 配置
JWT_SECRET=your_jwt_secret
```

## 📖 API 文档

### 部署管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/deployments | 获取部署列表 |
| POST | /api/v1/deployments | 创建部署 |
| GET | /api/v1/deployments/:id | 获取详情 |
| PUT | /api/v1/deployments/:id | 更新部署 |
| DELETE | /api/v1/deployments/:id | 删除部署 |
| POST | /api/v1/deployments/:id/start | 启动部署 |
| POST | /api/v1/deployments/:id/stop | 停止部署 |
| GET | /api/v1/deployments/:id/metrics | 获取监控数据 |
| GET | /api/v1/deployments/:id/logs | 获取日志 |

### 用户管理

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/auth/login | 用户登录 |
| GET | /api/v1/users/me | 当前用户 |
| GET | /api/v1/users | 用户列表 |
| PUT | /api/v1/users/:id | 更新用户 |

### 监控告警

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/monitor/system | 系统状态 |
| GET | /api/v1/monitor/alerts | 告警列表 |
| POST | /api/v1/monitor/alerts/:id/ack | 确认告警 |

## 🌐 访问地址

- **前端**: http://localhost:5173
- **后端 API**: http://localhost:8080/api/v1
- **健康检查**: http://localhost:8080/health

## 📊 功能特性

### 已实现
- ✅ 部署实例管理（CRUD）
- ✅ 阿里云 ECS 集成
- ✅ 实时监控数据
- ✅ 启动/停止控制
- ✅ 用户认证（JWT）
- ✅ 响应式 UI

### 待实现
- [ ] 域名自动配置
- [ ] SSL 证书申请
- [ ] 日志查看
- [ ] 告警通知
- [ ] 自动化部署流程

## 📝 开发指南

详见 [DEVELOPMENT_GUIDE.md](DEVELOPMENT_GUIDE.md)

## 📄 许可证

MIT License
