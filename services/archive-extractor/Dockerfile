# syntax=docker/dockerfile:1

FROM docker.io/library/golang:1.26-alpine@sha256:80fbb8f9b2fa541a7d34378f1ad10f4f1c433817c4ed39ddb3e2f3ec3e961271 AS builder
WORKDIR /src

# Copy module definitions first for better layer caching
COPY libs/go ./libs/go/
COPY services/archive-extractor/go.mod services/archive-extractor/go.sum ./services/archive-extractor/

# Download dependencies (cached layer)
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    cd services/archive-extractor && GOWORK=off go mod download

# Copy source code
COPY libs/go ./libs/go
COPY services/archive-extractor ./services/archive-extractor

# Build static binary (no CGO needed)
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    mkdir -p /go/pkg/mod /root/.cache/go-build && \
    cd services/archive-extractor && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    GOWORK=off go build -ldflags="-s -w" -o /out/extractor ./cmd/server

# Runtime stage
FROM docker.io/library/alpine:3.20@sha256:a4f4213abb84c497377b8544c81b3564f313746700372ec4fe84653e4fb03805
RUN apk add --no-cache ca-certificates
RUN addgroup -S extractor && adduser -S extractor -G extractor
COPY --from=builder /out/extractor /usr/local/bin/extractor
USER extractor
ENTRYPOINT ["/usr/local/bin/extractor"]
