# CMDB 对标 ServiceNow 差距与改进建议

## 当前基础能力

当前项目已经具备了 CMDB 的第一层基础：

- `CI`、`CI Type`、`CI Relationship` 基础模型已存在
- 支持拓扑、影响分析、对账、云资源绑定、发现结果记录
- 前端已具备 CMDB、拓扑、对账、质量报表等页面骨架

这意味着项目已经跨过了“从 0 到 1”的数据建模阶段，但距离 ServiceNow CMDB 的核心竞争力，主要还差在“数据质量治理”和“业务语义建模”。

## 与 ServiceNow CMDB 的关键差距

### 1. 类模型还不够像 `Class-based CMDB`

ServiceNow CMDB 的核心不是“有一张 CI 表”，而是有清晰的类层级、继承关系和标准类约束。

当前系统问题：

- `configuration_items` 仍以通用字段 + `attributes JSON` 为主
- `CIType` 更像“类型标签”，还不是稳定的类层级
- 缺少类似 `cmdb_ci` / `cmdb_ci_server` / `cmdb_ci_db_instance` 这种标准化继承语义

建议：

- 引入 `class_code`、`parent_class_id`、`class_path`，把 `CIType` 升级为真正的 CI Class
- 区分平台标准类与租户扩展类
- 前期先按 CSDM 思路建设最小类集：`Business Application`、`Application Service`、`Server`、`Database`、`Container`、`Cloud Resource`

### 2. 缺少 IRE（Identification and Reconciliation Engine）

ServiceNow 的强项在于多来源数据不会直接互相覆盖，而是先经过识别规则、匹配规则、优先级策略。

当前系统问题：

- `discovery_source`、`source` 已存在，但没有来源优先级治理
- 对账更多是“找未绑定和孤儿对象”，不是完整 reconciliation
- 多来源导入时没有稳定的识别键策略

建议：

- 为每个 CI Class 定义识别规则，如 `cloud_resource_id`、`serial_number`、`hostname + tenant`
- 增加 `data_source_priority`、`last_seen_at`、`last_reconciled_at`
- 建立 IRE 决策链：`identify -> compare -> reconcile -> audit`

### 3. 生命周期与运行状态没有完全拆开

ServiceNow 会区分安装状态、运行状态、生命周期阶段，而不只是一列 `status`。

当前系统问题：

- 主要依赖 `status`、`environment`、`criticality`
- 无法精确表达“已安装但未投产”“已退役但仍保留记录”“运行异常但资产仍有效”

建议：

- 拆分为 `install_status`、`operational_status`、`lifecycle_stage`
- 后续把变更、发布、下线流程与生命周期联动

### 4. 关系已具备，但业务语义还不够强

当前系统已经支持拓扑和影响分析，这是很好的基础；但仍偏技术对象关系，缺少业务服务语义。

建议：

- 强化 `Business Service -> Application Service -> Application -> Infrastructure` 链路
- 引入服务映射视角，而不只是 CI 关系图
- 在关系层面增加 `relationship_source`、`confidence_score`、`effective_from/to`

### 5. 数据质量治理还不够闭环

ServiceNow CMDB 很强调 Completeness、Correctness、Compliance。

当前系统问题：

- 有 `CIAttributeDefinition`，但此前没有真正接入 CI 创建/更新校验
- 缺少质量分、必填覆盖率、唯一性违规、过期数据、孤立关系等治理指标

建议：

- 已补第一步：CI 创建/更新时按属性定义做必填、类型、默认值、唯一性校验
- 下一步增加 CMDB 健康分和质量规则引擎

## 本次已落地的改进

### 已完成

- 把 `CIAttributeDefinition` 接入后端仓储层创建/更新流程
- 支持属性默认值回填
- 支持必填校验
- 支持基础类型标准化：`string`、`integer`、`float`、`boolean`、`date`、`datetime`
- 支持基于属性定义的唯一性校验

这一步虽然不大，但非常关键，因为它把 CMDB 从“能存数据”推进到“开始治理数据质量”。

## 建议的演进顺序

### Phase 1：数据质量先行

- 完成属性定义全量管理 API
- 增加前端属性模板编辑器
- 建立 CMDB 质量评分：必填率、唯一率、发现新鲜度、关系完整度

### Phase 2：引入 IRE

- 定义类级别识别规则
- 增加来源优先级
- 多源导入改为“识别 + 合并 + 审计”

### Phase 3：对齐 CSDM

- 建立业务应用、应用服务、技术服务、基础设施分层
- 把事件、问题、变更、服务目录和 CMDB 主干模型串起来

### Phase 4：AI Native CMDB

- 用 AI 辅助生成 CI Class、属性定义、识别规则
- 用 AI 对发现结果做异常归因和关系推荐
- 用 AI 从工单、监控、日志中反推 CI 影响范围

## 结论

如果目标是“国内版 ServiceNow 风格开源 ITSM”，CMDB 一定不能只做成资产台账，而要做成：

- 有标准类模型的配置管理底座
- 有识别与对账引擎的数据治理中心
- 有服务语义和影响语义的关系图谱
- 能和流程、AI、连接器、插件市场联动的企业数字化核心数据层

当前项目已经有不错的雏形，下一阶段最值得优先投入的是：

1. 属性治理
2. IRE/对账引擎
3. CSDM 分层模型
4. 与事件/变更/服务目录联动
