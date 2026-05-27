# Security Audit Log

All security-relevant submission events are recorded in the `security_events` table. Admins can view them per user in the admin panel (`GET /api/v1/internal/admin/users/:userID/security-events`).

---

## Event categories

| Category | Trigger | Stage |
|----------|---------|-------|
| `code_blocked` | AI validator returned `is_allowed=false` | Pre-execution (API) |
| `sandbox_escape_attempt` | Container stderr matched an escape pattern | Post-execution (worker → API) |

---

## `code_blocked`

Fires when the AI pre-validator rejects the submission before any execution job is created. Submission status is set to `blocked`.

`detail_json` shape:

```json
{
  "reason": "Code uses subprocess.run to spawn child processes",
  "language_key": "python"
}
```

The full structured violation list is in `ai_code_validations` (linked via `submission_id`).

---

## `sandbox_escape_attempt`

Fires when a submission passes AI pre-validation but the worker detects a runtime escape attempt by inspecting container stderr.

The Docker sandbox already blocks these operations at the OS level (network disabled, read-only rootfs, all capabilities dropped, no new privileges). This event records that the code *tried* — useful for identifying sophisticated evasion attempts and repeat offenders.

**Detected patterns (worker-side, `internal/worker/escape_detector.go`):**

| Category | Examples |
|----------|---------|
| Network escape | `network is unreachable`, `connection refused`, `ENETUNREACH`, `java.net.ConnectException` |
| Filesystem escape | `read-only file system`, `permission denied`, `operation not permitted` |
| Sensitive path probe | `/etc/passwd`, `/etc/shadow`, `/proc/self`, `/dev/mem` |
| Privilege escalation | `setuid`, `setgid`, `execve: permission denied` |

Detection scans the `stderr` field of every test case result returned by the runner. The first matched pattern wins.

`detail_json` shape:

```json
{
  "verdict": "SandboxViolation",
  "detail": "sandbox escape attempt detected: network is unreachable"
}
```

The submission is saved with verdict `SandboxViolation` and status `blocked`.

---

## Table schema

```sql
CREATE TABLE security_events (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID REFERENCES submissions(id) ON DELETE CASCADE,
  run_id        UUID REFERENCES runs(id) ON DELETE CASCADE,  -- reserved, currently NULL
  category      TEXT NOT NULL,
  severity      SECURITY_SEVERITY NOT NULL,  -- 'info' | 'warn' | 'high' | 'block'
  detail_json   JSONB NOT NULL DEFAULT '{}',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT security_events_target_chk CHECK (run_id IS NOT NULL OR submission_id IS NOT NULL)
);
```

Migration: `internal/api/migrations/000011_security_hardening.up.sql`

---

## Querying

```sql
-- All security events in the last 24 hours
SELECT se.created_at, se.category, se.severity, se.detail_json, sub.user_id
FROM security_events se
JOIN submissions sub ON sub.id = se.submission_id
WHERE se.created_at > NOW() - INTERVAL '24 hours'
ORDER BY se.created_at DESC;

-- Runtime escape attempts only
SELECT se.created_at, se.detail_json, sub.user_id, sub.source_text
FROM security_events se
JOIN submissions sub ON sub.id = se.submission_id
WHERE se.category = 'sandbox_escape_attempt'
ORDER BY se.created_at DESC;

-- Users with most violations (last 7 days) — potential repeat offenders
SELECT sub.user_id, COUNT(*) AS violation_count
FROM security_events se
JOIN submissions sub ON sub.id = se.submission_id
WHERE se.created_at > NOW() - INTERVAL '7 days'
GROUP BY sub.user_id
ORDER BY violation_count DESC
LIMIT 20;

-- Check if a user is currently in the suspension window (≥3 events in last 10 min)
SELECT COUNT(*) FROM security_events se
JOIN submissions sub ON sub.id = se.submission_id
WHERE sub.user_id = '<user-uuid>'
  AND se.created_at > NOW() - INTERVAL '10 minutes';
```

---

## Implementation

| File | Role |
|------|------|
| `internal/api/submissions/repository/submissions.go` | `LogSecurityEvent()`, `CountRecentSecurityEvents()` |
| `internal/api/submissions/service/submissions.go` | Calls `LogSecurityEvent` for both event types; checks violation count before execution |
| `internal/worker/escape_detector.go` | Pattern list and `detectEscapeAttempt()` |
| `internal/worker/docker_executor.go` | Calls `detectEscapeAttempt` on each test case stderr in `parseResults` |
| `internal/api/auth/repository/admin.go` | `GetUserSecurityEvents()` — used by admin API |
| `internal/api/auth/http/admin_users.go` | `GET /internal/admin/users/:id/security-events` handler |
