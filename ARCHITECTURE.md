# Architecture

## System Overview

```
┌─────────────┐      ┌─────────────────────┐      ┌──────────────┐
│  Frontend   │─────▶│  Backend (API+Proxy)│─────▶│  PostgreSQL  │
│ React + Vite│◀─────│  Go                 │◀─────│  persistence │
└─────────────┘      └─────────────────────┘      └──────────────┘
      :3000                  :8080 / :8081                 :5432
```

- **Frontend (`:3000`)** – React SPA served by Nginx. Talks to `/api/*`.
- **Backend (`:8080` & `:8081`)** – Go binary exposing both the raw proxy and the admin API.
- **PostgreSQL (`:5432`)** – stores users, proxy settings, traffic statistics, request logs, backup codes, and audit events.

## Backend (Go)

### Layout

```
backend/
├── main.go              # boots proxy + API servers
├── handlers/
│   ├── auth.go         # init + password login + 2FA verification
│   ├── twofa.go        # enrollment, verification, backup codes
│   ├── users.go        # admin CRUD with audit logging
│   ├── stats.go        # dashboard data, request logs, exports, audit feed
│   └── settings.go     # proxy configuration updates
├── database/            # SQL queries + schema helpers
├── middleware/          # JWT auth (enforces 2FA completion)
├── proxy/               # HTTP/HTTPS proxy implementation
└── utils/               # JWT, bcrypt, AES encryption, TOTP helpers
```

### Request Flows

1. **Proxy flow (port 8080)**
   ```
   Client ──Proxy-Authorization──▶ Authenticate (Basic/JWT)
         ──Allowed host?────────▶ Enforce whitelist/blacklist
         ──Forward traffic──────▶ Target server
         ──Log + accumulate─────▶ request_logs & traffic_stats
   ```

2. **API flow (port 8081)**
   ```
   Web UI ──Authorization: Bearer────▶ JWT middleware (checks 2FA flag)
         ──Handler──────────────────▶ Database
         ◀────────────────────────── Response
   ```

### Security Layers

- Passwords hashed with bcrypt.
- JWTs signed with HMAC-SHA256; temp tokens used before 2FA verification.
- TOTP secrets encrypted with AES-256 GCM using `TWOFA_ENCRYPTION_KEY`.
- Backup codes hashed with bcrypt and stored one-per-row.
- Rate limiting on 2FA attempts (5 per 5 minutes per user/IP).
- Context-aware middleware rejects admin endpoints unless `two_factor_verified` is true.

### Audit Logging

`admin_audit_logs` captures:

- Password login success/failure (with reason + IP).
- 2FA successes (`LOGIN_SUCCESS` with method info).
- User create/update/delete actions (diff serialized to JSON).
- Settings changes with previous and new values.

Handlers call `LogAdminAction` directly; the old blanket middleware has been removed to avoid noise.

## Frontend (React + Vite)

### Key Routes

```
/login            – password login (+ 2FA challenge modal)
/init-setup       – bootstrap wizard shown until an admin exists
/dashboard        – summary metrics
/users            – admin-only management
/logs             – request logs + traffic stats tabs
/audit            – advanced filtering, sorting, expandable columns
/settings         – proxy configuration (admin only)
/profile          – Profile & Security (2FA management)
```

### Notable Components

- `Layout` – navigation, profile dropdown, logout.
- `TwoFactorVerify` – auto-submitting TOTP/backup code form with countdown.
- `TwoFactorSetup` – QR display, manual secret input, enable/disable controls, backup code regeneration and download modal.
- `BackupCodesModal` – copy/download/close CTA with safety text.
- `ProtectedRoute` – enforces authentication and admin-only sections.

State is handled with React Hooks; authentication metadata lives in `localStorage`.

## Database Schema

Key tables (simplified):

- `users` – proxy/UI accounts (`twofa_secret`, `twofa_enabled`, proxy lists).
- `user_twofa_backup_codes` – hashed backup codes with usage flags.
- `twofa_logs` – rate limiting & monitoring of TOTP/backup attempts.
- `request_logs` – per-request details (method, URL, bytes, duration).
- `traffic_stats` – aggregated day/user counters.
- `proxy_settings` – editable configuration entries.
- `admin_audit_logs` – structured audit trail.

Indexes exist on frequently queried columns such as timestamps, user IDs, and status codes.

## Docker Topology

```yaml
services:
  postgres:
    image: postgres:16-alpine
    volumes: [postgres_data:/var/lib/postgresql/data]
    healthcheck: pg_isready

  backend:
    build: ./backend
    depends_on: { postgres: { condition: service_healthy } }
    ports: ["8080:8080", "8081:8081"]
    environment:
      - DB_*
      - JWT_SECRET
      - TWOFA_ENCRYPTION_KEY

  frontend:
    build: ./frontend
    depends_on: [backend]
    ports: ["3000:80"]
```

All containers join the `proxy-network` bridge, enabling service discovery via hostnames (`postgres`, `backend`, `frontend`).

## Future Enhancements

- Add Prometheus metrics and Grafana dashboards for proxy throughput and 2FA failures.
- Push request and audit logs into ELK/OpenSearch for long-term retention.
- Introduce Redis for caching JWT introspection and additional rate limits.
- Support clustering the proxy/API servers behind a load balancer.
