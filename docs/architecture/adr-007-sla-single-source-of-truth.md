# ADR-007：SLA 单一事实来源规范

> Status: proposed

## 状态

Proposed（2026-10-06）

## 背景

SLA 模块经历了从内嵌字段到独立表、从分散计算到统一引擎的演进，但当前仍处于三阶段迁移的中间态（Phase 1 双写），存在多套机制并行、数据双写不一致、配置碎片化等架构问题。

### 1. 两套并行 SLA 机制

```
机制 A — 传统工单/事件 SLA
  ├─ 配置：SLADefinition（sla_definitions 表）+ SLAPolicy（sla_policies 表）
  ├─ 运行时：Ticket.sla_* / Incident.sla_* 内嵌字段 + sla_states 表（双写）
  ├─ 计算：ticket_sla_service.go / sla_policy_service.go / sla_monitor_service.go
  └─ 监控：sla_monitor_service.go StartSLAWatcher（5 分钟轮询）

机制 B — BPMN 流程 SLA
  ├─ 配置：ProcessDefinition.sla_config JSONB + ProcessVariables["sla"] JSON
  ├─ 运行时：ProcessInstance / ProcessTask 实体（无 sla_states 关联）
  ├─ 计算：bpmn_sla_service.go calculateBusinessHoursDeadline（硬编码 Mon-Fri 9-18）
  └─ 监控：无独立 watcher，依赖流程任务超时
```

两套机制的配置来源、运行时存储、计算逻辑和监控方式完全独立。

### 2. 三阶段迁移中间态（Phase 1 双写）

```
Phase 1 — 双写（当前）
  ├─ Ticket/Incident 创建时：写入内嵌 sla_* 字段 + sla_states 表
  ├─ PauseSLA/ResumeSLA：同时更新内嵌字段和 sla_states
  ├─ 读取路径：Ticket/Incident 查询仍读内嵌字段（ticket_service.go:1705-1709）
  └─ 监控扫描：sla_monitor_service.go CheckSLAViolations 读内嵌字段

Phase 2 — 存量迁移（待实施）
  └─ 从内嵌字段回填 sla_states（尚无迁移脚本）

Phase 3 — 切读（待实施）
  ├─ 读取切到 sla_states
  └─ 移除内嵌字段（ticket.sla_* / incident.sla_*）
```

**证据**：
- `sla_monitor_service.go:975-978` — PauseSLA 双写 Ticket 内嵌字段
- `sla_monitor_service.go:1021-1024` — ResumeSLA 双写 Incident 内嵌字段
- `ticket_service.go:1705-1709` — GetTicketSLAInfo 优先读内嵌字段
- `dashboard_repository.go:127-188` — 统计查询直接读 `sla_response_deadline` / `sla_resolution_deadline` 内嵌列

### 3. 三套营业时间计算（已收敛但残留未清理）

```
实现 1 — ticket_sla_service.go
  └─ 独立 addBusinessMinutes 逻辑（已被 engine.go 替代，但代码仍在）

实现 2 — sla_policy_service.go CalculateSLAExpireTime
  └─ 独立营业时间计算（已被 engine.go 替代，但代码仍在）

实现 3 — bpmn_sla_service.go calculateBusinessHoursDeadline（:160）
  └─ 硬编码 Mon-Fri 9:00-18:00，不使用共享配置
  └─ 未接入统一引擎

统一引擎 — service/sla/engine.go（Phase 3 Step 3.4，2026-09-27）
  ├─ addBusinessMinutes：按天迭代消耗营业时间分钟，跳过非工作日/假日
  ├─ parseBusinessHoursConfig：向后兼容 key 别名（workdays/work_days 等）
  ├─ 24x7 模式旁路日历逻辑
  └─ 366 天循环守卫
```

**证据**：
- `engine.go:1-16` — 注释明确记录三阶段迁移与三套实现收敛
- `bpmn_sla_service.go:160` — 独立 `calculateBusinessHoursDeadline`，硬编码营业时间
- `20260916_sla_business_hours_key_normalize.sql` — 修复 business_hours JSON key 不一致

