# Authentication

Loom supports JWT tokens and API keys for authentication.

## JWT Authentication

Users log in via the web UI with username/password. The server returns a JWT token that authenticates subsequent requests.

Default credentials: `admin` / `admin`

```bash
# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin"}'

# Use the token
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/beads
```

## API Keys

For programmatic access, create API keys:

```bash
curl -X POST http://localhost:8080/api/v1/auth/api-keys \
  -H "Authorization: Bearer <jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "CI Pipeline", "permissions": ["read", "write"]}'
```

Use the API key in requests:

```bash
curl -H "X-API-Key: <key>" http://localhost:8080/api/v1/beads
```

## User Management

```bash
# List users
curl http://localhost:8080/api/v1/users

# Create a user
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"username": "dev1", "password": "password", "role": "developer"}'
```

## Security Configuration

```yaml
security:
  jwt_secret: "your-secret-here"   # Override with a strong random secret
  token_expiry: 24h
  api_key_enabled: true
```

!!! warning
    Always change the default credentials and JWT secret in production deployments.
