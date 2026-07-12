# ITSM 系统模块功能复盘与改善迭代计划

**日期**: 2026-07-10  
**范围**: 产品模块、前后端契约、质量门禁、企业级能力、AI Native 演进  
**目标**: 按模块复盘当前系统功能成熟度，并形成可执行的 v1.1-v1.5 改善迭代计划。  
**依据**: 当前代码扫描、`README.zh-CN.md`、`ROADMAP.md`、`docs/v1-ga/capability-matrix.md`、`docs/review/system-function-review-result-2026-07-01.md`、`docs/product/phased-improvement-plan.md`、`docs/review/servicenow-benchmark-2026-06-18.md`。

---

## 1. 总体结论

当前系统已经从“功能骨架”进入“企业级可用性打磨”阶段。前端 App Router 页面覆盖工单、事件、问题、变更、发布、服务目录、服务请求、CMDB、SLA、BPMN、审批、知识库、报表、MSP、Marketplace、AI、系统管理等核心模块；后端 Ent schema 和 controller/service 也覆盖了 ITIL 主流程和平台治理能力。

但下一阶段的主要矛盾不是继续堆模块，而是把已经存在的能力做实：

- **主流程闭环**: 每个角色至少一条端到端路径可自动回归，避免“页面存在但业务动作不完整”。
- **契约治理**: 后端路由、DTO、前端 API client、页面字段必须形成单一事实来源。
- **企业级增强**: RBAC、租户隔离、审计、SLA、工作流、CMDB 影响分析要成为默认门禁。
- **低代码与集成**: 表单设计器、BPMN 设计器、连接器生命周期是对标 ServiceNow 的关键差距。
- **AI Native 可度量**: AI 分诊、RAG、摘要、风险评估要有审计、评估集、准确率和失败降级。

推荐把 v1.1 定位为“可信可用 + 契约治理 + 角色闭环”，v1.2 定位为“低代码流程与体验增强”，v1.5 定位为“AI 评估与企业集成落地”。

---

## 2. 当前规模与证据

| 维度 | 当前观察 | 说明 |
|------|----------|------|
| 前端页面 | `itsm-frontend/src/app` 下超过 120 个 `page.tsx` | admin、cmdb、workflow、tickets、reports 等页面密集 |
| 后端 schema | `itsm-backend/ent/schema` 覆盖 90+ 业务表 | ITIL、CMDB、BPMN、SLA、Marketplace、AI audit 均有模型 |
| 后端测试文件 | 124 个 `*_test.go` | service/controller 已有回归基础，但覆盖率仍偏低 |
| 前端测试文件 | 37 个 `*.test.ts(x)` | 单测覆盖部分 API 和页面，E2E 角色闭环仍需补齐 |
| CLI 命令 | ticket、incident、change、cmdb、sla、workflow、connector、knowledge 等 | 已具备后续运维和自动化入口 |
| 历史验证 | 2026-07-02 后端全量测试、API smoke 已通过 | 覆盖率基线记录为 1.3%，Jest 有 open handle 治理项 |

---

## 3. 模块成熟度复盘

### 3.1 工单管理

**现状**: 工单 CRUD、详情、评论、附件、标签、模板、视图、SLA、自动化规则、智能分派、AI 创建、工作流状态查询等入口齐全。后端存在 `ticket_controller.go`、`ticket_service.go`、`ticket_lifecycle_service.go`、`ticket_workflow_service.go` 等服务支撑。

**成熟度**: 4/5。

**主要缺口**:

- 缺少 Kanban / 队列化处理视图，工程师日常处理效率不如成熟 ITSM。
- 生命周期端到端 E2E 不足，尤其是分派、接受、解决、关闭、重开、评分、SLA 触发的组合路径。
- 列表分页、筛选、排序和统计指标需要与 dashboard 保持一致。

**改善项**:

- v1.1 增加工单 Kanban 和批量操作。
- v1.1 建立工单生命周期契约测试和角色 E2E。
- v1.2 支持自定义字段和表单模板绑定。

### 3.2 事件管理

**现状**: 事件创建、列表、详情、编辑、升级、确认、解决、关闭、告警规则、事件指标、根因入口均已铺开。历史修复已处理创建默认值和字段校验问题。

**成熟度**: 3.5/5。

**主要缺口**:

- 告警接入到 incident 的自动化闭环仍需用真实监控数据验证。
- 事件转问题、关联 CMDB、关联 SLA 风险需要角色化回归。
- 分派和升级动作的审计完整性需要专项检查。

