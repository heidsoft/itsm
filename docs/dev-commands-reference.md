# ITSM 本地开发命令最佳实践

> 基于 Makefile + scripts/deploy-dev.sh + scripts/deploy-prod.sh + docker-compose 配置整理。
> 三种调用方式任选其一：`make xxx` ≈ `./scripts/deploy-xxx.sh cmd` ≈ `docker compose ...`

---

## ⚠️ 零、最重要的规则：启动和停止必须用同一套配置

**这是最容易犯的错误**：用 dev 配置启动，却用 prod 配置停止（或反过来）。

```bash
# ❌ 错误！启动用 dev，停止用 prod
docker compose --env-file .env      -f docker-compose.dev.yml  --profile dev up -d
docker compose --env-file .env.prod -f docker-compose.prod.yml down        # 配置不匹配！

# ✅ 正确！启动和停止用同一套
docker compose --env-file .env      -f docker-compose.dev.yml  --profile dev up -d
docker compose --env-file .env      -f docker-compose.dev.yml  --profile dev down

# ✅ 最安全！用 make 封装好的命令（自动匹配配置）
make dev-start          # 启动开发环境
make dev-stop           # 停止开发环境（与启动配置一致）
make prod-deploy        # 部署生产环境
make prod-stop          # 停止生产环境
```

**为什么会出错**：Docker Compose v2 用目录名作为 project name（本项目是 `itsm`）。dev 和 prod compose 文件共享同一个 project，用 prod 配置 `down` 时会按 project label 把 dev 容器也删掉，但 dev 网络不会被清理（变成孤儿），且命令会因找不到 prod 容器返回非零退出码。

**配置配对速查**：

| 环境 | env-file | compose 文件 | 额外参数 |
|------|----------|-------------|----------|
| 开发 | `.env` | `docker-compose.dev.yml` | `--profile dev` |
| 生产 | `.env.prod` | `docker-compose.prod.yml` | 无 |

---

## 一、日常开发（Development）

### 1.1 首次初始化

```bash
# 首次拉起完整开发环境（自动安装依赖、创建 .env、启动服务）
make dev-init
# 等价于：./scripts/deploy-dev.sh init
```

### 1.2 日常启停

项目支持**两种开发模式**，按需选择：

| 模式 | 说明 | 适用场景 |
|------|------|----------|
| `--docker` | 全 Docker Compose（前后端+基础设施均容器化） | 快速验证、CI、不想装 Go/Node |
| `--local` | 本地 Go/Next.js 进程 + Docker 基础设施（PG/Redis/MinIO） | 日常开发（热重载、断点调试） |

```bash
# ===== 启动 =====
make dev-start            # 自动检测模式（有 Docker 则用 docker，否则 local）
make dev-start-docker     # 强制 Docker 模式（全容器化）
make dev-start-local      # 强制本地模式（Go/Next.js 直接跑，推荐日常开发用）

# 快速重启（不重新构建镜像）
make dev-start-docker --no-build    # 或：./scripts/deploy-dev.sh up --docker --no-build

# ===== 停止 =====
make dev-stop              # 停止所有服务
make dev-stop-local        # 仅停止本地 Go/Next.js 进程（保留 PG/Redis 容器）
make dev-stop-docker      # 仅停止 Docker Compose 环境

# ===== 重启 =====
make dev-restart           # 重启所有服务
```

### 1.3 日志 / 状态 / 诊断

```bash
make dev-logs              # 查看所有日志（实时跟踪）
make dev-logs itsm-backend # 查看指定服务日志（等价：./scripts/deploy-dev.sh logs itsm-backend）
make dev-status            # 查看服务状态
make dev-health            # 运行健康检查
make dev-doctor            # 诊断环境问题（端口冲突、Docker 状态、磁盘空间）
```

### 1.4 访问地址

| 服务 | 地址 |
|------|------|
| 前端 | http://localhost:3000 |
| 后端 API | http://localhost:8090 |
| Swagger 文档 | http://localhost:8090/swagger |
| 本地开发登录账号 | admin / admin123（禁止用于生产） |
| PostgreSQL | localhost:5432 (itsm_user/dev123) |
| Redis | localhost:6379 |
| MinIO 控制台 | http://localhost:9001 (minioadmin/minioadmin123) |

---

## 二、测试环境（Testing）

### 2.1 重置开发环境（清除数据）

