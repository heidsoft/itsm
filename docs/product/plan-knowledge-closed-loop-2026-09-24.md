# P1-1 知识闭环详细 Plan — 2026-09-24

> **目的**：把 [owner-boards §@backend-itsm P1-1](./owner-boards-2026-09-24.md#p1-1--问题--ke--知识--rag-闭环主导) "问题 → KE → 知识 → RAG 闭环"展开为可执行 plan
> **跨 owner**：@backend-itsm（主导）+ @docs-itsm（协作）
> **截止**：v1.7 启动
> **关联**：本文是 owner-boards §P1-1 的"详细 plan"展开，对应主链路 B（重复事件 → 知识）

---

## 1. 端到端目标态

一条真实的"重复事件 → 知识可检索"路径在生产里跑通：

```
触发：同一类事件 N 分钟内报 ≥3 次（阈值可配）
       ↓
自动关联到现有问题 OR 创建新问题（事件聚类）
       ↓
问题调查：确定根因 CI + workaround
       ↓
创建 Known Error（KE）：有 workaround + 受影响 CI
       ↓
问题关闭：必须填"解决方案"或"无法确定原因"
       ↓
自动生成知识草稿（基于问题 + KE + 解决方案）
       ↓
知识审核：编辑 → 提交审核 → 批准 → 发布（版本+1）
       ↓
向量索引异步更新（发布后 5 分钟内）
       ↓
RAG 检索：用户/AI 检索 → 关键词降级 → 向量命中 → 权限过滤 → 引用源
       ↓
采纳/拒绝反馈（用户反馈 → AI 训练集）
```

**商业意义**：
- "重复事件不重复发生" = ITSM 的核心价值
- 知识从"沉淀"变成"被检索" = 知识库不是仓库是工具

---

## 2. 当前代码事实

| 项 | 当前状态 | 缺口 |
|---|---|---|
| **问题状态机** | 已有（创建 / 调查中 / KE / 已解决 / 已关闭） | 关闭路径不强制填根因 / 解决方案 |
| **Known Error 生命周期** | 部分实现（workaround 字段 + 受影响 CI 字段） | KE 状态转换未强制关联 CI 强引用 |
| **重复事件检测** | 无自动检测，需人工关联 | 需事件聚类（相似度 / 时间窗口 / CI 关联） |
| **知识草稿** | 需手工创建 | 需从问题 / KE 自动生成草稿 |
| **知识发布 workflow** | 有（草稿 / 审核 / 发布 / 下线） | 状态转换的权限 / 审计未与 RBAC 完全对齐 |
| **向量索引** | 部分实现（关键词降级 / 向量检索） | 向量删除未实现；发布后索引更新延迟不固定 |
| **RAG 检索** | 实现 | RBAC 可见性 / 租户过滤 / 版本过滤未端到端验证 |
| **采纳/拒绝反馈** | 框架存在（LLM Gateway 接受/拒绝） | 用户级反馈 UI 缺；反馈 → AI 训练缺闭环 |

---

## 3. 阶段拆分（7 阶段，预计 12 周）

### 阶段 1：问题关闭必填项（Week 1-2）

**目标**：关闭问题时必须填"根因"或"无法确定原因"；必须填"解决方案"或 workaround

**改动清单**：
- `ent/schema/problem.go`：加 `root_cause_required` 字段（枚举：`identified` / `workaround_only` / `unresolved`）
- `service/problem_service.go::CloseProblem`：校验必填项，缺则返回 409
- `handlers/problem/handler.go::CloseProblem`：返回 409 错误码 + 字段名
- 前端 `ProblemDetail.tsx`：关闭按钮加表单校验

**跨 owner 接口**：无（@backend-itsm 独立）

**Verification**：

```bash
# 1. 不填根因关单 → 409
curl -X POST http://localhost:8090/api/v1/problems/123/close \
  -H "Content-Type: application/json" -d '{}'
# 期望：{"code":409,"message":"root_cause_required"}

# 2. 填根因关单 → 200
curl -X POST ... -d '{"root_cause":"identified","solution":"重启服务"}'
# 期望：200

# 3. 历史问题数据迁移（脱敏副本先跑）
psql -f migrations/2026MMDD_problem_root_cause_required.sql <db>
```

---

### 阶段 2：KE 强引用 CI（Week 3）

**目标**：KE 的受影响 CI 字段从弱引用（string）改为强引用（CI ID），避免 CI 退役后查不到

**改动清单**：
- `ent/schema/known_error.go`：加 `affected_ci_ids []int` 字段（替代 string）
- 数据迁移：历史 string → 真实 CI ID（脱敏副本先跑一遍）
- `service/known_error_service.go::LinkCI`：校验 CI 存在 + 同租户
- 前端 `KnownErrorTab.tsx`：CI 选择器（类似工单的 CI 关联 UI）

**跨 owner 接口**：
- CMDB 提供 `GET /api/v1/cmdb/cis?ids=1,2,3` 批量查询（已有）

**Verification**：

```bash
# 1. 关联不存在的 CI → 404
curl -X POST .../api/v1/known-errors/123/link-ci -d '{"ci_ids":[99999]}'
# 期望：{"code":404,"message":"CI not found"}

# 2. 关联跨租户 CI → 403
curl -X POST ... -d '{"ci_ids":[<other_tenant_ci>]}'
# 期望：403

# 3. 强引用后 CI 退役，KE 仍可查询
#  测时序：KE 关联 CI-100 → CI-100 退役（soft delete）→ KE 查询仍可看到 CI-100 的引用
```

---

### 阶段 3：知识草稿自动生成（Week 4-5）

**目标**：问题关闭时自动生成知识草稿（基于问题 / KE / 解决方案）

**改动清单**：
- 新增 `service/knowledge_draft_generator.go`：
  - 输入：`problem_id` + `ke_id` + `solution`
  - 输出：knowledge article 草稿（标题 + body + tags）
  - 模板：LLM 生成（走 gateway） + 模板 fallback
- `service/problem_service.go::CloseProblem`：关闭时调用 draft generator
- `handlers/knowledge/handler.go`：新增 `POST /api/v1/knowledge/drafts/from-problem/{problem_id}`（备用入口）
- 前端 `ProblemDetail.tsx`：关闭后跳转知识草稿编辑页（可编辑）

**跨 owner 接口**：
- LLM Gateway：必须走（不直连 SDK，遵循 CONSTRAINTS#7）
- 知识草稿关联 `problem_id`（可追溯）

**Verification**：

```bash
# 1. 关闭问题 → 自动创建草稿
# 前端路径：打开问题 → 关闭 → 检查是否跳转 /knowledge/drafts/{draft_id}

# 2. 草稿含 problem_id
curl http://localhost:8090/api/v1/knowledge/drafts/{draft_id} | jq '.data.problem_id'
# 期望：非空

# 3. 草稿可编辑
curl -X PATCH .../api/v1/knowledge/drafts/{draft_id} -d '{"body":"修改后的正文"}'
# 期望：200

# 4. 草稿生成走 LLM Gateway（不直连 SDK）
grep -rn "openai\.\|anthropic\." itsm-backend/service/knowledge_draft_generator.go
# 期望：0 命中（必须走 gateway）
```

---

### 阶段 4：知识发布 workflow + RBAC（Week 6-7）

**目标**：知识发布流程完整；RBAC 可见性与版本管理对齐

**改动清单**：
- `service/knowledge_service.go`：
  - 状态机：draft → pending_review → approved → published → archived
  - 权限：谁能审核（`knowledge.reviewer` 角色）、谁能发布（`knowledge.publisher`）
- `ent/schema/knowledge.go`：加 `visibility`（枚举：`all` / `tenant` / `team` / `role`） + `version int`
- 数据迁移：加 visibility 列 + 默认值（`tenant`）
- 前端 `KnowledgeArticle.tsx`：发布按钮带权限校验（无权限 disable + tooltip）

**跨 owner 接口**（@docs-itsm）：
- 知识发布 = 内容治理，需要 docs-itsm 评审术语标准 / 分类
- docs-itsm 提供"知识分类标签清单"（`docs/knowledge/taxonomy.md`）

**Verification**：

```bash
# 1. draft → pending_review
curl -X POST .../api/v1/knowledge/{id}/submit-review
# 期望：状态变更，审计可见

# 2. 无权限审核 → 403
curl -X POST .../api/v1/knowledge/{id}/approve \
  -H "Authorization: Bearer <end_user_token>"
# 期望：403

# 3. visibility=team 的文章，跨团队检索 → 不可见
curl "http://localhost:8090/api/v1/knowledge/search?q=foo&team_id=<other_team>"
# 期望：不返回 visibility=team 的文章

# 4. 版本管理：发布后 version +1
curl .../api/v1/knowledge/{id} | jq '.data.version'
# 期望：每次发布递增
```

---

### 阶段 5：向量索引更新 + 删除（Week 8-9）

**目标**：知识发布后 5 分钟内向量索引更新；删除后立即从索引移除

**改动清单**：
- `service/vector_index_service.go`（可能新建）：
  - `OnKnowledgePublish(knowledge_id)`：异步触发 embedding + 索引更新
  - `OnKnowledgeDelete(knowledge_id)`：从向量索引移除
  - 重试 + 死信（沿用 outbox 模式）
- `itsm-rag/` Python 服务：加 `DELETE /vectors/{knowledge_id}` 端点
- `ent/schema/knowledge.go`：加 `vector_indexed_at TIMESTAMPTZ`

**跨 owner 接口**：
- 后端 → itsm-rag：HTTP API（已有） + 新增 DELETE 端点
- 索引状态可见性：返回 `vector_indexed_at`，前端可显示"索引中 / 已索引"

**Verification**：

```bash
# 1. 发布后 5 分钟内向量已索引
# 测时序：
T0:    知识发布
T0+5min: curl .../api/v1/knowledge/{id} | jq '.data.vector_indexed_at'
# 期望：非空

# 2. 删除后向量不可检索
T0:    删除知识
T0+1min: POST .../api/v1/rag/search -d '{"q":"关键词"}'
# 期望：不返回已删除

# 3. itsm-rag DELETE 端点直接验证
curl -X DELETE http://localhost:8081/vectors/{knowledge_id}
# 期望：204

# 4. 索引失败 → 重试 → 死信
# 故意关闭 itsm-rag → 发布知识 → 验证 outbox 重试 → 恢复后自动处理
```

---

### 阶段 6：RAG 检索 + 权限过滤（Week 10）

**目标**：RAG 检索结果严格按知识可见性 + 租户过滤；返回引用源 + 置信度

**改动清单**：
- `service/rag_service.go`：
  - 检索时按 actor's `tenant_id` + `team_ids` + `role` 过滤
  - 返回 `sources: [{knowledge_id, title, version, url}]` + `confidence: float`
- `itsm-rag/`：embedding 端点加 `tenant_id` 参数
- 前端 `RAGSearch.tsx`：展示引用源 + 点击跳转原知识

**跨 owner 接口**：
- 权限码：`knowledge.read`（已有）+ `knowledge.read.archived`（新增，@backend-itsm RBAC 加码）
- 流程：actor → 权限 → 可见性集合 → 检索 → 过滤

**Verification**：

```bash
# 1. 跨租户检索 → 不返回其他租户知识
curl -X POST .../api/v1/rag/search \
  -H "Authorization: Bearer <tenant_a_token>" \
  -d '{"q":"某关键词"}'
# 期望：结果不含 tenant_b 的知识（就算关键词命中）

# 2. 引用源可点击
curl .../api/v1/rag/search -d '{"q":"X"}' | jq '.data.sources[0].url'
# 期望：非空（可跳转）

# 3. 置信度字段存在
curl .../api/v1/rag/search -d '{"q":"X"}' | jq '.data.confidence'
# 期望：0.0-1.0 之间

# 4. 跨团队检索 → 不返回其他团队知识
curl .../api/v1/rag/search \
  -H "Authorization: Bearer <team_a_token>" \
  -d '{"q":"X"}'
# 期望：不返回 visibility=team 且 team_b 的知识
```

---

### 阶段 7：采纳/拒绝反馈闭环（Week 11）

**目标**：用户对 RAG 回答可点"有用/无用"；反馈进入 AI 训练集

**改动清单**：
- `ent/schema/rag_feedback.go`（新建）：`knowledge_id` + `answer_id` + `actor_id` + `feedback`（enum） + `comment`
- `service/rag_service.go::RecordFeedback`
- `handlers/rag/handler.go::POST /api/v1/rag/feedback`
- 前端 `RAGSearch.tsx`：每个回答下方 👍/👎 按钮
- 数据导出：每周导出 `rag_feedback` 给 AI 训练 pipeline

**跨 owner 接口**：
- @ai-itsm：接收 rag_feedback 数据（可选，v1.7 后启用）

**Verification**：

```bash
# 1. 提交反馈 → 200
curl -X POST .../api/v1/rag/feedback \
  -d '{"answer_id":"...","feedback":"useful"}'
# 期望：200

# 2. 反馈数据可查询
curl .../api/v1/rag/feedback/stats?knowledge_id=...
# 期望：返回 useful / useless 计数

# 3. UI 反馈按钮可点
# 前端：打开 RAG search → 点 👍/👎 → 看到 toast 提示

# 4. 重复反馈（同一 answer）→ 幂等
curl -X POST ... -d '{"answer_id":"...","feedback":"useful"}'  # 第 2 次
# 期望：200（不重复计数）
```

---

## 4. 跨 Owner 接口清单

| 接口 | 提供方 | 消费方 | 契约 |
|---|---|---|---|
| 知识草稿生成 | LLM Gateway | backend-itsm | 走 `internal/llm_gateway`（不直连 SDK，遵循 CONSTRAINTS#7） |
| CI 强引用校验 | CMDB（backend-itsm） | KE（backend-itsm） | `GET /api/v1/cmdb/cis/{id}` 返回 `tenant_id` |
| 知识分类标签 | docs-itsm | backend-itsm | `docs/knowledge/taxonomy.md` 静态清单 |
| 向量索引同步 | backend-itsm | itsm-rag | `POST /vectors` + `DELETE /vectors/{id}` |
| 权限码 `knowledge.read.archived` | backend-itsm | backend-itsm RBAC | 加码 → 跑 `cmd/authz-gen` → 守卫验证 |
| rag_feedback 数据 | backend-itsm | ai-itsm（future） | 每周导出 CSV 到 AI pipeline |

---

## 5. 端到端 Verification

### 5.1 真实数据 E2E

```bash
# 启动 dev 环境
docker compose -f docker-compose.dev.yml up -d

# 跑 E2E
cd itsm-backend
go test ./integration -run TestKnowledgeClosedLoop -v

# 期望场景：
# 1. 创建 5 个相似事件（同一类故障）
# 2. 自动聚类 → 关联到问题
# 3. 解决问题 → 创建 KE → 关闭
# 4. 自动生成知识草稿
# 5. 审核 → 发布
# 6. 5 分钟内向量索引更新
# 7. RAG 检索命中
# 8. 反馈 👍
# 全链路 PASS
```

### 5.2 探针矩阵（20 例）

| # | 场景 | 期望 |
|---|---|---|
| 1-5 | 5 种问题关闭路径（根因 / workaround / 未解决） | 200 / 409 区分 |
| 6-8 | 3 种 KE CI 关联（同租户 / 跨租户 / 不存在） | 200 / 403 / 404 |
| 9-11 | 3 种知识发布（draft / review / approved） | 200 / 权限区分 |
| 12-14 | 3 种向量操作（索引 / 删除 / 重试） | 200 / 200 / 200 |
| 15-17 | 3 种 RAG 检索（同租户 / 跨租户 / 无权限） | 有结果 / 空 / 403 |
| 18-20 | 3 种反馈（useful / useless / 重复） | 200 / 200 / 200 |

### 5.3 守卫验证

```bash
make docs-gate   # 应通过（含新增的 RBAC 权限码）
go test ./... -count=1   # 全绿
```

---

## 6. 时间盒

```
Week 1-2   阶段 1：问题关闭必填项
Week 3     阶段 2：KE 强引用 CI
Week 4-5   阶段 3：知识草稿自动生成
Week 6-7   阶段 4：知识发布 workflow + RBAC
Week 8-9   阶段 5：向量索引更新 + 删除
Week 10    阶段 6：RAG 检索 + 权限过滤
Week 11    阶段 7：采纳/拒绝反馈闭环
Week 12    集成测试 + 文档同步 + v1.7 启动
```

**总周期**：12 周 ≈ 3 个月（从 v1.7 启动开始）

---

## 7. 风险与缓解

| 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|
| 知识草稿 LLM 生成质量差 | 高 | 中 | 模板 fallback + 人工编辑必走 + 接受率监控 |
| 向量索引更新延迟 > 5 分钟 | 中 | 中 | 监控告警 + 异步重试 + 死信队列 |
| 权限过滤漏 → 跨租户泄露 | 低 | **高** | 探针矩阵 6-3 + 8 月评审 R5 守卫 + 灰度 |
| 反馈数据量爆炸 | 中 | 低 | 周导出 + 限额 + 归档 |
| 重复事件检测误判 | 中 | 中 | 阈值可配 + 人工确认兜底 |
| KE 强引用 CI 数据迁移丢失 | 低 | 中 | 脱敏副本先跑 + 双写兼容期 + 校验任务 |

---

## 8. 验收

**最终态**（v1.7 启动后）：
- [ ] 7 阶段全部完成
- [ ] E2E 全链路 PASS（20 例探针矩阵 0 失败）
- [ ] `make docs-gate` PASS
- [ ] 真实客户环境跑通（2 周观测）
- [ ] 知识接受率 ≥ 70%（发布后被 RAG 检索命中 / 被采纳）
- [ ] 重复事件率下降 ≥ 30%（同类事件不再重复发生）

---

## 关联文档

| 文档 | 关系 |
|---|---|
| [owner-boards §P1-1](./owner-boards-2026-09-24.md#p1-1--问题--ke--知识--rag-闭环主导) | 原始条目 |
| [业务快照 §4.2](./business-snapshot-2026-09-24.md#42--p1-本季度v17-启动后必做) | 优先级来源 |
| [商业能力契约 §链路 B](./itsm-commercial-capability-contract.md#链路-b重复事件到问题知识) | 主链路定义 |
| [领域所有权 §Problem / Known Error / Knowledge](../architecture/domain-ownership.md) | 各域迁移状态 |
| [itsm-rag README](../../itsm-rag/README.md) | RAG 服务接口 |

---

*详细 plan · 日期：2026-09-24 · 主导：@backend-itsm · 协作：@docs-itsm · 下次更新：阶段 1 完成后（预计 T+2 周）*