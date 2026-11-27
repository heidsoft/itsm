# CloudMesh ITSM P0/P1 差距与商业化阻塞点

> Status: current · 基线日期：2026-11-23 · 依赖：[current-state.md](./current-state.md)
>
> 本文的“差距”指**目标能力地图（用户提供的 A–E 蓝图）与当前仓库证据之间的差**，不是待办清单。
> 分数为初筛工具；P0/P1 的最终归属由**阻断性、依赖关系、证据等级**决定，不按分数机械排序——这与优先级模型自身的声明一致。

---

## 1. 判定原则与证据等级

| 证据等级 | 定义 | 本轮数量 |
|:---|:---|:---|
| Verified Demand | 真实客户书面/口头确认、LOI、签署 PoC 或付费承诺 | **0** |
| Observed Problem | 仓库代码、测试、CI 产物或既有文档中有可复核的问题证据 | 多数 |
| Hypothesis | 行业分析/产品推演，尚无客户或运行证据 | 少数 |

**红线**：`docs/business/*.md` 与 `docs/product/industry-verticals-analysis-2026-08-01.md` 中的客户与价格表述均为假设/占位，**不得在本轮或后续对外材料中升格为已验证需求**。

---

## 2. 能力地图差距总览

| 地图 | 目标 | 当前证据结论 | 主要差距 |
|:---|:---|:---|:---|
| A. ITSM Core | 11 类核心流程 | 11 Implemented / 1 Partial | 审批分层未收敛；CMDB 深度不足；SLA/变更跨域人工签字空白 |
| B. IT Operations | 监控接入、事件关联、巡检、运维执行、DB 治理、多云、应用环境 | 多为 Partial / Unknown | 入站告警→事件→工单闭环未核实；数据库运维仅权限骨架；无生产验收 |
| C. AI-Native | Runtime / Registry / Skill / Context / Tool / Memory / Eval / Observability / HITL | 治理强、运行时弱 | 无通用 Agent Registry；无声明式 ActionFlow；Memory Missing；Eval 真实模型档手工；RAG v2 缺 chunking/重排 |
| D. Governance | IAM/RBAC/租户/变更/配置/安全/合规/成本 | 应用层强、纵深弱 | RLS 仅 2 表；审计独立事务；基线三方合并未实现；成本治理无产品面 |
| E. Industry Solutions | 5 个行业方案 | Missing（产品化）/ Hypothesis（需求） | 无行业包、无行业模板、无客户验证 |

---

## 3. 优先级评分（初筛）

公式：`Priority = (痛点×0.30 + 付费×0.25 + 复用×0.25 + 匹配×0.20) / (1 + 复杂度×0.10 + 风险×0.10)`

| ID | 差距 | 痛点 | 付费 | 复用 | 匹配 | 复杂度 | 风险 | 分数 | 等级 | 2026-11-27 状态 |
|:---|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---|:---|
| GAP-01 | 社区版/企业版/行业方案边界与报价矩阵未定义 | 5 | 5 | 5 | 3 | 2 | 2 | **3.29** | P0 | 结构已建、**全部待决策**（↓0，未解） |
| GAP-02 | 版本/成熟度事实源漂移 | 4 | 3 | 4 | 5 | 2 | 3 | **2.63** | P0 | 版本面已对账（11-24）；**发版流程仍踩红**（§8.1） |
| GAP-03 | IM 入站消息→工单业务派发缺失 | 4 | 4 | 5 | 5 | 4 | 4 | **2.47** | P0 | 飞书已通（Pilot/默认旁路）；**企微/钉钉未通** |
| GAP-04 | CI 红灯 + docs-gate 非必需检查，交付质量不可证 | 4 | 2 | 4 | 5 | 3 | 3 | **2.31** | P0 | 失败收敛到 2 项且已修 2 项；**必需检查仍 4 项** |
| GAP-05 | AI 评测真实模型档手工、RAG v2 缺失 | 3 | 3 | 4 | 5 | 3 | 3 | 2.28 | P1 | 未变 |
| GAP-06 | 前端 hook 反模式 / 移动端适配薄弱 | 3 | 2 | 3 | 5 | 2 | 2 | 2.25 | P1 | 未变 |
| GAP-07 | 通用 Agent Runtime + 声明式 ActionFlow 缺失 | 4 | 4 | 5 | 4 | 5 | 4 | 2.24 | P1 | 未变（Run Manifest 骨架落地，非通用运行时） |
| GAP-08 | ServiceNow 级 CMDB 类模型 / IRE 缺失 | 4 | 5 | 5 | 3 | 5 | 5 | 2.15 | P1 | 未变（11-26 审计修的是正确性与权限，非类模型） |
| GAP-09 | 基线字段级三方合并未实现 | 4 | 4 | 5 | 4 | 5 | 5 | 2.13 | P1 | 未变 |
| GAP-10 | RLS 数据库层隔离未生效 | 4 | 3 | 5 | 4 | 4 | 5 | 2.11 | P1 | **口径修正**：策略覆盖 15 表，但 `RLS_MODE=off`、双连接池零调用方 → 实际零生效（§8.3） |
| GAP-11 | K8s 生产拓扑（HA/HPA/PDB/Helm）缺失 | 4 | 4 | 5 | 3 | 5 | 5 | 2.03 | P1 | 未变 |

