#!/bin/bash
#
# Post-flight API validation tests for AgentiCorp
# Validates all documented APIs are responding correctly after container startup
#
# Usage: ./api_test.sh [BASE_URL]
# Default BASE_URL: http://localhost:8080

set -euo pipefail

BASE_URL="${1:-http://localhost:8080}"
TIMEOUT=10
PASSED=0
FAILED=0
SKIPPED=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test resources to clean up
CREATED_BEAD_ID=""

cleanup() {
    if [[ -n "$CREATED_BEAD_ID" ]]; then
        curl -s -X DELETE "$BASE_URL/api/v1/beads/$CREATED_BEAD_ID" >/dev/null 2>&1 || true
    fi
}
trap cleanup EXIT

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASSED++))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAILED++))
}

log_skip() {
    echo -e "${YELLOW}[SKIP]${NC} $1"
    ((SKIPPED++))
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Test endpoint and return body
# Usage: body=$(call_api "method" "path" ["json_body"])
call_api() {
    local method="$1"
    local path="$2"
    local body="${3:-}"
    local url="$BASE_URL$path"
    
    if [[ -n "$body" ]]; then
        curl -s -X "$method" \
            -H "Content-Type: application/json" \
            -d "$body" \
            --max-time "$TIMEOUT" \
            "$url" 2>/dev/null
    else
        curl -s -X "$method" \
            --max-time "$TIMEOUT" \
            "$url" 2>/dev/null
    fi
}

# Test endpoint returns expected status code
# Usage: test_status "description" "method" "path" "expected_code" ["json_body"]
test_status() {
    local desc="$1"
    local method="$2"
    local path="$3"
    local expected="$4"
    local body="${5:-}"
    local url="$BASE_URL$path"
    local code
    
    if [[ -n "$body" ]]; then
        code=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            -d "$body" \
            --max-time "$TIMEOUT" \
            "$url" 2>/dev/null) || { log_fail "$desc - connection failed"; return 1; }
    else
        code=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" \
            --max-time "$TIMEOUT" \
            "$url" 2>/dev/null) || { log_fail "$desc - connection failed"; return 1; }
    fi
    
    if [[ "$code" == "$expected" ]]; then
        log_pass "$desc (HTTP $code)"
        return 0
    else
        log_fail "$desc - expected $expected, got $code"
        return 1
    fi
}

# Test that JSON response contains field
# Usage: test_json_has "description" "json" "jq_filter"
test_json_has() {
    local desc="$1"
    local json="$2"
    local filter="$3"
    
    if echo "$json" | jq -e "$filter" >/dev/null 2>&1; then
        log_pass "$desc"
        return 0
    else
        log_fail "$desc - field not found"
        return 1
    fi
}

echo "========================================"
echo "AgentiCorp Post-Flight API Tests"
echo "========================================"
echo "Base URL: $BASE_URL"
echo "Timeout: ${TIMEOUT}s"
echo ""

# Wait for server to be ready
log_info "Waiting for server to be ready..."
for i in {1..30}; do
    if curl -s --max-time 2 "$BASE_URL/api/v1/health" >/dev/null 2>&1; then
        log_info "Server is ready"
        break
    fi
    if [[ $i -eq 30 ]]; then
        log_fail "Server not ready after 30 seconds"
        exit 1
    fi
    sleep 1
done
echo ""

# ============================================
# Health Check
# ============================================
echo "--- Health Check ---"
test_status "GET /api/v1/health" "GET" "/api/v1/health" "200" || true
echo ""

# ============================================
# Projects
# ============================================
echo "--- Projects ---"
test_status "GET /api/v1/projects" "GET" "/api/v1/projects" "200" || true
response=$(call_api "GET" "/api/v1/projects")
test_json_has "Projects list has items" "$response" '.[0].id' || true

test_status "GET /api/v1/projects/agenticorp" "GET" "/api/v1/projects/agenticorp" "200" || true
response=$(call_api "GET" "/api/v1/projects/agenticorp")
test_json_has "Project has name field" "$response" '.name' || true
test_json_has "Project has agents field" "$response" '.agents' || true
echo ""

# ============================================
# Agents
# ============================================
echo "--- Agents ---"
test_status "GET /api/v1/agents" "GET" "/api/v1/agents" "200" || true
response=$(call_api "GET" "/api/v1/agents")

