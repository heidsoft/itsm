# OpenClaw 部署管理系统

基于 Go + React 的完整部署管理系统，支持阿里云 ECS 自动化部署。

## 🏗️ 技术栈

### 后端
- **语言**: Go 1.21+
- **框架**: Gin
- **数据库**: PostgreSQL
- **ORM**: GORM
- **云 SDK**: 阿里云 SDK

### 前端
- **框架**: React 18
- **路由**: React Router 6
- **状态管理**: React Hooks
- **UI 组件**: 自定义 + Font Awesome
- **构建工具**: Vite

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
│   │   └── utils/               # 工具函数
│   └── api/                     # API 文档
├── frontend/
│   ├── src/
│   │   ├── components/          # React 组件
│   │   ├── pages/               # 页面
│   │   ├── services/            # API 服务
│   │   └── styles/              # 样式
│   └── public/                  # 静态资源
└── docs/                        # 文档
```

## 🚀 快速开始

### 后端启动

```bash
cd backend

# 安装依赖
go mod download

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件配置数据库和阿里云信息

# 启动服务
go run cmd/server/main.go
```

### 前端启动

```bash
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

## 📖 API 文档

### 部署管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/deployments | 获取部署列表 |
| POST | /api/v1/deployments | 创建部署 |
| GET | /api/v1/deployments/:id | 获取部署详情 |
| PUT | /api/v1/deployments/:id | 更新部署 |
| DELETE | /api/v1/deployments/:id | 删除部署 |
| POST | /api/v1/deployments/:id/start | 启动部署 |
| POST | /api/v1/deployments/:id/stop | 停止部署 |
| GET | /api/v1/deployments/:id/metrics | 获取监控数据 |
| GET | /api/v1/deployments/:id/logs | 获取日志 |

### 监控告警

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/monitor/system | 系统状态 |
| GET | /api/v1/monitor/alerts | 告警列表 |
| POST | /api/v1/monitor/alerts/:id/ack | 确认告警 |

### 用户管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/users/me | 当前用户 |
| GET | /api/v1/users | 用户列表 |
| PUT | /api/v1/users/:id | 更新用户 |

## 🔧 配置说明

### 环境变量

```bash
# 服务器配置
PORT=8080
GIN_MODE=release

# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=openclaw_deploy

# 阿里云配置
ALIYUN_ACCESS_KEY_ID=your_access_key_id
ALIYUN_ACCESS_KEY_SECRET=your_access_key_secret
ALIYUN_REGION_ID=cn-shanghai
SECURITY_GROUP_ID=sg-xxx

# JWT 配置
JWT_SECRET=your_jwt_secret
JWT_EXPIRE=24h
```

## 📊 功能特性

### 部署管理
- ✅ 创建/删除部署
- ✅ 启动/停止实例
- ✅ 配置管理
- ✅ 日志查看

### 监控告警
- ✅ 实时监控（CPU/内存/磁盘）
- ✅ 性能指标（QPS/响应时间）
- ✅ 告警通知
- ✅ 告警确认

### 用户管理
- ✅ 用户认证
- ✅ 权限管理
- ✅ 账号管理

### 域名管理
- ✅ 域名绑定
- ✅ SSL 证书管理
- ✅ DNS 配置

## 🌐 访问地址

- **前端**: http://localhost:5173
- **后端 API**: http://localhost:8080/api/v1
- **API 文档**: http://localhost:8080/swagger

## 📝 开发计划

### 已完成
- [x] 项目架构设计
- [x] 后端主程序
- [x] 阿里云 SDK 封装
- [x] 部署管理 API
- [x] React 主应用
- [x] 部署管理页面

### 待完成
- [ ] 数据库模型
- [ ] 用户认证
- [ ] 监控告警
- [ ] 域名管理
- [ ] 自动化部署流程
- [ ] 单元测试
- [ ] 集成测试

## 📄 许可证

MIT License
