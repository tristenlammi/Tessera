FROM alpine:3.20

RUN apk add --no-cache postgresql16-client gzip

COPY backup.sh /usr/local/bin/backup.sh
RUN chmod +x /usr/local/bin/backup.sh

RUN mkdir -p /backups

# The entrypoint installs the cron schedule from the env var and runs crond
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
