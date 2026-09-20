# BPMN Timer Event 调度基础设施设计大纲

> Status: current

**文档编号**: ITSM-PRD-2026-002
**版本**: v0.5
**日期**: 2026-09-15
**状态**: Phase 1–5 全部交付（Timer Event 功能闭环）；遗留项 = 用户文档、non-interrupting Boundary（P2）、分布式锁 fencing（P2）

---

## 1. 背景与动机

### 1.1 现状

系统当前有五套独立的"时间驱动"机制，各自为政：

| 机制 | 实现方式 | 触发粒度 | 持久化 | 崩溃恢复 |
|------|---------|---------|--------|---------|
| SLA 违规扫描 | `sla_monitor_service.go` 后台 ticker | 1 分钟 | 无 | 无（重新扫描即可） |
| 升级调度 | `escalation_service.go` 后台 ticker | 5 分钟 | 无 | 无 |
| 事件升级调度 | `incident_escalation_service.go` 后台 ticker | 5 分钟 | 无 | 无 |
| 任务超时扫描 | `bpmn_timeout_scanner.go` 后台 ticker | 2 分钟 | outbox 通知 | 无 |
| 工作流自动升级 | `workflow_automation_service.go` 后台 ticker | 内部循环 | 无 | 无 |

此外 `bpmn_event_service.go` 定义了完整的事件类型体系（`TriggerTimer`、`EventBoundary`、`EventIntermediate`），但所有方法均为 stub（直接返回空数据）。

### 1.2 缺失能力

BPMN 2.0 规范定义了三种 Timer Event，当前均不支持：

| Timer Event 类型 | BPMN 语义 | 产品场景 |
|-----------------|----------|---------|
| **Timer Start Event** | 按 cron/ISO 8601 周期自动启动流程 | 每日巡检工单自动生成、周报自动提交 |
| **Timer Intermediate Catch Event** | 流程执行到该节点时等待指定时间后继续 | 审批后等待 3 个工作日自动归档、工单创建后 24h 无响应自动升级 |
| **Timer Boundary Event** | 附加在活动上，活动超时后中断并走异常路径 | 处理人 4h 未接单自动转派、实施超期自动触发 PIR |

### 1.3 与 TimeoutScanner 的区别

TimeoutScanner 是"任务级被动扫描"——只关注 `due_date < now` 的活跃任务，动作是改状态 + 发通知，**不推进 BPMN 流程**。

Timer Event 是"流程级主动调度"——定时器到期后需要**唤醒流程引擎**，让流程从等待节点继续执行到下一个 gateway/task。这是本质区别。

---

## 2. 产品场景优先级

建议按业务价值 / 实现复杂度排序，分三批落地：

### P0 — 第一批（v1.7）

| 场景 | Timer 类型 | 说明 |
|------|-----------|------|
| 审批超时自动升级 | Boundary Timer | 与 TimeoutScanner 的 `escalate` 动作合并，但需要推进流程而非仅改状态 |
| SLA 响应/解决倒计时 | Boundary Timer | 替代当前独立的 SLA 扫描器，统一到 Timer Event 基础设施 |

### P1 — 第二批（v1.8）

| 场景 | Timer 类型 | 说明 |
|------|-----------|------|
| 工单自动归档 | Intermediate Timer | 审批完成后等待 N 天无异议自动归档 |
| 定期巡检工单生成 | Start Timer | 按 cron 表达式每天/每周自动生成标准变更或巡检工单 |

### P2 — 第三批（v2.0）

| 场景 | Timer 类型 | 说明 |
|------|-----------|------|
| 事件子流程定时器 | Boundary Timer (non-interrupting) | 不中断主流程，并行触发提醒 |
| 多级超时链 | Boundary Timer 级联 | 一级超时升级 → 二级超时自动拒绝 → 三级超时通知管理员 |

---

## 3. 技术架构方案

### 3.1 方案对比

| 维度 | 方案 A：DB 轮询 | 方案 B：内存调度 + DB 持久化 | 方案 C：外部调度引擎 |
|------|---------------|--------------------------|-------------------|
| 实现 | 扩展 TimeoutScanner 模式，新增 `process_timer` 表 | `robfig/cron` + DB 持久化 timer 记录 | 引入 Temporal/Asynq 等外部调度器 |
| 精度 | 取决于轮询间隔（分钟级） | 秒级（进程内 `time.AfterFunc`） | 毫秒级 |
| 崩溃恢复 | 天然支持（重启后重新扫描） | 需要 recovery：启动时扫描 DB 中未触发的 timer | 外部引擎自带 |
| 多副本 | 需要分布式锁或 leader 选举 | 单副本模式，多副本需协调 | 天然支持 |
| 依赖 | 无新依赖 | `robfig/cron`（轻量） | 新增 Redis/Temporal 依赖 |
| 与现有引擎集成 | 直接调用 `CustomProcessEngine` | 需要 adapter 层 | 需要完整集成层 |
| 适合规模 | 单租户 / <1000 timer | 单副本 / <10000 timer | 多租户 / 任意规模 |

### 3.2 推荐方案：B（内存调度 + DB 持久化）

**部署前提**：当前为本地单副本 Docker Compose 部署。方案 B 的 In-Memory Wheel 依赖单进程假设，在多副本场景下需要升级。

