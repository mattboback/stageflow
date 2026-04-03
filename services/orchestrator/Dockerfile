# syntax=docker/dockerfile:1

FROM docker.io/library/golang:1.26 AS builder
WORKDIR /src

# Copy module definitions first for better layer caching
COPY libs/contracts/report/generated/go/go.mod ./libs/contracts/report/generated/go/
COPY libs/contracts/scanner-manifest/go.mod libs/contracts/scanner-manifest/go.sum ./libs/contracts/scanner-manifest/
COPY libs/go ./libs/go/
COPY services/orchestrator/go.mod services/orchestrator/go.sum ./services/orchestrator/

# Download dependencies (cached layer)
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    cd services/orchestrator && GOWORK=off go mod download

# Copy source code
COPY libs/contracts/report/generated/go ./libs/contracts/report/generated/go
COPY libs/contracts/scanner-manifest ./libs/contracts/scanner-manifest
COPY libs/go ./libs/go
COPY services/orchestrator ./services/orchestrator

# Build with optimizations
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    mkdir -p /go/pkg/mod /root/.cache/go-build && \
    cd services/orchestrator && \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    GOWORK=off go build -ldflags="-s -w" -o /out/orchestrator ./cmd/orchestrator && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    GOWORK=off go build -ldflags="-s -w" -o /out/orchestrator-healthcheck ./cmd/healthcheck

# Runtime stage - distroless
FROM gcr.io/distroless/base-debian12:nonroot

COPY --from=builder /out/orchestrator /usr/local/bin/orchestrator
COPY --from=builder /out/orchestrator-healthcheck /usr/local/bin/orchestrator-healthcheck

USER 65532:65532

# Note: Podman socket is bind-mounted at runtime via Quadlet
ENTRYPOINT ["/usr/local/bin/orchestrator"]
