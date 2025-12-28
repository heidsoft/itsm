# ITSM Backend

## Quick Start

1. Prerequisites: Go 1.21+, PostgreSQL
2. Configure env or `itsm-backend/config.yaml`; optional: `.env` (see project root `.env.example`)
3. Install deps and build:

```
make setup
```

4. Run:

```
cd itsm-backend && go run .
```

API base: `http://localhost:8080/api/v1`

## Swagger Docs

After server starts, open:

- Swagger UI: `http://localhost:8080/swagger/index.html`
- OpenAPI JSON: `http://localhost:8080/docs/swagger.json`

Generate docs locally:

```
# Install once
GO111MODULE=on go install github.com/swaggo/swag/cmd/swag@latest
$(go env GOPATH)/bin/swag init -g main.go -o ./docs
```

## Multi-tenancy & Auth

- JWT Bearer in `Authorization`
- `X-Tenant-Code` header supported; tenant id validated against JWT
