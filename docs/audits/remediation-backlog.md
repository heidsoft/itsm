# 整改 Backlog

> 基准 commit：`b3bfd748604041d5082a320ed6d5c447221ac76a`
> 状态：confirmed / suspected / resolved / blocked（见各专项报告）
> 原则：一个 PR 一个明确问题域；数据库/权限/部署变更需人工审批。

## P0

| ID | 问题 | 状态 | 处置 |
|:---|:---|:---|:---|
| AUDIT-001 | 空卷首装被 `mcp:read` 迁移打断 | `resolved`（PR #114 已合并） | 继续跟踪 AUDIT-002 的首装门禁 |
| AUDIT-002 | `configuration_items` 重复索引阻断 fresh-install 门禁 | `resolved`（本批已验证） | 已在 `duplicateIndexPairs` 补两对，保留 Ent 索引；真实 PG 回归覆盖删除/幂等/缺失保留方 |
| AUDIT-003 | Release 静默降级为 git-log 说明、无测试证据/制品校验 | `resolved`（G1/G3/G4 已验证） | 发布说明按版本提取、缺失即失败；manifest provenance 统一到 tag；新增文案门禁 |

## P1

| ID | 问题 | 状态 | 处置 | 审批要求 |
|:---|:---|:---|:---|:---|
| AUDIT-010 | `/tickets/types` 菜单/路由权限码不一致 | `confirmed` | PR-02：改路由为 `ticket_type:read`，重生成 precheck，补 HTTP 负例 | 权限变更，需审批 |
| AUDIT-011 | 服务请求两菜单用 `ticket:read` | `confirmed` | PR-02：基线码改 `service_request:read` | 低风险（仅显示） |
| AUDIT-012 | `/workflow/audit` 用 `audit:read` | `confirmed` | PR-02：基线码改 `bpmn:read` | 低风险（仅显示） |
| AUDIT-013 | 存量租户 AI 归位/重命名依赖手动端点 | `confirmed` | PR-02：迁移补 `name/parent_id` 修正 + 双租户测试 | 数据迁移，需审批 |
| AUDIT-070 | AI/MCP `update_ticket` 工具写路径跳过行级范围 | `resolved`（Run `cm-auto-20261125-01`） | 可信 actor 注入 + 租户/fail-closed 校验，见 PR #131 | 低风险（代码层，不改权限策略）；Run cm-auto-20261125-01: PR #131：可信 actor 注入 + fail-closed，红/绿测试与 CI 包级通过（red exit 1 -> green exit 0; backend-ci internal/services ok 41.502s） |
| AUDIT-020 | 权限停用不作用于人工 RBAC | `resolved`（本批已验证） | 人工路径解码策略信封跳过停用码；策略写入同步清缓存；真实 DB 回归 + 反向验证 | 已完成 |
| AUDIT-021 | DBOnly L3 回退到编译期默认值 | `resolved`（本批已验证） | L3 改走 AuthorizeResource 单一真源；真实 DB 回归含反向验证 | 已完成 |
| AUDIT-022 | 策略/ACL 缓存无跨实例广播 | `resolved`（本地与发布路径已验证；Redis 端到端待环境） | 独立频道广播 + 本地失效；解析与本地用例绿，Redis e2e 跳过 | 已完成实现 |
| AUDIT-023 | 权限/角色缺 DB 唯一约束 | `confirmed` | 版本化迁移 + 存量重复检测 | 数据库结构，需审批 |
| AUDIT-024 | seeder 回退清单资源名漂移 | `confirmed` | 删除回退清单或从 catalog 派生 | 低风险 |
| AUDIT-025 | 重复初始化覆盖托管菜单字段 | `confirmed` | 明确所有权清单或纳入保护 | 低风险 |
| AUDIT-051 | 备份调度未被部署验收强制 | `confirmed` | 文档 + 部署后断言 | 低风险 |
| AUDIT-053 | 重复索引（同 AUDIT-002） | `confirmed` | 见 P0 | 需审批 |
| AUDIT-060 | `lint:antd` / 方向守卫 / P0 文案守卫红 | `resolved`（本批已验证，三项守卫 exit 0） | 已修 9 处旧 API 调用 + 文案守卫识别 `title`；浏览器视觉回归仍待执行 | 低风险 |
| AUDIT-040/041 | Sidebar 双模块暗色失效 + 禁用态 Select 同色 | `resolved`（浏览器实测） | 统一 CSS Module 引用；禁用态高特异性规则 + 3 例回归 | 已完成 |
| AUDIT-062 | 禁用态 Select 选项同色 | `confirmed`+`suspected` | PR-05：层叠修复 + `select-option-states.test.ts` | 需浏览器证据 |
| AUDIT-063~065 | 平板宽度、移动遮罩 z-index、尺寸真源三方矛盾 | `confirmed` | PR-05：统一 token/选择器 | 需浏览器证据 |
| AUDIT-066 | LoadingSkeleton 渲染期随机高度 | `confirmed` | PR-05：确定性高度 | 低风险 |
| AUDIT-030~032 | 发布 provenance / 版本漂移 | `confirmed` | PR-04 | 低风险（CI 配置与脚本） |
| AUDIT-050~054 | 官网信任页、无来源数字、站点源码、法务入口、品牌 | `confirmed` | 官网专项（仓库外发布流程） | 需要产品/市场确认 |

