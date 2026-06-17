# Multi-stage Dockerfile for React Frontend
# Optimized for production with nginx

# Stage 1: Build
FROM docker.io/oven/bun:1.3.8-alpine AS builder
WORKDIR /app

# Copy package files first for better caching
COPY clients/web-react/package.json clients/web-react/bun.lock ./

# Install dependencies
RUN bun install --frozen-lockfile

# Copy the required monorepo libs contracts
COPY libs/contracts /libs/contracts

# Copy React source code and configurations
COPY clients/web-react/eslint.config.js clients/web-react/vite.config.ts clients/web-react/tsconfig.app.json clients/web-react/tsconfig.json clients/web-react/tsconfig.node.json ./
COPY clients/web-react/src ./src
COPY clients/web-react/public ./public
COPY clients/web-react/index.html ./

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

# Build the application
RUN bun run build

# Stage 2: Production server
FROM docker.io/library/nginx:1.29-alpine AS runner

# Pull in security upgrades
RUN apk upgrade --no-cache

# Remove default nginx content and config
RUN rm -rf /usr/share/nginx/html/* /etc/nginx/conf.d/default.conf

# Copy the repo-owned nginx config
COPY clients/web-react/nginx.conf /etc/nginx/conf.d/default.conf

# Copy built assets from builder (Vite outputs to dist/)
COPY --from=builder /app/dist /usr/share/nginx/html

# Expose port
EXPOSE 3020

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:3020/ || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