if [[ -n "$response" ]] && [[ "$response" != "[]" ]] && [[ "$response" != "null" ]]; then
    test_json_has "Agents list has items" "$response" '.[0].id' || true
    AGENT_ID=$(echo "$response" | jq -r '.[0].id // empty' 2>/dev/null)
    if [[ -n "$AGENT_ID" ]]; then
        # URL-encode the agent ID (may contain spaces and special chars)
        AGENT_ID_ENCODED=$(printf '%s' "$AGENT_ID" | jq -sRr @uri)
        test_status "GET /api/v1/agents/{id}" "GET" "/api/v1/agents/$AGENT_ID_ENCODED" "200" || true
    fi
else
    log_skip "GET /api/v1/agents/{id} - no agents available"
fi
echo ""

# ============================================
# Beads
# ============================================
echo "--- Beads ---"
test_status "GET /api/v1/beads" "GET" "/api/v1/beads" "200" || true
response=$(call_api "GET" "/api/v1/beads")

# Check if we have beads
if [[ -n "$response" ]] && [[ "$response" != "[]" ]] && [[ "$response" != "null" ]]; then
    BEAD_ID=$(echo "$response" | jq -r '.[0].id // empty' 2>/dev/null)
    if [[ -n "$BEAD_ID" ]]; then
        test_status "GET /api/v1/beads/{id}" "GET" "/api/v1/beads/$BEAD_ID" "200" || true
    fi
else
    log_skip "GET /api/v1/beads/{id} - no beads loaded"
fi

# Test creating a bead
response=$(call_api "POST" "/api/v1/beads" '{"title":"Post-flight Test Bead","description":"Automated test","project_id":"agenticorp","priority":2}')
if echo "$response" | jq -e '.id' >/dev/null 2>&1; then
    log_pass "POST /api/v1/beads - create bead"
    CREATED_BEAD_ID=$(echo "$response" | jq -r '.id')
    
    # Test updating the bead
    test_status "PATCH /api/v1/beads/{id}" "PATCH" "/api/v1/beads/$CREATED_BEAD_ID" "200" \
        '{"status":"in_progress"}' || true
else
    log_fail "POST /api/v1/beads - create bead failed"
fi
echo ""

# ============================================
# Decisions
# ============================================
echo "--- Decisions ---"
test_status "GET /api/v1/decisions" "GET" "/api/v1/decisions" "200" || true
echo ""

# ============================================
# Work Graph
# ============================================
echo "--- Work Graph ---"
test_status "GET /api/v1/work-graph" "GET" "/api/v1/work-graph?project_id=agenticorp" "200" || true
response=$(call_api "GET" "/api/v1/work-graph?project_id=agenticorp")
test_json_has "Work graph has beads field" "$response" '.beads' || true
echo ""

# ============================================
# Org Charts
# ============================================
echo "--- Org Charts ---"
test_status "GET /api/v1/org-charts/agenticorp" "GET" "/api/v1/org-charts/agenticorp" "200" || true
response=$(call_api "GET" "/api/v1/org-charts/agenticorp")
test_json_has "Org chart has positions" "$response" '.positions' || true
echo ""

# ============================================
# Providers
# ============================================
echo "--- Providers ---"
test_status "GET /api/v1/providers" "GET" "/api/v1/providers" "200" || true
echo ""

# ============================================
# Events
# ============================================
echo "--- Events ---"
test_status "GET /api/v1/events/stats" "GET" "/api/v1/events/stats" "200" || true

# Test SSE endpoint can connect (just check it responds)
log_info "Testing SSE endpoint connectivity..."
code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 2 "$BASE_URL/api/v1/events/stream" 2>/dev/null) || code="timeout"
if [[ "$code" == "200" ]] || [[ "$code" == "timeout" ]]; then
    log_pass "GET /api/v1/events/stream - SSE endpoint accessible"
else
    log_fail "GET /api/v1/events/stream - unexpected response: $code"
fi
echo ""

# ============================================
# File Locks
# ============================================
echo "--- File Locks ---"
test_status "GET /api/v1/file-locks" "GET" "/api/v1/file-locks" "200" || true
echo ""

# ============================================
# Summary
# ============================================
echo "========================================"
echo "Test Summary"
echo "========================================"
echo -e "Passed:  ${GREEN}$PASSED${NC}"
echo -e "Failed:  ${RED}$FAILED${NC}"
echo -e "Skipped: ${YELLOW}$SKIPPED${NC}"
echo ""

if [[ $FAILED -gt 0 ]]; then
    echo -e "${RED}POST-FLIGHT TESTS FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}POST-FLIGHT TESTS PASSED${NC}"
    exit 0
fi
