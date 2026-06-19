# Development Guide

## Prerequisites

- **Go** 1.24+
- **Node.js** 22+
- **Docker** & Docker Compose v2
- **pnpm** 9+
- **PostgreSQL** 15+ (provided via Docker)
- **Redis** 7+ (provided via Docker)

## Quick Start

```bash
# 1. Clone and enter directory
git clone https://github.com/heidsoft/itsm.git
cd itsm

# 2. Start development environment
./scripts/deploy-dev.sh init

# Or use Make
make quickstart
```

## Project Structure

```
itsm/
├── itsm-backend/          # Go backend (Gin + Ent ORM)
│   ├── cmd/              # Main entry point
│   ├── controller/       # HTTP handlers
│   ├── service/          # Business logic
│   ├── ent/schema/      # Database schemas
│   ├── middleware/       # Auth, CORS, RBAC
│   └── docs/            # Swagger specs
├── itsm-frontend/         # Next.js frontend
│   ├── src/app/         # App Router pages
│   ├── src/components/  # React components
│   ├── src/lib/         # API clients, stores, hooks
│   └── tests/           # E2E tests (Playwright)
├── scripts/              # Deployment scripts
│   ├── deploy-dev.sh    # Dev environment
│   └── deploy-prod.sh   # Production deployment
├── nginx/               # Reverse proxy config
└── monitoring/          # Prometheus + Grafana
```

## Running Locally

### Backend

```bash
cd itsm-backend

# Database migration
go run -tags migrate main.go

# Run with hot reload
go install github.com/air-verse/air@latest
air

# Run tests
go test ./... -v -count=1
```

### Frontend

```bash
cd itsm-frontend

# Install dependencies
npm install

# Start dev server
npm run dev

# Build for production
npm run build
```

### With Docker

```bash
# Full stack
docker compose -f docker-compose.dev.yml up -d

# Just infrastructure
docker compose -f docker-compose.dev.yml --profile infra up -d

# Backend + Frontend only
docker compose -f docker-compose.dev.yml up -d itsm-backend itsm-frontend
```

## Environment Variables

Create `.env` from the example:

```bash
cp .env.example .env
```

Key variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | localhost |
| `DB_PORT` | PostgreSQL port | 5432 |
| `DB_PASSWORD` | Database password | (set in .env) |
| `JWT_SECRET` | JWT signing key (min 32 chars) | - |
| `LOG_LEVEL` | Log level (debug/info/warn/error) | info |

## Testing

### Backend Tests

```bash
cd itsm-backend
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out  # View coverage
```

### Frontend Tests

```bash
cd itsm-frontend
npm test                    # All tests
npm run test:unit           # Unit only
npm run test:integration    # Integration only
npm run test:e2e           # E2E (requires backend)
npm run test:ci            # CI mode with coverage
```

### E2E Tests

```bash
cd itsm-frontend
npx playwright install --with-deps
npm run test:e2e
```

## Code Quality

```bash
# Backend lint
cd itsm-backend
gofumpt -w .               # Format
staticcheck ./...          # Lint
go vet ./...               # Vet

# Frontend lint
cd itsm-frontend
npm run lint:check         # Check only
npm run lint -- --fix      # Auto-fix
npm run type-check         # TypeScript
```

## Database

### Migrations

```bash
cd itsm-backend

# Create migration
go generate ent new MigrationName

# Apply migrations
go run -tags migrate main.go

# Rollback (manual)
psql -U itsm -d itsm -f scripts/down.sql
```

### Seed Data

```bash
# After first migration
docker compose -f docker-compose.dev.yml --profile dev exec itsm-backend sh -c "go run -tags create_user main.go"
```

## Git Workflow

1. Create feature branch: `git checkout -b feat/ticket-field`
2. Write tests first (TDD)
3. Implement and pass tests
4. Run `gofumpt` and `npm run lint -- --fix`
5. Commit with conventional format: `feat: add ticket priority field`
6. Push and create PR

See [CONTRIBUTING.md](../CONTRIBUTING.md) for details.

## Troubleshooting

### Backend fails to start

```bash
# Check database is running
docker compose -f docker-compose.dev.yml ps postgres

# Check migrations applied
docker compose -f docker-compose.dev.yml exec postgres psql -U itsm -d itsm -c "\dt"
```

### Frontend build fails

```bash
# Clear Next.js cache
rm -rf itsm-frontend/.next

# Reinstall dependencies
cd itsm-frontend && rm -rf node_modules && npm install
```

### Port conflicts

```bash
# Check what's using the port
lsof -i :8090  # Backend
lsof -i :3000  # Frontend
lsof -i :5432  # PostgreSQL
```

## Useful Commands

```bash
make dev-up           # Start dev
make dev-down         # Stop dev
make dev-logs         # View logs
make dev-doctor       # Diagnose issues
make lint             # Lint all
make test            # Run all tests
make swagger         # Generate API docs
```
