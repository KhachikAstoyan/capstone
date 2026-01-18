# API Database Migrations

This directory contains database migration files for the API service using [golang-migrate](https://github.com/golang-migrate/migrate).

## Migration Files

Migration files follow the naming convention:
```
{version}_{description}.up.sql    # Applied when migrating up
{version}_{description}.down.sql  # Applied when rolling back
```

Example:
- `000001_init.up.sql`
- `000001_init.down.sql`

## Creating New Migrations

Create new migration files with the next sequence number:

```bash
# Example: Create a new migration
cd internal/api/migrations

# Find the next number (if last is 000001, use 000002)
# Create both up and down files
touch 000002_add_users_table.up.sql
touch 000002_add_users_table.down.sql
```

## How Migrations Run

Migrations are **automatically run** when the API server starts:

1. API connects to database using `API_DATABASE_URL`
2. Runs all pending migrations from `internal/api/migrations/`
3. Exits with error if migrations fail

## Configuration

### Environment Variables

- `API_DATABASE_URL`: PostgreSQL connection string
  - Default: `postgres://postgres:postgres@localhost:5432/api?sslmode=disable`
- `API_MIGRATIONS_PATH`: Path to migrations directory
  - Default: `./internal/api/migrations`

### Example

```bash
# Set database URL
export API_DATABASE_URL="postgres://user:pass@localhost:5432/api_db?sslmode=disable"

# Run the API (migrations will run automatically)
./build/api
```

## Migration Best Practices

1. **Always test migrations** - Test both up and down migrations
2. **Keep migrations small** - One logical change per migration
3. **Never modify existing migrations** - Create new ones instead
4. **Write reversible migrations** - Always provide a down migration
5. **Be careful with data migrations** - Test with production-like data
6. **Use transactions** - Wrap DDL in transactions when possible

## Example Migration

### Up Migration (000002_add_users_table.up.sql)
```sql
BEGIN;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);

COMMIT;
```

### Down Migration (000002_add_users_table.down.sql)
```sql
BEGIN;

DROP TABLE IF EXISTS users;

COMMIT;
```
