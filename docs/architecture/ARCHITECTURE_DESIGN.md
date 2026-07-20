# ITSM 系统架构设计文档

## 1. 系统概述

### 1.1 项目背景

ITSM（IT Service Management）是一个企业级 IT 服务管理系统，遵循 ITIL v3/v4 标准，提供完整的 IT 服务管理流程，包括工单、事件、问题、变更、发布、服务目录、知识库、SLA 监控等核心功能。

### 1.2 设计目标

- **企业级**: 支持多租户、高可用、可扩展
- **AI 原生**: 内置 AI 能力，包括智能分类、摘要、RAG 等
- **流程驱动**: 基于 BPMN 的工作流引擎
- **生态友好**: 提供连接器市场，支持第三方系统集成
- **可观测**: 完善的监控、日志、审计能力

### 1.3 技术选型

| 层级 | 技术选型 | 说明 |
|------|---------|------|
| **前端** | Next.js + React + TypeScript | SSR 框架，类型安全 |
| **UI 组件** | Ant Design | 企业级 UI 组件库 |
| **状态管理** | Zustand | 轻量级状态管理 |
| **后端** | Go + Gin | 高性能 Web 框架 |
| **ORM** | Ent | 类型安全的 ORM |
| **数据库** | PostgreSQL | 关系型数据库 |
| **缓存** | Redis | 分布式缓存 |
| **工作流引擎** | 自研 BPMN 引擎 | 基于 nitram509/lib-bpmn-engine |
| **向量数据库** | PostgreSQL + pgvector | RAG 支持 |
| **消息队列** | Redis (可选) | 异步任务处理 |
| **日志** | Zap | 结构化日志 |
| **监控** | Prometheus + Grafana | 指标收集和可视化 |

---

## 2. 系统架构

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                前端层 (Frontend)                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │ Web 应用     │  │ 移动端 (待建) │  │ API 客户端   │  │ 管理后台     │   │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              网关层 (Gateway)                                │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐              │
│  │ 负载均衡        │  │ API 网关        │  │ WebSocket 网关  │              │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘              │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              应用层 (Application)                            │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        Controller 层 (API 入口)                         │ │
│  ├───────────────────────────────────────────────────────────────────────┤ │
│  │                        Service 层 (业务逻辑)                            │ │
│  ├───────────────────────────────────────────────────────────────────────┤ │
│  │                        Repository 层 (数据访问)                         │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐   │ │
│  │  │ 工单模块  │ │ 事件模块  │ │ 问题模块  │ │ 变更模块  │ │ 发布模块 │   │ │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └─────────┘   │ │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐   │ │
│  │  │ 服务目录  │ │ 知识库    │ │ SLA 监控  │ │ CMDB     │ │ 资产    │   │ │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └─────────┘   │ │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐   │ │
│  │  │ 工作流    │ │ 连接器    │ │ AI 服务   │ │ 通知     │ │ 权限    │   │ │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └─────────┘   │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              数据层 (Data)                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │ PostgreSQL   │  │ Redis        │  │ 对象存储     │  │ 搜索引擎     │   │
│  │ (主数据库)   │  │ (缓存/队列)  │  │ (文件)       │  │ (可选)       │   │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              基础设施层 (Infrastructure)                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │ 监控系统     │  │ 日志系统     │  │ 配置管理     │  │ 部署系统     │   │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 分层架构

#### 2.2.1 Controller 层

**职责**:
- 接收 HTTP 请求
- 参数验证和转换
- 调用 Service 层
- 统一响应格式
- 异常处理

**示例**:
```go
// controller/ticket_controller.go
func (c *TicketController) GetTicket(ctx *gin.Context) {
    id := ctx.Param("id")
    ticket, err := c.service.GetTicket(ctx, id)
    if err != nil {
        common.Fail(ctx, common.InternalErrorCode, err.Error())
        return
    }
    common.Success(ctx, dto.ToTicketResponse(ticket))
}
```

#### 2.2.2 Service 层

**职责**:
- 业务逻辑实现
- 事务管理
- 调用 Repository 层
- 事件发布
- 调用外部服务

**示例**:
```go
// service/ticket_service.go
func (s *TicketService) CreateTicket(ctx context.Context, req *dto.CreateTicketRequest) (*ent.Ticket, error) {
    // 业务逻辑
    tx, err := s.client.Tx(ctx)
    if err != nil {
        return nil, err
    }

    ticket, err := tx.Ticket.Create().
        SetTitle(req.Title).
        SetDescription(req.Description).
        Save(ctx)

    if err != nil {
        tx.Rollback()
        return nil, err
    }

    tx.Commit()
    return ticket, nil
}
```

#### 2.2.3 Repository 层

**职责**:
- 数据访问封装
- 查询构建
- 租户隔离
- 缓存集成

**示例**:
```go
// repository/ticket/repository.go
func (r *EntRepository) FindByID(ctx context.Context, id int, tenantID int) (*ent.Ticket, error) {
    return r.client.Ticket.Query().
        Where(ticket.ID(id)).
        Where(ticket.TenantIDEQ(tenantID)).
        Only(ctx)
}
```