### 4. 配置碎片化

```
SLADefinition（sla_definitions）
  ├─ 角色：SLA 时间目标定义（响应/解决时间 + 营业时间 + 升级规则）
  ├─ 匹配维度：service_type + priority
  └─ 被 Ticket/Incident 通过 sla_definition_id 引用

SLAPolicy（sla_policies）
  ├─ 角色：多维度策略匹配（customer_tier + ticket_type + priority）
  ├─ 匹配逻辑：特异性评分（3/2/1/0），回退到最低 priority_score
  └─ 可选关联 SLADefinition

BPMN sla_config（process_definitions.sla_config JSONB）
  ├─ 角色：流程级 SLA 配置
  ├─ 字段：response_time_hours, resolution_time_hours, business_hours_only, priority_overrides
  └─ 与 SLADefinition 完全独立

SLA 模板（sla_template_service.go）
  ├─ 6 个预置模板（incident P1/P2/P3, change normal/emergency, service_request standard）
  └─ InstallTemplate 创建 SLADefinition 记录
```

三套配置来源（SLADefinition、SLAPolicy、BPMN sla_config）之间没有统一的解析入口。

### 5. 双层 Service 架构

```
service/ 层（旧）
  ├─ ticket_sla_service.go（752 行）— 工单 SLA 计算与查询
  ├─ sla_monitor_service.go（1317 行）— 监控、违规检测、仪表盘
  ├─ sla_policy_service.go（431 行）— 策略匹配
  ├─ sla_alert_service.go（725 行）— 告警规则与冷却
  ├─ sla_template_service.go（344 行）— 模板安装
  └─ bpmn_sla_service.go（415 行）— BPMN 流程 SLA

handlers/sla/ 层（新，领域切片）
  ├─ entity.go（203 行）— 领域类型
  ├─ repository.go + repository_impl.go（1140 行）— 数据访问
  ├─ service.go（427 行）— 业务逻辑
  └─ handler.go（667 行）— HTTP 处理
```

两层都提供 SLA 查询能力，`handlers/sla/` 的 `loadSLACohort()` 与 `service/sla_monitor_service.go` 的 `GetDashboardMetrics()` 存在功能重叠。

### 6. 前端类型不匹配

```
sla-api.ts（527 行）
  └─ 实际使用的 API 客户端，与后端 DTO 对齐

sla-service.ts（403 行）
  └─ 更丰富的类型系统（SLAStatus/SLAType/SLAPriority 枚举、EscalationLevel）
  └─ 部分类型与后端 API 不匹配，可能是 aspirational 或已过期
```

## 决策

### D1：sla_states 为运行时 SLA 唯一真相源

**决策**：完成 Phase 2（存量回填）与 Phase 3（切读 + 移除内嵌字段），使 `sla_states` 表成为所有聚合根（ticket/incident/change/problem/service_request）SLA 运行时状态的唯一真相源。

**理由**：
- 当前双写增加了一致性风险（内嵌字段与 sla_states 不同步）
- `sla_states` 的多态设计（aggregate_type + aggregate_id）天然支持五类聚合根
- 唯一索引 `(tenant_id, aggregate_type, aggregate_id)` 保证每个聚合根只有一条 SLA 状态

**实施路径**：
1. Phase 2：编写迁移脚本，从 `tickets.sla_*` 和 `incidents.sla_*` 回填 `sla_states`
2. Phase 3a：将 `ticket_service.go:1705-1709`、`dashboard_repository.go:127-188` 等读取路径切到 `sla_states`
3. Phase 3b：移除 Ticket/Incident schema 中的 `sla_response_deadline`、`sla_resolution_deadline`、`sla_first_response_at`、`sla_resolved_at`、`sla_status`、`sla_paused_at`、`sla_pause_reason` 内嵌字段
4. 更新 `sla_monitor_service.go` 的 `CheckSLAViolations` 从 `sla_states` 扫描而非内嵌字段

**约束**：Phase 3b 的 schema 字段移除属于破坏性迁移，必须在 CHANGELOG 和 UPGRADE.md 中明确记录。

