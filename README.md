# Proxy Server with Admin UI

Production-ready HTTP/HTTPS proxy written in Go with a React/Vite control panel, PostgreSQL persistence, and Docker-based deployment.

## Highlights

- **Dual servers**: a low-level proxy on `:8080` and a REST API on `:8081`.
- **React Web UI** on `:3000` for administration, stats, and observability.
- **Two-factor authentication** with Google Authenticator (TOTP), QR setup, AES‑256 secret storage, backup codes, and login enforcement.
- **Granular audit log** that records admin logins (success & failure), user CRUD operations with before/after snapshots, settings changes (old vs new values), and 2FA activity.
- **Traffic visibility**: real-time request logs, per-user statistics, configurable retention, and export to PDF/XLSX.
- **User management**: create/update/disable/delete accounts, manage proxy access lists, enforce admin-only UI access.
- **Secure auth**: bcrypt password hashing, signed JWTs, rate limiting on 2FA, and context-aware middleware.
- **PostgreSQL storage** with migration helpers (`ensureProxySchema`) and Docker volume persistence.

## Quick Start (Docker Compose)

```bash
docker-compose up -d
# wait for backend, frontend, and postgres to become healthy
open http://localhost:3000
```

First launch prompts you to create the initial admin credentials. After the first password login you can enable 2FA from **Profile & Security**.

## Repository Layout

```
proxy/
├── backend/             # Go proxy + API server
│   ├── handlers/        # auth, users, stats/logs, settings, 2FA
│   ├── database/        # PostgreSQL access + migrations
│   ├── middleware/      # JWT auth, context helpers
│   ├── proxy/           # HTTP/HTTPS proxy implementation
│   ├── utils/           # JWT, encryption, TOTP helpers
│   └── main.go
├── frontend/            # React + Vite admin UI
│   ├── src/components   # Layout, 2FA widgets, etc.
│   ├── src/pages        # Dashboard, Users, Logs, Audit, Settings…
│   ├── src/services     # Axios API client
│   └── Dockerfile
├── postgresql/init.sql  # Base schema for fresh installs
├── docker-compose.yml
└── ARCHITECTURE.md / QUICKSTART.md / README.md
```

## Running Without Docker

### Backend

```bash
cd backend
go mod download
export DB_HOST=localhost DB_PORT=5432 DB_USER=proxyuser DB_PASSWORD=proxypass DB_NAME=proxydb
export PROXY_PORT=8080 API_PORT=8081 JWT_SECRET=change-me TWOFA_ENCRYPTION_KEY=change-me
go run main.go
```

### Frontend

```bash
cd frontend
npm install
VITE_API_URL=http://localhost:8081 npm run dev
```

> **Note:** The project depends on a running PostgreSQL instance (`postgresql/init.sql` can bootstrap the schema).

## Environment Variables

| Variable | Description |
| --- | --- |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | PostgreSQL connection |
| `PROXY_PORT` | Port for the raw HTTP/HTTPS proxy (default `8080`) |
| `API_PORT` | Port for the REST API (default `8081`) |
| `JWT_SECRET` | HMAC secret for access tokens |
| `TWOFA_ENCRYPTION_KEY` | Key used to encrypt TOTP secrets (falls back to `JWT_SECRET`, but set a dedicated value in production) |
| `VITE_API_URL` | Frontend → API URL (only for dev / non-Docker runs) |

## Web UI Tour

- **Dashboard** – summary metrics (users, requests, bandwidth) and latest activity.
- **Users** – admin-only CRUD with whitelist/blacklist controls, password resets, and 2FA states.
- **Logs** – paginated request logs with rich filtering, exports, and retention tuning.
- **Traffic Stats** – aggregated daily counters per user.
- **Audit** – sortable audit log showing login results, user changes, settings updates, and originating IPs.
- **Settings** – update proxy parameters, discover public IP, and confirm configuration.
- **Profile & Security** – per-admin page to enable/disable 2FA, regenerate backup codes, or download them securely.

## Proxy Usage Examples

```bash
# Basic Auth via curl
curl -x http://localhost:8080 -U username:password https://api.github.com

# Bearer token
curl -x http://localhost:8080 \
     -H "Proxy-Authorization: Bearer <JWT>" \
     https://api.github.com
```

Python `requests`:

```python
proxies = {
    'http': 'http://username:password@localhost:8080',
    'https': 'http://username:password@localhost:8080',
}
requests.get('https://api.github.com', proxies=proxies)
```

## Security Checklist

- Change all secrets (`JWT_SECRET`, `TWOFA_ENCRYPTION_KEY`, Postgres credentials) before going live.
- Terminate TLS in front of the frontend and API (e.g., Nginx or Traefik).
- Restrict exposed ports—only the proxy (8080), API (8081), and frontend (3000) should be reachable as needed.
- Back up the PostgreSQL volume and monitor audit logs for suspicious activity.
- Encourage admins to enable 2FA; rate limiting protects against brute-force attempts.

## Operational Commands

```bash
docker-compose logs -f backend
docker-compose logs -f frontend
docker-compose logs -f postgres
docker-compose down          # stop services
docker-compose down -v       # stop and remove persistent volumes
```

## Troubleshooting

- **Backend failing to start** – ensure PostgreSQL is healthy (`docker-compose logs postgres`).
- **Frontend cannot reach the API** – confirm backend container is up and `VITE_API_URL` matches.
- **Proxy rejects credentials** – verify the user is active, password is correct, and the account is not admin-only.
- **2FA issues** – regenerate backup codes from Profile and ensure server time is synced (TOTP tolerance ±1 period).

## License

MIT
