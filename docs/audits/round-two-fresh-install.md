# 第二轮空库首装验收及索引整改方案

基准为 `de8d8cf54f9905420efb471530dadf6083e7e345`。

| 检查 | 结果 | 证据 |
| --- | --- | --- |
| PR #114 合并与代码核对 | PASS | 合并提交 `44f2104e5cf1c5e0438e16406148f1f96ba14c4d`。空库由后续初始化种子建立权限。存量租户以各自租户为范围补齐权限。 |
| MCP 权限 SQL | PASS | 独立 PostgreSQL 上运行 `TestFreshInstall_MCPReadPermissionMigration*`，空租户、存量幂等、补码失败三个用例均通过。 |
| Migration ledger 校验和 | PASS | `TestSupersededMigrations_AllowedChecksumMatchesRepositoryFile` 和两个 `TestPendingMigrations_*Superseded*` 用例通过。登记仅接受特定版本的已评审旧校验和与当前校验和。未来任意漂移仍被拒绝，启动不自动重写已应用账本。 |
| 空卷首装门禁 | FAIL | [合并首装修复后的空卷 CI](https://github.com/heidsoft/itsm/actions/runs/44894838965) 已越过 MCP 迁移，在配置项两对重复索引处失败。[当前 main 的独立重跑](https://github.com/heidsoft/itsm/actions/runs/44911926113)。 |
| 索引实施 | BLOCKED | 等待数据库变更的人工审批。 |

复验命令为 `go test -count=1 -v ./migration -run 'MCP|Supersed|UnreviewedDrift'`。真实数据库由一次性容器提供，测试 DSN 指向 `itsm_init_test_round_two`。

## 索引变更提案

`20261119_ci_owner_fk_and_certification.sql` 创建普通索引 `ci_tenant_owner_idx` 与 `ci_tenant_department_idx`。Ent 在 `ent/schema/configurationitem.go` 对相同两组字段另行定义索引。重复索引增加写入和存储开销，当前首装门禁将其视为失败。

1. 在空库及存量升级演练库只读采集 `pg_index`、`pg_class` 和 `pg_get_indexdef` 的结果，核对列顺序、谓词、包含列、唯一性、有效性和约束依赖。保留 Etl 或运维脚本依赖检查及基线计划。
2. 经审批后，新增独立版本的迁移，保留有效的 Ent 同等索引。仅在确认定义等价且没有约束依赖后处理重复的手写索引。较大的生产表采用适配迁移器的非事务并发操作，先验证锁等待、超时和重复运行。不修改已发布 SQL 或其 ledger。
3. 如需回滚，使用审批前保存的索引定义，在保留索引验证有效的前提下重建手写索引。并发重建失败产生的无效索引由下一次经审核的恢复操作处理。复核查询计划和空卷门禁后结束。
4. 范围只包括 `configuration_items` 的租户与所有者、租户与部门索引。认证时间索引、外键、业务数据和权限保持现有配置。验收包括空卷首装、模拟历史库升级、重复执行、回滚恢复以及归属查询计划对比。
