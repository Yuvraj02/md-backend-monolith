#!/usr/bin/env bash
# Bootstrap Marketing Digest Postgres (Neon / Supabase / RDS / local).
# Always: generate app credentials → create role → migrate → write .env
#
# Usage:
#   export DATABASE_URL='postgresql://owner:pass@host/db?sslmode=require'  # admin URL
#   ./scripts/setup-db.sh
#   ./scripts/setup-db.sh --rotate   # force new passwords even if .env exists
#
# Schema only (no seed data). Env overrides (optional):
#   APP_DB_USER          default: marketing_digest
#   APP_DB_PASSWORD      default: generated (or reused from .env)
#   OWNER_STUDIO_SECRET  default: generated (or reused from .env)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
ENV_FILE="$ROOT/.env"

ROTATE=0
for arg in "$@"; do
  case "$arg" in
    --rotate) ROTATE=1 ;;
    -h|--help)
      sed -n '2,16p' "$0"
      exit 0
      ;;
    *)
      echo "Unknown arg: $arg" >&2
      exit 2
      ;;
  esac
done

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "DATABASE_URL is required (admin/owner Postgres URL, e.g. Neon connection string)." >&2
  exit 1
fi

normalize_url() {
  local u="$1"
  u="${u/postgresql:\/\//postgres:\/\/}"
  u="$(printf '%s' "$u" | sed -E 's/([&?])channel_binding=[^&]*&?/\1/g; s/[?&]$//; s/\?&/?/')"
  printf '%s' "$u"
}

gen_secret() {
  openssl rand -base64 32 | tr -d '/+=' | head -c 40
}

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing dependency: $1" >&2
    exit 1
  }
}
need atlas
need psql
need openssl
need python3

ADMIN_URL="$(normalize_url "$DATABASE_URL")"

eval "$(python3 - "$ADMIN_URL" <<'PY'
import sys, urllib.parse
u = urllib.parse.urlparse(sys.argv[1])
def q(s):
    return "'" + s.replace("'", "'\"'\"'") + "'"
print(f"ADMIN_USER={q(urllib.parse.unquote(u.username or ''))}")
print(f"PG_HOST={q(u.hostname or '')}")
print(f"PG_PORT={q(str(u.port or 5432))}")
print(f"PG_DB={q((u.path or '/').lstrip('/') or 'neondb')}")
qs = urllib.parse.parse_qs(u.query)
print(f"PG_SSLMODE={q((qs.get('sslmode') or ['require'])[0])}")
PY
)"

# Local stub DNS sometimes REFUSEs Neon CNAMEs; libpq hostaddr bypasses that
# while keeping hostname for TLS SNI.
resolve_hostaddr() {
  local host="$1"
  local ip=""
  ip="$(getent ahostsv4 "$host" 2>/dev/null | awk '{print $1; exit}')"
  if [[ -z "$ip" ]] && command -v dig >/dev/null 2>&1; then
    ip="$(dig +short A "$host" @8.8.8.8 2>/dev/null | awk 'NR==1 && $1 ~ /^[0-9.]+$/ {print; exit}')"
    if [[ -z "$ip" ]]; then
      # Follow one CNAME level then resolve
      local cname
      cname="$(dig +short CNAME "$host" @8.8.8.8 2>/dev/null | head -1 | sed 's/\.$//')"
      if [[ -n "$cname" ]]; then
        ip="$(dig +short A "$cname" @8.8.8.8 2>/dev/null | awk 'NR==1 && $1 ~ /^[0-9.]+$/ {print; exit}')"
      fi
    fi
  fi
  printf '%s' "$ip"
}

HOSTADDR=""
if [[ "$ADMIN_URL" == *"hostaddr="* ]]; then
  HOSTADDR="$(python3 - "$ADMIN_URL" <<'PY'
import sys, urllib.parse
q = urllib.parse.parse_qs(urllib.parse.urlparse(sys.argv[1]).query)
print((q.get("hostaddr") or [""])[0])
PY
)"
else
  HOSTADDR="$(resolve_hostaddr "$PG_HOST")"
  if [[ -n "$HOSTADDR" ]]; then
    if [[ "$ADMIN_URL" == *\?* ]]; then
      ADMIN_URL="${ADMIN_URL}&hostaddr=${HOSTADDR}"
    else
      ADMIN_URL="${ADMIN_URL}?hostaddr=${HOSTADDR}"
    fi
    echo "==> DNS: using hostaddr=${HOSTADDR} for ${PG_HOST}"
  else
    echo "Warning: could not resolve ${PG_HOST}; psql will use system DNS" >&2
  fi
