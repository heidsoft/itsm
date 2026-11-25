# 权限体系设计与实现审计（2026-11-25）

> 审计方式：codex（`gpt-6.1-sol`）只读全链审计 + 独立源码反向复核。
> 基线 HEAD：`faafe67f6 fix(bpmn): 会签恢复闭环三个崩溃/串行窗口（codex 审计 P2-01 收口）`。
> 工作树存在 52 个并行会话 WIP；本报告的证据文件均不在 WIP 清单内，未提交任何改动。

## 一句话结论

权限体系的**码空间单一源、三主体判定链与 AI 零信任 Gate 2.5 是自洽且实现扎实的**，但「路由权限」与「数据行级范围」之间存在两处可证实的租户内越权读取缺口，另有跨副本缓存失效、存量迁移授权遗漏与一处启动自检承诺未兑现；没有发现跨租户提权或 JWT 主体伪造路径。

## 发现总表

| 级别 | 标题 | 关键证据 | 复核结论 | 状态 |
|:--|:--|:--|:--|:--|
| P1 | AI 工具读面绕过行级范围 | `tool_registry.go:570`、`configuration_item_service.go:380` | 成立 | 新发现 |
| P1 | 审批后执行信任旧角色快照 | `tool_executor.go:118`、`ai/service.go:222` | 成立 | 新发现 |
| P1 | 权限停用后跨副本角色缓存未失效 | `role_service_impl.go:756`、`rbac.go:906` | 成立 | 已知遗留 AUDIT-022 的镜像残留 |
| P2 | service 主体审批后复核被还原为 agent | `ai/service.go:222`、`:256` | 成立（错误拒绝，非提权） | 新发现 |
| P2 | MCP 存量迁移遗漏 sysadmin 授权 | `20261118_mcp_read_permission.sql:103` | 成立 | 新发现 |
| P2 | `ValidateBootstrap` 启动自检全仓无调用点 | `registry.go:255`、`bootstrap/app.go:1095` | 成立 | 本次独立复核补充 |

---

## P1-1 AI 工具读面绕过行级范围

**现象。** 普通业务列表按当前用户与角色计算 DataScope：

```go
// handlers/ticket/service.go:168
func (s *Service) List(ctx context.Context, tenantID int, page, size int,
    filters map[string]interface{}, currentUserID int, currentRole string) ([]*Ticket, int, error) {
    return s.repo.List(ctx, tenantID, page, size, filters, dataScopeForRole(currentRole), currentUserID)
}
```

但 AI `list_tickets` 工具只限定租户和软删除：

```go
// internal/services/tool_registry.go:570
tickets, err := t.client.Ticket.Query().
    Where(ticket.TenantID(tenantID), ticket.DeletedAtIsNil()).
    Order(ent.Desc(ticket.FieldCreatedAt)).Offset(...).Limit(...).All(ctx)
```

`list_cis` 同理（`tool_registry.go:603`），而 CI 服务明确只在 HTTP handler 注入 scope 时才过滤：

```go
// internal/services/configuration_item_service.go:380-389
// 行级数据权限：只有 HTTP handler 显式注入 ScopeConfigured 才应用过滤。
// AI 工具/导出等内部调用方走同一 service 但不带 scope，保持租户级旧语义。
if req.ScopeConfigured { ... query = datascope.ApplyCIFilter(...) }
```

**为什么是问题。** Gate 2 只证明调用者持有 `ticket:read` / `cmdb:read`，不代表可以读该租户全部行。同一低权限角色在 HTTP 列表看不到他人单据，却可经 AI 工具/MCP 分页拉全量。human 主体不经过 Gate 2.5；非 human 主体通过策略后同样没有行级过滤。**影响是租户内越权读取，证据不支持跨租户泄漏。**

**修复建议。** 工具执行时传入不可由模型参数覆盖的认证 actor 与 DataScope，复用业务入口的范围判定；CI 读取显式区分「业务读」与「治理读」，不得靠缺省 `ScopeConfigured=false` 隐式获得全量范围。

