# Incident Operations Platform 架构蓝图

> 对应上游需求：[incident-operations-platform-requirements.md](../incident-operations-platform-requirements.md)
> 文档目标：将 14 个 Epic 的需求约束转化为 **决策完整** 的实施蓝图，所有改动都基于现有 `itsm-backend` / `itsm-frontend` / `itsm-ai-service` / `itsm-agent` / `itsm-cli` 现有架构资产。
>
> 受众：开发团队、Code Review、QA、运维。上游不允许再开放业务决策，所有"是否复用 / 是否新建 / 走什么路径"在本文件中固化。

---

## 0. 项目当前架构盘点（可复用 vs 新建）

> 这是本蓝图最重要的章节。需求文档中提出的 14 个 Epic 中，**超过 60% 的能力已经在现有代码中存在**。蓝图第一原则：**先复用，再扩展，最后才新建**。

### 0.1 现有架构资产（直接复用）

| 资产 | 路径 | 复用于 Epic |
|---|---|---|
| **Connector Marketplace + Manifest + Registry + Factory** | `itsm-backend/connector/` | 01, 03, 07 |
| **Connector SPI（Sender / Receiver / HealthCheck / Capability）** | `connector/connector.go` | 01, 03, 07 |
| **NotificationOutbox + NotificationDelivery（审计）** | `service/notification_*.go`, `ent/schema/notification_delivery.go` | 07, 09 |
| **OperationalCommand Outbox（Fencing / Lease / Retry / DLQ / Heartbeat）** | `internal/commandbus/commandbus.go`, `ent/schema/operational_command.go` | **08（核心底座）** |
| **Skill Registry + Manifest + ExtendedSkill** | `service/skill_registry.go`, `service/skill_manifest.go` | 11 |
| **11 个 Built-in Skill（含 Triage、Summarize、KnowledgeSearch、Analytics）** | `handlers/ai/builtin_skills.go` | 11 |
| **Notification Service + Preference + Outbox** | `service/notification_*` | 07 |
| **Tenant + RBAC + Endpoint ACL + MSP** | `ent/schema/tenant.go`, `middleware/`, `ent/schema/msp_allocation.go` | 全部（强制） |
| **CMDB（CI / CI Relationship / Topology / AttributeDefinition / Discovery）** | `ent/schema/configurationitem*.go`, `handlers/cmdb/` | 05 |
| **AuditLog** | `ent/schema/auditlog.go` | 09, 10 |
| **Incident 已有状态机**（`new → acknowledged → assigned → in_progress → triaged → escalated → on_hold → resolved → closed / cancelled`） | `common/constants.go` 中 `IsValidIncidentStatusTransition` | 16（演进） |
| **IncidentEscalationRule / IncidentAlert / IncidentEvent / IncidentRule / IncidentRuleExecution / IncidentMetric** | `ent/schema/incident*.go` | 03, 04, 07, 09 |
| **IncidentService（Create / List / Update / Escalate / LinkCIs / Monitoring）** | `service/incident_service.go`, `handlers/incident/service.go` | 全 Epic |
| **RootCauseAnalysis + ChangePIR 模板** | `ent/schema/root_cause_analysis.go`, `service/pir_service.go` | 10, 12 |
| **ReportExport（Excel / PDF）** | `service/report_export_service.go` | 12 |
| **IncidentMonitoringService（含 GenerateIncidentReport）** | `service/incident_monitoring_service.go` | 12, 13 |
| **BPMN Process Engine + Audit + Permission** | `service/bpmn_*`, `ent/schema/process_*` | 08, 10 |
| **TenantContract（已有 Contract）** | `ent/schema/contract.go` | 02（只复用表，不复用 Filed） |
| **AI Eval + Telemetry** | `service/ai_evaluator.go`, `service/ai_telemetry.go` | 11 |

### 0.2 现有架构缺口（必须新建）

| 缺口 | 复用到谁的领域 | 优先级 |
|---|---|---|
| `InboundEvent` 统一领域对象 + `ChannelProvider` SPI | 仅借用 `connector.Receiver` 的生命周期/健康检查，但 Domain 必须新建独立的 `InboundEvent` 归一化层 | P0 |
| `Customer` / `CustomerContact` / `CustomerIdentity` / `ServiceContract` / `ServiceEntitlement` 5 个实体 | 是 Core（新领域，非客户定制） | P0 |
| `AlertGroup`（去重聚合）+ `AlertEvent` 类型 | Core | P0 |
| `AlertProvider` SPI + 3 个 Provider（Webhook / Prometheus / Zabbix） | 走 Connector 机制 + 域内 Parse | P0 |
| `OnCallTeam` / `OnCallSchedule` / `OnCallRotation` / `OnCallShift` / `OnCallMember` / `OnCallOverride` / `EscalationPolicy` / `EscalationLevel` 8 个实体 | Core | P0 |
| `IncidentAcknowledgement` 实体（区分 NotificationDelivered / PhoneAnswered / EngineerAcknowledged） | Core | P0 |
| `EscalationPolicy` / `EscalationLevel`（把现有 `IncidentEscalationRule` 升级为 Policy/Level 二级结构） | Core | P0 |
| `IncidentWorklog` 实体（区分系统 Timeline 与工程师 Worklog） | Core | P0 |
| `IncidentReport` 实体（持久化报告 + 模板） | Core | P1 |
| `ShiftHandover` 实体（接班报告） | Core | P1 |
| `ResourceMatcher` + `DynamicPriority` 服务 | Core | P0 |
| `AiAuditRecord`（AI 调用全审计） | 现有 `tool_invocation` 可复用部分扩展 | P0 |
| `NOC Workspace` 前端模块（统一大屏） | Core | P1 |
| 5 个 AI Skill（CustomerEntityExtraction / KnowledgeRecommendation / RCA Assist / IncidentReport / ShiftHandover） | 复用现有 Skill Registry | P1 |

### 0.3 现有架构演进（不动业务，升级抽象）

| 现有 | 演进 | 原因 |
|---|---|---|
| `IncidentEscalationRule` | 升级为 `OnCallEscalationPolicy` + `EscalationLevel`（多级） | 现状仅支持单条规则；需求要求"initial / retry / ack_timeout / level 2 / level 3" |
| `IncidentEscalationRule.trigger_minutes` | 引入 `DurableTimer` Command（`incident.escalation.tick`），由 OperationalCommand 调度 | 需求 8 明确要求"不能依赖 goroutine 和内存 Timer" |
| `IncidentAlert` 与 incident N:1 关联 | 引入 `AlertGroup` 中间层，AlertGroup 1:1 Incident | 现状缺去重聚合 |
| `IncidentEvent` | 扩展 `actor_type` / `actor_source`（system / engineer / ai / provider），引入 `IncidentWorklog` 区分人写 | 需求 9 区分系统 Timeline 与工程师 Worklog |
| `RootCauseAnalysis.ticket_id` | 新增 `incident_id` 旁路（基线 ticket_id 保留） | RCA 同样服务于 Incident |
| `Contract` (vendor 维度) | 保留并新增 `ServiceContract` (customer 维度)，互不替代 | 现有 Contract 是供应商合同；需求是客户服务合同 |
| `IncidentStatusNew/Acknowledged/.../Closed/Cancelled` 状态机 | 在 `common/constants.go` 中新增 6 个状态 `received / qualifying / dispatching / waiting_ack / escalating / rca_pending`，**严禁覆盖现有状态** | 需求 16 要求 `received → qualifying → ... → closed` 完整链条；同时保留 `resolved` 软状态 |

