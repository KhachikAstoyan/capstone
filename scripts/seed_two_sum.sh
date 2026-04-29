#!/usr/bin/env bash
# Seed a "Two Sum" problem (function-call style) and four test cases.
# Requires an account with the problems.manage permission.
#
# Usage:
#   API=http://localhost:8080 EMAIL=admin@example.com PASSWORD=secret \
     ./scripts/seed_two_sum.sh
#
# Optional: pass JWT directly via TOKEN to skip login.

set -euo pipefail

API="${API:-http://localhost:8080}"
BASE="${API%/}/api/v1"

if [[ -z "${TOKEN:-}" ]]; then
  : "${EMAIL:?EMAIL required (or pass TOKEN)}"
  : "${PASSWORD:?PASSWORD required (or pass TOKEN)}"
  echo "==> logging in as ${EMAIL}"
  TOKEN="$(
    curl -fsS -X POST "${BASE}/auth/login" \
      -H 'Content-Type: application/json' \
      -d "$(printf '{"email":"%s","password":"%s"}' "${EMAIL}" "${PASSWORD}")" \
      | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p'
  )"
  if [[ -z "${TOKEN}" ]]; then
    echo "login failed: empty access_token" >&2
    exit 1
  fi
fi

AUTH_HEADER="Authorization: Bearer ${TOKEN}"

# api_post POST <path> <json-body>
# Captures HTTP status; on non-2xx prints server response then exits.
api_post() {
  local path="$1" body="$2"
  local http_code body_file
  body_file="$(mktemp)"
  http_code="$(
    curl -sS -o "${body_file}" -w '%{http_code}' \
      -X POST "${BASE}${path}" \
      -H 'Content-Type: application/json' \
      -H "${AUTH_HEADER}" \
      -d "${body}"
  )"
  if [[ "${http_code}" -lt 200 || "${http_code}" -ge 300 ]]; then
    echo "POST ${path} failed (HTTP ${http_code}):" >&2
    cat "${body_file}" >&2
    echo >&2
    rm -f "${body_file}"
    exit 1
  fi
  cat "${body_file}"
  rm -f "${body_file}"
}

echo "==> creating problem"
PROBLEM_JSON="$(api_post /internal/problems '{
      "title": "Two Sum",
      "summary": "Find indices of two numbers that add to a target.",
      "statement_markdown": "Given an array `nums` and an integer `target`, return indices of the two numbers such that they add up to `target`.\n\nYou may assume each input has exactly one solution, and you may not use the same element twice.",
      "time_limit_ms": 2000,
      "memory_limit_mb": 256,
      "tests_ref": "two-sum",
      "visibility": "published",
      "difficulty": "easy",
      "function_spec": {
        "function_name": "twoSum",
        "parameters": [
          {"name":"nums","type":"int[]"},
          {"name":"target","type":"int"}
        ],
        "return_type": "int[]"
      }
    }')"

PROBLEM_ID="$(printf '%s' "${PROBLEM_JSON}" | sed -n 's/.*"id":"\([0-9a-fA-F-]\{36\}\)".*/\1/p')"
if [[ -z "${PROBLEM_ID}" ]]; then
  echo "could not parse problem id from response: ${PROBLEM_JSON}" >&2
  exit 1
fi
echo "    problem_id = ${PROBLEM_ID}"

create_tc() {
  local order="$1" hidden="$2" input="$3" expected="$4"
  echo "==> adding test case #${order} (hidden=${hidden})"
  api_post "/internal/problems/${PROBLEM_ID}/test-cases" \
    "$(printf '{"input_data":%s,"expected_data":%s,"order_index":%s,"is_hidden":%s}' \
        "${input}" "${expected}" "${order}" "${hidden}")" >/dev/null
}

create_tc 1 false '{"nums":[2,7,11,15],"target":9}'        '[0,1]'
create_tc 2 false '{"nums":[3,2,4],"target":6}'            '[1,2]'
create_tc 3 true  '{"nums":[3,3],"target":6}'              '[0,1]'
create_tc 4 true  '{"nums":[-1,-2,-3,-4,-5],"target":-8}'  '[2,4]'

echo
echo "Done. Problem id: ${PROBLEM_ID}"
