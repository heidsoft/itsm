# Operations Guide

## Service Management

### Using Deployment Scripts

```bash
# Development
./scripts/deploy-dev.sh up          # Start
./scripts/deploy-dev.sh down        # Stop
./scripts/deploy-dev.sh restart    # Restart
./scripts/deploy-dev.sh logs        # Tail logs
./scripts/deploy-dev.sh health      # Health check
./scripts/deploy-dev.sh doctor      # Diagnose

# Production
./scripts/deploy-prod.sh deploy     # Deploy
./scripts/deploy-prod.sh rollback   # Rollback
./scripts/deploy-prod.sh backup    # Backup DB
./scripts/deploy-prod.sh health    # Health check
./scripts/deploy-prod.sh status    # Show status
./scripts/deploy-prod.sh down      # Stop
```

### Manual Docker Commands

```bash
# View running containers
docker compose -f docker-compose.prod.yml ps

# View logs
docker compose -f docker-compose.prod.yml logs -f itsm-backend
docker compose -f docker-compose.prod.yml logs -f itsm-frontend

# Restart service
docker compose -f docker-compose.prod.yml restart itsm-backend

# Access container shell
docker exec -it itsm-backend-prod sh
docker exec -it itsm-postgres-prod psql -U postgres -d itsm
```

## Monitoring

### Health Endpoints

| Service | URL |
|---------|-----|
| Backend | http://localhost:8090/api/v1/health |
| Frontend | http://localhost:3000 |
| MinIO | http://localhost:9000/minio/health/live |

### Metrics (Prometheus)

If monitoring stack is deployed:

- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3001` (admin/admin)

Key metrics:
- `itsm_ticket_total` - Total tickets by status
- `itsm_api_request_duration_seconds` - API latency
- `itsm_sla_breaches_total` - SLA breaches

## Log Management

### Application Logs

Backend logs (JSON format):

```json
{
  "level": "info",
  "ts": "2026-05-16T08:00:00Z",
  "caller": "handler.go:42",
  "msg": "Ticket created",
  "ticket_id": "123e4567-e89b-12d3-a456-426614174000"
}
```

### Log Levels

| Level | Use |
|-------|-----|
| debug | Development only |
| info | Normal operation |
| warn | Recoverable issues |
| error | Failures requiring attention |

### Log Rotation

Docker handles log rotation. Configure in `daemon.json`:

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
```

## Backup & Recovery

### Automated Backups

Backups are created automatically during each production deployment:

```bash
# List backups
ls -la backups/

# Restore from backup
gunzip < backups/itsm_prod_20260516_150229.sql.gz | psql -U postgres -d itsm
```

### Manual Backup

```bash
# Database only
docker compose -f docker-compose.prod.yml exec postgres pg_dump -U postgres itsm > backups/manual_$(date +%Y%m%d).sql

# Full stack backup
tar -czf itsm_backup_$(date +%Y%m%d).tar.gz \
  backups/ \
  .env.prod \
  nginx/conf.d/
```

## Performance Tuning

### Backend

Key GOMAXPROCS and connection pool settings:

```bash
# Set CPU cores
GOMAXPROCS=4

# Database pool
MAX_OPEN_CONNS=25
MAX_IDLE_CONNS=5
```

### Frontend

Next.js standalone mode minimizes memory footprint:

```bash
# Reduce Node.js memory
NODE_OPTIONS="--max-old-space-size=512"
```

### Database

PostgreSQL tuning in `postgresql.conf`:

```ini
max_connections = 100
shared_buffers = 256MB
effective_cache_size = 1GB
maintenance_work_mem = 64MB
wal_buffers = 16MB
checkpoint_completion_target = 0.9
```

## Security Hardening

1. **Change default passwords** - `admin123` must be changed
2. **Restrict CORS** - Set `ITSM_ALLOW_ALL_ORIGINS=false`
3. **Enable HTTPS** - Terminate SSL at reverse proxy
4. **Rotate secrets** - Update JWT_SECRET periodically
5. **Network isolation** - Use Docker networks, don't expose DB port
6. **Rate limiting** - Configure nginx rate limits
7. **Regular updates** - Enable Dependabot for dependency updates

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker compose -f docker-compose.prod.yml logs itsm-backend

# Common causes:
# - Port already in use
# - Missing environment variables
# - Database not ready
```

### High Memory Usage

```bash
# Check container stats
docker stats

# Restart heavy containers
docker compose -f docker-compose.prod.yml restart --no-deps itsm-frontend
```

### Database Connection Issues

```bash
# Test connectivity
docker compose -f docker-compose.prod.yml exec postgres pg_isready -U postgres

# Check connection string
docker compose -f docker-compose.prod.yml exec itsm-backend env | grep DB_
```

## Maintenance Windows

For zero-downtime deployments, use rolling updates:

```bash
# Pull latest images
docker pull itsm-backend:latest
docker pull itsm-frontend:latest

# Rolling restart
docker compose -f docker-compose.prod.yml up -d --no-deps itsm-backend
docker compose -f docker-compose.prod.yml up -d --no-deps itsm-frontend
```

## Disaster Recovery

### Full System Recovery

1. Restore database from latest backup
2. Pull images: `docker pull` all services
3. Deploy: `./scripts/deploy-prod.sh deploy`
4. Verify: `./scripts/deploy-prod.sh health`

### Point-in-Time Recovery

PostgreSQL WAL archiving for precise recovery:

```bash
# Enable in docker-compose.prod.yml
POSTGRES_HOST_AUTH_METHOD=trust  # NEVER in production
# Use pgBackRest or Barman for production WAL archiving
```