---

## 1. 架构原则（硬约束）

### 1.1 来自上游需求（不可妥协）

| 编号 | 原则 | 工程落地 |
|---|---|---|
| AR-1 | 100% 客户定制 → Adapter/Extension；100 个客户都会需要 → Core | 所有 Epic 决策前必须回答"100 个客户以后是否都会需要？" |
| AR-2 | Rule First，AI Second | 客户资格 / SLA / 升级 / 状态转换全部走规则引擎；AI 仅做 Entity Extraction / 推荐 / 总结 |
| AR-3 | 所有异步动作必须 Durable | 任何 Timer / Email / Voice / Escalation 必须通过 `commandbus.Enqueue*`，**禁止 goroutine + time.After** |
| AR-4 | 系统 / 业务 / AI 动作必须入 Timeline | 统一由 `IncidentEvent` 持久化，前端只读不写 |
| AR-5 | Alert ≠ Incident | 1 万 Alert → 几十 Incident；强制 AlertGroup 聚合 |
| AR-6 | 客户特殊系统走 Provider/Adapter | 不允许客户名字出现在 Core 模块命名 |

### 1.2 来自现有架构（不可破坏）

| 编号 | 原则 | 工程落地 |
|---|---|---|
| AR-7 | 业务逻辑必须跑在 Service 层 | 禁止 Controller 直连 Repository；遵循 `handlers/<domain>/` 新分层 |
| AR-8 | 返回必须走 DTO + Mapper | 严禁直接返回 Ent 模型；Mapping 单向 snake→camel |
| AR-9 | 多租户隔离 = 必须项 | 每个新表必须带 `tenant_id`；查询必须走 tenant-aware Repository |
| AR-10 | 审计是一等公民 | 所有 AI / Connector / Workflow / RAG / 高敏操作必须可审计 |
| AR-11 | 插件化能力走 Connector / Skill Registry | 不允许新增"第二个集成框架" |
| AR-12 | 持久化副作用走 OperationalCommand | 任何 Future / Asynchronous / External IO 必须通过 CommandBus |

---

## 2. 领域模型增量设计

### 2.1 新增 Ent Schema（按 Epic 归类）

#### Epic 01：Unified Incident Intake

```go
// ent/schema/inbound_event.go
type InboundEvent struct{ ent.Schema }
// 字段（精简，按需求 § 2.1 映射）：
//   id, tenant_id, source_type (email/api/wechat), source (实例名),
//   external_id (Message-ID), sender_id/sender_name/sender_address,
//   title, content, severity (已归一为 INFO/WARNING/MINOR/MAJOR/CRITICAL),
//   resource_type, resource_id, labels (JSON), metadata (JSON),
//   occurred_at, received_at, correlation_id, raw_payload (bytes),
//   status (received/normalized/dropped/converted_to_incident),
//   inbound_event_id (nullable, 关联 Incident/AlertGroup)
```

**索引**：`tenant_id, correlation_id` 唯一；`tenant_id, source_type, external_id` 唯一（消息幂等）。

#### Epic 02：Customer Identity & Entitlement

```go
type Customer struct{ ent.Schema }            // 客户（独立于 Tenant）
type CustomerContact struct{ ent.Schema }     // 客户联系人
type CustomerIdentity struct{ ent.Schema }    // 客户身份（email / phone / wechat 等）
type ServiceContract struct{ ent.Schema }     // 服务合同（与现有 Contract 平级）
type ServiceEntitlement struct{ ent.Schema }  // 服务权益（按合同 + 服务目录细分）
```

- `ServiceContract` 与现有 `Contract` **不能合并**：现有 Contract 绑定 Vendor（采购合同），ServiceContract 绑定 Customer（服务合同）。
- `CustomerIdentity` 必须唯一 `(tenant_id, identity_type, identity_value)`。
- `Customer` 多对多 `ServiceContract`；`ServiceContract` 一对多 `ServiceEntitlement`；`ServiceEntitlement` 关联 `ServiceCatalogItem`。

#### Epic 03：Alert Gateway

```go
type AlertGroup struct{ ent.Schema }
// 字段：tenant_id, alert_provider (prometheus/zabbix/webhook),
//       dedup_key, fingerprint, first_seen_at, last_seen_at, occurrences,
//       severity (已归一), resource_type, resource_id, status (open/suppressed/converted/cancelled),
//       incident_id (nullable)
type AlertEvent struct{ ent.Schema }
// 字段：tenant_id, alert_group_id, occurred_at, raw_payload, severity (原始),
//       normalized_severity, fingerprint, status (received/suppressed/dropped/included)
type AlertProviderConfig struct{ ent.Schema }
// 字段：tenant_id, provider (prometheus/zabbix/webhook/custom),
//       display_name, config (JSON), secret_ref (敏感数据走凭据服务),
//       is_enabled, health_status, last_check_at
type AlertSuppressionRule struct{ ent.Schema }
// 字段：tenant_id, name, kind (maintenance_window / change_window / rule / parent / ci / time),
//       matcher (JSON), starts_at, ends_at, enabled
```

#### Epic 06：On-Call Management

```go
type OnCallTeam struct{ ent.Schema }
// tenant_id, name, code, description, default_policy_id, escalation_policy_id
type OnCallSchedule struct{ ent.Schema }
// tenant_id, team_id, name, timezone, rotation_type (daily/weekly/custom), start_at
type OnCallShift struct{ ent.Schema }
// tenant_id, schedule_id, starts_at, ends_at, role (primary/secondary/supervisor), user_id
type OnCallRotation struct{ ent.Schema }
// tenant_id, schedule_id, name, pattern (cron-like), starts_at, ends_at, member_ids
type OnCallMember struct{ ent.Schema }
// tenant_id, team_id, user_id, role, skills (JSON)
type OnCallOverride struct{ ent.Schema }
// tenant_id, schedule_id, shift_id, original_user_id, override_user_id, starts_at, ends_at, reason
type EscalationPolicy struct{ ent.Schema }
// tenant_id, name, description, retry_interval, max_attempts, ack_timeout, stop_when (JSON)
type EscalationLevel struct{ ent.Schema }
// tenant_id, policy_id, level, target_type (oncall_team/oncall_user/role), target_id,
//       channels (JSON), wait_minutes
```

#### Epic 07：ACK

```go
type IncidentAcknowledgement struct{ ent.Schema }
// tenant_id, incident_id, oncall_user_id, source (web/email_link/phone_dtmf/api/mobile),
//       status (notification_delivered / phone_answered / engineer_acknowledged),
//       created_at, related_notification_id, notes
```

#### Epic 09：Incident Timeline & Worklog

复用并扩展 `ent/schema/incidentevent.go`，新增：

