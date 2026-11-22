# 连接器 required_permissions：从注册声明到执行强制

> 状态：设计锁定，本批实施。
> 事实依据：worktree `agentcp/2026-11-22-auto`（HEAD `8ff691517`，基于 `origin/main@bde1814df`）。
> 关键源码：`connector/connector.go:150-211`、`handlers/connector/handler.go:23-296`、
> `router/connector_routes.go:11-24`、`handlers/skill/handler.go:647-718`（对照样本）、
> `middleware/rbac.go:789-794`、`internal/authz/roles.go:258-319`、`internal/authz/catalog.go`。

## 1. 问题（代码事实）

`connector.Manifest.RequiredPermissions` 在注册链路上被三处强投入：

| 环节 | 代码 | 行为 |
|:--|:--|:--|
| 字段定义 | `connector/connector.go:170` | `RequiredPermissions []string` |
| 完整性校验和 | `connector/connector.go:183-193` | 排序后进入 `ComputeChecksum` 指纹 |
| 注册门禁 | `connector/connector.go:200-210` | 空列表直接拒绝注册（fail-closed） |
| 前端投影 | `handlers/connector/handler.go:107` | `ListMarket` 原样返回给 DTO |
| 真实声明 | `connector/builtin/{feishu,webhook,email,...}` | 如飞书声明 `connector:write,ticket:write,cmdb:write` |

但执行侧**零消费**：

- `router/connector_routes.go:17-22` 六个写端点全部只挂 `connector:write`/`connector:read`；
- `handlers/connector/` 与 `connector/` 非测试代码中 `HasResourcePermission` **零命中**；
- `handlers/connector/handler.go` 的 `Handler` 结构体（`:24-31`）**没有 ent client 字段**，连校验依赖都不具备。

后果：一个只持 `connector:write` 的角色，可以通过 `POST /connectors/feishu/send` 借飞书连接器向外部发消息，
绕过 manifest 自己声明的 `ticket:write`/`cmdb:write`。声明等于装饰——与 skill 侧同类缺口
（`handlers/skill/handler.go:647-654` 注释所述）完全同构，只是 skill 已修、connector 未修。

## 2. 设计原则

1. **单一判定源**：复用 `middleware.HasResourcePermission`（内部委托 `AuthorizeResourceForRole`），
   不在 connector 层另写一套 RBAC 匹配，避免"双判定源"。
2. **格式契约**：权限码统一 `resource:action`（与 `internal/authz/catalog.go` 一致，`strings.Cut` 解析）。
   格式非法视为 manifest 自身有问题 → **fail-closed 拒绝**（与 skill 侧一致）。
3. **fail-closed 依赖**：`client` 为 nil 且 DBOnly 模式时 `HasResourcePermission` 返回 false，
   所有声明了权限的连接器调用一律拒绝；测试用 HardcodeOnly 显式控制判定面。
4. **边界最小化**：只 gate **触发外部副作用的用户动作** `Send` / `Test`。
   `Provision`/`Revoke`/`RotateSecret` 保持路由级 `connector:write`：
   - 内部自动化链路（通知、告警、邮件回复）直接调 `manager.Send`，不经过 handler，不受影响；
   - 避免把 `email_intake:review`（当前**无任何内置角色持有**，见 `roles.go`）变成"配了也用不了"的死门。
5. **零新增权限码 / 零迁移**：列与 manifest 字段都已就位，不新增 RBAC 码、不改已发布迁移 checksum。
6. **空声明兼容**：`RequiredPermissions` 为空时放行（保持注册侧对可选字段的向后兼容口径，
   与 skill 侧 `enforceRequiredPermissions` 一致）。

## 3. 契约（新增 Go API）

```go
// Handler 增加 client *ent.Client 依赖；NewHandler 增加第 5 参数。
// bootstrap 注入真实 client；测试构造点注入 enttest client 或 nil（nil 用于 fail-closed 断言）。

// enforceRequiredPermissions 校验调用方是否持有连接器 manifest 声明的全部权限码。
// 任一不满足 / 格式非法 → 返回 false 且已写 403；全部满足或空列表 → 返回 true。
func (h *Handler) enforceRequiredPermissions(ctx *gin.Context, name string) bool
```

调用点：`Send`（`handler.go:243` 绑定参数后、`manager.Send` 前）、
`Test`（`handler.go:271` 取到 channel 后、`manager.Send` 前）。
未知 manifest（`registry.GetManifest` 未命中）放行，交由 `manager.Send` 报未知连接器——
与 skill 侧 `GetEntryScoped` 失败时放行的口径一致（不是权限判定失败）。

## 4. 验收标准（每条可独立验证）

| # | 判据 | 验证方式 |
|:--|:--|:--|
| V1 | manifest 声明 `ticket:write`，请求角色缺该码 → `Send` 403，manager 不被调用 | enttest + HardcodeOnly，角色 `admin` + email 连接器（缺 `email_intake:review`） |
| V2 | 角色持有 manifest 全部声明 → 放行并调用 manager | 角色 `admin` + 飞书连接器（持 connector/ticket/cmdb:write） |
| V3 | 权限码格式非法（如 `connector.write`）→ fail-closed 403 | 注册非法 manifest 夹具 |
| V4 | `RequiredPermissions` 为空 → 放行（兼容） | 空声明夹具 |
| V5 | `Test` 同样受 gate | 同 V1，改走 Test 入口 |
| V6 | `client` 丢失（DBOnly + nil）→ fail-closed | 对照构造：不注入 client 的同一请求 403 |
| V7 | 内部自动化链路不受影响 | `manager.Send` 直调不走 handler，编译 + 全包绿 |
| R1 | 反向验证：移除 `Send` 中 gate 调用 → V1 变红 | 手工注入 |

## 5. 明确不做（边界）

- gate `Provision` / `Revoke` / `RotateSecret`（保持路由级 `connector:write`）
- 给 `admin` 补 `email_intake:review` 权限码（属 RBAC 授权面独立议题，会连锁 parity/seeder 测试）
- 新增 HTTP 端点、前端提示、文档站改动
- 修改 manifest 的 `required_permissions` 内容或 checksum 算法