> 分数为 **2026-11-23 初筛值**，未随本轮复核重算——分数是初筛工具，P0/P1 归属由阻断性与依赖决定（见 §1）。末列只记状态变化，不改原始评分，避免用"分数下降"掩盖未解决。

> 分数集中在 2.0–3.3 是初筛模型的正常结果（分母惩罚复杂度与风险）。**P0 = 不解决就无法完成一次可信、可报价、可交付的商业闭环**；P1 = 提升竞争力但可在首个 PoC 之后分批投入。
>
> GAP-09/GAP-10 虽列 P1，但对**私有化升级承诺**是硬前提：若在合同中承诺“跨版本无损升级”，必须在承诺前提升为 P0。

---

## 4. P0 差距详情

### GAP-01 商业边界与报价矩阵未定义

- **证据**：`docs/product/itsm-commercial-capability-contract.md:209-214` 七个开放问题均未决策；`docs/business/{SALES_PLAYBOOK,POC_PLAYBOOK,IDEAL_CUSTOMER_PROFILE,PRICING_AND_PACKAGING}.md:4` 为占位符。
- **为什么阻断**：客户问“ community 版能用吗/企业版多多少钱”时没有可对外一致回答；销售无法写报价单，交付无法界定范围；`capability flag` 已有实现（`src/config/product-capabilities.ts` 5 个 disabled 契约），但没有商业语义映射。
- **验收标准**：形成经评审的边界文档 + 能力矩阵；每个能力标注 CE / EE / Industry 与理由；边界与现有 capability flag 对齐；定价文档从“建议测试价”升级为“待决策”或明确决策记录。
- **证据等级**：Observed Problem（文档缺口可复核）+ Hypothesis（分界方案）。

### GAP-02 版本与成熟度事实源漂移

- **证据**：本文件 §5.4 的 7 个表面（tag v1.6.60、ROADMAP v1.6.37/v1.6.48、UPGRADE 1.6.20、治理文档 v1.6.9、漂移基线 v1.6.59、.env.prod v1.6.70、build 日志 v1.6.71；README Go 1.25.13 vs go.mod 1.26.10）。
- **为什么阻断**：合同中的版本、SLA、升级路径无法指向唯一事实；客户部署后 `/version` 与文档不一致会直接损坏信任。
- **验收标准**：确定单一版本来源（建议 tag + `CHANGELOG.md` + `/version` 运行时三者一致）；修正 ROADMAP/UPGRADE/治理文档/漂移基线/README/`.env.prod` 六处；CI 增加跨表面一致性检查（不是登记历史漂移后放行）；保留 `docs-gate` C.4 证据。
- **证据等级**：Observed Problem。

### GAP-03 IM 入站消息→工单闭环缺失

- **证据**：`handlers/wecom/handler.go:91-94` 与 `handlers/dingtalk/handler.go:94-96` 入站只 log + ACK；`handlers/feishu/handler.go:292-316` 仅处理 `task_event`，其余返回 `"ignored"`；`connector/builtin/feishu/connector.go:84-92` 声明建单能力但只有出站 `SyncTicketToFeishu`；`ROADMAP.md:24` 自认未完成。
- **为什么阻断**：这是最直接可演示、最贴近“减少人工操作”的行业闭环；没有它，IM 连接器只是安全入口，不是业务价值。
- **验收标准**：选定飞书一个渠道；一条群/私聊消息经验签+解密+持久去重后，在正确租户下创建工单；带租户校验、权限、审计、失败分类；HTTP 层回归断言 401/403/重复消息幂等；生产租户验收仍需人工签字（不得由脚本代签）。
- **证据等级**：Observed Problem。

### GAP-04 CI 红灯与合并门禁失真