```go
type IncidentWorklog struct{ ent.Schema }
// tenant_id, incident_id, operator_id, started_at, ended_at, action, result,
//       worklog_type (system / engineer / ai), attachments (JSON),
//       related_timeline_event_id (nullable)
// 索引: (tenant_id, incident_id, started_at desc)
```

`IncidentEvent` 增加 `actor_type`（system / engineer / ai / provider）、`actor_source`（webhook / email / feishu / voice / web）、`is_ai_generated` 字段。

#### Epic 12：Incident Report

```go
type IncidentReport struct{ ent.Schema }
// tenant_id, incident_id, template_id, status (draft/generating/ready/failed),
//       format (html/pdf/markdown), storage_path, generated_at, generated_by_user_id,
//       // 关键：事实快照（防止后期数据漂移）
//       snapshot (JSON, 包含 timeline / worklog / cmdb / sla / rca / problem / change)
// 索引: (tenant_id, incident_id, generated_at desc)
type IncidentReportTemplate struct{ ent.Schema }
// tenant_id, name, format, sections (JSON), is_default
```

#### Epic 14：AI Shift Handover

```go
type ShiftHandover struct{ ent.Schema }
// tenant_id, handover_date, from_user_id, to_user_id, team_id,
//       window_starts_at, window_ends_at,
//       period_stats (JSON: 原始告警 / 降噪后 / 创建 Incident / P1,P2 / 未恢复数),
//       open_incidents (JSON), pending_rca (JSON), trend_risks (JSON),
//       status (draft/confirmed/cancelled), confirmed_at, notes
```

### 2.2 现有 Schema 演进

| Schema | 演进 |
|---|---|
| `incident_escalation_rule.go` | 保留，向后兼容；新逻辑走 `escalation_policy` + `escalation_level` |
| `incidentevent.go` | 新增 `actor_type` enum、 `actor_source` enum、 `is_ai_generated` bool |
| `incidentalert.go` | 保留，但 `incident_alert` 改为 `alert_group_id` + `alert_event_id` 双旁路（避免破坏现有数据） |
| `root_cause_analysis.go` | 新增 `incident_id` 可选字段（非破坏） |
| `common/constants.go` | 新增 6 个 IncidentStatus：`received / qualifying / dispatching / waiting_ack / escalating / rca_pending`；`IsValidIncidentStatusTransition` 表自动扩展 |

---

## 3. 14 个 Epic 的实施规划

### Epic 01：Unified Incident Intake

**目标**：禁止 Email / IM / Webhook 直接调用工单业务逻辑；统一领域对象 `InboundEvent`。

**新增**：
- `ent/schema/inbound_event.go`
- `domain/inbound/inbound_event.go`：`InboundEvent` 结构 + `Normalize()` 阶段（通道无关 → 业务无关）

**Provider SPI**（需求 2.3）：

```go
// domain/inbound/provider.go
type ChannelProvider interface {
    Code() string                                                // "email" / "webhook" / "wechat" / "mock"
    Receive(ctx context.Context) ([]InboundEvent, error)         // 主动拉取
    Send(ctx context.Context, msg OutboundMessage) error         // 自动回复
    HealthCheck(ctx context.Context) HealthStatus
}
```

**实现**：
- 第一阶段：EmailProvider（IMAP 拉取 + SMTP 发送）、WebhookProvider（HTTP 回调持久化到 inbound_event）、MockProvider（测试）
- 后续：继承 `connector.Receiver` 思维，但**走领域层 SPI**，不直接复用 Connector，避免与"出站消息"语义混用

**自动回复邮件模板**（需求 § 2.2）：
```
您好，您的故障已经受理。
故障单号：INC-20260820-000123
优先级：P2
当前状态：等待工程师接单
请保留该邮件主题，后续回复将自动关联到此故障。
```

**幂等**：`tenant_id + source_type + external_id` 唯一。

**入消息关联**：以 `Message-ID` 为锚，相同 `In-Reply-To` 关联到同一 Incident。

**事件**：通过 `commandbus.Enqueue` 入箱 `inbound.normalize`，经 `IncidentEvent` 落审计。

**复用**：`connector.HealthStatus` 复用 `connector.HealthStatus` 结构。

---

### Epic 02：Customer Identity & Entitlement

**目标**：客户资格验证必须确定性；AI 仅做候选抽取。

**新增 5 个实体**（见 § 2.1）。

**Customer Resolver**（需求 3.2）：

```go
// domain/customer/resolver.go
type Resolver interface {
    Resolve(ctx context.Context, sender IdentityHint) (VerificationResult, error)
}

type VerificationResult struct {
    Status        VerificationStatus   // VERIFIED / NOT_FOUND / AMBIGUOUS / CONTRACT_EXPIRED / NOT_ENTITLED / MANUAL_REVIEW
    CustomerID    *int
    ContactID     *int
    ContractID    *int
    EntitlementID *int
    Reason        string
}
```

**AI 介入位置**（仅 Entity Extraction）：
- 调用 `ai.entity_extraction` Skill（见 Epic 11）
- 输入：邮件正文 / 标题 / 发件人
- 输出：候选客户名 + 候选地址 + 候选联系电话 + 置信度
- **调用结果必须可审计**（model / prompt_version / confidence / accepted_by / timestamp）

**人工兜底**：`MANUAL_REVIEW` 状态落到工作台人工认领。

**入合同校验**：`ServiceContract.start_date ≤ now ≤ end_date`；`ServiceEntitlement.service_catalog_item_id` 必须存在于 Incident 关联的 service。

---

### Epic 03：Alert Gateway

**目标**：把 Zabbix / Prometheus / Webhook 告警统一归一、纳入 AlertGroup。

**AlertProvider SPI**（需求 4.1）：

```go
// domain/alert/provider.go
type AlertProvider interface {
    Code() string                                        // "prometheus" / "zabbix" / "webhook"
    Parse(ctx context.Context, payload []byte) ([]AlertEvent, error)
    HealthCheck(ctx context.Context) HealthStatus
}
```

**第一阶段实现**：
- Webhook AlertProvider（通用 payload 适配）
- Prometheus Alertmanager AlertProvider（解析 `alerts[]` 数组）
- Zabbix AlertProvider（解析 trigger.media payload）

**注册方式**：复用 `connector.MustRegister` 模式，但存放 `alert/registry/`，避免与 Connector 混淆。

**AlertGroup 聚合**（需求 5.2）：`dedup_key` 维度定义为 `fingerprint = sha256(source + rule + resource + sorted(labels))`。

**Suppression**（需求 5.3）：
- 维护窗口：与 `change.go` 联动（Change 实施窗口 `starts_at` 到 `ends_at`）
- 规则抑制：JSON matcher 规则（label / severity / resource_type）
- 父事件抑制：CMDB 拓扑父节点 AlertGroup 触发时，子节点自动静默
- 时间段抑制：`AlertSuppressionRule.starts_at / ends_at`

**Correlation**（需求 5.4）：
- 一阶段：Rule-based — `AlertCorrelationRule` 表（YAML/JSON）
- 二阶段：Topology（基于 `ci_relationship`）
- 三阶段：AI Semantic（Call `ai.correlation` Skill）

