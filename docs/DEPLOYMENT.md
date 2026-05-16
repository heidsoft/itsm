# Deployment Guide

## Environments

| Environment | Command | Use |
|-------------|---------|-----|
| Development | `make dev-up` / `./scripts/deploy-dev.sh up` | Local development |
| Production | `./scripts/deploy-prod.sh deploy` | Live deployment |
| Staging | (configure via `.env.staging`) | Pre-release testing |

## Production Requirements

- **OS**: Linux (Ubuntu 22.04+ recommended)
- **CPU**: 4+ cores
- **RAM**: 8+ GB
- **Disk**: 50+ GB SSD
- **Docker**: 24+ with Docker Compose v2

## Production Deployment

### 1. Prepare Environment

```bash
# Clone repository
git clone https://github.com/heidsoft/itsm.git
cd itsm

# Copy production environment template
cp .env.prod.example .env.prod
```

### 2. Configure Secrets

Edit `.env.prod` with real values:

```bash
# Generate secure values
openssl rand -base64 48    # JWT_SECRET
openssl rand -base64 48    # MINIO_SECRET_KEY

# Set strong database password (min 16 chars)
DB_PASSWORD=<your-secure-password>

# Required: change all [REQUIRED] marked values
```

### 3. Deploy

```bash
# Full deployment with build and verification
./scripts/deploy-prod.sh deploy

# Skip build (use pre-built images)
./scripts/deploy-prod.sh deploy --skip-build

# Dry run to preview
./scripts/deploy-prod.sh deploy --dry-run
```

### 4. Verify

```bash
# Health check all services
./scripts/deploy-prod.sh health

# Check status
./scripts/deploy-prod.sh status
```

## Service Endpoints

| Service | URL | Purpose |
|---------|-----|---------|
| Frontend | http://your-domain.com | User interface |
| Backend API | http://your-domain.com/api/v1 | REST API |
| Swagger Docs | http://your-domain.com/swagger | API documentation |
| MinIO Console | http://your-domain.com:9000 | File storage UI |

Default credentials: `admin` / `admin123` (change immediately)

## Docker Compose Production

Manual deployment without scripts:

```bash
# Build images
docker compose -f docker-compose.prod.yml build

# Start services
docker compose -f docker-compose.prod.yml up -d

# Check status
docker compose -f docker-compose.prod.yml ps

# View logs
docker compose -f docker-compose.prod.yml logs -f
```

## Reverse Proxy (Nginx)

The included `nginx/conf.d/default.conf` provides:
- HTTPS reverse proxy to backend (port 8090) and frontend (port 3000)
- Security headers (HSTS, CSP, X-Frame-Options)
- Gzip compression
- WebSocket support

For production, replace with your domain SSL certificates.

## Database Backup

```bash
# Manual backup
./scripts/deploy-prod.sh backup

# Automated backups are created during each deployment
# Located in ./backups/ (last 5 kept)
```

## Rollback

```bash
# Rollback to previous version
./scripts/deploy-prod.sh rollback

# Rollback is automatic on deploy failure
```

## HTTPS Setup

### Option 1: Nginx with Let's Encrypt

```bash
# Install certbot
sudo apt install certbot python3-certbot-nginx

# Obtain certificate
sudo certbot --nginx -d your-domain.com

# Auto-renewal
sudo crontab -e
# Add: 0 0 * * * certbot renew --quiet
```

### Option 2: Cloud Load Balancer

Terminate SSL at your cloud provider's load balancer and forward plain HTTP to the nginx container.

## Environment Variables Reference

See [`.env.prod.example`](../.env.prod.example) for all configuration options.

## Monitoring

Prometheus and Grafana are available in the `monitoring/` directory:

```bash
docker compose -f monitoring/docker-compose.monitoring.yml up -d
```

Access Grafana at `http://your-domain.com:3001` (default: admin/admin)

## Troubleshooting

```bash
# View all logs
docker compose -f docker-compose.prod.yml logs -f

# View specific service
docker compose -f docker-compose.prod.yml logs -f itsm-backend

# Restart a service
docker compose -f docker-compose.prod.yml restart itsm-backend

# Full restart
docker compose -f docker-compose.prod.yml down
docker compose -f docker-compose.prod.yml up -d
```