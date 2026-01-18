# Project Setup Guide

## Quick Start

```bash
# 1. Clone the repository
git clone <repo-url>
cd capstone

# 2. Install dependencies
make deps

# 3. Set up environment variables (optional - has defaults)
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/capstone?sslmode=disable"

# 4. Build the project
make build

# 5. Run the API
./build/api

# Or run the Worker
./build/worker
```

## Project Structure

```
capstone/
├── cmd/                    # Application entry points
│   ├── api/               # API server
│   └── worker/            # Background worker
├── internal/              # Private application code
│   ├── api/              # API-specific code & config
│   ├── worker/           # Worker-specific code & config
│   └── config/           # Shared configuration
├── pkg/                   # Public libraries
│   └── migrations/       # Migration helper
├── migrations/            # Database migrations
├── web/                   # Frontend application
└── docs/                  # Documentation
```

## Configuration

The project uses `envconfig` for environment-based configuration.

### Configuration Architecture

- **Shared Config** (`internal/config/`): Database, environment, logging
- **API Config** (`internal/api/config.go`): API-specific settings
- **Worker Config** (`internal/worker/config.go`): Worker-specific settings

See [config-structure.md](config-structure.md) for detailed architecture.

### Environment Variables

#### Shared (Both API & Worker)

| Variable | Description | Default |
|----------|-------------|---------|
| `ENVIRONMENT` | Environment name | `development` |
| `LOG_LEVEL` | Logging level | `info` |
| `DATABASE_URL` | PostgreSQL connection | `postgres://postgres:postgres@localhost:5432/capstone?sslmode=disable` |
| `DB_MAX_OPEN_CONNS` | Max open connections | `25` |
| `DB_MAX_IDLE_CONNS` | Max idle connections | `5` |
| `MIGRATIONS_PATH` | Migration files path | `./migrations` |

#### API-Specific

| Variable | Description | Default |
|----------|-------------|---------|
| `API_PORT` | HTTP server port | `8080` |
| `API_HOST` | HTTP server host | `0.0.0.0` |
| `API_ALLOWED_ORIGINS` | CORS origins | `*` |

#### Worker-Specific

| Variable | Description | Default |
|----------|-------------|---------|
| `WORKER_COUNT` | Concurrent workers | `5` |
| `WORKER_QUEUE_NAME` | Queue name | `default` |
| `WORKER_POLL_INTERVAL` | Poll interval (sec) | `5` |
| `WORKER_MAX_RETRIES` | Max retries | `3` |
| `WORKER_SHUTDOWN_TIMEOUT` | Shutdown timeout (sec) | `30` |

## Database Migrations

Migrations run automatically when the application starts.

### Creating Migrations

```bash
make migrate-create name=add_users_table
```

This creates:
- `migrations/000002_add_users_table.up.sql`
- `migrations/000002_add_users_table.down.sql`

See [migrations/README.md](../migrations/README.md) for details.

## Development

### Building

```bash
# Build both API and Worker
make build

# Build individually
make build-api
make build-worker
```

### Testing

```bash
# Run all tests
make test

# Run tests for specific package
go test ./internal/api/...
```

### Code Quality

```bash
# Format code
make fmt

# Tidy dependencies
make tidy

# Clean build artifacts
make clean
```

## Running the Applications

### API Server

```bash
# With defaults
./build/api

# With custom config
export API_PORT=9000
export DATABASE_URL="postgres://user:pass@localhost:5432/mydb"
./build/api
```

### Worker

```bash
# With defaults
./build/worker

# With custom config
export WORKER_COUNT=10
export WORKER_QUEUE_NAME="high-priority"
./build/worker
```

## CI/CD

The project uses GitHub Actions with two workflows:

1. **Test Workflow** (`.github/workflows/test.yml`)
   - Runs on push/PR to main
   - Checks code formatting
   - Runs tests

2. **Build Workflow** (`.github/workflows/build.yml`)
   - Runs on push/PR to main
   - Builds both binaries
   - Uploads artifacts

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make help` | Show available targets |
| `make build` | Build both API and Worker |
| `make build-api` | Build API only |
| `make build-worker` | Build Worker only |
| `make test` | Run tests |
| `make fmt` | Format code |
| `make tidy` | Tidy dependencies |
| `make deps` | Download dependencies |
| `make clean` | Remove build artifacts |
| `make migrate-create name=X` | Create new migration |

## Troubleshooting

### Database Connection Issues

```bash
# Test database connection
psql $DATABASE_URL

# Check if PostgreSQL is running
pg_isready -h localhost -p 5432
```

### Migration Issues

```bash
# Check migrations directory
ls -la migrations/

# Verify MIGRATIONS_PATH
export MIGRATIONS_PATH="./migrations"
./build/api
```

### Build Issues

```bash
# Clean and rebuild
make clean
make deps
make build
```

## Next Steps

1. **Set up PostgreSQL** - Install and configure PostgreSQL
2. **Create your first migration** - `make migrate-create name=init_schema`
3. **Implement API handlers** - Add routes in `internal/api/`
4. **Implement worker jobs** - Add job handlers in `internal/worker/`
5. **Set up frontend** - See `web/README.md`

## Additional Documentation

- [Configuration Structure](config-structure.md)
- [Migrations Guide](../migrations/README.md)
- [Internal Config](../internal/config/README.md)
- [Main README](../README.md)
