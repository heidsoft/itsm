# ITSM 部署架构分析与优化报告

> 目标：分析当前项目部署架构，优化 Dockerfile、压缩镜像体积、全面启用多阶段构建，并精简本地部署脚本，补齐 CI/CD。

---

## 1. 当前项目架构现状

### 1.1 服务构成

| 服务 | 语言/栈 | 镜像来源 | 说明 |
|------|---------|----------|------|
| `itsm-backend` | Go 1.25 / Gin + Ent | `itsm-backend/Dockerfile`、`Dockerfile.prod` | 主后端，含 init（迁移/种子）与 backend 两个容器 |
| `itsm-frontend` | Next.js 15 / Node 22 | `itsm-frontend/Dockerfile` | 4 阶段：deps / builder / production / runner |
| `itsm-ai-service` | Python 3.12 / FastAPI | `itsm-ai-service/Dockerfile` | 独立 AI 微服务，依赖 torch + transformers |
| `guidance_sidecar` | Python 3.11 / FastAPI | `itsm-backend/guidance_sidecar/Dockerfile` | 引导/分类 sidecar |
| postgres / redis / minio / nginx | 官方镜像 | docker-compose 引用 | 基础设施 |

### 1.2 编排与脚本

- **3 份 compose**：`docker-compose.yml`（默认/本地）、`docker-compose.dev.yml`（开发，带 profile）、`docker-compose.prod.yml`（生产，网络隔离 + 健康检查 + 回滚状态）。
- **部署脚本（scripts/）**：
  - `deploy-prod.sh`（~730 行，五阶段流水线：校验→备份→构建→部署→验证，含锁/回滚）
  - `deploy-dev.sh`（~810 行，本地/Docker 双模式，含 doctor）
  - `build-frontend-image.sh`（单行封装，仅构建前端）
  - `start-dev.sh` / `stop-dev.sh`（已是 `deploy-dev.sh` 的薄封装，设计良好）
  - `smoke-test.sh`、`release.sh`、`init_admin.sh`、`init_database.sh` 等
  - `local-codex-*.sh` × 4（Agent 辅助脚本，与部署无关）
  - `lib/common.sh`（共享函数库，设计良好）

**结论**：deploy-prod/deploy-dev 本身质量很高；真正的问题是**镜像构建层**和**脚本数量**。

---

## 2. 现有 Docker 镜像问题诊断

| 镜像 | 问题 | 影响 |
|------|------|------|
| backend（dev/prod） | 基础镜像 `golang:alpine` 未固定版本；无 BuildKit 缓存挂载；每次全量拉取/编译 | 构建慢、不可复现 |
| frontend | 生产阶段以 **root** 运行；未用 npm 缓存挂载 | 安全隐患、构建慢 |
| **ai-service** | **单阶段**；`requirements.txt` 安装 torch **CUDA 版**（默认 PyPI 提供 CUDA wheel） | 镜像 **3–4 GB**，拉取/推送极慢 |
| guidance_sidecar | 单阶段、root 运行、无 `.dockerignore` | 上下文臃肿、镜像偏大 |
| ai-service / guidance_sidecar | **完全没有 `.dockerignore`** | `.git`、本地 venv、缓存被塞进构建上下文 |

---

## 3. 优化方案（已落地）

### 3.1 后端 Dockerfile（dev + prod）

- 固定基础镜像：`golang:1.25-alpine` → `alpine:3.20`（之前 `alpine:latest` 浮动）。
- 引入 **BuildKit 缓存挂载**：`--mount=type=cache,target=/go/pkg/mod` 与 `target=/root/.cache/go-build`，重复构建命中缓存。
- 统一 `trimpath -ldflags="-s -w"`，剥离符号与调试信息，缩小二进制。
- 最终镜像保持静态、非 root（`app` 用户），仅拷贝二进制 + `config.yaml` / `migrations` / `config/seed`。

### 3.2 前端 Dockerfile

- 固定 `node:22-alpine`。
- npm 阶段增加 `--mount=type=cache,target=/root/.npm` 缓存。
- **production 阶段改为以内置 `node` 用户运行**（`USER node`），并修正 `.next/cache` 可写权限。
- 保留 `production` / `runner` 双 target，与既有 compose 完全兼容。

### 3.3 AI 服务 Dockerfile（多阶段 + 体积瘦身）

- 改为 **builder（构建 venv）→ runtime（slim）** 两阶段。
- **关键优化**：torch 安装改用 **CPU-only wheel**（`--extra-index-url https://download.pytorch.org/whl/cpu`，可通过 `TORCH_INDEX` 覆盖为 GPU），避免引入 2GB+ 的 CUDA 包。
- 运行时仅复制 venv + 源码，安装 `libgomp1`（torch 运行依赖），**非 root 用户**。
- 健康检查改用标准库 `urllib`，不再依赖运行时是否装了 `httpx`。