---

## 3. 核心模块设计

### 3.1 多租户架构

#### 3.1.1 租户隔离策略

采用 **Schema 隔离** + **行级隔离** 的混合策略：

```
PostgreSQL
├── public (系统表)
├── tenant_1 (租户 1 的 Schema)
│   ├── ticket
│   ├── incident
│   └── ...
├── tenant_2 (租户 2 的 Schema)
│   ├── ticket
│   ├── incident
│   └── ...
└── ...
```

#### 3.1.2 租户中间件

```go
// middleware/tenant_middleware.go
func TenantMiddleware(client *ent.Client) gin.HandlerFunc {
    return func(ctx *gin.Context) {
        tenantID, _ := ctx.Get("tenant_id")
        // 设置租户上下文
        ctx.Set("tenant_id", tenantID)
        ctx.Next()
    }
}
```

### 3.2 工作流引擎

#### 3.2.1 BPMN 引擎集成

```
BPMN 文件 (XML)
    │
    ▼
BPMN 解析器
    │
    ▼
流程定义 (Process Definition)
    │
    ▼
流程实例 (Process Instance)
    │
    ├─▶ 用户任务 (User Task)
    ├─▶ 服务任务 (Service Task)
    ├─▶ 网关 (Gateway)
    └─▶ 事件 (Event)
```

#### 3.2.2 流程绑定机制

流程可以与业务实体绑定：
- 工单
- 变更
- 服务请求
- 审批

### 3.3 AI 服务架构

#### 3.3.1 LLM 网关

```
LLM Gateway
    │
    ├─▶ OpenAI Provider
    ├─▶ Anthropic Provider
    ├─▶ Azure OpenAI Provider
    └─▶ Custom Provider
```

#### 3.3.2 RAG 实现

```
用户查询
    │
    ▼
查询向量化 (Embedder)
    │
    ▼
向量相似度搜索 (Vector Store)
    │
    ▼
结果重排序 (Hybrid Search)
    │
    ▼
上下文注入 + LLM 生成
    │
    ▼
最终回答
```

### 3.4 连接器架构

#### 3.4.1 连接器接口

```go
type Connector interface {
    Manifest() Manifest
    Init(ctx context.Context, cfg Config) error
    Send(ctx context.Context, msg *Message) error
    HealthCheck(ctx context.Context) HealthStatus
    Close() error
}
```

#### 3.4.2 内置连接器

- **飞书 (Feishu)**: 消息、用户同步
- **企业微信 (WeCom)**: 消息、用户同步
- **钉钉 (DingTalk)**: 消息、用户同步
- **Webhook**: 通用 HTTP 回调
- **邮件 (Email)**: 邮件通知

---

## 4. 数据模型设计

### 4.1 核心实体关系

```
Tenant (租户)
    │
    ├─▶ User (用户)
    │       ├─▶ Role (角色)
    │       └─▶ Permission (权限)
    │
    ├─▶ Ticket (工单)
    │       ├─▶ TicketComment (工单评论)
    │       ├─▶ TicketAttachment (工单附件)
    │       ├─▶ TicketTag (工单标签)
    │       ├─▶ TicketAssignmentRule (分配规则)
    │       └─▶ ProcessInstance (流程实例)
    │
    ├─▶ Incident (事件)
    │       ├─▶ IncidentEvent (事件记录)
    │       ├─▶ IncidentAlert (事件告警)
    │       └─▶ KnownError (已知错误)
    │
    ├─▶ Problem (问题)
    │       ├─▶ RootCause (根因分析)
    │       └─▶ KnownError (已知错误)
    │
    ├─▶ Change (变更)
    │       ├─▶ CAB (变更咨询委员会)
    │       ├─▶ ChangePIR (实施后回顾)
    │       └─▶ Release (发布)
    │
    ├─▶ ServiceCatalog (服务目录)
    │       └─▶ ServiceRequest (服务请求)
    │
    ├─▶ KnowledgeArticle (知识文章)
    │       ├─▶ KnowledgeArticleVersion (版本)
    │       └─▶ KnowledgeArticleLike (点赞)
    │
    ├─▶ SLAPolicy (SLA 策略)
    │       └─▶ SLAViolation (SLA 违规)
    │
    ├─▶ CMDB (配置管理)
    │       ├─▶ ConfigurationItem (配置项)
    │       ├─▶ CIRelationship (配置项关系)
    │       └─▶ CIHistory (配置项历史)
    │
    ├─▶ Asset (资产)
    │       ├─▶ AssetLicense (许可证)
    │       └─▶ Vendor (供应商)
    │
    ├─▶ Workflow (工作流)
    │       ├─▶ ProcessDefinition (流程定义)
    │       └─▶ ProcessInstance (流程实例)
    │
    └─▶ Notification (通知)
            └─▶ NotificationPreference (通知偏好)
```

