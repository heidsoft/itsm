# 前端业务 Bug 审计报告

- **日期**: 2026-07-12
- **审计范围**: `itsm-frontend`（Next.js 15 + antd 6 + TS）核心业务模块
- **方法**: 每个疑似 bug 均对照 `itsm-backend` 的真实路由（`router/router.go`）、响应 DTO（`dto/*.go`）、RBAC 权限表（`middleware/rbac.go`、`service/auth_service.go`）逐一核实，非主观推断。
- **结论**: 共确认 **10 个前端业务 bug**，其中 P0×2、P1×4、P2×4。最严重的是审批中心「空列表 + 批量审批伪造成功」，以及权限模型与后端契约全面错配。

---

## P0 — 核心业务流完全失效

### 1. 审批中心列表恒为空（读错响应字段）
- **文件**: `src/app/(main)/approvals/page.tsx:182-228`
- **根因**: `load()` 用 `ticketsResp.items` / `changesResp.items` / `srResp.items` 取数；但后端 `ListTicketsResponse` 的 JSON 标签是 `tickets`、`ChangeListResponse` 是 `changes`（已核实 `dto/ticket_dto.go:105` / `dto/change_dto.go:118`）。
- **影响**: `items` 永远 `undefined`，`|| []` 兜底后三个 Tab 始终显示「暂无待审批」，即使存在待审数据。审批中心形同虚设。
- **修复**: 改为 `ticketsResp.tickets`、`changesResp.changes`，并确认 service-request 列表的真实键名后对应调整。

### 2. 批量审批「伪造成功」——不调用任何后端
- **文件**: `src/app/(main)/approvals/page.tsx:106-135`（`handleBatchApprove` / `handleBatchReject`）
- **根因**: 两个函数仅 `Modal.confirm` 后直接 `message.success('已批准/已拒绝 N 项')` + 清空选择 + `load()`，**完全没有任何网络请求**。
- **影响**: 用户以为批量审批已完成，实际什么都没发生。属于数据完整性的严重欺骗性 bug，生产环境会让人误判审批进度。

---

## P1 — 关键功能 404 / 权限错配

### 3. 单项快速审批全部指向不存在的端点（且发空 body）
- **文件**: `src/app/(main)/approvals/page.tsx:142-175`（`handleQuickApprove` / `handleQuickReject`）
- **根因**: 
  - `ticket` → `/api/v1/tickets/${id}/approve`：**后端无此路由**（实际工单审批为 `POST /api/v1/tickets/workflow/approve`，body 含 `action/comment/ticketId`）。
  - `change` → `/api/v1/changes/${id}/reject`：**后端无此路由**（仅有 `/api/v1/changes/:id/approve`）。
  - `serviceRequest` → `/api/v1/service-requests/${id}/approve`：**后端无此路由**（实际 `POST /api/v1/service-requests/:id/approval`）。
  - `incident` → `/api/v1/incidents/${id}/approve|reject`：**后端无此路由**。
  - 且所有请求均发 `{}` 空 body，缺少 `comment`/`status`/`action` 等必填字段。
- **影响**: 点「批准/拒绝」必 404，快速审批完全不可用。

### 4. `ticket-approval-api.ts` 全部缺 `/v1/` 前缀 + 路径错
- **文件**: `src/lib/api/ticket-approval-api.ts:81,86,94,99,123,134`
- **根因**: 使用 `/api/tickets/approval/workflows` 等；后端审批工作流挂在 `/api/v1/.../approval-workflows`（路径是 `approval-workflows`），审批提交为 `/api/v1/tickets/approval/submit`，记录为 `/api/v1/tickets/approval/records`（均已核实 `router.go:560-572`）。
- **影响**: 审批工作流 CRUD、列表、提交全部 404；依赖它的 `TicketMultiLevelApproval.tsx` 等功能完全不可用。

### 5. `ticket-root-cause-api.ts` 缺 `/v1/` + 资源类型错
- **文件**: `src/lib/api/ticket-root-cause-api.ts:52,60,66,71`
- **根因**: 使用 `/api/tickets/${id}/root-cause/analyze|report`；后端根因分析仅存在于**事件**资源：`POST /api/v1/incidents/root-cause`、`POST /api/v1/incidents/:id/root-cause`（无 `analyze` 子路径，已核实 `router.go:792-795`）。
- **影响**: 工单根因分析功能请求必然 404。

### 6. 权限 System B 忽略后端真实权限 + 角色表缺键
- **文件**: `src/lib/hooks/use-permissions.ts:8-198`（被 `RouteGuard.tsx` / `PermissionGuard.tsx` 用作路由闸门）
- **根因**:
  - `getRolePermissions(role)` 仅查**硬编码前端角色表**，从不读取后端返回的真实 `user.permissions`（后端契约已核实：`auth_service.go:177-179` 把 `resource:action` 拼成 `ticket:read`/`ticket:write`/`ticket:delete`/`ticket:admin` 等返回）。
  - `rolePermissionMap` 仅有 `superAdmin/admin/manager/agent/user` 五键；`ROLES` 常量里定义的 `technician`、`end_user`（及后端 `sysadmin`）**没有对应键**。
  - 前端动词用 `create/update/assign/close/...`，与后端 `read/write/delete/admin` 不一致。
