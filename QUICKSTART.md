# Quick Start

This is the fastest way to get a working proxy server with the admin UI.

## 1) Start the stack

```bash
docker compose up -d
```

## 2) Open the UI

```
http://localhost:13000
```

On first launch you will be redirected to create the initial admin account.

## 3) Use the proxy

- Host: your server IP
- Port: `18080`
- Auth: Basic (`username:password`) or Bearer token

Example:

```bash
curl -x http://localhost:18080 -U user:pass https://api.github.com
```

If your password contains special characters, URL-encode it, for example `*` -> `%2A`.

For more details see `README.md`.
