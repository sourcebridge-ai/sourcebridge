#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors
#
# seed.sh — prepare the three QA parity benchmark repos on a running
# SourceBridge instance. Idempotent: re-running does not duplicate
# repositories; it just ensures each is registered and indexed.
#
# The three repos mirror the plan's repo A/B/C split:
#   A: multi-lang-repo (small, zero-requirement)
#   B: acme-api         (medium, REQ-* linked)
#   C: sourcebridge     (large, architecture stress)
#
# Usage:
#
#   SOURCEBRIDGE_URL=http://localhost:8080 \
#   SOURCEBRIDGE_API_TOKEN=<token> \
#   benchmarks/qa/seed.sh
#
# Output (written to stdout, one JSON object per line):
#
#   {"repo": "multi-lang-repo-go", "id": "abc123", "indexed": true}
#   {"repo": "acme-api", "id": "def456", "indexed": true}
#   {"repo": "sourcebridge", "id": "ghi789", "indexed": true}
#
# This file is the "repo side" of Phase 4 reproducibility. For the
# questions + judge + report, see benchmarks/qa/README.md.

set -euo pipefail

SERVER_URL="${SOURCEBRIDGE_URL:-http://localhost:8080}"
TOKEN="${SOURCEBRIDGE_API_TOKEN:-}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [[ -z "$TOKEN" ]]; then
  echo "ERROR: SOURCEBRIDGE_API_TOKEN must be set" >&2
  echo "       Run 'sourcebridge login' or set the env var directly." >&2
  exit 1
fi

probe_server() {
  local probe
  probe=$(curl -sS -o /dev/null -w '%{http_code}' "${SERVER_URL}/healthz" || echo "000")
  if [[ "$probe" != "200" ]]; then
    echo "ERROR: server at ${SERVER_URL}/healthz returned ${probe}" >&2
    exit 1
  fi
}

# graphql_call <operation-name> <query> <variables-json>
# Returns the JSON response on stdout; exits non-zero on HTTP or GraphQL error.
graphql_call() {
  local op="$1"
  local query="$2"
  local vars="$3"
  local payload
  payload=$(jq -n --arg q "$query" --arg op "$op" --argjson v "$vars" \
    '{operationName:$op, query:$q, variables:$v}')
  local resp
  resp=$(curl -sS -X POST "${SERVER_URL}/api/v1/graphql" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d "$payload")
  if echo "$resp" | jq -e '.errors' >/dev/null 2>&1; then
    echo "ERROR: GraphQL ${op} failed:" >&2
    echo "$resp" | jq '.errors' >&2
    exit 1
  fi
  echo "$resp"
}

# find_or_create_repo <name> <path>
# Looks up a repo by name; creates it via addRepository if absent.
# Prints the repo ID on stdout.
find_or_create_repo() {
  local name="$1"
  local path="$2"
  local query='query Q { repositories { id name } }'
  local list
  list=$(graphql_call "Q" "$query" '{}')
  local existing
  existing=$(echo "$list" | jq -r --arg n "$name" '.data.repositories[] | select(.name == $n) | .id' | head -n1)
  if [[ -n "$existing" ]]; then
    echo "$existing"
    return
  fi
  local mutation='mutation M($input: AddRepositoryInput!) { addRepository(input: $input) { id name } }'
  local vars
  vars=$(jq -n --arg n "$name" --arg p "$path" '{input: {name: $n, path: $p}}')
  local created
  created=$(graphql_call "M" "$mutation" "$vars")
  echo "$created" | jq -r '.data.addRepository.id'
}

# wait_for_index <repo-id> <timeout-seconds>
# Polls until the repo status reaches "ready" or the timeout elapses.
wait_for_index() {
  local id="$1"
  local timeout="${2:-300}"
  local deadline=$(( $(date +%s) + timeout ))
  local query='query Q($id: ID!) { repository(id: $id) { id status fileCount } }'
  while [[ $(date +%s) -lt $deadline ]]; do
    local vars
    vars=$(jq -n --arg id "$id" '{id: $id}')
    local resp
    resp=$(graphql_call "Q" "$query" "$vars")
    local status
    status=$(echo "$resp" | jq -r '.data.repository.status // "unknown"')
    if [[ "$status" == "ready" ]]; then
      return 0
    fi
    if [[ "$status" == "error" ]]; then
      echo "ERROR: repo $id indexing failed" >&2
      return 1
    fi
    sleep 5
  done
  echo "ERROR: repo $id did not reach ready within ${timeout}s" >&2
  return 1
}

# import_requirements <repo-id> <path-to-markdown>
# Non-fatal: if the file doesn't exist, prints a warning and returns 0.
import_requirements() {
  local id="$1"
  local file="$2"
  if [[ ! -f "$file" ]]; then
    echo "  (no requirements file at $file, skipping)" >&2
    return 0
  fi
  local mutation='mutation M($input: ImportRequirementsInput!) { importRequirements(input: $input) { imported updated failed } }'
  local content
  content=$(cat "$file")
  local vars
  vars=$(jq -n --arg id "$id" --arg c "$content" \
    '{input: {repositoryId: $id, content: $c, format: MARKDOWN, sourcePath: "README.md"}}')
  graphql_call "M" "$mutation" "$vars" >/dev/null
}

probe_server
echo "[seed] server reachable at ${SERVER_URL}" >&2

# Repo A: multi-lang-repo (small fixture, no requirements).
REPO_A_PATH="${REPO_ROOT}/tests/fixtures/multi-lang-repo"
echo "[seed] repo A: multi-lang-repo" >&2
ID_A=$(find_or_create_repo "multi-lang-repo" "$REPO_A_PATH")
wait_for_index "$ID_A" 300 || true
jq -n --arg n "multi-lang-repo" --arg id "$ID_A" '{repo: $n, id: $id, indexed: true}'

# Repo B: acme-api (medium with REQ-* links).
REPO_B_PATH="${REPO_ROOT}/examples/acme-api"
echo "[seed] repo B: acme-api" >&2
ID_B=$(find_or_create_repo "acme-api" "$REPO_B_PATH")
wait_for_index "$ID_B" 600 || true
import_requirements "$ID_B" "${REPO_B_PATH}/README.md"
jq -n --arg n "acme-api" --arg id "$ID_B" '{repo: $n, id: $id, indexed: true}'

# Repo C: sourcebridge itself (architecture stress).
REPO_C_PATH="$REPO_ROOT"
echo "[seed] repo C: sourcebridge" >&2
ID_C=$(find_or_create_repo "sourcebridge" "$REPO_C_PATH")
wait_for_index "$ID_C" 1800 || true
jq -n --arg n "sourcebridge" --arg id "$ID_C" '{repo: $n, id: $id, indexed: true}'

echo "[seed] done" >&2
