# Planly: AI Work Planner (Go + React + PostgreSQL, Microservices)

A work planner for students, undergraduates and employees, with an agentic AI assistant that plans, schedules and reschedules on the user's behalf (with approval).

---

## 1. Scope (realistic for one person)

**In scope (MVP):** auth, projects, tasks (priority, due date, status, subtasks, tags), calendar time blocks, reminders, in-app notifications, AI agent chat with tool use, AI goal-to-plan breakdown, AI reschedule proposals, daily briefing, persona modes.
**Out of scope:** teams/sharing, mobile app, third-party calendar sync, payments, self-hosted ML, vector search, file attachments.

## 2. Tech Stack

| Layer | Choice |
|---|---|
| Language | Go 1.23+ |
| HTTP router | chi |
| DB access | pgx + sqlc |
| Migrations | golang-migrate |
| Auth | JWT (golang-jwt), bcrypt |
| Config/logging | env vars (envconfig), `log/slog` |
| Database | PostgreSQL 16 (one server, **one database per service**) |
| Frontend | React 18 + TypeScript + Vite, TanStack Query, React Router, Tailwind, FullCalendar |
| LLM | Anthropic Go SDK behind an `LLMClient` interface (swappable, mockable) |
| Testing | `testing` + testify, testcontainers-go, httptest, Vitest + React Testing Library |
| Packaging | Docker + Docker Compose, GitHub Actions CI |

## 3. Architecture

```
React SPA (Vite)
      |  HTTPS / JSON
      v
 api-gateway  :8080   (JWT check, routing, CORS, rate limit)
   |        |          |           |
   v        v          v           v
auth-svc  planner-svc  notify-svc  agent-svc
 :8081     :8082        :8083       :8084
   |        |            |           |
 auth_db  planner_db   notify_db   agent_db     (PostgreSQL)

agent-svc  --HTTP (user's JWT forwarded)-->  planner-svc   (tools)
agent-svc  --HTTPS-->  LLM provider
planner-svc --HTTP (internal key)--> notify-svc  (create reminders)
notify-svc worker: polls due reminders every 30s -> creates notifications
agent-svc scheduler: daily briefing per user -> calls LLM -> posts notification via notify-svc
```

**Rules**
1. Each service owns its database. No cross-service SQL, no shared tables.
2. Gateway validates the JWT and forwards `X-User-ID` and the original `Authorization` header. Services also verify the JWT (defence in depth) using the shared public key/secret.
3. Service-to-service calls without a user context use `X-Internal-Key`.
4. Synchronous REST between services (simple and debuggable). No message broker for MVP.
5. All IDs are UUIDs. All timestamps are UTC (`timestamptz`); the client converts to the user's timezone.
6. Every service exposes `GET /healthz` and `GET /readyz`.

### Service responsibilities

| Service | Responsibility |
|---|---|
| **api-gateway** | Reverse proxy, JWT validation, rate limiting, CORS, request ID |
| **auth-svc** | Register, login, refresh, profile (persona, timezone, working hours) |
| **planner-svc** | Projects, tasks, subtasks, tags, time blocks, workload summary |
| **notify-svc** | Reminders, notification inbox, background due-reminder worker |
| **agent-svc** | Conversations, agent loop, tool registry, proposals (human approval), daily briefing |

### Agent design (core of the portfolio)

- **Loop:** user message → LLM with tool definitions → if tool call, execute → append result → repeat (max 8 iterations) → final answer.
- **Read tools (run immediately):** `list_tasks`, `get_task`, `list_projects`, `list_time_blocks`, `get_workload`.
- **Write tools (never run directly):** `create_task`, `update_task`, `delete_task`, `create_time_block`, `reschedule_tasks`. These create a **proposal** (stored JSON of the action list). The UI shows a diff; the user clicks Approve, and only then does agent-svc execute the actions against planner-svc.
- **Safety:** tool inputs validated against JSON schema; max iterations; per-user rate limit; all calls recorded in `agent_runs` and `tool_calls` for traceability.
- **System prompt** includes: persona, timezone, working hours, current date/time, rules ("never invent task IDs, always propose changes").