**改善项**:

- v1.1 建立“告警 -> 事件 -> 问题/变更”的 API smoke。
- v1.1 增加事件升级审计和租户隔离测试。
- v1.5 接入真实监控连接器和 SLA 风险预测。

### 3.3 问题管理

**现状**: 问题、Known Error、RCA、趋势分析、问题调查等入口存在，后端有 problem 和 investigation 控制器/服务。

**成熟度**: 3.5/5。

**主要缺口**:

- 问题与事件、变更、知识库之间的闭环需要更强的流程串联。
- Known Error 与 workaround 的复用价值需要在工单/知识推荐中体现。
- RCA AI 能力仍偏实验性，需要真实评估集。

**改善项**:

- v1.1 补“事件转问题、问题生成已知错误、知识沉淀”的端到端用例。
- v1.2 在工单详情和知识库中展示 Known Error 推荐。
- v1.5 做智能 RCA 评估和人工反馈闭环。

### 3.4 变更与发布管理

**现状**: 变更、CAB、PIR、标准变更、发布管理均有入口，CMDB 影响分析和审批体系已有基础。

**成熟度**: 3.5/5。

**主要缺口**:

- 风险评估、CAB 审批、实施窗口、回滚、PIR 的状态机需要更严格。
- 发布和变更之间的关联策略需要产品语义收敛。
- 标准变更模板仍应标为实验性，避免和完整变更流程混淆。

**改善项**:

- v1.1 固化变更状态机和审批回归测试。
- v1.1 交付标准变更模板 MVP。
- v1.5 引入 AI 变更风险评估，结合 CMDB、历史事件、SLA 风险。

### 3.5 服务目录与服务请求

**现状**: 服务目录、服务请求、我的请求、审批待办、管理端服务目录均已接入。历史修复已处理申请、取消、完成、审批待办、字段命名兼容等问题。

**成熟度**: 4/5。

**主要缺口**:

- 服务目录申请表单仍缺低代码字段配置能力。
- 管理端目录配置、用户端申请、审批端处理三者需要统一验收矩阵。
- 服务请求交付任务与 BPMN 流程绑定还可加强。

**改善项**:

- v1.1 建立服务目录三角色 E2E。
- v1.2 交付表单设计器 MVP，支持字段类型、必填、默认值、校验、可见条件。
- v1.2 服务请求交付任务通过 BPMN 编排。

### 3.6 CMDB

**现状**: CI 类型、CI 实例、关系、拓扑、云资源、调和、导入导出、历史、保存视图等模型和页面完整。文档已明确 CMDB 不等于资产表。

**成熟度**: 3.5/5。

**主要缺口**:

- 自动发现和调和需要真实数据源闭环。
- 影响分析要增加递归深度、关系类型、租户边界和性能保护测试。
- CI 类型扩展字段与服务目录表单设计器可共用低代码能力。

**改善项**:

- v1.1 补 CMDB impact analysis 集成测试。
- v1.2 建立 CI schema / 自定义字段配置中心。
- v1.5 接入 Kubernetes / 云厂商发现任务和调和策略。

### 3.7 SLA 与升级

**现状**: SLA definition、policy、template、monitor、alert、dashboard、escalation matrix 均有后端和前端入口。

**成熟度**: 3.5/5。

**主要缺口**:

- 业务时间、节假日、暂停计时、重开后的 SLA 重算需要规则固化。
- SLA 合规报表与工单/事件统计口径要一致。
- 升级通知和连接器通知需要运行时验证。

**改善项**:

- v1.1 补 SLA 业务时间和状态转换测试。
- v1.1 统一 SLA dashboard、reports、ticket detail 的指标口径。
- v1.5 增加 SLA breach forecast skill。

### 3.8 BPMN 工作流与审批

**现状**: 流程定义、实例、任务、变量、历史、审计、版本、流程绑定、工作流 SLA、审批待办等入口较完整。历史文档指出审批模板、工单流转、BPMN 工作流仍有语义割裂。

**成熟度**: 3.5/5。

**主要缺口**:

- 可视化设计器能力仍基础，需引入更成熟的 BPMN 编辑能力。
- 审批模板与 BPMN candidateGroups / user task 的关系需要统一。
- 流程历史、审计和业务对象追踪需要产品化展示。

**改善项**:

- v1.1 明确审批模板、工单流转、BPMN 三者边界，并在 UI 中显式标注。
- v1.2 引入 bpmn-js 或兼容层，支持可视化建模、节点配置、版本发布。
- v1.2 审批记录关联 `process_instance_id`，业务对象可反查流程实例。

