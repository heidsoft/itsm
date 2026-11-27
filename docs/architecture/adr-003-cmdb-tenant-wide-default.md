# ADR-003: CMDB HTTP 业务读面按行级权限收窄

> Status: current

## 状态

Accepted（2026-11-26），替代 2026-09-16 的默认 tenant-wide 读权限决策。

## 背景

CMDB 的 CI、关系、拓扑用于跨部门协作，但这不意味着每个具备 CMDB
读权限的用户都应看到全部已明确归属的资产。HTTP 业务读入口此前只有
列表接入 scope，搜索、详情和关系图可以绕过列表权限，必须统一收窄。

CI 不能照搬工单的申请人/受理人模型：工单天然有责任主体，资产则可以无主，
发现与存量导入尤其如此。将全部无主资产隐藏会导致一线无法认领资产，
也会让变更、事件处理失去跨部门依赖信息。因此保留公共资产例外，并明确
区分用户业务读面与可信内部分析。

## 决策

1. HTTP CI 列表、搜索、详情、统计、关系列表/详情、拓扑、影响分析和历史
   均从认证中间件身份显式注入 scope；请求体、查询参数和 Header 不得自报身份或权限。
   管理角色经 `datascope.ForCIReader` 得到 `DataScopeAll`，可见全租户业务资产；
   其余角色复用 `ApplyCIFilter`，按本人归属及既有 scope 规则收窄。
2. `owner_id` 与 `department_id` 双 NULL 的 CI 是租户公共资产，所有有效用户
   均可见。`departmentID<=0` 不添加部门谓词；需要用户身份的 scope 在
   `userID<=0` 时为空集，未知 scope 也为空集。不得更改既有原语的这些语义。
3. 关系两端必须同时可读；详情预加载、拓扑与影响分析遍历也遵守同样边界，
   不返回隐藏节点、相邻 CI 内容或跨租户关系。不可读的根 CI 按不存在处理。
4. HTTP scope 使用显式 typed context 标记，service 不根据普通 actor context
   无条件推断行级限制。后台任务、变更/事件的可信内部影响分析、导出和治理
   保持租户级语义；调用方仍须负责权限与 tenant scope。
5. AI 工具保留既有显式 scope 注入，以可信 actor 判定权限，不能从模型参数
   接受 scopeAll 或用户身份。不得把 AI 工具视为自动豁免的后台调用。
6. 治理报告的分母包含退役 CI，退役仍参与 Assessable。仅
   `ListCIsForGovernance` 使用 `database.WithIncludeRetiredCIs`，绕过 CI 的
   scrapped 读过滤；该私有 typed context key 不接受任何 HTTP 字段映射，
   不绕过租户谓词、写守卫或数据库 RLS。普通业务读面继续隐藏退役 CI。
7. 路由的 CMDB RBAC 仍是访问前提，行级权限不替代接口授权。

## 后果与边界

已归属的敏感资产不会因切换搜索、详情或拓扑入口而泄漏；公共资产仍支持
认领与跨部门协作。可信内部分析可保留完整租户依赖视图，调用边界需要审查，
不得把外部请求直接透传为无 scope 的内部调用。

关系读过滤把同一 `CIFilterPredicates` 下推到 `HasSourceCiWith` /
`HasTargetCiWith` 的 SQL EXISTS 子查询，先物化可见 CI ID 列表再用 `IDIn`
的做法已废弃——后者在大租户会撞 SQL 参数数量上限。谓词单一源在
`internal/datascope/ci_filter.go`，列表查询与关系读都从它派生。

## 落地证据

- `router/cmdb_routes.go`：CMDB 实际生产路由及 RequirePermission。
- `handlers/cmdb/production_service.go`：HTTP 可信身份与 scope 注入。
- `internal/services/configuration_item_service.go`：显式 scope 与 CI 查询过滤。
- `internal/services/ci_relationship_service.go`：关系两端与图遍历过滤。
- `internal/datascope/ci_filter.go`：公共资产、部门及 fail-closed 语义单一源。
- `handlers/cmdb/repository_impl.go` 与 `database/security.go`：治理专用退役查询开关。
- `handlers/cmdb/production_service_test.go`：生产 Handler 读权限与真实安全拦截器回归。

## 与既有 ADR 的关系

ADR-001 的模块化单体边界保持适用；ADR-002 的事件驱动不构成读取权限豁免，
异步消费者仍须明确其租户、权限、审计及可信内部调用边界。
