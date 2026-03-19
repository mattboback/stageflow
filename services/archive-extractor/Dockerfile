# syntax=docker/dockerfile:1

FROM docker.io/library/golang:1.25-alpine AS builder
WORKDIR /src

COPY go.work go.work.sum ./

# Copy module definitions first for better layer caching
COPY libs/go ./libs/go/
COPY services/archive-extractor/go.mod services/archive-extractor/go.sum ./services/archive-extractor/

# Download dependencies (cached layer)
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    cd services/archive-extractor && go mod download

# Copy source code
COPY libs/go ./libs/go
COPY services/archive-extractor ./services/archive-extractor

# Build static binary (no CGO needed)
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    mkdir -p /go/pkg/mod /root/.cache/go-build && \
    cd services/archive-extractor && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/extractor ./cmd/server

# Runtime stage
FROM docker.io/library/alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/extractor /usr/local/bin/extractor
ENTRYPOINT ["/usr/local/bin/extractor"]
