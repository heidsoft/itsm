# ============================================
# Stage 1: Dependencies Installation
# ============================================
FROM node:22-alpine AS deps

WORKDIR /app

# Clear proxy settings to avoid Docker Desktop proxy interference
ARG HTTP_PROXY=
ARG HTTPS_PROXY=
ARG http_proxy=
ARG https_proxy=
ARG NO_PROXY=localhost,127.0.0.1
ARG no_proxy=localhost,127.0.0.1

ARG NPM_REGISTRY
# Use configurable npm registry (defaults to official registry, override with build arg for China: --build-arg NPM_REGISTRY=https://registry.npmmirror.com)
RUN unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy ALL_PROXY all_proxy && \
    npm config set registry ${NPM_REGISTRY:-https://registry.npmjs.org}

# Copy package files first for caching
COPY package.json pnpm-lock.yaml* package-lock.json* ./

# Install pnpm and dependencies (without --ignore-scripts so postinstall hooks run)
RUN unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy ALL_PROXY all_proxy && \
    npm install -g pnpm@9.0.0 && \
    pnpm install --frozen-lockfile || pnpm install

# ============================================
# Stage 2: Build Next.js application (production)
# ============================================
FROM node:22-alpine AS builder

WORKDIR /app

ARG HTTP_PROXY=
ARG HTTPS_PROXY=
ARG http_proxy=
ARG https_proxy=
ARG NO_PROXY=localhost,127.0.0.1
ARG no_proxy=localhost,127.0.0.1

RUN unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy ALL_PROXY all_proxy && \
    npm config set registry ${NPM_REGISTRY:-https://registry.npmjs.org} && \
    npm install -g pnpm@9.0.0

# Copy dependencies from deps stage
COPY --from=deps /app/node_modules ./node_modules
COPY . .

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy ALL_PROXY all_proxy && \
    pnpm run build

# ============================================
# Stage 3a: Production runner
# ============================================
FROM node:22-alpine AS production

WORKDIR /app

ENV NODE_ENV=production
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

EXPOSE 3000

CMD ["node", "server.js"]

# ============================================
# Stage 3b: Development runner
# ============================================
FROM node:22-alpine AS runner

WORKDIR /app

# Install pnpm in runner stage (no network needed from host)
RUN unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy ALL_PROXY all_proxy && \
    npm config set registry ${NPM_REGISTRY:-https://registry.npmjs.org} && \
    npm install -g pnpm@9.0.0

# Copy everything from deps (node_modules ready) + source
COPY --from=deps /app/node_modules ./node_modules
COPY . .

RUN mkdir -p .next && chmod 755 .next

ENV NODE_ENV=development
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"
ENV ITSM_BACKEND_URL=http://host.docker.internal:8090
ENV NEXT_PUBLIC_API_URL=http://host.docker.internal:8090

EXPOSE 3000

CMD ["npx", "next", "dev"]