fi

export DATABASE_URL="$ADMIN_URL"

APP_DB_USER="${APP_DB_USER:-marketing_digest}"

env_get() {
  local key="$1"
  [[ -f "$ENV_FILE" ]] || return 0
  local line
  line="$(grep -E "^${key}=" "$ENV_FILE" | tail -n1 || true)"
  [[ -n "$line" ]] || return 0
  printf '%s' "${line#*=}"
}

EXISTING_APP_PASS="$(env_get DATABASE_PASSWORD || true)"
EXISTING_OWNER_SECRET="$(env_get OWNER_STUDIO_SECRET || true)"

# App DB password: explicit env > (reuse .env unless --rotate) > generate
if [[ -n "${APP_DB_PASSWORD:-}" && "$ROTATE" != "1" ]]; then
  :
elif [[ "$ROTATE" == "1" ]]; then
  APP_DB_PASSWORD="$(gen_secret)"
elif [[ -n "$EXISTING_APP_PASS" ]]; then
  APP_DB_PASSWORD="$EXISTING_APP_PASS"
else
  APP_DB_PASSWORD="$(gen_secret)"
fi

# Studio secret: same rules
if [[ -n "${OWNER_STUDIO_SECRET:-}" && "$ROTATE" != "1" ]]; then
  :
elif [[ "$ROTATE" == "1" ]]; then
  OWNER_STUDIO_SECRET="$(gen_secret)"
elif [[ -n "$EXISTING_OWNER_SECRET" ]]; then
  OWNER_STUDIO_SECRET="$EXISTING_OWNER_SECRET"
else
  OWNER_STUDIO_SECRET="$(gen_secret)"
fi

echo "==> ping admin database ($PG_HOST / $PG_DB as $ADMIN_USER)"
psql "$ADMIN_URL" -v ON_ERROR_STOP=1 -c 'SELECT current_database() AS db, current_user AS role;'

echo "==> create / update app role: ${APP_DB_USER}"
python3 - "$ADMIN_URL" "$APP_DB_USER" "$APP_DB_PASSWORD" "$PG_DB" <<'PY'
import subprocess, sys

admin_url, app_user, app_password, db_name = sys.argv[1:]

def run(sql: str) -> None:
    subprocess.run(
        ["psql", admin_url, "-v", "ON_ERROR_STOP=1", "-c", sql],
        check=True,
    )

# Dollar-quote password: $tag$content$tag$
tag = "mdcred"
while tag in app_password:
    tag += "x"
pwd_sql = "$" + tag + "$" + app_password + "$" + tag + "$"

# Identifiers: only allow safe role/db names we control
for name, label in ((app_user, "APP_DB_USER"), (db_name, "database")):
    if not name.replace("_", "").isalnum():
        raise SystemExit(f"Unsafe {label}: {name!r}")

run(
    f"""
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '{app_user}') THEN
    CREATE ROLE {app_user} LOGIN PASSWORD {pwd_sql};
  ELSE
    ALTER ROLE {app_user} WITH LOGIN PASSWORD {pwd_sql};
  END IF;
END
$$;
"""
)
run(f'GRANT CONNECT ON DATABASE "{db_name}" TO {app_user};')
run(f"GRANT USAGE, CREATE ON SCHEMA public TO {app_user};")
run(
    f"""
ALTER DEFAULT PRIVILEGES FOR ROLE CURRENT_USER IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO {app_user};
ALTER DEFAULT PRIVILEGES FOR ROLE CURRENT_USER IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO {app_user};
"""
)
PY

echo "==> atlas migrate apply"
atlas migrate apply --env local --dir "file://migrations" --url "$ADMIN_URL"

echo "==> grant table privileges to ${APP_DB_USER}"
psql "$ADMIN_URL" -v ON_ERROR_STOP=1 <<SQL
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ${APP_DB_USER};
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ${APP_DB_USER};
SQL

