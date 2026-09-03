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
#   - Reproducible npm ci install using the committed package-lock.json
#   - production stage runs as the built-in "node" user (no root)
#   - production stage ships ONLY the standalone bundle + static + public
#     (devDependencies never reach the final image)

# ============================================================
# Stage 1: Dependencies
# ============================================================
FROM node:22-alpine AS deps

WORKDIR /app

# Keep the install contract aligned with local development and CI.
ENV NPM_CONFIG_LEGACY_PEER_DEPS=true

ARG NPM_REGISTRY
RUN npm config set registry ${NPM_REGISTRY:-https://registry.npmjs.org} \
    && npm config set fund false \
    && npm config set audit false

COPY package.json package-lock.json ./

RUN --mount=type=cache,target=/root/.npm \
    npm ci --ignore-scripts

# Next.js 15 may auto-try pnpm when it detects next.config.ts is used without
# local typescript/@types packages (e.g. if /app/node_modules is an anonymous
# volume).  Install dev-only type packages + pnpm as backup so "next dev"
# doesn't crash with "spawn pnpm ENOENT" during type package auto-install.
RUN --mount=type=cache,target=/root/.npm \
    npm install --no-save --ignore-scripts typescript @types/react @types/node pnpm

# ============================================================
# Stage 2: Build
# ============================================================
FROM node:22-alpine AS builder

WORKDIR /app

ARG ITSM_BACKEND_URL=http://itsm-backend:8090
ARG NPM_REGISTRY
ARG NEXT_PUBLIC_ENABLE_AI=true
# NEXT_PUBLIC_* vars are baked at build time by Next.js. Declaring them as ARG
# (and re-exporting them as ENV) is the only way to make them visible to the
# inlined bundle. Default to empty so production traffic routes through nginx
# (browser → nginx:80 → backend:8090) instead of hitting the backend container
# directly. See Fix #5: audit log Docker IP leak.
ARG NEXT_PUBLIC_API_URL=
ARG NEXT_PUBLIC_API_VERSION=v1
ARG NEXT_PUBLIC_API_TIMEOUT=30000

ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    ITSM_BACKEND_URL=${ITSM_BACKEND_URL} \
    NEXT_PUBLIC_ENABLE_AI=${NEXT_PUBLIC_ENABLE_AI} \
    NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL} \
    NEXT_PUBLIC_API_VERSION=${NEXT_PUBLIC_API_VERSION} \
    NEXT_PUBLIC_API_TIMEOUT=${NEXT_PUBLIC_API_TIMEOUT}

COPY --from=deps /app/node_modules ./node_modules
COPY . .

# CI/build containers must not run repository lifecycle hooks; `prepare` invokes
# Husky, which is intentionally unavailable in the production image build.
RUN --mount=type=cache,target=/root/.npm \
    npm ci --legacy-peer-deps --ignore-scripts && \
    NEXT_DISABLE_TURBOPACK=1 ./node_modules/.bin/next build \
    && test -f .next/standalone/server.js

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

# Persist deps OUTSIDE the anonymous volume mount so the container always has
# a working `next` binary regardless of whether docker-compose first-created
# /app/node_modules as an empty volume. The entrypoint preferentially uses
# /opt/deps/... but falls back to the mount at /app/node_modules.
COPY --from=deps /app/node_modules /opt/deps/node_modules
COPY --from=deps /app/package.json /app/package-lock.json /opt/deps/

# Host-mounted source code goes at /app (docker-compose.dev.yml)
COPY . .

ENV NODE_ENV=development \
    PORT=3000 \
    HOSTNAME="0.0.0.0" \
    ITSM_BACKEND_URL=http://host.docker.internal:8090

EXPOSE 3000

