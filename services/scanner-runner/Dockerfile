# Scanner Runner - TypeScript/Playwright based scanner
# Uses multi-stage build for smaller runtime image

FROM mcr.microsoft.com/playwright:v1.57.0-jammy AS builder
WORKDIR /repo

# Install bun (pinned)
RUN apt-get update && apt-get install -y curl unzip && rm -rf /var/lib/apt/lists/*
RUN curl -fsSL https://bun.sh/install | bash -s "bun-v1.3.0"
ENV PATH="/root/.bun/bin:$PATH"

# Shared TypeScript base config used by all packages
COPY tsconfig.strict.json /repo/tsconfig.strict.json

# --- Build contracts package first (types only, already generated) ---
WORKDIR /repo/libs/contracts/report
COPY libs/contracts/report/package.json libs/contracts/report/bun.lock* ./
RUN bun install --frozen-lockfile --ignore-scripts || bun install --ignore-scripts
COPY libs/contracts/report/tsconfig.json ./tsconfig.json
COPY libs/contracts/report/generated ./generated
COPY libs/contracts/report/schema ./schema
RUN bunx tsc

# --- Build scanner-runner ---
WORKDIR /repo/services/scanner-runner
COPY libs/contracts/scanner-manifest /repo/libs/contracts/scanner-manifest
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
  && mkdir -p node_modules/@stageflow \
  && cp -R /repo/libs/contracts/scanner-manifest node_modules/@stageflow/contracts-scanner-manifest

# Runtime stage
FROM mcr.microsoft.com/playwright:v1.57.0-jammy
WORKDIR /app

ENV NODE_ENV=production \
    PORT=8080 \
    SCANNER_DATA_DIR=/data

# Create data directory with proper permissions
RUN mkdir -p /data && chown pwuser:pwuser /data

# Copy only production dependencies and built code
COPY --from=builder --chown=pwuser:pwuser /repo/services/scanner-runner/node_modules ./node_modules
COPY --from=builder --chown=pwuser:pwuser /repo/services/scanner-runner/dist ./dist
COPY --from=builder --chown=pwuser:pwuser /repo/services/scanner-runner/package.json ./

USER pwuser

VOLUME ["/data"]
EXPOSE 8080

CMD ["node", "dist/index.js"]
