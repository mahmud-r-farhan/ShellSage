# ── Build Stage ──
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates tzdata

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o shellsage .

# ── Runtime Stage ──
FROM alpine:3.21

WORKDIR /root/

RUN apk add --no-cache ca-certificates tzdata git bash curl

# Copy compiled binary from builder
COPY --from=builder /app/shellsage /usr/local/bin/shellsage

# Create application directories
RUN mkdir -p /root/.shellsage/personas /root/.shellsage/history /root/conversations /root/exports

ENV SHELLSAGE_IN_DOCKER=true

ENTRYPOINT ["shellsage"]
