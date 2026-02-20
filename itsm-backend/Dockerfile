# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files
COPY itsm-backend/go.mod itsm-backend/go.sum ./
RUN go mod download

# Copy source code
COPY itsm-backend/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Create non-root user
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

# Copy binary from builder
COPY --from=builder /app/main .
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/migrations ./migrations

# Create required directories
RUN mkdir -p /app/logs /app/uploads /app/temp && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./main"]
