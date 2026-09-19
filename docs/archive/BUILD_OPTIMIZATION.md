# ITSM 构建优化指南

## 概述

本文档介绍ITSM系统的构建优化策略，帮助开发者快速构建和部署系统。

## 快速开始

### 使用Makefile（推荐）

```bash
# 查看所有可用命令
make help

# 构建所有镜像
make build

# 并行构建（更快）
make build-parallel

# 构建并部署
make build deploy

# 运行测试
make test
```

### 使用构建脚本

```bash
# 基本构建
./scripts/build-optimized.sh

# 并行构建
./scripts/build-optimized.sh --parallel

# 无缓存构建
./scripts/build-optimized.sh --no-cache

# 指定版本
./scripts/build-optimized.sh --version v1.7.0
```

## 优化策略

### 1. Docker层缓存优化

**后端Dockerfile优化点：**
- 分离`go.mod`和源代码COPY，利用层缓存
- 使用BuildKit缓存挂载加速Go模块下载
- 使用`-trimpath`和`-ldflags="-s -w"`减小二进制体积

**前端Dockerfile优化点：**
- 分离`package.json`和源代码COPY
- 使用npm缓存挂载
- 多阶段构建，最终镜像只包含运行时必需文件

### 2. 并行构建

```bash
# 并行构建前后端
make build-parallel

# 或者直接使用docker compose
DOCKER_BUILDKIT=1 docker compose -f docker-compose.prod.yml build --parallel
```

**优势：**
- 后端和前端同时构建，节省约40-50%时间
- 充分利用多核CPU

### 3. 缓存策略

**Go模块缓存：**
```dockerfile
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
```

**npm缓存：**
```dockerfile
RUN --mount=type=cache,target=/root/.npm \
    npm ci --ignore-scripts
```

### 4. 镜像大小优化

**后端镜像：**
- 基础镜像：alpine:3.20（~7MB）
- 最终镜像大小：~232MB
- 包含：二进制文件、配置、迁移工具

**前端镜像：**
- 基础镜像：node:22-alpine
- 最终镜像大小：~320MB
- 包含：standalone Next.js应用

### 5. 构建参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `VERSION` | 版本号 | git describe |
| `ENV_FILE` | 环境文件 | .env.prod |
| `COMPOSE_FILE` | Compose文件 | docker-compose.prod.yml |
| `NO_CACHE` | 不使用缓存 | false |
| `PARALLEL` | 并行构建 | true |

## 性能对比

| 构建方式 | 耗时 | 说明 |
|----------|------|------|
| 顺序构建（无缓存） | ~15分钟 | 首次构建 |
| 顺序构建（有缓存） | ~5分钟 | 增量构建 |
| 并行构建（有缓存） | ~3分钟 | 推荐方式 |
| 并行构建（无缓存） | ~8分钟 | 完整重建 |

## 常见问题

### Q: 构建失败怎么办？

```bash
# 清理缓存重试
make clean
make build-no-cache

# 查看详细日志
docker compose -f docker-compose.prod.yml build --progress=plain
```

### Q: 如何加速Go模块下载？

在`.env.prod`中配置Go代理：
```bash
GOPROXY=https://goproxy.cn,direct
```

### Q: 如何减小镜像大小？

1. 使用多阶段构建（已配置）
2. 清理不必要文件（已配置）
3. 使用Alpine基础镜像（已配置）

### Q: 如何在CI/CD中使用？

```yaml
# GitHub Actions示例
- name: Build and Deploy
  run: |
    make build-parallel
    make deploy
```

## 最佳实践

1. **开发环境**：使用`docker-compose.dev.yml`，支持热重载
2. **测试环境**：使用`make build test`，确保测试通过
3. **生产环境**：使用`make build-parallel deploy`，快速部署
4. **CI/CD**：使用并行构建，配置缓存卷

## 监控和调试

### 查看构建缓存

```bash
# 查看Docker缓存使用情况
docker system df

# 清理未使用资源
docker system prune
```

### 查看镜像层

```bash
# 查看镜像构建历史
docker history itsm-backend:latest

# 分析镜像大小
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"
```

## 更新日志

- 2026-08-30: 初始版本，支持并行构建和缓存优化
