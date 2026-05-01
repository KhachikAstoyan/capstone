# Security Audit Log


Blocked submissions are recorded in the `security_events` table for audit and monitoring purposes.

## When an event is logged

An event is written whenever a submission is rejected by the AI code validator — i.e. when `is_allowed = false` is returned. The submission status is set to `blocked` and no execution job is created.

## Table schema

```sql
CREATE TABLE security_events (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID REFERENCES submissions(id) ON DELETE CASCADE,  -- set for blocked submissions
  run_id        UUID REFERENCES runs(id) ON DELETE CASCADE,         -- set for runtime events (future)
  category      TEXT NOT NULL,
  severity      SECURITY_SEVERITY NOT NULL,  -- 'info' | 'warn' | 'high' | 'block'
  detail_json   JSONB NOT NULL DEFAULT '{}',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT security_events_target_chk CHECK (run_id IS NOT NULL OR submission_id IS NOT NULL)
);
```

## Event categories

| Category | When |
|----------|------|
| `code_blocked` | AI validator rejected the submitted code |

## detail_json fields for `code_blocked`

```json
{
  "reason": "Code uses subprocess.run to spawn child processes",
  "language_key": "python3"
}
```

The full structured violation list is stored separately in `ai_code_validations` (linked via `submission_id`).

## Querying blocked submissions

```sql
-- All blocked submissions in the last 24 hours
SELECT
  se.created_at,
  se.severity,
  se.detail_json,
  s.user_id,
  s.problem_id
FROM security_events se
JOIN submissions s ON s.id = se.submission_id
WHERE se.category = 'code_blocked'
  AND se.created_at > NOW() - INTERVAL '24 hours'
ORDER BY se.created_at DESC;

-- Users with the most blocks (potential probing)
SELECT s.user_id, COUNT(*) AS block_count
FROM security_events se
JOIN submissions s ON s.id = se.submission_id
WHERE se.category = 'code_blocked'
  AND se.created_at > NOW() - INTERVAL '7 days'
GROUP BY s.user_id
ORDER BY block_count DESC
LIMIT 20;
```

## Implementation

- Repository: `internal/api/submissions/repository/submissions.go` — `LogSecurityEvent()`
- Service call: `internal/api/submissions/service/submissions.go` — called immediately after `StatusBlocked` is set
- Migration: `internal/api/migrations/000011_security_hardening.up.sql`
