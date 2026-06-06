# ITSM Backend

## Quick Start

1. Prerequisites: Go 1.21+, PostgreSQL
2. Configure env or `itsm-backend/config.yaml`; optional: `.env` (see project root `.env.example`)
3. Choose deployment mode with `DEPLOYMENT_MODE=private|saas|saas_msp`
3. Install deps and build:

```
make setup
```

1. Run:

```
cd itsm-backend && go run .
```

API base: `http://localhost:8090/api/v1`

## Initialization

- `ITSM_BOOTSTRAP_ONLY=true`: run one-shot migration + seed and exit
- `ITSM_AUTO_MIGRATE=true`: enable schema migration during bootstrap
- `ITSM_AUTO_SEED=true`: enable idempotent seed during bootstrap

In Docker Compose, the recommended flow is:

1. `itsm-init` runs once with `ITSM_BOOTSTRAP_ONLY=true`
2. `itsm-backend` starts after init completes
3. Frontend proxies browser requests through same-origin `/api`

For browser compatibility on localhost, auth cookies are host-only and only marked
`Secure` when the request is actually served over HTTPS.

## Swagger Docs

After server starts, open:

- Swagger UI: `http://localhost:8090/swagger/index.html`
- OpenAPI JSON: `http://localhost:8090/docs/swagger.json`

Generate docs locally:

```
# Install once
GO111MODULE=on go install github.com/swaggo/swag/cmd/swag@latest
$(go env GOPATH)/bin/swag init -g main.go -o ./docs
```

## Multi-tenancy & Auth

- JWT Bearer in `Authorization`
- `X-Tenant-Code` header supported; tenant id validated against JWT
