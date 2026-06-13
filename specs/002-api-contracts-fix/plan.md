# Implementation Plan: API Contracts Fix

**Branch**: `002-api-contracts-fix` | **Date**: 2026-06-13 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-api-contracts-fix/spec.md`

## Summary

统一 5 个 API 端点的分页响应格式（补充 totalPages、修复 size→pageSize、删除 items 冗余字段），为 assets 接口补全分页元数据，以及补充 SLA policies 种子数据。属于 Bug Fix + Data Seeding 类型，无需新增 API 路由。

## Technical Context

**Language/Version**: Go 1.22, TypeScript (前端 Next.js 14)

**Primary Dependencies**: Gin (HTTP), Ent ORM (DB), PostgreSQL 15

**Storage**: PostgreSQL

**Testing**: go test, Jest + Playwright

**Target Platform**: Linux (Docker container)

**Project Type**: Web Service + SPA (itsm-backend + itsm-frontend)

**Performance Goals**: API 响应时间 < 200ms（分页查询）

**Scale/Scope**: 5 个 DTO 文件修改，1 个 Service 文件修改，1 个 Seeder 文件修改

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Notes |
|------|--------|-------|
| 不引入新的外部依赖 | ✅ PASS | 仅修改现有代码 |
| 不破坏现有 API 兼容性 | ✅ PASS | 仅增加字段，不修改现有字段 |
| 后端修改需通过 go build + go vet | ✅ PASS | 待验证 |
| Seeder 修改需通过单元测试 | ✅ PASS | 待验证 |

## Project Structure

### Documentation (this feature)

```text
specs/002-api-contracts-fix/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (API contract specs)
└── tasks.md             # Phase 2 output
```

### Source Code (repository root)

```text
itsm-backend/
├── dto/
│   ├── ticket_dto.go          # + totalPages
│   ├── incident_dto.go        # - items, + totalPages
│   ├── problem_dto.go          # + totalPages
│   ├── change_dto.go          # size→pageSize, + totalPages
│   └── asset_dto.go           # + page, pageSize, totalPages
├── service/
│   └── asset_service.go       # 返回 totalPages
└── pkg/seeder/
    └── seeder.go              # + SLAPolicies seeding
```

**Structure Decision**: 标准 Go 项目结构，仅修改现有文件，不新增文件

## Phase 1: Data Model Changes

### DTO Changes

#### 1. `dto/ticket_dto.go` - TicketListResponse
```go
// BEFORE
type ListTicketsResponse struct {
    Tickets  []*TicketResponse `json:"tickets"`
    Total    int              `json:"total"`
    Page     int              `json:"page"`
    PageSize int              `json:"pageSize"`
}

// AFTER
type ListTicketsResponse struct {
    Tickets     []*TicketResponse `json:"tickets"`
    Total       int               `json:"total"`
    Page        int               `json:"page"`
    PageSize    int               `json:"pageSize"`
    TotalPages  int               `json:"totalPages"`
}
```

#### 2. `dto/incident_dto.go` - IncidentListResponse
```go
// BEFORE
type IncidentListResponse struct {
    Incidents  []*IncidentResponse `json:"incidents"`
    Items      []*IncidentResponse `json:"items,omitempty"`
    Total      int                 `json:"total"`
    Page       int                 `json:"page"`
    PageSize   int                 `json:"pageSize"`
    TotalPages int                 `json:"totalPages"`
}

// AFTER
type IncidentListResponse struct {
    Incidents  []*IncidentResponse `json:"incidents"`
    Total      int                `json:"total"`
    Page       int                `json:"page"`
    PageSize   int                `json:"pageSize"`
    TotalPages int                `json:"totalPages"`
}
```

#### 3. `dto/problem_dto.go` - ListProblemsResponse
```go
// BEFORE
type ListProblemsResponse struct {
    Problems []*ProblemResponse `json:"problems"`
    Total    int                `json:"total"`
    Page     int                `json:"page"`
    PageSize int                `json:"pageSize"`
}

// AFTER
type ListProblemsResponse struct {
    Problems    []*ProblemResponse `json:"problems"`
    Total       int                `json:"total"`
    Page        int                `json:"page"`
    PageSize    int                `json:"pageSize"`
    TotalPages  int                `json:"totalPages"`
}
```

#### 4. `dto/change_dto.go` - ChangeListResponse
```go
// BEFORE
type ChangeListResponse struct {
    Total   int               `json:"total"`
    Changes []ChangeResponse  `json:"changes"`
    Page    int               `json:"page"`
    Size    int               `json:"size"`  // ← 错误字段名
}

// AFTER
type ChangeListResponse struct {
    Total      int              `json:"total"`
    Changes    []ChangeResponse `json:"changes"`
    Page       int             `json:"page"`
    PageSize   int             `json:"pageSize"`
    TotalPages int             `json:"totalPages"`
}
```

#### 5. `dto/asset_dto.go` - AssetListResponse
```go
// BEFORE
type AssetListResponse struct {
    Total  int             `json:"total"`
    Assets []AssetResponse `json:"assets"`
}

// AFTER
type AssetListResponse struct {
    Total      int             `json:"total"`
    Assets     []AssetResponse `json:"assets"`
    Page       int             `json:"page"`
    PageSize   int             `json:"pageSize"`
    TotalPages int             `json:"totalPages"`
}
```

### Seeder Changes

#### 1. Add SLAPolicySeed type and SLAPolicies field in SeedConfig
#### 2. Add seedSLAPolicies function
#### 3. Add seedSLAPolicies call in seed function
#### 4. Add default.json seeder data

## Phase 2: Implementation Tasks

See [tasks.md](./tasks.md)

## Verification

- [ ] `go build ./...` 通过
- [ ] `go vet ./...` 无警告
- [ ] `go test ./pkg/seeder/...` 通过
- [ ] API 响应字段格式验证通过（见 quickstart.md）
