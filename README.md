# ITSM - Enterprise IT Service Management Platform

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24-blue" alt="Go">
  <img src="https://img.shields.io/badge/Next.js-15-000000" alt="Next.js">
  <img src="https://img.shields.io/badge/TypeScript-5-3178C6" alt="TypeScript">
  <img src="https://img.shields.io/badge/License-MIT-yellow" alt="License">
  <img src="https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml/badge.svg?branch=main" alt="Frontend CI">
  <img src="https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml/badge.svg?branch=main" alt="Backend CI">
</p>

A modern, full-featured IT Service Management (ITSM) platform built with Go/Gin backend and Next.js/React frontend. Supports ITIL best practices with AI-powered features.

> **Latest Release**: [v1.0.0](https://github.com/heidsoft/itsm/releases/tag/v1.0.0) | 
> **Status**: 🚧 Active Development |
> **Docs**: [Documentation](./docs/)

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

## 📊 Project Status

### Current Phase: Code Quality Optimization

- ✅ **Phase 1**: Dependency Management
  - ✅ Removed legacy-peer-deps conflicts
  - ✅ Generated complete package-lock.json
  - ✅ Optimized CI/CD pipeline

- 🔄 **Phase 2**: TypeScript Type Fixes (In Progress)
  - ✅ Fixed dates parameter types (11 files)
  - ✅ Added @ant-design/charts type declarations
  - 🔄 Fixing remaining type errors

- ⏳ **Phase 3**: Test Coverage (Upcoming)
  - ⏳ Unit tests
  - ⏳ Integration tests
  - ⏳ E2E tests

### Build Status

| Component | Status | Last Build |
|-----------|--------|------------|
| Frontend CI | 🔄 Fixing | [View](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml) |
| Backend CI | ✅ Ready | [View](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml) |
| Release | ✅ v1.0.0 | [Download](https://github.com/heidsoft/itsm/releases/tag/v1.0.0) |

## 📚 Documentation

- **[CI Optimization Guide](./docs/CI_OPTIMIZATION_SUMMARY.md)** - CI/CD optimization details
- **[Code Optimization Plan](./docs/CODE_OPTIMIZATION_PLAN.md)** - 4-phase optimization strategy
- **[Cron Optimization Plan](./docs/CRON_OPTIMIZATION_PLAN.md)** - Scheduled task optimization
- **[Release Guide](./skills/itsm-release-guide/SKILL.md)** - Release process documentation

## 🚀 Quick Start

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

We welcome contributions! Here's how you can help:

### Development Workflow

1. **Fork** the repository
2. **Create** your feature branch (`git checkout -b feature/AmazingFeature`)
3. **Develop** with tests and documentation
4. **Commit** your changes (`git commit -m 'Add some AmazingFeature'`)
5. **Push** to the branch (`git push origin feature/AmazingFeature`)
6. **Open** a Pull Request

### Code Quality Requirements

Before submitting a PR, ensure:
- [ ] All TypeScript type checks pass (`npm run type-check`)
- [ ] ESLint passes with no errors (`npm run lint`)
- [ ] Unit tests pass (`npm run test:unit`)
- [ ] Integration tests pass (`npm run test:integration`)
- [ ] Build succeeds (`npm run build`)

### Documentation

- Update relevant documentation in `./docs/`
- Add comments to new code
- Update API documentation if applicable

## 📈 Roadmap

### Q1 2026
- [x] v1.0.0 Release
- [x] CI/CD Optimization
- [ ] Complete TypeScript migration
- [ ] Test coverage > 60%

### Q2 2026
- [ ] E2E testing framework
- [ ] Performance optimization
- [ ] Mobile responsiveness
- [ ] Advanced analytics

## 🤝 Community

- **Discussions**: Share ideas and ask questions
- **Issues**: Report bugs and request features
- **Releases**: Stay updated with latest versions

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👥 Maintainers

- **@刘彬** - Initial work & Project Lead
- **@heidsoft** - Core Development

## 📞 Support

- **Bug Reports**: [Create an issue](https://github.com/heidsoft/itsm/issues)
- **Questions**: [GitHub Discussions](https://github.com/heidsoft/itsm/discussions)
- **Documentation**: [Wiki](https://github.com/heidsoft/itsm/wiki)
- **Email**: [Contact Maintainers](mailto:heidsoft@qq.com)

## 🙏 Acknowledgments

- [Ant Design](https://ant.design/) - UI Component Library
- [Next.js](https://nextjs.org/) - React Framework
- [Gin](https://gin-gonic.com/) - Go Web Framework
- [BPMN.js](https://bpmn.io/) - BPMN Workflow Designer
- All our amazing contributors!

---

<p align="center">
  <strong>⭐ Star this repo if you find it helpful!</strong>
</p>
