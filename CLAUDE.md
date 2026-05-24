# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ITSM (IT Service Management) system with a Go/Gin backend and Next.js/TypeScript frontend. Features include:

- Ticket/Incident/Problem/Change management
- Service Catalog
- Knowledge Base with RAG
- BPMN Workflow engine
- SLA monitoring and escalation
- AI-powered triage and summarization

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

## Important Patterns

### Backend

- Use `common.Success(c, data)` / `common.Fail(c, code, msg)` for responses
- Use `zap.S()` for logging, not `fmt.Println()`
- Controllers call services, never access DB directly
- Ent schemas in `ent/schema/*.go` generate CRUD

### Frontend

- Use App Router (no Pages Router)
- API calls via `src/app/lib/api/*.ts` classes
- Global state with Zustand in `src/app/lib/stores/`
- Tailwind CSS for all styling

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
threshold_percentage
```

### 类型检查

```bash
cd itsm-frontend && npm run type-check
```
