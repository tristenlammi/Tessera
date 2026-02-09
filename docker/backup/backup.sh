#!/bin/sh
# Tessera PostgreSQL Backup Script
# Runs inside the backup container via cron
set -e

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="/backups/tessera_${TIMESTAMP}.sql.gz"

echo "[$(date)] Starting database backup..."

PGPASSWORD="${DB_PASSWORD}" pg_dump \
    -h postgres \
    -U "${DB_USER:-tessera}" \
    -d "${DB_NAME:-tessera}" \
    --no-owner \
    --no-privileges \
    | gzip > "${BACKUP_FILE}"

BACKUP_SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
echo "[$(date)] Backup completed: ${BACKUP_FILE} (${BACKUP_SIZE})"

# Clean up old backups
RETAIN_DAYS="${BACKUP_RETAIN_DAYS:-30}"
echo "[$(date)] Removing backups older than ${RETAIN_DAYS} days..."
find /backups -name "tessera_*.sql.gz" -mtime +${RETAIN_DAYS} -delete

REMAINING=$(ls -1 /backups/tessera_*.sql.gz 2>/dev/null | wc -l)
echo "[$(date)] Backup retention: ${REMAINING} backups on disk"
