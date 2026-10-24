# 安全运营纵深产品设计：对标 ServiceNow 与开源生态

> **Status**: draft（设计蓝本，未进入实施）
> **Date**: 2026-10-22
> **前置**：[ai-security-operations-blueprint.md](./ai-security-operations-blueprint.md)（S1–S5 事件域）、[exposure-answer-design.md](./exposure-answer-design.md)（S0 暴露面自答）
> **下游**: [security-operations-feature-design.md](./security-operations-feature-design.md)（F1–F10 可实施功能规格；本文 D1–D5 对应 F5/F6/F7/F4/F10，C1–C8 对应 F1）。
> **证据分级**：本文引用 ServiceNow 的能力分三级标注——【官方文档】`servicenow.com/docs`、【官方发布】`newsroom.servicenow.com`、【社区架构文章】`community.architect-articles`（架构师撰写，非产品文档，结论性表述慎用）。

---

## 0. 一句话结论

**ServiceNow 用四层架构把安全运营做成了平台的衍生能力，而我们手里已经有其中两层的地基；开源阵营把"执行层"做得很热闹，却在"治理层"留着一整片真空。深入安全运营的正确姿势不是补齐别人的长板，是守住只有 ITSM 站得住的位置，并把 ServiceNow 2026 年验证过的三个方向用原生架构做出来。**

三个事实构成本文的出发点：

