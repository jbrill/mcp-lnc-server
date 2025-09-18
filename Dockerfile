# syntax=docker/dockerfile:1.7

# Build stage
FROM golang:1.24.5-bookworm AS build
WORKDIR /src

# Leverage go module cache before copying the full source tree
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy the remainder of the source
COPY . .

# Build the MCP LNC server binary with CGO enabled for lnd deps
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GO111MODULE=on go build -o /out/mcp-lnc-server .

# Runtime stage
FROM debian:bookworm-slim AS runtime

# Install CA certificates for TLS connections
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /out/mcp-lnc-server /app/mcp-lnc-server

# Ensure stdout/stderr are unbuffered for MCP stdio operation
ENV LANG=C.UTF-8 \
    LNC_ALLOW_MUTATING_TOOLS=false

ENTRYPOINT ["/app/mcp-lnc-server"]
