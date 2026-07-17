# syntax=docker/dockerfile:1

FROM docker.io/library/golang:1.26.5 AS builder
ARG TARGETOS=linux
ARG TARGETARCH=amd64
WORKDIR /src

# Generate schema contracts first; generated outputs are intentionally untracked.
COPY libs/contracts/report/schema ./libs/contracts/report/schema
RUN mkdir -p libs/contracts/report/generated/go \
    && go run github.com/atombender/go-jsonschema@v0.20.0 --package report --tags json --struct-name-from-title \
      libs/contracts/report/schema/unified-report.v2.schema.json > libs/contracts/report/generated/go/report_schema.go \
    && gofmt -w libs/contracts/report/generated/go/report_schema.go \
    && printf "module github.com/mattboback/stageflow/libs/contracts/report/generated/go\n\ngo 1.26.5\n" \
      > libs/contracts/report/generated/go/go.mod

COPY libs/contracts/scanner-manifest/go.mod libs/contracts/scanner-manifest/go.sum ./libs/contracts/scanner-manifest/
COPY libs/contracts/scanner-manifest/schema ./libs/contracts/scanner-manifest/schema
RUN cd libs/contracts/scanner-manifest \
    && go run github.com/atombender/go-jsonschema@v0.20.0 --package scannermanifest --tags json --struct-name-from-title \
      schema/scanner-manifest.schema.json > scanner_manifest.go \
    && sed -i.bak \
      -e 's|^type ManifestConfigSchema interface{}$|// JSON Schema for SCANNER_OPTIONS validation (see ManifestConfigSchema definition\n// in schema/scanner-manifest.schema.json). Carried as json.RawMessage so the\n// Go side can forward the original bytes to the runtime validator.\ntype ManifestConfigSchema = json.RawMessage|' \
      -e 's|^type ScannerManifestConfigSchema interface{}$|// Embedded JSON Schema for the scanner-level configSchema field. Same\n// rationale as ManifestConfigSchema: round-tripped as raw JSON.\ntype ScannerManifestConfigSchema = json.RawMessage|' \
      scanner_manifest.go \
    && rm -f scanner_manifest.go.bak \
    && gofmt -w scanner_manifest.go

# Copy module definitions first for better layer caching
COPY libs/go ./libs/go/
COPY services/orchestrator/go.mod services/orchestrator/go.sum ./services/orchestrator/

# Download dependencies (cached layer)
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    cd services/orchestrator && GOWORK=off go mod download

# Copy source code
COPY libs/contracts/scanner-manifest ./libs/contracts/scanner-manifest
COPY libs/go ./libs/go
COPY services/orchestrator ./services/orchestrator

# Build with optimizations
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    mkdir -p /go/pkg/mod /root/.cache/go-build && \
    cd services/orchestrator && \
    CGO_ENABLED=1 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    GOWORK=off go build -ldflags="-s -w" -o /out/orchestrator ./cmd/orchestrator && \
    CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    GOWORK=off go build -ldflags="-s -w" -o /out/orchestrator-healthcheck ./cmd/healthcheck

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
