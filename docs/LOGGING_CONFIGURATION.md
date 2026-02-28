# 日志系统配置指南

**更新时间**: 2026-02-28  
**维护者**: 开发团队

---

## 📋 概述

ITSM 系统采用统一的日志规范，前后端均支持多级日志输出和环境自适应配置。

---

## 🔧 后端日志配置 (Go/Zap)

### 配置文件位置

`itsm-backend/config/config.yaml` 或通过环境变量配置

### 配置参数

```yaml
log:
  level: info           # debug/info/warn/error
  path: ./logs          # 日志文件目录
  max_size: 100         # 单个日志文件最大大小 (MB)
  max_backups: 5        # 保留的旧日志文件数量
  max_age: 30           # 日志文件保留天数
  compress: true        # 是否压缩旧日志文件
  local_time: true      # 使用本地时间戳
  development: false    # 开发模式 (输出到 console)
```

### 环境变量覆盖

生产环境推荐使用环境变量覆盖配置：

```bash
# 日志级别 (生产环境建议：error 或 warn)
export LOG_LEVEL=error

# 日志路径
export LOG_PATH=/var/log/itsm

# 开发模式 (生产环境应为 false)
export LOG_DEVELOPMENT=false
```

### 日志级别说明

| 级别 | 说明 | 使用场景 |
|------|------|----------|
| `debug` | 调试信息 | 本地开发，问题排查 |
| `info` | 一般信息 | 开发/测试环境 |
| `warn` | 警告信息 | 生产环境 (推荐) |
| `error` | 错误信息 | 生产环境 (严格模式) |

### 使用示例

```go
import "go.uber.org/zap"

// 获取 logger (从 bootstrap 包)
logger := bootstrap.GetLogger()

// 日志输出
logger.Debug("调试信息", zap.String("user", "张三"))
logger.Info("用户登录", zap.String("userId", "123"))
logger.Warn("SLA 即将超时", zap.Int("ticketId", 456))
logger.Error("数据库连接失败", zap.Error(err))
```

### 生产环境推荐配置

```yaml
log:
  level: warn
  path: /var/log/itsm
  max_size: 100
  max_backups: 10
  max_age: 90
  compress: true
  local_time: true
  development: false
```

---

## 🎨 前端日志配置 (TypeScript)

### 日志工具位置

`itsm-frontend/src/lib/env.ts`

### 日志 API

```typescript
import { logger } from '@/lib/env';

// 调试日志 (仅开发环境)
logger.debug('组件渲染', { props });

// 信息日志 (仅开发环境)
logger.info('API 调用成功', { url, method });

// 警告日志 (仅开发环境)
logger.warn('表单验证失败', { errors });

// 错误日志 (所有环境)
logger.error('网络请求失败', error);

// 性能日志 (仅生产环境)
logger.performance('页面加载', { duration: '1.2s' });

// 安全日志 (所有环境，生产环境上报)
logger.security('登录尝试', { userId, ip });
```

### 环境配置

通过 `NODE_ENV` 环境变量控制日志行为：

| 环境 | `consoleLogs` | `performanceMonitoring` | `errorReporting` |
|------|---------------|------------------------|------------------|
| `development` | ✅ | ❌ | ❌ |
| `production` | ❌ | ✅ | ✅ |
| `test` | ❌ | ❌ | ❌ |

### 生产环境配置

确保 `.env.production` 或部署配置中设置：

```bash
NODE_ENV=production
NEXT_PUBLIC_API_URL=https://api.your-domain.com
```

### 性能监控

```typescript
import { performance } from '@/lib/env';

// 同步函数性能测量
const result = performance.measure('数据加载', () => {
  return loadData();
});

// 异步函数性能测量
const data = await performance.measureAsync('API 调用', async () => {
  return fetch('/api/data');
});

// 手动计时
const timer = performance.start('组件渲染');
// ... 执行代码
performance.end(timer);
```

### 错误处理

```typescript
import { errorHandler } from '@/lib/env';

// API 错误处理
try {
  const response = await api.getUser(id);
} catch (error) {
  const result = errorHandler.handleApiError(error, '获取用户信息');
  // result: { success: false, error: '...', context: '...' }
}

// 验证错误处理
const validation = errorHandler.handleValidationError({
  email: ['邮箱格式不正确'],
  phone: ['手机号必填'],
});

// 网络错误处理
const networkResult = errorHandler.handleNetworkError(error);
if (networkResult.type === 'network') {
  // 显示网络错误提示
}
```

### 开发工具

```typescript
import { devTools } from '@/lib/env';

// 仅开发环境执行
devTools.onlyInDev(() => {
  console.log('调试信息，生产环境不执行');
}, 'fallback-value');

// 组件调试信息
devTools.debugInfo('TicketList', { tickets, filters });

// 性能标记 (浏览器 DevTools 可见)
devTools.mark('component-mount');
devTools.measure('render-time', 'component-mount', 'component-rendered');
```

---

## 🔐 安全注意事项

### 禁止输出的敏感信息

- ❌ 用户密码
- ❌ JWT Token
- ❌ 数据库连接字符串
- ❌ API Key / Secret
- ❌ 个人身份信息 (PII)

### 日志脱敏

对于必须记录的敏感信息，应进行脱敏处理：

```go
// Go 示例
func maskEmail(email string) string {
    if len(email) < 5 {
        return "***"
    }
    parts := strings.Split(email, "@")
    return parts[0][:2] + "***@" + parts[1]
}

logger.Info("用户登录", zap.String("email", maskEmail(user.Email)))
```

```typescript
// TypeScript 示例
const maskPhone = (phone: string) => {
  return phone.replace(/(\d{3})\d{4}(\d{4})/, '$1****$2');
};

logger.info('短信发送', { phone: maskPhone(user.phone) });
```

---

## 📊 日志收集与分析

### 推荐方案

1. **ELK Stack** (Elasticsearch + Logstash + Kibana)
2. **Loki + Grafana** (轻量级方案)
3. **云服务商日志服务** (阿里云 SLS, AWS CloudWatch)

### 日志格式

后端生产环境使用 JSON 格式，便于日志收集系统解析：

```json
{
  "level": "error",
  "time": "2026-02-28T16:00:00.000+08:00",
  "logger": "ticket-service",
  "caller": "service/ticket_service.go:123",
  "msg": "工单创建失败",
  "userId": "123",
  "error": "database connection timeout"
}
```

---

## 🧪 测试建议

### 单元测试中的日志

```go
// Go 测试
func TestTicketService(t *testing.T) {
    // 测试环境使用 Development 模式
    logger, _ := zap.NewDevelopment()
    defer logger.Sync()
    
    service := NewTicketService(logger)
    // ... 测试代码
}
```

```typescript
// TypeScript 测试
jest.mock('@/lib/env', () => ({
  logger: {
    debug: jest.fn(),
    info: jest.fn(),
    warn: jest.fn(),
    error: jest.fn(),
  },
}));
```

---

## 📝 变更日志

| 日期 | 变更 | 负责人 |
|------|------|--------|
| 2026-02-28 | 统一前后端日志配置文档 | 开发助手 |
| 2026-02-27 | 添加前端性能监控 | 前端团队 |
| 2026-02-20 | 后端 Zap 日志系统集成 | 后端团队 |

---

_日志是系统的眼睛，配置好它才能看清问题所在。_

**文档维护**: 开发团队  
**下次审查**: 2026-03-28
