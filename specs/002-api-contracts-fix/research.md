# Research: API Contracts Fix

**Feature**: 002-api-contracts-fix
**Date**: 2026-06-13

---

## Decision 1: 统一分页响应格式

### Current State

所有列表接口响应格式不一致，字段名和字段缺失情况如下：

| DTO | 字段 | 状态 |
|-----|------|------|
| `TicketListResponse` | `tickets[]`, `total`, `page`, `pageSize` | 缺 `totalPages` |
| `IncidentListResponse` | `incidents[]`, `items[]`（冗余）, `total`, `page`, `pageSize`, `totalPages` | 有冗余 `items` |
| `ListProblemsResponse` | `problems[]`, `total`, `page`, `pageSize` | 缺 `totalPages` |
| `ChangeListResponse` | `total`, `changes[]`, `page`, `size` | 用 `size` 而非 `pageSize`，缺 `totalPages` |
| `AssetListResponse` | `total`, `assets[]` | **完全缺失所有分页字段** |

### Chosen Format

标准分页响应结构（4 个字段）：
```go
type PaginatedListResponse struct {
    Items     []any `json:"items"`      // 列表数据
    Total     int   `json:"total"`       // 总记录数
    Page      int   `json:"page"`         // 当前页（从 1 开始）
    PageSize  int   `json:"pageSize"`    // 每页条数
    TotalPages int  `json:"totalPages"`  // 总页数
}
```

### Alternatives Considered

- **仅返回 `page`/`pageSize`/`total` 三字段**：缺少 `totalPages` 导致前端无法计算总页数
- **统一用 `items` 作为列表字段名**：需要改所有现有 DTO 的列表字段名（`tickets`/`incidents`/`problems` 等），影响较大
- **保持现有字段 + 补充缺失字段**：最小改动，仅增加 `totalPages` 和修正 `size`

### Decision

采用**最小化改动方案**：保持现有列表字段名不变，统一补充缺失的分页字段。

### Affected Files

1. `dto/ticket_dto.go` → `TicketListResponse` + `totalPages`
2. `dto/incident_dto.go` → `IncidentListResponse` - `items` + `totalPages`
3. `dto/problem_dto.go` → `ListProblemsResponse` + `totalPages`
4. `dto/change_dto.go` → `ChangeListResponse` `size` → `pageSize` + `totalPages`
5. `dto/asset_dto.go` → `AssetListResponse` + `page`, `pageSize`, `totalPages`

---

## Decision 2: 补充 SLA Policies 种子数据

### Current State

- `sla_definitions` 表：6 条预设数据 ✅
- `sla_policies` 表：**0 条数据** ⚠️
- Seeder 无 `SlaPolicySeed` 类型和 `seedSLAPolicies` 函数

### Schema Analysis

`sla_policies` 表关键字段：
- `name`, `description`, `customer_tier`, `ticket_type`, `priority`
- `response_time_minutes`, `resolution_time_minutes`
- `business_hours` (JSON), `exclude_weekends`, `exclude_holidays`
- `escalation_rules` (JSON), `is_active`, `priority_score`, `tenant_id`

### Chosen Approach

在 `SeedConfig` 中添加 `SLAPolicies []SLAPolicySeed`，并在 seeder 中添加 `seedSLAPolicies` 函数，预设 3 条 P1/P2/P3 响应时间策略。

### Preset SLA Policies

| 名称 | Priority | 响应时间(min) | 解决时间(min) |
|------|---------|-------------|-------------|
| P1-紧急响应 | critical | 15 | 60 |
| P2-标准响应 | high | 30 | 240 |
| P3-一般响应 | medium | 120 | 480 |

### Affected Files

1. `pkg/seeder/seeder.go` → 添加 `SLAPolicies` 字段 + `SLAPolicySeed` 类型 + `seedSLAPolicies` 函数
2. `config/seed/default.json` → 添加 `sla_policies` 数组

---

## Decision 3: 修复 Assets 接口分页

### Current State

- Controller: ✅ 已接受 `page` 和 `pageSize` 查询参数
- Service: ✅ 已传入 `page` 和 `pageSize`
- DTO: ❌ `AssetListResponse` 仅返回 `total` + `assets[]`，无分页元数据

### Root Cause

DTO 缺少返回 `page`/`pageSize`/`totalPages` 字段。Service 层需要计算并返回 totalPages。

### Affected Files

1. `dto/asset_dto.go` → `AssetListResponse` 增加 `page`, `pageSize`, `totalPages`
2. `service/asset_service.go` → `ListAssets` 返回 `totalPages` 计算值

---

## Risk Assessment

| 风险 | 影响 | 缓解 |
|------|------|------|
| 前端依赖现有字段名 | 中 | 仅增加字段，不改现有字段 |
| SLA policies 关联工单逻辑 | 低 | 仅添加种子数据，不改关联逻辑 |
| incidents 多余 items 字段 | 中 | 确认前端未使用 items 字段后才删除 |
