# ITSM 系统架构设计文档

## 1. 系统概览

### 1.1 项目定位

**AI-Native ITSM (IT Service Management) 平台** — 一个面向中国市场的企业级 IT 服务管理系统，对标 ServiceNow，提供完整的 ITIL v3/v4 工作流支持，并深度集成 AI 能力。

### 1.2 核心特性

| 特性 | 描述 |
|------|------|
| **ITIL 全流程** | 工单、事件、问题、变更、发布、服务请求、服务目录、知识库、SLA |
| **AI 原生** | 智能分诊、根因分析、风险预测、RAG 知识问答、AI 评估 |
| **BPMN 工作流** | 自研流程引擎，支持 XML 解析、网关、服务任务、版本管理 |
| **多租户 + MSP** | JWT 租户隔离 + RLS 行级安全 + MSP 服务提供商模式 |
| **可观测性** | Prometheus + Grafana + Alertmanager 完整监控栈 |

### 1.3 技术栈

| 层级 | 技术选型 |
|------|----------|
| 前端 | Next.js 15 + React 19 + TypeScript + Ant Design 6 + Tailwind CSS 4 |
| 后端 | Go 1.21 + Gin + Ent ORM + Zap |
| 数据库 | PostgreSQL 17 (pgvector) + Redis 7 + MinIO |
| AI 服务 | Python FastAPI + OpenAI/Ollama + ChromaDB + RAG |
| 工作流 | 自研 BPMN 引擎 (nitram509/lib-bpmn-engine) |
| 监控 | Prometheus + Grafana + Alertmanager |
| 部署 | Docker Compose + Nginx |

---

## 2. 架构分层

### 2.1 分层总览

