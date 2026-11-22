# md-backend-monolith

Single HTTP process on `:8080` that serves the Marketing Digest blog and article API. It replaces the separate **gateway**, **blog-service** (gRPC), and **auth-service** deployments for lower hosting cost.

- Frontend talks only to this service over `/v1/*` (same JSON shapes as before).
- Studio writes use `X-Owner-Secret`; owner user is hardcoded as `user-1` (auth-service is **not** included — the UI never used it).
- No gateway process; handlers call blog/article services in-process.

## Run locally

```bash
cp .env.example .env
# Set DATABASE_* for Postgres (Neon: host, user, password, sslmode=require)
make run
```

Health: `GET /healthz`, `GET /readyz`, `GET /v1/blog/ping`.

## Database setup (Neon or any Postgres)

`scripts/setup-db.sh` always:

1. Generates app role + password (`marketing_digest`) and `OWNER_STUDIO_SECRET`
2. Creates/updates that role on Postgres
3. Applies Atlas migrations
4. Writes credentials into `.env` (also keeps `ADMIN_DATABASE_URL` for re-runs)

```bash
export DATABASE_URL='postgresql://OWNER:PASS@HOST/DB?sslmode=require'  # Neon admin URL
make setup-db          # schema + credentials only (no seed data)
./scripts/setup-db.sh --rotate   # force new passwords
```

Reuse later on another host — only change `DATABASE_URL`. Re-run without `--rotate` keeps existing `.env` secrets.

## Layout

- `internal/app/blog`, `internal/app/article` — domain + GORM stores (from blog-service)
- `internal/transport/http` — JSON HTTP handlers (from gateway, in-process calls)
- `internal/infrastructure/database` — Postgres via shared `github.com/marketing-digest/pkg`

Shared helpers: `replace github.com/marketing-digest/pkg => ../pkg`

## Free / low-cost hosting (cost crunch)

Recommended stack while AWS is paused:

| Piece | Free option | Notes |
|-------|-------------|--------|
| **Postgres** | [Neon](https://neon.tech) | Free tier, serverless Postgres, `sslmode=require`. Point `DATABASE_*` / `DATABASE_URL` at the Neon connection string. |
| **API (Go binary)** | [Railway](https://railway.app) or [Render](https://render.com) free/hobby | Deploy the Dockerfile from `backend/` (`docker build -f md-backend-monolith/Dockerfile …`). Set env from `.env.example`. Or [Fly.io](https://fly.io) free allowance. |
| **Frontend (Next.js)** | [Vercel](https://vercel.com) free | Set the API base URL to your Railway/Render/Fly public URL. |
| **Container registry** | Docker Hub / GHCR free | Skip ECR. |
| **Alt DB later** | Supabase / PlanetScale / self-host | Same Postgres driver; swap connection string when ready. |

### Neon setup (current plan)

1. Create a Neon project → copy the connection string.
2. Map it into env, e.g.:
   - `DATABASE_HOST=ep-….ap-southeast-1.aws.neon.tech`
   - `DATABASE_PORT=5432`
   - `DATABASE_NAME=neondb`
   - `DATABASE_USER=…`
   - `DATABASE_PASSWORD=…`
   - `DATABASE_SSLMODE=require`
3. Run Atlas once against Neon: `DATABASE_URL=… atlas migrate apply --env local`
4. Deploy `md-backend-monolith` with those env vars + `OWNER_STUDIO_SECRET` + `CORS_ALLOWED_ORIGINS` (your Vercel domain).

### What you can turn off on AWS

Gateway, auth-service, blog-service, ECR pulls for those images, and the k3s migrate Jobs for them — this one binary + Neon replaces that path.
