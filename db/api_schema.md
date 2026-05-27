# API Database Schema

Consolidated schema for the **main API database** after all migrations
(`internal/api/migrations/000001`–`000012`).

> Note: two result models coexist. The `runs` family (migration 1) is the
> original execution model; the `submissions.status` + `submission_results`
> family (migration 5+) is the current control-plane execution path. New work
> uses the latter.

---

## Domain Map

```
                          +--------------------+
                          |       users        |
                          +--------------------+
                           ^   ^   ^   ^    ^
          +----------------+   |   |   |    +----------------+
          |                    |   |   |                     |
   +-------------+   +------------------+   +--------------+   +----------------+
   |  user_roles |   |  auth_identities |   | refresh_     |   | email_verif_   |
   +-------------+   +------------------+   | tokens       |   | tokens         |
          |                                +--------------+   +----------------+
          v
   +-------------+        +------------------+
   |    roles    |<------>| role_permissions |<------> permissions
   +-------------+        +------------------+

   +--------------------+        +-----------+        +-------------+
   |      problems      |<------>| problem_  |<------>|  languages  |
   +--------------------+        | languages |        +-------------+
     ^   ^   ^   ^                +-----------+              ^
     |   |   |   +----------------+                          |
     |   |   |                    |                          |
     |   |   v                    v                          |
     | +-------------+   +-------------------+                |
     | | problem_tags|   | problem_test_cases|                |
     | +-------------+   +-------------------+                |
     |       v                                                |
     |    +------+                                            |
     |    | tags |                                            |
     |    +------+                                            |
     |                                                        |
     |   +-------------+        +-------------+                |
     +-->| test_groups |<-------|  testcases  |                |
         +-------------+        +-------------+                |
                                                               |
   +--------------------+                                      |
   |    submissions     |--------------------------------------+
   +--------------------+
     |        |        |
     v        v        v
 +--------+ +-----------------+ +---------------------+
 |  runs  | | submission_     | | ai_code_validations |
 +--------+ | results         | +---------------------+
   |  |     +-----------------+           |
   |  |                                   v
   |  v                          +--------------------+
   | +---------------------+      | ai_validation_logs |
   | | execution_summaries |      +--------------------+
   | +---------------------+
   v
 +------------------+      +------------------+
 | testcase_results |      | security_events  | (-> runs and/or submissions)
 +------------------+      +------------------+

   +----------------+
   | hint_messages  | (-> users, problems)
   +----------------+
```

---

## Enums

| Enum | Values |
|------|--------|
| `USER_STATUS` | `ACTIVE`, `BANNED` |
| `PROBLEM_VISIBILITY` | `draft`, `published`, `archived` |
| `PROBLEM_DIFFICULTY` | `easy`, `medium`, `hard` |
| `TEST_GROUP_VISIBILITY` | `public`, `hidden` |
| `RUN_STATE` | `queued`, `running`, `completed`, `failed` |
| `SECURITY_SEVERITY` | `info`, `warn`, `high`, `block` |
| `SUBMISSION_TYPE` | `test_run`, `submission` |
| `SUBMISSION_KIND` | `run`, `submit` |
| `SUBMISSION_STATUS` | `pending`, `queued`, `running`, `accepted`, `wrong_answer`, `time_limit_exceeded`, `memory_limit_exceeded`, `runtime_error`, `compilation_error`, `internal_error`, `blocked` |

---

## Auth / Users / RBAC

### users
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| handle | CITEXT | unique |
| email | CITEXT | unique, nullable |
| email_verified | BOOLEAN | default false |
| display_name | TEXT | nullable |
| avatar_url | TEXT | nullable |
| status | USER_STATUS | default `ACTIVE` |
| created_at / updated_at | TIMESTAMPTZ | |

### roles
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| name | TEXT | unique |
| description | TEXT | nullable |
| created_at / updated_at | TIMESTAMPTZ | |

### permissions
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| key | TEXT | unique |
| description | TEXT | nullable |
| created_at / updated_at | TIMESTAMPTZ | |

### user_roles
| Column | Type | Notes |
|--------|------|-------|
| user_id | UUID | FK users, PK part |
| role_id | UUID | FK roles, PK part |
| granted_by | UUID | FK users, nullable |
| granted_at | TIMESTAMPTZ | |

### role_permissions
| Column | Type | Notes |
|--------|------|-------|
| role_id | UUID | FK roles, PK part |
| permission_id | UUID | FK permissions, PK part |

### auth_identities
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| user_id | UUID | FK users |
| provider | TEXT | |
| provider_subject | TEXT | unique with provider |
| password_hash | TEXT | nullable |
| password_algo | TEXT | nullable |
| email_at_provider | CITEXT | nullable |
| email_verified_at_provider | BOOLEAN | nullable |
| created_at | TIMESTAMPTZ | |
| last_login_at | TIMESTAMPTZ | nullable |

### refresh_tokens
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| user_id | UUID | FK users |
| auth_identity_id | UUID | FK auth_identities, nullable |
| token_hash | BYTEA | unique |
| issued_at / expires_at | TIMESTAMPTZ | |
| revoked_at | TIMESTAMPTZ | nullable |
| replaced_by | UUID | FK refresh_tokens, nullable |

### email_verification_tokens
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| user_id | UUID | FK users |
| token_hash | BYTEA | unique |
| created_at | TIMESTAMPTZ | |

---

## Problems / Languages / Tags

