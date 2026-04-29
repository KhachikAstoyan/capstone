BEGIN;

-- Store test cases inline in the job row.
-- Each element: { "id": "...", "input": "...", "expected": "..." }
-- NULL means no test cases were provided with the job (e.g. run-only mode).
ALTER TABLE jobs
  ADD COLUMN IF NOT EXISTS test_cases JSONB;

COMMIT;
