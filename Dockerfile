# Build Stage — pinned to the exact Go version declared in go.mod
# (go 1.25.1) so deps build deterministically. Unpinned "golang:alpine"
# tracks the latest Go and can shift dependency builds between pipeline runs.
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod + go.sum first, then fetch dependencies as a cached layer.
# This layer is only rebuilt when go.mod/go.sum change, not on every commit.
COPY go.mod go.sum ./
RUN go mod download

# Now copy the rest of the source (kept small by .dockerignore)
COPY . .

# Static, reproducible build
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Build both binaries
RUN go build -o trading-bot ./cmd/bot \
 && go build -o trading-mcp ./cmd/mcp


# Final Minimal Stage
FROM alpine:latest

WORKDIR /app

# Root certificates for HTTPS (Binance / Telegram / Mongo TLS) and timezone data
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/trading-bot /app/trading-bot
COPY --from=builder /app/trading-mcp /app/trading-mcp

# Default entrypoint is the alert bot; the MCP container overrides it
# with /app/trading-mcp on the command line (see .gitlab-ci.yml deploy_mcp).
ENTRYPOINT ["/app/trading-bot"]
