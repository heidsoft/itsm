# Go 工具链对齐（GOTOOLCHAIN）

> Companion to **PR-0.2** of the Test Improvement Plan. See plan section
> "阶段 0 / PR-0.2" for the originating rationale.

## 1. 现状（v1.1 时点）

| 项 | 值 | 来源 |
|---|---|---|
| `itsm-backend/go.mod` 中 `go` 指令 | `1.25.13` | `itsm-backend/go.mod:3` |
| CI 安装版本 | `1.25.13`（按 `go-version-file` 自动跟随） | `.github/workflows/backend-ci.yml` 多处 `actions/setup-go@v7` |
| GH Actions 公共 README badge | `Go 1.25.13` | `README.md:9` |
| Dockerfile.prod 基础镜像 | `mirror.gcr.io/library/golang:1.25.13-alpine` | `itsm-backend/Dockerfile.prod:16` |
| Dockerfile dev 基础镜像 | `golang:1.25.13-alpine` | `itsm-backend/Dockerfile:18` |
| 本地开发机 | 通常 `1.25.x` 系列但与上游有偏差 | docs/ci/coverage-v1.1.md 记录 macOS Go `1.25.6` |
| README 给出的提示 | `GOTOOLCHAIN=auto go test ./...` | `README.md:248` |

## 2. 为什么需要 `GOTOOLCHAIN=auto`

`itsm-backend/go.mod` 顶部声明 `go 1.25.13`。Go 1.21+ 引入 **toolchain directive**：

- 当本地 `go version` < `go.mod` 要求时，**不设置 GOTOOLCHAIN** 会直接拒绝构建（"go.mod requires go >= 1.25.13 (running go 1.25.6; GOTOOLCHAIN=local)"）。
- 设置 `GOTOOLCHAIN=auto` 后，Go 工具链会自动下载并切换到所需版本（前提：本机 `go` ≥ 1.21，且网络可达 `proxy.golang.org` 或 `goproxy.cn`）。

CI 端通过 `actions/setup-go@v7` + `go-version-file: itsm-backend/go.mod` 直接安装 1.25.13，无需 `GOTOOLCHAIN`；但本地开发体验不同时，**`GOTOOLCHAIN=auto` 是契约**。

## 3. 本仓库的契约

| 场景 | 命令 | 备注 |
|---|---|---|
| **CI** | `actions/setup-go@v7` + `go-version-file: itsm-backend/go.mod` | workflow 默认；新 workflow 模板固定用这种方式 |
| **本地 `go test` 一键** | `make backend-test`（新） | 内部走 `cd itsm-backend && GOTOOLCHAIN=auto go test ./...` |
| **本地完整覆盖** | `make coverage-report` | 内部已设置 `GOTOOLCHAIN=${GOTOOLCHAIN:-auto}` |
| **本地 vet/lint** | `make backend-vet`（新） | `cd itsm-backend && GOTOOLCHAIN=auto go vet ./...` |
| **本地 build** | `make backend-build`（新） | `cd itsm-backend && GOTOOLCHAIN=auto go build -o itsm ./main.go` |
| **Docker** | `Dockerfile.prod` / `Dockerfile` | 已固定 `golang:1.25.13-alpine`，无需 `GOTOOLCHAIN` |
| **国内网络** | `export GOPROXY=https://goproxy.cn,direct && export GOSUMDB=sum.golang.org.cn` | 与 `Dockerfile.prod` 一致，可避免 toolchain 自动下载失败 |

## 4. 关键变更点（本 PR 落地）

| 文件 | 改动 |
|---|---|
| `itsm-backend/README.md` | 修复"Go 1.21+"为"Go 1.25.12"，并添加"GOTOOLCHAIN=auto 工具链说明"小节 |
| `itsm-backend/Makefile` *(新)* | 提供 `test` / `vet` / `build` / `tidy` / `cover` 目标；统一 `GOTOOLCHAIN=auto` |
| `Makefile`（根目录） | 新增 `backend-test` / `backend-vet` / `backend-build` / `backend-cover` 包装 |
| `.github/workflows/backend-ci.yml` | 顶层 `env` 增加 `GOTOOLCHAIN: auto` 显式声明，消除"是否启用自动切链"歧义 |
| `.github/workflows/test-coverage-guard.yml` | 不涉及 Go 测试，无需改动 |
| `docs/testing/go-toolchain.md` *(本文件)* | 用作长期文档 |

## 5. 验证步骤

```bash
# 1. CI 链路（actions/setup-go 自动满足）
.github/workflows/backend-ci.yml
  → go-version-file: itsm-backend/go.mod  → 安装 1.25.13

# 2. 本地链路（模拟本地 1.25.6）
go version                     # go1.25.6 darwin/arm64
cd itsm-backend && GOTOOLCHAIN=auto go version
# → go1.25.13 (toolchain auto-downloaded)

# 3. 跑测试不应再报工具链错误
make backend-test SKIP_DOCKER=1   # 实际需测试库；至少应看到 "ok" 字样
```

## 6. 兼容性矩阵

| 本地 `go` | 不设 `GOTOOLCHAIN` | 设置 `GOTOOLCHAIN=local` | 设置 `GOTOOLCHAIN=auto` |
|---|---|---|---|
| `>= 1.25.13` | ✅ | ✅ | ✅ |
| `1.21 ~ 1.25.11` | ❌ 报工具链错误 | ❌ 报工具链错误 | ✅ 自动下载 1.25.13 |
| `< 1.21` | ❌ | ❌ | ❌（Go 1.21+ 才支持 toolchain directive） |

## 7. 与 ROADMAP / CHANGELOG 的关系

- CHANGELOG v1.0.7+ 已记录 "Raised the backend build and release toolchain to Go 1.25.12"
- 此 PR 不引入新的 Go 升版；只把"工具链契约"显式化、可复现化
- 未来若 go.mod 升到 1.25.13 等新版本，只需更新 Dockerfile / docs/testing/go-toolchain.md，无需改 contract
