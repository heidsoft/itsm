# Static Analysis Gates

> Status: current

本文档定义了 Stage 5 的静态门禁。每条门禁对应一个
shell 脚本，位于 `scripts/static-gates/`（5.11 例外，位于 `itsm-frontend/tools/`）；
`run-all.sh` 只聚合后端契约类门禁
（5.1–5.5 与 5.10），前端门禁 5.6–5.9 与 5.11 需单独调用对应脚本，见「接入位置」。

| # | 规则 | 脚本 | 状态 | 阻断构建 |
|---|------|------|------|---------|
| 5.1 | 禁止 `c.JSON(...)` 绕过 `common.Success/Fail` | `check-bare-json.sh` | **HARD** | ❌（见下） |
| 5.2 | `common.Fail` 必须把 2002/2004/2005 映射到 401/403/404 | `check-http-status-mapping.sh` | **HARD** | ✅ |
| 5.3 | 前端禁用 raw `fetch` / `axios`，统一走 BaseApi | `check-raw-fetch.sh` | ADVISORY | ❌ |
| 5.4 | `service` 层 `go func` 内不得裸用 `context.Background()` | `check-context-bg.sh` | ADVISORY | ❌ |
| 5.5 | 列表信封只用 `items` + 标准分页键（分页列表五元组 / 不分页列表 `{items,total}`） | `check-pagination-shape.sh` → 委托 `tests/contract` 棘轮 | **HARD** | ✅（棘轮随 `go test ./...` 在 backend-ci 失败） |
| 5.6 | `next.config.ts` 不得启用 `ignoreBuildErrors` / `ignoreDuringBuilds` | `check-next-ignore-build-errors.sh` | ADVISORY | ❌ |
| 5.7 | 主要路由组必须具备 `loading.tsx` / `error.tsx` / `not-found.tsx` | `check-next-route-states.sh` | ADVISORY | ❌ |
| 5.8 | `ErrorBoundary` / `AccessDenied` 不得跳转 `/` 营销路径 | `check-error-boundary-target.sh` | ADVISORY | ❌ |
| 5.9 | 测试夹具不得硬编码共享唯一键（如 `ticket_categories.code`） | `check-test-fixture-uniqueness.sh` | ADVISORY | ❌ |
| 5.10 | `handlers/**` 分页 `gin.H` 响应体的集合键必须是 `items` | `check-list-envelope.sh` | **HARD** | ✅（backend-ci lint job） |
| 5.11 | Ant Design v4/v5 弃用 API 零新增（9 类，组件归属判定 + 基线棘轮） | `itsm-frontend/tools/check-antd-legacy.sh` | HARD（本地棘轮） | ❌（未接 CI） |

> 5.6–5.9 迁移自 [`docs/review/frontend-ux-review-2026-06-19.md`](../review/frontend-ux-review-2026-06-19.md) 与 [`docs/review/system-function-review-result-2026-07-01.md`](../review/system-function-review-result-2026-07-01.md)；脚本位于 `scripts/static-gates/`（与 5.1–5.5 并列）。

## 接入位置

### 本地开发

```bash
./scripts/static-gates/run-all.sh
```

### CI

2026-10-02 实测（`grep -rn "static-gates" .github/workflows/`）：**只有 5.10 真正在 CI 里跑**，
接在 `.github/workflows/backend-ci.yml` 的 lint job：

```yaml
- name: List envelope gate (handlers must use items)
  run: |
    bash scripts/static-gates/check-list-envelope.sh
```

`run-all.sh` 目前**没有任何 workflow 调用**，因此 5.1–5.4 只在本地执行（5.5 的判定逻辑
本身在 `tests/contract` 里，随 backend-ci 的 `go test` 硬失败，脚本只是本地入口）。
这直接掩盖了一条
已存在的 HARD 门禁违规：`check-bare-json.sh`（5.1）实测有 6 处命中（`handlers/dingtalk/handler.go`
3 处、`handlers/wecom/handler.go` 2 处、`handlers/approval/routes.go` 1 处），本地 `run-all.sh`
以退出码 1 结束，CI 却是绿的。把 `run-all.sh` 接进 CI 之前，需要先处置这 6 处（钉钉/企业微信
是外部回调协议，响应形状由对方规定，属于 5.1 的合理豁免面，应显式登记而不是继续裸用）。

> 5.3 / 5.4 为 advisory（exit 0），日志中可见违规命中；当历史命中全部迁移完成后会
> 切换为硬门禁（exit 1）。5.5 已于 2026-10-03 改为委托棘轮并升为 HARD（见该节）。

---

