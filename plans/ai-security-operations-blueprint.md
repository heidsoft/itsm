# AI 攻击常态化下的 ITSM 安全运营蓝图

> **Status**: draft（设计蓝本，未进入实施）
> **Date**: 2026-10-22
> **Scope**: 新增「安全运营」域（security operations），覆盖安全事件响应、AI 资产暴露面、处置动作治理、攻击知识沉淀、研判辅助与安全度量。
> **Out of scope**: SIEM / EDR / NDR / 沙箱 / 威胁情报采集引擎本身（一律通过 connector 对接，不自研）。
> **下游**: [security-operations-feature-design.md](./security-operations-feature-design.md)（F1–F10 可实施功能规格，含授权平面与数据模型增量）。

---

## 0. 结论先行

**ITSM 不该去做一个更小的 SOC 平台，该去做 SOC 缺的那一层：把「已确认的攻击」变成「可审计、可授权、可复盘的企业处置流程」。**

三条判断：

1. **瓶颈不在检测，在「检测之后的决策与执行」。** 平均单条告警调查 75 分钟、平均 28% 告警从未被调查、60% 的组织承认「被忽略的告警后来变成了实质安全事件」；同时攻击者 breakout time 已压到 29–48 分钟量级（[Prophet Security《State of AI in Security Operations 2026》](https://cybernoz.com/what-the-data-says-about-ai-in-security-operations-in-2026) / [SANS 2026](https://www.sans.org/white-papers/ai-human-collaboration-modern-socs)）。告警到结论 2 小时 vs 攻击者 29 分钟——这是**遏制问题，不是效率问题**。而遏制动作（隔离主机、禁用账号、下线模型、封禁出口）在企业里从来不是分析师能一个人拍板的事，它天生是**流程 + 审批 + 影响面 + 审计**——正是 ITSM 的地盘。
2. **企业最缺的不是"更快的自动化"，而是"自动化被授权的边界"。** 同一份调研里，57% 要求每条 AI 决策都要人工复核后才能关闭告警，44% 只让 AI 给建议、30% 只允许低风险自动处置，**没有任何一家给 AI 完全无人监督的自主权**。这是一份非常明确的规格说明书：市场要的是 *AI 建议 + 人授权 + 低风险自动 + 全量留痕*，而最后一项（全量留痕）是 ITSM 的既有强项。
3. **AI 攻击新增了一类必须进 CMDB 的资产：AI 自己。** prompt 资产、RAG 语料、模型端点、MCP/工具连接器、Agent 运行时——它们被投毒、被注入、被越权调用时，**唯一能回答"影响谁"的东西是配置项图谱**。ServiceNow 做 SecOps 的抓手正是 CMDB，这条路我们已经铺了三年（13 种关系词表 + 影响分析 + 云资源归一）。

因此本蓝图的产品定位一句话：

> **Security Response on ITSM** —— 不是替代 SOC，是给 SOC 补上"企业级处置的流程、授权与证据链"，同时把 AI 资产纳入配置治理。

---

## 1. 现场判断：攻击者的成本结构变了

| 变化 | 对防守方的直接后果 | 对产品的要求 |
|:---|:---|:---|
| 攻击成本下降、变种速度上升 | 告警量超出人力吸收能力（>25% 团队日收 500+） | 聚合去噪 + 一事件多告警，而非一告警一工单 |
| phishing 中约 80% 含 AI 生成内容 | 社工话术质量高，人工经验贬值 | 上报入口要极简（员工一键上报）+ 处置手册检索 |
| breakout time 压到分钟级 | 遏制窗口小于审批时延 | 处置动作要分风险等级：低风险预授权自动执行 |
| 第三方模型 / MCP / connector 成为供应链 | 供应商风险与变更风险叠加 | AI 资产进 CMDB，走变更审批与供应商台账 |
| 检测规则因人力不足被关闭（40% 组织） | 覆盖缺口被"合规地"制造出来 | 度量必须暴露"因资源关闭的检测面"，而不是只报 MTTD |

---

## 2. 为什么落点是 ITSM（能力错配分析）

| 安全响应必需能力 | SIEM / SOAR 现状 | ITSM 现状 | 结论 |
|:---|:---|:---|:---|
| 告警接收与去重 | ✅ 强 | 🟡 有 `alert` 表 + 入站端点，语义偏运维 | 复用 + 扩展，不重写 |
| 处置动作编排 | ✅ 强（playbook） | 🔴 无 | **不自研编排引擎**，playbook 走 BPMN |
| 变更/紧急变更审批 | 🔴 弱（无企业审批模型） | ✅ 强（CAS + 审批链 + 原子性） | **ITSM 独占优势** |
| 资产影响面 | 🔴 弱（缺 CMDB） | ✅ 强（CI 图谱 + 影响分析） | **ITSM 独占优势** |
| 处置 SLA 与升级 | 🟡 弱 | ✅ 强（多级 SLA + 升级矩阵 + 值班表） | **ITSM 独占优势** |
| 处置知识沉淀 | 🟡 弱（手册散落） | ✅ 强（知识库 + 评审状态机 + RAG） | **ITSM 独占优势** |
| 不可否认审计 | 🟡 有日志 | ✅ 强（审计域 + AI 决策留痕 + 行级数据范围） | **ITSM 独占优势** |

**结论：五项独占优势，缺的只有"安全事件的语义模型"。** 这正是本蓝图要补的唯一新东西。

---

## 3. 现状盘点（真实坐标，避免重复造轮子）

| 能力 | 现有资产 | 复用方式 |
|:---|:---|:---|
| 外部告警入站 | `POST /api/v1/alerts/sources/:source/ingest`（`router/alert_routes.go`，权限 `alert:write`） | 作为 EDR/SIEM/WAF/云审计的接入面，不新建管道 |
| 告警模型 | `ent/schema/alert.go`：`source` / `external_alert_id` / `severity` / `labels` / `annotations` / `raw_payload` / `fired_at` | 够用；新增仅"归并到的安全事件"外键 |
| 威胁/事件流程 | `handlers/incident`（状态机）+ `handlers/bpmn`（流程实例、任务分权、变量持久化） | 安全事件流程走 BPMN，不写第二套状态机 |
| 资产图谱 | `ent/schema/{citype,configurationitem,ci_relationship}.go`，CI 类型 `attribute_schema` 租户可自定义，13 种受控关系词表，影响分析 | AI 资产用**新增 CIType**，不改 CI 主表 |
| 时效约束 | `handlers/sla` + `sla_template` + `escalation_matrix` + `on_call_schedule` | 直接绑定；安全事件用独立 SLA 模板 |
| 处置审批 | `handlers/change` 状态机 + 审批链 + `WorkflowTx` 同事务（桥接 + 状态 CAS + 审批链收口） | 高危处置动作复用该事务骨架 |
| 知识沉淀 | `handlers/knowledge`（评审状态机：draft→under_review→approved 自动发布、RAG 索引同事务） | ATT&CK 标注 + 处置手册 |
| AI 辅助 | `service.SkillRegistry` + `handlers/ai/builtin_skills.go` 注入模式 + AI 审计日志端点 | 研判能力做成 skill，走同一审计面 |
| 行级授权 | `handlers/common/datascope`（单一源）+ `internal/authz`（catalog/roles 唯一手写源）+ `cmd/authz-gen` + 5 道守卫 | 新域必须同批进 ResourceActionMap 与生成物 |

🔴 **风险提示（来自本项目历史教训）**：新域一旦绕过 `ResourceActionMap` 声明，写路由守卫会静默放过；一旦在 `datascope` 之外自建过滤，行级权限就出现第二套真相。本蓝图所有新增表**必须**同批完成：`tenant_guard` 登记、权限码进 `internal/authz/catalog.go`、`go run ./cmd/authz-gen`、5 道守卫全绿。

---

## 4. 功能设计

> **前置环节见 [exposure-answer-design.md](./exposure-answer-design.md)（S0）**：漏洞型攻击（如 PHP2Shell）的第一个动作不是"开事件"，而是"回答我们是否暴露"。S0 产出四态暴露面结论，`confirmed` 且无法按期修复才升级为 4.1 的安全事件。本节 S1–S5 承接 S0 之后。

### 4.1 P0-A 安全事件域（`security_incident`）

**为什么不能塞进 incident**：三处语义冲突无法兼容——① SLA 口径不同（安全事件按"遏制时间"而非"解决时间"计）；② 可见性要求不同（涉及未修复漏洞、受影响客户名单，普通技术员不应可见）；③ 生命周期不同（`detected → triaged → contained → eradicated → recovered → closed`，与 incident 的 ack/resolve 不对齐）。

**模型**

```
security_incident
  incident_number      SEC-YYYY-NNNN（全局唯一序列，同 ci_number 做法）
  title / summary
  category             受控词表：phishing / account_takeover / malware / ransomware /
                       prompt_injection / data_poisoning / ai_supply_chain /
                       unauthorized_access / insider / ddos / other
  attack_tactic        受控词表，对齐 MITRE ATT&CK 战术级（14 项，不做 technique 全量）
  severity             critical / high / medium / low
  status               见上状态机
  detected_at / triaged_at / contained_at / recovered_at / closed_at
  detection_source     edr / siem / waf / cloud_audit / employee_report / ai_monitor
  affected_ci_ids      []int（首跳资产，展开由 CMDB 关系实时算，不落冗余）
  attacker_observable  文本：攻击者可观测面（用于复盘，不对普通用户展示）
  data_exposure        none / suspected / confirmed（决定合规上报路径）
  containment_state    open / partial / contained
  tenant_id / created_at / updated_at

security_alert_link         安全事件 ↔ 告警（M2M，多告警归并为一事件）
  security_incident_id / alert_id / tenant_id / linked_at / linked_by / reason

containment_action          处置动作台账（见 4.3）
  security_incident_id / action_type / target_type / target_ref
  execution_mode       auto / manual / approved
  state                proposed / approved / rejected / executing / succeeded / failed / rolled_back
  idempotency_key      唯一，防重复执行
  approved_by / approved_at / executed_by / executed_at
  rollback_ref / evidence_ref
  tenant_id

security_incident_visibility  额外可见性（见 §5.3）
  security_incident_id / subject_type（user/group/role）/ subject_id / level
```

**路由（新域 `handlers/security/`）**

```
POST   /api/v1/security/incidents                  security_incident:write
GET    /api/v1/security/incidents                  security_incident:read
GET    /api/v1/security/incidents/:id              security_incident:read
PATCH  /api/v1/security/incidents/:id              security_incident:write
POST   /api/v1/security/incidents/:id/transition   security_incident:write   （CAS + 状态机校验）
POST   /api/v1/security/incidents/:id/alerts       security_incident:write   （归并告警）
POST   /api/v1/security/incidents/:id/containment  containment:request       （提出处置动作）
POST   /api/v1/security/containment/:id/approve    containment:approve
POST   /api/v1/security/containment/:id/rollback   containment:approve
GET    /api/v1/security/incidents/:id/timeline     security_incident:read
```

**关键实现约束**

- 状态推进必须 **CAS + 业务拒绝不返 500**：`errors.Is` → 409/403/404，出口统一 `common.RespondError`（历史上 `FailWithErr` 覆盖 AppError 文案踩过两次）。
- 事件号生成走原生 SQL 序列，不靠 ent 唯一约束冲突（🔴 事务内吞 `IsConstraintError` 会让 PG 整条事务 aborted）。
- `detected_at` 由**调用方注入**，不用 `time.Now()`（可测 + 补录场景）。

---

### 4.2 P0-B AI 资产图谱（CMDB 扩展，差异化最强的一项）

攻击面无差别扫描已经过时，AI 攻击是**定向**的：谁会往里塞 prompt，谁就有价值。企业今天普遍**不知道自己有几个 Agent、它们调了哪些工具、语料从哪来**。这个问题 CMDB 天生能答。

**新增 CIType（数据，不是代码）**

| CI 类型 | 关键属性（`attribute_schema`） | 为什么必须是 CI |
|:---|:---|:---|
| `ai_application` | 入口 URL、认证方式、面向范围 | 被注入的第一落点 |
| `ai_agent` | 运行时、权限范围、自主等级（建议/受限自动/全自动）、可调用工具清单 | 越权工具调用 = confused deputy |
| `ai_model_endpoint` | 供应商、部署形态（外部 API / 私有推理）、模型版本、区域 | 供应链与数据出境 |
| `ai_tool_connector` | 协议（MCP/HTTP）、目标系统、权限凭据轮换周期 | 最常被忽略的越权跳板 |
| `prompt_asset` | 归属应用、版本、是否含系统指令、评审状态 | 被篡改即行为改变 |
| `rag_corpus` | 来源、更新频率、完整性校验值、可见性范围 | 投毒目标 |

**新增 2 条受控关系**（现 13 条，扩到 15 条，需同批改 `ent/schema.CIRelationshipTypeVocabulary` + 前端 `relationship-vocabulary.ts` 三处对齐）：

- `uses_prompt`：`ai_application → prompt_asset`
- `retrieves_from`：`ai_agent → rag_corpus`

**直接产出（免费得到的能力）**

- **爆炸半径计算**：给定一台被攻陷的主机 / 一个被投毒的语料 / 一个被劫持的 connector，一次图遍历回答"哪些 AI 应用、哪些业务、哪些 SLA 受影响"。
- **变更影响面**：改 prompt / 换模型版本走标准变更，审批时自动贴影响面（复用已有影响分析）。
- **暴露面看板**：`autonomy_level = 全自动` 且 `connector 权限 ⊇ 写权限` 的 Agent 清单——这是 CISO 最想看的一张表，而且市场上几乎没有产品能直接给。

---

### 4.3 P0-C 处置动作门禁（`containment_action`）

这是整套设计里**最需要克制**的部分。分析师要 30 秒隔离主机，但"谁有权隔离生产数据库主机"不能由分析师决定。

**风险分级（配置化，不硬编码）**

| 级别 | 例 | 通道 | 审计 |
|:---|:---|:---|:---|
| L0 低风险 | 加告警标签、拉黑单一 IP（单次 ≤1h）、只读快照 | **预授权自动执行** | 事后审计 + 可回滚 |
| L1 中风险 | 禁用账号、隔离办公终端、封禁出口 IP | 单人审批（on-call 安全负责人） | 执行前后快照 |
| L2 高风险 | 隔离生产数据库/核心链路、下线 AI 应用、吊销 connector 凭据、回滚模型版本 | **双人审批（SoD：申请人 ≠ 审批人）** + 影响面必填 | 全量证据链 |
| L3 极高 | 全局断网、批量账号冻结 | 走**紧急变更**流程（复用 change 状态机） | 事后 PIR |

**实现要点**

- 动作执行器 **connector 化**：`builtin:isolate_host` / `builtin:disable_account` / `builtin:revoke_token` / `builtin:disable_ai_app`，每个动作 = 参数 schema + 幂等键 + 回滚 schema。**禁止自由形式命令**（见 §5.1）。
- 执行与状态推进**同事务**优先，跨系统动作走 `commandbus` 幂等入队（不得依赖"先执行后写库"）。
- 动作必须**可回滚**才有资格进 L0 自动通道；没有回滚路径的一律 L2 起。
- Agent 身份调用：`actor_type=agent` 时 **L0 封顶**，L1+ 一律转人工（fail-closed）。这条直接堵住"被劫持的 agent 持有 technician 权限 = 超级用户"的路径。

---

### 4.4 P1-D 攻击知识域（ATT&CK 标注 + 处置手册）

复用知识库评审状态机与 RAG 索引同事务，新增两类标注：

- **战术/技术标注**：`attack_tactic`（复用 4.1 受控词表）+ `attack_technique`（自由文本 + 校验，不做全量枚举）。
- **处置手册结构**：`适用 CI 类型` + `适用攻击类型` + `前置条件` + `处置步骤` + `验证方法` + `回滚`。

**检索增强（这是 RAG 的真实价值点）**：告警进来时按 `attack_tactic + 受影响 CI 类型` 双过滤召回手册，而不是纯语义相似。历史教训：`rankByAuthority` 的元数据查询故意不带 `IsPublished`（只排序不准入），不要把它当缺陷改；同理安全手册的可见性必须走分类可见性，不能靠排序兜底。

**闭环指标**：手册被引用次数 → 处置成功率 → 手册版本迭代（作者 ≠ 审核人，沿用现有防自审 403/2003）。

---

### 4.5 P1-E AI 研判（受控建议，绝不作为指令）

三个 skill，全部走 `service.SkillRegistry` 注册 + AI 审计留痕：

| Skill | 输入 | 输出（**结构化 schema，非自然语言**） | 人的动作 |
|:---|:---|:---|:---|
| `security_triage` | 一批告警（已去重） | `{is_incident, category, tactic, confidence, dedup_group, evidence[]}` | 一键确认为安全事件 / 合并 |
| `blast_radius` | 受影响 CI | `{affected_cis[], affected_apps[], affected_slas[], path_evidence[]}` | 附到事件影响面 |
| `playbook_recommend` | 事件 + CI 类型 + tactic | `{playbook_id, steps[], required_actions[], risk_level}` | 提议处置动作（仍需走 4.3 门禁） |

🔴 **红线：LLM 输出永远不直接产生 `containment_action` 记录。** 它只能填 `proposed` 字段，且 `proposed` 的来源必须标记 `source=ai`，审批界面对 `source=ai` 的提议强制显示"AI 生成，请核对证据"。理由不只是合规——攻击者可以把指令藏进告警描述、工单正文、日志行里（间接 prompt injection），如果 AI 输出能直接触发隔离动作，**ITSM 就成了攻击者的执行器**。

---

### 4.6 P2-F 安全运营度量（四段计时）

不报"平均解决时间"这种钝指标，报四段：

```
detected → triaged  （研判时延）
triaged  → contained（遏制时延，核心）
contained → recovered（恢复时延）
recovered → closed  （复盘时延）
```

按 `category × severity × 资产关键度` 分桶，并**必须**同时暴露两个"负面指标"：

- **未研判告警数**（对应 28% 未调查的行业现实）
- **因资源关闭的检测面**（对应 40% 组织关闭规则——这个数字必须被看见，否则度量在自欺）

实现挂现有 dashboard KPI contract，不新开口径（历史教训：KPI 口径漂移是 docs-gate C.6 的检查项）。

---

## 5. 三条设计红线

### 5.1 AI 输出 = 不可信输入（防"ITSM 变成攻击执行器"）

| 环节 | 强制约束 |
|:---|:---|
| LLM 输入 | 告警文本 / 工单正文 / 邮件正文 / 日志行一律打 `untrusted` 标记，走结构化抽取，不拼指令 |
| LLM 输出 | 必须过 JSON schema 校验 + 字段白名单；自由文本字段不得进入任何执行路径 |
| 工具调用 | 白名单 + 参数化模板 + 幂等键；**不允许 LLM 生成命令字符串** |
| 决策 | AI 只能 `propose`，`approve` 必须是人（或 L0 预授权策略，且策略由人配置） |

依据同一份行业调研：57% 要求每条 AI 决策人工复核、44% 只接受 AI 建议、**零受访者允许完全自主**。这条红线不是保守，是市场规格。

### 5.2 处置动作必须"可授权、可回滚、可留痕"

- 无回滚路径 ⇒ 不得进自动通道（L0）。
- SoD：L2 以上申请人 ≠ 审批人，且审批链原子性复用 `WorkflowTx`（桥接 + 状态 CAS + 审批链收口同事务）。
- 每个动作落 `evidence_ref`（执行前后快照、命中的 playbook 版本、当时的审批记录）。

### 5.3 安全事件需要"额外可见性"，但不许开第二套过滤

现状：`handlers/common/datascope` 是行级权限单一源，tenant 隔离在 middleware。

**做法**：把 `security_clearance` 作为 DataScope 的**新维度**注入同一处（`level: none/restricted/full`），而不是在 security 域里自建 `WHERE`。
- 未授权用户读安全事件 ⇒ **404 而非 403**（不确认存在性，避免"探测式情报收集"）。
- 影响：该维度对非安全域默认 `full`，保证零行为变更；一旦有人绕过 datascope 写过滤，守卫必须红。

---

## 6. 数据模型增量汇总

| # | 表 | 类型 | 说明 |
|:--|:---|:---|:---|
| 1 | `security_incidents` | 新 | 安全事件主体 |
| 2 | `security_alert_links` | 新 | 事件↔告警 M2M（含 tenant_id） |
| 3 | `containment_actions` | 新 | 处置动作台账 |
| 4 | `security_incident_visibilities` | 新 | 额外可见性 |
| 5 | `security_incident_events` | 新 | 状态流转时间线（append-only，供 timeline 端点与审计） |
| 6 | `ci_types` | 数据 | 新增 6 类 AI 资产 CIType（租户级种子，非 schema 变更） |
| 7 | `ci_relationships` | 数据 | 关系词表 13 → 15 |
| 8 | `sla_templates` | 数据 | 安全事件专用模板（按遏制时间考核） |

**必须同批完成（否则守卫/隔离出洞）**
- `internal/authz/catalog.go` 新增 `security_incident:*` / `containment:*` / `security_asset:*` 权限码
- `go run ./cmd/authz-gen` 重新生成预检映射，5 道守卫全绿
- `internal/schema/tenant_guard.go` 登记 5 张新表
- 行级 `datascope` 覆盖读/写对称（写过 4 域的历史缺口，别重演）

---

## 7. 明确不做（防止范围爆炸）

| 不做 | 理由 | 替代 |
|:---|:---|:---|
| SIEM / 日志湖 | 重资产，红海 | connector 对接 Splunk/ELK/云审计 |
| EDR 探针 | 客户端工程，非本产品能力域 | 接 EDR API 做动作执行端 |
| 恶意样本沙箱 | 专用基础设施 | 接第三方沙箱，结果落事件证据 |
| 威胁情报采集引擎 | 数据合规与出境风险高 | connector + 客户自带 feeds |
| 自研 playbook 编排引擎 | 已有 BPMN，重复建设 | BPMN 流程 + 动作 schema |
| 漏洞扫描器 | 已有 `cloud discovery` 骨架，但扫描器是另一个产品 | 接扫描器结果，落 CI `attributes` |

一句话：**只做"决策与执行之间的那一层"，两侧都用 connector。**

---

## 8. 路线建议

| 阶段 | 内容 | 规模 | 前置 |
|:---|:---|:---|:---|
| **S0（前置）** | 暴露面自答：`software_instance` 对齐键 → advisory 归一 → 四态匹配 + 六问页（详见 [exposure-answer-design.md](./exposure-answer-design.md)） | L | 无（复用 CMDB/vendor/知识库），**且不依赖 S1 可独立上线** |
| **S1（最小闭环）** | 安全事件域 + 告警归并 + timeline + 四段计时埋点 | M | 无（复用 alert/BPMN/SLA） |
| **S2（授权落地）** | 处置动作台账 + 风险分级 + L0/L1/L2 门禁 + 2 个执行 connector | L | S1 + 权限码全链路 |
| **S3（图谱）** | 6 类 AI 资产 CIType + 2 条关系 + 暴露面看板 + 爆炸半径 skill | M | CMDB 已有能力，纯增量 |
| **S4（知识+研判）** | ATT&CK 标注 + 处置手册 + `security_triage` / `playbook_recommend` | M | 知识域评审状态机现成 |
| **S5（度量）** | 四段计时 + 负面指标 + 看板 | S | S1 |

**排期判断**：不建议塞进 v1.7（该版本已被 connector 生产化 + AI evaluator + i18n 占满，且全 UI 中文化是硬承诺）。建议作为 **v1.8「Security Response」独立线**，或先以 Pilot 域并行（S1+S3 可以完全独立于 S2 先行，因为 S3 只是数据，零风险）。

**验收口径**（不接受"功能已上线"这种说法）：
- S1：注入一条伪造 EDR 告警 → 归并成 1 个事件 → 状态机 CAS 拦下非法跃迁（返回 409 非 500）→ timeline 有完整三段记录
- S2：L2 动作在申请人=审批人时**必须 403**；无回滚 schema 的动作**必须**无法进 L0；执行 connector 故障时台账状态停在 `failed` 且可重放
- S3：投毒语料 CI → 一次图遍历列出全部受影响 AI 应用与其 SLA，且路径可解释
- 红线验证：构造"告警描述里藏指令"的样本 → `security_triage` 输出**不得**产生任何 `containment_action`

---

## 9. 风险与取舍

| 风险 | 判断 | 处置 |
|:---|:---|:---|
| 与专业 SOAR 撞车 | 低。SOAR 弱在企业审批/资产/审计，这正是我们的强项 | 定位写死在产品文案里，不喊"替代 SOC" |
| 安全团队不买 ITSM 的账 | 中高。安全与 IT 运维常是两个预算池 | 首发切"AI 资产暴露面"（CISO 关心且市场空白），而非"安全工单"（安全团队已有工具） |
| 可见性维度破坏行级权限单一源 | 中。项目历史证明自建过滤必出洞 | 作为 DataScope 新维度注入，守卫锁死 |
| AI 研判被投毒利用 | 高影响、低概率 | §5.1 四条强制约束 + 红线验收用例 |
| 范围爆炸 | 高 | §7 不做清单作为评审前置条件，任何新增须先删一项 |

---

## 附：一句话对外表述

> 企业不缺告警，缺的是"这条告警值得为它停掉一条生产线吗、谁签字、留什么证据"。
> ITSM 不抢 SIEM 的活，只把攻击变成一次**有授权、有影响面、有 SLA、有复盘**的企业级处置——
> 顺便，把你自己都数不清的 AI Agent 和 prompt 纳入配置管理。
