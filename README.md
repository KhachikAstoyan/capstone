# Capstone — Code Execution Platform

A secure, low-latency platform for executing and evaluating untrusted user code.
Users submit solutions to problems; sandboxed workers run them in Docker containers
and return per-testcase verdicts.

---

## Table of Contents

- [Capstone — Code Execution Platform](#capstone--code-execution-platform)
  - [Table of Contents](#table-of-contents)
  - [Architecture overview](#architecture-overview)
    - [Services at a glance](#services-at-a-glance)
    - [Why two databases?](#why-two-databases)
  - [Prerequisites](#prerequisites)
    - [macOS (Homebrew)](#macos-homebrew)
    - [Ubuntu / Debian](#ubuntu--debian)
  - [Infrastructure setup](#infrastructure-setup)
    - [PostgreSQL — create databases and user](#postgresql--create-databases-and-user)
    - [RabbitMQ](#rabbitmq)
  - [Database setup](#database-setup)
  - [Environment variables reference](#environment-variables-reference)
    - [API service](#api-service)
    - [Execution Control Plane](#execution-control-plane)
    - [Execution Worker](#execution-worker)
    - [Email Worker](#email-worker)
  - [Complete .env file](#complete-env-file)
  - [Docker images for code execution](#docker-images-for-code-execution)
    - [Adding a new language](#adding-a-new-language)
  - [Running the services](#running-the-services)
    - [API service](#api-service-1)
    - [Execution Control Plane](#execution-control-plane-1)
    - [Execution Worker](#execution-worker-1)
    - [Email Worker (optional)](#email-worker-optional)
    - [Frontend](#frontend)
  - [Startup order](#startup-order)
  - [Makefile reference](#makefile-reference)
  - [Project structure](#project-structure)
    - [Technology stack](#technology-stack)

---

## Architecture overview

```
**Browser**
  │
  ▼
┌─────────────────────┐        ┌────────────────────────────┐
│   API Service       │──────▶│  Execution Control Plane   │
│   :8000             │  HTTP  │  :9090                     │
│  (auth, problems,   │        │  (job queue, worker        │
│   submissions)      │        │   registry, lease mgmt)****    │
└─────────────────────┘        └────────────────────────────┘
         │                                   ▲
         │ RabbitMQ                           │ HTTP poll / report
         ▼                                   │
┌─────────────────────┐        ┌─────────────────────────────┐
│   Email Worker      │        │   Execution Worker(s)        │
│   (email delivery)  │        │   (Docker containers)        │
└─────────────────────┘        └─────────────────────────────┘
         │                                   │
         ▼                                   ▼
┌────────────────────────────────────────────────────────────┐
│   PostgreSQL                                               │
│   • capstone     — API service database                    │
│   • capstone_cp  — Control Plane database (separate)       │
└────────────────────────────────────────────────────────────┘

         +  RabbitMQ  (email verification events only)
```

### Services at a glance

| Binary          | Default port | Responsibility                                         |
| --------------- | ------------ | ------------------------------------------------------ |
| `api`           | 8000         | REST API — auth, problems, submissions, RBAC           |
| `control-plane` | 9090         | Job queue, worker registry, lease management           |
| `worker`        | —            | Polls control plane, runs user code in Docker          |
| `email`         | —            | Consumes RabbitMQ events and sends verification emails |

### Why two databases?

The Execution Control Plane is a separately deployable service that owns all
execution state (`jobs`, `workers`, `job_results`). It does not share a database
with the API service. Communication between the two is purely over HTTP.

---

## Prerequisites

| Tool          | Minimum version | Install                                   |
| ------------- | --------------- | ----------------------------------------- |
| Go            | 1.24            | <https://go.dev/dl/>                      |
| Node.js       | 20              | <https://nodejs.org/>                     |
| PostgreSQL    | 12              | see below                                 |
| RabbitMQ      | 3.x             | see below — only needed for email         |
| Docker Engine | 24              | <https://docs.docker.com/engine/install/> |

### macOS (Homebrew)

```bash
brew install go node postgresql@17 rabbitmq
brew install --cask docker          # Docker Desktop

brew services start postgresql@17
brew services start rabbitmq
```

### Ubuntu / Debian

```bash
# Go — download the tarball from https://go.dev/dl/ and follow instructions

# Node
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

# PostgreSQL
sudo apt-get install -y postgresql-17

# RabbitMQ
sudo apt-get install -y rabbitmq-server

# Docker — follow https://docs.docker.com/engine/install/ubuntu/
```

---

## Infrastructure setup

### PostgreSQL — create databases and user

Connect as a superuser and run:

```sql
CREATE USER capstone WITH PASSWORD 'capstone';

CREATE DATABASE capstone    OWNER capstone;   -- API service database
CREATE DATABASE capstone_cp OWNER capstone;   -- Control Plane database
```

One-liner:

```bash
psql -U postgres -c "CREATE USER capstone WITH PASSWORD 'capstone';"
psql -U postgres -c "CREATE DATABASE capstone    OWNER capstone;"
psql -U postgres -c "CREATE DATABASE capstone_cp OWNER capstone;"
```

Verify:

```bash
psql -U capstone -d capstone    -c '\conninfo'
psql -U capstone -d capstone_cp -c '\conninfo'
```

### RabbitMQ

No manual setup required. The API service and email worker declare the exchange
and queue automatically when they connect.

To inspect messages, enable the management plugin and open
`http://localhost:15672` (user `guest`, password `guest`):

```bash
rabbitmq-plugins enable rabbitmq_management
```

---

## Database setup

**Migrations run automatically on startup** — you do not need to run them
manually. Each service connects to its database, applies any pending migrations,
and then starts serving.

| Service       | Migrations directory                | Target database |
| ------------- | ----------------------------------- | --------------- |
| API           | `internal/api/migrations/`          | `capstone`      |
| Control Plane | `internal/controlplane/migrations/` | `capstone_cp`   |

If a migration fails (e.g. the database does not exist), the service exits
immediately with a clear error message.

---

## Environment variables reference

### API service

| Variable                                      | Required | Default                     | Description                                                 |
| --------------------------------------------- | :------: | --------------------------- | ----------------------------------------------------------- |
| `API_DATABASE_URL`                            |    ✅    | —                           | PostgreSQL DSN for the `capstone` database                  |
| `JWT_SECRET`                                  |    ✅    | —                           | Secret used to sign and verify JWTs                         |
| `API_PORT`                                    |          | `8080`                      | HTTP listen port                                            |
| `API_HOST`                                    |          | `0.0.0.0`                   | HTTP listen address                                         |
| `API_ALLOWED_ORIGINS`                         |          | `*`                         | Comma-separated CORS origins (e.g. `http://localhost:5173`) |
| `API_SECURE_COOKIES`                          |          | `false`                     | Set `true` in production (requires HTTPS)                   |
| `API_FRONTEND_URL`                            |          | `http://localhost:5173`     | SPA origin; used to build email verification links          |
| `API_MIGRATIONS_PATH`                         |          | `./internal/api/migrations` | Path to SQL migration files                                 |
| `JWT_ACCESS_TOKEN_DURATION`                   |          | `900`                       | Access token TTL in seconds (15 min)                        |
| `JWT_REFRESH_TOKEN_DURATION`                  |          | `604800`                    | Refresh token TTL in seconds (7 days)                       |
| `API_RABBITMQ_URL`                            |          | —                           | AMQP URL. If unset, email publishing is silently skipped    |
| `API_RABBITMQ_EXCHANGE`                       |          | `capstone.events`           | Topic exchange name                                         |
| `API_RABBITMQ_EMAIL_VERIFICATION_ROUTING_KEY` |          | `email.verification`        | Routing key for verification emails                         |
| `ENVIRONMENT`                                 |          | `development`               | `development` \| `production` (affects log format)          |

---

### Execution Control Plane

| Variable                       | Required | Default                              | Description                                                                                                                                                           |
| ------------------------------ | :------: | ------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `CP_DATABASE_URL`              |    ✅    | —                                    | PostgreSQL DSN for the `capstone_cp` database                                                                                                                         |
| `CP_PORT`                      |          | `9090`                               | HTTP listen port                                                                                                                                                      |
| `CP_HOST`                      |          | `0.0.0.0`                            | HTTP listen address                                                                                                                                                   |
| `CP_INTERNAL_KEY`              |          | —                                    | Shared secret checked in the `X-Internal-Key` request header. **If unset, all requests are accepted without auth** (development only — always set this in production) |
| `CP_MIGRATIONS_PATH`           |          | `./internal/controlplane/migrations` | Path to SQL migration files                                                                                                                                           |
| `CP_LEASE_DURATION_SEC`        |          | `60`                                 | Seconds a worker has to complete or renew a job before it is requeued                                                                                                 |
| `CP_LEASE_CHECK_INTERVAL_SEC`  |          | `10`                                 | How often (seconds) the background goroutine sweeps for expired leases                                                                                                |
| `CP_HEARTBEAT_TIMEOUT_SEC`     |          | `30`                                 | Worker is marked `offline` after this many seconds without a heartbeat                                                                                                |
| `CP_WORKER_SWEEP_INTERVAL_SEC` |          | `15`                                 | How often (seconds) the background goroutine sweeps for stale workers                                                                                                 |
| `ENVIRONMENT`                  |          | `development`                        | Log verbosity                                                                                                                                                         |

---

### Execution Worker

| Variable                            | Required | Default       | Description                                                                                                              |
| ----------------------------------- | :------: | ------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `WORKER_CP_URL`                     |    ✅    | —             | Base URL of the Control Plane (e.g. `http://localhost:9090`)                                                             |
| `WORKER_CP_KEY`                     |          | —             | Must match `CP_INTERNAL_KEY` when auth is enabled                                                                        |
| `WORKER_LANGUAGES`                  |          | `python,javascript,go` | Comma-separated languages this worker supports. Supported values: `python`, `javascript`, `go`                         |
| `WORKER_CAPACITY`                   |          | `1`           | Maximum concurrent jobs                                                                                                  |
| `WORKER_ALLOW_STUB_EXECUTOR`        |          | `false`       | Permit dev-only stub fallback if Docker is unavailable. Keep `false` outside local tests                                |
| `WORKER_ID`                         |          | random UUID   | Stable identifier. Reusing the same ID across restarts resumes the existing registry row instead of creating a duplicate |
| `WORKER_HEARTBEAT_INTERVAL_SEC`     |          | `10`          | Heartbeat frequency in seconds                                                                                           |
| `WORKER_POLL_INTERVAL_SEC`          |          | `2`           | Poll frequency in seconds (only when `active_jobs < capacity`)                                                           |
| `WORKER_LEASE_RENEWAL_INTERVAL_SEC` |          | `20`          | How often to extend the lease while a job is running                                                                     |
| `ENVIRONMENT`                       |          | `development` | Log verbosity                                                                                                            |

---

### Email Worker

| Variable                                        | Required | Default                       | Description                                         |
| ----------------------------------------------- | :------: | ----------------------------- | --------------------------------------------------- |
| `EMAIL_RABBITMQ_URL`                            |    ✅    | —                             | AMQP URL — must point to the same broker as the API |
| `EMAIL_RABBITMQ_EXCHANGE`                       |          | `capstone.events`             | Must match `API_RABBITMQ_EXCHANGE`                  |
| `EMAIL_RABBITMQ_EMAIL_VERIFICATION_ROUTING_KEY` |          | `email.verification`          | Must match the API routing key                      |
| `EMAIL_RABBITMQ_QUEUE`                          |          | `capstone.email.verification` | Durable queue name                                  |
| `ENVIRONMENT`                                   |          | `development`                 | Log verbosity                                       |

---

## Complete .env file

The file `capstone-code/.env` ships with local-dev defaults. Load it before
starting any service:

```bash
cd capstone-code
source .env
```

Full contents:

```bash
# ── API service ──────────────────────────────────────────────────────────────
export API_DATABASE_URL="postgresql://capstone:capstone@localhost:5432/capstone?sslmode=disable"
export API_PORT="8000"
export API_ALLOWED_ORIGINS="http://localhost:5173,http://localhost:5174"
export API_FRONTEND_URL="http://localhost:5173"
export API_RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
export API_SECURE_COOKIES="false"
export JWT_SECRET="BLABLABLA"   # ← replace with a strong random secret

# ── Email Worker ──────────────────────────────────────────────────────────────
export EMAIL_RABBITMQ_URL="amqp://guest:guest@localhost:5672/"

# ── Execution Control Plane ──────────────────────────────────────────────────
export CP_DATABASE_URL="postgresql://capstone:capstone@localhost:5432/capstone_cp?sslmode=disable"
export CP_PORT="9090"
export CP_INTERNAL_KEY="dev-internal-key"   # ← replace in production
export CP_LEASE_DURATION_SEC="60"
export CP_LEASE_CHECK_INTERVAL_SEC="10"
export CP_HEARTBEAT_TIMEOUT_SEC="30"
export CP_WORKER_SWEEP_INTERVAL_SEC="15"

# ── Execution Worker ──────────────────────────────────────────────────────────
export WORKER_CP_URL="http://localhost:9090"
export WORKER_CP_KEY="dev-internal-key"     # ← must match CP_INTERNAL_KEY
export WORKER_LANGUAGES="python,javascript,go"
export WORKER_ALLOW_STUB_EXECUTOR="false"
export WORKER_CAPACITY="1"
export WORKER_HEARTBEAT_INTERVAL_SEC="10"
export WORKER_POLL_INTERVAL_SEC="2"
export WORKER_LEASE_RENEWAL_INTERVAL_SEC="20"
```

> **Production secrets:** generate strong values with `openssl rand -hex 32`.
> Never commit real secrets to git.

---

## Docker images for code execution

The worker uses local Docker images as language runtimes. Build them once before
running the worker:

```bash
./docker/build-images.sh
```

This builds `capstone-python-runner:latest`, `capstone-js-runner:latest`, and
`capstone-go-runner:latest` from the Dockerfiles in `docker/`. To override the
tag, run `./docker/build-images.sh --tag <tag>`.

Each job container is launched with these security constraints:

- **No network** — `NetworkMode: none`
- **Read-only root filesystem** — source files are bind-mounted read-only
- **Memory cap** — set to the job's `memory_limit_mb`
- **No swap** — `MemorySwap` equals `Memory`
- **PID limit 64** — prevents fork bombs
- **All Linux capabilities dropped**
- **No new privileges** — `no-new-privileges` security option
- **Tmpfs at `/tmp`** — 64 MB in-memory scratch space only

If Docker is not running when the worker starts, startup fails by default. A
stub executor (returns `"Accepted"` for every job) is available only when
`WORKER_ALLOW_STUB_EXECUTOR=true`, and should be used for local control-plane
testing only.

### Adding a new language

Add an entry to `DefaultLanguages` in
[internal/worker/docker_executor.go](internal/worker/docker_executor.go):

```go
"cpp": {
    Image:          "gcc:13-alpine",
    SourceFile:     "solution.cpp",
    CompileCmd:     []string{"g++", "-O2", "-o", "/workspace/solution", "/workspace/solution.cpp"},
    RunnerFile:     "runner.sh",
    RunnerTemplate: cppRunnerTemplate,
    RunCmd:         []string{"sh", "/workspace/runner.sh"},
},
```

Then add the runner template string and Dockerfile, and build the runner images:

```bash
./docker/build-images.sh
```

---

## Running the services

All commands assume you are in `capstone-code/` with the environment loaded.

```bash
cd capstone-code
source .env
```

### API service

```bash
make run-api
```

First run applies all migrations to `capstone` and seeds RBAC roles and
permissions. Logs confirm: `"Migrations completed successfully"` and
`"RBAC seeding completed successfully"`.

```bash
curl http://localhost:8000/       # → "sup"
```

### Execution Control Plane

```bash
make run-control-plane
```

First run applies all migrations to `capstone_cp`. Logs confirm:
`"migrations complete"` and `"control plane ready"`.

```bash
curl http://localhost:9090/healthz    # → "ok"
```

### Execution Worker

Docker must be running before starting the worker.

```bash
# Pre-pull language images (one-time)
docker pull python:3.11-alpine
docker pull node:20-alpine

go run ./cmd/worker
```

The worker logs show:

1. `"starting worker"` — configuration loaded
2. `"docker executor ready"` — Docker daemon reachable (or a fallback warning)
3. Heartbeat sent to the control plane

### Email Worker (optional)

Only required if email verification needs to actually deliver emails.

```bash
make run-email
```

### Frontend

```bash
cd web
npm install
npm run dev     # → http://localhost:5173
```

---

## Startup order

Start services in this sequence. Later services depend on earlier ones being
healthy.

```
1.  PostgreSQL         ← must be running before API and Control Plane
2.  RabbitMQ           ← must be running before API and Email Worker
3.  API service        ← runs DB migrations on startup; frontend needs it
4.  Control Plane      ← runs DB migrations on startup; worker needs it
5.  Worker             ← polls Control Plane; Docker must be running
6.  Email Worker       ← optional; needs RabbitMQ
7.  Frontend           ← connects to API at :8000
```

Recommended setup using four terminal tabs:

```bash
# Tab 1 — API
cd capstone-code && source .env && make run-api

# Tab 2 — Control Plane
cd capstone-code && source .env && make run-control-plane

# Tab 3 — Worker
cd capstone-code && source .env && go run ./cmd/worker

# Tab 4 — Frontend
cd capstone-code/web && npm run dev
```

---

## Makefile reference

Run all `make` commands from `capstone-code/`.

| Target                     | Description                               |
| -------------------------- | ----------------------------------------- |
| `make build-all`           | Compile all four binaries into `./build/` |
| `make build-api`           | Compile `./build/api`                     |
| `make build-control-plane` | Compile `./build/control-plane`           |
| `make build-worker`        | Compile `./build/worker`                  |
| `make build-email`         | Compile `./build/email`                   |
| `make run-api`             | Run API with `go run` (no build step)     |
| `make run-control-plane`   | Run Control Plane with `go run`           |
| `make run-email`           | Run Email Worker with `go run`            |
| `make test`                | Run all Go tests (`go test ./...`)        |
| `make fmt`                 | Format all Go code with `gofmt`           |
| `make tidy`                | Run `go mod tidy`                         |
| `make deps`                | Download all Go module dependencies       |
| `make clean`               | Remove the `./build/` directory           |

---

## Project structure

```
capstone-code/
│
├── cmd/
│   ├── api/                      API server binary
│   │   ├── main.go               — wires all dependencies and starts the server
│   │   └── routes.go             — registers all HTTP routes
│   ├── control-plane/            Execution Control Plane binary
│   │   ├── main.go               — background sweeps, auth middleware, startup
│   │   └── routes.go             — Chi route tree
│   ├── worker/                   Execution worker binary
│   │   └── main.go               — wires client + executor, runs Worker.Run()
│   └── email/                    Email notification worker binary
│       └── main.go
│
├── internal/
│   ├── api/                      API domain modules (layered per domain)
│   │   ├── auth/                 — register, login, JWT, refresh, email verify
│   │   ├── problems/             — problem CRUD, test groups, testcases
│   │   ├── rbac/                 — roles, permissions, middleware
│   │   ├── tags/                 — problem tagging
│   │   ├── migrations/           — SQL migration files for the API database
│   │   └── config.go             — API environment variable config
│   │
│   ├── controlplane/             Execution Control Plane module
│   │   ├── domain/
│   │   │   └── models.go         — Job, Worker, Assignment, all request types
│   │   ├── repository/
│   │   │   ├── job_repo.go       — job queue SQL (SKIP LOCKED assignment)
│   │   │   └── worker_repo.go    — worker registry SQL
│   │   ├── service/
│   │   │   └── service.go        — scheduling, lease logic, background sweeps
│   │   ├── http/
│   │   │   ├── handler.go        — Handler struct + JSON helpers
│   │   │   ├── jobs.go           — API-facing endpoints (create, get, result)
│   │   │   └── workers.go        — worker-facing endpoints (heartbeat, poll, lease, result)
│   │   ├── migrations/           — SQL migration files for the CP database
│   │   └── config.go             — CP environment variable config
│   │
│   ├── worker/                   Execution worker internals
│   │   ├── config.go             — WORKER_* environment variable config
│   │   ├── client.go             — HTTP client wrapping all control plane calls
│   │   ├── executor.go           — Executor interface + StubExecutor
│   │   ├── docker_executor.go    — Docker-based sandboxed code execution
│   │   └── worker.go             — heartbeat / poll / per-job / lease-renewal loops
│   │
│   ├── email/                    Email worker internals
│   └── config/                   Shared config types (CommonConfig)
│
├── pkg/
│   ├── database/                 PostgreSQL connection helper (MustConnect)
│   ├── logger/                   Zap logger initialisation
│   ├── migrations/               golang-migrate runner (RunMigrations)
│   ├── rabbitmq/                 AMQP publisher / consumer helpers
│   └── permissions/              RBAC permission key constants
│
├── web/                          Main frontend (TanStack Start + React 19)
│   └── src/routes/               — file-based TanStack Router pages
│
├── src/                          Standalone Monaco code editor (React 18 + Vite)
│
├── db/ddl.sql                    Reference schema DDL (not used at runtime)
├── .env                          Local environment variables — source before running
├── go.mod / go.sum
└── Makefile
```

### Technology stack

| Layer                  | Technology                     |
| ---------------------- | ------------------------------ |
| Backend                | Go 1.24                        |
| HTTP router            | Chi v5                         |
| Database               | PostgreSQL 12+                 |
| Migrations             | golang-migrate                 |
| Auth                   | JWT — golang-jwt/jwt v5        |
| Async messaging        | RabbitMQ — amqp091-go          |
| Structured logging     | Uber Zap                       |
| Code execution sandbox | Docker — docker/docker SDK v28 |
| Frontend               | TanStack Start + React 19      |
| Styling                | Tailwind CSS 4                 |
| UI components          | Radix UI / shadcn              |
| Code editor            | Monaco Editor                  |
| Frontend tests         | Vitest                         |
