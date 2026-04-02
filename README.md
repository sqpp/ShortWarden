# ShortWarden

ShortWarden is a URL shortening service with:
- user accounts (JWT cookie sessions)
- API keys for integrations
- custom aliases
- expiring links
- click tracking + analytics
- domain management with DNS TXT verification
- import/export (JSON + CSV)
- tags 

## Requirements

- Go (recent)
- Docker Desktop (for PostgreSQL)

## Quickstart (Windows / PowerShell)

### Option A: Local dev (recommended)

Start Postgres:

```powershell
.\scripts\db-up.ps1
```

Run migrations:

```powershell
.\scripts\migrate-up.ps1
```

Run the API:

```powershell
.\scripts\dev.ps1
```

Run the frontend (separate terminal):

```powershell
cd .\frontend
npm install
npm run dev
```

Frontend URL:
- `http://localhost:5173`

API URL:
- `http://localhost:8080`

App:
- `http://localhost:5173/app/home`

OpenAPI spec:
- `http://localhost:8080/openapi.yaml`

Swagger UI:
- `http://localhost:8080/docs`

## Quickstart (Linux / macOS / WSL)

### Option A: Local dev (recommended)

Start Postgres:

```bash
./scripts/db-up.sh
```

Run migrations:

```bash
./scripts/migrate-up.sh
```

Run the API:

```bash
./scripts/dev.sh
```

Run the frontend (separate terminal):

```bash
cd ./frontend
npm install
npm run dev
```

### Option B: Docker (NGINX + API + DB)

This stack proxies the API and serves the built frontend behind NGINX.

```powershell
docker compose -f .\docker-compose.nginx.yml up -d --build
```

```bash
docker compose -f ./docker-compose.nginx.yml up -d --build
```

## Environment variables

- `SHORTWARDEN_HTTP_ADDR` (default `:8080`)
- `SHORTWARDEN_POSTGRES_DSN` (required)
- `SHORTWARDEN_JWT_SECRET` (required)
- `SHORTWARDEN_PUBLIC_BASE_URL` (default `http://localhost:8080`)
- `SHORTWARDEN_COOKIE_DOMAIN` (optional)
- `SHORTWARDEN_COOKIE_SECURE` (`1` for https-only cookies)
- `SHORTWARDEN_CORS_ALLOWED_ORIGINS` (optional; for Vue dev server)

