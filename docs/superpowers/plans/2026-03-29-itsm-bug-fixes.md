# ITSM Bug 修复计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复测试中发现的3个bug：CI创建失败、服务请求缺少更新/删除路由、change_approvals表缺失

**Architecture:** 修复控制器逻辑、添加路由、创建数据库迁移

**Tech Stack:** Go/Gin, PostgreSQL, Ent ORM

---

## 问题汇总

| # | 问题 | 位置 | 严重程度 |
|---|------|------|----------|
| 1 | CI创建时ciType为空导致5001错误 | `controller/cmdb_controller.go:59-65` | 高 |
| 2 | 服务请求缺少PUT/DELETE路由 | `router/router.go:511-519` | 高 |
| 3 | change_approvals表不存在 | 数据库迁移缺失 | 高 |

---

## Task 1: 修复 CI 创建时 ciType 为空的 bug

**Files:**
- Modify: `itsm-backend/controller/cmdb_controller.go:59-65`

**问题分析:**
当 `dtoReq.Type` 为空且 `dtoReq.CITypeID > 0` 时，代码尝试通过 `c.dddSvc.GetType` 获取类型名称。但如果 `CITypeID` 无效或类型不存在，错误被静默忽略，导致 `ciType` 保持为空字符串，最终在 `cmdbService.CreateCI` 时因 ciType 验证失败返回5001错误。

**修复方案:**
当 `ciType` 最终仍为空时，返回明确的参数错误提示。

- [ ] **Step 1: 修改 CreateCI 方法添加 ciType 空值校验**

```go
// 将 ci_type_id 解析为 ci_type 字符串
ciType := dtoReq.Type
if ciType == "" && dtoReq.CITypeID > 0 {
    if ct, err := c.dddSvc.GetType(ctx.Request.Context(), dtoReq.CITypeID, tenantID); err == nil {
        ciType = ct.Name
    }
}

// 验证 ciType 不为空
if ciType == "" {
    common.Fail(ctx, common.ParamErrorCode, "CI类型不能为空，请提供有效的 type 或 ci_type_id")
    return
}
```

- [ ] **Step 2: 验证修复**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend && go build -o /tmp/itsm-backend main.go
```

预期: 编译成功

- [ ] **Step 3: 提交代码**

```bash
git add itsm-backend/controller/cmdb_controller.go
git commit -m "fix: validate ciType is not empty in CreateCI"
```

---

## Task 2: 添加服务请求的更新和删除路由

**Files:**
- Modify: `itsm-backend/router/router.go:511-519`
- Modify: `itsm-backend/service/bpmn/service_request_handler.go`

**问题分析:**
服务请求路由组 (`/service-requests`) 只注册了 Create、List、Get、ApplyApproval 路由，缺少 Update 和 Delete 路由。测试发现 PUT 和 DELETE 请求返回 404。

**修复方案:**
1. 在 handler 中添加 `UpdateServiceRequest` 和 `DeleteServiceRequest` 方法
2. 在 router 中注册 PUT 和 DELETE 路由

- [ ] **Step 1: 在 service_request_handler.go 添加 Update 和 Delete 方法**

```go
// UpdateServiceRequest 更新服务请求
func (h *ServiceRequestHandler) UpdateServiceRequest(ctx *gin.Context) {
    idStr := ctx.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        common.ParamError(ctx, "Invalid service request ID")
        return
    }

    tenantID := ctx.GetInt("tenant_id")
    if tenantID == 0 {
        common.Fail(ctx, common.AuthFailedCode, "Tenant ID not found")
        return
    }

    var req dto.UpdateServiceRequestDTO
    if err := ctx.ShouldBindJSON(&req); err != nil {
        common.ParamError(ctx, "Invalid request body")
        return
    }

    svcReq, err := h.service.GetServiceRequest(ctx.Request.Context(), id, tenantID)
    if err != nil {
        common.Fail(ctx, common.NotFoundCode, "Service request not found")
        return
    }

    if req.Title != "" {
        svcReq.Title = req.Title
    }
    if req.Description != "" {
        svcReq.Description = req.Description
    }
    if req.Priority != "" {
        svcReq.Priority = req.Priority
    }

    if err := h.service.UpdateServiceRequest(ctx.Request.Context(), svcReq); err != nil {
        common.Fail(ctx, common.InternalErrorCode, "更新服务请求失败")
        return
    }

    common.Success(ctx, svcReq)
}

