# EasyCloud 经验迁移至 ITSM 产品实施计划

## 需求重述

从已服务大型保险公司的云管理平台 EasyCloud 中，提取经过生产验证的业务经验、架构模式和设计思想，应用到当前 ITSM 产品开发中，补齐短板、增强竞争力。

---

## 一、EasyCloud 核心经验提炼

通过分析 EasyCloud 的业务分析文档、业务逻辑文档和代码架构，提炼出 **8 大可迁移经验**：

### 1. 配额管控与冻结机制（Quota Freeze/Deduct/Reclaim）
- **EasyCloud 实现**：三级配额操作 — 检查(checkQuota) → 冻结(freezeQuota) → 扣除(deductQuota)，含欠费跟踪
- **核心价值**：资源预留防超卖，冻结-扣除两阶段保证一致性
- **ITSM 应用场景**：
  - 服务请求资源配额管控（限制每租户的工单数、服务请求数）
  - SLA 承诺容量的预留与消耗跟踪
  - MSP 客户的服务额度管理

### 2. 服务目录 + 购物车 + 订单 全链路
- **EasyCloud 实现**：浏览目录 → 选择配置 → 加入购物车 → 提交订单 → 工作流审批 → 资源交付 → 计费
- **核心价值**：用户体验流畅，从选到买一步到位，支持多商品合并结算
- **ITSM 应用场景**：
  - 服务目录增加「购物车」模式，支持批量提交多个服务请求
  - 服务请求增加订单追踪视图，端到端可视化交付进度
  - 增加「我的订单」页面，统一查看所有请求状态

### 3. 多租户层级隔离模型（Tenant → System → Environment）
- **EasyCloud 实现**：租户(公司) → 系统(应用) → 环境(开发/测试/生产) 三级隔离
- **核心价值**：细粒度权限与数据隔离，满足大型组织管控要求
- **ITSM 应用场景**：
  - 当前 ITSM 仅有租户级隔离，缺少系统和环境维度
  - 增加环境标签（生产/测试/开发），工单、变更、发布可关联环境
  - 变更管理强制区分生产环境变更审批流程

### 4. 资源生命周期状态机
- **EasyCloud 实现**：VM 状态机（待实施 → 运行中/已停止 → 中间态 → 销毁），含状态转换约束表
- **核心价值**：状态转换有明确规则约束，防止非法操作
- **ITSM 应用场景**：
  - 当前工单/事件/变更的状态转换缺少硬性约束
  - 实现正式状态机：定义合法转换路径、禁止非法跳转
  - 变更管理特别需要：草稿 → 评审 → 已批准 → 实施中 → 完成/失败 的强制路径

### 5. 适配器模式统一外部集成
- **EasyCloud 实现**：VirtualMachineTarget 接口 + CloudStack/VMware/OpenStack 适配器
- **核心价值**：新增云平台只需加适配器，不改核心业务逻辑
- **ITSM 应用场景**：
  - CMDB 发现源适配器：统一接口，支持 Zabbix/Prometheus/Ansible/AWS/Azure 等
  - 监控告警源适配器：不同监控系统通过统一接口注入事件
  - 通知渠道适配器：邮件/短信/飞书/企业微信/钉钉 统一抽象

### 6. 日结账单 + 月度汇总 + 欠费跟踪
- **EasyCloud 实现**：每日消费记录 → 系统日账单 → 系统月账单 → 租户账单，含欠费催收
- **核心价值**：精细化成本追踪，支持按资源/按系统/按租户多维度分析
- **ITSM 应用场景**：
  - IT 服务成本分摊：按工单处理时长、SLA 等级、服务类型计算成本
  - 部门级 IT 预算管控：每部门 IT 服务消费月报
  - MSP 场景：客户账单与对账

### 7. 定时任务体系（Quartz）
- **EasyCloud 实现**：Quartz 定时任务 — 日消费计算、配额检查、账单生成
- **核心价值**：自动化周期性业务，减少人工干预
- **ITSM 应用场景**：
  - SLA 超时自动升级（已有基础，需增强定时扫描）
  - 工单超时自动转派/升级
  - 定期生成运营报告
  - 变更窗口到期提醒
  - 许可证到期预警