### 4.2 Ent Schema 设计原则

1. **租户隔离**: 所有业务表都包含 `tenant_id` 字段
2. **时间戳**: 所有表都包含 `created_at` 和 `updated_at`
3. **软删除**: 使用 `deleted_at` 支持软删除
4. **审计字段**: 包含 `created_by` 和 `updated_by`
5. **索引优化**: 为常用查询字段创建索引

---

## 5. API 设计规范

### 5.1 RESTful 设计

```
GET    /api/v1/tickets              # 获取列表
POST   /api/v1/tickets              # 创建
GET    /api/v1/tickets/{id}         # 获取详情
PUT    /api/v1/tickets/{id}         # 更新
DELETE /api/v1/tickets/{id}         # 删除
POST   /api/v1/tickets/{id}/assign  # 操作
```

### 5.2 统一响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 5.3 分页设计

```
GET /api/v1/tickets?page=1&pageSize=20&sortBy=createdAt&sortOrder=desc

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [],
    "total": 100,
    "page": 1,
    "pageSize": 20,
    "totalPages": 5
  }
}
```

---

## 6. 安全设计

### 6.1 认证机制

- **JWT Token**: 无状态认证
- **Refresh Token**: 支持 token 刷新
- **HttpOnly Cookie**: 安全存储 token

### 6.2 授权机制

- **RBAC**: 基于角色的访问控制
- **权限粒度**: 功能级 + 数据级
- **租户隔离**: 跨租户访问保护

### 6.3 安全中间件

```go
// middleware 栈
r.Use(
    middleware.CORSMiddleware(),
    middleware.SecurityHeadersMiddleware(),
    middleware.RateLimitMiddleware(),
    middleware.AuthMiddleware(),
    middleware.RBACMiddleware(),
    middleware.TenantMiddleware(),
)
```

---

## 7. 性能优化

### 7.1 缓存策略

| 缓存类型 | 缓存内容 | TTL |
|---------|---------|-----|
| **应用缓存** | 用户会话、配置 | 30 分钟 |
| **查询缓存** | 频繁访问的数据 | 5-15 分钟 |
| **CDN 缓存** | 静态资源 | 7 天 |

### 7.2 数据库优化

- **索引优化**: 为查询字段创建合适的索引
- **查询优化**: 避免 N+1 查询，使用预加载
- **分页查询**: 大结果集必须分页
- **读写分离**: 可选的主从复制

### 7.3 异步处理

- **异步通知**: 消息推送使用队列
- **后台任务**: SLA 监控、报表生成
- **事件驱动**: 领域事件解耦

---

## 8. 部署架构

### 8.1 Docker 部署

```yaml
# docker-compose.yml
version: '3.8'
services:
  postgres:
    image: postgres:15
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7
    volumes:
      - redis_data:/data

  backend:
    build: ./itsm-backend
    ports:
      - "8090:8090"
    depends_on:
      - postgres
      - redis

  frontend:
    build: ./itsm-frontend
    ports:
      - "3000:3000"
    depends_on:
      - backend
```

### 8.2 Kubernetes 部署（可选）

支持 Helm Chart 部署，包含：
- Deployment
- Service
- Ingress
- ConfigMap
- Secret
- PVC

---

## 9. 监控与运维

### 9.1 日志规范

使用结构化日志 (Zap):

```go
logger.Infow("Ticket created",
    "ticket_id", ticket.ID,
    "title", ticket.Title,
    "tenant_id", tenantID,
    "user_id", userID,
)
```

### 9.2 指标监控

Prometheus 指标：
- HTTP 请求数/延迟
- 业务指标（工单创建数、SLA 达成率）
- 系统指标（CPU、内存、磁盘）

### 9.3 健康检查

```
GET /health
GET /readiness
GET /liveness
```

---

## 10. 开发规范

### 10.1 后端开发规范

1. **错误处理**: 使用自定义错误类型
2. **事务管理**: Service 层控制事务边界
3. **测试**: 单元测试 + 集成测试
4. **文档**: API 文档使用注释生成

### 10.2 前端开发规范

1. **组件**: 函数组件 + Hooks
2. **状态管理**: Zustand 全局状态
3. **类型**: TypeScript 严格模式
4. **样式**: Tailwind CSS

---

## 11. 扩展性设计

### 11.1 插件系统

- **连接器插件**: 自定义连接器
- **技能插件**: AI 技能
- **流程插件**: 自定义流程任务

### 11.2 事件驱动

```go
// 事件总线
type EventBus interface {
    Publish(topic string, event interface{}) error
    Subscribe(topic string, handler EventHandler) error
}
```

---

## 12. 总结

本架构设计遵循以下原则：

1. **高内聚低耦合**: 模块化设计，职责清晰
2. **可扩展性**: 插件化架构，易于扩展
3. **性能优先**: 缓存、异步、优化查询
4. **安全可靠**: 认证、授权、审计
5. **可观测**: 日志、监控、链路追踪

该架构能够支撑企业级 ITSM 系统的长期发展。
