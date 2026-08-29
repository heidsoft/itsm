# 领域所有权与迁移边界

| 领域 | 当前生产入口 | 目标所有者 | 事务所有者 | 当前策略 |
|---|---|---|---|---|
| Ticket | legacy controller/service | Ticket application service | Ticket service | 首版保留，不并入通用 Ticket 抽象 |
| Incident | 全部在 legacy（`/incidents` 30+ 端点） | `handlers/incident` | Incident application service | **延后至 v1.7**（2026-08-29 决策）：新层已实现约六成但零路由，生产化前不做切换；`handlers/incident` 实现保留，待补齐 9 个生命周期端点后一次切路由 |
| Change | legacy + `handlers/change` | `handlers/change` | Change application service | BPMN 启动只写 command |
| Problem / Known Error | `handlers/problem`、`handlers/known_error` | 对应领域切片 | 对应 application service | `/problems` 已在新层；调查子域 `/problem-investigation` 为独立互补功能，暂留旧层 |
| Service Request | legacy + `handlers/service_request` | `handlers/service_request` | Request application service | 审批、通知和 provisioning 使用 command |
| CMDB | `handlers/cmdb`（主体 21 端点） | `handlers/cmdb` | CMDB service / Job | 已切新层；仅剩弃用别名 `/configuration-items` 与 `/incidents/configuration-items`，禁止新增端点，待移除 |
| SLA | `handlers/sla`（`/sla` 全套） | `handlers/sla` | SLA service | 已切新层；仅模板留 `SLATemplateController`；仅 Worker 执行计时与升级 |
| Knowledge | `handlers/knowledge` | `handlers/knowledge` | Knowledge application service | **迁移完成**（2026-08-29）：零接线的旧控制器已删除 |
| Workflow | BPMN services | Workflow application service | 发起领域事务 + command | BPMN 是唯一编排层 |
| Notification | notification services | Notification delivery handler | 生产领域事务 + command | 所有渠道统一 `notification.deliver` |
| Connector | connector framework | Connector application service | Connector service | 飞书先行，Marketplace 暂只读 Pilot |

## 迁移完成定义

一个领域只有在以下条件同时满足后才能删除 legacy 装配：稳定 HTTP 合同保持兼容；全部端点已切换；状态机和租户规则在服务层；事务内产生审计与 command；契约、越权、并发和故障恢复测试通过。迁移过程中禁止同一路由随机落到两套实现。

---

## 跨领域不变量（v1.5 起强制）

> **迁移来源**：
> - [`docs/review/module-function-retrospective-2026-07-10.md`](../review/module-function-retrospective-2026-07-10.md) §1（主流程闭环 / 契约治理 / AI Native 可度量）
> - [`docs/review/system-function-review-result-2026-07-01.md`](../review/system-function-review-result-2026-07-01.md) §3（前端测试退出异常、Jest open handles）
>
> 与 [`workflow-cmdb-invariants.md`](./workflow-cmdb-invariants.md) 配套使用。

### 主流程闭环（每个领域至少一条 E2E）

每个核心领域（Ticket / Incident / Problem / Change / Service Request / Approval / CMDB / SLA / Knowledge）必须具备：

1. 一条端到端角色路径，**可自动回归**；
2. 至少 1 个 Playwright/Jest 角色级用例覆盖；
3. CI 中必须看到该路径的最近一次 green run。

仅"页面存在但业务动作不完整"不构成闭环，禁止作为 v1.1+ 验收依据。

### 契约治理单一事实源

| 层 | 来源 | 备注 |
|---|---|---|
| 后端路由 | `itsm-backend/router/` + 各 controller 的 `RegisterRoutes` | 由 `generate-acl-manifest.js` 重新生成 |
| 后端 DTO | `itsm-backend/dto/` | DTO 字段必须 camelCase |
| 前端 API Client | `itsm-frontend/src/lib/api/*Api.ts` | 类型与 DTO 1:1 对齐 |
| 前端类型 | `itsm-frontend/src/types/` | 不允许任何字段重命名 |

任何字段命名修复必须同时落库到这 4 处；不允许前端绕过 DTO 直接渲染 Ent 模型。

### 状态机与审批

- ITIL 生命周期迁移由后端 service 强制校验，**前端不得成为唯一状态流转入口**；
- BPMN 表达式 / 网关不变量见 [`workflow-cmdb-invariants.md`](./workflow-cmdb-invariants.md) §1；
- CMDB 跨租户与影响分析不变量见 [`workflow-cmdb-invariants.md`](./workflow-cmdb-invariants.md) §2。

### AI / 通知 / Connector 默认门禁

| 能力 | 必须达到的最低条件 |
|---|---|
| AI 能力 | 评估集、prompt/template 版本、模型/provider、置信度、accepted/rejected、operator 反馈全程留痕；LLM 调用必须经 LLM Gateway |
| 通知 | 多渠道统一走 `notification.deliver` command；事务内入箱 + 异步出箱 |
| Connector | 生命周期经过 installed → configured → enabled → healthy；密钥永不返回前端 |
| Manifest | `name/version/requiredPermissions` + `sha256:` checksum；缺一即拒绝注册（fail-closed），由 `connector/manifest_gate_test.go` 覆盖 |

### 测试夹具唯一性

任何 controller/service 测试夹具：

1. 不得硬编码共享唯一键（如 `ticket_categories.code = "incident"`）；必须含 `uniqueTestID()`；
2. 涉及 `tenant_id` / `user_id` 必须用 helper 创建真实对象，禁止 `tenant_id=0` 占位；
3. 跨测试共享的种子数据必须在 `TestMain` 或 setup helper 内统一清理（参见 [`testing/test-invariants.md`](../testing/test-invariants.md)）。

### 文档-代码同步

- 任何领域状态机变更必须同步更新本文件与对应 DTO 注释；
- 评审报告中识别的"已完成"必须能在当前 main 分支源码中复现，否则视为已回归并重新打开。
