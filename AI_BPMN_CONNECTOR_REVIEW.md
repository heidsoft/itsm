# AI能力落地 + BPMN流程引擎 + Connector生产化专项审查报告

## 审查范围
- `itsm-ai-service/` (Python FastAPI sidecar)
- `itsm-backend/service/llm_gateway.go`、`llm_providers.go`
- `itsm-backend/service/triage_service.go`、`summarize_service.go`、`rag_service.go`
- `itsm-backend/service/bpmn_*` / `bpmn/` 目录
- `itsm-backend/connector/` (生命周期管理)
- `itsm-backend/internal/commandbus/` (outbox pattern)
- `itsm-backend/internal/bootstrap/app.go` (DI wiring)

---

## 一、AI能力真实性审查

### 1.1 `itsm-ai-service` (Python sidecar) — ⚠️ 弱依赖 + 规则兜底

| 路径 | 结论 |
|------|------|
| `services/llm_client.py` | **真实 HTTP 调用后端 Go API** `/api/v1/ai/triage` 等，端到端经过 `LLMGateway` |
| `api/triage.py` | **优先调后端**，失败时走 `_rule_based_triage()` 规则兜底；规则兜底硬编码 `confidence=0.85`，无真实 LLM |
| `config.yaml` | `provider: "openai"`, `api_key: ""`（空），**依赖环境变量注入** |

**结论**：`itsm-ai-service` 本身不直接调 LLM，是后端 Go LLMGateway 的代理层 + 规则降级。真实 LLM 能力由 `llm_gateway.go` 注入的后端 provider 决定。

---

### 1.2 `llm_gateway.go` Provider 支持 — ✅ 真实多provider

| Provider | 状态 | 关键文件 |
|----------|------|----------|
| OpenAI (`gpt-4o-mini`等) | ✅ 真实 via `sashabaranov/go-openai` | `llm_providers.go:18–145` |
| Azure OpenAI | ✅ 真实 | `llm_providers.go:307–357` |
| MiniMax (Anthropic兼容) | ✅ 真实 | `llm_providers.go:379–494` |
| Local/Ollama | ✅ 真实 HTTP | `llm_providers.go:359–562` |
| 降级 fallback | ✅ keyword heuristic | 各 service 内置 |

**`llm_providers.go:387–396` MiniMax API Key 明文存储**（红队敏感信息泄露风险，P2）

**`llm_providers.go:574–612`**: `LoadLLMConfig()` 从 viper 读取，`NewProviderFromConfig()` 支持 env var 覆盖 (`OPENAI_API_KEY`/`AZURE_OPENAI_API_KEY`/`MINIMAX_API_KEY`)。

---

### 1.3 TriageService — ✅ 真实 LLM + 三级降级

```
优先级1: Guidance sidecar (约束生成)
优先级2: LLM Gateway 直接调用 → llmClassify() → /chat/completions
优先级3: keyword-based fallback (keywordBasedSuggest)
```

关键代码 `triage_service.go:141–205`:
- `llmClassify()` 直接调 `gateway.Chat()`，prompt 约束 JSON 输出
- 当 `llmResult.Confidence < 0.5` 时与 keyword 结果对比取高
- **P2 问题**: `suggestedFix` 字段 JSON key 是 `suggested_fix`（snake）而 struct tag 是 `SuggestedFix`（camel），可能导致反序列化丢失

---

### 1.4 SummarizeService — ✅ 真实 LLM

`summarize_service.go:27–77`:
- 调 `gateway.Chat()` 生成中文摘要
- 无 LLM gateway 时 `simpleTruncate()` 降级
- `GenerateActionItems()` 同样走 LLM，失败返空 slice

---

### 1.5 RAGService — ✅ 真实多层混合检索

`rag_service.go:22–43`:
- **Vector search** (via `connector/vector/` 可插拔 connector)
- **Keyword fallback** (PostgreSQL LIKE/FTS)
- **Hybrid search** (vector + keyword 合并)
- **Ontology graph** (1-hop 关系扩展，可选注入)
- `knowledgeGuard` (L0 可见性守卫，默认注入)
- `freshness` (L1 时效性守卫，默认注入 `DefaultFreshnessPolicy`)
- 启动期 `NewRAGServiceWithAutoConfig()` 测试 embedder 可用性，不可用时自动降级

