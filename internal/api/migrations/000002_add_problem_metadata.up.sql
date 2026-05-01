BEGIN;

CREATE TYPE PROBLEM_DIFFICULTY AS ENUM ('easy', 'medium', 'hard');
CREATE TYPE SUBMISSION_TYPE AS ENUM ('test_run', 'submission');

ALTER TABLE problems
  ADD COLUMN difficulty PROBLEM_DIFFICULTY NOT NULL DEFAULT 'medium';

CREATE INDEX problems_difficulty_idx ON problems (difficulty);

ALTER TABLE submissions
  ADD COLUMN submission_type SUBMISSION_TYPE NOT NULL DEFAULT 'submission';

CREATE INDEX submissions_type_idx ON submissions (submission_type);

COMMIT;
