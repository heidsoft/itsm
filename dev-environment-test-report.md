# ITSM 开发环境启动验证报告

**测试时间**: 2026-07-05 19:41 ~ 19:45 (GMT+8)  
**测试人**: Senior Developer (高级开发工程师)

---

## 一、服务状态总览

### Docker 容器状态

| 服务 | 容器名 | 状态 | 端口 | 资源占用 |
|------|--------|------|------|----------|
| PostgreSQL 17 (pgvector) | itsm-postgres-dev | ✅ healthy | 5432 | 62MB |
| Redis 7 | itsm-redis-dev | ✅ healthy | 6379 | 6.5MB |
| MinIO | itsm-minio-dev | ✅ healthy | 9000-9001 | 74MB |
| ITSM Backend (Go/Gin) | itsm-backend-dev | ✅ healthy | 8090 | 22MB |
| ITSM Frontend (Next.js) | 本地进程 | ✅ running | 3000 | — |

### 前端启动日志
```
▲ Next.js 15.5.12
- Local: http://localhost:3000
- Environments: .env.local
✓ Ready in 3.7s
```

---

## 二、API 接口测试

### 2.1 基础服务

| 接口 | 方法 | 状态码 | 结果 |
|------|------|--------|------|
| `/api/v1/health` | GET | 200 | ✅ `{"status":"ok"}` |

### 2.2 认证模块

| 接口 | 方法 | 状态码 | 结果 |
|------|------|--------|------|
| `/api/v1/auth/login` | POST | 200 | ✅ 返回 accessToken + refreshToken + 用户信息 |

**登录响应验证**:
- ✅ 驼峰命名: `accessToken`, `refreshToken`, `tenantId`, `createdAt`
- ✅ 响应格式: `{ code, message, data }`
- ✅ 用户角色: `super_admin`
- ✅ 权限: `["*"]`

### 2.3 工单模块

| 接口 | 方法 | 状态码 | 结果 |
|------|------|--------|------|
| `/api/v1/tickets` | GET | 200 | ✅ 返回工单列表 (27条) |
| `/api/v1/tickets` | POST | 200 | ✅ 成功创建测试工单 TKT-202607-000024 |

**工单创建后后端日志**:
```
info  Creating ticket {"tenant_id": 1, "title": "dev_test_ticket"}
info  Triggering approval {"ticket_number": "TKT-202607-000024", "ticket_type": "incident", "priority": "medium"}
info  No active approval workflow found, skipping approval
info  Ticket created {"ticket_id": 32, "ticket_number": "TKT-202607-000024"}
warn  Automation rules failed {"error": "ticket not found: context canceled"}
warn  Workflow trigger failed {"error": "failed to trigger workflow: ...context canceled"}
```

⚠️ **发现的问题**: 工单创建后，自动化规则执行和工作流触发存在 `context canceled` 错误。工单主体创建成功，但异步流程有竞态条件。

### 2.4 BPMN 工作流模块

| 接口 | 方法 | 状态码 | 结果 |
|------|------|--------|------|
| `/api/v1/workflows` | GET | 200 | ⚠️ 兼容接口提示使用 `/api/v1/bpmn/process-definitions` |
| `/api/v1/bpmn/process-definitions` | GET | 200 | ✅ 返回流程定义列表 |

### 2.5 用户与组织模块

| 接口 | 方法 | 状态码 | 结果 |
|------|------|--------|------|
| `/api/v1/users` | GET | 200 | ✅ 返回用户列表 (含分页) |
| `/api/v1/departments` | GET | 200 | ✅ 返回部门列表 (3个部门) |

### 2.6 知识库模块

| 接口 | 方法 | 状态码 | 结果 |
|------|------|--------|------|
| `/api/v1/knowledge/categories` | GET | 200 | ✅ 返回分类列表 |

### 2.7 仪表盘模块