---

### Epic 04：Event Intelligence

**Normalization**（需求 5.1）：归一表 `severity_normalization_map`（DB 表 + Cache），key = `provider + raw_severity`，value = `INFO/WARNING/MINOR/MAJOR/CRITICAL`。

**Deduplication**（需求 5.2）：基于 `AlertGroup.fingerprint`；新增 alert 时 `INSERT ... ON CONFLICT DO UPDATE SET occurrences = occurrences + 1, last_seen_at = now()`。

**Suppression**：见 Epic 03。

**Correlation Service**（需求 5.4）：
- Rule-based：第一阶段
- Topology：第二阶段（`service/cmdb/topology_service.go`）
- 时间窗：第二阶段（滑动窗口聚类）
- AI Semantic：第三阶段（Skill 注册）

---

### Epic 05：CMDB Enrichment & Impact Analysis

**ResourceMatcher**（需求 6）：
```go
// domain/alert/resource_matcher.go
type Matcher interface {
    Match(ctx context.Context, alert AlertEvent) ([]ResourceMatch, error)
}
type ResourceMatch struct {
    CIID    int
    Score   float64  // 0-1
    Reason  string   // 解释命中原因
    Path    string   // 拓扑路径（可选）
}
```

匹配优先级：
1. `resource_id` 精确匹配
2. `labels.hostname` / `labels.ip` 匹配 `ConfigurationItem.identifier`
3. `labels.app` 匹配 `Application`
4. 模糊匹配 + 人工候选

**DynamicPriority**（需求 6.1）：

```text
PriorityScore =
  AlertSeverity       (40%)
+ CI.Criticality      (25%)
+ BusinessCriticality (20%)
+ ImpactScope         (10%)
+ TimeContext         (5%)   // 业务高峰时段加权
```

结果映射：`P1 (≥90) / P2 (≥70) / P3 (≥50) / P4 (else)`。

**规则可解释**：每次 priority 决策落 `incident_event`，`data = { score, components, reason }`。

**AI 介入**：仅作为"建议 priority"，人类授权后覆盖。

---

### Epic 06：On-Call Management

**8 个新实体**（见 § 2.1）。

**核心查询 API**（需求 6）：

```http
GET /api/v1/oncall/teams
POST /api/v1/oncall/teams
GET /api/v1/oncall/schedules?team={code}
POST /api/v1/oncall/schedules
GET /api/v1/oncall/current?team={code}&at={RFC3339}
POST /api/v1/oncall/overrides
GET /api/v1/escalation-policies
POST /api/v1/escalation-policies
```

**`GET /oncall/current` 返回**：
```json
{
  "team": "db",
  "at": "2026-08-20T22:30:00+08:00",
  "primary": { "user_id": 1001, "name": "张三", "phone": "138****", "email": "zhangsan@..." },
  "secondary": { "user_id": 1002, "name": "李四" },
  "supervisor": { "user_id": 1003, "name": "王经理" },
  "effective_until": "2026-08-21T08:00:00+08:00"
}
```

**实现要点**：
- 使用 `cache/`（Redis）缓存 `OnCallSnapshot(team, time_bucket)` 5 分钟
- 节假日来源：默认 `holiday_calendar` 表（公开节假日 JSON 导入，Tenant 可覆盖）
- Override 优先级：Override > Shift > Rotation > Default

**取消排班查询冲突**：用 `uk_team_time` 唯一索引做冲突检测。

---

### Epic 07：Notification / ACK / Escalation

**NotificationProvider SPI**（需求 7.1）：

```go
type NotificationProvider interface {
    Channel() string                                  // "email" / "voice" / "sms" / "im"
    Send(ctx context.Context, req NotificationRequest) (DeliveryResult, error)
}
```

**实现**：
- EmailProvider：复用 `connector/builtin/` 内置 SMTP Connector
- VoiceProvider：新建 `connector/builtin/voice`（基于 SIP / 三方语音接口），第一阶段仅 Mock
- WebhookProvider：复用 Connector

**IncidentAcknowledgement**（需求 7.2）：

```go
type AckStatus string
const (
    AckNotificationDelivered AckStatus = "notification_delivered" // 邮件已送达
    AckPhoneAnswered         AckStatus = "phone_answered"           // 电话接通
    AckEngineerAcknowledged  AckStatus = "engineer_acknowledged"    // 工程师确认
)
```

**严格区分**：需求 7.2 明确要求"电话接通 ≠ ACK"。引擎必须按 `acknowledged_at` 字段判断，且要求显式工程师确认动作（电话按 1 / Web 点击 / 邮件回复）。

**EscalationPolicy**（需求 7.3）：
```yaml
initial:
  after: 0m
  targets: [primary]
  channels: [email, phone]
retry:
  interval: 3m
  max_attempts: 5
ack_timeout: 15m
escalation:
  - level: 2
    target: secondary
    after: 15m
  - level: 3
    target: supervisor
    after: 30m
stop_when: [acknowledged, resolved, cancelled]
```

**禁止 `while(!answered)`**：超时由 `OperationalCommand.available_at` 推进，Handler 必须 idempotent。

---

### Epic 08：Durable Timer & Reliable Execution（核心非功能）

**结论**：现有 `OperationalCommand` 框架 100% 承载需求 8，无需新建 Timer 框架。

新增 Command Types（注册到 `internal/commandbus/commandbus.go`）：

```go
const (
    CommandIncidentEscalationTick  = "incident.escalation.tick"
    CommandIncidentAckTimeoutTick  = "incident.ack.timeout"
    CommandIncidentSLAEvaluateTick = "incident.sla.evaluate"
    CommandIncidentNotification    = "incident.notification.send"
    CommandAlertNormalize          = "alert.normalize"
    CommandAlertSuppressionCheck   = "alert.suppression.check"
    CommandOnCallSnapshotRefresh   = "oncall.snapshot.refresh"
    CommandIncidentReportGenerate  = "incident.report.generate"
    CommandShiftHandoverGenerate   = "incident.shift_handover.generate"
    CommandInboundNormalize        = "inbound.normalize"
)
```

**复用要点**：
- Lease / Fencing / Retry / DLQ 全部已有
- Heartbeat 机制保证 Worker Crash 不丢任务
- `EnqueueSQLTx` 已支持 SQL 事务级 outbox，任何业务写 + Command 提交原子性

**新增薄封装**：
```go
// service/durable_timer.go
//   - Delay(ctx, aggregate, commandType, payload, available_at)
//   - Every(ctx, aggregate, commandType, payload, interval)
//   - Cancel(aggregate) -> 取消未触发 Command
```

**幂等约束**：所有 Handler 必须 `idempotency_key = aggregate_type:aggregate_id:command_type:dedup_hash`；同 Key 多次入箱只执行一次。

---

### Epic 09：Incident Timeline & Worklog

**两套记录**（需求 9.1, 9.2）：