## 4. Functional Requirements

**Auth (FR-A)**
- A1 Register with email/password; A2 login returns access (15 min) + refresh (7 days) tokens; A3 refresh; A4 view/update profile (name, persona: `student|undergraduate|employee`, timezone, work start/end hours).

**Planner (FR-P)**
- P1 CRUD projects (name, colour, archived).
- P2 CRUD tasks: title, description, project, priority (1-4), status (`todo|in_progress|done`), due date, estimate (minutes), tags.
- P3 Subtasks (checklist items on a task).
- P4 Filter/sort/search tasks (status, project, priority, due range, text).
- P5 Time blocks: scheduled slots (start/end) optionally linked to a task; reject overlaps.
- P6 Workload summary: planned minutes per day for a date range, flagging days over the user's daily capacity.
- P7 Dashboard data: today, overdue, upcoming 7 days.

**Notifications (FR-N)**
- N1 Create reminder for a task at a given time; N2 worker fires due reminders into the inbox; N3 list notifications, mark read; N4 unread count.

**Agent (FR-G)**
- G1 Chat with the assistant (streaming optional, plain responses acceptable for MVP).
- G2 "Plan my goal": goal + deadline → proposed tasks with estimates and due dates.
- G3 "Reschedule": overdue/missed tasks → proposed new dates respecting capacity.
- G4 Proposal review: approve all, reject, or approve selected actions.
- G5 Daily briefing generated each morning at the user's work start time.
- G6 Run history (what tools were called) visible to the user.

**Frontend pages:** Login/Register, Dashboard, Tasks (list + filters), Task detail drawer, Projects, Calendar, Assistant (chat + proposal cards), Notifications, Settings/Profile.

## 5. Non-Functional Requirements

| ID | Requirement |
|---|---|
| NFR-1 Performance | CRUD endpoints p95 < 300 ms locally; list endpoints paginated (default 20, max 100) |
| NFR-2 Security | bcrypt cost 12, JWT expiry, input validation, parameterised SQL (sqlc), CORS allow-list, rate limiting, secrets only via env vars, no secrets in git |
| NFR-3 Data isolation | Every query filters by `user_id`; tested with a "user B cannot read user A" test per endpoint |
| NFR-4 Reliability | Graceful shutdown, DB retry on startup, timeouts on all outbound HTTP (5s internal, 60s LLM), LLM failure returns a clear error without corrupting data |
| NFR-5 Observability | Structured JSON logs with request ID propagated across services; `/healthz`, `/readyz` |
| NFR-6 Testability | Unit tests for business logic, integration tests with real Postgres (testcontainers), LLM mocked via interface; target 70%+ coverage on service layers |
| NFR-7 Maintainability | Same layered layout in each service (handler → service → repository), migrations versioned, API documented in OpenAPI |
| NFR-8 Usability | Responsive UI, keyboard-friendly forms, loading and error states on every screen |
| NFR-9 Cost control | Agent max iterations, max tokens, per-user daily message cap |
| NFR-10 Portability | `docker compose up` starts everything |

## 6. Database Schema

### auth_db
```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  full_name TEXT NOT NULL,
  persona TEXT NOT NULL DEFAULT 'employee' CHECK (persona IN ('student','undergraduate','employee')),
  timezone TEXT NOT NULL DEFAULT 'UTC',
  work_start TIME NOT NULL DEFAULT '09:00',
  work_end TIME NOT NULL DEFAULT '17:00',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE refresh_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### planner_db
```sql
CREATE TABLE projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  name TEXT NOT NULL,
  colour TEXT NOT NULL DEFAULT '#3b82f6',
  archived BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_projects_user ON projects(user_id);

CREATE TABLE tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  priority SMALLINT NOT NULL DEFAULT 2 CHECK (priority BETWEEN 1 AND 4),
  status TEXT NOT NULL DEFAULT 'todo' CHECK (status IN ('todo','in_progress','done')),
  due_at TIMESTAMPTZ,
  estimate_minutes INT CHECK (estimate_minutes > 0),
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_tasks_user_status ON tasks(user_id, status);
CREATE INDEX idx_tasks_user_due ON tasks(user_id, due_at);

