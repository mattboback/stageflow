# syntax=docker/dockerfile:1

FROM docker.io/library/golang:1.25 AS builder
WORKDIR /src

# Copy go.mod/go.sum first for better layer caching
COPY packages/contracts/report/generated/go/go.mod ./packages/contracts/report/generated/go/
COPY packages/contracts/scanner-manifest/go.mod packages/contracts/scanner-manifest/go.sum ./packages/contracts/scanner-manifest/
COPY packages/shared-go/go.mod packages/shared-go/go.sum ./packages/shared-go/
COPY platform/orchestrator/go.mod platform/orchestrator/go.sum ./platform/orchestrator/

# Download dependencies (cached layer)
RUN --mount=type=cache,target=/go/pkg/mod \
    cd platform/orchestrator && go mod download

# Copy source code
COPY packages/contracts/report/generated/go ./packages/contracts/report/generated/go
COPY packages/contracts/scanner-manifest ./packages/contracts/scanner-manifest
COPY packages/shared-go ./packages/shared-go
COPY platform/orchestrator ./platform/orchestrator

# Build with optimizations
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd platform/orchestrator && \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/orchestrator ./cmd/orchestrator

# Runtime stage - distroless with debug for troubleshooting
FROM gcr.io/distroless/base-debian12:debug
COPY --from=builder /out/orchestrator /usr/local/bin/orchestrator

# Create /bin/sh symlink for healthchecks
RUN ["/busybox/ln", "-s", "/busybox/sh", "/bin/sh"]

# Note: Podman socket is bind-mounted at runtime via Quadlet
ENTRYPOINT ["/usr/local/bin/orchestrator"]
