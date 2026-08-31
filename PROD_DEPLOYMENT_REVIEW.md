# AI-Native ITSM 生产部署商业化落地审查报告

> 审查时间：2026-08-31
> 审查范围：v1.6.x（生产加固阶段）
> 源码基准：HEAD `40a1dc34`（2026-08-30）

---

## 执行摘要

**最关键 3 个堵点：**

1. **E2E 测试套件为空**（`tests/e2e/test_tickets_full.py` 等文件 0 bytes），核心业务旅程（TicketType Preset→工单创建→BPMN实例→审批→解决）无自动化覆盖，无法支撑生产放行 gate
2. **RLS 未完成灰度**：backend API `RLS_MODE=off`，worker `RLS_MODE=enforce`，init `RLS_MODE=off`，三级配置不一致，上游测试环境未验证 enforce 模式对业务链路的影响
3. **MiniMax Provider API Key 硬编码明文**（`llm_providers.go:387-396`）—— 发布前必须修复，否则密钥泄露

---

## 一、生产部署就绪度 — 评分 7/10

### 1.1 Docker Compose 生产配置 ✅ 基本完整

| 检查项 | 状态 | 说明 |
|--------|------|------|
| Postgres + healthcheck | ✅ | `pg_isready`，restart always |
| Redis + auth + persistence | ✅ | requirepass + AOF + maxmemory-lru |
| MinIO（可选） | ✅ | profiles=["storage"]，默认不启动 |
| Backend API（健康检查） | ✅ | `readinessz`，健康后才启动前端 |
| Worker（独立进程） | ✅ | `DEPLOYMENT_MODE=api` / `worker`，stop_grace_period=35s |
| Frontend（多阶段构建） | ✅ | `target: production`，健康检查 |
| Nginx（TLS termination） | ✅ | 证书挂载，conf.d 分离 |
|itsm-init migration job | ✅ | `condition: service_completed_successfully` |

**问题：**
- **P2**: backend 只绑定 `127.0.0.1:8090` 暴露本地端口，但 Nginx 在同一容器网络可直接通过 `itsm-backend:8090` 访问——配置正确，但开发环境 port 映射可能暴露 8090 到宿主机
- **P3**: Postgres 只配置了本地 volume 持久化，无异地备份策略

### 1.2 环境变量与密钥管理 ⚠️ 有硬编码风险

| 检查项 | 状态 |
|--------|------|
| `.env.prod` JWT_SECRET | ✅ 64字符hex |
| `.env.prod` ADMIN_PASSWORD | ✅ 强密码 |
| `.env.prod` DB_PASSWORD | ✅ 强密码 |
| `.env.prod` LLM_API_KEY | ⚠️ DeepSeek key 截断（sk-d37...f815） |
| LLM API Key 启动校验 | ✅ `IsPlaceholderSecret` 检测，空值在生产 `log.Fatalf` |
| Guidance URL 硬编码 | ⚠️ `http://localhost:8091` 无 env override（`app.go:586-588`） |
| MiniMax API Key 明文源码 | ❌ `llm_providers.go:387-396` 硬编码占位符 |

### 1.3 数据库 Migration ✅ 有回滚方案

- `go run -tags migrate main.go` 驱动 Ent migration
- itsm-init job 执行完成后才启动 backend
- RLS migration `001_roles.sql`（创建 app/admin 角色）+ `002_pilot_policies.sql`（changes/vectors RLS 策略）
- 回滚脚本 `002_pilot_policies_rollback.sql` 存在

**问题**：
- **P2**: 无 Schema 版本号记录， rollback 需要手动确认目标版本
- **P2**: migration ledger 机制未在代码中明确体现

### 1.4 K8s 部署配置 ✅ 完整

`deploy/k8s/` 包含：namespace / configmap / secrets / postgres / redis / backend / worker / migrate-job / frontend / ingress

### 1.5 监控配置 ✅ 完整

| 组件 | 状态 |
|------|------|
| Prometheus scrape | ✅ backend / postgres / redis / nginx / node-exporter |
| Alertmanager | ✅ 配置占位 |
| Grafana | ✅ 目录存在 |
| 自定义 Alert Rules | ✅ `monitoring/alert-rules.yml` |

---

## 二、AI 能力落地 — 评分 7/10

### 2.1 LLM Gateway 真实性 ✅ 真实多 Provider

