# SynergyFlow — 5-Minute Demo Script

## 1. Login (30s)

Open the app at `http://localhost:55173`.

Use the demo account:
- **Email**: `demo@synergyflow.dev`
- **Password**: `password123`

Callout: *"JWT authentication with refresh token rotation. No setup needed for the demo."*

## 2. Dashboard Health Overview (45s)

After login, the Dashboard shows:

- **KPI bar**: Total tasks, completed, overdue, high priority, unassigned
- **Completion card**: Percentage complete with health badge
- **Workload chart**: Team member assignment distribution
- **Risk queue**: Top 5 riskiest tasks with priority and reason
- **Status distribution**: Donut chart of tasks per column
- **Risk composition**: Donut chart of risk signal breakdown
- **Activity pulse**: Line chart of activity events over the last 7 days
- **Recent activity**: Latest workspace activity feed

Callout: *"Every chart answers a concrete question. Completion ratio, workload balance, risk drivers — all computed from live PostgreSQL data."*

## 3. Open a Project (30s)

Click a project from the sidebar to open the Kanban board.

Callout: *"Five columns: Backlog, Todo, In Progress, In Review, Done. Dense ordering is maintained transactionally on every drag."*

## 4. Move a Task (30s)

Drag a task card from one column to another.

Callout: *"Drag-and-drop uses @hello-pangea/dnd on the frontend. The backend runs a database transaction that closes the source gap, opens the destination gap, and updates the task position — all atomically."*

## 5. Real-time Updates (30s)

Open a second browser window to the same board. Move a task in one window — it updates in the other via SSE.

Callout: *"Server-Sent Events over Redis pub/sub. No polling. Nginx is configured with proxy_buffering off to keep the stream alive. Ping events every 25 seconds prevent proxy timeouts."*

## 6. Task Details (45s)

Click a task card to open the detail drawer:

- Edit title, description, priority, due date
- Change assignee(s)
- Add tags/labels
- Add comments
- Upload file attachments

Callout: *"Inline editing with dirty-state tracking and save confirmation. Comments, activity history, and attachments are all scoped to the task. Uploads go to S3-compatible storage — MinIO in development, AWS S3 in production."*

## 7. Members and Roles (30s)

Open the Members page from the sidebar:

- See workspace roster with role badges
- View open task counts per member
- Invite new members (with role selection)
- Admins can change roles or remove members

Callout: *"Four-tier RBAC: Viewer, Member, Admin, Owner. Invite tokens expire after 7 days. Members are merged with live board assignees for workload visibility."*

## 8. AI Project Analyst (60s)

Click the AI button (bottom-right) to open the analyst panel. Try these prompts:

- **"What is blocking this project?"** — Finds blocker-tagged tasks and overdue risks
- **"Who is overloaded?"** — Shows workload distribution with task counts
- **"What should we work on next?"** — Prioritizes urgent overdue items
- **"Summarize project health"** — Overall health snapshot

Callout: *"Deterministic analysis from live PostgreSQL data. No LLM calls — every signal and suggestion is computed from actual task, member, and activity data. The analysis is reproducible and testable."*

## 9. Notifications (15s)

Click the bell icon in the board header to see notifications. Click a notification to open the linked task.

Callout: *"Notifications link to resources via resource_type + resource_id. Unread counts, mark-all-read, and SSE-triggered refreshes."*

## 10. Architecture Summary (30s)

- **Backend**: Go + Gin + pgx + PostgreSQL + Redis
- **Frontend**: React + TypeScript + Vite + Tailwind
- **State**: React Query (server) + Zustand (UI)
- **Auth**: JWT access tokens + refresh token rotation
- **Realtime**: Redis → SSE fanout
- **Storage**: S3-compatible (MinIO / AWS S3)
- **Deployment**: Docker Compose + Nginx
- **CI**: GitHub Actions (Go tests, frontend build, Docker compose validation)

Callout: *"The full stack is containerized with Docker Compose. One command to start everything: `docker compose up --build`."*

## What This Demonstrates Technically

- Go API with Gin framework and middleware pipeline
- PostgreSQL schema design, migrations, indexed search, transactional operations
- JWT auth with refresh token rotation and session revocation
- Role-based access control with four permission tiers
- Redis-backed SSE for real-time collaboration
- S3-compatible file uploads with presigned URLs
- Deterministic AI analysis engine without external dependencies
- React/TypeScript dashboard with multiple chart types
- Docker/Nginx deployment with health checks
- CI pipeline with Go tests, TypeScript checks, and Docker validation