| 类别 | 实体 | 写入方 | 写入时机 |
|---|---|---|---|
| System Timeline | `IncidentEvent` | Domain Service / Command Handler | 状态机变更、通知出箱、Escalation 触发 |
| Engineer Worklog | `IncidentWorklog` | 工程师（手动） | 每轮处理动作 |
| AI-Generated | `IncidentEvent` (`is_ai_generated=true`) | SkillExecutor | AI 完成时 |

**Append-only 原则**：`IncidentEvent` 不允许 Service 层 update/delete；只能新增。

**Worklog API**：
```http
POST /api/v1/incidents/{id}/worklog          # 添加
GET  /api/v1/incidents/{id}/worklog          # 列表
POST /api/v1/incidents/{id}/timeline/note    # 工程师备注（落 IncidentEvent）
```

**Worklog DTO**（camelCase）：
```json
{
  "id": "wl_001",
  "incidentId": "inc_20260820_000123",
  "operatorId": 1001,
  "startedAt": "2026-08-20T09:20:00+08:00",
  "endedAt": "2026-08-20T09:35:00+08:00",
  "action": "检查数据库复制状态",
  "result": "发现 Replica Lag 180 秒",
  "worklogType": "engineer",
  "attachments": []
}
```

---

### Epic 10：RCA & Closure Governance

**关闭门禁**（需求 10）：

```go
type ClosureGate struct {
    RequireWorklog          bool   // Worklog 是否完整
    RequireRootCause        bool   // RCA 是否完整
    RequireResolution       bool   // 解决方案是否完整
    MinPriorityForEnforce   string // P1/P2 强制
}
```

服务层校验：
```go
func (s *IncidentService) ValidateClosure(ctx context.Context, incident *Incident) error {
    gate := s.GetClosureGate(ctx, incident.TenantID, incident.Priority)
    if !gate.RequireWorklog { return nil }
    wlCount, _ := s.repo.CountWorklog(ctx, incident.ID, "engineer")
    if wlCount == 0 { return ErrClosureGateWorklogMissing }
    if gate.RequireRootCause && !s.HasRootCause(ctx, incident.ID) { return ErrClosureGateRCAIncomplete }
    if gate.RequireResolution && incident.ResolutionSteps == nil { return ErrClosureGateResolutionMissing }
    return nil
}
```

**Incident → Problem / KnownError / Knowledge / Change**（已有 SDK）：
- `Problem.CreateFromIncident(ctx, incident_id)`：通过 `process.trigger` 复用 BPMN 流程
- `Knowledge.CreateFromIncident(ctx, incident_id)`：必须经过 `service.knowledge_article_*` 校验，避免污染知识库
- `Change.CreateFromIncident(ctx, incident_id)`：走 Change Router，确认关系 `incident.change_id`

**RCA 字段**（需求 § 10，建议结构）：
```json
{
  "symptom": "...",
  "directCause": "...",
  "rootCause": "...",
  "trigger": "...",
  "contributingFactors": ["..."],
  "resolution": "...",
  "correctiveAction": "...",
  "preventiveAction": "..."
}
```

---

### Epic 11：AI Incident Intelligence

**新增 5 个 Skill，复用 `service.SkillRegistry`**：

```go
// handlers/ai/skills_incident.go
NewCustomerEntityExtractionSkill(svc)     // ai.customer.entity_extraction
NewKnowledgeRecommendationSkill(svc)      // ai.knowledge.recommend
NewRCAAssistSkill(svc)                    // ai.rca.assist
NewIncidentReportSkill(svc)               // ai.incident.report
NewShiftHandoverSkill(svc)                // ai.incident.shift_handover
```

**Audit 字段**（需求 § 11）写入 `ai_audit_record`：

```go
type AIAuditRecord struct {
    ID              int
    TenantID        int
    SkillCode       string  // "ai.triage"
    Model           string  // "deepseek-chat"
    PromptVersion   string  // "v1.2.0"
    InputRef        string  // incident_id / ticket_id
    InputHash       string  // sha256
    Output          string  // JSON
    Confidence      float64
    Accepted        *bool   // 人工接受/拒绝
    AcceptedBy      *int
    CreatedAt       time.Time
    LatencyMs       int64
}
```

**AI 调用入口必须**：`base.SkillExecutor.Execute(ctx, audit)`，任何跳开审计的 AI 调用视为缺陷。

**失败兜底**：
- timeout → 走规则引擎默认推荐
- low confidence（<0.5）→ 要求人工确认
- disabled provider → 静默关闭 Skill，不阻塞主流程

---

### Epic 12：Incident Report

**逻辑**（需求 § 12）：

```text
事实来源（仅以下，不允许 LLM 自行补充）：
  Incident, CMDB, Timeline, Worklog, NotificationAttempts,
  ACK, SLA, RCA, Problem, Change
```

**实施**：
1. 创建 `IncidentReport` 实体（持久化事实快照）
2. `IncidentReportService.Generate(ctx, incident_id, template_id)`：
   - 收集事实 → 写入 `snapshot` JSON
   - 调用 `ai.incident.report` Skill（LLM 仅做模板填充）
   - 落 `format=html/pdf/markdown` 三种产物
3. 复用 `service/report_export_service.go` 的 PDF / Excel 导出能力

**模板**：`IncidentReportTemplate`（DB 表 + 默认模板 seed）。

**导出**：
- HTML：`html/template`
- PDF：复用 `report_export_service.go` PDF 路径（扩展为任意 HTML 输入）
- Markdown：直接拼接

**审计**：报告生成落 `IncidentEvent` + `ai_audit_record`。

---

### Epic 13：NOC / Incident Operations Workspace

**前端**：新建 `/app/(main)/operations/noc/page.tsx` + `/components/operations/NocWorkspace.tsx`。

**后端聚合 API**：
```http
GET /api/v1/operations/noc/snapshot
→ {
    "criticalCount": 3,
    "p1Count": 2,
    "unackedCount": 4,
    "slaRiskCount": 5,
    "activeIncidents": [ ... 5 条],
    "alertGroups": [ ... 10 条],
    "oncallSnapshot": { "db": {...}, "network": {...}, "app": {...} }
}
```

**布局**（需求 § 13）：

```
┌── Incident Operations ─────────────────────────────┐
│ Critical 3 │ P1 2 │ Unacked 4 │ SLA Risk 5          │
├─────────────────────────┬───────────────────────────┤
│ Active Incidents         │ Alert Groups             │
│ INC-0182 PACS DB  P1    │ PACS DB latency  38x     │
│ INC-0181 Network  P2    │ ESXi host down   21x     │
├─────────────────────────┼───────────────────────────┤
│ On-call                  │ Incident Detail          │
└─────────────────────────┴───────────────────────────┘
```

**SLA Risk 计算**：基于 `sla_policy.go` 已有 `sla_alert_rule` + `sla_violation` 实体，新增 `sla_remaining_minutes` 字段（Service 层计算）。

**复用**：
- `dto/dashboard_dto.go` 已有 `IncidentStats`，可扩展
- `handlers/dashboard/` 已有 dashboard handler，可新增 `noc_snapshot` endpoint

---

### Epic 14：AI Shift Handover

**实体**：`ShiftHandover`（见 § 2.1）。

