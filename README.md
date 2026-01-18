# Capstone

My capstone project for the American University of Armenia

## Project Structure

```
.
├── cmd/
│   ├── api/          # API server entry point
│   └── worker/       # Background worker entry point
├── internal/         # Private application code
│   ├── api/          # API-specific code
│   │   ├── config.go
│   │   └── migrations/  # API database migrations
│   ├── worker/       # Worker-specific code
│   │   └── config.go
│   └── config/       # Shared configuration
├── pkg/              # Public libraries
│   ├── database/     # Database connection helper
│   └── migrations/   # Migration runner
└── web/              # Frontend application
```

## Getting Started

### Prerequisites

- Go 1.24+
- PostgreSQL
- Node.js (for frontend)

### Configuration

The application uses environment variables for configuration. See [internal/config/README.md](internal/config/README.md) for full documentation.

**Note:** API and Worker use **separate databases** and only share domain structs.

#### Shared Configuration

```bash
# Environment
ENVIRONMENT=development
LOG_LEVEL=info
```

#### API Configuration

```bash
# API Server
API_PORT=8080
API_HOST=0.0.0.0
API_ALLOWED_ORIGINS=*

# API Database
API_DATABASE_URL=postgres://postgres:postgres@localhost:5432/api?sslmode=disable
API_MIGRATIONS_PATH=./internal/api/migrations
```

#### Worker Configuration

```bash
# Worker Settings
WORKER_COUNT=5
WORKER_QUEUE_NAME=default
WORKER_POLL_INTERVAL=5
WORKER_MAX_RETRIES=3
WORKER_SHUTDOWN_TIMEOUT=30

# Worker Database (if needed)
WORKER_DATABASE_URL=postgres://postgres:postgres@localhost:5432/worker?sslmode=disable
WORKER_MIGRATIONS_PATH=./internal/worker/migrations
```

### Building

```bash
# Build both API and Worker
make build

# Build individually
make build-api
make build-worker
```

### Running

Both the API and Worker will automatically run pending database migrations on startup.

```bash
# Run API server
./build/api

# Run worker
./build/worker
```

## Database Migrations

Migrations are automatically run when the application starts.

### API Migrations

API migrations are located in `internal/api/migrations/`. See [internal/api/migrations/README.md](internal/api/migrations/README.md) for details.

#### Creating a New API Migration

```bash
cd internal/api/migrations
# Create files with next sequence number
touch 000002_add_users_table.up.sql
touch 000002_add_users_table.down.sql
```

## Development

```bash
# Run tests
make test

# Format code
make fmt

# Tidy dependencies
make tidy

# Clean build artifacts
make clean
```

## CI/CD

The project uses GitHub Actions for:
- **Test workflow**: Runs formatting checks and tests
- **Build workflow**: Builds binaries

Both workflows run on push and pull requests to the `main` branch.