# Runtime package bootstrap.
# /app is host-mounted (hot-reload), so COPY . . at build time is mostly irrelevant,
# but docker-compose.dev.yml also mounts ./itsm-frontend:/app and an anonymous volume
# at /app/node_modules.  That anonymous volume is initially EMPTY, so Next.js 15 cannot
# find typescript/@types/tailwindcss, and its auto-install helper then fails because
# pnpm is not on PATH.  To fix both:
#   1. Keep the canonical install in /opt/deps/node_modules (safe from mounts).
#   2. Add a small bootstrap layer: /opt/bootstrap/node_modules holds a dereferenced
#      (-L) copy of the minimal packages Next.js/webpack resolver MUST see locally
#      (webpack ignores NODE_PATH for `require()` calls from within project files).
#   3. The entrypoint script copies the bootstrap tree INTO /app/node_modules when the
#      anonymous volume is still empty, which gives next/postcss/webpack access to
#      tailwindcss/typescript/autoprefixer without reinstalling.
RUN set -e; \
    mkdir -p /opt/bootstrap/node_modules/.bin; \
    for p in typescript pnpm tailwindcss postcss @tailwindcss autoprefixer next react react-dom antd @ant-design @antv @babel @hello-pangea @dnd-kit @emotion @rc-component; do \
      if [ -e "/opt/deps/node_modules/$p" ]; then \
        cp -rL "/opt/deps/node_modules/$p" "/opt/bootstrap/node_modules/$p"; \
      fi; \
    done; \
    for scope in @types @typescript-eslint; do \
      if [ -d "/opt/deps/node_modules/$scope" ]; then \
        mkdir -p "/opt/bootstrap/node_modules/$scope"; \
        for sub in "/opt/deps/node_modules/$scope"/*; do \
          name=$(basename "$sub"); \
          [ -d "$sub" ] && cp -rL "$sub" "/opt/bootstrap/node_modules/$scope/$name"; \
        done; \
      fi; \
    done; \
    for bin in next tsc pnpm tailwindcss postcss; do \
      for src in "/opt/deps/node_modules/.bin/$bin" "/opt/bootstrap/node_modules/.bin/$bin"; do \
        if [ -f "$src" ]; then \
          cp -L "$src" "/opt/bootstrap/node_modules/.bin/$bin"; \
          chmod +x "/opt/bootstrap/node_modules/.bin/$bin"; \
          break; \
        fi; \
      done; \
    done
# Autoprefixer is a dep of tailwindcss and is installed under node_modules, but may be
# a nested dependency only.  Find it under /opt/deps/node_modules and copy it up.
RUN set -e; \
  if [ ! -d /opt/deps/node_modules/autoprefixer ]; then \
    found=$(find /opt/deps/node_modules -maxdepth 3 -type d -name autoprefixer | head -1); \
    if [ -n "$found" ]; then \
      cp -rL "$found" /opt/bootstrap/node_modules/autoprefixer; \
    fi; \
  else \
    cp -rL /opt/deps/node_modules/autoprefixer /opt/bootstrap/node_modules/autoprefixer; \
  fi;
# Copy autoprefixer .bin if available
RUN set -e; \
  for src in "/opt/deps/node_modules/.bin/autoprefixer"; do \
    if [ -f "$src" ]; then \
      cp -L "$src" "/opt/bootstrap/node_modules/.bin/autoprefixer"; \
      chmod +x "/opt/bootstrap/node_modules/.bin/autoprefixer"; \
      break; \
    fi; \
  done;
RUN cat <<'SH' > /entrypoint-next.sh
#!/bin/sh
set -e
# When the anonymous docker-compose volume at /app/node_modules is empty, copy
# the FULL /opt/deps/node_modules tree into /app/node_modules.  This ensures
# Next.js/webpack/postcss can resolve *all* modules (including transitive ones
# like @tailwindcss/postcss, @rc-component/util, etc.) locally, since the
# webpack resolver used by Next.js does NOT consult NODE_PATH for project-side
# require()s.  Copy cost is paid once per anonymous volume lifecycle, guarded
# by a sentinel file so subsequent starts skip it.
SENTINEL=/app/node_modules/.strapi-full-copy-ok
if [ ! -f "$SENTINEL" ]; then
  echo "[entrypoint-next] Populating /app/node_modules from /opt/deps/node_modules..." >&2
  # Ensure target exists (anonymous volume may be empty dir)
  mkdir -p /app/node_modules/.bin
  # cp -a preserves symlinks.  /opt/deps/node_modules contains either real files
  # or valid symlinks pointing to files also inside /opt/deps (which exist in
  # the same fs overlay), so the result is a working node_modules tree.
  cp -a /opt/deps/node_modules/. /app/node_modules/ 2>/dev/null || true
  touch "$SENTINEL" 2>/dev/null || true
  echo "[entrypoint-next] /app/node_modules populated." >&2
fi
export PATH="/opt/deps/node_modules/.bin:/app/node_modules/.bin:$PATH"
for p in /app/node_modules/next/dist/bin/next \
         /opt/deps/node_modules/next/dist/bin/next \
         /app/node_modules/.bin/next; do
  if [ -f "$p" ]; then
    echo "[entrypoint-next] using: $p" >&2
    exec node "$p" dev -H 0.0.0.0
  fi
done
echo "[entrypoint-next] FAILED to locate next binary." >&2
ls /app/node_modules/.bin 2>&1 | head -30 >&2
exit 1
SH
RUN chmod +x /entrypoint-next.sh
CMD ["/entrypoint-next.sh"]
