# ADR-003: CMDB 行级权限默认 tenant-wide

## 状态

Accepted（2026-09-16）

> **背景**：本 ADR 把 R4-a「RBAC 行级 scope」调研结论中关于 CMDB 的口径**显式固化**。
> 此前 `handlers/cmdb/*` 零 DataScope 接入（grep 实证），其它核心 6 域（ticket /
> change / incident / problem / release / service_request）均已通过
> `handlers/common/datascope/datascope.go` + 各域 repository 层
> `Or(Owner, Assignee)` 强制收窄。CMDB 显式缺位并非缺陷，而是**有意识设计**——
> IT 基础设施（CI、关系、拓扑）天然是跨部门共享的工作底座，行级收窄反而
> 会让变更/事件/问题处理失明。本 ADR 把这层语义写进文档，避免后续误接入
> 行级过滤造成跨部门协作断裂。

## 背景

RBAC 行级权限（DataScope）原语在 ticket 域落地后（阻断 8 修复），逐步推广到
6 个核心域（commit 序列 09-11 至 09-16 见 `ROADMAP.md` v1.6.x）。CMDB 是
唯一显式未接入的域：

| 域 | DataScope 写 | DataScope 读 |
|---|---|---|
| ticket | ✅ service 收窄 | ✅ repository `Or(Requester, Assignee)` |
| change | ✅ service 收窄 | ✅ repository `Or(CreatedBy, Assignee)` |
| incident | ✅ service 收窄 | ✅ repository `Or(Reporter, Assignee)` |
| problem | ✅ service 收窄 | ✅ repository `Or(CreatedBy, Assignee)` |
| release | ✅ service 收窄 | ✅ repository `Or(CreatedBy, Owner)` |
| service_request | ✅ handler 收窄 | ✅ repository |
| **cmdb** | **tenant-wide**（无行级） | **tenant-wide**（无行级） |
| knowledge | knowledgeaccess 单独处理 | — |

CMDB 资源（CI / 关系 / 拓扑 / 云账号 / 发现作业）属于 IT 基础设施，天然
跨部门可见。变更/事件/问题管理需要查询 CI 关系图、拓扑、依赖——若 CMDB
按"申请人/受理人"行级收窄，处理人会查不到非自己创建但影响自己变更的 CI，
反而引入**决策盲区**与**协作断裂**。

## 决策

1. **CMDB 行级权限默认 = tenant-wide**：所有租户内角色（end_user / agent /
   manager / admin / super_admin 等）均可读全租户 CMDB。
2. **写路径仍受通用 RBAC 权限位控制**（`RequirePermission` 中间件），不放开
   普通 end_user 写入 CI；行级问题不存在，权限位问题已由 `RequirePermission`
   统一处理。
3. **Department tier 仍是死代码**——`ApplyTicketFilter` 0 调用方，`ctx.DepartmentID`
   0 注入点。CMDB 不引入 Department tier，因基础设施跨部门可见是工作常态。
4. **未来扩展点**（本 ADR 不实施）：若客户要求"敏感 CI（合同价/安全等级）
   限可见"，应新增 `ci_sensitivity` 字段 + 按字段过滤，**不**走 DataScope
   的三档枚举（DataScope 三档只覆盖「全员/本部门/本人」三种语义粒度）。

## 后果

### 正面

- CMDB 跨部门可见与工作流协作天然一致，避免变更/事件/问题处理失明；
- 不引入新的中间件/上下文注入，0 代码改动；
- 文档把口径显式化，避免后续误接入行级过滤造成跨部门协作断裂。

### 负面

- 无法直接按"我的 CI 视图"过滤（需客户端按 created_by 字段再过滤）；
- 敏感 CI（如合同价、安全事件相关）默认全员可见，需要未来按 `ci_sensitivity`
  字段二次过滤（本 ADR 不阻塞当前商业化路径）。

### 缓解

- `security_classification` 字段预留（`ent/schema/configuration_item.go`）；
- 客户端若需要"我创建的 CI"过滤，可直接传 `created_by` filter 参数
  （CMDB handler 已支持）。

## 与既有 ADR 的关系

- ADR-001 模块化单体：CMDB 作为领域切片保留，不引入跨服务调用；
- ADR-002 事件驱动：CMDB 变更（CI 创建/关系更新）通过事件总线异步广播，
  CMDB 行级权限的广播不受本 ADR 影响——广播范围仍按角色/权限位控制，
  非按数据可见性。

## 落地证据

- `handlers/common/datascope/datascope.go`：DataScope 原语单一源
- `handlers/cmdb/*`：grep `DataScope|datascope|OwnerID|RequesterID` 0 命中（实证 tenant-wide）
- `repository/ticket/repository_impl.go:479-489`：ticket 域 DataScopeOwnedOrAssigned 强制收窄样板
- `handlers/change/repository_impl.go:243-251`：change 域强制收窄样板
- `handlers/incident/repository_impl.go:250-258`：incident 域强制收窄样板
- `handlers/problem/repository_impl.go:284-292`：problem 域强制收窄样板
- `service/release_service.go:190-200`：release 域强制收窄样板
- `handlers/service_request/repository_impl.go:316-324`：service_request 域强制收窄样板

## 未来可触发本 ADR 重审的条件

- 客户要求 CMDB 按部门/角色过滤（触发本 ADR 升级为 CMDB scoped 版）；
- DataScope 引入第四档（如 "team" 团队级），CMDB 是否纳入；
- 安全合规审计要求 CMDB 记录按密级二次过滤。