**生成流程**：
1. 定时器（每日 9:00 / 21:00 + 临时触发）→ CommandBus `incident.shift_handover.generate`
2. 收集过去 12h 统计：原始告警、降噪后、Incident 数、P1/P2、未恢复列表
3. 调用 `ai.incident.shift_handover` Skill → 生成 markdown 摘要
4. 写入 `ShiftHandover.period_stats` + LLM 摘要
5. 通知 outgoing & incoming on-call 用户

**确认模型**：
- `status = draft → confirmed` 双方各签一次
- `confirmed_at` 必须双方都打过卡
- 接收方对遗留事项的注释追加到 `notes`

**复用**：
- AI 调用：`ai.incident.shift_handover` Skill
- 通知：`notification.deliver` Command
- 排班快照：`oncall.current` API

---

## 4. 状态机演进（需求 § 16）

### 4.1 现有 + 新增状态

| 现有状态 | 用途 | 保留 |
|---|---|---|
| `new` | 初始 | ✅ |
| `acknowledged` | 工程师已 ACK | ✅ |
| `assigned` | 已分配 | ✅ |
| `in_progress` | 处理中 | ✅ |
| `triaged` | 已分类 | ✅ |
| `escalated` | 已升级 | ✅ |
| `on_hold` | 暂停 | ✅ |
| `resolved` | 已解决 | ✅ |
| `closed` | 已关闭 | ✅ |
| `cancelled` | 已取消 | ✅ |

**新增**（需求 16 推荐）：

| 新状态 | 触发条件 | 含义 |
|---|---|---|
| `received` | InboundEvent 已入箱 | 接收 |
| `qualifying` | 客户验证 / AlertGroup 归一中 | 资格审核 |
| `dispatching` | 解析 OnCall、发送通知 | 调度 |
| `waiting_ack` | 已通知，等待 ACK | 等 ACK |
| `rca_pending` | resolved 后等待 RCA 提交 | 待 RCA |

**严禁删除**任何现有状态（破坏现有数据）。

### 4.2 转换表（最终版）

```go
// common/constants.go
var IncidentStatusTransitions = map[string][]string{
    "received":      {"qualifying", "cancelled"},
    "qualifying":    {"received", "rejected", "manual_review", "open", "cancelled"},
    // 兼容已有
    "new":           {"acknowledged", "assigned", "in_progress", "cancelled"},
    "open":          {"dispatching", "cancelled"},
    "dispatching":   {"waiting_ack", "cancelled"},
    "waiting_ack":   {"acknowledged", "escalating", "cancelled"},
    "escalating":    {"acknowledged", "in_progress", "cancelled"},
    "acknowledged":  {"in_progress", "on_hold", "cancelled"},
    "assigned":      {"in_progress", "escalated", "on_hold", "cancelled"},
    "in_progress":   {"resolved", "escalated", "on_hold", "cancelled"},
    "triaged":       {"in_progress", "escalated", "on_hold", "cancelled"},
    "escalated":     {"in_progress", "on_hold", "cancelled"},
    "on_hold":       {"in_progress", "cancelled"},
    "resolved":      {"rca_pending", "closed", "in_progress", "cancelled"},
    "rca_pending":   {"closed", "in_progress", "cancelled"},
    "closed":        {},
    "cancelled":     {},
}
```

> `REPORT_GENERATED` 不作为 Incident 主状态（需求 16），单独存 `IncidentReport.status`。

---

## 5. 三阶段实施路线图

### Phase 1：8-10 周（最小可用闭环）

**目标**：完成第一个客户 Demo / Pilot；能力全部沉淀到 Core。

| 周次 | 模块 | 关键交付 |
|---|---|---|
| W1-2 | Epic 01 InboundEvent + EmailProvider | inbound_event 表、EmailProvider（IMAP 拉取）、Webhook 入通道、自动回复 |
| W2-3 | Epic 02 Customer 5 实体 | customer / contact / identity / service_contract / service_entitlement 表 + Resolver |
| W3-4 | Epic 08 Durable Timer 复用 | 9 个新 CommandType 注册 + Delay/Every/Cancel helper |
| W4-5 | Epic 06 OnCall 8 实体 | team / schedule / shift / rotation / override / policy / level + `/oncall/current` API |
| W5-6 | Epic 07 NotificationProvider + ACK | ack 实体 + 3 个 Provider + Escalation Policy 引擎 |
| W6-7 | Epic 09 Timeline + Worklog | IncidentEvent 扩展 + IncidentWorklog 实体 |
| W7-8 | Epic 10 Closure Gate | 关闭门禁 + Incident→Problem/KnownError/Knowledge/Change 联通 |
| W8-9 | Epic 11 5 个 AI Skill | CustomerEntityExtraction / KnowledgeRecommendation / RCA Assist / IncidentReport / ShiftHandover |
| W9-10 | Epic 12 Incident Report | 报告生成 + HTML/PDF/MD 导出 + 模板引擎 |

**Phase 1 验收**：
- Email 报障 → 自动回复工单号 → 客户资格验证 → 排班 → 通知 → ACK → 升级 → 处理 → Worklog → RCA → 关闭 → 报告
- 全链路 Timeline 完整
- AI 调用 100% 审计
- 单元测试覆盖率 ≥ 80%
- E2E 测试关键路径 ≥ 1 条

### Phase 2：医院统一告警（6-8 周）

**目标**：医院场景增量；CMDB 影响分析落业务。

| 周次 | 模块 | 关键交付 |
|---|---|---|
| W1-2 | Epic 03 Alert Gateway | AlertProvider SPI + Webhook/Prometheus/Zabbix 3 个 Provider |
| W2-3 | AlertGroup + AlertEvent + AlertSuppressionRule | 去重聚合 + 抑制规则 |
| W3-4 | Epic 04 Event Intelligence | Normalization + Dedup + Suppression + Rule-based Correlation |
| W5-6 | Epic 05 CMDB Enrichment | ResourceMatcher + DynamicPriority |
| W6-7 | Epic 13 NOC Workspace | 前端大屏 + 聚合 API |
| W7-8 | SLA 风险 + 告警爆炸下的降级 | `sla_remaining_minutes` + 告警风暴策略 |

**Phase 2 验收**：
- 1 万 Alert 收敛到几十 Incident
- 灰盒 P1/P2 自动决策
- NOC 大屏 1 秒内响应

### Phase 3：高级事件智能（4-6 周）

**目标**：可扩展可分析能力。

| 模块 | 关键交付 |
|---|---|
| Epic 04 Stage 2 | Topology Correlation + Time-window Correlation |
| Epic 04 Stage 3 | AI Semantic Correlation Skill |
| AI 高级 | ShiftHandover Skill 正式启用 |
| IM Provider 扩展 | WeChat Provider / Feishu Incoming（双向） |
| Voice 接入 | 实体语音 Provider（供应商对接） |
| Analytics | 告警分析、值班负载分析、Upgrade SLA 报表 |

---

## 6. AI Skill 实施清单

