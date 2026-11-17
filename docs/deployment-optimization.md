# ITSM 部署架构分析与优化报告

> Status: current · 最后一次源码复核：2026-11-14（以实际 Dockerfile / Makefile / workflow 为准）

> 目标：分析当前项目部署架构，优化 Dockerfile、压缩镜像体积、全面启用多阶段构建，并精简本地部署脚本，补齐 CI/CD。

---

## 1. 当前项目架构现状

### 1.1 服务构成

| 服务 | 语言/栈 | 镜像来源 | 说明 |
|------|---------|----------|------|
| `itsm-backend` | Go 1.26.10 / Gin + Ent | `itsm-backend/Dockerfile`、`Dockerfile.prod` | 主后端，含 init（迁移/种子）与 backend 两个容器 |
| `itsm-frontend` | Next.js 15 / Node 22 | `itsm-frontend/Dockerfile` | 4 阶段：deps / builder / production / runner |
| `itsm-ai-service` | Python 3.12 / FastAPI | `itsm-ai-service/Dockerfile` | 独立 AI 微服务，依赖 torch + transformers |
| `guidance_sidecar` | Python 3.11 / FastAPI | `itsm-backend/guidance_sidecar/Dockerfile` | 引导/分类 sidecar |
| postgres / redis / minio / nginx | 官方镜像 | docker-compose 引用 | 基础设施 |

### 1.2 编排与脚本

- **2 份 compose**（仓库根目录**没有**默认的 `docker-compose.yml`，不带 `-f` 直接执行 `docker compose up` 会失败）：
  - `docker-compose.dev.yml`（本地开发，带 `--profile ai` / `--profile monitoring`）
  - `docker-compose.prod.yml`（生产，固定项目名 `itsm-prod`、网络隔离 + 健康检查；需 `.env.prod`）
  - dev/prod 项目名已隔离，避免同服务名互顶；prod 后端宿主机诊断口由 `BACKEND_DIAG_PORT` 配置（同机跑 dev 时设 8091，避开 8090）。
- **部署脚本（scripts/）**：
  - `deploy-prod.sh`（~730 行，五阶段流水线：校验→备份→构建→部署→验证，含锁/回滚）
  - `deploy-dev.sh`（~810 行，本地/Docker 双模式，含 doctor）
  - `build-frontend-image.sh`（单行封装，仅构建前端）
  - `start-dev.sh` / `stop-dev.sh`（已是 `deploy-dev.sh` 的薄封装，设计良好）
  - `smoke-test.sh`、`release.sh`、`init_admin.sh`、`init_database.sh` 等
  - ~~`local-codex-*.sh` × 4~~（Agent 辅助脚本，与部署无关；2026-11-14 已退役，见 `scripts/README.md` §2.7）
  - `lib/common.sh`（共享函数库，设计良好）

**结论**：deploy-prod/deploy-dev 本身质量很高；真正的问题是**镜像构建层**和**脚本数量**。

---

## 2. 现有 Docker 镜像问题诊断

> 下表是**优化前**的问题快照（2026-11-14 复核：这些问题已由 §3 的方案处理，
> 但 §3.3 的 torch 索引写法在本轮复核中又发现未真正生效，已二次修正）。

| 镜像 | 问题 | 影响 |
|------|------|------|
| backend（dev/prod） | 基础镜像 `golang:alpine` 未固定版本；无 BuildKit 缓存挂载；每次全量拉取/编译 | 构建慢、不可复现 |
| frontend | 生产阶段以 **root** 运行；未用 npm 缓存挂载 | 安全隐患、构建慢 |
| **ai-service** | **单阶段**；`requirements.txt` 安装 torch **CUDA 版**（默认 PyPI 提供 CUDA wheel） | 镜像 **3–4 GB**，拉取/推送极慢 |
| guidance_sidecar | 单阶段、root 运行、无 `.dockerignore` | 上下文臃肿、镜像偏大 |
| ai-service / guidance_sidecar | **完全没有 `.dockerignore`** | `.git`、本地 venv、缓存被塞进构建上下文 |

---

## 3. 优化方案（已落地；§3.3 / §3.4 / §5 于 2026-11-14 复核修正）

### 3.1 后端 Dockerfile（dev + prod）

- 固定基础镜像：**builder `golang:1.26.10-alpine`、runtime `alpine:3.20`**（此前是浮动
  `golang:alpine` / `alpine:latest`；两个阶段的镜像各自固定，不是把 builder 换成 alpine）。
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
- **关键优化（2026-11-14 复核修正）**：torch 必须用**专属索引** `--index-url`
  `https://download.pytorch.org/whl/cpu` 单独安装，再执行
  `pip install -r requirements.txt` 从 PyPI 补齐其余依赖；可通过 `TORCH_INDEX`
  覆盖为 GPU 索引。
  - 原实现写的是 `pip install -r requirements.txt --extra-index-url "${TORCH_INDEX}"`。
    `--extra-index-url` 只是**追加**索引：pip 仍会拿 PyPI 的 `torch>=2.1.0` 参与解析，
    其 Linux wheel 会拉入数 GB 的 `nvidia-*` CUDA 包；且当 PyTorch 索引不可达时，
    pip 会**静默回退**到 CUDA wheel，构建照样成功、体积悄悄涨回去。
    也就是说文档原先宣称的「减少 60–70% 体积」在旧写法下并不保证成立。
  - 拆成两条 `RUN` 后，torch 先落地并满足 requirements 的版本约束，第二步不会重装它。
    回归锁见 `scripts/__tests__/docker-optimization.test.js`（含改回
    `--extra-index-url` 必红的反向验证）。
