# 权限体系改进方案：DB 运行期单一真源 + AI 零信任

> 状态：后端实现完成（编译 + 定向测试通过）；前端策略编辑面板与文档同步为待办。
> 上游锚点：[`ROADMAP.md`](../ROADMAP.md)、[`plans/optimized-improvement-plan-2026-11-14.md`](optimized-improvement-plan-2026-11-14.md)。
> 方法：先定问题与设计目标，再给架构与实现清单，最后列待办与验证。

---

## 0. 一句话结论

权限码**只在 DB 的 `permissions` 行定义一次**（不再是散落各处的硬编码字典）；元数据以版本化 JSON 信封塞进既有的 `description` 字段（**不改 ent schema**）；AI 主体**默认零权限**，必须由管理员在运行期显式授权，并按风险分级走人工审批 / 双人审批。整条链路 **fail-closed**：任何读不出来、解析失败、信封损坏的判定都 deny，绝不回退到编译期码表。

---

## 1. 背景与问题

### 1.1 旧架构的三个痛点

1. **权限码散落、硬编码**：曾经在 `InitDefaultPermissions` 里维护第二张手写 `resourceActions` 字典（含 `dashboard:view` / `admin:all` / `groups:*` 等码空间里根本不存在的码），路由里内联声明、前端再内置一份词表。三处各自维护、互相漂移。
2. **AI 安全攻击持续且门槛低**：提示词注入只能改变 LLM「想调什么」，但如果权限判定来自编译期硬编码、且默认放行，攻击者就能让 AI 绕过管控执行敏感操作。
3. **人机分工变了**：人主要做审批确认，AI 做具体操作。细粒度操作权限因此成为企业内部的关键防线——必须「默认不给、显式授权、按风险分级审批」。

### 1.2 设计约束（为什么不改 ent schema）

- 本仓库的 ent 生成器与已提交产物存在**版本漂移**：重跑 `go generate ./ent` 会重写 **382 个生成文件**，在共享工作树里做 schema 变更会污染并行会话。
- 因此元数据信封复用既有的 `permissions.description` 字段，向后兼容存量纯文本描述。

---

## 2. 设计目标

| 目标 | 实现 |
|:--|:--|
| 单一真源 | 权限码只在 DB `permissions` 行定义一次；`authz.Definitions()` 仅作全新库一次性引导清单，运行期判定不读它 |
| 运行期可配 | 管理员改策略 → `PUT /api/v1/permissions/:id` → 写库 + `Invalidate(tenantID)` → 下一次判定立即生效，无需重编译发版 |
| AI 零权限默认 | `DerivePolicy` 的 `AllowedPrincipals` 恒为 `["human"]`；要授权给 AI 必须由管理员显式把 `agent` 加进某个码 |
| 风险分级 | 风险等级由 action 语义**按规则推导**（不是再维护一张逐码字典），管理员可逐码覆盖 |
| fail-closed | 读不出 / 解析失败 / 信封损坏 / 重复码 / 码三元组不一致 → 一律 deny，绝不宽松放行 |

---

## 3. 架构设计

### 3.1 引导清单与运行期真源的分工

- `authz.Definitions()`（`internal/authz/catalog.go`）是**唯一的引导清单**，只在全新库首次引导（`seeder.seedPermissions` / `InitDefaultPermissions`）时读一次。
- 运行期鉴权路径（`Registry.Lookup` / `AuthorizePrincipal`）只读 DB（带 TTL 缓存），**绝不回退到引导清单**。
- **尊重运行期配置**：已配置过的行（已经是 v1 信封）在 seed / 重启时一律不覆盖；只有缺失的码才补齐；存量纯文本行一次性升级为信封、人类描述原样保留。

### 3.2 策略信封（不改 schema）

- `PermissionPolicy` 以版本化 JSON（`v=1`）编码进 `permissions.description`（`EncodeDefinition` / `DecodeDefinition`）。
- 三种历史形态兼容：`空串` / `纯文本`（旧描述 + 派生默认策略，AI 仍零权限）/ `v=1 JSON`（完整运行期策略）。
- 未知版本、损坏 JSON、非法策略一律返回错误，由 `Registry` 走 **fail-closed**——绝不「解析失败就按宽松默认值放行」。

### 3.3 风险分级按规则推导

`RiskForAction`（`internal/authz/policy.go`）按 action 语义推导，**新增权限码无需同步任何字典**：

| action 类别 | 风险 | 最小审批人数 |
|:--|:--|:--|
| read/view/list/get/search（只读） | low | 0 |
| approve/rollback/escalate/archive（改变裁决/回滚） | high | 2 |
| delete/admin/all/provision/override/force-update（不可逆/外部交付） | critical | 2 |
| 未知动作 | medium（宁多一道审批，不静默归 low） | 1 |

