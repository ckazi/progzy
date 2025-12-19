# Quick Start

Launch the full stack in three steps.

1. **Start the stack**
   ```bash
   docker-compose up -d
   ```
2. **Open the UI**
   ```
   http://localhost:3000
   ```
3. **Create the initial admin**
   - Choose a username, password, and optional email.
   - After submitting, you will be logged in automatically.

### Next Steps

- Visit **Profile & Security** to enable two-factor authentication.
- Create standard proxy users in the **Users** section.
- Inspect activity in **Dashboard**, **Logs**, and the **Audit** tab.

### Using the Proxy

Configure your application or OS proxy settings:

- Host: `localhost`
- Port: `8080`
- Authentication: either *Basic Auth* (`username:password`) or *Bearer* (`Proxy-Authorization: Bearer <JWT>`).

Example:

```bash
curl -x http://localhost:8080 -U user:pass https://api.github.com
```

For more details, environment variables, and troubleshooting tips see [README.md](README.md).
