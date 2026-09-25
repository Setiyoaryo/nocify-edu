#!/bin/sh
# Daily logical backup with retention. Copy /backups off-host (e.g. rclone to
# object storage) — a backup on the same disk is not a backup.
set -eu
ts=$(date -u +%Y%m%dT%H%M%SZ)
pg_dump --format=custom --no-owner --file="/backups/eduplay-$ts.dump"
find /backups -name 'eduplay-*.dump' -mtime +"${RETENTION_DAYS:-14}" -delete
echo "backup ok: eduplay-$ts.dump"
