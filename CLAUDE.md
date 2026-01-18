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
go run main.go           # Start server (http://localhost:8080)
go build -o itsm main.go # Binary build
go test ./...            # Run all tests
# Database migrations (use build tags)
go run -tags migrate main.go
go run -tags create_user main.go
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

- Frontend: `NEXT_PUBLIC_API_URL` env var (default: http://localhost:8080)
- Backend: `config.yaml` or environment variables
- Backend runs on port 8080, frontend on 3000

## Build Tags for Backend
- `migrate` - Run database migrations
- `create_user` - Create test user
- Default (no tag) - Run normal server

## Testing
- Backend: Table-driven tests with `stretchr/testify`, use `enttest.NewClient()` for DB
- Frontend: Jest + React Testing Library, mock API calls