| Skill | Code | 复用现有 | 第一阶段 |
|---|---|---|---|
| Triage | `ai.triage` | ✅ 已实现 | — |
| Chat | `ai.chat` | ✅ 已实现 | — |
| KnowledgeSearch | `ai.knowledge.search` | ✅ 已实现 | — |
| Summarize | `ai.summarize` | ✅ 已实现 | — |
| Analyze | `ai.analyze` | ✅ 已实现 | — |
| Analytics | `ai.analytics` | ✅ 已实现 | — |
| TrendPrediction | `ai.trend_prediction` | ✅ 已实现 | — |
| CreateTicket | `ai.create_ticket` | ✅ 已实现 | — |
| AgentTool | `ai.agent_tool` | ✅ 已实现 | — |
| Metrics | `ai.metrics` | ✅ 已实现 | — |
| Feedback | `ai.feedback` | ✅ 已实现 | — |
| **CustomerEntityExtraction** | `ai.customer.entity_extraction` | 🆕 新增 | P1 |
| **KnowledgeRecommendation** | `ai.knowledge.recommend` | 🆕 新增 | P1 |
| **RCAAssist** | `ai.rca.assist` | 🆕 新增 | P1 |
| **IncidentReport** | `ai.incident.report` | 🆕 新增 | P1 |
| **ShiftHandover** | `ai.incident.shift_handover` | 🆕 新增 | P3 |

**AI Audit 字段**（需求 § 11）：
- Model（default = LLMGateway.Default）
- PromptVersion（manifest 版本）
- InputRef（incident_id / ticket_id / alert_id）
- InputHash（sha256）
- Output（脱敏后）
- Confidence（0-1）
- Accepted/Rejected（人工）
- Timestamp + LatencyMs

**审计表**：`ai_audit_record`（可基于现有 `tool_invocation.go` 扩展）。

---

## 7. DTO / API 规范（camelCase 约束）

所有 DTO 必须使用 camelCase；Mapper 实现 snake→camel 转换。

### 7.1 新增 DTO 命名约定

| 实体 | DTO | 命名 |
|---|---|---|
| InboundEvent | `inbound_event_dto.go` | `InboundEventResponse` |
| Customer | `customer_dto.go` | `CustomerResponse` / `CustomerRequest` |
| AlertGroup | `alert_group_dto.go` | `AlertGroupResponse` |
| AlertEvent | `alert_event_dto.go` | `AlertEventResponse` |
| OnCallTeam | `oncall_dto.go` | `OnCallTeamResponse` / `OnCallCurrentResponse` |
| EscalationPolicy | `escalation_policy_dto.go` | `EscalationPolicyResponse` |
| IncidentWorklog | `incident_worklog_dto.go` | `IncidentWorklogResponse` |
| IncidentReport | `incident_report_dto.go` | `IncidentReportResponse` |
| ShiftHandover | `shift_handover_dto.go` | `ShiftHandoverResponse` |
| NOCSnapshot | `operations_dto.go` | `NOCSnapshotResponse` |

### 7.2 路由注册

所有新路由走 `router/router.go`；按领域前缀 `oncall / alert / ai-incident / operations` 拆分。

---

## 8. 前端落地

### 8.1 新增页面

| 页面 | 路径 | 关键组件 |
|---|---|---|
| NOC Workspace | `/operations/noc` | `NocWorkspace.tsx` |
| OnCall 排班 | `/admin/oncall` | `OnCallSchedule.tsx` / `OnCallTeamEditor.tsx` |
| Customer 管理 | `/admin/customers` | `CustomerList.tsx` / `CustomerDetail.tsx` |
| Alert Group | `/operations/alert-groups` | `AlertGroupList.tsx` |
| Incident Report | `/incidents/[id]/report` | `IncidentReportViewer.tsx` |
| Shift Handover | `/operations/shift-handover` | `ShiftHandoverView.tsx` |

### 8.2 复用组件

- `BusinessPageTemplate`（已有）
- `UnifiedKanbanBoard`（已有）
- `BatchActionBar`（已有）

### 8.3 新增 API Client

- `src/lib/api/oncall-api.ts`
- `src/lib/api/alert-group-api.ts`
- `src/lib/api/customer-api.ts`
- `src/lib/api/operations-noc-api.ts`
- `src/lib/api/incident-worklog-api.ts`
- `src/lib/api/incident-report-api.ts`
- `src/lib/api/shift-handover-api.ts`

类型对齐：
- `IncidentStatus` 新增 6 个 enum value
- `IncidentPriority` 替换为 `P1/P2/P3/P4`
- `AckStatus` enum

---

## 9. 验证与测试策略

### 9.1 单元测试（Phase 1 强制）

- 所有新增 Service 方法 ≥ 80% 行覆盖
- 所有 Handler ≥ 70% 行覆盖
- 业务规则（状态机、优先级、关闭门禁）必须 100% 覆盖
- 表驱动测试 + `enttest.NewClient()`

### 9.2 集成测试

- Command Bus 流转（`commandbus` 已具备 harness）
- Notification 全链路（mock Provider）
- Customer Resolver 6 种状态
- OnCall Schedule 时间计算（含节假日、Override、跨时区）

### 9.3 E2E（必须）

- 邮件 → Incident → 工程师 → ACK → 升级 → 解决 → 报告端到端
- 1 万 Alert → 几十 Incident 收敛
- Service 重启 / Worker Crash 不丢任务

### 9.4 回归覆盖

- 现网 Incident CRUD 不破坏
- 现有 `IncidentStatus*` 状态机兼容
- 现有 `IncidentEscalationRule` 兼容（默认规则保留）

---

## 10. 风险与缓解

| 风险 | 影响 | 缓解 |
|---|---|---|
| 状态机扩展破坏现有数据 | 高 | 仅新增状态；老数据保留 `new/acknowledged/...`；新状态只用于新流程 |
| OperationalCommand 压力 | 中 | 复用已有 Partition + Worker 池；按 command_type 分桶 |
| AI 调用成本失控 | 中 | 强制 PromptVersion + 限速 + AcceptanceRate 监控 |
| 客户身份解析歧义 | 高 | 仅返回 VERIFIED / NOT_FOUND / AMBIGUOUS / MANUAL_REVIEW；AI 仅给候选 |
| 排班冲突 | 中 | 唯一索引 + 事务校验 + Override 优先级 |
| 通知风暴 | 中 | Suppression 规则 + Tenant 级 rate limit |
| E2E 测试不稳定 | 中 | 重用项目 Playwright harness；固化样本 |
| 双框架（Connector + AlertProvider）分裂 | 中 | 明确边界：Connector 管"出站 + 入站接收"，AlertProvider 管"监控告警解析"；二者不重叠 |

---

## 11. 文件清单（按 Epic 分配）

### 11.1 Backend (`itsm-backend/`)

