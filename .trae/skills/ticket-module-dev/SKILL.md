---
name: "ticket-module-dev"
description: "Provides guidelines, code templates, and validation for Ticket Management development. Invoke when user wants to add features, fix bugs, or refactor ticket-related code."
---

# Ticket Module Development Assistant

This skill encapsulates the best practices and architectural standards for the ITSM Ticket Management module. Use it to guide development, ensure consistency, and avoid common pitfalls.

## Core Architecture

-   **Backend**: Go (Gin + Ent).
    -   **Controller**: Handles HTTP requests, validation, and response formatting.
    -   **Service**: Contains business logic (SLA calculation, workflow triggers, notifications).
    -   **Ent Schema**: Defines data models and relationships.
-   **Frontend**: Next.js + Ant Design.
    -   **API Layer**: `src/lib/api/ticket-api.ts` (Ensure HTTP methods match backend).
    -   **Types**: `src/lib/api/api-config.ts` (Must align with Backend DTOs).

## Development Guidelines

### 1. API Consistency Check
*   **Method Matching**: Always verify the HTTP method defined in Backend `router.go` matches the Frontend `ticket-api.ts`.
    *   *Example*: Use `PUT` for updates if backend defines `PUT /:id`. Avoid `PATCH` unless explicitly supported.
*   **DTO Alignment**: Frontend interfaces (`CreateTicketRequest`) MUST match Backend DTO JSON tags.
    *   *Critical*: Watch out for `int` vs `string` mismatches (e.g., `category_id` vs `category`).

### 2. Data Handling
*   **Categories/Priorities**:
    *   Prefer using **IDs** for relationships (e.g., `category_id`) over names.
    *   If using names, ensure Backend has logic to lookup IDs or fall back gracefully.
*   **Validation**:
    *   Frontend: Use Ant Design Form rules.
    *   Backend: Use `binding:"required"` tags in DTOs.

### 3. Testing Standard
*   Always run the E2E test suite after modifying ticket logic:
    ```bash
    # Run specific ticket test
    source tests/e2e/venv/bin/activate
    pytest -s tests/e2e/test_tickets_full.py
    ```

## Common Pitfalls & Fixes

| Issue | Symptom | Fix |
|-------|---------|-----|
| **404/405 on Update** | Frontend calls `PATCH`, Backend has `PUT` | Change Frontend to `httpClient.put` |
| **Missing Category** | Ticket created without category | Ensure Frontend sends `category_id` OR Backend implements lookup by name |
| **CORS Error** | Login fails with Network Error | Check `cors.go` allows Origin headers properly |

## Code Templates

### Backend: Lookup Category by Name (Service Layer)
```go
if req.CategoryID != nil && *req.CategoryID > 0 {
    createBuilder = createBuilder.SetCategoryID(*req.CategoryID)
} else if req.Category != "" {
    // Lookup by name
    cat, err := s.client.TicketCategory.Query().
        Where(ticketcategory.NameEQ(req.Category), ticketcategory.TenantID(tenantID)).
        First(ctx)
    if err == nil {
        createBuilder = createBuilder.SetCategoryID(cat.ID)
    }
}
```

### Frontend: Create Ticket Request (Interface)
```typescript
export interface CreateTicketRequest {
  title: string;
  description: string;
  priority: string;
  category_id?: number; // Preferred
  category?: string;    // Fallback
  // ...
}
```
