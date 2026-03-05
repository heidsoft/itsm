# ITSM 日志管理指南

**最后更新**: 2026-03-04
**版本**: v1.0

---

## 📋 目录

- [日志概述](#日志概述)
- [日志配置](#日志配置)
- [日志查看](#日志查看)
- [日志分析](#日志分析)
- [日志级别](#日志级别)
- [日志格式](#日志格式)
- [结构化日志](#结构化日志)
- [日志收集](#日志收集)
- [常见问题](#常见问题)

---

## 日志概述

ITSM 系统采用结构化日志记录，包含以下日志：

| 日志类型 | 说明 | 位置 |
|----------|------|------|
| 应用日志 | 业务逻辑日志 | `./logs/app.log` |
| 访问日志 | HTTP 请求日志 | `./logs/access.log` |
| 错误日志 | 错误信息 | `./logs/error.log` |
| 审计日志 | 操作审计 | `./logs/audit.log` |

---

## 日志配置

### 环境变量配置

```bash
# 日志级别
LOG_LEVEL=debug           # debug/info/warn/error

# 日志目录
LOG_PATH=./logs

# 日志文件配置
LOG_MAX_SIZE=100          # 单文件最大 MB
LOG_MAX_AGE=30            # 保留天数
LOG_COMPRESS=true         # 是否压缩
```

### 配置文件

在 `.env` 或 `.env.prod` 中配置：

```bash
# ========== 日志配置 ==========
LOG_LEVEL=error
LOG_PATH=./logs
LOG_MAX_SIZE=100
LOG_MAX_AGE=30
LOG_COMPRESS=true

# JSON 格式日志
LOG_FORMAT=json

# 输出到标准输出（Docker）
LOG_STDOUT=true
```

---

## 日志查看

### Docker 环境

```bash
# 查看所有日志
docker compose logs -f

# 查看后端日志
docker compose logs -f backend

# 查看前端日志
docker compose logs -f frontend

# 查看最近 100 行
docker compose logs --tail=100 backend

# 查看错误日志
docker compose logs backend | grep ERROR

# 实时查看错误
docker compose logs -f backend | grep -i error
```

### 本地环境

```bash
# 实时查看应用日志
tail -f logs/app.log

# 查看错误日志
tail -f logs/error.log

# 查看最近 100 行
tail -n 100 logs/app.log

# 搜索错误
grep -i error logs/app.log

# 搜索特定内容
grep "ticket" logs/app.log

# 查看带行号
cat -n logs/app.log | tail -100
```

### Systemd 环境

```bash
# 查看服务日志
journalctl -u itsm -f

# 查看最近 100 行
journalctl -u itsm -n 100

# 按时间查看
journalctl -u itsm --since "1 hour ago"

# 按错误级别
journalctl -p err -u itsm
```

---

## 日志分析

### 常用命令

```bash
# 统计错误数量
grep -c ERROR logs/app.log

# 统计请求数量
grep "200 OK" logs/access.log | wc -l

# 查找慢请求
grep "slow" logs/app.log

# 按时间排序
sort -t'[' -k2 logs/app.log

# 提取时间戳
awk '{print $1, $2}' logs/app.log | head -10
```

### 日志聚合

使用 `jq` 解析 JSON 日志：

```bash
# 安装 jq
brew install jq  # macOS
sudo apt install jq  # Ubuntu

# 解析 JSON 日志
cat logs/app.log | jq .

# 过滤特定级别
cat logs/app.log | jq 'select(.level=="error")'

# 统计错误类型
cat logs/app.log | jq -r '.error' | sort | uniq -c | sort -rn
```

---

## 日志级别

| 级别 | 值 | 说明 | 使用场景 |
|------|-----|------|----------|
| `debug` | 0 | 调试信息 | 开发调试 |
| `info` | 1 | 正常信息 | 日常运行 |
| `warn` | 2 | 警告信息 | 需要注意 |
| `error` | 3 | 错误信息 | 错误日志 |
| `fatal` | 4 | 致命错误 | 进程退出 |

### 级别配置

```bash
# 开发环境
LOG_LEVEL=debug

# 测试环境
LOG_LEVEL=info

# 生产环境
LOG_LEVEL=error
```

---

## 日志格式

### 文本格式

```
2026-03-04 10:30:45 [INFO] [ticket_service.go:123] CreateTicket: ticket created successfully
2026-03-04 10:30:45 [ERROR] [db.go:456] Query failed: connection timeout
```

### JSON 格式

```json
{
  "time": "2026-03-04T10:30:45.123Z",
  "level": "info",
  "caller": "ticket_service.go:123",
  "message": "CreateTicket: ticket created successfully",
  "ticket_id": 12345,
  "user_id": 100
}
```

---

## 结构化日志

### 使用 Zap 日志库

```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
defer logger.Sync()

// 记录结构化日志
logger.Info("ticket created",
    zap.Int("ticket_id", 12345),
    zap.String("title", "Network issue"),
    zap.String("status", "open"),
)
```

### 日志字段规范

| 字段 | 类型 | 说明 |
|------|------|------|
| `time` | string | 时间戳 |
| `level` | string | 日志级别 |
| `caller` | string | 调用位置 |
| `message` | string | 日志消息 |
| `trace_id` | string | 请求追踪 ID |
| `user_id` | int | 用户 ID |
| `request_id` | string | 请求 ID |

---

## 日志收集

### 使用 ELK Stack

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌───────────┐
│  应用     │───▶│  Filebeat│───▶│Elasticsearch│───▶│  Kibana   │
│  日志     │    │          │    │            │    │           │
└──────────┘    └──────────┘    └──────────┘    └───────────┘
```

### Filebeat 配置

```yaml
# filebeat.yml
filebeat.inputs:
  - type: log
    paths:
      - /var/log/itsm/*.log
    json:
      keys_under_root: true
      overwrite_keys: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
```

### 使用 Loki + Grafana

```yaml
# promtail.yml
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: itsm
    static_configs:
      - targets:
          - localhost
        labels:
          job: itsm
          __path__: /var/log/itsm/*.log
```

---

## 日志轮转

### 使用 logrotate

创建 `/etc/logrotate.d/itsm`:

```
/opt/itsm/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0640 itsm itsm
    postrotate
        kill -USR1 $(cat /var/run/itsm.pid) 2>/dev/null || true
    endscript
}
```

### 测试配置

```bash
# 测试配置
logrotate -d /etc/logrotate.d/itsm

# 强制轮转
logrotate -f /etc/logrotate.d/itsm
```

---

## 性能优化

### 异步日志

```go
// 使用异步日志
logger, _ := zap.NewProduction()
defer logger.Sync()

// 异步记录
go func() {
    logger.Info("heavy operation", zap.Int64("duration", 1000))
}()
```

### 日志采样

```go
// 只记录 10% 的请求
if rand.Intn(100) < 10 {
    logger.Info("request details", zap.String("path", path))
}
```

---

## 常见问题

### 问题 1：日志文件过大

**解决方案**:

1. 配置日志轮转
2. 调整日志级别
3. 使用异步日志

```bash
# 限制日志大小
LOG_MAX_SIZE=100
LOG_MAX_AGE=30
```

### 问题 2：日志无法写入

**解决方案**:

```bash
# 检查目录权限
ls -la logs/

# 创建目录
mkdir -p logs
chmod 755 logs
```

### 问题 3：需要实时监控

**解决方案**:

```bash
# 使用 tail -f
tail -f logs/app.log

# 使用 watch
watch -n 1 "tail -20 logs/app.log"

# 使用 entr
echo logs/app.log | entr cat
```

---

## 日志查询示例

### 查询特定用户操作

```bash
grep "user_id=123" logs/app.log | grep -v DEBUG
```

### 查询慢请求

```bash
grep "slow" logs/app.log | awk '{print $1, $2, $NF}'
```

### 查询错误堆栈

```bash
grep -A 10 "ERROR" logs/app.log
```

---

## 监控告警

### 配置错误告警

使用 Prometheus Alertmanager：

```yaml
groups:
  - name: itsm_errors
    rules:
      - alert: HighErrorRate
        expr: rate(itsm_errors_total[5m]) > 0.1
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
```

---

**文档维护**: ITSM 开发团队
**最后更新**: 2026-03-04
