# SynergyFlow Deployment Guide

## Architecture

```
                         ┌──────────────┐
                         │   EC2 Instance   │
                         │  (Docker host)   │
                         │                  │
  Internet ──► Nginx ──┼──► frontend:80  │
        :80    │     │──► backend:8080  │
               │     │──► SSE (stream)  │
               │     │                  │
               │     ├── postgres:5432  │
               │     ├── redis:6379     │
               │     └── worker         │
               │                        │
               └── S3 (attachments) ────┘
```

- **EC2** runs all services via Docker Compose.
- **Nginx** serves HTTP on port 80 and routes `/` to frontend and `/api/*` to backend.
- **TLS** is not configured in the supplied production Compose/config; add a TLS listener or terminate TLS upstream before using HTTPS.
- **PostgreSQL** stores all application data.
- **Redis** handles pub/sub for SSE events. Sessions are stored in PostgreSQL.
- **Worker** processes email jobs on a polling loop.
- **S3** stores file attachments (presigned URLs).

## Required AWS Resources

| Resource        | Purpose                          | Notes                            |
|-----------------|----------------------------------|----------------------------------|
| EC2 instance    | Docker host for all services     | t3.medium or larger recommended  |
| Security group  | Inbound: 22 (SSH), 80 (HTTP), 443 (HTTPS) | Restrict SSH to your IP     |
| Elastic IP      | Static public IP for the domain  | Attach to the EC2 instance       |
| Domain          | `synergyflow.<your-domain>`      | Route53 or any DNS provider      |
| S3 bucket       | File attachments for tasks       | Private bucket, server-side encryption recommended |

## GitHub Secrets

Set these in your GitHub repository → Settings → Secrets and variables → Actions:

| Secret          | Description                                |
|-----------------|--------------------------------------------|
| `EC2_HOST`      | Public IP or DNS name of the EC2 instance  |
| `EC2_USER`      | SSH user (typically `ubuntu` or `ec2-user`)|
| `EC2_SSH_KEY`   | Private SSH key (PEM) for EC2 access       |

## Environment Variables (.env)

Create a `.env` file in the repository root on the EC2 instance. Copy from `.env.example`:

```bash
cp .env.example .env
```

### Required

| Variable            | Description                                        | Example                                               |
|---------------------|----------------------------------------------------|-------------------------------------------------------|
| `POSTGRES_DB`       | PostgreSQL database name                            | `synergyflow`                                         |
| `POSTGRES_USER`     | PostgreSQL user                                     | `synergy`                                             |
| `POSTGRES_PASSWORD` | PostgreSQL password                                 | `generate-a-strong-password`                          |
| `DATABASE_URL`      | Full PostgreSQL connection string                   | `postgres://synergy:pass@postgres:5432/synergyflow?sslmode=disable` |
| `REDIS_URL`         | Redis connection string                             | `redis://redis:6379/0`                                |
| `JWT_SECRET`        | JWT signing key (min 32 bytes base64)               | `openssl rand -base64 32`                             |
| `FRONTEND_URL`      | Public URL of the application (used for CORS)       | `https://synergyflow.example.com`                     |
| `AWS_REGION`        | AWS region for S3                                   | `us-east-1`                                           |
| `AWS_ACCESS_KEY_ID` | AWS access key                                      | `AKIA...`                                             |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key                                  | `...`                                                 |
| `S3_BUCKET`         | S3 bucket name for file attachments                 | `synergyflow-prod`                                    |
| `PORT`              | Backend listen port (default 8080)                  | `8080`                                                |

### Optional

| Variable            | Description                                        | Default                   |
|---------------------|----------------------------------------------------|---------------------------|
| `S3_ENDPOINT`       | Custom S3 endpoint (for S3-compatible storage)     | (empty = standard AWS S3) |
| `RESEND_API_KEY`    | Reserved for future transactional email support; current worker logs queued email jobs | (unused) |
| `FROM_EMAIL`        | Sender email address                               | `SynergyFlow <noreply@example.com>` |
| `EVENTLOG_ROOT`     | Durable SSE event-log directory inside backend container | `/var/lib/synergyflow/eventlog` |
| `EVENTLOG_FSYNC`    | Force fsync after event-log appends                | `false`                   |

The production Compose file sets `EVENTLOG_ROOT` to `/var/lib/synergyflow/eventlog` and persists it in the `eventlog` volume.
The bundled PostgreSQL service uses `sslmode=disable` in the example connection string.
For an external PostgreSQL service that requires TLS, use `sslmode=require`.
`VITE_API_URL` is read during the frontend image build, not from the running production container.
The supplied production image uses same-origin `/api` routing unless the frontend image build is changed to inject a separate API URL.

## First Deploy

### 1. Launch EC2 instance

- **AMI:** Ubuntu 24.04 or 22.04 LTS
- **Instance type:** t3.medium (or larger for production)
- **Security group:** Allow 22 (SSH), 80 (HTTP), 443 (HTTPS)
- **Storage:** 20 GB gp3 minimum
- **Elastic IP:** Allocate and associate