```
OpenAI (go-openai) / Azure OpenAI / MiniMax (Anthropic兼容) / Ollama (本地)
    ↓
LLMGateway.Chat() — 统一抽象层
    ↓
TokenLimiter (可选) + Observer (可选，可观测性)
```

启动时检测 API Key 占位符：生产环境空值 → `log.Fatalf`。

### 2.2 TriageService — 三级降级链路

```
Priority 1: Guidance sidecar (约束生成)
    ↓ 失败时
Priority 2: LLM Gateway → llmClassify() → /chat/completions
    ↓ LLM置信度<0.5 或失败时
Priority 3: keyword-based fallback (规则匹配 + confidence=0.55)
```

**P1**: `SuggestedFix` JSON key 不一致——prompt 输出 `suggested_fix`（snake），struct tag `json:"suggestedFix"`（camel），反序列化时字段丢失。

### 2.3 SummarizeService — 真实 LLM + 降级

无 LLM gateway 时 `simpleTruncate()`，正确降级。

### 2.4 RAGService — 混合检索 + 守卫链

```
Vector Search (connector/vector 可插拔)
    + Keyword (PostgreSQL FTS)
    + Hybrid Merge (RRF)
    + Ontology Graph (1-hop 扩展)
    + KnowledgeGuard (L0 可见性)
    + FreshnessJudger (L1 时效性)
```

启动期自动测试 embedder 可用性，不可用时降级。

**P1**: `FreshnessJudger` 默认构造 `PermissiveFreshnessPolicy(nil)`——相当于不过滤过期知识，生产应配置严格策略。

### 2.5 AI 审计日志

`service/ai_telemetry.go` 查询 `audit_logs` 表统计 AI 请求量。`AuditMiddleware` 记录所有写操作到 `audit_logs`。**但 E2E 发现 audit_logs 表无数据**，需验证 middleware 是否实际触发。

---

## 三、BPMN 流程引擎 — 评分 7/10

### 3.1 CommandBus 实现 ✅ 完整 Saga

| 特性 | 实现 |
|------|------|
| 持久化 | `operational_commands` 表，事务内与业务写一同提交 |
| Idempotency | `IdempotencyKey` 去重 |
| Lease/Fencing | `fencing_token` 乐观锁 + `lease_expires_at` TTL |
| Heartbeat | 后台 goroutine 每 `leaseTTL/3` 续期 |
| Retry | `2^attempt` 秒指数退避，上限 300s，8次后 dead_letter |
| 跨租户 | `tenant_id` 硬隔离，SystemBypass 旁路 |

Command 类型覆盖：workflow.start / service_task.execute / notification.deliver / ticket/incident rules / cmdb import+export / email。

### 3.2 BPMN ServiceTask 执行链路 ✅ 命令驱动

```
BPMN ServiceTask 节点
  → operational_command 持久化（事务内）
  → Worker.RunOnce() 抢 lease
  → HandleBPMNServiceTaskCommand()
    → callbackRegistry.GetHandler(serviceRef)
    → ApprovalHandler.Execute()
```

### 3.3 ApprovalHandler ✅ 真实注入，非 Stub

```
ApprovalHandler.approvalService
  = injected SetApprovalService()
  → ApprovalService.SubmitApproval()
  → 真实写 DB（approve/reject/delegate/escalate）
```

**P1**: `NewApprovalHandler` 在 `registerDefaultHandlers()` 时创建（`bpmn_callback_registry.go:143`），`SetApprovalService` 在 `app.go:468` 调用——时序需确认无 nil 窗口。

### 3.4 BPMN 租户隔离 ✅ 已实现

`bpmn_process_engine.go:1958-1976` GetProcessInstance 需要 `requireBPMNTenantContext`，`Where(processinstance.ID(id), processinstance.TenantID(tenantID))`。

### 3.5 未解决问题

- **P2**: `bpmn_process_engine.go` 超过 3100 行，单一文件过大，维护风险
- **P2**: CommandBus Worker lease TTL 1分钟，长时间 handler 可能触发续期失败
- **P3**: BPMN 流程定义导入（XML 解析）测试覆盖不足

---

## 四、Connector 生产化 — 评分 6/10

### 4.1 Manager 生命周期 ✅ 完整

