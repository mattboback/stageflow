# syntax=docker/dockerfile:1

FROM docker.io/library/golang:1.26 AS builder
WORKDIR /src

# Copy module definitions first for better layer caching
COPY libs/contracts/report/generated/go/go.mod ./libs/contracts/report/generated/go/
COPY libs/contracts/scanner-manifest/go.mod libs/contracts/scanner-manifest/go.sum ./libs/contracts/scanner-manifest/
COPY libs/go ./libs/go/
COPY services/platform-api/go.mod services/platform-api/go.sum ./services/platform-api/

# Download dependencies (cached layer)
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    cd services/platform-api && GOWORK=off go mod download

# Copy source code
COPY libs/contracts/report/generated/go ./libs/contracts/report/generated/go
COPY libs/contracts/scanner-manifest ./libs/contracts/scanner-manifest
COPY libs/go ./libs/go
COPY services/platform-api ./services/platform-api

# Build with optimizations
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    mkdir -p /go/pkg/mod /root/.cache/go-build && \
    cd services/platform-api && \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    GOWORK=off go build -ldflags="-s -w" -o /out/platform-api ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    GOWORK=off go build -ldflags="-s -w" -o /out/platform-api-healthcheck ./cmd/healthcheck

# Prep stage - create directories with correct ownership
FROM docker.io/library/debian:bookworm-slim AS prep
RUN mkdir -p /data && chown 65532:65532 /data

# Runtime stage
# Use debian-slim instead of distroless because podman-compose materializes
# healthchecks through /bin/sh -c, and distroless images do not provide /bin/sh.
FROM docker.io/library/debian:bookworm-slim

# Copy binaries
COPY --from=builder /out/platform-api /usr/local/bin/platform-api
COPY --from=builder /out/platform-api-healthcheck /usr/local/bin/platform-api-healthcheck

# Copy data directory with correct ownership for nonroot user (uid 65532)
COPY --from=prep --chown=65532:65532 /data /data

USER 65532:65532
WORKDIR /data

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/platform-api"]
