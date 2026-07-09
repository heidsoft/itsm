# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## Project Overview

ITSM (IT Service Management) system with a Go/Gin backend and Next.js/TypeScript frontend. Features include:

- Ticket/Incident/Problem/Change management
- Service Catalog
- Knowledge Base with RAG
- BPMN Workflow engine
- SLA monitoring and escalation
- AI-powered triage and summarization

## Product Direction

This project is building an enterprise-grade, open-source, AI-Native ITSM platform for the China market. The long-term benchmark is ServiceNow-class process capability, but with lighter private deployment, stronger local enterprise integration, and open extensibility.

Core product goals:

- Cover complete ITIL v3/v4 service management workflows: ticket, incident, problem, change, release, service request, service catalog, SLA, knowledge, and CMDB.
- Make workflow customization a first-class capability through BPMN, process binding, form/config templates, and auditable task execution.
- Build AI into the service management lifecycle instead of adding a chatbot beside it: triage, summarization, knowledge retrieval, impact analysis, workflow recommendation, audit review, and controlled tool invocation.
- Prepare for Feishu, WeCom, DingTalk, Webhook, connector marketplace, skill marketplace, plugin marketplace, and CLI-driven operations.
- Support private deployment, SaaS, and SaaS + MSP modes without forking the core data model.

When making architecture choices, prefer enterprise correctness, auditability, tenant isolation, and extensibility over quick feature-only shortcuts.

## Current Product Stage

The repository is past v1.0 GA foundation work and is moving through v1.1 hardening:

- v1.0 delivered ITIL core flows, BPMN workflow engine, CMDB v1, knowledge/RAG scaffold, SLA, RBAC, multi-tenant/MSP foundations, Docker Compose, GHCR images, and basic AI/connector scaffolding.
- v1.1 focus is coverage backfill, controller splitting, connector marketplace v1, RBAC hardening, AI audit console, and integration test coverage.
- v1.5+ focus is measurable AI evaluator, Feishu/DingTalk/WeCom production connectors, Skill registry, performance budgets, and stronger security scans.

For new work, align with the roadmap rather than creating parallel mechanisms. If a feature overlaps with workflow, connector, AI skill, or marketplace direction, extend the existing extension point.

## Development Commands

### Frontend (itsm-frontend)

```bash
cd itsm-frontend
npm install              # Install dependencies
npm run dev              # Start dev server (http://localhost:3000)
npm run build            # Production build
npm run lint             # Lint with auto-fix
npm run lint:check       # Lint check only
npm run type-check       # TypeScript type check
npm test                 # Run all tests
npm run test:unit        # Unit tests only
npm run test:integration # Integration tests only
npm run test:e2e         # E2E tests
```

### Backend (itsm-backend)

```bash
cd itsm-backend
go run main.go           # Start server (http://localhost:8090)
go build -o itsm-backend main.go # Binary build
./itsm-backend           # Run binary
go test ./...            # Run all tests
# Database migrations (use build tags)
go run -tags migrate main.go
go run -tags create_user main.go
```

### Environment Setup

```bash
# Configure .env file in itsm-backend/
LOG_LEVEL=info
DB_PASSWORD=your_password
JWT_SECRET=your-jwt-secret
ADMIN_PASSWORD=admin123
```

## Architecture

### System Boundaries

- **itsm-backend** is the source of truth for domain rules, permissions, workflow execution, tenant isolation, audit logs, and integration contracts.
- **itsm-frontend** is the operator/admin/user experience layer. It should not duplicate business rules that belong in backend services.
- **itsm-ai-service / guidance_sidecar / RAG services** provide AI assistance and retrieval. They must remain observable and fail safely.
- **itsm-agent / itsm-skill / itsm-cli** are future-facing extension surfaces. Do not hard-code behavior into the core app if it clearly belongs in a connector, skill, or CLI operation.
- **connector marketplace** integrations must go through lifecycle, health check, config, permission, and audit boundaries instead of ad hoc HTTP calls from controllers.

### Backend Structure

