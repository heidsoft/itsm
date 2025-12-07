# 端到端回归测试（结合 PostgreSQL MCP）

- 日期: 2025-12-07
- 范围: itsm-backend（API+DB） × itsm-prototype（前端） × PostgreSQL（MCP）
- 目标: 以数据库真实结构与健康信息为依据，对关键业务链路进行端到端回归分析与建议

## 测试对象与数据基础
- 数据库模式：
  - 架构：public（用户业务），information_schema/pg_catalog（系统）
- 关键表：
  - tickets（工单）
  - incidents（事件）
  - users（用户）
- 表结构节选（MCP）：
  - tickets：`id,title,status,priority,ticket_number,requester_id,assignee_id,created_at,tenant_id` 等，索引含 `ticket_tenant_id_status`, `ticket_status_priority`, `tickets_ticket_number_key` 等
  - incidents：`id,title,status,priority,source,type,incident_number,is_major_incident,tenant_id,reporter_id,assignee_id,created_at` 等，索引含 `incident_tenant_id_status`, `incident_status_priority`, `incidents_incident_number_key` 等
  - users：`id,username,email,name,department,phone,password_hash,active,tenant_id,role,...`，索引含 `users_username_key`, `users_email_key`, `user_tenant_id_email` 等

## 端到端链路与典型查询
- Tickets 列表（过滤 + 排序 + 分页）：
  - SQL（示例）：`SELECT ... FROM tickets WHERE tenant_id=$1 AND status IN('open','in_progress') ORDER BY created_at DESC LIMIT $2;`
  - 执行计划：索引扫描 + 排序，受 `ticket_tenant_id_status`, `ticket_created_at` 影响（MCP EXPLAIN）
- Incidents 列表（过滤 + 排序 + 分页）：
  - SQL（示例）：`SELECT ... FROM incidents WHERE tenant_id=$1 AND status IN('new','in_progress') ORDER BY created_at DESC LIMIT $2;`
  - 执行计划：顺序扫描 + 排序；建议更多使用组合索引（如 `incident_tenant_id_status` + 覆盖 `created_at`）（MCP EXPLAIN）
- 计数统计（列表总数）：
  - Tickets：`SELECT COUNT(*) FROM tickets WHERE tenant_id=$1 AND status=$2;`
  - Incidents：`SELECT COUNT(*) FROM incidents WHERE tenant_id=$1 AND status=$2;`

## 数据库健康（MCP）
- 无无效索引、无膨胀索引（bloat）
- 存在大量“重复/被覆盖索引”（可审计清理）：
  - 示例：`tickets` 表的 `ticket_ticket_number` 被 `tickets_ticket_number_key` 覆盖；`ticket_status` 被 `ticket_status_priority` 覆盖
  - `incidents` 表的 `incident_status` 被 `incident_status_priority` 覆盖；`incident_source` 被 `incident_source_type` 覆盖
- 罕用索引：大量云厂商字段相关索引与安全事件索引扫描次数为 0（建议按真实负载保留/清理）
- 慢查询统计扩展未安装：
  - MCP 提示安装 `pg_stat_statements` 获取真实慢查询数据
- 假设索引评估扩展未安装：
  - MCP 提示安装 `hypopg` 辅助评估潜在索引收益

## 前端集成验证
- 统一 API 层使用（已改造）：
  - 事件列表 `IncidentManagement.tsx` 改为调用 `IncidentAPI.listIncidents`，通过 `http-client` 自动携带 `Authorization`/`X-Tenant-ID/Code` 与错误处理
- API 集成测试结果：
  - `src/app/lib/__tests__/api-integration.test.ts` 19/19 通过（CORS/网络/错误码/超时/鉴权/并发/大负载/数据结构）

## 端到端回归建议
- 数据库索引优化（按优先级）：
  - 清理重复/被覆盖索引：降低维护与写入开销（Tickets/Incidents 多个候选）
  - 建议安装扩展：
    - `CREATE EXTENSION pg_stat_statements;`（慢查询统计）
    - `CREATE EXTENSION hypopg;`（假设索引评估）
  - 针对列表查询优化排序：考虑 `tenant_id,status,created_at DESC` 组合索引（视负载而定；使用 `hypopg` 评估）
- 服务端 API 与查询优化：
  - 列表查询分页与过滤参数一致性（后端 DTO 与索引对齐）；避免 N+1，使用 Ent 预加载（With）
  - 计数统计与列表查询拆分优化；必要时启用近似统计或缓存
- 前端集成与一致性：
  - 全量迁移组件内直接 `fetch/localStorage` 到 API 层封装；统一筛选/分页与 Loading/Empty/Error 模板
  - 列表页 URL 同步（分享/复现）；A11y 键盘与焦点统一

## 验收与监控
- 验收
  - Tickets/Incidents 列表在租户与权限隔离下正确返回；分页与排序一致；统计数据与列表一致
  - 前端错误提示与重试可用，空态与加载模板统一；集成测试通过率 ≥ 80%
- 监控
  - 启用 `pg_stat_statements` 后观察资源排名前列的查询；根据真实负载调整索引
  - 前端与后端错误日志与审计一致，可用于回溯与性能诊断

## 附录：MCP输出摘要
- 架构：public 可用；关键表存在完备的索引集合（部分重复/罕用）
- EXPLAIN：
  - Tickets：Index Scan + Sort，过滤 `status IN (...)`
  - Incidents：Seq Scan + Sort，过滤 `status IN (...) AND tenant_id = 1`
- 扩展建议：pg_stat_statements / hypopg 未安装

