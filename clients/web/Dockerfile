# Multi-stage Dockerfile for React Frontend
# Optimized for production with nginx

# Stage 1: Build
# Use Node base so the react-router build CLI (#!/usr/bin/env node) runs under
# real Node — the `oven/bun` image ships a `node` shim that proxies to bun,
# which then picks the `bun` export condition for react-dom/server and resolves
# to server.bun.js, which does not export renderToPipeableStream and breaks the
# react-router SSR/prerender pass.
FROM docker.io/library/node:22-bookworm-slim AS builder
WORKDIR /app

# Install bun for the frozen-lockfile install (bun.lock is bun-specific)
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl unzip \
    && curl -fsSL https://bun.sh/install | bash -s "bun-v1.3.8" \
    && ln -s /root/.bun/bin/bun /usr/local/bin/bun \
    && rm -rf /var/lib/apt/lists/*

# Copy package files first for better caching
COPY clients/web/package.json clients/web/bun.lock ./

# Install dependencies
RUN bun install --frozen-lockfile

# Copy the required monorepo libs contracts
COPY libs/contracts /libs/contracts
RUN mkdir -p /libs/contracts/report/generated/typescript \
    && bun x json-schema-to-typescript /libs/contracts/report/schema/unified-report.v2.schema.json \
      -o /libs/contracts/report/generated/typescript/unified-report.v2.ts \
      --unreachableDefinitions

# Copy React source code and configurations
COPY clients/web/eslint.config.js clients/web/vite.config.ts clients/web/react-router.config.ts clients/web/tsconfig.app.json clients/web/tsconfig.json clients/web/tsconfig.node.json ./
COPY clients/web/app ./app
COPY clients/web/public ./public

# Build arguments for environment variables.
# These are baked into the static build at image build-time.
ARG VITE_API_URL
ARG VITE_SITE_TITLE
ARG VITE_SITE_URL
ARG VITE_GITHUB_URL
ARG VITE_TAGLINE
ARG VITE_AI_NAVIGATOR_DEFAULT_MODEL
ARG VITE_DEPLOYMENT_MODE

ENV VITE_API_URL=${VITE_API_URL} \
    VITE_SITE_TITLE=${VITE_SITE_TITLE} \
    VITE_SITE_URL=${VITE_SITE_URL} \
    VITE_GITHUB_URL=${VITE_GITHUB_URL} \
    VITE_TAGLINE=${VITE_TAGLINE} \
    VITE_AI_NAVIGATOR_DEFAULT_MODEL=${VITE_AI_NAVIGATOR_DEFAULT_MODEL} \
    VITE_DEPLOYMENT_MODE=${VITE_DEPLOYMENT_MODE}

# Build the application via node (not bun) so react-dom's `node` export
# condition is used and renderToPipeableStream resolves correctly.
RUN node ./node_modules/.bin/react-router build

# Stage 2: Production server
FROM docker.io/library/nginx:1.29-alpine AS runner

# Pull in security upgrades
RUN apk upgrade --no-cache

# Remove default nginx content and config
RUN rm -rf /usr/share/nginx/html/* /etc/nginx/conf.d/default.conf

# Copy the repo-owned nginx config
COPY clients/web/nginx.conf /etc/nginx/conf.d/default.conf

# Copy built assets from builder (React Router SPA output)
COPY --from=builder /app/build/client /usr/share/nginx/html

# Expose port
EXPOSE 3020

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:3020/ || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
