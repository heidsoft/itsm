<div align="center">

# 🤖 AI-Native ITSM

## Enterprise IT Service Management | AI First, Not AI After

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Next.js](https://img.shields.io/badge/Next.js-15.5-000000?style=flat&logo=nextdotjs)](https://nextjs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-6.0-3178C6?style=flat&logo=typescript)](https://typescriptlang.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-yellowgreen?style=flat)](LICENSE)
[![Backend CI](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml/badge.svg)](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml)
[![Frontend CI](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml/badge.svg)](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml)
[![Stars](https://img.shields.io/github/stars/heidsoft/itsm?style=flat)](https://github.com/heidsoft/itsm/stargazers)

**[简体中文](./README.md)** · **English** · **[日本語](./README.ja.md)**

**ITIL Processes · BPMN Workflow · CMDB · AI Decision Support · Apache-2.0**

</div>

## Overview

ITSM is an open-source enterprise service management platform designed for digital process governance. It aims to provide ServiceNow-class core ITSM capabilities while remaining lightweight, private-deployment friendly, and extensible.

The platform covers tickets, incidents, problems, changes, releases, service requests, service catalogs, knowledge, SLA, CMDB, and BPMN orchestration. AI is embedded into triage, summarization, knowledge retrieval, workflow recommendations, audit trails, and controlled tool execution.

The project is currently in the v1.1 hardening phase. Before a production rollout, validate security configuration, backup and recovery, capacity, SSO and organization synchronization, monitoring, and disaster recovery for your environment.

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

The development-only initial account is `admin / admin123`.

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
- [Build and deployment guide](./docs/deployment.md)
- [Operations guide](./docs/operations.md)
- [Configuration](./docs/configuration.md)
- [Production readiness program](./docs/delivery/production-readiness-program.md)
- [Contributing](./CONTRIBUTING.md)

## License

Licensed under the [Apache License 2.0](./LICENSE).
