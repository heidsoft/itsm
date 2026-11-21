# 部署与租户隔离审计

> 基准 commit：`b3bfd748604041d5082a320ed6d5c447221ac76a`

## 1. 生产配置默认值（实测）

| 变量 | 生产值 | 安全判断 |
|:---|:---|:---|
| `JWT_SECRET` | `docker-compose.prod.yml:125,197,387` `${JWT_SECRET:?...}`，强校验 `default_credentials_guard.go:63-71,150-175` | 无不安全默认值 |
| `ADMIN_PASSWORD` | `docker-compose.prod.yml:128,193` `${ADMIN_PASSWORD:?...}`，弱口令闸门 `credential/policy.go:15-26` | 无不安全默认值 |
| `ITSM_AUTO_MIGRATE` | init=true（`:130`），backend/worker=false（`:206,392`） | 单一职责，符合设计 |
| `ITSM_AUTO_SEED` | init=true（`:131`），backend/worker=false（`:207,393`） | 同上 |
| `RLS_MODE` | init 硬编码 `off`（`:144`），app `${RLS_MODE:-off}`（`:212,396`） | 分阶段设计，不是可直接断言的 P0 |
| `BOOTSTRAP_TOKEN_ENABLED` | init `${...:-0}`（`:143`），backend `${...:-1}`（`:194`） | 与一次性 bootstrap 路径一致 |

`RLS_MODE` 解析为严格模式（`database/rls/driver.go:70-84`），未知值启动失败；readiness 探测（`database/rls/readiness.go:109-154`）在连接角色为 superuser/BYPASSRLS 或策略缺失时拒绝声称 `enforce`。`rls-golden-path.yml:12-17` 说明生产 RLS 仍是试点，仅 `changes`、`vectors` 有策略。

**不得在没有完整兼容性测试的情况下把生产默认值翻成 `enforce`。**

## 2. 应用层租户隔离证据

租户解析 fail-closed：`middleware/tenant.go:24-30,44-58,133-157,159-183` 忽略客户端 `X-Tenant-ID`，以 JWT claim 锁定租户，不匹配即拒绝。
RBAC 每请求重新加载用户与角色（`middleware/rbac.go:531-562`，按 `IsActive && TenantID` 过滤）。
全局写守卫仅覆盖 Create（`database/security.go:94-149`），Update/Delete 依赖各域 `TenantIDEQ`——这是残余风险属性，不是已证实漏洞。

已存在的真实 Router/Handler 入口负例测试：

- `tests/rbac/cross_tenant_test.go`：`TestCrossTenantUserList/Enumeration/TicketEnumeration/CIEnumeration/KnowledgeEnumeration`
- `tests/scenarios/scenario6_tenant_isolation_test.go`
- `handlers/analytics/cross_tenant_http_test.go`（精确计数隔离）
- `handlers/ticket_notification/cross_tenant_http_test.go`（含 outbox 隔离）
- `handlers/ticket_attachment/`、`ticket_comment/`、`ticket_view/`、`ticket_dependency/`、`change_review/`、`asset/`、`auditlog/`、`provisioning/` 各 `cross_tenant_http_test.go`
- `internal/services/bpmn_cross_tenant_e2e_test.go`、`tests/multi_tenant/rls_matrix_test.go`

结论：**未发现已证实的跨租户越权路径**。

## 3. 发现项

```yaml
id: AUDIT-050
priority: P2
category: deployment/secrets
status: confirmed
title: .env.prod.example 关闭 /metrics 鉴权，与 compose 默认值相反
file: .env.prod.example:148
evidence: METRICS_REQUIRE_AUTH=false；docker-compose.prod.yml:227 默认 ${METRICS_REQUIRE_AUTH:-true}；router/router.go:414-421 仅在值为 false 时跳过 AuthMiddleware
reproduction: cp .env.prod.example .env.prod 后启动，未带 JWT GET /metrics 返回 200
impact: 本机用户或隧道访问者可读取进程指标；后端仅绑定 127.0.0.1:289，影响有限
root_cause: 示例 env 为监控 profile 编写，但它是所有部署的默认复制模板
fix: 改为 true，并把关闭方式写成注释；加静态守卫禁止示例文件关闭鉴权
acceptance: 示例文件复制后未授权 /metrics 返回 401；守卫对注入的 false 变红
status_after_fix: resolved (PR-01)
```

```yaml
id: AUDIT-051
priority: P1
category: deployment/backup
status: confirmed
title: 备份依赖宿主定时脚本，compose 与部署后验收未强制
file: docker-compose.prod.yml:28 (./backups:/backups:ro)
evidence: scripts/prod-backup.sh、prod-restore-drill.sh、com.itsm.prod-backup.plist.template 均存在，但 compose 无调度服务，prod-post-deploy-verify 未断言存在备份
impact: 按 README/compose 部署后 backups/ 为空，漏装定时即无本地恢复点
fix: 部署文档加入"安装定时"步骤；部署后检查至少一份备份且 drill 成功
acceptance: prod-post-deploy-verify.sh 断言备份文件存在且 prod-restore-drill.sh exit 0
```

```yaml
id: AUDIT-052
priority: P2
category: deployment/supply-chain
status: confirmed
title: 基础设施镜像使用浮动标签
file: docker-compose.prod.yml:17,45,84,494
evidence: pgvector/pgvector:pg17、redis:7-alpine、minio/minio:latest、nginx:alpine；应用镜像通过 ${VERSION} 固定
impact: 重新拉取可在应用未发版时改变运行内容；latest 最差
fix: 固定次要版本或 digest，并记录刷新节奏
```

```yaml
id: AUDIT-053
priority: P1
category: deployment/db-index
status: confirmed
title: 重复索引阻断 fresh-install 门禁
file: migrations/20261119_ci_owner_fk_and_certification.sql:76-77；ent/schema/configurationitem.go:225-226
evidence: ci_tenant_owner_idx / ci_tenant_department_idx 与 configurationitem_tenant_id_owner_id / configurationitem_tenant_id_department_id 同列；database/database.go:161-195 收敛表未含这两对；fresh-install gate run 44882459697 报错
impact: 首装门禁持续红；每次 CI 写操作维护两棵相同 B-tree
root_cause: SQL 迁移与 Ent Schema 两个 DDL 来源，收敛表漏登记
fix: 在 duplicateIndexPairs 增加两对，保留 Ent 生成名
rollback: 撤销收敛项后手写索引不会自动重建（迁移已入账），需新的版本化迁移；Ent 索引不受影响
approval: 数据库结构改动，需人工审批
```

### AUDIT-002 本轮已验证修复

基准 `a2103b16`。`database/database.go` 的 `duplicateIndexPairs` 增加两对：`configurationitem_tenant_id_owner_id` 保留、`ci_tenant_owner_idx` 删除；`configurationitem_tenant_id_department_id` 保留、`ci_tenant_department_idx` 删除。收敛逻辑抽出可测内核 `ensureIndexConvergenceWithPairs`，新增真实 Postgres 回归：夹具造出两对同列索引后，调用一次删除 2 个且保留 Ent 索引，重复调用删除 0 个，保留方缺失时拒绝删除并不动冗余方。静态守卫同时锁住两对的方向。未改写任何已发布迁移 SQL，账本 checksum 不变。

## 4. 部署验收清单（PR-01 交付物）

见 `docs/audits/remediation-backlog.md` 的 PR-01 条目；清单覆盖：env 示例安全、迁移/播种职责、RLS 模式与 readiness、备份与恢复演练、镜像固定、健康与资源限制。
