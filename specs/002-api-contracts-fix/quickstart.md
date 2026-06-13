# Quickstart: API Contracts Fix Validation

## Prerequisites

- Backend running on `http://localhost:8090`
- Admin token (login with `admin` / `admin123`)
- PostgreSQL container `itsm-postgres` accessible

## Validation Scenarios

### Scenario 1: Tickets API Pagination

```bash
# Get token
TOKEN=$(curl -s -X POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")

# Test pagination response
curl -s "http://localhost:8090/api/v1/tickets?page=1&page_size=5" \
  -H "Authorization: Bearer $TOKEN" | \
  python3 -c "import sys,json; d=json.load(sys.stdin); data=d['data']
  keys=sorted(data.keys())
  print('Fields:', keys)
  assert 'page' in data, 'Missing page field'
  assert 'pageSize' in data, 'Missing pageSize field'
  assert 'total' in data, 'Missing total field'
  assert 'totalPages' in data, 'Missing totalPages field'
  assert 'tickets' in data, 'Missing tickets field'
  print('PASS: All required pagination fields present')
  print(f'totalPages={data[\"totalPages\"]}, total={data[\"total\"]}')"
```

**Expected**: `PASS` with all 5 fields present.

---

### Scenario 2: Assets API Pagination (Previously Broken)

```bash
curl -s "http://localhost:8090/api/v1/assets?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN" | \
  python3 -c "import sys,json; d=json.load(sys.stdin); data=d['data']
  keys=sorted(data.keys())
  print('Fields:', keys)
  assert 'page' in data, 'Missing page field'
  assert 'pageSize' in data, 'Missing pageSize field'
  assert 'totalPages' in data, 'Missing totalPages field'
  print('PASS: Assets API now has pagination metadata')"
```

**Expected**: `PASS` (previously failed, now passes).

---

### Scenario 3: Changes API Uses pageSize (Not size)

```bash
curl -s "http://localhost:8090/api/v1/changes?page=1&page_size=5" \
  -H "Authorization: Bearer $TOKEN" | \
  python3 -c "import sys,json; d=json.load(sys.stdin); data=d['data']
  keys=sorted(data.keys())
  print('Fields:', keys)
  assert 'pageSize' in data, 'Missing pageSize field'
  assert 'size' not in data, 'Old size field still present'
  assert 'totalPages' in data, 'Missing totalPages field'
  print('PASS: Changes API uses pageSize instead of size')"
```

**Expected**: `PASS` (previously had `size`, now has `pageSize`).

---

### Scenario 4: Incidents API No Redundant items Field

```bash
curl -s "http://localhost:8090/api/v1/incidents?page=1&page_size=3" \
  -H "Authorization: Bearer $TOKEN" | \
  python3 -c "import sys,json; d=json.load(sys.stdin); data=d['data']
  keys=sorted(data.keys())
  print('Fields:', keys)
  assert 'items' not in data, 'Redundant items field still present'
  assert 'totalPages' in data, 'Missing totalPages field'
  print('PASS: No redundant items field')"
```

**Expected**: `PASS` (redundant `items` removed).

---

### Scenario 5: SLA Policies Seed Data

```bash
# Check sla_policies table
docker exec itsm-postgres psql -U itsm -d itsm -c \
  "SELECT name, priority, response_time_minutes, resolution_time_minutes FROM sla_policies;" 2>&1
```

**Expected**: At least 3 rows (P1/P2/P3).

---

### Scenario 6: Backend Build & Tests

```bash
cd itsm-backend
go build ./...
go vet ./...
go test ./pkg/seeder/... -v
```

**Expected**: All build, vet, and tests pass.

---

## Run All Validations

```bash
#!/bin/bash
TOKEN=$(curl -s -X POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")

echo "=== Scenario 1: Tickets ==="
curl -s "http://localhost:8090/api/v1/tickets?page=1&page_size=5" \
  -H "Authorization: Bearer $TOKEN" | \
  python3 -c "import sys,json; d=json.load(sys.stdin); data=d['data']
  print('Fields:', sorted(data.keys()))
  assert 'totalPages' in data, 'FAIL: missing totalPages'"
echo "PASS"

echo "=== Scenario 2: Assets ==="
curl -s "http://localhost:8090/api/v1/assets?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN" | \
  python3 -c "import sys,json; d=json.load(sys.stdin); data=d['data']
  print('Fields:', sorted(data.keys()))
  assert 'page' in data, 'FAIL: missing page'"
echo "PASS"

echo "=== Scenario 3: Changes ==="
curl -s "http://localhost:8090/api/v1/changes?page=1&page_size=5" \
  -H "Authorization: Bearer $TOKEN" | \
  python3 -c "import sys,json; d=json.load(sys.stdin); data=d['data']
  assert 'pageSize' in data, 'FAIL: missing pageSize'
  assert 'size' not in data, 'FAIL: old size field present'"
echo "PASS"

echo "=== All scenarios passed ==="
```
