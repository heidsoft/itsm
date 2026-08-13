# Static Analysis Gates

本文档定义了 v1.1 收尾阶段 (Stage 5) 的 5 条静态门禁。每条门禁对应一个
shell 脚本，位于 `scripts/static-gates/`，由 `scripts/static-gates/run-all.sh`
统一调用。

| # | 规则 | 脚本 | 状态 | 阻断构建 |
|---|------|------|------|---------|
| 5.1 | 禁止 `c.JSON(...)` 绕过 `common.Success/Fail` | `check-bare-json.sh` | **HARD** | ✅ |
| 5.2 | `common.Fail` 必须把 2002/2004/2005 映射到 401/403/404 | `check-http-status-mapping.sh` | **HARD** | ✅ |
| 5.3 | 前端禁用 raw `fetch` / `axios`，统一走 BaseApi | `check-raw-fetch.sh` | ADVISORY | ❌ |
| 5.4 | `service` 层 `go func` 内不得裸用 `context.Background()` | `check-context-bg.sh` | ADVISORY | ❌ |
| 5.5 | `*ListResponse` 必须含 `items/total/page/pageSize/totalPages` 五元组 | `check-pagination-shape.sh` | ADVISORY | ❌ |

## 接入位置

### 本地开发

```bash
./scripts/static-gates/run-all.sh
```

### CI

在 `.github/workflows/backend-ci.yml` 与 `frontend-ci.yml` 的最后一步
加入：

```yaml
- name: Static analysis gates
  run: ./scripts/static-gates/run-all.sh
```

> 当前 5.3 / 5.4 / 5.5 为 advisory（exit 0），日志中可见违规命中；当历史
> 命中全部迁移完成后会切换为硬门禁（exit 1）。

---

## 5.1 — 禁止裸 c.JSON()

**目的**：所有 HTTP 响应必须经过 `common.Success / common.Fail /
common.SuccessWithList`，确保 `{code, message, data}` 三元组与 HTTP 状态码
映射契约不会被绕过。

**实现**：扫描 `itsm-backend/handlers`、`itsm-backend/service`、
`itsm-backend/controller` 下的 `.go` 文件（排除 `_test.go` / `_mock.go`），
匹配 `c.JSON(<digit>, …)`。

**当前状态**：✅ 通过。最近一次扫描未发现违规。

**修复示例**：

```go
// 反例：
c.JSON(http.StatusOK, gin.H{"foo": bar})

// 正例：
common.Success(c, gin.H{"foo": bar})
// 或
common.SuccessWithList(c, items, total, page, pageSize)
```

---

## 5.2 — HTTP 状态映射契约