- builder / runtime 均启用 BuildKit pip 缓存挂载
  `--mount=type=cache,target=/root/.cache/pip`；同时移除 `PIP_NO_CACHE_DIR=1`
  （它会让缓存挂载完全失效）。
- 运行时仅复制 venv + 源码，安装 `libgomp1`（torch 运行依赖），**非 root 用户**。
- 健康检查改用标准库 `urllib`，不再依赖运行时是否装了 `httpx`。

### 3.4 guidance_sidecar Dockerfile（多阶段 + 非 root）

- builder 构建 venv → runtime 仅复制 venv + 源码，均带 BuildKit pip 缓存挂载
  （2026-11-14 一并移除 `PIP_NO_CACHE_DIR=1`）。
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
| **itsm-ai-service** | **~3.0–4.0 GB**（估算） | **1.33 GB**（2026-11-14 实测） | **CPU-only torch：实测 `torch-2.15.0+cpu`、零 `nvidia-*` 包** |
| guidance_sidecar | ~400–600 MB | ~300–450 MB | 多阶段 venv + 非 root |

> 说明：除 AI 服务外均为基于依赖构成的工程估算，**不是实测值**，以 `docker images` 为准。
> 2026-11-14 补记实测：本机 BuildKit 全量构建 `itsm-ai-service:verify-20261114`
> （`BUILD_EXIT=0`，5m37s）得到 **1.33 GB**，落在原 1.2–1.6 GB 估算区间内。
> 构建日志显示 `torch-2.15.0+cpu` wheel 仅 209.5 MB、安装列表无 `nvidia-*`；
> 第二条 `RUN` 输出 `Requirement already satisfied: torch>=2.1.0 ... (2.15.0+cpu)`，
> 证明 `requirements.txt` 阶段没有从 PyPI 重装 CUDA 版。容器内复核：
> `torch.__version__=2.15.0+cpu`、`torch.version.cuda=None`、`uid=1000`（非 root）、
> `pip list | grep -i nvidia` 零命中。
> **优化前 3–4 GB 未实测**（旧写法已不存在），因此仍不宣称「体积下降 60–70% 已验证」；
> backend / frontend / sidecar 三镜像同样未实测。

---

## 5. 部署脚本优化

- **统一构建入口**：新增 `scripts/build-images.sh`，一条命令构建全部服务镜像（支持 `VERSION`、`REGISTRY`、`GOPROXY`、`NPM_REGISTRY`、`TORCH_INDEX` 等环境变量，启用 BuildKit 内联缓存）。
- **收敛冗余脚本**：`build-frontend-image.sh` 改为调用 `build-images.sh frontend` 的兼容封装，命令名不变。
- **Makefile 新增 `make build-images`（2026-11-14 复核修正）**：该目标此前写作
  `build-images: build`，只是 compose 构建 `backend` + `frontend` **两个**镜像的别名，
  且完全忽略 `REGISTRY`——与本节「等价于 `./scripts/build-images.sh`」以及
  `docs/dev-commands-reference.md` 中 `VERSION=... REGISTRY=... make build-images`
  的用法不符。现已改为实际调用统一构建器（4 个镜像，透传 `VERSION` / `REGISTRY`）；
  `make build` 仍保留 compose 语义不变。回归锁见
  `scripts/__tests__/docker-optimization.test.js`。
- `deploy-prod.sh` / `deploy-dev.sh` 顶部 `export DOCKER_BUILDKIT=1`，生产构建追加 `--build-arg BUILDKIT_INLINE_CACHE=1`，让多阶段缓存挂载与层缓存生效。
- `start-dev.sh` / `stop-dev.sh` 已是薄封装，保留；`local-codex-*.sh` 属于 Agent 辅助、与部署解耦，当时不在该次收敛范围，**已于 2026-11-14 退役**（硬编码本机路径、LaunchAgent 调度已停用，原件归档于本地备份）。

**脚本收敛效果**：原本分散的“前端构建”逻辑（build-frontend-image.sh）并入统一构建器，后续新增服务只需在 `build-images.sh` 的 `ALL_SERVICES` 数组追加一行。

---

## 6. CI/CD（2026-11-14 按实际 workflow 清单复核）

本节原先写的 `.github/workflows/ci.yml` **在仓库中不存在**，属文档失实。当前真实分工是
多个按路径触发的专职 workflow，而非单一 `ci.yml`：

