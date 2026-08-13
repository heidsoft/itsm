# ITSM Backend

## Quick Start

1. Prerequisites: **Go 1.25.12** (see "Go 工具链" below), PostgreSQL
2. Configure env or `itsm-backend/config.yaml`; optional: `.env` (see project root `.env.example`)
3. Choose deployment mode with `DEPLOYMENT_MODE=private|saas|saas_msp`
3. Install deps and build:

```
make setup
```

1. Run:

```
make run                    # uses itsm-backend/Makefile, GOTOOLCHAIN=auto baked in
# or
make backend-build && ./itsm-backend/itsm
```

API base: `http://localhost:8090/api/v1`

## Go 工具链（GOTOOLCHAIN）

`itsm-backend/go.mod` 声明 `go 1.25.12`。CI 通过 `actions/setup-go@v7` + `go-version-file: itsm-backend/go.mod` 自动安装该版本，本地推荐以下三种之一保持一致：

```bash
# 1. 推荐：装 go1.25.12，让 `go version` 直接对齐
go install golang.org/dl/go1.25.12@latest && go1.25.12 download

# 2. 备选：保留本地任意版本，让 Go 自动下载切换
export GOTOOLCHAIN=auto         # 本仓库所有脚本/Make 目标都已默认开启

# 3. 国内/受限网络，加 goproxy.cn（与 Dockerfile.prod 一致）
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.org.cn
```

便捷命令（`itsm-backend/Makefile` 已默认 `GOTOOLCHAIN=auto`）：

| 命令 | 作用 |
|---|---|
| `make backend-test` | `go test ./... -count=1` |
| `make backend-test-ci` | `go test ./... -coverprofile -covermode=set`（对齐 CI） |
| `make backend-vet` | `go vet ./...` |
| `make backend-build` | `go build -o itsm ./main.go` |
| `make backend-cover` / `backend-cover-html` | 覆盖率 profile / HTML |
| `make backend-lint` | staticcheck v0.6.1（与 CI 同版本） |
| `make backend-tidy` | `go mod tidy` |

详见：[`docs/testing/go-toolchain.md`](../../docs/testing/go-toolchain.md)。

## Initialization

- `ITSM_BOOTSTRAP_ONLY=true`: run one-shot migration + seed and exit
- `ITSM_AUTO_MIGRATE=true`: enable schema migration during bootstrap
- `ITSM_AUTO_SEED=true`: enable idempotent seed during bootstrap

In Docker Compose, the recommended flow is:

1. `itsm-init` runs once with `ITSM_BOOTSTRAP_ONLY=true`
2. `itsm-backend` starts after init completes
3. Frontend proxies browser requests through same-origin `/api`

For browser compatibility on localhost, auth cookies are host-only and only marked
`Secure` when the request is actually served over HTTPS.

## Swagger Docs

After server starts, open:

- Swagger UI: `http://localhost:8090/swagger/index.html`
- OpenAPI JSON: `http://localhost:8090/docs/swagger.json`

Generate docs locally:

```
# Install once
GO111MODULE=on go install github.com/swaggo/swag/cmd/swag@latest
$(go env GOPATH)/bin/swag init -g main.go -o ./docs
```

## Multi-tenancy & Auth

- JWT Bearer in `Authorization`
- `X-Tenant-Code` header supported; tenant id validated against JWT