理由：
1. 当前部署模式为单副本 Docker Compose，方案 C 引入的外部依赖过早
2. 方案 B 正常路径精度为秒级（`time.AfterFunc`），崩溃恢复路径退化为分钟级（DB 扫描）——崩溃是低频事件，退化精度完全可接受
3. 方案 B 在崩溃恢复上通过"启动时 DB 扫描 + 重调度"即可覆盖，不需要分布式协调
4. 与现有 TimeoutScanner、SLA Monitor 的 polling 模式可以渐进合并

**多副本升级路径**（K8s 多 Pod 部署时）：

| 升级方案 | 思路 | 适用时机 |
|---------|------|---------|
| DB 轮询 + 短间隔（10-15s） | 回到方案 A 模式，CAS 更新天然幂等，多副本安全 | 最务实选择，SLA 分钟级精度够用 |
| K8s Leader Election | 只有 leader Pod 跑 scheduler，其余只处理请求 | 不引入新中间件，但 leader 切换有 5-15s 间隙 |
| Redis Sorted Set | ZSET 做共享 time wheel，`fire_at` 作为 score，Pod 竞争消费 | 需要秒级精度时引入，但新增 Redis 强依赖 |

Phase 1 按单副本方案 B 实现，在 Scheduler 层预留 `TimerStore` 接口抽象，未来切换存储后端（内存 → DB polling → Redis）时不改上层逻辑。

### 3.3 核心组件

```
┌─────────────────────────────────────────────────┐
│                BPMN Process Engine               │
│  ┌─────────────┐  ┌──────────────────────────┐  │
│  │ CustomProcess│  │ TimerEventSubscription   │  │
│  │ Engine       │←─│ Manager                  │  │
│  │ (流程推进)   │  │ (定时器订阅管理)          │  │
│  └─────────────┘  └──────────┬───────────────┘  │
│                              │                   │
│  ┌───────────────────────────┴───────────────┐  │
│  │         Timer Scheduler                    │  │
│  │  ┌─────────┐  ┌──────────┐  ┌──────────┐ │  │
│  │  │ DB Store│  │ In-Memory│  │ Recovery │ │  │
│  │  │ (持久化) │  │ Wheel    │  │ Scanner  │ │  │
│  │  └─────────┘  └──────────┘  └──────────┘ │  │
│  └───────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
```

#### 组件职责

| 组件 | 职责 |
|------|------|
| **Timer Scheduler** | 管理进程内定时器生命周期：注册、触发、取消、恢复 |
| **DB Store** | `process_timer` 表持久化所有待触发 timer，支持崩溃后重建 |
| **In-Memory Wheel** | 基于 `time.AfterFunc` 或 time-wheel 的高精度触发 |
| **Recovery Scanner** | 进程启动时扫描 DB 中 `status=pending AND fire_at < now` 的 timer，补触发或重调度 |
| **TimerEventSubscription Manager** | 与流程引擎交互：timer 到期时调用 `CustomProcessEngine` 推进流程到下一节点 |

---

## 4. 数据模型

### 4.1 `process_timer` 表（新增 Ent Schema）

```go
// 核心字段
field.String("timer_id").Unique().NotEmpty()           // 全局唯一 ID
field.String("timer_type").NotEmpty()                   // start / intermediate / boundary
field.String("process_definition_key").NotEmpty()       // 流程定义 key
field.Int("process_instance_id").Optional()             // 流程实例 ID（start timer 无此字段）
field.String("activity_id").Optional()                  // 绑定的活动 ID（boundary timer 必填）
field.String("timer_expression").NotEmpty()             // ISO 8601 duration 或 cron 表达式，支持 ${variable} 占位符
field.String("expression_type").NotEmpty()              // duration / cron / date
field.Time("fire_at")                                   // 计划触发时间（绝对时间）
field.Time("fired_at").Optional()                       // 实际触发时间（审计用）
field.String("status")                                  // pending / fired / cancelled / failed
field.Int("version").Default(1)                         // 乐观锁，状态流转时 CAS 保护
field.String("idempotency_key").Unique().NotEmpty()     // 幂等 key = timer_id + fire_at + tenant_id
field.Int("retry_count").Default(0)                     // 已重试次数
field.Int("max_retries").Default(3)                     // 最大重试次数
field.Time("last_fire_attempt").Optional()              // 上次触发尝试时间
field.String("failure_reason").Optional()               // 失败原因
field.JSON("context_variables", map[string]interface{}{}) // 触发时需要注入流程的变量

// SLA 暂停/恢复字段（Cancel + Recreate 模式）
field.Float("total_duration_seconds").Optional()        // 总时长（从 timer_expression 解析）
field.Float("elapsed_seconds").Default(0)               // 已消耗时间（暂停时计算）
field.String("pause_state").Default("running")          // running / paused
field.Int("parent_timer_id").Optional()                 // 恢复时创建的新 timer 指向原始 timer

field.Int("tenant_id").Positive()
field.Time("created_at").Default(time.Now)
field.Time("updated_at").Default(time.Now)
```

### 4.2 索引

```go
index.Fields("status", "fire_at")          // Recovery Scanner 核心查询
index.Fields("tenant_id", "status")        // 租户隔离
index.Fields("process_instance_id")        // 按流程实例查 timer
index.Fields("timer_id").Unique()          // 幂等
index.Fields("idempotency_key").Unique()   // 幂等触发保护
index.Fields("pause_state", "status")      // 暂停 timer 查询
```

