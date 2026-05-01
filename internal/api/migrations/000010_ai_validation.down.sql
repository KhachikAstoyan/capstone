BEGIN;

DROP INDEX IF EXISTS ai_validation_logs_validation_idx;
DROP TABLE IF EXISTS ai_validation_logs;

DROP INDEX IF EXISTS ai_code_validations_is_allowed_idx;
DROP INDEX IF EXISTS ai_code_validations_user_created_idx;
DROP INDEX IF EXISTS ai_code_validations_submission_idx;
DROP TABLE IF EXISTS ai_code_validations;

COMMIT;