```
┌─────────────────────────────────────────────────────────────┐
│                     客户端层                                 │
│   Web (Next.js) | CLI (Ink) | Webhook | 飞书/钉钉 | Agent   │
├─────────────────────────────────────────────────────────────┤
│                     网关层                                    │
│              Nginx 反向代理 + 安全头 + WebSocket              │
├─────────────────────────────────────────────────────────────┤
│                     后端服务层                                │
│  Handler 域 | 核心服务 | Middleware | 数据访问 | 后台任务   │
├─────────────────────────────────────────────────────────────┤
│                     数据层                                    │
│    PostgreSQL (pgvector) | Redis | MinIO | ChromaDB        │
├─────────────────────────────────────────────────────────────┤
│                     AI 服务层                                 │
│          FastAPI + LLM + Triage/RCA/Risk + RAG             │
├─────────────────────────────────────────────────────────────┤
│                   监控/可观测性层                             │
│            Prometheus + Grafana + Alertmanager              │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. 核心模块详解

### 3.1 前端 (itsm-frontend)

**技术架构**：
- Next.js 15 App Router + React 19
- 状态管理：Zustand (客户端) + TanStack React Query (服务端)
- UI 框架：Ant Design 6 + Tailwind CSS 4
- HTTP 客户端：自封装 fetch (非 Axios)

**目录结构**：
```
itsm-frontend/src/
├── app/                    # App Router 页面
│   ├── (auth)/            # 认证路由 (login, register)
│   ├── (main)/            # 主应用路由 (90+ 页面)
│   └── api/               # API 代理
├── components/            # 36 个业务域组件
│   ├── ticket/            # 工单
│   ├── incident/         # 事件
│   ├── cmdb/             # 配置管理
│   ├── workflow/         # 工作流
│   └── ...
├── lib/
│   ├── api/              # 76 个 API 客户端文件
│   ├── hooks/            # 32 个自定义 Hooks
│   ├── store/            # Zustand 状态存储
│   └── services/         # 业务服务层
└── types/                # TypeScript 类型定义
```

**核心设计决策**：
1. **HttpOnly Cookie 认证**：Token 不存储在前端，通过 HttpOnly Cookie 传递
2. **CSRF 双重提交**：CSRF token 注入 + 重试机制
3. **Snake/Camel 自动转换**：请求/响应自动转换命名风格
4. **4 级缓存策略**：STATIC / SEMI_STATIC / DYNAMIC / REAL_TIME

### 3.2 后端 (itsm-backend)

**Handler 域** (18 个业务域)：

| 域 | 职责 | 路由前缀 |
|----|------|----------|
| `ticket` | 工单 CRUD + 生命周期 | `/api/v1/tickets` |
| `incident` | 事件管理 + 升级 + 根因 | `/api/v1/incidents` |
| `problem` | 问题管理 + 调查 | `/api/v1/problems` |
| `change` | 变更管理 + CAB + PIR | `/api/v1/changes` |
| `cmdb` | 配置项 + 关系 + 拓扑 | `/api/v1/cmdb` |
| `knowledge` | 知识库 + RAG | `/api/v1/knowledge` |
| `sla` | SLA 定义 + 策略 + 告警 | `/api/v1/sla` |
| `ai` | AI 工具 + 摘要 + 评估 | `/api/v1/ai` |
| `service_catalog` | 服务目录 | `/api/v1/service-catalog` |
| `service_request` | 服务请求 | `/api/v1/service-requests` |
| `skill` | AI 技能注册表 | `/api/v1/skills` |
| `cab` | CAB 成员管理 | `/api/v1/cab` |
| `standard_change` | 标准变更模板 | `/api/v1/standard-changes` |
| `known_error` | 已知错误库 | `/api/v1/known-errors` |
| `release` | 发布管理 | `/api/v1/releases` |
| `asset` | 资产管理 | `/api/v1/assets` |
| `dashboard` | 仪表盘 | `/api/v1/dashboard` |
| `operations` | 运维命令 | `/api/v1/admin/operations` |

**核心服务**：

| 服务 | 文件 | 职责 |
|------|------|------|
| BPMN Engine | `service/bpmn_*.go` | 流程实例、网关、服务任务、版本管理 |
| Approval | `service/approval_*.go` | 审批链、审批人解析、求值引擎 |
| SLA Monitor | `service/sla_*.go` | SLA 监控、告警、违规检测 |
| LLM Gateway | `service/llm_gateway.go` | 统一 LLM 接口、Token 限流、可观测性 |
| RAG Service | `service/rag_service.go` | 向量检索 + 关键字降级 |
| Notification | `service/notification_*.go` | 邮件/短信/IM + Outbox 模式 |
| Ticket Core | `service/ticket_*.go` | 工单核心逻辑、分配、自动化规则 |

**中间件栈**：

| 中间件 | 文件 | 职责 |
|--------|------|------|
| Auth | `middleware/auth.go` | JWT 解析、Token 验证 |
| RBAC | `middleware/rbac.go` | 角色权限检查、内存缓存 |
| Tenant | `middleware/tenant.go` | 多租户隔离、来源优先级 |
| MSP | `middleware/msp_*.go` | MSP 上下文、权限映射 |
| RateLimit | `middleware/rate_limiter*.go` | 滑动窗口限流、Redis 分布式 |
| Audit | `middleware/audit.go` | 写操作审计日志 |
| CSRF | `middleware/csrf.go` | 双提交 Cookie 保护 |
| Security | `middleware/security.go` | 安全头、SQL 注入、XSS 防护 |

**Ent Schema** (95 个实体)：

```
工单域: ticket, ticket_approval, ticket_comment, ticket_attachment...
事件域: incident, incidentalert, incident_escalation_rule...
问题域: problem, known_error, root_cause_analysis
变更域: change, change_pir, standard_change
CMDB域: configurationitem, citype, ci_relationship...
SLA域: sladefinition, slapolicy, slaviolation...
BPMN域: process_definition, process_instance, process_task...
审批域: approval_record, approval_workflow
服务域: servicecatalog, servicerequest
知识域: knowledge_article, conversation, message
资产域: asset, asset_license, contract, vendor, release
权限域: permission, role_permission, endpoint_acl
云资源域: cloud_account, cloud_resource, discovery_job
```

### 3.3 数据层

**PostgreSQL 17 (pgvector)**：
- 业务主数据库，存储所有实体数据
- 向量扩展 pgvector，支持知识库语义检索
- RLS (Row-Level Security) 行级安全 (可选模式)
- 端口：5432 (dev) / 5433 (prod)

**Redis 7**：
- 会话缓存、限流计数器
- 工单编号序列生成器
- Token 撤销列表
- 端口：6379 (dev) / 6380 (prod)

**MinIO**：
- 对象存储，存放附件、图片等
- S3 兼容 API
- 端口：9000 (dev) / 内部 (prod)

**ChromaDB**：
- 向量数据库，RAG 语义检索
- 本地 PersistentClient 模式

**Watermill**：
- 事件总线，跨服务异步通信
- Outbox 模式保证事务一致性

### 3.4 AI 服务层 (itsm-ai-service)

**FastAPI Python 服务**，端口 8000：

| 端点 | 功能 |
|------|------|
| `POST /api/v1/triage/` | 智能工单分类 (分类/优先级/分配建议) |
| `POST /api/v1/triage/batch` | 批量分类 |
| `POST /api/v1/rca/analyze` | 根因分析 (5 Whys/鱼骨图/AI) |
| `POST /api/v1/risk/predict` | 变更风险预测 |
| `POST /api/v1/risk/batch` | 批量风险预测 |

**LLM 配置** (config.yaml)：
- OpenAI (gpt-4o-mini, 默认)
- Ollama (本地开发)
- vLLM / DeepSeek / 智谱 / 百度 / 阿里

### 3.5 RAG 服务 (itsm-rag)

**运维知识问答系统**：

```
Confluence API → JSON → 分块 → Embedding → ChromaDB → Rerank → LLM 生成
```

**技术栈**：
- Embedding: `shibing624/text2vec-base-chinese` (768维)
- Reranker: `BAAI/bge-reranker-base`
- LLM: `qwen2.5:7b` (Ollama)
- UI: Gradio (端口 7860)

### 3.6 监控栈 (monitoring/)

**Prometheus** (端口 9090)：
- 10s 抓取间隔
- 抓取目标：Backend + NodeExporter + Redis + PostgreSQL + Nginx

**Grafana** (端口 3001)：
- 预置仪表盘：ITSM 总览、业务指标、BPMN 监控

**Alertmanager** (端口 9093)：
- 业务告警：工单量异常、SLA 违规、变更失败率
- 系统告警：服务 down、CPU/内存/磁盘高负载

---

## 4. 数据流设计

### 4.1 请求处理流程

```
1. 客户端请求
      ↓
