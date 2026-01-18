# ITSM - Enterprise IT Service Management Platform

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24-blue" alt="Go">
  <img src="https://img.shields.io/badge/Next.js-15-000000" alt="Next.js">
  <img src="https://img.shields.io/badge/TypeScript-5-3178C6" alt="TypeScript">
  <img src="https://img.shields.io/badge/License-MIT-yellow" alt="License">
</p>

A modern, full-featured IT Service Management (ITSM) platform built with Go/Gin backend and Next.js/React frontend. Supports ITIL best practices with AI-powered features.

## Features

### Core ITSM Modules
- **Ticket Management** - Full ticket lifecycle, SLA support, automated workflows
- **Incident Management** - Real-time monitoring, intelligent triage, escalation
- **Problem Management** - Root cause analysis, known error database
- **Change Management** - Risk assessment, multi-level approvals
- **Service Catalog** - Self-service portal, service requests

### AI-Powered Features
- **Smart Triage** - LLM-powered ticket classification and priority suggestion
- **RAG Knowledge Base** - Retrieval Augmented Generation for intelligent Q&A
- **Auto-Summarization** - AI-generated ticket summaries and action items

### Workflow Engine
- **BPMN 2.0** - Full BPMN workflow engine with visual designer
- **Custom Processes** - Build custom approval and escalation workflows
- **Automation Rules** - Event-driven automation

### SLA & Monitoring
- **SLA Management** - Define and monitor service level agreements
- **Real-time Alerts** - Proactive SLA breach notifications
- **Performance Analytics** - Dashboards and reports

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.24, Gin, Ent ORM |
| Frontend | Next.js 15, React 19, TypeScript 5 |
| UI Library | Ant Design 5, Tailwind CSS 4 |
| Database | PostgreSQL |
| BPMN Engine | nitram509/lib-bpmn-engine |
| AI/ML | OpenAI API, pgvector (optional) |
| State Management | Zustand, TanStack Query |

## Quick Start

### Prerequisites
- Go 1.24+
- Node.js 18+
- PostgreSQL 14+

### Backend Setup

```bash
cd itsm-backend

# Install dependencies
go mod download

# Configure database (edit config.yaml)
vim config.yaml

# Run database migrations
go run -tags migrate main.go

# Start the server
go run main.go
```

The backend will start at `http://localhost:8080`

### Frontend Setup

```bash
cd itsm-frontend

# Install dependencies
npm install

# Configure environment
cp .env.example .env.local
# Edit .env.local with your API URL

# Start development server
npm run dev
```

The frontend will start at `http://localhost:3000`

### Docker Deployment

```bash
# Using docker-compose
docker-compose up -d
```

## Project Structure

```
itsm/
├── itsm-backend/          # Go/Gin backend
│   ├── controller/        # HTTP handlers
│   ├── service/           # Business logic
│   ├── ent/               # Database models (Ent ORM)
│   ├── middleware/        # Auth, RBAC, tenant isolation
│   ├── internal/domain/   # DDD domain layer
│   └── config.yaml        # Configuration
│
├── itsm-frontend/         # Next.js frontend
│   ├── src/app/           # App Router pages
│   ├── src/components/    # React components
│   ├── src/app/lib/       # API clients, stores
│   └── .env.example       # Environment template
│
├── nginx/                 # Nginx configuration
├── scripts/               # Utility scripts
└── docker-compose.yml     # Docker deployment
```

## API Documentation

API documentation available at `http://localhost:8080/swagger/index.html` after starting the backend.

## Environment Variables

### Backend (`itsm-backend/config.yaml`)
```yaml
database:
  host: localhost
  port: 5432
  user: dev
  password: "your-password"
  dbname: itsm

llm:
  provider: openai        # openai, azure, local
  model: gpt-4o-mini
  api_key: ""             # Set via environment

vector:
  enabled: false          # Set to true when pgvector is installed
  dimension: 1536
```

### Frontend (`itsm-frontend/.env.local`)
```env
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_API_TIMEOUT=30000
```

## Key Features Deep Dive

### AI Triage Service
The intelligent triage system classifies incoming tickets using LLM:
- Automatic category detection (database, network, server, etc.)
- Priority suggestion based on urgency
- Assignee recommendation
- Falls back to keyword matching when LLM unavailable

### RAG Knowledge Base
Hybrid search combining vector similarity and keyword search:
- Semantic search using OpenAI embeddings
- Keyword fallback for exact matches
- Automatic knowledge article indexing

### BPMN Workflow Engine
Custom BPMN 2.0 workflow engine with:
- Visual process designer (bpmn-js)
- Version-controlled process definitions
- User task assignment and delegation
- Process monitoring and analytics

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- Create an issue for bug reports
- Discussions for questions and ideas
- Wiki for documentation
