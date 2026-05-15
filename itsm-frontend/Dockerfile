# ============================================
# Stage 1: Dependencies Installation
# ============================================
FROM node:22-alpine AS deps

WORKDIR /app

# Use China mirror for faster downloads
RUN npm config set registry https://registry.npmmirror.com

# Copy package files first for caching
COPY package.json package-lock.json* pnpm-lock.yaml* ./

# Install pnpm and dependencies
RUN npm install -g pnpm@9.0.0 && \
    pnpm install --ignore-scripts

# ============================================
# Stage 2: Build Next.js application (production)
# ============================================
FROM node:22-alpine AS builder

WORKDIR /app

RUN npm config set registry https://registry.npmmirror.com && \
    npm install -g pnpm@9.0.0

# Copy dependencies from deps stage
COPY --from=deps /app/node_modules ./node_modules
COPY . .

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN pnpm install --ignore-scripts && \
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
RUN npm config set registry https://registry.npmmirror.com && \
    npm install -g pnpm@9.0.0

# Copy everything from deps (node_modules ready) + source
COPY --from=deps /app/node_modules ./node_modules
COPY . .

RUN mkdir -p .next && chmod 777 .next

ENV NODE_ENV=development
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"
ENV ITSM_BACKEND_URL=http://host.docker.internal:8090
ENV NEXT_PUBLIC_API_URL=http://host.docker.internal:8090

EXPOSE 3000

CMD ["npx", "next", "dev"]