## P2

| ID | 问题 | 状态 | 处置 |
|:---|:---|:---|:---|
| AUDIT-050(deploy) | `.env.prod.example` 关闭 metrics 鉴权 | `confirmed` → `resolved` | PR-01 已修（改示例 + 静态守卫） |
| AUDIT-052(deploy) | 基础设施镜像浮动标签 | `confirmed` | 固定次要版本/digest |
| AUDIT-014(menu) | `/ai/chat/list` 别名可复活退役菜单 | `confirmed` | 扩展退役迁移 WHERE |
| AUDIT-015 | 菜单文档 102→101 漂移 | `confirmed` | 重新生成表格 + docs-gate 禁止退役字符串 |
| AUDIT-016/017 | 菜单 fixture 假绿、缺跨层守卫 | `confirmed` | 改为读基线；新增路径/码守卫 |
| AUDIT-046~049 | token、z-index、表格/弹窗 overflow、i18n | `confirmed`/`suspected` | PR-05 分批 |
| AUDIT-055 | 博客数字标题待核实 | `suspected` | 核实来源后改写 |
| F-01(rbac) | `PermissionDefinition` 死模型 | `confirmed` | 标记退役或版本化删表，不接入 RBAC |

## 已解决/不要重复

PR #114（首装 P0001）、跨租户负例矩阵、RBAC 缓存中毒与角色移除即时失效、RLS readiness 门禁、权限策略信封与只增不减授权、菜单退役与 `bpmn:*` 对齐、AntD v6 主体迁移。

## 本批已实施

- AUDIT-050(deploy)：`.env.prod.example` 默认开启 `/metrics` 鉴权，新增静态守卫测试，形成 `audit/production-tenant-readiness` 分支的 PR-01 低风险子集。

## 2026-11-28 质量治理增量

基准 `021f08c0e`。源码证据与运行验收分别记录。每轮独立交付一个问题域。

