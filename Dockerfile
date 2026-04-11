# Build Stage
FROM golang:alpine AS builder

WORKDIR /app

# Copy go.mod (and go.sum if it exists later) and source code
COPY go.mod ./
COPY . .

# Set environment variables for a clean, static build
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Build the application
RUN go build -o trading-bot ./cmd/bot


# Final Minimal Stage
FROM alpine:latest

WORKDIR /app

# Install root certificates for HTTPS requests (Required for Binance and Telegram APIs)
# Install tzdata just in case Go needs correct timezone handling for signal logging
RUN apk --no-cache add ca-certificates tzdata

# Copy the compiled binary from the builder stage
COPY --from=builder /app/trading-bot /app/trading-bot

# Start the bot
ENTRYPOINT ["/app/trading-bot"]