### 2. Bootstrap the instance

```bash
ssh -i your-key.pem ubuntu@<elastic-ip>
git clone https://github.com/<your-org>/synergy-flow.git ~/synergy-flow
cd ~/synergy-flow
chmod +x scripts/*.sh
./scripts/bootstrap-ec2.sh
```

Then log out and back in (or run `newgrp docker`) for the docker group to take effect.

### 3. Configure environment

```bash
cd ~/synergy-flow
cp .env.example .env
# Edit .env with production values:
#   - Set DATABASE_URL with real password
#   - Set JWT_SECRET (openssl rand -base64 32)
#   - Set FRONTEND_URL to your domain
#   - Set AWS credentials for S3
```

### 4. Start the stack

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

### 5. Verify

```bash
./scripts/healthcheck.sh
# Or:
curl -s http://localhost/health
curl -s http://localhost/ready
```

### 6. Configure DNS

Point `synergyflow.example.com` to the EC2 Elastic IP.

### 7. Configure TLS separately (optional)

The supplied Nginx config has no TLS listener or certificate mounts.
Terminate TLS upstream, or add a complete HTTPS server block and certificates to `infra/nginx/default.conf` before exposing HTTPS.

## Redeploy Flow

The GitHub Actions workflow (`.github/workflows/deploy.yml`) handles this automatically:

1. Push to `main` on GitHub.
2. GitHub Actions runs checks (go vet, go test, go build, npm ci, npm build, docker compose config).
3. If checks pass, SSH into EC2.
4. If `EC2_HOST`, `EC2_USER`, and `EC2_SSH_KEY` secrets are all set, EC2 pulls latest code, rebuilds containers, and restarts the stack. If any are missing, the deploy job fails before SSH.

### Manual redeploy

```bash
ssh -i your-key.pem ubuntu@<elastic-ip>
cd ~/synergy-flow
git fetch origin main
git reset --hard origin/main
docker compose -f docker-compose.prod.yml up -d --build
docker image prune -f
```

## Demo Reset Flow

To reset the demo database to its seeded state (destroys all data!):

```bash
cd ~/synergy-flow
RESET_DEMO_DB_CONFIRM=yes ./scripts/reset-demo-db.sh
```

This drops the public schema, re-runs migrations, and re-imports seed data.
The demo account `demo@synergyflow.dev` / `password123` will be available again.

## Troubleshooting

### Nginx 502 Bad Gateway

- Backend or frontend container is not running or not healthy.
- Check: `docker compose -f docker-compose.prod.yml ps`
- Check logs: `docker compose -f docker-compose.prod.yml logs backend`
- The backend takes a moment to start while waiting for Postgres/Redis.

### Backend not ready

- Check `/health` and `/ready` endpoints: `curl -s http://localhost/health`
- If `/ready` returns 503, check database and Redis connectivity:
  ```bash
  docker compose -f docker-compose.prod.yml logs backend
  ```

### Database migration failed

- Check migrate logs:
  ```bash
  docker compose -f docker-compose.prod.yml logs migrate
  ```
- Common issues: wrong DATABASE_URL, Postgres not fully healthy yet.
- Manual migration:
  ```bash
  docker compose -f docker-compose.prod.yml run --rm migrate
  ```

### CORS issue

- Ensure `FRONTEND_URL` in `.env` exactly matches the domain users access in the browser.
- If accessing via Elastic IP directly, set `FRONTEND_URL=http://<elastic-ip>` or use the domain.
- The Nginx config includes CORS preflight handling for `/api/` routes.

### SSE not updating

- SSE timeout middleware (30s) should no longer apply — verify you have the latest code.
- Check the browser's Network tab: the SSE connection to `/api/projects/<id>/events` should stay open.
- Nginx `proxy_buffering off` and `proxy_read_timeout 1h` must be present.
- Check Redis pub/sub is working: events are published to `project:<id>` channels.
- Durable reconnect replay reads backend event-log records after `Last-Event-ID` or the `?after=` query value. In production this path is persisted by the `eventlog` Docker volume at `/var/lib/synergyflow/eventlog`.
- The `middleware.TimeoutMiddleware` in `app.go` must NOT wrap the SSE route.

### S3 upload failing

- Verify AWS credentials in `.env` have `s3:PutObject` permission.
- Check `S3_BUCKET` and `AWS_REGION` match the actual bucket.
- The S3 endpoint defaults to standard AWS S3 when `S3_ENDPOINT` is empty.
- For custom S3-compatible storage, set `S3_ENDPOINT` to the service URL.

### GitHub Actions SSH failure

- Verify `EC2_HOST`, `EC2_USER`, `EC2_SSH_KEY` are set in GitHub secrets.
- Ensure the EC2 security group allows SSH from GitHub Actions IPs (or use `0.0.0.0/0` with key-based auth).
- The SSH key in the secret must be the private key (PEM format), not the public key.
- Test SSH manually: `ssh -i key.pem ubuntu@<elastic-ip>`
