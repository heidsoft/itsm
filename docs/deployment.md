# 构建与部署指南

> Status: current。历史认证、测试报告或旧 Compose 示例不能覆盖本文；发布状态按[文档状态与事实源](documentation-governance.md)判定。

## 命令入口

| 场景 | 推荐命令 | 环境文件 |
|---|---|---|
| Docker 开发环境 | `make dev-start-docker` | `.env` |
| 本机 Go/Next.js 开发 | `make dev-start-local` | `.env` |
| 完整生产部署 | `make prod-deploy` | `.env.prod` |
| 启动已有生产镜像 | `make prod-start` | `.env.prod` |
| 构建版本化发布镜像 | `VERSION=v1.2.0 make build-images` | 构建参数 |

脚本是部署行为的单一入口。手工执行 Docker Compose 时，开发环境和生产环境也必须分别显式传入 `.env` 与 `.env.prod`。

## 开发环境

### Docker Compose 模式（推荐）

```bash
git clone https://github.com/heidsoft/itsm.git
cd itsm
cp .env.dev.example .env

# 构建镜像并启动 PostgreSQL、Redis、MinIO、初始化器、后端和前端
make dev-start-docker

make dev-status
make dev-health
make dev-logs
```

等价的原生命令：

```bash
docker compose --env-file .env -f docker-compose.dev.yml --profile dev up -d --build
docker compose --env-file .env -f docker-compose.dev.yml --profile dev ps
docker compose --env-file .env -f docker-compose.dev.yml --profile dev logs -f
```

停止环境：

```bash
make dev-stop-docker

# 删除数据卷会清空本地数据库和对象存储数据
docker compose --env-file .env -f docker-compose.dev.yml --profile dev down -v
```

### 本机开发模式

此模式在本机运行 Go 后端和 Next.js 前端，基础设施按需使用 Docker。

```bash
cp .env.dev.example .env
make dev-start-local

./scripts/deploy-dev.sh status --local
./scripts/deploy-dev.sh logs --local
make dev-stop-local
```

访问地址：

- 前端：`http://localhost:3000`
- 后端：`http://localhost:8090`
- 健康检查：`http://localhost:8090/api/v1/health`
- Swagger：`http://localhost:8090/swagger/index.html`

## 生产环境

### 环境要求

- Linux（建议 Ubuntu 22.04 或更新版本）
- Docker 24+ 与 Docker Compose v2
- 4 核 CPU、8 GB 内存、50 GB SSD 起
- 已规划域名、TLS、备份、日志留存与监控告警

### 首次初始化

```bash
git clone https://github.com/heidsoft/itsm.git
cd itsm

# 创建 .env.prod，并生成部分随机密钥
make prod-init
```

部署前检查 `.env.prod`，必须替换所有 `REQUIRED` 项和默认凭据，尤其是：

- `DB_PASSWORD`
- `REDIS_PASSWORD`
- `JWT_SECRET`
- `BOOTSTRAP_TOKEN_ENABLED=1`（首位管理员使用一次性 token；`ADMIN_PASSWORD` 只用于显式兼容旧流程）
- MinIO 访问密钥

生产环境不得使用 `admin123` 或仓库开发默认值，也不得把固定首管理员密码长期注入 Web/Worker 容器。

### 构建并部署

```bash
# 校验配置 → 备份 → 构建前后端镜像 → 启动 → 健康检查
make prod-deploy

make prod-status
make prod-health
```

仅使用已存在的生产镜像启动：

```bash
make prod-start
```

手工执行时必须显式传入 `.env.prod`：

```bash
docker compose --env-file .env.prod -f docker-compose.prod.yml build itsm-backend itsm-frontend
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d
docker compose --env-file .env.prod -f docker-compose.prod.yml ps
docker compose --env-file .env.prod -f docker-compose.prod.yml logs -f
```

整体链路验证：

```bash
curl -fsS http://localhost/health
curl -fsS http://localhost:8090/api/v1/readiness/ga
```

首次认证还必须验证当前发布实际提供 bootstrap 状态和创建管理员接口。接口缺失、token 可重放或没有审计时，应视为部署未就绪，而不是回退到文档中的历史默认账号。

## 构建版本化镜像

`make prod-deploy` 面向当前主机的 Compose 部署；`make build-images` 面向 Registry 推送、离线交付或版本发布。

```bash
# 构建全部应用镜像
VERSION=v1.2.0 make build-images

# 添加 Registry 前缀
VERSION=v1.2.0 REGISTRY=ghcr.io/heidsoft make build-images

# 单独构建
VERSION=v1.2.0 make build-backend
VERSION=v1.2.0 make build-frontend

# 指定跨平台构建目标（需要对应 builder 支持）
BUILDPLATFORM=linux/amd64 VERSION=v1.2.0 make build-images
```

默认构建当前宿主机平台；不要在 Apple Silicon 开发机上无条件强制 `linux/amd64`，否则会显著降低日常构建速度。

## 备份与回滚

```bash
make prod-backup
make prod-rollback
make prod-logs
make prod-stop
```

完整生产部署会在部署前执行备份，并在失败时进入回滚流程。备份文件位于仓库的 `backups/` 目录，应另行同步到受控的异地存储。

## 发布前校验

```bash
# 构建/启动脚本语法与回归测试
make verify-scripts

# 跨文件 API、部署、Docker 与文档契约
make check-contracts

# 展开并验证 Compose 配置
docker compose --env-file .env -f docker-compose.dev.yml --profile dev config >/dev/null
docker compose --env-file .env.prod -f docker-compose.prod.yml config >/dev/null
```

## 常见排查

```bash
make dev-doctor
make prod-status
make prod-health
make prod-logs

docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
docker logs <container> --tail 30
docker inspect <container> --format '{{json .NetworkSettings.Networks}}'
```

生产和开发 Compose 使用不同网络。出现容器 DNS 解析失败时，先确认目标容器是否处于同一个 Compose 网络。

环境变量完整说明见 [`.env.prod.example`](../.env.prod.example)，运维流程见[运维手册](operations.md)。