**P2**: `rag_service.go:42` `freshness` 构造 `PermissiveFreshnessPolicy(nil)` — 生产应显式配置严格策略。

---

## 二、BPMN 流程引擎可靠性审查

### 2.1 Outbox + Command Bus — ✅ 完整Saga可靠执行

`internal/commandbus/commandbus.go`:

| 特性 | 实现 |
|------|------|
| Durable write | `operational_commands` 表，事务内与业务写一同提交 |
| Idempotency | `IdempotencyKey` 去重（`EnqueueResourceNotificationTx` 用 `OccurrenceKey` 组合） |
| Lease / Fencing | `fencing_token` 乐观锁 + `lease_expires_at`  TTL |
| Heartbeat | 后台 goroutine 每 `leaseTTL/3` 续期 |
| Retry backoff | `2^attempt` 秒 capped 300s，8次后进 `dead_letter` |
| 跨租户 | `tenant_id` 硬隔离，SystemContext 旁路 |

**Command 类型**:
- `workflow.start` / `workflow.service_task.execute`
- `notification.deliver`
- `ticket.rules.execute` / `incident.rules.execute`
- `cmdb.import.process` / `cmdb.export.process`
- `email_intake.email.send` / `email_intake.message.process`

### 2.2 BPMN ServiceTask 执行链路 — ✅ 命令驱动

```
BPMN ServiceTask 节点触发
  → operational_command (commandbus) 持久化（事务内）
  → Worker.RunOnce() 抢 lease
  → HandleBPMNServiceTaskCommand()
    → callbackRegistry.GetHandler(serviceRef)
    → handler.Execute() (ApprovalHandler 等)
    → 事务提交 ServiceTask 完成 + 流程推进
```

关键文件: `service/bpmn_service_task_command_handler.go`

### 2.3 ApprovalHandler — ✅ 真实注入，非 stub

`service/bpmn/approval_handler.go`:
- **不是 stub**：依赖 `ApprovalServiceInterface.SubmitApproval()` 真实写 DB
- DI 链路: `app.go:468` → `processEngine.SetApprovalService(approvalService)` → `callbackRegistry.SetApprovalService()` → 注入 `ApprovalHandler.approvalService`
- 支持 `approve` / `reject` / `delegate` / `escalate` 四种 action
- 租户隔离强制校验 (L57–60)
- **P1**: `app.go:468` 顺序正确，但 `processEngine` 是在 `app.go:335` 创建，`approvalService` 何时创建需确认无空窗期

---

## 三、Connector 生命周期管理审查

### 3.1 Manager 生命周期 — ✅ 完整

`connector/manager.go`:

| 方法 | 职责 |
|------|------|
| `Provision()` | 创建/更新实例，Init → 旧实例 Close → PollingReceiver.Start |
| `Revoke()` | Close + 从 map 删除 |
| `Get()` | 按 `tenantID + name` 查找（`Enabled` 过滤） |
| `GetByCallbackInstanceID()` | Webhook 回调路由（不暴露 tenantID） |
| `HealthCheckAll()` | 5s timeout 遍历所有实例 |
| `CloseAll()` | 优雅停机 |
| `SetInboundHandler()` | 动态注册 polling consumer |

**PollingReceiver 支持**: `connector/manager.go:91–97` — `PollingReceiver.SetInboundHandler()` + `Start(ctx)` 动态绑定。

### 3.2 Registry — ✅ 按 name 注册 factory

`connector/registry.go`: `map[name]func() Connector` — 启动时注册 factory，Provision 时调用 factory 创建实例。

### 3.3 内置 Connector 目录
`connector/builtin/`: alert, builtin
`connector/`: manager, registry, router, persistent_store, connector.go (interface 定义)

---

## 四、问题汇总

### P0 — 生产堵点（必须修复）

| # | 域 | 问题 | 文件 |
|---|-----|------|------|
| P0-1 | AI | **生产环境 `LLM_API_KEY` 空值时 `log.Fatalf` 终止启动**（`app.go:509`）— 合理，但需确保 CI/CD 在发布前配置好密钥注入机制，否则发布流程断裂 | `internal/bootstrap/app.go:505–510` |
| P0-2 | AI | **MiniMax API Key 在 `llm_providers.go:389` 硬编码 `***`（实际代码中明文）** — `apiKey` 字段敏感信息在源码中 | `service/llm_providers.go:389` |

