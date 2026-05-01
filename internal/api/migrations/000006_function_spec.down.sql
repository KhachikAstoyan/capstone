BEGIN;

ALTER TABLE problem_test_cases
    DROP COLUMN IF EXISTS input_data,
    DROP COLUMN IF EXISTS expected_data,
    ADD COLUMN input    TEXT NOT NULL DEFAULT '',
    ADD COLUMN expected TEXT NOT NULL DEFAULT '';

ALTER TABLE problems DROP COLUMN IF EXISTS function_spec;

COMMIT;
