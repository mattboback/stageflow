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
    && curl -fsSL https://bun.sh/install | bash -s "bun-v1.3.10" \
    && ln -s /root/.bun/bin/bun /usr/local/bin/bun \
    && rm -rf /var/lib/apt/lists/*

# Copy package files first for better caching
COPY clients/web-react/package.json clients/web-react/bun.lock ./

# Install dependencies
RUN bun install --frozen-lockfile

# Copy the required monorepo libs contracts
COPY libs/contracts /libs/contracts

# Copy React source code and configurations
COPY clients/web-react/eslint.config.js clients/web-react/vite.config.ts clients/web-react/react-router.config.ts clients/web-react/tsconfig.app.json clients/web-react/tsconfig.json clients/web-react/tsconfig.node.json ./
COPY clients/web-react/app ./app
COPY clients/web-react/public ./public

# Build arguments for environment variables.
# These are baked into the static build at image build-time.
ARG VITE_API_URL
ARG VITE_SITE_TITLE
ARG VITE_SITE_URL
ARG VITE_GITHUB_URL
ARG VITE_TAGLINE

ENV VITE_API_URL=${VITE_API_URL} \
    VITE_SITE_TITLE=${VITE_SITE_TITLE} \
    VITE_SITE_URL=${VITE_SITE_URL} \
    VITE_GITHUB_URL=${VITE_GITHUB_URL} \
    VITE_TAGLINE=${VITE_TAGLINE}

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
COPY clients/web-react/nginx.conf /etc/nginx/conf.d/default.conf

# Copy built assets from builder (React Router SPA output)
COPY --from=builder /app/build/client /usr/share/nginx/html

# Expose port
EXPOSE 3020

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:3020/ || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
