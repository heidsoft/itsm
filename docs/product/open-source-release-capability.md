# 开源产品能力说明

> Status: current。最后复盘：2026-08-21。本文描述当前源码可交付的业务边界，不以页面、规划或历史测试报告代替运行时事实。

## 产品承诺

开源用户可以从干净检出开始，通过 Docker Compose 部署一个可登录、可验证、可扩展的 ITSM 实例。产品围绕“请求进入—类型解析—分派与时限—流程执行—解决与审计”构建业务主线，并以 CMDB、知识、AI 和连接器提供上下文与扩展能力。

## 使用角色与核心闭环

| 角色 | 可以完成的核心工作 |
|:---|:---|
| 员工/请求人 | 从服务门户或工单入口提交请求，填写类型驱动的动态表单，跟踪处理状态 |
| 服务台工程师 | 分类、分派、关联 CI、处理 SLA、记录沟通并解决工单或事件 |
| 流程参与人 | 接收 BPMN 用户任务，完成审批或处理，并保留实例与执行历史 |
| ITIL 负责人 | 管理事件、问题、变更、服务目录、知识和 SLA 策略 |
| CMDB 管理员 | 管理 CI 类型、实例、关系、拓扑、历史与影响分析 |
| 平台管理员 | 管理租户、用户、角色权限、TicketType、流程绑定、审计与部署状态 |

工单入口的当前执行链路为：

`TicketType/Preset → 动态表单校验 → Ticket 与类型快照持久化 → Assignment/SLA/Workflow 统一绑定解析 → command/outbox → Worker 执行 → 历史与审计`

TicketType 是运行时配置，不是前端静态枚举。创建工单时会校验租户、启用状态、字段类型、必填项和引用权限，并保存 `ticketTypeId`、类型快照与 `formFields`；后续类型配置变化不会抹去已创建工单的业务语义。

## 当前能力矩阵

状态含义：**GA 候选**表示可进入企业生产验收；**Pilot**表示存在真实实现，但仍需按部署场景补齐闭环或非功能验证；**Disabled/规划中**不得作为交付承诺。

| 业务域 | 状态 | 当前可验证能力 | 主要边界 |
|:---|:---:|:---|:---|
| 工单、事件 | GA 候选 | 生命周期、分类、分派、CI、SLA、评论、附件、可靠工作流触发 | 仍需按目标容量验证恢复与并发 |
| TicketType 与动态表单 | GA 候选 | 字段设计、完整 Renderer、类型快照、Preset 安装、归档恢复、Workflow/SLA/Assignment 绑定、独立 RBAC 和审计 | 复杂表单升级与大规模配置需专项验收 |
| 变更管理 | **Pilot** | 风险、受影响 CI、审批、实施/回滚计划、PIR 基础 | BPMN 审批与业务状态落库尚非原子；窗口冲突和自动化回滚仍需强化；发布前需重新评估 GA 候选资格 |
| 问题/Known Error | Pilot | 根因、临时方案、关联事件、趋势热点、评论、知识沉淀基础 | CI 与知识发布闭环仍需加强 |
| 服务目录/服务请求 | Pilot | 目录、请求、审批与服务任务基础 | 目录版本、补偿与履约闭环需加强 |
| CMDB 核心 | GA 候选 | CI 类型、实例、关系、历史、拓扑、影响分析、导入导出基础 | 数据质量、规模、恢复与调和需场景验收 |
| 云发现 | Pilot | 云账号、资源与阿里云适配基础 | 生产级调度、密钥、Diff 与退役治理待完善 |
| BPMN/审批 | Pilot | 定义、版本、绑定、实例、任务、变量和历史；工单/事件使用持久化命令执行 | 剩余业务域需继续统一可靠触发 |
| SLA | GA 候选 | 策略、业务日历、截止时间、预警、违规与指标 | 暂停恢复和跨领域一致性需持续验证 |
| 知识/RAG | Pilot | 文章、可见性过滤、关键词/向量检索、推荐与降级 | 索引删除一致性和质量评估需加强 |
| AI | Pilot | LLM Gateway、分诊、摘要、RAG 和审计框架 | 不默认自动决策；Evaluator 与反馈闭环未达 GA |
| 通知/连接器 | Pilot | 可靠通知 outbox、投递审计、连接器生命周期与市场框架 | 真实渠道验签、健康检查与重放运维需逐项验收 |
| RBAC/多租户 | GA 候选 | 角色权限、Endpoint ACL、租户过滤、TicketType 独立权限、审计 | 新域仍必须补 ACL 与跨租户回归 |

