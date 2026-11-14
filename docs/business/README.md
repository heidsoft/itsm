# 商业化与开源运营 · 文档入口

> 本目录是《CloudMesh 商业化战略与开源运营指导手册 V1.0》的落地锚点（手册 §13 要求将战略文档放入 `docs/business/`）。
> **本文件是唯一入口**：所有商业化/开源运营相关的战略、差距、规划、模板都应从本目录派生，**不得出现与 `COMMERCIALIZATION_STRATEGY.md` 冲突的产品定位或路线图**。

## 1. 上位战略（唯一权威源）

| 文件 | 说明 |
|---|---|
| [`COMMERCIALIZATION_STRATEGY.md`](./COMMERCIALIZATION_STRATEGY.md) | 战略指导手册 V1.0（定位、版本体系、架构治理、商业模式、开源运营、内容/销售、90 天计划、决策机制、风险、文档树）。**所有其他文件的上位指导文档。** |

## 2. 现状与差距

| 文件 | 说明 |
|---|---|
| [`GAP_MATRIX.md`](./GAP_MATRIX.md) | 手册 §14 第一步产物：已实现 / 不完整 / 缺失 / 有缺陷矩阵 + 优先级行动 + 红线。每次战略复盘先读此文件。 |

## 3. 既有派生文档（散落仓库，归并引用，不重写）

以下文档已存在，作为本战略的支撑证据/前置规划，按主题归并：

- **产品能力边界**：[`docs/product/open-source-release-capability.md`](../product/open-source-release-capability.md)、[`docs/product/itsm-commercial-capability-contract.md`](../product/itsm-commercial-capability-contract.md)
- **竞品与行业**：[`docs/product/servicenow-gap-analysis.md`](../product/servicenow-gap-analysis.md)、[`docs/product/industry-verticals-analysis-2026-08-01.md`](../product/industry-verticals-analysis-2026-08-01.md)
- **前置商业化蓝图**：[`plans/open-source-commercialization-2026q3-blueprint.md`](../../plans/open-source-commercialization-2026q3-blueprint.md)、[`plans/cmdb-commercialization-blueprint.md`](../../plans/cmdb-commercialization-blueprint.md)
- **社区运营雏形**：[`output/growth/05-community-playbook.md`](../../output/growth/05-community-playbook.md)、[`output/growth/README.md`](../../output/growth/README.md)
- **标签与 Issue 治理**：[`docs/label-schema.md`](../label-schema.md)
- **架构治理（ADR）**：[`docs/architecture/`](../architecture/)

## 4. 手册 §13 占位骨架（已建文件，均标注"待真实数据"，不得虚构）

> 为落实手册 §13 业务/运营/模板文档树而建的占位骨架。内容结构已就位，**真实客户、价格、成交、指标须由真实数据填充**，不得编造。

**业务（docs/business/）**

| 文件 | 对应手册 | 用途 |
|---|---|---|
| [`IDEAL_CUSTOMER_PROFILE.md`](./IDEAL_CUSTOMER_PROFILE.md) | §2 / §3.3 | 理想客户画像（行业/规模/痛点/预算） |
| [`PRICING_AND_PACKAGING.md`](./PRICING_AND_PACKAGING.md) | §3.2 / §5.2 | 开源免费 + 企业服务双轨定价（测试价、以 SOW 为准） |
| [`POC_PLAYBOOK.md`](./POC_PLAYBOOK.md) | §5.3 / §9 R1 | 首发场景 PoC 流程与验收 |
| [`SALES_PLAYBOOK.md`](./SALES_PLAYBOOK.md) | §8.4 / §7.3 | 销售材料清单与叙事主线 |
| [`COMPETITIVE_LANDSCAPE.md`](./COMPETITIVE_LANDSCAPE.md) | §2 / §3.2 | 公开事实竞品对比（不编造） |

**运营（docs/operations/）**

| 文件 | 对应手册 | 用途 |
|---|---|---|
| [`OPEN_SOURCE_PLAYBOOK.md`](../operations/OPEN_SOURCE_PLAYBOOK.md) | §6 / §7.3 | 开源运营节奏与内容转化 |
| [`ISSUE_TRIAGE.md`](../operations/ISSUE_TRIAGE.md) | §6.5 / §6.6 | Issue/PR 分诊 SLA 与步骤 |
| [`RELEASE_PROCESS.md`](../operations/RELEASE_PROCESS.md) | §6.4 #6 / §9 | 发布检查单与 GA 门槛 |
| [`COMMUNITY_METRICS.md`](../operations/COMMUNITY_METRICS.md) | §6.6 / §10.2 | 开源经营指标口径 |

**模板（docs/templates/）**

| 文件 | 对应手册 | 用途 |
|---|---|---|
| [`CUSTOMER_INTERVIEW.md`](../templates/CUSTOMER_INTERVIEW.md) | §13 模板 | 客户访谈记录模板 |
| [`POC_ACCEPTANCE.md`](../templates/POC_ACCEPTANCE.md) | §13 模板 / §5.3 | PoC 验收表（事先约定阈值） |
| [`FEATURE_REQUEST.md`](../templates/FEATURE_REQUEST.md) | §13 模板 / §6.4 #10 | 功能请求模板 |

## 5. 维护规则

- 每月复核一次商业假设、客户反馈、路线图；每季度复核定位/定价/开源边界/经营指标（手册结尾维护规则）。
- 任何重大策略变更记录日期、依据、影响范围、决策人。
- 对外文案（官网/README/公众号）必须与 `COMMUNITY.md` 口径一致：**v1.6.57 / Apache-2.0 / 生产加固未 GA**，不写"替代 ServiceNow"、不宣称 GA、不编造客户案例。
