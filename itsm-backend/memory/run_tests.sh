#!/bin/bash
REPORT="/Users/heidsoft/Downloads/research/itsm/itsm-backend/memory/final-e2e-validation-manual-2026-03-08.md"
TS=$(date "+%Y-%m-%d %H:%M:%S")

# Get auth token
AUTH_RESP=$(curl -s -X POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')
TOKEN=$(echo $AUTH_RESP | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "ERROR: Failed to get authentication token"
  exit 1
fi

echo "# ITSM E2E Validation Report - Manual Test" > $REPORT
echo "**Date:** $TS" >> $REPORT
echo "**Tester:** Automated Script" >> $REPORT
echo "" >> $REPORT

# Function to test endpoint
test_endpoint() {
  local method=$1
  local endpoint=$2
  local data=$3
  local desc=$4
  
  echo "## Testing: $desc" >> $REPORT
  echo "**Endpoint:** $method $endpoint" >> $REPORT
  
  start=$(date +%s%3N)
  if [ "$method" = "GET" ]; then
    response=$(curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:8090/api/v1$endpoint")
  else
    response=$(curl -s -X $method -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$data" "http://localhost:8090/api/v1$endpoint")
  fi
  end=$(date +%s%3N)
  duration=$((end - start))
  
  code=$(echo $response | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('code', 'N/A'))" 2>/dev/null || echo "parse_error")
  
  if [ "$code" = "0" ] || [ "$code" = "success" ]; then
    echo "**Status:** ✅ PASS" >> $REPORT
    echo "**Response Time:** ${duration}ms" >> $REPORT
  else
    echo "**Status:** ❌ FAIL" >> $REPORT
    echo "**Error Code:** $code" >> $REPORT
    echo "**Raw Response (truncated):** ${response:0:200}..." >> $REPORT
  fi
  echo "" >> $REPORT
}

# Run all tests
test_endpoint "POST" "/auth/login" '{"username":"admin","password":"admin123"}' "Authentication - Login"

test_endpoint "GET" "/dashboard/stats" "" "Dashboard Statistics"

test_endpoint "GET" "/tickets" "" "List Tickets"
TEST_TICKET_ID=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8090/api/v1/tickets | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['tickets'][0]['id'] if d.get('code')==0 and d.get('data',{}).get('tickets') else '')" 2>/dev/null)
if [ -n "$TEST_TICKET_ID" ]; then
  test_endpoint "GET" "/tickets/$TEST_TICKET_ID" "" "View Ticket Details"
  test_endpoint "PATCH" "/tickets/$TEST_TICKET_ID" '{"status":"in_progress"}' "Update Ticket Status"
fi

test_endpoint "GET" "/incidents" "" "List Incidents"
TEST_INCIDENT_ID=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8090/api/v1/incidents | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data'][0]['id'] if d.get('code')==0 and d.get('data') else '')" 2>/dev/null)
if [ -n "$TEST_INCIDENT_ID" ]; then
  test_endpoint "GET" "/incidents/$TEST_INCIDENT_ID" "" "View Incident"
  test_endpoint "PATCH" "/incidents/$TEST_INCIDENT_ID" '{"status":"resolved"}' "Edit Incident"
fi

test_endpoint "GET" "/cmdb/cis" "" "List CMDB Items"
test_endpoint "POST" "/cmdb/cis" '{"name":"E2E Test Asset","item_type":"server","serial_number":"TEST-123","manufacturer":"Test","model":"TestModel","status":"active"}' "Create CMDB Asset"

test_endpoint "GET" "/changes" "" "List Changes"
TEST_CHANGE_ID=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8090/api/v1/changes | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data'][0]['id'] if d.get('code')==0 and d.get('data') else '')" 2>/dev/null)
if [ -n "$TEST_CHANGE_ID" ]; then
  test_endpoint "POST" "/changes/$TEST_CHANGE_ID/submit" "" "Submit Change Request"
fi
test_endpoint "POST" "/changes" '{"title":"E2E Change","description":"Test","change_type":"standard"}' "Create Change Request"

test_endpoint "GET" "/knowledge/articles" "" "List Knowledge Articles"
test_endpoint "GET" "/knowledge/stats" "" "Knowledge Stats"

test_endpoint "GET" "/groups" "" "List User Groups"
test_endpoint "POST" "/groups" '{"name":"E2E Group","description":"Test group"}' "Create User Group"

test_endpoint "GET" "/sla/metrics" "" "SLA Metrics"
test_endpoint "GET" "/sla/definitions" "" "SLA Definitions"

test_endpoint "GET" "/problems" "" "List Problems"

test_endpoint "GET" "/users" "" "List Users"

test_endpoint "GET" "/org/teams" "" "List Teams"

# Summary
echo "## Summary" >> $REPORT
echo "Tested all critical E2E scenarios for ITSM system." >> $REPORT
echo "" >> $REPORT
echo "**Recommendation:** See individual test results above." >> $REPORT

echo "Report generated: $REPORT"