CREATE TABLE subtasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  done BOOLEAN NOT NULL DEFAULT false,
  position INT NOT NULL DEFAULT 0
);

CREATE TABLE tags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  name TEXT NOT NULL,
  UNIQUE (user_id, name)
);
CREATE TABLE task_tags (
  task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
  tag_id UUID REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (task_id, tag_id)
);

CREATE TABLE time_blocks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  task_id UUID REFERENCES tasks(id) ON DELETE SET NULL,
  title TEXT NOT NULL,
  starts_at TIMESTAMPTZ NOT NULL,
  ends_at TIMESTAMPTZ NOT NULL,
  CHECK (ends_at > starts_at)
);
CREATE INDEX idx_blocks_user_start ON time_blocks(user_id, starts_at);
```

### notify_db
```sql
CREATE TABLE reminders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  task_id UUID,
  message TEXT NOT NULL,
  remind_at TIMESTAMPTZ NOT NULL,
  fired BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_reminders_due ON reminders(remind_at) WHERE fired = false;

CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('reminder','briefing','agent','system')),
  title TEXT NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  read BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_notif_user ON notifications(user_id, read, created_at DESC);
```

### agent_db
```sql
CREATE TABLE conversations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  title TEXT NOT NULL DEFAULT 'New chat',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  role TEXT NOT NULL CHECK (role IN ('user','assistant')),
  content JSONB NOT NULL,           -- provider-format content blocks
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE agent_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  trigger TEXT NOT NULL CHECK (trigger IN ('chat','briefing','plan_goal','reschedule')),
  status TEXT NOT NULL CHECK (status IN ('running','completed','failed')),
  iterations INT NOT NULL DEFAULT 0,
  input_tokens INT NOT NULL DEFAULT 0,
  output_tokens INT NOT NULL DEFAULT 0,
  error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE tool_calls (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id UUID NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
  tool_name TEXT NOT NULL,
  input JSONB NOT NULL,
  output JSONB,
  is_error BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE proposals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  run_id UUID REFERENCES agent_runs(id) ON DELETE SET NULL,
  summary TEXT NOT NULL,
  actions JSONB NOT NULL,           -- [{id, tool, input, status}]
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected','partial','failed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  decided_at TIMESTAMPTZ
);
CREATE TABLE briefing_log (
  user_id UUID NOT NULL,
  briefing_date DATE NOT NULL,
  PRIMARY KEY (user_id, briefing_date)   -- prevents duplicate briefings
);
```

## 7. API Summary (via gateway, prefix `/api/v1`)

| Service | Endpoints |
|---|---|
| auth | `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `GET/PUT /auth/me` |
| planner | `/projects` (CRUD), `/tasks` (CRUD + query params), `/tasks/{id}/subtasks`, `/tags`, `/time-blocks` (CRUD), `GET /workload?from&to`, `GET /dashboard` |
| notify | `POST /reminders`, `GET /notifications`, `POST /notifications/{id}/read`, `GET /notifications/unread-count` |
| agent | `POST /agent/chat`, `GET /agent/conversations`, `GET /agent/conversations/{id}`, `POST /agent/plan-goal`, `POST /agent/reschedule`, `GET /agent/proposals`, `POST /agent/proposals/{id}/approve`, `POST /agent/proposals/{id}/reject`, `GET /agent/runs/{id}` |

Error format everywhere: `{"error":{"code":"validation_error","message":"..."}}`.

## 8. Repository Layout

```
planly/
  docker-compose.yml
  .github/workflows/ci.yml
  docs/            (architecture.md, openapi/*.yaml)
  pkg/             (shared: httpx, jwtx, logx, config, dbx)
  services/
    gateway/   auth/   planner/   notify/   agent/
      cmd/server/main.go
      internal/{handler,service,repository,model}
      migrations/
      sqlc.yaml
      Dockerfile
  web/             (React app)
```
Use a Go workspace (`go.work`) so `pkg/` is shared across services.
