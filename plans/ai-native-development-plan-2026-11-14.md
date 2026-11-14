# AI-Native 后续开发计划（2026-11-14）

> Status: draft（设计输入，未进入实施；判断实现状态必须回到源码、迁移、运行时 API 与最新 CI，本文不构成完成证明）
> 上游锚点：根目录 [`ROADMAP.md`](../ROADMAP.md) v1.7「Connector productionization + AI evaluator + business E2E」与 v2.0「AI auto-triage GA」。
> 事实基线：`7804845bc`（feat(ai): 工单调查编排一等公民入口 + AI 域底账去时间化与契约硬化）。
> 所有「缺口」均为 2026-11-14 对 `handlers/ai`、`internal/services`、`itsm-ai-service`、`itsm-agent`、`itsm-frontend` 摸底确认的一手事实，标注了证据位置。

## 0. 现状快照（摸底结论，写计划前先对齐事实）

已有且可用（不要重复建设）：

| 能力 | 位置 | 状态 |
|:--|:--|:--|
| 三级分诊链（Guidance→LLM→keyword） | `internal/services/triage_service.go` | 可用，按工单幂等 |
| RAG 五轮工具循环 | `internal/services/rag_service.go` | 可用 |
| 工具目录 14 个（KB/CMDB/工单读写） | `internal/services/tool_registry.go` | 只读/写工具分级，写走审批 |
| 工具审批流（pending→approve + outbox） | `handlers/ai/repository_impl.go` + `commandbus` | 可用；双人分级审批为**另一会话进行中 WIP**（见 §6） |
| 决策底账 + 采纳/拒绝翻转 | `ai_audit` + `PATCH /ai/audit-logs/:id` | 可用 |
| 内置 Skill 12 个（含 ai.investigate） | `handlers/ai/skills.go` + `builtin_skills.go` | 注册表为进程内存态 |
| 工单调查编排 `POST /ai/tickets/:id/investigate` | `handlers/ai/investigation.go` | 后端可用，**前端零消费方** |
| AI Evaluator CI gate（四指标阈值） | `ai/eval` + backend-ci | CI 降级档已有；LLM 档人工验收未执行（ROADMAP 剩余项） |
| AI 审计日志查询/评估报告 | `GET /ai/audit-logs`、`GET /ai/evaluation` | 可用 |

已确认的缺口（本计划要收的账）：

1. **编排无前端消费方**：`investigation.go` 无 UI 入口，`suggestedActions` 无采纳路径。
2. **Skill Invoke 身份注入薄弱**：`InvestigateSkill.Execute` 从 input map 读 `userId`/`role`，依赖调用方自觉注入；输入自报身份 = 可伪造归因。
3. **Skill 注册表不持久化**：`handlers/skill/registry.go` 头注释明确「未直接持久化自定义 Skill」；重启即失、无租户级 skill 管理。
4. **AI 主体零信任在做但未完成**：另一会话 WIP（Gate 2.5 + `authz.Catalog` + 风险分级双人审批），未提交、未接线完成。
5. **itsm-ai-service 欠账**：snake_case DTO、rca 错误回显、fail-open 令牌（批次 0 摸底记录）。
6. **itsm-agent 欠账**：阻塞审批、`--admin` 权限合并、零测试（同上）。
7. **Evaluator 覆盖不足**：四指标只覆盖 triage/摘要/RAG/ROC 四场景，investigation 编排无质量基线；LLM 真实语料验收未归档。
8. **降级语义不完整**：AI 不可用时部分端点仍返回 `code=0` 正常信封（批次 0 边界记录），前端无法统一识别 degraded。

## 1. 批次规划（按优先级；每批独立可交付、可验证、可回滚）

### 批次 A：调查编排前端闭环（把后端能力变成操作员可用功能）

**目标**：工单详情页一键触发调查、读懂结论、把建议变成动作、把采纳打回底账。

