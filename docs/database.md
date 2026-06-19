# Database Guide

## Schema Overview

The ITSM system uses PostgreSQL 15+ with the following main entities:

```
users ────────┬───── user_roles ─────── roles
              │
tickets ──────┼───── ticket_comments ─── attachments
              │
incidents ────┼───── change_records
              │
problems ─────┤
              │
kb_articles ──┴───── knowledge_tags
```

## Ent ORM

The project uses [Ent](https://entgo.io/) as the ORM. Schemas are defined in `itsm-backend/ent/schema/`.

### Generate Code

```bash
cd itsm-backend
go generate ./ent/schema/...
```

### Create Migration

```bash
go generate ent new MigrationName
go generate ./ent
```

### Apply Migrations

```bash
go run -tags migrate main.go
```

## Tables

### users

| Column | Type | Description |
|--------|------|-------------|
| id | uuid | Primary key |
| username | varchar(50) | Unique username |
| email | varchar(255) | Email address |
| password_hash | varchar(255) | Bcrypt hash |
| tenant_id | uuid | Multi-tenant support |
| created_at | timestamp | Creation time |
| updated_at | timestamp | Last update |

### tickets

| Column | Type | Description |
|--------|------|-------------|
| id | uuid | Primary key |
| title | varchar(255) | Ticket title |
| description | text | Full description |
| priority | int | 1=Low, 2=Medium, 3=High, 4=Critical |
| status | int | 0=Open, 1=In Progress, 2=Resolved, 3=Closed |
| category | varchar(50) | Category type |
| requester_id | uuid | FK to users |
| assignee_id | uuid | FK to users (nullable) |
| sla_deadline | timestamp | SLA target time |

### roles

| Column | Type | Description |
|--------|------|-------------|
| id | uuid | Primary key |
| name | varchar(50) | Role name (admin/l1/l2/l3) |
| permissions | jsonb | Permission list |

## pgvector (Vector Search)

Vector similarity search is used for the AI-powered knowledge base:

```sql
-- Enable extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Knowledge articles with embeddings
ALTER TABLE kb_articles ADD COLUMN embedding vector(1536);
```

Note: `pgvector` requires PostgreSQL 15+. If not available, vector features are disabled gracefully.

## Indexes

Key indexes for performance:

```sql
-- Ticket lookup by status
CREATE INDEX idx_tickets_status ON tickets(status);

-- User ticket assignment
CREATE INDEX idx_tickets_assignee ON tickets(assignee_id);

-- SLA monitoring
CREATE INDEX idx_tickets_sla_deadline ON tickets(sla_deadline) WHERE status < 2;

-- Full-text search
CREATE INDEX idx_tickets_search ON tickets USING gin(to_tsvector('english', title || ' ' || description));
```

## Connection Pool

Default pool settings (tune for production):

```yaml
# Ent connection pool config
max_open_conns: 25
max_idle_conns: 5
conn_max_lifetime: 5m
```

## Backup

```bash
# Full database dump
pg_dump -U postgres -d itsm > backup_$(date +%Y%m%d).sql

# Compressed backup
pg_dump -U postgres -d itsm | gzip > backup_$(date +%Y%m%d).sql.gz

# Restore
psql -U postgres -d itsm < backup_20260101.sql
```