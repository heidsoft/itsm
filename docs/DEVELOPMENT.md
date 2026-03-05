# ITSM 开发环境搭建指南

**最后更新**: 2026-03-04
**版本**: v2.0

---

## 📋 目录

- [前置条件](#前置条件)
- [项目克隆](#项目克隆)
- [依赖安装](#依赖安装)
- [环境配置](#环境配置)
- [数据库设置](#数据库设置)
- [启动开发环境](#启动开发环境)
- [联调配置](#联调配置)
- [常用命令](#常用命令)
- [故障排除](#故障排除)

---

## 前置条件

### 开发机器要求

| 工具 | 最低版本 | 推荐版本 | 安装链接 |
|------|---------|---------|----------|
| Go | 1.21+ | 1.25+ | https://go.dev/dl/ |
| Node.js | 18+ | 22+ | https://nodejs.org/ |
| PostgreSQL | 14+ | 17 | https://www.postgresql.org/download/ |
| Redis | 6+ | 7+ | https://redis.io/download/ |
| Docker | 20+ (可选) | 24+ | https://www.docker.com/products/docker-desktop |
| Make | 任何版本 | - | macOS: `brew install make` |

### 验证安装

```bash
# 检查 Go
go version  # 应显示 go version go1.25.x

# 检查 Node.js
node --version  # 应显示 v22.x.x 或 v18.x.x
npm --version

# 检查 PostgreSQL
psql --version  # 应显示 psql (PostgreSQL) 14.x 或更高

# 检查 Redis
redis-cli --version

# 检查 Docker (可选)
docker --version
docker compose version

# 检查 Make
make --version
```

### macOS 开发环境快速安装

```bash
# 使用 Homebrew 安装所有依赖
brew install go node postgresql@17 redis docker docker-compose

# 启动服务
brew services start postgresql@17
brew services start redis

# 验证
brew services list
```

---

## 项目克隆

```bash
# 克隆主仓库
git clone https://github.com/heidsoft/itsm.git
cd itsm

# 查看分支
git branch -a

# 切换到开发分支
git checkout main

# 查看当前状态
git status
```

---

## 依赖安装

### 后端依赖

```bash
cd itsm-backend

# 下载 Go 模块
go mod download

# 验证依赖
go mod verify

# 整理依赖（如有需要）
go mod tidy

# 安装开发工具
go install github.com/air-verse/air@latest       # 热重载
go install github.com/swaggo/swag/cmd/swag@latest # API 文档
go install mvdan.cc/gofumpt@latest               # 代码格式化
```

### 前端依赖

```bash
cd itsm-frontend

# 使用 npm（推荐）
npm install

# 或使用 pnpm（更快）
corepack enable
pnpm install

# 或使用 yarn
yarn install

# 安装全局依赖（如需要）
npm install -g pm2 serve
```

---

## 环境配置

### 后端环境变量

```bash
cd itsm-backend

# 复制示例配置
cp .env.example .env

# 编辑 .env 文件
vim .env
```

**后端 .env 文件说明**:

```env
# ========== 服务器配置 ==========
SERVER_PORT=8090
SERVER_ENV=development
GIN_MODE=debug

# ========== 日志配置 ==========
LOG_LEVEL=debug
LOG_PATH=./logs

# ========== 数据库配置 ==========
DB_HOST=localhost
DB_PORT=5432
DB_USER=itsm
DB_PASSWORD=itsm_dev_password
DB_NAME=itsm
DB_SSLMODE=disable

# ========== Redis 配置 ==========
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# ========== JWT 配置 ==========
JWT_SECRET=dev-secret-key-please-change-in-production
JWT_EXPIRE=86400
JWT_REFRESH_EXPIRE=604800

# ========== 功能开关 ==========
ENABLE_SWAGGER=true
ENABLE_CORS=true

# ========== AI 配置（可选）==========
# OPENAI_API_KEY=your-api-key
# OLLAMA_BASE_URL=http://localhost:11434
```

### 前端环境变量

```bash
cd itsm-frontend

# 创建本地环境配置
cp .env.example .env.local

# 编辑 .env.local
vim .env.local
```

**前端 .env.local 文件说明**:

```env
# API 地址配置
NEXT_PUBLIC_API_URL=http://localhost:8090

# 超时设置
NEXT_PUBLIC_API_TIMEOUT=30000

# 功能开关
NEXT_PUBLIC_ENABLE_AI=true
NEXT_PUBLIC_ENABLE_BPMN=true

# 文件上传限制（字节）
NEXT_PUBLIC_MAX_UPLOAD_SIZE=10485760

# 开发模式
NODE_ENV=development
```

---

## 数据库设置

### 方式一：使用 Docker（推荐）

```bash
# 启动 PostgreSQL 和 Redis
docker compose up -d postgres redis

# 验证服务
docker ps | grep -E "postgres|redis"

# 查看日志
docker compose logs postgres

# 停止服务
docker compose down
```

### 方式二：本地安装

**PostgreSQL 初始化 (macOS)**:

```bash
# 启动 PostgreSQL
brew services start postgresql@17
brew services list

# 创建数据库
createdb itsm

# 创建用户
createuser -s itsm

# 设置密码
psql -c "ALTER USER itsm PASSWORD 'itsm_dev_password';"
```

**PostgreSQL 初始化 (Ubuntu/Debian)**:

```bash
# 启动 PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql

# 创建数据库和用户
sudo -u postgres createuser -s itsm
sudo -u postgres psql -c "ALTER USER itsm PASSWORD 'itsm_dev_password';"
sudo -u postgres createdb itsm

# 授予权限
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE itsm TO itsm;"
```

### 运行数据库迁移

```bash
cd itsm-backend

# 自动迁移（应用启动时执行）
go run main.go

# 手动执行迁移
go run -tags migrate main.go

# 或使用 Make
make db-migrate
```

验证数据库连接:

```bash
# 方式一：Docker
docker exec -it itsm-postgres psql -U itsm -d itsm

# 方式二：本地
psql -h localhost -U itsm -d itsm

# 查看表
\dt

# 退出
\q
```

---

## 启动开发环境

### 方式一：使用 Docker Compose（推荐）

```bash
cd itsm

# 启动所有服务
make dev-up
# 或: docker compose -f docker-compose.dev.yml up -d

# 查看日志
docker compose logs -f

# 停止服务
make dev-down
# 或: docker compose -f docker-compose.dev.yml down

# 查看状态
docker compose ps
```

**服务地址**:
- 前端: http://localhost:3000
- 后端: http://localhost:8090
- API 文档: http://localhost:8090/swagger
- PostgreSQL: localhost:5432
- Redis: localhost:6379

### 方式二：本地启动

**终端 1 - 启动数据库**:

```bash
# macOS
brew services start postgresql@17
brew services start redis

# Linux
sudo systemctl start postgresql
sudo systemctl start redis

# 或使用 Docker 单独启动
docker run -d --name itsm-postgres -e POSTGRES_PASSWORD=itsm_dev_password -p 5432:5432 postgres:17-alpine
docker run -d --name itsm-redis -p 6379:6379 redis:7-alpine
```

**终端 2 - 启动后端**:

```bash
cd itsm-backend

# 方式一：直接运行
go run main.go

# 方式二：使用 Make
make run

# 方式三：热重载模式（推荐）
make run-dev
# 或
air

# 服务地址: http://localhost:8090
```

**终端 3 - 启动前端**:

```bash
cd itsm-frontend

# 方式一：npm
npm run dev

# 方式二：Make
make frontend-run

# 服务地址: http://localhost:3000
```

---

## 联调配置

### API 代理配置

前端开发服务器已配置代理，无需额外设置。`.env.local` 中的 `NEXT_PUBLIC_API_URL` 应指向后端地址:

```env
NEXT_PUBLIC_API_URL=http://localhost:8090
```

### 跨域问题 (CORS)

后端已配置 CORS，默认允许 `http://localhost:3000`。如需修改，编辑 `itsm-backend/config/cors.go`:

```go
cors.Default() // 允许所有来源（开发环境）
// 或
cors.Config{
    AllowOrigins: []string{"http://localhost:3000", "http://localhost:3001"},
    AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
}
```

### 调试技巧

**后端调试**:

```bash
# 使用 Delve 调试器
go install github.com/go-delve/delve/cmd/dlv@latest

# 启动调试
dlv debug main.go

# 或使用 VS Code
# 在 IDE 中配置 launch.json，附加到进程
```

**前端调试**:

```bash
# 使用 Chrome DevTools
# - 打开浏览器 DevTools (F12)
# - Next.js 内置 Source Maps，可设置断点
# - React Dev Tools 浏览器扩展
# - Redux DevTools（如果使用 Redux）
```

---

## 常用命令

### 后端命令

```bash
# 运行
go run main.go

# 使用 Make
make run

# 热重载
make run-dev

# 构建
go build -o bin/itsm .

# 测试
go test ./... -v
go test ./service/ticket_service_test.go -v  # 单个测试
go test -run TestTicketServiceCreate -v      # 单个测试用例

# 覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 代码检查
go vet ./...
staticcheck ./...
gofumpt -w .

# 依赖管理
go mod tidy
go mod download
go mod verify

# API 文档
swag init -g main.go -o docs
```

### 前端命令

```bash
# 开发
npm run dev

# 构建
npm run build
npm run build:analyze  # 构建分析

# 生产运行
npm run start

# 测试
npm test
npm run test:watch
npm run test:coverage

# Lint & Format
npm run lint
npm run lint:fix
npm run type-check
npm run format

# 环境检查
node -v
npm -v
```

### 数据库命令

```bash
# 备份
pg_dump -U itsm itsm > backup_$(date +%Y%m%d).sql

# 恢复
psql -U itsm itsm < backup_20260304.sql

# 查看连接
psql -U itsm -d itsm -c "SELECT pid, state, query FROM pg_stat_activity;"

# 清理数据库（⚠️ 谨慎！）
dropdb itsm
createdb itsm

# 使用 Docker
docker exec -it itsm-postgres psql -U itsm -d itsm
```

### Docker 命令

```bash
# 构建镜像
docker build -t itsm-backend:dev -f Dockerfile.backend ./itsm-backend

# 启动服务
docker compose -f docker-compose.dev.yml up -d

# 查看日志
docker compose logs -f backend
docker compose logs -f frontend

# 进入容器
docker exec -it itsm-backend sh
docker exec -it itsm-frontend sh
docker exec -it itsm-postgres psql -U itsm

# 停止并清理
docker compose down -v  # -v 删除卷
```

---

## 故障排除

### 问题 1: 数据库连接失败

**错误信息**: `dial tcp 127.0.0.1:5432: connect: connection refused`

**解决方案**:

1. 确认 PostgreSQL 是否运行:
   ```bash
   # macOS
   brew services list | grep postgres

   # Linux
   sudo systemctl status postgresql

   # Docker
   docker ps | grep postgres
   ```

2. 启动 PostgreSQL:
   ```bash
   brew services start postgresql@17  # macOS
   sudo systemctl start postgresql    # Linux
   ```

3. 检查端口占用:
   ```bash
   lsof -i :5432
   ```

4. 验证连接:
   ```bash
   psql -h localhost -U itsm -d itsm
   ```

---

### 问题 2: Redis 连接超时

**错误信息**: `dial tcp 127.0.0.1:6379: i/o timeout`

**解决方案**:

1. 启动 Redis:
   ```bash
   brew services start redis      # macOS
   sudo systemctl start redis     # Linux
   ```

2. 测试连接:
   ```bash
   redis-cli ping  # 应返回 PONG
   ```

---

### 问题 3: 端口冲突

**错误信息**: `listen tcp :8090: bind: address already in use`

**解决方案**:

1. 查找占用端口的进程:
   ```bash
   lsof -i :8090
   lsof -i :3000
   ```

2. 终止进程或修改端口:
   ```bash
   kill -9 <PID>
   # 或修改 .env 中的 SERVER_PORT
   ```

---

### 问题 4: Go 依赖下载慢

**解决方案**:

1. 配置 Go 代理:
   ```bash
   go env -w GOPROXY=https://goproxy.cn,direct
   go env -w GOSUMDB=sum.golang.google.cn
   ```

2. 清理模块缓存:
   ```bash
   go clean -modcache
   go mod download
   ```

---

### 问题 5: 前端 npm install 失败

**错误信息**: `npm ERR! code ENOTFOUND` 或网络超时

**解决方案**:

1. 配置 npm 镜像:
   ```bash
   npm config set registry https://registry.npmmirror.com/
   ```

2. 清理缓存:
   ```bash
   npm cache clean --force
   rm -rf node_modules package-lock.json
   npm install
   ```

3. 使用 pnpm（更快）:
   ```bash
   corepack enable
   pnpm install
   ```

---

### 问题 6: 数据库迁移失败

**错误信息**: `ERROR: relation "users" already exists`

**解决方案**:

1. 检查是否已迁移:
   ```bash
   docker exec -it itsm-postgres psql -U itsm -d itsm -c "\dt"
   ```

2. 如需重新迁移:
   ```bash
   # 删除并重建数据库
   docker exec -it itsm-postgres psql -U postgres -c "DROP DATABASE itsm;"
   docker exec -it itsm-postgres psql -U postgres -c "CREATE DATABASE itsm;"

   # 重新运行迁移
   cd itsm-backend && go run main.go
   ```

---

### 问题 7: CORS 错误

**错误信息**: `Access to fetch at '...' from origin 'http://localhost:3000' has been blocked by CORS policy`

**解决方案**:

确认后端 CORS 配置包含前端地址。编辑 `itsm-backend/config/cors.go`:

```go
cors.Config{
    AllowOrigins: []string{
        "http://localhost:3000",
        "http://localhost:3001",
    },
    AllowCredentials: true,
}
```

重启后端服务。

---

### 问题 8: 无法编译（版本不匹配）

**错误信息**: `go: go.mod file requires go 1.25`

**解决方案**:

1. 升级 Go 版本至 1.25+:
   ```bash
   # 使用 asdf
   asdf install golang 1.25.0
   asdf global golang 1.25.0

   # 或直接下载
   # https://go.dev/dl/

   go version  # 确认版本
   ```

2. 清理并重新下载依赖:
   ```bash
   go clean -modcache
   go mod download
   ```

---

## 获取帮助

- 📖 查看 [完整文档](./README.md)
- 📖 查看 [部署指南](./DEPLOYMENT.md)
- 📖 查看 [配置参考](./CONFIGURATION.md)
- 📖 查看 [数据库](./DATABASE.md)
- 📖 查看 [日志管理](./LOGGING.md)
- 🐛 提交 [Issue](https://github.com/heidsoft/itsm/issues)
- 💬 加入 [Discussions](https://github.com/heidsoft/itsm/discussions)

---

**文档维护**: ITSM 开发团队
**最后更新**: 2026-03-04