## 5.1 — 禁止裸 c.JSON()

**目的**：所有 HTTP 响应必须经过 `common.Success / common.Fail /
common.SuccessWithList`，确保 `{code, message, data}` 三元组与 HTTP 状态码
映射契约不会被绕过。

**实现**：扫描 `itsm-backend/handlers`、`itsm-backend/service`、
`itsm-backend/controller` 下的 `.go` 文件（排除 `_test.go` / `_mock.go`），
匹配 `c.JSON(<digit>, …)`。

**当前状态**：❌ 失败（2026-10-02 实测 6 处命中，见「接入位置 / CI」）。本门禁标记为
HARD，但因 `run-all.sh` 未接入 CI，实际不阻断任何合并。

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

**当前状态**：⚠️ advisory。仓库现存 7 处历史命中（2026-10-03 实测，详见
`scripts/static-gates/run-all.sh` 输出）。集中在 `service/ticket_service.go`
（4）、`service/incident_service.go`（2）、`service/common/event/event_bus.go`
（1）。原先并列的 `problem_service.go` 与 `change_service.go` 已不在名单里：
前者是零生产构造的死第二实现，已整文件删除（台账 E4-13）。

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

## 5.5 — 列表信封与分页形状契约

**目的**：列表响应只允许两种形状，且分页字段不得双轨。

- **分页列表**：`data: {items, total, page, pageSize, totalPages}`（由 `common.SuccessWithList` 产出）
- **不分页列表**：`data: {items, total}`——`total` 必须是全量计数，契约见
  [`docs/api-reference.md`](../api-reference.md)「不分页的列表」

两种形状都禁止把集合挂在领域名键下（`tickets`/`changes`/`releases`/`cloudAccounts` …），
也禁止 `size`/`limit`/`offset`/`totalCount`/`totalPage` 这类分页别名。旧版本要求
「所有 `*ListResponse` 必须含五元组」，与上面第二种形状直接矛盾，因此照该规则升硬
只会永远无法满足；2026-10-03 已按实测改掉这个判定。

**实现（2026-10-03 起为委托）**：`check-pagination-shape.sh` 不再自带扫描器，只做两件事——

1. 跑 `common` 的分页序列化单元测试（`totalPages` 由 `NewListResponse` 真实算出）；
2. 跑 `itsm-backend/tests/contract/list_envelope_ratchet_test.go` 的三条棘轮
   （`TestListEnvelopeRatchet` 领域名集合键、`TestListEnvelopeKeys` 分页键不完整、
   `TestListEnvelopePagingAliases` 分页别名残留），任一失败即 **exit 1**。

为什么交给棘轮而不是脚本自己扫：棘轮用 AST 扫 `dto/*.go`，判定条件是「名称含 List 且带
total」，实测是旧文本扫描器的**超集**——当前 28 个 `*ListResponse` 全部命中且全部含
`json:"total"`（本批删除 8 个死 DTO 前是 36 个），棘轮另外还覆盖 11 个不以 `ListResponse`
结尾的信封（`ListTicketsResponse`/`ListCIsResponse`/`ListProblemsResponse`/
`ListAuditLogsResponse` 等，旧扫描器对它们完全失明）。同一件债务写在两处正是本批次在消除的问题。

**当前状态**：**HARD**（棘轮随 `go test ./...` 在 backend-ci 硬失败；脚本本身只在本地
`run-all.sh` 里跑，而 `run-all.sh` 未接任何 workflow，见「接入位置」）。存量债务以基线
形式登记在测试文件内，2026-10-03（E4-6f 收敛服务目录信封后）实测 **20 条 / 20 个结构体**
（20 领域名集合键 + 0 分页键不完整 + 0 分页别名；`envelopeKeyBaseline` 与 `envelopeAliasBaseline`
均为空集，因此「带分页键但不全」和「用 `size` 顶替 `pageSize`」两类违规已从存量债务变成硬失败），
只减不增：新增违规失败，
**基线过期（收敛后忘记删条目）同样失败**。
轨迹：33 条 / 27 个结构体 →（E4-6e 把 CMDB 五个信封补成五元组、删掉零引用的 `ListCIsResponse`、
`envelopeKeyBaseline` 清零）22 条 / 21 个结构体 →（E4-6f 把 `ServiceCatalogListResponse` 的
`catalogs`→`items`、`size`→`pageSize`+`totalPages`，并删掉 `dto.GetServiceCatalogsRequest`
的 `page`/`size`——第四条页长真相）20 条 / 20 个结构体。

