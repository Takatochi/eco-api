#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# test-api.sh — manual end-to-end API tests
# Usage: ./scripts/test-api.sh [BASE_URL]
# ---------------------------------------------------------------------------

BASE_URL="${1:-http://localhost:8080}"
PASS=0
FAIL=0

# ── helpers ─────────────────────────────────────────────────────────────────

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
RESET='\033[0m'

section() { echo -e "\n${BOLD}── $* ──${RESET}"; }

pass() {
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${RESET} $*"
}

fail() {
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${RESET} $*"
}

# Send a request and return (status_code, body) separated by newline.
# Args: METHOD URL [BODY]
request() {
  local method="$1" url="$2" body="${3:-}"
  if [[ -n "$body" ]]; then
    curl -s -w '\n%{http_code}' -X "$method" "$url" \
      -H 'Content-Type: application/json' \
      -d "$body"
  else
    curl -s -w '\n%{http_code}' -X "$method" "$url"
  fi
}

# Assert the response has the expected HTTP status code.
# Args: label expected_code actual_code body
assert_status() {
  local label="$1" expected="$2" actual="$3" body="$4"
  if [[ "$actual" == "$expected" ]]; then
    pass "$label → HTTP $actual"
  else
    fail "$label → expected HTTP $expected, got $actual | body: $body"
  fi
}

# Assert the response body contains a substring.
# Args: label substring body
assert_contains() {
  local label="$1" needle="$2" body="$3"
  if echo "$body" | grep -qF "$needle"; then
    pass "$label contains '$needle'"
  else
    fail "$label expected to contain '$needle' | body: $body"
  fi
}

# Assert the response body does NOT contain a substring.
assert_not_contains() {
  local label="$1" needle="$2" body="$3"
  if ! echo "$body" | grep -qF "$needle"; then
    pass "$label does not expose '$needle'"
  else
    fail "$label should NOT contain '$needle' | body: $body"
  fi
}

# Split "body\ncode" output from request().
body_of()   { echo "$1" | head -n -1; }
status_of() { echo "$1" | tail -n 1; }

# ── unique timestamp per run to avoid 409 on re-runs ────────────────────────
TS_BASE="2026-02-21T$(date +%H:%M:%S)Z"
TS_1="$TS_BASE"
TS_2="2026-02-21T$(date -d '+1 second' +%H:%M:%S 2>/dev/null || date -v+1S +%H:%M:%S)Z"

# ============================================================================
section "Health & Ping"

resp=$(request GET "$BASE_URL/ping")
assert_status "GET /ping" 200 "$(status_of "$resp")" "$(body_of "$resp")"
assert_contains "GET /ping" "pong" "$(body_of "$resp")"

resp=$(request GET "$BASE_URL/health")
assert_status "GET /health" 200 "$(status_of "$resp")" "$(body_of "$resp")"
assert_contains "GET /health" "ok" "$(body_of "$resp")"

resp=$(request GET "$BASE_URL/health/ping")
assert_status "GET /health/ping" 200 "$(status_of "$resp")" "$(body_of "$resp")"

# ============================================================================
section "OpenAPI spec"

resp=$(request GET "$BASE_URL/openapi.json")
assert_status "GET /openapi.json" 200 "$(status_of "$resp")" "$(body_of "$resp")"
assert_contains "GET /openapi.json" "openapi" "$(body_of "$resp")"

# ============================================================================
section "POST /api/measurements — happy path"

resp=$(request POST "$BASE_URL/api/measurements" \
  "{\"deviceId\":\"test-sensor\",\"timestamp\":\"$TS_1\",\"temperature\":18.5,\"ph\":7.2,\"turbidity\":1.1,\"conductivity\":450.0}")
assert_status "all fields" 201 "$(status_of "$resp")" "$(body_of "$resp")"
assert_contains "all fields" "id" "$(body_of "$resp")"
CREATED_ID=$(body_of "$resp" | grep -o '"id":[0-9]*' | grep -o '[0-9]*')

resp=$(request POST "$BASE_URL/api/measurements" \
  "{\"deviceId\":\"test-sensor\",\"timestamp\":\"$TS_2\"}")
assert_status "only required fields" 201 "$(status_of "$resp")" "$(body_of "$resp")"

# ============================================================================
section "POST /api/measurements — validation errors (expect 400)"

resp=$(request POST "$BASE_URL/api/measurements" \
  '{"timestamp":"2026-02-21T10:00:00Z"}')
assert_status "missing deviceId" 400 "$(status_of "$resp")" "$(body_of "$resp")"
assert_contains "missing deviceId" "deviceId" "$(body_of "$resp")"

resp=$(request POST "$BASE_URL/api/measurements" \
  '{"deviceId":"test-sensor"}')
assert_status "missing timestamp" 400 "$(status_of "$resp")" "$(body_of "$resp")"
assert_contains "missing timestamp" "timestamp" "$(body_of "$resp")"

resp=$(request POST "$BASE_URL/api/measurements" \
  '{"deviceId":"test-sensor","timestamp":"not-a-date"}')
