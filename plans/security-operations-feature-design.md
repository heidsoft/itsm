# 安全运营功能设计：F1–F10 功能规格

> **Status**：draft（功能规格，未进入实施）
> **Date**：2026-10-22
> **前置**：[ai-security-operations-blueprint.md](./ai-security-operations-blueprint.md)（S1–S5）、[exposure-answer-design.md](./exposure-answer-design.md)（S0）、[security-operations-product-deep-dive.md](./security-operations-product-deep-dive.md)（D1–D5、C1–C8）
> **本文定位**：把三份产品分析翻译为**可实施规格**。每个功能给出数据模型增量、API、授权平面动作、AI 边界、验收口径、依赖。凡是断言仓库现状的，均为 2026-10-22 实测坐标。

---

## 0. 一页结论

**产品缺的不是"安全角色"，是角色背后的整套语义——而角色本身已经空转很久了。**

三个实测事实：

1. `domain/role/role.go:32` 早已定义 `SecurityAdmin = "security_admin"`（Tier 1 内置主角色，`users.role` 枚举 + `roles` 种子表都在，legacy `"security"` 已退役并入它），`Rank()` 里它的位阶是 **80，高于 admin 的 70**——与 `audit_admin` 同档。
2. 但 `internal/authz/catalog.go` 里**大小写不敏感搜索 "security" 零命中**——这个角色今天没有任何权限码。59 个 handlers 域也没有任何安全运营域。
3. 与此同时，安全响应真正需要的零件全部现成：告警入站 `POST /alerts/sources/:source/ingest`（`router/alert_routes.go:12`）、`alert` schema、CMDB 13 种受控关系（`ent/schema/ci_relationship.go:57`）、change 审批 CAS + `WorkflowTx`、SLA + 升级矩阵 + 值班表、知识评审状态机、`SkillRegistry`（skill 注册带 `requiredPermissions` 约束）。

所以这套功能设计的本质是：**把一个空转的高位角色激活成一套完整的安全运营能力**，而不是新造一个安全产品。10 个功能分四期落地，P0 三件（F1 采得到 / F2 答得出 / F3 管得住）构成最小可用闭环。

---

## 1. 功能清单与分期

### 1.1 功能地图

| # | 功能 | 上游 | 一句话 | 分期 |
|:--|:---|:---|:---|:--:|
| F1 | 资产发现与软件实例对齐键 | C1–C8 + `software_instance` | 让"我们有什么"从猜变成有置信度的台账 | **P0** |
| F2 | 威胁公告归一与暴露面自答 | S0 | 公告发布后 30 分钟内答出"我们哪几台受影响" | **P0** |
| F3 | 安全事件域 | S1 | 已确认攻击的流程、授权、审计容器 | **P0** |
| F4 | 处置动作门禁 | S2 + D3 | 每个遏制动作可授权、可回滚、可留痕 | P1 |
| F5 | 统一暴露面实体 | D1 | 漏洞/配置/暴露/姿态/身份风险五类收敛排序 | P1 |
| F6 | 根因修复 | D2 | 修复挂镜像模板，不挂 300 台机器 | P1 |
| F7 | AI 资产图谱与 Agent 身份 | S3 + D4 | 爆炸半径 + 暴露面看板 + agent 也是身份 | P2 |
| F8 | 研判辅助与处置知识 | S4 | 三个只出结构化 JSON 的 skill + ATT&CK playbook | P2 |
| F9 | 安全运营度量 | S5 | 四段计时 + 两个负面指标 | P2 |
| F10 | 合规控制映射 | D5 | 控制项 ↔ 流程 ↔ 证据链 | P3 |

### 1.2 分期原则

- **P0 = 最小可用闭环**：F1（数据地基）→ F2（暴露面结论）→ F3（事件容器）。三者依赖弱，F1 完成后 F2/F3 可并行。
- **P1 = 从"知道"到"动手"**：F4（动作门禁）依赖 F3；F6 依赖 F1 的 C4 采集通道，**这是硬依赖**——拿不到镜像模板派生关系，D2 只能降级成任务式。
- **P2 = 纵深与规模**：F7/F8 依赖 F3/F4 的容器与门禁存在；F9 依赖 F3 的计时字段。
- **P3 = 治理外延**：F10 依赖全链路的证据沉淀。

🔴 **社区版铁律**（承接纵深文档的产品分层结论）：任何裁剪**不得破坏 F2 六问页答全**——"答得出来"是与"又一个开源 SOAR"的分界线。商业版"管得起来"（F4–F6），企业版"治得住"（F7–F10）。

---

## 2. 域划分与代码落点