分页键那条只判**部分分页**：信封里出现了 `page`/`pageSize`/`totalPages` 任一键就必须凑满
五元组，纯 `{items,total}` 属上面第二种合法形状、不登记为违规——此前该规则与文档矛盾，
8 条里有 2 条（工单视图、工单评论）实测是 service 层无 `Limit` 的诚实不分页，被当成债务
登记；另 1 条（变更 PIR）确实真分页却只回 `{items,total}`，按补全五元组收口而不是降级。
改判后剩 5 条（全在 CMDB 一侧：`page` + `size` 且无 `totalPages`），2026-10-03 E4-6e 已把这 5 条
补成五元组并清零 `envelopeKeyBaseline`，因此 CMDB 侧再引入「带分页键但不全」的信封会直接失败。
改判不是放宽：实测注入「只有 `page`」的探针仍 FAIL。

**接入位置的真实差别**（避免误读成「脚本变硬 = CI 变硬」）：

| 层次 | 谁在跑 | 是否阻断 CI |
|------|--------|------------|
| 结构体层（`dto/`） | `tests/contract` 棘轮 ← backend-ci 的 `go test $TESTABLE_PKGS` | ✅ |
| handler 层（就地拼的 `gin.H`） | 5.10 `check-list-envelope.sh` ← backend-ci lint job | ✅ |
| 本地聚合 | `scripts/static-gates/run-all.sh`（无 workflow 调用） | ❌ |

**收敛一条的做法**：把 json tag 改成 `items`（并按端点是否真分页补齐
`page/pageSize/totalPages`，或确认属「不分页的列表」），同步 Mapper、真实路由契约测试、
前端 `src/lib/api` 类型与调用点，然后从对应基线删除该条。

**修复示例**：

```go
// 反例：集合挂在领域名键，且用 size 代替 pageSize
type FooListResponse struct {
    Foos  []Foo `json:"foos"`
    Total int   `json:"total"`
    Size  int   `json:"size"`
}

// 正例（分页列表）：
type FooListResponse struct {
    Items      []Foo `json:"items"`
    Total      int   `json:"total"`
    Page       int   `json:"page"`
    PageSize   int   `json:"pageSize"`
    TotalPages int   `json:"totalPages"`
}

// 正例（端点确实不分页）：
type FooListResponse struct {
    Items []Foo `json:"items"`
    Total int   `json:"total"` // 必须是全量计数
}
```

历史拼写修复（`TotalPage` → `TotalPages`）已完成于 `dto/role_dto.go`、`dto/user_dto.go`
及 `controller/role_controller.go`、`controller/group_controller.go`、`service/user_service.go`
调用点；`totalPage` 现已作为分页别名之一登记进棘轮的 `forbiddenPagingKeys`，不需要再靠
脚本里的文本 grep 兜住。

---

## 5.6 — `next.config.ts` 不得启用 ignoreBuildErrors

**目的**：任何 `typescript.ignoreBuildErrors` / `eslint.ignoreDuringBuilds` 都会让类型错误 / lint 问题进入生产构建。评审 P0-4 明确指出该选项需删除。

**实现**：扫描 `itsm-frontend/next.config.ts` 与 `next.config.js`，匹配：

- `ignoreBuildErrors\s*:\s*true`
- `ignoreDuringBuilds\s*:\s*true`

**当前状态**：⚠️ advisory。仓库现存 1 处历史命中，移除后切换硬门禁。

**修复示例**：

```ts
// 反例：
const nextConfig = {
  typescript: { ignoreBuildErrors: true },
  eslint: { ignoreDuringBuilds: true },
};

// 正例：删除两个 ignore 配置；CI 强制 `tsc --noEmit` 与 `next lint`。
const nextConfig = {
  reactStrictMode: true,
};
```

---

## 5.7 — 路由必备状态文件

**目的**：评审 P0-2 / P0-3 / P1-2 指出 Next.js 路由缺少 `loading.tsx` / `error.tsx` / `not-found.tsx` / `global-error.tsx` 会导致整页白屏或默认 404。

**实现**：扫描 `itsm-frontend/src/app/` 下每个路由目录（含 `(main)/`、`(auth)/`）：

- 必须存在 `error.tsx`（路由级错误边界）
- 公共路由（`/`、`(main)`）必须存在 `loading.tsx`、`not-found.tsx`
- 根 `app/` 必须存在 `global-error.tsx`

**豁免**：动态路由目录、API 路由（`api/`）。

**当前状态**：⚠️ advisory。仓库当前已补齐 `(main)` 路由的 `loading.tsx` / `error.tsx`，待补 `not-found.tsx` 与根 `global-error.tsx`。

