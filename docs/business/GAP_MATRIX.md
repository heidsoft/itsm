# CloudMesh / heidsoft/itsm · 商业化与开源运营现状差距矩阵

> 本文件是《CloudMesh 商业化战略与开源运营指导手册 V1.0》（`COMMERCIALIZATION_STRATEGY.md`）的**第一步落地产物**（手册 §14："复核现有仓库，建立已实现 / 不完整 / 缺失 / 有缺陷矩阵"）。
> 审计基准：2026-11-14；仓库 `github.com/heidsoft/itsm` 本地工作树 `main@6dbc98e`；版本 **v1.6.57**；协议 **Apache-2.0**；官网 `cloudmesh-website`（部署于 39.97.54.216）。
> 审计方法：实地盘点仓库文件 + 官网 HTML + 手册逐条对照，不依赖记忆。

---

## 0. 一句话结论

仓库工程与开源运营成熟度**远超预期**（许可证、双语文档、标签体系、ADR、CI、docker-compose、MCP、前置商业化蓝图齐备）。当前**不是"缺文档"，而是"战略未归一到入口 + 一处对外商业信号严重脱节"**。最高优先级两件事：

1. **建立 `docs/business/` 规范入口**（本文件 + 战略文档已落位，下一步归并散落规划）。
2. **修复官网 `pricing.html` 与手册定价/交付模型的冲突**（最大商业化一致性风险，见 §4.1）。

---

## 1. 已实现（对标手册已达标）

| 手册条款 | 现状 | 证据 |
|---|---|---|
| §6.3 许可证 | Apache-2.0，含 NOTICE | `LICENSE`(Apache 2.0) + `NOTICE` |
| §6.4 仓库首页（定位/场景/截图/快速体验/架构/状态矩阵/Roadmap/模板/安全/贡献/商业入口） | 中英文 README + 功能状态矩阵 + 截图 + docker-compose + 架构文档 + Roadmap + Issue/PR 模板 + SECURITY + COMMUNITY | `README.md` / `README.en.md` / `ROADMAP.md` / `SECURITY.md` / `COMMUNITY.md` / `.github/` |
| §6.4 #2/#3/#10 贡献/行为准则/安全披露 | 齐备 | `CONTRIBUTING.md` / `CODE_OF_CONDUCT.md` / `SECURITY.md` |
| §6.4 #6 变更日志 | 极详尽 | `CHANGELOG.md`（~900KB） |
| §6.5 Issue 标签体系 | 43 标签，分类完整 | `docs/label-schema.md`（area 13 / kind 8 / priority 4 / status 7 / size 4 …） |
| §6.4 #9 / §6.5 模板 | 3 类 Issue 模板 + PR 模板 + CODEOWNERS + FUNDING + dependabot | `.github/ISSUE_TEMPLATE/`、`pull_request_template.md`、`CODEOWNERS`、`FUNDING.yml` |
| §6.4 #7 架构文档 | 20+ ADR + 概览 + mkdocs 站点 | `docs/architecture/`（adr-001~007、domain-ownership、tenant-isolation…）、`mkdocs.yml` |
| §6.4 #5 Docker Compose 安装 | dev/prod/e2e/fresh-gate 多套 | `docker-compose*.yml` |
| §1.2 现状/规划分离 | README 功能区状态矩阵（可用/预览）+ 独立能力说明 | `README.md` 状态表 + `docs/product/open-source-release-capability.md` |
| §6.1–6.2 开源版能力底线 | 核心 ITSM/CMDB/知识库/连接器/MCP 开源 | `README.md` 能力域表、`docs/tools/itsm-mcp.md` |
| §2 定位（官网） | index.html 守口径：Apache-2.0 可商用、AI 为预览、逐项公开、MSP 多租户 | `index.html:194/201/241` |
| 前置战略规划 | 已有多份商业化/社区蓝图 | `plans/open-source-commercialization-2026q3-blueprint.md`、`plans/cmdb-commercialization-blueprint.md`、`output/growth/05-community-playbook.md`、`docs/product/servicenow-gap-analysis.md`、`itsm-commercial-capability-contract.md` |

