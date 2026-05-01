# Rate Limiting

Submission endpoints are rate-limited per authenticated user using an in-memory sliding-window algorithm.

## Limits

| Endpoint | Limit |
|----------|-------|
| `POST /problems/:id/submit` | 5 requests per 60 seconds |
| `POST /problems/:id/run` | 15 requests per 60 seconds |

Each user has an independent counter. Limits are enforced independently per endpoint.

## Behaviour

When a user exceeds the limit the API returns:

```
HTTP 429 Too Many Requests
Retry-After: 1m0s

{"error": "rate limit exceeded, please slow down"}
```

## Why

Code execution is expensive. Each submission creates a job on the control plane and consumes a worker slot. Without rate limiting a single user can flood the execution queue, degrading service for everyone.

The `run` limit is looser than `submit` because test runs only execute visible test cases (fewer jobs) and are used interactively while writing code.

## Implementation

`internal/api/middleware/ratelimit.go` — `SubmissionRateLimit(limit int, window time.Duration)` returns a standard Chi middleware. It is applied per route in `cmd/api/routes.go`.

The window is sliding (not fixed bucket): each request records a timestamp and the count is the number of timestamps within the last `window` duration. Stale entries are pruned on every request; idle user entries are swept from memory every 5 minutes.

## Adjusting Limits

Change the constants in `cmd/api/routes.go`:

```go
r.With(apimiddleware.SubmissionRateLimit(5, 60*time.Second)).Post(...)  // submit
r.With(apimiddleware.SubmissionRateLimit(15, 60*time.Second)).Post(...) // run
```

This is a single-instance in-memory implementation. If the API is ever scaled horizontally, rate limit state will not be shared across instances — switch to a Redis-backed implementation at that point.