2. Nginx 反向代理 (安全头、WebSocket 升级)
      ↓
3. Go Backend (Gin Router)
      ↓
4. Global Middleware (CORS → RateLimit → Security → RequestID → Logger)
      ↓
5. Auth Middleware (JWT 解析 → Token 验证)
      ↓
6. RBAC Middleware (权限检查)
      ↓
7. Tenant Middleware (租户隔离)
      ↓
8. Handler (业务逻辑)
      ↓
9. Service (领域服务)
      ↓
10. Repository (Ent ORM)
      ↓
11. PostgreSQL / Redis / MinIO
```

### 4.2 异步处理流程 (Outbox 模式)

```
1. 业务事务提交
      ↓
2. Outbox 记录写入 (同一事务)
      ↓
3. EventBus (Watermill) 发布事件
      ↓
4. Command Handler 消费事件
      ↓
5. 外部操作 (邮件/短信/通知/IM)
```

### 4.3 BPMN 流程执行

```
1. 流程触发 (工单创建/状态变更)
      ↓
2. Process Engine 解析 XML
      ↓
3. 创建 Process Instance
      ↓
4. 执行节点 (UserTask/ServiceTask/Gateway)
      ↓
5. 审批服务集成 (审批链求值)
      ↓
6. 通知服务集成 (消息推送)
      ↓
7. 流程结束 → 审计日志
```

### 4.4 AI 能力调用

```
1. 前端调用 AI API
      ↓
2. Backend LLM Gateway (限流、路由)
      ↓
3. itsm-ai-service (FastAPI)
      ↓
4. LLM Provider (OpenAI/Ollama)
      ↓
