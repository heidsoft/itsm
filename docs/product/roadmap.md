# 四周研发执行计划（2026-11-23 起）

> Status: current · 基线日期：2026-11-23 · 上级事实源：[根目录 ROADMAP.md](../../ROADMAP.md)
>
> **定位声明**：`docs/product/README.md:16` 规定本目录不维护并行的总控计划，根 `ROADMAP.md` 是路线与交付节奏的唯一事实源。本文**不是**第二份产品路线图，而是受其约束的**四周工程执行计划**：只描述近期三个研发切片及其验收方式，不重新定义版本目标、不覆盖根路线图的成熟度口径。若与根路线图冲突，以根路线图为准并需先解决冲突。
>
> 输入：[current-state.md](./current-state.md) · [gap-analysis.md](./gap-analysis.md) · [technical-debt.md](./technical-debt.md)
>
> 本计划在本轮只完成“分析与计划”，**未执行任何切片开发**。

---

## 1. 排序逻辑

```
Slice A（W1）可信交付基座  ──▶ Slice B（W2–W3）行业闭环  ──▶ Slice C（W4）商业化封装
       解除“无法验证”                 产出“可演示价值”            转为“可报价承诺”
```

- 先做 A：在 CI/版本口径失真状态下，任何后续切片的“验证通过”都不可信。
- 再做 B：产生唯一可现场演示、可减少人工操作的业务闭环（REQ-003）。
- 最后 C：把 A 与 B 的结果固化为边界与报价依据（REQ-001），不凭空讨论定价。

**本周只选一个切片**。三个切片按依赖顺序推进，不并行混改同一模块。

---

## 2. Slice A（W1）：单一版本事实源与必需质量门禁

| 项 | 内容 |
|:---|:---|
| 对应差距 / 债务 | GAP-02、GAP-04 / TD-01、TD-02、TD-03、TD-13（仅 `useTickets` 静态 message 若顺路） |
| 需求记录 | REQ-002 |
| 用户价值 | 交付工程师与客户能确认“部署的是哪个版本、经过哪些门禁”，减少验收争议 |
| 代码范围 | `CHANGELOG.md`、`ROADMAP.md`、`UPGRADE.md`、`docs/documentation-governance.md`、`scripts/version-drift-baseline.json`、`README.md`（Go 版本）、`.github/workflows/docs.yml`、新增 1 个版本一致性检查（建议放 `scripts/docs-gate/` 或 `tests/parity/`）；**不碰生产 `.env.prod`** |
| 明确不做 | 批量 gofumpt 90 文件；生产配置改动；打 tag；部署 |

### 验收标准

1. 六个版本表面指向同一版本；`git tag` 最大值 = `CHANGELOG.md` 最新发布 = `/version` 运行时返回值（运行时验证在具备环境时进行，本轮只静态对齐）。
2. README Go 版本与 `itsm-backend/go.mod:3` 一致。
3. 文档门禁“6 gates”文案改为 8 gates；ROADMAP RAG 阈值改为 ≥0.8。
4. 版本一致性检查能在版本不一致时**变红**；附反向验证证据（改回旧值 → 红）。
5. `output/github-ci-review-2026-11-14.md` 的 39 个失败被分类为“存量基线 / 已修 / 待修”三栏，Trivy HIGH 有修复或书面风险接受。
6. docs-gate 进入必需检查；若暂不进入，必须在分支保护策略中记录明确豁免与移除条件（不得默认放行）。
7. `make docs-gate` 运行并如实记录结果；任一失败按“基线已有 / 新引入”分类。

### 验证命令（实际执行后填入结果，禁止预填“通过”）

```bash
git tag --list --sort=-v:refname | head -5
grep -n '^## \[' CHANGELOG.md | head -5
cd itsm-backend && go build ./...
cd ../itsm-frontend && npm run type-check
make docs-gate
git diff --check
```

### 回滚

文档与 CI 配置改动可逐文件 `git checkout -- <file>`；版本一致性检查若引入误报，先降级为 advisory 并登记，不得删除检查后宣称完成。

---

## 3. Slice B（W2–W3）：飞书入站消息→工单闭环（首个行业 PoC 入口）