```bash
make dev-clean             # 停止并删除所有容器 + 数据卷（⚠️ 会清除 DB 数据）
# 等价于：./scripts/deploy-dev.sh reset
```

### 2.2 数据库操作

```bash
make db-migrate            # 运行数据库迁移
make db-seed               # 填充测试数据
```

### 2.3 后端测试

```bash
cd itsm-backend

# 运行所有测试
go test ./...

# 运行指定包测试
go test ./internal/service/... -v

# 运行测试并生成覆盖率报告
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# 运行基准测试
go test ./internal/service/... -bench=. -benchmem

# 运行竞态检测
go test -race ./...
```

### 2.4 前端测试

```bash
cd itsm-frontend

# 运行测试
npm test

# 运行测试并生成覆盖率
npm test -- --coverage

# 类型检查
npm run type-check

# Lint 检查（自动修复）
npm run lint

# Lint 检查（仅检查不修复）
npm run lint:check

# 单元测试（无覆盖率）
npm run test:unit

# 集成测试
npm run test:integration

# E2E 测试（需要环境运行）
npm run test:e2e

# 角色权限 E2E
npm run test:e2e:roles

# 业务流程 E2E
npm run test:e2e:flows

# 冒烟测试
npm run test:smoke
```

### 2.5 工程契约检查

```bash
make check-contracts        # 校验 API 路径、部署配置、Docker 配置一致性
make verify-scripts         # 验证构建/启动脚本语法
```

### 2.6 功能冒烟测试

```bash
# Shell 版冒烟测试（健康检查 + 登录 + 核心API + 数据库连接）
./scripts/smoke-test.sh

# 自定义测试参数
# 仅用于本地开发 seed 数据；生产烟测应使用临时测试 actor/secret 注入
ITSM_BACKEND_URL=http://localhost:8090 ITSM_ADMIN_USER=admin ITSM_ADMIN_PASS=admin123 ./scripts/smoke-test.sh

# Python 版功能测试（更全面，需要后端运行中）
python3 output/itsm_functional_test.py
```

---

## 三、生产环境（Production）

### 3.1 初始化配置

```bash
make prod-init              # 生成 .env.prod（自动填充随机密钥）
# 等价于：./scripts/deploy-prod.sh init

# ⚠️ 必须检查以下配置：
#   - CORS_ALLOWED_ORIGINS（改为实际域名）
#   - LLM_PROVIDER / OPENAI_API_KEY
#   - 域名 / SSL 证书路径
```

### 3.2 完整部署（5 阶段流水线）

```bash
make prod-deploy            # validate → backup → build → deploy → verify
# 等价于：./scripts/deploy-prod.sh deploy

# 预览部署计划（不实际执行）
./scripts/deploy-prod.sh deploy --dry-run

# 使用已有镜像部署（跳过构建和备份，快速启动）
make prod-start
# 等价于：./scripts/deploy-prod.sh deploy --skip-build --skip-backup

# 显示完整构建输出
./scripts/deploy-prod.sh deploy --verbose
```

### 3.3 生产运维

```bash
# 状态 / 健康
make prod-status            # 查看服务状态 + 部署信息
make prod-health            # 健康检查（HTTP 端点 + 容器状态）

# 日志
make prod-logs              # 查看所有日志
./scripts/deploy-prod.sh logs itsm-backend   # 查看指定服务

# 停止 / 重启
make prod-stop              # 停止所有生产服务
make prod-restart           # 重启生产环境

# 回滚
make prod-rollback          # 回滚到上一个部署版本

# 备份
make prod-backup            # 手动备份数据库（自动保留最近 5 份）
```

### 3.4 生产环境注意事项

- **端口隔离**：生产 PG 映射到 `127.0.0.1:5433`、Redis 映射到 `127.0.0.1:6380`（不暴露公网）
- **必须先跑 `ent migrate`**：新增 schema 字段后，镜像上线前必须执行迁移（否则 `column does not exist` 500 错误）
- **生产必须显式传 `--env-file .env.prod`**：`deploy-prod.sh` 已自动处理，直接用 docker compose 时需手动传
- **部署锁**：`deploy-prod.sh` 自带部署锁，防止并发部署
- **自动回滚**：部署失败时自动回滚到上一版本

---

## 四、Docker Compose 原生命令（不通过脚本）

