# 审批组引用关系只读查询视图设计（CloudMesh Product Hardening V3.0 第 3 轮）

## 1. 背景与缺口

V3.0 第 1 轮在 `handlers/group` 建了变更期引用守卫：被流程或审批链引用的组，改名/删除返回 409，并在文案里列出引用来源。第 2 轮在 `internal/services/bpmn_process_definition_service.go` 建了发布期守卫：待发布流程引用「组存在但没有合格成员」时返回 409。

两条守卫都在**写路径**上拦，但管理员在按下保存之前没有任何入口回答一个前置问题：**这个组到底被谁引用？**

第 1、2 轮的交付报告都把「组引用关系主动查询视图（`GET /groups/:id/references`）」列为明确遗留项；产品需求第 5 节也要求组管理支持「引用关系查看」。本轮只闭合这一项，不动成员禁用过滤、重置密码权限等级、共享弹窗等并行会话主题。

## 2. 根因与不变式

根因不是「缺一个接口」，而是**读写两侧可能各自实现一套扫描逻辑**：
- 写侧（改名/删除 409）已经有一套精确的 `findGroupReferences`；
- 如果读侧另写一套（例如只查流程、漏掉审批链，或把 CSV 当子串匹配），视图会与守卫漂移，产生比没有视图更糟的假诊断：管理员看到「无引用」→ 点击改名 → 被 409 拒绝，且不知道去哪里改。

因此本轮的唯一核心不变式是：

> **视图所见 = 守卫所拦。** 读侧必须调用写侧同一个 `findGroupReferences`，不允许复制、包装或"优化"扫描口径。

## 3. 设计方案

### 3.1 落点

| 层 | 文件 | 变更 |
|:---|:---|:---|
| DTO | `dto/group_dto.go` | 新增 `GroupReferenceDTO` / `GroupReferencesResponse` |
| Service | `handlers/group/service.go` | 新增 `GetGroupReferences(ctx, id, tenantID)`（追加到已有文件，不新增生产文件） |
| Handler | `handlers/group/handler.go` | 新增 `GetGroupReferences`（追加到已有文件） |
| Router | `router/common_system_routes.go` | 新增 `GET /:id/references`，复用 `group:read` |
| Swagger | `itsm-backend/docs/{docs.go,swagger.json,swagger.yaml}` | `make swagger-gen` 生成 |
| RBAC 预检 | `itsm-backend/middleware/rbac_precheck_gen.go` | `go run ./cmd/authz-gen` 同步新路由条目（+1 行） |

零 schema、零迁移、零新权限码、零新路由前缀。生产代码全部追加到已有文件：不触发 `internal/services` 顶层文件棘轮（125），也不增加 handler 域对共享实现包的依赖（L.3 仍为 0）。

### 3.2 Service 的两条安全约束

1. **租户归属先确认**：按 `group.IDEQ(id) AND group.TenantIDEQ(tenantID)` 查组，查不到即 404。否则猜一个他租户的组 ID 就能读到该组的引用面（跨租户 IDOR）。
2. **失败关闭**：`findGroupReferences` 返回错误（如流程 XML 损坏）时返回 500，而不是空列表。只读接口也不能把「未知」伪装成「可以安全改名」。

扫描本身完全复用 `findGroupReferences`：只扫生效中流程定义的 `candidateGroups`（CSV 整 token 相等）与审批链 `chain[].role="group:<name>"`，严格限定当前租户。

### 3.3 Handler

- `id` 非数字 → 400/1001，不退化成 404；
- 租户只从 `middleware.TenantIDOrUnauthorized` 取，不接受查询参数覆盖；
- 错误统一走 `common.RespondError`，让 `AppError` 的业务码与安全文案透出（不用 `FailWithErr` 吞掉）；
- `items := make([]*dto.GroupReferenceDTO, 0, len(refs))`，保证零引用输出 `[]` 而非 `null`。

## 4. API 契约

```
GET /api/v1/groups/{id}/references
权限：group:read
```