- 改动面：
  - `itsm-frontend/src/lib/api/aiApi.ts`：新增 `investigateTicket(ticketId)`，camelCase 契约对齐 `TicketInvestigationResponse`。
  - 工单详情页新增「AI 调查」区块：`steps[]` 时间线（triage/knowledge/cmdb/synthesis，degraded 步骤红点+原因）、`findings`、`suspectedRootCause`、`impactedServices`、`suggestedActions`。
  - `suggestedActions` 交互：点击「创建关联工单」→ 打开既有建单弹窗并预填标题/描述（**不自动提交**）；`link_ticket_ci` 建议 → 跳 CMDB 关联入口。所有动作仅是导航+预填，执行仍走各自表单与审批。
  - 底账闭环：区块内展示 `auditId`，操作员点「有用/不准」调既有 `PATCH /api/v1/ai/audit-logs/:id`。
  - 状态完备：loading / empty（无工具权限时全降级也如实展示）/ 404 / 403 / degraded 徽标 / 成功。
- 回归锁：组件 jest 测试（camelCase 键断言 + degraded 展示 + 404 态）；`api-contract.test.ts` 自动覆盖新路径；解除 `flows/flow-ai-tool-rbac.spec.ts` 的两处 `test.skip(true)` 或新增调查编排 E2E spec（带环境开关）。
- 验收：`npm run type-check` + `npm run lint:antd` + 目标 jest + E2E 本地跑绿。
- 边界：不做自动重跑轮询；不做调查历史对比视图（留批次 F）。

### 批次 B：Skill Invoke 身份注入硬化 + 执行审计统一（安全批次，小而关键）

**目标**：Skill 路径的身份与审计达到与 HTTP 路径同级的可信度。

- 改动面：
  - `handlers/skill/` Invoke 链：`tenantId`/`userId`/`role` 改为**只**从认证上下文注入（Invoke 侧 `c.Set` 等价物或显式参数），input map 中的同名字段一律覆盖/拒绝；`InvestigateSkill.Execute` 等从 input 读身份的代码全部收口。
  - 每次成功/失败 Invoke 落统一审计（复用 `ai_audit`，scenario=`skill:<code>`），失败也留痕。
  - 与 Gate 2.5 对齐：Skill Execute 统一走 agent 主体上下文（依赖 §6 的 WIP 落地，**排在 B 内最后一小步**，未落地前先做身份注入两小步）。
- 回归锁：Invoke 身份覆盖测试（input 自报 admin 身份必须被覆盖）；审计写入断言；跨租户 skill 调用拒绝测试。
- 边界：不做 skill 权限模型重设计（沿用 RequiredPermissions）。

### 批次 C：Skill 注册表持久化（平台化第一步）

**目标**：skill 元数据进库，重启不丢，租户可管理，为市场做准备。

- 改动面：
  - 新 ent schema `skill_definition`（tenant_id、code 唯一按租户组合、manifest JSON、版本、checksum、maturity、enabled）——遵循 Schema 强制规则（租户字段、索引、迁移可版本化）。
  - `handlers/skill/registry.go`：启动时内置 skill 注册后回写/同步 DB（幂等 seed，按 checksum 判变更）；`/admin/skills` CRUD 走 DB。
  - 冲突预检：唯一约束上线前的存量重复检测方案（Schema 规则强制）。
- 回归锁：注册幂等（重启两次不产生重复行）、租户隔离（A 租户自建 skill B 不可见）、checksum 变更审计。
- 边界：不做 skill 代码上传/执行（只做元数据与生命周期）；执行仍走内置 handler。

### 批次 D：AI Evaluator 扩展——investigation 场景基线 + LLM 档验收（ROADMAP 剩余项）

**目标**：编排链有可测量质量基线；真实模型验收按 ROADMAP 要求人工执行并归档。

- 改动面：
  - `ai/eval` 新增 investigation 场景指标：步骤完整率（4 步齐）、工具 RBAC 拒绝率（应降级非报错）、degraded 率、`suggestedActions` 白名单命中率、LLM JSON 解析成功率。
  - 评测夹具：构造典型工单（磁盘告警/网络抖动/账号锁定）× 依赖 fake，断言链路行为而非文本相似度（编排是结构化输出，ROUGE 不适用——用结构断言）。
  - `cmd/ai-eval -mode llm` 跑真实模型档（需要真实 API key，人工执行），报告归档到 docs/acceptance/ 并登记 ROADMAP（**须人工执行，AI 不代签**）。
- 回归锁：CI 降级档新增场景指标进 backend-ci；结构断言负证明（改坏白名单过滤必红）。
- 边界：不设定 investigation 的阈值门槛（先观察两个版本再定阈值，避免拍脑袋门禁）。

### 批次 E：itsm-ai-service 与 itsm-agent 硬化（AI 面收账）

