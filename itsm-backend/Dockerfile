# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# 接收构建参数（禁用代理确保直连）
ARG HTTP_PROXY=
ARG HTTPS_PROXY=
ARG http_proxy=
ARG https_proxy=
ARG no_proxy=localhost,127.0.0.1

# 安装 git（go mod download 需要）
# Must unset proxy before apk since apk respects HTTP_PROXY env var
RUN unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy ALL_PROXY all_proxy && \
    apk add --no-cache git ca-certificates

# 接收构建参数（禁用代理确保直连）
# Note: Use Chinese mirror goproxy.cn for better connectivity in China region
# Must unset proxy vars to prevent interference with direct connections
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=off
ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=${GOSUMDB}

# Copy go mod files
COPY go.mod go.sum ./

# go mod download - use goproxy.cn mirror
RUN unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy ALL_PROXY all_proxy && \
    GOPROXY=https://goproxy.cn,direct GOSUMDB=off go mod download

# Copy source code
COPY . ./

# Build the application
# Unset proxy vars and use goproxy.cn for dependencies
RUN unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy ALL_PROXY all_proxy && \
    GOPROXY=https://goproxy.cn,direct CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

# Create non-root user
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

# Copy binary from builder
COPY --from=builder /app/main .
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/config/seed ./config/seed

# Create required directories
RUN mkdir -p /app/logs /app/uploads /app/temp && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8090/api/v1/health || exit 1

# Run the application
CMD ["./main"]