- **影响**:
  - 任何 `role === 'technician'|'end_user'|'sysadmin'` 的用户 → `getRolePermissions` 返回 `[]` → `canAccessRoute` 恒 false → **被路由守卫锁死，无法进入任何业务页面**。
  - 其他角色权限与后端实际授权脱节（前端增删的细粒度权限后端不认可，后端返回的覆盖权限前端也不生效），造成误放行或误拦截。
- **对比**: `auth-store.ts` 另有一套 `usePermissions`（`canViewTickets` 等），用 `ticket:view`/`ticket:create` 风格去 `user.permissions.includes()`——与后端 `ticket:read`/`write`/`delete`/`admin` 同样**全部对不上**（仅 `ticket:delete` 巧合匹配）。两套权限模型互相矛盾，且各自都与后端契约不符。

---

## P2 — 数据一致性 / 多租户 / 健壮性

### 7. 多租户 UI 切换失效（租户上下文从未写入）
- **文件**: `src/lib/store/auth-store.ts:10,71-72,90-91,112-113,124-125` + `src/lib/auth/tenant-context.ts` + `src/lib/api/http-client.ts:105-113,137-162`
- **根因**: `auth-store` 的 `login`/`logout`/`setCurrentTenant` 调的是 `httpClient.setTenantId/Code`，而这两个方法在 `http-client.ts:105-113` 是 **deprecated 空操作**（仅打 debug 日志），从不调 `tenant-context.setTenant`。真正发请求头时 `getHeaders()`（http-client.ts:147-159）读的是 `tenant-context` 的模块级 state，它永远是 `null`。
- **影响**: `X-Tenant-ID` / `X-Tenant-Code` 头永不发送。后端以 JWT `tenant_id` 为准（单租户用户仍正常），但**通过 UI 切换租户时不会生效**，多租户用户看到/操作的是原租户数据。

### 8. 变更列表「风险等级」筛选被静默丢弃
- **文件**: `src/components/change/ChangeList.tsx:79-88,90-98`
- **根因**: 父页面 `changes/page.tsx` 把 `risk` 传给 `ChangeList`，但 effect 里只 `form.setFieldsValue({ search, status, type: undefined })`（注释「risk 先不处理」），`loadData` 也从不把 `risk` 放入查询参数。
- **影响**: 用户选「高风险」筛选无效果，返回未过滤的全量数据。

### 9. `http-client` 容错把错误当成功
- **文件**: `src/lib/api/http-client.ts:344-352`（及 `304-308` 重试分支）
- **根因**: 仅当 `responseData.code` 存在且 ≠ 0 才抛错；`code` 缺失时直接 `return toCamelCase(responseData.data)`——可能返回 `undefined`/`null` 而不报错。
- **影响**: 后端返回错误响应但未带 `code`（或异常返回 `{error:...}`）时，失败被静默吞掉，上层拿到 `undefined` 继续渲染，用户无错误提示。

### 10. 变更看板编号重复判断且字段不存在
- **文件**: `src/app/(main)/changes/page.tsx:273-280`
- **根因**: `getItemNumber` 为 `(data.changeNumber) || (data.changeNumber) || 'C-${id}'`——两个 `||` 操作数完全相同；且本页 `Change` 类型（`change-api.ts`）**没有 `changeNumber` 字段**（只有 `id/title/...`）。
- **影响**: 看板卡片编号永远渲染为 `C-{id}`，后端真实的变更单号（`change_number`）丢失不显示。

---

## 建议修复优先级

| 优先级 | Bug | 修复投入 |
|---|---|---|
| P0 | #1 审批列表空（字段读错） | 小（改 3 个字段名） |
| P0 | #2 批量审批伪造成功 | 中（需逐条调审批 API） |
| P1 | #3 快速审批端点 404 | 中（对齐后端审批路由 + body） |
| P1 | #4 #5 审批/根因 API 缺 /v1/ | 小（补前缀 + 路径） |
| P1 | #6 权限模型与后端错配 | 大（统一为读 `user.permissions` 解析 `resource:action`） |
| P2 | #7 多租户切换失效 | 小（auth-store 改调 `tenant-context.setTenant`） |
| P2 | #8 风险筛选丢弃 | 小（把 risk 并入查询） |
| P2 | #9 错误被静默吞掉 | 小（code 缺失也视为失败） |
| P2 | #10 看板编号重复 | 小（用真实单号字段） |

> 注：本次为「查找 + 确认」阶段，未改动源码。后端变更审批链状态卡死的修复（`service/change_approval_service.go`）已在上一轮完成，本前端 #1/#3 的发现与之形成「端到端审批链路」的配套问题。