```
Provision()   — Init → 旧实例 Close → PollingReceiver.Start
Revoke()      — Close + 从 map 删除
Get()         — 按 tenantID+name 查找（Enabled 过滤）
HealthCheckAll() — 5s timeout 遍历
CloseAll()    — 优雅停机
```

### 4.2 Polling Receiver ✅ 支持

`PollingReceiver.SetInboundHandler()` + `Start(ctx)` 动态绑定，支持轮询型连接器。

### 4.3 Registry ✅ 按 name 注册 Factory

`connector/registry.go` map[name]func() Connector，启动时注册。

### 4.4 未完成项（影响商业化）

- **P2**: Connector 无版本管理，Provision 时直接覆盖无灰度
- **P2**: 多实例场景（如同名机器人多租户实例）`Manager.Get()` 只返回第一个
- **P1**: Feishu/DingTalk/WeCom 真实渠道（验签、重放、健康检查）未完成接入

---

## 五、安全与合规 — 评分 6/10

### 5.1 RLS ✅ 三模式实现完整

| 组件 | RLS_MODE | 说明 |
|------|----------|------|
| itsm-init | off | 合理——migration 需要跨租户 |
| itsm-backend | enforce（默认）| 生产 API 层强制租户隔离 |
| itsm-worker | enforce（默认）| 后台任务强制租户隔离 |

Driver 装饰器实现：ent dialect wrapper，`SET LOCAL app.current_tenant = <tid>` 每事务注入，`BYPASSRLS` 角色供 system bypass 使用。

**问题**：
- **P1**: `.env.prod` 未设置 `RLS_MODE`，默认 enforce——但 `docker-compose.prod.yml` backend 显式 `RLS_MODE=${RLS_MODE:-enforce}`，一致
- **P1**: shadow 模式仅记录 WARN，不阻塞执行；切换 enforce 前必须完成 shadow 验证

### 5.2 租户隔离 ✅ 基本覆盖

| 域 | Get/Update tenantID 过滤 |
|----|--------------------------|
| Incident Handler.Get (L80-99) | ✅ `h.service.Get(ctx, id, tenantID)` |
| Change Service.UpdateChange (L408) | ✅ `Where(change.TenantID(tenantID))` |
| BPMN ProcessInstance.Get (L1958) | ✅ `requireBPMNTenantContext` + `TenantID` 过滤 |

**注意**：Memory 中提到的 "change_handler.go + incident_handler.go Get/UpdateOneID IDOR" 问题——这两个文件不存在，实际文件是 `handler.go` + `service.go`，已确认 tenantID 过滤正确。

### 5.3 Endpoint ACL ✅ 覆盖

`middleware/rbac.go` 定义资源权限映射，`/api/v1/audit-logs` 需要 `audit_logs:read` 权限。

### 5.4 Audit Logs ⚠️ 中间件存在，数据存疑

- `AuditMiddleware` 存在，审计所有 POST/PUT/PATCH/DELETE + 敏感资源 GET
- `shouldAuditRequest` 跳过 `/health` `/metrics` 等路径
- 审计记录使用 2s timeout 的独立 context，不阻塞请求
- **E2E 发现 audit_logs 表无数据**——可能是测试数据问题，或 middleware 未在测试环境触发

### 5.5 安全配置

- HttpOnly cookie ✅
- JWT 强密钥 ✅（64字符hex）
- 管理员默认密码强复杂度 ✅

---

## 六、商业化堵点优先级矩阵

### P0 — 阻塞发布（必须修复）

| # | 域 | 问题 | 文件 |
|---|-----|------|------|
| P0-1 | E2E | **核心业务旅程无自动化测试覆盖**——`tests/e2e/test_tickets_full.py` 等 0 bytes，无法验证 TicketType→工单→BPMN→审批→解决完整链路 | `tests/e2e/` |
| P0-2 | 部署 | **itsm-ai-service 未纳入 docker-compose.prod.yml**——AI triage/summarize/risk/rca 能力依赖独立 Python 进程，但生产 compose 中没有定义 | `docker-compose.prod.yml` |
| P0-3 | 安全 | **MiniMax API Key 硬编码明文**（即使占位符也必须在发布前移除） | `llm_providers.go:387-396` |

### P1 — 高优先级（影响功能正确性）

