# syntax=docker/dockerfile:1

FROM docker.io/library/golang:1.26.6 AS builder
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
COPY services/platform-api ./services/platform-api

# Build with optimizations
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    mkdir -p /go/pkg/mod /root/.cache/go-build && \
    CGO_ENABLED=1 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build -ldflags="-s -w" -o /out/platform-api ./services/platform-api/cmd/server && \
    CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build -ldflags="-s -w" -o /out/platform-api-healthcheck ./services/platform-api/cmd/healthcheck

# Prep stage - create directories with correct ownership
FROM docker.io/library/debian:bookworm-slim AS prep
RUN mkdir -p /data && chown 65532:65532 /data

# Runtime stage
# Use debian-slim instead of distroless because podman-compose materializes
# healthchecks through /bin/sh -c, and distroless images do not provide /bin/sh.
FROM docker.io/library/debian:bookworm-slim

# ca-certificates: page discovery fetches HTTPS sites; without a CA bundle every
# fetch fails certificate verification and discovery returns only the start URL.
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get upgrade -y \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy binaries
COPY --from=builder /out/platform-api /usr/local/bin/platform-api
COPY --from=builder /out/platform-api-healthcheck /usr/local/bin/platform-api-healthcheck

# Copy data directory with correct ownership for nonroot user (uid 65532)
COPY --from=prep --chown=65532:65532 /data /data

USER 65532:65532
WORKDIR /data

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/platform-api"]
