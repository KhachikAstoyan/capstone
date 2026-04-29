BEGIN;

CREATE TYPE SUBMISSION_KIND AS ENUM ('run', 'submit');

ALTER TABLE submissions
    ADD COLUMN kind SUBMISSION_KIND NOT NULL DEFAULT 'submit';

CREATE INDEX submissions_kind_idx ON submissions (kind);

COMMIT;
