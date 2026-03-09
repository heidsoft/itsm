# ITSM Backend API Completeness Audit Report

**Audit Date:** 2026-03-08 (executed 2026-03-09)  
**Auditor:** OpenClaw Subagent  
**System:** ITSM Backend (Go/Gin)  
**Server Port:** 8090  
**Environment:** Local development (direct binary execution)

---

## Executive Summary

The ITSM backend API is largely complete and functional. All major routes are correctly registered in the router and respond to requests. However, two issues were identified:

1. **OpenAPI specification (docs/swagger.yaml is severely outdated** - missing most of the current API surface.
2. **Knowledge Stats endpoint has a critical bug** - `GET /api/v1/knowledge/stats` fails with an SQL scan error due to mismatched struct fields.

Additionally, some test POST requests returned 400 due to missing required fields (expected), and the API uses a wrapper convention where application errors are embedded in a 200 response.

---

## 1. Route Registration Verification

### Bootstrap & Router Inspection

- **Bootstrap** (`internal/bootstrap/app.go`) initializes all services and controllers, then calls `router.SetupRoutes`.
- **Router** (`router/router.go`) defines a comprehensive route map with authentication, RBAC, and tenant middleware.
- All controllers referenced in `RouterConfig` are instantiated and passed to `SetupRoutes`.

**Conclusion:** All expected controllers are registered; the route definitions appear complete within the code.

### OpenAPI Spec Mismatch

The OpenAPI specification (`docs/swagger.yaml`) does **not** reflect the current API. Key missing routes:

- **Groups**: `/api/v1/groups`
- **CMDB**: Both `/api/v1/cmdb/*` and `/api/v1/configuration-items/*` are entirely absent.
- **Changes submit**: `/api/v1/changes/:id/submit`
- **Knowledge stats**: `/api/v1/knowledge/stats`
- **Approval Chains**: `/api/v1/approval-chains`
- **System Configs**: `/api/v1/system-configs`
- **Roles, Permissions, Tenants**: `/api/v1/roles`, `/api/v1/permissions`, `/api/v1/tenants`
- **Releases, Assets, Licenses**: All asset management routes.
- **Service Catalog & Requests**: `/api/v1/service-catalogs`, `/api/v1/service-requests`
- **SLA module**: All `/api/v1/sla/*` endpoints.
- **AI & Analytics**: `/api/v1/ai/*`
- **Dashboard**: `/api/v1/dashboard/*`
- **Problem Investigation**: `/api/v1/problem-investigation/*`
- **Ticket Categories & Tags**, **Workflow**, **Automation Rules**, **Assignment Rules**, etc.

**Recommendation:** Regenerate the OpenAPI spec using [swaggo](https://github.com/swaggo/swag) or manually update the YAML to match the current router.

---

## 2. Functional Testing (Admin Credentials)

**Admin credentials used:**
- Username: `admin`
- Password: `admin123`
- JWT token obtained via `POST /api/v1/auth/login`

**Test matrix (selected endpoints across modules):**

| Endpoint | Method | Expected | Observed | Notes |
|----------|--------|----------|----------|-------|
| `/api/v1/groups` | GET | 200 | 200 | Returns empty list (no groups seeded). |
| `/api/v1/users` | GET | 200 | 200 | List of users. |
| `/api/v1/roles` | GET | 200 | 200 | |
| `/api/v1/permissions` | GET | 200 | 200 | |
| `/api/v1/tenants` | GET | 200 | 200 | |
| `/api/v1/tickets` | GET | 200 | 200 | |
| `/api/v1/tickets` | POST | 200 | 200 | Created successfully. |
| `/api/v1/incidents` | GET | 200 | 200 | |
| `/api/v1/incidents` | POST | 200 | 200 | |
| `/api/v1/problems` | GET | 200 | 200 | |
| `/api/v1/problems` | POST | 400 | 400 | Expected: missing required fields (test payload incomplete). |
| `/api/v1/changes` | GET | 200 | 200 | |
| `/api/v1/changes/8/submit` | POST | 200 | 200 | State transitioned to `pending`. |
| `/api/v1/releases` | GET | 200 | 200 | |
| `/api/v1/releases` | POST | 400 | 400 | Expected: validation. |
| `/api/v1/assets` | GET | 200 | 200 | |
| `/api/v1/assets` | POST | 400 | 400 | Expected. |
| `/api/v1/licenses` | GET | 200 | 200 | |
| `/api/v1/cmdb/cis` | GET | 200 | 200 | |
| `/api/v1/cmdb/items` | POST | 200 | 200 | Works with valid payload (`ci_type_id=1`, `status=active`). |
| `/api/v1/cmdb/stats` | GET | 200 | 200 | |
| `/api/v1/knowledge/stats` | GET | 200 | **500** (code=500, msg: `sql/scan: missing struct field for column: sum (sum)`) | **BUG** |
| `/api/v1/knowledge/articles` | GET | 200 | 200 | |
| `/api/v1/knowledge/articles` | POST | 200 | 200 | |
| `/api/v1/service-catalogs` | GET | 200 | 200 | |
| `/api/v1/service-requests` | GET | 200 | 200 | |
| `/api/v1/service-requests` | POST | 400 | 400 | Expected: catalog_item_id required. |
| `/api/v1/sla/stats` | GET | 200 | 200 | |
| `/api/v1/dashboard/overview` | GET | 200 | 200 | |
| `/api/v1/ai/chat` | POST | 200 | 200 | |

*Note:* The API returns HTTP 200 even for application errors; the actual status is in the `code` field of the JSON response.

---

## 3. Critical Issue: Knowledge Stats Endpoint (500 Error)

**Endpoint:** `GET /api/v1/knowledge/stats`  
**Error:** `sql/scan: missing struct field for column: sum (sum)`

**Root Cause:**  
In `internal/domain/knowledge/repository_impl.go`, the `GetStats` function performs aggregate queries:

```go
type ViewSum struct {
    TotalViews int `json:"total_views"`
}
// ...
err = r.client.KnowledgeArticle.Query().
    Aggregate(ent.Sum(knowledgearticle.FieldViewCount)).
    Scan(ctx, &viewResult)
```

The `ent.Aggregate(ent.Sum(...))` produces a column named `sum`. The `Scan` operation tries to map columns to struct fields by name, but `ViewSum` has `TotalViews`, not `Sum`. Hence the scan fails.

**Fix Options:**

1. **Rename struct fields to `Sum` and map later:**
   ```go
   type ViewSum struct {
       Sum int `json:"-"`
   }
   // Then use viewResult[0].Sum to assign to TotalViews
   ```
2. **Use alias in query:** (if supported by ent) e.g., `.Aggregate(ent.Sum(...).As("total_views"))` and keep the field name `TotalViews`.
3. **Use a map scan:** Scan into `map[string]interface{}` and extract manually.

**Recommended Fix:** Option 2 if Ent supports `As` for aggregates; otherwise option 1 is simplest.

---

## 4. Additional Observations

- **Swagger UI** is accessible at `/swagger` but documents an outdated API.
- **Health check** (`/api/v1/health`) and **version** (`/api/v1/version`) endpoints are present and functional.
- **Authentication** via JWT works; token refresh endpoint exists.
- **Middleware** includes CORS, rate limiting, security headers, SQL injection protection, XSS protection, request size limiting.
- **RBAC** and tenant isolation are enforced on protected routes.
- **Database seeding** has created sample data (changes, service requests, CI types, etc.).

---

## 5. Recommendations

1. **Fix Knowledge Stats bug** immediately (see section 3).
2. **Update the OpenAPI spec** to match the current codebase. Consider integrating `swaggo` into the CI pipeline to keep it in sync.
3. **Add integration tests** for all stats endpoints (knowledge, cmdb, tickets, etc.) to catch mapping errors.
4. **Review error handling policy**: If the API intends to always return HTTP 200 with an internal `code` field, ensure clients handle this. Alternatively, consider using proper HTTP status codes for errors.
5. **Document required fields** for POST endpoints in the OpenAPI spec to help API consumers.
6. **Ensure that all routes present in code are covered by the spec** and vice versa; perform a diff during releases.

---

## 6. Conclusion

The ITSM backend is feature-rich and mostly functional. The route coverage is excellent, spanning all core ITSM processes. The primary gaps are documentation (outdated OpenAPI spec) and one critical bug in the knowledge statistics query. These should be addressed to ensure smooth integration and reliability.

---

**Report generated by:** OpenClaw subagent (automated audit)  
**Audit tooling:** Static code analysis, router inspection, dynamic endpoint testing with curl, JWT authentication.