### 4.3 状态机

```
         ┌──────────┐
  创建 → │ pending  │ ← 重调度（retry）
         └────┬─────┘    ↑
              │           │ 恢复（创建新 timer，remaining = total - elapsed）
              │           │
              ▼           │
         ┌──────────┐     │
         │  fired   │     │
         └────┬─────┘     │
              │           │
        ┌─────┴─────┐     │
        ▼           ▼     │
   流程推进成功   流程推进失败
        │           │     │
        ▼           ▼     │
   (删除/归档)  ┌────────┐ │
               │ failed │→┘ (retry_count < max_retries)
               └────────┘

   暂停：pending → cancelled + 记录 elapsed_seconds
   恢复：创建新 pending timer（parent_timer_id 指向原始 timer）

   任意时刻：流程实例取消 → timer 状态 → cancelled
```

---

## 5. 关键流程

### 5.1 Timer 注册（流程引擎 → Scheduler）

当 `CustomProcessEngine` 执行到 Timer Intermediate Catch Event 或挂载 Boundary Timer Event 时：

1. 解析 BPMN XML 中的 `timerEventDefinition`（`timeDuration` / `timeDate` / `timeCycle`）
2. 解析 `timer_expression` 中的 `${variable}` 占位符，从流程变量中替换并计算绝对 `fire_at` 时间
3. 写入 `process_timer` 表（status=pending），包含 `idempotency_key`、`version`、`total_duration_seconds`
4. **事务提交后**注册到 In-Memory Wheel（`time.AfterFunc(fire_at - now, callback)`）

> 注意：步骤 3 是 DB 事务，步骤 4 是进程内内存操作，两者不在同一事务中。如果事务提交成功但内存注册失败（理论上不会发生），Recovery Scanner 会在启动时补注册。

### 5.2 Timer 触发（Scheduler → 流程引擎）

In-Memory Wheel 回调触发时：

1. 从 DB 加载 timer 记录，校验 status=pending（fencing）
2. CAS 更新 status → fired（`WHERE id AND status='pending' AND version=?`，同时 `version=version+1`，`fired_at=now()`）
3. 根据 timer_type 调用流程引擎：
   - **intermediate**: 唤醒等待该 timer 的流程实例，继续执行后续节点
   - **boundary (interrupting)**: 取消当前活动，走异常处理路径
   - **start**: 创建新的流程实例
4. 若步骤 3 失败，status → failed，retry_count++，若 < max_retries 则重新计算 fire_at 并回退到 pending

### 5.3 崩溃恢复（Recovery Scanner）

进程启动时：

1. 扫描 `status=pending AND fire_at <= now` 的 timer → 立即触发
2. 扫描 `status=pending AND fire_at > now` 的 timer → 重新注册到 In-Memory Wheel
3. 扫描 `status=fired AND updated_at < now - 5min` 的 timer → 可能触发后崩溃，重试或标记 failed
4. 扫描 `status=failed AND retry_count < max_retries` → 重新调度

### 5.4 取消（流程实例终止时）

流程实例取消或完成时，级联取消所有关联的 pending timer：

```sql
UPDATE process_timer SET status = 'cancelled', version = version + 1
WHERE process_instance_id = ? AND status = 'pending' AND tenant_id = ?
```

### 5.5 暂停与恢复（SLA Cancel + Recreate 模式）

**暂停**（如：工单状态变为"等待客户回复"）：

1. 计算已消耗时间：`elapsed = now - (last_resume_time or created_at)`
2. 更新 timer：`status = 'cancelled'`，`pause_state = 'paused'`，`elapsed_seconds = elapsed`，`version = version + 1`
3. 从 In-Memory Wheel 中取消该 timer

**恢复**（如：客户回复后工单状态变回"处理中"）：

1. 计算剩余时间：`remaining = total_duration_seconds - elapsed_seconds`
2. 创建新 timer：`fire_at = now + remaining`，`parent_timer_id = 原始 timer ID`，`elapsed_seconds = 0`，`pause_state = 'running'`
3. 注册到 In-Memory Wheel

**审计链**：通过 `parent_timer_id` 可以追溯完整的暂停/恢复历史。每次 cancel 和 recreate 都记录审计日志，包含操作人、原因和 elapsed 时间。

**P0 覆盖场景**：显式暂停/恢复（等待客户回复、等待第三方响应）。
**P2 考虑**：Working Hours Calendar（只计算工作时间），需要 `sla_calendar` 表支持业务时段和节假日配置。

---

## 6. 与现有系统的合并路径

### 6.1 TimeoutScanner → Timer Event

当前 TimeoutScanner 的 `notify` 和 `escalate` 动作仅改状态 + 发通知。升级为 Timer Event 后：

- `escalate` 动作改为 Boundary Timer Event，到期后**推进流程**到升级路径
- `notify` 动作保留为轻量级任务超时处理（不改流程走向），仍由 TimeoutScanner 负责
- 新增配置项：流程设计器中可选择"超时后走 Timer Event 路径"

### 6.2 SLA Monitor → Timer Event

SLA 违规检测可以建模为 Boundary Timer Event：

