# Trello Clone

A Trello-like project management app built with **Next.js** (frontend) and **Go** (API), backed by PostgreSQL and Redis.

## Features

- Workspaces, boards, lists, tasks, and comments
- Drag-and-drop Kanban board (`@dnd-kit`)
- Real-time updates via WebSockets + Redis pub/sub
- Workspace member invites with roles (owner, admin, member)
- Labels, assignees, and due dates on tasks
- Email/password auth + Google/GitHub OAuth

## Quick start

### Prerequisites

- Docker & Docker Compose
- Go 1.22+ (for local API dev without Docker)
- Node.js 22+ (for local frontend dev)

### Run with Docker

```bash
cp .env.example .env
docker compose up --build
```

- Frontend: http://localhost:3000
- API: http://localhost:8080
- Health: http://localhost:8080/health

### Run locally (without Docker for app services)

```bash
# Start infrastructure
docker compose up postgres redis -d

# Backend
cd backend
go run ./cmd/server

# Frontend (separate terminal)
cd frontend
npm run dev
```

## API overview

| Area | Endpoints |
|------|-----------|
| Auth | `POST /api/v1/auth/register`, `/login`, `/refresh`, `GET /auth/me` |
| OAuth | `GET /api/v1/auth/google`, `/auth/github` (+ callbacks) |
| Workspaces | CRUD + members + invitations |
| Boards | CRUD, lists, tasks, comments, labels, assignees |
| Real-time | `WS /ws?board_id=...&token=...` |

## Project structure

```
trello-clone/
├── frontend/     # Next.js 15 App Router
├── backend/      # Go API (chi, pgx, WebSockets)
├── docker-compose.yml
└── README.md
```

## Environment variables

See [`.env.example`](.env.example). Set OAuth client IDs/secrets to enable Google/GitHub login.
