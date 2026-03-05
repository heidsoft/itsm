# ITSM 程序编译与启动指南

**最后更新**: 2026-03-04
**版本**: v1.0

---

## 📋 目录

- [编译概述](#编译概述)
- [后端编译](#后端编译)
- [前端编译](#前端编译)
- [程序启动](#程序启动)
- [Docker 构建](#docker-构建)
- [生产环境部署](#生产环境部署)
- [启动验证](#启动验证)
- [常见问题](#常见问题)

---

## 编译概述

ITSM 项目包含两个独立的应用：

| 应用 | 目录 | 技术栈 | 输出 |
|------|------|--------|------|
| 后端 | `itsm-backend/` | Go/Gin | 二进制文件 |
| 前端 | `itsm-frontend/` | Next.js | 静态资源 |

---

## 后端编译

### 本地编译

#### 方式一：直接编译

```bash
cd itsm-backend

# 清理旧构建
go clean

# 编译
go build -o bin/itsm main.go

# 验证
./bin/itsm --version
```

#### 方式二：使用 Makefile

```bash
# 编译后端
make build-backend

# 输出路径: itsm-backend/bin/itsm
```

### 编译参数

```bash
# 指定输出文件名
go build -o itsm-server main.go

# 编译并压缩
go build -ldflags="-s -w" -o itsm-server main.go

# 添加版本信息
go build -ldflags="-X main.Version=1.0.0 -X main.BuildTime=$(date -u +%Y-%m-%d-%H:%M:%S)" -o itsm-server main.go
```

### 交叉编译

```bash
# 编译 Linux AMD64
GOOS=linux GOARCH=amd64 go build -o itsm-linux-amd64 main.go

# 编译 macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o itsm-darwin-arm64 main.go

# 编译 Windows
GOOS=windows GOARCH=amd64 go build -o itsm.exe main.go
```

### 编译优化

```bash
# 生产优化编译
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o itsm main.go

# 查看依赖
go list -deps
go mod graph
```

---

## 前端编译

### 开发构建

```bash
cd itsm-frontend

# 安装依赖（如需要）
npm install

# 开发模式启动（带热更新）
npm run dev
# 访问: http://localhost:3000
```

### 生产构建

```bash
# 使用 npm
npm run build

# 使用 Make
make frontend-build

# 输出目录: itsm-frontend/.next/
```

### 构建参数

```bash
# 构建分析
ANALYZE=true npm run build

# 自定义输出目录
NEXT_OUTDIR=dist npm run build

# 不使用 Typescript
SKIP_TYPECHECK=true npm run build
```

### 输出产物

```
.next/
├── static/          # 静态资源
│   └── chunks/     # JS 代码块
├── server/         # 服务端文件
├── image/          # 优化图片
└── prbuild-manifest.json
```

---

## 程序启动

### 后端启动

#### 开发模式（推荐）

```bash
# 方式一：直接运行
cd itsm-backend
go run main.go

# 方式二：使用 Make
make run

# 方式三：热重载模式（使用 air）
make run-dev
# 或
air
```

#### 生产模式

```bash
# 运行编译后的二进制
cd itsm-backend
./bin/itsm

# 指定配置文件
./bin/itsm --config=/path/to/config.yaml

# 后台运行
nohup ./bin/itsm > /var/log/itsm.log 2>&1 &

# 使用 systemd
sudo cp itsm.service /etc/systemd/system/
sudo systemctl start itsm
```

### 前端启动

#### 开发模式

```bash
cd itsm-frontend

# 方式一：npm
npm run dev

# 方式二：Make
make frontend-run

# 访问: http://localhost:3000
```

#### 生产模式

```bash
# 构建后启动
npm run build
npm run start

# 或使用 PM2
pm2 start npm --name "itsm-frontend" -- run start
```

### Docker 启动

#### 开发环境

```bash
# 启动所有服务
docker compose up -d

# 查看日志
docker compose logs -f

# 查看特定服务
docker compose logs -f backend
docker compose logs -f frontend

# 停止服务
docker compose down
```

#### 生产环境

```bash
# 拉取最新镜像
docker compose -f docker-compose.prod.yml pull

# 启动服务
docker compose -f docker-compose.prod.yml up -d

# 查看状态
docker compose -f docker-compose.prod.yml ps

# 重启服务
docker compose -f docker-compose.prod.yml restart
```

---

## Docker 构建

### 构建镜像

```bash
# 构建后端镜像
docker build -t itsm-backend:latest -f Dockerfile.backend ./itsm-backend

# 构建前端镜像
docker build -t itsm-frontend:latest -f Dockerfile.frontend ./itsm-frontend

# 或使用 Make
make build
```

### 多平台构建

```bash
# 构建多平台镜像
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t itsm-backend:latest \
  -f Dockerfile.backend \
  ./itsm-backend
```

### 优化镜像大小

```bash
# 使用多阶段构建
# 后端使用 alpine 镜像
# 前端使用 nginx:alpine
```

---

## 生产环境部署

### 部署方式对比

| 方式 | 适用场景 | 优点 | 缺点 |
|------|---------|------|------|
| Docker Compose | 小规模 | 简单、一致 | 扩展性有限 |
| Kubernetes | 大规模 | 高可用、弹性 | 复杂 |
| 直接部署 | 传统服务器 | 简单 | 维护困难 |

### Docker Compose 部署

```bash
# 1. 准备配置
cp .env.example .env.prod
vim .env.prod

# 2. 构建镜像
docker compose -f docker-compose.prod.yml build

# 3. 启动服务
docker compose -f docker-compose.prod.yml up -d

# 4. 检查状态
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs -f
```

### Systemd 部署

创建服务文件 `/etc/systemd/system/itsm.service`:

```ini
[Unit]
Description=ITSM Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=itsm
Group=itsm
WorkingDirectory=/opt/itsm/backend
ExecStart=/opt/itsm/backend/itsm
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable itsm
sudo systemctl start itsm
sudo systemctl status itsm
```

---

## 启动验证

### 健康检查

```bash
# 检查后端健康状态
curl http://localhost:8090/health
# 返回: {"status":"ok"}

# 检查数据库连接
curl http://localhost:8090/health/db

# 检查 Redis 连接
curl http://localhost:8090/health/redis
```

### API 测试

```bash
# 测试登录
curl -X POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# 测试获取用户信息
curl -H "Authorization: Bearer <token>" \
  http://localhost:8090/api/v1/users/me
```

### 前端验证

```bash
# 访问 http://localhost:3000
# 验证页面加载正常
# 验证登录功能正常
# 验证 API 请求正常
```

---

## 启动参数

### 后端参数

```bash
# 查看帮助
./bin/itsm --help

# 常用参数
./bin/itsm --port=8080           # 指定端口
./bin/itsm --config=/path/config.yaml  # 指定配置
./bin/itsm --log-level=info      # 日志级别
./bin/itsm --version            # 显示版本
```

### 环境变量

```bash
# 使用环境变量启动
SERVER_PORT=8080 \
JWT_SECRET=your-secret \
./bin/itsm
```

---

## 常见问题

### 问题 1：端口被占用

```bash
# 查找占用端口的进程
lsof -i :8090

# 杀死进程
kill -9 <PID>

# 或使用其他端口
SERVER_PORT=8091 ./bin/itsm
```

### 问题 2：数据库连接失败

```bash
# 检查数据库是否运行
docker ps | grep postgres

# 检查连接配置
cat .env | grep DB_

# 测试连接
psql -h localhost -U itsm -d itsm
```

### 问题 3：前端构建失败

```bash
# 清理缓存
rm -rf .next node_modules
npm install
npm run build
```

### 问题 4：内存不足

```bash
# 增加 Node.js 内存限制
NODE_OPTIONS="--max-old-space-size=4096" npm run build

# 或使用 Swap
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

---

## 启动脚本示例

### 生产启动脚本

```bash
#!/bin/bash
# start.sh

set -e

# 配置
APP_DIR="/opt/itsm"
APP_NAME="itsm-backend"
PORT=8090
LOG_FILE="/var/log/itsm.log"

# 启动
cd $APP_DIR
nohup ./$APP_NAME --port=$PORT > $LOG_FILE 2>&1 &

echo "ITSM started on port $PORT"
```

### Docker Compose 一键启动

```bash
#!/bin/bash
# quick-start.sh

set -e

echo "Starting ITSM..."

# 启动数据库和缓存
docker compose up -d postgres redis

# 等待数据库就绪
echo "Waiting for database..."
sleep 10

# 启动后端
docker compose up -d backend

# 启动前端
docker compose up -d frontend

echo ""
echo "ITSM is running!"
echo "  Frontend: http://localhost:3000"
echo "  Backend:  http://localhost:8090"
echo "  API Doc:  http://localhost:8090/swagger"
echo ""
echo "Login: admin / admin123"
```

---

**文档维护**: ITSM 开发团队
**最后更新**: 2026-03-04