### D2：统一引擎为营业时间计算唯一实现

**决策**：`service/sla/engine.go` 的 `Engine.ComputeDeadlines()` 与 `addBusinessMinutes()` 为营业时间计算的唯一实现。所有其他营业时间计算代码必须删除或委托到统一引擎。

**理由**：
- 三套独立实现（ticket_sla_service、sla_policy_service、bpmn_sla_service）在 key 解析、假日处理、时区逻辑上存在微妙差异
- `20260916_sla_business_hours_key_normalize.sql` 迁移已证明 key 不一致曾导致 24x7 模板退化为默认 9-18
- 统一引擎提供向后兼容的 key 别名解析，消除了数据层修复后计算层仍不一致的风险

**实施路径**：
1. `bpmn_sla_service.go:160` 的 `calculateBusinessHoursDeadline` 改为调用 `engine.ComputeDeadlines()`
2. 删除 `ticket_sla_service.go` 中残留的独立营业时间计算逻辑
3. 删除 `sla_policy_service.go` 的 `CalculateSLAExpireTime` 中的独立计算逻辑
4. 添加 CI 守卫：禁止在 `service/sla/` 之外出现 `addBusinessMinutes` / `calculateBusinessHoursDeadline` / `CalculateSLAExpireTime` 等函数定义

### D3：SLADefinition 与 SLAPolicy 职责分离

**决策**：明确 SLADefinition 为"时间目标定义"，SLAPolicy 为"策略匹配规则"。SLAPolicy 匹配后必须解析到 SLADefinition 获取时间目标，不得自带独立的响应/解决时间。

**理由**：
- 当前 SLAPolicy 自带 `response_time_minutes` 和 `resolution_time_minutes`，与 SLADefinition 的同名字段语义重叠
- 匹配链路不清晰：Ticket 创建时走 SLADefinition 直接匹配还是 SLAPolicy 匹配后关联 SLADefinition，取决于调用方
- 统一解析入口缺失

**实施路径**：
1. 明确 SLAPolicy 的 `response_time_minutes` / `resolution_time_minutes` 为覆盖值（override），仅在未关联 SLADefinition 时生效
2. 创建 `ResolveSLATarget(tenantID, ticketType, priority, customerTier)` 统一解析函数，返回 SLADefinition + 覆盖值
3. 所有 SLA 创建路径统一调用 `ResolveSLATarget`

### D4：BPMN SLA 收敛到统一引擎

**决策**：BPMN 流程 SLA 的截止时间计算必须使用统一引擎（`service/sla/engine.go`），流程 SLA 运行时状态必须写入 `sla_states` 表。

**理由**：
- `bpmn_sla_service.go` 的 `calculateBusinessHoursDeadline` 硬编码 Mon-Fri 9-18，不支持租户时区和自定义营业时间
- BPMN 流程任务无 SLA 状态跟踪，无法在仪表盘统一展示
- `process_definitions.sla_config` 与 `sla_definitions` 完全独立，管理员需要在两个地方配置 SLA

**实施路径**：
1. `bpmn_sla_service.go` 改为调用 `engine.ComputeDeadlines()`
2. 流程实例/任务创建时写入 `sla_states`（aggregate_type = "process_instance" / "process_task"）
3. 长期：`process_definitions.sla_config` 关联到 `sla_definitions`，避免双重配置

### D5：内嵌字段退役路径

**决策**：Ticket 和 Incident 的 SLA 内嵌字段（`sla_response_deadline`、`sla_resolution_deadline` 等）在 Phase 3 完成后必须移除。过渡期内，读取路径必须优先读 `sla_states`，内嵌字段只作为降级备份。

**理由**：
- 双写增加了写入复杂度和一致性风险
- `dashboard_repository.go` 的统计查询直接读内嵌列，绕过了 `sla_states` 的多态设计
- 内嵌字段不支持 change/problem/service_request 等新的聚合根类型

