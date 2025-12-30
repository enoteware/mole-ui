# Mole Web UI Dockerfile
#
# NOTE: For full macOS system metrics and cleaning capabilities,
# running directly on the Mac Mini is recommended (see docker-compose.yml).
# This container provides the web UI with host filesystem access.

FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY cmd/ cmd/

# Build the server
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /mole-web ./cmd/web

# Final image
FROM alpine:3.19

RUN apk add --no-cache bash ca-certificates tzdata jq bc procps

WORKDIR /app

# Copy binary
COPY --from=builder /mole-web /app/mole-web

# Copy mole scripts (needed for clean/uninstall operations)
COPY mole mo ./
COPY bin/ bin/
COPY lib/ lib/

# Make scripts executable
RUN chmod +x mole mo bin/*.sh

# Environment
ENV MOLE_DIR=/app
ENV MOLE_HOST=0.0.0.0
ENV MOLE_PORT=8080
ENV MOLE_NO_OPEN=1

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/mole-web"]