---

## 2. 不完整 / 待收口

| 项 | 状态 | 建议 |
|---|---|---|
| §13 战略文档入仓 | 本次已补 `docs/business/COMMERCIALIZATION_STRATEGY.md` | 下一步在 `docs/business/README.md` 建立入口，归并散落规划 |
| §13 `docs/business/` + `docs/operations/` 规范树 | 战略文档散落 `plans/`、`docs/product/`、`output/growth/`，无统一入口 | 以本目录为锚，用索引软链/引用，不重复造内容 |
| §6.5 标签自动同步 | 标签仅在 `docs/label-schema.md` 文档化，仓库无 label-sync action，靠维护者手动打标 | 加 `.github/actions` 或 probot/label-sync，按 schema 自动创建 |
| §6.6 开源经营指标 | 无采集机制（README→安装、首次部署成功率、PR 合并周期、外部贡献者数） | 建 `docs/operations/COMMUNITY_METRICS.md` 定义口径 + 轻量采集 |
| §3.3 首发可收费闭环 AI Email-to-Ticket | 需确认是否已端到端可演示（R1） | 核对 `itsm-ai-service` 与连接器现状，补 Demo 与验收指标 |
| §1.2 / §3.2 商业版边界 | 仓库"未 GA"，但官网 pricing 呈现可付费商用形态（见 §4.1） | 决策 pricing 处置 |

---

## 3. 缺失（手册要求但无；多数需真实数据，勿编造）

| 手册条款 | 建议文件 | 处理方式 |
|---|---|---|
| §13 业务文档树 | `IDEAL_CUSTOMER_PROFILE.md`、`PRICING_AND_PACKAGING.md`、`POC_PLAYBOOK.md`、`SALES_PLAYBOOK.md`、`COMPETITIVE_LANDSCAPE.md` | **先建占位骨架 + "待填真实数据"标注**，不虚构客户/价格 |
| §13 运营文档树 | `docs/operations/OPEN_SOURCE_PLAYBOOK.md`、`ISSUE_TRIAGE.md`、`RELEASE_PROCESS.md`、`COMMUNITY_METRICS.md` | `output/growth/05-community-playbook.md` 可升格/引用，不必重写 |
| §13 模板 | `docs/templates/CUSTOMER_INTERVIEW.md`、`POC_ACCEPTANCE.md`、`FEATURE_REQUEST.md` | 建占位模板 |
| §6.3 CLA/DCO 决策 | `CONTRIBUTING.md` 增补 DCO/CLA 段 | 需你定夺是否采用（影响企业贡献合规） |
| §7.3 内容转化路径 | 文章结尾 CTA（部署 Demo / 申请 PoC / 商业支持） | 核查公众号/官网文章是否系统化落地 |
| §8.4 销售材料清单 | 1 页介绍 / Deck / Demo 视频 / 功能矩阵 / 部署架构图 / 安全说明 / PoC 模板 / 报价单 / SOW / 访谈表 | 多数缺失，按 90 天计划 §9 补齐 |

---

## 4. 有缺陷 / 风险（需修）

### 4.1 【高】官网 `pricing.html` 与战略严重脱节 —— 最大商业化一致性风险

**现状（`cloudmesh-website/pricing.html`）：**
- 个人版 **免费**（5 个 Skill 安装、基础 AI 对话、适合个人用户）
- 专业版 **¥99/月**（50 Skill、优先支持、自定义 Skill）
- 企业版 **¥299/月**（不限 Skill、私有化部署）
- "30 天免费试用，无需信用卡"、"支持支付宝/微信支付/银行转账"、"3 分钟完成部署"

