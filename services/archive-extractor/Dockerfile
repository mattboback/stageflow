# syntax=docker/dockerfile:1

FROM docker.io/library/golang:1.26.9-alpine AS builder
ARG TARGETOS=linux
ARG TARGETARCH=amd64
WORKDIR /src

# One module for the repo: download its dependencies first (cached layer).
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    go mod download

# The packages this service imports. Contract code is committed, not generated here.
COPY libs/go ./libs/go
COPY internal/archive ./internal/archive
COPY services/archive-extractor ./services/archive-extractor

# Build static binary (no CGO needed)
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    mkdir -p /go/pkg/mod /root/.cache/go-build && \
    CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build -ldflags="-s -w" -o /out/extractor ./services/archive-extractor/cmd/server

# Runtime stage
FROM docker.io/library/alpine:3.22
RUN apk add --no-cache ca-certificates && apk upgrade --no-cache
RUN addgroup -S extractor && adduser -S extractor -G extractor
COPY --from=builder /out/extractor /usr/local/bin/extractor
USER extractor
ENTRYPOINT ["/usr/local/bin/extractor"]