**验证方式。** 真实 Router 下同租户甲乙各有工单/CI：甲的普通列表与 `list_tickets`/`list_cis` 结果范围必须一致；加入 tenant B 断言不串页；agent 策略放行后重复，证明 Gate 2.5 不替代行级权限。

---

## P1-2 审批后执行仍信任旧角色快照

**现象。** Worker 确有执行前重判，但角色取自历史记录：

```go
// internal/services/tool_executor.go:118
allowed, reason, guardErr = e.guard.RevalidateApprovedTool(
    ctx, tenantID, inv.UserID, inv.RoleSnapshot, inv.ToolName, inv.PolicyVersion)
```

重判服务直接消费该快照（`ai/service.go:222-227`），`evaluateToolCall` 的 Gate 2 再用它查权限（`:281`）。整条链没有按 `inv.UserID` 重新加载当前账号。

**为什么是问题。** 「旧角色本身仍有权限」不等于「发起人当前仍持有该角色」。用户提交后若被降级、停用或移出原归属，只要旧角色权限绑定未删，durable command 仍会按旧快照放行并执行副作用；AuthMiddleware 的撤销检查覆盖不到后台执行路径。

**验证方式。** 扩展 `tool_executor_test.go`：建待执行 invocation 后**只改用户当前角色、保留旧角色权限绑定**，经真实 `HandleCommand` 执行，断言工具调用次数为 0。现有 `TestToolExecutorRevalidatesApprovedSnapshot` 的 `revoked_role` 场景删的是角色权限关联，覆盖不到本路径。

---

## P1-3 权限停用后的跨副本角色缓存未失效（已知遗留的镜像残留）

**现象。** 策略写入路径同时刷两类缓存：

```go
// handlers/rbac/role_service_impl.go:756-760
middleware.InvalidateAuthzCaches(tenantID)          // 有 Redis 广播
middleware.InvalidateAllPermissionCachesEx()        // 只清本地
```

而广播订阅端只清 Registry 与 ACL：

```go
// middleware/permission_cache_broadcast.go:130-135
authz.Shared().Invalidate(tenantID)
InvalidateTenantACLCache(tenantID)
```

`InvalidateAllPermissionCachesEx`（`rbac.go:906`）没有广播，角色权限缓存 TTL 为 5 分钟。

**为什么是问题。** 多副本下副本 B 预热角色权限后，副本 A 停用权限：B 的策略目录被刷新，但角色→权限缓存仍给旧授权，人工 `RequirePermission` 与 Gate 2 可在最长 5 分钟内继续放行。非 human 主体另受 Gate 2.5 约束，不能据此认定必然放行。

**与既有记录的关系。** `docs/audits/rbac-audit.md` 的 AUDIT-022 记录「三类缓存只有一类广播失效」，当时是角色缓存有广播、Registry/ACL 没有；现状已反转为 Registry/ACL 有广播、角色缓存全量失效无广播。属于同一问题的残留面，不是全新技术债。

**修复建议。** 把策略停用广播扩展到目标租户的角色权限缓存，或写入路径显式广播角色缓存失效；两类缓存失效收敛为一个操作。

---

## P2-4 service 主体进入审批后无法执行（错误拒绝）

**现象。** service 主体合法携带 `userID=0` 进入判定（`ai/service.go:256`），但复核用非空策略版本反推主体：

```go
// handlers/ai/service.go:222-226
principal := authz.PrincipalHuman
if storedPolicyVersion != "" { principal = authz.PrincipalAgent }
```

于是 service 发起的调用在 Worker 复核时被还原成 agent，随后被 `userID<=0 && principal!=service` 拒绝。