### 8. 基础设施层级模型
- **EasyCloud 实现**：数据中心 → 可用区 → Pod → 集群 → 主机 → 虚拟机
- **核心价值**：清晰的基础设施拓扑，支持影响分析
- **ITSM 应用场景**：
  - CMDB CI 拓扑增加基础设施层级视图
  - 事件影响分析：主机故障 → 自动关联受影响的 VM → 自动关联依赖这些 VM 的业务系统
  - 变更影响评估：主机维护变更 → 列出所有受影响服务

---

## 二、实施优先级排序

基于 **业务价值 × 实施难度** 评估：

| 优先级 | 经验 | 业务价值 | 实施难度 | 建议阶段 |
|--------|------|----------|----------|----------|
| **P0** | 资源生命周期状态机 | 极高 | 低 | 第一阶段 |
| **P0** | 适配器模式统一集成 | 高 | 中 | 第一阶段 |
| **P1** | 服务目录购物车模式 | 高 | 中 | 第二阶段 |
| **P1** | 多租户环境维度 | 高 | 中 | 第二阶段 |
| **P1** | 定时任务体系增强 | 高 | 低 | 第二阶段 |
| **P2** | 配额管控机制 | 中 | 中 | 第三阶段 |
| **P2** | IT服务成本分摊 | 中 | 高 | 第三阶段 |
| **P2** | 基础设施层级模型 | 中 | 中 | 第三阶段 |

---

## 三、分阶段实施计划

### 第一阶段：核心架构增强（P0）

#### 任务 1：资源生命周期状态机

**目标**：为 Ticket/Incident/Change/Problem 实现正式状态机，约束合法状态转换

**后端实现**：
1. 创建 `internal/statemachine/` 包
   - `statemachine.go` — 状态机核心：状态定义、转换规则、守卫条件
   - `ticket_sm.go` — 工单状态机配置
   - `incident_sm.go` — 事件状态机配置
   - `change_sm.go` — 变更状态机配置（最复杂，含审批门控）
   - `problem_sm.go` — 问题状态机配置
2. 状态机核心逻辑：
   ```
   type Transition struct {
       From      string
       To        string
       Guard     func(ctx context.Context, entity interface{}) bool
       OnEnter   func(ctx context.Context, entity interface{}) error
       OnExit    func(ctx context.Context, entity interface{}) error
   }

   type StateMachine struct {
       states      map[string]bool
       transitions []Transition
       initial     string
   }
   ```
3. 在 service 层集成：`ticket_core_service.go` 调用状态机验证转换合法性
4. 返回明确错误：非法转换返回 `code: 1006, message: "状态转换不合法: {from} → {to}"`

**前端实现**：
1. 根据当前状态动态显示可用操作按钮
2. 状态流转历史可视化

**涉及文件**：
- 新增：`itsm-backend/internal/statemachine/*.go`
- 修改：`itsm-backend/service/ticket_core_service.go`, `service/incident_*.go`, `handlers/change/service.go`
- 新增：`itsm-frontend/src/lib/statemachine/*.ts`

#### 任务 2：适配器模式统一外部集成

**目标**：建立统一的外部系统集成适配器框架

**后端实现**：
1. 创建 `integration/adapter/` 包
   - `adapter.go` — 适配器注册表与统一接口
   - `monitor/` — 监控源适配器接口
     - `monitor_adapter.go` — `MonitorAdapter` 接口定义
     - `zabbix_adapter.go` — Zabbix 实现
     - `prometheus_adapter.go` — Prometheus 实现
   - `notification/` — 通知渠道适配器接口
     - `notification_adapter.go` — `NotificationAdapter` 接口定义
     - `email_adapter.go` — 邮件实现
     - `feishu_adapter.go` — 飞书实现
     - `wechat_adapter.go` — 企业微信实现
   - `discovery/` — CMDB 发现源适配器接口
     - `discovery_adapter.go` — `DiscoveryAdapter` 接口定义
     - `ansible_adapter.go` — Ansible 实现
     - `cloud_adapter.go` — 云平台实现
2. 适配器接口定义：
   ```go
   type Adapter interface {
       Name() string
       Type() AdapterType
       HealthCheck(ctx context.Context) error
       Initialize(ctx context.Context, config map[string]interface{}) error
   }

   type AdapterRegistry struct {
       adapters map[string]Adapter
   }
   ```
