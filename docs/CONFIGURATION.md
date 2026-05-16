# Configuration Reference

## Environment Variables

All configuration is done via environment variables. See `.env.prod.example` for production settings.

### Backend

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `DB_HOST` | PostgreSQL host | Yes | localhost |
| `DB_PORT` | PostgreSQL port | No | 5432 |
| `DB_USER` | Database user | Yes | postgres |
| `DB_PASSWORD` | Database password | Yes | - |
| `DB_NAME` | Database name | No | itsm |
| `REDIS_HOST` | Redis host | Yes | localhost |
| `REDIS_PORT` | Redis port | No | 6379 |
| `REDIS_PASSWORD` | Redis password | No | - |
| `JWT_SECRET` | JWT signing key (min 32 chars) | Yes | - |
| `LOG_LEVEL` | Log level: debug/info/warn/error | No | info |
| `PORT` | Backend HTTP port | No | 8090 |
| `ENABLE_SWAGGER` | Enable Swagger UI | No | true |
| `CORS_ALLOWED_ORIGINS` | CORS allowed origins (comma-separated) | No | * |
| `ITSM_ALLOW_ALL_ORIGINS` | Allow all CORS origins | No | false |

### AI / LLM

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `OPENAI_API_KEY` | OpenAI API key | No | - |
| `OPENAI_BASE_URL` | OpenAI API base URL | No | https://api.openai.com/v1 |
| `MINIMAX_API_KEY` | MiniMax API key | No | - |
| `MINIMAX_BASE_URL` | MiniMax API base URL | No | https://api.minimax.chat/v1 |
| `NEXT_PUBLIC_ENABLE_AI` | Enable AI features in frontend | No | true |

### Object Storage (MinIO)

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `MINIO_ENDPOINT` | MinIO server endpoint | No | minio:9000 |
| `MINIO_ACCESS_KEY` | MinIO access key | Yes | - |
| `MINIO_SECRET_KEY` | MinIO secret key | Yes | - |
| `MINIO_BUCKET` | MinIO bucket name | No | itsm-uploads |
| `MINIO_USE_SSL` | Use HTTPS for MinIO | No | false |

### Email (SMTP)

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `SMTP_HOST` | SMTP server host | No | - |
| `SMTP_PORT` | SMTP port | No | 587 |
| `SMTP_USERNAME` | SMTP username | No | - |
| `SMTP_PASSWORD` | SMTP password | No | - |
| `SMTP_FROM` | From email address | No | noreply@itsm.local |

### Notifications

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `DINGTALK_WEBHOOK` | DingTalk robot webhook URL | No | - |
| `WECOM_WEBHOOK` | WeCom robot webhook URL | No | - |

## Next.js (Frontend)

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `NEXT_PUBLIC_API_URL` | Backend API URL | Yes | http://localhost:8090 |
| `NEXT_PUBLIC_ENABLE_AI` | Enable AI features | No | true |

## Configuration Files

### Backend (config.yaml)

The backend also supports `config.yaml` for static configuration:

```yaml
server:
  port: 8090
  log_level: info

database:
  host: localhost
  port: 5432
  user: postgres
  password: your-password
  name: itsm

redis:
  host: localhost
  port: 6379
```

Environment variables take precedence over `config.yaml`.

### Docker Compose Override

For Docker deployments, create `docker-compose.override.yml`:

```yaml
version: '3.8'
services:
  itsm-backend:
    environment:
      DB_HOST: postgres
      REDIS_HOST: redis
```

## Feature Flags

| Flag | Description |
|------|-------------|
| `ENABLE_SWAGGER` | Swagger API docs at /swagger |
| `ENABLE_AI_TRIAGE` | AI-powered ticket triage |
| `ENABLE_AI_SUMMARY` | AI-generated ticket summaries |
| `ENABLE_RAG` | RAG-based knowledge base search |