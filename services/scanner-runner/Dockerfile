# Scanner Runner - TypeScript/Playwright based scanner
# Uses multi-stage build for smaller runtime image

FROM docker.io/oven/bun:1.3.8@sha256:371d30538b69303ced927bb5915697ac7e2fa8cb409ee332c66009de64de5aa3 AS bun-bin
FROM docker.io/library/node:22.22.2-bookworm-slim AS node-bin

FROM mcr.microsoft.com/playwright:v1.60.0-noble AS builder
WORKDIR /repo

# Install bun from pinned image stage (no curl|bash installer)
COPY --from=bun-bin /usr/local/bin/bun /usr/local/bin/bun
# Keep the scanner runtime on the same Node major declared in package metadata.
COPY --from=node-bin /usr/local/bin/node /usr/local/bin/node
RUN test "$(command -v node)" = "/usr/local/bin/node" && node --version | grep -E '^v22\.'

# Shared TypeScript base config used by all packages
COPY tsconfig.strict.json /repo/tsconfig.strict.json

# --- Build contracts package first (types only, already generated) ---
WORKDIR /repo/libs/contracts/report
COPY libs/contracts/report/package.json libs/contracts/report/bun.lock* ./
RUN bun install --frozen-lockfile --ignore-scripts || bun install --ignore-scripts
COPY libs/contracts/report/tsconfig.json ./tsconfig.json
COPY libs/contracts/report/generated ./generated
COPY libs/contracts/report/schema ./schema
RUN bun x tsc

# --- Build scanner-runner ---
WORKDIR /repo/services/scanner-runner
COPY libs/contracts/scanner-manifest /repo/libs/contracts/scanner-manifest
COPY libs/contracts/provenance /repo/libs/contracts/provenance
COPY services/scanner-runner/package.json services/scanner-runner/bun.lock ./
RUN bun install --frozen-lockfile --ignore-scripts
COPY services/scanner-runner/tsconfig.json ./tsconfig.json
COPY services/scanner-runner/tsconfig.build.json ./tsconfig.build.json
COPY services/scanner-runner/src ./src
COPY services/scanner-runner/scripts ./scripts
COPY libs/go/scannercatalog/manifests /repo/libs/go/scannercatalog/manifests
COPY libs/contracts/scanner-manifest/generated /repo/libs/contracts/scanner-manifest/generated
COPY libs/contracts/scanner-manifest/schema /repo/libs/contracts/scanner-manifest/schema
RUN bun run build
RUN rm -rf node_modules \
  && bun install --frozen-lockfile --production --ignore-scripts \
  && rm -rf node_modules/@stageflow/contracts-scanner-manifest \
  && rm -rf node_modules/@stageflow/contracts-provenance \
  && mkdir -p node_modules/@stageflow \
  && cp -R /repo/libs/contracts/scanner-manifest node_modules/@stageflow/contracts-scanner-manifest \
  && cp -R /repo/libs/contracts/provenance node_modules/@stageflow/contracts-provenance

# Runtime stage
FROM mcr.microsoft.com/playwright:v1.60.0-noble
WORKDIR /app

ENV NODE_ENV=production \
    PORT=8080 \
    SCANNER_DATA_DIR=/data

# NOTE: gstreamer1.0-plugins-bad / libgstreamer-plugins-bad1.0-0 are intentionally
# kept. The WebKit engine's Linux media stack dlopen()s them at launch — purging
# them (a past slimming/hardening step) makes `browser=webkit` scans fail with
# "Host system is missing dependencies to run browsers". Chromium/Firefox don't
# need them, but we expose WebKit as a selectable engine, so they must stay.
RUN apt-get update \
  && DEBIAN_FRONTEND=noninteractive apt-get upgrade -y \
  && rm -rf /var/lib/apt/lists/* /usr/lib/node_modules/npm /usr/bin/node /usr/bin/npm /usr/bin/npx

COPY --from=node-bin /usr/local/bin/node /usr/local/bin/node

RUN test "$(command -v node)" = "/usr/local/bin/node" && node --version | grep -E '^v22\.'

# Create data directory with proper permissions
RUN mkdir -p /data && chown pwuser:pwuser /data

# Copy only production dependencies and built code
COPY --from=builder --chown=pwuser:pwuser /repo/services/scanner-runner/node_modules ./node_modules
COPY --from=builder --chown=pwuser:pwuser /repo/services/scanner-runner/dist ./dist
COPY --from=builder --chown=pwuser:pwuser /repo/services/scanner-runner/package.json ./

USER pwuser

VOLUME ["/data"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD node dist/healthcheck.js

CMD ["node", "dist/index.js"]
