BEGIN;

INSERT INTO languages (key, display_name, is_enabled)
VALUES ('java', 'Java', TRUE)
ON CONFLICT (key) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    is_enabled = TRUE,
    updated_at = NOW();

COMMIT;
