# syntax=docker/dockerfile:1

FROM docker.io/library/golang:1.26.9 AS builder
ARG TARGETOS=linux
ARG TARGETARCH=amd64
WORKDIR /src

# One module for the repo: download its dependencies first (cached layer).
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    go mod download

# The packages this service imports. Contract code is committed, not generated here.
COPY libs/contracts/report/generated/go ./libs/contracts/report/generated/go
COPY libs/contracts/scanner-manifest ./libs/contracts/scanner-manifest
COPY libs/go ./libs/go
COPY internal ./internal
COPY services/orchestrator ./services/orchestrator

# Build with optimizations
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    mkdir -p /go/pkg/mod /root/.cache/go-build && \
    CGO_ENABLED=1 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build -ldflags="-s -w" -o /out/orchestrator ./services/orchestrator/cmd/orchestrator && \
    CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build -ldflags="-s -w" -o /out/orchestrator-healthcheck ./services/orchestrator/cmd/healthcheck

# Runtime stage
# Use debian-slim instead of distroless because podman-compose materializes
# healthchecks through /bin/sh -c, and distroless images do not provide /bin/sh.
FROM docker.io/library/debian:bookworm-slim

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get upgrade -y \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/orchestrator /usr/local/bin/orchestrator
COPY --from=builder /out/orchestrator-healthcheck /usr/local/bin/orchestrator-healthcheck

USER 65532:65532

# Note: Podman socket is bind-mounted at runtime via Quadlet
ENTRYPOINT ["/usr/local/bin/orchestrator"]
