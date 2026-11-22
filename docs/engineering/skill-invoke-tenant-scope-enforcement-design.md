# 技能调用入口的租户收口：从 owner 声明到执行强制

> 状态：设计锁定，本批实施。
> 事实依据：worktree `agentcp/2026-11-22-auto`（HEAD `be8afbb00`，基于 `origin/main@bde1814df`）。
> 关键源码：`handlers/skill/registry.go`（`owners` / `GetScoped` / `InvokeWithMetrics`）、
> `handlers/skill/handler.go`（`Invoke` / `enforceRequiredPermissions`）、
> `handlers/skill/contracts.go`、`service/marketplace/skill_delivery.go:317-323`、
> `internal/layering/skill_domain_contract_test.go`。

## 1. 问题（代码事实）

`SkillRegistry` 的租户归属在**发现面**是真实消费的，在**执行面**却是纯声明：

| 环节 | 实现 | 租户维度是否消费 |
|:--|:--|:--|
| 注册 | `RegisterScoped(tenantID, skill)` 写 `owners[code]` | 是 |
| 列举 / 读取 | `ListEntriesScoped` / `GetEntryScoped` / `UnregisterScoped` | 是 |
| 执行 | `InvokeWithMetrics(ctx, code, input)` → `Get(code)` | **否** |

`handlers/skill/handler.go` 的 `Invoke` 把两个缺陷叠在一起：

1. 先做 `entry, entryErr := GetEntryScoped(tenantID, code)`，但只在 `entryErr == nil` 时才跑
   `enforceRequiredPermissions`；查不到时**不写响应、直接继续**。
2. 随后调用无租户视角的 `InvokeWithMetrics(ctx, code, input)`，内部 `Get(code)`
   按全局 code 命中注册表并真的执行。

于是：租户 B 只要知道租户 A 自定义技能的 code，就能执行该技能，且 manifest 声明的
`requiredPermissions` **整段被跳过**。市场交付门禁也拦不住——
`service/marketplace/skill_delivery.go` 对「未由商品目录交付」的技能返回 `Allowed: true`，
租户自建技能通常正是这一类。

这不是理论推演：反向验证里把 handler 恢复成 `entryErr == nil && ...` 后，
spy 注册表收到跨租户请求实际返回 `200 {"output":"executed"}`（见 §5）。

原回归锁 `TestSkillRegistry_ScopedInvokeRejectsForeignSkill` 是假绿：
它把调用包在 `if GetEntryScoped(2, ...) == nil` 里，跨租户分支从来没走到执行入口。

## 2. 设计原则

1. **两道独立防线，各自可验证**：执行入口按租户收口（防线 A）+ handler 的 scoped 查找失败即停（防线 B）。
   只做 B，别的调用方仍能绕过；只做 A，handler 的「查不到还往下走」仍然存在，将来新增执行入口会复发。
2. **签名即约束**：`tenantID` 进入 `Invoke` / `InvokeWithMetrics` / 窄契约 `SkillRegistryCatalog`
   的参数表，让「不带租户地执行」在编译期就不成立，而不是靠调用方记得先查一次。
3. **单一真源**：租户可见性只走 `GetScoped` → `visibleLocked`（内置 owner=0 或本人租户），
   不在 handler 里再写一份 owner 比较；权限判定只走 `middleware.HasResourcePermission`。
4. **身份只解析一次**：`handlerctx.ResolveTenantID` 的结果显式传给归属判定与权限判定，
   不再从 gin context 二次读取 `tenant_id`，避免两道门用上不同的租户。
5. **失败语义一致**：跨租户与真不存在对调用方都是 404（`ErrSkillNotFound`），
   不通过错误码差异暴露某个 code 是否被别的租户注册；底账沿用 `ok=false` + `errorClass=not_found`。
6. **零迁移**：不动 schema、不动权限码、不动已发布迁移 checksum。
7. **不改内置技能语义**：owner=0 的技能对所有租户可见可执行，避免把租户门禁变成「一律拒绝」。

## 3. 契约变更