| 编号 / 优先级 | 模块、根因与证据 | 影响 | 修复与验收 | 测试 / 剩余风险 |
| --- | --- | --- | --- | --- |
| PQ-01 / P0 | `ticket_routes.go` → `TicketWorkflowHandler.ApproveTicket` → `TicketWorkflowService.ApproveTicket`。事务分支选择前校验审批记录归属和 pending，缺少与工单申请人直接比较。状态聚合也单独判断待审人。独立 `/workflow/reject` 只检查租户、状态和版本，业务服务未核验待审人或流程任务归属 | 历史或迁移数据产生自指派时，分支行为可能不一致 | 在领域层共享职责分离判定，覆盖动作投影及写入。验证申请人被拒且决策、历史和状态无变化，独立审批人成功，另验独立驳回、BPMN 任务和通用审批入口 | 已确认共享前置条件缺失；是否可从正常配置到达、下游已拦截哪些路径及历史数据量待验 |
| PQ-02 / P0 | `SetupTicketRoutes` 显示同名解决、关闭业务操作有两组路由与权限 | 兼容入口可能出现角色越权或行为分叉 | 枚举旧调用方，统一领域守卫。相同 actor、tenant、版本和状态下验证两个入口的成功与拒绝，并测试竞争只生效一次 | 源码确认权限不同；写入结果待验 |
| PQ-03 / P1 | `getStatusConfig` 将未识别键回退为 open，常量单测明确固定该行为 | 展示可能把存量非法值解释为待处理 | 先按数据库值、API DTO、流程投影和页面调用点追踪来源。保留合法 API 编码，统一中文异常展示；异常状态不可操作 | 已确认展示行为；`normalizeBpmnStatus` 也会把监控认可的 failed 状态回退为 not_started。没有非法存量数据证据 |
| PQ-04 / P1 | `internal/bpmntemplates/bpmn` 有五组带 `_cn` 文件；`ProcessDefinition`、`ProcessBinding`、实例分别持久化 | 误合并会影响启动匹配和历史引用 | 比对 BPMN 图、规则、版本、绑定及实例引用，再治理展示名称和推荐模板 | 已确认模板共存；发布记录与等价性待验 |
| PQ-05 / P1 | 类型、模板、类别、目录分别存储。工单服务存在 formFields 持久化更新路径；全局搜索中的同字段契约未证实 | 配置到检索、审计可能断链，升级可能影响租户配置 | 五类预设分别建立配置版本、字段往返、权限、搜索脱敏、幂等安装和租户覆盖保护验收 | 现有模板和绑定测试可复用；真实安装与检索待验 |
| PQ-06 / P1 | `GetWorkflowRules` 对非空类型仍返回同一租户全部活跃规则；分配页对条件和目标使用 JSON 文本框，自动化页的动作配置也使用 JSON 文本框并调用 JSON.parse | 管理者可能误读生效范围 | 先确定类型筛选契约及消费方，再提取共享 RuleBuilder。模拟无写入，执行有权限、历史和循环保护 | 源码发现；执行行为待验 |
| PQ-07 / P1 | AI 创建页 triage 测试和 Agent 审批实现分属不同链路 | 草稿、真实提交、采纳审计可能被混淆 | 测试生成、补齐与重试期间工单数不变，确认后只创建一张；覆盖无模型回退和受控工具执行 | 源码审查；真实 provider 条件待验 |
| PQ-08 / P2 | UI 问题见 [UI 静态审计](ui-ux-audit.md)；相关统计接口含固定值，通知方法有日志占位 | 页面可用性和信息可信度 | 共享布局与组件逐步治理，统计按权限和真实数据计算，通知以可观测 delivery 验收 | 运行证据待采集 |

执行顺序为 PQ-01 → PQ-02 → PQ-03/PQ-04 → PQ-05/PQ-06/PQ-07 → PQ-08。完成一轮后按 [E2E 矩阵](../acceptance/e2e-acceptance-matrix.md) 记录独立验收证据与剩余风险。

### PQ-01 实施与生产路由验证

- 生产入口为 `SetupTicketRoutes → TicketWorkflowHandler → TicketWorkflowService`。服务端在审批分支前校验申请人身份，`state-v2` 对申请人隐藏审批动作。独立驳回在写事务内复用 BPMN 任务归属检查，未绑定流程时读取当前级别的待审人。
- `router/ticket_approval_ownership_route_test.go` 通过真实 Gin 注册、JWT 鉴权和隔离 SQLite 数据库验证自审及自驳回、独立审批、无待审资格、已被他人认领的任务和跨租户拒绝。拒绝路径比对工单和审批记录快照，检查无流转记录，并确认 BPMN 拒绝路径保留原任务及无审批决定。
- 定向路由、处理器和服务测试均通过。专项目标是工单流转入口的权限边界。独立驳回成功后的 BPMN 同步、通用审批与 BPMN 任务入口的统一职责分离、已登录浏览器验收另列后续验证。
