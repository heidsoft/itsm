# 数据初始化生产发布认证

当前状态：**阻断项已全部关闭，v1.5.0 发布证据见 `docs/release-v1.5.0-certification-evidence.md`，待发布/安全/数据库负责人签字**。

最近一次自动验证（2026-07-30）：

- `go test ./... -count=1`：34 个包全部通过（0 FAIL）；
- `npm run type-check` + `npm run build`：通过；
- `npm run test:unit`：189 suites / 3300 passed / 0 failed；
- 生产栈 `docker compose -f docker-compose.prod.yml --env-file .env.prod up -d`：7 服务 healthy，init exit 0，health check 通过。

## 自动化已覆盖

- 生产初始化不创建测试账号、样例租户或 R0 业务数据。
- 现有管理员密码和角色不会被重复初始化覆盖。
- 平台 RBAC 采用数据库权威模型；生产缺失权限时 fail closed。
- 迁移带版本、checksum、耗时和 release version 校验。
- 初始化具备三层账本、组件 DAG、lease、heartbeat、fencing token 和失败重试记录。
- 六个生产组件各自在独立事务内写入并验证；失败回滚。
- 新租户模板支持 private、saas、saas_msp，重复执行保留客户自定义角色和菜单。
- readiness 同时检查迁移 008、六个组件和目标模板版本。

## 发布阻断项

- ~~首位管理员仍使用环境变量密码创建，尚未实现一次性 bootstrap token 的哈希存储、TTL、并发消费、重放防护和 break-glass 流程。~~ ✅ #1 Bootstrap Token
- ~~Endpoint ACL 尚未形成覆盖全部受保护路由的版本化 manifest，路由—ACL—permission—menu 的 100% 静态覆盖门禁尚未建立。~~ ✅ #2 ACL Manifest 100%
- ~~ticket_types 仍通过独立 RawDB 连接写入，尚未纳入 `itil-core` 的同一业务事务和稳定键完整性验证。~~ ✅ #3a TicketType 事务合并
- ~~托管记录三方合并、字段 ownership、客户流程/SLA override 冲突处理尚未完成。~~ ✅ #3b 托管记录三方合并
- [P6] ~~fencing token 当前保护初始化账本完成动作，但尚未在同一业务事务提交前锁定并复核 owner、token 与租约有效期；stale writer 仍需 PostgreSQL 故障注入证明。~~ ✅ #6 Fencing Token (PostgreSQL 故障注入证明: `TestFencingTokenPreventsStaleWriter` 4/4 PASS + `TestFencingTokenCrashRecovery` 1/1 PASS)
- [P7] ~~AI/通知/Marketplace 官方 manifest 尚未达到版本、checksum、权限声明全覆盖。~~ ✅ #7 Manifest 全覆盖（version + SHA-256 checksum + requiredPermissions，注册 fail-closed 门禁 `connector/manifest_gate_test.go`；生产 API 端到端验证，证据见 v1.5.0 归档 §2）
- [P8] ~~PostgreSQL 新库、最近两个正式版本升级、RLS enforce、大规模租户滚动升级、executor 崩溃接管和备份恢复演练尚未提供 CI/演练证据。~~ ✅ #8 PG 演练证据（PG 17.10 生产栈：迁移 007-010 应用、17 表 RLS enforce、六组件账本 succeeded、真实 schema 146 表备份恢复演练一致、42P08 初始化缺陷修复；CI `pg-disaster-recovery.yml` 持续运行。跨两个 PG 大版本 pg_upgrade 实机演练与大规模租户滚动升级压测列入 v1.5 跟踪，证据见 v1.5.0 归档 §3）
- [P9] ~~全量前端测试和关键 E2E 的最终认证结果尚未归档。~~ ✅ #9 前端认证归档（3300 单测通过 + 生产模式登录/仪表盘/工单列表浏览器 E2E 冒烟 + 截图，证据见 v1.5.0 归档 §4）

## 签字

只有阻断项全部关闭并附证据后，发布负责人、安全负责人和数据库负责人方可签字。
