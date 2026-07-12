# syntax=docker/dockerfile:1.7
#
# ITSM Frontend — multi-stage Next.js build
#
# Stages:
#   deps       — install dependencies (cached)
#   builder    — next build (standalone output)
#   production — minimal runtime server (non-root)
#   runner     — dev server with hot reload (local only)
#
# Optimizations:
#   - Pinned node:22-alpine
#   - BuildKit cache mount for npm (~/.npm)
#   - production stage runs as the built-in "node" user (no root)
#   - production stage ships ONLY the standalone bundle + static + public
#     (devDependencies never reach the final image)

# ============================================================
# Stage 1: Dependencies
# ============================================================
FROM node:22-alpine AS deps

WORKDIR /app

# eslint v10 peer-range escape hatch (mirrors prior Dockerfile behavior).
ENV NPM_CONFIG_LEGACY_PEER_DEPS=true

ARG NPM_REGISTRY
RUN npm config set registry ${NPM_REGISTRY:-https://registry.npmjs.org}

COPY package.json package-lock.json ./

RUN --mount=type=cache,target=/root/.npm \
    npm ci --prefer-offline

# ============================================================
# Stage 2: Build
# ============================================================
FROM node:22-alpine AS builder

WORKDIR /app

ARG ITSM_BACKEND_URL=http://itsm-backend:8090
ARG NPM_REGISTRY
ARG NEXT_PUBLIC_ENABLE_AI=true

ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    ITSM_BACKEND_URL=${ITSM_BACKEND_URL} \
    NEXT_PUBLIC_ENABLE_AI=${NEXT_PUBLIC_ENABLE_AI}

COPY --from=deps /app/node_modules ./node_modules
COPY . .

RUN --mount=type=cache,target=/root/.npm \
    npm run build

# ============================================================
# Stage 3a: Production runtime
# ============================================================
FROM node:22-alpine AS production

WORKDIR /app

ENV NODE_ENV=production \
    PORT=3000 \
    HOSTNAME="0.0.0.0" \
    ITSM_BACKEND_URL=http://itsm-backend:8090 \
    NEXT_TELEMETRY_DISABLED=1

# Standalone bundle embeds only the modules it needs; no node_modules copied.
COPY --from=builder --chown=node:node /app/public ./public
COPY --from=builder --chown=node:node /app/.next/standalone ./
COPY --from=builder --chown=node:node /app/.next/static ./.next/static

# Next writes a small on-disk cache at runtime; make it writable by "node".
RUN mkdir -p .next/cache && chown -R node:node .next

USER node

EXPOSE 3000

CMD ["node", "server.js"]

# ============================================================
# Stage 3b: Development runtime (hot reload)
# ============================================================
FROM node:22-alpine AS runner

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

ENV NODE_ENV=development \
    PORT=3000 \
    HOSTNAME="0.0.0.0" \
    ITSM_BACKEND_URL=http://host.docker.internal:8090

EXPOSE 3000

CMD ["npx", "next", "dev"]
