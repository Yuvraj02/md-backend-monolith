#!/usr/bin/env bash
# Dump marketing_digest_blog from private RDS via SSM port-forward on the cluster host.
# Prereqs: AWS_PROFILE with SSM access, session-manager-plugin, pg_dump
set -euo pipefail

REGION="${AWS_REGION:-ap-south-1}"
PROFILE="${AWS_PROFILE:-marketing-digest}"
TARGET="${SSM_TARGET:-i-0efd4226192f843bc}"
RDS_HOST="${RDS_HOST:-marketing-digest-production-postgres.c5imcgsg0f2z.ap-south-1.rds.amazonaws.com}"
LOCAL_PORT="${LOCAL_PORT:-5433}"
DB_NAME="${DB_NAME:-marketing_digest_blog}"
DB_USER="${DB_USER:-md_admin}"
if [[ -z "${PGPASSWORD:-}" ]]; then
  echo "PGPASSWORD is required" >&2
  exit 1
fi
export PGPASSWORD

OUT_DIR="${OUT_DIR:-$(cd "$(dirname "$0")/.." && pwd)/dumps}"
mkdir -p "$OUT_DIR"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_FILE="${OUT_DIR}/${DB_NAME}_${STAMP}.sql"

echo "Starting SSM port-forward ${RDS_HOST}:5432 -> localhost:${LOCAL_PORT} ..."
AWS_PROFILE="$PROFILE" aws ssm start-session \
  --target "$TARGET" \
  --document-name AWS-StartPortForwardingSessionToRemoteHost \
  --parameters "{\"host\":[\"${RDS_HOST}\"],\"portNumber\":[\"5432\"],\"localPortNumber\":[\"${LOCAL_PORT}\"]}" \
  --region "$REGION" &
SSM_PID=$!

cleanup() {
  kill "$SSM_PID" 2>/dev/null || true
}
trap cleanup EXIT

# Wait for local port
for i in $(seq 1 30); do
  if (echo >/dev/tcp/127.0.0.1/"$LOCAL_PORT") >/dev/null 2>&1; then
    break
  fi
  sleep 1
  if ! kill -0 "$SSM_PID" 2>/dev/null; then
    echo "SSM session exited early — check AWS login / SSM permissions" >&2
    exit 1
  fi
  if [[ "$i" -eq 30 ]]; then
    echo "Timed out waiting for localhost:${LOCAL_PORT}" >&2
    exit 1
  fi
done

echo "Dumping ${DB_NAME} -> ${OUT_FILE}"
pg_dump \
  --host=127.0.0.1 \
  --port="$LOCAL_PORT" \
  --username="$DB_USER" \
  --dbname="$DB_NAME" \
  --no-owner \
  --no-acl \
  --format=plain \
  --file="$OUT_FILE"

echo "Done: ${OUT_FILE}"
ls -lh "$OUT_FILE"