- 工单创建时，根据 SLA 策略计算 `fire_at`，注册 Timer
- Timer 到期触发违规动作（通知、升级、自动关单）

**SLA 暂停/恢复（P0 — Cancel + Recreate 模式）**：

- 工单状态变为"等待客户回复"时 → 暂停：取消 timer，记录 `elapsed_seconds`
- 客户回复后 → 恢复：用 `remaining = total - elapsed` 创建新 timer
- 审计链通过 `parent_timer_id` 追溯

**Working Hours Calendar（P2）**：

- "只计算工作时间"（如 9:00-18:00、排除周末和节假日）需要 `sla_calendar` 表
- 注册 timer 时用 `addWorkingTime(now, duration, calendar)` 计算 `fire_at`
- 这是独立于暂停/恢复的另一个维度，P0/P1 不需要

### 6.3 Escalation Service → Timer Event

升级链可以建模为多个级联 Boundary Timer：

- 一级：30 分钟未响应 → 通知处理人
- 二级：1 小时未响应 → 通知主管
- 三级：2 小时未响应 → 自动分配

每个升级级别是一个独立的 Timer，在前一个 Timer 触发时注册下一个。

---

## 7. 管理 API

Timer 是"沉默的基础设施"，需要运维管理接口支撑排查和干预：

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/v1/timers` | GET | 按租户/状态/流程实例查询 timer 列表 |
| `/api/v1/timers/:id` | GET | 查询单个 timer 详情（含暂停/恢复历史） |
| `/api/v1/timers/:id/cancel` | POST | 手动取消 timer |
| `/api/v1/timers/:id/reschedule` | POST | 手动重调度（修改 fire_at） |
| `/api/v1/timers/:id/pause` | POST | 手动暂停 timer |
| `/api/v1/timers/:id/resume` | POST | 手动恢复 timer |
| `/api/v1/timers/stats` | GET | timer 统计（待触发/已触发/失败/取消/暂停） |

所有接口必须包含 `tenant_id` 校验，管理操作记录审计日志。

Phase 1 包含查询和统计接口，写操作接口（cancel/reschedule/pause/resume）可在 Phase 2 补齐。

---

## 8. 可观测性

### 8.1 Metrics

| Metric | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `timer_fired_total` | Counter | type, status, tenant_id | timer 触发计数 |
| `timer_fire_latency_seconds` | Histogram | type, tenant_id | fire_at 到实际触发的延迟 |
| `timer_recovery_total` | Counter | tenant_id | 恢复扫描触发次数 |
| `timer_retry_total` | Counter | type, tenant_id | 重试计数 |
| `timer_paused_total` | Counter | tenant_id | 暂停/恢复计数 |

### 8.2 告警规则

| 规则 | 条件 | 严重度 |
|------|------|--------|
| Timer 触发延迟过高 | `timer_fire_latency > 5min` 持续 3 分钟 | Warning |
| 失败 timer 堆积 | `failed timer count > 10` 持续 5 分钟 | Critical |
| 恢复扫描频繁触发 | `timer_recovery_total > 3/hour` | Warning |

### 8.3 审计日志

每次 timer 触发/重试/取消/暂停/恢复都记录审计记录，包含：
- `timer_id`、`timer_type`
- 原始 `fire_at`、实际 `fired_at`、延迟秒数
- `tenant_id`、`process_instance_id`
- 操作人（系统/手动）

---

## 9. 流程设计器集成

### 9.1 前端变更

在 `WorkflowNodeInspector.tsx` 中新增 Timer 配置面板：

| 节点类型 | Timer 配置 |
|---------|-----------|
| 中间捕获事件 | Timer 表达式输入（duration / date / cron） |
| User Task | 边界 Timer（可选）：超时时长 + 超时动作（升级/转派/自动完成） |
| 开始事件 | 可选切换为 Timer Start：cron 表达式 |

### 9.2 BPMN XML 扩展

利用现有 `itsm:` 命名空间扩展：

```xml
<bpmn:intermediateCatchEvent id="TimerWait" name="等待3天">
  <bpmn:timerEventDefinition>
    <bpmn:timeDuration>PT72H</bpmn:timeDuration>
  </bpmn:timerEventDefinition>
</bpmn:intermediateCatchEvent>

<bpmn:userTask id="HandleTask" name="处理工单">
  <bpmn:boundaryEvent id="TimeoutBoundary" attachedToRef="HandleTask">
    <bpmn:timerEventDefinition>
      <bpmn:timeDuration>PT4H</bpmn:timeDuration>
    </bpmn:timerEventDefinition>
  </bpmn:boundaryEvent>