| 项 | 内容 |
|:---|:---|
| 对应差距 / 债务 | GAP-03 / TD-09 |
| 需求记录 | REQ-003 |
| 用户价值 | 群内故障消息不再手工录入工单；重复消息不产生重复单；全过程可审计 |
| 代码范围 | `itsm-backend/handlers/feishu/`、`connector/builtin/feishu/`、`internal/commandbus/`（复用，不新增总线）、`ent/schema/`（仅必要时精准增量生成）、`router/feishu_routes*`、`itsm-frontend/`（如需一个入站消息映射配置页，复用现有 API client） |
| 明确不做 | 企微/钉钉入站派发；通用多 IM 抽象重构；自动执行任何生产动作；覆盖用户 `website/index.html` WIP |

### 必须满足的架构约束（来自 `AGENTS.md`）

- 身份与租户来自验签后的消息上下文与 connector 绑定，**不信任消息体自报 tenant**。
- 验签 + AES 解密 + **持久去重**；不得依赖单进程 in-memory 去重作为唯一机制。
- 建单走事务型 durable command（`EnqueueTx` / `EnqueueTxIdempotent`），不在 HTTP handler 里直接建单后异步 fire-and-forget。
- 一个 recipient + channel 对应一条可追踪 delivery；失败进死信并可人工重放。
- AI 分诊若启用，只产出建议（分类/置信度/依据/推荐负责人），必须人工确认，不自动改变工单状态。
- 日志不含消息正文、token、secret；原始 provider 错误净化后再记录。
- 新增或修改路由时同步 Router、权限注册、前端 API client、mock、契约测试与文档。

### 验收标准

1. 真实 HTTP 入口 E2E（不是孤立 helper）：一条合法飞书事件 → 在正确租户创建工单；断言 HTTP status、业务 code、camelCase DTO、无敏感字段。
2. 重复事件（相同去重键）不重复建单；幂等键含 tenant + command type + aggregate + 事件唯一标识。
3. 跨租户 / 无权限 / 未配置 connector 三类失败分别映射 403 / 409 / 503（或现有错误语义），不得全返回 500。
4. enqueue 失败回滚业务写入；业务失败不留 command；重复请求不产生重复副作用。
5. 审计记录包含租户、操作人/来源、aggregate、command id、稳定错误类。
6. 前端契约测试 `api-contract.test.ts` 与 `npm run type-check` 通过（若新增前端调用）。
7. **人工签字**：生产飞书租户的真实消息验收需由人执行并签字；本轮不得由脚本或 AI 代签，未签字则维持 Preview 表述。

### 验证命令（实际执行后填入结果）

```bash
cd itsm-backend && go test ./handlers/feishu/... ./connector/... ./internal/commandbus/...
cd itsm-backend && go test ./tests/scenarios/...   # 若新增场景用例
cd ../itsm-frontend && npm run test:unit -- --runTestsByPath src/lib/__tests__/api-contract.test.ts
cd ../itsm-frontend && npm run type-check
git diff --check
```

### 回滚

新增命令类型通过注册开关/迁移 `_down.sql` 回退；保留 `operational_commands` 数据不删（审计需要）；前端按文件回退。任何生产 connector 配置变更走既有人工审批流程。

---

## 4. Slice C（W4）：商业边界与可承诺交付矩阵

| 项 | 内容 |
|:---|:---|
| 对应差距 / 债务 | GAP-01 / TD-01 副本 |
| 需求记录 | REQ-001 |
| 用户价值 | 销售与交付能用同一份边界回答“开源包含什么、企业版为什么收费、PoC 范围是什么” |
| 代码范围 | 主要为决策与文档：`docs/product/itsm-commercial-capability-contract.md`（关闭 7 个开放问题）、`README.md` 成熟度表、`website/`（**需用户明确授权后才改**）、可选 `src/config/product-capabilities.ts` 与后端 capability 声明对齐 |
| 明确不做 | 虚构客户名、LOI、价格或付费承诺；任何对外发布；修改用户 WIP 文件 |

### 验收标准

1. 七个开放问题逐条有决策记录，含“谁决定、依据、影响模块、后续动作”。
2. 能力矩阵逐项标 CE / EE / Industry，并说明交付成本与复用理由；边界与代码中的 capability flag 一致，无“文档说收费、代码默认开”的错配。
3. 定价文档从“建议测试价”升级为**明确的决策状态**（已决策则写数并注明依据；未决策则写“待商业决策，禁止对外报价”），不得保留模糊建议价。
4. 基于 Slice B 产出一份可演示脚本：入站消息 → 工单 → 分诊建议 → 人工确认 → 审计；明确其中哪些环节属 EE。
5. 所有材料标注证据等级；不得出现 `Verified Demand`（本轮为 0）。
6. 对外发布（官网、报价单、文章）**需人工审批**，本轮只产出待审材料。