3. 与现有 notification_service.go、cmdb_service.go 集成

**涉及文件**：
- 新增：`itsm-backend/integration/adapter/**/*.go`
- 修改：`itsm-backend/service/notification_service.go`、`service/cmdb_service.go`
- 新增：`itsm-frontend/src/lib/api/adapter-api.ts`

---

### 第二阶段：业务能力增强（P1）

#### 任务 3：服务目录购物车模式

**目标**：支持批量选择服务、购物车结算、订单追踪

**后端实现**：
1. 新增 Ent Schema：`shopping_cart.go`、`service_order.go`
2. 新增 Service：`shopping_cart_service.go`、`service_order_service.go`
3. 订单与工作流集成：订单提交自动触发审批工作流
4. 订单状态追踪：提交 → 审批中 → 已批准 → 交付中 → 已完成/已拒绝

**前端实现**：
1. 服务目录页增加「加入购物车」按钮
2. 新增购物车侧边栏/弹窗
3. 新增「我的订单」页面，含订单详情和进度追踪
4. 头部导航增加购物车图标和订单入口

**涉及文件**：
- 新增：`itsm-backend/ent/schema/shopping_cart.go`、`service_order.go`
- 新增：`itsm-backend/service/shopping_cart_service.go`、`service_order_service.go`
- 新增：`itsm-backend/controller/shopping_cart_controller.go`、`service_order_controller.go`
- 新增：`itsm-frontend/src/app/(main)/service-catalog/components/Cart*.tsx`
- 新增：`itsm-frontend/src/app/(main)/my-orders/`

#### 任务 4：多租户环境维度

**目标**：增加 System 和 Environment 维度，支持细粒度隔离

**后端实现**：
1. 新增 Ent Schema：`environment.go`（开发/测试/预发/生产）
2. 修改 Ticket/Incident/Change/Problem schema 增加 `environment_id` 字段
3. 变更管理：生产环境变更强制走完整审批链
4. 中间件增强：环境上下文传播

**前端实现**：
1. 工单/事件/变更表单增加「环境」选择
2. 管理后台增加环境管理页面
3. 列表页支持按环境筛选

**涉及文件**：
- 新增：`itsm-backend/ent/schema/environment.go`
- 修改：`itsm-backend/ent/schema/ticket.go`、`incident.go`、`change.go`
- 新增：`itsm-frontend/src/app/(main)/admin/environments/`

#### 任务 5：定时任务体系增强

**目标**：系统化定时任务管理，替代分散的 ad-hoc 实现

**后端实现**：
1. 创建 `internal/scheduler/` 包
   - `scheduler.go` — 任务注册、Cron 调度
   - `jobs/` — 具体任务实现
     - `sla_breach_scan.go` — SLA 超时扫描与自动升级
     - `ticket_escalation.go` — 工单超时升级
     - `change_window_reminder.go` — 变更窗口提醒
     - `license_expiry_check.go` — 许可证到期预警
     - `report_generation.go` — 定期运营报告
     - `quota_check.go` — 配额用量检查（为第三阶段准备）
2. 新增 Ent Schema：`scheduled_job.go` — 记录任务执行历史
3. 管理界面：查看任务状态、手动触发、暂停/恢复

**涉及文件**：
- 新增：`itsm-backend/internal/scheduler/**/*.go`
- 新增：`itsm-backend/ent/schema/scheduled_job.go`
- 修改：`itsm-backend/service/sla_monitor_service.go`、`service/escalation_service.go`

---

### 第三阶段：商业化能力（P2）

#### 任务 6：配额管控机制

**目标**：为租户/部门建立 IT 服务配额管理

**后端实现**：
1. 新增 Ent Schema：`service_quota.go`、`quota_usage.go`
2. 新增 Service：`quota_service.go`
   - `CheckQuota(tenantID, resourceType, amount)` — 检查配额
   - `FreezeQuota(tenantID, resourceType, amount)` — 冻结配额
   - `DeductQuota(tenantID, resourceType, amount)` — 扣除配额
   - `ReleaseQuota(tenantID, resourceType, amount)` — 释放配额