成功：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {"kind":"process_definition","refId":12,"refKey":"change_flow","nodeId":"approve_task"},
      {"kind":"approval_chain","refId":3,"refKey":"变更审批链","nodeId":"level-2"}
    ],
    "total": 2
  }
}
```

| 场景 | HTTP | 应用码 |
|:---|:---|:---|
| 正常（含零引用） | 200 | 0 |
| 组 ID 非数字 | 400 | 1001 |
| 组不存在 / 属于他租户 | 404 | 4004 |
| 缺少租户上下文 | 401 | 2001/2002 |
| 流程 XML 损坏 | 500 | 5001 |

## 5. 权限与安全分析

- **授权**：复用 `group:read`，不新增权限码、不改 `rbac_precheck_gen.go`、不改种子数据。能看到组详情的人就能看到它的引用面，这与"改名按钮本身要 `group:write`"是两个不同门槛，读侧放宽不会带来写侧越权。
- **租户隔离**：两层。handler 只从认证上下文取 tenantID；service 用 `tenantID` 同时过滤组归属和扫描范围。他租户同名引用不会串入本租户视图。
- **不引入写副作用**：只读，不改任何表、不自动建用户/授角色/通过审批。
- **错误信息**：500 只返回固定安全文案，不把底层 XML 解析错误原文透给客户端（原文仅进日志）。

## 6. 验证

`handlers/group` 包 10 个新用例 + 9 个既有守卫用例全绿：

- 两类引用来源同时返回（BPMN + 审批链），字段与节点 ID 精确；
- HTTP 信封字段完整、业务码为 0；
- 零引用返回 `[]` 而非 `null`；
- 跨租户组 ID → 404，跨租户同名引用不串入本租户；
- CSV 整 token 匹配（`approvers-l` 不命中 `approvers-l3`）；
- 非生效流程不计入；
- 坏 XML → 500 且不返回 `items`；
- 无租户上下文 → 401；非数字 ID → 400。

**四路反向验证**（备份 → `assert old in s` 精确替换 → 跑测试 → `cmp` 逐字恢复）：

| 中和对象 | 变红的用例 | 红点 |
|:---|:---|:---|
| service 去掉 `TenantIDEQ` | `跨租户组ID返回404` | 返回 200 而非 404 |
| service 吞掉扫描错误 | `坏XML失败关闭` | 返回 200 空列表 |
| handler 改用 `var items` | `无引用返回空数组而非null` | 输出 `null` |
| service 丢弃扫描结果恒返回空 | 5 个有引用/失败用例 | 引用层断言 |

每一路都在预期断言处变红，恢复后 `cmp` 显示与备份逐字一致，全绿复核通过。这证明测试不是假绿。

**全量测试暴露并闭合的一处遗漏**：首次 `go test ./...` 在 271 个包中仅 `middleware/TestPrecheckMapIsFresh` 一红，报 `rbac_precheck_gen.go` 与路由声明不同步。这是本轮**真实引入**的红（新增路由改变了 `GeneratePrecheckSource()` 的期望输出），不归为既有红。按测试指引执行 `go run ./cmd/authz-gen` 后生成物仅增 1 行（`/api/v1/groups/*/references` → `group:read`），前端 `catalog.gen.ts` 因权限码未新增而零变化；定向复测 `TestPrecheckMapIsFresh` / `TestNoPermissionDeclarationIsDropped` / `TestRoutePrecheckAlignment` 全绿，随后全量重跑。

## 7. 影响范围与兼容性

- 新增一个只读端点，不改动任何既有端点的请求/响应形状。
- `GroupReferencesResponse` 是新 DTO，不影响 `GroupResponse` / `GroupDetailResponse`。
- 前端当前未接线（本轮只交付 API 与文档）；既有组管理页行为完全不变。
- Swagger 生成物更新，CI `Swagger Docs Freshness` 应保持绿。

## 8. 回滚

按提交粒度整体 revert 即可：删除 `GET /:id/references` 路由、`GetGroupReferences` handler/service、两个 DTO、测试文件，重跑 `make swagger-gen`。无数据迁移、无基线数据、无权限码变更，回滚不触碰任何存量数据。

## 9. 遗留

- 前端组管理页尚未接这个接口（下一轮可加"查看引用"抽屉，属于 UI 主题，不与本轮后端混做）。
- 存量已发布流程的"当前无合格审批人"只读诊断看板未做（需扫描所有生效流程的候选组并逐组解析成员，属另一独立任务）。
- 审批链 `group:<name>` 的发布期诊断未做（审批链无独立发布入口；变更期阻断仍有效）。