**目标**：两个边缘组件达到与核心同级的契约与安全标准。

- sidecar：
  - snake_case DTO → camelCase（对外 HTTP 契约统一）；
  - rca 错误回显净化（不回传内部错误原文）；
  - fail-open 令牌校验改为显式 401/degraded（认证边界不得静默放行）；
  - sidecar 是独立服务，契约变更加版并同步 backend 适配层（外部协议在 adapter 边界转换，核心保持 camelCase）。
- agent：
  - 审批从阻塞等待改为轮询+超时（可中断）；
  - `--admin` 拆分为显式子命令；
  - 建立最小测试套件（当前零测试）。
- 回归锁：sidecar 契约测试（pytest）；agent 冒烟测试（审批超时路径）。
- 边界：sidecar 的 Python 依赖/镜像不在本批（已在部署优化批处理）。

### 批次 F：编排深化 + 主动式 AI（v1.7→v2.0 方向，前置依赖 A–D）

- investigation v2：多 CI 候选并行影响分析（当前只取第一个命中）、跨工单相似调查引用（历史 `ai_analysis_results` 复用做参考而非幂等复用——与「新鲜度敏感」语义不冲突，因为只是参考输入）、调查结论一键生成知识库草稿（走知识库写审批）。
- 主动触发：SLA 违规风险预测 / 连接器告警消息 → 触发调查 → 结果回推送（Feishu 卡片）。依赖连接器生产化（v1.7 主线，另行批次）。
- 调查历史时间线视图（前端）：同一工单多次调查的 diff。

## 2. 依赖与顺序

```
A（前端闭环）─→ F（深化/主动）
B（身份硬化）─→ C（持久化）─→ Skill 市场（远期）
     └──────────────→ 依赖 §6 Gate 2.5 落地
D（Evaluator）─→ 独立，可并行
E（边缘硬化）─→ 独立，可并行
```

推荐执行序：A → B → D → C → E → F。A 是用户价值最大且风险最低的一步；B 是安全债，量小优先还；D 直接收 ROADMAP 的 v1.7 剩余项。

## 3. 统一约束（每批验收前自查，摘自 AGENTS.md 强制规则）

- camelCase 契约零新增 snake_case；`git diff` 命名扫描必跑。
- 租户 fail-closed：每个新租户资源接口至少一个跨租户拒绝测试，覆盖真实 Router 入口。
- AI = 决策支持：任何写动作必须经既有审批流，编排/Skill/agent 不得静默执行。
- 每个能力区分 disabled / unconfigured / degraded / ready，不把未配置伪装成空成功。
- 测试覆盖真实生产入口；bug 修复必带「修复前失败」回归测试。
- 提交逐文件显式挑选（工作树长期混有并行会话 WIP，禁用 `git add -A`）。

## 4. 明确不做（本计划范围内）

- 不引入第二套审批引擎（BPMN 优先原则）。
- 不做 skill 代码动态执行/沙箱（市场能力另立评审）。
- 不把 LLM 档 evaluator 结果当 CI 门禁（真实模型有波动，人工验收归档即可）。
- 不代签真实租户跨域人工验收（workflow 签字栏维持空）。

## 5. 与 ROADMAP 的关系

- 批次 A/B/C 推进能力矩阵 `ai` 行从预览走向可用的证据链。
- 批次 D 直接收 ROADMAP「AI Audit/Evaluator」剩余项（LLM 档人工验收）。
- 批次 F 的连接器触发部分依赖 v1.7「Connector productionization」主线，本计划不重复规划连接器本身。

## 6. 并行会话协同（避免撞车）

以下改动在 2026-11-14 工作树中为另一会话的进行中 WIP（未提交），本计划**不包含**它们，落地后按其真实形态对齐：

- Gate 2.5 AI 主体零信任：`internal/authz/policy.go`、`registry.go`（untracked）、`handlers/ai/service.go` 的 `authz.Catalog`/`SetPermissionCatalog`/`PrincipalAgent`、`bootstrap/app.go` 的 `ConfigureShared`。
- 双人分级审批状态机：`handlers/ai/repository_impl.go` 的 `pending_agent`/`pending_agent_two`/`awaiting_second`。

批次 B 的最后一步（Skill 走 agent 主体）与批次 F 的审批交互设计，须等该 WIP 提交并评审后再动。