`MinApprovalsFor`：low=0，medium=1，high/critical=2。

### 3.4 主体类型与零信任

- 主体类型：`human` / `agent` / `service`（`authz.PrincipalHuman/Agent/Service`）。
- `principal` 是**请求级**身份（一次动作的发起者），不放在 JWT claim 里：同一个人可能自己点按钮（human），也可能让 AI 代做（agent）。
- `Registry.AuthorizePrincipal` 是 AI 执行链的**单一判定入口**：
  1. 码必须在 DB 登记且启用；
  2. 主体必须被该码 `AllowedPrincipals` 显式允许（默认只有 human → AI 零权限）；
  3. 返回该主体需要的人类确认强度。
- `ValidatePolicy` 闭集校验（安全不变量在写入前与读出后都强制）：允许 AI 的中高风险**必须开审批**；允许 AI 的高/危急风险**必须双人审批**；低风险不得强制审批；`AllowedPrincipals` 不得为空或重复。

### 3.5 双人审批 / 防自批

- `ExecuteTool` 把 AI 写操作创建为待审批调用，状态区分：`pending`（人发起，单人）/ `pending_agent`（AI 发起，单人 + 禁止自批）/ `pending_agent_two`（AI 发起，双人 + 禁止自批 + 两名不同审批人）。
- `repository_impl.DecideToolInvocation` 阻止自审批（`actorID != existing.UserID`）并对 pending 状态做 CAS，避免并发重复放行。
- 审批与待执行任务在同一事务内落库（`ApproveTool` → `DecideToolInvocation`），通过审批不会因进程退出或队列满而丢失执行。

### 3.6 缓存与失效

- `Registry` 按租户 TTL 缓存（默认 5min，与 `middleware` 权限缓存口径对齐）。
- `sharedRegistry` 是进程级**原子指针、全局唯一实例**；所有写入方（权限管理 API）与读取方（AI 执行链）必须用同一个实例，否则写后失效只清掉一方缓存。
- 写入后调用 `authz.Shared().Invalidate(tenantID)` 精确失效。

---

## 4. 实现清单（已完成）

### 4.1 核心包

| 文件 | 职责 |
|:--|:--|
| `internal/authz/policy.go` | 主体类型、风险分级规则、`PermissionPolicy`、信封编解码、`ValidatePolicy`、`DerivePolicy` |
| `internal/authz/registry.go` | `Registry`（TTL 缓存 + 共享实例 + fail-closed）、`AuthorizePrincipal`、`Catalog` 接口（便于单测注入 fake）、重复码检测 |
| `internal/authz/catalog.go` | `Definitions()` 单一引导清单、`PermissionDef` |
| `internal/authz/roles.go` | `BuiltinRolePermissionCodes()`、`RetiredRolePermissionCodes()`（CI 守卫锚点） |
| `dto/permission_dto.go` | `PermissionResponse`（富策略字段）、`PermissionPolicyPayload`、`UpdatePermissionRequest`、`PermissionCatalogMetaResponse` |

### 4.2 应用层

| 文件 | 职责 |
|:--|:--|
| `handlers/rbac/handler.go` | `ListPermissions` / `CreatePermission` / `UpdatePermission` / `GetPermission` / `GetPermissionCatalogMeta` |
| `handlers/rbac/role_service_impl.go` | 移除硬编码 `InitDefaultPermissions` 字典；改用 `DerivePolicy` + `validateCodeTriple`；信封编解码；`CatalogMeta` 返回闭集 |
| `handlers/ai/service.go` | `ExecuteTool` Gate 2.5 AI 主体零信任；`WithPrincipal` 标记；`ApproveTool` 双人审批；`recordToolAudit` 服务端底账；`ChatStream` 注入 `authz.WithPrincipal(ctx, PrincipalAgent)` |
| `handlers/ai/repository_impl.go` | `DecideToolInvocation` 防自批 + CAS |
| `pkg/seeder/seeder.go` | `seedPermissions` 写信封；尊重运行期配置（`IsDefinitionEnvelope` 跳过已配置行）；存量纯文本行一次性升级 |
| `internal/bootstrap/app.go` | `authz.ConfigureShared(client)`（已修复装配顺序，见 §6.1） |
| `router/common_system_routes.go` | 移除 `permissions.POST("/init", ...)`（初始化改由 seed / `InitDefaultPermissions` 处理） |
| `cmd/authz-gen/main.go` + `rbac_precheck_gen.go` | 重生成（CI 守卫用） |

