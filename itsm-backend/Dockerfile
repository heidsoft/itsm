# syntax=docker/dockerfile:1.7
#
# ITSM Backend — development image (multi-stage)
#
# Optimizations vs. the previous version:
#   - Pinned base image (golang:1.25-alpine) instead of floating "golang:alpine"
#   - BuildKit cache mounts for go modules + build cache (repeat builds are fast)
#   - trimpath + "-s -w" linker flags to shrink the binary
#   - Static, non-root runtime on a pinned alpine:3.20
#   - Only the binary + runtime assets (config.yaml, migrations, config/seed) ship
#
# NOTE: dev builds intentionally keep debug symbols off-net but the base image
# matches the prod toolchain so dev/prod behavior stays aligned.

# ============================================================
# Stage 1: Build
# ============================================================
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

WORKDIR /src

# Region-tunable module proxy (China default; override for overseas CI).
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.org
ARG TARGETOS=linux
ARG TARGETARCH=amd64

ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB} \
    GOFLAGS=-p=2 \
    GOMAXPROCS=2 \
    GOGC=50 \
    GOMEMLIMIT=2GiB

# ca-certificates for any TLS module fetches; git for go mod download.
RUN apk add --no-cache ca-certificates git

# Cache module download across builds.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Build (sources copied after deps so the dep layer stays cached).
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/main ./main.go

# ============================================================
# Stage 2: Runtime
# ============================================================
FROM alpine:3.20

WORKDIR /app

# Non-root user + required directories.
RUN addgroup -S app && adduser -S -G app app \
    && mkdir -p /app/logs /app/uploads /app/temp /app/config \
    && chown -R app:app /app

# ca-certificates already ship with alpine; copy only what runtime needs.
COPY --from=builder --chown=app:app /out/main ./main
COPY --chown=app:app config.yaml ./
COPY --chown=app:app migrations ./migrations
COPY --chown=app:app config/seed ./config/seed

USER app

EXPOSE 8090

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8090/api/v1/health || exit 1

CMD ["./main"]