### 3.9 知识库与 RAG

**现状**: 知识文章、版本、评审、全文搜索、RAG 服务、知识推荐入口已具备。AI/RAG 当前应继续标注为实验性。

**成熟度**: 3.5/5。

**主要缺口**:

- RAG 的权限过滤、引用来源、召回质量和降级策略需要专项验证。
- 知识沉淀与事件/问题/工单解决方案之间的闭环不够强。
- 缺少可量化的 RAG hit-rate 与人工反馈。

**改善项**:

- v1.1 补知识权限过滤和版本状态测试。
- v1.2 将 Known Error、工单解决方案转知识作为标准动作。
- v1.5 建立 RAG 评估集，目标 hit-rate >= 70%。

### 3.10 AI Native 能力

**现状**: LLM gateway、triage、summarize、RAG、AI audit、prompt template、tool invocation 等模型和服务存在。onboarding 中仍有 mock AI 演示，需清楚标注。

**成熟度**: 2.5/5。

**主要缺口**:

- AI 功能缺少统一评估集和回归门禁。
- AI 输出的 accepted/rejected、operator feedback、成本、延迟需要产品化。
- 高风险动作必须经过权限、审计和人工确认。

**改善项**:

- v1.1 交付 AI audit console，至少可查看、筛选、接受/拒绝建议。
- v1.5 建立 AI evaluator，分诊准确率目标 >= 85%。
- v1.5 将 AI 变更风险评估和 SLA 预测作为可度量 skill。

### 3.11 连接器、Marketplace、Skill、Plugin

**现状**: Marketplace、connector lifecycle、Feishu callback、CLI connector 命令、installation 模型已存在。README 和 roadmap 已把飞书、钉钉、企微、Webhook 作为重点。

**成熟度**: 3/5。

**主要缺口**:

- 连接器 manifest、配置 schema、凭据加密、健康检查、测试发送、禁用/卸载需要闭环验收。
- 入站回调验签、幂等、租户隔离、审计需要按高风险面处理。
- Skill / Plugin 市场还处在架构方向，未形成可交付 MVP。

**改善项**:

- v1.1 交付 connector marketplace v1，优先 Feishu、DingTalk、WeCom、Webhook。
- v1.2 将连接器能力注册为 BPMN ServiceTask。
- v1.5 建立 skill registry v1，包含 manifest、权限、输入输出和审计。

### 3.12 权限、多租户与 MSP

**现状**: RBAC、菜单、角色、权限、用户、团队、部门、租户、MSP 管理均有页面和后端模型。历史 checklist 已提出角色闭环测试。

**成熟度**: 4/5。

**主要缺口**:

- 菜单权限、API 权限、数据租户过滤需要联合验证，不能只靠隐藏菜单。
- tenant_admin、security、approver、engineer、end_user 等角色缺少完整 E2E。
- MSP 管理端体验和配额/计量可观测性仍需产品化。

**改善项**:

- v1.1 建立角色 E2E 测试矩阵。
- v1.1 增加跨租户读写回归测试，覆盖工单、CMDB、知识、服务请求。
- v2.0 再推进 MSP billing 和使用量计费。

### 3.13 Dashboard、报表与运营

**现状**: dashboard、reports、ticket analytics、SLA dashboard、工作流 dashboard、CMDB quality、incident trends 等入口已经存在。

**成熟度**: 3/5。

**主要缺口**:

- 部分高级报表组件仍有 mock 数据风险，需要逐项接入真实 API。
- 报表指标口径需要和业务模块统一。
- 缺少用户自定义 dashboard 和可保存视图。

**改善项**:

- v1.1 清理运营报表 mock，未接入真实数据的功能显式标注实验性。
- v1.2 支持自定义 dashboard widget、布局保存和共享。
- v1.5 引入性能预算和运营指标看板。

### 3.14 开源交付与工程质量

**现状**: Docker Compose、CI、GA gate、README、SECURITY、CONTRIBUTING、CLI、监控配置均已存在。历史记录显示后端全量测试可通过，但覆盖率基线仍低。

**成熟度**: 3.5/5。

**主要缺口**:

- 后端总覆盖率低，核心服务和控制器需要重点补齐。
- controller/service 文件仍偏大，维护成本上升。
- OpenAPI 文档缺失，不利于外部贡献和企业集成。

**改善项**:

- v1.1 覆盖率从 1.3% 拉升到 40%，优先服务层和控制器关键路径。
- v1.1 拆分大 controller，按领域和动作分组。
- v1.4 交付 OpenAPI / Swagger UI 并纳入 CI。

---

## 4. 优先级矩阵

| 优先级 | 主题 | 影响 | 建议版本 |
|--------|------|------|----------|
| P0 | 角色闭环 E2E、API 契约矩阵、跨租户测试 | 防止核心流程不可用或越权 | v1.1 |
| P0 | 工单/事件/服务请求/变更状态机回归 | 保障 ITIL 主流程可信 | v1.1 |
| P1 | Connector marketplace v1 | 企业落地必须依赖飞书/企微/钉钉/Webhook | v1.1 |
| P1 | AI audit console | AI Native 能力可解释、可追溯 | v1.1 |
| P1 | 工单 Kanban、批量操作、通知中心 UX | 提升操作员日常效率 | v1.1 |
| P2 | 表单设计器、BPMN 可视化设计器 | 对标 ServiceNow 低代码核心 | v1.2 |
| P2 | CMDB discovery / reconciliation | 从手工 CMDB 走向自动化治理 | v1.5 |
| P2 | RAG / AI evaluator | AI 从演示走向可度量 | v1.5 |
| P3 | SSO/LDAP/SAML、OpenAPI、多语言 | 企业集成和开源贡献体验 | v1.4-v1.5 |

---

## 5. 改善迭代计划

### Sprint A: v1.1 基线可信度与契约治理

**周期**: 2-3 周  
**目标**: 让已有模块具备可回归、可发布、可验收的基本可信度。

交付项:

- 建立 API 契约矩阵：模块、页面、前端 client、HTTP 方法、后端路由、DTO、权限、测试覆盖。
- 统一列表分页字段：`page`、`pageSize`、`total`、`totalPages`。
- 清理 controller 直接返回 Ent 模型的风险点，关键响应统一 DTO。
- 补齐 P0 角色闭环 E2E：end_user、engineer、approver、sd_manager、super_admin、security、tenant_admin。
- 补齐跨租户回归：工单、服务请求、CMDB、知识、审批。
- 修复 Jest open handle，保证前端测试稳定退出。

验收:

- `cd itsm-backend && go test ./...`
- `cd itsm-frontend && npm run type-check && npm run build`
- 角色 E2E 至少覆盖 7 条主路径。
- API 契约矩阵覆盖 P0/P1 模块。

### Sprint B: v1.1 操作员体验与 ITIL 主流程闭环

**周期**: 3-4 周  
**目标**: 提升工程师和服务台经理的日常使用效率。

交付项:

- 工单 Kanban：按状态/队列拖拽，后端状态机校验。
- 批量操作：批量分派、关闭、导出，必须有权限和审计。
- 事件 -> 问题 -> Known Error -> 知识沉淀闭环。
- 变更 CAB 审批、实施、回滚、PIR 回归。
- SLA 业务时间、暂停、重开、升级规则测试。
- 通知中心 UX 重构，统一站内、邮件、Webhook、IM 消息状态。

验收:

- 每条 ITIL 主流程至少 1 条 API 集成测试 + 1 条前端 smoke。
- 状态机非法转换返回明确错误。
- 所有高风险动作产生审计记录。

### Sprint C: v1.1 Connector Marketplace v1 与 AI Audit

**周期**: 3-4 周  
**目标**: 形成企业集成和 AI 可追溯的最小闭环。

交付项:

- Connector manifest、配置 schema、安装、启用、禁用、卸载、健康检查、测试发送。
- Feishu、DingTalk、WeCom、Webhook 四类连接器至少完成一条真实发送路径。
- 入站 webhook 验签、幂等、租户解析和审计。
- AI audit console：查看 prompt version、model、confidence、输入摘要、输出摘要、operator feedback。
- AI 建议接受/拒绝写入反馈，为 evaluator 预留数据。

验收:

- 连接器 secrets 不返回前端，仅返回 masked metadata。
- 连接器动作和 AI 建议动作均有 audit log。
- LLM 不可用时有规则降级或明确失败提示。

### Sprint D: v1.2 低代码流程平台

**周期**: 6-8 周  
**目标**: 让流程和表单定制成为产品核心能力。

交付项:

