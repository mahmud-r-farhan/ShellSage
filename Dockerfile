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

WORKDIR /workspace

RUN apk add --no-cache ca-certificates tzdata git bash curl

# Copy compiled binary from builder
COPY --from=builder /app/shellsage /usr/local/bin/shellsage

# Real-world container hygiene: unprivileged user, config in a named location.
RUN addgroup -g 1000 shellsage && adduser -u 1000 -G shellsage -h /home/shellsage -D shellsage \
    && mkdir -p /home/shellsage/.shellsage /workspace && chown -R shellsage:shellsage /home/shellsage /workspace

USER shellsage
ENV SHELLSAGE_IN_DOCKER=true
ENV SHELLSAGE_HOME=/home/shellsage/.shellsage
VOLUME ["/home/shellsage/.shellsage", "/workspace"]
WORKDIR /workspace

ENTRYPOINT ["shellsage"]
CMD ["--help"]
