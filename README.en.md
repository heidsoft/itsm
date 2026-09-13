<div align="center">

# AI-Native ITSM

An open-source IT service management system for enterprises, covering ITIL core processes with BPMN workflow orchestration, CMDB, SLA, knowledge base, and multi-tenancy.

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Next.js](https://img.shields.io/badge/Next.js-15.5-000000?style=flat&logo=nextdotjs)](https://nextjs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-6.0-3178C6?style=flat&logo=typescript)](https://typescriptlang.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-yellowgreen?style=flat)](LICENSE)
[![Backend CI](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml/badge.svg)](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml)
[![Frontend CI](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml/badge.svg)](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml)
[![Stars](https://img.shields.io/github/stars/heidsoft/itsm?style=flat)](https://github.com/heidsoft/itsm/stargazers)

**[简体中文](./README.md)** · **English** · **[日本語](./README.ja.md)**

</div>

## Overview

This project aims to provide an enterprise ITSM system that actually works. It connects tickets, incidents, problems, changes, SLA, CMDB, and knowledge base into complete business processes, with audit trails, permission control, and multi-tenant isolation.

Key design choices:

- **BPMN for workflow orchestration**: Uses BPMN 2.0 standard for approvals and workflows, no custom engine
- **CMDB is more than an asset table**: Configuration items and relationships feed into incident, change, and other processes for impact analysis
- **AI as assistant, not replacement**: Triage, summarization, and knowledge retrieval can use AI, but must degrade gracefully, have audit records, and never bypass human approval
- **Reliable async operations**: Workflow triggers, notifications, and other critical operations use transaction + outbox pattern, not goroutine fire-and-forget

> **Current version v1.6.x**: In production hardening phase. Core ITIL processes (tickets, incidents, problems, changes, SLA, CMDB) are available, but module maturity varies. Some features are still in Pilot stage. Check the [open-source capability statement](./docs/product/open-source-release-capability.md) before production use.

## Core Capabilities

- ITIL service management: ticket, incident, problem, change, release, and request management
- BPMN workflow definitions, process instances, user tasks, and process bindings
- CMDB with CI types, configuration items, relationships, topology, and impact analysis
- Service catalog, knowledge base, SLA monitoring, and escalation
- AI-assisted triage, summarization, RAG, audit records, and deterministic fallback
- RBAC, tenant isolation, MSP foundations, and organization management
- Connector lifecycle and marketplace foundations for Feishu, WeCom, DingTalk, and Webhook

## Quick Start

### Docker Development

```bash
git clone https://github.com/heidsoft/itsm.git
cd itsm
cp .env.dev.example .env

make dev-start-docker
make dev-status
make dev-health
```

Open:

- Frontend: `http://localhost:3000`
- Backend: `http://localhost:8090`
- Swagger: `http://localhost:8090/swagger/index.html`

> **Language note:** through v1.6.x the product UI is **Chinese-first** (targeting China private-deployment scenarios). English/Japanese READMEs cover onboarding; full UI localization (en-US etc.) is planned for v1.7 — see the [ROADMAP](ROADMAP.md).

The development-only initial account is `admin / admin123`. For production deployments, the admin password is set by `ADMIN_PASSWORD` in your `.env.prod` (applied by the one-shot `itsm-init` container on first start) — never use documentation example passwords there.

### (Optional) One-Command Demo Data

A fresh system ships with configuration templates only — no business records. Run the following
to seed a demo dataset (8 incidents, 2 problems, 3 changes, 5 knowledge articles) so you can
explore each module with realistic content:

```bash
make dev-seed-demo   # idempotent, safe to re-run
```

Demo records use fixed numbers (`INC-DEMO-xxxx` / `PRB-DEMO-xxxx` / `CHG-DEMO-xxxx`) covering
the incident lifecycle (new → escalated → closed). Production deployments never seed these
fictional records.

Stop the environment:

```bash
make dev-stop-docker
```

### Local Go and Next.js Development

```bash
cp .env.dev.example .env
make dev-start-local

# Stop local application processes
make dev-stop-local
```

## Production Deployment

```bash
# Generate .env.prod and initial random secrets
make prod-init

# Replace every REQUIRED/default credential in .env.prod

# Validate, back up, build, deploy, and verify
make prod-deploy

make prod-status
make prod-health
```

When running Compose manually, always pass the production environment file explicitly:

```bash
docker compose --env-file .env.prod -f docker-compose.prod.yml build itsm-backend itsm-frontend
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d
```

`docker-compose.prod.yml` pins the Compose project name to `itsm-prod` (dev keeps the
directory-derived project `itsm`), so dev and prod stacks no longer collide or evict each
other's containers. The backend's host diagnostic port defaults to `127.0.0.1:8090`; set
`BACKEND_DIAG_PORT` in `.env.prod` (e.g. `8091`) when a dev stack is running on the same host.

Never use development passwords in production. Configure TLS, off-host backups, log retention, and monitoring before go-live.

## Building Versioned Images

```bash
# Local tags such as itsm-backend:v1.2.0
VERSION=v1.2.0 make build-images

# Registry-prefixed tags
VERSION=v1.2.0 REGISTRY=ghcr.io/heidsoft make build-images

# Build a single application image
VERSION=v1.2.0 make build-backend
VERSION=v1.2.0 make build-frontend
```

The image builder targets the native host platform by default. Set `BUILDPLATFORM=linux/amd64` or another supported target when producing cross-platform delivery images.

## Verification

```bash
make verify-scripts
make check-contracts

cd itsm-backend && go test ./...
cd ../itsm-frontend && npm run type-check && npm run build
```

## Documentation

- [Documentation index](./docs/README.md)
- [Build and deployment guide](./docs/DEPLOYMENT_OPTIMIZATION.md)
- [Operations runbook](./docs/runbooks/production-initialization.md)
- [Configuration reference](./docs/getting-started/install.md)
- [Production readiness program](./docs/delivery/production-readiness-program.md)
- [Upgrade guide](./UPGRADE.md)
- [Contributing](./CONTRIBUTING.md)

## License

Licensed under the [Apache License 2.0](./LICENSE).