| 新域 | 目录 | 内容 |
|:---|:---|:---|
| 安全事件 | `handlers/securityincident/` | `security_incident` + links + members + timeline + containment |
| 暴露面 | `handlers/exposure/` | advisory + observation + assessment + rollup + finding + solution |
| 复用 CMDB | `handlers/cmdb/` | `software_instance` + C1–C4 采集（`discovery_job_lifecycle.go` 已在此）+ AI CIType |
| 复用 AI | `handlers/ai/` | 三个研判 skill（`builtin_skills.go` 注册模式） |
| 复用知识 | `handlers/knowledge/` | playbook 评审状态机复用 |
| 复用 vendor | `handlers/vendor/` | 版本疑难仲裁通道 |

命名决策：不用 `handlers/security/`——与既有 `handlers/asset/` 边界模糊；`securityincident` 与 `incident`（ITIL 事件）在目录名上就明确切开，防止后来者把两者混为一谈（`ent/schema/incidentalert.go` 那张表是事件**通知**语义，不是告警归并，不能复用）。

前端落点：`src/app/(main)/security/`（事件列表/详情/处置/看板）、`src/app/(main)/exposure/`（六问页 + 评估）、台账页挂 `src/app/(main)/cmdb/` 下。

---

## 3. 授权平面总表

### 3.1 新增权限码（`internal/authz/catalog.go` → `Definitions()`）

按"独立资源面"原则（catalog 包注释的三条新增约束：独立资源面、有路由引用、有角色绑定）：

| 权限码 | 名称 | 用途 |
|:---|:---|:---|
| `security_incident:read` | 查看安全事件 | 列表/详情（叠加 §3.3 可见性档） |
| `security_incident:write` | 管理安全事件 | 创建/编辑 |
| `security_incident:triage` | 研判安全事件 | detected→triaged、dismiss |
| `security_incident:approve` | 审批安全处置 | L1/L2 审批 |
| `security_incident:close` | 关闭安全事件 | closed 终态 |
| `security_incident:export` | 导出安全事件 | |
| `security_incident:admin` | 安全事件配置 | |
| `exposure:read` / `exposure:write` / `exposure:review` / `exposure:export` | 暴露面 | review = 人工确认 assessment |
| `advisory:read` / `advisory:write` / `advisory:review` | 威胁公告 | review = `reviewed_by` 生效 |
| `containment:read` / `containment:execute` / `containment:approve` / `containment:admin` | 处置动作 | execute 与 approve **分离**（SoD） |
| `asset_discovery:read` / `asset_discovery:run` / `asset_discovery:admin` | 资产采集 | 采集任务 |
| `security_metrics:read` | 安全度量 | 四段计时 + 负面指标 |

### 3.2 角色绑定（`builtinRoleBindings`）

| 角色 | 绑定 | 理由 |
|:---|:---|:---|
| `security_admin` | 全部 read/triage/write/review/execute + `security_incident:approve` + `containment:approve` | **激活这个空转角色**——它就是安全团队负责人 |
| `super_admin` / `admin` | 全部 | |
| `manager` | `security_incident:read` + `security_incident:approve` | 业务侧影响面确认（L2 双人中的业务签） |
| 其余角色 | **零** | 不撒权限 |

为什么这样绑：`SecurityAdmin` **不在** `IsAdminLike` 词表（`domain/role/role.go:126-128`：super_admin/admin/manager/sysadmin/audit_admin），所以它不会搭管理角色的便车拿到全租户 DataScopeAll——安全团队的读范围必须显式设计（§3.3），这正是激活它的正确方式。若哪天把 security_admin 加进 IsAdminLike，它会顺带看到全租户工单与 CMDB，那是越权不是赋能。

### 3.3 路由与生成（每波次必做）

1. 新建 `router/securityincident_routes.go` / `router/exposure_routes.go` 等，逐路由挂 `middleware.RequirePermission("security_incident", "triage")`；
2. `go run ./cmd/authz-gen` 重新生成 `middleware/rbac_precheck_gen.go`；
3. 过守卫：`TestPrecheckMapIsFresh`、`TestRoutePrecheckAlignment`、`pkg/seeder/role_permission_guard_test.go`、`router/permission_code_catalog_guard_test.go`；
4. 🔴 POST 动作路由（`POST /security-incidents/:id/triage`）必须落在 `ResourceActionMap` 的 **POST 段**——写进 GET 段等于没放，这是仓库既有踩坑记录。

### 3.4 行级可见性：DataScope 组合，不建旁路

`security_incident.visibility` 两档：

