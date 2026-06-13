# Data Model: API Contracts Fix

## Paginated List Response (Unified)

All list endpoints MUST return the following structure:

```go
type PaginatedListResponse struct {
    Items     []any `json:"items"`       // List data (varies by entity)
    Total     int   `json:"total"`       // Total record count
    Page      int   `json:"page"`         // Current page (1-based)
    PageSize  int   `json:"pageSize"`     // Records per page
    TotalPages int  `json:"totalPages"`   // Total page count
}
```

### Calculation Rule

```
totalPages = ceil(total / pageSize)
```

When `total == 0`, `totalPages = 0`.

---

## Affected DTOs

| DTO | Entity | List Field | Changes |
|-----|--------|-----------|---------|
| `TicketListResponse` | Ticket | `tickets[]` | + `totalPages` |
| `IncidentListResponse` | Incident | `incidents[]` | - `items[]`, + `totalPages` |
| `ListProblemsResponse` | Problem | `problems[]` | + `totalPages` |
| `ChangeListResponse` | Change | `changes[]` | `size` → `pageSize`, + `totalPages` |
| `AssetListResponse` | Asset | `assets[]` | + `page`, `pageSize`, `totalPages` |

---

## SLAPolicy Entity (New Seed Data)

### Schema Fields

| Field | Type | Nullable | Default | Description |
|-------|------|---------|---------|-------------|
| `id` | bigint | NO | auto | Primary key |
| `name` | varchar | NO | - | SLA policy name |
| `description` | text | YES | - | Description |
| `customer_tier` | varchar | YES | - | platinum/gold/silver/bronze |
| `ticket_type` | varchar | YES | - | incident/problem/change/request |
| `priority` | varchar | YES | - | critical/high/medium/low |
| `response_time_minutes` | int | NO | - | Response SLA in minutes |
| `resolution_time_minutes` | int | NO | - | Resolution SLA in minutes |
| `business_hours` | jsonb | YES | - | Business hours config |
| `exclude_weekends` | bool | NO | false | Exclude weekends |
| `exclude_holidays` | bool | NO | false | Exclude holidays |
| `escalation_rules` | jsonb | YES | - | Escalation rules config |
| `is_active` | bool | NO | true | Active status |
| `priority_score` | int | NO | 0 | Sort priority |
| `tenant_id` | bigint | NO | - | Tenant ID |

### Relationships

- `sla_policies` → `tenants` (tenant_id FK)
- `sla_policies` → `sla_definitions` (via name/service_type)

### Preset Values

| name | priority | response_time_minutes | resolution_time_minutes | is_active |
|------|---------|----------------------|------------------------|-----------|
| P1-紧急响应 | critical | 15 | 60 | true |
| P2-标准响应 | high | 30 | 240 | true |
| P3-一般响应 | medium | 120 | 480 | true |

---

## SeedConfig Extension

```go
type SeedConfig struct {
    // ... existing fields ...
    SLAPolicies []SLAPolicySeed `json:"sla_policies"`
}

type SLAPolicySeed struct {
    Name                  string `json:"name"`
    Description           string `json:"description"`
    CustomerTier         string `json:"customer_tier"`
    TicketType           string `json:"ticket_type"`
    Priority             string `json:"priority"`
    ResponseTimeMinutes  int    `json:"response_time_minutes"`
    ResolutionTimeMinutes int    `json:"resolution_time_minutes"`
    ExcludeWeekends      bool   `json:"exclude_weekends"`
    ExcludeHolidays      bool   `json:"exclude_holidays"`
    IsActive             bool   `json:"is_active"`
    PriorityScore        int    `json:"priority_score"`
}
```
