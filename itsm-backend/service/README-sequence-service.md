# Redis 序列号生成服务

## 模块概述

`sequence_service.go` 提供基于 Redis 的高性能序列号生成服务，支持按月重置、自动过期和数据库同步。

## 核心功能

### 1. 序列号生成

- **原子操作**：使用 Redis `INCR` 命令保证并发安全
- **自动过期**：支持设置过期时间，实现按月/按年重置
- **DB 同步**：Redis 初始化时从数据库同步起点，避免编号冲突

### 2. 序列号格式

工单序列号格式：`TKT-YYYYMM-NNNN`

示例：`TKT-202607-0001` 表示 2026年7月的第 1 个工单

## API 接口

### GetNextSequence

```go
// 获取下一个序列号（简单模式）
seq, err := sequenceService.GetNextSequence(ctx, "sequence:ticket:202607")
// seq = 1, 2, 3, ...
```

### GetNextSequenceWithExpiry

```go
// 获取下一个序列号并设置过期时间（推荐）
expiredAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
seq, err := sequenceService.GetNextSequenceWithExpiry(ctx, "sequence:ticket:202607", expiredAt)
```

### GetCurrentSequence

```go
// 获取当前序列号（不递增）
seq, err := sequenceService.GetCurrentSequence(ctx, "sequence:ticket:202607")
```

### ResetSequence

```go
// 重置序列号
err := sequenceService.ResetSequence(ctx, "sequence:ticket:202607")
```

## 数据库同步机制

当 Redis 序列不存在或为 0 时，自动从数据库同步：

```
┌─────────────┐     序列不存在      ┌─────────────┐
│   请求1     │ ─────────────────►  │   查询DB    │
│ GetNextSeq  │                     │ MAX(seq)    │
└─────────────┘                     └─────────────┘
        │                                   │
        │  SetNX(dbMax)                     │
        ▼                                   ▼
┌─────────────────────────────────────────────────────┐
│                    Redis                             │
│  sequence:ticket:202607 = 1000 (从DB同步)            │
└─────────────────────────────────────────────────────┘
        │
        │  INCR
        ▼
┌─────────────┐
│   返回 1001 │
└─────────────┘
```

## 使用示例

### 初始化服务

```go
sequenceService := service.NewSequenceService(
    "localhost",      // host
    6379,             // port
    "password",       // password
    0,                // db
    logger,
)

// 设置DB查询函数（用于Redis初始化同步）
sequenceService.SetDBQueryFunc(func(key string) (int64, error) {
    // 从数据库查询当前最大序列号
    return queryMaxTicketSeqFromDB(key)
})
```

### 工单号生成

```go
// 生成工单号
seq, err := s.sequenceService.GetNextSequenceWithExpiry(
    ctx,
    fmt.Sprintf("sequence:ticket:%d:%04d%02d", tenantID, year, month),
    time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC),
)
ticketNumber := fmt.Sprintf("TKT-%04d%02d-%04d", year, month, seq)
```

## 错误处理

| 错误场景 | 处理方式 |
|---------|---------|
| Redis 连接失败 | 返回 nil 服务，上层使用数据库回退 |
| INCR 命令失败 | 返回错误，调用方需重试 |
| DB 同步失败 | 跳过同步，继续使用 Redis 初始值 1 |

## 配置项

在 `config.yaml` 中配置：

```yaml
redis:
  host: localhost
  port: 6379
  password: ""
  db: 0  # 序列号使用 DB 0
```

## 注意事项

1. **高可用**：生产环境建议使用 Redis Cluster
2. **数据持久化**：确保 Redis 开启 RDB/AOF 持久化
3. **监控告警**：监控 Redis 连接状态和序列号增长速率
4. **过期策略**：合理设置过期时间，避免序列号无限增长

## 相关文件

- `internal/container/container.go` - 依赖注入容器
- `service/ticket_service.go` - 工单服务（使用序列号）
