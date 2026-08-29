# 生产就绪评估修复日志(2026-08-29)

> 对应报告:`reports/prod-readiness/2026-08-29-production-readiness-assessment.md`(NO-GO,4 项 P0)
> 状态图例:✅代码已修并测试 🚧待部署验证 🔴未开始

## P0 修复

| # | 问题 | 根因 | 修复 | 状态 |
|---|---|---|---|---|
| P0-1 | `POST /changes/:id/approve` 500 "failed to get approval history" | `GetApprovalHistory` 使用 `string_agg(integer, ',')`;PG 聚合参数不做隐式 int→text 转换,查询必失败;且 `TransitionStatus` 吞掉根因错误 | `handlers/change/repository_impl.go`:levels 改独立查询 + Go 侧拼接(双方言安全);`service.go` 记录根因日志;回归测试 `repository_approval_history_test.go`(生产库复现+验证通过) | ✅🚧 |
| P0-2a | 仪表盘指标失真(待处理 4/31、完成 0/2、超时按假状态计数) | `dashboard_service.go` 使用不存在状态 `"submitted"`、完成口径只计 `"closed"` | 引入统一口径 `ticketPendingStatuses`/`ticketCompletedStatuses`(对齐 `common.TicketStatus*`);超时工单改为真实 SLA 语义(未响应过期限/未解决过期限) | ✅🚧 |
| P0-2b | SLA 合规率 8.1%、平均响应/解决时间 0 | 演示数据多数工单从未被响应(`first_response_at`/`resolved_at` 为空),违规判定本身正确;指标 0 由 P0-2a 的假口径放大 | 口径修复后指标将反映真实值;建议按报告准备 ≥100 工单数据集复测 | ✅🚧(复测待部署) |
| P0-3 | RLS 未启用(`RLS_MODE=off`) | enforce 模式把每条语句包进短事务,而 ent 的 Query 把 Rows 句柄交回调用方稍后扫描——事务提前提交导致 "sql: Rows are closed",启动探针崩溃 | `database/rls/driver.go`:改用 ent 原生 `sql.WithVar` 会话变量(专用连接语句级 SET/RESET),不再开短事务;显式事务路径保留 SET LOCAL;回归测试锁定"不开短事务+变量注入" | ✅🚧(部署先切 shadow 观察,再 enforce + 启用策略) |
| P0-4 | README 密码与实际不符 | 文档只写开发默认值,未说明生产密码来源 | 四语种 README 补充"生产管理员密码由 `.env.prod` 的 `ADMIN_PASSWORD` 决定,勿用示例密码" | ✅ |

## 部署顺序(单次重建)

1. 等待并行会话的 ent schema 变更(ticket.ci_id CMDB 本体链路)生成完成并全量构建通过
2. `./scripts/deploy-prod.sh deploy`(脚本含备份与验证)
3. `RLS_MODE` 先置 `shadow`,验证启动与探针正常、观察 "query without tenant scope" 告警
4. 无异常后切 `enforce` 并应用 `002_pilot_policies.sql`(changes + vectors),复测跨租户拒绝与 `/changes/:id/approve`、仪表盘指标

## 报告中的其余问题(未在本批处理)

- P1-1 契约违规(12 个列表接口 `items` 化):独立契约迁移批次,需前后端同步,避免单方面破坏前端
- P1-3 `/workflows` 兼容接口 400→301/302;P1-6 BPMN 节点分析菜单 404
- P1-4 业务错误码 4xx/5xx 区分与 i18n
- P1-5 Worker 日志为空:部署后复核 `ITSM_PROCESS_MODE=worker` 日志输出
- P2-*:URL 别名、跨租户拒绝码 403 化、连接器启用
