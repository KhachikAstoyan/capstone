# CORS Configuration

## Problem
The frontend (running on `http://localhost:5173` or `http://localhost:5174`) was unable to make requests to the backend API (running on `http://localhost:3000`) due to CORS (Cross-Origin Resource Sharing) restrictions.

## Solution
Added CORS middleware to the Go backend using the `github.com/go-chi/cors` package.

## Changes Made

### 1. Installed CORS Package
```bash
go get github.com/go-chi/cors
```

### 2. Updated `cmd/api/main.go`

Added import:
```go
"github.com/go-chi/cors"
```

Added CORS middleware before other middleware:
```go
r.Use(cors.Handler(cors.Options{
    AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:5174"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
    ExposedHeaders:   []string{"Link"},
    AllowCredentials: true,
    MaxAge:           300,
}))
```

## Configuration Details

The CORS origins are now configured via environment variable:

**Environment Variable**: `API_ALLOWED_ORIGINS`
- Comma-separated list of allowed origins
- Example: `"http://localhost:5173,http://localhost:5174"`
- Default: `"*"` (allows all origins - not recommended for production)

**Other CORS Settings**:
- **AllowedMethods**: All HTTP methods needed for REST API
- **AllowedHeaders**: Standard headers including Authorization for JWT
- **AllowCredentials**: Enabled to allow cookies/auth headers
- **MaxAge**: Preflight cache duration (5 minutes)

## Environment Configuration

### Development (.env file)
```bash
export API_ALLOWED_ORIGINS="http://localhost:5173,http://localhost:5174"
```

### Production
```bash
export API_ALLOWED_ORIGINS="https://your-production-domain.com,https://www.your-production-domain.com"
```

### Allow All Origins (Not Recommended for Production)
```bash
export API_ALLOWED_ORIGINS="*"
# or simply don't set the variable (defaults to "*")
```

## Implementation

The code in `cmd/api/main.go` now reads from config:

```go
allowedOrigins := []string{"*"}
if cfg.AllowedOrigins != "" && cfg.AllowedOrigins != "*" {
    allowedOrigins = strings.Split(cfg.AllowedOrigins, ",")
    // Trim whitespace from each origin
    for i := range allowedOrigins {
        allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
    }
}

r.Use(cors.Handler(cors.Options{
    AllowedOrigins: allowedOrigins,
    // ... rest of config
}))
```

## Testing

1. Start the backend: `./build/api`
2. Start the frontend: `npm run dev` (in web directory)
3. Open browser to `http://localhost:5174`
4. Try to register or login - CORS errors should be gone

## Verification

Check browser console - you should no longer see errors like:
```
Access to fetch at 'http://localhost:3000/api/v1/auth/register' from origin 'http://localhost:5174' 
has been blocked by CORS policy: No 'Access-Control-Allow-Origin' header is present on the requested resource.
```

Instead, you should see successful API responses.
