# ITSM Prototype (Next.js)

## Quick Start

1. Node 18+
2. Install deps:

```
npm install
```

3. Configure env:

```
cp ../.env.example .env.local
# ensure NEXT_PUBLIC_API_URL points to backend
```

4. Dev:

```
npm run dev
```

## Testing

- Unit/RTL: `npm test`
- E2E/Playwright: add tests under `tests/e2e` and run `npx playwright test`

## Lint/Format

```
npm run lint
npm run format
```
