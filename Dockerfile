# syntax=docker/dockerfile:1
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Install build dependencies for CGO (SQLite)
RUN apk add --no-cache gcc musl-dev

# Copy source
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
COPY api ./api
COPY web ./web

# Build the server binary
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -trimpath -o /bin/gofolio ./cmd/server

# Runtime stage
FROM alpine:latest

# sqlite for the CLI tools, libc6-compat for CGO binary compatibility
RUN apk --no-cache add ca-certificates sqlite libc6-compat

# Run as non-root
RUN addgroup -S gofolio && adduser -S -G gofolio gofolio

WORKDIR /app

COPY --from=builder /bin/gofolio ./gofolio
COPY web ./web

# Ensure database and uploaded media live in a writable, mounted location
RUN mkdir -p /app/data && chown -R gofolio:gofolio /app

USER gofolio

ENV PORT=8080 \
    DATABASE_PATH=/app/data/gofolio.db

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz/ >/dev/null 2>&1 || exit 1

ENTRYPOINT ["./gofolio"]
