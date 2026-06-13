# API Contracts: List Endpoints

All list endpoints follow the unified API response envelope:

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

## Unified Paginated List Response

All list endpoints MUST return this structure in the `data` field:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [...],       // entity list
    "total": 30,          // total record count
    "page": 1,            // current page (1-based)
    "pageSize": 10,       // records per page
    "totalPages": 3       // total page count
  }
}
```

**Note**: Some endpoints use entity-specific field names (e.g., `tickets[]` instead of `items[]`) for backwards compatibility. The four pagination metadata fields (`total`, `page`, `pageSize`, `totalPages`) are always present.

---

## Contract: GET /api/v1/tickets

**Query Parameters**:
- `page` (int, optional, default=1, min=1)
- `page_size` (int, optional, default=10, min=1, max=100)

**Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "tickets": [...],
    "total": 30,
    "page": 1,
    "pageSize": 10,
    "totalPages": 3
  }
}
```

---

## Contract: GET /api/v1/incidents

**Query Parameters**:
- `page` (int, optional, default=1)
- `page_size` (int, optional, default=10, max=100)

**Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "incidents": [...],
    "total": 26,
    "page": 1,
    "pageSize": 10,
    "totalPages": 3
  }
}
```

**Note**: `items` field removed (was redundant with `incidents`).

---

## Contract: GET /api/v1/problems

**Query Parameters**:
- `page` (int, optional, default=1)
- `page_size` (int, optional, default=10, max=100)

**Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "problems": [...],
    "total": 20,
    "page": 1,
    "pageSize": 10,
    "totalPages": 2
  }
}
```

---

## Contract: GET /api/v1/changes

**Query Parameters**:
- `page` (int, optional, default=1)
- `page_size` (int, optional, default=10, max=100)

**Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "changes": [...],
    "total": 22,
    "page": 1,
    "pageSize": 10,
    "totalPages": 3
  }
}
```

**Note**: `size` field renamed to `pageSize` (breaking change for existing clients).

---

## Contract: GET /api/v1/assets

**Query Parameters**:
- `page` (int, optional, default=1, min=1)
- `page_size` (int, optional, default=10, max=100)

**Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "assets": [...],
    "total": 0,
    "page": 1,
    "pageSize": 10,
    "totalPages": 0
  }
}
```

**Note**: Previously returned only `total` + `assets[]`. Now includes full pagination metadata.

---

## Error Responses

| code | message | scenario |
|------|---------|----------|
| 1001 | 参数错误 | page < 1 or pageSize > 100 |
| 5001 | 系统错误 | database error |
| 2001 | 认证失败 | missing/invalid token |
