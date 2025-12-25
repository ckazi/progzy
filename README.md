# Progzy Proxy Server

Production-ready HTTP/HTTPS proxy with an admin UI, PostgreSQL storage, and Docker-first deployment.

![Dashboard](assets/dashboard.png)
![Users](assets/users.png)

## Quick start (one command)

```bash
docker compose up -d
```

Open the UI:

```
http://localhost:13000
```

First visit redirects to the initial admin setup page.

## How it works

- Proxy listens on `18080` and accepts Basic Auth or JWT Bearer tokens.
- Admin UI runs on `13000` and talks to the API through its own nginx proxy.
- PostgreSQL data is stored in `./postgres_data` next to `docker-compose.yml`.
- Database schema is initialized automatically inside the API container.

## Proxy usage

Basic Auth:

```bash
curl -x http://localhost:18080 -U username:password https://api.github.com
```

Bearer token:

```bash
curl -x http://localhost:18080 \
  -H "Proxy-Authorization: Bearer <JWT>" \
  https://api.github.com
```

If your password contains special characters, URL-encode it (example: `*` -> `%2A`).

## Default ports

- Proxy: `18080`
- Admin UI: `13000`
- API: internal only (proxied via `/api`)

You can change ports in `docker-compose.yml` if needed.

## Features

- Admin-only UI with users, logs, audit, stats, and settings
- 2FA (TOTP), backup codes, and secure password hashing
- Traffic logging, filtering, and exports (PDF/XLSX)
- Per-user proxy lists (whitelist/blacklist)
- Automatic log retention cleanup

## Useful commands

```bash
docker compose logs -f backend
docker compose logs -f frontend
docker compose logs -f postgres
docker compose down
docker compose down -v
```

## Troubleshooting

- **UI is blank**: wait for the containers to finish starting.
- **Proxy auth fails**: confirm user is active and not admin-only.
- **2FA issues**: make sure server time is in sync.

## License

MIT
