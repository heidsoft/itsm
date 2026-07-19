# ITSM Frontend (Next.js)

## Quick Start

1. Node 22+
2. Install deps:

```
npm install
```

3. Configure env:

```
cp .env.example .env.local
# Local direct-backend development may set NEXT_PUBLIC_API_URL=http://localhost:8090.
# Production should leave it empty so /api/v1/* stays same-origin behind Nginx.
```

4. Dev:

```
npm run dev
```

## Testing

- Type check: `npm run type-check`
- Unit/RTL: `npm run test:unit`
- Integration: `npm run test:integration`
- E2E/Playwright: `npm run test:e2e`

## Lint/Format

```
npm run lint
npm run lint:check
```

## Production build

```bash
npm run build
test -f .next/standalone/server.js
```

The browser-facing API base is empty by default. Requests already include
`/api/v1`, while server-side proxying uses `ITSM_BACKEND_URL`.