- **证据**：`output/github-ci-review-2026-11-14.md:27,32-45`（39/100 失败；gofumpt 90 文件、Swagger 陈旧、2 Jest 断言、RLS 角色断言、Trivy HIGH CVE）；`scripts/docs-gate/run-all.sh:21-27` C.6/C.7 hard 但 `.github/workflows/docs.yml:58-62` 以 advisory 调用；分支保护仅 4 项必需检查。
- **为什么阻断**：无法向客户证明每次交付都经过同样的质量门禁；“本轮全绿”的说法在存量红灯面前不可信。
- **验收标准**：把 39 个失败按“基线存量 / 本轮新增”分类并登记；使 docs-gate 真正成为必需检查或在策略中明确豁免范围；修复 gofumpt/Swagger/Jest 三类确定性红灯；处理 Trivy HIGH CVE 或记录风险接受；报告必须准确写出未执行命令而非声称全绿。
- **证据等级**：Observed Problem。

---

## 5. P1 差距摘要

| ID | 差距 | 关键证据 | 触发升级为 P0 的条件 |
|:---|:---|:---|:---|
| GAP-05 | AI 评测真实模型档手工；RAG 无 chunking/真重排 | `cmd/ai-eval/main.go:28`；`ROADMAP.md:203-209` | 合同承诺“AI 准确率 ≥ X%” |
| GAP-06 | `useTickets.ts` 无界批量、静态 message、`any`；`useResponsive` 零消费者 | 前端审计 §4/§8 | 批量场景成为主打功能 / 移动端成为交付范围 |
| GAP-07 | 无通用 Agent Registry / 声明式 ActionFlow | `rag_service.go:1170-1293`、全仓无 ActionFlow 符号 | 客户要求自定义 Agent 编排 |
| GAP-08 | CMDB 无类模型 / IRE / 业务服务语义 | `servicenow-gap-analysis.md:19-89` | 投标要求 CSDM/IRE 对标 |
| GAP-09 | 字段级三方合并未实现 | `initialization-data-classification.md:79-81` | 合同承诺跨版本无损升级 |
| GAP-10 | RLS 仅 `changes`/`vectors` 两表 | `tests/multi_tenant/rls_matrix_test.go:1-25` | 等保/合规要求数据库层强制隔离 |
| GAP-11 | K8s 仅脚手架 | `deploy/k8s/README.md:32-45` | 合同承诺私有化 HA / 商业 SLA |

---

## 6. 标准化需求记录

### REQ-001 企业版边界与可承诺交付矩阵

| 字段 | 内容 |
|:---|:---|
| Requirement ID | REQ-001 |
| Customer Segment | 内部 IT 服务台、运维中心、MSP（尚未选定首个） |
| Industry | 通用企业 IT；优先物流/制造/电商之一（未决策） |
| Business Problem | 无法回答“哪些能力开源、哪些收费、交付边界在哪”，报价与合同无法形成 |
| Current Workflow | 销售凭口头解释；能力合同保留 7 个开放问题；业务文档为占位符 |
| Proposed Solution | 能力矩阵 + 边界决策记录 + 与 capability flag 对齐的企业能力声明 |
| Product Module | Governance / 商业化交付（非新菜单） |
| Priority | P0（分数 3.29） |
| Business Value | 使首次报价、PoC 范围和升级承诺有唯一依据 |
| Engineering Cost | 低（决策 + 文档 + 少量 flag 对齐），需要产品/商业决策人参与 |
| Acceptance Criteria | 7 个开放问题逐条有结论；每能力标 CE/EE/Industry 与依据；官网/销售文档口径一致 |
| Commercial Hypothesis | 明确边界后 4 周内可产出对外报价单与 PoC 范围（**未验证**） |
| Evidence Level | Observed Problem + Hypothesis |

### REQ-002 单一版本事实源与合并门禁

| 字段 | 内容 |
|:---|:---|
| Requirement ID | REQ-002 |
| Customer Segment | 所有私有化客户、交付工程师 |
| Industry | 通用 |
| Business Problem | 7 个版本表面互相矛盾，部署与升级承诺不可信 |
| Current Workflow | 各文档/配置独立记录版本；漂移基线只登记历史值 |
| Proposed Solution | 统一版本来源 + 跨表面一致性 CI 检查 + 必需门禁 |
| Product Module | Delivery Engineering / Documentation |
| Priority | P0（分数 2.63） |
| Business Value | 合同版本、`/version`、发布说明一致，降低交付争议 |
| Engineering Cost | 中低（6 处文档/配置修改 + 1 个 CI 检查） |
| Acceptance Criteria | 所有表面指向同一版本；一致性检查进入必需检查；README Go 版本与 go.mod 对齐 |
| Commercial Hypothesis | 消除部署版本争议，缩短验收沟通（**未验证**） |
| Evidence Level | Observed Problem |

