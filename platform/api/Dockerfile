# syntax=docker/dockerfile:1

FROM docker.io/library/golang:1.25 AS builder
WORKDIR /src

# Copy go.mod/go.sum first for better layer caching
COPY packages/contracts/report/generated/go/go.mod ./packages/contracts/report/generated/go/
COPY packages/contracts/scanner-manifest/go.mod packages/contracts/scanner-manifest/go.sum ./packages/contracts/scanner-manifest/
COPY packages/shared-go/go.mod packages/shared-go/go.sum ./packages/shared-go/
COPY platform/api/go.mod platform/api/go.sum ./platform/api/

# Download dependencies (cached layer)
RUN --mount=type=cache,target=/go/pkg/mod \
    cd platform/api && go mod download

# Copy source code
COPY packages/contracts/report/generated/go ./packages/contracts/report/generated/go
COPY packages/contracts/scanner-manifest ./packages/contracts/scanner-manifest
COPY packages/shared-go ./packages/shared-go
COPY platform/api ./platform/api

# Build with optimizations
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd platform/api && \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/platform-api ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/platform-api-healthcheck ./cmd/healthcheck

# Prep stage - create directories with correct ownership
FROM docker.io/library/debian:bookworm-slim AS prep
RUN mkdir -p /data && chown 65532:65532 /data

# Runtime stage - distroless with static busybox for healthcheck shell support
FROM gcr.io/distroless/base-debian12:nonroot

# Copy busybox for shell support (needed for podman-compose healthchecks)
COPY --from=docker.io/library/busybox:1.36-musl /bin/busybox /bin/sh

# Copy binaries
COPY --from=builder /out/platform-api /usr/local/bin/platform-api
COPY --from=builder /out/platform-api-healthcheck /usr/local/bin/platform-api-healthcheck

# Copy data directory with correct ownership for nonroot user (uid 65532)
COPY --from=prep --chown=65532:65532 /data /data

USER 65532:65532
WORKDIR /data

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/platform-api"]