**目的**：锁定 `common.Fail(c, code, msg)` 与 `common.FailWithData` 中
业务码 → HTTP 状态码的映射。这是 [对齐审计 P0 #3] 的回归防护：
- `2001` / `2002` (AuthFailed / Unauthorized) → **401**
- `2003` / `2004` (Forbidden / ToolPermissionDenied) → **403**
- `2005` (UnknownTool) / `4004` (NotFound) → **404**

**实现**：运行 `common` 包下 6 个固定名称的测试：
- `TestFail_Unauthorized2002`
- `TestFail_ToolPermissionDenied2004`
- `TestFail_UnknownTool2005`
- `TestFailWithData_Unauthorized2002`
- `TestFailWithData_ToolPermissionDenied2004`
- `TestFailWithData_UnknownTool2005`

测试代码位于 `itsm-backend/common/response_test.go`，当 switch 分支被改动
时（删除或改名）会立即失败。

**当前状态**：✅ 通过。映射关系已写入 `common/response.go`：

```go
case AuthFailedCode, UnauthorizedCode:
    statusCode = http.StatusUnauthorized
case ForbiddenCode, ToolPermissionDeniedCode:
    statusCode = http.StatusForbidden
case NotFoundCode, UnknownToolCode:
    statusCode = http.StatusNotFound
```

---

## 5.3 — 前端禁用 raw fetch / axios

**目的**：所有 HTTP 调用必须经过 `BaseApi` / `request` 拦截器链（统一
CSRF token 注入、X-Tenant-ID、401 跳登录、错误规范化）。裸 `fetch` /
`axios` 调用会绕过拦截器。

**实现**：扫描 `itsm-frontend/src/` 下匹配 `\bfetch\(` / `\baxios\(` 的
文件，排除：
- `utils/api*` / `BaseApi.ts` / `request.ts` / `apiClient`
- `http-client.ts` / `auth-api.ts` / `service-request-api.ts`（封装层）
- `lib/services/*-service.ts`（流式导出端点，不能用 BaseApi）
- `__tests__` / `mocks` / `fixtures` / `node_modules`

**当前状态**：⚠️ advisory。仓库现存 13 处历史命中，主要分布在
`services/ticket-service.ts`、`services/auth-service.ts` 等。

**修复示例**：

```ts
// 反例：
const response = await fetch('/api/v1/foo', { method: 'GET' })

// 正例：
import { BaseApi } from '@/utils/api'
const data = await BaseApi.get('/foo')
```

---

## 5.4 — context.Background() 蔓延检测

**目的**：防止 `service` 层在 `go func(){...}()` 内裸用
`context.Background()`。正确做法是从入参 `ctx` 派生
`context.WithTimeout(ctx, ...)`，这样上游请求结束时后台 goroutine 也能
被取消（同时保留 RLS tenant_id 上下文）。

**实现**：扫描 `itsm-backend/service/` 下同时包含 `go func` 与
`context.Background()` 的文件。

**当前状态**：⚠️ advisory。仓库现存 16 处历史命中（详见
`scripts/static-gates/run-all.sh` 输出）。集中在 `ticket_service.go`、
`incident_service.go`、`problem_service.go`、`change_service.go`、
`event_bus.go`。

**修复示例**：

```go
// 反例：
go func() {
    ctx2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    ...
}()

// 正例（推荐）：
go func() {
    ctx2, cancel := context.WithTimeout(ctx, 30*time.Second) // 继承入参 ctx
    defer cancel()
    ...
}()
```

---

## 5.5 — ListResponse 分页形状契约

**目的**：所有 `*ListResponse` 必须含
`{items, total, page, pageSize, totalPages}` 五元组，避免前端拿到不一致的
形状（审计文档列出过 5 套分页形状不统一的问题）。

**实现**：
1. 跑 `common.PaginationResponse` 单元测试（`totalPages` 字段存在性）。
2. 静态扫描 `itsm-backend/dto/` 下所有 `*ListResponse` 结构体定义，
   检查字段集合 + 拼写（拒绝 `TotalPage` 这种缺 s 的拼写错误）。

**当前状态**：⚠️ advisory。已修复以下拼写错误：

| 文件 | 修复 |
|------|------|
| `dto/role_dto.go` | `RoleListResponse.TotalPage` → `TotalPages` |
| `dto/user_dto.go` | `PaginationResponse.TotalPage` → `TotalPages` |
| `controller/role_controller.go` | 调用点同步 |
| `controller/group_controller.go` | 调用点同步 |
| `service/user_service.go` | 调用点同步 |

剩余 violation 集中在以下模块（需独立 PR 修复）：
- `dto/asset_license_dto.go` (LicenseListResponse)
- `dto/change_pir_dto.go` (ChangePIRListResponse)
- `dto/cmdb_dto.go` (ConfigurationItemListResponse, CIHistoryListResponse)
- `dto/notification_dto.go` (NotificationListResponse)
- `dto/project_dto.go` (ProjectListResponse)
- `dto/survey_dto.go` (SurveyListResponse)
- `dto/system_config_dto.go` (SystemConfigListResponse)

**修复示例**：

```go
// 反例（缺字段）：
type FooListResponse struct {
    Items []Foo `json:"items"`
    Total int   `json:"total"`
}

// 正例：
type FooListResponse struct {
    Items      []Foo `json:"items"`
    Total      int   `json:"total"`
    Page       int   `json:"page"`
    PageSize   int   `json:"pageSize"`
    TotalPages int   `json:"totalPages"`
}
```

---

## 升级 advisory → hard 的条件

当以下条件同时满足时，对应门禁切换为硬门禁（`exit 1`）：
- `advisory` 模式下连续 10 次 CI 运行未出现新违规；
- 模块 owner 确认剩余 hit 已迁移或被豁免（豁免须写明理由并加 TODO）。

升级时只需移除脚本中的 `exit 0` / 添加 `exit 1` 即可。