### REQ-003 飞书入站消息→工单闭环（首个行业 PoC 入口）

| 字段 | 内容 |
|:---|:---|
| Requirement ID | REQ-003 |
| Customer Segment | 使用飞书的重资产运维团队（物流仓储 / 制造 IT-OT，未验证） |
| Industry | 优先物流或制造（待客户输入决策） |
| Business Problem | 群内故障消息需人工复制进工单，重复通知与漏单并存 |
| Current Workflow | IM 消息 → 人工判断 → 手工建单；现有连接器只验签/去重后 ACK |
| Proposed Solution | 验签/解密/持久去重 → 租户与权限校验 → 建单命令（durable outbox）→ 结果回执与审计；高风险动作保留人工确认 |
| Product Module | Connector + Ticket + AI-Native（可选智能分诊建议，不自动执行） |
| Priority | P0（分数 2.47） |
| Business Value | 可现场演示的“少一次人工录入”闭环，是行业 PoC 的核心故事 |
| Engineering Cost | 中高（需命令/outbox 复用、租户校验、幂等、回执、E2E） |
| Acceptance Criteria | 真实 HTTP 入口 E2E：新消息建单成功；重复消息不重复建单；跨租户/无权限拒绝；失败进死信可重放；审计完整；生产租户验收人工签字 |
| Commercial Hypothesis | 单渠道闭环可支撑 1.5–3 万 PoC 报价（价格来自占位文档，**未验证**） |
| Evidence Level | Observed Problem（能力缺口可复核）+ Hypothesis（付费意愿） |

### 其余候选（简表）

| ID | 客户/行业 | 业务问题 | 证据等级 | 当前判定 |
|:---|:---|:---|:---|:---|
| REQ-004 | 私有化客户 | 升级可能覆盖客户配置 | Observed Problem | P1，合同承诺升级时升 P0 |
| REQ-005 | AI-Native 早期采纳者 | 无法定义自己的 Agent 编排 | Hypothesis | P1，等客户输入 |
| REQ-006 | 私有化大客户 | 无 HA/HPA，不能承诺 SLA | Observed Problem | P1，合同承诺 HA 时升 P0 |

---

## 7. 商业化阻塞点（按阻塞强度排序）

1. **无 Verified Demand** — 全部需求为 Observed Problem 或 Hypothesis。没有客户确认就定企业版功能与价格，是把推测当承诺。
2. **商业边界未定义** — 7 个开放问题未决策，报价单无法生成（GAP-01）。
3. **事实源漂移** — 版本/成熟度至少 7 个矛盾表面，合同附件无法引用（GAP-02）。
4. **质量承诺不可证** — 最近 100 次 CI 39 次失败；docs-gate 不在必需检查；存量红灯未分类（GAP-04）。
5. **缺少可演示的行业闭环** — IM→工单未打通，数据库治理仅骨架，行业方案无产品化载体（GAP-03 + B 节）。
6. **交付形态不足** — K8s 仅脚手架，私有化 HA 与商业支持无依据（GAP-11）。
7. **成熟度表述越界风险** — workflow 维持 Preview 但跨域人工签字空白；不得把脚本绿灯写成 GA。

> 这些阻塞点中，**只有 #2/#3/#4 是本轮可以在不询问客户的前提下收敛的**；#1 必须由真实客户输入解决，AI 与仓库分析都不能代替。

---

## 8. 2026-11-27 复核：四条 P0 差距的实际收敛度

前文 §4 是 11-23 的差距定义，未改写。本节只回答一个问题：**经过 v1.6.60 → v1.6.61 两个版本，这四条 P0 各自收了多少？**

### 8.1 GAP-02 事实源漂移 → 版本面对账完成，发版流程仍有结构缺陷

**已收敛**：10 个受控版本表面 + 2 个 Go 徽章由 `scripts/check-version-consistency.js` 守卫，`--canonical=v1.6.61 --strict` 实测 PASS（漂移 0）。11-23 记录的 7 个矛盾面全部对齐。

**仍存在的结构缺陷（不是漂移，是设计问题）**：`Version Truth` 作业从 `git describe --tags --abbrev=0` 取 canonical。发版提交 `87cf29416` 在 `01:34:21Z` 推送到 main 时，tag `v1.6.61` 尚未创建（`01:34:32Z`）；该 run 的 attempt 1 因此拿 `v1.6.60` 作 canonical，把 1.6.61 的**正确**表面全部判为 FAIL，attempt 2 重跑才绿。