1. ServiceNow 的安全能力不是独立产品，是**同一个 CMDB + 同一个工作流引擎 + 同一个数据模型**的四层叠加（【官方文档】[Demystifying ServiceNow Security](https://servicenow.com/community/platform-privacy-security-blog/demystifying-servicenow-security-it-s-not-fragmented-it-s/ba-p/3526882)）。
2. ServiceNow 2026 年的两个关键收购（Armis 资产发现、Veza 身份权限图谱）买的正是**我们自 2023 年就在建的东西**——资产图谱与身份治理（【官方发布】[Autonomous Security & Risk, 2026-05-05](https://newsroom.servicenow.com/press-releases/details/2026/ServiceNow-launches-Autonomous-Security--Risk-integrating-Armis-and-Veza-to-govern-every-AI-agent-identity-and-connected-asset/default.aspx)）。
3. 开源 SOAR/案件管理阵营 2026 年出现真空：TheHive 5 已**转为商业 freemium**（3/4 版本 EOL），开源侧剩下的是执行层工具（Shuffle / Tracecat / Wazuh），**没有一个同时持有 CMDB、变更审批、SLA 与行级审计**。

---

## 1. ServiceNow 怎么做的

### 1.1 四层架构（【官方文档】）

| 层 | 应用 | 核心数据 |
|:---|:---|:---|
| L1 Platform Security | ACL、API 策略、Security Center、Vault | 平台自身加固 |
| L2 Security Operations | **USEM**（统一暴露面管理，聚合 infra/app/container/config 四类暴露）、Security Posture Control（工具覆盖度 + 资产卫生）、**SIR**（SIEM 导入威胁数据自动建事件）、MSIM（重大事件升级）、VR（扫描器发现映射 CI + 业务服务）、DLP IR | CMDB + Service Graph connectors |
| L3 Risk & Compliance | IRM/GRC，把漏洞翻译成业务影响与合规项 | CMDB 真实资产 + SecOps 暴露与事件 |
| L4 AI Governance | **AI Control Tower**：AI 用例清单、风险与控制映射、访问与行为度量 | 同一平台数据 |

原文的表述可以直接引用：*"You don't need to build a security stack on top of ServiceNow. You already have one."* —— 这就是我们要复制的那句话，只是主语换成 ITSM。

### 1.2 五个可复制的设计原则

| # | ServiceNow 的做法 | 为什么重要 | 我们的对应 |
|:--|:---|:---|:---|
| 1 | **风险优先而非数量优先**：VR 的价值是"不是 10000 个漏洞，而是这 12 个威胁到关键收入服务"（【官方文档】+ 第三方解读一致） | 漏洞数量不可行动，资产 + 业务上下文才可行动 | S0 的 `software_instance` + CMDB 关系展开；S1 的受影响 CI 联动 SLA |
| 2 | **VR Prime 把修复回溯到根因 CI** 而非管理单个任务记录（【社区架构文章】） | 打 500 台机器的补丁是体力活，改一个镜像模板才是修复 | 🔴 **我们原设计缺这一层，见 §3 D2** |
| 3 | **确定性 playbook + agentic 工作流 + human-in-the-loop 并存**（2026 SOAR 行业共识，FortiSOAR 8.0 明确表述为 "agentic AI + deterministic playbooks + governance controls"） | 自主性必须有确定性路径作边界 | S1 的 L0–L3 风险分级，见 §3 D3 上升为架构原则 |
| 4 | **Agent 也是身份，且权限几乎肯定是为人类设计的**（【官方发布】Autonomous Security & Risk 核心论述） | 被劫持的 agent = 持有不当权限的超级用户 | S1 的 `actor_type=agent` 时 L0 封顶；见 §3 D4 |
| 5 | **治理是流程的一部分，不是事后补丁**（【官方文档】L4 表述："governance is part of the workflow, not something that happens after the fact"） | 事后治理永远追不上 agent 增殖速度 | 审批链 + 审计同事务，见 §3 D5 |

### 1.3 授权分层（【社区架构文章】，2026-04）

SIR 与 VR 都改成 Foundation / Advanced / Prime 三层，**能力被硬门控在 tier 上**：MSIM、Patch Orchestration、CISO 看板在 Advanced；DLP IR 只在 SIR Prime；VR Prime 才有 Vulnerability Solution Management；Now Assist GenAI 消耗量与 tier 绑定。

> 对产品的启示：**能力分层必须与授权分层解耦设计**，否则会出现"开源版功能残缺到不能用、商业版又不敢砍"的窘境。我们的分层要按**场景完整性**切，不按功能数量切（见 §5）。

---

## 2. 开源阵营的真实位置（2026）

| 能力层 | 代表产品 | 2026 状态 | 缺口 |
|:---|:---|:---|:---|
| 检测/日志/SIEM | Wazuh 4.x（含 active response：block IP / kill process / isolate host） | 活跃，合规映射 PCI/HIPAA/NIST 800-53/GDPR/SOC 2 强 | 无 GUI playbook 编排；无企业治理 |
| SOAR 编排 | Shuffle（AGPLv3/MIT）、Tracecat（Apache 2.0，Python playbook 可版本控制 + CI/CD）、Keep（MIT，YAML 工作流，2025-05 被 Elastic 收购） | 活跃，生态成熟 | **都是执行层，没有授权边界**——Shuffle 默认安装无 autonomous investigator；Wazuh active response 直接执行脚本，无人审批概念 |
| 案件管理 | TheHive + Cortex | 🔴 **TheHive 5 已转商业 freemium，3/4 EOL**（社区版限功能） | 开源 case 管理出现真空；CSIRT/取证协作场景没有免费替代 |
| 威胁情报 | MISP / OpenCTI | 活跃 | 与资产无对齐（同 §1 的"到处找"问题） |
| **资产/配置管理** | — | 🔴 **空白** | 开源阵营没有 CMDB。这是所有 SOAR 都答不出"我们受影响吗"的根因 |
| **变更/审批/SLA/审计** | — | 🔴 **空白** | 开源产品没有企业流程治理。处置动作谁授权、谁复核、SoD、不可否认审计，全部缺位 |
| AI agent 治理 | 零星 | 🔴 **空白** | 无人清点 agent、无人评分、无最小权限执行 |

**结论：开源阵营的执行层已经很拥挤，治理层完全空白。** 而治理层恰好是 ITSM 的原生能力——这不是市场空白，是**结构性错配**：SOAR 团队不会去建 CMDB，ITSM 团队不会去建 SOAR。

---

## 3. 深入设计：五个纵深

以下 D1–D5 是对前两份蓝图的深化，不是并列功能。每一项都给出"ServiceNow 对应物 + 开源对应物 + 我们的差异化"。

### D1 从「CVE 匹配」到「统一暴露面实体」（对标 USEM）

S0 目前只处理漏洞型事件。USEM 的做法是把 **infrastructure / application / container / configuration** 四类暴露收敛到一个统一视图（【官方文档】）。对应改造：

```
exposure_finding（新实体，S0 的 exposure_assessment 上提一层）
  finding_type      vulnerability | misconfiguration | exposure | posture_gap | identity_risk
  source            scanner | sbom | posture_control | cloud_audit | advisory_match
  severity + cvss_version
  asset_ref         CI（或 software_instance）
  risk_score        ← 不是 severity，是 severity × 资产关键度 × 暴露面 × 在野利用状态
  evidence
  status            open | accepted_risk | remediated | false_positive
```

**关键设计：`risk_score` 是唯一排序依据，`severity` 只是输入之一。** ServiceNow 的核心卖点"不是 10000 个漏洞而是 12 个"（§1.2-1），实现上就是一个分数。我们的差异化在于这个分数**可解释**：每一项因子都从 CMDB 真实字段来，而不是黑箱评分——因为我们要开源，评分函数必须可审计。

开源对应物：无（Wazuh 只产告警，Shuffle 只编排，没人做收敛与排序）。

### D2 从「打补丁」到「根因 CI 修复」（对标 VR Prime Solution Management）

这是 ServiceNow 2026 年最有价值的一个变化，也是**我们前两份设计最大的疏漏**：把修复回溯到**根因 CI**，而不是在每台机器上开一个任务（【社区架构文章】）。

对 PHP2Shell 这类场景，两种模式的差别是数量级的：

| 模式 | 做法 | PHP2Shell 下的结果 |
|:---|:---|:---|
| 任务式（我们原设计） | 每台命中 CI 开一个补丁任务 | 300 台 → 300 个任务、300 次审批、300 次验证 |
| **根因式（D2）** | 定位到「镜像模板 / 基线配置 / 软件实例模板」，改一处，派生实例批量收敛 | 12 个镜像模板 → 12 个变更，300 台随发布自动收敛 |

新增实体：

```
exposure_solution                修复方案（根因收敛）
  root_cause_ref                 CI 或 software_instance 模板（镜像/基线/配方）
  finding_type / product
  remediation_kind               patch | config_change | template_update | migration
  linked_findings                []finding_id
  execution_plan                 BPMN processDefinitionKey（复用工作流引擎）
  state                          draft | approved | in_progress | verifying | closed
  verification_evidence          收敛后重新扫描/复扫结果
```

**为什么只有 ITSM 能做**：根因修复的本质是"一个变更影响 N 个派生资产"，这需要 CMDB 的派生关系 + 变更审批 + 发布编排。SOAR 只能"在 300 台机上跑 300 次脚本"——它没有资产派生语义。开源对应物：无。

### D3 自主性分层上升为架构原则（对标 deterministic + agentic + HITL）

2026 SOAR 行业的共识表述（FortiSOAR 8.0 为例）：**同时提供 agentic AI 与确定性 playbook，并保留治理控制**。这把 S1 的 L0–L3 分级从"配置项"提升为架构原则：

> **每一个处置动作必须同时存在两条实现路径：确定性路径（模板 + 参数 schema + 人工触发）与 agentic 路径（AI 专家 agent 建议执行）。agentic 路径永不超出确定性路径已声明的授权边界。**

落到产品上：

| 动作 | 确定性路径 | Agentic 路径 | 边界 |
|:---|:---|:---|:---|
| 补丁升级 | change 审批 → 发布编排 | 漏洞清理 agent 识别积压、起草变更、提请审批 | agent 只能起草，不能批 |
| 钓鱼调查 | playbook：查 URL/附件/发件人 | 钓鱼调查 agent 完成富化与初判，附证据 | agent 结案必须人确认 |
| 主机隔离 | 处置门禁 L0–L3 | 隔离 agent 按策略提议 | 同上，且 agent 调用 L0 封顶 |
| 暴露面收敛 | D2 变更 | 收敛 agent 定期扫描"模板漂移" | agent 不能改模板 |

ServiceNow 的两个 AI specialist agent 做的正是前两件事（漏洞积压清理 + 钓鱼调查），且明确表述为 **"alongside human teams"**（【官方发布】）——不是替代。这与 §1.2-3 完全一致，也再次印证 57%/44%/30%/0% 那份调研的市场规格。

**我们的差异化**：ServiceNow 的 agent 治理靠 AI Control Tower（第四层产品）；我们的 agent 治理**内生**在 RBAC + `actor_type` + 审批链里，因为 agent 从一开始就是数据模型里的一等身份，而不是后加的第四层。

### D4 身份与权限图谱（对标 Armis + Veza）

ServiceNow 2026 花大价钱买的两样东西（【官方发布】）：
- **Armis**：无代理资产发现（设备类型/分类/固件/行为/风险态势）→ 直接入 CMDB，"把静态清单变成实时攻击面"
- **Veza Access Graph**：每个身份（含非人类）的有效权限图谱，实时最小权限执行

我们的对应（前两份已铺底，此处补纵深）：

| ServiceNow | 我们已有 | D4 新增 |
|:---|:---|:---|
| Armis 无代理发现 | CMDB + cloud discovery 骨架 | 把 `software_instance` 的采集扩展到容器/镜像/SBOM（D1 的输入源） |
| Veza Access Graph | RBAC + DataScope + 5 道守卫 | **`agent_identity` 图谱**：每个 agent 的身份、绑定的工具连接器、有效权限、最近一次权限变更的审批记录 |
| AI Control Tower 治理 agent | — | **AI 资产准入流程**：新 agent / 新模型端点 / 新 prompt 上线必须走 change（PIR 事后复盘），与 §4.2 的 6 类 CIType 闭环 |

关键实体（小而准）：

```
agent_identity
  name / owner_ci_id
  actor_type            agent（与 human / service_account 并列）
  bound_connectors      []connector_id（它实际能碰到哪些系统）
  effective_permissions  []permission_code（从 RBAC 派生，非手填）
  autonomy_level        advisory | restricted_auto | auto（受审批约束）
  last_review_at / reviewer
  risk_signals          JSON（长期未审、权限超出角色、绑定高危 connector）
```

🔴 **一条硬规则**：`autonomy_level=auto` 的 agent 若绑定写权限 connector，**必须**在审批链上有具名的人类审批人，且 agent 的每次执行都落 `actor_type=agent` 审计。这是 Veza 的"权限几乎肯定是为人类设计的"在 ITSM 里的落地形态。

### D5 治理闭环：控制项 ↔ 流程 ↔ 证据（对标 IRM/GRC + 合规映射）

ServiceNow L3 把漏洞翻译成业务影响与合规项；开源侧 Wazuh 的合规映射（PCI/HIPAA/NIST 800-53/GDPR/SOC 2）是它最强的开箱卖点。

**ITSM 做这件事有独特优势：我们手里天然握着"控制证据"**——审批记录、变更窗口、SoD 执行、审计日志。合规的本质是"你能证明你按控制项做了"，而证明恰恰是流程系统的产物。

设计（轻量，不做成 GRC 产品）：

```
control_mapping
  framework           iso27001 | pci_dss | soc2 | djcp | 等保
  control_id          A.12.6.1 / 8.3.6 / ...
  mapped_flow         BPMN processDefinitionKey 或 ITSM 流程标识
  evidence_query      （如何从审计/审批表取证明）
  last_attested_at / attested_by
```

输出物：**按框架自动生成的证据包**（哪些流程在跑、跑了多少次、谁批的、异常在哪）。这比"我们部署了 Wazuh"有力得多——前者是控制证据，后者是工具清单。

---

## 4. 产品分层建议（对标 Foundation/Advanced/Prime，但按场景切）

不按功能数量切，按**场景闭环是否完整**切：

| 版本 | 场景闭环 | 包含 | 对应 ServiceNow |
|:---|:---|:---|:---|
| **社区版** | 答得出来 | S0 暴露面自答 + D1 收敛 + 六问页 | USEM + SIR 的读面 |
| **商业版** | 管得起来 | + S1 事件域 + D2 根因修复 + D5 证据包 + L0–L3 门禁 | SIR + VR Advanced + MSIM |
| **企业版** | 治得住 | + D4 agent 身份图谱 + AI 专家 agent + 定制评分 + 多框架合规 | SIR/VR Prime + Armis/Veza + AI Control Tower |

**一条铁律**：任何一层的裁剪**不得破坏"六问页能答全"**——那是社区版的核心承诺，也是我们与"又一个开源 SOAR"的区别所在。

---

## 5. 路线整合（更新后的全景）

```
S0  暴露面自答 ─────────── D1 统一暴露面实体 + C1–C4 无代理采集（云 API / SNMP / LLDP-CDP / registry）
S1  安全事件域 ─────────── D3 自主性分层（L0–L3 上升为架构原则）
S2  处置门禁 ───────────── D3 + agent 身份（actor_type）
S3  AI 资产图谱 ─────────── D4 agent_identity + AI 资产准入流程
S4  知识与研判 ──────────── D3 agentic 路径（只走已授权边界）
S5  度量 ───────────────── D5 控制证据 + 负面指标
D2  根因修复 ───────────── 新增，依赖 S0 的 C4 镜像清单 + S1 的变更复用
```

**依赖修正（两条）**：
- D2 依赖"镜像模板/基线的派生关系"进 CMDB，而派生关系来自 **C4（registry/k8s API）采集**，**必须在 S0 就采**，不能等 D2 再做。
- S0 的匹配引擎质量取决于采集覆盖度。C1–C4（零/低网络改造）应作为 S0 的第一批采集器交付；C5–C8 属增强，不影响 S0 上线。

---

## 6. 度量（新增三个）

| 指标 | 定义 | 对标 |
|:---|:---|:---|
| **根因收敛率** | `closed via solution / closed total` | VR Prime 的核心价值主张 |
| **agent 权限漂移数** | 有效权限超出角色声明的 agent 数 | Veza / AI Control Tower |
| **控制证据完备率** | 映射的控制项中能在系统内取到证据的比例 | IRM/GRC |

---

## 7. 无代理采集的分层落地（技术可行性评估）

> 本节为 2026-10-22 追加，**修正原 §7「无代理资产发现整体不做」的判断**。原判断的错误在于把「Armis 级被动分析引擎」和「低成本采集通道」混为一谈：前者不该做，后者**必须做**——S0 的 `software_instance`、D1 的 `exposure_finding`、D2 的模板派生关系，数据全部来自这里。不做的话 D1/D2 是空中楼阁。

### 7.1 先纠正一个印象：Armis 也不是纯被动

Armis 的官方架构是**混合式**：被动侧吃 SPAN/TAP 流量、flow、无线控制器、交换机表、以及各工具 API 拉取的清单；同时提供 **Smart Active Querying** 主动查询，用来「验证设备固件是否已更新」和发现休眠/不通信设备（[Armis 官方](https://www.armis.com/?p=2543415)）。它自称的"agentless、非侵入"指的是**不在终端上装 agent**，不是零交互。

ServiceNow 收购它的技术拆解显示其后端是 Cassandra 存非结构化资产 + Kafka/Flink 做亚秒级流式异常检测，核心资产是 **50 亿设备的知识库**与 50+ 工业协议解析（[技术拆解](https://automatewithaiagent.com/servicenow-armis-acquisition)，收购价 $7.75B all-cash）。**这三样（指纹库规模、工业协议深度、流处理规模）正是我们不该自建的壁垒。**

### 7.2 关键结论：设备类型与固件版本是两个难度等级

这是评估能否落地的核心，多数产品宣传把两者混成一句"自动识别设备与固件"。

| 目标 | 可靠信号 | 可达准确度 | 主要失效场景 |
|:---|:---|:---|:---|
| **设备类型/厂商** | MAC OUI + DHCP 指纹（Option 55 顺序 / Option 60 / hostname）+ mDNS/SSDP 广播名 + HTTP User-Agent + TLS JA3/JA4 | 高（多信号叠加） | 见 7.3 |
| **固件/OS 版本** | **SNMP `sysDescr`(1.3.6.1.2.1.1.1.0) + `sysObjectID`**、LLDP/CDP 的机箱序列号与软件版本、HTTP/Telnet/SSH banner | 中，且**只对管理型设备有效** | IoT/嵌入式普遍不响应 SNMP；版本串可能被隐藏、定制或不准确；OEM 贴牌掩盖真实制造商 |

**这句话对 PHP2Shell 场景是致命的**：最需要知道固件版本的恰恰是打印机、网络设备、工控网关这类**不响应 SNMP 的嵌入式设备**。Ripple20 事件总结的教训很直接：扫描通常只能**缩小候选范围，不能给最终结论**；厂商与固件验证最终要落到 OEM 公告（[TechBloat](https://www.techbloat.com/ripple20-finding-vulnerable-devices-and-detecting-attacks.html)）。

因此产品设计必须承认：**无代理采集的产出是"候选 + 置信度"，不是"结论"。** 这与 S0 的四态设计天然咬合——采集不到的版本进 `unknown` 桶，交给 vendor 台账与人工确认，而不是假装知道。

### 7.3 五个正在恶化的技术约束（不能当静态假设设计）

| 约束 | 影响 | 设计对策 |
|:---|:---|:---|
| **MAC 随机化成为默认**：iOS 14+ / Android 10+ / Win10-11 默认随机硬件地址，IEEE 802.11bh 将正式化 | MAC OUI 作为主信号正在失效，NAT/静态 IP/IPv6 双栈下 DHCP 指纹也抓不到 | MAC 只能做**辅信号**；主信号改为 mDNS/SSDP + TLS 指纹 + 交换机端口映射 |
| **虚拟 MAC**：HA 主备共享虚拟 MAC、SDN/overlay(VXLAN) 内 MAC 即时生成、NAC 可强制匿名 MAC | 查不到真实厂商，且分不清物理机 | 以「交换机端口 + LLDP 邻居」为物理锚点，MAC 仅作关联线索 |
| **嵌入式设备不耐受主动探测**：医疗设备、楼控、打印机、工业网关可能被扫崩溃或进入故障态 | 主动扫描在 OT 场景可能造成安全事故 | 主动探测必须**配置化、低速率、限端口、限维护窗口**，且默认关闭；优先被动 |
| **合规边界**：被动监听涉及流量隐私（GDPR/个保法） | 抓全量 payload 有法律风险 | 只提取**元数据与指纹**，不落 payload（Armis 官方明确"No raw data or payloads (PII/HPI) is sent to the Armis SaaS"） |
| **网络位置是组织问题不是技术问题**：SPAN/TAP 要网络团队批，跨 VLAN 要防火墙开口，主动扫描常需变更审批 | 往往是落地周期的真正瓶颈 | 采集通道按"申请难度"排序推进（见 7.4），先摘不需要网络改造的果子 |

### 7.4 分层落地：按申请难度排序，不按技术先进性排序

| 层 | 通道 | 覆盖 | 网络改造 | 优先级 |
|:---|:---|:---|:---|:---|
| **C1** | 云 API（AWS/Azure/GCP describe API） | 云主机/容器/托管服务，**软件清单质量最高** | 无 | **P0**（仓库已有 `cloud_account` / `discovery_job` 骨架，直接复用） |
| **C2** | SNMP v2c/v3 只读轮询（`sysDescr`/`sysObjectID` + 厂商 MIB） | 网络设备、打印机、管理型服务器——**固件版本的主要来源** | 单 UDP 端口 | **P0** |
| **C3** | LLDP/CDP 邻居表（交换机本就在广播） | 拓扑 + 序列号 + 软件版本 | 无（读交换机） | **P0** |
| **C4** | 容器镜像 registry + k8s API | 现代工作负载的**镜像/包清单**，天然对应 D2 的"镜像模板" | 无 | **P0** |
| **C5** | 交换机 MAC 表 + DHCP lease（syslog / API 拉取） | 回答"有什么设备"，**不回答版本** | 低（读日志） | **P1** |
| **C6** | mDNS/SSDP 组播监听（不需要全量 SPAN） | 设备友好名与能力（`_ipp._tcp` 打印机、`_smb._tcp` NAS） | 中（一个组播监听口） | **P1** |
| **C7** | SPAN/TAP 全流量被动指纹 | 最全，但维护重 | 高（需网络团队） | **P2**，仅客户明确要求且愿意给网络位置时做，且只做轻量指纹 |
| **C8** | 主动探测（nmap 风格，限端口/低速率/维护窗口） | 补 MAC 随机化盲区 | 高 + 需审批 | **P2，默认关闭**，每网段显式授权 |

**指纹库不自建**：OUI 用开源库（IEEE 注册表 / Wireshark OUI 库），DHCP 指纹接 Fingerbank 社区库（有 REST API，可独立于 PacketFence 使用）。自建 50 亿指纹库不在讨论范围。

### 7.5 与既有设计的接口（三处必须接上）

1. **`software_instance.confidence` 是必需品不是可选项**：`confirmed`（SNMP/banner 明确读到）/ `inferred`（多信号推断）/ `stale`（超期未复采）。S0 的四态结论直接消费它。
2. **`vendor` 域复用为"版本疑难仲裁通道"**：OEM 贴牌、版本串不可信、嵌入式设备无 SNMP——这三类统一转 vendor 记录，由人确认，而不是让系统猜。
3. **C4 是 D2 的前置依赖**：D2 要改"镜像模板"让 300 台收敛，前提是能从 registry 看到这些镜像被谁用了。C4 不做，D2 只能降级成任务式。

---

## 8. 不做什么（增量，§7 修正后）

| 不做 | 理由 |
|:---|:---|
| Armis 级被动分析引擎（50 亿指纹库 + 50+ 工业协议 + Kafka/Flink 流处理） | $7.75B 的技术壁垒，且非 ITSM 能力域；客户真需要时走 connector 接 Armis 等 |
| 全流量深度包检测与 payload 留存 | 合规风险；只提取元数据与指纹 |
| 通用 agent 运行时 | 我们治理 agent，不托管 agent |
| 自建威胁情报平台 | 接 MISP/OpenCTI |
| 完整 GRC 平台 | 红海且重；只做"控制项 ↔ 证据"映射（D5） |

---

## 9. 一页结论

**ServiceNow 证明了这条路值钱（Security & Risk 已超 $1B ACV，【官方发布】），并且用两次收购暴露了这条路的真成本：资产图谱与身份图谱是要花几十亿买的能力。我们不需要买——这两样东西从 v1.0 就在这个仓库里长着。**

开源阵营证明了这个市场的另一面：执行层已经足够拥挤，治理层完全空白，而**治理层才是企业安全运营真正卡住的地方**。补丁打不下去不是因为缺 SOAR，是因为没人有权批、没人知道影响谁、没人留得下证据。

深入安全运营的产品定义因此收敛为一句话：

> **用 ITSM 的流程、资产与审计，把安全运营从"检测驱动的体力活"变成"风险驱动的治理动作"——并且让每一个自主动作都落在一条已声明的授权边界内。**