| Workflow | 职责 | 与镜像构建的关系 |
|:--|:--|:--|
| `backend-ci.yml` | Go 格式化 / staticcheck / build / `go test`（含 AI Eval 门禁）/ 模块校验 | 只构建二进制，不构建镜像 |
| `frontend-ci.yml` | P0 守卫 / ESLint / type-check / Jest / 生产构建 / standalone 校验 | 不构建镜像 |
| `release.yml` | 打 `v*` tag 时创建 Release；`docker/build-push-action@v7` 矩阵构建并**推送** GHCR 镜像 | 唯一使用 `cache-from: type=gha` / `cache-to: type=gha,mode=max` 的流水线 |
| `security.yml` | Trivy（`scan-type: fs`，仓库文件扫描）、gosec、govulncheck | **不扫描镜像**，见 §8.2 |
| `docs-gate.yml` | 文档门禁 + `Script Guard Tests`（整目录 `node --test`） | 含镜像构建契约自测 |
| `ga-gate.yml` | 起 compose 整栈跑 smoke + TicketType E2E | 会 `up -d --build` 构建 dev 镜像 |

即：多阶段 Dockerfile 同时被本地 `build-images.sh`、prod/dev compose 与 `release.yml`
共用；但**没有「每次 PR 都构建全部 4 个镜像且不推送」的 docker-build job**，
原 §6 对 `ci.yml` 的这段描述不成立。若需要 PR 级镜像可构建性验证，应新增独立
workflow（只构建不推送），而不是在文档里假定它已存在。

---

## 7. 使用方式

```bash
# 本地一键构建所有镜像（自动启用 BuildKit 缓存）
make build-images                       # 或 ./scripts/build-images.sh
./scripts/build-images.sh v1.2.0       # 打版本标签
REGISTRY=ghcr.io/heidsoft/ ./scripts/build-images.sh v1.2.0   # 带 registry 前缀
VERSION=v1.2.0 REGISTRY=ghcr.io/heidsoft/ make build-images   # 等价，2026-11-14 起 REGISTRY 才真正生效

# 开发 / 生产部署（行为不变，已启用缓存）
./scripts/deploy-dev.sh up
./scripts/deploy-prod.sh init && ./scripts/deploy-prod.sh deploy

# CI 推送（打 tag 触发，或手动）
git tag v1.2.0 && git push origin v1.2.0
```

---

## 8. 后续建议（2026-11-14 复核状态）

1. **GPU 需求**：仍只需在 `build-images.sh` / CI 传入
   `TORCH_INDEX=https://download.pytorch.org/whl/cu121`，无需改 Dockerfile。
   但当前 torch **未 pin 版本**（requirements 为 `torch>=2.1.0`，专属索引取最新版），
   可复现性弱于已固定的基础镜像；2026-11-14 实测默认取到 **`torch-2.15.0+cpu`**，
   与 `transformers-5.23.0` 一同装成。若要锁版本，需在两条 `RUN` 与 requirements 间
   统一一个明确版本号，并实测 CPU / GPU 两个索引都提供该 wheel（`+cpu` 后缀版本
   在 cu121 索引里通常不存在，直接 pin 会破坏 GPU 构建）。
2. **镜像漏洞扫描：部分落地**。`security.yml` 已有 Trivy，但 `scan-type: fs` 扫的是
   **仓库文件**，不是镜像层；`docker scout cves` 未接入。建议在 `release.yml` 的
   `Build and push Docker image` 之后，对已推送的镜像跑一次
   `trivy image <tag>`（复用同一矩阵，避免重复构建 4 次）。本轮未改发布流水线。
3. **distroless**：生产镜像可进一步尝试 `gcr.io/distroless/static` 替代 `alpine`
   （需把 `wget` 健康检查改为二进制内建端点或 `HEALTHCHECK` 之外的方式）。
4. **PR 级镜像可构建性**：§6 指出的缺口——目前只有发布时才用 Buildx 构建全部镜像。
   建议新增只构建不推送的 workflow（按 `**/Dockerfile*` 与 `requirements*.txt` 触发），
   并复用 `type=gha` 缓存；AI 服务镜像拉取 torch 成本高，可单独设更长 timeout。
5. **体积实测：AI 服务已落地，其余待做**。2026-11-14 已实跑 AI 服务全量构建并回填 §4
   （1.33 GB / `torch-2.15.0+cpu` / 无非 root 问题）；backend / frontend / sidecar
   三镜像仍需在有完整构建环境时补 `docker images` 实测，不得沿用估算冒充实测。

本轮（2026-11-14）已落地的是 §3.3 的 torch 索引修正、Python builder 的 pip 缓存挂载、
§5 的 `make build-images` 真实接线、§6/§8 的口径纠正，以及
`scripts/__tests__/docker-optimization.test.js` 的 8 个契约用例与反向验证；
**未改发布流水线、未推送镜像、未重建 prod**。

2026-11-14 追加实跑闭环：本机全量构建 AI 服务镜像成功（`BUILD_EXIT=0`，5m37s），
从构建日志与容器内双重确认 §3.3 的 `--index-url` 修正确实生效，§4 已回填 1.33 GB 实测值。
验证镜像 tag 为 `itsm-ai-service:verify-20261114`，仅用于本机验证，
**不是发布产物，也未替换 prod 的 `itsm-ai-service:v1.6.58`**。
