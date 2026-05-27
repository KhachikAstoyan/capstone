# Rate Limiting

Submission endpoints are protected by two independent rate-limiting layers.

---

## Layer 1 — Per-minute sliding window (middleware)

Applied per authenticated user, per endpoint, in memory.

| Endpoint | Limit |
|----------|-------|
| `POST /problems/:id/submit` | 5 requests per 60 seconds |
| `POST /problems/:id/run` | 15 requests per 60 seconds |
| `POST /problems/:id/hint` | 10 requests per 60 seconds |

When exceeded:

```
HTTP 429 Too Many Requests
Retry-After: 1m0s

{"error": "rate limit exceeded, please slow down"}
```

**Implementation:** `internal/api/middleware/ratelimit.go` — `SubmissionRateLimit(limit, window)` Chi middleware, applied per route in `cmd/api/routes.go`.

The window is sliding (not fixed bucket): each request records a timestamp; the count is the number of timestamps within the last `window` duration. Stale entries are pruned on every check; idle user entries are swept every 5 minutes.

**Adjusting limits** — edit the constants in `cmd/api/routes.go`:

```go
r.With(apimiddleware.SubmissionRateLimit(5, 60*time.Second)).Post(...)  // submit
r.With(apimiddleware.SubmissionRateLimit(15, 60*time.Second)).Post(...) // run
```

---

## Layer 2 — Violation-based execution suspension (service)

Users who accumulate repeated security violations are temporarily blocked from creating new execution jobs. This applies on top of Layer 1.

**Threshold:** 3 or more security events within a rolling 10-minute window.

**Trigger events:** any row written to `security_events` — currently:
- `code_blocked` — AI validator rejected the code before execution
- `sandbox_escape_attempt` — container stderr indicated a runtime escape attempt

When suspended:

```
HTTP 429 Too Many Requests

{"error": "execution temporarily suspended due to repeated security violations"}
```

The suspension lifts automatically once the oldest qualifying security event falls outside the 10-minute window. No manual intervention required for transient violations; persistent abusers accumulate events faster than they expire.

**Implementation:**
- `internal/api/submissions/repository/submissions.go` — `CountRecentSecurityEvents(ctx, userID, since)`
- `internal/api/submissions/service/submissions.go` — checked at the top of `create()`, before any DB writes

**Adjusting thresholds** — edit constants in `internal/api/submissions/service/submissions.go`:

```go
const (
    violationWindow    = 10 * time.Minute
    violationThreshold = 3
)
```

---

## Why two layers

| Layer | Protects against |
|-------|-----------------|
| Sliding window | Burst abuse — hammering submit to saturate the queue |
| Violation suspension | Persistent bad actors — repeated escape attempts or blocked code across the window |

The sliding window is stateless and fast (no DB). The violation check is DB-backed but only fires once per submission attempt, after the window check passes.

---

## Horizontal scaling note

Layer 1 is in-memory and not shared across API instances. If the API is scaled horizontally, switch to a Redis-backed implementation. Layer 2 is DB-backed and works correctly with multiple API instances.