---

## 5.8 — 错误边界跳转目标

**目的**：评审 P1-3 / P2-10 指出 `ErrorBoundary.handleGoHome` 与 `AccessDenied` "返回首页"在无历史记录时跳 `/`（营销页 / 重定向到登录），导致用户被登出。

**实现**：扫描 `itsm-frontend/src/components/common/ErrorBoundary.tsx`、`AuthGuard.tsx`：

- 不得出现 `router.push('/')` 或 `window.location.href = '/'`
- 必须跳 `/dashboard` 或基于认证状态动态决定

**当前状态**：⚠️ advisory。评审已识别 2 处需修复。

---

## 5.9 — 测试夹具共享唯一键

**目的**：评审 F-6..F-9 指出 controller 测试硬编码 `ticket_categories.code = "incident"` 会导致唯一约束冲突，9 个 ticket 测试因此失败。

**实现**：扫描 `itsm-backend/controller/*_test.go`、`service/*_test.go`，匹配：

- `SetCode\(["']incident["']\)`（不带 uniqueTestID）
- `SetName\(["']incident["']\)` 同模式
- 任何 `SetCode(["']` + 字面量 + `["'])` 后跟 `.Save(ctx)` 且未含 `unique` / `+.*ID` / `fmt.Sprintf`

豁免：测试夹具必须含 `uniqueTestID()`、`uuid`、`fmt.Sprintf` 或 `time.Now()` 等唯一化逻辑。

**当前状态**：⚠️ advisory。F-6..F-9 已在 2026-08-12 修复，但守门规则缺失；本门禁防止未来再次引入同类硬编码。

---

## 5.10 — handler 列表信封的集合键必须是 items

**目的**：5.5 只看 `dto/*ListResponse` 结构体，管不到 handler 里就地拼的
`gin.H{...}`。E2-3 盘点时 `handlers/` 下有一批响应把集合放在领域键
（`tickets`/`reports`/`instances`/`logs`/`templates`/`notifications`/`categories`…）
下，前端只能按每种别名取值；更糟的是几个**不分页**的接口为了凑形状伪造
`page: 1, pageSize: len(list)`，调用方据此以为存在一套不存在的分页协议。
本门禁把「分页字段与 `items` 键必须同生同灭」变成硬约束。

**实现**：解析 `itsm-backend/handlers/**/*.go`（排除 `_test.go`）里每个
`gin.H{...}` 字面量块，提取其中的键名后判定两条规则：

1. 块内出现 `page` / `pageSize` / `totalPages` 但没有 `items` → 违规；
2. 块内出现领域列表键黑名单（`tickets incidents problems changes releases reports
   templates notifications instances logs records allocations customers users tasks
   comments`）且同时带分页字段 → 违规。

**当前状态**：✅ 通过，且已在 CI（backend-ci lint job）执行。新增于 2026-10-02，
引入当轮即命中 `handlers/known_error` 的统计接口伪造分页与 `categories` 别名，
顺带挖出 Ent 统计查询 AND 串联导致 6 个计数器恒为 0 的真实缺陷。

**修复示例**：

```go
// 反例 1：分页信封缺 items
common.Success(c, gin.H{"tickets": tickets, "total": total, "page": page, "pageSize": pageSize})

// 反例 2：不分页的列表伪造分页
common.Success(c, gin.H{"templates": normalized, "total": len(normalized), "page": 1, "pageSize": len(normalized)})

// 正例 1：分页列表走统一构造
common.SuccessWithList(c, tickets, total, page, pageSize)

// 正例 2：全量列表只带诚实的子集
common.Success(c, gin.H{"items": normalized, "total": len(normalized)})
```

---

## 5.11 — Ant Design 弃用 API 零新增（lint:antd）

**目的**：项目使用 antd v6，v4/v5 遗留 API 禁止新增（AGENTS.md「Ant Design v6 旧 API
零新增规则」）。原脚本只覆盖 3 类全局文本模式，`visible=`/`destroyOnClose`/`bodyStyle`/
`overlay`/`dropdownRender`/`onDropdownVisibleChange` 六类完全无门禁。2026-10-03 补齐。

**接入**：`cd itsm-frontend && npm run lint:antd`（本地门禁；frontend-ci 只跑 eslint，
未接 CI）。

**两层检测**：

1. **全局模式**（历史行为不变，按行匹配）：`Space direction=`、`<Tabs.TabPane`、
   `Form.(Input|TextArea|Select|DatePicker|Radio|Checkbox)` 复合组件。