**与手册冲突：**
- **§5.2 定价模型不符**：手册 Professional **1.98–4.98 万/年**、Enterprise **8–20 万/年起**、私有化部署 **10–30 万起**、付费 PoC **1.5–3 万/项目**。官网价格低 **10–40 倍**且形态为自助 SaaS，与"年度授权 + 私有化交付 + 专业服务"完全两样。
- **§2.5 市场边界不符**：手册明确"暂不优先进入通用个人 AI 助手市场"，但 pricing 的"个人版 5 个 Skill、基础 AI 对话、适合个人用户"正是该定位。
- **§3.2 版本边界不符**：手册企业版支持为"合同约定"，私有化部署是**核心交付形态**而非企业版付费附加项；官网把私有化列为 ¥299/月权益项。
- **仓库口径冲突**：`COMMUNITY.md`/`README` 声明"**v1.6.57 生产加固未 GA**"，pricing 却呈现可付费商用形态，与 index.html（"Apache-2.0 可商用、AI 为预览、逐项公开"）也不一致。

**影响：** 对外商业信号混乱——潜在客户/合作方对"是否真开源、是否要付费、怎么买"产生误判；与开源信任建立路径（手册 §6.1）相悖。

**处置选项（待你决策）：**
- **A（推荐）：** 改造为"开源 + 企业服务"双轨——开源永久免费（Apache-2.0 自托管），商业仅列"私有化部署 / 专业服务 / PoC / 年度支持"企业级报价（对齐 §5.2），删除 ¥99/月自助 SaaS 与个人 AI 助手叙事。
- **B：** 若 pricing.html 服务于另一条 CloudMesh 子产品线（非 heidsoft/itsm），应在站内明确区隔品牌与适用对象，避免与 ITSM 开源定位混淆。
- **C：** 暂下线 pricing 入口，待 §3.2 商业边界与 §5.2 定价在仓库内定稿后再上线。

### 4.2 【中】标签体系未自动同步
`docs/label-schema.md` 已定义 43 标签，但仓库无 label 自动创建/同步 action，依赖维护者手动，违背 §6.5"每周 triage、缩短首次响应"的效率目标。建议加 label-sync 工作流。

### 4.3 【低】官网"全自动"用语
`ai-news.html:371` 出现"全自动"。若指完全无人值守的 Agent 执行，与 §0.3 原则 2（生产可靠性 > 自治、需审批/审计）存在表述风险，建议改为"自动化辅助 / 半自动 + 人工确认"。

---

## 5. 优先级行动（对齐手册 §14 六步 + §9 的 90 天计划）

| 优先级 | 动作 | 对应手册 | 产出 |
|---|---|---|---|
| **P0** | 战略文档入仓 + 建立 `docs/business/` 入口 | §13、§14 第一步 | 已完成本文件 + `COMMERCIALIZATION_STRATEGY.md` |
| **P0** | **决策 `pricing.html` 处置（§4.1）** | §2.5、§3.2、§5.2、§6.2 | 选项 A/B/C 择一执行 |
| **P1** | 归并散落规划到 `docs/business/` + `docs/operations/` 入口（引用不重写） | §13 | `docs/business/README.md` 索引 |
| **P1** | CLA/DCO 决策与 `CONTRIBUTING.md` 增补 | §6.3 | 决策记录 |
| **P1** | 建 §13 缺失文档**占位骨架**（标注待真实数据） | §13 | 5 个业务 + 4 个运营 + 3 个模板占位 |
| **P1** | 配 label 自动同步 action | §6.5 | `.github/` workflow |
| **P2** | 开源经营指标口径 + 轻量采集 | §6.6、§10.2 | `COMMUNITY_METRICS.md` |
| **P2** | 锁定首发场景 AI Email-to-Ticket，做端到端 Demo + 验收指标 | §3.3、§5.3、§9 R1 | Demo + PoC 验收表 |
| **持续** | 内容转化路径（§7.3）+ 销售材料清单（§8.4）按节奏补齐 | §7、§8、§9 | 文章 CTA、销售物料 |

---

## 6. 红线（始终守住，与既有口径一致）

- 协议 **Apache-2.0**；版本 **v1.6.57**；**生产加固未 GA**。
- 不写"替代 ServiceNow"、不擅自升 v1.6.58、不宣称 GA、不编造客户案例/准确率/节省比例。
- 安全底线、租户隔离、基本审计、数据可迁移性**不锁在付费版**（手册 §3.2 / §6.2）。
- 缺失的业务/运营文档用占位 + "待真实数据"标注，**不得虚构**客户、价格或指标。