**为什么是问题。** 这是**错误拒绝而非提权**：默认策略只允许 human，须管理员在运行期显式放行 service 并要求人工确认才触发，MCP service token 通道同样受影响。根因是 `policy_version` 同时承担了「策略指纹」和「主体身份编码」两个职责。

**修复建议。** 落库新增并持久保存原始 principal，执行时读取并校验；策略版本只表达策略指纹。

---

## P2-5 MCP 存量迁移遗漏 sysadmin 授权（新发现）

**现象。** 默认角色给 sysadmin 全量码：

```go
// internal/authz/roles.go:29
domainrole.SysAdmin: allPermissionCodes(),
```

但存量迁移只建两个角色的关联：

```sql
-- migrations/20261118_mcp_read_permission.sql:103-106
SELECT r.id, r.tenant_id FROM roles r
WHERE r.code IN ('agent', 'audit_admin')
```

**为什么是问题。** 存量、已配置 DB 角色且关闭自动播种的实例执行迁移后，`mcp:read` 行存在但 sysadmin 无关联；DBOnly `configured` 态尊重 DB 空集、不回退默认值，sysadmin 访问 MCP 被 403，而全新安装经 seeder 可正常访问。`super_admin` 的 `*:*` 直通会掩盖该缺口，不能用它验收。

**修复建议。** 补一个版本迁移为存量 sysadmin 建立关联（已执行过的部署不能只改旧 SQL）；DB 回读逐租户核对，并用真实 sysadmin（非 super_admin）做端到端验收。

---

## P2-6 `ValidateBootstrap` 启动自检未接线（本次补充发现）

**现象。** 注册表提供了部署期自检，注释明确「供生产启动自检使用」：

```go
// internal/authz/registry.go:255-257
// ValidateBootstrap 校验「引导清单 ⊆ DB 定义」与「码三元组一致」，
// 供生产启动自检使用
func (r *Registry) ValidateBootstrap(ctx context.Context, tenantID int) error { ... }
```

但全仓检索只有定义处与文档引用：`docs/audits/rbac-audit.md:15` 把它当作既有防线描述；生产装配 `internal/bootstrap/app.go:1095` 只调用了 `authz.ConfigureShared(client)`，没有任何调用点。

**为什么是问题。** DB 缺码、`code != resource:action`、三元组不一致这类部署缺陷本应在启动时 fail-fast，现在只能等到运行期被 fail-closed 拒绝并从日志反查。P2-5 的迁移遗漏若接了自检即可在启动期暴露——这是「部署期 lint」防线的缺失，不是运行期越权。

**修复建议。** 在 `ConfigureShared` 之后按租户调用 `ValidateBootstrap`；多租户场景需明确是逐租户校验还是仅校验已播种租户，失败时拒绝启动并输出缺失码清单。此项涉及启动语义，**需用户拍板**后再实施。

---

## 已检查但未发现问题的范围

| 范围 | 源码结论 |
|:--|:--|
| 默认权限单一源 | `defaults.go` 从 `BuiltinRolePermissionCodes()` 派生，seeder 同源；未见第二张授权表。 |
| 码空间一致性 | 后端 `catalog.go` 与前端 `catalog.gen.ts` 码集合一致（codex 只读提取比对）；这只证明码空间一致，不代表逐页动作覆盖完整。 |
| category 闭集 | 合法八类，MCP 迁移使用 `aiExtensions`，历史注释中的非法值不是当前缺陷。 |
| 三主体入站 | `auth.go` 校验 principal 闭集与 service 二次校验（角色/`userID=0`/正 tenant），未发现伪造提权。 |
| service 两处短路 | `rbac.go:547` 与 `ai/service.go:256` 均在。 |
| Registry 失败语义 | 缺码/停用/策略不允许/信封损坏/重复码一律 deny，不回退编译期清单。 |
| 审批资格与自批 | 路由要求 `ai:approve`；放行票检查目标工具权限、拒绝票宽进；AI 发起禁止发起人自批、二审不同人。human 的 `pending` 不套自批限制，属现有实现边界。 |
| 审批落库与入队 | 条件更新审批状态与 outbox 入队同事务，无「先批后入队」。 |
| MCP 复用链与 schema | 装配入口复用既有执行链；`outputSchema.type` 保持字面量 `object`。 |
| CI 双 NULL | `datascope/ci_filter.go` 明确把双 NULL 定义为租户公共资产，是显式产品规则，非意外 fail-open。 |
| 播种 DB 回读 | `initialization_adapter.go` 有按租户回读校验，不能说「完全没有回读」。 |

