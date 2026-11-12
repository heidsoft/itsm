# BPMN 核心组实现归位：拆分边界设计（2026-11-11）

> 状态：**G0 已完整落地（G0 纯契约 2026-11-11 + G0-b 内置模板 2026-11-12）**；2026-11-12 收口批已把整个 `service/` 顶层包（含本文 G1/G2/G3 候选文件）整体归位 `internal/services/`（包名不变、零行为变更）。**G1–G3 仍为设计、未落地**：归位只解决物理位置，按域拆分仍须先做共享私有符号解耦。本文划分边界与迁移顺序；已落地批次均不改运行时行为、不重建 prod 镜像。
> 权威读数仍以 `scripts/docs-gate/check-product-drift.sh` 的 C.8 硬模式输出与
> `plans/service-layer-migration-plan.md` 文末落地记录为准。

## 1. 为什么需要设计而不是继续按文件清单搬

L.3 已归零（全部 handler 生产文件零 `import "itsm-backend/service"`），但 `service/`
顶层仍有 128 个生产文件。此前 skill / escalation / sla 三批能整组 `git mv`，是因为它们
「实现自包含 + 消费方可枚举」。BPMN 核心组不满足这个前提，本轮用 Go AST 逐文件统计
同包顶层符号的双向引用，并 grep 复核跨组引用，实测结果：

| 组 | 生产文件 | 代表文件（行数） | 耦合事实 |
| --- | --- | --- | --- |
| `bpmn_` | 32 | `bpmn_process_executor.go`(2160)、`bpmn_task_service.go`(1556)、`bpmn_xml_parser.go`(1255)、`bpmn_monitoring_service.go`(1111) | 与 `approval_` 双向引用；`bpmn_template_digest.go` 取包级 `bpmnTemplates` embed FS |
| `timer_` | 6 | `timer_event_handler.go`(499)、`timer_store.go`(502)、`timer_scheduler.go`(483) | `timer_event_handler.go` 直接取 `engine.executeStep` / `engine.client` / `engine.timerScheduler`（`CustomProcessEngine` **私有字段**） |
| `approval_` | 6 | `approval_service.go`(839)、`approval_chain_evaluation.go`(507) | `approval_chain_evaluation.go` / `approval_service.go` 引用 `BPMN*`；`bpmn_process_binding_service.go` / `bpmn_process_engine.go` 反向引用 `Approval*` |
| `process_` / `workflow_` | 4 | `process_routing_service.go`(418)、`workflow_start_command_handler.go`(358) | 编排入口，由 `app.go` 462–470 与引擎、绑定、触发器一起装配 |
| `ci_` + `cmdb_` | 8 | `cmdb_import_export_service.go`(1545) | **与 BPMN 零耦合**（双向 grep 均无命中），但 `CMDBImportExportService` 直接持有 `*ConfigurationItemService`（组外文件），整组迁会牵出核心 CI 服务 |

`bpmn` / `ci` / `cmdb` 与 BPMN 的双向 grep 均无命中，是唯一能独立成批的中型域；代价是必须把
`configuration_item_service.go` 及 `handlers/cmdb/contracts.go` 的 8 个窄契约一并裁决。

**结论**：BPMN 核心组实际是 **48 个生产文件互相咬合**（32 + 6 + 6 + 4），不能按文件名前缀
逐批搬；`timer_` 尤其不能单批迁——它访问的私有字段在 `CustomProcessEngine` 上，单独搬会
立刻编译断，或更糟地被"复制一份私有访问器"绕过而静默分叉。

## 2. 四组边界与迁移顺序

按「先无状态、后引擎、最后编排」排序，每组都必须自带回归锁与基线精确收紧。

### G0 纯类型 / 纯接口 / 纯函数层（风险最低，建议首批）

- 候选：`bpmn_types.go`(425)、`bpmn_process_trigger_interface.go`(79)、
  `bpmn_template_digest.go`(62)、`bpmn_service_task_adapter.go`(104)、
  `workflow_helper.go`(65)、`bpmn_process_trigger_interface.go` 里的 4 个 `*Interface`。
- 目标：`dto/`（纯值类型与接口）或 `internal/bpmncontracts/`（带包级常量的契约），
  `service` 侧保留 `type X = dto.X` 别名与薄委托。
- 难点：`bpmn_template_digest.go` 依赖 `bpmnTemplates`（`//go:embed`，定义在其他
  service 文件），需把 embed FS 一起收进同一包，否则会出现两份模板清单。