- **controller/** - HTTP handlers, receive requests, call services
- **service/** - Business logic, orchestrate operations
- **ent/schema/** - Database schema definitions (Ent ORM)
- **middleware/** - Auth, logging, CORS, tenant isolation
- **dto/** - Request/response DTOs
- **cache/** - Redis integration
- **router/** - Route registration

### Frontend Structure

- **src/app/** - Next.js App Router pages and layouts
- **src/app/(main)/** - Protected page routes
- **src/app/lib/** - API clients, utilities, stores
- **src/app/components/** - Reusable UI components
- **src/app/hooks/** - Custom React hooks

### API Response Format

All APIs return `{ code: number, message: string, data: any }`:

- `code: 0` = success
- `code: 1001+` = param errors
- `code: 2001` = auth failed
- `code: 5001` = internal error

### Key Services

- **BPMN Workflow**: `service/bpmn_*`, uses `nitram509/lib-bpmn-engine`
- **RAG/Knowledge**: `service/rag_service.go`, `service/vector_store.go`
- **AI Features**: `service/llm_gateway.go`, `service/triage_service.go`
- **SLA**: `service/sla_monitor_service.go`, `service/escalation_service.go`

### Domain Ownership

- Ticket, incident, problem, change, release, and service request are separate business domains. Reuse shared helpers, but do not collapse their lifecycle rules into one generic ticket abstraction unless the existing code already does so.
- BPMN/process execution is the orchestration layer. Approval chains, service catalog fulfillment, SLA escalation, and AI suggestions should integrate through workflow/process records where possible.
- CMDB is not an asset table. Preserve CI type, CI instance, relationship, topology, discovery source, impact analysis, and reconciliation concepts.
- Knowledge/RAG features must keep source attribution, versioning, and permission filtering. Do not return knowledge content across tenant or permission boundaries.
- MSP and multi-tenant behavior must be considered for every new table, query, API, menu item, and background job.

## Important Patterns

### Backend

- Use `common.Success(c, data)` / `common.Fail(c, code, msg)` for responses
- Use `zap.S()` for logging, not `fmt.Println()`
- Controllers call services, never access DB directly
- Ent schemas in `ent/schema/*.go` generate CRUD
- Keep controller methods thin: bind/validate request, call service, map DTO, return response.
- Put transaction boundaries in service/domain layers, not in frontend or controller glue.
- New domain tables must include tenant/account ownership fields where applicable and must be protected by tenant-aware queries.
- Prefer idempotent seed/migration/init logic. Default initialization must create product templates/configuration, not fake customer business data.
- Any high-risk action triggered by AI, connector, workflow automation, or bulk operation must create an audit record.

### Frontend

- Use App Router (no Pages Router)
- API calls via `src/app/lib/api/*.ts` classes
- Global state with Zustand in `src/app/lib/stores/`
- Tailwind CSS for all styling
- Treat backend API DTOs as the contract. Do not patch around backend field bugs in UI code without also fixing the DTO/mapper.
- Keep operational screens dense and scannable: tables, filters, status, owners, timestamps, and actions should be easy to compare.
- Prefer feature-local components/hooks under the route module for domain-specific UI; use shared components only when reuse is real.
- User-facing enterprise workflows must show loading, empty, error, permission-denied, and success states.

## AI-Native Engineering Rules

- AI suggestions are decision support by default, not silent authority. For triage, impact analysis, workflow recommendation, and automation, keep confidence, prompt/template version, model/provider, accepted/rejected status, and operator feedback.
- LLM calls must go through the LLM gateway or existing AI service abstraction. Do not call model providers directly from random controllers or frontend components.
- Always provide deterministic fallback behavior for AI failure, timeout, low confidence, or disabled provider.
- Prompts and skills should be versioned, testable, and auditable. Avoid burying large prompts in controller code.
- RAG must respect tenant, RBAC, knowledge visibility, and article version state before retrieval and before final response.
- Tool invocation by AI must pass explicit permission checks and produce audit logs.

## Workflow And ITIL Rules

- Workflow changes must preserve process definition, process instance, task, variable, execution history, and audit log integrity.
- Do not introduce a second approval engine when BPMN/process binding can represent the behavior.
- ITIL lifecycle transitions must be explicit and validated in services. Avoid frontend-only status transitions.
- SLA timers and escalations must be computed from authoritative timestamps and policy bindings, not from UI-derived state.
- Change management must preserve risk, CAB/approval, release window, implementation result, and PIR concepts.
- Problem management must preserve root cause, workaround, known error, and linked incident relationships.

## CMDB Rules

- CI type/schema, CI instance, CI relationship, relationship type, discovery source, import/export task, saved view, topology, and impact analysis are separate concerns.
- Every CI mutation should preserve history or auditability where the existing model supports it.
- Discovery/import/reconciliation must be idempotent and source-aware. Avoid creating duplicate CIs from repeated discovery runs.
- Relationship and topology APIs should enforce tenant isolation and permission checks.
- Impact analysis should traverse CI relationships deliberately and protect against unbounded recursion.

## Connector, Skill, And Plugin Rules

- New enterprise integrations should be implemented as connectors with lifecycle states: installed, configured, enabled, health-checked, disabled/uninstalled where supported.
- Connector secrets must never be returned to the frontend. Return masked metadata and health status only.
- Feishu, WeCom, DingTalk, and Webhook integrations should share marketplace/lifecycle abstractions instead of one-off endpoint designs.
- Skills should be declarative where possible: manifest, inputs, outputs, permissions, audit behavior, and evaluation hooks.
- Plugins must not bypass RBAC, tenant isolation, audit logging, or API response contracts.
- CLI operations should call stable backend APIs or documented service commands. Do not let CLI logic become a hidden business-rule fork.

## Security And Compliance

- Never commit real secrets, tokens, customer data, production database dumps, or unmasked connector credentials.
- Authentication, RBAC, menu permissions, endpoint ACL, and tenant filters must be considered together. Hiding a menu is not authorization.
- Cross-tenant access must fail closed. Add tests when touching tenant-scoped queries.
- Logs must not expose passwords, JWTs, API keys, connector secrets, prompt secrets, or private ticket content unless explicitly designed as protected audit content.
- File upload, import, connector callback, webhook, and AI tool invocation endpoints are high-risk surfaces and need validation, size limits, and audit logs.

## Configuration

- Frontend: `NEXT_PUBLIC_API_URL` env var (default: <http://localhost:8090>)
- Backend: `config.yaml` or environment variables
- Backend runs on port 8090, frontend on 3000

## Build Tags for Backend

- `migrate` - Run database migrations
- `create_user` - Create test user
- Default (no tag) - Run normal server

## Testing

- Backend: Table-driven tests with `stretchr/testify`, use `enttest.NewClient()` for DB
- Frontend: Jest + React Testing Library, mock API calls

### Verification Expectations

- For backend service/controller changes, run the narrow package tests first, then `cd itsm-backend && go test ./...` when practical.
- For frontend type/API/UI changes, run `cd itsm-frontend && npm run type-check` and the relevant Jest or Playwright test.
- For contract changes, update backend DTOs/mappers, frontend API clients/types, and tests in the same change.
- For workflow, CMDB, connector, AI, auth, tenant, or deployment changes, include at least one regression test or a documented manual verification path.
- For production deployment changes, validate with explicit env file usage and health checks.

## API 响应规范

### Controller必须返回DTO，禁止直接返回Ent模型

- ✅ `common.Success(c, dto.ToTicketResponse(ticket))`
- ❌ `common.Success(c, ticket)` // ticket是*ent.Ticket

### 使用已有的Mapper函数

- `dto.ToTicketResponse()` / `ToTicketResponseList()`
- `dto.ToIncidentResponse()` / `ToIncidentResponseList()`
- `dto.ToUserDetailResponse()` / `ToUserDetailResponseList()`
- `dto.ToTenantResponse()` / `ToTenantResponseList()`

### List响应必须通过Service层转换

- Service层返回 `*TicketListResponse` 而非 `[]*ent.Ticket`
- Controller层使用Mapper包装单个对象

### 字段命名规范

- 后端Ent模型使用snake_case（assignee_id, ticket_number）
- 前端期望camelCase（assigneeId, ticketNumber）
- DTO响应必须使用camelCase字段名
- Mapper函数负责转换snake_case到camelCase

## Docker 部署规范

### 生产环境启动（必须显式传入env-file）

```bash
# 正确：显式传入 --env-file
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d

# 错误：缺少环境变量文件
docker-compose -f docker-compose.prod.yml up -d
```

### 常见问题排查

```bash
# 检查容器状态
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# 检查容器日志
docker logs <container> --tail 30

# 检查容器网络
docker inspect <container> --format '{{json .NetworkSettings.Networks}}' | jq -r 'keys[]'

# 从容器内测试API
docker exec <container> wget -qO- http://localhost:8090/api/v1/health
```

### 网络隔离问题

生产容器与开发容器可能运行在不同网络：

- `itsm_itsm-network` - 开发网络
- `itsm_itsm-prod-network` - 生产网络
如遇DNS解析失败，先检查容器所在网络。

## TypeScript 开发规范

### Ant Design 组件导入

```tsx
// ❌ 错误：使用点链式访问 CompoundedComponent
Form.Input
Form.TextArea
Form.Select

// ✅ 正确：使用命名导入
import { Input, TextArea, Select } from 'antd';
```

### 组件迁移（Ant Design 升级）

| 旧API | 新API |
|-------|-------|
| `Space direction="vertical"` | `Space orientation="vertical"` |
| `Tabs.TabPane` | `items` 属性数组 |
| `Form.useForm()` | `const [form] = Form.useForm()` |

### 字段命名一致性

```typescript
// ❌ 错误：与后端API不匹配
threshold_percent
notify_owners

// ✅ 正确：与API响应类型一致
thresholdPercentage
notifyOwners
```

### 类型检查

```bash
cd itsm-frontend && npm run type-check
```

## 数据结构约定

### 前后端字段命名规范

| 层级 | 命名风格 | 示例 |
|------|---------|------|
| 后端 Ent Schema | snake_case | `assignee_id`, `ticket_number`, `created_at` |
| 后端 DTO 响应 | camelCase | `assigneeId`, `ticketNumber`, `createdAt` |
| 前端 TypeScript | camelCase | `assigneeId`, `ticketNumber`, `createdAt` |
| 数据库字段 | snake_case | `assignee_id`, `ticket_number` |

### 核心规则

1. **后端 → 前端**：DTO 必须使用 camelCase 响应
2. **前端 → 后端**：API 请求 payload 使用 camelCase
3. **数据库**：Ent Schema 使用 snake_case
4. **Mapper 转换**：DTO 层负责 snake_case → camelCase 转换

### API 字段契约（单一事实来源）

- 所有 HTTP/JSON 交互字段统一使用 `camelCase`，包括请求体、响应体、查询参数约定、前端类型定义
- `snake_case` 仅允许出现在 Ent Schema、数据库字段、SQL、内部持久化模型中
- Controller 不直接暴露 Ent 模型；进入接口边界前必须通过 DTO/Mapper 完成字段转换
- 新增接口时，先定义前端期望的 `camelCase` DTO，再在后端 Mapper 中完成 `snake_case` 到 `camelCase` 的映射
- 如历史接口仍返回 `snake_case`，应视为待修复兼容问题，不应继续沿用

### 请求体与响应体示例

```json
{
  "serviceCatalogId": "svc_123",
  "assigneeId": "user_001",
  "notifyOwners": true,
  "thresholdPercentage": 85
}
```

对应数据库/Schema 字段可为：`service_catalog_id`、`assignee_id`、`notify_owners`、`threshold_percentage`。

### 响应类型驼峰约定

所有 API 响应类型必须使用驼峰命名：

```typescript
// ✅ 正确：使用驼峰命名的响应类型
interface TicketResponse {
  id: string;
  ticketNumber: string;
  title: string;
  assigneeId: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}

// ❌ 错误：使用蛇形命名
interface TicketResponse {
  id: string;
  ticket_number: string;  // 不符合规范
  assignee_id: string;   // 不符合规范
}
```

### DTO Mapper 实现规范

```go
// ✅ 正确：在 DTO 层完成转换
func ToTicketResponse(ticket *ent.Ticket) *TicketResponse {
    return &TicketResponse{
        ID:           ticket.ID.String(),
        TicketNumber: ticket.TicketNumber,
        Title:        ticket.Title,
        AssigneeID:   ticket.AssigneeID,      // Ent 字段 snake_case
        Status:       ticket.Status,            // Ent 字段 snake_case
        CreatedAt:    ticket.CreatedAt.Format(), // 转换为字符串
        UpdatedAt:    ticket.UpdatedAt.Format(),
    }
}

// ❌ 错误：直接返回 Ent 模型
func GetTicket(c echo.Context) error {
    ticket, _ := svc.GetTicket(...)
    return c.JSON(200, ticket) // 泄漏 Ent 模型
}
```

## 文件命名规范

### 命名边界原则

- 接口字段命名与文件命名是两套规则：JSON/API 字段使用 `camelCase`，文件名按语言与框架约定执行
- 不要因为接口字段是 `camelCase`，就把 Go 后端文件命名成 `ticketService.go`
- 不要因为数据库字段是 `snake_case`，就把前端 TypeScript 文件命名成 `ticket_service.ts`
- 优先保持“看文件名就知道它属于哪一层、承担什么职责”

### 后端 (Go)

| 类型 | 命名风格 | 示例 |
|------|---------|------|
| Controller | `*_controller.go` | `ticket_controller.go` |
| Service | `*_service.go` | `ticket_service.go` |
| DTO | `*_dto.go` | `ticket_dto.go` |
| Schema | `*.go` (ent) | `ticket.go` |
| Middleware | `*_middleware.go` | `auth_middleware.go` |
| Repository | `*_repository.go` / `repository_impl.go` | `ticket_repository.go` |
| Router | `*_router.go` / `router.go` | `ticket_router.go` |

后端补充规则：

- Go 文件统一使用 `snake_case`
- 同一资源尽量使用统一前缀：`ticket_controller.go`、`ticket_service.go`、`ticket_dto.go`
- 避免无语义后缀：`ticket_handler_new.go`、`ticket_final.go`、`ticket_temp.go`
- 若文件承载的是实现细分，可使用职责后缀：`ticket_query_service.go`、`ticket_mapper.go`

### 前端 (TypeScript/Next.js)

| 类型 | 命名风格 | 示例 |
|------|---------|------|
| 页面 | `page.tsx` | `tickets/page.tsx` |
| 组件 | `*.tsx` (PascalCase) | `TicketList.tsx` |
| 工具函数 | `*.ts` (camelCase) | `formatDate.ts` |
| API 客户端 | `*Api.ts` | `TicketApi.ts` |
| 类型定义 | `*.ts` | `types/ticket.ts` |
| Hooks | `use*.ts` | `useTicket.ts` |
| Store | `*Store.ts` / `use*Store.ts` | `ticketStore.ts` |

前端补充规则：

- React 组件文件使用 `PascalCase`
- Hook、工具函数、store、API client 文件使用 `camelCase`
- Next.js 路由目录使用 `kebab-case`，动态路由使用 `[id]` 形式
- 同一模块的页面、组件、API、类型命名尽量围绕同一个业务词根，例如 `ticket` / `Ticket`
- 避免混用风格：不要同时出现 `TicketList.tsx`、`ticket-list.tsx`、`ticket_list.tsx`

### 目录结构命名

```
src/
├── app/                    # Next.js App Router
│   ├── (main)/            # 路由组 (括号命名)
│   ├── tickets/           # 页面目录 (kebab-case)
│   │   ├── page.tsx
│   │   └── [id]/
│   │       └── page.tsx
│   └── components/        # 组件目录
│       ├── TicketList/    # 组件文件夹 (PascalCase)
│       │   └── index.tsx
│       └── ui/            # 通用组件
└── lib/
    ├── api/               # API 客户端
    ├── stores/            # Zustand stores
    └── utils/             # 工具函数
```

### 禁止的命名方式

- ❌ `TicketListComponent.tsx` → ✅ `TicketList.tsx`
- ❌ `ticket_service.ts` → ✅ `ticketService.ts`
- ❌ `get_tickets.go` → ✅ `ticket_service.go`
- ❌ `APIUtils.ts` → ✅ `apiUtils.ts`
- ❌ `ticketService.go` → ✅ `ticket_service.go`
- ❌ `ticket_list.tsx` → ✅ `TicketList.tsx`
- ❌ `ticket-list-api.ts` → ✅ `ticketApi.ts` 或 `TicketApi.ts`

## 工程要求摘要

### 必须遵守的规则

1. **DTO 返回**：Controller 必须返回 DTO，禁止返回 Ent 模型
2. **字段命名**：后端 DTO 使用 camelCase，前端使用 camelCase
3. **文件命名**：遵循上述命名规范（Go 用 snake_case，TypeScript 用 PascalCase/camelCase）
4. **Mapper 转换**：DTO 层负责 snake_case → camelCase 转换
5. **命名边界**：接口字段、数据库字段、文件名各自遵循本层规则，禁止相互污染
6. **租户隔离**：新增查询、后台任务、导入导出、AI/RAG、连接器回调必须考虑 tenant/MSP 边界
7. **审计优先**：AI 建议、流程流转、审批、连接器动作、批量操作、高风险变更必须可追踪
8. **扩展优先**：流程、连接器、Skill、插件、CLI 方向已有扩展点时，优先扩展现有机制
9. **测试闭环**：改业务规则必须配套服务层/接口层/前端契约验证，不能只靠手工点击
10. **安全默认**：权限、密钥、日志、上传、Webhook、AI 工具调用默认按企业级高风险面处理
