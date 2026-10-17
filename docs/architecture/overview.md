# Architecture Overview

> Status: current

> **状态**：当前
> **更新日期**：2026-10-17

## 系统视图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              Clients                                    │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────────┐        │
│  │  Web (Next.js)  │  │  Mobile (PWA)   │  │  itsm-cli (CLI)  │        │
│  └────────┬────────┘  └────────┬────────┘  └────────┬─────────┘        │
│           │ HTTPS / WSS         │                    │                  │
└───────────┼──────────────────────┼────────────────────┼──────────────────┘
            │                      │                    │
            ▼                      ▼                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        Nginx (Reverse Proxy)                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                   │
│  │   /api/v1/*  │  │   /ws/*      │  │   /metrics   │                   │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘                   │
└─────────┼──────────────────┼──────────────────┼──────────────────────────┘
          │                  │                  │
          ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                  itsm-backend (Go 1.25 + Gin)                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  middleware/                                                     │   │
│  │    ├─ CORS / Security Headers / SQL Injection / XSS Protection  │   │
│  │    ├─ Rate Limiter (Redis / in-memory fallback)                 │   │
│  │    ├─ JWT Auth                                                   │   │
│  │    ├─ RBAC (role-based) + ABAC (expr-lang)                      │   │
│  │    ├─ Tenant Isolation                                           │   │
│  │    └─ MSP (multi-tenant service provider)                        │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  handlers/<domain>/  (HTTP + 领域规则,返回 DTO)                │   │
│  │  ├─ ticket / incident / problem / change / release              │   │
│  │  ├─ cmdb / sla / knowledge / approval / catalog                │   │
│  │  ├─ bpmn / workflow / connector / ai / analytics                │   │
│  │  ├─ auth / user / role / permission / menu / tenant            │   │
│  │  └─ skill / asset / standard_change / known_error               │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  service/  (跨域共享业务逻辑,按域下沉)                          │   │
│  │    ├─ ticket_service / incident_service / change_service        │   │
│  │    ├─ cmdb_service / sla_service / knowledge_service           │   │
│  │    ├─ bpmn_* (engine / process / trigger / monitoring)          │   │
│  │    ├─ llm_gateway / rag_service / triage_service                │   │
│  │    ├─ escalation_service / sla_monitor_service                 │   │
│  │    └─ asset / vendor / survey / global_search / etc.            │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  cache/  (Redis 客户端封装)                                    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  ent/  (ORM — schema → codegen → migrate)                       │   │
│  │      135 个 ent/schema 文件: ticket / incident / ...            │   │
│  │    / user / tenant / role / sla / bpmn / knowledge / etc.       │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────┬───────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                  Storage & External Services                            │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌─────────────────┐    │
│  │ PostgreSQL │  │   Redis    │  │  pgvector   │  │  LLM Providers  │    │
│  │  (主存储)   │  │ (缓存/队列) │  │ (向量检索)  │  │  DeepSeek/OAI   │    │
│  └────────────┘  └────────────┘  └────────────┘  └─────────────────┘    │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────────────────────┐  │
│  │ Prometheus │  │  Grafana   │  │  Connectors (飞书/Slack/钉钉/Teams) │  │
│  └────────────┘  └────────────┘  └────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

## 模块划分

### 后端 (itsm-backend)

- **`handlers/<domain>/`**：HTTP 入口 + 领域规则（`handler.go` / `service.go` / `repository.go` / `entity.go`），只调用本域或公开 service，**不直接访问 Ent**。返回 DTO，**禁止返回 Ent 模型**
- **`service/`**：跨领域共享业务逻辑；单一领域规则应下沉到 `handlers/<domain>/service.go`
- **`internal/commandbus/` + `operational_commands`**：可靠异步副作用（事务型 outbox + Worker 租约认领）；通知、审批、连接器、BPMN 均经此路径
- **`ent/schema/`**：数据模型，**改 schema 必须 `go generate`**
- **`middleware/`**：横切关注点（auth / rbac / tenant / logging / metrics）
- **`dto/`**：请求/响应结构，使用 **camelCase 字段名**
- **`cache/`**：Redis 客户端封装（缓存与限流）
- **`router/`**：路由注册，装配各 `handlers/<domain>/` 的 handler（横向 `controller/` 层已于 `48eae74a` 退役，不再新增）

### 前端 (itsm-frontend)

- **`src/app/(main)/`**：受保护页面（需登录）
- **`src/lib/api/`**：API 客户端类（按域分文件：`ticket-api.ts`、`incident-api.ts`）
- **`src/lib/store/`**：Zustand 状态
- **`src/components/`**：按业务域组织的组件
- **`src/lib/hooks/`**：业务 hooks（`use*.ts`）

### 工具链

- **`itsm-cli`** (Node.js)：命令行工具，供运维和开发使用
- **`itsm-agent`** (Go)：AI 开发 Agent（轮询 GitHub issue → 生成代码 → 开 PR），非运行时组件
- **`itsm-ai-service`** (Python)：LLM 服务（triage / RCA / risk）+ Guidance sidecar，可独立部署
- **`itsm-skill`**：MCP 技能市场插件（本地扩展目录，`.gitignore` 中，未纳入版本库）

## 数据流：典型工单生命周期

```
1. 客户提交工单
   ├─ Web Form → POST /api/v1/tickets
   └─ 邮件 (Connector) → POST /api/v1/feishu/webhook/:instance_id

2. Handler 接收
   ├─ middleware: Auth → RBAC → Tenant
   └─ handlers/ticket Handler.CreateTicket → dto 校验

3. 领域处理
   ├─ handlers/ticket Service.Create（委托 service/ticket_service.go）
   │   ├─ Ent: INSERT INTO tickets ...
   │   ├─ 写入 operational_commands（同一事务）→ Worker 认领执行
   │   └─ SLA 到期时间按权威时间戳落库
   │
   └─ 后台 Worker / 定时任务
       ├─ 通知投递 → 邮件 / 飞书 / 钉钉 / 企微
       ├─ SLAMonitorService → 扫描到期时间并建违规记录
       ├─ KnowledgeService / RAG → 推荐相关文章
       └─ AnalyticsService → 实时统计

4. 工程师处理
   ├─ GET /api/v1/tickets/:id
   ├─ PUT /api/v1/tickets/:id/status  (状态流转)
   ├─ POST /api/v1/tickets/:id/comments
   └─ POST /api/v1/tickets/:id/resolve

5. 关闭 + 复盘
   ├─ POST /api/v1/tickets/:id/close
   ├─ 触发 SLA 合规检查 → 写入 sla_violations 表
   └─ 审计记录留痕（写操作可追踪）
```

## 关键设计决策

| 决策 | 原因 | 备选 |
|:---|:---|:---|
| **Ent ORM** | Schema 优先 + 类型安全 + 自动迁移 | GORM / sqlc / raw SQL |
| **域切片 `handlers/<domain>/`** | 域规则与 HTTP 入口同域收口，避免横向层漂移 | `controller/` + `service/` 横向分层 |
| **事务型 Outbox** | 业务事务内落库 `operational_commands` + Worker 租约认领，副作用不丢不重 | Watermill pub/sub / 直接 Redis / Kafka |
| **expr-lang** | ABAC 表达式引擎，支持运行时编译 | Casbin / OPA |
| **BPMN 引擎外置** | 复用成熟 lib（nitram509），避免重写 | 自研引擎 |
| **DTO camelCase** | 与前端 TypeScript 类型对齐 | Ent 原生 snake_case |
| **WebSocket 票据** | 避免 JWT 在 URL 中泄露 | 长连接 + refresh token |
| **多包结构** | 子项目可独立 CI / 独立部署 | monorepo 单一包 |

详见：

- [领域归属与租户边界](domain-ownership.md)
- [BPMN 审批运行时契约](approval-runtime-contract.md)
- [系统架构设计](architecture.md)