5. 响应返回 + Telemetry 记录
```

---

## 5. 部署架构

### 5.1 开发环境 (docker-compose.dev.yml)

```
┌────────────────────────────────────────────────────────┐
│                   Docker Network (bridge)              │
├────────────────────────────────────────────────────────┤
│  itsm-frontend :3000  │  itsm-backend :8090           │
│  itsm-ai-service:8000│  postgres :5432                │
│  redis :6379         │  minio :9000                  │
│  ollama :11434       │  prometheus :9090             │
│  grafana :3001       │                                │
└────────────────────────────────────────────────────────┘
```

### 5.2 生产环境 (docker-compose.prod.yml)

```
┌────────────────────────────────────────────────────────┐
│  Nginx (80/443)  ← letsencrypt 证书                   │
├────────────────────────────────────────────────────────┤
│  itsm-frontend :3000  │  itsm-backend :8090           │
│  itsm-worker (无 HTTP) │  itsm-init (迁移)             │
├────────────────────────────────────────────────────────┤
│  PostgreSQL :5433 (localhost) │  Redis :6380           │
│  MinIO (内部)               │                         │
├────────────────────────────────────────────────────────┤
│  itsm-ai-service :8000  │  监控栈 (非 Docker)         │
└────────────────────────────────────────────────────────┘
```

**生产特性**：
- RLS 模式 enforce
- CORS 白名单
- DB_PASSWORD / JWT_SECRET 强制环境变量
- GIN_MODE=release
- JSON 日志格式
- itsm-worker 独立进程处理后台任务

---

## 6. 安全架构

### 6.1 认证与授权

| 层级 | 技术 | 描述 |
|------|------|------|
| 认证 | JWT | Access Token + Refresh Token 双令牌 |
| 授权 | RBAC | 角色-资源-动作权限模型 |
| 租户 | Tenant | JWT > X-Tenant-Code > Subdomain 优先级 |
| MSP | MSP Context | 客户租户锁定 + 权限映射 |

### 6.2 安全防护

| 防护项 | 实现 |
|--------|------|
| SQL 注入 | 中间件过滤 |
| XSS | DOMPurify + 服务端消毒 |
| CSRF | 双提交 Cookie 模式 |
| 限流 | 滑动窗口 (500/min) |
| 安全头 | Nginx + Middleware 双层 |
| 敏感脱敏 | 日志自动遮蔽 |

---

## 7. 扩展性设计

### 7.1 Handler 扩展

新增业务域只需：
1. 创建 `handlers/<domain>/` 目录
2. 实现 `handler.go` + `service.go` + `repository.go`
3. 在 `router.go` 注册路由

### 7.2 BPMN 扩展

新增服务任务：
1. 在 `service/bpmn/` 创建适配器
2. 实现 `Execute()` 接口
3. 在流程 XML 中引用

### 7.3 连接器扩展

支持飞书/钉钉/企微/Webhook：
1. 实现 `Connector` 接口
2. 注册到 `ConnectorManager`
3. 配置 webhook 回调

---

## 8. 附录

### 8.1 端口映射

| 服务 | 开发端口 | 生产端口 |
|------|----------|----------|
| Frontend | 3000 | 3000 |
| Backend | 8090 | 8090 |
| PostgreSQL | 5432 | 5433 |
| Redis | 6379 | 6380 |
| MinIO | 9000/9001 | 内部 |
| AI Service | 8000 | 8000 |
| RAG (Gradio) | 7860 | - |
| Prometheus | 9090 | 9090 |
| Grafana | 3001 | 3001 |

### 8.2 环境变量

| 变量 | 描述 | 示例 |
|------|------|------|
| `DB_PASSWORD` | 数据库密码 | (必填) |
| `JWT_SECRET` | JWT 密钥 | (必填) |
| `REDIS_PASSWORD` | Redis 密码 | (生产必填) |
| `ITSM_BACKEND_URL` | 后端地址 | http://localhost:8090 |
| `LLM_PROVIDER` | LLM 提供商 | openai |
| `DEPLOYMENT_MODE` | 部署模式 | private/msp |

### 8.3 相关文档

- [API 文档](./API.md)
- [数据库 Schema](./DATABASE.md)
- [部署指南](./DEPLOYMENT.md)
- [BPMN 流程设计](./BPMN.md)
- [AI 能力说明](./AI_FEATURES.md)
