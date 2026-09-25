#!/bin/sh
# Usage: docker compose -f infra/compose/compose.prod.yaml exec backup sh /restore.sh /backups/eduplay-<ts>.dump
# Restores into the live database (drops existing objects). Test restores regularly on staging.
set -eu
pg_restore --clean --if-exists --no-owner --dbname="$PGDATABASE" "$1"
