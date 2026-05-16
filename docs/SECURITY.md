# SynergyFlow Security

## Authentication

- **Password hashing**: bcrypt with cost factor 12
- **JWT signing**: HMAC-SHA256 with configurable secret
- **Access token TTL**: 15 minutes
- **Refresh token TTL**: 30 days, stored as SHA-256 hash
- **Token rotation**: Each refresh invalidates the previous refresh token
- **Session revocation**: Tokens are checked against `sessions.revoked_at`
- **Logout**: Revokes the session; future refresh attempts fail

### Security guarantees

- Passwords are never stored in plaintext
- Refresh tokens are never returned as URL parameters
- Access tokens can be passed as `Authorization: Bearer` header or `token` query param
- Auth errors return generic "invalid credentials" — no account existence leakage

## Authorization

Role-based access control (RBAC) with four tiers:

```
Viewer < Member < Admin < Owner
```

Each endpoint checks the minimum required role before processing:

- **Viewer**: Read workspace, projects, board, tasks, comments, activity
- **Member**: Create/update tasks, comments, attachments, create projects
- **Admin**: Manage members (invite, remove, change roles)
- **Owner**: Delete projects, delete workspace, manage all roles

### Protection rules

- Users cannot remove themselves from a workspace
- Users cannot change the role of another user with equal or higher rank
- The final Owner cannot be removed or downgraded
- Task movement is restricted to the same project
- Attachment deletion requires task-level Member access
- Invite creation requires workspace-level Admin access

## Data Protection

- All database passwords are stored with bcrypt
- Refresh token hashes use SHA-256
- Session data includes user-agent and IP (when available)
- File uploads use random UUID prefixes to prevent enumeration
- Presigned URLs for attachment downloads have configurable TTL

## HTTP Security

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Strict-Transport-Security` (when served over HTTPS)
- CORS restricted to configured `FRONTEND_URL`
- Request body size limited to 12 MB
- Request timeout of 30 seconds
- Request ID tracking for audit trails

## Production Checklist

1. Generate a strong JWT secret: `openssl rand -base64 32`
2. Use HTTPS with valid TLS certificates
3. Set `FRONTEND_URL` to the exact frontend domain
4. Use strong database credentials
5. Restrict S3 bucket access with IAM policies
6. Keep Resend API key secure
7. Use Docker non-root user (already configured)
8. Enable Postgres SSL in production (`sslmode=require`)
9. Set secure session and JWT TTLs
10. Monitor failed login attempts at the reverse proxy level

## Known Limitations

- Rate limiting is not implemented (relies on the reverse proxy)
- No brute-force protection on login endpoints
- No email verification for new accounts
- API keys are not supported (JWT only)
- No audit log for admin actions (only general activity events)
- No Web Application Firewall (WAF) configured by default