APP_URL="$(python3 - "$PG_HOST" "$PG_PORT" "$PG_DB" "$APP_DB_USER" "$APP_DB_PASSWORD" "$PG_SSLMODE" "${HOSTADDR:-}" <<'PY'
import sys, urllib.parse
host, port, db, user, password, sslmode, hostaddr = sys.argv[1:]
netloc = f"{urllib.parse.quote(user, safe='')}:{urllib.parse.quote(password, safe='')}@{host}:{port}"
q = f"sslmode={sslmode}"
if hostaddr:
    q += f"&hostaddr={hostaddr}"
print(f"postgres://{netloc}/{db}?{q}")
PY
)"

echo "==> verify app role can connect"
psql "$APP_URL" -v ON_ERROR_STOP=1 <<'SQL'
SELECT current_user AS app_role, current_database() AS db;
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
ORDER BY 1;
SELECT 'blogs' AS t, count(*)::bigint AS n FROM blogs
UNION ALL SELECT 'articles', count(*) FROM articles
UNION ALL SELECT 'comments', count(*) FROM comments
UNION ALL SELECT 'drafts', count(*) FROM drafts;
SQL

echo "==> write ${ENV_FILE}"
export ENV_FILE PG_HOST PG_PORT PG_DB APP_DB_USER APP_DB_PASSWORD PG_SSLMODE OWNER_STUDIO_SECRET ADMIN_URL
export DATABASE_HOSTADDR="${HOSTADDR:-}"
python3 <<'PY'
import os
from pathlib import Path

env_path = Path(os.environ["ENV_FILE"])
values = {
    "APP_ENV": "production",
    "SERVICE_NAME": "md-backend-monolith",
    "HTTP_PORT": "8080",
    "SHUTDOWN_TIMEOUT": "15s",
    "CORS_ALLOWED_ORIGINS": "*",
    "OWNER_STUDIO_SECRET": os.environ["OWNER_STUDIO_SECRET"],
    "DATABASE_HOST": os.environ["PG_HOST"],
    "DATABASE_PORT": os.environ["PG_PORT"],
    "DATABASE_NAME": os.environ["PG_DB"],
    "DATABASE_USER": os.environ["APP_DB_USER"],
    "DATABASE_PASSWORD": os.environ["APP_DB_PASSWORD"],
    "DATABASE_SSLMODE": os.environ["PG_SSLMODE"],
    "DB_MAX_OPEN_CONNS": "10",
    "DB_MAX_IDLE_CONNS": "2",
    "DB_CONN_MAX_LIFETIME": "30m",
    # Admin URL so you can re-run setup/migrate without pasting Neon again.
    "ADMIN_DATABASE_URL": os.environ["ADMIN_URL"],
}
if os.environ.get("DATABASE_HOSTADDR"):
    values["DATABASE_HOSTADDR"] = os.environ["DATABASE_HOSTADDR"]

existing = {}
if env_path.exists():
    for line in env_path.read_text().splitlines():
        if not line.strip() or line.lstrip().startswith("#") or "=" not in line:
            continue
        k, _, v = line.partition("=")
        existing[k] = v

for k, v in values.items():
    existing[k] = v

order = [
    "APP_ENV", "SERVICE_NAME", "HTTP_PORT", "SHUTDOWN_TIMEOUT",
    "CORS_ALLOWED_ORIGINS", "OWNER_STUDIO_SECRET",
    "DATABASE_HOST", "DATABASE_HOSTADDR", "DATABASE_PORT", "DATABASE_NAME",
    "DATABASE_USER", "DATABASE_PASSWORD", "DATABASE_SSLMODE",
    "DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME",
    "ADMIN_DATABASE_URL",
]
order = [k for k in order if k in existing]
lines = [f"{k}={existing[k]}" for k in order]
for k, v in existing.items():
    if k not in order:
        lines.append(f"{k}={v}")
env_path.write_text("\n".join(lines) + "\n")
print(f"Wrote {env_path}")
print(f"  DATABASE_USER={values['DATABASE_USER']}")
print(f"  DATABASE_HOST={values['DATABASE_HOST']}")
print(f"  DATABASE_NAME={values['DATABASE_NAME']}")
print(f"  OWNER_STUDIO_SECRET=(set)")
print(f"  DATABASE_PASSWORD=(set, length {len(values['DATABASE_PASSWORD'])})")
PY

echo "Done. Credentials are in .env — start with: make run"