## 边界与待拍板

- **并行 WIP 排除**：`router/router.go`、`handlers/mcp/adapters.go`、AI 审批页面等存在未提交改动；本报告只依据已提交源码，不评价 WIP 行为。
- **需要产品拍板**：① CI 双 NULL 是否继续代表公共资产（P1-1 修复会改变一线可见性）；② human 发起的写工具是否也禁止自批；③ `ValidateBootstrap` 是否作为启动 fail-fast 条件（多租户语义）。
- **覆盖限制**：未连接生产库证明缺码已发生，未执行跨副本缓存探针，前端只做码集合比对与审批页抽查；不能据此宣称前端全量无问题或生产已复现。

## 建议收口顺序

1. **P1-1**：AI 工具行级范围（真实租户内越权读取，需产品确认可见性口径）。
2. **P1-2**：执行前按发起人 ID 重载当前角色/状态（后台执行路径的身份新鲜度）。
3. **P1-3**：角色缓存广播失效（沿用 AUDIT-022 的修复通道）。
4. **P2-6 → P2-5**：先接启动自检，再补 MCP sysadmin 迁移，避免只改 SQL 不防复发。
5. **P2-4**：持久化 principal，拆开策略版本与主体身份编码。

> 所有改动仍按 `itsm-authz-change` 流程执行：先 `session-ledger enter --claim`，单一源改动 + 迁移同批，验证矩阵全绿后显式暂存提交；本报告不触发任何代码改动。

---

## 补录（2026-11-25）：写工具 `update_ticket` 行级范围收口

本轮在已收口的 P1-1（读面范围）基础上继续核查同族写路径，发现并修复一处未列入原审计清单的缺口：

- **问题（基准 `c3cbe9a3`）**：`internal/services/tool_registry.go` 的 `update_ticket` 分支调用 `UpdateTicket(..., 0, "")`，以「系统操作」为由显式跳过 `enforceTicketRowScope`。HTTP 更新走同一服务时强制当前用户为请求人或处理人；AI/MCP 审批通过后的 durable command 却可改写同租户任意工单。Gate 2 只要求 `ticket:write`，该码由 `end_user` 等普通角色持有，因此是租户内跨记录写入，不是跨租户提权。
- **修复**：工具写路径与读面共用 `datascope.ActorFromContext(ctx, tenantID)`；actor 由服务器在鉴权后写入 context，模型参数无法伪造；缺失或租户不匹配时 fail-closed，并把 `actor.UserID` / `actor.Role` 传入 `TicketService.UpdateTicket`。管理员角色仍保留租户全量写入语义；未修改权限码目录、DB schema、迁移或部署策略。
- **回归证据**：新增 `internal/services/tool_registry_update_ticket_scope_test.go::TestUpdateTicketToolEnforcesTrustedActorRowScope`，覆盖非属主拒绝（且 DB 未变更）、属主允许、缺失 actor 拒绝、异租户 actor 拒绝、管理员全量允许。先回退修复运行为红（exit 1），恢复修复运行为绿（exit 0）。
- **剩余边界（不在本次范围）**：`get_ci` / `get_ci_tickets` 等同族只读工具仍未应用 CI 行级范围（P1-1 残留面）；service 主体 `userID=0` 仍按既有系统语义跳过行级校验，属既定设计边界，改变它需要产品拍板。
