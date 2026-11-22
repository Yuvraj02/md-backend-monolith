# Marketing Digest backend

Monolith HTTP API (`md-backend-monolith`) + shared `pkg`. Replaces gateway / auth-service / blog-service.

## Local

```bash
cd md-backend-monolith
cp .env.example .env   # or: export DATABASE_URL='…' && make setup-db
make run
```

## Docker (Render / Fly / Railway)

Build from this directory (`backend/`):

```bash
docker build -f md-backend-monolith/Dockerfile -t md-backend-monolith .
```

Listen port: set `HTTP_PORT` (defaults to `8080`). On Render, also set `HTTP_PORT` to the same value as `PORT` (usually `10000`), or set `HTTP_PORT=$PORT` in the dashboard.

Required env: `DATABASE_HOST`, `DATABASE_PORT`, `DATABASE_NAME`, `DATABASE_USER`, `DATABASE_PASSWORD`, `DATABASE_SSLMODE`, `OWNER_STUDIO_SECRET`, `CORS_ALLOWED_ORIGINS`.
