# ============================================
# Stage 1: Dependencies Installation
# ============================================
FROM node:22-alpine AS deps

WORKDIR /app

# Clear proxy settings that may be inherited from host
# Use China mirror (npmmirror) for faster downloads
RUN unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy ALL_PROXY all_proxy && \
    npm config delete proxy && \
    npm config delete https-proxy && \
    npm config set registry https://registry.npmmirror.com

# Copy package files first for caching
COPY package.json package-lock.json* ./

# Install dependencies with pnpm (faster and more reliable than npm)
# Must unset proxy before npm install since npm caches proxy settings
RUN unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy ALL_PROXY all_proxy && \
    npm install -g pnpm@9.0.0 && \
    pnpm install --ignore-scripts

# ============================================
# Stage 2: Build Next.js application
# ============================================
FROM node:22-alpine AS builder

WORKDIR /app

# Clear proxy settings and install pnpm
# Use China mirror for faster downloads
RUN unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy ALL_PROXY all_proxy && \
    npm config delete proxy && \
    npm config delete https-proxy && \
    npm config set registry https://registry.npmmirror.com && \
    npm install -g pnpm@9.0.0

# Copy dependencies from deps stage
COPY --from=deps /app/node_modules ./node_modules
COPY . .

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Build Next.js application using pnpm
RUN unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy ALL_PROXY all_proxy && \
    pnpm install --ignore-scripts && \
    pnpm run build

# ============================================
# Stage 3: Run Next.js application
# ============================================
FROM node:22-alpine AS runner

WORKDIR /app

# Add non-root user for security
RUN addgroup -g 1001 -S nodejs && \
    adduser -S nextjs -u 1001

# Copy production assets
COPY --from=builder --chown=nextjs:nodejs /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

# Ensure .next directory exists with correct permissions
RUN mkdir -p .next && chown nextjs:nodejs .next

# Switch to non-root user
USER nextjs

ENV NODE_ENV=production
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"
ENV ITSM_BACKEND_URL=http://host.docker.internal:8090
ENV NEXT_PUBLIC_API_URL=http://host.docker.internal:8090

EXPOSE 3000

CMD ["node", "server.js"]