### 3.4 guidance_sidecar Dockerfile（多阶段 + 非 root）

- builder 构建 venv → runtime 仅复制 venv + 源码。
- 非 root 用户运行。

### 3.5 补齐 `.dockerignore`

- 新增 `itsm-ai-service/.dockerignore`、`itsm-backend/guidance_sidecar/.dockerignore`（此前缺失）。
- 增强 `itsm-backend/.dockerignore`、`itsm-frontend/.dockerignore`（排除测试、docs、`.github`，缩小上下文）。

---

## 4. 镜像体积估算对比（构建后）

| 镜像 | 优化前（估算） | 优化后（估算） | 主要收益 |
|------|----------------|----------------|----------|
| itsm-backend | ~40–60 MB | ~35–45 MB | 版本固定 + 缓存 + trimpath（更可复现） |
| itsm-frontend | ~180–250 MB | ~170–220 MB | 非 root + npm 缓存（更安全、更快） |
| **itsm-ai-service** | **~3.0–4.0 GB** | **~1.2–1.6 GB** | **CPU-only torch，减少 ~60–70%** |
| guidance_sidecar | ~400–600 MB | ~300–450 MB | 多阶段 venv + 非 root |

> 说明：数值为基于依赖构成的工程估算，实际以 `docker images` 为准。AI 服务是最大的优化点。

---

## 5. 部署脚本优化

- **统一构建入口**：新增 `scripts/build-images.sh`，一条命令构建全部服务镜像（支持 `VERSION`、`REGISTRY`、`GOPROXY`、`NPM_REGISTRY`、`TORCH_INDEX` 等环境变量，启用 BuildKit 内联缓存）。
- **收敛冗余脚本**：`build-frontend-image.sh` 改为调用 `build-images.sh frontend` 的兼容封装，命令名不变。
- **Makefile** 新增 `make build-images`（等价于 `./scripts/build-images.sh`）。
- `deploy-prod.sh` / `deploy-dev.sh` 顶部 `export DOCKER_BUILDKIT=1`，生产构建追加 `--build-arg BUILDKIT_INLINE_CACHE=1`，让多阶段缓存挂载与层缓存生效。
- `start-dev.sh` / `stop-dev.sh` 已是薄封装，保留；`local-codex-*.sh` 属于 Agent 辅助，与部署解耦，不在本次收敛范围。

**脚本收敛效果**：原本分散的“前端构建”逻辑（build-frontend-image.sh）并入统一构建器，后续新增服务只需在 `build-images.sh` 的 `ALL_SERVICES` 数组追加一行。

---

## 6. CI/CD（新增）

- **`.github/workflows/ci.yml`**：push/PR 到 `main` 触发。
  - `backend-test`：Go 单测。
  - `frontend-check`：ESLint + TypeScript 类型检查。
  - `docker-build`：用 Buildx + **GitHub Actions 缓存（gha）** 构建全部 4 个镜像（多阶段 + 缓存挂载），验证 Dockerfile 可构建（不推送）。
- **`.github/workflows/publish.yml`**：打 `v*` tag（或手动触发）时，登录 GHCR，跨 `linux/amd64,linux/arm64` 构建并推送 `itsm-<svc>:<version>` 与 `:latest`。

这让“本地脚本”升级为**可复现的云端流水线**，镜像构建与本地 `build-images.sh` 共用同一套多阶段 Dockerfile。

---

## 7. 使用方式

```bash
# 本地一键构建所有镜像（自动启用 BuildKit 缓存）
make build-images                       # 或 ./scripts/build-images.sh
./scripts/build-images.sh v1.2.0       # 打版本标签
REGISTRY=ghcr.io/heidsoft/ ./scripts/build-images.sh v1.2.0   # 带 registry 前缀

# 开发 / 生产部署（行为不变，已启用缓存）
./scripts/deploy-dev.sh up
./scripts/deploy-prod.sh init && ./scripts/deploy-prod.sh deploy

# CI 推送（打 tag 触发，或手动）
git tag v1.2.0 && git push origin v1.2.0
```

---

## 8. 后续建议（可选）

1. AI 服务若确有 GPU 需求，仅需在 `build-images.sh` / CI 传入 `TORCH_INDEX=https://download.pytorch.org/whl/cu121` 即可，无需改 Dockerfile。
2. 可引入 `docker scout cves` 或 Trivy 在 CI `docker-build` 后做镜像漏洞扫描。
3. 生产镜像可进一步尝试 `gcr.io/distroless/static` 替代 `alpine`（需把健康检查改为二进制内建端点）。
