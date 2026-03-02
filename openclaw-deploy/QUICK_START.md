# OpenClaw 部署管理系统 - 快速开始

## 🚀 后端启动

```bash
cd openclaw-deploy/backend

# 安装 Go 依赖
go mod download

# 启动服务
go run cmd/server/main.go
```

服务将在 http://localhost:8080 启动

### API 测试

```bash
# 获取部署列表
curl http://localhost:8080/api/v1/deployments

# 创建部署
curl -X POST http://localhost:8080/api/v1/deployments \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u001","plan":"pro","instance_name":"test-001"}'

# 获取监控数据
curl http://localhost:8080/api/v1/deployments/1/metrics
```

---

## ⚛️ 前端启动

```bash
cd openclaw-deploy/frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

前端将在 http://localhost:5173 启动

---

## 📖 API 文档

### 部署管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/deployments | 获取部署列表 |
| POST | /api/v1/deployments | 创建部署 |
| GET | /api/v1/deployments/:id | 获取部署详情 |
| POST | /api/v1/deployments/:id/start | 启动部署 |
| POST | /api/v1/deployments/:id/stop | 停止部署 |
| GET | /api/v1/deployments/:id/metrics | 获取监控数据 |

### 监控告警

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/monitor/system | 系统状态 |
| GET | /api/v1/monitor/alerts | 告警列表 |

### 用户管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/users/me | 当前用户 |

---

## 🔧 配置

### 环境变量（.env）

```bash
# 服务器配置
PORT=8080
GIN_MODE=release

# 阿里云配置
ALIYUN_ACCESS_KEY_ID=your_access_key_id
ALIYUN_ACCESS_KEY_SECRET=your_access_key_secret
ALIYUN_REGION_ID=cn-shanghai
SECURITY_GROUP_ID=sg-xxx

# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=openclaw_deploy

# JWT 配置
JWT_SECRET=your_jwt_secret
```

---

## 📊 功能状态

### 已完成
- ✅ 项目架构
- ✅ 后端主程序
- ✅ 部署管理 API（Mock）
- ✅ React 主应用
- ✅ 部署管理页面

### 待完成
- [ ] 阿里云 ECS 集成
- [ ] 数据库集成
- [ ] 用户认证
- [ ] 真实监控数据
- [ ] 域名管理
- [ ] 自动化部署流程

---

**开发中... 敬请期待！** 🚀
