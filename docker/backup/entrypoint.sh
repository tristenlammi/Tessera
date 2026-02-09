#!/bin/sh
set -e

CRON_SCHEDULE="${BACKUP_CRON:-0 2 * * *}"

echo "[$(date)] Tessera Backup — scheduling: ${CRON_SCHEDULE}"

# Write the cron job (Alpine uses BusyBox cron)
# Pass all environment variables into the cron job
env > /etc/environment

cat > /etc/crontabs/root <<EOF
${CRON_SCHEDULE} . /etc/environment; /usr/local/bin/backup.sh >> /var/log/backup.log 2>&1
EOF

# Run an initial backup on startup
echo "[$(date)] Running initial backup..."
/usr/local/bin/backup.sh

# Start crond in the foreground
echo "[$(date)] Starting cron daemon..."
exec crond -f -l 2
