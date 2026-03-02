# OpenClaw 部署管理系统 - 开发指南

## 📦 依赖安装

### 后端

```bash
cd openclaw-deploy/backend

# 安装 Go 依赖
go mod download

# 验证安装
go build -o server cmd/server/main.go
```

### 前端

```bash
cd openclaw-deploy/frontend

# 安装 Node.js 依赖
npm install
```

---

## 🔧 环境配置

### 1. 阿里云配置

在 `.env` 文件中配置：

```bash
# 阿里云账号（访问 https://ram.console.aliyun.com 获取）
ALIYUN_ACCESS_KEY_ID=LTAI5t...
ALIYUN_ACCESS_KEY_SECRET=...
ALIYUN_REGION_ID=cn-shanghai

# 安全组 ID（访问 ECS 控制台获取）
SECURITY_GROUP_ID=sg-xxx
```

### 2. 数据库配置

```bash
# PostgreSQL 连接
DB_DSN=host=localhost user=postgres password=password dbname=openclaw_deploy port=5432 sslmode=disable
```

### 3. 服务器配置

```bash
PORT=8080
GIN_MODE=release
```

---

## 🚀 启动服务

### 后端

```bash
cd backend
go run cmd/server/main.go
```

服务将在 http://localhost:8080 启动

### 前端

```bash
cd frontend
npm run dev
```

前端将在 http://localhost:5173 启动

---

## 📖 API 使用

### 创建部署实例

```bash
curl -X POST http://localhost:8080/api/v1/deployments \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "u001",
    "plan": "pro",
    "instance_name": "test-001",
    "domain": "test.openclaw.cn"
  }'
```

### 启动实例

```bash
curl -X POST http://localhost:8080/api/v1/deployments/1/start
```

### 停止实例

```bash
curl -X POST http://localhost:8080/api/v1/deployments/1/stop
```

### 获取监控数据

```bash
curl http://localhost:8080/api/v1/deployments/1/metrics
```

---

## 🔨 开发流程

### 1. 添加新的 API Handler

在 `cmd/server/main.go` 中添加：

```go
func yourNewHandler(c *gin.Context) {
    // 处理逻辑
    c.JSON(200, gin.H{"message": "success"})
}
```

### 2. 集成阿里云 ECS

在 `pkg/aliyun/ecs.go` 中已有封装，直接使用：

```go
import "openclaw-deploy/pkg/aliyun"

// 创建实例
response, err := aliyun.CreateInstance(aliyun.CreateInstanceArgs{
    InstanceType: "ecs.n4.small",
    InstanceName: "my-instance",
    Bandwidth: 3,
    DiskSize: 40,
})
```

### 3. 数据库操作

在 `pkg/database/deployment.go` 中已有封装：

```go
import "openclaw-deploy/pkg/database"

// 创建部署
deployment := &database.Deployment{
    UserID: "u001",
    Plan: "pro",
    InstanceName: "test-001",
}
database.CreateDeployment(deployment)
```

---

## 📊 数据库迁移

### 自动迁移

程序启动时会自动执行：

```go
db.AutoMigrate(&Deployment{})
```

### 手动迁移

```bash
# 使用 GORM 工具
go run cmd/migrate/main.go
```

---

## 🧪 测试

### 单元测试

```bash
cd backend
go test ./...
```

### API 测试

```bash
# 使用 Postman 或 curl 测试 API
```

---

## 📝 待实现功能

### 后端
- [ ] 真实数据库集成
- [ ] JWT 用户认证
- [ ] 阿里云 ECS 真实调用
- [ ] 域名自动配置
- [ ] SSL 证书自动申请
- [ ] 监控数据真实获取
- [ ] 日志收集

### 前端
- [ ] 登录页面
- [ ] 部署创建表单
- [ ] 监控图表
- [ ] 日志查看器
- [ ] 配置管理

---

## 🔒 安全注意事项

1. **API Key 保护**: 不要将阿里云密钥提交到代码库
2. **密码加密**: 用户密码使用 bcrypt 加密
3. **JWT 认证**: 所有 API 需要认证
4. **输入验证**: 所有输入需要验证
5. **SQL 注入**: 使用 GORM 防止 SQL 注入

---

## 📚 参考文档

- [阿里云 ECS API](https://help.aliyun.com/document_detail/25484.html)
- [Gin 框架文档](https://gin-gonic.com/docs/)
- [GORM 文档](https://gorm.io/docs/)
- [React 文档](https://react.dev/)

---

**开发愉快！** 🚀
