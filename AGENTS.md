# Repository Guidelines

## Project Structure & Module Organization
This repository is a two-app workspace:
- `itsm-frontend/` is a Next.js app. Core code lives in `itsm-frontend/src/app/` (App Router), `itsm-frontend/src/components/` (ui/business/forms), `itsm-frontend/src/lib/` (api/utils/config), `itsm-frontend/src/stores/`, `itsm-frontend/src/types/`, and styles in `itsm-frontend/src/styles/`. Static assets are in `itsm-frontend/public/`.
- `itsm-backend/` is a Go API. Main layers include `itsm-backend/controller/`, `itsm-backend/service/`, `itsm-backend/dto/`, `itsm-backend/ent/`, `itsm-backend/middleware/`, `itsm-backend/router/`, and configuration in `itsm-backend/config/` or `itsm-backend/config.yaml`.
- Shared and tooling folders include `shared-types/`, `scripts/`, `tools/`, and `prd/`.

## Build, Test, and Development Commands
Frontend (from repo root):
- `cd itsm-frontend && npm install` installs dependencies.
- `npm run dev` starts the dev server.
- `npm run build` builds production assets; `npm run start` serves the build.
- `npm run lint` runs ESLint with auto-fix; `npm run format` formats; `npm run type-check` runs `tsc`.
- `npm test` runs Jest.

Backend:
- `cd itsm-backend && go run .` runs the API on `http://localhost:8080/api/v1`.
- `go test ./...` runs all Go tests.
- `gofmt -w .` formats Go files.
Repo-wide:
- `scripts/code-quality-check.sh` runs a combined frontend/backend quality sweep (lint, type-check, gofmt, coverage).

## Coding Style & Naming Conventions
Follow `CODING-STANDARDS.md`. Highlights:
- Frontend: React components and props are PascalCase; variables/functions are camelCase; API/config/type files often use kebab-case; Next.js `page.tsx` and `layout.tsx` are fixed names.
- Backend: Go files are snake_case; tests end with `_test.go`.
- Prefer `eslint` (Next.js) and `gofmt` for formatting consistency.

## Testing Guidelines
- Frontend: Jest + React Testing Library. Unit/integration tests are under `itsm-frontend/src/app/**/__tests__/`. E2E tests go in `itsm-frontend/tests/e2e` and run with `npx playwright test`.
- Backend: use `go test ./...`. The quality script targets 60%+ coverage (`scripts/code-quality-check.sh`).

## Commit & Pull Request Guidelines
Commit history shows short, imperative summaries (often Chinese), with occasional Conventional Commits like `feat:`. Keep messages concise and descriptive. For PRs, include a clear summary, testing notes, linked issues, and screenshots for UI changes.

## Security & Configuration
Use `.env.example` at the repo root for local defaults. Frontend expects `.env.local` with `NEXT_PUBLIC_API_URL`, and backend supports `itsm-backend/config.yaml` or `.env`. Do not commit secrets.