```
ent/schema/
  inbound_event.go                  # Epic 01
  customer.go                       # Epic 02
  customer_contact.go               # Epic 02
  customer_identity.go              # Epic 02
  service_contract.go               # Epic 02
  service_entitlement.go            # Epic 02
  alert_group.go                    # Epic 03
  alert_event.go                    # Epic 03
  alert_provider_config.go          # Epic 03
  alert_suppression_rule.go         # Epic 03
  alert_normalization_map.go        # Epic 04
  alert_correlation_rule.go         # Epic 04
  oncall_team.go                    # Epic 06
  oncall_schedule.go                # Epic 06
  oncall_shift.go                   # Epic 06
  oncall_rotation.go                # Epic 06
  oncall_member.go                  # Epic 06
  oncall_override.go                # Epic 06
  escalation_policy.go              # Epic 07
  escalation_level.go               # Epic 07
  incident_acknowledgement.go       # Epic 07
  incident_worklog.go               # Epic 09
  incident_report.go                # Epic 12
  incident_report_template.go       # Epic 12
  shift_handover.go                 # Epic 14
  ai_audit_record.go                # Epic 11

domain/
  inbound/                          # Epic 01
    provider.go
    email_provider.go
    webhook_provider.go
    mock_provider.go
  customer/                         # Epic 02
    resolver.go
    verification_status.go
  alert/                            # Epic 03 / 04
    provider.go
    alert_group_service.go
    normalization_service.go
    dedup_service.go
    suppression_service.go
    correlation_service.go
  oncall/                           # Epic 06
    schedule_resolver.go
    snapshot_service.go
  incident/                         # Epic 07 / 09 / 10 / 12
    ack_service.go
    worklog_service.go
    closure_gate.go
    report_service.go
  shift_handover/                   # Epic 14
    handover_service.go

handlers/skill/                     # Epic 11
  customer_entity_extraction_skill.go
  knowledge_recommendation_skill.go
  rca_assist_skill.go
  incident_report_skill.go
  shift_handover_skill.go

internal/commandbus/                # Epic 08
  incident_command_types.go         # 9 个新 CommandType

service/
  durable_timer.go                  # Epic 08 - Delay/Every/Cancel
  incident_dynamic_priority.go      # Epic 05
  incident_resource_matcher.go      # Epic 05
  ai_audit_service.go               # Epic 11
```

### 11.2 Frontend (`itsm-frontend/src/`)

```
app/(main)/
  operations/
    noc/page.tsx                    # Epic 13
    alert-groups/page.tsx           # Epic 03
    shift-handover/page.tsx         # Epic 14
  admin/
    oncall/page.tsx                 # Epic 06
    customers/page.tsx              # Epic 02
  incidents/
    [id]/report/page.tsx            # Epic 12
components/
  operations/
    NocWorkspace.tsx
    AlertGroupList.tsx
    OnCallSchedule.tsx
    ShiftHandoverView.tsx
  incident/
    IncidentWorklog.tsx
    IncidentAckButton.tsx
    IncidentReportViewer.tsx
lib/api/
  oncall-api.ts
  alert-group-api.ts
  customer-api.ts
  operations-noc-api.ts
  incident-worklog-api.ts
  incident-report-api.ts
  shift-handover-api.ts
```

---

## 12. 决策记录（ADR 摘要）

### ADR-01：Durable Timer 复用 OperationalCommand

**结论**：不新建 DurableTimer 框架；OperationalCommand 100% 覆盖需求 8 全部能力。

**依据**：
- 已有 Lease / Fencing / Heartbeat / Retry / DLQ
- 已有 `EnqueueSQLTx` 支持业务事务原子入箱
- 已有 7 个 CommandType + 11 个 Worker 模板

**替代方案**：
- ❌ 引入 BullMQ / Temporal / Asynq — 引入新组件
- ❌ 自建 DurableTimer 表 — 重复建设

### ADR-02：Customer 实体独立于 Tenant

**结论**：`Customer` 是 Tenant 的"客户"，一个 Tenant 可服务多个 Customer；Customer 多对多 Tenant（MSP 场景）。

**依据**：
- MSP 模式：一家 IT 服务商服务多家客户
- 客户身份、合同、权益需独立治理
- 现有 `tenant` 是"ITSM 部署租户"；Customer 是"被服务方"

### ADR-03：IncidentEscalationRule 升级而非替换

**结论**：保留旧表（向后兼容），新增 `escalation_policy` + `escalation_level`；Service 层优先新表，fallback 旧表。

**依据**：
- 现网数据不可破坏
- 新需求支持多级（initial / 2 / 3 / supervisor），旧表 schema 难扩展
- 灰度过渡：Service 层做 routing

### ADR-04：AlertProvider 走 Connector 还是独立？

**结论**：AlertProvider **独立**于 Connector，但**复用 Connector 注册机制风格**。

**依据**：
- Connector 偏"出站消息"语义；AlertProvider 偏"告警解析"
- 接收频率 / 数据结构差异巨大（Alert 万级/s vs IM 几十/s）
- 复用风格：Manifest + Registry + Factory + HealthCheck

### ADR-05：状态机不删除旧状态

**结论**：新增 6 个状态（旧数据继续可用），迁移路径通过 Service 层转换。

**依据**：
- AR-7（业务逻辑必须跑在 Service 层）
- 不破坏现有 100+ 客户数据
- 新流程仅在 Phase 1/2/3 全程开启后逐步迁移

### ADR-06：AI 不做不可逆决策

**结论**：AI 仅用于 Entity Extraction / 推荐 / 总结；状态转换 / 升级 / 关闭 / 派单 全部走规则引擎。

**依据**：
- AR-2（Rule First, AI Second）
- 客户合规要求（ITIL 审计）
- 现有 `ai_audit_record` 字段足够追溯

---

## 13. 文档更新（后续 Sprint）

- `CLAUDE.md`：补充 Incident Operations Platform 章节
- `docs/architecture/incident-operations.md`：架构图 + 数据流
- `docs/runbooks/incident-operations.md`：值班手册
- `README.md`：补充 Phase 1 视频 / 截图
- `CHANGELOG.md`：按 Epic 标注 Phase 1 / 2 / 3

---

## 14. 验收 Check List（开发生命周期）

每个 Epic 完成后必须：

- [ ] Ent schema 生成（`go generate ./ent`）
- [ ] Migration 文件生成（`go run -tags migrate main.go`）
- [ ] DTO + Mapper 单测 ≥ 80%
- [ ] Service 层表驱动测试 ≥ 80%
- [ ] Handler 单元测试
- [ ] 路由注册（`router/router.go`）
- [ ] Frontend API client + 类型
- [ ] Frontend 页面（如涉及 UI）
- [ ] 审计落 `incident_event` / `ai_audit_record`
- [ ] Connector / Skill 注册（如涉及）
- [ ] OpenAPI 文档（若采用 swag）
- [ ] i18n 键（中文 + 英文）
- [ ] RBAC 校验（每个 endpoint 必走 ACL）
- [ ] Tenant 隔离校验
- [ ] E2E 至少 1 条

---

**Phase 1 启动条件**：
1. 本蓝图评审通过
2. ADR 全部确认
3. 现有 `incident_escalation_rule` 兼容性验证用例准备
4. 容量评估（OperationalCommand 吞吐、邮件 IMAP 频率）

**蓝图状态**：v1.0 / 等待用户 Review
**维护人**：Incident Operations Platform Squad
**联系**：见 `CLAUDE.md` 团队结构