// DeleteServiceRequest 删除服务请求
func (h *ServiceRequestHandler) DeleteServiceRequest(ctx *gin.Context) {
    idStr := ctx.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        common.ParamError(ctx, "Invalid service request ID")
        return
    }

    tenantID := ctx.GetInt("tenant_id")
    if tenantID == 0 {
        common.Fail(ctx, common.AuthFailedCode, "Tenant ID not found")
        return
    }

    if err := h.service.DeleteServiceRequest(ctx.Request.Context(), id, tenantID); err != nil {
        common.Fail(ctx, common.InternalErrorCode, "删除服务请求失败")
        return
    }

    common.Success(ctx, nil)
}
```

- [ ] **Step 2: 在 router.go 添加 PUT 和 DELETE 路由**

在 `router.go` 第511-519行的服务请求路由组中添加:

```go
if config.ServiceRequestHandler != nil {
    sr := tenant.(*gin.RouterGroup).Group("/service-requests")
    {
        sr.POST("", middleware.RequirePermission("service_request", "write"), config.ServiceRequestHandler.Create)
        sr.GET("", middleware.RequirePermission("service_request", "read"), config.ServiceRequestHandler.List)
        sr.GET("/me", middleware.RequirePermission("service_request", "read"), config.ServiceRequestHandler.List)
        sr.GET("/approvals/pending", middleware.RequirePermission("service_request", "read"), config.ServiceRequestHandler.ListPending)
        sr.GET("/:id", middleware.RequirePermission("service_request", "read"), config.ServiceRequestHandler.Get)
        sr.PUT("/:id", middleware.RequirePermission("service_request", "write"), config.ServiceRequestHandler.UpdateServiceRequest)
        sr.DELETE("/:id", middleware.RequirePermission("service_request", "delete"), config.ServiceRequestHandler.DeleteServiceRequest)
        sr.POST("/:id/approval", middleware.RequirePermission("service_request", "write"), config.ServiceRequestHandler.ApplyApproval)
    }
}
```

- [ ] **Step 3: 在 dto 包中添加 UpdateServiceRequestDTO**

检查 `dto.UpdateServiceRequestDTO` 是否存在，如不存在则添加:

```go
type UpdateServiceRequestDTO struct {
    Title       string `json:"title"`
    Description string `json:"description"`
    Priority    string `json:"priority"`
}
```

- [ ] **Step 4: 验证修复**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend && go build -o /tmp/itsm-backend main.go
```

预期: 编译成功

- [ ] **Step 5: 提交代码**

```bash
git add itsm-backend/router/router.go itsm-backend/service/bpmn/service_request_handler.go
git commit -m "feat: add PUT/DELETE routes for service-requests"
```

---

## Task 3: 添加 change_approvals 数据库迁移

**Files:**
- Modify: `itsm-backend/migration/migrations.go`

**问题分析:**
`change_approval_service.go` 和 `handlers/change/repository_impl.go` 使用 `change_approvals` 表，但该表从未被创建。查询变更审批记录时返回 "relation 'change_approvals' does not exist" 错误。

**修复方案:**
在迁移文件中添加创建 `change_approvals` 表的迁移。

- [ ] **Step 1: 在 migrations.go 中添加 change_approvals 迁移**

在 `RegisteredMigrations` 数组中添加:

```go
{
    Version:     "006_add_change_approvals",
    Description: "Add change approvals table for change workflow",
    RollbackSQL: `DROP TABLE IF EXISTS change_approvals;`,
},
```

在 `GetMigrationSQL` 函数中添加:

```go
case "006_add_change_approvals":
    return `
CREATE TABLE IF NOT EXISTS change_approvals (
    id SERIAL PRIMARY KEY,
    change_id INTEGER NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
    approver_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    comment TEXT,
    decided_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_change_approvals_change ON change_approvals(change_id);
CREATE INDEX IF NOT EXISTS idx_change_approvals_approver ON change_approvals(approver_id);
CREATE INDEX IF NOT EXISTS idx_change_approvals_status ON change_approvals(status);
`
```

- [ ] **Step 2: 运行迁移**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend && go run -tags migrate main.go -up
```

预期: 应用迁移成功，创建 `change_approvals` 表

- [ ] **Step 3: 提交代码**

```bash
git add itsm-backend/migration/migrations.go
git commit -m "feat: add migration for change_approvals table"
```

---

## 修复后验证

修复完成后，重新测试以下场景:

1. **CI创建测试**: 使用有效的 ci_type_id 或 type 创建CI，验证返回成功
2. **服务请求更新/删除测试**: 使用 PUT/DELETE `/api/v1/service-requests/:id`，验证返回成功
3. **变更审批记录测试**: 查询 GET `/api/v1/changes/:id/approvals`，验证返回审批记录

---

## 预期结果

| 测试项 | 修复前 | 修复后 |
|--------|--------|--------|
| CI创建 (有效ci_type_id) | FAIL (5001) | PASS |
| 服务请求更新 | FAIL (404) | PASS |
| 服务请求删除 | FAIL (404) | PASS |
| 变更审批记录查询 | FAIL (表不存在) | PASS |

---

**Plan complete and saved to `docs/superpowers/plans/2026-03-29-itsm-bug-fixes.md`**

**Two execution options:**

**1. Subagent-Driven (recommended)** - 每个bug分配一个 subagent 修复，并行处理

**2. Inline Execution** - 在当前 session 中按批次执行修复，有检查点

**选择哪个方式?**