3. 与服务请求集成：提交请求时冻结配额，交付后扣除

**前端实现**：
1. 管理后台增加配额管理页面
2. 服务请求提交时显示配额余额
3. 配额不足时提示并阻止提交

**涉及文件**：
- 新增：`itsm-backend/ent/schema/service_quota.go`、`quota_usage.go`
- 新增：`itsm-backend/service/quota_service.go`
- 修改：`itsm-backend/handlers/service_request/service.go`
- 新增：`itsm-frontend/src/app/(main)/admin/quota/`

#### 任务 7：IT 服务成本分摊

**目标**：追踪 IT 服务成本，支持部门级成本分析和 MSP 对账

**后端实现**：
1. 新增 Ent Schema：`cost_record.go`、`department_budget.go`、`monthly_cost_report.go`
2. 新增 Service：`cost_service.go`
   - 日结：每日按服务类型、部门、租户统计成本
   - 月结：月度成本汇总报告
   - 预算预警：部门预算超支提醒
3. 成本计算维度：
   - 工单处理成本（按处理时长 × 人工费率）
   - SLA 等级成本（不同 SLA 等级不同单价）
   - 基础设施成本（关联 CMDB CI 的折旧/租赁成本）

**前端实现**：
1. 新增成本分析仪表盘
2. 部门 IT 消费月报
3. MSP 客户账单页面

**涉及文件**：
- 新增：`itsm-backend/ent/schema/cost_record.go`、`department_budget.go`
- 新增：`itsm-backend/service/cost_service.go`、`controller/cost_controller.go`
- 新增：`itsm-frontend/src/app/(main)/reports/cost-analysis/`

#### 任务 8：基础设施层级模型

**目标**：CMDB 增加基础设施层级拓扑，支持影响分析

**后端实现**：
1. 增强 CI 类型体系：预定义基础设施 CI 类型（数据中心、可用区、集群、主机、VM）
2. 增强 CI 关系：层级依赖关系（contains、depends_on、runs_on）
3. 影响分析算法：给定 CI 故障，向上追溯所有受影响业务服务
4. 与变更管理集成：变更影响评估自动列出受影响服务

**前端实现**：
1. 基础设施拓扑可视化（树形/图形）
2. 影响分析结果展示
3. 变更创建时自动显示影响范围

**涉及文件**：
- 修改：`itsm-backend/handlers/cmdb/`、`service/cmdb_service.go`、`service/ci_relationship_service.go`
- 新增：`itsm-backend/service/impact_analysis_service.go`
- 修改：`itsm-frontend/src/app/(main)/cmdb/`

---

## 四、风险与注意事项

| 风险 | 等级 | 缓解措施 |
|------|------|----------|
| 状态机引入导致现有流程中断 | 高 | 状态机初始配置与当前行为一致，渐进式增加约束 |
| 适配器框架过度设计 | 中 | 先实现 2 个适配器验证框架，再扩展 |
| 购物车增加服务目录复杂度 | 中 | 作为可选模式，不替代现有的单步提交流程 |
| 配额管控影响用户体验 | 中 | 配额不足时给出明确提示和申请入口 |
| 成本分摊计算精度 | 中 | 初期使用简化模型，逐步精细化 |

## 五、架构原则（源自 EasyCloud 经验）

1. **两阶段提交思想**：配额冻结 → 扣除，适用于所有资源预留场景
2. **适配器隔离外部系统**：核心业务不直接依赖外部 API
3. **状态机约束业务流转**：避免非法状态跳转
4. **定时任务自动化**：减少人工干预，保证时效性
5. **多级账单聚合**：明细 → 日结 → 月结 → 租户汇总

## 六、预估复杂度

| 阶段 | 后端工作量 | 前端工作量 | 测试工作量 |
|------|-----------|-----------|-----------|
| 第一阶段 | 状态机 3d + 适配器 4d | 状态机 1d + 适配器 1d | 3d |
| 第二阶段 | 购物车 4d + 环境 2d + 定时 3d | 购物车 4d + 环境 2d | 4d |
| 第三阶段 | 配额 3d + 成本 5d + 基础设施 4d | 配额 2d + 成本 3d + 基础设施 2d | 5d |
