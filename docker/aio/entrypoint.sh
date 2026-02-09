#!/bin/bash
set -e

# =============================================================================
# Tessera AIO — Entrypoint & Process Supervisor
# =============================================================================
# Starts PostgreSQL, Redis, MinIO, the Go backend, and Nginx in a single
# container. Handles first-run initialization and graceful shutdown.
# =============================================================================

DATA_DIR="/data"
PG_DATA="${DATA_DIR}/postgres"
MINIO_DATA="${DATA_DIR}/minio"
REDIS_DATA="${DATA_DIR}/redis"

PIDS=()

# ── Cleanup on exit ─────────────────────────────────────────────────────────
cleanup() {
    echo "[tessera] Shutting down..."
    # Stop in reverse order
    for pid in $(echo "${PIDS[@]}" | tr ' ' '\n' | tac); do
        if kill -0 "$pid" 2>/dev/null; then
            echo "[tessera] Stopping PID $pid..."
            kill -TERM "$pid" 2>/dev/null || true
        fi
    done
    # Wait for all to exit
    for pid in "${PIDS[@]}"; do
        wait "$pid" 2>/dev/null || true
    done
    # Stop PostgreSQL cleanly
    if [ -f "${PG_DATA}/postmaster.pid" ]; then
        su postgres -c "pg_ctl -D '${PG_DATA}' -m fast stop" 2>/dev/null || true
    fi
    echo "[tessera] All services stopped."
    exit 0
}
trap cleanup SIGTERM SIGINT SIGQUIT

# ── Ensure directories ──────────────────────────────────────────────────────
mkdir -p "${PG_DATA}" "${MINIO_DATA}" "${REDIS_DATA}" "${DATA_DIR}/backups" /run/postgresql /var/log/tessera
chown -R postgres:postgres "${PG_DATA}" /run/postgresql
chown -R redis:redis "${REDIS_DATA}"

# ── Auto-generate secrets on first run ───────────────────────────────────────
generate_secret() {
    head -c 48 /dev/urandom | base64 | tr -d '\n/+=' | head -c "$1"
}

SECRETS_FILE="${DATA_DIR}/.tessera-secrets"

# Load saved secrets first (so they survive container recreation)
if [ -f "${SECRETS_FILE}" ]; then
    echo "[tessera] Loading saved secrets..."
    set -a
    . "${SECRETS_FILE}" 2>/dev/null || true
    set +a
fi

# Set defaults (env vars > saved secrets > generated)
export DB_NAME="${DB_NAME:-tessera}"
export DB_USER="${DB_USER:-tessera}"
: "${DB_PASSWORD:=$(generate_secret 32)}"
: "${REDIS_PASSWORD:=$(generate_secret 32)}"
: "${JWT_SECRET:=$(generate_secret 48)}"
: "${ENCRYPTION_KEY:=$(head -c 32 /dev/urandom | base64)}"
: "${MINIO_ACCESS_KEY:=tessera}"
: "${MINIO_SECRET_KEY:=$(generate_secret 32)}"
export DB_PASSWORD REDIS_PASSWORD JWT_SECRET ENCRYPTION_KEY MINIO_ACCESS_KEY MINIO_SECRET_KEY
export MINIO_BUCKET="${MINIO_BUCKET:-tessera-files}"
export MAX_UPLOAD_SIZE="${MAX_UPLOAD_SIZE:-10737418240}"
export CHUNK_SIZE="${CHUNK_SIZE:-10485760}"
export FRONTEND_URL="${FRONTEND_URL:-http://localhost:8080}"

# Backend listens internally on 3000; nginx exposes 8080
export SERVER_HOST="0.0.0.0"
export SERVER_PORT="3000"
export APP_ENV="${APP_ENV:-production}"
export APP_DEBUG="${APP_DEBUG:-false}"

# Internal service addresses
export DB_HOST="127.0.0.1"
export DB_PORT="5432"
export DB_SSLMODE="disable"
export REDIS_HOST="127.0.0.1"
export REDIS_PORT="6379"
export MINIO_ENDPOINT="127.0.0.1:9000"
export MINIO_USE_SSL="false"
export MIGRATIONS_PATH="file:///app/migrations"

# Save secrets (first run or update)
cat > "${SECRETS_FILE}" <<EOF
# Tessera auto-generated secrets — $(date -Iseconds)
# These are needed if you recreate the container but keep the /data volume.
DB_PASSWORD=${DB_PASSWORD}
REDIS_PASSWORD=${REDIS_PASSWORD}
JWT_SECRET=${JWT_SECRET}
ENCRYPTION_KEY=${ENCRYPTION_KEY}
MINIO_ACCESS_KEY=${MINIO_ACCESS_KEY}
MINIO_SECRET_KEY=${MINIO_SECRET_KEY}
EOF
chmod 600 "${SECRETS_FILE}"

