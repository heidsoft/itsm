# 租户隔离（Tenant Isolation）

本文档是 `internal/schema/tenant_guard.go` 头部注释与 `TenantExemptTables`
所引用的同步产物（代码注释第 2 条同步义务）。治理真相源在代码里；
本文档只做索引与解释，不复制第二份清单。

## 两层防线

1. **应用层 tenant predicate（强制，主防线）**：所有 Ent 查询（Get/Only/Exist/
   Count/List/Update/Delete、关联表、聚合、Raw SQL、缓存 key、后台扫描、导出）
   必须携带租户谓词；租户上下文缺失时 **fail closed** —— 禁止默认租户、
   禁止回退 tenant 1、禁止以全局管理员角色名推断跨租户权限。
   身份（tenantId/userId/requester/actor）只来自认证中间件上下文，
   不信任请求体、查询参数或客户端 Header 的同名字段。
2. **DB RLS（纵深防御）**：RLS 不能替代应用层谓词，两层都必须 fail closed。
   CMDB 租户范围默认值的取舍见
   [adr-003-cmdb-tenant-wide-default.md](./adr-003-cmdb-tenant-wide-default.md)。

MSP 场景（operator tenant / customer tenant / delegated scope）与跨租户
系统任务（显式 system context + 审计）的语义按仓库根 `AGENTS.md`
「身份、租户与数据范围」一节执行，此处不重复。

## 无 tenant_id 表的豁免注册表

排除项的**单一真相源**是
[`internal/schema/tenant_guard.go` 的 `TenantExemptTables`](../../itsm-backend/internal/schema/tenant_guard.go)。
本节只收录登记依据摘要（与代码注释保持一致，修订以代码为准）：

| 表组 | 依据 |
| --- | --- |
| `tenants` | 租户自身元数据，递归依赖无法含 tenant_id |
| `schema_migrations` | 迁移历史表，全局唯一 |
| `user_roles` / `role_permissions` / `role_permissions_join` | 平台级 RBAC 关系表，受租户过滤拦截会破坏单点授权 |
| `*_tags` 系列 | 用户可自定义标签，颜色/标签云跨租户共享是合理设计 |
| `knowledge_article_participants` / `sessions` / `version` | 协作快照，session id 是天然唯一键 |
| `msp_allocations` | MSP 模式下的租户-服务分配，是租户维度之上的概念 |

新增/修改豁免条目必须**三同步**：

1. 至少一条 `TableGuardTest` 单元测试（断言表确实不含 tenant_id，或豁免理由准确）；
2. 更新 `internal/schema/tenant_guard.go` 的 `TenantExemptTables` 及本文档上表；
3. `CHANGELOG.md`（Unreleased）。

每个条目带 `Owner`（治理负责人）与 `ReviewedAt`（上次复核日期）；
超过 180 天未复核会触发告警（不阻断启动）。

## 测试矩阵

租户隔离的回归证据由两类测试承载：

- **HTTP 入口级**：各 `handlers/<domain>/cross_tenant_http_test.go` —— 通过
  `router.SetupRoutes` 真实布线 + 真 JWT 认证 + RBAC，断言跨租户读取零泄漏、
  写入零副作用、错误形态可区分（当前已覆盖 analytics / auditlog /
  provisioning / ticket_comment / ticket_dependency / ticket_notification /
  ticket_view 七个域，随默认 `go test ./...` 进 CI）。新域接入沿用该模式。
- **Schema 级**：`internal/schema` 的 TableGuard 测试，守护无 tenant_id
  表的豁免清单不被静默扩散。