- 表单设计器 MVP：字段、校验、默认值、可见条件、布局、版本。
- 服务目录申请表单绑定设计器。
- CI 类型扩展字段绑定设计器。
- 引入 BPMN 可视化设计器，支持用户任务、审批节点、服务任务、条件网关。
- 审批模板与 BPMN 绑定，审批记录关联 `process_instance_id`。
- 流程版本发布、灰度、回滚和审计展示。

验收:

- 普通管理员可创建一个服务目录表单并绑定审批流程。
- 业务对象详情页可反查流程实例、任务、审批记录、变量。
- 旧审批模板保持兼容，不破坏历史数据。

### Sprint E: v1.5 AI Native 与 CMDB 自动化

**周期**: 8-10 周  
**目标**: AI 和 CMDB 从实验性能力进入可度量、可运营阶段。

交付项:

- AI evaluator v1：分诊准确率、摘要质量、RAG hit-rate、人工反馈。
- RAG v2：权限过滤、版本状态过滤、引用来源、混合检索、重排。
- AI SLA 预测：基于历史工单、队列、优先级、SLA 策略输出违约风险。
- AI 变更风险评估：结合 CMDB 影响、历史事件、窗口期和回滚方案。
- CMDB discovery：Kubernetes / 云资源发现，调和策略，重复 CI 识别。
- 性能预算：top 10 API k6 baseline，纳入 CI 趋势跟踪。

验收:

- AI 分诊准确率 >= 85%，RAG hit-rate >= 70%。
- AI 输出全部有审计、版本、置信度和人工反馈。
- CMDB 重复发现不产生重复 CI。
- 影响分析有递归深度、租户隔离和性能保护。

---

## 6. 建议立即创建的任务

| 编号 | 任务 | Owner 建议 | 产出 |
|------|------|------------|------|
| T1 | 生成 API 契约矩阵脚本 | Backend + Frontend | `docs/review/api-contract-matrix.md` |
| T2 | 角色 E2E 测试矩阵落地 | QA + Frontend | Playwright specs |
| T3 | 工单 Kanban 技术设计 | Frontend + Backend | 设计文档 + API 状态机校验 |
| T4 | Connector lifecycle 验收清单 | Platform | manifest/schema/health/audit 测试 |
| T5 | AI audit console 产品设计 | AI + Frontend | 页面、筛选、反馈 API |
| T6 | 表单设计器 RFC | Product + Architecture | JSON Schema/FormIO/自研方案对比 |
| T7 | BPMN 与审批统一 RFC | Workflow | 兼容策略、迁移路径、数据模型 |
| T8 | 覆盖率补齐 Sprint | Backend | service/controller 覆盖率提升到 40% |

---

## 7. 风险与决策点

| 风险 | 影响 | 决策建议 |
|------|------|----------|
| 页面数量增长快但契约不统一 | 新增功能容易引入 404、字段错配、假成功 | v1.1 先做契约矩阵和 DTO 治理 |
| AI 功能宣传强于评估 | 企业客户难以信任 AI 输出 | v1.1 做 audit，v1.5 做 evaluator |
| 审批模板、工单流转、BPMN 并行 | 用户无法判断应该在哪配置流程 | v1.2 前完成统一语义和迁移策略 |
| CMDB 自动发现未闭环 | CMDB 数据质量难以维持 | v1.5 以 discovery + reconciliation 为核心 |
| 覆盖率偏低 | 重构和迭代风险高 | 覆盖率提升作为 v1.1 发布门禁 |
| 连接器凭据和回调安全 | 企业集成高风险面 | secrets masking、验签、幂等、审计作为硬门禁 |

---

## 8. 下一步执行顺序

1. 先冻结 v1.1 范围：契约治理、角色闭环、Connector v1、AI audit、工单 Kanban。
2. 为 P0/P1 模块生成 API 契约矩阵，不再靠人工记忆对齐前后端。
3. 补角色 E2E 和跨租户回归，确保开源用户和企业评估路径真实可跑。
4. 将表单设计器、BPMN 设计器、审批统一方案作为 v1.2 RFC，不急于直接施工。
5. 将 AI evaluator、RAG v2、CMDB discovery 放入 v1.5，以可度量指标验收。

---

## 9. 本次复盘判定

系统功能覆盖已经达到“开源 AI-Native ITSM 基础完整”的水平，下一轮不建议继续横向扩模块。最优迭代路径是：

**v1.1 把可信度做硬，v1.2 把定制能力做深，v1.5 把 AI 和连接器做实。**

这条路线最符合当前项目定位：面向国内企业数字化流程治理，对标 ServiceNow，但保持开源、轻量、私有化、AI Native 和连接器生态优势。