### 验证

文档一致性检查（`make docs-gate`）、能力矩阵与 capability flag 对照表、对外材料人工审查签字栏留空。

---

## 5. 四周排期

| 周 | 切片 | 主要产出 | 完成判定 |
|:---|:---|:---|:---|
| W1 | A | 版本统一 + 门禁强化 + 红灯分类 | §2 验收 1–7 全满足，`git diff --check` 干净 |
| W2 | B（前半） | 入站验签/解密/持久去重 + 建单命令 + 租户与权限门 | 幂等与跨租户单测通过 |
| W3 | B（后半） | E2E + 死信重放 + 审计 + 前端契约（如有） | §3 验收 1–7 全满足，人工签字栏按实留空 |
| W4 | C | 边界决策 + 能力矩阵 + PoC 演示脚本 | §4 验收 1–6 全满足，对外材料待人工审批 |

---

## 6. 明确不在本四周计划内

| 项 | 触发条件 |
|:---|:---|
| 通用 Agent Runtime / Memory / 声明式 ActionFlow（TD-08） | 出现 Verified Demand 或客户明确要求自定义编排 |
| ServiceNow 级 CMDB 类模型 / IRE（TD-11） | 投标要求或付费客户确认 |
| 字段级三方合并、6 个基线 schema（TD-04） | 合同承诺“跨版本无损升级” |
| 全量 RLS（TD-05） | 合规要求数据库层强制隔离，且通过人工审批 |
| K8s HA / Helm（TD-07） | 合同承诺私有化 HA 与商业 SLA |
| 企微/钉钉入站派发 | 飞书闭环完成且有第二个客户渠道需求 |
| 全量 gofumpt / Ent 全量重生成 | 并行会话 WIP 清空 + 专用批次 |

---

## 7. 自动化权限边界（本计划适用）

| 动作 | 本轮能否自主执行 |
|:---|:---|
| 代码分析、文档编写、测试编写、本地验证、创建分支与 PR | 允许 |
| 生产数据库迁移、生产后端部署、破坏性数据操作 | **必须人工审批** |
| IAM / 权限体系重大调整、Secret 与凭证变更 | **必须人工审批** |
| 对外发布商业承诺、定价、客户数据操作 | **必须人工审批** |
| 打 tag / 修改生产 `.env.prod` / 触碰 `website/index.html` WIP | 需用户明确授权 |

> 分析轮实际执行：仅产出四份分析文档；未修改代码、未运行测试套件、未创建分支/PR、未部署、未触碰用户 WIP。2026-11-24 用户选择 Slice A 后按下方执行记录实施。

---

## 8. Slice A 执行记录（2026-11-24）

用户以“1”选择 Next Actions 1，已执行 Slice A。会话账本：`20261124-version-truth-a`（并行会话活跃，进入时 HEAD `712fac9d5`）。

### 实际变更

- 对齐版本表面到 `v1.6.60`：`.env.prod.example`、`router.defaultAppVersion`、`handlers/mcp/adapters.go`、`ga_readiness.go`、`main.go` swagger `@version`、`itsm-frontend/package.json` + lock、`docs/documentation-governance.md`、`UPGRADE.md`、`ROADMAP.md`、`deploy/k8s/*`。
- `make swagger-gen` 重生成 `docs/{docs.go,swagger.json,swagger.yaml}`，连续两次生成 SHA-256 一致。
- README / README.en Go 徽章改为 1.26.10，与 `go.mod` 一致。
- `scripts/version-drift-baseline.json` 登记历史漂移已解决；`scripts/check-version-consistency.js` 扩展到 10 个版本表面 + Go 徽章校验；`scripts/__tests__/version-consistency.test.js` 扩为 5 个测试（含 10 个表面逐一反向注入）。
- `.github/workflows/docs.yml` 新增 `Version Truth` 作业并扩展触发路径；`docs-gate.yml` 的“6 gates”文案改为“8 gates”。
- `docs/product/technical-debt.md` 增加 CI 失败分类表；`docs/product/current-state.md` 增加 §10 跟进。