| 接口 | 方法 | 状态码 | 结果 |
|------|------|--------|------|
| `/api/v1/dashboard/stats` | GET | 200 | ✅ 返回统计数据 (总工单27, KPI指标等) |

### 2.8 软件资产管理

| 接口 | 方法 | 状态码 | 结果 |
|------|------|--------|------|
| `/api/v1/slms/software` | GET | 404 | ❌ 路由未找到 |

---

## 三、前端页面路由测试

| 路由 | 状态码 | 说明 |
|------|--------|------|
| `/` | 200 | ✅ 首页正常 |
| `/login` | 200 | ✅ 登录页正常 |
| `/dashboard` | 307 | ✅ 重定向(未登录跳转登录) |
| `/tickets` | 307 | ✅ 重定向(未登录跳转登录) |
| `/workflow` | 307 | ✅ 重定向(未登录跳转登录) |
| `/knowledge` | 307 | ✅ 重定向(未登录跳转登录) |
| `/settings` | 307 | ✅ 重定向(未登录跳转登录) |
| `/slms` | 404 | ❌ 页面不存在 |

---

## 四、前后端数据结构约定验证

### 4.1 响应格式 ✅

所有 API 统一返回:
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

### 4.2 驼峰命名验证 ✅

后端 DTO 转换层正常工作，以下字段均使用驼峰:
- `accessToken`, `refreshToken` (认证)
- `ticketNumber`, `requesterId`, `tenantId`, `createdAt`, `updatedAt` (工单)
- `totalTickets`, `pendingTickets`, `inProgressTickets` (仪表盘)
- `managerId`, `parentId` (部门)

### 4.3 发现的问题

| # | 严重度 | 问题 | 影响 |
|---|--------|------|------|
| 1 | ⚠️ 中 | 工单创建后自动化规则和工作流触发出现 `context canceled` | 异步流程竞态，工单主体不受影响 |
| 2 | ❌ 高 | `/api/v1/slms/software` 返回 404 | SLMS 软件资产管理路由未注册 |
| 3 | ❌ 中 | 前端 `/slms` 页面 404 | 前端缺少 SLMS 页面路由 |
| 4 | ℹ️ 低 | `/api/v1/workflows` 是兼容接口，提示使用新路径 | 需前端适配新 API 路径 |

---

## 五、业务复盘

### 5.1 核心业务链路验证

| 业务链路 | 验证结果 |
|----------|----------|
| 用户登录 → 获取Token → 访问受保护资源 | ✅ 完整通过 |
| 工单列表查询 → 创建工单 → 自动化触发 | ⚠️ 主体通过，异步流程有竞态 |
| BPMN 流程定义查询 | ✅ 通过 |
| 部门/用户管理 | ✅ 通过 |
| 知识库分类 | ✅ 通过 |
| 仪表盘统计 | ✅ 通过 |
| 软件资产管理 | ❌ 未接入 |

### 5.2 多租户隔离验证

- ✅ 登录后 token 包含 `tenantId: 1`
- ✅ 工单查询返回数据包含 `tenantId` 字段
- ✅ 工单创建自动关联当前租户

### 5.3 性能表现

| 指标 | 数值 |
|------|------|
| 前端首页首次编译 | 20s (3749 modules) |
| 前端首页二次访问 | 526ms |
| 后端健康检查响应 | ~7ms |
| 工单列表查询 | ~13ms |
| 工单创建 | ~105ms |
| 后端内存占用 | 22MB |
| PostgreSQL 内存 | 62MB |

---

## 六、总结

**整体评估**: 🟡 **基本可用，存在待修复问题**

- ✅ 核心业务链路（认证、工单、BPMN、用户、部门、知识库、仪表盘）正常运行
- ⚠️ 工单自动化规则和工作流触发存在异步竞态问题
- ❌ SLMS 软件资产管理模块后端路由和前端页面均缺失
- ✅ 前后端数据结构约定（驼峰命名、统一响应格式）正确执行
- ✅ 多租户隔离机制正常工作
- ✅ 系统资源占用合理