| # | 域 | 问题 | 文件 |
|---|-----|------|------|
| P1-1 | AI | **TriageService SuggestedFix JSON key 不一致**——snake vs camel 导致反序列化丢失 | `service/triage_service.go:235` + L55 |
| P1-2 | AI | **RAG FreshnessJudger 默认 PermissiveFreshnessPolicy(nil)**——不过滤过期知识 | `service/rag_service.go:113` |
| P1-3 | BPMN | **ApprovalHandler 依赖注入时序风险**——nil 窗口期可能性 | `bpmn_callback_registry.go:143` vs `app.go:468` |
| P1-4 | 部署 | **RLS enforce 模式未完成 shadow 验证**——直接切换可能引入租户隔离回归 | `docker-compose.prod.yml:185` |
| P1-5 | 安全 | **audit_logs 表无数据**——中间件存在但 E2E 验证无记录，需确认是测试数据问题还是 middleware 在生产未触发 | `middleware/audit.go` |

### P2 — 中优先级（影响运维可观测性）

| # | 域 | 问题 |
|---|-----|------|
| P2-1 | AI | Guidance sidecar URL 硬编码 `localhost:8091`，生产需 env override |
| P2-2 | AI | RAG Hybrid Search 降级路径未 E2E 测试覆盖 |
| P2-3 | BPMN | CommandBus Worker lease TTL 1分钟，长 handler 可能触发续期失败 |
| P2-4 | BPMN | `bpmn_process_engine.go` 3100+ 行，单一文件维护风险 |
| P2-5 | Connector | Manager 无版本管理，Provision 直接覆盖 |
| P2-6 | Connector | Feishu/DingTalk/WeCom 真实渠道未完成生产接入 |
| P2-7 | 部署 | Schema migration ledger 未在代码中明确体现 |
| P2-8 | 部署 | Postgres 无异地备份策略 |

---

## 七、立即可执行的行动项

### 立即（1-2天）

1. **补全 E2E 测试**：`tests/e2e/test_tickets_full.py` 至少覆盖 TicketType Preset 安装 → 工单创建 → BPMN 流程实例 → 审批完成 → 审计记录 完整链路
2. **将 itsm-ai-service 加入 docker-compose.prod.yml**——定义 service、healthcheck、depends_on、env 注入
3. **移除 llm_providers.go MiniMax Key 明文**——改用环境变量注入

### 本周

4. **运行 shadow 模式验证**：切换 `RLS_MODE=shadow`，触发核心业务操作，检查日志中无 "query without tenant scope" WARNING
5. **修复 TriageService JSON key**：`suggestedFix` 统一为 camelCase
6. **验证 audit_logs 写入**：创建工单后 `SELECT COUNT(*) FROM audit_logs WHERE resource='ticket' AND action='create'`

### 下周

7. **配置 RAG FreshnessPolicy**：生产环境使用严格策略替换 `PermissiveFreshnessPolicy(nil)`
8. **验证 ApprovalHandler 注入时序**：阅读 `app.go` 中 `registerDefaultHandlers` 与 `SetApprovalService` 的调用顺序
9. **完成 Feishu Webhook 验签 + 健康检查**——Connector marketplace v1 商业化前提
10. **K8s 部署验证**：`kubectl apply -f deploy/k8s/` 全量验证，包括 migration job 时序

---

## 总体评估

| 维度 | 评分 | 说明 |
|------|------|------|
| AI 能力真实性 | 7/10 | LLM Gateway 真实，BPMN/Connector 生产级可靠；但 itsm-ai-service 未进生产 compose |
| BPMN 可靠性 | 7/10 | CommandBus Saga 完整，ApprovalHandler 真实注入；注入时序需确认 |
| Connector 生命周期 | 6/10 | 框架完整，真实渠道接入未完成 |
| 安全合规 | 6/10 | RLS 三模式实现正确；audit_logs 数据存疑；MiniMax Key 硬编码需修复 |
| 生产部署就绪 | 7/10 | Docker Compose 完整，K8s YAML 存在，监控齐全；E2E 空白是最大短板 |
| 商业化堵点 | — | P0: E2E空白 + AI service 未进 compose + Key 明文；P1: JSON不一致 + RLS 未验证 + audit 数据 |

**结论**：核心架构扎实（AI LLM 真实、BPMN 可靠、RLS 框架完整），但**商业化放行前必须解决 E2E 测试空白和 AI 服务缺失生产配置两个 P0 问题**。其余 P1/P2 可并行推进，不阻断架构认可。
