# Scanner Runner - TypeScript/Playwright based scanner
# Multi-stage build. The runtime image ships Node, the production node_modules
# and Chromium only: StageFlow runs every scan on Chromium.

# Global ARGs: must precede the first FROM to be visible to every stage that
# redeclares them. Keep NODE_MAJOR in step with the node tag below and
# engines.node in package.json. Keep PLAYWRIGHT_VERSION equal to the pinned
# playwright dependency in package.json; the runtime stage asserts it.
ARG NODE_MAJOR=24
ARG PLAYWRIGHT_VERSION=1.63.0

FROM docker.io/oven/bun:1.3.8@sha256:371d30538b69303ced927bb5915697ac7e2fa8cb409ee332c66009de64de5aa3 AS bun-bin

FROM docker.io/library/node:24.18.0-bookworm-slim AS builder
WORKDIR /repo

# Install bun from pinned image stage (no curl|bash installer)
COPY --from=bun-bin /usr/local/bin/bun /usr/local/bin/bun
ARG NODE_MAJOR
RUN node --version | grep -E "^v${NODE_MAJOR}\."

# Shared TypeScript base config used by all packages
COPY tsconfig.strict.json /repo/tsconfig.strict.json

# --- Build contracts package first (types only, already generated) ---
WORKDIR /repo/libs/contracts/report
COPY libs/contracts/report/package.json libs/contracts/report/bun.lock* ./
RUN bun install --frozen-lockfile --ignore-scripts
COPY libs/contracts/report/tsconfig.json ./tsconfig.json
COPY libs/contracts/report/schema ./schema
COPY libs/contracts/report/generated/typescript ./generated/typescript
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
COPY libs/contracts/scanner-manifest/schema /repo/libs/contracts/scanner-manifest/schema
RUN bun run build:prepared
RUN rm -rf node_modules \
  && bun install --frozen-lockfile --production --ignore-scripts \
  && rm -rf node_modules/@stageflow/contracts-scanner-manifest \
  && rm -rf node_modules/@stageflow/contracts-provenance \
  && mkdir -p node_modules/@stageflow \
  && cp -R /repo/libs/contracts/scanner-manifest node_modules/@stageflow/contracts-scanner-manifest \
  && cp -R /repo/libs/contracts/provenance node_modules/@stageflow/contracts-provenance

# Runtime stage: Node plus Chromium and its system libraries, nothing else.
FROM docker.io/library/node:24.18.0-bookworm-slim
WORKDIR /app

ENV NODE_ENV=production \
    PORT=8080 \
    SCANNER_DATA_DIR=/data \
    PLAYWRIGHT_BROWSERS_PATH=/ms-playwright

ARG NODE_MAJOR
ARG PLAYWRIGHT_VERSION

# Chromium and its libraries come from Playwright's own installer, pinned to
# the same version as the playwright dependency so the browser build matches
# what the code loads. This layer changes only when that version does. npm is
# removed afterwards: the runtime never needs it.
RUN apt-get update \
  && DEBIAN_FRONTEND=noninteractive apt-get upgrade -y \
  && npx --yes "playwright@${PLAYWRIGHT_VERSION}" install --with-deps chromium \
  && chmod -R a+rX /ms-playwright \
  && rm -rf /var/lib/apt/lists/* /root/.npm /root/.cache \
  && rm -rf /usr/local/lib/node_modules/npm /usr/local/bin/npm /usr/local/bin/npx \
  && node --version | grep -E "^v${NODE_MAJOR}\."

# The image's own user. Job containers set their uid at create time; the image
# default only keeps the files owned by a non-root user.
RUN groupadd --gid 65532 pwuser \
  && useradd --uid 65532 --gid 65532 --create-home --shell /usr/sbin/nologin pwuser \
  && mkdir -p /data && chown pwuser:pwuser /data

# Copy only production dependencies and built code
COPY --from=builder --chown=pwuser:pwuser /repo/services/scanner-runner/node_modules ./node_modules
COPY --from=builder --chown=pwuser:pwuser /repo/services/scanner-runner/dist ./dist
COPY --from=builder --chown=pwuser:pwuser /repo/services/scanner-runner/package.json ./

# The installed browser must match the playwright package the code loads.
RUN test "$(node -p "require('./node_modules/playwright/package.json').version")" = "${PLAYWRIGHT_VERSION}"

# glibc gives each thread its own malloc arena, and freed screenshot buffers
# fragment them: Node's resident memory grew about 5 MB per page while its
# heap stayed flat. Two arenas keep it level. Set here, after the browser
# layer, so that layer stays cached.
ENV MALLOC_ARENA_MAX=2

USER pwuser

VOLUME ["/data"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD node dist/healthcheck.js

CMD ["node", "dist/index.js"]