```go
// handlers/skill/contracts.go
InvokeWithMetrics(ctx context.Context, tenantID int, code string, input interface{}) (dto.SkillInvokeResult, error)

// handlers/skill/registry.go
func (r *SkillRegistry) Invoke(ctx context.Context, tenantID int, code string, input interface{}) (interface{}, error)
func (r *SkillRegistry) InvokeWithMetrics(ctx context.Context, tenantID int, code string, input interface{}) (SkillInvokeResult, error)

// handlers/skill/handler.go
func (h *Handler) enforceRequiredPermissions(c *gin.Context, entry dto.SkillEntry, tenantID int) bool
```

`Invoke` 的执行对象由 `Get(code)` 改为 `GetScoped(tenantID, code)`。
`internal/layering/skill_domain_contract_test.go` 的契约形态断言同步更新，
防止窄契约与实现各写一套。

`handler.Invoke` 的新顺序：解析身份 → scoped 查找（失败即 404 + 底账 + 返回）
→ 市场交付门禁 → `requiredPermissions` 门禁（无条件）→ 带租户执行 → 成功/失败底账。

## 4. 行为影响（明确披露）

- 跨租户调用自定义技能从「200 且执行」变为「404 且不执行」。这是恢复
  `registry.go` 注释已声明的契约（「只有该租户能看到、改、删、调用」），不是新增限制。
- 既有市场交付测试原先让租户 2 调用租户 1 注册的技能来验证「安装记录跨租户不生效」；
  该用例实际依赖的就是本缺陷。现改为把技能按平台内置（owner=0）注册，
  使唯一变量回到「谁装了商品」；另加独立用例锁定技能归属维度。
- 404 分支现在会写一条失败底账（此前由执行入口的 `ErrSkillNotFound` 写），
  保持审计口径不倒退。

## 5. 验收标准与验证

回归矩阵（`handlers/skill`）：

| 用例 | 断言 |
|:--|:--|
| `TestSkillRegistry_ScopedInvokeRejectsForeignSkill` | 跨租户执行入口返 `ErrSkillNotFound` 且 `calls=0`；本租户与内置技能可执行 |
| `TestInvoke_ForeignTenantSkillIsNotExecuted` | HTTP 404、`probe.calls=0`；切回本租户 200 且执行一次 |
| `TestInvoke_ScopedLookupFailureNeverReachesExecution` | spy 注册表：scoped 失败时 `InvokeWithMetrics` 零调用 |
| `TestInvoke_MalformedRequiredPermissionFailsClosed` | 非法权限码 403 且技能不执行 |
| `TestInvoke_BuiltinSkillCallableFromAnyTenant` | owner=0 技能在租户 7 仍可执行 |
| `TestInvoke_CrossTenantInstallationDoesNotGrant`（既有，改夹具） | 内置技能下安装记录仍按租户隔离 |

**反向验证（两轮，均红在断言上）**：

- A：把 `InvokeWithMetrics` 的执行对象恢复成 `Get(code)` →
  `registry_tenant_test.go:120` 红在「跨租户调用必须在执行入口被拒」，证明防线 A 被锁住。
- B：把 handler 恢复成 `false && entryErr != nil` + `entryErr == nil && !enforce...` →
  `invoke_scope_test.go:170` 红在 `expected: 404, actual: 200`，
  响应体含 `"output":"executed"`，证明防线 B 被锁住且原缺陷可被真实利用。

两轮反向改动均已还原，还原后 `go test ./handlers/skill/ -count=1` 全绿。

## 6. 边界与后续（本批未做）

- `InputSchema` 声明/持久化齐全但执行前不校验，属独立的 P2 契约缺口。
- MCP `tools/list` 未按 `Resource/Action` 过滤（AI `GET /agent/tools` 已过滤），
  列表面与执行面口径分叉，属 MCP Gateway 的独立缺口。
- 已批准的写工具在异步执行时未复用 `evaluateToolCall` 做二次判定（策略/权限可能在批准后变更），
  属 Action Gateway 的独立缺口，需单独设计主体推导与回放语义。
- 以上三项均未在本次改动中处理，不作「已覆盖」声明。
