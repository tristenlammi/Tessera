# =============================================================================
# Tessera All-in-One Image
# =============================================================================
# Single container with PostgreSQL, Redis, MinIO, Go backend, and Nginx.
# Designed for Unraid / homelab single-container deployment.
#
# Build context: project root
#   docker build -f docker/aio/Dockerfile -t tessera:latest .
# =============================================================================

# ---------------------------------------------------------------------------
# Stage 1: Build Go backend
# ---------------------------------------------------------------------------
FROM golang:1.23-alpine AS backend-builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /tessera ./cmd/server

# ---------------------------------------------------------------------------
# Stage 2: Build Vue frontend
# ---------------------------------------------------------------------------
FROM node:20-alpine AS frontend-builder

WORKDIR /app

RUN npm install -g pnpm@latest-9

COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY frontend/ .
RUN pnpm exec vite build

# ---------------------------------------------------------------------------
# Stage 3: All-in-One runtime
# ---------------------------------------------------------------------------
FROM alpine:3.20

LABEL maintainer="Tessera <https://github.com/tristenlammi/Tessera>"
LABEL org.opencontainers.image.title="Tessera"
LABEL org.opencontainers.image.description="Self-hosted productivity platform — all-in-one"

# ── Install runtime packages ────────────────────────────────────────────────
RUN apk add --no-cache \
    postgresql16 postgresql16-client postgresql16-contrib \
    redis \
    nginx \
    ca-certificates tzdata curl bash \
    && rm -rf /var/cache/apk/*

# ── Install MinIO binary ────────────────────────────────────────────────────
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then ARCH=amd64; \
    elif [ "$ARCH" = "aarch64" ]; then ARCH=arm64; \
    fi && \
    curl -fsSL "https://dl.min.io/server/minio/release/linux-${ARCH}/minio" -o /usr/local/bin/minio && \
    chmod +x /usr/local/bin/minio

# ── Create data directories ─────────────────────────────────────────────────
RUN mkdir -p \
    /data/postgres \
    /data/redis \
    /data/minio \
    /data/backups \
    /run/postgresql \
    /run/nginx \
    /var/log/tessera \
    && chown -R postgres:postgres /data/postgres /run/postgresql \
    && chown -R redis:redis /data/redis

# ── Copy Go backend binary ──────────────────────────────────────────────────
COPY --from=backend-builder /tessera /app/tessera

# ── Copy database migrations ────────────────────────────────────────────────
COPY migrations/ /app/migrations/

# ── Copy frontend static files ──────────────────────────────────────────────
COPY --from=frontend-builder /app/dist /usr/share/nginx/html

# ── Nginx config (proxy to local backend) ───────────────────────────────────
COPY docker/aio/nginx.conf /etc/nginx/http.d/default.conf
# Remove default nginx config that conflicts
RUN rm -f /etc/nginx/http.d/default.conf.bak

# ── Entrypoint script ───────────────────────────────────────────────────────
COPY docker/aio/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# ── Expose single port ──────────────────────────────────────────────────────
EXPOSE 8080

# ── Health check ─────────────────────────────────────────────────────────────
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=5 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/ready || exit 1

# ── Volume for all persistent data ──────────────────────────────────────────
VOLUME ["/data"]

ENTRYPOINT ["/entrypoint.sh"]