# ── Initialize PostgreSQL ───────────────────────────────────────────────────
if [ ! -f "${PG_DATA}/PG_VERSION" ]; then
    echo "[tessera] First run — initializing PostgreSQL..."
    su postgres -c "initdb -D '${PG_DATA}' --auth-local=trust --auth-host=md5"

    # Set password for the default postgres user, then create app user
    cat >> "${PG_DATA}/postgresql.conf" <<PGCONF
listen_addresses = '127.0.0.1'
port = 5432
max_connections = 50
shared_buffers = 128MB
work_mem = 4MB
maintenance_work_mem = 64MB
logging_collector = off
log_min_messages = warning
PGCONF

    # Start temporarily
    su postgres -c "pg_ctl -D '${PG_DATA}' -w start -o '-c listen_addresses=127.0.0.1'"

    # Create user and database
    su postgres -c "psql -c \"CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASSWORD}';\"" 2>/dev/null || true
    su postgres -c "psql -c \"CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};\"" 2>/dev/null || true
    su postgres -c "psql -c \"GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};\"" 2>/dev/null || true
    su postgres -c "psql -d '${DB_NAME}'" <<SQL
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
SQL

    su postgres -c "pg_ctl -D '${PG_DATA}' -w stop"
    echo "[tessera] PostgreSQL initialized."
fi

# ── Start PostgreSQL ─────────────────────────────────────────────────────────
echo "[tessera] Starting PostgreSQL..."
su postgres -c "pg_ctl -D '${PG_DATA}' -w start -o '-c listen_addresses=127.0.0.1'"

# Wait for PostgreSQL to be ready
for i in $(seq 1 30); do
    if pg_isready -h 127.0.0.1 -U "${DB_USER}" -d "${DB_NAME}" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done
echo "[tessera] PostgreSQL is ready."

# ── Start Redis ──────────────────────────────────────────────────────────────
echo "[tessera] Starting Redis..."
cat > /etc/redis-tessera.conf <<REDIS_CONF
bind 127.0.0.1
port 6379
requirepass ${REDIS_PASSWORD}
dir ${REDIS_DATA}
appendonly yes
maxmemory 256mb
maxmemory-policy allkeys-lru
loglevel warning
daemonize no
REDIS_CONF
chown redis:redis /etc/redis-tessera.conf

redis-server /etc/redis-tessera.conf &
PIDS+=($!)

# Wait for Redis to be ready
for i in $(seq 1 15); do
    if redis-cli -h 127.0.0.1 -a "${REDIS_PASSWORD}" --no-auth-warning ping 2>/dev/null | grep -q PONG; then
        break
    fi
    sleep 1
done
echo "[tessera] Redis is ready."

# ── Start MinIO ─────────────────────────────────────────────────────────────
echo "[tessera] Starting MinIO..."
export MINIO_ROOT_USER="${MINIO_ACCESS_KEY}"
export MINIO_ROOT_PASSWORD="${MINIO_SECRET_KEY}"

/usr/local/bin/minio server "${MINIO_DATA}" \
    --address ":9000" \
    --console-address ":9001" \
    --quiet &
PIDS+=($!)

# Wait for MinIO to be healthy
for i in $(seq 1 30); do
    if curl -sf http://127.0.0.1:9000/minio/health/live >/dev/null 2>&1; then
        break
    fi
    sleep 1
done
echo "[tessera] MinIO is ready."

# Create bucket via S3 API (PUT request with empty body)
HTTP_CODE=$(curl -sf -o /dev/null -w "%{http_code}" \
    -X PUT "http://127.0.0.1:9000/${MINIO_BUCKET}" \
    -u "${MINIO_ACCESS_KEY}:${MINIO_SECRET_KEY}" 2>/dev/null || echo "000")

if [ "$HTTP_CODE" = "200" ]; then
    echo "[tessera] MinIO bucket '${MINIO_BUCKET}' created."
elif [ "$HTTP_CODE" = "409" ]; then
    echo "[tessera] MinIO bucket '${MINIO_BUCKET}' already exists."
else
    echo "[tessera] MinIO bucket setup returned HTTP ${HTTP_CODE} (may already exist)."
fi

# ── Start Tessera backend ────────────────────────────────────────────────────
echo "[tessera] Starting Tessera backend..."
/app/tessera &
PIDS+=($!)

# Wait for backend to be ready
for i in $(seq 1 30); do
    if wget --spider --quiet http://127.0.0.1:3000/api/ready 2>/dev/null; then
        break
    fi
    sleep 1
done
echo "[tessera] Backend is ready."

# ── Start Nginx ──────────────────────────────────────────────────────────────
echo "[tessera] Starting Nginx..."
nginx -g "daemon off;" &
PIDS+=($!)

echo "============================================================"
echo "  Tessera is running!"
echo "  Open http://localhost:8080 in your browser"
echo "============================================================"

# ── Wait for any process to exit ─────────────────────────────────────────────
# If any service crashes, bring everything down
wait -n "${PIDS[@]}" 2>/dev/null || true

echo "[tessera] A service exited unexpectedly, shutting down..."
cleanup
