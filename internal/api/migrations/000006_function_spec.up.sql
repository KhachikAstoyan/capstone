BEGIN;

ALTER TABLE problems ADD COLUMN function_spec JSONB;

ALTER TABLE problem_test_cases
    DROP COLUMN input,
    DROP COLUMN expected,
    ADD COLUMN input_data    JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN expected_data JSONB NOT NULL DEFAULT '{}';

INSERT INTO languages (key, display_name, is_enabled)
VALUES
    ('python', 'Python 3', TRUE),
    ('javascript', 'JavaScript', TRUE),
    ('go', 'Go', TRUE)
ON CONFLICT (key) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    is_enabled = TRUE,
    updated_at = NOW();

COMMIT;