### 实际验证结果

| 命令 | 结果 |
|:---|:---|
| `node scripts/check-version-consistency.js --canonical=v1.6.60 --strict` | exit 0，10 表面 + 2 徽章全对齐，已登记漂移 0 |
| `node scripts/__tests__/version-consistency.test.js` | 5 tests / 0 fail |
| `cd itsm-backend && go build ./...` | exit 0 |
| `cd itsm-backend && go test ./router/... ./handlers/mcp/...` | 均 ok（router 27s、mcp 12s） |
| Swagger 重生成前后 SHA-256 | 相同（`39e9b678…`） |
| `git diff --check` | exit 0 |
| `bash scripts/docs-gate/run-all.sh --strict` | exit 1：C.6（359>345、2289>2275）+ C.8（`bpmn_timer_runtime.go`）——**本批前后均同样失败，属并行会话 WIP，未调基线** |
| `make docs-gate` | make Error 1（脚本 exit 2），同样 2 failed gates，非本批引入 |
| `cd itsm-frontend && npm run type-check` | 失败：`customers/page.tsx` 缺 `historicalNames`——**并行会话 WIP，本批未碰该文件** |

### 验收标准逐条状态

1. 版本表面统一：**达成**（11 项 committed surface；本地 `.env.prod` 与 build 日志属未入库产物，需人工确认）。
2. README Go 版本对齐：**达成**。
3. “6 gates → 8 gates”、RAG ≥0.8：**达成**。
4. 版本检查可变红并附反向证据：**达成**（10 个表面逐一注入 9.9.9 均 FAIL；Go 徽章错配亦 FAIL）。
5. 39 个 CI 失败分类：**部分达成**（仅仓库有逐项证据的 9 类可分类，其余 30 项无本地明细，未编造）。
6. docs-gate 成为必需检查：**未达成**（需 GitHub 仓库设置人工加入；本批只新增 `Version Truth` 作业并如实记录）。
7. `make docs-gate` 运行并记录：**达成**（通过 `run-all.sh --strict` 执行，结果如上，未声称全绿）。

### 未执行 / 需人工

- 未提交、未推送、未创建 PR、未部署、未打 tag。
- `CHANGELOG.md` 被并行会话暂存，本批未写 `[Unreleased]` 条目，待其提交后补记。
- 生产 `.env.prod`、分支保护设置、Trivy CVE 修复均需人工审批或操作。
- `website/index.html` 用户 WIP 全程只读未触碰。

---

## 9. Slice B 执行记录（2026-11-24）

用户以“B”选择 Slice B。会话账本 `20261124-feishu-inbound-ticket-b`。

### 实际变更

- `handlers/feishu/inbound_ticket.go`（新增）：`CreateTicketFromInboundMessage` —— 租户由已验证 connector instance 传入；`feishu_inbound.mode` 默认 `observe_only`，显式 `auto_create` 才建单；发送者 `open_id` 映射本租户活跃用户，否则回落同租户自动化账号；消息级持久去重（provider `message_id`）+ 租户内唯一工单号；建单/审计/`notification.deliver` durable command 同事务提交。
- `handlers/feishu/handler.go`：`im.message.receive_v1` 从“ignored”改为派发到新服务；错误稳定映射 403/409/503/400/500，失败释放 nonce 去重行。
- `dto/feishu_sync_dto.go`：`FeishuWebhookResponse` 增补 `ticketId` / `ticketNumber`（camelCase，可选）。
- 测试：`handlers/feishu/inbound_message_test.go`（8 例：成功、重复、观察态、未授权、未配置、自动化账号、跨租户、去重状态）；`router/feishu_inbound_ticket_route_test.go`（真实路由 E2E：建单 + nonce 重放 403 + 消息重复 409 + 未知实例 400 + 无敏感字段）。
- 文档：`docs/api-reference.md` 新增“飞书消息 → 工单（受控开关）”；`CHANGELOG.md` [Unreleased] 补记 Slice A/B。

### 实际验证结果