- 锁点：模板 key 集合与 digest 必须与 HEAD 逐字节一致（编辑模板应改版本账本）。

**G0 已落地部分（2026-11-11）**：`bpmn_types.go`(425) 与 `bpmn_process_trigger_interface.go`(79)
以 `git mv` 迁入 `internal/bpmncontracts/`（`types.go` / `interfaces.go`，包名 `bpmncontracts`），
`service/bpmn_types.go` 保留为 28 个 `type X = bpmncontracts.X` 别名过渡层（未新增生产文件、
`app.go` 净增 0 行、54 个外部测试包零改动）；`workflow_helper.go` 经全仓零消费方实证后退役。
回归锁 `internal/layering/bpmn_contracts_ownership_test.go` 含 XML 往返行为锁与 7/7 反向验证。
L.1 128→126、C.6.4 `service_go_files` 335→333，详见 `plans/service-layer-migration-plan.md` 文末落地记录。

**G0-b 已落地（2026-11-12）**：20 个 `.bpmn` 模板与 `BuiltinProcessTemplateKeys` /
`EmbeddedBPMNDigest` 整体 `git mv` 进 `internal/bpmntemplates/bpmn/`（embed 声明与模板同包，
FS 内路径仍为 `bpmn/<file>`）；`service/bpmn_template_digest.go` 退役为
`bpmn_template_service.go` 里的两个薄委托，`bpmnTemplates` 过渡变量由
`bpmntemplates.FS()` 提供，`app.go` 零改动。模板内容逐字节未变（迁移前后摘要
`69a25a6808d1ef39a9f29d7a60a5e4d0937c4306c300625051099ce0d15f7ce6` 一致）。
**设计里没预见到的第三类读面**：除 `domain/role/contract_test.go` 与
`pkg/seeder/process_bindings_manifest_test.go` 两处磁盘枚举外，service 包内还有 5 个
模板门禁测试按包内相对路径 `bpmn/…` 读磁盘（目录里只剩 Go 文件后静默读成空集合）；
现统一改走 `bpmnTemplates.Read*`，与生产同源，后续再搬模板不再需要改测试。
回归锁 `internal/layering/bpmn_template_embed_ownership_test.go` 锁完整性清单、
摘要锚点、独立重算、委托形态、读面同源、零复活与两份基线棘轮，反向验证 8/8 全红。
实测 L.1 **126→125**、C.6.4 `service_go_files` **333→332**，详见
`plans/service-layer-migration-plan.md` 文末落地记录。

**仍未落地**：`bpmn_service_task_adapter.go` 依赖 service 私有符号与 `service/bpmn`
子包，不属纯契约，归 G3；`timer_` 私有字段访问改造仍属 G2 前置，ServiceTask
完成事务凭据不得触碰；G1 域服务、G2 引擎核心、G3 编排桥接均保持本文原设计。

### G1 域服务层 → `handlers/bpmn/`

- 候选：`bpmn_monitoring_service.go` / `bpmn_audit_service.go` / `bpmn_metrics_service.go` /
  `bpmn_version_service.go` / `bpmn_template_service.go` / `bpmn_deployment_service.go` /
  `bpmn_lint_service.go` / `bpmn_pending_inventory.go` / `bpmn_permission_service.go` /
  `bpmn_variable_service.go` / `bpmn_event_service.go` / `bpmn_sla_service.go` /
  `bpmn_tenant_service.go` / `bpmn_process_instance_service.go` 等约 18 个文件。
- `handlers/bpmn/contracts.go` 已有 8+ 窄契约与 48 个方法签名（cmdb 专项 B 用 Go AST 抽取过），
  是现成的接线面；迁入实现时须核对域内已有接口名与具体类型是否重名（escalation 批踩过
  `EscalationMatrixService` 同名冲突，解法是域内接口改名 `MatrixService`）。
- 锁点：域包零 `import service`、构造点限定符、旧文件名零复活、L.1 基线删行。

### G2 引擎核心 → `internal/bpmnengine/`（必须与 `timer_` 同批）

- 候选：`bpmn_process_engine.go`(201)、`bpmn_process_executor.go`(2160)、
  `bpmn_xml_parser.go`(1255)、`bpmn_task_service.go`(1556)、`timer_*` 6 个，
  以及 `bpmn_process_lifecycle.go` / `bpmn_timeout_scanner.go`。
