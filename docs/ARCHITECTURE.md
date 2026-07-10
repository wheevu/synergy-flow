# SynergyFlow Architecture

## Overview

SynergyFlow is a real-time collaborative project management platform built on a
Go backend with a React/TypeScript frontend. The architecture follows a
service-oriented pattern within a monorepo, with clear separation between API,
worker, database, and frontend responsibilities.

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Frontend   │────▶│   Backend    │────▶│  PostgreSQL  │
│  React/TS    │     │   Go/Gin     │     │              │
│  Vite/Tailwind│    │   pgx        │     │              │
└──────┬───────┘     └──────┬───────┘     └──────────────┘
       │                    │                    ▲
       │ SSE (events)       │ Redis pub/sub ─────┘
       │                    │
       │                    ▼
       │            ┌──────────────┐
       └────────────│   Worker     │
                    │  email jobs  │
                    └──────────────┘
```

## Directory Structure

```
backend/
├── cmd/
│   ├── server/main.go      # HTTP server entrypoint
│   └── worker/main.go      # Background worker entrypoint
├── internal/
│   ├── app/
│   │   ├── app.go           # Core server: routes, handlers, auth, SSE
│   │   ├── helpers.go       # Pure helper functions (slugify, hash, rank)
│   │   ├── helpers_test.go  # Unit tests for helpers
│   │   └── analyzer_test.go # Unit tests for analysis logic
│   └── middleware/
│       └── middleware.go     # Request ID, security headers, logging, timeouts
├── migrations/              # goose-managed SQL migrations
├── go.mod
└── go.sum

frontend/
├── src/
│   ├── main.tsx             # App entrypoint and all components
│   ├── styles.css           # Tailwind + custom CSS
│   ├── types/index.ts       # Shared TypeScript domain types
│   ├── api/client.ts        # Axios client with JWT refresh interceptor
│   ├── lib/
│   │   ├── store.ts         # Zustand auth + UI state stores
│   │   └── helpers.ts       # Pure utility functions
│   └── components/
│       └── shared.tsx       # Shared UI components (charts, layouts)
├── Dockerfile
└── nginx.conf
```

## Backend Architecture

### HTTP Layer

- **Gin framework** with custom middleware stack:
  - Request ID generation (`X-Request-ID` header)
  - Security headers (`X-Content-Type-Options`, `X-Frame-Options`, etc.)
  - Structured request logging
  - Panic recovery
  - Request body size limiting (12 MB)
  - Context timeout (30 seconds)
  - CORS (configured via `FRONTEND_URL`)

### Authentication

JWT-based with refresh token rotation:

1. **Access tokens**: 15-minute lifetime, signed with HS256, contain user ID
2. **Refresh tokens**: 30-day lifetime, stored as SHA-256 hash in `sessions` table
3. **Rotation**: Each refresh revokes the previous token and issues a new pair
4. **Revocation**: Logout marks the session as revoked; expired tokens are rejected

### Authorization

Role-based workspace permissions enforced through a middleware helper chain:

| Role   | Workspace Level | Project Level | Task Level |
|--------|----------------|---------------|------------|
| Viewer | Read           | Read          | Read       |
| Member | Read + Invites | Create/Edit   | Full CRUD  |
| Admin  | Manage members | Edit/Delete   | Full CRUD  |
| Owner  | Full control   | Delete        | Full CRUD  |

Permission checks follow: `can()` → `canProject()` → `canTask()`, each looking up
the user's workspace role and comparing against the required minimum rank.

### Real-time Events (SSE)

1. Backend actions publish events to Redis channels (`project:{projectId}`)
2. Clients connect to `GET /projects/:id/events` with an SSE stream
3. Redis subscription forwards events to all connected clients
4. Ping events every 25 seconds keep the connection alive
5. Nginx is configured with `proxy_buffering off` for SSE compatibility

### Task Movement

Drag-and-drop position updates use a database transaction:

1. Lock the source and destination column task rows with `SELECT ... FOR UPDATE`
2. Close the gap in the source column (`position-1`)
3. Open a gap in the destination column (`position+1`)
4. Update the moved task's `column_id` and `position`
5. Publish a `task.moved` event to Redis

This maintains dense integer ordering and prevents cross-project moves.

### File Uploads

- 10 MB size limit (configurable)
- Filenames are prefixed with a UUID to prevent collisions
- Storage key format: `attachments/{uuid}/{original_filename}`
- Metadata (name, type, size, uploader) is stored in PostgreSQL
- Presigned GET URLs for secure downloads with configurable TTL
- Objects are stored in S3-compatible storage (MinIO for local dev)

### AI Project Analyst

Deterministic analysis using live PostgreSQL data — no external LLM calls:

1. Fetches all tasks, members, and activity for the project
2. Computes metrics: overdue count, urgency distribution, workload balance
3. Detects signals: blocked tasks, stale tasks, risk concentration
4. Matches prompt keywords to answer templates
5. Returns structured signals, suggested actions, and a natural-language answer

### Database

PostgreSQL 16 with pgx/v5 connection pooling:

- Full-text search via generated `tsvector` column (`search_vector`)
- JSONB metadata for flexible activity audit trails
- Proper foreign keys with `ON DELETE CASCADE`
- Check constraints for role and priority values
- Indexed hot paths (tasks, activity, notifications, search)

### Worker

The worker process runs in a separate container and:

1. Polls `email_jobs` table every 10 seconds
2. Logs queued email jobs to stdout (the `RESEND_API_KEY` config is reserved; no Resend API call is currently implemented)
3. Retries failed jobs up to 5 times
4. Runs as a separate binary (`cmd/worker/main.go`)

## Frontend Architecture

### State Management

- **Zustand**: Auth/session state, UI state (task drawer, toasts, command menu)
- **React Query**: Server state (workspaces, projects, board, members, activity) with
  automatic cache invalidation after mutations and SSE-triggered refetches

### Data Flow

1. API client (`api/client.ts`) attaches JWT via interceptor
2. 401 responses trigger automatic refresh token rotation
3. React Query manages caching, background refetching, and optimistic updates
4. SSE events invalidate board queries for real-time updates
5. Zustand stores handle UI-local state only

### Key Libraries

- **@hello-pangea/dnd**: Drag-and-drop Kanban board
- **lucide-react**: Icon library
- **date-fns**: Date formatting (available in dependencies)
- **clsx**: Conditional class names (available in dependencies)

## Deployment

See [DEPLOYMENT.md](DEPLOYMENT.md) for production deployment instructions.
See `docker-compose.yml` for local development setup.
See `docker-compose.prod.yml` for production deployment.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `REDIS_URL` | Yes | — | Redis connection string |
| `JWT_SECRET` | Yes | — | At least 32 random bytes |
| `FRONTEND_URL` | Yes | — | Public frontend URL (CORS) |
| `AWS_ACCESS_KEY_ID` | S3 | — | S3-compatible storage access key |
| `AWS_SECRET_ACCESS_KEY` | S3 | — | S3-compatible storage secret key |
| `S3_BUCKET` | S3 | — | Bucket name for attachments |
| `S3_ENDPOINT` | No | — | Custom S3 endpoint (empty = AWS) |
| `RESEND_API_KEY` | No | — | Transactional email API key |
| `FROM_EMAIL` | No | — | Sender email address |
| `PORT` | No | `8080` | Backend listen port |
| `VITE_API_URL` | No | — | Frontend API base URL |
