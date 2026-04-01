# NGINX reverse proxy (Docker)

This setup serves the built Vue frontend and proxies API/redirect routes to the Go backend.

## 1) Build the frontend

From repo root:

```powershell
cd frontend
npm install
npm run build
cd ..
```

## 2) Start full stack with NGINX

```powershell
docker compose -f .\docker-compose.nginx.yml up -d --build
```

Then open:
- `http://brohosting.space/` (frontend)
- `http://brohosting.space/docs` (Swagger UI)
- `http://brohosting.space/openapi.yaml`

## Notes

- Update `SHORTWARDEN_PUBLIC_BASE_URL` in `docker-compose.nginx.yml` to your desired primary domain (http/https).
- Add your domains to `server_name` in `nginx/nginx.conf` if you want strict host matching.
- For HTTPS, put a TLS terminator (nginx + certs, or Caddy/Traefik) in front and set:
  - `SHORTWARDEN_COOKIE_SECURE=1`
  - `SHORTWARDEN_PUBLIC_BASE_URL=https://yourdomain`