> 当脚本不满足需求时，可直接使用 docker compose 原生命令。

### 4.1 开发环境

```bash
# 启动全部开发服务
docker compose --env-file .env -f docker-compose.dev.yml --profile dev up -d

# 仅启动基础设施（PG + Redis + MinIO）
docker compose --env-file .env -f docker-compose.dev.yml --profile dev up -d postgres redis minio

# 重新构建并启动后端
docker compose --env-file .env -f docker-compose.dev.yml --profile dev up -d --build itsm-backend

# 查看日志
docker compose --env-file .env -f docker-compose.dev.yml --profile dev logs -f itsm-backend

# 进入容器
docker exec -it itsm-backend-dev sh

# 停止（保留数据）
docker compose --env-file .env -f docker-compose.dev.yml --profile dev down

# 停止并删除数据卷（⚠️ 清除所有数据）
docker compose --env-file .env -f docker-compose.dev.yml --profile dev down -v
```

### 4.2 可选 Profile

```bash
# 启动 Ollama（本地 LLM）
docker compose --env-file .env -f docker-compose.dev.yml --profile dev --profile ai up -d ollama

# 启动监控（Prometheus + Grafana）
docker compose --env-file .env -f docker-compose.dev.yml --profile dev --profile monitoring up -d
```

### 4.3 生产环境

```bash
# 启动生产环境
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d

# 停止
docker compose --env-file .env.prod -f docker-compose.prod.yml down

# 重新构建
docker compose --env-file .env.prod -f docker-compose.prod.yml build --build-arg BUILDKIT_INLINE_CACHE=1
```

---

## 五、镜像构建与发布

```bash
# 构建所有镜像（指定版本和仓库）
VERSION=v1.0.0 REGISTRY=registry.example.com make build-images

# 仅构建后端镜像
VERSION=v1.0.0 make build-backend

# 仅构建前端镜像
VERSION=v1.0.0 make build-frontend

# 创建发布包
VERSION=v1.0.0 make release
```

---

## 六、Docker 维护与清理

### 6.1 日常清理（安全）

```bash
# 查看磁盘使用情况
docker system df

# 清理已停止的容器（安全）
docker container prune -f

# 清理悬空镜像（dangling，安全）
docker image prune -f

# 清理未使用的网络（安全）
docker network prune -f

# 清理构建缓存（⚠️ 下次 build 会变慢）
docker builder prune -f
```

### 6.2 深度清理（⚠️ 谨慎）

```bash
# 清理所有未被使用的资源（镜像、容器、网络，不含数据卷）
docker system prune -f

# 包含数据卷（⚠️ 会删除未挂载的 named volume，可能包含数据）
docker system prune -a --volumes -f

# 清理所有构建缓存（含正在引用的）
docker builder prune --all -f
```

### 6.3 定期维护建议

```bash
# 推荐每月执行一次的安全清理脚本
docker container prune -f && \
docker image prune -f && \
docker network prune -f && \
docker builder prune -f && \
docker system df
```

### 6.4 数据卷管理（⚠️ 高危操作）

```bash
# 查看所有数据卷
docker volume ls

# 查看悬空数据卷（未被任何容器引用）
docker volume ls -f "dangling=true"

# ⚠️ 删除数据卷前务必确认：
#   - itsm_postgres_prod_data / itsm_redis_prod_data = 生产数据，绝对不能删
#   - itsm_postgres_dev_data / itsm_redis_dev_data = 开发数据，可重建
#   - 匿名 volume（hash 命名）= 通常是构建残留，可安全删除

# 备份生产数据库
docker exec itsm-postgres-prod pg_dump -U itsm -d itsm_prod | gzip > backups/itsm_prod_$(date +%Y%m%d).sql.gz
```

---

## 七、常用排查命令

### 7.1 孤儿网络清理

用错配置停服务后，会产生孤儿网络（compose 文件不定义它，`down` 不清理）：

```bash
# 查看 itsm 相关网络
docker network ls --filter "name=itsm"

# 检查网络是否有容器在用（0 表示无引用，可安全删除）
docker network inspect <网络名> --format '{{len .Containers}} containers attached'

# 删除孤儿网络
docker network rm itsm_itsm-dev-network    # dev 网络
docker network rm itsm_itsm-network       # 基础 compose 网络
docker network rm itsm_itsm-prod-network  # prod 网络
```