### P1 — 高优先级（影响功能正确性）

| # | 域 | 问题 | 文件 |
|---|-----|------|------|
| P1-1 | AI | **TriageService `SuggestedFix` JSON key 不一致** — prompt 输出 `suggested_fix`（snake），struct tag `json:"suggestedFix"`（camel），导致反序列化丢失 | `service/triage_service.go:235` + `triage_service.go:55` |
| P1-2 | AI | **RAG `FreshnessJudger` 默认装配 `PermissiveFreshnessPolicy(nil)`** — 生产应使用严格策略，当前等于不过滤过期知识 | `service/rag_service.go:113` |
| P1-3 | BPMN | **ApprovalHandler 依赖注入时序风险** — `NewApprovalHandler` 在 `registerDefaultHandlers()` 时创建（`bpmn_callback_registry.go:143`），但 `SetApprovalService` 在 `app.go:468` 调用。若 `approvalService` 创建晚于 `processEngine`，存在 nil 窗口 | `service/bpmn/bpmn_callback_registry.go:143` vs `internal/bootstrap/app.go:468` |

### P2 — 中优先级（安全/配置）

| # | 域 | 问题 | 文件 |
|---|-----|------|------|
| P2-1 | AI | **MiniMaxProvider API Key 在源码明文**（实际代码不是 `***`，是真实 key 占位符） | `service/llm_providers.go:389` |
| P2-2 | AI | **guidance sidecar URL 硬编码** `http://localhost:8091`（`app.go:586–588`），生产环境需通过环境变量覆盖 | `internal/bootstrap/app.go:586–588` |
| P2-3 | AI | **RAG Hybrid Search 降级路径未显式测试** — `NewRAGServiceWithAutoConfig` 自动降级，但无 E2E 测试覆盖 vector store 完全不可用场景 | `service/rag_service.go:117–149` |
| P2-4 | Connector | **Connector `Get()` 只返回第一个匹配实例** — 多实例场景（如同名机器人的多租户实例）只能拿到第一个，callback routing 用 `GetByCallbackInstanceID` 弥补但普通 Send 不支持 | `connector/manager.go:122–131` |
| P2-5 | BPMN | **CommandBus Worker lease TTL 1分钟** — 长时间运行 handler（如外部 API 调用）可能触发 lease 续期失败，需确认 heartbeat 间隔与 handler 时长关系 | `internal/commandbus/commandbus.go:165` |

### P3 — 低优先级（可改进）

| # | 域 | 问题 |
|---|-----|------|
| P3-1 | AI | `llm_gateway.go` token 估算用 rune count /4，非真实 tokenizer，长文本可能偏差 |
| P3-2 | AI | TriageService keyword fallback confidence 固定 `0.55`，高于某些 LLM 低质量结果但未动态调节 |
| P3-3 | Connector | Manager 无连接器版本管理，Provision 时直接覆盖无灰度 |

---

## 五、总体评估

| 维度 | 评级 | 说明 |
|------|------|------|
| AI LLM 真实性 | ✅ 真实 | Go 侧 `LLMGateway` 直接调各 provider API，非 stub；Python sidecar 是代理降级层 |
| AI 降级健壮性 | ✅ 良好 | Triage/Summarize/RAG 均有三级降级（LLM → keyword/规则 → truncate） |
| BPMN Outbox 可靠性 | ✅ 完整 | CommandBus 实现标准 Saga：持久化 + idempotency + lease + heartbeat + dead-letter |
| BPMN ApprovalHandler | ✅ 真实 | 非 stub，通过 DI 链路真实调用 `ApprovalService.SubmitApproval()` |
| Connector 生命周期 | ✅ 完整 | Provision/Revoke/Get/HealthCheck/CloseAll 闭环，有 polling receiver 支持 |
| 生产密钥安全 | ⚠️ 需修复 | MiniMax API Key 需移除明文，GUIDANCE_URL 需环境变量化 |

**核心结论**: AI 能力和 BPMN 引擎均为**真实生产级实现**，无 stub 冒充问题。主要风险在配置管理（密钥、环境变量）和个别 JSON 序列化细节（P1-1）。Connector 生命周期管理完整，生产可用。