assert_status "invalid timestamp" 400 "$(status_of "$resp")" "$(body_of "$resp")"

resp=$(request POST "$BASE_URL/api/measurements" \
  '{"deviceId":"test-sensor","timestamp":"2026-02-21T10:00:01Z","ph":99}')
assert_status "ph out of range" 400 "$(status_of "$resp")" "$(body_of "$resp")"
assert_contains "ph out of range" "ph" "$(body_of "$resp")"

resp=$(request POST "$BASE_URL/api/measurements" \
  '{"deviceId":"test-sensor","timestamp":"2026-02-21T10:00:02Z","temperature":999}')
assert_status "temperature out of range" 400 "$(status_of "$resp")" "$(body_of "$resp")"
assert_contains "temperature out of range" "temperature" "$(body_of "$resp")"

resp=$(request POST "$BASE_URL/api/measurements" '{not json}')
assert_status "invalid JSON" 400 "$(status_of "$resp")" "$(body_of "$resp")"

# ============================================================================
section "POST /api/measurements — duplicate (expect 409)"

resp=$(request POST "$BASE_URL/api/measurements" \
  "{\"deviceId\":\"test-sensor\",\"timestamp\":\"$TS_1\",\"temperature\":18.5,\"ph\":7.2}")
assert_status "duplicate device+timestamp" 409 "$(status_of "$resp")" "$(body_of "$resp")"

# ============================================================================
section "POST /api/measurements — internal errors do not leak details"

# A 409 error message is expected (unique violation), but for 500s the body
# should say "internal server error", not raw postgres/Go internals.
# We test 409 body is safe (no stack trace / driver details).
resp=$(request POST "$BASE_URL/api/measurements" \
  "{\"deviceId\":\"test-sensor\",\"timestamp\":\"$TS_1\"}")
assert_not_contains "409 body hides internals" "pgconn" "$(body_of "$resp")"
assert_not_contains "409 body hides internals" "ERROR" "$(body_of "$resp")"

# ============================================================================
section "GET /api/measurements — query"

resp=$(request GET "$BASE_URL/api/measurements?deviceId=test-sensor&from=2026-02-21T00:00:00Z&to=2026-02-21T23:59:59Z")
assert_status "valid query" 200 "$(status_of "$resp")" "$(body_of "$resp")"
assert_contains "valid query" "test-sensor" "$(body_of "$resp")"
assert_contains "valid query" "dataHash" "$(body_of "$resp")"

# Verify the record we created earlier is present
if [[ -n "$CREATED_ID" ]]; then
  assert_contains "created record in results" "\"id\":$CREATED_ID" "$(body_of "$resp")"
fi

# ============================================================================
section "GET /api/measurements — validation errors (expect 400)"

resp=$(request GET "$BASE_URL/api/measurements")
assert_status "missing all params" 400 "$(status_of "$resp")" "$(body_of "$resp")"

resp=$(request GET "$BASE_URL/api/measurements?deviceId=test-sensor")
assert_status "missing from/to" 400 "$(status_of "$resp")" "$(body_of "$resp")"

resp=$(request GET "$BASE_URL/api/measurements?deviceId=test-sensor&from=bad&to=2026-02-21T23:59:59Z")
assert_status "invalid from" 400 "$(status_of "$resp")" "$(body_of "$resp")"

resp=$(request GET "$BASE_URL/api/measurements?deviceId=test-sensor&from=2026-02-21T23:59:59Z&to=2026-02-21T00:00:00Z")
assert_status "to before from" 400 "$(status_of "$resp")" "$(body_of "$resp")"

resp=$(request GET "$BASE_URL/api/measurements?deviceId=test-sensor&from=2026-02-21T00:00:00Z&to=2026-02-21T23:59:59Z&limit=0")
assert_status "limit=0" 400 "$(status_of "$resp")" "$(body_of "$resp")"

resp=$(request GET "$BASE_URL/api/measurements?deviceId=test-sensor&from=2026-02-21T00:00:00Z&to=2026-02-21T23:59:59Z&limit=9999")
assert_status "limit>5000" 400 "$(status_of "$resp")" "$(body_of "$resp")"

resp=$(request GET "$BASE_URL/api/measurements?deviceId=test-sensor&from=2026-02-21T00:00:00Z&to=2026-02-21T23:59:59Z&limit=100")
assert_status "custom limit" 200 "$(status_of "$resp")" "$(body_of "$resp")"

# ============================================================================
section "GET /api/measurements — empty result"

resp=$(request GET "$BASE_URL/api/measurements?deviceId=no-such-device&from=2000-01-01T00:00:00Z&to=2000-01-02T00:00:00Z")
assert_status "unknown device" 200 "$(status_of "$resp")" "$(body_of "$resp")"
assert_contains "unknown device returns empty array" "[]" "$(body_of "$resp")"

# ============================================================================
echo ""
echo -e "${BOLD}Results: ${GREEN}$PASS passed${RESET}, ${RED}$FAIL failed${RESET}"
echo ""
[[ "$FAIL" -eq 0 ]]