### 7.2 常用排查命令

```bash
# 查看容器资源使用
docker stats --no-stream

# 查看容器日志（最近 100 行）
docker logs --tail 100 -f itsm-backend-dev

# 进入数据库
docker exec -it itsm-postgres-dev psql -U itsm_user -d itsm

# 进入 Redis
docker exec -it itsm-redis-dev redis-cli

# 查看后端健康
curl -s http://localhost:8090/api/v1/health | jq

# 查看后端路由列表
curl -s http://localhost:8090/swagger/doc.json | jq '.paths | keys'

# 检查端口占用
lsof -nP -iTCP:8090 -sTCP:LISTEN
lsof -nP -iTCP:3000 -sTCP:LISTEN
```

---

## 八、命令速查表

| 场景 | 命令 |
|------|------|
| 首次初始化 | `make dev-init` |
| 日常启动（本地模式） | `make dev-start-local` |
| 日常启动（Docker 模式） | `make dev-start-docker` |
| 快速重启（不构建） | `./scripts/deploy-dev.sh up --docker --no-build` |
| 停止 | `make dev-stop` |
| 查看日志 | `make dev-logs` |
| 查看状态 | `make dev-status` |
| 健康检查 | `make dev-health` |
| 环境诊断 | `make dev-doctor` |
| 重置环境（删数据） | `make dev-clean` |
| 数据库迁移 | `make db-migrate` |
| 填充测试数据 | `make db-seed` |
| 契约检查 | `make check-contracts` |
| 生产初始化 | `make prod-init` |
| 生产部署 | `make prod-deploy` |
| 生产预览 | `./scripts/deploy-prod.sh deploy --dry-run` |
| 生产回滚 | `make prod-rollback` |
| 生产备份 | `make prod-backup` |
| 生产日志 | `make prod-logs` |
| Docker 清理 | `docker system prune -f` |
| 深度清理 | `docker system prune -a --volumes -f` |

---

## 九、实用技巧

### 9.1 快速重启单个服务

```bash
# Docker 模式下重启后端（保留其他服务）
docker compose --env-file .env -f docker-compose.dev.yml restart itsm-backend

# Docker 模式下重建并重启前端
docker compose --env-file .env -f docker-compose.dev.yml up -d --build itsm-frontend
```

### 9.2 查看实时资源占用

```bash
# 实时查看所有容器资源（CPU/内存/网络）
docker stats

# 仅查看特定服务
docker stats itsm-backend-dev itsm-frontend-dev
```

### 9.3 端口冲突快速诊断

```bash
# 检查端口占用
lsof -nP -iTCP:8090 -iTCP:3000 -iTCP:5432 -iTCP:6379 -iTCP:9001

# 或一键检查所有开发相关端口
make dev-doctor
```

### 9.4 进入容器调试

```bash
# 进入后端容器
docker exec -it itsm-backend-dev sh

# 进入前端容器
docker exec -it itsm-frontend-dev sh

# 进入数据库（带自动补全）
docker exec -it itsm-postgres-dev psql -U itsm_user -d itsm

# 进入 Redis
docker exec -it itsm-redis-dev redis-cli
```

### 9.5 数据库常用操作

```bash
# 查看表结构
docker exec -it itsm-postgres-dev psql -U itsm_user -d itsm -c "\d tickets"

# 执行 SQL 文件
docker exec -it itsm-postgres-dev psql -U itsm_user -d itsm -f /path/to/sqlfile.sql

# 备份单个表
docker exec itsm-postgres-dev pg_dump -U itsm_user -d itsm -t users > users.sql
```

### 9.6 多环境快速切换

```bash
# 同时运行开发和生产环境（端口隔离）
# 终端1：开发环境
make dev-start

# 终端2：生产环境（端口不同：5433/6380/8091/3001）
make prod-start

# 验证两个环境独立运行
curl -s http://localhost:8090/api/v1/health  # 开发
curl -s http://localhost:8091/api/v1/health  # 生产
```

### 9.7 构建缓存加速

```bash
# 首次构建后，后续构建会使用 BuildKit 缓存
# 如需强制重新构建：
docker compose --env-file .env -f docker-compose.dev.yml build --no-cache itsm-backend

# 清理特定镜像的缓存
docker builder prune --filter "type=exec.digest::sha256:$(docker images itsm-backend-dev -q | head -1)"
```