**退役字段清单**：
- `tickets.sla_definition_id`、`sla_response_deadline`、`sla_resolution_deadline`、`sla_status`、`sla_paused_at`、`sla_pause_reason`
- `incidents.sla_definition_id`、`sla_response_deadline`、`sla_resolution_deadline`、`sla_first_response_at`、`sla_resolved_at`、`sla_status`、`sla_paused_at`、`sla_pause_reason`

**约束**：退役前必须确认 `sla_states` 回填完整且所有读取路径已切换。退役迁移必须保留 `sla_definition_id` 外键列（用于历史关联），只退役运行时字段。

### D6：handlers/sla/ 与 service/ 层职责边界

**决策**：`handlers/sla/` 为 SLA 监控、报表、性能分析的唯一 API 层。`service/` 层的 SLA 服务只负责运行时计算与违规检测，不提供与 `handlers/sla/` 重叠的查询接口。

**理由**：
- `handlers/sla/repository_impl.go` 的 `loadSLACohort()` 与 `sla_monitor_service.go` 的 `GetDashboardMetrics()` 功能重叠
- 两层都提供仪表盘数据，但统计口径可能不同（cohort vs 全量扫描）
- 违反 AGENTS.md 的"一个用例只能有一个业务规则所有者"原则

**实施路径**：
1. `sla_monitor_service.go` 的 `GetDashboardMetrics()` 委托到 `handlers/sla/service.go`
2. `sla_monitor_service.go` 只保留 `StartSLAWatcher()`、`CheckSLAViolations()` 等运行时职责
3. 添加 CI 守卫：禁止 `service/` 层新增 SLA 查询/报表函数

### D7：前端类型与后端 DTO 对齐

**决策**：`sla-service.ts` 的丰富类型系统（SLAStatus/SLAType 枚举等）必须与后端 DTO 对齐。不对齐的类型必须删除或标记为 deprecated。

**理由**：
- `sla-api.ts` 是实际使用的 API 客户端，`sla-service.ts` 的部分类型与后端不匹配
- 违反 AGENTS.md 的"API 契约单一事实来源"原则
- 前端枚举值与后端字符串不匹配会导致运行时错误

**实施路径**：
1. 对比 `sla-service.ts` 类型与 `dto/sla_dto.go` / `handlers/sla/entity.go`
2. 删除或标记 deprecated 不匹配的类型
3. 统一使用 `sla-api.ts` 的类型定义

## 后果

### 正面

- **一致性**：sla_states 为唯一运行时真相源，消除双写不一致风险
- **可维护性**：统一引擎消除三套营业时间计算的维护负担
- **可扩展性**：多态 sla_states 天然支持新的聚合根类型
- **可观测性**：统一仪表盘数据源，消除两层查询口径差异

### 负面

- **迁移风险**：Phase 2 存量回填需要停机或在线迁移窗口
- **破坏性变更**：Phase 3b 移除内嵌字段属于破坏性 schema 变更
- **实施周期**：完整实施需要 3-4 个版本迭代

### 风险

- **回填完整性**：Phase 2 回填必须验证 100% 覆盖，遗漏的工单将丢失 SLA 状态
- **读取路径切换**：Phase 3a 必须逐个切换读取路径并验证，不能一次性切换
- **BPMN SLA 关联**：D4 要求流程实例写入 sla_states，但 BPMN 引擎的任务生命周期与 SLA 状态生命周期不完全对齐

## 参考

- `service/sla/engine.go` — 统一 SLA 计算引擎
- `service/sla/store.go` — 统一 SLA 持久化层
- `ent/schema/sla_state.go` — sla_states 多态 schema
- `service/sla_monitor_service.go` — SLA 监控与违规检测
- `service/bpmn_sla_service.go` — BPMN 流程 SLA
- `handlers/sla/` — SLA 领域切片 API 层
- `20260925_retire_legacy_sla_definitions.sql` — 旧代 SLA 定义退役迁移
- `20260916_sla_business_hours_key_normalize.sql` — 营业时间 key 规范化迁移
- `docs/testing/test-cases/TC-SLA.md` — 133 个 SLA 测试用例