## 约束

- 默认部署路径必须以 `docker compose` 和 `.env.example` / `.env.prod.example` 为入口，不依赖本地私有配置。
- 前端默认后端地址统一为 `http://localhost:8090`；Docker 生产构建使用 `http://itsm-backend:8090`。
- API 健康检查以 `/api/v1/health` 为准。
- 所有用户可见的写入操作必须真实持久化，或显式提示未接入后端；禁止用本地模拟数据伪装成功。
- 默认账号只能作为开发/首次安装引导出现，生产部署文档必须要求更改密码和密钥。
- 多租户、RBAC、审批、TicketType 引用校验和服务请求状态流转属于发布门禁，不能作为演示功能处理。

## 实现契约

- Actors: 系统管理员、服务台工程师、审批人、普通员工、开源部署者。
- Surfaces: `README.md`、`.env.example`、`.env.prod.example`、`docker-compose.yml`、`docker-compose.prod.yml`、前端自助门户、后端 `/api/v1/*`。
- States and transitions: TicketType 支持 draft/active/inactive 与归档恢复；禁用或跨租户类型不得用于新建工单。ITIL 状态流转由后端服务校验。
- Interfaces: HTTP JSON 使用 camelCase，Controller 返回 DTO；工单类型以 `/api/v1/ticket-types` 和 `/api/v1/ticket-type-presets` 为真实来源，服务目录以 `/api/v1/service-catalogs` 和 `/api/v1/service-requests` 为真实来源。
- Data implications: 收藏、评分、门户配置、服务分析、目录导出在后端模型和接口补齐前，不应显示为可持久化能力。
- Operator requirements: 发布前需要通过前端 `type-check`、前端生产构建、后端 `go test ./...`、Docker Compose 健康检查和登录烟测。

## 非目标

- 不在本能力中新增商业化计费、多组织 SaaS 订阅或云市场分发。
- 不承诺所有规划中的 AI 能力在无 LLM Key 时完整可用；必须有清晰降级或关闭入口。
- 不用前端假数据替代未完成的后端服务。

## 开源用户的验收路径

1. 按根目录 README 启动开发环境，确认前后端健康检查通过。
2. 使用管理员创建或安装一个 TicketType Preset，配置动态字段及 Workflow/SLA/Assignment 绑定。
3. 使用普通用户读取启用类型并创建工单，确认非法字段、禁用类型和跨租户引用被拒绝。
4. 确认工单保存类型/表单快照，并产生预期的 SLA、流程实例、任务和审计记录。
5. 生产前执行后端全量测试、前端类型检查与构建、API 契约检查、数据库迁移演练、备份恢复和目标环境 E2E。

具体命令与运维要求见[部署指南](../DEPLOYMENT_OPTIMIZATION.md)、[生产初始化运行手册](../runbooks/production-initialization.md)和[测试指南](../testing/README.md)。运行时是否向当前用户开放某能力，以认证后的 `GET /api/v1/capabilities` 为准。

## 明确不承诺

- Pilot 能力不等于未经验收即可用于生产；开源发行版也不承诺某个云厂商、消息渠道或 LLM 在所有环境开箱即用。
- 页面、路由、Schema、seed 数据或测试文件存在，不单独证明业务闭环已完成。
- AI 默认提供建议与降级路径，不绕过租户、RBAC、审批或人工责任。
- Scaffold、演示数据和未来 Roadmap 不计入当前可交付能力。

## 贡献与后续方向

贡献者应优先沿现有领域边界扩展：业务规则放后端服务，流程使用 BPMN/command，企业集成进入 connector lifecycle，AI 调用进入 LLM Gateway，所有新增表和接口同时补 tenant、ACL、DTO、审计与回归测试。当前优先级见根目录 [ROADMAP](../../ROADMAP.md)。