### languages
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| key | TEXT | unique (`python`, `javascript`, `go`, `java`) |
| display_name | TEXT | |
| is_enabled | BOOLEAN | default true |
| created_at / updated_at | TIMESTAMPTZ | |

### problems
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| slug | TEXT | unique |
| title | TEXT | |
| statement_markdown | TEXT | |
| summary | TEXT | default `''` |
| difficulty | PROBLEM_DIFFICULTY | default `medium` |
| time_limit_ms | INTEGER | > 0 |
| memory_limit_mb | INTEGER | > 0 |
| tests_ref | TEXT | |
| tests_hash | TEXT | nullable |
| function_spec | JSONB | nullable |
| visibility | PROBLEM_VISIBILITY | default `draft` |
| created_by_user_id | UUID | FK users, nullable |
| created_at / updated_at | TIMESTAMPTZ | |

### problem_languages
| Column | Type | Notes |
|--------|------|-------|
| problem_id | UUID | FK problems, PK part |
| language_id | UUID | FK languages, PK part |

### tags
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| name | TEXT | unique |
| created_at | TIMESTAMPTZ | |

### problem_tags
| Column | Type | Notes |
|--------|------|-------|
| problem_id | UUID | FK problems, PK part |
| tag_id | UUID | FK tags, PK part |

### test_groups
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| problem_id | UUID | FK problems |
| name | TEXT | |
| visibility | TEST_GROUP_VISIBILITY | |
| order_index | INTEGER | >= 0, unique per problem |
| points_weight | NUMERIC | nullable |
| is_active | BOOLEAN | default true |

### testcases
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| problem_id | UUID | FK problems |
| group_id | UUID | FK test_groups |
| order_index | INTEGER | >= 0, unique per problem |
| external_id | TEXT | unique per problem |
| points | NUMERIC | nullable |
| is_active | BOOLEAN | default true |

### problem_test_cases
Current model (control-plane execution path).
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| problem_id | UUID | FK problems |
| external_id | TEXT | unique per problem |
| input_data | JSONB | default `{}` |
| expected_data | JSONB | default `{}` |
| order_index | INTEGER | >= 0 |
| is_active | BOOLEAN | default true |
| is_hidden | BOOLEAN | default false |
| created_at | TIMESTAMPTZ | |

---

## Submissions / Runs / Results

### submissions
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| user_id | UUID | FK users |
| problem_id | UUID | FK problems |
| language_id | UUID | FK languages |
| source_ref | TEXT | nullable; exactly one of source_ref / source_text set |
| source_text | TEXT | nullable |
| source_sha256 | TEXT | nullable |
| submission_type | SUBMISSION_TYPE | default `submission` |
| kind | SUBMISSION_KIND | default `submit` |
| cp_job_id | UUID | control-plane job id, nullable |
| status | SUBMISSION_STATUS | default `pending` |
| created_at | TIMESTAMPTZ | |

### runs
Original execution model.
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| submission_id | UUID | FK submissions |
| state | RUN_STATE | default `queued` |
| queued_at / started_at / finished_at | TIMESTAMPTZ | started/finished nullable |
| failure_reason | TEXT | nullable |
| result_digest | TEXT | nullable |
| worker_id | TEXT | nullable |

### execution_summaries
| Column | Type | Notes |
|--------|------|-------|
| run_id | UUID | PK, FK runs |
| overall_verdict | TEXT | |
| total_time_ms / max_memory_kb / wall_time_ms | INTEGER | nullable |
| compiler_output_ref | TEXT | nullable |
| created_at | TIMESTAMPTZ | |

### testcase_results
| Column | Type | Notes |
|--------|------|-------|
| run_id | UUID | FK runs, PK part |
| testcase_id | UUID | FK testcases, PK part |
| verdict | TEXT | |
| time_ms / memory_kb | INTEGER | nullable |
| stdout_ref / stderr_ref | TEXT | nullable |

### submission_results
Current result model (control-plane execution path).
| Column | Type | Notes |
|--------|------|-------|
| submission_id | UUID | PK, FK submissions |
| overall_verdict | TEXT | |
| total_time_ms / max_memory_kb / wall_time_ms | INTEGER | nullable |
| compiler_output | TEXT | nullable |
| testcase_results | JSONB | default `[]` |
| created_at | TIMESTAMPTZ | |

---

## Security / AI

### security_events
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| run_id | UUID | FK runs, nullable |
| submission_id | UUID | FK submissions, nullable |
| category | TEXT | |
| severity | SECURITY_SEVERITY | |
| detail_json | JSONB | default `{}` |
| created_at | TIMESTAMPTZ | |

CHECK: at least one of `run_id`, `submission_id` is set.

### ai_code_validations
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| submission_id | UUID | FK submissions |
| user_id | UUID | FK users |
| problem_id | UUID | FK problems |
| code | TEXT | |
| language_id | UUID | FK languages |
| is_allowed | BOOLEAN | |
| severity | TEXT | nullable |
| reason | TEXT | nullable |
| validation_metadata | JSONB | nullable |
| created_at / updated_at | TIMESTAMPTZ | |

### ai_validation_logs
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| validation_id | UUID | FK ai_code_validations |
| request_body / response_body | JSONB | nullable |
| error_message | TEXT | nullable |
| tokens_used / response_time_ms | INTEGER | nullable |
| created_at | TIMESTAMPTZ | |

### hint_messages
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| user_id | UUID | FK users |
| problem_id | UUID | FK problems |
| role | TEXT | CHECK in (`user`, `assistant`) |
| content | TEXT | |
| created_at | TIMESTAMPTZ | |
