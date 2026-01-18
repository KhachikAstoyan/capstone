# Configuration Architecture

## Directory Structure

```
internal/
├── config/
│   ├── common.go          # CommonConfig (shared across all apps)
│   ├── database.go        # DatabaseConfig (embedded in CommonConfig)
│   └── README.md
├── api/
│   ├── config.go          # API Config (embeds CommonConfig + API-specific)
│   └── smth.go
└── worker/
    └── config.go          # Worker Config (embeds CommonConfig + Worker-specific)

cmd/
├── api/
│   └── main.go            # Loads api.Config
└── worker/
    └── main.go            # Loads worker.Config
```

## Config Hierarchy

```
┌─────────────────────────────────────────────┐
│          CommonConfig (shared)              │
│  - Environment                              │
│  - LogLevel                                 │
│  - Database (DatabaseConfig)                │
│    - URL                                    │
│    - MaxOpenConns                           │
│    - MaxIdleConns                           │
│    - MigrationsPath                         │
└─────────────────────────────────────────────┘
                    ▲
                    │ embedded
        ┌───────────┴───────────┐
        │                       │
┌───────┴────────┐    ┌────────┴────────┐
│  API Config    │    │  Worker Config  │
│  + ServerPort  │    │  + WorkerCount  │
│  + ServerHost  │    │  + QueueName    │
│  + Timeouts    │    │  + PollInterval │
│  + CORS        │    │  + MaxRetries   │
└────────────────┘    └─────────────────┘
        │                       │
        │                       │
   cmd/api/main.go        cmd/worker/main.go
```

## Loading Flow

### API Server

```go
// 1. Load config in cmd/api/main.go
cfg, err := api.LoadConfig()

// 2. Access shared config
cfg.Environment          // from CommonConfig
cfg.Database.URL         // from DatabaseConfig (via CommonConfig)

// 3. Access API-specific config
cfg.ServerPort          // from API Config
cfg.ServerHost          // from API Config
```

### Worker

```go
// 1. Load config in cmd/worker/main.go
cfg, err := worker.LoadConfig()

// 2. Access shared config
cfg.Environment          // from CommonConfig
cfg.Database.URL         // from DatabaseConfig (via CommonConfig)

// 3. Access Worker-specific config
cfg.WorkerCount         // from Worker Config
cfg.QueueName           // from Worker Config
```

## Benefits of This Structure

### ✅ DRY (Don't Repeat Yourself)
- Database config defined once in `internal/config/database.go`
- Both API and Worker use the same database config
- No duplication of shared settings

### ✅ Separation of Concerns
- Shared config in `internal/config/`
- App-specific config in respective app packages
- Clear boundaries between shared and specific

### ✅ Type Safety
- All config is strongly typed
- Compile-time checks for config access
- IDE autocomplete support

### ✅ Easy to Extend
- Add shared config → edit `internal/config/`
- Add API-specific → edit `internal/api/config.go`
- Add Worker-specific → edit `internal/worker/config.go`

### ✅ Testability
- Easy to create test configs
- Can mock different configs per app
- Clear config dependencies

## Example: Adding New Shared Config

```go
// 1. Add to internal/config/common.go
type CommonConfig struct {
    Environment string `envconfig:"ENVIRONMENT" default:"development"`
    LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`
    
    // NEW: Add Redis config
    RedisURL    string `envconfig:"REDIS_URL" default:"localhost:6379"`
    
    Database    DatabaseConfig
}
```

Now both API and Worker automatically have access to `cfg.RedisURL`!

## Example: Adding New App-Specific Config

```go
// Add to internal/api/config.go
type Config struct {
    config.CommonConfig
    
    ServerPort     int    `envconfig:"API_PORT" default:"8080"`
    
    // NEW: Add rate limiting
    RateLimitRPS   int    `envconfig:"API_RATE_LIMIT_RPS" default:"100"`
}
```

Only the API has access to `cfg.RateLimitRPS`, Worker does not.

## When to Use Each

### Use `internal/config/` when:
- Config is needed by multiple apps (API + Worker)
- It's infrastructure-related (DB, Redis, logging)
- It's environment-related (dev/staging/prod)

### Use app-specific config when:
- Config is only relevant to one app
- It's about app behavior (server port, worker count)
- Different apps need different values

## Anti-Patterns to Avoid

❌ **Don't put app-specific config in shared config**
```go
// BAD: API port in CommonConfig
type CommonConfig struct {
    APIPort int  // Only API needs this!
}
```

❌ **Don't duplicate shared config**
```go
// BAD: Database config in both places
type APIConfig struct {
    DatabaseURL string  // Should be in CommonConfig!
}
```

❌ **Don't use global variables**
```go
// BAD: Global config
var GlobalConfig *Config

// GOOD: Pass config as parameter
func NewServer(cfg *Config) *Server
```