| 命令 | 结果 |
|:---|:---|
| `go test ./handlers/feishu/... -run TestInboundTicket -count=1` | ok（8 例） |
| `go test ./router/... -run TestFeishuInboundMessageCreatesTicketThroughRealRoute -count=1` | ok |
| `go test ./handlers/feishu/... ./router/... ./dto/... -count=1` | feishu / router ok；**dto 既有失败** `TestBPMNProcessDefinitionList_DropsXMLButDetailKeepsIt`（本批未碰该区域） |
| `go build ./...` | exit 0 |
| 版本守卫 `--strict` | exit 0（Slice A 成果未受影响） |

### 验收标准逐条状态

1. 真实 HTTP 入口 E2E 建单：**达成**（断言 status/code/camelCase 字段/无敏感字段）。
2. 重复事件不重复建单 + 幂等键含 tenant/command/aggregate/事件身份：**达成**（双层去重；命令键含 tenant+messageId+ticketId）。
3. 403 / 409 / 503 差异映射：服务层**达成**；HTTP 层已覆盖 403(nonce 重放)/409(消息重复)/400(未知实例)，未授权与未配置的 HTTP 分支由同一 `errors.Is` 映射锁死但未单独写路由断言——列为待补。
4. enqueue 失败回滚业务写入 / 业务失败不留 command：机制由既有 `ticket_notification_tx_test.go:141-181` 与 `commandbus_test.go:197-235` 证明；本批未新增专门注入 enqueue 失败的红绿用例——列为待补。
5. 审计含租户/来源/aggregate/命令标识、不含正文：**达成**。
6. 前端契约与 type-check：本批未新增前端调用，未执行前端测试；仓库 `npm run type-check` 仍有并行会话 WIP 造成的 `customers/page.tsx` 既有失败。
7. 生产租户人工签字：**留空**，维持 Preview，不代签、不宣称 GA。

### 回滚

新增文件可直接删除；`handler.go` 分支与 DTO 增补字段可按文件回退；无需数据库 schema/迁移变更，去重与命令复用现有表。若需关闭能力，把租户 `SystemConfig.feishu_inbound.mode` 设回 `observe_only` 即可停止建单，无需发版。

---

## 10. Slice C 执行记录（2026-11-25）

用户以“C”选择 Slice C。会话账本 `20261124-commercial-boundary-c`。

### 实际变更

- `docs/product/itsm-commercial-capability-contract.md`：7 个开放问题改为决策记录（状态全部 `待决策`，含建议选项、证据等级、决策人角色、所需输入、影响模块、下一步）；新增 CE/EE/Industry 建议矩阵，逐行标注代码现状与“无 edition gate”事实。
- `docs/business/PRICING_AND_PACKAGING.md`：状态改为“待商业决策 — 禁止对外报价”，历史数字移入作废附录。
- `docs/product/poc-demo-script-feishu-inbound.md`（新增）：真实演示链路、前置条件、证据清单、版本归属建议、人工签字栏。
- `docs/product/README.md` 增补索引；`README.en.md` 修两处与中文版的事实漂移（云发现水位线、管理者统一视图）；`CHANGELOG.md` [Unreleased] 补记。

### 验收标准逐条状态

| # | 标准 | 状态 | 说明 |
|:---|:---|:---|:---|
| 1 | 7 个开放问题逐条有决策记录 | **部分达成** | 每条都有状态/建议/证据/决策人角色/输入/影响/下一步；但没有真实决策人签字，不能标“已决策”——这是证据纪律要求，不是遗漏 |
| 2 | 能力矩阵标 CE/EE/Industry 且与代码一致 | **达成** | 逐行给出代码现状与建议归属；明确无 edition gate、readiness 仅报告、两套前端开关，未制造错配 |
| 3 | 定价改为明确决策状态 | **达成** | “待商业决策 — 禁止对外报价”，历史价标作废且不得引用 |
| 4 | PoC 演示脚本（含分诊建议→人工确认） | **部分达成** | 脚本基于真实已接线链路；AI 分诊建议与人工确认明确标注**未接线**，不虚构演示步骤 |
| 5 | 标注证据等级、无 Verified Demand | **达成** | 全部标 Hypothesis / Observed Problem / Doc-only / Proposal；Verified Demand = 0 |
| 6 | 对外发布需人工审批 | **达成** | 未改官网、未发报价、未对外发布；`website/index.html` 用户 WIP 只读 |

### 未执行 / 需人工

