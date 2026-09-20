# CMDB 与工作流 / ITIL 4 集成设计

## 设计目标

CMDB 不应该只是“配置项台账”，而应成为：

- 变更启用（Change Enablement）的风险输入源
- 事件管理（Incident Management）的影响分析源
- 问题管理（Problem Management）的根因追踪源
- 服务请求、服务目录、自动化工作流的上下文底座

本项目当前已经具备：

- CMDB：CI、关系、拓扑、影响分析、对账
- 工作流：BPMN 流程引擎、审批链、任务流转
- ITSM：事件、问题、变更、服务目录

下一步的重点不是再复制模块，而是建立统一的“CMDB 语义输入”。

## 一体化设计原则

### 1. 所有流程都引用同一套 CI 语义

统一使用以下字段驱动流程：

- `criticality`
- `environment`
- `install_status`
- `operational_status`
- `lifecycle_stage`
- `relationship_type`
- `discovery_source`

这样工作流、审批流、事件分诊和对账中心都能使用同一判断依据。

### 2. 工作流不直接维护风险，CMDB 提供风险上下文

工作流负责状态推进，CMDB 负责提供：

- 影响对象
- 依赖范围
- 当前事件负载
- 关键 CI 命中情况
- 推荐审批和回滚要求

### 3. ITIL 4 实践通过“CMDB + 流程策略”落地

建议将 ITIL 4 实践映射为系统能力：

- `Service Configuration Management`：CMDB、发现、对账、数据质量
- `Change Enablement`：基于 CI 风险的审批、CAB、变更窗口
- `Incident Management`：基于拓扑和依赖的影响范围分析
- `Problem Management`：事件聚类到 CI、关系链、Known Error
- `Service Request Management`：服务目录和 CI Type / Cloud Service 联动
- `Monitoring and Event Management`：告警与 CI 自动关联
- `Release Management`：发布对象与应用服务 / 组件 CI 绑定

## 推荐的关键联动场景

### 变更启用

变更创建或提交流程时，由 CMDB 自动输出：

- 受影响 CI 数量
- 是否命中关键 CI
- 是否存在高风险依赖
- 当前是否有未关闭事件
- 建议风险等级
- 是否需要 CAB
- 是否必须回滚计划

### 事件管理

事件建单时，自动根据 CI 输出：

- 上下游依赖
- 受影响业务服务
- 同类历史事件
- 潜在问题记录

### 问题管理

问题单应聚焦：

- 同一 CI 的重复事件
- 同一依赖链上的关联故障
- 高影响高频问题对象

### 服务目录 / 请求管理

服务目录项应绑定：

- `CI Type`
- `Cloud Service`
- 标准交付流程
- 默认审批链
- 默认配置模板

## 本次已落地能力

本轮新增了面向变更流程的 `CMDB 影响摘要` 接口，用于把 CMDB 数据直接供给变更审批与工作流：

- 自动统计受影响 CI 数
- 自动识别关键 CI
- 自动统计高风险依赖
- 自动统计未关闭事件
- 自动输出推荐风险等级与影响范围
- 自动给出工作流建议和 ITIL 4 实践建议

这一步的意义是：让 CMDB 不再只是“被查阅”，而是开始“驱动流程决策”。

## 下一阶段建议

### Phase 1

- 补 `install_status`、`operational_status`、`lifecycle_stage`
- 引入变更前检查策略：关键 CI 必须审批、必须回滚、必须窗口校验

### Phase 2

- 事件建单自动基于 CI 拓扑生成影响范围
- 问题单自动关联重复事件和根因候选 CI

### Phase 3

- 服务目录与 CI Class / Cloud Service 模型全面打通
- 流程模板与 CMDB Class 绑定

### Phase 4

- AI 生成变更风险建议
- AI 生成 CAB 摘要
- AI 生成事件影响分析和问题候选根因