### 4.3 测试

- `internal/authz/policy_test.go`：风险推导、信封编解码、闭集校验回归。
- `internal/authz/registry_test.go`：用 `enttest.Open(t, "sqlite3", "file:...?mode=memory&_fk=1")` + sqlite 驱动；重复码 fail-closed。
- `handlers/rbac/handler_test.go`：移除 `InitDefaultPermissions` mock。
- 全量 `go build ./...` 通过；`internal/authz`、`handlers/rbac`、`handlers/ai`、`pkg/seeder`、`router` 定向测试通过。

---

## 5. 待办（前端 + 文档 + 守卫）

### 5.1 前端权限管理页策略编辑（关键缺口）

- 现状：前端类型已就绪（`api-config.ts` 的 `PermissionCatalogItem` / `PermissionPolicyPayload` / `PermissionCatalogMeta`），但 `admin/permissions/page.tsx`（1116 行）**不渲染也不编辑**任何策略字段（`riskLevel` / `allowedPrincipals` / `minApprovals` / `isActive` / `configured` / `category` 均无引用）。
- 需要补的 UI：
  1. **策略编辑面板**：风险等级选择、AI 主体授权开关（`human` 默认开，`agent`/`service` 需管理员显式开）、审批强度（`requiresApproval` / `minApprovals`）。
  2. **`configured` 提示**：存量纯文本行（`configured=false`）提示管理员显式确认一次（派生初值恒为 AI 零权限，符合最小权限）。
  3. **分类筛选 / 标签**：用 `GET /api/v1/permissions/meta` 返回的闭集（`categories` / `categoryLabels` / `riskLevels` / `principals` / `minApprovalsMax`），前端零硬编码。
  4. **停用而非删除**：`isSystem=true` 的引导码只允许停用，UI 禁用删除。

### 5.2 文档与发布口径

- `CHANGELOG.md` 的 `[Unreleased]` 增加「权限体系：DB 运行期单一真源 + AI 零信任」小节。
- `ROADMAP.md` 发布口径：把「AI 零信任权限」作为安全能力条目（避免未评审就改发布口径）。
- 管理员操作手册：如何给某个码授权给 AI、风险评估指引、双人审批流程。

### 5.3 端到端验证

- E2E 演示脚本：AI 发起高危操作（如 `ticket:delete`）→ 校验默认被零权限拒绝 → 管理员授权 `agent` 并设双人审批 → 提交待审批 → 两名不同审批人通过 → 执行。
- 守卫：路由声明的 `(resource, action)` ⊆ 码空间（`checkPermissionMatch` 精确匹配，引用不存在的码对除 `super_admin` 外永久 403）。

---

## 6. 关键修复记录

### 6.1 bootstrap 装配顺序（已修复）

- **原问题**：`app.go` 在 AI 域装配时（`aiServiceDomain.SetPermissionCatalog(authz.Shared())`）捕获的是 `init()` 创建的 **nil-client** 注册表；之后 `authz.ConfigureShared(client)` 又创建了**第二个**带 client 的注册表并存入 `sharedRegistry`。结果：AI 服务永远持有 nil-client 实例，`AuthorizePrincipal` 一律返回 `ErrCatalogUnavailable`，**所有 agent 发起的工具调用被 fail-closed 拒绝**——AI 零信任链名存实亡。
- **修复**：把 `authz.ConfigureShared(client)` 移到 AI 域装配**之前**调用一次，并删除其后重复的 `ConfigureShared` 调用。AI 域与权限管理 API 现在共用同一个 client-wired 实例，写后 `Invalidate` 对双方都生效。

---

## 7. 验证与回滚

### 7.1 验证清单

- [x] `go build ./...` 通过
- [x] `internal/authz` / `handlers/rbac` / `handlers/ai` / `pkg/seeder` / `router` 定向测试通过
- [ ] 前端策略编辑面板（§5.1）
- [ ] 端到端 AI 授权 + 双人审批演示（§5.3）
- [ ] `CHANGELOG` / `ROADMAP` 同步（§5.2）

### 7.2 回滚

- 架构方向是 fail-closed：即使 `ConfigureShared` 装配失败，AI 主体判定也只会 deny（安全方向），不会放宽。
- 如需回退运行期策略改动，直接把对应 `permissions` 行的 `description` 改回纯文本或删除信封即可，`DecodeDefinitionLenient` 会降级为派生默认策略（AI 零权限）。
- 代码回滚：撤销 `app.go` 的 `ConfigureShared` 改动即回到 fail-closed 状态（安全方向），不影响既有 RBAC 链路。