- 官网与销售材料的任何发布：需用户明确授权并人工审批。
- 7 项商业决策：需真实决策人填写姓名与日期；AI 未代填。
- 真实客户访谈、PoC SOW、付费意愿验证：无仓库证据，必须由人执行。

### 回滚

本片全部为文档变更，无代码/Schema/配置改动；按文件 `git checkout -- <file>` 即可回退，新增文件直接删除。

---

## 11. Slice B 补强执行记录（2026-11-25）

### 实际变更

- **HTTP 错误语义**：新增真实路由测试，未映射发送者 → `403/2003`、无申请人且未配置自动化账号 → `503/5003`，均不建单。
- **enqueue 失败回滚**：新增红绿用例，用不匹配的 `tenantctx` 触发 commandbus 跨租户守卫，在业务 INSERT 之后真实令 enqueue 失败；断言工单/审计/命令全部回滚、去重行释放，且用正确租户重投成功。反向探针：临时忽略 enqueue 错误 → 测试红（Expected error but got nil）；恢复 → 绿。
- **分诊建议 → 人工确认（接线完成）**：
  - `handlers/feishu/contracts.go` 新增窄契约 `InboundTriageAdvisor` / `InboundDecisionAuditor`，避免 handlers→internal/services 层次倒置。
  - `FeishuSyncService.SetInboundAdvisors` 由 bootstrap 注入 `triageService` 与 `aiTelemetryService`。
  - 建单事务提交后在事务外生成分诊建议（Guidance→LLM→keyword 降级链），写 `ai_feedbacks` 决策底账（`item_type='ai_audit'`、`accepted=NULL`），响应新增 `triageSuggestion` / `triageAuditId`；操作者经既有 `PATCH /ai/audit-logs/:id` 确认，工单详情采纳 category/priority（带 version）。**建议不自动改写工单字段**，落账失败不影响建单。
  - 测试注入 fake advisor/auditor：断言建议溯源字段、底账 scenario/inputRef/confidence、工单 priority 保持默认（证明无 silent authority）、落账失败不阻断建单。
- 文档：`docs/api-reference.md`、`docs/product/poc-demo-script-feishu-inbound.md` 同步为已接线口径；PoC 脚本移除“未接线”表述。

### 验证结果

| 命令 | 结果 |
|:---|:---|
| `go build ./...` | exit 0 |
| `go test ./handlers/feishu/... ./router/... -count=1` | 两包全绿 |
| 反向探针（忽略 enqueue 错误） | 目标测试 FAIL（Expected error but got nil），恢复后 PASS |
| `git diff --check` | exit 0 |

### 仍待人工

生产飞书租户收发与死信重放演练签字栏留空；AI 模型真实语料评测（`cmd/ai-eval -mode llm`）未执行；能力维持 Preview。

---

## 12. 生产验收准备包（2026-11-26）

用户以“2”选择生产验收准备。产物：`docs/acceptance/feishu-inbound-production-acceptance.md`。

### 内容

- 审批边界表：文档可由 AI 产出；凭证配置/密钥轮换/生产部署/故障注入/签字均标为人工审批。
- Step A 能力面基线（只读 `GET /api/v1/capabilities`，记录 `connector.feishu` 的 maturity/pilotReason/acceptanceVersion）。
- Step B 连接器配置与 `rotate-secret` 的精确路径、字段与脱敏判定（并要求 `credentials` 全为 `******`）。
- Step C 真实租户收发 + 工单/审计/命令/AI 底账取证 + 403/409 重放拒绝。
- Step D 受控故障 + 死信重放演练：`prod-dead-letter-check.sh`、`GET/POST /admin/operations/commands`（replay 保留幂等键、递增 fencing token）、通知投递重试、中止条件。
- Step E 建议确认：`PATCH /ai/audit-logs/:id`（`ai:write`），采纳 category/priority 带 version。
- E1–E8 证据清单（只留脱敏字段）、空白签字表、回滚步骤。

### 本轮未执行（人工边界）

未配置真实回调、未轮换密钥、未访问生产、未注入故障、未重放命令、未部署、未签字。
本文件只是准备包，不能作为验收结论。能力维持 `pilot`，`ITSM_FEISHU_READY` 由运维按部署策略申报。

### 回滚

纯文档产物；按文件 `git checkout -- <file>` 回退，新增文件直接删除。