2. **组件归属模式**：六类 prop 仅当 JSX 标签是**本文件从 `'antd'` 具名导入**（含
   `as` 别名与 `import type`）的组件时才判违规。项目自有组件可以合法拥有同名
   prop（如自研 `Drawer` 的 `visible`），归属由 import 决定，不做全局文本误伤。

   判定算法：提取 `<Alias` 到下一个 `>` 的开标签区域 → 迭代剥离平衡 `{...}` 与
   引号字符串 → 对剩余裸属性名匹配。因此布尔简写（`<Drawer destroyOnClose`）与
   值形式（`visible={x}`）都能命中，而表达式内部的同名词（`title={visible ? ...}`、
   `a.visible ===`）不会误报。

**弃用面**（已逐一对照 `node_modules/antd` 6.2.2 的 `.d.ts` @deprecated 注释核实）：

| 类别 | 触发组件 | 替换 |
|------|---------|------|
| `visible` | Modal / Drawer（v6 已移除） | `open` |
| `destroyOnClose` | Modal / Drawer | `destroyOnHidden` |
| `bodyStyle` | Card / Modal / Drawer | `styles={{ body }}` |
| `overlay` | Dropdown（v6 已移除） | `menu` |
| `dropdownRender` | Select / TreeSelect / AutoComplete / Cascader / Dropdown | `popupRender` |
| `onDropdownVisibleChange` | Select / TreeSelect / AutoComplete / Cascader | `onOpenChange` |

**基线棘轮**：各类别命中数与 `itsm-frontend/tools/antd-legacy-baseline.txt` 比较——
超过基线 → exit 1；低于基线 → 提示收紧（只许降不许升）；新类别命中且未登记 →
exit 1（缺类别按 0 处理）。`npm run lint:antd -- --list` 打印全部 file:line 明细，
用于收紧基线前核对。

**当前状态**：✅ 通过。基线登记于 2026-10-03：仅 `bodyStyle 5`
（`src/components/business/AISuggestionPanel.tsx` ×4 + `src/components/common/LazyLoadWrapper.tsx`
×1，均为 Card 的旧式 `bodyStyle`，待迁移为 `styles={{ body }}` 后同步下调基线），
其余 8 类为 0。检测逻辑经正反探针验证：六类各造一例违规全部命中（含布尔简写），
项目自有同名组件零误报。

---

## 5.12 — 文件级孤儿检测（C.7.4）

**目的**：发现 `src/components/**` 与 `src/lib/api/**` 中不被任何 `src/app/**` 页面到达的文件。

**脚本**：`scripts/docs-gate/check-scope-creep.sh` 的 C.7.4 段，委托 `scripts/docs-gate/find-frontend-orphans.py` 执行 BFS 可达性分析。

**算法**：
1. 种子：`src/app/**` 下所有 `.tsx/.ts` 文件 + 根 `layout.tsx` + `middleware.ts`
2. 从种子出发，解析静态 `import`/`export from`、动态 `import()`、`require()` 三种形式
3. 解析 `@/` 别名到 `src/`，相对路径按当前文件位置展开，尝试 `.ts`/`.tsx`/`index.ts`/`index.tsx` 四种后缀
4. BFS 遍历构建可达文件集合
5. 扫描 `src/components/**` 和 `src/lib/api/**`，报告不在可达集合中的文件

**当前状态**：advisory（仅 WARN，不阻断）。原因：`check-scope-creep.sh:22-33` 记录的教训——静态检测假阳性会逼人删掉正确的东西。需要 R5 人工复核清理后再转 blocking。

**用法**：

```bash
# 仅报告（advisory）
./scripts/docs-gate/check-scope-creep.sh --advisory

# 单独运行孤儿检测
python3 scripts/docs-gate/find-frontend-orphans.py itsm-frontend
```

**已知限制**：
- 不识别 `React.lazy(() => import(...))` 以外的间接动态引用
- barrel 文件（`index.ts`）的 re-export 链会被跟踪，但如果 barrel 自身不可达，其下所有文件都报为孤儿
- 测试文件（`__tests__/`、`*.test.tsx`）不作为种子，因此只被测试引用的生产文件会报为孤儿（这通常是正确的——如果生产代码只被测试引用，说明生产侧确实没用到）

---

## 升级 advisory → hard 的条件

当以下条件同时满足时，对应门禁切换为硬门禁（`exit 1`）：
- `advisory` 模式下连续 10 次 CI 运行未出现新违规；
- 模块 owner 确认剩余 hit 已迁移或被豁免（豁免须写明理由并加 TODO）。

升级时只需移除脚本中的 `exit 0` / 添加 `exit 1` 即可。