- 前置改造：把 `timer_event_handler.go` 对 `engine.client` / `engine.executeStep` /
  `engine.timerScheduler` 的私有字段访问，改为引擎暴露的**窄访问器接口**（注入而非直接取字段）；
  这一步本身就是行为不变的装配改造，需单列回归锁（访问器语义 + exactly-once 凭据不受影响）。
- 风险最高：ServiceTask exactly-once 的凭据锚点在完成事务里（`_serviceTaskExecutedOccurrences`
  + `_serviceTaskReceipts`），迁移不得改变事务边界；`workflow` 仍 Pilot，本组任何改动都不得
  被解读为 GA 证据。
- 锁点：跨租户跨域 E2E、补偿恢复、ServiceTask 凭据计数与对账。

### G3 编排 / 桥接（最后裁决）

- 候选：`approval_*` 6、`bpmn_approval_bridge_service.go`、`process_*` 2、`workflow_*` 2、
  `bpmn_service_task_command_handler.go`、`bpmn_ai_generator_service.go` 等。
- 与 `handlers/approval` / `handlers/approval_chain` / `handlers/service_request` / `handlers/change`
  的既有窄契约交叉，须等 G1/G2 稳定后按「谁构造、谁拥有」裁决，避免 `service → handlers`
  反向依赖。

## 3. 每批的验收契约

沿用已验证的模板，缺一不可：

1. `git mv` 保留 blame；`service` 侧只留 `dto` 别名 / 薄委托，禁止复制实现或另起一份口径。
2. 新归位包零 `import "itsm-backend/service"`、零 `import "itsm-backend/handlers/"`。
3. 回归锁同时覆盖「编译可过但语义全错」：别名退化、私有符号双份、旧文件名复活、
   L.1 基线行复活、C.6.4 上界回退；棘轮用解析数值 + 只许继续下降，不硬编码本批值。
4. 反向验证逐处改回旧值必须由**断言变红**（至少一半用例不能靠编译失败），
   try/finally 恢复 byte-identical 后重新转绿。
5. 基线按 `measure` 口径精确下调（L.1 删行、C.6.4 `service_go_files`），行尾写清加减账；
   `app.go` 若净增行须同 commit 上调 `bootstrap_app_lines` 并写理由。
6. 并行会话 WIP 高峰用 HEAD 快照 blob + `cacheinfo` 隔离提交，不把对方 WIP 编进镜像或提交。

## 4. 本批（向量组）在路线中的位置

本批先把与 BPMN 零耦合的 `vector_store.go` / `embed_pipeline.go` / `embedder_openai.go` /
`ai_services.go` 归入 `internal/vectorstore`，验证「共享基础设施迁 `internal/`」这条
收口路径可行：`app.go` 零改动、类型身份由别名保证、L.1 132→128、C.6.4 339→335、
反向验证 6/6 全红。它是 G2 之前的热身，不替代 BPMN 设计本身。

## 5. 明确不在本文范围

- `workflow` 真实租户跨域联调人工签字（`docs/acceptance/workflow-cross-domain-signoff.md`
  签字栏空白，不由 AI 代签）。
- 三套成熟度口径统一（README / 能力矩阵 / 商业能力契约）与 `ROADMAP` v1.6.x 登记条目团队评审。
- L.6 符号级共享层依赖白名单守卫（独立治理项，可与 G0 并行）。


---

## 收口批与本文的关系（2026-11-12）

收口批把 `service/` 顶层 332 个 `.go`（125 生产 + 207 测试）整体 `git mv` 进
`internal/services/`，包名仍为 `service`；本文 G1/G2/G3 列出的候选文件**一个都没有按域拆开**，
只是换了目录。原因由 Go AST 实测支撑：332 个文件的无向连通分量只有 5 个，最大 329 文件
（124 生产 + 205 测试），包内私有符号与共享测试 helper 让「改一部分文件的包名」立刻编译失败。
因此 G2 的「`timer_` 私有字段访问改造」仍是 G2 的前置条件，ServiceTask 完成事务凭据不得触碰；
G1 迁入 `handlers/bpmn/` 前也仍需先解决同名接口与具体类型的重名问题。

归位后 L.1 文件名棘轮改锁 `internal/services/`（125 条逐行不变），新增 L.1b 守旧目录零 `.go`，
C.6.4 新增 `shared_services_go_files=332` 上界；后续在 `internal/services/` 内部按域拆分时，
每拆出一个包就应同步下调该上界并在 CHANGELOG 记录，不得放宽。
详见 `output/service-layer-relocation-2026-11-12.md`。