</bpmn:userTask>
```

---

## 10. 工作量估算

> **状态（2026-09-16 supersede 注）**：Phase 1–5 全部 ✅ 已完成（Timer Event 功能闭环）。
> 累计交付 commit 序列见 `output/product-canonical-2026-09-16.md` §二。
> 唯一遗留项 = **用户文档未写**（Phase 5 测试+文档行已标 🟡）；代码事实反查见下方状态列。
> 按 09-12 文档治理规范，本表原文保留（不回改历史背景），仅顶部加状态 banner。

| 阶段 | 内容 | 估算 | 状态 |
|------|------|------|------|
| **Phase 1: 基础设施** | `process_timer` schema + Timer Scheduler + Recovery Scanner + DB Store + 查询/统计 API + Metrics | 4-5 天 | **✅ 已完成（2026-09-15，commit 89dad5b1 + metrics 补齐）**——TimerStore/Scheduler/Recovery/管理 API 落地；五项 Prometheus metrics 已实现（itsm_timer_fired_total/fire_latency/recovery/retry/paused，2026-09-15） |
| **Phase 1.5: Spike** | 验证 `lib-bpmn-engine` Timer Event 支持能力 | 1 天 | **✅ 已完成** — 见附录 C |
| **Phase 2: 引擎集成** | CustomProcessEngine 支持 Timer Intermediate / Boundary 节点的注册与触发 + SLA 暂停/恢复 | 3-5 天 | **✅ 已完成（2026-09-15）**——intermediate/boundary/start 注册触发 ✅（89dad5b1）；SLA 暂停/恢复 Cancel+Recreate ✅（挂起取消/恢复重建/终止取消，4 例 E2E） |
| **Phase 3: 设计器集成** | WorkflowNodeInspector Timer 配置面板 + BPMN XML 序列化/反序列化 | 2-3 天 | **✅ 已完成（2026-09-15，commit 3146bade）**——extractor ✅ + 前端 Timer 配置面板（start/intermediate/boundary）✅ + XML 规范化写入 ✅ + lint 规则组 1.1 ✅ |
| **Phase 4: 系统合并** | TimeoutScanner escalate 路径迁移 + SLA Monitor Timer 化 | 2-3 天 | **🟡 核心完成（2026-09-15）**——BPMN dueDate 落库（修复零写入休眠循环）+ task_due 定时器（注册/到期分发/完成取消）+ TimeoutScanner 降级恢复兜底（claim-once 双路径安全）；**SLA Monitor 保持轮询**（聚合策略评估不适合逐票 timer，决策见 §12） |
| **Phase 5: Timer Start** | 定时启动流程 + cron 表达式解析 + 产品模板 | 2 天 | **✅ 已完成（2026-09-15）**——`timer_cron.go`（表达式四分类 + cron 按 `tenants.timezone` 求 `Next`、fire_at 统一 UTC 落库）+ `timer_start_schedule.go`（部署与设计器发布共用，整体替换语义；停用同步取消）+ `handleStartTimer`（businessKey 二次启动幂等 + cron/cycle 重排）+ 产品模板 `scheduled_inspection_flow.bpmn`。**修复两处真实缺陷**：cron 被 `CycleRemaining` 误判为一次性导致时间表触发一次即终止；`bpmnProcessDefinitionService` 发布/停用从未同步 start timer（主路径静默不生效）。**边界**：仅 P0 可见不可跳过；非法时区名静默回退默认时区 |
| **测试 + 文档** | 单元测试 + 集成测试 + CHANGELOG + 用户文档 | 2-3 天 | **🟡 部分完成**——29 例 E2E + Phase 5 新增 18 例单测 + 单测全绿、17 模板门禁通过、CHANGELOG 已补（2026-09-15）；**用户文档未写（遗留项）** |

### 10.1 遗留项追踪（2026-09-16）

| 遗留 | 原因 | 建议 |
|------|------|------|
| **Timer Event 用户文档未写** | Timer Event 设计复杂（含四类事件/五种表达式/cron 时区/整体替换语义），开发优先级压过文档；产品模板 `scheduled_inspection_flow.bpmn` 已落地作参考 | 下一轮 P1 立项 1 天工作量 |
| SLA Monitor Timer 化（Phase 4 标 🟡） | CheckSLAViolations 是聚合策略评估（多级阈值/批量工单），逐票 timer 带来注册风暴与重建复杂度，收益不成比例 | 不实施（决策见 §11.2 第 4 条） |

> **Phase 1.5 Spike 结论（2026-09-13）**：`lib-bpmn-engine v0.2.4` 仅支持 `IntermediateCatchEvent > timeDuration` 的被动轮询模型，无 Boundary Timer、Timer Start、`timeDate`、`timeCycle` 支持，且无 timer 回调注册 API。`CustomProcessEngine` 完全绕过该库（自有解析器 + DB 状态机），因此 **不需要 fork 引擎**——直接在 CustomProcessEngine 层自建 timer → 流程推进桥接。Phase 2 风险从"高"降为"中"，估算从 4-6 天降为 3-5 天。详见附录 C。

---

## 11. 风险与开放问题

### 11.1 技术风险

| 风险 | 影响 | 缓解 | 状态 |
|------|------|------|------|
| ~~`lib-bpmn-engine` v0.2.4 不原生支持 Timer Event 回调~~ | ~~需要在 CustomProcessEngine 层自行实现 timer → 流程推进的桥接~~ | ~~评估升级到更新版本或 fork 修改~~ | **已解决** — Spike 确认：库仅支持 `IntermediateCatchEvent > timeDuration` 被动轮询，无 Boundary Timer / Timer Start / 回调 API。`CustomProcessEngine` 完全绕过该库（自有解析器 + DB 状态机），直接在 Custom 层自建即可，无需 fork。详见附录 C |
| 多副本部署时 In-Memory Wheel 重复触发 | 同一 timer 被多个进程同时触发 | Phase 1 用 DB CAS 做 fencing；未来引入分布式锁 | 开放 |
| 大量 timer 同时到期（如每日 9:00 的 Start Timer） | 瞬时 DB 和引擎压力 | time-wheel 分散 + 批量处理 + 限流 | 开放 |

### 11.2 产品开放问题（2026-09-14 评审已全部决策）

1. **✅ 已决策：Timer Start cron 按租户时区解析。** `tenants.timezone` 字段已存在（默认 `Asia/Shanghai`，见 `ent/schema/tenant.go`），cron 计算绝对 `fire_at` 时按租户时区执行 `time.LoadLocation`，无新增基础设施。用户级时区挂账 P2+。文档统一标注"租户时区语义"。
2. **✅ 已决策：P0 仅支持 interrupting Boundary Timer。** non-interrupting 需要活动继续执行 + 并行分支的执行模型，改 executor fork 逻辑复杂度翻倍；`timer_type` 枚举已预留，P2（事件子流程批次）再扩展 `is_interrupting` 字段。
3. **✅ 已决策：到期动作 = 预定义枚举 + 结构化参数，不做自定义表达式。** 枚举收敛在现有 `notify/escalate/auto_reject/auto_approve` 四动作 + P1 增补；参数走结构化 JSON `params`（escalate 目标组/用户、notify 模板 ID），可 Lint 可审计。自定义表达式与 AGENTS.md"高-risk 动作"约束冲突（审计与安全面失控），门关死。
4. **✅ 已决策：维持 P0/P1 只做 Cancel + Recreate，Working Hours Calendar 留 P2。** 边界声明：timer 的 `fire_at` 一律 UTC 存储 + 租户时区展示，杜绝现有 SLA 时区 bug（路线图挂账项）在 Timer 体系重演；该边界在 Phase 2 SLA 合并时顺带解决。
5. **✅ 已决策：P0 可见不可跳过，P1 增加手动跳过。** 可见：`WorkflowProgressCard` 对 intermediate/boundary timer 渲染"等待中，预计 xx 触发"（数据源 `process_timer` 查询，现成）。可跳过：`POST /timers/:id/skip` 走手动触发语义（CAS fencing 后按 fired 处理 + 审计），P1 与管理 API 写操作一起交付。

### 11.3 与现有 AGENTS.md 约束的关系

- Timer 触发属于"高-risk 动作"，必须走 outbox/command 模式，不能 fire-and-forget
- 所有 timer 操作必须包含 tenant_id，恢复扫描必须按租户隔离
- Timer 触发的流程推进必须记录审计日志（包含 timer_id、原始 fire_at、实际触发时间）
- 幂等 key 必须包含 `timer_id + fire_at`，防止重复触发

### 11.4 与 `bpmn_event_service.go` 的关系

当前 `bpmn_event_service.go` 定义了 `EventDefinition`、`EventInstance`、`TriggerTimer` 等类型，但所有方法均为 stub。Timer Event 基础设施与它的关系：

- **`process_timer` 是底层调度层**：负责 timer 的持久化、触发、重试、恢复
- **`EventInstance` 是上层业务视图**：timer 触发后产生的事件实例记录，供前端流程跟踪和审计查询
- Timer Event 基础设施**实现** `bpmn_event_service.go` 中定义的 `TriggerTimer` 处理逻辑，而非替代它
- Phase 2 引擎集成时，将 `process_timer` 的触发回调桥接到 `BPMNEventService.TriggerEvent()`

---

## 12. 决策记录

| 日期 | 决策 | 原因 |
|------|------|------|
| 2026-09-13 | 选择方案 B（内存调度 + DB 持久化） | 当前单副本部署，无需外部依赖；秒级精度满足 SLA 场景；崩溃恢复通过启动扫描实现 |
| 2026-09-13 | TimeoutScanner 的 notify 动作保留，不迁移到 Timer Event | notify 不改流程走向，用 polling 足够；避免过度设计 |
| 2026-09-13 | Timer Start Event 放到 P1 | 当前无产品模板支持，且需要 cron 解析基础设施 |
| 2026-09-13 | SLA 暂停/恢复采用 Cancel + Recreate 模式（P0） | 实现简单，与 timer 生命周期兼容；审计链通过 parent_timer_id 追溯；Working Hours Calendar 延后到 P2 |
| 2026-09-13 | 多副本升级路径预留 TimerStore 接口抽象 | 当前单副本，但 K8s 多 Pod 是明确方向；接口抽象确保未来切换存储后端不改上层 |
| 2026-09-13 | process_timer 表增加 version 乐观锁 | AGENTS.md 要求高并发资源必须乐观锁；多副本 CAS 触发依赖 version fencing |
| 2026-09-13 | Phase 1 包含管理 API（查询+统计）和 Metrics | Timer 是沉默基础设施，无可观测性 = 无法运维 |
| 2026-09-13 | 新增 Phase 1.5 Spike（1 天） | lib-bpmn-engine v0.2.4 Timer Event 支持未验证，先 spike 再估 Phase 2 |
| 2026-09-13 | Spike 结论：不 fork lib-bpmn-engine，在 CustomProcessEngine 层自建 Timer 调度 | 库仅支持 IntermediateCatchEvent+timeDuration 被动轮询；无 Boundary Timer / Timer Start / 回调 API；CustomProcessEngine 完全绕过该库（自有解析器 + DB 状态机），fork 收益为零 |
| 2026-09-13 | Phase 2 引擎集成路径确认：Timer Scheduler 触发回调 → CustomProcessEngine 推进流程 | 不经过 lib-bpmn-engine 的执行循环；Timer 到期后直接调用 CustomProcessEngine 的方法推进到下一节点，与现有 TimeoutScanner 的 auto_reject/auto_approve 模式一致 |
| 2026-09-14 | 产品评审 Q1：cron 按租户 timezone 解析（缺省 Asia/Shanghai） | `tenants.timezone` 字段已存在，零新增基础设施；MSP/SaaS 多租户"每天 9:00"语义正确 |
| 2026-09-14 | 产品评审 Q2：P0 仅 interrupting Boundary Timer，non-interrupting 留 P2 | 避免 P0 改 executor fork 执行模型；timer_type 枚举已预留扩展位 |
| 2026-09-14 | 产品评审 Q3：到期动作 = 四动作枚举 + 结构化 params，禁用自定义表达式 | 结构化参数可 Lint 可审计；表达式注入与 AGENTS.md 高-risk 动作约束冲突 |
| 2026-09-14 | 产品评审 Q4：Working Hours Calendar 维持 P2；fire_at 一律 UTC 存储 + 租户时区展示 | Cancel+Recreate 已覆盖 P0 场景；UTC 存储边界杜绝 SLA 时区 bug 重演 |
| 2026-09-14 | 产品评审 Q5：P0 等待状态前端可见（WorkflowProgressCard），P1 加手动跳过 | 数据源 process_timer 现成；跳过走手动触发语义（CAS fencing + 审计），与 P1 管理 API 写操作同批交付 |
| 2026-09-15 | Phase 4：任务超时主路径 = task_due 定时器，TimeoutScanner 降级为恢复兜底 | timer 到期分发复用 scanner 的 dispatchTimeoutAction；claim-once 条件更新保证双路径只生效一次；扫描器继续兜底无 timer 任务（注册失败/timer 丢失/功能关闭） |
| 2026-09-15 | Phase 4：SLA Monitor 保持轮询，不做逐票 Timer 化 | CheckSLAViolations 是聚合策略评估（多级阈值/批量工单），轮询是正确工具；逐票 timer 带来注册风暴与重建复杂度，收益不成比例 |
| 2026-09-15 | BPMN dueDate 语义定为 `yyyy-mm-dd` → 当日 23:59:59（服务器本地时区） | XML 解析器既有校验格式即日期；任务截止按自然日结束计算符合业务直觉；时区统一问题随 Q4 的 UTC 存储边界（Phase 2）一并治理 |
| 2026-09-15 | Phase 5：start timer 时间表同步采用**整体替换**（先取消同 `(tenant, process_key)` 全部 pending，再按新定义重建） | 增量同步会在每次发布/部署时累积时间表（timer_id 为新 uuid，幂等键失效）→ 同一 cron 被重复触发并重复启动流程实例 |
| 2026-09-15 | Phase 5：cron 采用 robfig/cron v3 **标准 5 字段**，显式不接受 6 字段（带秒） | 秒级调度在 ITSM 定时启动场景无业务意义；收窄语法可把误配挡在发布前而非运行期 |
| 2026-09-15 | Phase 5：cycle 次数语义（`CycleRemaining`）仅对 `exprType == cycle` 生效 | 无 `/` 的 cron 字符串会被判成"一次性（remaining=1）"→ cron 触发一次后时间表被永久终止；cron 天然无限重复，必须绕开次数分支 |
| 2026-09-15 | Phase 5：重排（rearm）加 `hasPendingStartTimer` 幂等守卫 | 崩溃恢复/重放会以同一 timer 记录二次进入重排；无守卫则叠加第二份时间表，cron 逐跳放大成 2 倍实例 |
| 2026-09-15 | Phase 5：已过期的一次性（`timeDate`）start timer **不注册**（静默跳过，由 lint 提示误配） | 启动一个"本应在过去启动"的流程无业务语义；不跳过会造成重启后的一次性补偿洪峰 |
| 2026-09-15 | Phase 5：`parseISO8601Duration` 要求至少一个 `<数字><单位>` 片段，否则报错 | 旧实现把 `PT`/`PTxxX` 静默解析为 0 时长 → 定时器"立即触发"且调用方无感知，属静默失败（与可观测性治理冲突） |

---

## 附录 A：ISO 8601 Duration 参考

| 表达式 | 含义 |
|--------|------|
| `PT30M` | 30 分钟 |
| `PT4H` | 4 小时 |
| `P1D` | 1 天 |
| `P3D` | 3 天 |
| `P1W` | 1 周 |

## 附录 B：现有调度机制清单

| 服务 | 文件 | 调度方式 | 间隔 |
|------|------|---------|------|
| SLA Monitor | `sla_monitor_service.go` | `safeGo` + `time.NewTicker` | 1 分钟 |
| Escalation | `escalation_service.go` | `safeGo` + `time.NewTicker` | 5 分钟 |
| Incident Escalation | `incident_escalation_service.go` | `safeGo` + `time.NewTicker` | 5 分钟 |
| Timeout Scanner | `bpmn_timeout_scanner.go` | `safeGo` + `time.NewTicker` | 2 分钟 |
| Workflow Auto Escalation | `workflow_automation_service.go` | `safeGo` + `time.NewTicker` | 内部循环 |

以上 5 套独立 ticker 是 Timer Event 基础设施的统一合并候选。

## 附录 C：Phase 1.5 Spike 报告（2026-09-13）

### C.1 Spike 目标

验证 `lib-bpmn-engine v0.2.4` 是否支持 Timer Event 回调，确定 Phase 2 引擎集成路径。

### C.2 lib-bpmn-engine v0.2.4 Timer 能力盘点

#### 数据模型

库在 `engine_timer.go` 中定义了 `Timer` 结构体：

```go
type Timer struct {
    ElementId          string
    ElementInstanceKey int64
    ProcessKey         int64
    ProcessInstanceKey int64
    State              TimerState   // TimerCreated | TimerTriggered | TimerCancelled
    CreatedAt          time.Time
    DueAt              time.Time
    Duration           time.Duration
}
```

Timer 存储在 `BpmnEngineState.timers []*Timer`（未导出字段）。

#### XML 模型支持范围

```go
// model.go — 仅定义了 timeDuration
type TTimerEventDefinition struct {
    Id           string        `xml:"id,attr"`
    TimeDuration TTimeDuration `xml:"timeDuration"`
}
```

| BPMN Timer 元素 | 支持 |
|----------------|------|
| `timeDuration`（相对时长） | 仅 IntermediateCatchEvent |
| `timeDate`（绝对时间） | 不支持，XML struct 中无此字段 |
| `timeCycle`（周期重复） | 不支持，XML struct 中无此字段 |
| Boundary Timer | 不支持，库中无 `BoundaryEvent` 概念 |
| Timer Start Event | 不支持，`TStartEvent` 无 `TimerEventDefinition` |

#### 执行模型：被动轮询

库的 timer 触发依赖调用方反复调用 `RunOrContinueInstance()`：

1. 流程执行到 `IntermediateCatchEvent` + `TimerEventDefinition` 时，创建 `Timer{State: TimerCreated, DueAt: now + duration}`
2. `handleElement()` 返回 `false`，流程暂停，实例保持 `ACTIVE`
3. 调用方必须轮询 `RunOrContinueInstance()`；每次调用检查 `time.Now().After(timer.DueAt)`
4. 到期后 timer 状态变为 `TimerTriggered`，流程继续

库自带示例：
```go
for ; instance.GetState() == process_instance.ACTIVE && err == nil; time.Sleep(2 * time.Second) {
    _, err = bpmnEngine.RunOrContinueInstance(instance.GetInstanceKey())
}
```

#### Handler 注册 API

| API | 用途 | 与 Timer 相关 |
|-----|------|-------------|
| `AddTaskHandler(taskId, func(ActivatedJob))` | Service/User Task 回调 | 无关 |
| `PublishEventForInstance(instanceKey, messageName, vars)` | 消息注入 | 无关 |
| `GetTimersScheduled()` | 只读查看 timer 状态 | 只读，无回调注册 |
| `AddTimerHandler(...)` | **不存在** | — |

### C.3 CustomProcessEngine 与 lib-bpmn-engine 的关系

| 维度 | CustomProcessEngine | EngineAdapter (lib-bpmn-engine) |
|------|--------------------|---------------------------------|
| 解析器 | 自有 `BPMNParser` | 库的 XML 解析 |
| 状态存储 | Ent DB（process_instances, process_tasks 等） | 内存（`processInstances`, `timers` 等未导出字段） |
| 表达式引擎 | 自有 `ExpressionEngine` | 无 |
| 执行模型 | DB 驱动状态机 + 回调注册 | 内存轮询 |
| Timer 支持 | 无（通过 TimeoutScanner 做任务级超时） | 仅 IntermediateCatchEvent + timeDuration |

**关键发现**：`CustomProcessEngine` 不持有也不使用 `EngineAdapter` 或 `BpmnEngineState`。两者是完全独立的执行引擎。

### C.4 结论与决策

| 问题 | 结论 |
|------|------|
| 能否通过 `AddTaskHandler` hook timer 节点？ | 不能。`AddTaskHandler` 只处理 ServiceTask/UserTask，不处理 Event 节点 |
| 能否通过库的 timer 机制实现 Boundary Timer？ | 不能。库无 BoundaryEvent 概念 |
| 能否 fork 库补齐 timer 能力？ | 技术上可行但收益为零——CustomProcessEngine 完全绕过该库 |
| 推荐路径 | **在 CustomProcessEngine 层自建 Timer 调度**，Timer 到期后直接调用 CustomProcessEngine 方法推进流程，与现有 TimeoutScanner 的 `auto_reject`/`auto_approve` 模式一致 |

### C.5 对 Phase 2 的影响

- **不需要 fork 或升级 lib-bpmn-engine**
- Phase 2 的工作简化为：在 `CustomProcessEngine` 中新增 timer 触发后的流程推进方法
- Timer Scheduler 的触发回调直接调用 `CustomProcessEngine`（类似 TimeoutScanner 调用 `CustomProcessEngine.CompleteTask()`）
- BPMN XML 中 timer 定义的解析由自有 `BPMNParser` 扩展完成
- 风险从"高"降为"中"，估算从 4-6 天降为 3-5 天