| 档 | 可见范围 |
|:---|:---|
| `tenant` | 持 `security_incident:read` 且 DataScopeAll 角色（含 security_admin） |
| `restricted` | 仅 `security_incident_member` 成员 + super_admin/admin（涉未公开漏洞/客户数据泄露的事件） |

落地机制（对齐仓库单一源原则）：

- `internal/datascope/datascope.go` 新增 `ApplySecurityIncidentFilter(query, userID, role, ds)`，与既有 `ApplyTicketFilter`（107 行）同模式同包；
- `security_incident_member(incident_id, user_id, member_role)` 无 `tenant_id` → 必须登记 `internal/schema/tenant_guard.go`；
- 未授权读 → **返 404 不返 403**（不确认存在性）。

为什么不能直接放行 security_admin 读全部：它的读范围是"安全事件"，不是"全租户一切"。用权限码 + 成员表精确圈定，而不是改 DataScope 枚举语义。

---

## 4. 数据模型增量总表

### 4.1 新 ent schema（全部带 `tenant_id` + `created_at`/`updated_at`，遵循仓库模式）

| 表 | 关键字段 | 归属域 | tenant_guard 登记 |
|:---|:---|:---|:--:|
| `software_instance` | `ci_id` / `product` / `vendor` / `version` / `version_branch` / `deployment_sapi`(枚举) / `confidence`(confirmed/inferred/stale) / `source`(C1–C8) / `discovered_at` | cmdb | — |
| `threat_advisory` | `external_ids`(JSON，5 类编号 alias) / `title` / `cve_ids` / `affected_products`(JSON) / `severity` / `cvss` / `published_at` / `status`(draft/reviewed) / `reviewed_by` / `reviewed_at` | exposure | — |
| `advisory_source_observation` | `advisory_id` / `source` / `observed_at` / `raw_fields`(JSON) / `conflicts_with`(JSON) | exposure | — |
| `exposure_assessment` | `advisory_id` / `ci_id` / `status`(confirmed/likely/not_affected/**unknown**) / `matched_version` / `reason` / `assessed_by` / `assessed_at` | exposure | — |
| `advisory_exposure_rollup` | `advisory_id` / `confirmed_cnt` / `likely_cnt` / `not_affected_cnt` / `unknown_cnt` / `answer_latency_sec` | exposure | — |
| `security_incident` | `sec_code` / `title` / `severity` / `status`(7 态) / `detected_at` / `triaged_at` / `contained_at` / `closed_at` / `visibility` / `attack_techniques`(JSON) | securityincident | — |
| `security_alert_link` | `incident_id` / `alert_id` / `linked_by` / `link_reason` | securityincident | ✅ |
| `security_incident_member` | `incident_id` / `user_id` / `member_role` | securityincident | ✅ |
| `containment_action` | `incident_id` / `action_id`(connector 目录) / `risk_level`(L0–L3) / `params`(JSON Schema 校验) / `idempotency_key` / `status` / `rollback_ref` / `evidence_ref` / `approved_by` / `executed_by` | securityincident | — |
| `exposure_finding` | `type`(五类) / `subject_ref` / `risk_score` / `score_factors`(JSON) / `status` | exposure | — |
| `exposure_solution` | `finding_id` / `root_cause_ci_id` / `change_id` / `status` | exposure | — |
| `agent_identity` | `name` / `owner_user_id` / `autonomy_level` / `bound_connectors`(JSON) / `effective_permissions`(JSON，RBAC 派生) | securityincident | — |
| `control_mapping` | `framework` / `control_id` / `process_ref` / `evidence_query` | exposure | — |

### 4.2 既有 schema 增量

- `citype`：6 个 AI CIType 种子（`ai_application` / `ai_agent` / `ai_model_endpoint` / `ai_tool_connector` / `prompt_asset` / `rag_corpus`）进 `pkg/seeder`；
- `ent/schema/ci_relationship.go`：`CIRelationshipTypeVocabulary` **13 → 15**（新增 `uses_prompt` / `retrieves_from`），同步递增 `handlers/cmdb/ontology.go` 的 `cmdbOntologyVersion`（当前 `"2026-09-04.1"`，破坏性词表变更按注释约定必须递增，Agent 侧据此缓存失效）；
- `alert`：**不加字段**（source / external_alert_id / labels / raw_payload 已够），告警归并走 link 表；
- `security_incident.sec_code`：独立序列（不复用工单号——跨域污染编号空间是审计噪音）。

---

## 5. 功能详设

### F1 资产发现与软件实例对齐键（P0）

**用户故事**：运维登记"web-prod-03 跑 php-fpm 8.2.34"，48 小时内这条记录自动带上置信度与来源；打印机、工控网关进 unknown 桶转 vendor 仲裁，而不是被猜成"没影响"。

**采集通道**（按申请难度排序，不按技术先进性）：

| 层 | 通道 | 网络改造 | 优先级 |
|:--|:---|:---|:--:|
| C1 | 云 API（AWS/Azure/GCP describe） | 无，软件清单质量最高 | P0 |
| C2 | SNMP v2c/v3 只读轮询 | 单 UDP 端口，**固件版本主要来源** | P0 |
| C3 | LLDP/CDP 邻居表 | 无，交换机本来在广播 | P0 |
| C4 | 镜像 registry + k8s API | 无，天然对应 F6 的镜像模板 | P0 |
| C5/C6 | 交换机 MAC 表 + DHCP lease / mDNS-SSDP 组播 | 低 / 中 | P1 |
| C7/C8 | SPAN/TAP 全流量 / 主动探测 | 高 + 需审批 | P2，**主动探测默认关闭** |

C1–C4 零或低网络改造，且 `cloud_account` / `discovery_job` / `discovery_result` schema 与 `handlers/cmdb/discovery_job_lifecycle.go` 骨架已存在，直接扩展。指纹库不自建：OUI 用 IEEE/Wireshark 开源库，DHCP 指纹接 Fingerbank 社区库。

**API**：

| 方法 | 路径 | 权限 | 要点 |
|:---|:---|:---|:---|
| GET | `/api/v1/cmdb/software-instances` | `cmdb:read` | 按 product/version/confidence 过滤，返回 unknown 计数 |
| POST | `/api/v1/cmdb/software-instances` | `cmdb:write` | 手工登记——**对齐键的起点，先做这个** |
| POST | `/api/v1/cmdb/discovery-jobs/:id/run` | `asset_discovery:run` | 触发 C1–C4 采集 |
| GET | `/api/v1/cmdb/software-instances/unknown` | `cmdb:read` | 待仲裁队列 |
| POST | `/api/v1/cmdb/software-instances/:id/arbitrate` | `cmdb:write` | 人工裁定 → `confidence=confirmed` |

**三条硬规则**：

1. `confidence` 三态是必需品：`inferred` 参与匹配但暴露面结论降一级；`stale`（超过 N 天未复采）自动降 unknown；
2. `deployment_sapi` 必填枚举：`mod_php` / `fpm` / `cgi` / `embed` / `unknown`——同为 `php 8.2.34`，mod_php 是未认证单请求 RCE 路径，fpm 是确认路径，`unknown` 降级；
3. 版本区间比较按**分支语义**（分段数值比较），单测锁 `8.2.4 < 8.2.35`——字符串比较会因 `4 > 3` 漏掉最该报的资产。

**AI 边界**：无（纯采集归一，不调 LLM）。
**验收**：① 手工登记 → C4 采到同版本镜像 → confidence=confirmed ② 无 SNMP 打印机 → unknown 桶 + 仲裁任务 ③ `8.2.4` vs `8.2.35` 比较用例 ④ stale 自动降级 ⑤ 采集任务失败不影响既有 CMDB 读路径。
**依赖**：无（最先做）。

---

### F2 威胁公告归一与暴露面自答（P0）

**用户故事**：PHP2Shell 级公告发布后 30 分钟，安全负责人在一个页面答全六问：有没有影响 / 哪几台 / 什么版本 / 谁负责 / 多久打完 / 依据哪几家源。

**六问页**（`GET /api/v1/exposure/advisories/:id/answer`，权限 `exposure:read`）：

1. 是否受影响——四态汇总，**四个计数分别返回**；
2. 受影响资产清单——CI + 业务服务 + 负责人 + SLA；
3. 版本与 SAPI 明细；
4. 处置责任人与时限；
5. 待办——补丁变更 / 核查命令（`php -v`、`apachectl -M | grep php`）/ 仲裁；
6. 依据来源——多源观测 + 冲突标注。

**四态结论**：`confirmed` / `likely` / `not_affected` / `unknown`。🔴 `unknown` **独立成桶，禁止并入 not_affected**——把"我不知道"并进"我确认没影响"是漏洞管理最常见的自欺，rollup 表四列分别计数，unknown 单独派核查任务。

**AI 边界**（红线集中区）：

- LLM 只做公告结构化抽取，落 `threat_advisory` 草稿，**`status=draft` 直到 `reviewed_by` 人工确认才参与匹配**；
- 冲突只标注不裁决（`advisory_source_observation.conflicts_with`）——CERT-EU 与 BtCIRT 对 `memory_limit` 给出相反建议时，产品把两份都摆出来，不替用户选；
- 外部公告文本视同不可信输入：投毒一份公告正文诱导抽错版本区间 = 系统性漏报。抽取 prompt 做隔离标记，输出过 schema 白名单，**禁止 LLM 生成版本区间之外的任何内容**。

**实施顺序（反直觉但正确）**：先 F1 的 `software_instance`（哪怕只有手工导入），再 advisory 归一，最后匹配。反过来做，接十个 feed 也只会得到"我们可能有影响"——那正是今天的现状换了个界面。

**验收**（夹具即 PHP2Shell，10 条用例）：① 四态各自计数 ② `8.2.4` 不漏报 ③ sapi=mod_php 升 confirmed、sapi=unknown 降 likely ④ 嵌入式设备（版本串不可信）置 likely/unknown ⑤ 草稿公告不参与匹配 ⑥ **公告正文藏注入指令 → 抽取结果不改变任何资产结论** ⑦ `answer_latency_sec` 计时 ⑧ 冲突双列展示不互相覆盖 ⑨ unknown 桶派核查任务 ⑩ 六问页字段答全（缺任一 CI 关联即失败）。
**依赖**：F1。

---

### F3 安全事件域（P0）

**为什么不塞进 incident**：SLA 按"遏制时间"考核、可见性要额外隔离（§3.4）、生命周期是攻防语义而非 ITIL 语义。

**状态机**（7 态）：`detected → triaged → contained → eradicated → recovered → closed`，旁路 `dismissed`（误报关闭，必填理由）。

| 转换 | 门禁 |
|:---|:---|
| detected → triaged | 持 `security_incident:triage` + 至少一条 `security_alert_link` |
| triaged → contained | 至少一个 approved 且 executed 的 containment_action，**或**显式"无需处置"理由 |
| * → dismissed | 必填理由 |
| * → closed | 复盘记录（PIR）或误报理由 |

**API**：

| 方法 | 路径 | 权限 | 要点 |
|:---|:---|:---|:---|
| POST | `/api/v1/security-incidents` | `security_incident:write` | 从告警归并或 exposure confirmed 创建 |
| POST | `/api/v1/security-incidents/:id/alerts` | `security_incident:write` | 多告警归并一事件 |
| GET | `/api/v1/security-incidents/:id/timeline` | `security_incident:read` | 时序（复用 ticket_comment 模式） |
| POST | `/api/v1/security-incidents/:id/triage` | `security_incident:triage` | |
| POST | `/api/v1/security-incidents/:id/dismiss` | `security_incident:triage` | 误报 + 必填理由 |
| GET | `/api/v1/security-incidents/:id/impact` | `security_incident:read` | 受影响 CI + 业务 + SLA（复用 CMDB 影响分析） |
| POST | `/api/v1/security-incidents/:id/members` | `security_incident:write` | restricted 档成员管理 |

**复用清单**：告警入站 `POST /alerts/sources/:source/ingest` 不变；升级复用 `handlers/on_call` + `handlers/escalation_matrix`；重大事件升级走 BPMN（MSIM 式）；SLA 按 `severity × 资产关键度` 派生，复用 `handlers/sla`。

**AI 边界**：`security_triage` skill 只输出结构化 JSON 建议（ATT&CK 技战术候选、相关 CI、建议动作清单），落 `proposed_*` 字段；人确认后才进正式字段。
**验收**：① 一告警一事件 → 三告警归并一事件 ② 状态机非法转换 409（CAS + `errors.Is` → 409，禁 500）③ contained 门禁拦截 ④ 非成员读 restricted → 404 ⑤ 注入样本不产生 containment_action ⑥ 四段计时字段与 timeline 一致。
**依赖**：F2（confirmed 且无法按期修复才升级为事件）。

---

### F4 处置动作门禁（P1）

**四档风险**：

| 档 | 授权要求 | 动作例 |
|:---|:---|:---|
| L0 预授权自动 | **有回滚路径才准进** | 隔离测试环境 CI、吊销测试密钥 |
| L1 单人审批 | 持 `containment:approve` | 隔离办公网主机 |
| L2 双人 SoD | 两人 + 影响面必填 + **审批人 ≠ 执行人** | 停业务服务、封生产 IP |
| L3 紧急变更 | 走 change 紧急变更流（`WorkflowTx` 同事务） | 关站、断网 |

**动作 connector 化**：每个动作 = `action_id`（connector 目录注册）+ `parameter_schema`（JSON Schema）+ `rollback_action_id`（可空）+ `idempotency_key`。🔴 **禁止自由形式命令字符串**——LLM 与人都只能从目录选动作 + 填参。这一条是"ITSM 不变成攻击执行器"的机制保证：间接 prompt injection 能让 AI 建议"隔离主机 X"，但 X 必须是目录里已注册、已授权、可回滚的动作，且最终执行必须过人。

**执行链路**：propose → approve → execute（connector.Invoke）→ 记录 `evidence_ref`（执行前后快照 + 命中的 playbook 版本 + 当时审批记录）→ 可 rollback。

**幂等**：`idempotency_key` 走 `commandbus.EnqueueTxIdempotent`——仓库既有教训：PG 冲突即 aborted(25P02)，吞 `ent.IsConstraintError` 做幂等会让整条事务连坐。

🔴 **`actor_type=agent` 时 L0 封顶**：L1+ 强制转人工。被劫持的 agent 拿着 technician 权限不能成为超级用户——`itsm-cli` 的 agent 直调模式已经存在，这个口子必须现在堵。中间件 + service 双层校验（路由声明 + service 复核）。

**API**：`GET /api/v1/security-incidents/:id/containment-actions` / `POST .../containment-actions`（propose）/ `POST .../actions/:aid/approve` / `POST .../actions/:aid/execute` / `POST .../actions/:aid/rollback` / `GET /api/v1/containment/catalog`（动作目录）。
**验收**：① L2 审批人=执行人 → 409（SoD）② 无 `rollback_action_id` 的动作进不了 L0 ③ 同 `idempotency_key` 重复 execute → 幂等 ④ agent actor 提 L1 → 409 转人工 ⑤ 回滚后状态可逆 ⑥ SQLite trigger 故障注入证明审批与执行同事务（读路径仍正常）。
**依赖**：F3。

---

### F5 统一暴露面实体（P1）

`exposure_finding` 五类收敛：`vulnerability` / `misconfig` / `exposure` / `posture` / `identity_risk`。`risk_score` 是唯一排序依据，**`score_factors` JSON 必须可解释**——开源要求评分函数可审计：因子 = 严重度 + 资产关键度 + 暴露面 + 在野利用 + 时效衰减，每个因子带得分、权重与权重表版本。计算放 service 纯函数 + 单测（权重表版本化，改权重必须留痕）。

**API**：`GET /api/v1/exposure/findings?sort=risk_score` / `POST .../findings/:id/accept`（接受风险，必填理由）/ `POST .../findings/:id/mitigate`。
**验收**：① 任一 finding 的 score_factors 能还原出 risk_score（可解释性）② 同分按 detected_at 稳定排序 ③ 接受风险必须理由 ④ F2 confirmed 的 assessment 自动建 finding ⑤ 评分函数权重变更走版本号 + 单测锁。
**依赖**：F2。

---

### F6 根因修复（P1，硬依赖 F1 的 C4）

`exposure_solution`：修复挂 `root_cause_ci_id`（镜像模板 / 基线配置）而非每台机器。300 台机器 → 12 个镜像模板变更，300 台随发布自动收敛。方案创建时按 CMDB 派生关系自动展开受影响 CI 清单，发布后自动收敛验证。

**API**：`POST /api/v1/exposure/findings/:id/solutions`（选根因 CI）→ 自动创建 change（复用 change 域审批）→ `GET .../solutions/:id/convergence`（收敛进度：多少 CI 已随发布更新）。
**验收**：① 300 CI 归并 12 模板 ② 模板变更发布后关联 CI 的 confidence 自动刷新 ③ 收敛率未达标前 solution 不可关闭 ④ 🔴 **C4 未启用时明确降级为任务式**，不允许假装根因式（降级要在 UI 明示）。
**依赖**：F1 的 C4（registry + k8s API）——**S0 阶段就要采模板派生关系**，不能等 D2 再做。

---

### F7 AI 资产图谱与 Agent 身份（P2）

**6 类 AI CIType**（seeder 种子）+ **2 条新关系**（`uses_prompt` / `retrieves_from`，词表 13→15 + ontology version 递增）。

**两张表**（产品差异化的锋利刀口）：

1. **爆炸半径**：投毒 `rag_corpus` X → 经 `retrieves_from` / `uses_prompt` 反向展开 → 受影响 AI 应用 → 业务服务 → SLA。复用 CMDB 影响分析（`handlers/cmdb/production_service.go`）。
2. **暴露面看板**：`autonomy_level=auto` × `bound_connectors` 含写权限的 agent 清单。每个必须有具名人类审批人（`agent_identity.owner_user_id`）——ServiceNow 花 $7.75B 买 Veza 解决的问题（"AI agent 是企业身份，权限几乎肯定是为人类设计的"），我们从建模上就不让它发生。

`agent_identity.effective_permissions` **从 RBAC 派生**（查该 agent 绑定角色的权限码），不手填——手填必漂移。

**API**：`GET /api/v1/cmdb/ontology`（自动含新 CIType/关系）/ `GET /api/v1/security/ai-blast-radius?ci_id=` / `GET /api/v1/security/ai-exposure-board` / `POST /api/v1/security/agent-identities`。
**AI 边界**：新 agent 注册走 BPMN 审批（声明 autonomy_level + connectors + owner）。
**验收**：① 投毒语料 → 展开到业务 SLA 的完整链路 ② auto + 写权限 connector 无 owner → 创建即拒绝 ③ 新关系进 ontology 且 version 递增（Agent 缓存失效）④ effective_permissions 与 RBAC 实际权限一致（派生校验测试）。
**依赖**：F3。

---

### F8 研判辅助与处置知识（P2）

**三个只出结构化 JSON 的 skill**（`handlers/ai/builtin_skills.go` 加 `NewSecurityTriageSkill` / `NewExposureAdvisoryExtractSkill` / `NewPlaybookMatchSkill`，按 `RegisterBuiltinSkills` 既有模式注册，每个声明 `requiredPermissions`）：

| skill | 输入 | 输出 | 硬约束 |
|:---|:---|:---|:---|
| `security_triage` | 告警集合 | ATT&CK 技战术候选 + 相关 CI + 建议动作清单 | **只列 action_id，不给命令** |
| `advisory_extract` | 公告文本 | 结构化字段草稿 | 输出过 schema 白名单 |
| `playbook_match` | 事件上下文 | 匹配 playbook 版本 | 只读已评审知识 |

**知识域扩展**：playbook 作为知识文章，`attack_techniques`（JSON 数组）标注 ATT&CK；RAG 检索按 **tactic + CI 类型双过滤**召回，不走纯语义——防召回到不相关行业的 playbook。复用 knowledge 评审状态机（draft→under_review→approved 才进索引，reject/withdraw 出索引，同事务）。

**AI 边界**（红线集中区）：三个 skill 输出全部过 schema 白名单；间接注入样本进回归集（告警 description / 公告正文藏指令，≥10 条）；skill **不直接调用 containment execute**——只能 propose。
**验收**：① schema 校验失败即拒收 ② 注入回归集全过 ③ 未评审 playbook 不进 RAG 索引 ④ 双过滤召回命中率抽验 ⑤ `rankByAuthority` 只排序不准入的既有语义不被破坏。
**依赖**：F3/F4。

---

### F9 安全运营度量（P2）

**四段计时**：`detected_at` / `triaged_at` / `contained_at` / `closed_at` → `time_to_triage` / `time_to_contain` / `time_to_close`，按 severity 分位呈现。

**两个负面指标**（🔴 不报就是自欺）：

1. **未研判告警数**：`status=confirmed` 且未 link 任何 security_incident 的告警数——对应行业数据"28% 告警从未被调查、60% 组织承认被忽略的告警后来变成实质事件"；
2. **因资源关闭的检测面**：V1 简化——手工登记 `detection_source` + 启停原因；V2 从 alert source 活跃度推导。对应"40% 组织曾关闭检测规则"。

**API**：`GET /api/v1/security/metrics/four-stage` / `GET /api/v1/security/metrics/negative`。
**验收**：① 四段计时与 timeline 一致 ② 未研判告警计数正确（dismissed 不计入）③ 负面指标**单独一区**，不与正面指标混排 ④ 看板数字与列表下钻一致（对齐 ForTicketReader 三处同源原则）。
**依赖**：F3。

---

### F10 合规控制映射（P3）

`control_mapping`：框架控制项（等保 / ISO 27001 / SOC 2）↔ BPMN 流程实例 ↔ 证据查询（审批记录 / 变更窗口 / SoD 执行 / 审计日志）。ITSM 的独特优势：**手里天然握着控制证据**——合规是"能证明按控制项做了"，而证明就是流程系统的产物。

**API**：`GET /api/v1/exposure/control-mappings?framework=` / `POST .../mappings/:id/evidence`（跑证据查询导出）。
**验收**：① 一个控制项能导出完整证据链 ② 无流程实例映射的控制项标 gap ③ 证据查询只读审计域（复用 `handlers/auditlog`）④ 证据导出含时间戳与操作人，不可篡改。
**依赖**：F3–F5。

---

## 6. 三条红线的代码层落地机制

| 红线 | 机制 | 守卫/验收 |
|:---|:---|:---|
| **AI 输出 = 不可信输入** | ① 所有 AI 产出落 `proposed_*` / draft 列，`reviewed_by` 后才生效 ② 输出过 schema 白名单 ③ 禁止 LLM 生成命令字符串（动作只能从 connector 目录选）④ 外部文本进 prompt 前隔离标记 | 注入回归集 ≥10 条（F2 用例 6、F8 用例 2）；skill 输出 schema 校验测试；每个 skill 一条负向用例 |
| **无回滚不自动** | `rollback_action_id` 可空；L0 准入检查必须非空；`evidence_ref` 必填（前后快照 + playbook 版本 + 审批记录） | L0 准入单测；回滚链路测试；SQLite trigger 故障注入证明同事务 |
| **可见性组合不旁路** | `internal/datascope` 新增 applier（同 `ApplyTicketFilter` 模式）；成员表登记 tenant_guard；未授权返 404 | 跨租户 / 非成员读 404 测试（对齐既有 `cross_tenant_http_test.go` 模式） |
| **agent L0 封顶** | `actor_type=agent` → L1+ 拒绝，中间件 + service 双层 | agent actor 提 L1 → 409 用例；itsm-cli agent 模式回归 |

---

## 7. 实施波次与每波授权平面动作模板

| 波次 | 功能 | 出口验收 |
|:---|:---|:---|
| W1 | F1（先手工导入 + C4） | software_instance 置信度链路 + 版本比较 bug 锁 |
| W2 | F2 | PHP2Shell 夹具 10 用例 + 六问页答全 |
| W3 | F3 | 状态机门禁 + 告警归并 + 404 可见性 |
| W4 | F4 | L0–L3 门禁 + SoD + 幂等 + 回滚 |
| W5 | F5 + F6 | 可解释评分 + 根因收敛（C4 就绪为前提） |
| W6 | F7 + F8 | 爆炸半径 + 注入回归集 |
| W7 | F9 + F10 | 负面指标 + 证据链 |

**每波次授权平面动作模板**（漏一步即漂移，按序执行）：

1. `internal/authz/catalog.go` `Definitions()` 加权限码 + `builtinRoleBindings` 绑角色；
2. `router/<domain>_routes.go` 逐路由 `RequirePermission`；
3. `go run ./cmd/authz-gen` 重新生成预检映射；
4. 过 5 道守卫（`TestPrecheckMapIsFresh` / `TestRoutePrecheckAlignment` / `role_permission_guard_test` / `permission_code_catalog_guard_test` / 空壳检测不涉及则跳过）；
5. 新 M2M 关联表（无 tenant_id）登记 `internal/schema/tenant_guard.go`；
6. ent schema → 迁移 SQL（已应用迁移改一字即 checksum mismatch）；dev 环境 `docker compose build itsm-init`；
7. 后端 build + `up -d`；权限缓存 TTL 5min 无失效端点，**改权限必重启 backend**；
8. 前端页面 + 契约测试 + 负向测试（每个门禁配一条"注入回归 → FAIL → 恢复"验证）。

---

## 8. 明确不做（边界）

| 不做 | 归属 |
|:---|:---|
| SIEM / EDR / 检测引擎 | 接 `POST /alerts/sources/:source/ingest` |
| 沙箱 / 样本分析 | 接外部 |
| 威胁情报平台 | 接 MISP/OpenCTI——F2 的 advisory 是轻量归一，不是情报库 |
| Armis 级被动流量分析引擎 | 采集走 C1–C4 低成本通道；深度分析接 Cortex/Armis |
| 通用 playbook 引擎 | playbook = 知识文章 + BPMN，不自建 DSL |
| 漏洞扫描器 | 是 F5 的数据源之一（connector），不是产品的一部分 |

---

## 9. 进入 ROADMAP 的建议与待决策项

**排期**：不塞 v1.7（connector 生产化 + AI evaluator + 全 UI 中文化已占满，i18n 是硬承诺）。建议 **v1.8「Security Response」独立线**：W1–W4 进 v1.8，W5–W7 进 v1.9。社区版出口 = F1 + F2 + F3。

**需要用户决策的四件事**：

1. **v1.8 独立线是否批准**（还是并入现有某条线）；
2. **`security_admin` 绑定到全部安全权限码**是否确认——这是本文最核心的授权决策（§3.2），它决定"安全团队负责人"这个既有角色是就此激活还是继续空转；
3. **C7（SPAN）与 C8（主动探测）默认关闭**是否符合目标客户的安全基线——部分企业会要求主动扫描，需要保留配置化开关；
4. **`unknown` 桶的派活策略**：自动建核查工单（走 service_catalog）还是仅在看板提示——前者增加流程噪音，后者可能被忽略。

---

*本文所有仓库坐标为 2026-10-22 实测；行业数据引自 Prophet Security《State of AI in Security Operations 2026》与 SANS 2026，已在纵深文档标注证据分级。*