- **含义**：这条门禁的正确性依赖「tag 已先于 branch 存在」的时序巧合。用 `scripts/prepare-release.sh`（先打 tag 再 push）走流程不会踩；手工先 push commit 再打 tag 就必踩误报。
- **为什么不能只改 canonical 来源**：若改成读代码内常量，守卫就变成"自己证明自己"（常量本就来自同一个值），失去跨表面比对的意义。正解是让 CI 接受**显式 canonical 输入**（PR 用 base 分支最新 tag、push 到 main 用本次提交对应 tag 或提交内常量），或让 `prepare-release.sh` 成为唯一发版入口并在 CI 校验"提交已打好 tag"。
- **本轮未修**：属流程设计变更，需产品/发布决策，不宜在修红批次里顺手改。已登记为 GAP-02 的残留项。

### 8.2 GAP-03 IM 入站 → 单渠道闭环成立，跨渠道复用性未验证

飞书渠道已打通验收级证据链（验签 → 解密 → nonce 去重 → 消息级持久去重 → 租户解析 → 建单/审计/命令同事务 → AI 分诊建议 + 人工确认）。这是一条**可现场演示的真实业务闭环**，也是 P0 里唯一在 v1.6.61 前实质推进的一条。

**但要诚实标注三点**：
1. 默认 `observe_only`——不配置就不会建单，演示前需显式改租户配置；
2. 企微/钉钉**未复用**这套派发逻辑，等于每加一个渠道要重写一遍。这正是 GAP-07（缺声明式 ActionFlow）在业务侧的具体表现；
3. 真实租户签字栏空白，能力 `pilot`。**不得以此条对外宣称"IM 集成可用"**。

### 8.3 GAP-10 RLS → 比 11-23 描述更差，须下调对外口径

11-23 记「RLS 仅 2 表试点」。实测：策略层覆盖 **15 张表**（9 个迁移 `ENABLE ROW LEVEL SECURITY`，含 `tickets`/`incidents`/`problems`/`changes`/`configuration_items` 五张核心事务表），但：

- `docker-compose.prod.yml` 三处 `RLS_MODE=off`（含默认值）；
- `enforce` 的前置条件双连接池未接线（`AppDSN()`/`AdminDSN()` 零调用方）；
- 试点策略迁移 `database/rls/migrations/002_pilot_policies.sql` 不在迁移链，仅靠 `rebuild_and_verify.sh` 手工 `docker cp`。

**净结论：数据库层隔离零生效。** 比"2 表试点"更准确的说法是"策略已铺到 15 表，但运行时全部关闭，隔离 100% 依赖应用层谓词"。**对外承诺等保/合规类数据库级隔离前必须先做双连接池批次**，`docs/operations/rls-rollout.md` §3 已写明这是唯一需要写代码的前置项。

### 8.4 GAP-04 CI 红灯 → 失败面收敛到 2 项，但"必需检查"缺口未动

v1.6.61 后最近 60 次运行只有 2 次 failure，且两次都已修（§8.1 能力矩阵、`Sync Labels` 执行位）。这比 11-14 的 39/100 好得多。

**但 P0 判定不变**，原因不在失败数而在**门禁覆盖**：

| 门禁 | 实际状态 | 阻断合并？ |
|:---|:---|:---|
| `Lint` / `Build` / `Test` / `Frontend` | 分支保护必需检查 | ✅ 是 |
| `docs-gate` | C.6/C.7 hard、C.1–C.5 仅 `--strict`；PR 上 30 次里 6 次被 concurrency 取消 | ❌ 否（不在必需检查） |
| `Version Truth` | 在 `docs.yml` 内，11-24 新增 | ❌ 否（不在必需检查） |
| `fresh-install-gate` | 空库全新安装 | ✅ 是（v1.6.61 实跑 success） |
| `Security Scan` / `acl-gate` / `api-contract-check` | 均绿 | ❌ 否 |

分支保护仍是 4 项（`gh api .../branches/main/protection` 实测 `["Lint","Build","Test","Frontend"]`，`required_approving_review_count=1`）。**把 `docs-gate` 与 `Version Truth` 加进必需检查需在 GitHub 仓库设置里人工操作**，本批无法远程配置。

另有一条已修但不属 GAP-04 的：`backend-ci` 的 PR 路径过滤已在 #134 去掉（`push` 仍过滤），纯前端 PR 不再因缺检查被阻断。
